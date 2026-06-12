package dependencies

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GoModuleChecker validates Go module freshness for the target scenario.
type GoModuleChecker interface {
	Check(ctx context.Context) GoModuleResult
}

type GoModuleResult struct {
	Success      bool
	Error        error
	Remediation  string
	Observations []Observation
	Checked      int
}

type goModuleChecker struct {
	scenarioDir string
	settings    GoModuleSettings
}

func NewGoModuleChecker(scenarioDir string, settings GoModuleSettings) GoModuleChecker {
	return &goModuleChecker{scenarioDir: scenarioDir, settings: settings}
}

func (c *goModuleChecker) Check(ctx context.Context) GoModuleResult {
	if !c.settings.Enabled {
		return GoModuleResult{
			Success:      true,
			Observations: []Observation{NewSkipObservation("Go module checks disabled via .vrooli/testing.json")},
		}
	}
	apiDir := filepath.Join(c.scenarioDir, "api")
	modPath := filepath.Join(apiDir, "go.mod")
	if !fileExists(modPath) {
		return GoModuleResult{
			Success:      true,
			Observations: []Observation{NewInfoObservation("no scenario API go.mod detected")},
		}
	}
	var observations []Observation
	if c.settings.LocalReplaceResolution {
		missing, err := missingLocalReplaces(apiDir, modPath)
		if err != nil {
			return GoModuleResult{
				Success:      false,
				Error:        fmt.Errorf("read local replace directives: %w", err),
				Remediation:  "Fix api/go.mod so local replace directives can be read.",
				Observations: observations,
				Checked:      1,
			}
		}
		if len(missing) > 0 {
			return GoModuleResult{
				Success:      false,
				Error:        fmt.Errorf("local replace targets are missing: %s", strings.Join(missing, ", ")),
				Remediation:  "Update api/go.mod replace directives so every local target exists.",
				Observations: append(observations, NewErrorObservation("local replace target missing: "+strings.Join(missing, ", "))),
				Checked:      1,
			}
		}
		observations = append(observations, NewSuccessObservation("Go local replace targets resolve"))
	}
	if c.settings.TidyDiff {
		out, err := runGo(ctx, apiDir, "mod", "tidy", "-diff")
		if err != nil {
			text := strings.TrimSpace(out)
			if text != "" {
				return GoModuleResult{
					Success:      false,
					Error:        fmt.Errorf("go module drift detected in %s", filepath.ToSlash(modPath)),
					Remediation:  "Run `cd " + filepath.ToSlash(apiDir) + " && GOWORK=off go mod tidy`, then rerun the dependencies phase.",
					Observations: append(observations, NewErrorObservation("go_module_drift: api/go.mod or api/go.sum is stale")),
					Checked:      1,
				}
			}
			return GoModuleResult{
				Success:      false,
				Error:        fmt.Errorf("go mod tidy -diff failed: %w", err),
				Remediation:  "Inspect `go mod tidy -diff` output from the scenario API directory.",
				Observations: observations,
				Checked:      1,
			}
		}
		observations = append(observations, NewSuccessObservation("Go module metadata is tidy"))
	}
	if c.settings.Build {
		out, err := runGo(ctx, apiDir, "build", "./...")
		if err != nil {
			return GoModuleResult{
				Success:      false,
				Error:        fmt.Errorf("go build ./... failed: %s", strings.TrimSpace(out)),
				Remediation:  "Fix Go build errors or disable dependencies.go_modules.build only when another required phase owns the build gate.",
				Observations: append(observations, NewErrorObservation("Go build probe failed")),
				Checked:      1,
			}
		}
		observations = append(observations, NewSuccessObservation("Go build probe passed"))
	}
	return GoModuleResult{Success: true, Observations: observations, Checked: 1}
}

func runGo(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

var localReplaceRE = regexp.MustCompile(`^([^\s]+)(?:\s+v\S+)?\s+=>\s+(\S+)`)

func missingLocalReplaces(apiDir, modPath string) ([]string, error) {
	data, err := os.ReadFile(modPath)
	if err != nil {
		return nil, err
	}
	var missing []string
	inBlock := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if miss := missingReplaceTarget(apiDir, line); miss != "" {
				missing = append(missing, miss)
			}
			continue
		}
		if strings.HasPrefix(line, "replace (") {
			inBlock = true
			continue
		}
		if strings.HasPrefix(line, "replace ") {
			if miss := missingReplaceTarget(apiDir, strings.TrimPrefix(line, "replace ")); miss != "" {
				missing = append(missing, miss)
			}
		}
	}
	return missing, nil
}

func missingReplaceTarget(apiDir, line string) string {
	match := localReplaceRE.FindStringSubmatch(line)
	if match == nil || !isLocalPath(match[2]) {
		return ""
	}
	target := match[2]
	if !filepath.IsAbs(target) {
		target = filepath.Join(apiDir, filepath.FromSlash(target))
	}
	if dirExists(target) {
		return ""
	}
	return filepath.ToSlash(match[2])
}

func isLocalPath(path string) bool {
	return strings.HasPrefix(path, ".") || filepath.IsAbs(path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

var _ GoModuleChecker = (*goModuleChecker)(nil)
