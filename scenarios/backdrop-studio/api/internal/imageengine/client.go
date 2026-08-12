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

// GenerationRequest is the inference contract. Concrete model identity stays
// owned by image-tools' selector; Backdrop Studio declares the geometry, the
// sampler settings and the routing policy it needs.
//
// Every field here reaches the wire. The previous shape carried Role and
// Profile, validated them, and then never sent them — AIParams has no such
// fields — so the type described a contract that did not exist, and the next
// reader would reasonably assume role-based routing worked.
type GenerationRequest struct {
	Prompt, Negative, Conditioner string
	Seed                          int64
	// Width and Height are the generation canvas. They come from the caller
	// because the caller knows the delivery surface; the previous literal
	// pinned 320x180 for a product whose hero is 1440x720.
	Width, Height int
	Steps         int
	CFGScale      float64
	// Strength is the img2img denoising strength. Only meaningful when
	// Conditioning is supplied; zero leaves the model default.
	Strength float64
	// QualityPolicy, FallbackPolicy and AllowBYOK are the routing policy.
	//
	// These are load-bearing, not decoration. Hardcoding them to
	// ("quality", "any", true) sent every model-backed render to a paid cloud
	// provider while an installed local GPU served the same request in about
	// fifteen seconds — billable, slower, and silent.
	QualityPolicy, FallbackPolicy string
	AllowBYOK                     bool
	// Priority is the capacity claim tier: "batch", "service" or
	// "interactive". Backdrop renders are not interactive — an operator submits
	// and waits — so batch is correct and lets latency-sensitive work win.
	Priority string
	// AllowReclaim permits image-tools to free idle lower-value GPU residents
	// to satisfy this claim. On a shared host this is the difference between a
	// render and an out-of-memory failure: this box routinely holds 13GB of
	// idle language models against a 16GB card.
	AllowReclaim bool
	Conditioning []byte
}

// GenerationResult is the image plus the facts needed to disclose it.
//
// Generate used to return only bytes, which meant the model that drew the
// image was known to image-tools, reported on the wire, and then discarded one
// stack frame later. A synthetic image whose model nobody recorded cannot be
// honestly labelled, so those two fields are part of the result rather than
// something a caller reconstructs.
type GenerationResult struct {
	PNG []byte
	// ModelID is the registry model image-tools' selector actually chose. It is
	// never a model this scenario asked for: routing is image-tools' decision,
	// and recording a requested model as the used one would be a fabricated
	// disclosure.
	ModelID string
	// Tier is where it ran — "local-gpu", "local-cpu" or "byok-cloud".
	Tier string
}

type Generator interface {
	Generate(context.Context, GenerationRequest) (GenerationResult, error)
}

// submitGenerationResponse decodes image-tools' REST submit edge.
//
// The field name is `job_id`, not `jobId`. image-tools serialises its REST
// multipart edge with protojson's proto names while its Connect edge uses the
// camelCase JSON names — so `SubmitAIResponse.job_id` arrives snake_case and
// `Job.resultRef` arrives camelCase from the very same service. Decoding the
// wrong one here is why every model-backed render failed with "inference
// returned no job id" while generation was in fact working.
type submitGenerationResponse struct {
	JobID string `json:"job_id"`
	// ModelID and Tier arrive on the same submit response as the job id and
	// were previously dropped. They are the disclosure record's two hardest
	// facts to reconstruct after the fact.
	ModelID string `json:"model_id"`
	Tier    string `json:"tier"`
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
		params, paramsErr := mergedParams(treatment, overrides[treatment], palette)
		if paramsErr != nil {
			return nil, paramsErr
		}
		result, err = run(ctx, httpClient, baseURL, treatment, result, params)
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
func (c *Client) Generate(ctx context.Context, req GenerationRequest) (GenerationResult, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return GenerationResult{}, fmt.Errorf("image-tools inference: a prompt is required")
	}
	if req.Width <= 0 || req.Height <= 0 {
		return GenerationResult{}, fmt.Errorf("image-tools inference: generation geometry is required (got %dx%d)", req.Width, req.Height)
	}
	resolve := c.Resolve
	if resolve == nil {
		return GenerationResult{}, fmt.Errorf("image-tools inference capability unavailable: URL resolver is not configured")
	}
	base, err := resolve(ctx)
	if err != nil {
		return GenerationResult{}, fmt.Errorf("image-tools inference capability unavailable: %w", err)
	}
	operation := "text_to_image"
	if len(req.Conditioning) > 0 {
		operation = "image_to_image"
	}
	params := map[string]any{
		"prompt":         req.Prompt,
		"negativePrompt": req.Negative,
		"seed":           req.Seed,
		"width":          req.Width,
		"height":         req.Height,
		"qualityPolicy":  req.QualityPolicy,
		"fallbackPolicy": req.FallbackPolicy,
		"allowByok":      req.AllowBYOK,
		"priority":       req.Priority,
		"allowReclaim":   req.AllowReclaim,
	}
	if req.Steps > 0 {
		params["steps"] = req.Steps
	}
	if req.CFGScale > 0 {
		params["cfgScale"] = req.CFGScale
	}
	if req.Strength > 0 && len(req.Conditioning) > 0 {
		params["strength"] = req.Strength
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if len(req.Conditioning) > 0 {
		part, e := mw.CreateFormFile("file", "scaffold.png")
		if e != nil {
			return GenerationResult{}, e
		}
		if _, e = part.Write(req.Conditioning); e != nil {
			return GenerationResult{}, e
		}
	}
	raw, _ := json.Marshal(params)
	if err := mw.WriteField("params", string(raw)); err != nil {
		return GenerationResult{}, err
	}
	if err := mw.Close(); err != nil {
		return GenerationResult{}, err
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/v1/ai/"+operation, &body)
	if err != nil {
		return GenerationResult{}, err
	}
	request.Header.Set("Content-Type", mw.FormDataContentType())
	response, err := client.Do(request)
	if err != nil {
		return GenerationResult{}, fmt.Errorf("image-tools inference %s: %w", operation, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 8<<10))
		return GenerationResult{}, fmt.Errorf("image-tools inference capability unavailable: %s: %s", response.Status, strings.TrimSpace(string(detail)))
	}
	var submitted submitGenerationResponse
	if err := json.NewDecoder(response.Body).Decode(&submitted); err != nil || submitted.JobID == "" {
		return GenerationResult{}, fmt.Errorf("image-tools inference returned no job id")
	}
	waitBody, _ := json.Marshal(map[string]string{"id": submitted.JobID})
	waitReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/vrooli.image_tools.v1.jobs.JobsService/WaitJob", bytes.NewReader(waitBody))
	if err != nil {
		return GenerationResult{}, err
	}
	waitReq.Header.Set("Content-Type", "application/json")
	waitResp, err := client.Do(waitReq)
	if err != nil {
		return GenerationResult{}, fmt.Errorf("image-tools inference wait: %w", err)
	}
	defer waitResp.Body.Close()
	if waitResp.StatusCode < 200 || waitResp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(waitResp.Body, 8<<10))
		return GenerationResult{}, fmt.Errorf("image-tools inference wait failed: %s", strings.TrimSpace(string(detail)))
	}
	var waited waitGenerationResponse
	if err := json.NewDecoder(waitResp.Body).Decode(&waited); err != nil {
		return GenerationResult{}, fmt.Errorf("image-tools inference wait response: %w", err)
	}
	if waited.Job.Error != "" {
		return GenerationResult{}, fmt.Errorf("image-tools inference failed: %s", waited.Job.Error)
	}
	if waited.Job.ResultRef == "" {
		return GenerationResult{}, fmt.Errorf("image-tools inference completed without a result reference")
	}
	resultURL := strings.TrimRight(base, "/") + "/api/v1/blobs/" + strings.TrimLeft(waited.Job.ResultRef, "/")
	resultResp, err := client.Get(resultURL)
	if err != nil {
		return GenerationResult{}, fmt.Errorf("image-tools inference result: %w", err)
	}
	defer resultResp.Body.Close()
	if resultResp.StatusCode < 200 || resultResp.StatusCode >= 300 {
		return GenerationResult{}, fmt.Errorf("image-tools inference result returned %s", resultResp.Status)
	}
	output, err := io.ReadAll(resultResp.Body)
	if err != nil {
		return GenerationResult{}, err
	}
	if len(output) == 0 {
		return GenerationResult{}, fmt.Errorf("image-tools inference returned an empty image")
	}
	return GenerationResult{PNG: output, ModelID: submitted.ModelID, Tier: submitted.Tier}, nil
}

// ToPNG re-encodes bytes as PNG through image-tools.
//
// It exists because a model backend answers in whatever format it likes — the
// BYOK cloud route returned JPEG — while this scenario's candidate field is
// named `image_png` and every downstream consumer decodes it as one. Rather
// than teach Backdrop Studio to encode pixels, which is image-tools' job by
// charter, the normalization is one more call to the service that owns it.
func (c *Client) ToPNG(ctx context.Context, input []byte) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("image-tools: cannot convert empty bytes")
	}
	if bytes.HasPrefix(input, []byte("\x89PNG\r\n\x1a\n")) {
		return input, nil
	}
	resolve := c.Resolve
	if resolve == nil {
		return nil, fmt.Errorf("image-tools: URL resolver is not configured")
	}
	baseURL, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("image-tools: resolve scenario: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return run(ctx, httpClient, strings.TrimRight(baseURL, "/"), "convert", input, `{"convert":{"format":"png"}}`)
}

// Resize scales bytes to an exact geometry through image-tools.
//
// The model-backed lane needs it because a diffusion model must generate near
// its training resolution — SD-1.5 draws two horizons and two suns when pushed
// far past 512 — while the delivery surface is whatever the operator asked for.
// So the lane generates native and scales here, and the candidate reaches the
// surface's declared geometry instead of silently shipping the model's.
func (c *Client) Resize(ctx context.Context, input []byte, width, height int) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("image-tools: cannot resize empty bytes")
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("image-tools: resize needs positive geometry (got %dx%d)", width, height)
	}
	resolve := c.Resolve
	if resolve == nil {
		return nil, fmt.Errorf("image-tools: URL resolver is not configured")
	}
	baseURL, err := resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("image-tools: resolve scenario: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	// "fill" covers the target and centre-crops, which preserves the subject's
	// proportions. "stretch" would reach the same geometry by distorting the
	// image, which is the defect that shipped an elliptical sun.
	params := fmt.Sprintf(`{"resize":{"width":%d,"height":%d,"fit":"fill","gravity":"center"}}`, width, height)
	return run(ctx, httpClient, strings.TrimRight(baseURL, "/"), "resize", input, params)
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

// UnresolvedSlotError reports a "$brand.*" slot that no palette entry and no
// style ink default could bind.
//
// It is a distinct type because the handler edge maps it to
// connect.CodeFailedPrecondition: the caller has to supply a brand or the
// catalog has to declare a default, and neither is something a retry fixes.
type UnresolvedSlotError struct {
	Slot, Operation, Field string
}

func (e *UnresolvedSlotError) Error() string {
	return fmt.Sprintf("image-tools: brand slot %q on operation %q field %q resolves to nothing; bind a brand that defines it or give the style an ink default for it", e.Slot, e.Operation, e.Field)
}

// ResolveParams is the exported form of the merge: it produces the exact
// parameter JSON that would be sent to image-tools for one operation, with
// $brand.* slots bound against the supplied palette. Callers use it to assert
// on what will actually go over the wire rather than on the catalog's stored
// intent.
func ResolveParams(operation, override string, palette map[string]string) (string, error) {
	return mergedParams(operation, override, palette)
}

// mergedParams overlays a style's op parameters onto the palette-derived
// defaults. Overrides may name "$brand.*" slots, which resolve against the
// effective palette here rather than being baked into the catalog — the same
// mechanism that lets one style render correctly for several brands.
//
// Resolution fails closed. This used to fall through and write the literal
// slot string onto the wire when the lookup missed, which image-tools rejected
// with `422 invalid color "$brand.primary"` — that single fall-open made ten of
// sixteen seeded styles unrenderable from the CLI, and no unit test could see
// it because they all resolved against a bound brand.
func mergedParams(operation, override string, palette map[string]string) (string, error) {
	base, err := paramsFor(operation, palette)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(override) == "" {
		return base, nil
	}
	var baseObj map[string]map[string]any
	if err := json.Unmarshal([]byte(base), &baseObj); err != nil || baseObj[operation] == nil {
		baseObj = map[string]map[string]any{operation: {}}
	}
	var over map[string]any
	if err := json.Unmarshal([]byte(override), &over); err != nil {
		return "", fmt.Errorf("image-tools: operation %q parameters are not a JSON object: %w", operation, err)
	}
	for k, v := range over {
		if sv, ok := v.(string); ok && strings.HasPrefix(sv, "$brand.") {
			resolved, ok := palette[sv]
			if !ok || strings.TrimSpace(resolved) == "" {
				return "", &UnresolvedSlotError{Slot: sv, Operation: operation, Field: k}
			}
			baseObj[operation][k] = resolved
			continue
		}
		baseObj[operation][k] = v
	}
	out, err := json.Marshal(baseObj)
	if err != nil {
		return "", fmt.Errorf("image-tools: encode %q parameters: %w", operation, err)
	}
	return string(out), nil
}

// defaultInks are the inks an operation falls back to when the effective
// palette names neither a brand token nor a style default. They exist so a
// treatment chain that never mentions a slot still renders; a chain that *does*
// mention one is resolved strictly by mergedParams and never reaches these.
const (
	defaultDarkInk  = "#0f172a"
	defaultLightInk = "#e0f2fe"
)

func paramsFor(operation string, palette map[string]string) (string, error) {
	dark := palette["$brand.primary"]
	if strings.TrimSpace(dark) == "" {
		dark = defaultDarkInk
	}
	light := palette["$brand.background"]
	if strings.TrimSpace(light) == "" {
		light = defaultLightInk
	}
	switch operation {
	case "duotone":
		return fmt.Sprintf(`{"duotone":{"dark":%q,"light":%q}}`, dark, light), nil
	case "posterize":
		return fmt.Sprintf(`{"posterize":{"levels":5,"dark":%q,"light":%q}}`, dark, light), nil
	case "halftone":
		// LPI is lines across the image width, so it is already resolution
		// independent. The default is fine rather than coarse: a coarse screen
		// erases the subject it is supposed to modulate, which is what made
		// `engraved-colonnade` render as moire. See docs/reference/taxonomy.md.
		return fmt.Sprintf(`{"halftone":{"lpi":120,"angle":15,"dot":"circle","dark":%q,"light":%q}}`, dark, light), nil
	case "dither_ordered":
		return fmt.Sprintf(`{"dither_ordered":{"dark":%q,"light":%q}}`, dark, light), nil
	case "dither_diffusion":
		return fmt.Sprintf(`{"dither_diffusion":{"dark":%q,"light":%q}}`, dark, light), nil
	case "grain":
		return `{"grain":{"seed":1,"amount":0.06,"contrast_multiplier":1.04}}`, nil
	case "scrim":
		return fmt.Sprintf(`{"scrim":{"color":%q,"opacity":0.34,"direction":"top"}}`, dark), nil
	case "line_screen", "stipple", "engraving", "ascii_mosaic":
		// These render ink on paper and honour Dark/Light, so they take the
		// brand palette like the Tier-1 screens rather than a hardcoded ink.
		return fmt.Sprintf(`{%q:{%s"dark":%q,"light":%q}}`, operation, spatialDefaults[operation], dark, light), nil
	default:
		if rel := spatialDefaults[operation]; rel != "" {
			return fmt.Sprintf(`{%q:{%s}}`, operation, strings.TrimSuffix(rel, ",")), nil
		}
		return fmt.Sprintf(`{%q:{}}`, operation), nil
	}
}

// spatialDefaults are this scenario's screen and extent defaults, expressed as
// a fraction of the delivered image's short edge so a style tuned once holds
// its look at every surface. Each fragment is a JSON object body ending in a
// comma, ready to splice ahead of the ink pair.
//
// The pixel values quoted below are what each fraction resolves to on the
// 1440x720 `web.hero` surface, which is the surface the catalog is art-directed
// against. On `web.hero-mobile` (390x844, short edge 390) they resolve to
// roughly half those figures, which is the intent: the screen stays the same
// coarseness relative to the picture rather than the same size in pixels.
//
// Absolute equivalents are deliberately absent. A style that means pixels can
// still send them — image-tools keeps the absolute form — but nothing seeded
// here does, and `TestSeededStylesSendNoAbsoluteSpatialParameter` enforces it.
var spatialDefaults = map[string]string{
	// ~9px pitch on a hero: fine enough to read as tone at arm's length,
	// coarse enough that the line is visibly a line.
	"line_screen": `"spacing_rel":0.0125,`,
	// ~7px cell. Stipple carries tone through dot count, so it needs a
	// tighter grid than the line screen to reach the same tonal range.
	"stipple": `"spacing_rel":0.0097,`,
	// ~9px period, matching the line screen: engraving is the same gesture
	// with a cross-hatch, and the two should sit together in one catalog.
	"engraving": `"spacing_rel":0.0125,`,
	// ~14px cell = two glyph advances, so characters blit at 2:1 and stay
	// crisp. See the snapping rule in image-tools' ops.ResolveSpatialParams.
	"ascii_mosaic": `"block_size_rel":0.02,`,
	// ~52px wavelength with a ~5px throw: a slow swell, not a ripple.
	"displacement": `"spacing_rel":0.072,"amplitude_rel":0.007,`,
	// ~4px separation: visible as colour fringing, not as a double image.
	"aberration": `"distance_rel":0.0056,`,
	// ~6px lift on the highlights.
	"bloom": `"radius_rel":0.008,`,
	// ~4px defocus.
	"defocus": `"radius_rel":0.0056,`,
	// ~10px smear.
	"motion_blur": `"distance_rel":0.014,`,
}
