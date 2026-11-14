package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSpeakerVerificationLive(t *testing.T) {
	if os.Getenv("RESOURCE_LIVE_TEST") != "1" {
		t.Skip("set RESOURCE_LIVE_TEST=1 to run against speaker-verification")
	}
	base := os.Getenv("SPEAKER_VERIFICATION_BASE_URL")
	if base == "" {
		base = "http://127.0.0.1:11452"
	}
	ready, err := http.Get(base + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	defer ready.Body.Close()
	body, _ := io.ReadAll(ready.Body)
	if ready.StatusCode != http.StatusOK {
		t.Fatalf("GET /ready = %d: %s", ready.StatusCode, body)
	}
	info, err := http.Get(base + "/v1/info")
	if err != nil {
		t.Fatal(err)
	}
	defer info.Body.Close()
	body, _ = io.ReadAll(info.Body)
	if info.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/info = %d: %s", info.StatusCode, body)
	}

	fixture := filepath.Join("..", "..", "..", "scenarios", "audio-tools", "api", "internal", "diagnostics", "smokedata", "quality_speech.wav")
	profileID := "live-test-" + time.Now().UTC().Format("20060102150405.000000000")
	t.Cleanup(func() {
		req, _ := http.NewRequest(http.MethodDelete, base+"/v1/profiles/"+profileID, nil)
		if req != nil {
			_, _ = http.DefaultClient.Do(req)
		}
	})
	audio, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	// The shared smoke fixture is shorter than the resource's 3-second minimum.
	// Repeat its PCM payload into a valid WAV so the live lifecycle exercises
	// genuine enrollment instead of merely proving the validation error path.
	audio = repeatWAV(t, audio, 2)
	enroll := postAudio(t, base+"/v1/profiles", audio, "quality_speech.wav", map[string]string{"profile_id": profileID, "display_name": "live test"})
	if enroll.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(enroll.Body)
		_ = enroll.Body.Close()
		t.Fatalf("POST /v1/profiles = %d: %s", enroll.StatusCode, body)
	}
	_ = enroll.Body.Close()
	verify := postAudio(t, base+"/v1/verify", audio, "quality_speech.wav", map[string]string{"profile_id": profileID})
	body, _ = io.ReadAll(verify.Body)
	_ = verify.Body.Close()
	if verify.StatusCode != http.StatusOK {
		t.Fatalf("POST /v1/verify = %d: %s", verify.StatusCode, body)
	}
	var result struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("decode verify response: %v: %s", err, body)
	}
	if result.Score <= 0 {
		t.Fatalf("verify score = %f, want positive genuine score", result.Score)
	}
}

func postAudio(t *testing.T, target string, audio []byte, filename string, fields map[string]string) *http.Response {
	t.Helper()
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		for key, value := range fields {
			_ = mw.WriteField(key, value)
		}
		part, err := mw.CreateFormFile("audio", filename)
		if err == nil {
			_, err = io.Copy(part, bytes.NewReader(audio))
		}
		_ = mw.Close()
		_ = pw.CloseWithError(err)
	}()
	req, err := http.NewRequest(http.MethodPost, target, pr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func repeatWAV(t *testing.T, wav []byte, copies int) []byte {
	t.Helper()
	if copies < 1 || len(wav) < 20 || string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatal("fixture is not a canonical PCM WAV")
	}
	dataSizeOffset, payloadOffset, payloadEnd := 0, 0, 0
	for offset := 12; offset+8 <= len(wav); {
		size := int(binary.LittleEndian.Uint32(wav[offset+4 : offset+8]))
		if offset+8+size > len(wav) {
			break
		}
		if string(wav[offset:offset+4]) == "data" {
			dataSizeOffset, payloadOffset, payloadEnd = offset+4, offset+8, offset+8+size
			break
		}
		offset += 8 + size
		if size%2 != 0 {
			offset++
		}
	}
	if payloadOffset == 0 {
		t.Fatal("fixture WAV has no data chunk")
	}
	payload := wav[payloadOffset:payloadEnd]
	out := make([]byte, payloadOffset, payloadOffset+len(payload)*copies)
	copy(out, wav[:payloadOffset])
	for range copies {
		out = append(out, payload...)
	}
	binary.LittleEndian.PutUint32(out[4:8], uint32(len(out)-8))
	binary.LittleEndian.PutUint32(out[dataSizeOffset:dataSizeOffset+4], uint32(len(out)-payloadOffset))
	return out
}
