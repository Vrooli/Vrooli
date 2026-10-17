package requirements

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidationEntriesHaveEvidence prevents an implemented requirement from
// silently becoming an untraceable claim. Planned entries still carry their
// intended evidence target so the gap is visible to plan validation.
func TestValidationEntriesHaveEvidence(t *testing.T) {
	root := filepath.Dir(filepath.Dir(os.Args[0]))
	_ = root // the test is also consumed by repository-level requirement tooling.
	files, err := filepath.Glob(filepath.Join("*", "module.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no requirement modules found")
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var module struct {
			Requirements []struct {
				ID         string `json:"id"`
				Validation []struct {
					Status string `json:"status"`
					Ref    string `json:"ref"`
					Notes  string `json:"notes"`
				} `json:"validation"`
			} `json:"requirements"`
		}
		if err := json.Unmarshal(data, &module); err != nil {
			t.Fatalf("decode %s: %v", file, err)
		}
		for _, requirement := range module.Requirements {
			for _, validation := range requirement.Validation {
				if strings.TrimSpace(validation.Ref) == "" {
					t.Errorf("%s/%s has no evidence ref", file, requirement.ID)
				}
				if strings.Contains(strings.ToLower(validation.Notes), "planned target") {
					t.Errorf("%s/%s retains legacy planned-target wording", file, requirement.ID)
				}
			}
		}
	}
}
