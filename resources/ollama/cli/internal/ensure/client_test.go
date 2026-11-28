package ensure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRunningRejectsInvalidVRAMMeasurements(t *testing.T) {
	for name, body := range map[string]string{
		"negative residency":  `{"models":[{"name":"target:8b","size_vram":8589934592},{"name":"other:4b","size_vram":-2147483648}]}`,
		"aggregate overflow":  `{"models":[{"name":"target:8b","size_vram":9223372036854775807},{"name":"other:4b","size_vram":1}]}`,
		"individual overflow": `{"models":[{"name":"target:8b","size_vram":9223372036854775808}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/ps" {
					t.Errorf("request path = %q, want /api/ps", r.URL.Path)
				}
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			client := &Client{BaseURL: server.URL, HTTP: server.Client()}
			if _, err := client.ListRunning(context.Background()); err == nil {
				t.Fatal("ListRunning succeeded with an invalid residency measurement")
			}
		})
	}
}
