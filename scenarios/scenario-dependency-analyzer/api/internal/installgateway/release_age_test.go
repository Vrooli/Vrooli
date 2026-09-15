package installgateway

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseAgePreservesStricterPolicyAndWorkspace(t *testing.T) {
	for _, policy := range []string{"", "minimumReleaseAge: 60\n", "minimumReleaseAge: 20000\n"} {
		t.Run(policy, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "pnpm-workspace.yaml")
			before := "packages:\n  - .\n# reviewed package exception\nminimumReleaseAgeExclude:\n  - example\n" + policy
			if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := ensureReleaseAge(root); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			want := "minimumReleaseAge: 10080"
			if strings.Contains(policy, "20000") {
				want = "minimumReleaseAge: 20000"
			}
			for _, part := range []string{want, "- .", "# reviewed package exception", "- example"} {
				if !strings.Contains(string(raw), part) {
					t.Errorf("lost %q in %s", part, raw)
				}
			}
			if err := ensureReleaseAge(root); err != nil {
				t.Fatal(err)
			}
			after, _ := os.ReadFile(path)
			if string(after) != string(raw) {
				t.Fatal("policy is not idempotent")
			}
		})
	}
}
