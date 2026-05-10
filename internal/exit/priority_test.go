package exit

import (
	"testing"

	"github.com/payamd/HellGate/internal/frame"
)

func TestTunnelDrainTierForTarget(t *testing.T) {
	all := frame.RelayCapsMessaging // WA + IG
	tests := []struct {
		raw  string
		caps byte
		want int
	}{
		{"g.whatsapp.net:443", all, tunnelTierRealtime},
		{"udp://g.whatsapp.net:3487", all, tunnelTierRealtime},
		{"31.13.80.175:5222", all, tunnelTierRealtime}, // TUN-style Meta IP
		{"57.144.222.192:443", all, tunnelTierRealtime},
		{"57.146.1.1:443", all, tunnelTierRealtime},      // /14 upper quad
		{"57.141.10.55:443", all, tunnelTierRealtime},
		{"163.70.200.50:443", all, tunnelTierRealtime},
		{"173.252.90.60:443", all, tunnelTierRealtime},
		{"66.220.150.1:443", all, tunnelTierRealtime},
		{"69.171.230.100:443", all, tunnelTierRealtime},
		{"102.132.100.54:443", all, tunnelTierRealtime},
		{"i.instagram.com:443", all, tunnelTierRealtime},
		{"edge-chat.instagram.com:443", all, tunnelTierRealtime},
		{"g.whatsapp.net:443", frame.RelayCapInstagram, tunnelTierNormal},
		{"i.instagram.com:443", frame.RelayCapWhatsApp, tunnelTierNormal},
		{"example.com:443", all, tunnelTierNormal},
		{"i.instagram.com:443", 0, tunnelTierNormal},
		{"udp://8.8.8.8:53", all, tunnelTierNormal},
		{"213.155.156.114:443", all, tunnelTierNormal}, // outside AS32934 aggregates here
		{"157.240.150.1:443", all, tunnelTierNormal}, // gap between /17 and /192.0/18
	}
	for _, tt := range tests {
		got := tunnelDrainTierForTarget(tt.raw, tt.caps)
		if got != tt.want {
			t.Errorf("tunnelDrainTierForTarget(%q, %#x) = %d, want %d",
				tt.raw, tt.caps, got, tt.want)
		}
	}
}
