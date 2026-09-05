package onboard

import "testing"

func TestValidateControlPlaneURL(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		mode      ReachabilityMode
		wantErr   bool
	}{
		{"lan IP", "http://192.168.1.173:18767", ReachabilityLAN, false},
		{"tunnel HTTPS", "https://bridge.example.test", ReachabilityTunnel, false},
		{"manual loopback", "http://127.0.0.1:18767", ReachabilityManual, false},
		{"lan loopback", "http://localhost:18767", ReachabilityLAN, true},
		{"credentials", "http://user:secret@bridge.test", ReachabilityLAN, true},
		{"path", "http://bridge.test/not-a-base", ReachabilityLAN, true},
		{"not HTTP", "ssh://bridge.test", ReachabilityLAN, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateControlPlaneURL(tc.raw, tc.mode)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateControlPlaneURL(%q) error=%v, wantErr=%v", tc.raw, err, tc.wantErr)
			}
		})
	}
}
