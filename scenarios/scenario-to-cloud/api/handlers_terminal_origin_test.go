package main

import (
	"net/http/httptest"
	"testing"
)

// [REQ:STC-P0-014] The terminal upgrade admits only browser origins the
// management boundary admits; an absent Origin is refused.
func TestTerminalWebSocketOriginPolicy(t *testing.T) {
	srv := newTestServer()
	upgrader := srv.terminalUpgrader()
	for _, tc := range []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "same origin", origin: "https://cloud.example.test", host: "cloud.example.test", want: true},
		{name: "same origin with port", origin: "http://localhost:8080", host: "localhost:8080", want: true},
		{name: "loopback ui origin on loopback bind", origin: "http://localhost:23013", host: "localhost:15672", want: true},
		{name: "cross origin", origin: "https://attacker.example", host: "cloud.example.test", want: false},
		{name: "lookalike host", origin: "https://cloud.example.test.attacker", host: "cloud.example.test", want: false},
		{name: "missing origin", origin: "", host: "cloud.example.test", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://cloud.example.test/terminal", nil)
			r.Host = tc.host
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := upgrader.CheckOrigin(r); got != tc.want {
				t.Fatalf("CheckOrigin(%q)=%v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}
