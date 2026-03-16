package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type ResourceChecker struct {
	URL    string
	Client *http.Client
}

func (c *ResourceChecker) Check(ctx context.Context) (CapabilityStatus, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "resource is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusTemporaryRedirect {
		return StatusAvailable, "resource is healthy"
	}

	return StatusUnavailable, "resource returned unexpected status"
}

// WhisperChecker verifies that Whisper can actually transcribe audio by
// sending a minimal silent WAV to the /asr endpoint. This catches cases
// where the Whisper health endpoint (GET /) responds but transcription
// is broken (e.g. model not loaded, ffmpeg issues).
type WhisperChecker struct {
	// BaseURL is the Whisper base URL (e.g. "http://localhost:8090").
	BaseURL string
	Client  *http.Client
}

// generateSilentWAV builds a minimal 16kHz mono 16-bit PCM WAV file
// containing ~0.1s of silence (1600 samples = 3200 bytes of audio data).
func generateSilentWAV() []byte {
	const (
		sampleRate  = 16000
		numChannels = 1
		bitsPerSamp = 16
		numSamples  = 1600 // 0.1s
		dataSize    = numSamples * numChannels * (bitsPerSamp / 8)
	)

	buf := &bytes.Buffer{}
	// RIFF header
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	// fmt sub-chunk
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16)) // sub-chunk size
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))  // PCM
	_ = binary.Write(buf, binary.LittleEndian, uint16(numChannels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*numChannels*bitsPerSamp/8))
	_ = binary.Write(buf, binary.LittleEndian, uint16(numChannels*bitsPerSamp/8))
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSamp))
	// data sub-chunk
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(make([]byte, dataSize)) // silence

	return buf.Bytes()
}

func (c *WhisperChecker) Check(ctx context.Context) (CapabilityStatus, string) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	// First do a quick liveness check (GET /).
	liveReq, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}
	liveResp, err := client.Do(liveReq)
	if err != nil {
		return StatusUnavailable, "resource is not responding"
	}
	liveResp.Body.Close()
	if liveResp.StatusCode != http.StatusOK && liveResp.StatusCode != http.StatusTemporaryRedirect {
		return StatusUnavailable, "resource returned unexpected status"
	}

	// Send a minimal silent WAV to verify transcription works.
	wav := generateSilentWAV()

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	go func() {
		defer pw.Close()
		part, err := writer.CreateFormFile("audio_file", "health.wav")
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := part.Write(wav); err != nil {
			pw.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	asrURL := c.BaseURL + "/asr?output=json"
	asrReq, err := http.NewRequestWithContext(ctx, "POST", asrURL, pr)
	if err != nil {
		return StatusUnavailable, "failed to create transcription request: " + err.Error()
	}
	asrReq.Header.Set("Content-Type", writer.FormDataContentType())

	asrResp, err := client.Do(asrReq)
	if err != nil {
		return StatusUnavailable, "transcription request failed: " + err.Error()
	}
	defer asrResp.Body.Close()

	if asrResp.StatusCode != http.StatusOK {
		return StatusUnavailable, "transcription endpoint returned non-200 status"
	}

	// Verify the response is valid JSON with a "text" field.
	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(asrResp.Body).Decode(&result); err != nil {
		return StatusUnavailable, "transcription response is not valid JSON"
	}

	return StatusAvailable, "resource is healthy and transcription verified"
}
