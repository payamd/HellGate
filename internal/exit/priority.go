package exit

import (
	"net"
	"strings"

	"github.com/payamd/HellGate/internal/frame"
)

// Drain tiers for multiplexed tunnel scheduling: lower = served first vs peers on
// the same client batch. Mirrors demonica/lib/tunnel/target_priority.dart.
const (
	tunnelTierRealtime = 0
	tunnelTierNormal   = 1
)

func tunnelTargetHost(raw string) string {
	t := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(t, "udp://") {
		t = t[len("udp://"):]
	}
	if t == "" {
		return ""
	}
	// IPv6: [dead:beef::1]:3478
	if strings.HasPrefix(t, "[") {
		closeBracket := strings.Index(t, "]")
		if closeBracket > 1 {
			inside := t[1:closeBracket]
			rest := t[closeBracket+1:]
			if strings.HasPrefix(rest, ":") && len(rest) > 1 {
				return inside
			}
			return inside
		}
		return ""
	}
	idx := strings.LastIndex(t, ":")
	if idx <= 0 || idx >= len(t)-1 {
		return t
	}
	return t[:idx]
}

func isLikelyWhatsAppRealtimeHost(host string) bool {
	if host == "" {
		return false
	}
	switch {
	case strings.HasSuffix(host, ".whatsapp.net"):
		return true
	case host == "whatsapp.net":
		return true
	case strings.HasSuffix(host, ".cdn.whatsapp.net"):
		return true
	case strings.HasSuffix(host, ".whatsapp.com"):
		return true
	case host == "whatsapp.com":
		return true
	}
	return strings.Contains(host, ".whatsapp.")
}

func isLikelyInstagramRealtimeHost(host string) bool {
	if host == "" {
		return false
	}
	h := strings.ToLower(host)
	switch {
	case strings.HasSuffix(h, ".instagram.com"), h == "instagram.com":
		return true
	case strings.HasSuffix(h, ".cdninstagram.com"):
		return true
	}
	return false
}

// Android TUN → SOCKS forwards dst as IP:port. Hostname rules never match.
// Aggregates align with AS32934 (Meta) summaries for IG/WA/Facebook edge traffic.
func isLikelyMetaMessagingIPv4(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	v4 := ip.To4()
	if v4 == nil {
		return false
	}
	a, b, c := v4[0], v4[1], v4[2]
	if a == 31 && b == 13 {
		return true // 31.13.0.0/16
	}
	if a == 57 && b >= 144 && b <= 147 {
		return true // 57.144.0.0/14
	}
	if a == 57 && b == 141 && c <= 21 {
		return true
	}
	if a == 129 && b == 134 && c < 128 {
		return true // 129.134.0.0/17
	}
	if a == 157 && b == 240 {
		return c < 128 || c >= 192 // 157.240.0.0/17 ∪ 157.240.192.0/18
	}
	if a == 163 && b == 70 && c >= 128 {
		return true // 163.70.128.0/17
	}
	if a == 173 && b == 252 && c >= 64 && c < 128 {
		return true
	}
	if a == 66 && b == 220 && c >= 144 && c < 160 {
		return true
	}
	if a == 69 && b == 171 && c >= 224 {
		return true // 69.171.224.0/19
	}
	if a == 69 && b == 63 && c >= 176 && c < 192 {
		return true
	}
	if a == 102 && b == 132 && c >= 96 && c < 112 {
		return true
	}
	if a == 45 && b == 64 && c >= 40 && c < 44 {
		return true
	}
	if a == 74 && b == 119 && c >= 76 && c < 80 {
		return true
	}
	if a == 103 && b == 4 && c >= 96 && c < 100 {
		return true
	}
	if a == 179 && b == 60 && c >= 192 && c < 196 {
		return true
	}
	if a == 185 && b == 60 && c >= 216 && c < 220 {
		return true
	}
	if a == 185 && b == 89 && c >= 216 && c < 220 {
		return true
	}
	if a == 204 && b == 15 && c >= 20 && c < 24 {
		return true
	}
	return false
}

func tunnelDrainTierForTarget(target string, relayCaps byte) int {
	if relayCaps == 0 {
		return tunnelTierNormal
	}
	h := tunnelTargetHost(target)
	if relayCaps&frame.RelayCapWhatsApp != 0 && isLikelyWhatsAppRealtimeHost(h) {
		return tunnelTierRealtime
	}
	if relayCaps&frame.RelayCapInstagram != 0 && isLikelyInstagramRealtimeHost(h) {
		return tunnelTierRealtime
	}
	if relayCaps&(frame.RelayCapWhatsApp|frame.RelayCapInstagram) != 0 && isLikelyMetaMessagingIPv4(h) {
		return tunnelTierRealtime
	}
	return tunnelTierNormal
}
