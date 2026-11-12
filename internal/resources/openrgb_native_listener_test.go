package resources

import (
	platform "github.com/vrooli/platform-go"
	"net/netip"
	"testing"
)

func TestNativeOpenRGBRejectsForeignWildcardAndAmbiguousListeners(t *testing.T) {
	endpoint := netip.MustParseAddrPort("127.0.0.1:6742")
	exact := platform.ProcessTCPListener{Address: "127.0.0.1", Port: 6742, PID: 41}
	for _, tc := range []struct {
		name      string
		listeners []platform.ProcessTCPListener
		good      bool
	}{
		{"exact", []platform.ProcessTCPListener{exact}, true},
		{"absent", nil, false},
		{"foreign", []platform.ProcessTCPListener{{Address: "127.0.0.1", Port: 6742, PID: 42}}, false},
		{"wildcard", []platform.ProcessTCPListener{{Address: "0.0.0.0", Port: 6742, PID: 41}}, false},
		{"ipv6", []platform.ProcessTCPListener{{Address: "::1", Port: 6742, PID: 41}}, false},
		{"mapped", []platform.ProcessTCPListener{{Address: "::ffff:127.0.0.1", Port: 6742, PID: 41}}, false},
		{"port", []platform.ProcessTCPListener{{Address: "127.0.0.1", Port: 6743, PID: 41}}, false},
		{"second-foreign", []platform.ProcessTCPListener{exact, {Address: "::", Port: 6742, PID: 42}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := verifyOpenRGBNativeListeners(endpoint, 41, tc.listeners); (err == nil) != tc.good {
				t.Fatalf("accepted=%v error=%v", tc.good, err)
			}
		})
	}
}
