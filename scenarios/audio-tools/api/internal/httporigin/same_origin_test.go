package httporigin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatches(t *testing.T) {
	tests := []struct {
		name   string
		host   string
		scheme string
		origin string
		proxy  string
		want   bool
	}{
		{name: "direct same origin", host: "audio-tools.example", scheme: "https", origin: "https://audio-tools.example", want: true},
		{name: "direct scheme mismatch", host: "audio-tools.example", scheme: "http", origin: "https://audio-tools.example", want: false},
		{name: "default https port", host: "audio-tools.example:443", scheme: "https", origin: "https://audio-tools.example", want: true},
		{name: "nondefault port", host: "audio-tools.example:8443", scheme: "https", origin: "https://audio-tools.example", want: false},
		{name: "proxy preserves browser host", host: "127.0.0.1:15373", scheme: "http", origin: "https://audio-tools.example", proxy: "audio-tools.example", want: true},
		{name: "cross origin through proxy", host: "127.0.0.1:15373", scheme: "http", origin: "https://attacker.example", proxy: "audio-tools.example", want: false},
		{name: "multiple forwarded hosts refused", host: "127.0.0.1:15373", scheme: "http", origin: "https://audio-tools.example", proxy: "audio-tools.example, attacker.example", want: false},
		{name: "opaque origin refused", host: "audio-tools.example", scheme: "https", origin: "null", want: false},
		{name: "origin path refused", host: "audio-tools.example", scheme: "https", origin: "https://audio-tools.example/path", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.scheme+"://"+tt.host+"/ws/stt/transcribe-stream-v2", nil)
			req.Host = tt.host
			if tt.proxy != "" {
				req.Header.Set("X-Forwarded-Host", tt.proxy)
			}
			if got := Matches(req, tt.origin); got != tt.want {
				t.Fatalf("Matches(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}
