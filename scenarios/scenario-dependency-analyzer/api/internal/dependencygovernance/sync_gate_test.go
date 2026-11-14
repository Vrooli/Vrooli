package dependencygovernance

import (
	"strings"
	"testing"
)

func TestGateRequirementSyncAppliesInstallGateToDirectAndRecordedDecisionsToIndirect(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"records": [
			{"ecosystem":"go","package_name":"golang.org/x/mod","version_range":"v0.11.0","range_policy":"major_line","state":"approved","rationale":"Module parsing."},
			{"ecosystem":"go","package_name":"example.com/api-only","version_range":"*","state":"approved","rationale":"API surfaces only.","allowed_surfaces":["api"]},
			{"ecosystem":"go","package_name":"example.com/denied","version_range":"*","state":"denied","rationale":"Denied.","replacement":"Use example.com/maintained."},
			{"ecosystem":"go","package_name":"example.com/vulnerable","version_range":">=1.0.0 <1.2.0","range_policy":"security_denied","state":"denied","rationale":"CVE.","replacement":"Upgrade to >=1.2.0."}
		]
	}`)
	registry := NewRegistry(repoRoot)

	cases := []struct {
		name        string
		surface     string
		requirement SyncRequirement
		wantVerdict string // "" means allowed
	}{
		{"indirect approved", "api", SyncRequirement{Path: "golang.org/x/mod", Version: "v0.41.0"}, ""},
		{"indirect above approved range", "api", SyncRequirement{Path: "golang.org/x/mod", Version: "v1.0.0"}, "out_of_range"},
		{"indirect unrecorded is transitive", "api", SyncRequirement{Path: "example.com/unrecorded", Version: "v1.0.0"}, ""},
		{"indirect outside surface scope is transitive", "cli", SyncRequirement{Path: "example.com/api-only", Version: "v1.0.0"}, ""},
		{"indirect denied", "api", SyncRequirement{Path: "example.com/denied", Version: "v1.0.0"}, "denied"},
		{"indirect inside security-denied range", "api", SyncRequirement{Path: "example.com/vulnerable", Version: "v1.1.0"}, "denied"},
		{"indirect outside security-denied range", "api", SyncRequirement{Path: "example.com/vulnerable", Version: "v1.2.0"}, ""},
		{"direct approved", "api", SyncRequirement{Path: "golang.org/x/mod", Version: "v0.41.0", Direct: true}, ""},
		{"direct unrecorded", "api", SyncRequirement{Path: "example.com/unrecorded", Version: "v1.0.0", Direct: true}, "unrecorded"},
		{"direct outside surface scope", "cli", SyncRequirement{Path: "example.com/api-only", Version: "v1.0.0", Direct: true}, "surface_not_allowed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refusals, err := registry.GateRequirementSync("demo", tc.surface, []SyncRequirement{tc.requirement})
			if err != nil {
				t.Fatal(err)
			}
			if tc.wantVerdict == "" {
				if len(refusals) != 0 {
					t.Fatalf("refusals = %#v, want none", refusals)
				}
				return
			}
			if len(refusals) != 1 || refusals[0].Verdict != tc.wantVerdict {
				t.Fatalf("refusals = %#v, want one %q", refusals, tc.wantVerdict)
			}
			if len(refusals[0].NextSteps) == 0 {
				t.Fatalf("a refusal must name its next step: %#v", refusals[0])
			}
		})
	}
}

func TestGateRequirementSyncRefusalNamesApprovalCommand(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{"records": []}`)
	refusals, err := NewRegistry(repoRoot).GateRequirementSync("demo", "api", []SyncRequirement{{Path: "example.com/new", Version: "v1.0.0", Direct: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(refusals) != 1 || !strings.Contains(refusals[0].String(), "deps approved approve-observed go/example.com/new") {
		t.Fatalf("refusal must carry the exact approval command: %#v", refusals)
	}
}
