package main

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

type SpeakerVerificationProfile struct {
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

type SpeakerVerificationProfileList struct {
	Profiles []SpeakerVerificationProfile `json:"profiles"`
	Count    int                          `json:"count"`
}

type SpeakerVerificationResourceInfo struct {
	Backend      string `json:"backend"`
	Model        string `json:"model"`
	Device       string `json:"device"`
	SampleRate   int    `json:"sample_rate"`
	Version      string `json:"version"`
	EmbeddingDim int    `json:"embedding_dim"`
}

type SpeakerVerificationResourceReady struct {
	Status         string `json:"status"`
	ModelLoaded    bool   `json:"model_loaded"`
	ProfileStoreOK bool   `json:"profile_store_ok"`
	TempDirOK      bool   `json:"temp_dir_ok"`
}

type SpeakerVerificationEnrollmentResponse struct {
	ProfileID              string  `json:"profile_id"`
	DisplayName            string  `json:"display_name"`
	EmbeddingDim           int     `json:"embedding_dim"`
	SampleRate             int     `json:"sample_rate"`
	EnrollmentAudioSeconds float64 `json:"enrollment_audio_seconds"`
	ModelName              string  `json:"model_name"`
	CreatedAt              string  `json:"created_at"`
}

type SpeakerVerificationResult struct {
	ProfileID    string  `json:"profile_id"`
	Matched      bool    `json:"matched"`
	Score        float64 `json:"score"`
	Threshold    float64 `json:"threshold"`
	DurationMs   float64 `json:"duration_ms"`
	Backend      string  `json:"backend"`
	Model        string  `json:"model"`
	AudioSeconds float64 `json:"audio_seconds"`
}

type SpeakerVerificationResourceClient struct {
	BaseURL string
	Client  *http.Client
}

func (c *SpeakerVerificationResourceClient) httpClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 5 * time.Second}
}

func (c *SpeakerVerificationResourceClient) endpoint(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + path
}

func (c *SpeakerVerificationResourceClient) getJSON(ctx context.Context, path string, out any) error {
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

func (c *SpeakerVerificationResourceClient) doJSON(req *http.Request, out any) error {
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

func (c *SpeakerVerificationResourceClient) postMultipart(
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

func (c *SpeakerVerificationResourceClient) Ready(ctx context.Context) (SpeakerVerificationResourceReady, error) {
	var out SpeakerVerificationResourceReady
	err := c.getJSON(ctx, "/ready", &out)
	return out, err
}

func (c *SpeakerVerificationResourceClient) Info(ctx context.Context) (SpeakerVerificationResourceInfo, error) {
	var out SpeakerVerificationResourceInfo
	err := c.getJSON(ctx, "/v1/info", &out)
	return out, err
}

func (c *SpeakerVerificationResourceClient) ListProfiles(ctx context.Context) (SpeakerVerificationProfileList, error) {
	var out SpeakerVerificationProfileList
	err := c.getJSON(ctx, "/v1/profiles", &out)
	return out, err
}

func (c *SpeakerVerificationResourceClient) Enroll(
	ctx context.Context,
	audio []byte,
	profileID string,
	displayName string,
	notes string,
) (SpeakerVerificationEnrollmentResponse, error) {
	var out SpeakerVerificationEnrollmentResponse
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

func (c *SpeakerVerificationResourceClient) Verify(
	ctx context.Context,
	audio []byte,
	profileID string,
	threshold float64,
) (SpeakerVerificationResult, error) {
	var out SpeakerVerificationResult
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

func (c *SpeakerVerificationResourceClient) DeleteProfile(ctx context.Context, profileID string) error {
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
