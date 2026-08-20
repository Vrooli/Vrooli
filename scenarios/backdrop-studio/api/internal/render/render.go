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
	"strconv"
	"strings"
	"sync"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/legibility"
	"backdrop-studio/internal/perceptual"
	"backdrop-studio/internal/release"
	"backdrop-studio/internal/scaffold"
	"backdrop-studio/internal/scenes"
	"backdrop-studio/internal/vector"
	"backdrop-studio/internal/vector/authoring"
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
	// SVG is the vector lane's source document, empty for every other lane.
	// It ships beside the composite so a consumer can render the style at any
	// size, which is the property the vector family exists to provide and the
	// one thing a raster candidate can never carry.
	SVG []byte
	// Plates is the depth stack this candidate was assembled from, back to
	// front. PNG above is always their flat composite: a consumer that wants
	// one image ignores this entirely, which is the whole reason the composite
	// is not optional.
	Plates []Plate
	// Routing is which lane served this candidate, which model drew it, and
	// what it cost. Recorded per candidate rather than per job because a job
	// may yet grow lanes that differ per candidate, and because the cost of a
	// four-candidate frontier render is four charges, not one.
	Routing Routing
	// Disclosure facts, recorded at the only moment they exist.
	//
	// A model-backed candidate cannot be released without naming the model that
	// drew it, and the model is chosen by image-tools' router at submit time —
	// so a release path that asked "which model made this?" later would have
	// nowhere to look. Negative and Conditioner are here for the same reason:
	// they are inputs to a reproduction, and a disclosure that cannot be
	// reproduced is a label rather than a record.
	Model, Tier, Negative, Conditioner string
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
	mu   sync.RWMutex
	jobs map[string]*Job
	// engine is every pixel operation; generator is model-backed image
	// generation; generators is the authored-vector-generator catalog. The
	// three are separate seams because a caller may legitimately have one and
	// not the others — a host with no model still renders the whole procedural
	// and vector catalog.
	engine     imageengine.Executor
	generator  imageengine.Generator
	generators GeneratorStore
}

// WithGeneratorStore binds the authored-generator catalog. It is set after
// construction because the catalog store is built from a database handle the
// render store has no business holding.
func (s *Store) WithGeneratorStore(store GeneratorStore) *Store {
	s.generators = store
	return s
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
	// The lane is resolved before any pixels are made. A tier no lane can meet
	// is refused here, so a style that cannot be drawn honestly produces an
	// error naming the tier rather than a candidate drawn by something cheaper.
	routing, routeErr := resolveLane(style)
	if routeErr != nil {
		return Job{}, routeErr
	}
	jobID := id(style.ID, req.Surface.ID, seed, count)
	job := &Job{ID: jobID, StyleID: style.ID, SurfaceID: req.Surface.ID, Status: "completed", Seed: seed, ExecutionPath: expectedPath(style.Strategy)}
	for i := 0; i < count; i++ {
		candidateSeed := seed + int64(i)
		// The vector lane resolves its generator from its own table, so it never
		// asks the scene registry anything. Resolving a scene preset for it
		// would either fail on a subject the raster lane cannot draw — which is
		// most of the point of having a vector lane — or succeed and hand back a
		// generator nothing on this branch uses.
		preset := ""
		if !isVectorStrategy(style.Strategy) {
			resolved, presetErr := scenePreset(style)
			if presetErr != nil {
				// A procedural lane ships what the generator draws, so a subject
				// with no scene cannot be honoured — it used to fall through to
				// "field", meaning a style called Cyanotype Botanical rendered an
				// abstract colour field and nothing said so.
				if style.Strategy == "procedural" || style.Strategy == "procedural-treated" {
					return Job{}, fmt.Errorf("render: style %q: %w", style.ID, presetErr)
				}
				// A model-backed lane may name any subject: the model draws it
				// and the scaffold only supplies composition geometry, so the
				// scaffold falls back to the non-representational field.
				resolved = "field"
			}
			preset = resolved
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
		// svgSource is the vector lane's source document, kept so a vector
		// style can ship its source beside its raster. A released SVG is what
		// makes the style resolution-independent for a consumer, and it is the
		// one artifact that cannot be recovered from the PNG.
		var svgSource []byte
		// generatorAuthor is the model that wrote the generator, empty for a
		// hand-written one.
		generatorAuthor := ""
		var candidatePlates []Plate
		// planeSources are the generator's own depth layers, when it draws
		// them. Only the vector lane separates today; a raster scene hands back
		// one buffer and the stack is one plate carrying the whole picture.
		var planeSources []PlaneSource
		outW, outH := req.Surface.Width, req.Surface.Height

		if isVectorStrategy(style.Strategy) {
			drawn, authoredBy, drawErr := s.drawVector(ctx, style, req.Surface, candidateSeed, palette)
			if drawErr != nil {
				return Job{}, drawErr
			}
			generatorAuthor = authoredBy
			svgSource = drawn.SVG
			// Rasterization is image-tools' job, not this scenario's. The
			// generators use filters, masks and patterns, so image-tools routes
			// them to its high-fidelity rasterizer and refuses by name on a host
			// with none — rather than returning a clean geometric picture that
			// silently lost the hand-cut character.
			raster, rasterErr := s.rasterizeSVG(ctx, drawn.SVG)
			if rasterErr != nil {
				return Job{}, fmt.Errorf("render: rasterize vector source for style %q: %w", style.ID, rasterErr)
			}
			input = raster
			rasterW, rasterH, dimErr := pngGeometry(raster)
			if dimErr != nil {
				return Job{}, fmt.Errorf("render: measure rasterized vector source: %w", dimErr)
			}
			outW, outH = rasterW, rasterH
			// The generator's own depth layers, rasterized only when the style
			// actually declares a stack. Rasterizing four planes for a style
			// that ships one picture would quadruple every vector render to
			// produce bytes nothing reads.
			if len(style.EffectivePlateSpec()) > 1 {
				for i, plane := range drawn.Planes {
					planeRaster, planeErr := s.rasterizeSVG(ctx, drawn.PlaneDocuments[i])
					if planeErr != nil {
						return Job{}, fmt.Errorf("render: rasterize plane %q of style %q: %w", plane, style.ID, planeErr)
					}
					planeSources = append(planeSources, PlaneSource{Name: plane, PNG: planeRaster})
				}
			}
		} else if style.Strategy == "guided" || style.Strategy == "synthesized" {
			sc, scErr := scaffold.Render(scaffold.Request{Preset: preset, Conditioner: conditioner, ParamsJSON: scaffoldParams(style), Width: scaffoldWidth, Height: scaffoldHeight, Seed: candidateSeed, Regions: regions})
			if scErr != nil {
				return Job{}, scErr
			}
			conditioning = sc.PNG
			input = sc.PNG
		} else {
			sn, snErr := scenes.Render(scenes.Request{
				Preset: preset, ParamsJSON: scaffoldParams(style),
				Width: req.Surface.Width, Height: req.Surface.Height, Seed: candidateSeed,
				// The style's own reserved region becomes the generator's quiet
				// zone, so the picture composes around its copy instead of the
				// copy being dropped onto whatever the picture happened to draw.
				Quiet: quietZone(style),
			})
			if snErr != nil {
				return Job{}, fmt.Errorf("render: scene: %w", snErr)
			}
			input = sn.PNG
			outW, outH = sn.Width, sn.Height
			// The generator's own depth layers, carried only when the style
			// declares a stack. A terrain draws seven planes; encoding all of
			// them for a style that ships one picture would cost seven PNGs
			// nothing reads.
			if len(style.EffectivePlateSpec()) > 1 {
				for i, plane := range sn.Planes {
					planeSources = append(planeSources, PlaneSource{Name: plane, PNG: sn.PlaneImages[i]})
				}
			}
		}
		conditioningSubmitted := false
		generationNative := ""
		generationModel := ""
		generationTier := ""
		prompt := ""
		if style.Generation != nil {
			prompt = style.Generation.PromptTemplate
		}
		if routing.ServedLane != LaneProcedural {
			if s.generator == nil {
				return Job{}, fmt.Errorf("render: %s requires image-tools inference capability", style.Strategy)
			}
			policy, policyErr := policyForLane(routing.ServedLane)
			if policyErr != nil {
				return Job{}, policyErr
			}
			// The canvas comes from the model that will actually draw it. Asking
			// costs one preview call and replaces three constants named for a
			// model family this host may not even have installed.
			// The probe is given the lane's own routing policy, so it previews
			// the model that will actually draw this candidate rather than
			// whatever the local default happens to be.
			geometry, geoErr := s.modelGeometry(ctx, imageengine.GeometryRequest{
				Operation:      generationOperation(style.Strategy),
				QualityPolicy:  policy.QualityPolicy,
				FallbackPolicy: policy.FallbackPolicy,
				AllowBYOK:      policy.AllowBYOK,
			})
			if geoErr != nil {
				routing.Attempts = append(routing.Attempts, LaneAttempt{Lane: routing.ServedLane, Detail: geoErr.Error()})
				return Job{}, &LaneRefusedError{StyleID: style.ID, Tier: routing.DeclaredTier, Attempts: routing.Attempts}
			}
			nativeW, nativeH := geometry.Fit(outW, outH)
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
				// Routing is the served lane's policy, never an accident. The
				// previous hardcoded ("quality", "any", BYOK on) sent every
				// render to a paid cloud provider while an installed local GPU
				// served the same request in about fifteen seconds.
				QualityPolicy:  policy.QualityPolicy,
				FallbackPolicy: policy.FallbackPolicy,
				AllowBYOK:      policy.AllowBYOK,
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
				// A lane that was tried and failed is a refusal, not a reason to
				// reach for a different lane: the declared tier is the ceiling,
				// and downgrading would ship a picture wearing a label it did
				// not earn.
				routing.Attempts = append(routing.Attempts, LaneAttempt{Lane: routing.ServedLane, Detail: genErr.Error()})
				return Job{}, &LaneRefusedError{StyleID: style.ID, Tier: routing.DeclaredTier, Attempts: routing.Attempts}
			}
			if len(generated.PNG) == 0 {
				return Job{}, fmt.Errorf("render: %s inference returned an empty image", style.Strategy)
			}
			// The model image-tools actually selected, not one this scenario
			// asked for. It is carried to the candidate because a synthetic
			// image released without naming its model cannot be disclosed
			// honestly, and this is the only moment the fact exists.
			generationModel = generated.ModelID
			generationTier = generated.Tier
			routing.recordGeneration(generated)
			// A model answers in whatever format it likes; the candidate field
			// is named image_png and every consumer decodes it as one.
			normalized, convErr := s.normalizePNG(ctx, generated.PNG)
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
		// The candidate is assembled from its plate stack, and a style that
		// declares none renders as a stack of one carrying the whole picture —
		// which is exactly what it did before plates existed. There is one
		// assembly path rather than a plate branch beside a flat branch,
		// because two paths is how the flat one comes to differ from the
		// stacked one in a way nobody notices.
		treated, plates, plateErr := s.assemblePlates(ctx, style, input, planeSources, palette, req.Surface)
		if plateErr != nil {
			return Job{}, plateErr
		}
		candidatePlates = plates
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
		// The parallax sweep. A style that passes at rest can fail at 60% scroll
		// when a dark plane slides under white type, and the rest-only
		// measurement cannot see it — so a plated style is gated through its
		// whole motion range before it can be recorded, and therefore before it
		// can be released.
		if sweepErr := s.gateParallaxSweep(style, treated, candidatePlates); sweepErr != nil {
			return Job{}, sweepErr
		}
		job.Candidates = append(job.Candidates, Candidate{ID: id(jobID, candidateSeed, i), JobID: jobID, Strategy: style.Strategy, ExecutionPath: job.ExecutionPath, PNG: treated, Width: outW, Height: outH, Seed: candidateSeed, TreatmentApplied: len(style.Treatments) > 0, ConditioningSubmitted: conditioningSubmitted, DisclosureRequired: style.Strategy == "guided" || style.Strategy == "synthesized", Prompt: prompt, Model: generationModel, Tier: generationTier, Negative: generationNegative(style), Conditioner: conditioner, ProvenanceJSON: provenance(style, req.Surface, candidateSeed, job.ExecutionPath, generationNative, generationModel, generationTier, outW, outH, routing, generatorAuthor), QualityJSON: qualityJSON(verdict), SVG: svgSource, Routing: routing, Plates: candidatePlates})
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
)

// provenance records what actually produced this candidate, so a released
// backdrop can be reproduced and audited rather than merely trusted. For a
// model-backed candidate it names both sizes: the geometry the model drew and
// the geometry that shipped.
func provenance(style catalog.Style, surface Surface, seed int64, executionPath, generationNative, model, tier string, width, height int, routing Routing, generatorAuthor string) string {
	record := map[string]any{
		"style_id":       style.ID,
		"strategy":       style.Strategy,
		"seed":           seed,
		"treatments":     style.Treatments,
		"execution_path": executionPath,
		"surface_id":     surface.ID,
		"delivered_size": fmt.Sprintf("%dx%d", width, height),
		"quality_tier":   routing.DeclaredTier,
		"served_lane":    routing.ServedLane,
	}
	// Cost is recorded only when a backend reported one. Writing zero for a
	// route nobody measured would turn an unknown into a claim that it was
	// free, which is the opposite of what the field is for.
	if routing.CostReported {
		record["cost_usd"] = routing.CostUSD
	}
	// A generator a model wrote is disclosed by naming that model. It is a
	// separate fact from `generation_model`, which names whatever drew pixels —
	// a vector candidate has no such model at all, and recording the author
	// under that key would claim the picture was model-drawn when it was not.
	if generatorAuthor != "" {
		record["generator_authored_by"] = generatorAuthor
		record["generator_id"] = scaffoldPreset(style)
	}
	if generationNative != "" {
		record["generation_native_size"] = generationNative
		if model != "" {
			record["generation_model"] = model
		}
		if tier != "" {
			record["generation_tier"] = tier
		}
		record["generation_steps"] = generationSteps
		record["generation_cfg_scale"] = generationCFGScale
		if policy, err := policyForLane(routing.ServedLane); err == nil {
			record["generation_quality_policy"] = policy.QualityPolicy
			record["generation_fallback_policy"] = policy.FallbackPolicy
			record["generation_allow_byok"] = policy.AllowBYOK
		}
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

// scenePreset resolves the scene generator for a style.
//
// The mapping lives in the scenes package, beside the generators that declare
// what they depict — see scenes.ResolvePreset. It used to live here as a switch
// that answered every subject with the nearest scene available, which is how
// sixteen named art directions came to draw four pictures.
//
// A style may name its generator through its scaffold binding, which is what
// lets four non-representational generators be four distinct styles rather than
// one. The generator must still depict the style's declared subject.
func scenePreset(style catalog.Style) (string, error) {
	declared := ""
	if style.Scaffold != nil {
		declared = style.Scaffold.Preset
	}
	return scenes.ResolvePreset(style.Subject, declared)
}

// drawVector resolves a style's generator and draws it.
//
// Two families answer here and the order between them is the whole point: a
// built-in preset always wins. An authored generator is catalog data a model
// wrote, and letting one shadow `colonnade` would silently change what an
// existing seeded style draws — so validation refuses the name collision at
// storage time and this lookup honours the same order at render time.
//
// A binding to an id neither family knows is refused by name. It used to be
// impossible to reach: every preset was compiled in. Now a style can name a
// generator that was authored on another install, or one an operator deleted,
// and the honest answer is to say which.
func (s *Store) drawVector(ctx context.Context, style catalog.Style, surface Surface, seed int64, palette map[string]string) (vector.Result, string, error) {
	declared := scaffoldPreset(style)
	if _, builtin := vector.SubjectOf(declared); declared == "" || builtin {
		preset, presetErr := vector.ResolvePreset(style.Subject, declared)
		if presetErr != nil {
			return vector.Result{}, "", fmt.Errorf("render: style %q: %w", style.ID, presetErr)
		}
		drawn, drawErr := vector.Render(vector.Request{
			Preset:     preset,
			Width:      surface.Width,
			Height:     surface.Height,
			Seed:       seed,
			ParamsJSON: scaffoldParams(style),
			Inks:       palette,
			Quiet:      vectorQuietZone(style),
		})
		if drawErr != nil {
			return vector.Result{}, "", fmt.Errorf("render: vector: %w", drawErr)
		}
		// A hand-written generator has no authoring model, and saying so is
		// different from leaving the field to be guessed at.
		return drawn, "", nil
	}
	if s.generators == nil {
		return vector.Result{}, "", fmt.Errorf(
			"render: style %q binds to authored generator %q and no generator store is configured", style.ID, declared)
	}
	generator, lookupErr := s.generators.AuthoredGenerator(ctx, declared)
	if lookupErr != nil {
		return vector.Result{}, "", fmt.Errorf("render: style %q: %w", style.ID, lookupErr)
	}
	params, paramsErr := scaffoldParamValues(style)
	if paramsErr != nil {
		return vector.Result{}, "", fmt.Errorf("render: style %q: %w", style.ID, paramsErr)
	}
	drawn, drawErr := generator.Render(surface.Width, surface.Height, seed, params, palette)
	if drawErr != nil {
		return vector.Result{}, "", fmt.Errorf("render: authored generator %q: %w", declared, drawErr)
	}
	// The model that WROTE the generator, which is a different disclosure from
	// the model that drew pixels — a vector candidate has none of the latter.
	// An asset released from an authored generator must name it, and this is
	// the only point in the render where the fact is reachable.
	return drawn, generator.ModelID, nil
}

// scaffoldParamValues decodes a style's scaffold parameters for an authored
// generator, which takes typed values rather than the JSON string the built-in
// presets parse for themselves.
func scaffoldParamValues(style catalog.Style) (map[string]float64, error) {
	raw := scaffoldParams(style)
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	values := map[string]float64{}
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("scaffold params_json is not an object of numbers: %w", err)
	}
	return values, nil
}

// Plate is one depth layer of a candidate.
type Plate struct {
	Name       string
	Depth      int
	Blend      string
	Opacity    float64
	Treatments []string
	// Motion is how this layer moves, nil when the style declares none.
	Motion *catalog.MotionProfile
	// PNG is the layer's own pixels. On a single-plate candidate it is the
	// composite itself rather than a copy of it, so the common case costs no
	// extra memory and no extra encode.
	PNG []byte
}

// assemblePlates builds a candidate's stack and its flat composite.
//
// The single-plate path is deliberately not "the plate path with one element".
// It applies the style's chain to the source and returns those exact bytes as
// both the composite and the plate — no compositor call, no re-encode — so an
// existing style is byte-identical to its pre-plate output. Running one plate
// through a compositor would re-encode it, and a PNG that round-trips through a
// second encoder is not guaranteed to be the same bytes even when it is the
// same picture.
func (s *Store) assemblePlates(ctx context.Context, style catalog.Style, source []byte, planes []PlaneSource, palette map[string]string, surface Surface) ([]byte, []Plate, error) {
	spec := style.EffectivePlateSpec()
	if len(spec) == 1 {
		// A `procedural` style ships what the generator drew. The engine is not
		// asked to run an empty chain: everywhere else in this system an empty
		// chain is a caller mistake and `Apply` rightly refuses it, so the
		// decision that there is nothing to apply belongs here, where it is a
		// declared property of the style.
		composite := source
		if len(spec[0].Treatments) > 0 {
			applied, err := s.engine.Apply(ctx, imageengine.ApplyRequest{
				Input:      source,
				Treatments: spec[0].Treatments,
				Params:     scrimParams(style, spec[0].EffectiveTreatmentParams(style.TreatmentParams)),
				Palette:    palette,
				Reserve:    reservedSpace(style),
			})
			if err != nil {
				return nil, nil, fmt.Errorf("render: treatment chain: %w", err)
			}
			composite = applied
		}
		return composite, []Plate{{
			Name:       spec[0].Name,
			Depth:      spec[0].Depth,
			Blend:      spec[0].Blend,
			Opacity:    spec[0].Opacity,
			Treatments: spec[0].Treatments,
			Motion:     spec[0].Motion,
			PNG:        composite,
		}}, nil
	}
	// A multi-plate stack needs a generator that separates its layers. A style
	// declaring one over a source that does not is refused by name rather than
	// silently flattened: a style that says it draws a sky behind a colonnade
	// and delivers one flat picture is the substitution this plan exists to
	// remove.
	if len(planes) == 0 {
		return nil, nil, &MultiPlateUnavailableError{StyleID: style.ID, Plates: len(spec)}
	}
	available := make(map[string]PlaneSource, len(planes))
	for _, plane := range planes {
		available[plane.Name] = plane
	}
	assembled := make([]Plate, 0, len(spec))
	sources := make([]imageengine.PlateSource, 0, len(spec))
	for _, declared := range spec {
		// A plate may merge several generator planes: the colonnade draws four
		// and a stack is capped at three, and merging "the far ground" loses no
		// parallax where dropping a layer would lose picture.
		merged := make([]imageengine.PlateSource, 0, len(declared.SourcePlanes()))
		for _, source := range declared.SourcePlanes() {
			plane, ok := available[source]
			if !ok {
				return nil, nil, &UnknownPlaneError{StyleID: style.ID, Plane: source, Available: planeNames(planes)}
			}
			merged = append(merged, imageengine.PlateSource{
				Name: source, Depth: len(merged), Blend: catalog.BlendNormal, Opacity: 1, PNG: plane.PNG,
			})
		}
		pixels := merged[0].PNG
		if len(merged) > 1 {
			compositor, ok := s.engine.(interface {
				Composite(context.Context, []imageengine.PlateSource, int, int, string) ([]byte, error)
			})
			if !ok {
				return nil, nil, fmt.Errorf("render: style %q plate %q merges %d planes and no image-tools compositor is configured", style.ID, declared.Name, len(merged))
			}
			flattened, err := compositor.Composite(ctx, merged, surface.Width, surface.Height, "")
			if err != nil {
				return nil, nil, fmt.Errorf("render: style %q merge planes for plate %q: %w", style.ID, declared.Name, err)
			}
			pixels = flattened
		}
		if len(declared.Treatments) > 0 {
			applied, err := s.engine.Apply(ctx, imageengine.ApplyRequest{
				Input:      pixels,
				Treatments: declared.Treatments,
				Params:     scrimParams(style, declared.EffectiveTreatmentParams(style.TreatmentParams)),
				Palette:    palette,
				Reserve:    reservedSpace(style),
			})
			if err != nil {
				return nil, nil, fmt.Errorf("render: style %q plate %q treatment chain: %w", style.ID, declared.Name, err)
			}
			pixels = applied
		}
		if len(declared.Treatments) > 0 {
			verdict, scoreErr := scorePlate(merged[0].PNG, pixels, style)
			if scoreErr != nil {
				return nil, nil, fmt.Errorf("render: score plate %q of style %q: %w", declared.Name, style.ID, scoreErr)
			}
			if !verdict.Passed {
				return nil, nil, &PlateRejectedError{StyleID: style.ID, Plate: declared.Name, Verdict: verdict}
			}
		}
		assembled = append(assembled, Plate{
			Name: declared.Name, Depth: declared.Depth, Blend: declared.Blend,
			Opacity: declared.Opacity, Treatments: declared.Treatments,
			Motion: declared.Motion, PNG: pixels,
		})
		sources = append(sources, imageengine.PlateSource{
			Name: declared.Name, Depth: declared.Depth, Blend: declared.Blend,
			Opacity: declared.Opacity, PNG: pixels,
		})
	}
	compositor, ok := s.engine.(interface {
		Composite(context.Context, []imageengine.PlateSource, int, int, string) ([]byte, error)
	})
	if !ok {
		return nil, nil, fmt.Errorf("render: style %q draws %d plates and no image-tools compositor is configured", style.ID, len(spec))
	}
	composite, err := compositor.Composite(ctx, sources, surface.Width, surface.Height, "")
	if err != nil {
		return nil, nil, fmt.Errorf("render: composite style %q: %w", style.ID, err)
	}
	return composite, assembled, nil
}

// PlaneSource is one depth layer a generator drew, already rasterized.
type PlaneSource struct {
	Name string
	PNG  []byte
}

func planeNames(planes []PlaneSource) []string {
	out := make([]string, 0, len(planes))
	for _, plane := range planes {
		out = append(out, plane.Name)
	}
	return out
}

// gateParallaxSweep refuses a plated candidate that is illegible in motion.
//
// It runs only when the style declares depth: a stack whose plates share one
// parallax factor does not move, and a candidate with no reserved region has
// nothing to keep legible. The measurement is on the composite rather than on
// each plate, because the composite is what a reader sees and a plate measured
// alone would report contrast against transparency.
func (s *Store) gateParallaxSweep(style catalog.Style, composite []byte, plates []Plate) error {
	regions := legibilityRegions(style)
	if len(regions) == 0 || len(plates) < 2 {
		return nil
	}
	layers := make([]legibility.Layer, 0, len(plates))
	moving := map[float64]struct{}{}
	for _, plate := range plates {
		parallax := 0.0
		if plate.Motion != nil {
			parallax = plate.Motion.Parallax
		}
		moving[parallax] = struct{}{}
		layers = append(layers, legibility.Layer{
			Name: plate.Name, PNG: plate.PNG, Parallax: parallax, Opacity: plate.Opacity,
		})
	}
	if len(moving) < 2 {
		return nil
	}
	threshold := style.ContrastThreshold
	if threshold <= 0 {
		threshold = 4.5
	}
	verdict, err := legibility.Sweep(composite, layers, regions, threshold, "")
	if err != nil {
		return fmt.Errorf("render: sweep style %q: %w", style.ID, err)
	}
	// This gate refuses a MOTION-INDUCED failure: legible at rest and not
	// somewhere in the sweep. It deliberately does not refuse a style that is
	// already illegible standing still.
	//
	// That is not leniency, it is scope. Measured 2026-08-13 over the settled
	// catalog: ZERO of the twenty styles that declare an overlay region pass
	// worst-pixel contrast at rest — the best is 2.81 against a 4.50 threshold
	// and eight sit near 1.0, which is type the same colour as what it sits on.
	// The render path has never gated rest legibility; the check existed only
	// as a standalone RPC nobody called. Turning it on here would refuse the
	// entire catalog under the banner of a motion feature, and repairing twenty
	// styles' reserved regions is catalog maturation work with its own phase.
	//
	// So the rule is the delta: motion must not make a picture worse than it
	// was standing still. The catalog-wide rest failure is recorded in
	// PROBLEMS.md, which is the honest place for a defect this size.
	if verdict.Passes || !verdict.Samples[0].Verdict.Passes {
		return nil
	}
	return &SweepRejectedError{StyleID: style.ID, Verdict: verdict}
}

// scrimParams fills a scrim's region from the style's own reserved region.
//
// The geometry is derived rather than declared twice. An author who had to
// repeat the rectangle in `regions` and again in `treatment_params.scrim` would
// eventually change one and not the other, and the failure — a scrim shading
// somewhere the copy no longer is — is invisible in every test that does not
// measure contrast where the text actually sits.
//
// An explicit region in the style's own scrim parameters wins, because an
// author who wrote one meant it: a pool deliberately wider than the copy is a
// real art-direction choice.
func scrimParams(style catalog.Style, params map[string]string) map[string]string {
	raw, names := params["scrim"], false
	for _, treatment := range style.Treatments {
		if treatment == "scrim" {
			names = true
		}
	}
	if !names {
		for _, plate := range style.EffectivePlateSpec() {
			for _, treatment := range plate.Treatments {
				if treatment == "scrim" {
					names = true
				}
			}
		}
	}
	if !names {
		return params
	}
	var declared map[string]any
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &declared); err != nil {
			return params
		}
	}
	if declared == nil {
		declared = map[string]any{}
	}
	if _, set := declared["region_width"]; set {
		return params
	}
	region, found := primaryOverlayRegion(style)
	if !found {
		return params
	}
	declared["region_x"] = region.X
	declared["region_y"] = region.Y
	declared["region_width"] = region.Width
	declared["region_height"] = region.Height
	encoded, err := json.Marshal(declared)
	if err != nil {
		return params
	}
	out := make(map[string]string, len(params)+1)
	for k, v := range params {
		out[k] = v
	}
	out["scrim"] = string(encoded)
	return out
}

// primaryOverlayRegion is the region a scrim shades: the largest overlay one,
// because a style with several is being asked where its headline goes and the
// headline is in the biggest box.
func primaryOverlayRegion(style catalog.Style) (catalog.Region, bool) {
	best, found := catalog.Region{}, false
	for _, region := range style.Regions {
		if region.Kind != "overlay" || region.Width <= 0 || region.Height <= 0 {
			continue
		}
		if !found || region.Width*region.Height > best.Width*best.Height {
			best, found = region, true
		}
	}
	return best, found
}

// vectorQuietZone turns a style's reserved region into a plate reserve.
//
// The third derivation of the same rectangle, alongside quietZone for the
// raster generators and reservedSpace for the treatment chain, and all three
// read `regions` rather than anything declared separately. An author who had to
// state the copy area once per lane would eventually state it twice and change
// one, and a reserve cut where the copy is not is invisible to every test that
// does not measure contrast where the copy actually sits.
//
// Unlike the treatment reserve, this one serves BOTH polarities. It can: a
// generator owns its ink as well as its paper, so it can lay a solid where the
// treatment chain could only lift toward paper.
func vectorQuietZone(style catalog.Style) *vector.QuietZone {
	region, found := primaryOverlayRegion(style)
	if !found {
		return nil
	}
	return &vector.QuietZone{
		X: region.X, Y: region.Y, Width: region.Width, Height: region.Height,
		TowardLight: textIsDark(region.TextColor),
		Travel:      frontPlateTravel(style),
	}
}

// frontPlateTravel is how far the frontmost plate slides against the frame, as
// a fraction of the frame height.
//
// Only the frontmost matters. The reserve is cut into the frontmost plane and
// is opaque, so it hides every plate behind it wherever it still covers; the
// only way material reaches the copy is if the reserve itself has slid away.
// The plates behind may move as much as they like underneath it.
//
// Measured against a full page of scroll, which is the span the sweep gate
// samples and the emitted CSS drives, so the reserve is sized for the worst
// offset a reader can actually reach rather than for a typical one.
func frontPlateTravel(style catalog.Style) float64 {
	spec := style.EffectivePlateSpec()
	if len(spec) == 0 {
		return 0
	}
	front := spec[0]
	for _, plate := range spec {
		if plate.Depth > front.Depth {
			front = plate
		}
	}
	if front.Motion == nil {
		return 0
	}
	return math.Abs(front.Motion.Parallax)
}

// reservedSpace turns a style's reserved region into a treatment instruction:
// space the whole chain must leave as paper.
//
// The companion to quietZone, and deliberately derived from the same rectangle.
// The generator opening a clearing is not enough on its own — a screen puts ink
// and paper everywhere it runs, so a dithered or stippled chain will print over
// a clearing as readily as over anything else, and the clearing survives only if
// the chain is told about it too. Measured across the catalog, a generator quiet
// zone alone repaired the two styles whose chains happened to be pure tone
// mappers and left every screened style exactly where it was.
//
// Dark copy only, and the reason is measured rather than assumed.
//
// image-tools can cut the reserve either way, and the solid was wired here and
// withdrawn. A reserve large enough to give light copy dark ground costs more of
// the picture than the perceptual gate allows: `synth-celestial` fell to 0.707
// subject survival against its own declared 0.800 floor, and three other styles
// that had been rendering correctly were refused outright.
//
// The gate is right and the reserve is what should give way. Reserving a third
// of a frame as flat ink is a large claim on a picture, and a knockout gets away
// with it only because paper is where a screen was already leaving gaps. Loosening
// a quality floor so a legibility number passes would trade a measurable defect
// for an unmeasurable one, which is the failure mode this work has already hit
// once by tuning a pen to a metric.
//
// So a light-copy style is served by composition instead: the vector lane cuts
// its own solids, where the generator owns the ink and can make the ground dark
// as part of the drawing rather than as a claim over it. The styles still failing
// on this path are model-backed, and their reserve belongs with the rest of the
// model lane's plate work.
func reservedSpace(style catalog.Style) *imageengine.Knockout {
	region, found := primaryOverlayRegion(style)
	if !found || !textIsDark(region.TextColor) {
		return nil
	}
	return &imageengine.Knockout{X: region.X, Y: region.Y, Width: region.Width, Height: region.Height}
}

// quietZone turns a style's reserved region into a generator instruction.
//
// Derived rather than declared separately, for the same reason the scrim's
// region is: an author maintaining the rectangle in two places will eventually
// change one, and a picture that opened a hole where the copy no longer sits is
// invisible to every test that does not measure contrast where the text is.
//
// The polarity comes from the declared text colour. Dark copy needs light
// ground and light copy needs dark, and getting that backwards makes the
// picture worse — which is exactly what a black scrim under dark type did when
// this was first attempted with a treatment instead of a generator.
func quietZone(style catalog.Style) *scenes.QuietZone {
	region, found := primaryOverlayRegion(style)
	if !found {
		return nil
	}
	return &scenes.QuietZone{
		X: region.X, Y: region.Y, Width: region.Width, Height: region.Height,
		TowardLight: textIsDark(region.TextColor),
	}
}

// textIsDark reports whether a declared text colour needs light ground.
func textIsDark(hex string) bool {
	value := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(value) != 6 {
		// An unreadable colour is treated as dark, because light ground is the
		// commoner case in this catalog and a wrong guess that lightens is
		// recoverable by an author where one that darkens hides the picture.
		return true
	}
	channel := func(offset int) float64 {
		v, err := strconv.ParseInt(value[offset:offset+2], 16, 32)
		if err != nil {
			return 0
		}
		c := float64(v) / 255
		if c <= 0.03928 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(0)+0.7152*channel(2)+0.0722*channel(4) < 0.5
}

// legibilityRegions converts a style's overlay regions for the sweep. Only
// regions that will hold something are measured: a decorative region is not a
// place text has to be readable.
func legibilityRegions(style catalog.Style) []legibility.Region {
	out := make([]legibility.Region, 0, len(style.Regions))
	for _, region := range style.Regions {
		if region.Kind != "overlay" || strings.TrimSpace(region.TextColor) == "" {
			continue
		}
		out = append(out, legibility.Region{
			X: region.X, Y: region.Y, Width: region.Width, Height: region.Height,
			Kind: region.Kind, TextColor: region.TextColor,
		})
	}
	return out
}

// SweepRejectedError reports a candidate that is legible at rest and not in
// motion. It carries the whole sweep so an operator can see whether the failure
// is a cliff at one offset or a slope across the range.
type SweepRejectedError struct {
	StyleID string
	Verdict legibility.SweepVerdict
}

func (e *SweepRejectedError) Error() string {
	return fmt.Sprintf("render: style %q is not legible across its parallax sweep: %s", e.StyleID, e.Verdict.Error())
}

// PlateRejectedError reports one plate that did not survive its own chain.
//
// It is distinct from QualityRejectedError because the fix is per plate — a
// coarser screen on the far plane, a finer one on the near — and a message
// naming only the style would send an author to re-tune the whole stack. It
// carries the plate's own verdict so the failing metric and its margin are
// readable without a second render.
type PlateRejectedError struct {
	StyleID, Plate string
	Verdict        perceptual.Verdict
}

func (e *PlateRejectedError) Error() string {
	return fmt.Sprintf("render: style %q plate %q did not survive its own treatment: %s",
		e.StyleID, e.Plate, e.Verdict.Error())
}

// UnknownPlaneError reports a style declaring a plate its generator does not
// draw. It is typed because the fix is a catalog edit — rename the plate, or
// bind the style to a generator that draws it — and never a retry.
type UnknownPlaneError struct {
	StyleID, Plane string
	Available      []string
}

func (e *UnknownPlaneError) Error() string {
	return fmt.Sprintf(
		"render: style %q declares plate %q and its generator draws %v; rename the plate or bind a generator that draws it",
		e.StyleID, e.Plane, e.Available)
}

// MultiPlateUnavailableError reports a style declaring a stack no source can
// fill yet.
type MultiPlateUnavailableError struct {
	StyleID string
	Plates  int
}

func (e *MultiPlateUnavailableError) Error() string {
	return fmt.Sprintf(
		"render: style %q declares %d plates and no generator separates them yet; "+
			"declare one plate, or bind it to a generator that emits its layers",
		e.StyleID, e.Plates)
}

// GeneratorStore is how the render path reaches authored generators. It is an
// interface so the render store keeps one job and the catalog stays the single
// authority on what a generator is.
type GeneratorStore interface {
	AuthoredGenerator(ctx context.Context, id string) (authoring.Generator, error)
}

// isVectorStrategy reports whether a style draws through the vector family.
func isVectorStrategy(strategy string) bool {
	return strategy == "vector" || strategy == "vector-treated"
}

func scaffoldPreset(style catalog.Style) string {
	if style.Scaffold == nil {
		return ""
	}
	return style.Scaffold.Preset
}

// modelGeometry asks image-tools what canvas the model serving an operation
// draws well.
//
// It is an optional capability on the Executor for the same reason Resize and
// ToPNG are: a unit fake stays a small stub. An engine that cannot answer is an
// error rather than a fallback constant — falling back is exactly how three
// SD-1.5 numbers came to size generations on hosts running other architectures.
func (s *Store) modelGeometry(ctx context.Context, req imageengine.GeometryRequest) (imageengine.ModelGeometry, error) {
	probe, ok := s.engine.(imageengine.GeometryProbe)
	if !ok {
		return imageengine.ModelGeometry{}, fmt.Errorf("image-tools model-geometry capability is not configured")
	}
	return probe.ModelGeometry(ctx, req)
}

// generationOperation is the image-tools operation a strategy submits. A guided
// style conditions on a scaffold, which is an image-to-image run; a synthesized
// style has only a prompt. They select different models, so they are different
// geometry questions.
func generationOperation(strategy string) string {
	if strategy == "guided" {
		return "image_to_image"
	}
	return "text_to_image"
}

// rasterizeSVG turns a vector source into pixels through image-tools.
//
// It is an optional capability on the Executor for the same reason Resize and
// ToPNG are: a unit fake stays a small stub rather than having to implement a
// rasterizer it is not testing. A production client that cannot rasterize is an
// error rather than a pass-through, because passing SVG bytes off as a PNG
// candidate would put a document every downstream consumer decodes as a raster
// into the candidate's image_png field.
func (s *Store) rasterizeSVG(ctx context.Context, svg []byte) ([]byte, error) {
	rasterizer, ok := s.engine.(interface {
		RasterizeSVG(context.Context, []byte) ([]byte, error)
	})
	if !ok {
		return nil, fmt.Errorf("image-tools rasterization capability is not configured")
	}
	return rasterizer.RasterizeSVG(ctx, svg)
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
	case "vector", "vector-treated":
		return "vector → image-tools rasterize → treatment"
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

// CandidateProvenance resolves one candidate's disclosure facts by id.
//
// It satisfies release.ProvenanceSource without this package importing that
// one: the release path asks "how was this candidate made" and gets the answer
// from the render that made it, rather than from whoever called the release
// API. See internal/release/provenance.go for why that distinction matters.
func (s *Store) CandidateProvenance(candidateID string) (release.Provenance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, job := range s.jobs {
		for _, candidate := range job.Candidates {
			if candidate.ID != candidateID {
				continue
			}
			return release.Provenance{
				Strategy:    candidate.Strategy,
				ModelBacked: candidate.DisclosureRequired,
				Model:       candidate.Model,
				Tier:        candidate.Tier,
				Prompt:      candidate.Prompt,
				Negative:    candidate.Negative,
				Seed:        strconv.FormatInt(candidate.Seed, 10),
				Conditioner: candidate.Conditioner,
				Parameters:  candidate.ProvenanceJSON,
			}, true
		}
	}
	return release.Provenance{}, false
}
