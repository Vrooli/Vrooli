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

<test-case id="no-raw-hex-native-color-input" should-fail="false">
  <description>A native color-input value is an HTML color value, not component styling</description>
  <input>
    [ui/src/components/BrandColor.tsx]
    export function BrandColor() { return <input type="color" value="#0f172a" />; }
  </input>
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

// nativeColorInputPattern identifies a complete JSX native color input. HTML
// requires its value to be a simple color (currently serialized as hex), so
// that value is data for the browser control rather than an inline style that
// bypasses the token layer. Dynamic input types remain checked.
var nativeColorInputPattern = regexp.MustCompile(`(?is)<input\b[^>]*\btype\s*=\s*(?:"color"|'color'|\{\s*["']color["']\s*\})[^>]*>`)

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

	files := sourceFiles(ctx, "ui/src")
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
		ext := strings.ToLower(filepath.Ext(f.RelPath))
		if ext != ".tsx" && ext != ".jsx" && ext != ".css" && ext != ".scss" && ext != ".less" {
			continue
		}
		if _, exempt := tokenDefinitionFiles[strings.ToLower(filepath.Base(f.RelPath))]; exempt {
			continue
		}
		matches := rawHexMatchesOutsideNativeColorInputs(f.Content)
		if len(matches) == 0 {
			continue
		}
		violations = append(violations, uiinterop.Violation{
			RuleID:         ruleID,
			Severity:       "low",
			Title:          "Raw hex color in component source",
			Description:    f.RelPath + " hardcodes raw hex color(s): " + strings.Join(matches, ", "),
			FilePath:       f.RelPath,
			Line:           lineOf(f.Content, matches[0]),
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

func rawHexMatchesOutsideNativeColorInputs(content string) []string {
	colorInputs := nativeColorInputPattern.FindAllStringIndex(content, -1)
	if len(colorInputs) == 0 {
		return uniqueStrings(rawHexPattern.FindAllString(content, -1))
	}

	var matches []string
	for _, hex := range rawHexPattern.FindAllStringIndex(content, -1) {
		insideNativeColorInput := false
		for _, input := range colorInputs {
			if hex[0] >= input[0] && hex[1] <= input[1] {
				insideNativeColorInput = true
				break
			}
		}
		if !insideNativeColorInput {
			matches = append(matches, content[hex[0]:hex[1]])
		}
	}
	return uniqueStrings(matches)
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
