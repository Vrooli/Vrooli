/*
Rule: Design Token Bypass
ID: standard_design_token_bypass
Description: Tokenized React UIs should not style component surfaces primarily
  with raw Tailwind palette classes, arbitrary color values, or inline literal
  colors. Files are flagged only when bypass styling is systematic: at least
  three bypass signals and at least 60% of the file's color/radius token signals
  are non-token. Single stray literals stay advisory through standard_no_raw_hex.
Why: Governed components can still undermine the design system if their chrome
  is painted with slate/gray/sky classes, bg-[#...] values, or inline colors
  instead of app-* token classes, rounded-control/panel tokens, and CSS token
  variables. Majority-token enforcement separates a token-native component
  from an unthemed surface without spamming legacy or one-off code.
Category: standards
Severity: low
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Replace raw palette utilities and inline literal colors with
  app-* Tailwind token classes, rounded-control/panel/sheet/pill radius tokens,
  or CSS variables from the design-token layer.
Standard: vrooli-ui-theming-v1

GoodExample:
    export function Card() {
      return <section className="rounded-panel border border-app-border bg-app-surface text-app-foreground" />;
    }

BadExample:
    export function Card() {
      return <section className="rounded-lg border border-slate-700 bg-slate-950 text-slate-100" />;
    }

<test-case id="design-token-bypass-token-native" should-fail="false">
  <description>Token-native component styling is clean</description>
  <input>
    [.vrooli/service.json]
    {"generation":{"design":{"id":"vrooli-default","adapter":"react-vite-tailwind"}}}
    [ui/src/components/ui/Card.tsx]
    export function Card() {
      return <section className="rounded-panel border border-app-border bg-app-surface text-app-foreground shadow-sm" style={{ color: "var(--text-primary)" }}>x</section>;
    }
  </input>
</test-case>

<test-case id="design-token-bypass-unscoped-legacy" should-fail="false">
  <description>Legacy UIs without declared design style are not spammed</description>
  <input>
    [ui/src/components/Card.tsx]
    export function Card() {
      return <section className="rounded-lg border border-slate-700 bg-slate-950 text-slate-100">x</section>;
    }
  </input>
</test-case>

<test-case id="design-token-bypass-systematic" should-fail="true">
  <description>Systematic raw palette styling bypasses the token system</description>
  <input>
    [.vrooli/service.json]
    {"generation":{"design":{"id":"vrooli-default","adapter":"react-vite-tailwind"}}}
    [ui/src/features/components/EditorChrome.tsx]
    export function EditorChrome() {
      return <section className="rounded-lg border border-slate-700 bg-[#05070d] text-slate-100"><p className="text-slate-400">x</p></section>;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>systematically bypasses design tokens</expected-message>
</test-case>
*/

package checks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_design_token_bypass", checkDesignTokenBypass)
}

var (
	rawTailwindPaletteClassPattern = regexp.MustCompile(`(?:^|[^A-Za-z0-9_/-])(?:[a-z0-9-]+:)*(?:bg|text|border|ring|outline|divide|placeholder|accent|decoration|from|via|to)-(?:slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-(?:50|100|200|300|400|500|600|700|800|900|950)(?:/[0-9]+)?\b|(?:^|[^A-Za-z0-9_/-])(?:[a-z0-9-]+:)*(?:bg|text|border|ring|outline|divide|placeholder|accent|decoration|from|via|to)-(?:white|black)(?:/[0-9]+)?\b`)
	arbitraryColorUtilityPattern   = regexp.MustCompile(`(?:^|[^A-Za-z0-9_/-])(?:[a-z0-9-]+:)*(?:bg|text|border|ring|outline|shadow|divide|placeholder|accent|decoration|from|via|to)-\[(?:#[0-9a-fA-F]{3,8}|rgb[a]?\(|hsl[a]?\()`)
	inlineLiteralColorPattern      = regexp.MustCompile(`(?i)(?:color|backgroundColor|background|borderColor|outlineColor|fill|stroke)\s*:\s*["'](?:#[0-9a-f]{3,8}|rgb[a]?\([^"']+\)|hsl[a]?\([^"']+\)|(?:slate|gray|grey|sky|blue|black|white|red|green|yellow|orange|purple|pink|cyan|teal|indigo|violet|amber|emerald|rose|zinc|neutral|stone))["']`)
	tokenStylingPattern            = regexp.MustCompile(`(?:^|[^A-Za-z0-9_/-])(?:[a-z0-9-]+:)*(?:bg|text|border|ring|outline|divide|placeholder|accent|decoration|from|via|to)-app-[A-Za-z0-9/-]+|(?:^|[^A-Za-z0-9_/-])(?:[a-z0-9-]+:)*rounded-(?:control|panel|sheet|pill)\b|var\(--(?:color|surface|text|radius|font|touch|sidebar|scrollbar)-[A-Za-z0-9-]+\)`)
)

func checkDesignTokenBypass(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_design_token_bypass"
	files := sourceFiles(ctx, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{RuleID: ruleID, Skipped: true, SkipReason: "no ui/src directory found", Message: "no ui/src directory found; skipping design-token bypass check"}
	}
	tokenViolations, hasTokenNativeVendor := missingVendoredTokenViolations(ctx, files, ruleID)

	if !serviceDeclaresDesignStyle(ctx.ScenarioRoot) && !hasTokenNativeVendor {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no declared design style",
			Message:    "no declared design style; skipping design-token bypass check",
		}
	}

	violations := tokenViolations
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.RelPath))
		if ext != ".tsx" && ext != ".jsx" && ext != ".css" && ext != ".scss" && ext != ".less" {
			continue
		}
		if _, exempt := tokenDefinitionFiles[strings.ToLower(filepath.Base(f.RelPath))]; exempt {
			continue
		}

		rawMatches := append(rawTailwindPaletteClassPattern.FindAllString(f.Content, -1), arbitraryColorUtilityPattern.FindAllString(f.Content, -1)...)
		rawMatches = append(rawMatches, inlineLiteralColorPattern.FindAllString(f.Content, -1)...)
		tokenMatches := tokenStylingPattern.FindAllString(f.Content, -1)
		rawCount := len(rawMatches)
		tokenCount := len(tokenMatches)
		if rawCount < 3 {
			continue
		}
		total := rawCount + tokenCount
		if total == 0 || rawCount*100/total < 60 {
			continue
		}

		examples := uniqueStrings(trimRegexMatches(rawMatches))
		if len(examples) > 5 {
			examples = examples[:5]
		}
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "low",
			Title:          "Component styling bypasses design tokens",
			Description:    fmt.Sprintf("%s systematically bypasses design tokens: %d raw styling signal(s), %d token styling signal(s); examples: %s", f.RelPath, rawCount, tokenCount, strings.Join(examples, ", ")),
			FilePath:       f.RelPath,
			Line:           lineOf(f.Content, strings.TrimSpace(rawMatches[0])),
			Recommendation: "Replace raw palette utilities and inline literal colors with app-* token classes, rounded-control/panel/sheet/pill tokens, or CSS variables from the design-token layer.",
		})
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "component styling systematically bypasses design tokens",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "component styling primarily uses design tokens",
	}
}

func missingVendoredTokenViolations(ctx uiinterop.CheckContext, files []uiinterop.SourceFile, ruleID string) ([]uiinterop.Violation, bool) {
	repoRoot := findRepoRoot(ctx.ScenarioRoot)
	if repoRoot == "" {
		return nil, false
	}
	componentDir := filepath.Join(repoRoot, "scenarios", "react-component-library", "library", "components")
	entries, err := os.ReadDir(componentDir)
	if err != nil {
		return nil, false
	}
	required := map[string][]string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(componentDir, entry.Name(), "component.json"))
		if err != nil {
			continue
		}
		var meta struct {
			LibraryID      string   `json:"libraryId"`
			RequiredTokens []string `json:"requiredTokens"`
		}
		if json.Unmarshal(data, &meta) == nil && len(meta.RequiredTokens) > 0 {
			required[meta.LibraryID] = meta.RequiredTokens
		}
	}
	allCSS := ""
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.RelPath), ".css") {
			allCSS += "\n" + f.Content
		}
	}
	var violations []uiinterop.Violation
	hasTokenNativeVendor := false
	for _, f := range files {
		libraryID := provenanceField(f.Content, "@vrooliComponentSource")
		tokens := required[libraryID]
		if len(tokens) == 0 {
			continue
		}
		hasTokenNativeVendor = true
		missing := []string{}
		for _, token := range tokens {
			if !strings.Contains(allCSS, token+":") {
				missing = append(missing, token)
			}
		}
		if len(missing) > 0 {
			violations = append(violations, uiinterop.Violation{RuleID: ruleID, Severity: "high", Title: "Vendored component requires missing design tokens", Description: fmt.Sprintf("%s vendors %s but host CSS does not define: %s", f.RelPath, libraryID, strings.Join(missing, ", ")), FilePath: f.RelPath, Line: lineOf(f.Content, "@vrooliComponentSource"), Recommendation: "Adopt the governed design-token layer or define the required tokens in host CSS."})
		}
	}
	return violations, hasTokenNativeVendor
}

func trimRegexMatches(matches []string) []string {
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.Trim(match, " \t\r\n\"'`<{(["))
	}
	return out
}
