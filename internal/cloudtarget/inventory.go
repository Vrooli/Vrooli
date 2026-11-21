package cloudtarget

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MutableDirectoryNames is the legacy name heuristic scenario-to-cloud used to
// decide which directories survive an update. It is a migration aid only:
// the inventory reports what it would have matched so an operator can map
// each path to a declared binding before the heuristic is retired.
var MutableDirectoryNames = []string{"data", "uploads", "storage", "state", "cache", "logs", "runtime", "tmp", "files"}

// inventoryMaxDepth mirrors the `find -maxdepth 5` the heuristic used.
const inventoryMaxDepth = 5

// Binding is a declared persistent-data binding. Path is relative to the
// scenario directory (forward slashes).
type Binding struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

// InventoryRequest names the deployment workdir and scenario to inventory.
type InventoryRequest struct {
	DeploymentID string
	Workdir      string
	Scenario     string
	Bindings     []Binding
}

// InventoryEntry is one heuristic-matched mutable directory.
type InventoryEntry struct {
	Path      string `json:"path"`
	Bytes     int64  `json:"bytes"`
	Files     int64  `json:"files"`
	Covered   bool   `json:"covered"`
	BindingID string `json:"binding_id,omitempty"`
}

// InventoryReport is the migration aid output. No field is mutated on disk.
type InventoryReport struct {
	DeploymentID   string           `json:"deployment_id"`
	Scenario       string           `json:"scenario"`
	ScenarioDir    string           `json:"scenario_dir"`
	Heuristic      []string         `json:"heuristic_names"`
	Bindings       []Binding        `json:"bindings"`
	Entries        []InventoryEntry `json:"entries"`
	UncoveredCount int              `json:"uncovered_count"`
}

// ParseBindings decodes the --bindings JSON: an array of {id, path}.
func ParseBindings(raw string) ([]Binding, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var bindings []Binding
	if err := json.Unmarshal([]byte(raw), &bindings); err != nil {
		return nil, refuse(CodeInvalidArgument, "parse bindings: %v", err)
	}
	for i := range bindings {
		path := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(bindings[i].Path))))
		if path == "" || path == "." || filepath.IsAbs(path) || path == ".." || strings.HasPrefix(path, "../") {
			return nil, refuse(CodeInvalidArgument, "binding %q must be a relative path inside the scenario directory", bindings[i].Path)
		}
		bindings[i].Path = path
		if strings.TrimSpace(bindings[i].ID) == "" {
			bindings[i].ID = path
		}
	}
	return bindings, nil
}

// Inventory lists the heuristic-matched mutable directories under one
// scenario and whether a declared binding covers each. Coverage uses the same
// rule as the legacy cleanup: an entry is covered when it equals a binding
// path, lies beneath one, or contains one.
func Inventory(req InventoryRequest) (InventoryReport, error) {
	if err := validIdentifier("deployment id", req.DeploymentID); err != nil {
		return InventoryReport{}, err
	}
	if err := validIdentifier("scenario id", req.Scenario); err != nil {
		return InventoryReport{}, err
	}
	workdir := filepath.Clean(strings.TrimSpace(req.Workdir))
	if workdir == "" || !filepath.IsAbs(workdir) {
		return InventoryReport{}, refuse(CodeInvalidArgument, "workdir must be an absolute path")
	}
	scenarioDir := filepath.Join(workdir, "scenarios", req.Scenario)
	report := InventoryReport{DeploymentID: req.DeploymentID, Scenario: req.Scenario, ScenarioDir: scenarioDir, Heuristic: MutableDirectoryNames, Bindings: req.Bindings, Entries: []InventoryEntry{}}
	if report.Bindings == nil {
		report.Bindings = []Binding{}
	}
	names := map[string]struct{}{}
	for _, name := range MutableDirectoryNames {
		names[name] = struct{}{}
	}
	var matches []string
	err := filepath.WalkDir(scenarioDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if path == scenarioDir {
				return walkErr
			}
			return nil
		}
		if path == scenarioDir || !entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(scenarioDir, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(rel, string(os.PathSeparator)) + 1
		if depth > inventoryMaxDepth {
			return fs.SkipDir
		}
		if _, ok := names[entry.Name()]; ok {
			matches = append(matches, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return InventoryReport{}, fail(CodeStoreIO, "walk scenario directory: %v", err)
	}
	sort.Strings(matches)
	for _, rel := range matches {
		bytes, files := measure(filepath.Join(scenarioDir, filepath.FromSlash(rel)))
		entry := InventoryEntry{Path: rel, Bytes: bytes, Files: files}
		for _, binding := range req.Bindings {
			if rel == binding.Path || strings.HasPrefix(rel, binding.Path+"/") || strings.HasPrefix(binding.Path, rel+"/") {
				entry.Covered = true
				entry.BindingID = binding.ID
				break
			}
		}
		if !entry.Covered {
			report.UncoveredCount++
		}
		report.Entries = append(report.Entries, entry)
	}
	return report, nil
}

// measure sums regular-file sizes and counts beneath dir without following
// symlinks, matching what a preservation tar would carry.
func measure(dir string) (int64, int64) {
	var bytes, files int64
	_ = filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		files++
		bytes += info.Size()
		return nil
	})
	return bytes, files
}
