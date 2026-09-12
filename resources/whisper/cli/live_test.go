package main

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWhisperLive(t *testing.T) {
	if os.Getenv("RESOURCE_LIVE_TEST") != "1" {
		t.Skip("set RESOURCE_LIVE_TEST=1 to run against whisper")
	}
	base := os.Getenv("WHISPER_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:8090"
	}
	fixture := filepath.Join("..", "..", "..", "scenarios", "audio-tools", "api", "internal", "diagnostics", "smokedata", "quality_speech.wav")
	f, err := os.Open(fixture)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("audio_file", filepath.Base(fixture))
		if err == nil {
			_, err = io.Copy(part, f)
		}
		_ = mw.Close()
		_ = pw.CloseWithError(err)
	}()
	req, err := http.NewRequest(http.MethodPost, base+"/asr", pr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /asr = %d: %s", resp.StatusCode, body)
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		t.Fatal("POST /asr returned an empty transcript")
	}
}
