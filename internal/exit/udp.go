package exit

import (
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/payamd/HellGate/internal/frame"
)

const (
	udpTargetPrefix = "udp://"

	// Datagram read buffer (max IPv4 UDP payload).
	udpReadBuf = 65535

	// udpIdleTimeout reaps idle UDP NAT sessions (no client-side frames).
	// 3m helps VoIP / WhatsApp-style flows with occasional quiet periods on the
	// tunnel (media is still active upstream).
	udpIdleTimeout = 3 * time.Minute
)

// udpSession is a single logical UDP flow (one udp:// target, one session ID).
// Each upstream datagram becomes one downstream frame; each client frame
// payload becomes one upstream Write.
type udpSession struct {
	id     [frame.SessionIDLen]byte
	owner  [frame.ClientIDLen]byte
	target string // wire form: "udp://host:port"
	conn   *net.UDPConn

	mu     sync.Mutex
	down   [][]byte // pending payloads to tunnel toward client (each = one datagram)
	txSeq  uint64
	closed atomic.Bool
	onKick func()
}

func udpStripPrefix(target string) (hostPort string, ok bool) {
	if !strings.HasPrefix(target, udpTargetPrefix) {
		return "", false
	}
	return target[len(udpTargetPrefix):], true
}

func (s *Server) openUDPSession(
	id [frame.SessionIDLen]byte,
	target string, // "udp://host:port"
	owner [frame.ClientIDLen]byte,
) (*udpSession, error) {
	if s.cfg.UpstreamProxy != "" {
		return nil, errUDPProxyUnsupported
	}
	hostPort, ok := udpStripPrefix(target)
	if !ok || hostPort == "" {
		return nil, errUDPBadTarget
	}
	addr, err := net.ResolveUDPAddr("udp", hostPort)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	us := &udpSession{
		id:     id,
		owner:  owner,
		target: target,
		conn:   conn,
		onKick: nil,
	}
	us.onKick = func() {
		s.mu.Lock()
		s.udpReady[id] = struct{}{}
		s.mu.Unlock()
		s.kick(owner)
	}

	s.mu.Lock()
	s.udpSessions[id] = us
	s.sessionOwners[id] = owner
	s.firstReply[id] = struct{}{}
	s.lastActivity[id] = time.Now()
	s.mu.Unlock()
	s.stats.sessionsOpen.Add(1)

	log.Printf("[exit] new udp session %x owner=%x -> %s", id[:4], owner[:4], hostPort)

	go us.readLoop(s)

	return us, nil
}

func (us *udpSession) readLoop(s *Server) {
	buf := make([]byte, udpReadBuf)
	for !us.closed.Load() {
		_ = us.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := us.conn.Read(buf)
		if err != nil {
			if us.closed.Load() {
				return
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				continue
			}
			if err != io.EOF {
				log.Printf("[exit] udp upstream read %x: %v", us.id[:4], err)
			}
			us.shutdownFromUpstream(s)
			return
		}
		if n == 0 {
			continue
		}
		p := make([]byte, n)
		copy(p, buf[:n])
		us.enqueueDown(p)
	}
}

func (us *udpSession) enqueueDown(payload []byte) {
	us.mu.Lock()
	if us.closed.Load() {
		us.mu.Unlock()
		return
	}
	us.down = append(us.down, payload)
	kick := us.onKick
	us.mu.Unlock()
	if kick != nil {
		kick()
	}
}

func (us *udpSession) writeUpstream(payload []byte) {
	if len(payload) == 0 {
		return
	}
	if us.closed.Load() {
		return
	}
	_, err := us.conn.Write(payload)
	if err != nil && !us.closed.Load() {
		log.Printf("[exit] udp upstream write %x: %v", us.id[:4], err)
	}
}

// drainFrames returns up to maxFrames plaintext frames for the tunnel (downstream).
func (us *udpSession) drainFrames(maxFrames int) []*frame.Frame {
	us.mu.Lock()
	defer us.mu.Unlock()
	if len(us.down) == 0 {
		return nil
	}
	n := len(us.down)
	if maxFrames > 0 && n > maxFrames {
		n = maxFrames
	}
	out := make([]*frame.Frame, 0, n)
	for i := 0; i < n; i++ {
		p := us.down[i]
		us.txSeq++
		out = append(out, &frame.Frame{
			SessionID: us.id,
			Seq:       us.txSeq,
			Flags:     frame.FlagDATAGRAM,
			Payload:   p,
		})
	}
	us.down = us.down[n:]
	return out
}

func (us *udpSession) hasPendingDown() bool {
	us.mu.Lock()
	defer us.mu.Unlock()
	return len(us.down) > 0
}

// closeAndUnregister closes the UDP socket, removes server maps, and bumps stats.
// Safe to call more than once: idempotent after first close.
func (us *udpSession) closeAndUnregister(s *Server) {
	if us.closed.Swap(true) {
		return
	}
	if us.conn != nil {
		_ = us.conn.Close()
	}
	us.mu.Lock()
	us.down = nil
	us.mu.Unlock()
	s.mu.Lock()
	delete(s.udpSessions, us.id)
	delete(s.sessionOwners, us.id)
	delete(s.udpReady, us.id)
	delete(s.firstReply, us.id)
	delete(s.lastActivity, us.id)
	s.mu.Unlock()
	s.stats.sessionsClose.Add(1)
}

func (us *udpSession) shutdownFromUpstream(s *Server) {
	owner := us.owner
	sessID := us.id
	us.closeAndUnregister(s)
	fin := &frame.Frame{SessionID: sessID, Flags: frame.FlagFIN}
	s.mu.Lock()
	s.pendingRSTs[owner] = append(s.pendingRSTs[owner], fin)
	s.mu.Unlock()
	s.kick(owner)
}

// idleReapFinish closes the UDP socket after idle GC has already removed this
// session from Server maps. Idempotent.
func (us *udpSession) idleReapFinish() {
	if us.closed.Swap(true) {
		return
	}
	if us.conn != nil {
		_ = us.conn.Close()
	}
	us.mu.Lock()
	us.down = nil
	us.mu.Unlock()
}

var (
	errUDPProxyUnsupported = errString("udp: upstream_proxy does not support UDP yet; leave empty for direct UDP egress")
	errUDPBadTarget        = errString("udp: invalid udp:// target")
)

type errString string

func (e errString) Error() string { return string(e) }
