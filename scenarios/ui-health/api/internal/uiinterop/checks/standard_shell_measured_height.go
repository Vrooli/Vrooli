/*
Rule: Measured App-Shell Height
ID: standard_shell_measured_height
Description: A full-viewport mobile app-shell root (a scroll-locked, column flex
  container that holds the app's chrome, e.g. a bottom nav) must size its height
  from a JS-measured viewport variable (for example `height: var(--app-height)`),
  not directly from a raw viewport unit (100vh / 100dvh / 100svh / 100lvh) or a
  percentage chain (height: 100%).
Why: iOS Safari resolves 100vh/100dvh/100svh/100lvh and the height:100% chain
  (which bottoms out at the initial containing block) against the LARGE,
  URL-bar-hidden viewport on first paint, and does not re-evaluate until the
  user scrolls. An in-flow bottom nav then floats below the visible area with a
  gap under it until a swipe forces a reflow — and no headless/Chromium render
  reproduces it, so it ships invisibly. The robust fix, used by web-console and
  git-control-tower, is a CSS variable set from window.visualViewport
  (useAppViewport → --app-height), with 100dvh kept only as a pre-hydration
  fallback inside var(). A measured var is also correct inside the host iframe
  (it measures the frame's client box), so it supersedes the older
  interop_h_screen "use 100%" advice for the shell root specifically.
Category: standards
Severity: high
Slot: [D]
SlotFile: ui/src
TechStack: React
Recommendation: Size the app-shell root from a JS-measured viewport variable,
  e.g. `height: var(--app-height, 100dvh)` where --app-height is set from
  window.visualViewport (copy scenarios/web-console/ui/src/hooks/useAppViewport.ts
  or git-control-tower's useAppViewport). Do not size the shell directly from
  100vh/100dvh/100%; those are iOS-unsafe on first paint.
Standard: ui-health-v1

GoodExample:
    .app-shell {
      height: var(--app-height, 100dvh);
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }

BadExample:
    .app-shell {
      height: 100%;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
    (height:100% resolves against the large iOS viewport on first paint)

<test-case id="shell-measured-var-clean" should-fail="false">
  <description>App-shell sizes from a measured viewport variable</description>
  <input>
    [ui/src/styles.css]
    .app-shell {
      height: var(--app-height, 100dvh);
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
  </input>
</test-case>

<test-case id="shell-percent-chain-unsafe" should-fail="true">
  <description>App-shell sized with height:100% is iOS-unsafe</description>
  <input>
    [ui/src/styles.css]
    html, body, #root { height: 100%; }
    .app-shell {
      height: 100%;
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>measured viewport variable</expected-message>
</test-case>

<test-case id="shell-bare-dvh-unsafe" should-fail="true">
  <description>App-shell sized with bare 100dvh is iOS-unsafe on first paint</description>
  <input>
    [ui/src/styles.css]
    .mobile-shell {
      display: flex;
      flex-direction: column;
      overflow: hidden;
      block-size: 100dvh;
    }
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>measured viewport variable</expected-message>
</test-case>

<test-case id="root-height-percent-not-a-shell-clean" should-fail="false">
  <description>html/body/#root height:100% is expected and not a shell violation</description>
  <input>
    [ui/src/styles.css]
    html, body, #root { height: 100%; min-height: 100%; }
    .panel { overflow: auto; height: 100%; }
  </input>
</test-case>

<test-case id="dialog-max-height-clean" should-fail="false">
  <description>A dialog capping max-height at 100vh is not the first-paint shell bug</description>
  <input>
    [ui/src/styles.css]
    .responsive-dialog__content {
      max-height: calc(100vh - 64px);
      display: flex;
      flex-direction: column;
      overflow: hidden;
    }
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
	uiinterop.Register("standard_shell_measured_height", checkShellMeasuredHeight)
}

var (
	// Innermost declaration block: selector(s) + body with no nested braces.
	// This naturally skips @media wrappers (their body has braces) while still
	// matching the individual rules nested inside them.
	cssRuleBlockPattern = regexp.MustCompile(`(?s)([^{}]+)\{([^{}]*)\}`)

	cssDisplayFlexPattern    = regexp.MustCompile(`(?i)display\s*:\s*(?:inline-)?flex`)
	cssColumnPattern         = regexp.MustCompile(`(?i)flex-direction\s*:\s*column|flex-flow\s*:[^;]*column`)
	cssOverflowHiddenPattern = regexp.MustCompile(`(?i)overflow(?:-y|-block)?\s*:\s*(?:hidden|clip)`)
	// Vertical size declarations whose value we must inspect. The leading
	// boundary keeps `max-height` / `line-height` from matching as `height`
	// (max-height is a ceiling used by dialogs, not the shell's primary height).
	cssVerticalSizePattern = regexp.MustCompile(`(?i)(?:^|[\s;{])(?:min-)?(?:height|block-size)\s*:\s*([^;}]+)`)
	// Raw viewport-unit values: these name the viewport directly, so on a
	// scroll-locked column shell they are iOS-unsafe regardless of selector name.
	cssViewportUnitPattern = regexp.MustCompile(`(?i)(100vh|100dvh|100svh|100lvh|100vmin|100vmax|-webkit-fill-available)`)
	// A percentage height only chains to the viewport on the actual root shell;
	// inner cards/panels legitimately use height:100% to fill a measured parent.
	// Gate it behind a shell-root name signal to avoid flagging inner components.
	cssPercentHeightPattern = regexp.MustCompile(`(?i)\b100\s*%`)
	cssShellSelectorPattern = regexp.MustCompile(`(?i)(shell|viewport)`)
	cssVarPattern           = regexp.MustCompile(`(?i)var\s*\(`)
	cssCommentPattern       = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

// stylesheetExts are the source extensions that carry CSS rule blocks.
var stylesheetExts = map[string]struct{}{
	".css":  {},
	".scss": {},
	".sass": {},
	".less": {},
}

func checkShellMeasuredHeight(ctx uiinterop.CheckContext) uiinterop.RuleResult {
	const ruleID = "standard_shell_measured_height"

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
		if _, ok := stylesheetExts[strings.ToLower(filepath.Ext(f.RelPath))]; !ok {
			continue
		}
		for _, block := range cssRuleBlockPattern.FindAllStringSubmatch(f.Content, -1) {
			selector := strings.TrimSpace(block[1])
			body := block[2]
			if !isColumnShellBlock(body) {
				continue
			}
			decl, value, ok := unsafeShellHeight(body, selector)
			if !ok {
				continue
			}
			violations = append(violations, uiinterop.Violation{
				RuleID:         ruleID,
				Severity:       "high",
				Title:          "App-shell height not measured (iOS-unsafe)",
				Description:    f.RelPath + ": shell rule `" + firstSelector(selector) + "` sizes its height from `" + strings.TrimSpace(value) + "`, which iOS Safari resolves against the large URL-bar-hidden viewport on first paint — leaving the bottom chrome floating until a scroll.",
				FilePath:       f.RelPath,
				Line:           lineOf(f.Content, decl),
				Recommendation: "Size the shell from a JS-measured viewport variable, e.g. `height: var(--app-height, 100dvh)` set from window.visualViewport (see web-console / git-control-tower useAppViewport). Do not size the shell directly from 100vh/100dvh/100%.",
			})
		}
	}

	if len(violations) > 0 {
		return uiinterop.RuleResult{
			RuleID:     ruleID,
			Passed:     false,
			Message:    "app-shell root sizes its height from a raw viewport unit instead of a measured viewport variable",
			Violations: violations,
		}
	}
	return uiinterop.RuleResult{
		RuleID:  ruleID,
		Passed:  true,
		Message: "no iOS-unsafe app-shell height sizing found",
	}
}

// isColumnShellBlock reports whether a CSS declaration body looks like a
// full-viewport app-shell root: a scroll-locked, column flex container. This is
// the signature that matters for the iOS first-paint gap (an in-flow bottom nav
// sits at the end of the column and floats if the shell oversizes). Inner panels
// that merely fill a measured parent do not match (they lack the flex-column +
// overflow-hidden shell signature together with a viewport-scale height).
func isColumnShellBlock(body string) bool {
	return cssDisplayFlexPattern.MatchString(body) &&
		cssColumnPattern.MatchString(body) &&
		cssOverflowHiddenPattern.MatchString(body)
}

// unsafeShellHeight returns the raw declaration, its value, and true when a
// vertical-size declaration in the block is an iOS-unsafe primary height with no
// var() (measured) source. A var() value — e.g. var(--app-height, 100dvh) — is
// safe because the fallback only applies before the hook has measured.
//
// Raw viewport units (100vh/100dvh/...) are flagged on any column shell. A bare
// percentage is flagged only when the selector names a root shell (shell /
// viewport), because inner cards and panels legitimately use height:100% to fill
// an already-measured parent.
func unsafeShellHeight(body, selector string) (decl string, value string, ok bool) {
	shellNamed := cssShellSelectorPattern.MatchString(selector)
	for _, m := range cssVerticalSizePattern.FindAllStringSubmatch(body, -1) {
		val := m[1]
		if cssVarPattern.MatchString(val) {
			continue
		}
		decl := strings.TrimLeft(m[0], " ;{\n\t\r")
		if cssViewportUnitPattern.MatchString(val) {
			return decl, val, true
		}
		if shellNamed && cssPercentHeightPattern.MatchString(val) {
			return decl, val, true
		}
	}
	return "", "", false
}

func firstSelector(selector string) string {
	selector = cssCommentPattern.ReplaceAllString(selector, " ")
	selector = strings.TrimSpace(selector)
	if i := strings.IndexAny(selector, ",\n"); i >= 0 {
		selector = strings.TrimSpace(selector[:i])
	}
	return selector
}
