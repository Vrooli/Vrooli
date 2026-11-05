package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestSherpaOnnxLiveContract(t *testing.T) {
	if os.Getenv("RESOURCE_LIVE_TEST") != "1" {
		t.Skip("set RESOURCE_LIVE_TEST=1 to run against sherpa-onnx")
	}
	base := os.Getenv("SHERPA_ONNX_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8881"
	}
	voices, err := http.Get(base + "/v1/audio/voices")
	if err != nil {
		t.Fatal(err)
	}
	defer voices.Body.Close()
	body, _ := io.ReadAll(voices.Body)
	if voices.StatusCode != http.StatusOK || !strings.Contains(string(body), "af_heart") {
		t.Fatalf("voices = %d: %s", voices.StatusCode, body)
	}
	resp, err := http.Post(base+"/v1/audio/speech", "application/json", bytes.NewBufferString(`{"model":"kokoro","input":"Runtime honesty test.","voice":"af_heart","response_format":"wav"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	audio, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || len(audio) < 44 || string(audio[:4]) != "RIFF" {
		t.Fatalf("speech = %d, bytes=%d", resp.StatusCode, len(audio))
	}
}
