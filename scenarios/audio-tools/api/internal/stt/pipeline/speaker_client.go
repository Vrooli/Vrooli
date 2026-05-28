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

	"audio-tools/internal/httpc"
)

type SpeakerProfile struct {
	ID                 string  `json:"id"`
	DisplayName        string  `json:"display_name"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	ModelName          string  `json:"model_name"`
	EmbeddingDim       int     `json:"embedding_dim"`
	SampleRate         int     `json:"sample_rate"`
	ClipCount          int     `json:"clip_count"`
	TotalVoicedSeconds float64 `json:"total_voiced_seconds"`
	Notes              string  `json:"notes"`
}

type SpeakerProfileList struct {
	Profiles []SpeakerProfile `json:"profiles"`
	Count    int              `json:"count"`
}

// SpeakerProfileClip is one labeled enrollment clip's metadata (the raw
// embedding never crosses the wire).
type SpeakerProfileClip struct {
	ClipID               string  `json:"clip_id"`
	Label                string  `json:"label"`
	VoicedSeconds        float64 `json:"voiced_seconds"`
	AudioSeconds         float64 `json:"audio_seconds"`
	SelfConsistencyScore float64 `json:"self_consistency_score"`
	VadModel             string  `json:"vad_model"`
	CreatedAt            string  `json:"created_at"`
	EmbeddingDim         int     `json:"embedding_dim"`
}

// SpeakerProfileDetail is the GET /v1/profiles/{id} response: profile metadata
// plus its clip list.
type SpeakerProfileDetail struct {
	SpeakerProfile
	Clips []SpeakerProfileClip `json:"clips"`
}

type SpeakerProfileClipList struct {
	ProfileID string               `json:"profile_id"`
	Clips     []SpeakerProfileClip `json:"clips"`
	Count     int                  `json:"count"`
}

// SpeakerClipDeleteResult is the DELETE .../clips/{clip_id} response. When the
// deleted clip was the profile's last, DeletedProfile is true and the profile
// is gone.
type SpeakerClipDeleteResult struct {
	ProfileID          string  `json:"profile_id"`
	ClipID             string  `json:"clip_id"`
	DeletedProfile     bool    `json:"deleted_profile"`
	ClipCount          int     `json:"clip_count"`
	TotalVoicedSeconds float64 `json:"total_voiced_seconds"`
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

// SpeakerEnrollmentResponse is the response to appending one enrollment clip.
//
// In v0.4 the resource adds enrollment-time self-consistency diagnostics: the
// new clip's max-cosine against the strongest existing clip in the same
// profile, plus a warning flag when that score is below the resource's
// self-consistency threshold. The clip is stored regardless — the warning is
// informational, exposed so callers can prompt the user to re-record in
// matching conditions.
type SpeakerEnrollmentResponse struct {
	ProfileID                    string  `json:"profile_id"`
	ClipID                       string  `json:"clip_id"`
	Label                        string  `json:"label"`
	VoicedSeconds                float64 `json:"voiced_seconds"`
	AudioSeconds                 float64 `json:"audio_seconds"`
	ClipCount                    int     `json:"clip_count"`
	TotalVoicedSeconds           float64 `json:"total_voiced_seconds"`
	EmbeddingDim                 int     `json:"embedding_dim"`
	SampleRate                   int     `json:"sample_rate"`
	ModelName                    string  `json:"model_name"`
	VadModel                     string  `json:"vad_model"`
	SelfConsistencyScore         float64 `json:"self_consistency_score"`
	SelfConsistencyThreshold     float64 `json:"self_consistency_threshold"`
	SelfConsistencyWarning       bool    `json:"self_consistency_warning"`
	SelfConsistencyBestClipID    string  `json:"self_consistency_best_clip_id"`
	SelfConsistencyBestClipLabel string  `json:"self_consistency_best_clip_label"`
	CreatedAt                    string  `json:"created_at"`
}

// SpeakerVerifyResult is the parsed response from the resource's /v1/verify
// endpoint. v0.4 dropped centroid aggregation: scoring is now max-over-clips
// only, so ScoreAgg reads "max" and best_clip_id / best_clip_score expose the
// concrete winning clip so callers can surface "which enrollment matched".
type SpeakerVerifyResult struct {
	ProfileID     string  `json:"profile_id"`
	Matched       bool    `json:"matched"`
	Score         float64 `json:"score"`
	Threshold     float64 `json:"threshold"`
	Sufficient    bool    `json:"sufficient"`
	VoicedSeconds float64 `json:"voiced_seconds"`
	DurationMs    float64 `json:"duration_ms"`
	Backend       string  `json:"backend"`
	Model         string  `json:"model"`
	AudioSeconds  float64 `json:"audio_seconds"`
	ScoreAgg      string  `json:"score_agg"`
	VadModel      string  `json:"vad_model"`
	NClips        int     `json:"n_clips"`
	BestClipLabel string  `json:"best_clip_label"`
	BestClipID    string  `json:"best_clip_id"`
	BestClipScore float64 `json:"best_clip_score"`
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
	Doer    httpc.Doer
}

func (c *SpeakerClient) endpoint(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + path
}

func (c *SpeakerClient) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(path), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := c.Doer.Do(req)
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
	resp, err := c.Doer.Do(req)
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

// Enroll appends one labeled enrollment clip to a profile (creating it if new),
// uploading audio the resource decodes by content sniffing (filename is
// cosmetic) and embeds over its voiced span. Callers normalize to canonical-PCM
// WAV before calling so the enrollment embedding matches the verification
// embedding; filename reflects that.
func (c *SpeakerClient) Enroll(
	ctx context.Context,
	audio []byte,
	profileID, displayName, notes, label, filename string,
) (SpeakerEnrollmentResponse, error) {
	var out SpeakerEnrollmentResponse
	err := c.postMultipart(
		ctx,
		"/v1/profiles",
		map[string]string{
			"profile_id":   profileID,
			"display_name": displayName,
			"notes":        notes,
			"label":        label,
		},
		"audio",
		filename,
		audio,
		&out,
	)
	return out, err
}

// GetProfile returns one profile's metadata + clip list.
func (c *SpeakerClient) GetProfile(ctx context.Context, profileID string) (SpeakerProfileDetail, error) {
	var out SpeakerProfileDetail
	err := c.getJSON(ctx, "/v1/profiles/"+profileID, &out)
	return out, err
}

// ListClips returns the enrollment clips of a profile.
func (c *SpeakerClient) ListClips(ctx context.Context, profileID string) (SpeakerProfileClipList, error) {
	var out SpeakerProfileClipList
	err := c.getJSON(ctx, "/v1/profiles/"+profileID+"/clips", &out)
	return out, err
}

// DeleteClip removes one clip from a profile and recomputes its centroid;
// deleting the last clip deletes the profile (DeletedProfile=true).
func (c *SpeakerClient) DeleteClip(ctx context.Context, profileID, clipID string) (SpeakerClipDeleteResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint("/v1/profiles/"+profileID+"/clips/"+clipID), nil)
	if err != nil {
		return SpeakerClipDeleteResult{}, fmt.Errorf("create request: %w", err)
	}
	var out SpeakerClipDeleteResult
	err = c.doJSON(req, &out)
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

	resp, err := c.Doer.Do(req)
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
	resp, err := c.Doer.Do(req)
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
