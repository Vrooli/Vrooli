package hygiene

import (
	"os"
	"path/filepath"
	"testing"
)

// protoRepo builds a repo skeleton: a proto surface per name in surfaces, an
// owning directory per entry in owners ("scenarios/demo"), and source files per
// entry in sources (path -> contents).
func protoRepo(t *testing.T, surfaces, owners []string, sources map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range surfaces {
		dir := filepath.Join(root, "packages", "proto", "schemas", name, "v1", "health")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "health.proto"), []byte("syntax = \"proto3\";\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, owner := range owners {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(owner)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for rel, body := range sources {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestOrphanedProtoFlagsSurfaceWithNoOwnerAndNoConsumer(t *testing.T) {
	root := protoRepo(t,
		[]string{"live-scenario", "throwaway-probe"},
		[]string{"scenarios/live-scenario"},
		nil,
	)

	report := Report{Root: root, Success: true}
	Service{}.checkOrphanedProtoSurfaces(&report)

	finding, ok := findingCodes(report)["orphaned_proto_surface"]
	if !ok {
		t.Fatalf("expected orphaned_proto_surface finding, got %v", keys(findingCodes(report)))
	}
	if !containsString(finding.Locations, "packages/proto/schemas/throwaway-probe") {
		t.Fatalf("expected the orphan's schemas path, got %v", finding.Locations)
	}
	for _, notWanted := range []string{
		"packages/proto/schemas/live-scenario",
		"packages/proto/gen/go/live-scenario",
	} {
		if containsString(finding.Locations, notWanted) {
			t.Fatalf("owned surface %s must not be reported, got %v", notWanted, finding.Locations)
		}
	}
}

// The cross-cutting contracts (common, scenario-validation, ...) have no owning
// directory by design. They must be recognised via their consumers, not an
// allowlist, so a new shared contract needs no registration.
func TestOrphanedProtoAcceptsSharedContractWithConsumers(t *testing.T) {
	root := protoRepo(t,
		[]string{"scenario-validation"},
		nil,
		map[string]string{
			"scenarios/demo/api/main.go": `package main
import v1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/validation"
var _ = v1.Nothing
`,
		},
	)

	report := Report{Root: root, Success: true}
	Service{}.checkOrphanedProtoSurfaces(&report)

	if f, ok := findingCodes(report)["orphaned_proto_surface"]; ok {
		t.Fatalf("a consumed shared contract must not be reported, got %v", f.Locations)
	}
}

// A template owns its surface from templates/scenarios/, not scenarios/.
func TestOrphanedProtoAcceptsTemplateOwnedSurface(t *testing.T) {
	root := protoRepo(t,
		[]string{"landing-page-react-vite"},
		[]string{"templates/scenarios/landing-page-react-vite"},
		nil,
	)

	report := Report{Root: root, Success: true}
	Service{}.checkOrphanedProtoSurfaces(&report)

	if f, ok := findingCodes(report)["orphaned_proto_surface"]; ok {
		t.Fatalf("a template-owned surface must not be reported, got %v", f.Locations)
	}
}

// Generated output references its own surface; if that counted as consumption
// every surface would look alive and the check would never fire.
func TestOrphanedProtoIgnoresSelfReferenceFromGeneratedOutput(t *testing.T) {
	root := protoRepo(t, []string{"throwaway-probe"}, nil, map[string]string{
		"packages/proto/gen/go/throwaway-probe/v1/health/health.pb.go": `package health
// import path packages/proto/gen/go/throwaway-probe/v1/health
`,
	})

	report := Report{Root: root, Success: true}
	Service{}.checkOrphanedProtoSurfaces(&report)

	if _, ok := findingCodes(report)["orphaned_proto_surface"]; !ok {
		t.Fatal("generated self-reference must not rescue an orphan")
	}
}

// The reported footprint must cover all six outputs, including the python
// underscore form and the lock manifest, or a follow-up cleanup misses residue.
func TestOrphanedProtoReportsFullFootprint(t *testing.T) {
	root := protoRepo(t, []string{"throwaway-probe"}, nil, nil)

	report := Report{Root: root, Success: true}
	Service{}.checkOrphanedProtoSurfaces(&report)

	finding := findingCodes(report)["orphaned_proto_surface"]
	for _, want := range []string{
		"packages/proto/schemas/throwaway-probe",
		"packages/proto/gen/go/throwaway-probe",
		"packages/proto/gen/typescript/throwaway-probe",
		"packages/proto/gen/typescript/js/throwaway-probe",
		"packages/proto/gen/python/throwaway_probe",
		"packages/proto/gen/manifests/throwaway-probe.lock.json",
	} {
		if !containsString(finding.Locations, want) {
			t.Errorf("footprint missing %q\ngot: %v", want, finding.Locations)
		}
	}
}

func TestOrphanedProtoPassesWhenEverySurfaceIsOwned(t *testing.T) {
	root := protoRepo(t,
		[]string{"alpha", "beta"},
		[]string{"scenarios/alpha", "packages/beta"},
		nil,
	)

	report := Report{Root: root, Success: true}
	Service{}.checkOrphanedProtoSurfaces(&report)

	if _, ok := findingCodes(report)["orphaned_proto_surface"]; ok {
		t.Fatal("fully owned proto tree must not report an orphan")
	}
	var saw bool
	for _, c := range report.Checks {
		if c.Name == "orphaned_proto_surfaces" {
			saw, _ = true, c
			if !c.Passed {
				t.Fatalf("check failed on a clean tree: %s", c.Message)
			}
		}
	}
	if !saw {
		t.Fatalf("expected an orphaned_proto_surfaces check, got %+v", report.Checks)
	}
}
