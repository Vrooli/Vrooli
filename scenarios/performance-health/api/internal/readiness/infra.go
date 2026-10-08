package readiness

import (
	"os"
	"path/filepath"
	"strings"
)

// The four Tier-1 perf-build infra rule IDs. They are the single source of
// truth shared by the readiness detector (which emits a finding per missing
// piece) and the autofix domain (which registers one Fixer per rule). Keeping
// the IDs here means a finding's Code lines up exactly with the Fixer.RuleID a
// caller passes to `readiness fix --rule <code>`.
const (
	// RuleViteProfileMode covers the vite profile-mode branch: the
	// react-dom/profiling alias AND esbuild.keepNames, both gated on a
	// profile build mode. Detected in ui/vite.config.ts.
	RuleViteProfileMode = "perf_build_vite_profile_mode"
	// RuleBuildProfileScript covers the package.json build:profile script
	// (and the mode-aware build script). Detected in ui/package.json.
	RuleBuildProfileScript = "perf_build_profile_script"
	// RuleProfilerUtil covers ui/src/lib/profiler.ts exporting an onRender
	// handler that emits performance.measure("⚛ …").
	RuleProfilerUtil = "perf_build_profiler_util"
	// RuleProfilerBoundary covers the top-level <React.Profiler> boundary in
	// the app root (main.tsx) wired to the profiler util.
	RuleProfilerBoundary = "perf_build_profiler_boundary"
)

// infraPiece is one detectable perf-build infra requirement. Present reports
// whether the piece is already in place; FilePath is the file the finding (and
// the fixer) targets, relative paths resolved under uiRoot.
type infraPiece struct {
	RuleID   string
	Title    string
	Missing  string // human-readable "what's missing" message when absent
	FilePath func(uiRoot string) string
	Present  func(uiRoot string) bool
}

// perfBuildInfra returns the four perf-build infra pieces in detection order.
// Both the readiness detector and the autofix registry iterate this slice so
// the detect/fix sets never drift.
func perfBuildInfra() []infraPiece {
	return []infraPiece{
		{
			RuleID:   RuleViteProfileMode,
			Title:    "Vite profile-mode perf build",
			Missing:  "vite.config.ts lacks a profile build mode aliasing react-dom/client → react-dom/profiling with esbuild.keepNames:true.",
			FilePath: func(ui string) string { return filepath.Join(ui, "vite.config.ts") },
			Present:  viteProfileModePresent,
		},
		{
			RuleID:   RuleBuildProfileScript,
			Title:    "build:profile package script",
			Missing:  "package.json lacks a build:profile script that builds with --mode profile.",
			FilePath: func(ui string) string { return filepath.Join(ui, "package.json") },
			Present:  buildProfileScriptPresent,
		},
		{
			RuleID:   RuleProfilerUtil,
			Title:    "onProfilerRender util",
			Missing:  "src/lib/profiler.ts is missing an onRender handler that emits performance.measure(\"⚛ …\").",
			FilePath: func(ui string) string { return filepath.Join(ui, "src", "lib", "profiler.ts") },
			Present:  profilerUtilPresent,
		},
		{
			RuleID:   RuleProfilerBoundary,
			Title:    "top-level <React.Profiler> boundary",
			Missing:  "src/main.tsx lacks a top-level <React.Profiler> boundary wired to onProfilerRender.",
			FilePath: func(ui string) string { return filepath.Join(ui, "src", "main.tsx") },
			Present:  profilerBoundaryPresent,
		},
	}
}

// uiRoot resolves the UI surface directory for a scenario root. The canonical
// layout is <root>/ui; when that directory is absent the scenario root itself
// is returned so detectors fail closed (everything reported missing) rather
// than panicking.
func uiRoot(scenarioRoot string) string {
	candidate := filepath.Join(scenarioRoot, "ui")
	if st, err := os.Stat(candidate); err == nil && st.IsDir() {
		return candidate
	}
	return scenarioRoot
}

// readFile returns the file contents, or empty string when the file is absent
// or unreadable. Detectors treat both as "piece missing".
func readFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func viteProfileModePresent(ui string) bool {
	content := readFile(filepath.Join(ui, "vite.config.ts"))
	if content == "" {
		return false
	}
	hasAlias := strings.Contains(content, "react-dom/profiling")
	hasKeepNames := strings.Contains(content, "keepNames")
	// The alias + keepNames must be conditional on a profile mode/env so the
	// default prod bundle stays lean — a bare unconditional alias is not the
	// perf-build infra the skill prescribes.
	hasModeGate := strings.Contains(content, "profile")
	return hasAlias && hasKeepNames && hasModeGate
}

func buildProfileScriptPresent(ui string) bool {
	content := readFile(filepath.Join(ui, "package.json"))
	if content == "" {
		return false
	}
	return strings.Contains(content, `"build:profile"`) && strings.Contains(content, "--mode profile")
}

func profilerUtilPresent(ui string) bool {
	content := readFile(filepath.Join(ui, "src", "lib", "profiler.ts"))
	if content == "" {
		return false
	}
	emitsMeasure := strings.Contains(content, "performance.measure") && strings.Contains(content, "⚛")
	exportsHandler := strings.Contains(content, "onProfilerRender") || strings.Contains(content, "onRender")
	return emitsMeasure && exportsHandler
}

func profilerBoundaryPresent(ui string) bool {
	content := readFile(filepath.Join(ui, "src", "main.tsx"))
	if content == "" {
		return false
	}
	hasBoundary := strings.Contains(content, "React.Profiler") || strings.Contains(content, "<Profiler")
	wiredToUtil := strings.Contains(content, "onProfilerRender")
	return hasBoundary && wiredToUtil
}

// detectInfra returns a finding per missing perf-build infra piece for the UI
// root, each flagged autofixable (every piece has a deterministic fixer).
func detectInfra(scenarioRoot string) []Finding {
	ui := uiRoot(scenarioRoot)
	var findings []Finding
	for _, piece := range perfBuildInfra() {
		if piece.Present(ui) {
			continue
		}
		findings = append(findings, Finding{
			Code:        piece.RuleID,
			Message:     piece.Missing + " (" + piece.FilePath(ui) + ")",
			Severity:    "warning",
			Autofixable: true,
		})
	}
	return findings
}

// infraComplete reports whether all four perf-build infra pieces are present —
// i.e. the scenario is Tier-1-ready rather than React-but-divergent.
func infraComplete(scenarioRoot string) bool {
	ui := uiRoot(scenarioRoot)
	for _, piece := range perfBuildInfra() {
		if !piece.Present(ui) {
			return false
		}
	}
	return true
}
