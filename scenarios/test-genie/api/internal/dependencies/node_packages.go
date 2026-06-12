package dependencies

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// NodePackageChecker validates JavaScript package-manager/install state.
type NodePackageChecker interface {
	Check() NodePackageResult
}

type NodePackageResult struct {
	Success      bool
	Error        error
	FailureClass FailureClass
	Remediation  string
	Observations []Observation
	Checked      int
}

type nodePackageChecker struct {
	scenarioDir string
	settings    NodePackageSettings
}

type nodeWorkspace struct {
	dir       string
	rel       string
	lockfiles []lockfile
}

type lockfile struct {
	path    string
	manager string
}

func NewNodePackageChecker(scenarioDir string, settings NodePackageSettings) NodePackageChecker {
	return &nodePackageChecker{scenarioDir: scenarioDir, settings: settings}
}

func (c *nodePackageChecker) Check() NodePackageResult {
	if !c.settings.Enabled {
		return NodePackageResult{
			Success:      true,
			Observations: []Observation{NewSkipObservation("Node package checks disabled via .vrooli/testing.json")},
		}
	}
	workspaces := c.detectWorkspaces()
	if len(workspaces) == 0 {
		return NodePackageResult{
			Success:      true,
			Observations: []Observation{NewInfoObservation("no JavaScript package workspace detected")},
		}
	}
	var observations []Observation
	var failures []string
	for _, workspace := range workspaces {
		if len(workspace.lockfiles) == 0 && c.settings.LockfileRequired {
			failures = append(failures, fmt.Sprintf("%s missing lockfile", workspace.rel))
		}
		if len(workspace.lockfiles) > 1 {
			var names []string
			for _, lf := range workspace.lockfiles {
				names = append(names, filepath.Base(lf.path))
			}
			sort.Strings(names)
			failures = append(failures, fmt.Sprintf("%s has multiple lockfiles: %s", workspace.rel, strings.Join(names, ", ")))
		}
		if len(workspace.lockfiles) == 1 {
			lf := workspace.lockfiles[0]
			observations = append(observations, NewSuccessObservation(
				fmt.Sprintf("JavaScript lockfile uses %s: %s", lf.manager, filepath.ToSlash(lf.path)),
			))
		}
		if c.settings.RequireNodeModules && !dirExists(filepath.Join(workspace.dir, "node_modules")) {
			failures = append(failures, fmt.Sprintf("%s missing node_modules", workspace.rel))
		}
	}
	if len(failures) > 0 {
		sort.Strings(failures)
		return NodePackageResult{
			Success:      false,
			Error:        fmt.Errorf("JavaScript package state invalid: %s", strings.Join(failures, "; ")),
			FailureClass: FailureClassMissingDependency,
			Remediation:  "Install JavaScript dependencies in the reported workspace, for example `pnpm install --ignore-workspace`.",
			Observations: append(observations, NewErrorObservation("node_install_state_stale: "+strings.Join(failures, "; "))),
			Checked:      len(workspaces),
		}
	}
	observations = append(observations, NewSuccessObservation("JavaScript package state is ready"))
	return NodePackageResult{Success: true, Observations: observations, Checked: len(workspaces)}
}

func (c *nodePackageChecker) detectWorkspaces() []nodeWorkspace {
	candidates := []struct {
		dir string
		rel string
	}{
		{dir: c.scenarioDir, rel: "."},
		{dir: filepath.Join(c.scenarioDir, "ui"), rel: "ui"},
	}
	var workspaces []nodeWorkspace
	for _, candidate := range candidates {
		if !fileExists(filepath.Join(candidate.dir, "package.json")) {
			continue
		}
		workspaces = append(workspaces, nodeWorkspace{
			dir:       candidate.dir,
			rel:       candidate.rel,
			lockfiles: detectLockfiles(candidate.dir),
		})
	}
	return workspaces
}

func detectLockfiles(dir string) []lockfile {
	candidates := []lockfile{
		{path: filepath.Join(dir, "pnpm-lock.yaml"), manager: "pnpm"},
		{path: filepath.Join(dir, "package-lock.json"), manager: "npm"},
		{path: filepath.Join(dir, "yarn.lock"), manager: "yarn"},
	}
	var out []lockfile
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate.path); err == nil && !info.IsDir() {
			out = append(out, candidate)
		}
	}
	return out
}

var _ NodePackageChecker = (*nodePackageChecker)(nil)
