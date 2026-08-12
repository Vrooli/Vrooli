package render

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"math"
	// Registering the JPEG decoder is what lets pngGeometry say "candidate is
	// jpeg, but the contract is PNG" instead of "unknown format". A model
	// backend really did return JPEG, and the precise message is the finding.
	_ "image/jpeg"
	_ "image/png"
	"sync"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/scaffold"
	"backdrop-studio/internal/scenes"
)

type Candidate struct {
	ID, JobID, Strategy, ExecutionPath        string
	PNG                                       []byte
	Width, Height                             int
	Seed                                      int64
	TreatmentApplied                          bool
	ConditioningSubmitted, DisclosureRequired bool
	Prompt, ProvenanceJSON                    string
	// QualityJSON is the perceptual verdict this candidate passed, carried so an
	// operator can see the margin rather than only the pass.
	QualityJSON string
}

type Job struct {
	ID, StyleID, Status, ExecutionPath string
	SurfaceID                          string
	Seed                               int64
	Candidates                         []Candidate
	SelectedCandidateID, SelectedBy    string
}

// Scaffold size. A scaffold is conditioning input for a preprocessor and gains
// nothing from extra pixels; the delivery geometry, by contrast, is never a
// constant here — it comes from the target surface record.
const (
	scaffoldWidth  = 512
	scaffoldHeight = 320
)

// Surface is the delivery target a render is sized from. It is passed in rather
// than looked up so this package keeps one job — orchestration — and the
// catalog stays the single authority on what a surface is.
type Surface struct {
	ID            string
	Width, Height int
}

func (s Surface) valid() bool { return s.ID != "" && s.Width > 0 && s.Height > 0 }

type Store struct {
	mu        sync.RWMutex
	jobs      map[string]*Job
	engine    imageengine.Executor
	generator imageengine.Generator
}

func NewStore(engine ...imageengine.Executor) *Store {
	var executor imageengine.Executor
	if len(engine) > 0 {
		executor = engine[0]
	}
	return &Store{jobs: map[string]*Job{}, engine: executor}
}

func NewStoreWithGenerator(engine imageengine.Executor, generator imageengine.Generator) *Store {
	return &Store{jobs: map[string]*Job{}, engine: engine, generator: generator}
}

// Request is one render submission. It is a struct rather than a parameter list
// because the list had reached six positional arguments, and the next reader
// could not tell placement from surface id at the call site.
type Request struct {
	Style       catalog.Style
	Surface     Surface
	Placement   string
	Seed        int64
	Count       int
	BrandTokens map[string]string
}

func (s *Store) SubmitWithContext(ctx context.Context, req Request) (Job, error) {
	style, placement, seed, count := req.Style, req.Placement, req.Seed, req.Count
	if style.ID == "" || style.Strategy == "" {
		return Job{}, fmt.Errorf("render: style is required")
	}
	// Delivery geometry has one authority: the surface record. A missing
	// surface is an error rather than a fallback constant, because a silent
	// default is how every store asset came out as a 1.6:1 landscape.
	if !req.Surface.valid() {
		return Job{}, fmt.Errorf("render: style %q needs a target surface with positive geometry", style.ID)
	}
	// One authority for what a "$brand.*" slot means during this render: the
	// style's declared ink defaults, overlaid by whatever the bound brand
	// supplies. A cold install renders from the defaults; a bound brand
	// changes the art without a catalog edit.
	palette := style.EffectivePalette(req.BrandTokens)
	if placement != "" && !contains(style.Placements, placement) {
		return Job{}, fmt.Errorf("render: placement %q is not permitted by style %q", placement, style.ID)
	}
	if count <= 0 {
		count = 1
	}
	if count > 16 {
		return Job{}, fmt.Errorf("render: candidate_count must be between 1 and 16")
	}
	jobID := id(style.ID, req.Surface.ID, seed, count)
	job := &Job{ID: jobID, StyleID: style.ID, SurfaceID: req.Surface.ID, Status: "completed", Seed: seed, ExecutionPath: expectedPath(style.Strategy)}
	for i := 0; i < count; i++ {
		candidateSeed := seed + int64(i)
		preset, ok := scenePreset(style.Subject)
		if !ok {
			// A procedural lane ships what the generator draws, so a subject
			// with no scene cannot be honoured — it used to fall through to
			// "field", meaning a style called Cyanotype Botanical rendered an
			// abstract colour field and nothing said so. Model-backed lanes may
			// still use any subject: the model draws it and the scaffold only
			// supplies composition geometry.
			if style.Strategy == "procedural" || style.Strategy == "procedural-treated" {
				return Job{}, fmt.Errorf("render: subject %q has no procedural scene; use a model-backed strategy or a subject with a scene (%v)", style.Subject, scenes.Presets)
			}
			preset = "field"
		}
		regions := make([]scaffold.Region, 0, len(style.Regions))
		for _, region := range style.Regions {
			regions = append(regions, scaffold.Region{X: region.X, Y: region.Y, Width: region.Width, Height: region.Height})
		}
		if style.Strategy == "guided" && style.Scaffold == nil {
			return Job{}, fmt.Errorf("render: guided style %q is missing its scaffold capability", style.ID)
		}
		conditioner := ""
		if style.Scaffold != nil {
			conditioner = style.Scaffold.Conditioner
		}
		if s.engine == nil {
			return Job{}, fmt.Errorf("render: image-tools executor is not configured")
		}

		// The procedural lanes ship what this step produces, so they render a
		// finished scene. The model-backed lanes only need conditioning
		// geometry, so they render a scaffold. Serving both from the scaffold
		// generators is what made the procedural lane emit blocked-out shapes.
		var input []byte
		var conditioning []byte
		outW, outH := req.Surface.Width, req.Surface.Height

		if style.Strategy == "guided" || style.Strategy == "synthesized" {
			sc, scErr := scaffold.Render(scaffold.Request{Preset: preset, Conditioner: conditioner, ParamsJSON: scaffoldParams(style), Width: scaffoldWidth, Height: scaffoldHeight, Seed: candidateSeed, Regions: regions})
			if scErr != nil {
				return Job{}, scErr
			}
			conditioning = sc.PNG
			input = sc.PNG
		} else {
			sn, snErr := scenes.Render(scenes.Request{Preset: preset, ParamsJSON: scaffoldParams(style), Width: req.Surface.Width, Height: req.Surface.Height, Seed: candidateSeed})
			if snErr != nil {
				return Job{}, fmt.Errorf("render: scene: %w", snErr)
			}
			input = sn.PNG
			outW, outH = sn.Width, sn.Height
		}
		conditioningSubmitted := false
		generationNative := ""
		prompt := ""
		if style.Generation != nil {
			prompt = style.Generation.PromptTemplate
		}
		if style.Strategy == "guided" || style.Strategy == "synthesized" {
			if s.generator == nil {
				return Job{}, fmt.Errorf("render: %s requires image-tools inference capability", style.Strategy)
			}
			nativeW, nativeH := generationGeometry(outW, outH)
			generated, genErr := s.generator.Generate(ctx, imageengine.GenerationRequest{
				Prompt:      prompt,
				Negative:    generationNegative(style),
				Seed:        candidateSeed,
				Conditioner: conditioner,
				Width:       nativeW,
				Height:      nativeH,
				Steps:       generationSteps,
				CFGScale:    generationCFGScale,
				Strength:    generationStrength,
				// Local-first by default. Routing is a declared policy, never
				// an accident: the previous hardcoded ("quality", "any", true)
				// sent every render to a paid cloud provider while an installed
				// local GPU served the same request in about fifteen seconds.
				QualityPolicy:  generationQualityPolicy,
				FallbackPolicy: generationFallbackPolicy,
				AllowBYOK:      false,
				Priority:       "batch",
				AllowReclaim:   true,
				Conditioning: func() []byte {
					if style.Strategy == "guided" {
						conditioningSubmitted = true
						return conditioning
					}
					return nil
				}(),
			})
			if genErr != nil {
				return Job{}, fmt.Errorf("render: %s inference capability: %w", style.Strategy, genErr)
			}
			if len(generated) == 0 {
				return Job{}, fmt.Errorf("render: %s inference returned an empty image", style.Strategy)
			}
			// A model answers in whatever format it likes; the candidate field
			// is named image_png and every consumer decodes it as one.
			normalized, convErr := s.normalizePNG(ctx, generated)
			if convErr != nil {
				return Job{}, fmt.Errorf("render: normalize generated source: %w", convErr)
			}
			// Proven decodable here so a bad generation is reported at its
			// source rather than as a confusing treatment-chain failure.
			nativeGenW, nativeGenH, dimErr := pngGeometry(normalized)
			if dimErr != nil {
				return Job{}, fmt.Errorf("render: measure generated source: %w", dimErr)
			}
			// The model draws near its native resolution; the surface decides
			// what ships. Without this step a model-backed style silently
			// delivered the model's geometry — 768x512 for a 1440x720 hero —
			// and nothing in the system said so.
			delivered, resizeErr := s.resizeTo(ctx, normalized, req.Surface.Width, req.Surface.Height)
			if resizeErr != nil {
				return Job{}, fmt.Errorf("render: scale generated source to surface %q: %w", req.Surface.ID, resizeErr)
			}
			generationNative = fmt.Sprintf("%dx%d", nativeGenW, nativeGenH)
			input = delivered
		}
		treated, err := s.engine.Apply(ctx, input, style.Treatments, style.TreatmentParams, palette)
		if err != nil {
			return Job{}, fmt.Errorf("render: treatment chain: %w", err)
		}
		// Treatments are same-size transforms, but asserting rather than
		// assuming is what makes the recorded geometry trustworthy on every
		// branch instead of on the branches someone remembered.
		treatedW, treatedH, treatedErr := pngGeometry(treated)
		if treatedErr != nil {
			return Job{}, fmt.Errorf("render: measure treated candidate: %w", treatedErr)
		}
		outW, outH = treatedW, treatedH

		// The perceptual gate runs here — after the chain, before the candidate
		// is recorded — because a candidate that reached the record is a
		// candidate someone can release. `engraved-colonnade` shipped illegible
		// moire past a fully green suite; nothing in the system could observe
		// that an image was unusable, and a contrast check cannot, since
		// high-contrast noise has excellent contrast.
		verdict, scoreErr := scoreCandidate(input, treated, style)
		if scoreErr != nil {
			return Job{}, fmt.Errorf("render: score candidate: %w", scoreErr)
		}
		if !verdict.Passed {
			return Job{}, &QualityRejectedError{StyleID: style.ID, Seed: candidateSeed, Verdict: verdict}
		}
		job.Candidates = append(job.Candidates, Candidate{ID: id(jobID, candidateSeed, i), JobID: jobID, Strategy: style.Strategy, ExecutionPath: job.ExecutionPath, PNG: treated, Width: outW, Height: outH, Seed: candidateSeed, TreatmentApplied: true, ConditioningSubmitted: conditioningSubmitted, DisclosureRequired: style.Strategy == "guided" || style.Strategy == "synthesized", Prompt: prompt, ProvenanceJSON: provenance(style, req.Surface, candidateSeed, job.ExecutionPath, generationNative, outW, outH), QualityJSON: qualityJSON(verdict)})
	}
	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()
	return copyJob(job), nil
}

// Generation sampler defaults. They are constants here and become per-style
// declared values in the quality-tier work; naming them is already better than
// the anonymous literals they replace.
const (
	generationSteps    = 30
	generationCFGScale = 7.5
	// generationStrength is how far img2img may travel from the conditioning
	// image. Low enough that the scaffold's composition survives.
	generationStrength = 0.72
	// Routing is declared, never accidental. Hardcoding these to
	// ("quality", "any", BYOK on) sent every model-backed render to a paid
	// cloud provider while an installed local GPU sat idle.
	generationQualityPolicy  = "balanced"
	generationFallbackPolicy = "local_only"
	// sdNativeEdge is SD-1.5's training resolution. Generating far above it
	// produces duplicated subjects — two horizons, two suns — so the model
	// generates near native and the result is scaled to the delivery size.
	sdNativeEdge = 512
	// sdSizeQuantum is the latent-space stride every SD-architecture model
	// requires its canvas to be a multiple of.
	sdSizeQuantum = 64
	// sdMaxEdge caps the long axis at 1.5x native. Past roughly this ratio an
	// SD-1.5 model repeats the composition rather than extending it.
	sdMaxEdge = 768
)

// generationGeometry maps a delivery size onto a canvas the model can actually
// draw well: the delivery aspect preserved where the model can hold it, the
// short edge at native resolution, both axes quantised to the latent stride.
//
// The long edge is capped because aspect is not free. An SD-architecture model
// pushed far past its training resolution on one axis starts repeating the
// composition — two horizons, two suns, a second colonnade — and no treatment
// downstream can repair that. A 9:19.5 store portrait is therefore drawn at a
// moderate aspect and cover-cropped to the surface, which loses some framing
// but never produces a duplicated subject.
func generationGeometry(deliveryW, deliveryH int) (int, int) {
	if deliveryW <= 0 || deliveryH <= 0 {
		return sdNativeEdge, sdNativeEdge
	}
	aspect := float64(deliveryW) / float64(deliveryH)
	width, height := sdNativeEdge, sdNativeEdge
	if aspect >= 1 {
		width = int(math.Min(float64(sdNativeEdge)*aspect, sdMaxEdge))
	} else {
		height = int(math.Min(float64(sdNativeEdge)/aspect, sdMaxEdge))
	}
	return quantise(width), quantise(height)
}

func quantise(v int) int {
	if v < sdSizeQuantum {
		return sdSizeQuantum
	}
	return (v / sdSizeQuantum) * sdSizeQuantum
}

// provenance records what actually produced this candidate, so a released
// backdrop can be reproduced and audited rather than merely trusted. For a
// model-backed candidate it names both sizes: the geometry the model drew and
// the geometry that shipped.
func provenance(style catalog.Style, surface Surface, seed int64, executionPath, generationNative string, width, height int) string {
	record := map[string]any{
		"style_id":       style.ID,
		"strategy":       style.Strategy,
		"seed":           seed,
		"treatments":     style.Treatments,
		"execution_path": executionPath,
		"surface_id":     surface.ID,
		"delivered_size": fmt.Sprintf("%dx%d", width, height),
	}
	if generationNative != "" {
		record["generation_native_size"] = generationNative
		record["generation_steps"] = generationSteps
		record["generation_cfg_scale"] = generationCFGScale
		record["generation_quality_policy"] = generationQualityPolicy
		record["generation_fallback_policy"] = generationFallbackPolicy
		if style.Strategy == "guided" {
			record["generation_strength"] = generationStrength
		}
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		// Unreachable: every value above is a plain string, number or slice.
		return fmt.Sprintf(`{"style_id":%q,"provenance_error":%q}`, style.ID, err.Error())
	}
	return string(encoded)
}

// resizeTo scales through image-tools when the engine offers it. The capability
// is optional on the Executor interface so a unit fake stays a small stub.
func (s *Store) resizeTo(ctx context.Context, in []byte, width, height int) ([]byte, error) {
	resizer, ok := s.engine.(interface {
		Resize(context.Context, []byte, int, int) ([]byte, error)
	})
	if !ok {
		return in, nil
	}
	return resizer.Resize(ctx, in, width, height)
}

// normalizePNG re-encodes through image-tools when the engine can do it, and
// otherwise passes the bytes through. The capability is optional on the
// Executor interface so a unit fake stays a two-line stub.
func (s *Store) normalizePNG(ctx context.Context, in []byte) ([]byte, error) {
	converter, ok := s.engine.(interface {
		ToPNG(context.Context, []byte) ([]byte, error)
	})
	if !ok {
		return in, nil
	}
	return converter.ToPNG(ctx, in)
}

// scenePreset maps a catalog subject onto a scene generator. Only these
// subjects can be rendered procedurally; the rest need a model.
func scenePreset(subject string) (string, bool) {
	switch subject {
	case "horizon", "aquatic", "atmospheric":
		return "horizon", true
	case "statuary_architecture", "interior":
		return "arcade", true
	case "geological", "cartographic":
		return "terrain", true
	case "non_representational", "textile_material", "object_metaphor":
		return "field", true
	}
	return "", false
}

func scaffoldParams(style catalog.Style) string {
	if style.Scaffold == nil {
		return ""
	}
	return style.Scaffold.ParamsJSON
}

func generationNegative(style catalog.Style) string {
	if style.Generation == nil {
		return ""
	}
	return style.Generation.Negative
}

func (s *Store) Get(id string) (Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return Job{}, fmt.Errorf("render: job %q not found", id)
	}
	return copyJob(job), nil
}

func (s *Store) Select(jobID, candidateID, actor string) (Job, error) {
	if actor == "" {
		return Job{}, fmt.Errorf("render: actor is required for selection")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return Job{}, fmt.Errorf("render: job %q not found", jobID)
	}
	for _, candidate := range job.Candidates {
		if candidate.ID == candidateID {
			job.SelectedCandidateID = candidateID
			job.SelectedBy = actor
			job.Status = "selected"
			return copyJob(job), nil
		}
	}
	return Job{}, fmt.Errorf("render: candidate %q does not belong to job %q", candidateID, jobID)
}

func (s *Store) Candidate(jobID, candidateID string) (Candidate, error) {
	job, err := s.Get(jobID)
	if err != nil {
		return Candidate{}, err
	}
	for _, c := range job.Candidates {
		if c.ID == candidateID {
			return c, nil
		}
	}
	return Candidate{}, fmt.Errorf("render: candidate %q not found", candidateID)
}

func expectedPath(strategy string) string {
	switch strategy {
	case "guided":
		return "scaffold → image-tools inference → treatment"
	case "synthesized":
		return "image-tools inference → treatment"
	default:
		return "procedural → treatment"
	}
}

// pngGeometry reads a candidate's real pixel dimensions. Decoding only the
// header is metadata inspection, not a pixel operation, so it stays inside this
// scenario's charter of owning no raster implementation.
func pngGeometry(encoded []byte) (int, int, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(encoded))
	if err != nil {
		return 0, 0, fmt.Errorf("decode candidate header: %w", err)
	}
	if format != "png" {
		return 0, 0, fmt.Errorf("candidate is %s, but the contract is PNG", format)
	}
	return cfg.Width, cfg.Height, nil
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func id(parts ...interface{}) string {
	h := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(h, "%v\x00", part)
	}
	return hex.EncodeToString(h.Sum(nil))[:20]
}

func copyJob(job *Job) Job {
	out := *job
	out.Candidates = make([]Candidate, len(job.Candidates))
	for i, c := range job.Candidates {
		out.Candidates[i] = c
		out.Candidates[i].PNG = append([]byte(nil), c.PNG...)
	}
	return out
}
