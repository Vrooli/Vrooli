package imageengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/vrooli/api-core/discovery"
)

// Executor is the only raster seam Backdrop Studio depends on. Implementations
// must send every treatment to image-tools; Backdrop Studio deliberately has no
// local pixel implementation.
type Executor interface {
	// params carries per-style overrides keyed by op name, merged over the
	// palette-derived defaults. A style that names an op without parameters
	// still gets a sensible default; a style that wants a specific screen can
	// state it.
	Apply(ctx context.Context, input []byte, treatments []string, params map[string]string, palette map[string]string) ([]byte, error)
}

// GenerationRequest is the role/profile-based inference contract. Concrete
// model identity and routing stay owned by image-tools; Backdrop Studio only
// declares the capability it needs and, for guided styles, a conditioning
// image produced by its scaffold domain.
type GenerationRequest struct {
	Prompt, Negative, Role, Profile, Conditioner string
	Seed                                         int64
	Conditioning                                 []byte
}

type Generator interface {
	Generate(context.Context, GenerationRequest) ([]byte, error)
}

type submitGenerationResponse struct {
	JobID string `json:"jobId"`
}
type waitGenerationResponse struct {
	Job struct {
		State     string `json:"state"`
		ResultRef string `json:"resultRef"`
		Error     string `json:"error"`
	} `json:"job"`
}

// Client calls image-tools' synchronous deterministic operation edge. The
// resolver is injectable so unit tests can exercise the transport without
// depending on a running scenario.
type Client struct {
	HTTPClient *http.Client
	Resolve    func(context.Context) (string, error)
}

func NewClient() *Client {
	return &Client{
		HTTPClient: &http.Client{},
		Resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "image-tools")
		},
	}
}

// Apply runs the chain, merging per-style overrides over the palette-derived
// defaults. Without overrides every style naming "halftone" rendered the same
// screen at the same line frequency, so the catalog could name an art direction
// but never express one.
func (c *Client) Apply(ctx context.Context, input []byte, treatments []string, overrides map[string]string, palette map[string]string) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("image-tools: input image is empty")
	}
	if len(treatments) == 0 {
		return nil, fmt.Errorf("image-tools: treatment chain is empty")
	}
	resolve := c.Resolve
	if resolve == nil {
		return nil, fmt.Errorf("image-tools: URL resolver is not configured")
	}
	baseURL, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("image-tools: resolve scenario: %w", err)
	}
	baseURL = strings.TrimRight(baseURL, "/")
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	result := append([]byte(nil), input...)
	for _, treatment := range treatments {
		treatment = strings.TrimSpace(treatment)
		if treatment == "" || strings.HasPrefix(treatment, "$brand.") {
			return nil, fmt.Errorf("image-tools: invalid treatment %q", treatment)
		}
		result, err = run(ctx, httpClient, baseURL, treatment, result, mergedParams(treatment, overrides[treatment], palette))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Generate deliberately fails closed until image-tools exposes a synchronous
// role/profile inference seam. The production client does not invent a model,
// call a provider directly, or silently substitute a procedural image. Tests
// and future image-tools adapters implement Generator to prove the guided and
// synthesized paths without coupling this scenario to a model backend.
func (c *Client) Generate(ctx context.Context, req GenerationRequest) ([]byte, error) {
	if strings.TrimSpace(req.Role) == "" || strings.TrimSpace(req.Profile) == "" {
		return nil, fmt.Errorf("image-tools inference capability unavailable: role and profile are required")
	}
	resolve := c.Resolve
	if resolve == nil {
		return nil, fmt.Errorf("image-tools inference capability unavailable: URL resolver is not configured")
	}
	base, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("image-tools inference capability unavailable: %w", err)
	}
	operation := "text_to_image"
	if len(req.Conditioning) > 0 {
		operation = "image_to_image"
	}
	params := map[string]any{"prompt": req.Prompt, "negativePrompt": req.Negative, "seed": req.Seed, "width": 320, "height": 180, "qualityPolicy": "quality", "fallbackPolicy": "any", "allowByok": true}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if len(req.Conditioning) > 0 {
		part, e := mw.CreateFormFile("file", "scaffold.png")
		if e != nil {
			return nil, e
		}
		if _, e = part.Write(req.Conditioning); e != nil {
			return nil, e
		}
	}
	raw, _ := json.Marshal(params)
	if err := mw.WriteField("params", string(raw)); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/v1/ai/"+operation, &body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", mw.FormDataContentType())
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("image-tools inference %s: %w", operation, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
		return nil, fmt.Errorf("image-tools inference capability unavailable: %s: %s", response.Status, strings.TrimSpace(string(detail)))
	}
	var submitted submitGenerationResponse
	if err := json.NewDecoder(response.Body).Decode(&submitted); err != nil || submitted.JobID == "" {
		return nil, fmt.Errorf("image-tools inference returned no job id")
	}
	waitBody, _ := json.Marshal(map[string]string{"id": submitted.JobID})
	waitReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/vrooli.image_tools.v1.jobs.JobsService/WaitJob", bytes.NewReader(waitBody))
	if err != nil {
		return nil, err
	}
	waitReq.Header.Set("Content-Type", "application/json")
	waitResp, err := client.Do(waitReq)
	if err != nil {
		return nil, fmt.Errorf("image-tools inference wait: %w", err)
	}
	defer waitResp.Body.Close()
	if waitResp.StatusCode < 200 || waitResp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(waitResp.Body, 8<<10))
		return nil, fmt.Errorf("image-tools inference wait failed: %s", strings.TrimSpace(string(detail)))
	}
	var waited waitGenerationResponse
	if err := json.NewDecoder(waitResp.Body).Decode(&waited); err != nil {
		return nil, fmt.Errorf("image-tools inference wait response: %w", err)
	}
	if waited.Job.Error != "" {
		return nil, fmt.Errorf("image-tools inference failed: %s", waited.Job.Error)
	}
	if waited.Job.ResultRef == "" {
		return nil, fmt.Errorf("image-tools inference completed without a result reference")
	}
	resultURL := strings.TrimRight(base, "/") + "/api/v1/blobs/" + strings.TrimLeft(waited.Job.ResultRef, "/")
	resultResp, err := client.Get(resultURL)
	if err != nil {
		return nil, fmt.Errorf("image-tools inference result: %w", err)
	}
	defer resultResp.Body.Close()
	if resultResp.StatusCode < 200 || resultResp.StatusCode >= 300 {
		return nil, fmt.Errorf("image-tools inference result returned %s", resultResp.Status)
	}
	output, err := io.ReadAll(resultResp.Body)
	if err != nil {
		return nil, err
	}
	if len(output) == 0 {
		return nil, fmt.Errorf("image-tools inference returned an empty image")
	}
	return output, nil
}

func run(ctx context.Context, client *http.Client, baseURL, operation string, input []byte, params string) ([]byte, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "backdrop.png")
	if err != nil {
		return nil, fmt.Errorf("image-tools: create multipart image: %w", err)
	}
	if _, err := part.Write(input); err != nil {
		return nil, fmt.Errorf("image-tools: write multipart image: %w", err)
	}
	if err := writer.WriteField("params", params); err != nil {
		return nil, fmt.Errorf("image-tools: write params: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("image-tools: close multipart request: %w", err)
	}
	u := baseURL + "/api/v1/ops/" + url.PathEscape(operation) + "?output=bytes"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &body)
	if err != nil {
		return nil, fmt.Errorf("image-tools: create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("image-tools: execute %s: %w", operation, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return nil, fmt.Errorf("image-tools: %s returned %s: %s", operation, resp.Status, strings.TrimSpace(string(detail)))
	}
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("image-tools: read %s result: %w", operation, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("image-tools: %s returned an empty image", operation)
	}
	return out, nil
}

// ResolveParams is the exported form of the merge: it produces the exact
// parameter JSON that would be sent to image-tools for one operation, with
// $brand.* slots bound against the supplied palette. Callers use it to assert
// on what will actually go over the wire rather than on the catalog's stored
// intent.
func ResolveParams(operation, override string, palette map[string]string) string {
	return mergedParams(operation, override, palette)
}

// mergedParams overlays a style's op parameters onto the palette-derived
// defaults. Overrides may name "$brand.*" slots, which resolve against the
// active brand here rather than being baked into the catalog — the same
// mechanism that lets one style render correctly for several brands.
func mergedParams(operation, override string, palette map[string]string) string {
	base := paramsFor(operation, palette)
	if strings.TrimSpace(override) == "" {
		return base
	}
	var baseObj map[string]map[string]any
	if err := json.Unmarshal([]byte(base), &baseObj); err != nil || baseObj[operation] == nil {
		baseObj = map[string]map[string]any{operation: {}}
	}
	var over map[string]any
	if err := json.Unmarshal([]byte(override), &over); err != nil {
		// Unreachable for anything stored through the catalog: ValidateChain
		// rejects malformed parameters at write time, on both CreateStyle and
		// ImportStylePack. This is the defence-in-depth branch for parameters
		// that reached the engine some other way, and it ships the default
		// rather than failing the render. That is a deliberate trade — the
		// alternative is a render that dies on a bad character — and it is safe
		// only because the write path refuses to store this shape.
		return base
	}
	for k, v := range over {
		if sv, ok := v.(string); ok && strings.HasPrefix(sv, "$brand.") {
			if resolved, ok := palette[sv]; ok && resolved != "" {
				baseObj[operation][k] = resolved
				continue
			}
		}
		baseObj[operation][k] = v
	}
	out, err := json.Marshal(baseObj)
	if err != nil {
		return base
	}
	return string(out)
}

func paramsFor(operation string, palette map[string]string) string {
	dark := palette["$brand.primary"]
	if dark == "" {
		dark = "#0f172a"
	}
	light := palette["$brand.background"]
	if light == "" {
		light = "#e0f2fe"
	}
	switch operation {
	case "duotone":
		return fmt.Sprintf(`{"duotone":{"dark":%q,"light":%q}}`, dark, light)
	case "posterize":
		return fmt.Sprintf(`{"posterize":{"levels":5,"dark":%q,"light":%q}}`, dark, light)
	case "halftone":
		return fmt.Sprintf(`{"halftone":{"lpi":18,"angle":15,"dot":"circle","dark":%q,"light":%q}}`, dark, light)
	case "dither_ordered":
		return fmt.Sprintf(`{"dither_ordered":{"dark":%q,"light":%q}}`, dark, light)
	case "dither_diffusion":
		return fmt.Sprintf(`{"dither_diffusion":{"dark":%q,"light":%q}}`, dark, light)
	case "grain":
		return `{"grain":{"seed":1,"amount":0.12,"contrast_multiplier":1.05}}`
	case "scrim":
		return fmt.Sprintf(`{"scrim":{"color":%q,"opacity":0.34,"direction":"top"}}`, dark)
	case "line_screen", "stipple", "engraving", "ascii_mosaic":
		// These render ink on paper and honour Dark/Light, so they take the
		// brand palette like the Tier-1 screens rather than a hardcoded ink.
		return fmt.Sprintf(`{%q:{"dark":%q,"light":%q}}`, operation, dark, light)
	default:
		return fmt.Sprintf(`{%q:{}}`, operation)
	}
}
