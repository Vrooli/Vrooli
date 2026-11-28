package facts

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

func TestResolveTargetKindsUseDeclaredRoots(t *testing.T) {
	repo := t.TempDir()
	for _, dir := range []string{
		filepath.Join(repo, "scenarios", "demo"),
		filepath.Join(repo, "packages", "shared"),
		filepath.Join(repo, "cmd", "vrooli"),
		filepath.Join(repo, "internal"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	servicePath := filepath.Join(repo, "scenarios", "demo", ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(servicePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(servicePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		target *factsv1.CodeTarget
		kind   factsv1.TargetKind
		root   string
	}{
		{"project", &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PROJECT, RepoRoot: repo}, factsv1.TargetKind_TARGET_KIND_PROJECT, repo},
		{"repo", &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_REPO, RepoRoot: repo}, factsv1.TargetKind_TARGET_KIND_REPO, repo},
		{"package", &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_PACKAGE, PackageName: "shared", RepoRoot: repo}, factsv1.TargetKind_TARGET_KIND_PACKAGE, filepath.Join(repo, "packages", "shared")},
		{"control-plane", &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE, RepoRoot: repo}, factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE, repo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := resolveTarget(tc.target)
			if err != nil {
				t.Fatal(err)
			}
			if resolved.GetResolvedKind() != tc.kind || resolved.GetRootPath() != tc.root {
				t.Fatalf("resolved target = %#v, want kind=%s root=%s", resolved, tc.kind, tc.root)
			}
			if resolved.GetRequested().GetKind() != tc.kind {
				t.Fatalf("requested kind = %s, want %s", resolved.GetRequested().GetKind(), tc.kind)
			}
		})
	}
	control, err := resolveTarget(cases[3].target)
	if err != nil || len(control.GetRootPaths()) != 2 {
		t.Fatalf("control-plane roots = %v, err=%v; want cmd/vrooli and internal", control.GetRootPaths(), err)
	}
}

func TestResolveScenarioHonorsExplicitPhysicalPath(t *testing.T) {
	repo := t.TempDir()
	canonical := filepath.Join(repo, "scenarios", "demo")
	copyRoot := filepath.Join(t.TempDir(), "demo")
	for _, dir := range []string{canonical, copyRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	resolved, err := resolveTarget(&factsv1.CodeTarget{
		Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo", RepoRoot: repo, Path: copyRoot,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.GetRootPath() != copyRoot || resolved.GetScenario() != "demo" {
		t.Fatalf("resolved root=%q scenario=%q; want %q and demo", resolved.GetRootPath(), resolved.GetScenario(), copyRoot)
	}
}

func TestDescribeScenarioUsesSurfacesFromExplicitPhysicalPath(t *testing.T) {
	repo := t.TempDir()
	canonical := filepath.Join(repo, "scenarios", "demo")
	copyRoot := filepath.Join(t.TempDir(), "demo")
	for _, dir := range []string{filepath.Join(canonical, "ui"), filepath.Join(copyRoot, "api")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	report, err := NewService().Describe(context.Background(), &factsv1.DescribeCodeFactsRequest{
		Target:  &factsv1.CodeTarget{Kind: factsv1.TargetKind_TARGET_KIND_SCENARIO, Scenario: "demo", RepoRoot: repo, Path: copyRoot},
		Include: []factsv1.FactFamily{factsv1.FactFamily_FACT_FAMILY_SURFACES},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.GetTarget().GetRootPath() != copyRoot {
		t.Fatalf("target root = %q, want %q", report.GetTarget().GetRootPath(), copyRoot)
	}
	requireSurface(t, report.GetSurfaces(), "api", factsv1.SurfaceStatus_SURFACE_STATUS_KNOWN)
	requireSurface(t, report.GetSurfaces(), "ui", factsv1.SurfaceStatus_SURFACE_STATUS_MISSING)
}

func TestValidateTargetRejectsDeferredKindsByName(t *testing.T) {
	deferred := []factsv1.TargetKind{
		factsv1.TargetKind_TARGET_KIND_MODULE,
		factsv1.TargetKind_TARGET_KIND_RESOURCE,
		factsv1.TargetKind_TARGET_KIND_TOOL,
		factsv1.TargetKind_TARGET_KIND_SAFEGUARD,
		factsv1.TargetKind_TARGET_KIND_DOCS,
		factsv1.TargetKind_TARGET_KIND_TEAM,
	}
	for _, kind := range deferred {
		t.Run(kind.String(), func(t *testing.T) {
			err := validateTarget(&factsv1.CodeTarget{Kind: kind, Path: "/tmp/ignored"})
			if err == nil || !strings.Contains(err.Error(), kind.String()) {
				t.Fatalf("error = %v, want typed unsupported error naming %s", err, kind)
			}
		})
	}
}
