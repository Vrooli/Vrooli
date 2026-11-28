package dependencyhealth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidationBudgetUsesProviderDescriptorAndRejectsInvalidBudgets(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenario-dependency-analyzer", ".vrooli", "test-genie.json")
	if _, err := loadValidationBudget(root); err == nil {
		t.Fatal("missing descriptor accepted")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body string
		want time.Duration
	}{{`{"timeout":"250ms"}`, 250 * time.Millisecond}, {`{"timeout":"900s"}`, 15 * time.Minute}, {`{"timeout":"0s"}`, 0}, {`{"timeout":"-1s"}`, 0}, {`{"timeout":"wrong"}`, 0}, {`{}`, 0}, {`{`, 0}} {
		if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := loadValidationBudget(root)
		if tc.want == 0 {
			if err == nil {
				t.Fatalf("invalid budget accepted: %s", tc.body)
			}
		} else if err != nil || got != tc.want {
			t.Fatalf("descriptor budget=%v err=%v want=%v", got, err, tc.want)
		}
	}
}
