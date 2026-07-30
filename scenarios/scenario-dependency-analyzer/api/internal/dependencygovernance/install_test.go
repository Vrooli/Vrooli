package dependencygovernance

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	"scenario-dependency-analyzer/internal/installgateway"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestGovernInstallVerdicts(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"records": [
			{"ecosystem":"npm","package_name":"react-hook-form","version_range":">=7.0.0 <8.0.0","state":"approved","rationale":"Approved forms lib.","allowed_surfaces":["ui"]},
			{"ecosystem":"npm","package_name":"react-hook-form-x","version_range":"^4.0.0","state":"approved","rationale":"Approved successor."},
			{"ecosystem":"npm","package_name":"left-pad","version_range":"*","state":"denied","rationale":"Denied.","replacement":"Use native padding."}
		]
	}`)
	registry := NewRegistry(repoRoot)

	cases := []struct {
		name                             string
		ecosystem, pkg, surface, version string
		wantVerdict                      string
		wantBlocked                      bool
	}{
		{"approved-in-range", "npm", "react-hook-form", "ui", "7.2.0", "approved", false},
		{"out-of-range", "npm", "react-hook-form", "ui", "8.1.0", "out_of_range", true},
		{"surface-not-allowed", "npm", "react-hook-form", "api", "7.2.0", "surface_not_allowed", true},
		{"denied", "npm", "left-pad", "ui", "1.0.0", "denied", true},
		{"approved-npm-alias", "npm", "react-hook-form", "ui", "npm:react-hook-form-x@^4.0.0", "approved", false},
		{"unrecorded", "npm", "totally-new-pkg", "ui", "1.0.0", "unrecorded", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &governancev1.InstallDependencyRequest{
				Ecosystem:   tc.ecosystem,
				PackageName: tc.pkg,
				Surface:     tc.surface,
				Version:     tc.version,
			}
			verdict, blocked, _, _ := registry.governInstall(msg, installgateway.Resolution{})
			if verdict != tc.wantVerdict || blocked != tc.wantBlocked {
				t.Fatalf("verdict=%q blocked=%v, want verdict=%q blocked=%v", verdict, blocked, tc.wantVerdict, tc.wantBlocked)
			}
		})
	}
}

func TestNpmAliasTarget(t *testing.T) {
	pkg, version, ok := npmAliasTarget("npm:@scope/plugin@^4.0.0")
	if !ok || pkg != "@scope/plugin" || version != "^4.0.0" {
		t.Fatalf("npmAliasTarget() = %q, %q, %v", pkg, version, ok)
	}
	if _, _, ok := npmAliasTarget("not-an-alias"); ok {
		t.Fatal("non-alias must not parse as npm alias")
	}
}

// recordingInstaller is a fake PackageInstaller that records whether Install ran.
type recordingInstaller struct {
	called bool
	plan   installgateway.Resolution
}

func (r *recordingInstaller) Install(_ context.Context, res installgateway.Resolution) (string, error) {
	r.called = true
	r.plan = res
	return "ok", nil
}

func TestInstallDependencyDryRunDoesNotInstall(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"records": [
			{"ecosystem":"npm","package_name":"react-hook-form","version_range":">=7.0.0 <8.0.0","state":"approved","rationale":"Approved.","allowed_surfaces":["ui"]}
		]
	}`)
	uiDir := filepath.Join(repoRoot, "scenarios", "demo", "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uiDir, "pnpm-lock.yaml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	installer := &recordingInstaller{}
	h := &connectHandler{
		scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") },
		installer:    installer,
	}

	// Dry run (apply=false): the command is returned but the installer never runs.
	resp, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageName: "react-hook-form", Version: "7.2.0",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if installer.called {
		t.Fatal("dry run must not invoke the installer")
	}
	if !resp.Msg.GetDryRun() || resp.Msg.GetInstalled() {
		t.Fatalf("dry run resp = %#v", resp.Msg)
	}
	if resp.Msg.GetCommand() != "pnpm add react-hook-form@7.2.0" {
		t.Fatalf("command = %q", resp.Msg.GetCommand())
	}

	// Apply: the installer runs.
	resp, err = h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageName: "react-hook-form", Version: "7.2.0", Apply: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !installer.called || !resp.Msg.GetInstalled() {
		t.Fatalf("apply should install: called=%v resp=%#v", installer.called, resp.Msg)
	}
	if installer.plan.SurfaceRoot != uiDir {
		t.Fatalf("installer surface root = %q, want %q", installer.plan.SurfaceRoot, uiDir)
	}
}

func TestInstallDependencyBlockedNeverInstalls(t *testing.T) {
	repoRoot := t.TempDir()
	writeRegistry(t, repoRoot, `{
		"records": [
			{"ecosystem":"npm","package_name":"left-pad","version_range":"*","state":"denied","rationale":"Denied."}
		]
	}`)
	uiDir := filepath.Join(repoRoot, "scenarios", "demo", "ui")
	if err := os.MkdirAll(uiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(uiDir, "pnpm-lock.yaml"), nil, 0o644)

	installer := &recordingInstaller{}
	h := &connectHandler{
		scenariosDir: func() string { return filepath.Join(repoRoot, "scenarios") },
		installer:    installer,
	}
	// Even with apply=true, a denied package must never install (fail closed).
	resp, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
		Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageName: "left-pad", Apply: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if installer.called {
		t.Fatal("blocked install must never invoke the installer")
	}
	if !resp.Msg.GetBlocked() || resp.Msg.GetInstalled() {
		t.Fatalf("blocked resp = %#v", resp.Msg)
	}
}
