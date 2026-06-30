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
// Greenfield: this provider selects NO concrete model itself. Each request
// resolves an OpenRouter policy ROLE (e.g. image.generate.default) through
// `resource-openrouter policy resolve`, which returns the concrete model slug,
// endpoint family, and bounded request defaults. Generation rides OpenRouter's
// dedicated image API (`POST /api/v1/images`): the prompt (and, for edits,
// input_references) go up and the response carries the image as base64 in
// data[].b64_json.
const openRouterImagesURL = "https://openrouter.ai/api/v1/images"

// resolvedImageRole is the subset of `resource-openrouter policy resolve --json`
// the image provider needs to build an /images request.
type resolvedImageRole struct {
	Model           string `json:"model"`
	Endpoint        string `json:"endpoint"`
	RequestDefaults struct {
		OutputFormat string `json:"output_format"`
		Background   string `json:"background"`
		AspectRatio  string `json:"aspect_ratio"`
		Resolution   string `json:"resolution"`
		Size         string `json:"size"`
		Quality      string `json:"quality"`
	} `json:"request_defaults"`
}

// openRouterProvider implements backends.Provider over the OpenRouter image API.
type openRouterProvider struct {
	httpClient  *http.Client
	baseURL     string // /images endpoint (overridable in tests)
	resolveKey  func(ctx context.Context) string
	resolveRole func(ctx context.Context, role string) (resolvedImageRole, error)
}

// newOpenRouterProvider builds the production provider. The model is resolved
// per-request from an OpenRouter policy role; the key is resolved lazily (env →
// vault) so the provider can be registered unconditionally and simply report
// unavailable when no key is configured.
func newOpenRouterProvider() *openRouterProvider {
	return &openRouterProvider{
		httpClient:  &http.Client{Timeout: 4 * time.Minute},
		baseURL:     openRouterImagesURL,
		resolveKey:  resolveOpenRouterKey,
		resolveRole: resolveOpenRouterRole,
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
			Detail:    "OpenRouter BYOK cloud (role-resolved image model)",
			Provision: "set OPENROUTER_API_KEY (or store it in resource-vault)",
		}
	}
	return backends.Availability{
		Available: false,
		Detail:    "OpenRouter API key not configured",
		Provision: "set OPENROUTER_API_KEY (or store it in resource-vault) to enable BYOK cloud image ops",
	}
}

// roleForRequest maps an op (and optional explicit intent) to an OpenRouter
// image policy role. A caller may carry intent through Params["openrouter_role"]
// (e.g. brand-manager requesting image.generate.logo or image.edit.identity);
// otherwise the op maps to the default generate/edit role.
func roleForRequest(req backends.Request) string {
	if r := strings.TrimSpace(req.Params["openrouter_role"]); r != "" {
		return r
	}
	if req.Operation == "text_to_image" {
		return "image.generate.default"
	}
	return "image.edit.default"
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

	role := roleForRequest(req)
	resolved, err := p.resolveRole(ctx, role)
	if err != nil {
		return backends.Result{}, err
	}

	reqBody := map[string]any{
		"model":  resolved.Model,
		"prompt": prompt,
	}
	if v := resolved.RequestDefaults.OutputFormat; v != "" {
		reqBody["output_format"] = v
	}
	if v := resolved.RequestDefaults.Background; v != "" {
		reqBody["background"] = v
	}
	if v := resolved.RequestDefaults.AspectRatio; v != "" {
		reqBody["aspect_ratio"] = v
	}
	if v := resolved.RequestDefaults.Resolution; v != "" {
		reqBody["resolution"] = v
	}
	if v := resolved.RequestDefaults.Size; v != "" {
		reqBody["size"] = v
	}
	if v := resolved.RequestDefaults.Quality; v != "" {
		reqBody["quality"] = v
	}

	// Edits / img2img carry the source image as a base64 data URL reference.
	if req.Operation != "text_to_image" {
		in, err := technique.Input0(req)
		if err != nil {
			return backends.Result{}, fmt.Errorf("ai: openrouter %s: %w", req.Operation, err)
		}
		data, err := os.ReadFile(in)
		if err != nil {
			return backends.Result{}, fmt.Errorf("ai: openrouter read input: %w", err)
		}
		reqBody["input_references"] = []any{map[string]any{
			"type":      "image_url",
			"image_url": map[string]string{"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)},
		}}
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: openrouter marshal request: %w", err)
	}

	endpoint := p.baseURL
	if endpoint == "" {
		endpoint = openRouterImagesURL
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

	img, mediaType, err := decodeOpenRouterImage(out)
	if err != nil {
		return backends.Result{}, err
	}
	if err := os.WriteFile(req.Output.LocalPath, img, 0o644); err != nil {
		return backends.Result{}, fmt.Errorf("ai: openrouter write output: %w", err)
	}
	meta := map[string]string{"backend": models.BackendOpenRouter, "model": resolved.Model, "role": role}
	if mediaType != "" {
		meta["media_type"] = mediaType
	}
	return backends.Result{
		OutputRef: req.Output.LocalPath,
		Tier:      backends.TierBYOK,
		Meta:      meta,
	}, nil
}

// openRouterImagesResponse is the subset of the /api/v1/images response that
// carries a generated image. Images are returned base64-encoded in
// data[].b64_json; vector models additionally set media_type (e.g.
// image/svg+xml).
type openRouterImagesResponse struct {
	Data []struct {
		B64JSON   string `json:"b64_json"`
		MediaType string `json:"media_type"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// decodeOpenRouterImage extracts the generated image bytes and media type from
// an /images response.
func decodeOpenRouterImage(body []byte) ([]byte, string, error) {
	var parsed openRouterImagesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, "", fmt.Errorf("ai: openrouter decode response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return nil, "", fmt.Errorf("ai: openrouter error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 || strings.TrimSpace(parsed.Data[0].B64JSON) == "" {
		return nil, "", fmt.Errorf("ai: openrouter returned no image (the model may not support image output)")
	}
	img, err := base64.StdEncoding.DecodeString(parsed.Data[0].B64JSON)
	if err != nil {
		return nil, "", fmt.Errorf("ai: openrouter decode image data: %w", err)
	}
	if len(img) == 0 {
		return nil, "", fmt.Errorf("ai: openrouter image is empty")
	}
	return img, strings.TrimSpace(parsed.Data[0].MediaType), nil
}

// resolveOpenRouterRole shells out to the OpenRouter resource policy authority to
// turn a logical role into a concrete model slug + request defaults. resource-
// openrouter is the single source of truth; this provider never reads the policy
// file directly.
func resolveOpenRouterRole(ctx context.Context, role string) (resolvedImageRole, error) {
	out, err := exec.CommandContext(ctx, "resource-openrouter", "policy", "resolve", "--role", role, "--json").Output()
	if err != nil {
		return resolvedImageRole{}, fmt.Errorf("ai: openrouter policy resolve %q: %w", role, err)
	}
	var resolved resolvedImageRole
	if err := json.Unmarshal(out, &resolved); err != nil {
		return resolvedImageRole{}, fmt.Errorf("ai: openrouter decode policy resolve %q: %w", role, err)
	}
	if strings.TrimSpace(resolved.Model) == "" {
		return resolvedImageRole{}, fmt.Errorf("ai: openrouter policy resolve %q returned no model", role)
	}
	return resolved, nil
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
