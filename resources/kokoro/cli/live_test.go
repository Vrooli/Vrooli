package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestKokoroLive(t *testing.T) {
	if os.Getenv("RESOURCE_LIVE_TEST") != "1" {
		t.Skip("set RESOURCE_LIVE_TEST=1 to run against kokoro")
	}
	base := os.Getenv("KOKORO_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8880"
	}
	voices, err := http.Get(base + "/v1/audio/voices")
	if err != nil {
		t.Fatal(err)
	}
	defer voices.Body.Close()
	vbody, _ := io.ReadAll(voices.Body)
	if voices.StatusCode != http.StatusOK || !strings.Contains(string(vbody), "af_heart") {
		t.Fatalf("GET voices = %d: %s", voices.StatusCode, vbody)
	}
	req, err := http.NewRequest(http.MethodPost, base+"/v1/audio/speech", bytes.NewBufferString(`{"model":"kokoro","input":"Runtime honesty test.","voice":"af_heart","response_format":"wav"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || len(audio) == 0 {
		t.Fatalf("POST speech = %d, bytes=%d", resp.StatusCode, len(audio))
	}
}
