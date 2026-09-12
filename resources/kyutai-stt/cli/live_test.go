package main

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestKyutaiLive(t *testing.T) {
	if os.Getenv("RESOURCE_LIVE_TEST") != "1" {
		t.Skip("set RESOURCE_LIVE_TEST=1 to run against kyutai-stt")
	}
	base := os.Getenv("KYUTAI_STT_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8094"
	}
	for _, path := range []string{"/ready", "/v1/info"} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d: %s", path, resp.StatusCode, body)
		}
		if path == "/v1/info" && !strings.Contains(string(body), "sample_rate") {
			t.Fatalf("GET /v1/info missing sample_rate: %s", body)
		}
	}
}
