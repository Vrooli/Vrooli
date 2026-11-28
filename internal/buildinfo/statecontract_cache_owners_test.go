package buildinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/statecontract"
)

func TestCacheAndMarkerFormatChangesAffectOwner(t *testing.T) {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ owner, path, before, after string }{
		{"host-facts-cache", "internal/hostinventory/facts_cache.go", `json:"boot_id"`, `json:"boot_identity"`},
		{"host-facts-cache", "internal/hostinventory/workloads.go", `Kind: "service-unit"`, `Kind: "service"`},
		{"host-facts-cache", "internal/workloadowner/enumeration.go", `Kind: "service-unit"`, `Kind: "service"`},
		{"template-validation-markers", "internal/templatevalidation/marker.go", `json:"runId"`, `json:"run_id"`},
	} {
		t.Run(tc.owner, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, tc.path))
			if err != nil {
				t.Fatal(err)
			}
			fixture := t.TempDir()
			if tc.owner == "host-facts-cache" {
				writeTestFile(t, fixture, "internal/workloadowner/fixture.go", "package workloadowner\n")
				writeTestFile(t, fixture, "internal/hostinventory/fixture.go", "package hostinventory\n")
			}
			writeTestFile(t, fixture, tc.path, string(data))
			digest := func() string {
				t.Helper()
				d, err := wholeSourceDigest(fixture, statecontract.WholeSources[tc.owner])
				if err != nil {
					t.Fatal(err)
				}
				return d
			}
			before := digest()
			changed := strings.Replace(string(data), tc.before, tc.after, 1)
			if changed == string(data) {
				t.Fatal("format fixture missing")
			}
			writeTestFile(t, fixture, tc.path, changed)
			if digest() == before {
				t.Fatal("persisted format change escaped owner")
			}
		})
	}
}
