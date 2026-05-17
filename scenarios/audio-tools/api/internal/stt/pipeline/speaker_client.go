package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type SpeakerProfile struct {
	ID                     string  `json:"id"`
	DisplayName            string  `json:"display_name"`
	CreatedAt              string  `json:"created_at"`
	UpdatedAt              string  `json:"updated_at"`
	ModelName              string  `json:"model_name"`
	EmbeddingDim           int     `json:"embedding_dim"`
	SampleRate             int     `json:"sample_rate"`
	EnrollmentAudioSeconds float64 `json:"enrollment_audio_seconds"`
	Notes                  string  `json:"notes"`
}

type SpeakerProfileList struct {
	Profiles []SpeakerProfile `json:"profiles"`
	Count    int              `json:"count"`
}

type SpeakerResourceInfo struct {
	Backend      string `json:"backend"`
	Model        string `json:"model"`
	Device       string `json:"device"`
	SampleRate   int    `json:"sample_rate"`
	Version      string `json:"version"`
	EmbeddingDim int    `json:"embedding_dim"`
}

type SpeakerResourceReady struct {
	Status         string `json:"status"`
	ModelLoaded    bool   `json:"model_loaded"`
	ProfileStoreOK bool   `json:"profile_store_ok"`
	TempDirOK      bool   `json:"temp_dir_ok"`
}

type SpeakerEnrollmentResponse struct {
	ProfileID              string  `json:"profile_id"`
	DisplayName            string  `json:"display_name"`
	EmbeddingDim           int     `json:"embedding_dim"`
	SampleRate             int     `json:"sample_rate"`
	EnrollmentAudioSeconds float64 `json:"enrollment_audio_seconds"`
	ModelName              string  `json:"model_name"`
	CreatedAt              string  `json:"created_at"`
}

type SpeakerVerifyResult struct {
	ProfileID    string  `json:"profile_id"`
	Matched      bool    `json:"matched"`
	Score        float64 `json:"score"`
	Threshold    float64 `json:"threshold"`
	DurationMs   float64 `json:"duration_ms"`
	Backend      string  `json:"backend"`
	Model        string  `json:"model"`
	AudioSeconds float64 `json:"audio_seconds"`
}

// SpeakerExtractionResult holds the response from the /v1/extract endpoint.
type SpeakerExtractionResult struct {
	Audio        []byte
	Score        float64
	Matched      bool
	DurationMs   float64
	AudioSeconds float64
}

// SpeakerClient is the HTTP client for the speaker-verification resource.
type SpeakerClient struct {
	BaseURL string
	Client  *http.Client
}

func (c *SpeakerClient) httpClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 5 * time.Second}
}

func (c *SpeakerClient) endpoint(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + path
}

func (c *SpeakerClient) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(path), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *SpeakerClient) doJSON(req *http.Request, out any) error {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *SpeakerClient) postMultipart(
	ctx context.Context,
	path string,
	fields map[string]string,
	fileField string,
	filename string,
	fileContents []byte,
	out any,
) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("write field %s: %w", key, err)
		}
	}
	part, err := writer.CreateFormFile(fileField, filename)
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(fileContents); err != nil {
		return fmt.Errorf("write file body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), &body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return c.doJSON(req, out)
}

func (c *SpeakerClient) Ready(ctx context.Context) (SpeakerResourceReady, error) {
	var out SpeakerResourceReady
	err := c.getJSON(ctx, "/ready", &out)
	return out, err
}

func (c *SpeakerClient) Info(ctx context.Context) (SpeakerResourceInfo, error) {
	var out SpeakerResourceInfo
	err := c.getJSON(ctx, "/v1/info", &out)
	return out, err
}

func (c *SpeakerClient) ListProfiles(ctx context.Context) (SpeakerProfileList, error) {
	var out SpeakerProfileList
	err := c.getJSON(ctx, "/v1/profiles", &out)
	return out, err
}

func (c *SpeakerClient) Enroll(
	ctx context.Context,
	audio []byte,
	profileID, displayName, notes string,
) (SpeakerEnrollmentResponse, error) {
	var out SpeakerEnrollmentResponse
	err := c.postMultipart(
		ctx,
		"/v1/profiles",
		map[string]string{
			"profile_id":   profileID,
			"display_name": displayName,
			"notes":        notes,
		},
		"audio",
		"enrollment.webm",
		audio,
		&out,
	)
	return out, err
}

func (c *SpeakerClient) Verify(
	ctx context.Context,
	audio []byte,
	profileID string,
	threshold float64,
) (SpeakerVerifyResult, error) {
	var out SpeakerVerifyResult
	err := c.postMultipart(
		ctx,
		"/v1/verify",
		map[string]string{
			"profile_id": profileID,
			"threshold":  fmt.Sprintf("%.6f", threshold),
		},
		"audio",
		"segment.webm",
		audio,
		&out,
	)
	return out, err
}

func (c *SpeakerClient) Extract(
	ctx context.Context,
	audio []byte,
	profileID string,
	verify bool,
) (SpeakerExtractionResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("profile_id", profileID); err != nil {
		return SpeakerExtractionResult{}, fmt.Errorf("write profile_id: %w", err)
	}
	verifyStr := "true"
	if !verify {
		verifyStr = "false"
	}
	if err := writer.WriteField("verify", verifyStr); err != nil {
		return SpeakerExtractionResult{}, fmt.Errorf("write verify: %w", err)
	}
	part, err := writer.CreateFormFile("audio", "segment.webm")
	if err != nil {
		return SpeakerExtractionResult{}, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return SpeakerExtractionResult{}, fmt.Errorf("write audio: %w", err)
	}
	if err := writer.Close(); err != nil {
		return SpeakerExtractionResult{}, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/v1/extract"), &body)
	if err != nil {
		return SpeakerExtractionResult{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return SpeakerExtractionResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return SpeakerExtractionResult{}, fmt.Errorf("extract: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return SpeakerExtractionResult{}, fmt.Errorf("read response body: %w", err)
	}

	result := SpeakerExtractionResult{Audio: audioBytes}
	if s := resp.Header.Get("X-Speaker-Score"); s != "" {
		_, _ = fmt.Sscanf(s, "%f", &result.Score)
	}
	if s := resp.Header.Get("X-Speaker-Matched"); s != "" {
		result.Matched = strings.EqualFold(s, "true")
	}
	if s := resp.Header.Get("X-Duration-Ms"); s != "" {
		_, _ = fmt.Sscanf(s, "%f", &result.DurationMs)
	}
	if s := resp.Header.Get("X-Audio-Seconds"); s != "" {
		_, _ = fmt.Sscanf(s, "%f", &result.AudioSeconds)
	}
	return result, nil
}

func (c *SpeakerClient) DeleteProfile(ctx context.Context, profileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint("/v1/profiles/"+profileID), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
