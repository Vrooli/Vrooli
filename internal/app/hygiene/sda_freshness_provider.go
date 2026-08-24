package hygiene

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const dependencyFreshnessProviderID = "dependency_freshness"

type sdaFreshnessProvider struct {
	root   string
	runner DependencyFreshnessRunner
}

type DependencyFreshnessRunner interface {
	CheckDependencyFreshness(ctx context.Context, root string) (sdaFreshnessReport, error)
}

type commandDependencyFreshnessRunner struct{}

type sdaFreshnessReport struct {
	Clean       bool                  `json:"clean"`
	Root        string                `json:"root"`
	Mode        string                `json:"mode"`
	Touched     []string              `json:"touched"`
	Surfaces    []sdaFreshnessSurface `json:"surfaces"`
	Summary     sdaFreshnessSummary   `json:"summary"`
	NextActions []sdaFreshnessAction  `json:"next_actions"`
	ElapsedMs   int64                 `json:"elapsed_ms"`
}

type sdaFreshnessSummary struct {
	Checked       int `json:"checked"`
	Stale         int `json:"stale"`
	Errors        int `json:"errors"`
	NeedsDownload int `json:"needs_download"`
}

type sdaFreshnessSurface struct {
	Scenario  string   `json:"scenario"`
	Surface   string   `json:"surface"`
	GoModPath string   `json:"go_mod_path"`
	Status    string   `json:"status"`
	DiffPaths []string `json:"diff_paths"`
	Error     string   `json:"error"`
}

type sdaFreshnessAction struct {
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	Command    string     `json:"command"`
	Fixability Fixability `json:"fixability,omitempty"`
}

func (p sdaFreshnessProvider) ID() string { return dependencyFreshnessProviderID }

func (p sdaFreshnessProvider) Budget() time.Duration { return 45 * time.Second }

func (p sdaFreshnessProvider) Run(ctx context.Context, _ Request, report *Report) error {
	root := filepath.Clean(strings.TrimSpace(p.root))
	runner := p.runner
	if runner == nil {
		runner = commandDependencyFreshnessRunner{}
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	freshness, err := runner.CheckDependencyFreshness(ctx, root)
	if err != nil {
		p.reportUnavailable(report, err.Error())
		return nil
	}
	p.apply(report, freshness)
	return nil
}

func (commandDependencyFreshnessRunner) CheckDependencyFreshness(ctx context.Context, root string) (sdaFreshnessReport, error) {
	cmd := exec.CommandContext(ctx, "scenario-dependency-analyzer", "freshness", "--touched", "--json", "--repo-root", root)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		detail := err.Error()
		if exitErr, ok := err.(*exec.ExitError); ok {
			detail = strings.TrimSpace(string(exitErr.Stderr))
			if detail == "" {
				detail = err.Error()
			}
		}
		return sdaFreshnessReport{}, fmt.Errorf("%s", detail)
	}
	var freshness sdaFreshnessReport
	if err := json.Unmarshal(out, &freshness); err != nil {
		return sdaFreshnessReport{}, fmt.Errorf("parse SDA freshness JSON: %v", err)
	}
	return freshness, nil
}

func (p sdaFreshnessProvider) reportUnavailable(report *Report, reason string) {
	message := fmt.Sprintf("Scenario Dependency Analyzer freshness provider unavailable: %s", reason)
	action := Action{
		Code:       "inspect_sda_freshness",
		Message:    "Inspect the SDA freshness command and rerun hygiene after correcting the provider failure.",
		Command:    "scenario-dependency-analyzer freshness --touched --json",
		Fixability: FixabilityGuided,
	}
	report.addCheck("dependency_freshness", false, SeverityError, message)
	report.addFinding(Finding{
		Severity:    SeverityError,
		Code:        "dependency_freshness_provider",
		Message:     message,
		Why:         "Scenario Dependency Analyzer owns dependency freshness; root hygiene only aggregates the provider result.",
		Fixability:  FixabilityGuided,
		NextActions: []Action{action},
	})
	report.Actions = append(report.Actions, action)
}

func (p sdaFreshnessProvider) apply(report *Report, freshness sdaFreshnessReport) {
	drift := p.compatSharedDrift(freshness)
	report.SharedDrift = &drift
	if freshness.Summary.NeedsDownload > 0 {
		report.addCheck("dependency_freshness_cold_cache", true, SeverityInfo, fmt.Sprintf("SDA needed downloads for %d cold-cache surface(s); this is host state, not a surface defect", freshness.Summary.NeedsDownload))
	}
	if freshness.Clean {
		message := "SDA reports no stale dependency surfaces"
		if freshness.Summary.Checked == 0 {
			message = "SDA reports no touched dependency surfaces"
		}
		report.addCheck("dependency_freshness", true, SeverityInfo, message)
		return
	}

	message := fmt.Sprintf("SDA reports %d stale and %d errored dependency surface(s)", freshness.Summary.Stale, freshness.Summary.Errors)
	report.addCheck("dependency_freshness", false, SeverityError, message)
	actions := make([]Action, 0, len(freshness.NextActions))
	for _, next := range freshness.NextActions {
		command := strings.TrimSpace(next.Command)
		if command == "" {
			command = "scenario-dependency-analyzer freshness --touched --apply"
		}
		actions = append(actions, Action{
			Code:       firstNonEmpty(next.Code, "fix_dependency_freshness"),
			Message:    firstNonEmpty(next.Message, "Run SDA-owned package freshness repair for impacted Go surfaces."),
			Command:    command,
			Fixability: dependencyFreshnessActionFixability(next),
		})
	}
	if len(actions) == 0 {
		actions = append(actions, Action{
			Code:       "inspect_dependency_freshness",
			Message:    "Inspect SDA package freshness findings.",
			Command:    "scenario-dependency-analyzer freshness --touched --json",
			Fixability: FixabilityGuided,
		})
	}
	report.addFinding(Finding{
		Severity:    SeverityError,
		Code:        "dependency_freshness",
		Message:     message,
		Why:         "Stale dependency metadata causes scenario starts and tests to fail after in-repo packages change.",
		Locations:   freshnessLocations(freshness),
		Fixability:  dependencyFreshnessFixability(freshness),
		NextActions: actions,
	})
	report.Actions = append(report.Actions, actions...)
}

func dependencyFreshnessFixability(freshness sdaFreshnessReport) Fixability {
	if freshness.Summary.Errors > 0 {
		return FixabilityGuided
	}
	if freshness.Summary.Stale > 0 {
		return FixabilityAutomatic
	}
	return FixabilityGuided
}

func dependencyFreshnessActionFixability(action sdaFreshnessAction) Fixability {
	if action.Fixability != "" {
		return action.Fixability
	}
	switch action.Code {
	case "apply_go_tidy":
		return FixabilityAutomatic
	case "preview_missing_local_replaces":
		return FixabilityGuided
	default:
		return FixabilityGuided
	}
}

func (p sdaFreshnessProvider) compatSharedDrift(freshness sdaFreshnessReport) DependencyFreshnessCompatReport {
	out := DependencyFreshnessCompatReport{
		Clean:           freshness.Clean,
		Root:            firstNonEmpty(freshness.Root, p.root),
		TouchedPackages: freshness.Touched,
		OnlyTouchedUsed: freshness.Mode == "touched",
		ElapsedMs:       freshness.ElapsedMs,
	}
	for _, surface := range freshness.Surfaces {
		status := DependencyFreshnessStatusClean
		switch surface.Status {
		case "stale":
			status = DependencyFreshnessStatusStaleModules
		case "error":
			status = DependencyFreshnessStatusError
		}
		out.Scenarios = append(out.Scenarios, DependencyFreshnessScenario{
			Path:      filepath.ToSlash(filepath.Join("scenarios", surface.Scenario)),
			APIDir:    surfaceDir(surface.GoModPath),
			Status:    status,
			DiffPaths: surface.DiffPaths,
			Error:     surface.Error,
		})
	}
	return out
}

func freshnessLocations(report sdaFreshnessReport) []string {
	var out []string
	for _, surface := range report.Surfaces {
		if surface.Status == "clean" {
			continue
		}
		if len(surface.DiffPaths) > 0 {
			out = append(out, surface.DiffPaths...)
			continue
		}
		if surface.GoModPath != "" {
			out = append(out, surface.GoModPath)
		}
	}
	const maxLocations = 5
	out = compact(out)
	if len(out) > maxLocations {
		return append(out[:maxLocations], fmt.Sprintf("... %d more (see dependency_freshness summary)", len(out)-maxLocations))
	}
	return out
}

func surfaceDir(goModPath string) string {
	if goModPath == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Dir(goModPath))
}

func compact(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
