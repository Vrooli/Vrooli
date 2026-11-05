package lifecycle

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoopbackOnly(t *testing.T) {
	h := LoopbackOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	for _, tc := range []struct {
		name, remote string
		want         int
	}{
		{name: "ipv4", remote: "127.0.0.1:1234", want: http.StatusNoContent},
		{name: "ipv6", remote: "[::1]:1234", want: http.StatusNoContent},
		{name: "remote", remote: "192.0.2.10:1234", want: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://localhost", nil)
			req.RemoteAddr = tc.remote
			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			if resp.Code != tc.want {
				t.Fatalf("status = %d, want %d", resp.Code, tc.want)
			}
		})
	}
}
