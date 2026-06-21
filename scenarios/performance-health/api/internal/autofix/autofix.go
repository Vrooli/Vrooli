// Package autofix adapts the shared packages/maturity-go/autofix registry to
// performance-health's readiness fixers: format-preserving edits that add the
// four Tier-1 perf-build infra pieces. Dry-run is the default (Preview); Apply
// is never implicit. Apply is Preview-then-write and idempotent (each fixer
// re-checks state, so re-running over an already-fixed tree yields nothing).
//
// The fixers are structured/targeted text edits — NOT a blind parse-then-
// reserialize round-trip — so existing files keep their comments and layout:
//   - package.json: a minimal structured JSON edit adding the build:profile
//     script and making the build script mode-aware.
//   - vite.config.ts: inject the conditional profile-mode block into the
//     existing defineConfig, leaving the rest of the file untouched.
//   - src/lib/profiler.ts: scaffold the canonical util when absent.
//   - src/main.tsx: wrap the rendered app in a top-level <React.Profiler> and
//     import onProfilerRender, leaving the surrounding bootstrap intact.
package autofix

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/maturity-go/autofix"

	"performance-health/internal/readiness"
)

// Candidate re-exports the shared autofix candidate so handlers/CLI map straight
// through to the scenario-validation FixCandidate wire shape.
type Candidate = autofix.Candidate

// Service drives readiness autofix over the shared registry.
type Service struct {
	registry *autofix.Registry
}

// NewService builds the autofix Service with the four perf-build-infra fixers.
func NewService() *Service {
	return &Service{registry: autofix.NewRegistry(perfBuildFixers()...)}
}

// Preview returns the edits readiness would apply for the requested rules
// (empty => every fixable rule), without writing.
func (s *Service) Preview(root string, ruleIDs []string) ([]Candidate, error) {
	if s == nil || s.registry == nil {
		return nil, errors.New("autofix: service not wired")
	}
	if root == "" {
		return nil, errors.New("autofix: scenario root is required")
	}
	return s.registry.Preview(root, ruleIDs)
}

// Apply writes the edits for the requested rules and reports what changed.
func (s *Service) Apply(root string, ruleIDs []string) ([]Candidate, error) {
	if s == nil || s.registry == nil {
		return nil, errors.New("autofix: service not wired")
	}
	if root == "" {
		return nil, errors.New("autofix: scenario root is required")
	}
	return s.registry.Apply(root, ruleIDs)
}

// CanFix reports whether the named rule can currently remediate the finding.
func (s *Service) CanFix(root, ruleID, findingPath string) bool {
	if s == nil || s.registry == nil {
		return false
	}
	return s.registry.CanFix(root, ruleID, findingPath)
}

// perfBuildFixers returns the four readiness perf-build-infra fixers, one per
// shared readiness rule ID so a fix targeted by `--rule <code>` lines up with
// the finding the detector emitted.
func perfBuildFixers() []autofix.Fixer {
	return []autofix.Fixer{
		{
			RuleID:  readiness.RuleViteProfileMode,
			Preview: previewViteProfileMode,
			CanFix:  func(root, _ string) bool { return changes(previewViteProfileMode(root)) },
		},
		{
			RuleID:  readiness.RuleBuildProfileScript,
			Preview: previewBuildProfileScript,
			CanFix:  func(root, _ string) bool { return changes(previewBuildProfileScript(root)) },
		},
		{
			RuleID:  readiness.RuleProfilerUtil,
			Preview: previewProfilerUtil,
			CanFix:  func(root, _ string) bool { return changes(previewProfilerUtil(root)) },
		},
		{
			RuleID:  readiness.RuleProfilerBoundary,
			Preview: previewProfilerBoundary,
			CanFix:  func(root, _ string) bool { return changes(previewProfilerBoundary(root)) },
		},
	}
}

func changes(candidates []Candidate, err error) bool {
	return err == nil && len(candidates) > 0
}

// uiRoot mirrors the readiness package's resolution: prefer <root>/ui, fall
// back to root itself.
func uiRoot(scenarioRoot string) string {
	candidate := filepath.Join(scenarioRoot, "ui")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return candidate
	}
	return scenarioRoot
}

// --- (i) vite.config.ts profile-mode block -------------------------------

func previewViteProfileMode(root string) ([]Candidate, error) {
	path := filepath.Join(uiRoot(root), "vite.config.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No vite config to mechanically fix — leave for manual setup
			// rather than guessing a whole config out of thin air.
			return nil, nil
		}
		return nil, err
	}
	before := string(raw)
	after, changed := injectViteProfileMode(before)
	if !changed {
		return nil, nil
	}
	return []Candidate{{
		RuleID:      readiness.RuleViteProfileMode,
		FilePath:    path,
		Description: "Add the conditional profile-mode block (react-dom/profiling alias + esbuild.keepNames) to vite.config.ts.",
		Before:      before,
		After:       after,
	}}, nil
}

// injectViteProfileMode adds the profile-mode resolve/esbuild block to a config
// whose default export is `defineConfig(({ mode }) => ({ ... }))` or
// `defineConfig(() => ({ ... }))`. It returns (content, false) when the block
// is already present or the config shape is not mechanically recognizable.
func injectViteProfileMode(content string) (string, bool) {
	if strings.Contains(content, "react-dom/profiling") && strings.Contains(content, "keepNames") {
		return content, false
	}
	// Locate the returned config object literal of defineConfig.
	idx := strings.Index(content, "defineConfig(")
	if idx < 0 {
		return content, false
	}
	// Confirm the config has a recognizable returned object literal before
	// mutating: either `=> ({` (arrow with implicit return) or `return {`.
	if !strings.Contains(content[idx:], "=> ({") && !strings.Contains(content[idx:], "return {") {
		return content, false
	}
	// Ensure `mode` is available to gate the block; if defineConfig takes no
	// args, rewrite the empty arg list to destructure mode.
	content = ensureModeParam(content, idx)
	// Compute the insertion point (the brace begins the returned config) after
	// the possible param rewrite.
	region := content[idx:]
	var braceOffset int
	switch {
	case strings.Contains(region, "=> ({"):
		braceOffset = idx + strings.Index(region, "=> ({") + len("=> ({")
	case strings.Contains(region, "return {"):
		braceOffset = idx + strings.Index(region, "return {") + len("return {")
	default:
		return content, false
	}
	block := "\n" + viteProfileBlock()
	out := content[:braceOffset] + block + content[braceOffset:]
	return out, true
}

func ensureModeParam(content string, defineIdx int) string {
	// Match an empty-arg arrow: `defineConfig(() =>` or `defineConfig((): ...`
	region := content[defineIdx:]
	if strings.Contains(region, "({ mode }") || strings.Contains(region, "{ mode }") {
		return content
	}
	for _, empty := range []string{"defineConfig(() =>", "defineConfig((): UserConfig =>", "defineConfig(():"} {
		if at := strings.Index(content[defineIdx:], empty); at >= 0 {
			abs := defineIdx + at
			replaced := strings.Replace(empty, "()", "({ mode })", 1)
			return content[:abs] + replaced + content[abs+len(empty):]
		}
	}
	return content
}

func viteProfileBlock() string {
	return `    // Perf-build channel (managed by Performance Health). A regular build ships
    // the lean prod artifact; ` + "`vite build --mode profile`" + ` keeps React's
    // profiling instrumentation + real component names so <React.Profiler>'s
    // onRender fires and CPU samples display readable names.
    ...(mode === "profile"
      ? {
          resolve: {
            alias: {
              "react-dom/client": "react-dom/profiling",
              "react-dom$": "react-dom/profiling",
            },
          },
          esbuild: { keepNames: true },
        }
      : {}),`
}

// --- (ii) build:profile package.json script ------------------------------

func previewBuildProfileScript(root string) ([]Candidate, error) {
	path := filepath.Join(uiRoot(root), "package.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	before := string(raw)
	after, changed, err := injectBuildProfileScript(before)
	if err != nil || !changed {
		return nil, err
	}
	return []Candidate{{
		RuleID:      readiness.RuleBuildProfileScript,
		FilePath:    path,
		Description: "Add the build:profile script and make build mode-aware in package.json.",
		Before:      before,
		After:       after,
	}}, nil
}

func injectBuildProfileScript(before string) (string, bool, error) {
	var pkg map[string]json.RawMessage
	if err := json.Unmarshal([]byte(before), &pkg); err != nil {
		return before, false, err
	}
	scripts := map[string]string{}
	if rawScripts, ok := pkg["scripts"]; ok {
		if err := json.Unmarshal(rawScripts, &scripts); err != nil {
			return before, false, err
		}
	}
	changed := false
	if _, ok := scripts["build:profile"]; !ok {
		base := scripts["build"]
		if strings.TrimSpace(base) == "" {
			scripts["build:profile"] = "vite build --mode profile"
		} else if strings.Contains(base, "--mode profile") {
			scripts["build:profile"] = base
		} else {
			scripts["build:profile"] = strings.TrimSpace(base) + " --mode profile"
		}
		changed = true
	}
	// Make `build` mode-aware so VROOLI_BUILD_MODE=profile produces the perf
	// bundle through the standard lifecycle path.
	if base, ok := scripts["build"]; ok && !strings.Contains(base, "VROOLI_BUILD_MODE") && !strings.Contains(base, "--mode profile") {
		scripts["build"] = strings.TrimSpace(base) + ` $([ "$VROOLI_BUILD_MODE" = profile ] && echo --mode profile)`
		changed = true
	}
	if !changed {
		return before, false, nil
	}
	out, err := setScriptsBlock(before, scripts)
	if err != nil {
		return before, false, err
	}
	return out, true, nil
}

// setScriptsBlock rewrites only the "scripts" object's value, preserving the
// rest of the file's formatting (key order, comments outside scripts, trailing
// whitespace). The scripts object is re-emitted indented to match.
func setScriptsBlock(before string, scripts map[string]string) (string, error) {
	indent := detectIndent(before)
	rendered := renderScripts(scripts, indent)
	// Replace an existing "scripts": { ... } block when present.
	if start := indexOfKey(before, "scripts"); start >= 0 {
		braceStart := strings.Index(before[start:], "{")
		if braceStart < 0 {
			return before, errors.New("scripts: malformed block")
		}
		braceStart += start
		braceEnd := matchBrace(before, braceStart)
		if braceEnd < 0 {
			return before, errors.New("scripts: unbalanced braces")
		}
		return before[:braceStart] + rendered + before[braceEnd+1:], nil
	}
	// No scripts block: insert after the opening brace of the root object.
	open := strings.Index(before, "{")
	if open < 0 {
		return before, errors.New("package.json: no root object")
	}
	insertion := "\n" + indent + `"scripts": ` + rendered + ","
	return before[:open+1] + insertion + before[open+1:], nil
}

func renderScripts(scripts map[string]string, indent string) string {
	keys := orderedScriptKeys(scripts)
	var b strings.Builder
	b.WriteString("{\n")
	for i, k := range keys {
		v := jsonString(scripts[k])
		kb := jsonString(k)
		b.WriteString(indent + indent + kb + ": " + v)
		if i < len(keys)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString(indent + "}")
	return b.String()
}

// orderedScriptKeys keeps a stable, human-friendly order: build then
// build:profile adjacent, then the remaining keys sorted.
func orderedScriptKeys(scripts map[string]string) []string {
	var out []string
	seen := map[string]bool{}
	for _, k := range []string{"dev", "build", "build:profile"} {
		if _, ok := scripts[k]; ok {
			out = append(out, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range scripts {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sortStrings(rest)
	return append(out, rest...)
}

// --- (iii) src/lib/profiler.ts util --------------------------------------

func previewProfilerUtil(root string) ([]Candidate, error) {
	path := filepath.Join(uiRoot(root), "src", "lib", "profiler.ts")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	before := string(existing)
	if strings.Contains(before, "performance.measure") && strings.Contains(before, "⚛") &&
		(strings.Contains(before, "onProfilerRender") || strings.Contains(before, "onRender")) {
		return nil, nil
	}
	after := profilerUtilSource()
	if after == before {
		return nil, nil
	}
	return []Candidate{{
		RuleID:      readiness.RuleProfilerUtil,
		FilePath:    path,
		Description: "Scaffold src/lib/profiler.ts with the onProfilerRender handler emitting performance.measure(\"⚛ …\").",
		Before:      before,
		After:       after,
	}}, nil
}

func profilerUtilSource() string {
	return `/**
 * Shared <React.Profiler> onRender callback (managed by Performance Health).
 *
 * Wrapping a subtree in <Profiler id="X" onRender={onProfilerRender}> emits a
 * performance.measure entry every time that subtree commits. The "⚛" prefix
 * groups them in Chrome DevTools' Performance panel. onRender only fires when
 * React's profiling instrumentation is present (the perf-build channel), so the
 * wrapper is inert in regular prod and safe to ship permanently.
 */

import type { ProfilerOnRenderCallback } from "react";

export const onProfilerRender: ProfilerOnRenderCallback = (
  id,
  phase,
  actualDuration,
) => {
  try {
    performance.measure(` + "`⚛ ${id} (${phase})`" + `, {
      start: performance.now() - actualDuration,
      duration: actualDuration,
    });
  } catch {
    // Never let a measurement failure surface in the running app.
  }
};
`
}

// --- (iv) src/main.tsx <React.Profiler> boundary -------------------------

func previewProfilerBoundary(root string) ([]Candidate, error) {
	path := filepath.Join(uiRoot(root), "src", "main.tsx")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	before := string(raw)
	after, changed := injectProfilerBoundary(before)
	if !changed {
		return nil, nil
	}
	return []Candidate{{
		RuleID:      readiness.RuleProfilerBoundary,
		FilePath:    path,
		Description: "Wrap the rendered app in a top-level <React.Profiler> boundary wired to onProfilerRender in src/main.tsx.",
		Before:      before,
		After:       after,
	}}, nil
}

// injectProfilerBoundary wraps the top-level <App /> render in a
// <React.Profiler> boundary and ensures onProfilerRender is imported. It works
// on the canonical react-vite entry shape and is a no-op when a boundary is
// already wired.
func injectProfilerBoundary(before string) (string, bool) {
	if strings.Contains(before, "React.Profiler") || strings.Contains(before, "<Profiler") {
		return before, false
	}
	app := "<App />"
	at := strings.Index(before, app)
	if at < 0 {
		return before, false
	}
	// Add the required imports BEFORE wrapping so the import check inspects the
	// original file's import statements, not the JSX we are about to insert.
	withImports := ensureProfilerImports(before)
	// Re-locate the render site (imports prepended above may have shifted it).
	at = strings.Index(withImports, app)
	indent := lineIndent(withImports, at)
	wrapped := `<React.Profiler id="App" onRender={onProfilerRender}>` + "\n" +
		indent + "  " + app + "\n" +
		indent + "</React.Profiler>"
	out := withImports[:at] + wrapped + withImports[at+len(app):]
	return out, true
}

// ensureProfilerImports adds `import React from "react"` and
// `import { onProfilerRender } from "./lib/profiler"` when the entry's import
// statements do not already provide them. It inspects import lines only, so a
// later JSX reference cannot mask a missing import.
func ensureProfilerImports(content string) string {
	importsOnly := importSection(content)
	var additions []string
	if !strings.Contains(importsOnly, "import React ") && !strings.Contains(importsOnly, "import React,") {
		additions = append(additions, `import React from "react";`)
	}
	if !strings.Contains(importsOnly, "onProfilerRender") {
		additions = append(additions, `import { onProfilerRender } from "./lib/profiler";`)
	}
	if len(additions) == 0 {
		return content
	}
	block := strings.Join(additions, "\n")
	if at := strings.LastIndex(content, "\nimport "); at >= 0 {
		lineEnd := strings.Index(content[at+1:], "\n")
		if lineEnd >= 0 {
			pos := at + 1 + lineEnd
			return content[:pos] + "\n" + block + content[pos:]
		}
	}
	return block + "\n" + content
}

// importSection returns the contiguous leading import statements of a module so
// import-presence checks ignore later code that merely references a symbol.
func importSection(content string) string {
	var b strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || trimmed == "" {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		break
	}
	return b.String()
}

// --- small helpers (no third-party deps) ---------------------------------

func detectIndent(content string) string {
	// Find the first indented line and reuse its leading whitespace.
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" || trimmed == line {
			continue
		}
		return line[:len(line)-len(trimmed)]
	}
	return "  "
}

func lineIndent(content string, at int) string {
	lineStart := strings.LastIndex(content[:at], "\n") + 1
	prefix := content[lineStart:at]
	return prefix[:len(prefix)-len(strings.TrimLeft(prefix, " \t"))]
}

// indexOfKey finds the byte offset of a top-level-ish quoted JSON key.
func indexOfKey(content, key string) int {
	return strings.Index(content, `"`+key+`"`)
}

// matchBrace returns the offset of the brace matching the one at openIdx.
func matchBrace(content string, openIdx int) int {
	depth := 0
	inString := false
	escaped := false
	for i := openIdx; i < len(content); i++ {
		c := content[i]
		if inString {
			switch {
			case escaped:
				escaped = false
			case c == '\\':
				escaped = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// jsonString encodes a Go string as a JSON string literal WITHOUT HTML escaping
// so shell operators in scripts (&&, $, etc.) survive verbatim.
func jsonString(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimRight(b.String(), "\n")
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
