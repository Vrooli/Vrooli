/*
Rule: No Raw Hex Colors
ID: standard_no_raw_hex
Description: Component source and stylesheets under ui/src must not hardcode raw
  hex color literals (#rgb / #rrggbb / #rrggbbaa). Colors belong to the design
  token layer (theme/tokens.css, the generated design-tokens.css), consumed via
  CSS custom properties (var(--surface-app)) or Tailwind token classes. The
  token-definition files themselves are exempt — that is where the primitives
  live.
Why: Raw hex scattered through components bypasses theming: a dark-mode toggle
  or a design-kit swap re-binds the tokens, but inline hex stays frozen, so the
  component renders the wrong color and breaks contrast. Routing every color
  through a token keeps light/dark and kit changes a config edit, not a
  find-and-replace across the codebase.
Category: visual
Severity: low
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Replace the hex literal with the appropriate semantic token
  (var(--text-primary), var(--surface-app), …) or a Tailwind token class. Define
  any genuinely new color in the token layer, never inline.
Standard: vrooli-ui-theming-v1

GoodExample:
    .card { color: var(--text-primary); background: var(--surface-elevated); }

BadExample:
    .card { color: #0f172a; background: #ffffff; }

<test-case id="no-raw-hex-tokens" should-fail="false">
  <description>Component styles reference tokens, not raw hex</description>
  <input>
    [ui/src/components/Card.tsx]
    export function Card() { return <div style={{ color: "var(--text-primary)" }}>x</div>; }
  </input>
</test-case>

<test-case id="no-raw-hex-token-file-exempt" should-fail="false">
  <description>Raw hex inside the token-definition file is allowed (that is where primitives live)</description>
  <input>
    [ui/src/theme/tokens.css]
    :root { --surface-app: #ffffff; --text-primary: #0f172a; }
  </input>
</test-case>

<test-case id="no-raw-hex-component" should-fail="true">
  <description>Component hardcodes a hex color</description>
  <input>
    [ui/src/components/Card.tsx]
    export function Card() { return <div style={{ color: "#0f172a" }}>x</div>; }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>#0f172a</expected-message>
</test-case>
*/

package checks

import (
	"path/filepath"
	"regexp"
	"strings"

	"ui-health/internal/uiinterop"
)

func init() {
	uiinterop.Register("standard_no_raw_hex", checkNoRawHex)
}

// rawHexPattern matches a CSS-style hex color literal: a '#' followed by exactly
// 3, 4, 6, or 8 hex digits with a non-hex boundary after, so it does not match
// substrings of longer hashes/ids.
var rawHexPattern = regexp.MustCompile(`#([0-9a-fA-F]{8}|[0-9a-fA-F]{6}|[0-9a-fA-F]{4}|[0-9a-fA-F]{3})\b`)

// tokenDefinitionFiles are the design-token source files where raw hex
// primitives legitimately live and are therefore exempt from the scan.
var tokenDefinitionFiles = map[string]struct{}{
	"tokens.css":        {},
	"design-tokens.css": {},
	"theme-tokens.css":  {},
	"palette.css":       {},
}

func checkNoRawHex(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_no_raw_hex"

	files := walkUISource(ctx.ScenarioRoot, "ui/src")
	if len(files) == 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Skipped:    true,
			SkipReason: "no ui/src directory found",
			Message:    "no ui/src directory found; skipping",
		}
	}

	var violations []uiinterop.Violation
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f.relPath))
		if ext != ".tsx" && ext != ".jsx" && ext != ".css" && ext != ".scss" && ext != ".less" {
			continue
		}
		if _, exempt := tokenDefinitionFiles[strings.ToLower(filepath.Base(f.relPath))]; exempt {
			continue
		}
		matches := uniqueStrings(rawHexPattern.FindAllString(f.content, -1))
		if len(matches) == 0 {
			continue
		}
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "low",
			Title:          "Raw hex color in component source",
			Description:    f.relPath + " hardcodes raw hex color(s): " + strings.Join(matches, ", "),
			FilePath:       f.relPath,
			Line:           lineOf(f.content, matches[0]),
			Recommendation: "Replace the hex literal with a semantic token (var(--…)) or a Tailwind token class; define new colors in the token layer",
		})
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "raw hex colors found in component source",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no raw hex colors outside the token layer",
	}
}

// uniqueStrings de-duplicates a slice while preserving first-seen order.
func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
