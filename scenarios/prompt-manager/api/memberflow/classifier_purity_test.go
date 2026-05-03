package memberflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestClassifierPurity_RegisteredClassifiers_AreMemberAgnostic enforces the
// `non_portable_classifier` rule against every registered skill whose id
// suggests a classifier role. This is belt-and-suspenders with the runtime
// validation rule so the rule cannot quietly regress while a real
// classifier skill leaks team coupling.
func TestClassifierPurity_RegisteredClassifiers_AreMemberAgnostic(t *testing.T) {
	storeDir := "/home/matthalloran8/Vrooli/scenarios/prompt-manager/store"
	if _, err := os.Stat(storeDir); err != nil {
		t.Skip("real store not available in this environment")
	}
	paths, err := LoadSkillPaths(storeDir)
	if err != nil {
		t.Fatalf("LoadSkillPaths: %v", err)
	}
	suffixes := []string{"-classifier", "-triage"}
	checked := 0
	for id, mdPath := range paths {
		match := false
		for _, suffix := range suffixes {
			if strings.HasSuffix(id, suffix) {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		checked++
		data, err := os.ReadFile(mdPath)
		if err != nil {
			t.Errorf("read %s: %v", mdPath, err)
			continue
		}
		body := string(data)
		for _, sub := range classifierForbiddenSubstrings {
			if strings.Contains(body, sub) {
				rel, _ := filepath.Rel(storeDir, mdPath)
				t.Errorf("classifier-or-triage skill %q (%s) contains forbidden pattern %q — judgment skills must be member-agnostic", id, rel, sub)
				break
			}
		}
	}
	if checked == 0 {
		t.Errorf("no -classifier or -triage skills found in registry; expected at least the three migrated by the inbox-flow refactor")
	}
}
