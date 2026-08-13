package main

import (
	"net/http/httptest"
	"testing"
)

func TestTerminalWebSocketOriginPolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		origin string
		host   string
		want   bool
	}{
		{name: "same origin", origin: "https://cloud.example.test", host: "cloud.example.test", want: true},
		{name: "same origin with port", origin: "http://localhost:8080", host: "localhost:8080", want: true},
		{name: "cross origin", origin: "https://attacker.example", host: "cloud.example.test", want: false},
		{name: "lookalike host", origin: "https://cloud.example.test.attacker", host: "cloud.example.test", want: false},
		{name: "non browser client", origin: "", host: "cloud.example.test", want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "http://cloud.example.test/terminal", nil)
			r.Host = tc.host
			r.Header.Set("Origin", tc.origin)
			if got := upgrader.CheckOrigin(r); got != tc.want {
				t.Fatalf("CheckOrigin(%q)=%v, want %v", tc.origin, got, tc.want)
			}
		})
	}
}
