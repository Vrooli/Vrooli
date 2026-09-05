package manifestvalidation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// fixtureCase pairs a fixture manifest with a proto surface and the finding
// codes the validator must emit. Severity rules are checked separately
// (Passed is derived from error count).
type fixtureCase struct {
	name         string
	manifestFile string
	surface      ProtoSurface
	expectCodes  []string
	expectPass   bool
}

func TestFixtures(t *testing.T) {
	cases := []fixtureCase{
		{
			name:         "orphan_method_fails_with_proto_orphan_finding",
			manifestFile: "orphan_method.manifest.json",
			surface: ProtoSurface{Services: []ProtoService{
				{Name: "Svc", Methods: []string{"Do", "Extra"}},
			}},
			expectCodes: []string{CodeProtoOrphanMethod},
			expectPass:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("testdata", tc.manifestFile))
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			svc := New(Deps{
				Manifests: stubLoader{raw: raw, path: tc.manifestFile},
				Schema:    stubSchema{},
				Protos:    stubProto{surface: tc.surface},
			})
			r, err := svc.ValidateScenario(context.Background(), "fixture")
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			if r.Passed != tc.expectPass {
				t.Fatalf("Passed=%v want %v; findings=%+v", r.Passed, tc.expectPass, r.Findings)
			}
			for _, code := range tc.expectCodes {
				if !findingHasCode(r.Findings, code) {
					t.Fatalf("missing expected code %q; findings=%+v", code, r.Findings)
				}
			}
		})
	}
}
