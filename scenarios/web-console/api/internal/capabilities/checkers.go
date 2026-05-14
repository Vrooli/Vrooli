package capabilities

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

type ResourceChecker struct {
	URL    string
	Client *http.Client
}

func (c *ResourceChecker) Check(ctx context.Context) (Status, string) {
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
		numSamples  = 1600
		dataSize    = numSamples * numChannels * (bitsPerSamp / 8)
	)

	buf := &bytes.Buffer{}
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(numChannels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate*numChannels*bitsPerSamp/8))
	_ = binary.Write(buf, binary.LittleEndian, uint16(numChannels*bitsPerSamp/8))
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSamp))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(dataSize))
	buf.Write(make([]byte, dataSize))

	return buf.Bytes()
}

func (c *WhisperChecker) Check(ctx context.Context) (Status, string) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

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

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(asrResp.Body).Decode(&result); err != nil {
		return StatusUnavailable, "transcription response is not valid JSON"
	}

	return StatusAvailable, "resource is healthy and transcription verified"
}

// KokoroChecker verifies that Kokoro can actually synthesize audio by
// sending a minimal text input to the /v1/audio/speech endpoint. This
// catches cases where the voices endpoint responds but synthesis is broken.
type KokoroChecker struct {
	BaseURL       string
	Client        *http.Client
	ContainerName string
	InspectState  func(ctx context.Context, containerName string) (exists bool, running bool, err error)
}

func inspectDockerContainerState(ctx context.Context, containerName string) (exists bool, running bool, err error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", containerName)
	output, err := cmd.Output()
	if err != nil {
		return false, false, err
	}

	state := strings.TrimSpace(string(output))
	return true, state == "true", nil
}

func (c *KokoroChecker) Check(ctx context.Context) (Status, string) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	liveReq, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/v1/audio/voices", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}
	liveResp, err := client.Do(liveReq)
	if err != nil {
		inspectState := c.InspectState
		if inspectState == nil {
			inspectState = inspectDockerContainerState
		}
		containerName := c.ContainerName
		if containerName == "" {
			containerName = "kokoro"
		}
		if exists, running, inspectErr := inspectState(ctx, containerName); inspectErr == nil {
			if !exists {
				return StatusUnavailable, "resource is not installed"
			}
			if !running {
				return StatusUnavailable, "resource is stopped"
			}
		}
		return StatusUnavailable, "resource is not responding"
	}
	liveResp.Body.Close()
	if liveResp.StatusCode != http.StatusOK {
		return StatusUnavailable, "resource returned unexpected status"
	}

	body := `{"input":"test","voice":"af_heart","response_format":"mp3","model":"kokoro"}`
	synthReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/v1/audio/speech", strings.NewReader(body))
	if err != nil {
		return StatusUnavailable, "failed to create synthesis request: " + err.Error()
	}
	synthReq.Header.Set("Content-Type", "application/json")

	synthResp, err := client.Do(synthReq)
	if err != nil {
		return StatusUnavailable, "synthesis request failed: " + err.Error()
	}
	defer synthResp.Body.Close()

	if synthResp.StatusCode != http.StatusOK {
		return StatusUnavailable, "synthesis endpoint returned non-200 status"
	}

	buf := make([]byte, 4)
	n, _ := io.ReadFull(synthResp.Body, buf)
	if n < 4 {
		return StatusUnavailable, "synthesis returned empty audio"
	}

	return StatusAvailable, "resource is healthy and synthesis verified"
}

// OllamaChecker verifies that Ollama is running by hitting its /api/tags
// endpoint.
type OllamaChecker struct {
	BaseURL string
	Client  *http.Client
}

func (c *OllamaChecker) Check(ctx context.Context) (Status, string) {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/tags", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "Ollama is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusAvailable, "Ollama is running"
	}

	return StatusUnavailable, "Ollama returned unexpected status"
}

// OpenRouterChecker verifies that OpenRouter is configured and reachable.
type OpenRouterChecker struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func (c *OpenRouterChecker) Check(ctx context.Context) (Status, string) {
	if c.APIKey == "" {
		return StatusUnavailable, "OPENROUTER_API_KEY not configured"
	}

	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}

	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/v1/models", nil)
	if err != nil {
		return StatusUnavailable, "failed to create request: " + err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		return StatusUnavailable, "OpenRouter is not reachable"
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return StatusAvailable, "OpenRouter is configured and reachable"
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return StatusUnavailable, "OpenRouter API key is invalid"
	}

	return StatusUnavailable, "OpenRouter returned unexpected status"
}
