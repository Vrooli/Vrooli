package catalog

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/scenes"
	"backdrop-studio/internal/vector"
)

type Surface struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	Placements  []string `json:"placements"`
	Authority   string   `json:"authority"`
	ConfirmedOn string   `json:"confirmed_on"`
}

type Region struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Kind      string  `json:"kind"`
	TextColor string  `json:"text_color"`
}

type ScaffoldBinding struct {
	Preset      string `json:"preset"`
	Conditioner string `json:"conditioner"`
	ParamsJSON  string `json:"params_json"`
}

// GenerationBlock declares a style's model-backed source. It names no concrete
// model, provider or credential: image-tools owns selection, and a catalog that
// pinned a model would freeze every install at whatever existed when the style
// was written.
type GenerationBlock struct {
	PromptTemplate string `json:"prompt_template"`
	Negative       string `json:"negative"`
	Model          string `json:"model,omitempty"`
	ProviderURL    string `json:"provider_url,omitempty"`
	Credential     string `json:"credential,omitempty"`
}

type Style struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Version    int      `json:"version"`
	Role       string   `json:"role"`
	Subject    string   `json:"subject"`
	Lineage    string   `json:"lineage"`
	Strategy   string   `json:"strategy"`
	ParentID   string   `json:"parent_id,omitempty"`
	Treatments []string `json:"treatments"`
	Placements []string `json:"placements"`
	Regions    []Region `json:"regions"`

	ContrastThreshold float64          `json:"contrast_threshold"`
	Scaffold          *ScaffoldBinding `json:"scaffold,omitempty"`
	Generation        *GenerationBlock `json:"generation,omitempty"`

	// Quality is this style's own perceptual bar, overriding the per-family
	// default resolved by EffectiveQuality. Nil means "the family default is
	// right for me", which is true of most styles; a deliberately extreme
	// style states its numbers here so the gate does not have to be loosened
	// for everyone to accommodate one.
	Quality *Quality `json:"quality,omitempty"`

	// TreatmentParams carries per-style parameters for the ops named in
	// Treatments, keyed by op name, each value a JSON object merged over the
	// palette-derived defaults at render time. Without it every style using
	// "halftone" produced the same screen at the same line frequency, so the
	// catalog could name an art direction but never actually express one.
	// Values may reference "$brand.*" slots, which resolve against the
	// effective palette rather than being baked in.
	TreatmentParams map[string]string `json:"treatment_params,omitempty"`

	// Inks are this style's own defaults for the "$brand.*" slots its
	// parameters reference, keyed by the same slot names brand-manager emits.
	// They are what lets a style render on a cold install with no brand bound
	// while still rendering differently once one is: the effective palette is
	// these defaults overlaid by the bound brand's tokens. A slot named by a
	// parameter and covered by neither is a hard error, never a literal on the
	// wire.
	Inks map[string]string `json:"inks,omitempty"`

	// QualityTier is the bar this style's SOURCE must meet, which the render
	// path's capability router resolves to a serving lane. Empty means
	// `procedural`, which is what every style in seed versions 1 through 7
	// actually was — the field is new, the behaviour it describes is not.
	//
	// It lives on the style because the style is the only place that knows. A
	// Truchet tiling never needs a diffusion model; a colonnade with a statue
	// and a sea behind it always does. A global switch was rejected because it
	// forces one cost profile onto a catalog whose styles differ genuinely.
	QualityTier string `json:"quality_tier,omitempty"`
	// PlateSpec declares the depth layers this style draws. Empty means one
	// plate — the whole picture — which is what every style drew before the
	// plate model existed and what most still draw. It is stored as declared
	// rather than derived from the generator so a style can ship fewer plates
	// than its generator separates, which is the common case for a style whose
	// art direction wants the sea and the shore as one mass.
	PlateSpec []PlateSpec `json:"plate_spec,omitempty"`

	// Origin distinguishes rows this binary ships from rows an operator
	// authored. Seed upgrades rewrite the former and never touch the latter.
	Origin string `json:"-"`
}

// EffectivePalette resolves the ink slots this style renders with. Brand tokens
// win over the style's declared defaults, which is the whole point: one style,
// several brands, no catalog edit.
func (v Style) EffectivePalette(brand map[string]string) map[string]string {
	out := make(map[string]string, len(v.Inks)+len(brand))
	for slot, ink := range v.Inks {
		out[slot] = ink
	}
	for slot, ink := range brand {
		if strings.TrimSpace(ink) == "" {
			continue
		}
		out[slot] = ink
	}
	return out
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

//go:embed schema.sql
var schemaFile []byte

func Schema() string { return string(schemaFile) }

func validateSurface(v Surface) error {
	if v.ID == "" || v.Kind == "" || v.Width <= 0 || v.Height <= 0 || len(v.Placements) == 0 || v.Authority == "" || v.ConfirmedOn == "" {
		return fmt.Errorf("surface %q must declare kind, positive pixel geometry, placements, authority, and confirmation date", v.ID)
	}
	return nil
}

// PutSurface records an operator-authored surface. Seed upgrades never
// overwrite it.
func (s *Store) PutSurface(ctx context.Context, v Surface) error {
	if err := validateSurface(v); err != nil {
		return err
	}
	return s.insertSurface(ctx, v, OriginOperator, 0)
}

func (s *Store) insertSurface(ctx context.Context, v Surface, origin string, seedVersion int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO backdrop_surfaces(id,name,kind,width,height,placements,authority,confirmed_on,origin,seed_version) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.Name, v.Kind, v.Width, v.Height, mustJSON(v.Placements), v.Authority, v.ConfirmedOn, origin, seedVersion)
	return err
}

func (s *Store) ListSurfaces(ctx context.Context) ([]Surface, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,kind,width,height,placements,authority,confirmed_on FROM backdrop_surfaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Surface
	for rows.Next() {
		var v Surface
		var raw string
		if err := rows.Scan(&v.ID, &v.Name, &v.Kind, &v.Width, &v.Height, &raw, &v.Authority, &v.ConfirmedOn); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &v.Placements)
		out = append(out, v)
	}
	return out, rows.Err()
}

// GetSurface resolves one surface record. Delivery geometry comes from here, so
// a missing surface must be an error rather than a silent default.
func (s *Store) GetSurface(ctx context.Context, id string) (Surface, error) {
	surfaces, err := s.ListSurfaces(ctx)
	if err != nil {
		return Surface{}, err
	}
	for _, surface := range surfaces {
		if surface.ID == id {
			return surface, nil
		}
	}
	return Surface{}, fmt.Errorf("catalog: surface %q not found", id)
}

// BrandSlots are the ink slots a style may reference. They are exactly the
// token names brand-manager's GetTokens emits, so the catalog and the palette
// authority share one vocabulary and no mapping table can drift between them.
// See docs/reference/configuration.md.
var BrandSlots = []string{
	"$brand.primary",
	"$brand.secondary",
	"$brand.accent",
	"$brand.background",
	"$brand.surface",
	"$brand.text",
	"$brand.error",
}

func knownBrandSlot(slot string) bool {
	for _, known := range BrandSlots {
		if known == slot {
			return true
		}
	}
	return false
}

// Quality tiers. They are the string form of the proto enum, and they are
// declared here because the store is what a seed file and an operator write
// against.
const (
	TierProcedural    = "procedural"
	TierLocalModel    = "local_model"
	TierFrontierModel = "frontier_model"
)

// PlateSpec is one declared depth layer.
type PlateSpec struct {
	Name  string `json:"name"`
	Depth int    `json:"depth"`
	Blend string `json:"blend,omitempty"`
	// Planes are the generator planes this plate merges, in the generator's own
	// depth order. Empty means the single plane whose name matches Name.
	//
	// The indirection is what lets a style ship fewer plates than its generator
	// separates. The colonnade draws four planes and this plan caps a stack at
	// three, and the choice between them is an art-direction one: distance and
	// headland are both "the far ground" and move together, so merging them
	// loses no parallax, while dropping either would lose a layer of the
	// picture. Without this a cap would be enforced by deleting content.
	Planes     []string `json:"planes,omitempty"`
	Opacity    float64  `json:"opacity,omitempty"`
	Treatments []string `json:"treatments,omitempty"`
	// TreatmentParams are this plate's own parameters, keyed by operation name
	// and overlaid on the style's.
	//
	// Depth-grading is exactly a parameter difference: a coarser screen on the
	// far plane and a finer one on the near is the same operation at two
	// rulings. Without per-plate parameters, two plates naming `halftone`
	// necessarily got the same ruling, so per-plate CHAINS could express "screen
	// this layer and not that one" but never "screen both, differently" — which
	// is the depth cue.
	TreatmentParams map[string]string `json:"treatment_params,omitempty"`
	// Motion is how this layer moves. Absent means it does not, which is the
	// honest default: a plate with no declared parallax is one the art
	// direction has not decided about, and guessing a factor would invent
	// depth the author did not choose.
	Motion *MotionProfile `json:"motion,omitempty"`
}

// MotionProfile is a plate's declared movement. It mirrors internal/motion's
// shape rather than importing it, so the catalog stays the authority on what a
// style declares and the motion package stays the authority on how it renders.
type MotionProfile struct {
	Parallax         float64 `json:"parallax"`
	Ambient          string  `json:"ambient,omitempty"`
	AmbientSeconds   float64 `json:"ambient_seconds,omitempty"`
	AmbientAmplitude float64 `json:"ambient_amplitude,omitempty"`
}

// EffectiveTreatmentParams overlays a plate's parameters on the style's.
//
// The style's entry is the default and the plate's the override, per operation:
// a style that tunes `grain` once should not have to repeat it on every plate
// to also tune `halftone` on one.
func (p PlateSpec) EffectiveTreatmentParams(style map[string]string) map[string]string {
	if len(p.TreatmentParams) == 0 {
		return style
	}
	merged := make(map[string]string, len(style)+len(p.TreatmentParams))
	for op, params := range style {
		merged[op] = params
	}
	for op, params := range p.TreatmentParams {
		merged[op] = params
	}
	return merged
}

// SourcePlanes are the generator planes a declared plate is built from.
func (p PlateSpec) SourcePlanes() []string {
	if len(p.Planes) == 0 {
		return []string{p.Name}
	}
	return p.Planes
}

// Blend modes a plate may declare. They mirror image-tools' compositor exactly:
// a mode this catalog could name and that scenario could not run would be a
// contract that only looks complete.
const (
	BlendNormal   = "normal"
	BlendMultiply = "multiply"
	BlendScreen   = "screen"
)

// EffectivePlateSpec is the stack a style actually renders.
//
// A style declaring none renders as one plate carrying the whole picture, at
// full opacity, blended normally — which is exactly what it did before plates
// existed. Materialising that default here rather than branching at each use
// site is what makes the single-plate path provably identical: there is one
// assembly path, and the old behaviour is a stack of length one.
func (v Style) EffectivePlateSpec() []PlateSpec {
	if len(v.PlateSpec) == 0 {
		return []PlateSpec{{Name: "composite", Depth: 0, Blend: BlendNormal, Opacity: 1, Treatments: v.Treatments}}
	}
	out := make([]PlateSpec, 0, len(v.PlateSpec))
	for _, plate := range v.PlateSpec {
		if strings.TrimSpace(plate.Blend) == "" {
			plate.Blend = BlendNormal
		}
		if plate.Opacity == 0 {
			plate.Opacity = 1
		}
		// A plate that declares no chain inherits the style's. Without the
		// default, adding a plate spec to a treated style would silently strip
		// the treatment from every plate — the style's own chain would apply to
		// nothing, and a screened style would ship as an untreated one.
		//
		// A plate that wants NO treatment over an inherited chain says so with
		// an explicit empty list in JSON, which decodes to a non-nil empty
		// slice and is distinguishable from an absent key.
		if plate.Treatments == nil {
			plate.Treatments = v.Treatments
		}
		out = append(out, plate)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Depth < out[j].Depth })
	return out
}

// EffectiveQualityTier resolves the tier a style is served at, treating an
// unset tier as procedural.
func (v Style) EffectiveQualityTier() string {
	if strings.TrimSpace(v.QualityTier) == "" {
		return TierProcedural
	}
	return v.QualityTier
}

func validateStyle(v *Style) error {
	valid := func(axis, value string, allowed map[string]bool) error {
		if !allowed[value] {
			return fmt.Errorf("catalog: invalid %s value %q", axis, value)
		}
		return nil
	}
	if v.ID == "" || v.Name == "" {
		return fmt.Errorf("catalog: style id and name are required")
	}
	if err := valid("role", v.Role, map[string]bool{"ambient": true, "focal": true, "evidential": true}); err != nil {
		return err
	}
	// `vector` is the third generator family. It sits beside the two procedural
	// strategies rather than replacing them: a mesh gradient is a raster idea
	// and a burin cut is a vector one, and neither is the other's fallback.
	if err := valid("strategy", v.Strategy, map[string]bool{"procedural": true, "procedural-treated": true, "vector": true, "vector-treated": true, "guided": true, "synthesized": true}); err != nil {
		return err
	}
	// An empty tier is the procedural default rather than an error, so every
	// style seeded before the field existed keeps describing itself correctly.
	if v.QualityTier == "" {
		v.QualityTier = TierProcedural
	}
	if err := valid("quality_tier", v.QualityTier, map[string]bool{TierProcedural: true, TierLocalModel: true, TierFrontierModel: true}); err != nil {
		return err
	}
	if err := valid("subject", v.Subject, map[string]bool{"non_representational": true, "horizon": true, "statuary_architecture": true, "interior": true, "botanical": true, "industrial": true, "atmospheric": true, "celestial": true, "aquatic": true, "geological": true, "textile_material": true, "cartographic": true, "figure": true, "object_metaphor": true}); err != nil {
		return err
	}
	if err := valid("lineage", v.Lineage, map[string]bool{"cyanotype": true, "metaphysical": true, "city_pop": true, "swiss_international": true, "bauhaus": true, "constructivist": true, "art_deco": true, "art_nouveau": true, "ukiyo_e": true, "mid_century_modern": true, "wpa_poster": true, "scientific_plate": true, "op_art": true, "psychedelic": true, "memphis": true, "demoscene": true, "vaporwave": true, "cyberpunk": true, "frutiger_aero": true, "technical_minimalism": true, "solarpunk": true, "neo_brutalist": true, "wabi_sabi": true, "riso_zine": true}); err != nil {
		return err
	}
	// An empty chain is legal for exactly one strategy, and it is the point of
	// that strategy: `procedural` ships what the generator drew. A mesh
	// gradient is finished when the generator finishes — putting a screen over
	// it would add the mechanical texture the look exists to avoid — so
	// requiring a treatment would force every such style to declare one it does
	// not want. Every other strategy must name at least one, because a
	// `procedural-treated` style with no treatment is a mislabelled
	// `procedural` one, and a model-backed style with none has no chain to
	// validate.
	// `vector` joins `procedural` in being legal with an empty chain, and for
	// the same reason: the generator draws a finished picture. A burin cut is
	// finished when the burin stops, and putting a screen over line work is how
	// the raster lane produced texture with no picture under it.
	if len(v.Treatments) == 0 && v.Strategy != "procedural" && v.Strategy != "vector" {
		return fmt.Errorf("catalog: strategy %q must declare at least one treatment", v.Strategy)
	}
	// Restricted to the operations image-tools actually implements. The
	// previous set also allowed caustics, voronoi, letterpress, l_system and
	// two dozen others that no engine serves, so a style could validate, be
	// released, and then fail or silently no-op at render time. A catalog that
	// accepts a look nothing can produce is worse than a smaller catalog.
	validTreatments := map[string]bool{
		// tonal / ink mapping
		"duotone": true, "posterize": true, "adjust": true,
		// screens
		"halftone": true, "line_screen": true, "stipple": true, "engraving": true,
		// quantization
		"dither_ordered": true, "dither_diffusion": true,
		// photographic
		"grain": true, "scrim": true, "bloom": true, "aberration": true,
		"curve": true, "defocus": true, "motion_blur": true,
		// symbolic / displacement
		"ascii_mosaic": true, "pixel_sort": true, "displacement": true,
	}
	for _, treatment := range v.Treatments {
		if err := valid("treatment", treatment, validTreatments); err != nil {
			return err
		}
	}
	// A plate spec is checked as a whole rather than per entry, because the
	// mistakes that matter are relational: two plates claiming one depth, a
	// blend nothing can run, a treatment the engine does not implement.
	if err := validatePlateSpec(v.ID, v.PlateSpec, validTreatments); err != nil {
		return err
	}
	validPlacements := map[string]bool{"full_bleed": true, "split_panel": true, "framed_inset": true, "corner_bleed": true, "type_mask": true, "device_center": true, "caption_above_device": true, "caption_below_device": true, "caption_only": true, "feature_graphic": true}
	for _, p := range v.Placements {
		if err := valid("placement", p, validPlacements); err != nil {
			return err
		}
	}
	// Parameters are checked against the engine's wire contract here, at the
	// only moment the author can still fix them. A style whose parameters the
	// engine will reject used to store cleanly and fail at first render.
	if err := imageengine.ValidateChain(v.Treatments, v.TreatmentParams); err != nil {
		return fmt.Errorf("catalog: style %q: %w", v.ID, err)
	}
	if err := validateInks(*v); err != nil {
		return err
	}
	for i, r := range v.Regions {
		if r.X < 0 || r.Y < 0 || r.Width <= 0 || r.Height <= 0 || r.X+r.Width > 1 || r.Y+r.Height > 1 {
			return fmt.Errorf("catalog: invalid region %d geometry", i)
		}
		if r.Kind != "overlay" && r.Kind != "occlusion" {
			return fmt.Errorf("catalog: invalid region %d kind %q", i, r.Kind)
		}
	}
	if v.Strategy == "vector" || v.Strategy == "vector-treated" {
		if v.Generation != nil {
			return fmt.Errorf("catalog: strategy %q cannot carry a generation block", v.Strategy)
		}
		if v.Scaffold != nil && v.Scaffold.Conditioner != "" {
			return fmt.Errorf("catalog: strategy %q cannot declare a scaffold conditioner: there is no model to condition", v.Strategy)
		}
	}
	if v.Strategy == "procedural" || v.Strategy == "procedural-treated" {
		// A procedural style may carry a scaffold block, but only the parts of
		// it that mean something without a model: `preset` selects which scene
		// generator draws the style — the mechanism that lets four
		// non-representational generators be four distinct styles rather than
		// one — and `params_json` tunes it, which the render path has always
		// passed through. `conditioner` names a ControlNet preprocessor and is
		// meaningless with no model to condition, so declaring one is a
		// statement about this style that is not true.
		//
		// The previous rule forbade the whole block, which contradicted a
		// render path that was already reading params_json out of it.
		if v.Scaffold != nil && v.Scaffold.Conditioner != "" {
			return fmt.Errorf("catalog: strategy %q cannot declare a scaffold conditioner: there is no model to condition", v.Strategy)
		}
		if v.Generation != nil {
			return fmt.Errorf("catalog: strategy %q cannot carry a generation block", v.Strategy)
		}
	}
	if v.Strategy == "guided" {
		if v.Scaffold == nil || v.Scaffold.Preset == "" {
			return fmt.Errorf("catalog: guided strategy requires a scaffold")
		}
		if v.Generation == nil {
			return fmt.Errorf("catalog: guided strategy requires a generation block")
		}
	}
	if v.Strategy == "synthesized" && v.Generation == nil {
		return fmt.Errorf("catalog: synthesized strategy requires a generation block")
	}
	if v.Generation != nil {
		if v.Generation.PromptTemplate == "" {
			return fmt.Errorf("catalog: generation block requires a prompt_template")
		}
		if v.Generation.Model != "" || v.Generation.ProviderURL != "" || v.Generation.Credential != "" {
			return fmt.Errorf("catalog: generation block may not name a concrete model, provider URL, or credential")
		}
	}
	if v.Scaffold != nil && (v.Scaffold.Preset == "" || (v.Scaffold.Conditioner != "" && v.Scaffold.Conditioner != "depth" && v.Scaffold.Conditioner != "edge")) {
		return fmt.Errorf("catalog: scaffold requires a valid preset and conditioner")
	}
	if v.ContrastThreshold <= 0 {
		v.ContrastThreshold = 4.5
	}
	return nil
}

// validateInks proves, at write time, that every "$brand.*" slot the style's
// parameters reference can actually be resolved on a cold install. This is the
// write-side half of the fail-closed contract: without it, a style with a typo
// in a slot name would store cleanly and then fail its first render with a
// colour error from image-tools — which is exactly the defect that shipped
// twelve unrenderable styles.
func validateInks(v Style) error {
	for slot := range v.Inks {
		if !knownBrandSlot(slot) {
			return fmt.Errorf("catalog: style %q declares ink for unknown slot %q (known: %s)", v.ID, slot, strings.Join(BrandSlots, ", "))
		}
	}
	for _, op := range sortedKeys(v.TreatmentParams) {
		var fields map[string]any
		if err := json.Unmarshal([]byte(v.TreatmentParams[op]), &fields); err != nil {
			continue // ValidateChain already reported the shape
		}
		for _, field := range sortedKeys(fields) {
			text, ok := fields[field].(string)
			if !ok || !strings.HasPrefix(text, "$brand.") {
				continue
			}
			if !knownBrandSlot(text) {
				return fmt.Errorf("catalog: style %q operation %q references unknown brand slot %q (known: %s)", v.ID, op, text, strings.Join(BrandSlots, ", "))
			}
			if strings.TrimSpace(v.Inks[text]) == "" {
				return fmt.Errorf("catalog: style %q operation %q references %q with no declared ink default; a style must render without a brand bound", v.ID, op, text)
			}
		}
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// CreateStyle records an operator-authored style. Seed upgrades never overwrite
// it, which is what makes upgrading the catalog safe for someone who has built
// their own art direction on top of it.
func (s *Store) CreateStyle(ctx context.Context, v Style) error {
	if err := validateStyle(&v); err != nil {
		return err
	}
	if err := ValidateSubjectCoherence(v); err != nil {
		return err
	}
	if err := ValidateTierCoherence(v); err != nil {
		return err
	}
	return s.insertStyle(ctx, v, OriginOperator, 0)
}

func styleVersion(v Style) int {
	if v.Version == 0 {
		return 1
	}
	return v.Version
}

// validatePlateSpec refuses a stack that could not composite.
//
// Every rule here is one the compositor would otherwise discover at render
// time, when the only honest response is to fail a picture someone asked for.
// A catalog that accepts a stack nothing can merge is worse than a smaller one.
func validatePlateSpec(styleID string, spec []PlateSpec, validTreatments map[string]bool) error {
	if len(spec) == 0 {
		return nil
	}
	if len(spec) > maxPlates {
		return fmt.Errorf("catalog: style %q declares %d plates; the compositor takes at most %d", styleID, len(spec), maxPlates)
	}
	depths := map[int]string{}
	names := map[string]bool{}
	// sources tracks which plate claimed each generator plane. A plane in two
	// plates would be drawn twice, which composites to a different picture than
	// the generator drew and is never what an author meant.
	sources := map[string]string{}
	for _, plate := range spec {
		name := strings.TrimSpace(plate.Name)
		if name == "" {
			return fmt.Errorf("catalog: style %q declares a plate with no name; a plate names what it depicts", styleID)
		}
		if names[name] {
			return fmt.Errorf("catalog: style %q declares two plates named %q", styleID, name)
		}
		names[name] = true
		if other, taken := depths[plate.Depth]; taken {
			return fmt.Errorf("catalog: style %q gives plates %q and %q the same depth %d; the stack order would be arbitrary",
				styleID, other, name, plate.Depth)
		}
		depths[plate.Depth] = name
		switch strings.TrimSpace(plate.Blend) {
		case "", BlendNormal, BlendMultiply, BlendScreen:
		default:
			return fmt.Errorf("catalog: style %q plate %q declares blend %q; the compositor runs %q, %q and %q",
				styleID, name, plate.Blend, BlendNormal, BlendMultiply, BlendScreen)
		}
		if plate.Opacity < 0 || plate.Opacity > 1 {
			return fmt.Errorf("catalog: style %q plate %q declares opacity %g; it must be between 0 and 1", styleID, name, plate.Opacity)
		}
		for _, treatment := range plate.Treatments {
			if !validTreatments[treatment] {
				return fmt.Errorf("catalog: style %q plate %q names treatment %q, which no engine operation implements", styleID, name, treatment)
			}
		}
		if plate.Motion != nil {
			if plate.Motion.Parallax < 0 || plate.Motion.Parallax > 1 {
				return fmt.Errorf("catalog: style %q plate %q declares parallax %g; it is a fraction of the viewport's travel and must be between 0 and 1",
					styleID, name, plate.Motion.Parallax)
			}
			switch strings.TrimSpace(plate.Motion.Ambient) {
			case "", "drift", "sway", "breathe":
			default:
				return fmt.Errorf("catalog: style %q plate %q declares ambient %q; the emitter renders %q, %q and %q",
					styleID, name, plate.Motion.Ambient, "drift", "sway", "breathe")
			}
			if plate.Motion.Ambient != "" {
				// A loop a reader can time is a distraction rather than an
				// atmosphere, and one they can measure against the frame is a
				// wobble rather than a breath.
				if plate.Motion.AmbientSeconds < 8 {
					return fmt.Errorf("catalog: style %q plate %q loops every %gs; an ambient motion a reader can time reads as a distraction, so the floor is 8s",
						styleID, name, plate.Motion.AmbientSeconds)
				}
				if plate.Motion.AmbientAmplitude <= 0 || plate.Motion.AmbientAmplitude > 0.05 {
					return fmt.Errorf("catalog: style %q plate %q declares ambient amplitude %g; it is a fraction of the short edge and above 0.05 the picture wobbles rather than breathes",
						styleID, name, plate.Motion.AmbientAmplitude)
				}
			}
		}
		for operation := range plate.TreatmentParams {
			if !validTreatments[operation] {
				return fmt.Errorf("catalog: style %q plate %q parameterises %q, which no engine operation implements", styleID, name, operation)
			}
			declared := false
			for _, treatment := range plate.Treatments {
				if treatment == operation {
					declared = true
				}
			}
			// Parameters for an operation this plate does not run are dead
			// weight that reads as intent. An author who tuned a ruling and
			// then removed the screen should learn it here, not by wondering
			// why the picture never changed.
			if !declared && len(plate.Treatments) > 0 {
				return fmt.Errorf("catalog: style %q plate %q parameterises %q but does not run it", styleID, name, operation)
			}
		}
		for _, source := range plate.Planes {
			if strings.TrimSpace(source) == "" {
				return fmt.Errorf("catalog: style %q plate %q lists an empty source plane", styleID, name)
			}
			if claimed, taken := sources[source]; taken {
				return fmt.Errorf("catalog: style %q gives generator plane %q to both %q and %q; a plane belongs to one plate",
					styleID, source, claimed, name)
			}
			sources[source] = name
		}
	}
	return nil
}

// maxPlates is the ceiling this plan sets on a stack. It is a plan boundary
// rather than a technical limit — the field is typed as a list so raising it
// needs no migration — and it exists so the first plate work cannot quietly
// become an unbounded layer editor.
const maxPlates = 3

// ValidateTierCoherence proves a style's quality tier and its strategy agree.
//
// A model-backed strategy at the procedural tier declares that it needs no
// model while having nothing but a prompt — there is no picture to ship. A
// generator-drawn strategy at a model tier declares a spend authorisation
// nothing will ever use. It is this pairing that makes "a procedural style
// never reaches a model" a shape the catalog cannot express, rather than a
// behaviour a later edit could regress.
//
// Like ValidateSubjectCoherence, it is judged over the settled catalog rather
// than per seed version: the field postdates most of the shipped versions, and
// a shipped seed value is never edited in place.
func ValidateTierCoherence(v Style) error {
	tier := v.EffectiveQualityTier()
	modelBacked := v.Strategy == "guided" || v.Strategy == "synthesized"
	if modelBacked && tier == TierProcedural {
		return fmt.Errorf("catalog: style %q has strategy %q, which is drawn by a model, but declares the %q tier; declare %q or %q",
			v.ID, v.Strategy, TierProcedural, TierLocalModel, TierFrontierModel)
	}
	if !modelBacked && tier != TierProcedural {
		return fmt.Errorf("catalog: style %q has strategy %q, which is drawn in-process, but declares the %q tier; a generator reaches no model",
			v.ID, v.Strategy, tier)
	}
	return nil
}

// ValidateSubjectCoherence proves a procedural style names a subject some
// generator actually depicts.
//
// Without it the render lane substituted the nearest scene it had: `interior`
// rendered an arcade, `cartographic` rendered a terrain, `aquatic` rendered a
// horizon. The catalog named sixteen art directions and drew four, and nothing
// anywhere said so.
//
// It is deliberately NOT part of validateStyle, which every seed version is
// checked against as it is applied. A seed version is an immutable historical
// record; re-judging history by today's rules means every tightening of the
// rules breaks the ability to bootstrap from scratch. The guarantee that
// matters is about the catalog someone renders from, not about every state it
// passed through — so this runs on an operator's write, at the moment they can
// still choose differently, and over the final seeded state, where a stale row
// that no later version corrected is a real defect.
func ValidateSubjectCoherence(v Style) error {
	declared := ""
	if v.Scaffold != nil {
		declared = v.Scaffold.Preset
	}
	switch v.Strategy {
	case "procedural", "procedural-treated":
		if _, err := scenes.ResolvePreset(v.Subject, declared); err != nil {
			return fmt.Errorf("catalog: style %q: %w", v.ID, err)
		}
	case "vector", "vector-treated":
		if _, err := vector.ResolvePreset(v.Subject, declared); err != nil {
			return fmt.Errorf("catalog: style %q: %w", v.ID, err)
		}
	}
	return nil
}

func (s *Store) insertStyle(ctx context.Context, v Style, origin string, seedVersion int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,parent_id,treatment_params,inks,quality,quality_tier,plate_spec,origin,seed_version,released) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0)`,
		v.ID, v.Name, styleVersion(v), v.Role, v.Subject, v.Lineage, v.Strategy, mustJSON(v.Treatments), mustJSON(v.Placements), mustJSON(v.Regions), v.ContrastThreshold, mustJSON(v.Scaffold), mustJSON(v.Generation), v.ParentID, mustJSON(v.TreatmentParams), mustJSON(v.Inks), mustJSON(v.Quality), v.EffectiveQualityTier(), mustJSON(v.PlateSpec), origin, seedVersion)
	return err
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func (s *Store) ListStyles(ctx context.Context, role, subject, treatment, lineage, placement string) ([]Style, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,parent_id,treatment_params,inks,quality,quality_tier,plate_spec,origin FROM backdrop_styles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Style
	for rows.Next() {
		var v Style
		var ts, ps, rs, scaffold, generation, tparams, inks, quality, plateSpec string
		if err := rows.Scan(&v.ID, &v.Name, &v.Version, &v.Role, &v.Subject, &v.Lineage, &v.Strategy, &ts, &ps, &rs, &v.ContrastThreshold, &scaffold, &generation, &v.ParentID, &tparams, &inks, &quality, &v.QualityTier, &plateSpec, &v.Origin); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(ts), &v.Treatments)
		if tparams != "" && tparams != "null" {
			_ = json.Unmarshal([]byte(tparams), &v.TreatmentParams)
		}
		if inks != "" && inks != "null" {
			_ = json.Unmarshal([]byte(inks), &v.Inks)
		}
		_ = json.Unmarshal([]byte(ps), &v.Placements)
		_ = json.Unmarshal([]byte(rs), &v.Regions)
		if scaffold != "null" && scaffold != "" {
			v.Scaffold = &ScaffoldBinding{}
			_ = json.Unmarshal([]byte(scaffold), v.Scaffold)
		}
		if generation != "null" && generation != "" {
			v.Generation = &GenerationBlock{}
			_ = json.Unmarshal([]byte(generation), v.Generation)
		}
		if plateSpec != "null" && plateSpec != "" {
			_ = json.Unmarshal([]byte(plateSpec), &v.PlateSpec)
		}
		if quality != "null" && quality != "" {
			v.Quality = &Quality{}
			_ = json.Unmarshal([]byte(quality), v.Quality)
		}
		if role != "" && v.Role != role || subject != "" && v.Subject != subject || lineage != "" && v.Lineage != lineage || treatment != "" && !contains(v.Treatments, treatment) || placement != "" && !contains(v.Placements, placement) {
			continue
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) GetStyle(ctx context.Context, id string) (Style, error) {
	styles, err := s.ListStyles(ctx, "", "", "", "", "")
	if err != nil {
		return Style{}, err
	}
	for _, style := range styles {
		if style.ID == id {
			return style, nil
		}
	}
	return Style{}, fmt.Errorf("catalog: style %q not found", id)
}

func contains(xs []string, w string) bool {
	for _, x := range xs {
		if x == w {
			return true
		}
	}
	return false
}

func (s *Store) TouchStyle(ctx context.Context, id string) error {
	var released bool
	if err := s.db.QueryRowContext(ctx, `SELECT released FROM backdrop_styles WHERE id=?`, id).Scan(&released); err != nil {
		return err
	}
	if released {
		return fmt.Errorf("catalog: style version %q is immutable because a released backdrop references it", id)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE backdrop_styles SET version=version+1 WHERE id=?`, id)
	return err
}

// ForkStyle creates a new style by changing exactly one declared axis. A fork
// is intentionally narrow: changing several axes destroys the lineage signal
// that makes catalog families useful to an operator.
func (s *Store) ForkStyle(ctx context.Context, parentID, childID string, changes map[string]string) (Style, error) {
	parent, err := s.GetStyle(ctx, parentID)
	if err != nil {
		return Style{}, err
	}
	if childID == "" {
		return Style{}, fmt.Errorf("catalog: fork child id is required")
	}
	if len(changes) != 1 {
		return Style{}, fmt.Errorf("catalog: a fork must mutate exactly one axis")
	}
	child := parent
	child.ID, child.ParentID, child.Version = childID, parent.ID, 1
	for axis, value := range changes {
		switch axis {
		case "role":
			child.Role = value
		case "subject":
			child.Subject = value
		case "lineage":
			child.Lineage = value
		case "strategy":
			child.Strategy = value
		case "treatment":
			child.Treatments = []string{value}
			// A fork that keeps parameters for operations the child no longer
			// runs would fail ValidateChain, so the chain change carries its
			// own parameter set.
			child.TreatmentParams = map[string]string{}
		default:
			return Style{}, fmt.Errorf("catalog: %q is not a forkable axis", axis)
		}
	}
	if err := s.CreateStyle(ctx, child); err != nil {
		return Style{}, err
	}
	return child, nil
}

type StylePack struct {
	Version int     `json:"version"`
	Styles  []Style `json:"styles"`
}

func ExportStylePack(styles []Style) ([]byte, error) {
	return json.MarshalIndent(StylePack{Version: 1, Styles: styles}, "", "  ")
}

func ImportStylePack(data []byte) ([]Style, error) {
	var pack StylePack
	if err := json.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("catalog: invalid style pack: %w", err)
	}
	if pack.Version != 1 {
		return nil, fmt.Errorf("catalog: unsupported style pack version %d", pack.Version)
	}
	for i := range pack.Styles {
		if err := validateStyle(&pack.Styles[i]); err != nil {
			return nil, err
		}
	}
	return pack.Styles, nil
}
