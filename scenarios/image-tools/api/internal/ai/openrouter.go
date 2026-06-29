package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/technique"
)

// OpenRouter image generation/editing — the BYOK cloud provider (the last tier
// in the selection ladder). image-tools is local-first: this provider is chosen
// only when the caller sets allow_byok AND no local backend is available, or
// when explicitly forced via model_override. It holds no local weights; the
// model lives on OpenRouter, so the registry model (backend "openrouter") is
// always "installed" and runnability is gated by Available() (a usable key).
//
// Image generation rides OpenRouter's chat-completions surface with the image
// output modality — the only OpenRouter image path that actually works (there is
// no DALL·E-style /images/generations endpoint). A prompt (and, for edits, an
// input image) goes up as a chat message; the response carries the generated
// image as a base64 data URL.
const (
	openRouterChatURL           = "https://openrouter.ai/api/v1/chat/completions"
	defaultOpenRouterImageModel = "google/gemini-2.5-flash-image-preview"
)

// openRouterProvider implements backends.Provider over the OpenRouter API.
type openRouterProvider struct {
	httpClient *http.Client
	baseURL    string // chat-completions endpoint (overridable in tests)
	model      string // OpenRouter image model slug
	resolveKey func(ctx context.Context) string
}

// newOpenRouterProvider builds the production provider. The image model slug is
// OPENROUTER_IMAGE_MODEL or a sensible default; the key is resolved lazily (env
// → vault) so the provider can be registered unconditionally and simply report
// unavailable when no key is configured.
func newOpenRouterProvider() *openRouterProvider {
	model := sanitizeModelSlug(os.Getenv("OPENROUTER_IMAGE_MODEL"))
	if model == "" {
		model = defaultOpenRouterImageModel
	}
	return &openRouterProvider{
		httpClient: &http.Client{Timeout: 4 * time.Minute},
		baseURL:    openRouterChatURL,
		model:      model,
		resolveKey: resolveOpenRouterKey,
	}
}

func (p *openRouterProvider) Name() string { return models.BackendOpenRouter }

func (p *openRouterProvider) Operations() []string {
	return []string{"text_to_image", "image_to_image", "edit_instruct"}
}

func (p *openRouterProvider) Standalone() bool { return true }
func (p *openRouterProvider) IsCloud() bool    { return true }
func (p *openRouterProvider) GPUCapable() bool { return false }

func (p *openRouterProvider) Available(ctx context.Context) bool {
	return openRouterKeyUsable(p.resolveKey(ctx))
}

func (p *openRouterProvider) Availability(ctx context.Context) backends.Availability {
	if p.Available(ctx) {
		return backends.Availability{
			Available: true,
			Detail:    fmt.Sprintf("OpenRouter BYOK cloud (%s)", p.model),
			Provision: "set OPENROUTER_API_KEY (or store it in resource-vault)",
		}
	}
	return backends.Availability{
		Available: false,
		Detail:    "OpenRouter API key not configured",
		Provision: "set OPENROUTER_API_KEY (or store it in resource-vault) to enable BYOK cloud image ops",
	}
}

func (p *openRouterProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: openrouter backend requires a local output path")
	}
	key := p.resolveKey(ctx)
	if !openRouterKeyUsable(key) {
		return backends.Result{}, fmt.Errorf("ai: openrouter backend unavailable: OPENROUTER_API_KEY not configured")
	}
	prompt := strings.TrimSpace(req.Params["prompt"])
	if prompt == "" {
		return backends.Result{}, fmt.Errorf("ai: openrouter image generation requires a prompt")
	}

	content := []any{map[string]any{"type": "text", "text": prompt}}
	// Edits / img2img carry the source image as a data URL alongside the prompt.
	if req.Operation != "text_to_image" {
		in, err := technique.Input0(req)
		if err != nil {
			return backends.Result{}, fmt.Errorf("ai: openrouter %s: %w", req.Operation, err)
		}
		data, err := os.ReadFile(in)
		if err != nil {
			return backends.Result{}, fmt.Errorf("ai: openrouter read input: %w", err)
		}
		content = append(content, map[string]any{
			"type":      "image_url",
			"image_url": map[string]string{"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)},
		})
	}

	reqBody := map[string]any{
		"model":      p.model,
		"modalities": []string{"image", "text"},
		"messages":   []any{map[string]any{"role": "user", "content": content}},
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: openrouter marshal request: %w", err)
	}

	endpoint := p.baseURL
	if endpoint == "" {
		endpoint = openRouterChatURL
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return backends.Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("HTTP-Referer", "https://vrooli.com")
	httpReq.Header.Set("X-Title", "Vrooli image-tools")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: openrouter request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return backends.Result{}, fmt.Errorf("ai: openrouter returned %d: %s", resp.StatusCode, strings.TrimSpace(string(out)))
	}

	img, err := decodeOpenRouterImage(out)
	if err != nil {
		return backends.Result{}, err
	}
	if err := os.WriteFile(req.Output.LocalPath, img, 0o644); err != nil {
		return backends.Result{}, fmt.Errorf("ai: openrouter write output: %w", err)
	}
	return backends.Result{
		OutputRef: req.Output.LocalPath,
		Tier:      backends.TierBYOK,
		Meta:      map[string]string{"backend": models.BackendOpenRouter, "model": p.model},
	}, nil
}

// openRouterChatResponse is the subset of the chat-completions response that
// carries a generated image. OpenRouter returns image outputs in
// message.images[].image_url.url as a base64 data URL.
type openRouterChatResponse struct {
	Choices []struct {
		Message struct {
			Images []struct {
				ImageURL struct {
					URL string `json:"url"`
				} `json:"image_url"`
			} `json:"images"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// decodeOpenRouterImage extracts the generated image bytes from a chat response.
func decodeOpenRouterImage(body []byte) ([]byte, error) {
	var parsed openRouterChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("ai: openrouter decode response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, fmt.Errorf("ai: openrouter error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || len(parsed.Choices[0].Message.Images) == 0 {
		return nil, fmt.Errorf("ai: openrouter returned no image (the model may not support image output)")
	}
	url := parsed.Choices[0].Message.Images[0].ImageURL.URL
	idx := strings.Index(url, ",")
	if !strings.HasPrefix(url, "data:") || idx < 0 {
		return nil, fmt.Errorf("ai: openrouter image is not a data URL")
	}
	img, err := base64.StdEncoding.DecodeString(url[idx+1:])
	if err != nil {
		return nil, fmt.Errorf("ai: openrouter decode image data: %w", err)
	}
	if len(img) == 0 {
		return nil, fmt.Errorf("ai: openrouter image is empty")
	}
	return img, nil
}

// sanitizeModelSlug trims an env-provided model slug and rejects an
// unsubstituted lifecycle placeholder (e.g. "${OPENROUTER_IMAGE_MODEL}" when the
// var is unset) so the provider falls back to the default rather than POSTing a
// bogus model id.
func sanitizeModelSlug(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" || strings.HasPrefix(v, "${") {
		return ""
	}
	return v
}

// openRouterKeyUsable reports whether a resolved key looks like a real OpenRouter
// credential (sk-or- prefix + length floor) rather than a placeholder/stub.
func openRouterKeyUsable(key string) bool {
	key = strings.TrimSpace(key)
	return strings.HasPrefix(key, "sk-or-") && len(key) >= 40
}

// resolveOpenRouterKey resolves the API key from the environment first (injected
// by the scenario lifecycle from the vault SSOT), then from a direct vault export
// as a best-effort fallback. Returns "" when no usable key is found.
func resolveOpenRouterKey(ctx context.Context) string {
	if key := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")); openRouterKeyUsable(key) {
		return key
	}
	for _, scope := range []string{"openrouter", "opencode"} {
		out, err := exec.CommandContext(ctx, "resource-vault", "secrets", "export", scope).Output()
		if err != nil {
			continue
		}
		if key := parseExportedKey(string(out), "OPENROUTER_API_KEY"); openRouterKeyUsable(key) {
			return key
		}
	}
	return ""
}

// parseExportedKey extracts NAME's value from `resource-vault secrets export`
// shell-export lines like `export NAME=value` or `NAME="value"`.
func parseExportedKey(out, name string) string {
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 || line[:eq] != name {
			continue
		}
		return strings.Trim(strings.TrimSpace(line[eq+1:]), `"'`)
	}
	return ""
}
