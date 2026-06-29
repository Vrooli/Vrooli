// Package validation implements brand-manager's branding validation: a
// self-contained scan of a target scenario's on-disk branding artifacts that
// produces normalized findings + a maturity assessment. It is served through
// the shared scenario-validation/v1.ScenarioValidationService so test-genie can
// run it as the `branding` delegated phase.
//
// Rules are registered in registry.go with the surfaces they require and an
// optional deterministic fixer. A finding's AutofixAvailable flag is not set by
// the rule; it is computed centrally from whether the registered fixer can
// produce a candidate, so the advertised flag and the implemented fixer can
// never drift.
package validation

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"brand-manager/internal/contrast"
)

// Severity is the normalized severity string a finding carries.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Finding is a single branding rule result.
type Finding struct {
	RuleID                 string
	Severity               Severity
	Title                  string
	Description            string
	FilePath               string // scenario-relative, best effort
	WhyItMatters           string
	RecommendedRemediation string
	Evidence               map[string]any
	AutofixAvailable       bool
}

// ScanResult is the full outcome of a branding scan for one scenario.
type ScanResult struct {
	Scenario string
	Findings []Finding
}

// ScanScenario evaluates every applicable branding rule against the scenario
// rooted at root and returns the findings (empty when fully compliant). A rule
// is skipped silently when a surface it requires (ui/, cli/) is absent, so a
// CLI/API-only scenario never collects false-positive UI findings.
func ScanScenario(scenario, root string) *ScanResult {
	c := newScanContext(scenario, root)
	res := &ScanResult{Scenario: scenario}
	for _, spec := range specs {
		if !c.hasAll(spec.surfaces) {
			continue
		}
		f, fired := spec.eval(c)
		if !fired {
			continue
		}
		f.RuleID = spec.id
		f.AutofixAvailable = autofixAvailable(spec.id, root)
		res.Findings = append(res.Findings, f)
	}
	return res
}

// --- helpers ---------------------------------------------------------------

func readFile(root, rel string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	return string(b), true
}

// designSystemCSSRel is where the react-vite design kit installs CSS custom
// properties; it is the canonical color/typography source for a generated
// scenario.
const designSystemCSSRel = "ui/src/design-tokens.css"

// scheme selects which color scheme's custom-property values to read.
type scheme int

const (
	schemeLight scheme = iota
	schemeDark
)

var cssVarRe = regexp.MustCompile(`(?m)^\s*(--[a-zA-Z0-9-]+)\s*:\s*([^;]+);`)

// cssVarsForScheme returns the effective value of each CSS custom property for
// the requested scheme. For light it is the first-declared value of each
// property (the :root block, which precedes any dark override). For dark it is
// the values declared inside dark-scheme blocks (.dark, [data-theme="dark"],
// @media (prefers-color-scheme: dark)); an empty map means no dark block ships.
func cssVarsForScheme(content string, s scheme) map[string]string {
	if s == schemeDark {
		content = darkSchemeBlocks(content)
	}
	out := map[string]string{}
	for _, m := range cssVarRe.FindAllStringSubmatch(content, -1) {
		name := strings.TrimSpace(m[1])
		if _, seen := out[name]; seen {
			continue
		}
		out[name] = strings.TrimSpace(m[2])
	}
	return out
}

var darkSelectorRe = regexp.MustCompile(`(?i)(\.dark\b|\[data-theme\s*[~|]?=\s*['"]?dark['"]?\]|@media[^{]*prefers-color-scheme\s*:\s*dark)`)

// darkSchemeBlocks returns the concatenated bodies of every dark-scheme rule in
// content (so cssVarsForScheme can read dark overrides). It performs balanced
// brace matching from each dark selector's opening brace.
func darkSchemeBlocks(content string) string {
	var b strings.Builder
	for _, loc := range darkSelectorRe.FindAllStringIndex(content, -1) {
		body, ok := braceBody(content, loc[1])
		if ok {
			b.WriteString(body)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// braceBody returns the text inside the first balanced {...} at or after start.
func braceBody(content string, start int) (string, bool) {
	open := strings.IndexByte(content[start:], '{')
	if open < 0 {
		return "", false
	}
	open += start
	depth := 0
	for i := open; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[open+1 : i], true
			}
		}
	}
	return "", false
}

// hasDarkScheme reports whether the design tokens ship any dark-scheme block.
func hasDarkScheme(content string) bool {
	return len(cssVarsForScheme(content, schemeDark)) > 0
}

func anyFileMatches(root string, dirs []string, patterns []string) (string, bool) {
	for _, dir := range dirs {
		for _, pat := range patterns {
			matches, _ := filepath.Glob(filepath.Join(root, filepath.FromSlash(dir), pat))
			if len(matches) > 0 {
				if rel, err := filepath.Rel(root, matches[0]); err == nil {
					return filepath.ToSlash(rel), true
				}
				return matches[0], true
			}
		}
	}
	return "", false
}

// --- rules: identity + visual system + assets + applied --------------------

func ruleHasDisplayName(c *scanContext) (Finding, bool) {
	if _, ok := c.read(".vrooli/service.json"); !ok {
		return Finding{
			Severity:               SeverityError,
			Title:                  "No service.json to declare a brand display name",
			Description:            "The scenario has no .vrooli/service.json, so it cannot declare a brand display name.",
			FilePath:               ".vrooli/service.json",
			WhyItMatters:           "The display name is the minimum brand identity every API/CLI/UI surface renders.",
			RecommendedRemediation: "Add .vrooli/service.json with a service.displayName.",
		}, true
	}
	id := c.identity()
	if !id.HasIdentity() {
		return Finding{
			Severity:               SeverityError,
			Title:                  "Brand display name is missing or a placeholder",
			Description:            "service.displayName is empty, a template placeholder, or just the raw scenario id.",
			FilePath:               ".vrooli/service.json",
			WhyItMatters:           "The display name is the minimum brand identity every API/CLI/UI surface renders.",
			RecommendedRemediation: "Set service.displayName to a meaningful product name.",
			Evidence:               map[string]any{"display_name": id.DisplayName, "scenario_id": id.Slug},
		}, true
	}
	return Finding{}, false
}

// coreColorTokens are the design-token names a coherent color system defines.
var coreColorTokens = []string{"--color-background", "--color-foreground", "--color-primary"}

func ruleHasColorSystem(c *scanContext) (Finding, bool) {
	vars, ok := c.tokens()
	var missing []string
	for _, t := range coreColorTokens {
		if strings.TrimSpace(vars[t]) == "" {
			missing = append(missing, t)
		}
	}
	if !ok || len(missing) > 0 {
		return Finding{
			Severity:               SeverityWarning,
			Title:                  "Canonical color-token contract is incomplete",
			Description:            "The canonical design-token file does not define the core color tokens (background, foreground, primary).",
			FilePath:               designSystemCSSRel,
			WhyItMatters:           "A canonical color-token contract keeps every surface visually coherent and themeable.",
			RecommendedRemediation: "Define the core --color-* custom properties in ui/src/design-tokens.css.",
			Evidence:               map[string]any{"missing_tokens": missing, "tokens_file_present": ok},
		}, true
	}
	return Finding{}, false
}

func ruleHasTypography(c *scanContext) (Finding, bool) {
	vars, _ := c.tokens()
	for name, v := range vars {
		if strings.HasPrefix(name, "--font-") && strings.TrimSpace(v) != "" {
			return Finding{}, false
		}
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "Typography tokens are not defined",
		Description:            "No --font-* design tokens define the scenario's heading/body typography.",
		FilePath:               designSystemCSSRel,
		WhyItMatters:           "Shared typography tokens keep text consistent across the scenario's surfaces.",
		RecommendedRemediation: "Define --font-sans (and optionally --font-mono) in the design tokens.",
	}, true
}

func ruleHasLogo(c *scanContext) (Finding, bool) {
	if _, ok := anyFileMatches(c.root,
		[]string{"ui/public", "ui/public/public", "ui/src/assets", "public", "assets", "ui/public/brand"},
		[]string{"logo.*", "logo-*.*", "*-logo.*"},
	); ok {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "No brand logo asset found",
		Description:            "No logo.* asset was found under the scenario's public/asset directories.",
		FilePath:               "ui/public",
		WhyItMatters:           "A logo is the primary visual brand mark users associate with the scenario.",
		RecommendedRemediation: "Generate or add a logo asset (e.g. ui/public/logo.svg) via brand-manager.",
	}, true
}

func ruleHasFavicon(c *scanContext) (Finding, bool) {
	if hasFaviconAsset(c.root) || headReferencesIcon(c.head()) {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "No favicon found",
		Description:            "No favicon asset or <link rel=\"icon\"> reference was found.",
		FilePath:               "ui/public",
		WhyItMatters:           "The favicon is the brand mark shown in browser tabs and bookmarks.",
		RecommendedRemediation: "Add a favicon (e.g. ui/public/favicon.svg) and reference it from ui/index.html.",
	}, true
}

func hasFaviconAsset(root string) bool {
	_, ok := anyFileMatches(root, []string{"ui/public", "public", "ui"}, []string{"favicon.*", "favicon-*.*"})
	return ok
}

func headReferencesIcon(h *headDoc) bool {
	if _, ok := h.linkByRel("icon"); ok {
		return true
	}
	if _, ok := h.linkByRel("shortcut icon"); ok {
		return true
	}
	if _, ok := h.linkByRel("apple-touch-icon"); ok {
		return true
	}
	return false
}

func ruleWCAGContrast(c *scanContext) (Finding, bool) {
	vars, ok := c.tokens()
	if !ok {
		return Finding{}, false // covered by has-color-system
	}
	failures, worst := contrastFailures(vars)
	if len(failures) == 0 {
		return Finding{}, false
	}
	return Finding{
		Severity:               SeverityWarning,
		Title:                  "Color pairings fail WCAG AA contrast",
		Description:            "One or more core color pairings do not meet the WCAG 2.1 AA normal-text contrast ratio of 4.5:1.",
		FilePath:               designSystemCSSRel,
		WhyItMatters:           "Low contrast makes text unreadable for low-vision users and fails accessibility audits.",
		RecommendedRemediation: "Adjust the failing color tokens so each pairing reaches at least 4.5:1.",
		Evidence:               map[string]any{"failures": failures, "worst_ratio": worst},
	}, true
}

// contrastPairs lists the foreground→background token pairings a readable UI
// must satisfy. Pairs whose tokens are absent are skipped (uncheckable), so a
// minimal token set checks only what it declares.
//
// Tier choice is deliberate: the core reading surfaces (body/primary text and
// the primary button label) are held to AA normal-text (4.5:1). Accent and
// semantic state colors are predominantly used for emphasis, icons, badges,
// borders, and large text, where the applicable WCAG bar is 3:1 (large-text /
// non-text contrast 1.4.11). Checking those at 4.5 would over-report on standard
// vivid palettes; 3:1 still catches a genuinely unusable color.
var contrastPairs = []struct {
	name, fg, bg string
	largeOK      bool // true ⇒ emphasis/non-text color, AA bar is 3:1 not 4.5:1
}{
	{name: "foreground-on-background", fg: "--color-foreground", bg: "--color-background"},
	{name: "foreground-on-surface", fg: "--color-foreground", bg: "--color-surface"},
	{name: "primary-on-background", fg: "--color-primary", bg: "--color-background"},
	{name: "primary-foreground-on-primary", fg: "--color-primary-foreground", bg: "--color-primary"},
	{name: "accent-on-background", fg: "--color-accent", bg: "--color-background", largeOK: true},
	{name: "error-on-background", fg: "--color-error", bg: "--color-background", largeOK: true},
	{name: "success-on-background", fg: "--color-success", bg: "--color-background", largeOK: true},
	{name: "warning-on-background", fg: "--color-warning", bg: "--color-background", largeOK: true},
	{name: "info-on-background", fg: "--color-info", bg: "--color-background", largeOK: true},
}

// contrastFailures runs every applicable pairing in vars and returns the failing
// ones plus the worst ratio seen. fg/bg must both resolve to a parseable color.
func contrastFailures(vars map[string]string) ([]map[string]any, float64) {
	var failures []map[string]any
	worst := 999.0
	for _, p := range contrastPairs {
		fg, bg := vars[p.fg], vars[p.bg]
		if fg == "" || bg == "" {
			continue
		}
		pr, err := contrast.CheckPair(fg, bg)
		if err != nil {
			continue
		}
		required := contrast.DefaultAANormalText
		passed := pr.AANormal
		if p.largeOK {
			required, passed = contrast.DefaultAALargeText, pr.AALarge
		}
		if !passed {
			failures = append(failures, map[string]any{
				"pair": p.name, "foreground": fg, "background": bg,
				"ratio": pr.Ratio, "required": required,
			})
			if pr.Ratio < worst {
				worst = pr.Ratio
			}
		}
	}
	return failures, worst
}

func ruleBrandMarkersApplied(c *scanContext) (Finding, bool) {
	if hasMarker(filepath.Join(c.root, "ui", "src")) {
		return Finding{}, false
	}
	for _, rel := range []string{"ui/manifest.json", "ui/public/manifest.json", "manifest.json"} {
		if manifest, ok := c.read(rel); ok && strings.Contains(manifest, "_brand") {
			return Finding{}, false
		}
	}
	return Finding{
		Severity:               SeverityInfo,
		Title:                  "No brand-manager markers applied",
		Description:            "No /* brand-manager:* */ CSS markers or manifest _brand keys were found — the scenario has not adopted a brand-manager-applied brand.",
		FilePath:               "ui/src",
		WhyItMatters:           "Applied markers let brand-manager re-apply, diff, and validate the assigned brand idempotently.",
		RecommendedRemediation: "Assign and apply a brand via `brand-manager apply` to inject managed markers.",
	}, true
}

var markerRe = regexp.MustCompile(`/\*\s*brand-manager:`)

func hasMarker(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found || d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".css", ".scss", ".ts", ".tsx", ".js":
			if b, err := os.ReadFile(path); err == nil && markerRe.Match(b) {
				found = true
			}
		}
		return nil
	})
	return found
}
