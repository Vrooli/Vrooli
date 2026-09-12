package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// requirementsModule mirrors the fields of requirements/*/module.json that
// the consistency guard inspects.
type requirementsModule struct {
	Requirements []struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		Validation []any  `json:"validation"`
	} `json:"requirements"`
}

// requirementsDir resolves THIS scenario's requirements directory by
// anchoring on the test file's own location (api/requirements_registry_test.go
// → ../requirements). It deliberately does NOT consult VROOLI_SCENARIO_DIR,
// which in a shared dev shell can point at a different scenario.
func requirementsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed; cannot locate requirements dir")
	}
	dir := filepath.Join(filepath.Dir(thisFile), "..", "requirements")
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("requirements dir not found at %s: %v", dir, err)
	}
	return dir
}

// TestRequirementsRegistry_NoDoneWithoutValidation enforces the registry
// honesty contract documented in requirements/README.md: a requirement
// marked status:"done" must carry concrete validation evidence, and an
// unfinished requirement must not be marked done. This guards against the
// historical drift where every requirement read "planned" with an empty
// validation array despite shipped code, and against the opposite failure
// (claiming done with no evidence).
func TestRequirementsRegistry_NoDoneWithoutValidation(t *testing.T) {
	dir := requirementsDir(t)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read requirements dir: %v", err)
	}

	checked := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name(), "module.json")
		raw, err := os.ReadFile(path)
		if err != nil {
			// Not every subdir is a module (none expected, but tolerate).
			continue
		}
		var mod requirementsModule
		if err := json.Unmarshal(raw, &mod); err != nil {
			t.Errorf("%s: invalid JSON: %v", path, err)
			continue
		}
		for _, r := range mod.Requirements {
			checked++
			switch r.Status {
			case "done":
				if len(r.Validation) == 0 {
					t.Errorf("%s: requirement %s is status:done but has empty validation[]", path, r.ID)
				}
			case "planned", "in_progress", "deferred":
				// Allowed unfinished states.
			default:
				t.Errorf("%s: requirement %s has unknown status %q", path, r.ID, r.Status)
			}
		}
	}
	if checked == 0 {
		t.Fatalf("no requirements parsed from %s", dir)
	}
}
