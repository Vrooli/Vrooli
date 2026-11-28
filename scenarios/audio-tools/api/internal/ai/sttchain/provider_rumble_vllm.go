package sttchain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/httpc"
)

const (
	rumbleModelID         = "Reson8/Rumble-2-Flash"
	rumbleMaxAudioBytes   = 10 << 20
	rumbleMaxResponseSize = 1 << 20
	rumbleMaxInFlight     = 1
	rumbleProbeTimeout    = 3 * time.Second
)

// RumbleVLLMBackend implements the upstream model card's vLLM OpenAI chat
// completions contract. It is intentionally only a BatchBackend: Rumble is
// batch ASR and does not provide the WebSocket streaming protocol used by
// Kyutai and Sherpa. The endpoint must be explicitly configured; this adapter
// neither starts vLLM nor downloads model weights.
type RumbleVLLMBackend struct {
	BaseURL     string
	ServedModel string
	Doer        httpc.Doer
	ProbeDoer   httpc.Doer
	Audio       *audioformat.Engine
	MaxAudio    int
	MaxResponse int64
	Languages   map[string]string // ISO-639-1 code -> upstream English language name.
	semaphore   chan struct{}
}

func NewRumbleVLLMBackend(baseURL string, doer httpc.Doer) *RumbleVLLMBackend {
	if doer == nil {
		doer = httpc.NoRedirectDoer(120 * time.Second)
	}
	return &RumbleVLLMBackend{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"), ServedModel: rumbleModelID,
		Doer: doer, ProbeDoer: httpc.NoRedirectDoer(rumbleProbeTimeout), Audio: audioformat.New(), MaxAudio: rumbleMaxAudioBytes,
		MaxResponse: rumbleMaxResponseSize, semaphore: make(chan struct{}, rumbleMaxInFlight),
		Languages: rumbleLanguageNames,
	}
}

func (b *RumbleVLLMBackend) Model() string {
	if b == nil || strings.TrimSpace(b.ServedModel) == "" {
		return rumbleModelID
	}
	return strings.TrimSpace(b.ServedModel)
}

// IsAvailable performs read-only probes against the vLLM server and requires
// that the configured model is actually served. It never reconciles or starts
// an external process.
func (b *RumbleVLLMBackend) IsAvailable(ctx context.Context) bool {
	if ctx == nil || ctx.Err() != nil || b == nil || b.apiURL("/v1/models") == "" {
		return false
	}
	probeDoer := b.ProbeDoer
	if probeDoer == nil {
		probeDoer = httpc.NoRedirectDoer(rumbleProbeTimeout)
	}
	healthReq, err := http.NewRequestWithContext(ctx, http.MethodGet, b.apiURLRoot()+"/health", nil)
	if err != nil {
		return false
	}
	healthResp, err := probeDoer.Do(healthReq)
	if err != nil {
		return false
	}
	closeErr := healthResp.Body.Close()
	if healthResp.StatusCode != http.StatusOK || closeErr != nil {
		return false
	}
	modelsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, b.apiURL("/v1/models"), nil)
	if err != nil {
		return false
	}
	modelsResp, err := probeDoer.Do(modelsReq)
	if err != nil {
		return false
	}
	defer modelsResp.Body.Close()
	if modelsResp.StatusCode != http.StatusOK {
		return false
	}
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(modelsResp.Body, rumbleMaxResponseSize))
	if decoder.Decode(&models) != nil {
		return false
	}
	for _, model := range models.Data {
		if model.ID == b.Model() {
			return true
		}
	}
	return false
}

// IsServiceable is safe for inventory/readiness checks: both calls are
// bounded, read-only probes and never start or reconcile vLLM.
func (b *RumbleVLLMBackend) IsServiceable(ctx context.Context) bool { return b.IsAvailable(ctx) }

func (b *RumbleVLLMBackend) Transcribe(ctx context.Context, req Request) (BatchResult, error) {
	if ctx == nil {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: context is nil")
	}
	// Check cancellation before audio conversion and base64 encoding. A ready
	// semaphore send can win a select against ctx.Done(), so the select alone
	// does not avoid expensive work for an already-canceled request.
	if err := ctx.Err(); err != nil {
		return BatchResult{}, err
	}
	if b == nil || b.Doer == nil || b.semaphore == nil || b.apiURL("/v1/chat/completions") == "" {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: vLLM endpoint is not configured")
	}
	if len(req.Audio) == 0 {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: audio is empty")
	}
	maxAudio := b.MaxAudio
	if maxAudio <= 0 {
		maxAudio = rumbleMaxAudioBytes
	}
	if len(req.Audio) > maxAudio {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: audio exceeds %d byte limit", maxAudio)
	}
	kwargs := map[string]any{}
	if language := strings.TrimSpace(req.Language); language != "" {
		name, ok := b.Languages[strings.ToLower(language)]
		if !ok {
			return BatchResult{}, fmt.Errorf("audio-tools/rumble: unsupported language code %q; configure its upstream language name", language)
		}
		kwargs["language"] = name
	}
	select {
	case b.semaphore <- struct{}{}:
		defer func() { <-b.semaphore }()
	case <-ctx.Done():
		return BatchResult{}, ctx.Err()
	}

	audioData, mimeType, err := b.audioData(req.Audio, req.Format)
	if err != nil {
		return BatchResult{}, err
	}
	content := []struct {
		Type     string `json:"type"`
		AudioURL struct {
			URL string `json:"url"`
		} `json:"audio_url"`
	}{{Type: "audio_url"}}
	content[0].AudioURL.URL = "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(audioData)
	extraBody := map[string]any{"chat_template_kwargs": kwargs}
	body := struct {
		Model       string `json:"model"`
		Messages    []any  `json:"messages"`
		Temperature int    `json:"temperature"`
		ExtraBody   any    `json:"extra_body"`
	}{
		Model: b.Model(), Temperature: 0, ExtraBody: extraBody,
		Messages: []any{map[string]any{"role": "user", "content": content}},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: encode request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, b.apiURL("/v1/chat/completions"), bytes.NewReader(raw))
	if err != nil {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := b.Doer.Do(httpReq)
	if err != nil {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: vLLM request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: vLLM returned HTTP %d", resp.StatusCode)
	}
	maxResponse := b.MaxResponse
	if maxResponse <= 0 {
		maxResponse = rumbleMaxResponseSize
	}
	responseBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: read response: %w", err)
	}
	if int64(len(responseBytes)) > maxResponse {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: response exceeds %d byte limit", maxResponse)
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: decode response: %w", err)
	}
	if len(response.Choices) == 0 {
		return BatchResult{}, fmt.Errorf("audio-tools/rumble: response has no choices")
	}
	text, detected := parseRumbleTranscription(response.Choices[0].Message.Content, b.Languages)
	return BatchResult{Text: text, DetectedLanguage: detected}, nil
}

func (b *RumbleVLLMBackend) audioData(raw []byte, format string) ([]byte, string, error) {
	codec := audioformat.CodecFromString(format)
	if codec == audioformat.CodecUnknown {
		head := raw
		if len(head) > 64 {
			head = head[:64]
		}
		var err error
		codec, err = audioformat.Detect(audioformat.CodecUnknown, head)
		if err != nil {
			return nil, "", err
		}
	}
	engine := b.Audio
	if engine == nil {
		engine = audioformat.New()
	}
	prepared, _, err := engine.PrepareForWhisper(codec, raw)
	if err != nil {
		return nil, "", err
	}
	mimeType, err := rumbleMIME(codec)
	if err != nil {
		return nil, "", err
	}
	return prepared, mimeType, nil
}

func rumbleMIME(codec audioformat.Codec) (string, error) {
	switch codec {
	case audioformat.CodecPCMS16LE, audioformat.CodecWAV:
		return "audio/wav", nil
	case audioformat.CodecMP3:
		return "audio/mpeg", nil
	case audioformat.CodecFLAC:
		return "audio/flac", nil
	case audioformat.CodecOGG, audioformat.CodecOpus:
		return "audio/ogg", nil
	case audioformat.CodecWebM:
		return "audio/webm", nil
	case audioformat.CodecAAC:
		return "audio/aac", nil
	default:
		return "", fmt.Errorf("audio-tools/rumble: unsupported audio codec %q", codec.String())
	}
}

func (b *RumbleVLLMBackend) apiURL(path string) string {
	base := b.apiBaseURL()
	if base == "" {
		return ""
	}
	return base + strings.TrimPrefix(path, "/v1")
}

func (b *RumbleVLLMBackend) apiURLRoot() string {
	if b == nil {
		return ""
	}
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(b.BaseURL), "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return ""
	}
	u.Path = strings.TrimSuffix(strings.TrimRight(u.Path, "/"), "/v1")
	return strings.TrimRight(u.String(), "/")
}

func (b *RumbleVLLMBackend) apiBaseURL() string {
	if b == nil {
		return ""
	}
	u, err := url.Parse(strings.TrimRight(strings.TrimSpace(b.BaseURL), "/"))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return ""
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/v1") {
		u.Path += "/v1"
	}
	return strings.TrimRight(u.String(), "/")
}

func parseRumbleTranscription(value string, names map[string]string) (text, language string) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(value), "language ") {
		return value, ""
	}
	remainder := strings.TrimSpace(value[len("language "):])
	marker := strings.Index(remainder, "<asr_text>")
	if marker < 0 {
		return value, ""
	}
	name := strings.TrimSpace(remainder[:marker])
	text = strings.TrimSpace(remainder[marker+len("<asr_text>"):])
	for code, candidate := range names {
		if strings.EqualFold(candidate, name) {
			return text, code
		}
	}
	return text, strings.ToLower(name)
}

// Keep the API's existing ISO-639-1 language contract while translating to
// the human-readable names required by Rumble's vLLM chat template.
var rumbleLanguageNames = map[string]string{
	"ar": "Arabic", "bg": "Bulgarian", "bn": "Bengali", "ca": "Catalan", "cs": "Czech",
	"da": "Danish", "de": "German", "el": "Greek", "en": "English", "es": "Spanish",
	"et": "Estonian", "fa": "Persian", "fi": "Finnish", "fr": "French", "gu": "Gujarati",
	"he": "Hebrew", "hi": "Hindi", "hr": "Croatian", "hu": "Hungarian", "id": "Indonesian",
	"it": "Italian", "ja": "Japanese", "kn": "Kannada", "ko": "Korean", "lt": "Lithuanian",
	"lv": "Latvian", "ml": "Malayalam", "mr": "Marathi", "ms": "Malay", "nl": "Dutch",
	"no": "Norwegian", "pa": "Punjabi", "pl": "Polish", "pt": "Portuguese", "ro": "Romanian",
	"ru": "Russian", "sk": "Slovak", "sl": "Slovenian", "sr": "Serbian", "sv": "Swedish",
	"sw": "Swahili", "ta": "Tamil", "te": "Telugu", "th": "Thai", "tr": "Turkish",
	"uk": "Ukrainian", "ur": "Urdu", "vi": "Vietnamese", "zh": "Chinese",
}

var _ BatchBackend = (*RumbleVLLMBackend)(nil)
