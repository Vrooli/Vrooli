// Package validation implements brand-manager's branding validation: a
// self-contained scan of a target scenario's on-disk branding artifacts that
// produces normalized findings + a maturity assessment. It is served through
// the shared scenario-validation/v1.ScenarioValidationService so test-genie can
// run it as the `branding` delegated phase.
//
// The rules here re-express (and expand) the branding concepts the prior REST
// build encoded in its orphaned audit provider — but they validate a scenario's
// REAL files (display name, color tokens, typography, logo, favicon, applied
// brand markers, WCAG-AA contrast) rather than a brand record's fields.
package validation

import (
	"encoding/json"
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

// ScanScenario evaluates every branding rule against the scenario rooted at
// root and returns the findings (empty when the branding is fully compliant).
// A missing/unreadable root is the caller's responsibility (resolve first); an
// empty findings slice means "passed".
func ScanScenario(scenario, root string) *ScanResult {
	res := &ScanResult{Scenario: scenario}
	for _, rule := range allRules {
		if f, fired := rule(root); fired {
			res.Findings = append(res.Findings, f)
		}
	}
	return res
}

// rule evaluates one branding rule against a scenario root. It returns the
// finding and true when the rule is violated, or false when satisfied.
type rule func(root string) (Finding, bool)

var allRules = []rule{
	ruleHasDisplayName,
	ruleHasColorSystem,
	ruleHasTypography,
	ruleHasLogo,
	ruleHasFavicon,
	ruleWCAGContrast,
	ruleBrandMarkersApplied,
}

// --- helpers ---------------------------------------------------------------

func readFile(root, rel string) (string, bool) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	return string(b), true
}

// designTokensPath is where the react-vite design kit installs CSS custom
// properties; it is the canonical color/typography source for a generated
// scenario.
const designSystemCSSRel = "ui/src/design-tokens.css"

var cssVarRe = regexp.MustCompile(`(?m)^\s*(--[a-zA-Z0-9-]+)\s*:\s*([^;]+);`)

// firstCSSVars returns the first-declared value of each CSS custom property in
// the given content (the light :root block wins, matching render default).
func firstCSSVars(content string) map[string]string {
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

// --- rules -----------------------------------------------------------------

func ruleHasDisplayName(root string) (Finding, bool) {
	content, ok := readFile(root, ".vrooli/service.json")
	if !ok {
		return Finding{
			RuleID:                 "has-display-name",
			Severity:               SeverityError,
			Title:                  "No service.json to declare a brand display name",
			Description:            "The scenario has no .vrooli/service.json, so it cannot declare a brand display name.",
			FilePath:               ".vrooli/service.json",
			WhyItMatters:           "The display name is the minimum brand identity every API/CLI/UI surface renders.",
			RecommendedRemediation: "Add .vrooli/service.json with a service.displayName.",
		}, true
	}
	var svc struct {
		Service struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"service"`
	}
	_ = json.Unmarshal([]byte(content), &svc)
	display := strings.TrimSpace(svc.Service.DisplayName)
	// A title-cased slug (e.g. "Widget Shop" for "widget-shop") is a perfectly
	// good display name, so only the raw slug itself, an empty value, or a
	// template bracket placeholder counts as missing.
	placeholder := display == "" ||
		strings.Contains(display, "[") ||
		strings.EqualFold(display, svc.Service.Name)
	if placeholder {
		return Finding{
			RuleID:                 "has-display-name",
			Severity:               SeverityError,
			Title:                  "Brand display name is missing or a placeholder",
			Description:            "service.displayName is empty, a template placeholder, or just the raw scenario id.",
			FilePath:               ".vrooli/service.json",
			WhyItMatters:           "The display name is the minimum brand identity every API/CLI/UI surface renders.",
			RecommendedRemediation: "Set service.displayName to a meaningful product name.",
			Evidence:               map[string]any{"display_name": display, "scenario_id": svc.Service.Name},
		}, true
	}
	return Finding{}, false
}

// coreColorTokens are the design-token names a coherent color system defines.
var coreColorTokens = []string{"--color-background", "--color-foreground", "--color-primary"}

func ruleHasColorSystem(root string) (Finding, bool) {
	content, ok := readFile(root, designSystemCSSRel)
	vars := map[string]string{}
	if ok {
		vars = firstCSSVars(content)
	}
	var missing []string
	for _, t := range coreColorTokens {
		if strings.TrimSpace(vars[t]) == "" {
			missing = append(missing, t)
		}
	}
	if !ok || len(missing) > 0 {
		return Finding{
			RuleID:                 "has-color-system",
			Severity:               SeverityWarning,
			Title:                  "Core color system is incomplete",
			Description:            "The design tokens do not define the core color tokens (background, foreground, primary).",
			FilePath:               designSystemCSSRel,
			WhyItMatters:           "A defined color system keeps every surface visually coherent and themeable.",
			RecommendedRemediation: "Define the core --color-* custom properties in the design tokens.",
			Evidence:               map[string]any{"missing_tokens": missing, "tokens_file_present": ok},
			AutofixAvailable:       true,
		}, true
	}
	return Finding{}, false
}

func ruleHasTypography(root string) (Finding, bool) {
	content, ok := readFile(root, designSystemCSSRel)
	vars := map[string]string{}
	if ok {
		vars = firstCSSVars(content)
	}
	hasFont := false
	for name, v := range vars {
		if strings.HasPrefix(name, "--font-") && strings.TrimSpace(v) != "" {
			hasFont = true
			break
		}
	}
	if !hasFont {
		return Finding{
			RuleID:                 "has-typography",
			Severity:               SeverityInfo,
			Title:                  "Typography tokens are not defined",
			Description:            "No --font-* design tokens define the scenario's heading/body typography.",
			FilePath:               designSystemCSSRel,
			WhyItMatters:           "Shared typography tokens keep text consistent across the scenario's surfaces.",
			RecommendedRemediation: "Define --font-sans (and optionally --font-mono) in the design tokens.",
		}, true
	}
	return Finding{}, false
}

func ruleHasLogo(root string) (Finding, bool) {
	if _, ok := anyFileMatches(root,
		[]string{"ui/public", "ui/src/assets", "public", "assets", "ui/public/brand"},
		[]string{"logo.*", "logo-*.*", "*-logo.*"},
	); ok {
		return Finding{}, false
	}
	return Finding{
		RuleID:                 "has-logo",
		Severity:               SeverityWarning,
		Title:                  "No brand logo asset found",
		Description:            "No logo.* asset was found under the scenario's public/asset directories.",
		FilePath:               "ui/public",
		WhyItMatters:           "A logo is the primary visual brand mark users associate with the scenario.",
		RecommendedRemediation: "Generate or add a logo asset (e.g. ui/public/logo.svg) via brand-manager.",
	}, true
}

func ruleHasFavicon(root string) (Finding, bool) {
	if _, ok := anyFileMatches(root,
		[]string{"ui/public", "public", "ui"},
		[]string{"favicon.*", "favicon-*.*"},
	); ok {
		return Finding{}, false
	}
	// Fall back to an index.html <link rel="icon"> reference.
	if html, ok := readFile(root, "ui/index.html"); ok {
		if strings.Contains(html, `rel="icon"`) || strings.Contains(html, "rel='icon'") || strings.Contains(strings.ToLower(html), "apple-touch-icon") {
			return Finding{}, false
		}
	}
	return Finding{
		RuleID:                 "has-favicon",
		Severity:               SeverityWarning,
		Title:                  "No favicon found",
		Description:            "No favicon asset or <link rel=\"icon\"> reference was found.",
		FilePath:               "ui/public",
		WhyItMatters:           "The favicon is the brand mark shown in browser tabs and bookmarks.",
		RecommendedRemediation: "Add a favicon (e.g. ui/public/favicon.png) and reference it from ui/index.html.",
		AutofixAvailable:       true,
	}, true
}

func ruleWCAGContrast(root string) (Finding, bool) {
	content, ok := readFile(root, designSystemCSSRel)
	if !ok {
		// No tokens to check — covered by has-color-system, not a contrast finding.
		return Finding{}, false
	}
	vars := firstCSSVars(content)
	fg := vars["--color-foreground"]
	bg := vars["--color-background"]
	surface := vars["--color-surface"]
	primary := vars["--color-primary"]
	if fg == "" || bg == "" {
		return Finding{}, false // uncheckable without the core pair
	}
	type pair struct {
		name   string
		fg, bg string
	}
	pairs := []pair{{"foreground-on-background", fg, bg}}
	if surface != "" {
		pairs = append(pairs, pair{"foreground-on-surface", fg, surface})
	}
	if primary != "" {
		pairs = append(pairs, pair{"primary-on-background", primary, bg})
	}
	var failures []map[string]any
	worst := 999.0
	for _, p := range pairs {
		pr, err := contrast.CheckPair(p.fg, p.bg)
		if err != nil {
			continue // unparseable color value; skip this pair
		}
		if !pr.AANormal {
			failures = append(failures, map[string]any{
				"pair": p.name, "foreground": p.fg, "background": p.bg, "ratio": pr.Ratio, "required": contrast.DefaultAANormalText,
			})
			if pr.Ratio < worst {
				worst = pr.Ratio
			}
		}
	}
	if len(failures) == 0 {
		return Finding{}, false
	}
	return Finding{
		RuleID:                 "wcag-aa-contrast",
		Severity:               SeverityWarning,
		Title:                  "Color pairings fail WCAG AA contrast",
		Description:            "One or more core color pairings do not meet the WCAG 2.1 AA normal-text contrast ratio of 4.5:1.",
		FilePath:               designSystemCSSRel,
		WhyItMatters:           "Low contrast makes text unreadable for low-vision users and fails accessibility audits.",
		RecommendedRemediation: "Adjust the failing color tokens so each pairing reaches at least 4.5:1.",
		Evidence:               map[string]any{"failures": failures, "worst_ratio": worst},
	}, true
}

func ruleBrandMarkersApplied(root string) (Finding, bool) {
	// Look for applied brand-manager CSS markers in any UI stylesheet.
	if hasMarker(filepath.Join(root, "ui", "src")) {
		return Finding{}, false
	}
	// Or a manifest.json _brand key.
	if manifest, ok := readFile(root, "ui/manifest.json"); ok && strings.Contains(manifest, "_brand") {
		return Finding{}, false
	}
	if manifest, ok := readFile(root, "manifest.json"); ok && strings.Contains(manifest, "_brand") {
		return Finding{}, false
	}
	return Finding{
		RuleID:                 "brand-markers-applied",
		Severity:               SeverityInfo,
		Title:                  "No brand-manager markers applied",
		Description:            "No /* brand-manager:* */ CSS markers or manifest _brand keys were found — the scenario has not adopted a brand-manager-applied brand.",
		FilePath:               "ui/src",
		WhyItMatters:           "Applied markers let brand-manager re-apply, diff, and validate the assigned brand idempotently.",
		RecommendedRemediation: "Assign and apply a brand via `brand-manager apply` to inject managed markers.",
		AutofixAvailable:       true,
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
