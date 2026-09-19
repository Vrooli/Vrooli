// Package zones classifies scenario paths into template-declared code-layout
// zones. The scenario-level template manifest (templates/scenarios/<id>/
// manifest.json) is the SSOT for the zone taxonomy; a per-scenario
// `.vrooli/architecture.json` overlay is the only override and every deviation
// it introduces is recorded for finding-level reporting.
package zones

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"architecture-cartographer/internal/domains"
)

const (
	Unknown         = ""
	Transport       = "transport"
	Domain          = "domain"
	Substrate       = "substrate"
	CompositionRoot = "composition-root"
	CLI             = "cli"
	UI              = "ui"

	defaultTemplateID = "react-vite"
	envExtraSubstrate = "CARTOGRAPHER_NON_DOMAIN_FOLDERS"
)

type Config struct {
	PathPatterns              map[string][]string
	BuiltinSubstrateSegments  map[string]struct{}
	CompositionRootSegments   map[string]struct{}
	CoordinatingArchetypesSet map[string]struct{}
	// Deviations records every zone-taxonomy key a per-scenario
	// `.vrooli/architecture.json` overlay overrode away from the template SSOT.
	// Phase 2 zone classification surfaces these as findings: the template
	// manifest is authoritative and any local override is reported, never
	// silently honored.
	Deviations []Deviation
}

// Deviation is one per-scenario override of the template zone taxonomy.
type Deviation struct {
	// Field is the overridden zone-config key (e.g. "pathPatterns.transport",
	// "builtinSubstrateSegments").
	Field string
	// TemplateValue / OverlayValue are the SSOT vs overlay values, sorted.
	TemplateValue []string
	OverlayValue  []string
}

type Info struct {
	Path      string
	Zone      string
	Domain    string
	Archetype string
	Declared  bool
}

func LoadForScenario(scenarioDir string) Config {
	cfg := defaultConfig()
	repoRoot := findRepoRoot(scenarioDir)
	if repoRoot == "" {
		cfg.applyEnv()
		return cfg
	}
	templateID := templateIDForScenario(scenarioDir)
	manifestPath := filepath.Join(repoRoot, "templates", "scenarios", templateID, "manifest.json")
	if loaded, ok := loadFromManifest(manifestPath); ok {
		cfg = loaded
	}
	cfg.applyOverlay(filepath.Join(scenarioDir, ".vrooli", "architecture.json"))
	cfg.applyEnv()
	return cfg
}

func LoadForScenarioName(scenario string) Config {
	root := findRepoRoot("")
	if root == "" {
		return defaultConfig()
	}
	return LoadForScenario(filepath.Join(root, "scenarios", strings.TrimSpace(scenario)))
}

func (c Config) Classify(repoPath string, m domains.DerivedDomainMap) Info {
	path := strings.Trim(strings.TrimSpace(repoPath), "/")
	if path == "" {
		return Info{}
	}
	domain := m.DomainFor(path)
	archetype := archetypeFor(domain, m)
	declared := domain != ""

	if zone, owner, ok := c.matchDomainPattern(path, Transport, domain); ok {
		return Info{Path: path, Zone: zone, Domain: firstNonEmpty(domain, owner), Archetype: archetype, Declared: declared}
	}
	if zone, owner, ok := c.matchDomainPattern(path, CLI, domain); ok {
		return Info{Path: path, Zone: zone, Domain: firstNonEmpty(domain, owner), Archetype: archetype, Declared: declared}
	}
	if zone, owner, ok := c.matchDomainPattern(path, UI, domain); ok {
		return Info{Path: path, Zone: zone, Domain: firstNonEmpty(domain, owner), Archetype: archetype, Declared: declared}
	}
	if c.isCompositionRoot(path) {
		return Info{Path: path, Zone: CompositionRoot, Archetype: CompositionRoot, Declared: true}
	}
	if m.IsSharedSubstrate(path) || c.isSubstrate(path) {
		return Info{Path: path, Zone: Substrate, Declared: true}
	}
	if zone, owner, ok := c.matchDomainPattern(path, Domain, domain); ok {
		return Info{Path: path, Zone: zone, Domain: firstNonEmpty(domain, owner), Archetype: archetype, Declared: declared}
	}
	return Info{Path: path, Zone: Unknown, Domain: domain, Archetype: archetype, Declared: declared}
}

func (c Config) MayCoordinate(archetype string) bool {
	_, ok := c.CoordinatingArchetypesSet[strings.ToLower(strings.TrimSpace(archetype))]
	return ok
}

func (c Config) CoordinatingVocabulary() string {
	roles := make([]string, 0, len(c.CoordinatingArchetypesSet))
	for role := range c.CoordinatingArchetypesSet {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return strings.Join(roles, ", ")
}

func (c Config) matchDomainPattern(path, zone, knownDomain string) (string, string, bool) {
	for _, pattern := range c.PathPatterns[zone] {
		prefix, suffix, ok := splitDomainPattern(pattern)
		if !ok {
			continue
		}
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		rest := strings.TrimPrefix(path, prefix)
		owner := segment(rest)
		if owner == "" {
			continue
		}
		if suffix != "" {
			afterOwner := strings.TrimPrefix(rest, owner)
			if !strings.HasPrefix(afterOwner, suffix) && afterOwner != "" {
				continue
			}
		}
		if knownDomain != "" && owner != knownDomain {
			owner = knownDomain
		}
		return zone, owner, true
	}
	return "", "", false
}

func splitDomainPattern(pattern string) (string, string, bool) {
	pattern = strings.Trim(strings.TrimSpace(pattern), "/") + "/"
	const placeholder = "<domain>"
	i := strings.Index(pattern, placeholder)
	if i < 0 {
		return "", "", false
	}
	prefix := pattern[:i]
	suffix := strings.TrimPrefix(pattern[i+len(placeholder):], "/")
	return prefix, suffix, true
}

func (c Config) isSubstrate(path string) bool {
	for _, pattern := range c.PathPatterns[Substrate] {
		prefix, _, ok := splitSegmentPattern(pattern)
		if !ok || !strings.HasPrefix(path, prefix) {
			continue
		}
		_, ok = c.BuiltinSubstrateSegments[segment(strings.TrimPrefix(path, prefix))]
		return ok
	}
	return false
}

func (c Config) isCompositionRoot(path string) bool {
	for _, pattern := range c.PathPatterns[CompositionRoot] {
		pattern = strings.Trim(strings.TrimSpace(pattern), "/")
		if pattern != "" && (path == pattern || strings.HasPrefix(path, pattern+"/")) {
			return true
		}
	}
	for _, pattern := range c.PathPatterns[Substrate] {
		prefix, _, ok := splitSegmentPattern(pattern)
		if !ok || !strings.HasPrefix(path, prefix) {
			continue
		}
		_, ok = c.CompositionRootSegments[segment(strings.TrimPrefix(path, prefix))]
		return ok
	}
	return false
}

func splitSegmentPattern(pattern string) (string, string, bool) {
	pattern = strings.Trim(strings.TrimSpace(pattern), "/") + "/"
	const placeholder = "<segment>"
	i := strings.Index(pattern, placeholder)
	if i < 0 {
		return "", "", false
	}
	return pattern[:i], pattern[i+len(placeholder):], true
}

func (c *Config) applyEnv() {
	for _, segment := range splitCSV(os.Getenv(envExtraSubstrate)) {
		c.BuiltinSubstrateSegments[segment] = struct{}{}
	}
}

// overlay mirrors the zones block of a per-scenario `.vrooli/architecture.json`.
// Every populated field overrides the template SSOT for this scenario only and
// is recorded as a Deviation so Phase 2 can report it.
type overlay struct {
	Zones struct {
		PathPatterns             map[string][]string `json:"pathPatterns"`
		BuiltinSubstrateSegments []string            `json:"builtinSubstrateSegments"`
		CompositionRootSegments  []string            `json:"compositionRootSegments"`
		CoordinatingArchetypes   []string            `json:"coordinatingArchetypes"`
	} `json:"zones"`
}

func (c *Config) applyOverlay(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var o overlay
	if err := json.Unmarshal(data, &o); err != nil {
		return
	}
	for zone, patterns := range o.Zones.PathPatterns {
		zone = strings.TrimSpace(zone)
		if zone == "" || len(patterns) == 0 {
			continue
		}
		c.recordDeviation("pathPatterns."+zone, c.PathPatterns[zone], patterns)
		c.PathPatterns[zone] = append([]string(nil), patterns...)
	}
	if len(o.Zones.BuiltinSubstrateSegments) > 0 {
		c.recordDeviation("builtinSubstrateSegments", setKeys(c.BuiltinSubstrateSegments), o.Zones.BuiltinSubstrateSegments)
		c.BuiltinSubstrateSegments = stringSet(o.Zones.BuiltinSubstrateSegments)
	}
	if len(o.Zones.CompositionRootSegments) > 0 {
		c.recordDeviation("compositionRootSegments", setKeys(c.CompositionRootSegments), o.Zones.CompositionRootSegments)
		c.CompositionRootSegments = stringSet(o.Zones.CompositionRootSegments)
	}
	if len(o.Zones.CoordinatingArchetypes) > 0 {
		c.recordDeviation("coordinatingArchetypes", setKeys(c.CoordinatingArchetypesSet), o.Zones.CoordinatingArchetypes)
		c.CoordinatingArchetypesSet = stringSet(o.Zones.CoordinatingArchetypes)
	}
}

func (c *Config) recordDeviation(field string, templateVal, overlayVal []string) {
	tv := append([]string(nil), templateVal...)
	ov := append([]string(nil), overlayVal...)
	sort.Strings(tv)
	sort.Strings(ov)
	if strings.Join(tv, ",") == strings.Join(ov, ",") {
		return // overlay restates the template; not a deviation
	}
	c.Deviations = append(c.Deviations, Deviation{Field: field, TemplateValue: tv, OverlayValue: ov})
}

func setKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func loadFromManifest(path string) (Config, bool) {
	type manifest struct {
		Zones struct {
			PathPatterns             map[string][]string `json:"pathPatterns"`
			BuiltinSubstrateSegments []string            `json:"builtinSubstrateSegments"`
			CompositionRootSegments  []string            `json:"compositionRootSegments"`
			CoordinatingArchetypes   []string            `json:"coordinatingArchetypes"`
		} `json:"zones"`
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, false
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Config{}, false
	}
	cfg := defaultConfig()
	if len(m.Zones.PathPatterns) > 0 {
		cfg.PathPatterns = clonePatternMap(m.Zones.PathPatterns)
	}
	if len(m.Zones.BuiltinSubstrateSegments) > 0 {
		cfg.BuiltinSubstrateSegments = stringSet(m.Zones.BuiltinSubstrateSegments)
	}
	if len(m.Zones.CompositionRootSegments) > 0 {
		cfg.CompositionRootSegments = stringSet(m.Zones.CompositionRootSegments)
	}
	if len(m.Zones.CoordinatingArchetypes) > 0 {
		cfg.CoordinatingArchetypesSet = stringSet(m.Zones.CoordinatingArchetypes)
	}
	return cfg, true
}

func defaultConfig() Config {
	return Config{
		PathPatterns: map[string][]string{
			Transport:       {"api/handlers/<domain>/"},
			Domain:          {"api/internal/<domain>/"},
			CLI:             {"cli/domains/<domain>/"},
			UI:              {"ui/src/features/<domain>/"},
			CompositionRoot: {"api/internal/app/", "api/internal/module/", "api/internal/modules/"},
			Substrate:       {"api/internal/<segment>/"},
		},
		BuiltinSubstrateSegments:  stringSet([]string{"clock", "config", "database", "git", "httpc", "httpx", "middleware", "observability", "server", "suppressions", "testutil"}),
		CompositionRootSegments:   stringSet([]string{"app", "module", "modules"}),
		CoordinatingArchetypesSet: stringSet([]string{"aggregation", "composition-root", "infrastructure", "mutation", "orchestration", "provider", "service"}),
	}
}

func findRepoRoot(start string) string {
	start = filepath.Clean(start)
	if start == "." {
		if cwd, err := os.Getwd(); err == nil {
			start = cwd
		}
	}
	for dir := start; dir != "" && dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "templates", "scenarios", defaultTemplateID, "manifest.json")); err == nil {
			return dir
		}
	}
	return ""
}

func templateIDForScenario(scenarioDir string) string {
	type serviceConfig struct {
		Generation struct {
			Template struct {
				ID string `json:"id"`
			} `json:"template"`
		} `json:"generation"`
	}
	data, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return defaultTemplateID
	}
	var cfg serviceConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultTemplateID
	}
	if id := strings.TrimSpace(cfg.Generation.Template.ID); id != "" {
		return id
	}
	return defaultTemplateID
}

func archetypeFor(domain string, m domains.DerivedDomainMap) string {
	for _, d := range m.Domains {
		if d.Name == domain {
			return d.PrimaryArchetype()
		}
	}
	return ""
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func splitCSV(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func clonePatternMap(in map[string][]string) map[string][]string {
	out := make(map[string][]string, len(in))
	for k, v := range in {
		out[strings.TrimSpace(k)] = append([]string(nil), v...)
	}
	return out
}

func segment(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return path
}

func firstNonEmpty(parts ...string) string {
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			return part
		}
	}
	return ""
}
