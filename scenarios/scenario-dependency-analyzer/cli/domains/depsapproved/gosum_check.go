package depsapproved

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"scenario-dependency-analyzer/cli/internal/support"
)

// goDepsLister resolves a module's package graph and returns the combined
// toolchain output. It is injectable so the drift check is testable without
// executing the Go toolchain.
type goDepsLister func(dir string) (string, error)

// goSumDrift names one module whose go.sum is missing an entry required by an
// in-repo module requirement.
type goSumDrift struct {
	Module  string   `json:"module"`
	Details []string `json:"details,omitempty"`
}

// goSumDriftReport is the machine-readable result of `deps reconcile --check`.
type goSumDriftReport struct {
	Check   string       `json:"check"`
	Modules int          `json:"modules_checked"`
	Drift   []goSumDrift `json:"drift,omitempty"`
}

// checkGoSumDrift resolves every module in moduleDirs and reports the ones whose
// go.sum drifts behind an in-repo module's transitive requirements. The Go
// toolchain fails such a module with a `missing go.sum entry` diagnostic and no
// other error is treated as drift, so genuine compile errors are ignored.
func checkGoSumDrift(moduleDirs []string, list goDepsLister) goSumDriftReport {
	report := goSumDriftReport{Check: "gosum-drift"}
	sorted := append([]string(nil), moduleDirs...)
	sort.Strings(sorted)
	for _, dir := range sorted {
		report.Modules++
		out, err := list(dir)
		if err == nil {
			continue
		}
		details := missingGoSumDetails(out)
		if len(details) == 0 {
			continue
		}
		report.Drift = append(report.Drift, goSumDrift{Module: dir, Details: details})
	}
	return report
}

// reconcileCheck runs the drift check and returns a non-nil error (non-zero
// exit) when any module drifts, so the check can gate a fleet operation.
func reconcileCheck(moduleDirs []string, list goDepsLister, jsonOutput bool) error {
	report := checkGoSumDrift(moduleDirs, list)
	if jsonOutput {
		if err := support.PrintReportJSON(report); err != nil {
			return err
		}
	} else {
		fmt.Printf("gosum-drift: checked %d module(s)\n", report.Modules)
		for _, drift := range report.Drift {
			fmt.Printf("  %s\n", drift.Module)
			for _, detail := range drift.Details {
				fmt.Printf("    %s\n", detail)
			}
		}
	}
	if len(report.Drift) > 0 {
		return fmt.Errorf("gosum-drift: %d module(s) missing go.sum entries for in-repo module requirements", len(report.Drift))
	}
	if !jsonOutput {
		fmt.Println("gosum-drift: pass")
	}
	return nil
}

// missingGoSumDetails extracts the distinct `missing go.sum entry` lines from
// toolchain output.
func missingGoSumDetails(output string) []string {
	var details []string
	seen := map[string]struct{}{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "missing go.sum entry") {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		details = append(details, line)
	}
	sort.Strings(details)
	return details
}

// goListDeps is the production lister: it resolves the module's package graph,
// which exercises go.sum verification for every imported package. It runs with
// -mod=readonly so the check never writes go.mod or go.sum — a drifted module
// fails with a `missing go.sum entry` diagnostic instead of being silently
// repaired.
func goListDeps(dir string) (string, error) {
	cmd := exec.Command("go", "list", "-deps", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=readonly")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// tidySurface runs `go mod tidy` for one module directory and reports whether
// go.sum changed. It is the client-side lockfile-sync step of
// `deps reconcile --apply`: syncing go.sum for the already-approved packages an
// in-repo module requires is lockfile sync, not new dependency approval.
func tidySurface(dir string) (bool, error) {
	sumPath := filepath.Join(dir, "go.sum")
	sumBefore, _ := os.ReadFile(sumPath)
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	if out, err := cmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	sumAfter, _ := os.ReadFile(sumPath)
	return !bytes.Equal(sumBefore, sumAfter), nil
}

// filterDirsBySurface keeps module directories that belong to the named surface
// (api|cli|ui|…) when one is requested.
func filterDirsBySurface(dirs []string, surface string) []string {
	surface = strings.TrimSpace(surface)
	if surface == "" {
		return dirs
	}
	needle := "/" + surface + "/"
	kept := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		if strings.Contains(filepath.ToSlash(dir)+"/", needle) {
			kept = append(kept, dir)
		}
	}
	return kept
}

// scenarioGoMods returns the go.mod files under a scenario directory.
func scenarioGoMods(scenarioDir string) []string {
	var matches []string
	_ = filepath.WalkDir(scenarioDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "vendor", "data", "dist", "build", ".cache", "phase-cache", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" {
			matches = append(matches, path)
		}
		return nil
	})
	sort.Strings(matches)
	return matches
}
