package dev_tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

// defaultScoreFloor is the completeness score a golden must meet or
// exceed by default. Overridable per tool via the expectation file.
const defaultScoreFloor = 96.0

// expectationsRelDir is the repo-relative directory holding the
// committed per-tool expectation files. It is the seam PROBLEMS.md
// (2026-05-18) named for tool expectation parsing.
const expectationsRelDir = "scenarios/development-toolchain-validator/data/tool-expectations"

// Expectation is the per-tool success contract for validating a golden.
// It is loaded from data/tool-expectations/<tool>.json; absent or
// partial files fall back to built-in defaults so a tool always has a
// usable expectation.
type Expectation struct {
	// Tool is the tool name this expectation applies to (informational).
	Tool string `json:"tool"`
	// Preset is the test-genie preset to run. Defaults to "comprehensive"
	// (every catalog phase) when empty. An operator may pin a lighter
	// preset (e.g. "architecture-audit") when runtime infra is
	// unavailable — but comprehensive is the intended default.
	Preset string `json:"preset,omitempty"`
	// ScoreFloor is the minimum completeness score a golden must meet.
	// Defaults to defaultScoreFloor when zero.
	ScoreFloor float64 `json:"score_floor,omitempty"`
	// TemplateVersionPin / ToolVersionPin record the template + tool
	// versions this expectation was last calibrated against. Stored for
	// staleness reasoning; not yet auto-enforced.
	TemplateVersionPin string `json:"template_version_pin,omitempty"`
	ToolVersionPin     string `json:"tool_version_pin,omitempty"`
}

// defaultExpectation returns the built-in expectation for a tool, used
// when no expectation file is present on disk.
func defaultExpectation(tool string) Expectation {
	return Expectation{
		Tool:       tool,
		Preset:     "comprehensive",
		ScoreFloor: defaultScoreFloor,
	}
}

// resolveExpectation reads <dir>/<tool>.json and overlays it onto the
// built-in default. A missing file yields the default; a malformed file
// is an error so a typo fails loudly rather than silently using
// defaults. When dir is empty it is resolved from the repo root.
func resolveExpectation(dir, tool string) (Expectation, error) {
	exp := defaultExpectation(tool)

	resolvedDir := dir
	if strings.TrimSpace(resolvedDir) == "" {
		root, err := repocontract.FindRepoRootFromEnvOrCWD()
		if err != nil {
			// No repo root resolvable (e.g. a stripped runtime): fall back
			// to the built-in default rather than failing the whole run.
			return exp, nil
		}
		resolvedDir = filepath.Join(root, filepath.FromSlash(expectationsRelDir))
	}

	path := filepath.Join(resolvedDir, tool+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return exp, nil
		}
		return Expectation{}, fmt.Errorf("read expectation %q: %w", path, err)
	}

	var fileExp Expectation
	if err := json.Unmarshal(raw, &fileExp); err != nil {
		return Expectation{}, fmt.Errorf("parse expectation %q: %w", path, err)
	}
	// Overlay non-zero file fields onto the default.
	if strings.TrimSpace(fileExp.Tool) != "" {
		exp.Tool = fileExp.Tool
	}
	if strings.TrimSpace(fileExp.Preset) != "" {
		exp.Preset = fileExp.Preset
	}
	if fileExp.ScoreFloor > 0 {
		exp.ScoreFloor = fileExp.ScoreFloor
	}
	exp.TemplateVersionPin = fileExp.TemplateVersionPin
	exp.ToolVersionPin = fileExp.ToolVersionPin
	return exp, nil
}
