package catalog

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"backdrop-studio/internal/imageengine"
	"backdrop-studio/internal/scenes"
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
	if err := valid("strategy", v.Strategy, map[string]bool{"procedural": true, "procedural-treated": true, "guided": true, "synthesized": true}); err != nil {
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
	if len(v.Treatments) == 0 && v.Strategy != "procedural" {
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
	return s.insertStyle(ctx, v, OriginOperator, 0)
}

func styleVersion(v Style) int {
	if v.Version == 0 {
		return 1
	}
	return v.Version
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
	if v.Strategy != "procedural" && v.Strategy != "procedural-treated" {
		return nil
	}
	declared := ""
	if v.Scaffold != nil {
		declared = v.Scaffold.Preset
	}
	if _, err := scenes.ResolvePreset(v.Subject, declared); err != nil {
		return fmt.Errorf("catalog: style %q: %w", v.ID, err)
	}
	return nil
}

func (s *Store) insertStyle(ctx context.Context, v Style, origin string, seedVersion int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,parent_id,treatment_params,inks,quality,origin,seed_version,released) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0)`,
		v.ID, v.Name, styleVersion(v), v.Role, v.Subject, v.Lineage, v.Strategy, mustJSON(v.Treatments), mustJSON(v.Placements), mustJSON(v.Regions), v.ContrastThreshold, mustJSON(v.Scaffold), mustJSON(v.Generation), v.ParentID, mustJSON(v.TreatmentParams), mustJSON(v.Inks), mustJSON(v.Quality), origin, seedVersion)
	return err
}

func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func (s *Store) ListStyles(ctx context.Context, role, subject, treatment, lineage, placement string) ([]Style, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,parent_id,treatment_params,inks,quality,origin FROM backdrop_styles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Style
	for rows.Next() {
		var v Style
		var ts, ps, rs, scaffold, generation, tparams, inks, quality string
		if err := rows.Scan(&v.ID, &v.Name, &v.Version, &v.Role, &v.Subject, &v.Lineage, &v.Strategy, &ts, &ps, &rs, &v.ContrastThreshold, &scaffold, &generation, &v.ParentID, &tparams, &inks, &quality, &v.Origin); err != nil {
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
