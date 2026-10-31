package dependencygovernance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestPythonBootstrapGovernsEveryAuthoredDirectRequirement(t *testing.T) {
	const wheelVersion = "2.8.3.post2+cu12torch2.7cxx11abiTRUE"
	for _, tc := range []struct {
		name, record, verdict string
		blocked               bool
	}{
		{"unrecorded", "", "unrecorded", true},
		{"denied", `,{"ecosystem":"pip","package_name":"flash-attn","version_range":"*","state":"denied"}`, "denied", true},
		{"wrong version", `,{"ecosystem":"pip","package_name":"flash-attn","version_range":"2.8.3.post2","state":"approved"}`, "range_unparseable", true},
		{"wrong surface", `,{"ecosystem":"pip","package_name":"flash-attn","version_range":"*","state":"approved","allowed_surfaces":["api"]}`, "surface_not_allowed", true},
		{"exact wheel approved", `,{"ecosystem":"pip","package_name":"flash-attn","version_range":"` + wheelVersion + `","state":"approved","allowed_surfaces":["resource"]}`, "approved", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			fixture := `{"records":[{"ecosystem":"pip","package_name":"torch","version_range":"2.7.1+cu126","state":"approved","allowed_surfaces":["resource"]}` + tc.record + `]}`
			writeRegistry(t, repo, strings.ReplaceAll(fixture, `"state":`, `"rationale":"Isolated reviewed fixture","state":`))
			root := filepath.Join(repo, "resources", "demo")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, data := range map[string]string{
				"requirements.in": "torch==2.7.1+cu126\nflash-attn @ https://example.com/flash_attn-" + wheelVersion + "-cp312-cp312-linux_x86_64.whl#sha256=" + strings.Repeat("a", 64),
				"resource.json":   `{"compose":[{"kind":"python-wheels","lockfile":"requirements.lock","index_url":"https://pypi.org/simple"}]}`,
			} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			installer := &recordingInstaller{}
			h := &connectHandler{scenariosDir: func() string { return filepath.Join(repo, "scenarios") }, installer: installer}
			resp, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{Scenario: "demo", Surface: "resource", Ecosystem: "pip", PackageName: "torch", Version: "2.7.1+cu126", Apply: true}))
			if err != nil {
				t.Fatal(err)
			}
			if resp.Msg.Blocked != tc.blocked || resp.Msg.Verdict != tc.verdict || installer.called == tc.blocked {
				t.Fatalf("verdict=%s blocked=%t installer=%t", resp.Msg.Verdict, resp.Msg.Blocked, installer.called)
			}
			if !tc.blocked && (!resp.Msg.SourceUpdated || resp.Msg.Installed) {
				t.Fatal("lock bootstrap claimed runtime installation")
			}
		})
	}
}

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
	if resp.Msg.GetCommand() != "pnpm add --ignore-scripts react-hook-form@7.2.0" {
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

func TestInstallDependencyEnforcesScenarioScopeBeforeEffects(t *testing.T) {
	for _, tc := range []struct {
		name, scope string
		blocked     bool
	}{
		{"global", ``, false},
		{"allowed", `,"allowed_scenarios":["demo"]`, false},
		{"other-scenario", `,"allowed_scenarios":["other"]`, true},
		{"denied", `,"denied_scenarios":["demo"]`, true},
		{"denial-precedes-grant", `,"allowed_scenarios":["demo"],"denied_scenarios":["demo"]`, true},
	} {
		for _, version := range []string{"4.3.2", "override:4.3.2"} {
			t.Run(tc.name+"/"+version, func(t *testing.T) {
				root := t.TempDir()
				writeRegistry(t, root, fmt.Sprintf(`{"records":[{"ecosystem":"npm","package_name":"js-yaml","version_range":">=4.3.2","state":"approved","rationale":"fixture"%s}]}`, tc.scope))
				dir := filepath.Join(root, "scenarios", "demo", "ui")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				const original = `{"name":"fixture","pnpm":{"overrides":{"js-yaml@^4.0.0":"4.3.1"}}}`
				manifest := filepath.Join(dir, "package.json")
				if err := os.WriteFile(manifest, []byte(original), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
				installer := &recordingInstaller{}
				h := &connectHandler{scenariosDir: func() string { return filepath.Join(root, "scenarios") }, installer: installer}
				resp, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{
					Scenario: "demo", Surface: "ui", Ecosystem: "npm", PackageName: "js-yaml", Version: version, Apply: true,
				}))
				if err != nil {
					t.Fatal(err)
				}
				if resp.Msg.GetBlocked() != tc.blocked || installer.called == tc.blocked {
					t.Fatalf("blocked=%v installer called=%v, want blocked=%v", resp.Msg.GetBlocked(), installer.called, tc.blocked)
				}
				if tc.blocked {
					data, err := os.ReadFile(manifest)
					if err != nil || string(data) != original || resp.Msg.GetInstalled() {
						t.Fatalf("rejected request changed manifest or installed: %s, %v", data, err)
					}
				}
			})
		}
	}
}

func TestPythonLockUpdateDoesNotClaimRuntimeInstallation(t *testing.T) {
	repo := t.TempDir()
	writeRegistry(t, repo, `{"records":[{"ecosystem":"pip","package_name":"trimesh","version_range":"4.7.4","state":"approved","rationale":"Isolated mesh fixture","allowed_surfaces":["api"]}]}`)
	api := filepath.Join(repo, "scenarios", "demo", "api")
	owner := filepath.Join(api, "internal", "pydeps")
	if err := os.MkdirAll(owner, 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(api, "requirements.txt"):    "-r internal/pydeps/requirements.lock\n",
		filepath.Join(owner, "requirements.in"):   "trimesh>=4\n",
		filepath.Join(owner, "requirements.lock"): "trimesh==4.7.3\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	installer := &recordingInstaller{}
	h := &connectHandler{scenariosDir: func() string { return filepath.Join(repo, "scenarios") }, installer: installer}
	result, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{Scenario: "demo", Surface: "api", Ecosystem: "pip", PackageName: "trimesh", Version: "4.7.4", Apply: true}))
	if err != nil {
		t.Fatal(err)
	}
	if !installer.called || result.Msg.Blocked || !result.Msg.SourceUpdated || result.Msg.Installed {
		t.Fatalf("lock update was reported as runtime install: %+v", result.Msg)
	}
}

func TestHarnessInstallGovernanceBeforeEffects(t *testing.T) {
	for _, tc := range []struct{ state, version, surface, verdict string }{{"approved", "0.164.0", "host-tool", "approved"}, {"denied", "0.164.0", "host-tool", "denied"}, {"approved", "0.163.0", "host-tool", "out_of_range"}, {"approved", "0.164.0", "cli", "surface_not_allowed"}, {"unrecorded", "0.164.0", "host-tool", "unrecorded"}} {
		t.Run(tc.verdict, func(t *testing.T) {
			root := t.TempDir()
			record := `{"records":[]}`
			if tc.state != "unrecorded" {
				record = fmt.Sprintf(`{"records":[{"ecosystem":"npm","package_name":"@openai/codex","state":"%s","rationale":"Reviewed fixture.","version_range":"0.164.0","allowed_surfaces":["%s"]}]}`, tc.state, tc.surface)
			}
			writeRegistry(t, root, record)
			dir := filepath.Join(root, "resources", "codex")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "..", "resources", "codex", "resource.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "resource.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			installer := &recordingInstaller{}
			h := &connectHandler{scenariosDir: func() string { return filepath.Join(root, "scenarios") }, installer: installer}
			resp, err := h.InstallDependency(context.Background(), connect.NewRequest(&governancev1.InstallDependencyRequest{Scenario: "codex", Surface: "host-tool", Ecosystem: "npm", PackageName: "@openai/codex", Version: tc.version, ExactVersion: true, Apply: true}))
			if err != nil {
				t.Fatal(err)
			}
			want := tc.verdict == "approved"
			if installer.called != want || resp.Msg.GetInstalled() != want || resp.Msg.GetBlocked() == want || resp.Msg.GetVerdict() != tc.verdict {
				t.Fatalf("unsafe governance outcome: called=%v resp=%v", installer.called, resp.Msg)
			}
		})
	}
}
