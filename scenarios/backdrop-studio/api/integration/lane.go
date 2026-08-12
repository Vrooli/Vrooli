//go:build integration

// Package integration holds the lane that renders the real catalog through
// really running scenarios.
//
// It exists because of a specific, expensive failure: twelve of sixteen seeded
// styles were unrenderable while every Go unit test passed. The unit suite
// tests against a fake executor that never reaches image-tools' REST edge, and
// image-tools tests its treatments below the wire, so neither side could see
// the boundary between them. The one contract test that did cross it resolved
// parameters against a bound brand — the single path a CLI caller never takes.
//
// The rule this lane enforces: a style is not shippable until its exact bytes
// have made a round trip through a running image-tools and come back as an
// image. Everything else is inference.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"backdrop-studio/internal/buildinfo"

	"github.com/vrooli/api-core/discovery"
)

// Environment is everything the lane needs to reach the system under test,
// resolved once so a failure to reach a dependency is reported as a named
// unavailable dependency rather than as a render failure.
type Environment struct {
	BackdropURL   string
	ImageToolsURL string
	HTTP          *http.Client
}

// Resolve locates the running scenarios through the same discovery every
// cross-scenario call uses. A missing dependency is an error here, at setup,
// so it can never be misread later as "this style does not render".
func Resolve(ctx context.Context) (Environment, error) {
	client := &http.Client{Timeout: 10 * time.Minute}
	backdrop, err := discovery.ResolveScenarioURLDefault(ctx, "backdrop-studio")
	if err != nil {
		return Environment{}, fmt.Errorf("integration: backdrop-studio is not reachable (start it with `vrooli scenario start backdrop-studio`): %w", err)
	}
	tools, err := discovery.ResolveScenarioURLDefault(ctx, "image-tools")
	if err != nil {
		return Environment{}, fmt.Errorf("integration: image-tools is not reachable (start it with `vrooli scenario start image-tools`): %w", err)
	}
	return Environment{
		BackdropURL:   strings.TrimRight(backdrop, "/"),
		ImageToolsURL: strings.TrimRight(tools, "/"),
		HTTP:          client,
	}, nil
}

// BuildReport mirrors the API's freshness payload.
type BuildReport struct {
	Service            string `json:"service"`
	Version            string `json:"version"`
	Fingerprint        string `json:"fingerprint"`
	SeedVersion        int    `json:"seed_version"`
	AppliedSeedVersion int    `json:"applied_seed_version"`
}

// AssertFreshBinary refuses to judge anything until the running API is built
// from the working tree.
//
// Two audits in two days drew false conclusions from a stale binary, and one of
// them recorded a working capability as missing. A lane that renders against
// yesterday's server produces findings about yesterday's server while claiming
// to describe today's source — which is worse than no lane at all, because the
// output looks like evidence.
func (e Environment) AssertFreshBinary(ctx context.Context) (BuildReport, error) {
	var report BuildReport
	if err := e.getJSON(ctx, e.BackdropURL+"/api/v1/build", &report); err != nil {
		return report, fmt.Errorf("integration: read build report: %w", err)
	}
	root, err := buildinfo.SourceRoot()
	if err != nil {
		return report, fmt.Errorf("integration: locate working tree: %w", err)
	}
	want, err := buildinfo.Compute(root)
	if err != nil {
		return report, fmt.Errorf("integration: fingerprint working tree: %w", err)
	}
	if report.Fingerprint != want {
		return report, fmt.Errorf(
			"integration: the running API is not built from this working tree\n  running: %s\n  tree:    %s\n  fix:     (cd scenarios/backdrop-studio && make build) && vrooli scenario restart backdrop-studio",
			report.Fingerprint, want)
	}
	if report.AppliedSeedVersion < report.SeedVersion {
		return report, fmt.Errorf(
			"integration: the install has not applied the catalog this binary ships (applied v%d, shipped v%d); restart the scenario",
			report.AppliedSeedVersion, report.SeedVersion)
	}
	return report, nil
}

// Capabilities is what image-tools can actually do right now. The lane records
// it so a skipped assertion always has a reason a reader can check, rather than
// a silent pass.
type Capabilities struct {
	Operations   []string
	ImageModels  []string
	Adapters     []string
	GenerationOK bool
	ControlNetOK bool
}

type opsListResponse struct {
	Operations []struct {
		Name string `json:"name"`
	} `json:"operations"`
}

// installState mirrors image-tools' nested install block. Installation is a
// sub-message, not a flat boolean — reading it as flat is how the first version
// of this probe reported "no image models installed" on a host that was
// generating in fifteen seconds, and turned every model-backed assertion into a
// silent skip.
type installState struct {
	Installed bool `json:"installed"`
}

type modelsListResponse struct {
	Models []struct {
		ID         string       `json:"id"`
		Operations []string     `json:"operations"`
		Enabled    bool         `json:"enabled"`
		Install    installState `json:"install"`
	} `json:"models"`
}

type adaptersListResponse struct {
	Adapters []struct {
		ID      string       `json:"id"`
		Kind    string       `json:"kind"`
		Enabled bool         `json:"enabled"`
		Ready   bool         `json:"ready"`
		Install installState `json:"install"`
	} `json:"adapters"`
}

// ProbeCapabilities asks image-tools what it can do instead of assuming.
func (e Environment) ProbeCapabilities(ctx context.Context) (Capabilities, error) {
	var caps Capabilities

	var ops opsListResponse
	if err := e.postJSON(ctx, e.ImageToolsURL+"/vrooli.image_tools.v1.ops.OpsService/ListOperations", struct{}{}, &ops); err != nil {
		return caps, fmt.Errorf("integration: list image-tools operations: %w", err)
	}
	for _, op := range ops.Operations {
		caps.Operations = append(caps.Operations, op.Name)
	}

	var models modelsListResponse
	if err := e.postJSON(ctx, e.ImageToolsURL+"/vrooli.image_tools.v1.models.ModelsService/ListModels", struct{}{}, &models); err != nil {
		// Model discovery being unavailable is a named degradation, not a
		// render failure: the deterministic lane still runs.
		return caps, nil
	}
	for _, model := range models.Models {
		if !model.Enabled || !model.Install.Installed {
			continue
		}
		caps.ImageModels = append(caps.ImageModels, model.ID)
		for _, op := range model.Operations {
			if op == "text_to_image" || op == "image_to_image" {
				caps.GenerationOK = true
			}
		}
	}

	var adapters adaptersListResponse
	if err := e.postJSON(ctx, e.ImageToolsURL+"/vrooli.image_tools.v1.adapters.AdaptersService/ListAdapters", struct{}{}, &adapters); err == nil {
		for _, adapter := range adapters.Adapters {
			if !adapter.Enabled || !adapter.Ready || !adapter.Install.Installed {
				continue
			}
			caps.Adapters = append(caps.Adapters, adapter.ID)
			if adapter.Kind == "controlnet" {
				caps.ControlNetOK = true
			}
		}
	}
	return caps, nil
}

// Candidate is one rendered result as the lane reads it back.
type Candidate struct {
	ID             string `json:"id"`
	ImagePNG       []byte `json:"imagePng"`
	Width          int32  `json:"width"`
	Height         int32  `json:"height"`
	Strategy       string `json:"strategy"`
	ExecutionPath  string `json:"executionPath"`
	ProvenanceJSON string `json:"provenanceJson"`
	QualityJSON    string `json:"qualityJson"`
}

// RenderJob is the Submit response.
type RenderJob struct {
	ID            string      `json:"id"`
	StyleID       string      `json:"styleId"`
	SurfaceID     string      `json:"surfaceId"`
	Status        string      `json:"status"`
	ExecutionPath string      `json:"executionPath"`
	Candidates    []Candidate `json:"candidates"`
}

// SubmitOptions is one render request from the lane's point of view.
type SubmitOptions struct {
	StyleID     string
	Seed        int64
	Placement   string
	SurfaceID   string
	BrandID     string
	BrandTokens map[string]string
}

// Submit renders one style through the running API, which renders it through
// the running image-tools. Any non-2xx is returned as an error carrying the
// server's own message, because that message is the finding.
func (e Environment) Submit(ctx context.Context, opts SubmitOptions) (RenderJob, error) {
	body := map[string]any{
		"style": map[string]any{"id": opts.StyleID},
		"seed":  fmt.Sprintf("%d", opts.Seed),
	}
	if opts.Placement != "" {
		body["placement"] = opts.Placement
	}
	if opts.SurfaceID != "" {
		body["surface_id"] = opts.SurfaceID
	}
	if opts.BrandID != "" {
		body["brand_id"] = opts.BrandID
	}
	if len(opts.BrandTokens) > 0 {
		body["brand_tokens"] = opts.BrandTokens
	}
	var job RenderJob
	if err := e.postJSON(ctx, e.BackdropURL+"/vrooli.backdrop_studio.v1.render.RenderService/Submit", body, &job); err != nil {
		return job, err
	}
	return job, nil
}

// Scaffold renders one deterministic composition scaffold at the requested
// geometry and returns its PNG bytes. The treatment gallery uses it so every
// sample in the gallery is screened over the same picture, and so the gallery
// can be regenerated at any size without a checked-in source image.
func (e Environment) Scaffold(ctx context.Context, preset string, width, height int, seed int64) ([]byte, error) {
	// The Connect edge emits camelCase, unlike the REST edge next door — see
	// the note on RenderJob.Candidates and PROBLEMS.md.
	var out struct {
		ImagePNG []byte `json:"imagePng"`
	}
	body := map[string]any{
		"preset": preset,
		"width":  width,
		"height": height,
		"seed":   fmt.Sprintf("%d", seed),
	}
	if err := e.postJSON(ctx, e.BackdropURL+"/vrooli.backdrop_studio.v1.scaffold.ScaffoldService/Render", body, &out); err != nil {
		return nil, err
	}
	if len(out.ImagePNG) == 0 {
		return nil, fmt.Errorf("integration: scaffold %q returned no image", preset)
	}
	return out.ImagePNG, nil
}

// DecodePNG returns the candidate's real pixel geometry. The lane compares it
// against the recorded width and height because a candidate that reports a size
// it does not have makes every downstream size decision wrong.
func DecodePNG(png []byte) (int, int, error) {
	if len(png) == 0 {
		return 0, 0, fmt.Errorf("candidate carries no image bytes")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(png))
	if err != nil {
		return 0, 0, fmt.Errorf("candidate bytes are not a decodable image: %w", err)
	}
	return cfg.Width, cfg.Height, nil
}

// Style is the catalog record the lane iterates.
type Style struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Strategy   string            `json:"strategy"`
	Subject    string            `json:"subject"`
	Treatments []string          `json:"treatments"`
	Placements []string          `json:"placements"`
	Inks       map[string]string `json:"inks"`
}

type catalogListResponse struct {
	Styles []Style `json:"styles"`
}

// Styles lists the catalog the running install actually holds — not the one the
// working tree ships. AssertFreshBinary has already proven they agree.
func (e Environment) Styles(ctx context.Context) ([]Style, error) {
	var resp catalogListResponse
	if err := e.postJSON(ctx, e.BackdropURL+"/vrooli.backdrop_studio.v1.catalog.CatalogService/ListStyles", struct{}{}, &resp); err != nil {
		return nil, fmt.Errorf("integration: list styles: %w", err)
	}
	if len(resp.Styles) == 0 {
		return nil, fmt.Errorf("integration: the running install has an empty catalog")
	}
	return resp.Styles, nil
}

// Surface is a declared output target.
type Surface struct {
	ID         string   `json:"id"`
	Width      int32    `json:"width"`
	Height     int32    `json:"height"`
	Placements []string `json:"placements"`
}

type surfacesListResponse struct {
	Surfaces []Surface `json:"surfaces"`
}

// PermittedSurfaces returns every surface whose placements intersect the
// style's, smallest area first. The lane renders the extremes because a single
// geometry hides exactly the defects this phase repaired: a delivery constant
// that ignored the surface record, and a candidate reporting a size it did not
// have.
func PermittedSurfaces(style Style, surfaces []Surface) []Surface {
	var out []Surface
	for _, surface := range surfaces {
		for _, want := range style.Placements {
			if containsString(surface.Placements, want) {
				out = append(out, surface)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return int(out[i].Width)*int(out[i].Height) < int(out[j].Width)*int(out[j].Height)
	})
	return out
}

func containsString(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

// Surfaces lists the declared output geometries.
func (e Environment) Surfaces(ctx context.Context) ([]Surface, error) {
	var resp surfacesListResponse
	if err := e.postJSON(ctx, e.BackdropURL+"/vrooli.backdrop_studio.v1.surfaces.SurfacesService/ListSurfaces", struct{}{}, &resp); err != nil {
		return nil, fmt.Errorf("integration: list surfaces: %w", err)
	}
	return resp.Surfaces, nil
}

// ModelBacked reports whether a style needs an image model to render at all.
func (s Style) ModelBacked() bool {
	return s.Strategy == "guided" || s.Strategy == "synthesized"
}

// IsGPUCapacityFailure distinguishes "this host cannot serve generation right
// now" from "this style is broken".
//
// The distinction is load-bearing. A shared workstation routinely holds ten
// gigabytes of idle language models against a sixteen-gigabyte card, so a
// diffusion job loses the allocation race through no fault of the catalog.
// Reporting that as a style failure would send the next reader hunting a defect
// that is not there; reporting it as a pass would be a lie. It is a named skip.
func IsGPUCapacityFailure(err error) bool {
	if err == nil {
		return false
	}
	text := err.Error()
	for _, marker := range []string{
		"ErrorOutOfDeviceMemory",
		"Device memory allocation of size",
		"alloc params backend buffer failed",
		"out of memory",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func (e Environment) getJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return e.do(req, out)
}

func (e Environment) postJSON(ctx context.Context, url string, payload, out any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return e.do(req, out)
}

func (e Environment) do(req *http.Request, out any) error {
	resp, err := e.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", req.Method, req.URL.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if readErr != nil {
		return fmt.Errorf("%s %s: read body: %w", req.Method, req.URL.Path, readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned %s: %s", req.Method, req.URL.Path, resp.Status, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s %s: decode response: %w (body %.400s)", req.Method, req.URL.Path, err, body)
	}
	return nil
}
