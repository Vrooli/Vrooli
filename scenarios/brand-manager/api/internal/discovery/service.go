package discovery

import (
	"context"
	"encoding/json"
	"log"
	"path"
	"regexp"
	"strings"

	"brand-manager/internal/brandsurface"
)

// Scanned file locations within a scenario's source tree, in priority order.
const (
	serviceJSONPath  = ".vrooli/service.json"
	brandingJSONPath = ".vrooli/branding.json"
)

// manifestCandidates are the web-app-manifest locations the scanner probes, in
// priority order. The fleet's /public/ convention (brandsurface) nests publicly
// fetchable branding assets in ui/public/public so they stay world-readable
// behind an access-gated app; ui/public is the pre-convention location.
var manifestCandidates = []string{
	path.Join(brandsurface.PublicAssetSourceDir, "manifest.json"),
	path.Join(brandsurface.RootAssetSourceDir, "manifest.json"),
}

// assetDirCandidates are the directories the scanner lists for favicon/logo
// files, in the same priority order as manifestCandidates.
var assetDirCandidates = []string{
	brandsurface.PublicAssetSourceDir,
	brandsurface.RootAssetSourceDir,
}

// themeCSSCandidates are the CSS files the scanner probes for brand-related
// custom properties, in priority order. ui/src/design-tokens.css is the
// template's canonical brand-token file (the `--color-*` / `--font-*` contract
// scenarios declare their identity in); the rest are older or hand-rolled
// layouts.
var themeCSSCandidates = []string{
	"ui/src/design-tokens.css",
	"ui/src/styles.css",
	"ui/src/styles/theme.css",
	"ui/src/styles/brand.css",
	"ui/src/index.css",
}

// brandColorAliases maps a draft color slot onto the CSS custom properties that
// carry it, in priority order. The canonical `--color-*` names come first, then
// the `--brand-*` and bare fallbacks older scenarios use.
var brandColorAliases = []struct {
	slot  func(*Colors) *string
	props []string
}{
	{func(c *Colors) *string { return &c.Primary }, []string{"--color-primary", "--brand-primary", "--primary"}},
	{func(c *Colors) *string { return &c.Secondary }, []string{"--color-secondary", "--brand-secondary", "--secondary"}},
	{func(c *Colors) *string { return &c.Accent }, []string{"--color-accent", "--brand-accent", "--accent"}},
	{func(c *Colors) *string { return &c.Background }, []string{"--color-background", "--brand-background", "--background"}},
	{func(c *Colors) *string { return &c.Surface }, []string{"--color-surface", "--brand-surface", "--surface"}},
	{func(c *Colors) *string { return &c.Text }, []string{"--color-foreground", "--color-text", "--brand-text", "--text"}},
	{func(c *Colors) *string { return &c.Error }, []string{"--color-danger", "--color-error", "--brand-error"}},
}

// cssDeclRe matches one custom-property declaration (`--name: value;`).
var cssDeclRe = regexp.MustCompile(`(--[A-Za-z0-9_-]+)\s*:\s*([^;{}]+);`)

// cssLiteralColorRe matches a color value that is statically known — a hex,
// rgb()/rgba() or hsl()/hsla() literal. A value built from var() or color-mix()
// resolves only in the browser, so the scanner records it as signal but never
// as a color.
var cssLiteralColorRe = regexp.MustCompile(`^(#[0-9A-Fa-f]{3,8}|rgba?\([^()]*\)|hsla?\([^()]*\))$`)

// brandCSSProps are the CSS custom-property names that count as branding signal.
var brandCSSProps = []string{
	"--brand-primary", "--brand-secondary", "--brand-accent",
	"--brand-background", "--brand-text", "--color-primary", "--primary",
}

// Service is the application-layer surface the discovery handlers depend on. It
// scans a scenario for branding state (Discover) and, on Import, persists the
// inferred draft as a new brand. The handler is intentionally thin around it:
// decode → call service → translate errors.
type Service interface {
	// Discover scans the scenario and returns the draft brand it would import,
	// the sources found, an overall confidence, and suggestions — WITHOUT creating
	// anything. Returns ErrScenarioNotFound when the scenario directory is absent.
	Discover(ctx context.Context, scenario string) (Result, error)

	// Import scans the scenario and persists the inferred draft as a new brand.
	// Returns ErrScenarioNotFound when the scenario is absent, or
	// ErrNoBrandingFound when the scan matched no sources.
	Import(ctx context.Context, scenario string) (ImportResult, error)
}

type service struct {
	scanner Scanner
	brands  BrandStore
	logger  *log.Logger
}

// NewService constructs the production Service. A nil logger defaults to
// log.Default().
func NewService(scanner Scanner, brands BrandStore, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	return &service{scanner: scanner, brands: brands, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Discover(ctx context.Context, scenario string) (Result, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Result{}, ErrInvalidDiscovery{Field: "scenario_name", Reason: "required"}
	}
	exists, err := s.scanner.ScenarioExists(ctx, scenario)
	if err != nil {
		return Result{}, err
	}
	if !exists {
		return Result{}, ErrScenarioNotFound{Scenario: scenario}
	}
	return s.scan(ctx, scenario)
}

func (s *service) Import(ctx context.Context, scenario string) (ImportResult, error) {
	result, err := s.Discover(ctx, scenario)
	if err != nil {
		return ImportResult{}, err
	}
	if !result.HasSources() {
		return ImportResult{}, ErrNoBrandingFound{Scenario: scenario}
	}

	draft := result.Draft
	if draft.Name == "" {
		draft.Name = scenario
	}
	created, err := s.brands.Create(ctx, draft)
	if err != nil {
		return ImportResult{}, err
	}
	return ImportResult{
		Brand:      created,
		Sources:    result.Sources,
		Confidence: result.Confidence,
	}, nil
}

// scan runs every source probe in priority order, infers the draft brand,
// computes the mean confidence, and appends suggestions for missing facets.
func (s *service) scan(ctx context.Context, scenario string) (Result, error) {
	result := Result{Scenario: scenario, Draft: DraftBrand{Name: scenario}}

	for _, probe := range []func(context.Context, string, *Result) error{
		s.scanServiceJSON,
		s.scanBrandingJSON,
		s.scanManifest,
		s.scanThemeCSS,
		s.scanAssets,
	} {
		if err := probe(ctx, scenario, &result); err != nil {
			return Result{}, err
		}
	}

	if len(result.Sources) > 0 {
		var total float64
		for _, src := range result.Sources {
			total += src.Confidence
		}
		result.Confidence = total / float64(len(result.Sources))
	} else {
		// No branding found — present an empty draft (name only) so callers see
		// nothing was inferred.
		result.Draft = DraftBrand{}
	}

	result.Suggestions = suggestionsFor(result.Draft)
	return result, nil
}

// scanServiceJSON reads .vrooli/service.json for branding hints. The identity
// fields live in the document's `service` block (the service.schema.json shape
// brandsurface.ParseService reads); a flat document is tolerated so a
// hand-rolled or legacy file still yields signal.
func (s *service) scanServiceJSON(ctx context.Context, scenario string, result *Result) error {
	doc, err := s.readJSON(ctx, scenario, serviceJSONPath)
	if err != nil || doc == nil {
		return err
	}
	svc := doc
	if nested, ok := doc["service"].(map[string]interface{}); ok {
		svc = nested
	}
	fields := 0
	// displayName is the human brand name; name is the scenario slug and only a
	// fallback for a document that carries no display name.
	name := jsonString(svc, "displayName")
	if name == "" {
		name = jsonString(svc, "name")
	}
	if name != "" {
		result.Draft.Identity.DisplayName = name
		fields++
	}
	if desc := jsonString(svc, "description"); desc != "" {
		result.Draft.Description = desc
		fields++
	}
	if tags, ok := svc["tags"].([]interface{}); ok && len(tags) > 0 {
		fields++
	}
	if fields > 0 {
		result.Sources = append(result.Sources, Source{
			File: serviceJSONPath, Type: SourceServiceJSON, Confidence: 0.5, Fields: fields,
		})
	}
	return nil
}

// scanBrandingJSON reads .vrooli/branding.json (the legacy portable-branding
// schema): site name, tagline, theme colors, and a logo URL.
func (s *service) scanBrandingJSON(ctx context.Context, scenario string, result *Result) error {
	branding, err := s.readJSON(ctx, scenario, brandingJSONPath)
	if err != nil || branding == nil {
		return err
	}
	fields := 0
	if siteName := jsonString(branding, "site_name"); siteName != "" {
		result.Draft.Identity.DisplayName = siteName
		fields++
	}
	if tagline := jsonString(branding, "tagline"); tagline != "" {
		result.Draft.Identity.Tagline = tagline
		fields++
	}
	if theme, ok := branding["theme"].(map[string]interface{}); ok {
		fields += applyThemeColors(theme, &result.Draft.Colors)
	}
	if logoURL := jsonString(branding, "logo_url"); logoURL != "" {
		result.Draft.Identity.LogoPath = logoURL
		fields++
	}
	if fields > 0 {
		confidence := float64(fields) / 8.0 // up to 8 distinguishable fields
		if confidence > 1.0 {
			confidence = 1.0
		}
		result.Sources = append(result.Sources, Source{
			File: brandingJSONPath, Type: SourceBrandingJSON, Confidence: confidence, Fields: fields,
		})
	}
	return nil
}

// scanManifest reads the web-app manifest for PWA branding, filling only the
// fields a higher-priority source did not already set. The first candidate
// location that exists wins.
func (s *service) scanManifest(ctx context.Context, scenario string, result *Result) error {
	var (
		manifest     map[string]interface{}
		manifestPath string
	)
	for _, rel := range manifestCandidates {
		doc, err := s.readJSON(ctx, scenario, rel)
		if err != nil {
			return err
		}
		if doc != nil {
			manifest, manifestPath = doc, rel
			break
		}
	}
	if manifest == nil {
		return nil
	}
	fields := 0
	if name := jsonString(manifest, "name"); name != "" {
		if result.Draft.Identity.DisplayName == "" {
			result.Draft.Identity.DisplayName = name
		}
		fields++
	}
	if desc := jsonString(manifest, "description"); desc != "" {
		if result.Draft.Description == "" {
			result.Draft.Description = desc
		}
		fields++
	}
	if bg := jsonString(manifest, "background_color"); bg != "" {
		if result.Draft.Colors.Background == "" {
			result.Draft.Colors.Background = bg
		}
		fields++
	}
	if tc := jsonString(manifest, "theme_color"); tc != "" {
		if result.Draft.Colors.Primary == "" {
			result.Draft.Colors.Primary = tc
		}
		fields++
	}
	if fields > 0 {
		result.Sources = append(result.Sources, Source{
			File: manifestPath, Type: SourceManifest, Confidence: 0.6, Fields: fields,
		})
	}
	return nil
}

// scanThemeCSS probes the scenario's brand-token CSS files for brand-related
// custom properties and lifts every statically-known color literal into the
// draft, filling only slots a higher-priority source left empty. A property
// whose value is a var()/color-mix() expression still counts as signal, because
// only the browser can resolve it.
func (s *service) scanThemeCSS(ctx context.Context, scenario string, result *Result) error {
	for _, rel := range themeCSSCandidates {
		data, err := s.scanner.ReadFile(ctx, scenario, rel)
		if err != nil {
			return err
		}
		if data == nil {
			continue
		}
		content := string(data)
		fields := applyCSSColors(parseCSSColorVars(content), &result.Draft.Colors)
		if fields == 0 {
			// Nothing resolvable — fall back to counting the properties present,
			// so the file is still reported as branding signal.
			for _, prop := range brandCSSProps {
				if strings.Contains(content, prop) {
					fields++
				}
			}
		}
		if fields > 0 {
			result.Sources = append(result.Sources, Source{
				File: rel, Type: SourceThemeCSS, Confidence: 0.7, Fields: fields,
			})
		}
	}
	return nil
}

// scanAssets scans the public asset directories for the first favicon and logo
// files, recording each as a source and pinning its path on the draft identity.
func (s *service) scanAssets(ctx context.Context, scenario string, result *Result) error {
	type dirEntries struct {
		dir     string
		entries []string
	}
	var dirs []dirEntries
	for _, dir := range assetDirCandidates {
		entries, err := s.scanner.ListDir(ctx, scenario, dir)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			dirs = append(dirs, dirEntries{dir: dir, entries: entries})
		}
	}
	// Probe in a stable order so the first-match-per-type is deterministic.
	for _, kind := range []string{"favicon", "logo"} {
		rel := ""
		for _, d := range dirs {
			for _, entry := range d.entries {
				if strings.HasPrefix(strings.ToLower(entry), kind) {
					rel = path.Join(d.dir, entry)
					break
				}
			}
			if rel != "" {
				break
			}
		}
		if rel == "" {
			continue
		}
		result.Sources = append(result.Sources, Source{
			File: rel, Type: SourceAsset, Confidence: 0.8, Fields: 1,
		})
		switch kind {
		case "favicon":
			if result.Draft.Identity.FaviconPath == "" {
				result.Draft.Identity.FaviconPath = rel
			}
		case "logo":
			if result.Draft.Identity.LogoPath == "" {
				result.Draft.Identity.LogoPath = rel
			}
		}
	}
	return nil
}

// parseCSSColorVars extracts every custom property whose value is a statically
// known color literal, keyed by the lower-cased property name. The first
// declaration of a property wins, matching the cascade for a single file's
// :root block.
func parseCSSColorVars(content string) map[string]string {
	out := map[string]string{}
	for _, m := range cssDeclRe.FindAllStringSubmatch(content, -1) {
		name := strings.ToLower(m[1])
		value := strings.TrimSpace(m[2])
		if i := strings.Index(value, "/*"); i >= 0 {
			value = strings.TrimSpace(value[:i])
		}
		if !cssLiteralColorRe.MatchString(value) {
			continue
		}
		if _, seen := out[name]; !seen {
			out[name] = value
		}
	}
	return out
}

// applyCSSColors maps parsed custom properties onto the draft color slots,
// returning how many slots it filled. An already-set slot is left alone so a
// higher-priority source keeps precedence.
func applyCSSColors(vars map[string]string, colors *Colors) int {
	set := 0
	for _, alias := range brandColorAliases {
		dst := alias.slot(colors)
		if *dst != "" {
			continue
		}
		for _, prop := range alias.props {
			if v := vars[prop]; v != "" {
				*dst = v
				set++
				break
			}
		}
	}
	return set
}

// readJSON reads and unmarshals a scenario file into a generic map. A missing
// file or malformed JSON yields (nil, nil) — discovery treats an unreadable
// source as simply absent, never as a hard error.
func (s *service) readJSON(ctx context.Context, scenario, rel string) (map[string]interface{}, error) {
	data, err := s.scanner.ReadFile(ctx, scenario, rel)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		s.logger.Printf("discovery: skipping malformed %s in %s: %v", rel, scenario, err)
		return nil, nil
	}
	return out, nil
}

// applyThemeColors maps a legacy-branding theme block onto the draft colors,
// returning how many color slots it set.
func applyThemeColors(theme map[string]interface{}, colors *Colors) int {
	set := 0
	assign := func(dst *string, keys ...string) {
		if *dst != "" {
			return
		}
		for _, k := range keys {
			if v, ok := theme[k].(string); ok && v != "" {
				*dst = v
				set++
				return
			}
		}
	}
	assign(&colors.Primary, "primary", "primary_color")
	assign(&colors.Secondary, "secondary", "secondary_color")
	assign(&colors.Accent, "accent", "accent_color")
	assign(&colors.Background, "background", "background_color")
	assign(&colors.Text, "text", "text_color")
	return set
}

// suggestionsFor reports the branding facets that were not discovered, so a
// caller knows what to fill in by hand.
func suggestionsFor(draft DraftBrand) []string {
	var out []string
	if !draft.Colors.HasAny() {
		out = append(out, "No color system found. Consider defining brand colors.")
	}
	if draft.Identity.LogoPath == "" {
		out = append(out, "No logo found. Consider uploading a brand logo.")
	}
	if draft.Identity.Tagline == "" {
		out = append(out, "No tagline found. Consider adding a brand tagline.")
	}
	return out
}

// jsonString returns the string value at key, or "" when absent or non-string.
func jsonString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
