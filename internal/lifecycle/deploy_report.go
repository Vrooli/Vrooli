package lifecycle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
)

// Running-build states reported per artifact.
const (
	DeployRunningCurrent  = "current"     // a process runs the artifact on disk
	DeployRunningReplaced = "replaced"    // a process runs an older binary that has since been replaced
	DeployRunningNone     = "not_running" // no process runs the artifact
	DeployRunningUnknown  = "unknown"     // the host cannot tell (no /proc)
	// DeployRunningNotApplicable marks an artifact that is not an executable
	// (a UI bundle served by another process).
	DeployRunningNotApplicable = "not_applicable"
)

// envSwitchPatterns find the environment variable names a Go source reads as
// switches: any name passed literally to Getenv/LookupEnv, plus every
// VROOLI_* or *_ENFORCE* literal (staged gates are declared as constants and
// read elsewhere, as VROOLI_AUTH01_ENFORCE and VROOLI_DELIVERY_EFFORT_ENFORCE
// were). Names assembled at runtime from a prefix are not seen.
var envSwitchPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?:Getenv|LookupEnv)\(\s*"([A-Z][A-Z0-9_]*)"`),
	regexp.MustCompile(`"(VROOLI_[A-Z0-9_]+)"`),
	regexp.MustCompile(`"([A-Z][A-Z0-9_]*_ENFORCE[A-Z0-9_]*)"`),
}

// DeployReport answers "what would a restart deploy?" for one scenario: per
// build artifact, the source files that differ from the recorded build and the
// environment switches those sources add. It reads only; it never builds,
// stamps or restarts.
type DeployReport struct {
	Scenario  string                 `json:"scenario"`
	Changed   bool                   `json:"changed"`
	Artifacts []DeployReportArtifact `json:"artifacts"`
}

// DeployReportArtifact compares one artifact's recorded build with the
// current tree. Baseline is "recorded" when the build left a manifest, else
// "missing" and nothing can be compared.
type DeployReportArtifact struct {
	Component  string                 `json:"component"`
	CheckType  string                 `json:"check_type"`
	Target     string                 `json:"target"`
	Baseline   string                 `json:"baseline"`
	BuiltAt    *time.Time             `json:"built_at,omitempty"`
	Running    string                 `json:"running"`
	KeyChanges []DeployReportKeyInput `json:"key_changes,omitempty"`
	Packages   []DeployReportPackage  `json:"packages,omitempty"`
	Files      []DeployReportFile     `json:"files,omitempty"`
	// SwitchScan is true for an artifact built from Go sources, the only
	// sources scanned for env switches. SwitchesKnown is false when the build
	// predates switch recording; then SwitchesInChangedFiles lists candidates
	// instead of NewSwitches.
	SwitchScan             bool                 `json:"switch_scan"`
	SwitchesKnown          bool                 `json:"switches_known"`
	NewSwitches            []DeployReportSwitch `json:"new_switches,omitempty"`
	RemovedSwitches        []string             `json:"removed_switches,omitempty"`
	SwitchesInChangedFiles []DeployReportSwitch `json:"switches_in_changed_files,omitempty"`
}

// DeployReportFile is one repo-relative input that was added, modified or
// removed since the build.
type DeployReportFile struct {
	Path   string `json:"path"`
	Change string `json:"change"`
}

// DeployReportPackage groups changed files by directory. Shared is true for a
// directory outside the scenario (packages/*, internal/*, another scenario).
type DeployReportPackage struct {
	Dir      string `json:"dir"`
	Shared   bool   `json:"shared"`
	Added    int    `json:"added,omitempty"`
	Modified int    `json:"modified,omitempty"`
	Removed  int    `json:"removed,omitempty"`
}

// DeployReportKeyInput is a non-file build input (toolchain, build mode) whose
// value differs from the recorded build.
type DeployReportKeyInput struct {
	Name     string `json:"name"`
	Recorded string `json:"recorded"`
	Current  string `json:"current"`
}

// DeployReportSwitch names one environment switch and the files that read it.
type DeployReportSwitch struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
}

// deployRunningExecutables lists the executables of live processes, as
// /proc/<pid>/exe reads them. A seam so tests do not depend on the host.
var deployRunningExecutables = procRunningExecutables

// DeployReportByName resolves a scenario and reports what a restart would
// deploy relative to its recorded builds.
func (r *Runner) DeployReportByName(ctx context.Context, name, customPath string) (DeployReport, error) {
	if err := ctx.Err(); err != nil {
		return DeployReport{}, err
	}
	item, err := r.loadScenario(name, customPath)
	if err != nil {
		return DeployReport{}, err
	}
	return r.DeployReport(ctx, item)
}

// DeployReport compares every build artifact of item with the current sources.
func (r *Runner) DeployReport(ctx context.Context, item scenario.Scenario) (DeployReport, error) {
	report := DeployReport{Scenario: item.Slug}
	deps := r.hostProbeDeps()
	scenarioDir := relUnder(r.Root, item.SourcePath())
	executables, procOK := deployRunningExecutables()

	names := make([]string, 0, len(item.Manifest.Components))
	for name := range item.Manifest.Components {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return DeployReport{}, err
		}
		component := item.Manifest.Components[name]
		artifacts, err := componentFreshnessArtifactsContextWithName(ctx, item.SourcePath(), r.Root, item.Slug, name, component, deps)
		if err != nil {
			return DeployReport{}, fmt.Errorf("component %s inputs: %w", name, err)
		}
		for _, art := range artifacts {
			entry, err := r.deployReportArtifact(name, scenarioDir, art)
			if err != nil {
				return DeployReport{}, fmt.Errorf("component %s: %w", name, err)
			}
			entry.Running = DeployRunningNotApplicable
			if entry.SwitchScan {
				entry.Running = deployRunningState(art.ArtifactPath, executables, procOK)
			}
			if entry.Baseline != "recorded" || len(entry.Files) > 0 || len(entry.KeyChanges) > 0 || entry.Running == DeployRunningReplaced {
				report.Changed = true
			}
			report.Artifacts = append(report.Artifacts, entry)
		}
	}
	return report, nil
}

func (r *Runner) deployReportArtifact(component, scenarioDir string, art artifactFreshness) (DeployReportArtifact, error) {
	entry := DeployReportArtifact{Component: component, CheckType: art.CheckType, Target: art.Target, Baseline: "missing"}
	recorded, ok, err := cliutil.ReadFreshnessManifest(art.ManifestPath)
	if err != nil || !ok {
		return entry, nil
	}
	entry.Baseline = "recorded"
	if recorded.WrittenAtNS > 0 {
		builtAt := time.Unix(0, recorded.WrittenAtNS).UTC()
		entry.BuiltAt = &builtAt
	}
	current, err := cliutil.ComputeFreshnessManifest(art.Spec, art.CheckType, art.KeyInputs, 0)
	if err != nil {
		return entry, err
	}
	entry.KeyChanges = deployKeyInputChanges(recorded.KeyInputs, current.KeyInputs)

	walkRoot := art.Spec.ContextRoot
	if strings.TrimSpace(walkRoot) == "" {
		walkRoot = art.Spec.SourceRoot
	}
	repoPath := func(rel string) string {
		return relUnder(r.Root, filepath.Join(walkRoot, filepath.FromSlash(rel)))
	}
	// Overlapping directory inputs list a file more than once; compare the
	// file sets, not the entry lists.
	before := make(map[string]string, len(recorded.Files))
	for _, file := range recorded.Files {
		before[file.Rel] = file.Hash
	}
	now := make(map[string]string, len(current.Files))
	for _, file := range current.Files {
		now[file.Rel] = file.Hash
	}
	for rel, hash := range now {
		switch was, existed := before[rel]; {
		case !existed:
			entry.Files = append(entry.Files, DeployReportFile{Path: repoPath(rel), Change: "added"})
		case was != hash:
			entry.Files = append(entry.Files, DeployReportFile{Path: repoPath(rel), Change: "modified"})
		}
		if strings.HasSuffix(rel, ".go") {
			entry.SwitchScan = true
		}
	}
	for rel := range before {
		if _, still := now[rel]; !still {
			entry.Files = append(entry.Files, DeployReportFile{Path: repoPath(rel), Change: "removed"})
		}
	}
	sort.Slice(entry.Files, func(i, j int) bool { return entry.Files[i].Path < entry.Files[j].Path })
	entry.Packages = deployReportPackages(scenarioDir, entry.Files)

	if !entry.SwitchScan {
		return entry, nil
	}
	currentSwitches := envSwitchesByName(walkRoot, current.Files)
	if recorded.EnvSwitchesRecorded {
		entry.SwitchesKnown = true
		had := make(map[string]bool, len(recorded.EnvSwitches))
		for _, name := range recorded.EnvSwitches {
			had[name] = true
		}
		for _, name := range sortedSwitchNames(currentSwitches) {
			if !had[name] {
				entry.NewSwitches = append(entry.NewSwitches, deploySwitch(name, currentSwitches[name], repoPath))
			}
		}
		for _, name := range recorded.EnvSwitches {
			if _, still := currentSwitches[name]; !still {
				entry.RemovedSwitches = append(entry.RemovedSwitches, name)
			}
		}
		return entry, nil
	}
	changed := map[string]bool{}
	for rel, hash := range now {
		if was, existed := before[rel]; !existed || was != hash {
			changed[rel] = true
		}
	}
	for _, name := range sortedSwitchNames(currentSwitches) {
		var files []string
		for _, rel := range currentSwitches[name] {
			if changed[rel] {
				files = append(files, rel)
			}
		}
		if len(files) > 0 {
			entry.SwitchesInChangedFiles = append(entry.SwitchesInChangedFiles, deploySwitch(name, files, repoPath))
		}
	}
	return entry, nil
}

// recordEnvSwitches stores the switch names of a Go build in its manifest so a
// later deploy report can name the switches newer sources add.
func recordEnvSwitches(art artifactFreshness, manifest *cliutil.FreshnessManifest) {
	root := art.Spec.ContextRoot
	if strings.TrimSpace(root) == "" {
		root = art.Spec.SourceRoot
	}
	manifest.EnvSwitches = sortedSwitchNames(envSwitchesByName(root, manifest.Files))
	manifest.EnvSwitchesRecorded = true
}

// envSwitchesByName scans the Go files of a manifest file set and maps each
// switch name to the rel paths that mention it. Unreadable files are skipped:
// the report is advisory and must not fail a build stamp.
func envSwitchesByName(root string, files []cliutil.FileManifestEntry) map[string][]string {
	out := map[string][]string{}
	scanned := map[string]bool{}
	for _, file := range files {
		if !strings.HasSuffix(file.Rel, ".go") || scanned[file.Rel] {
			continue
		}
		scanned[file.Rel] = true
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Rel)))
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, pattern := range envSwitchPatterns {
			for _, match := range pattern.FindAllSubmatch(data, -1) {
				name := string(match[1])
				if !seen[name] {
					seen[name] = true
					out[name] = append(out[name], file.Rel)
				}
			}
		}
	}
	return out
}

func sortedSwitchNames(switches map[string][]string) []string {
	names := make([]string, 0, len(switches))
	for name := range switches {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func deploySwitch(name string, rels []string, repoPath func(string) string) DeployReportSwitch {
	files := make([]string, 0, len(rels))
	for _, rel := range rels {
		files = append(files, repoPath(rel))
	}
	slices.Sort(files)
	return DeployReportSwitch{Name: name, Files: files}
}

func deployKeyInputChanges(recorded, current map[string]string) []DeployReportKeyInput {
	names := map[string]bool{}
	for name := range recorded {
		names[name] = true
	}
	for name := range current {
		names[name] = true
	}
	var out []DeployReportKeyInput
	for name := range names {
		if recorded[name] != current[name] {
			out = append(out, DeployReportKeyInput{Name: name, Recorded: recorded[name], Current: current[name]})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func deployReportPackages(scenarioDir string, files []DeployReportFile) []DeployReportPackage {
	byDir := map[string]*DeployReportPackage{}
	for _, file := range files {
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(file.Path)))
		pkg, ok := byDir[dir]
		if !ok {
			pkg = &DeployReportPackage{Dir: dir, Shared: dir != scenarioDir && !strings.HasPrefix(dir, scenarioDir+"/")}
			byDir[dir] = pkg
		}
		switch file.Change {
		case "added":
			pkg.Added++
		case "removed":
			pkg.Removed++
		default:
			pkg.Modified++
		}
	}
	out := make([]DeployReportPackage, 0, len(byDir))
	for _, pkg := range byDir {
		out = append(out, *pkg)
	}
	// Scenario-owned directories first, then shared ones: the shared changes
	// are the ones another session most likely left in the tree.
	sort.Slice(out, func(i, j int) bool {
		if out[i].Shared != out[j].Shared {
			return !out[i].Shared
		}
		return out[i].Dir < out[j].Dir
	})
	return out
}

// deployRunningState says whether a live process runs the artifact on disk.
// Linux marks the exe link of a process whose binary was replaced (unlinked or
// renamed over) with " (deleted)": that process runs an older build than the
// recorded manifest describes.
func deployRunningState(artifactPath string, executables []string, procOK bool) string {
	if !procOK {
		return DeployRunningUnknown
	}
	artifactPath = filepath.Clean(artifactPath)
	state := DeployRunningNone
	for _, exe := range executables {
		switch exe {
		case artifactPath + " (deleted)":
			return DeployRunningReplaced
		case artifactPath:
			state = DeployRunningCurrent
		}
	}
	return state
}

func procRunningExecutables() ([]string, bool) {
	if runtime.GOOS != "linux" {
		return nil, false
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, false
	}
	var out []string
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		if exe, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe")); err == nil {
			out = append(out, exe)
		}
	}
	return out, true
}
