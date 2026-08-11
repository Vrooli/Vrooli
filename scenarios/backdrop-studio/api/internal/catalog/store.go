package catalog

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
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
	Treatments, Placements           []string
	Regions                          []Region
	ContrastThreshold                float64
	Scaffold                         *ScaffoldBinding
	Generation                       *GenerationBlock
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

//go:embed schema.sql
var schemaFile []byte

func Schema() string { return string(schemaFile) }

func (s *Store) Seed(ctx context.Context) error {
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
		for _, v := range []Style{
			{ID: "horizon-ink", Name: "Horizon Ink", Version: 1, Role: "ambient", Subject: "horizon", Lineage: "wpa_poster", Treatments: []string{"duotone", "halftone"}, Placements: []string{"full_bleed", "split_panel"}, Strategy: "procedural-treated", Regions: []Region{{X: .08, Y: .12, Width: .48, Height: .28, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "arcade-noir", Name: "Arcade Noir", Version: 1, Role: "ambient", Subject: "statuary_architecture", Lineage: "metaphysical", Treatments: []string{"posterize", "grain"}, Placements: []string{"full_bleed", "corner_bleed"}, Strategy: "procedural", Regions: []Region{{X: .06, Y: .1, Width: .5, Height: .3, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "terrain-riso", Name: "Terrain Riso", Version: 1, Role: "ambient", Subject: "geological", Lineage: "riso_zine", Treatments: []string{"dither_diffusion", "duotone"}, Placements: []string{"full_bleed", "framed_inset"}, Strategy: "procedural-treated", Regions: []Region{{X: .08, Y: .1, Width: .42, Height: .25, Kind: "overlay", TextColor: "#111827"}}, ContrastThreshold: 4.5},
			{ID: "field-guided", Name: "Field Guided", Version: 1, Role: "ambient", Subject: "non_representational", Lineage: "technical_minimalism", Treatments: []string{"scrim", "grain"}, Placements: []string{"full_bleed"}, Strategy: "guided", Scaffold: &ScaffoldBinding{Preset: "field", Conditioner: "depth"}, Generation: &GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "a restrained non-representational colour field"}, Regions: []Region{{X: .08, Y: .1, Width: .5, Height: .3, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "cyanotype-botanical", Name: "Cyanotype Botanical", Version: 1, Role: "ambient", Subject: "botanical", Lineage: "cyanotype", Treatments: []string{"duotone"}, Placements: []string{"full_bleed"}, Strategy: "procedural-treated", Regions: []Region{{X: .06, Y: .08, Width: .42, Height: .28, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "bauhaus-industrial", Name: "Bauhaus Industrial", Version: 1, Role: "ambient", Subject: "industrial", Lineage: "bauhaus", Treatments: []string{"posterize"}, Placements: []string{"split_panel"}, Strategy: "procedural", Regions: []Region{{X: .08, Y: .12, Width: .4, Height: .24, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "op-art-aquatic", Name: "Op Art Aquatic", Version: 1, Role: "ambient", Subject: "aquatic", Lineage: "op_art", Treatments: []string{"halftone"}, Placements: []string{"full_bleed"}, Strategy: "procedural-treated", Regions: []Region{{X: .08, Y: .1, Width: .4, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "swiss-cartographic", Name: "Swiss Cartographic", Version: 1, Role: "ambient", Subject: "cartographic", Lineage: "swiss_international", Treatments: []string{"dither_ordered"}, Placements: []string{"framed_inset"}, Strategy: "procedural", Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "neo-brutalist-object", Name: "Neo Brutalist Object", Version: 1, Role: "ambient", Subject: "object_metaphor", Lineage: "neo_brutalist", Treatments: []string{"grain"}, Placements: []string{"corner_bleed"}, Strategy: "procedural-treated", Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "riso-atmosphere", Name: "Riso Atmosphere", Version: 1, Role: "ambient", Subject: "atmospheric", Lineage: "riso_zine", Treatments: []string{"dither_diffusion"}, Placements: []string{"full_bleed"}, Strategy: "procedural-treated", Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "vaporwave-celestial", Name: "Vaporwave Celestial", Version: 1, Role: "ambient", Subject: "celestial", Lineage: "vaporwave", Treatments: []string{"scrim"}, Placements: []string{"full_bleed"}, Strategy: "guided", Scaffold: &ScaffoldBinding{Preset: "field", Conditioner: "edge"}, Generation: &GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "a restrained celestial gradient"}, Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "scientific-interior", Name: "Scientific Interior", Version: 1, Role: "ambient", Subject: "interior", Lineage: "scientific_plate", Treatments: []string{"posterize"}, Placements: []string{"split_panel"}, Strategy: "procedural", Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "memphis-textile", Name: "Memphis Textile", Version: 1, Role: "ambient", Subject: "textile_material", Lineage: "memphis", Treatments: []string{"halftone", "grain"}, Placements: []string{"full_bleed"}, Strategy: "procedural-treated", Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "synthesized-horizon", Name: "Synthesized Horizon", Version: 1, Role: "ambient", Subject: "horizon", Lineage: "solarpunk", Treatments: []string{"scrim", "duotone"}, Placements: []string{"full_bleed"}, Strategy: "synthesized", Generation: &GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "an abstract horizon with quiet negative space"}, Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
			{ID: "constructivist-figure", Name: "Constructivist Figure", Version: 1, Role: "ambient", Subject: "figure", Lineage: "constructivist", Treatments: []string{"posterize"}, Placements: []string{"framed_inset"}, Strategy: "synthesized", Generation: &GenerationBlock{Role: "image.generate.default", Profile: "PROFILE_QUALITY_FIRST", PromptTemplate: "a non-identifiable constructivist silhouette"}, Regions: []Region{{X: .08, Y: .1, Width: .45, Height: .25, Kind: "overlay", TextColor: "#ffffff"}}, ContrastThreshold: 4.5},
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
	validTreatments := map[string]bool{"halftone": true, "line_screen": true, "risograph": true, "stipple": true, "engraving": true, "letterpress": true, "dither_ordered": true, "dither_diffusion": true, "posterize": true, "duotone": true, "tritone": true, "thermal": true, "grain": true, "scrim": true, "bloom": true, "aberration": true, "long_exposure": true, "bokeh": true, "godray": true, "solarization": true, "cross_process": true, "crt_scanline": true, "anaglyph": true, "mesh_gradient": true, "caustics": true, "noise_field": true, "metaball": true, "flow_field": true, "voronoi": true, "reaction_diffusion": true, "cellular_automata": true, "wave_function_collapse": true, "truchet": true, "l_system": true, "strange_attractor": true, "contour": true, "pixel_sort": true, "glitch": true, "displacement": true, "fluted_glass": true, "kaleidoscope": true, "slit_scan": true, "typographic_mosaic": true, "pixel": true, "photomosaic": true}
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
	_, err := s.db.ExecContext(ctx, `INSERT INTO backdrop_styles(id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation,released,payload) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,0,?)`, v.ID, v.Name, v.Version, v.Role, v.Subject, v.Lineage, v.Strategy, mustJSON(v.Treatments), mustJSON(v.Placements), mustJSON(v.Regions), v.ContrastThreshold, mustJSON(v.Scaffold), mustJSON(v.Generation), string(raw))
	return err
}
func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }

func (s *Store) ListStyles(ctx context.Context, role, subject, treatment, lineage, placement string) ([]Style, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,version,role,subject,lineage,strategy,treatments,placements,regions,contrast_threshold,scaffold,generation FROM backdrop_styles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Style
	for rows.Next() {
		var v Style
		var ts, ps, rs, scaffold, generation string
		if err := rows.Scan(&v.ID, &v.Name, &v.Version, &v.Role, &v.Subject, &v.Lineage, &v.Strategy, &ts, &ps, &rs, &v.ContrastThreshold, &scaffold, &generation); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(ts), &v.Treatments)
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
	regions TEXT NOT NULL, contrast_threshold REAL NOT NULL, scaffold TEXT NOT NULL DEFAULT 'null', generation TEXT NOT NULL DEFAULT 'null', released INTEGER NOT NULL,
  payload TEXT NOT NULL
);`
