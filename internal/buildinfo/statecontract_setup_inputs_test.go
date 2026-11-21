package buildinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestSetupRecoveryAndHandoffInputsArePinned(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, before, after string }{
		{"internal/setup/migration.go", `record.Cursors[class] = result.LastPath`, `record.Cursors[class] = "changed"`},
		{"internal/config/repair.go", `req.ResumeAfter = filepath.Clean(req.ResumeAfter)`, `req.ResumeAfter = filepath.Clean(req.ResumeAfter + "/next")`},
		{"internal/onboardinghandoff/handoff.go", `Decision{Action: "url"`, `Decision{Action: "url-v2"`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			fixture := t.TempDir()
			for _, pattern := range statecontract.WholeSources["setup-state"] {
				path := pattern + "/fixture.go"
				if strings.HasSuffix(pattern, "*") {
					path = strings.TrimSuffix(pattern, "*") + "_fixture.go"
				}
				writeTestFile(t, fixture, path, "package fixture\n")
			}
			data, err := os.ReadFile(filepath.Join(root, tc.path))
			if err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, fixture, tc.path, string(data))
			digest := func() string {
				t.Helper()
				v, err := wholeSourceDigest(fixture, statecontract.WholeSources["setup-state"])
				if err != nil {
					t.Fatal(err)
				}
				return v
			}
			before := digest()
			changed := strings.Replace(string(data), tc.before, tc.after, 1)
			if changed == string(data) {
				t.Fatal("mutation anchor missing")
			}
			writeTestFile(t, fixture, tc.path, changed)
			if digest() == before {
				t.Fatal("stored recovery or handoff semantics escaped setup-state")
			}
		})
	}
}
