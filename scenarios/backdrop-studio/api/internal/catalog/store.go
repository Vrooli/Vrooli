package catalog

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"

	"backdrop-studio/internal/imageengine"
)

type Surface struct {
	ID, Name, Kind         string
	Width, Height          int
	Placements             []string
	Authority, ConfirmedOn string
}

type Region struct {
	X, Y, Width, Height float64
	Kind, TextColor     string
}

type ScaffoldBinding struct {
	Preset, Conditioner, ParamsJSON string
}

type GenerationBlock struct {
	Role, Profile, PromptTemplate, Negative string
	Model, ProviderURL, Credential          string
}

type Style struct {
	ID, Name                         string
	Version                          int
	Role, Subject, Lineage, Strategy string
	ParentID                         string
	Treatments, Placements           []string
	Regions                          []Region
	ContrastThreshold                float64
	Scaffold                         *ScaffoldBinding
	Generation                       *GenerationBlock
	// TreatmentParams carries per-style parameters for the ops named in
	// Treatments, keyed by op name, each value a JSON object merged over the
	// palette-derived defaults at render time. Without it every style using
	// "halftone" produced the same screen at the same line frequency, so the
	// catalog could name an art direction but never actually express one.
	// Values may reference "$brand.*" slots, which resolve against the active
	// brand rather than being baked in.
	TreatmentParams map[string]string
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

//go:embed schema.sql
var schemaFile []byte

func Schema() string { return string(schemaFile) }

func (s *Store) Seed(ctx context.Context) error {
	// Keep databases created before lineage was introduced readable while the
	// canonical schema remains the source for fresh installs.
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE backdrop_styles ADD COLUMN parent_id TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE backdrop_styles ADD COLUMN treatment_params TEXT NOT NULL DEFAULT '{}'`)
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM backdrop_surfaces").Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		seeds := []Surface{
			{ID: "web.hero", Name: "Landing page hero", Kind: "product", Width: 1440, Height: 720, Placements: []string{"full_bleed", "split_panel", "framed_inset", "corner_bleed"}, Authority: "Backdrop Studio product geometry", ConfirmedOn: "2026-08-11"},
			{ID: "web.hero-mobile", Name: "Mobile landing page hero", Kind: "product", Width: 390, Height: 844, Placements: []string{"full_bleed", "framed_inset"}, Authority: "Backdrop Studio product geometry", ConfirmedOn: "2026-08-11"},
			{ID: "web.auth-panel", Name: "Authentication panel", Kind: "product", Width: 640, Height: 900, Placements: []string{"split_panel", "full_bleed"}, Authority: "Backdrop Studio product geometry", ConfirmedOn: "2026-08-11"},
			{ID: "play.feature-graphic", Name: "Google Play feature graphic", Kind: "store", Width: 1024, Height: 500, Placements: []string{"feature_graphic"}, Authority: "Google Play Console Help — Graphic assets", ConfirmedOn: "2026-08-11"},
			{ID: "play.phone-screenshot", Name: "Google Play phone screenshot", Kind: "store", Width: 1080, Height: 1920, Placements: []string{"device_center", "caption_above_device", "caption_below_device", "caption_only"}, Authority: "Google Play Console Help — Graphic assets", ConfirmedOn: "2026-08-11"},
			{ID: "play.tablet-screenshot", Name: "Google Play tablet screenshot", Kind: "store", Width: 1920, Height: 1080, Placements: []string{"device_center", "caption_above_device", "caption_below_device", "caption_only"}, Authority: "Google Play Console Help — Graphic assets", ConfirmedOn: "2026-08-11"},
			{ID: "app-store-6.7-screenshot", Name: "App Store 6.7-inch screenshot", Kind: "store", Width: 1290, Height: 2796, Placements: []string{"device_center", "caption_above_device", "caption_below_device", "caption_only"}, Authority: "Apple App Store Connect Help — screenshot specifications", ConfirmedOn: "2026-08-11"},
			{ID: "app-store-6.5-screenshot", Name: "App Store 6.5-inch screenshot", Kind: "store", Width: 1284, Height: 2778, Placements: []string{"device_center", "caption_above_device", "caption_below_device", "caption_only"}, Authority: "Apple App Store Connect Help — screenshot specifications", ConfirmedOn: "2026-08-11"},
			{ID: "app-store-12.9-screenshot", Name: "App Store 12.9-inch screenshot", Kind: "store", Width: 2048, Height: 2732, Placements: []string{"device_center", "caption_above_device", "caption_below_device", "caption_only"}, Authority: "Apple App Store Connect Help — screenshot specifications", ConfirmedOn: "2026-08-11"},
		}
		for _, v := range seeds {
			if err := s.PutSurface(ctx, v); err != nil {
				return err
			}
		}
	}
	var styles int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM backdrop_styles").Scan(&styles); err != nil {
		return err
	}
	if styles == 0 {
		// The seeded catalog. Every entry names a subject that maps to a real
		// scene (or is model-backed), only ops image-tools implements, and —
		// critically — its own parameters. Without TreatmentParams every style
		// naming "halftone" produced an identical screen, so the catalog
		// described fifteen art directions while rendering about four.
		//
		// Ink slots stay as $brand.* so one style renders correctly per brand
		// rather than baking a palette into the catalog.
		overlay := func(text string) []Region {
			return []Region{{X: .06, Y: .18, Width: .48, Height: .40, Kind: "overlay", TextColor: text}}
		}
		const onDark, onLight = "#ffffff", "#111827"
		for _, v := range []Style{
			// ── architecture: screens over the arcade scene ───────────────
			{
				ID: "cyanotype-arcade", Name: "Cyanotype Arcade", Version: 1, Role: "ambient",
				Subject: "statuary_architecture", Lineage: "cyanotype", Strategy: "procedural-treated",
				Treatments: []string{"duotone", "halftone"}, Placements: []string{"full_bleed", "split_panel", "framed_inset"},
				TreatmentParams: map[string]string{
					"duotone":  `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
					"halftone": `{"lpi":72,"angle":15,"dot":"circle"}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "engraved-colonnade", Name: "Engraved Colonnade", Version: 1, Role: "ambient",
				Subject: "statuary_architecture", Lineage: "scientific_plate", Strategy: "procedural-treated",
				Treatments: []string{"engraving"}, Placements: []string{"framed_inset", "split_panel"},
				TreatmentParams: map[string]string{
					"engraving": `{"spacing":7,"dark":"$brand.primary","light":"$brand.background"}`,
				},
				Regions: overlay(onLight), ContrastThreshold: 4.5,
			},
			{
				ID: "op-art-interior", Name: "Op Art Interior", Version: 1, Role: "ambient",
				Subject: "interior", Lineage: "op_art", Strategy: "procedural-treated",
				Treatments: []string{"line_screen"}, Placements: []string{"full_bleed", "corner_bleed"},
				TreatmentParams: map[string]string{
					"line_screen": `{"spacing":6,"angle":45,"dark":"$brand.primary","light":"$brand.background"}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},

			// ── horizon family ───────────────────────────────────────────
			{
				ID: "riso-horizon", Name: "Riso Horizon", Version: 1, Role: "ambient",
				Subject: "horizon", Lineage: "riso_zine", Strategy: "procedural-treated",
				Treatments: []string{"dither_diffusion", "grain"}, Placements: []string{"full_bleed", "split_panel"},
				TreatmentParams: map[string]string{
					"dither_diffusion": `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
					"grain":            `{"amount":0.05,"contrast_multiplier":1.0,"seed":11}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "city-pop-horizon", Name: "City Pop Horizon", Version: 1, Role: "ambient",
				Subject: "horizon", Lineage: "city_pop", Strategy: "procedural-treated",
				Treatments: []string{"posterize", "grain"}, Placements: []string{"full_bleed", "corner_bleed"},
				TreatmentParams: map[string]string{
					"posterize": `{"levels":7,"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
					"grain":     `{"amount":0.09,"contrast_multiplier":1.10,"seed":4}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "solar-bloom-horizon", Name: "Solar Bloom Horizon", Version: 1, Role: "ambient",
				Subject: "horizon", Lineage: "solarpunk", Strategy: "procedural-treated",
				Treatments: []string{"bloom", "grain", "scrim"}, Placements: []string{"full_bleed"},
				TreatmentParams: map[string]string{
					"bloom": `{"threshold":0.62,"radius":18}`,
					"grain": `{"amount":0.06,"contrast_multiplier":1.04,"seed":9}`,
					"scrim": `{"color":"$brand.primary","opacity":0.55,"direction":"left"}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "ukiyo-tide", Name: "Ukiyo Tide", Version: 1, Role: "ambient",
				Subject: "aquatic", Lineage: "ukiyo_e", Strategy: "procedural-treated",
				Treatments: []string{"posterize", "halftone"}, Placements: []string{"full_bleed", "framed_inset"},
				TreatmentParams: map[string]string{
					"posterize": `{"levels":4,"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
					"halftone":  `{"lpi":110,"angle":30,"dot":"circle"}`,
				},
				Regions: overlay(onLight), ContrastThreshold: 4.5,
			},

			// ── terrain family ───────────────────────────────────────────
			{
				ID: "demoscene-terrain", Name: "Demoscene Terrain", Version: 1, Role: "ambient",
				Subject: "geological", Lineage: "demoscene", Strategy: "procedural-treated",
				Treatments: []string{"dither_ordered"}, Placements: []string{"full_bleed", "split_panel"},
				TreatmentParams: map[string]string{
					"dither_ordered": `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "stipple-massif", Name: "Stipple Massif", Version: 1, Role: "ambient",
				Subject: "geological", Lineage: "wpa_poster", Strategy: "procedural-treated",
				Treatments: []string{"stipple"}, Placements: []string{"framed_inset", "corner_bleed"},
				TreatmentParams: map[string]string{
					"stipple": `{"spacing":7,"seed":19,"dark":"$brand.primary","light":"$brand.background"}`,
				},
				Regions: overlay(onLight), ContrastThreshold: 4.5,
			},
			{
				ID: "swiss-contour", Name: "Swiss Contour", Version: 1, Role: "ambient",
				Subject: "cartographic", Lineage: "swiss_international", Strategy: "procedural-treated",
				Treatments: []string{"posterize", "line_screen"}, Placements: []string{"framed_inset", "split_panel"},
				TreatmentParams: map[string]string{
					"posterize":   `{"levels":5,"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
					"line_screen": `{"spacing":11,"angle":0,"dark":"$brand.primary","light":"$brand.background"}`,
				},
				Regions: overlay(onLight), ContrastThreshold: 4.5,
			},

			// ── field family ─────────────────────────────────────────────
			{
				ID: "technical-field", Name: "Technical Field", Version: 1, Role: "ambient",
				Subject: "non_representational", Lineage: "technical_minimalism", Strategy: "procedural-treated",
				Treatments: []string{"duotone", "grain"}, Placements: []string{"full_bleed", "split_panel", "corner_bleed"},
				TreatmentParams: map[string]string{
					"duotone": `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
					"grain":   `{"amount":0.04,"contrast_multiplier":1.02,"seed":2}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "ascii-field", Name: "ASCII Field", Version: 1, Role: "ambient",
				Subject: "non_representational", Lineage: "demoscene", Strategy: "procedural-treated",
				Treatments: []string{"ascii_mosaic"}, Placements: []string{"full_bleed", "split_panel"},
				TreatmentParams: map[string]string{
					"ascii_mosaic": `{"block_size":7,"dark":"$brand.primary","light":"$brand.background"}`,
				},
				Regions: overlay(onLight), ContrastThreshold: 4.5,
			},
			{
				ID: "memphis-weave", Name: "Memphis Weave", Version: 1, Role: "ambient",
				Subject: "textile_material", Lineage: "memphis", Strategy: "procedural-treated",
				Treatments: []string{"displacement", "posterize"}, Placements: []string{"full_bleed", "corner_bleed"},
				TreatmentParams: map[string]string{
					"displacement": `{"amplitude":18,"spacing":26,"seed":5}`,
					"posterize":    `{"levels":6,"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "vaporwave-drift", Name: "Vaporwave Drift", Version: 1, Role: "ambient",
				Subject: "object_metaphor", Lineage: "vaporwave", Strategy: "procedural-treated",
				Treatments: []string{"aberration", "bloom", "scrim"}, Placements: []string{"full_bleed"},
				TreatmentParams: map[string]string{
					"aberration": `{"distance":9}`,
					"bloom":      `{"threshold":0.58,"radius":22}`,
					"scrim":      `{"color":"$brand.primary","opacity":0.42,"direction":"bottom"}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},

			// ── model-backed lanes ───────────────────────────────────────
			// These name subjects with no procedural scene, which is exactly
			// why they are model-backed. Both stay unreleasable until
			// asset-studio exposes byte ingress (knw-1786507241786326657).
			{
				ID: "guided-botanical", Name: "Guided Botanical", Version: 1, Role: "ambient",
				Subject: "botanical", Lineage: "art_nouveau", Strategy: "guided",
				Treatments: []string{"duotone", "grain"}, Placements: []string{"full_bleed", "split_panel"},
				Scaffold: &ScaffoldBinding{Preset: "field", Conditioner: "depth", ParamsJSON: `{"focal_x":0.68,"depth_ramp":0.8}`},
				Generation: &GenerationBlock{
					Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST",
					PromptTemplate: "dense botanical forms pressed flat against the picture plane, hard-edged leaves, no visible brushwork, generous empty ground at left",
					Negative:       "text, watermark, signature, photorealistic skin, lens flare",
				},
				TreatmentParams: map[string]string{
					"duotone": `{"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
					"grain":   `{"amount":0.07,"contrast_multiplier":1.05,"seed":3}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
			{
				ID: "constructivist-figure", Name: "Constructivist Figure", Version: 1, Role: "ambient",
				Subject: "figure", Lineage: "constructivist", Strategy: "synthesized",
				Treatments: []string{"posterize"}, Placements: []string{"framed_inset", "split_panel"},
				Generation: &GenerationBlock{
					Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST",
					PromptTemplate: "a non-identifiable silhouetted figure in flat geometric planes, steep diagonal composition, large areas of unbroken ground",
					Negative:       "recognisable face, portrait likeness, text, watermark",
				},
				TreatmentParams: map[string]string{
					"posterize": `{"levels":4,"dark":"$brand.primary","light":"$brand.background","normalize":true}`,
				},
				Regions: overlay(onDark), ContrastThreshold: 4.5,
			},
		} {
			if err := s.CreateStyle(ctx, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) PutSurface(ctx context.Context, v Surface) error {
	if v.ID == "" || v.Kind == "" || v.Width <= 0 || v.Height <= 0 || len(v.Placements) == 0 || v.Authority == "" || v.ConfirmedOn == "" {
		return fmt.Errorf("surface %q must declare kind, positive pixel geometry, placements, authority, and confirmation date", v.ID)
	}
	p, _ := json.Marshal(v.Placements)
	_, err := s.db.ExecContext(ctx, `INSERT INTO backdrop_surfaces(id,name,kind,width,height,placements,authority,confirmed_on) VALUES(?,?,?,?,?,?,?,?)`, v.ID, v.Name, v.Kind, v.Width, v.Height, string(p), v.Authority, v.ConfirmedOn)
	return err
}

func (s *Store) ListSurfaces(ctx context.Context) ([]Surface, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,kind,width,height,placements,authority,confirmed_on FROM backdrop_surfaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func validateStyle(v Style) error {
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
	if len(v.Treatments) == 0 {
		return fmt.Errorf("catalog: treatment must contain at least one value")
	}
	// Restricted to the operations image-tools actually implements. The
	// previous set also allowed caustics, voronoi, letterpress, l_system and
	// two dozen others that no engine serves, so a style could validate, be
	// released, and then fail or silently no-op at render time. A catalog that
	// accepts a look nothing can produce is worse than a smaller catalog.
	validTreatments := map[string]bool{
		// tonal / ink mapping
		"duotone": true, "posterize": true,
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
	for i, r := range v.Regions {
		if r.X < 0 || r.Y < 0 || r.Width <= 0 || r.Height <= 0 || r.X+r.Width > 1 || r.Y+r.Height > 1 {
			return fmt.Errorf("catalog: invalid region %d geometry", i)
		}
		if r.Kind != "overlay" && r.Kind != "occlusion" {
			return fmt.Errorf("catalog: invalid region %d kind %q", i, r.Kind)
		}
	}
	if v.Strategy == "procedural" || v.Strategy == "procedural-treated" {
		if v.Generation != nil || v.Scaffold != nil {
			return fmt.Errorf("catalog: strategy %q cannot carry generation or scaffold fields", v.Strategy)
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
		if v.Generation.Role == "" || v.Generation.Profile == "" || v.Generation.PromptTemplate == "" {
			return fmt.Errorf("catalog: generation block requires role, profile, and prompt_template")
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

func (s *Store) CreateStyle(ctx context.Context, v Style) error {
	if err := validateStyle(v); err != nil {
		return err
	}
	if v.Version == 0 {
		v.Version = 1
	}
	raw, _ := json.Marshal(v)
	_, err := s.db.ExecContext(ctx, `INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,parent_id,treatment_params,released,payload) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,0,?)`, v.ID, v.Name, v.Version, v.Role, v.Subject, v.Lineage, v.Strategy, mustJSON(v.Treatments), mustJSON(v.Placements), mustJSON(v.Regions), v.ContrastThreshold, mustJSON(v.Scaffold), mustJSON(v.Generation), v.ParentID, mustJSON(v.TreatmentParams), string(raw))
	return err
}
func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func (s *Store) ListStyles(ctx context.Context, role, subject, treatment, lineage, placement string) ([]Style, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,parent_id,treatment_params FROM backdrop_styles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Style
	for rows.Next() {
		var v Style
		var ts, ps, rs, scaffold, generation, tparams string
		if err := rows.Scan(&v.ID, &v.Name, &v.Version, &v.Role, &v.Subject, &v.Lineage, &v.Strategy, &ts, &ps, &rs, &v.ContrastThreshold, &scaffold, &generation, &v.ParentID, &tparams); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(ts), &v.Treatments)
		if tparams != "" && tparams != "null" {
			_ = json.Unmarshal([]byte(tparams), &v.TreatmentParams)
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
	for _, style := range pack.Styles {
		if err := validateStyle(style); err != nil {
			return nil, err
		}
	}
	return pack.Styles, nil
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS backdrop_surfaces (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, kind TEXT NOT NULL,
  width INTEGER NOT NULL, height INTEGER NOT NULL, placements TEXT NOT NULL,
  authority TEXT NOT NULL, confirmed_on TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS backdrop_styles (
  id TEXT PRIMARY KEY, name TEXT NOT NULL, version INTEGER NOT NULL,
  role TEXT NOT NULL, subject TEXT NOT NULL, lineage TEXT NOT NULL,
  strategy TEXT NOT NULL, treatments TEXT NOT NULL, placements TEXT NOT NULL,
	regions TEXT NOT NULL, contrast_threshold REAL NOT NULL, scaffold TEXT NOT NULL DEFAULT 'null', generation TEXT NOT NULL DEFAULT 'null', parent_id TEXT NOT NULL DEFAULT '', released INTEGER NOT NULL,
  payload TEXT NOT NULL
);`
