package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/scopecatalog"

	"scenario-to-cloud/cli/internal/clidoc"
)

// catalogRoot builds an isolated repository root holding the real schema,
// a minimal project manifest and only this scenario, so the catalog test
// proves this manifest and is not coupled to concurrent edits of other
// manifests in the working tree.
func catalogRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	real := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	root := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755))
	must(os.Symlink(filepath.Join(real, ".vrooli", "schemas"), filepath.Join(root, ".vrooli", "schemas")))
	must(os.MkdirAll(filepath.Join(root, "cli"), 0o755))
	must(os.WriteFile(filepath.Join(root, "cli", "manifest.json"), []byte(`{"name":"vrooli","groups":[{"name":"scenario","commands":[{"name":"status","binding":{"kind":"local"},"governance":{"effect":"read","run_eligible":true}}]}]}`), 0o644))
	// WalkDir does not follow symlinked directories, so the scenario's two
	// declaration files are copied.
	scenario := filepath.Join(root, "scenarios", "scenario-to-cloud")
	must(os.MkdirAll(filepath.Join(scenario, "cli"), 0o755))
	must(os.MkdirAll(filepath.Join(scenario, ".vrooli"), 0o755))
	for _, rel := range []string{filepath.Join("cli", "manifest.json"), filepath.Join(".vrooli", "service.json")} {
		raw, err := os.ReadFile(filepath.Join(real, "scenarios", "scenario-to-cloud", rel))
		must(err)
		must(os.WriteFile(filepath.Join(scenario, rel), raw, 0o644))
	}
	return root
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	return raw
}

// TestScopeCatalogDerivesScenarioScopes [REQ:STC-P0-037] [REQ:STC-P0-012]
// proves the CLI manifest yields the three scenario-to-cloud scopes with an
// authentication profile that matches service.json, and that the secret,
// recovery, rollback and credential verbs are never read-scoped.
func TestScopeCatalogDerivesScenarioScopes(t *testing.T) {
	catalog, err := scopecatalog.Build(catalogRoot(t))
	if err != nil {
		t.Fatalf("scopecatalog.Build: %v", err)
	}
	for _, scope := range []string{"scenario-to-cloud:read", "scenario-to-cloud:write", "scenario-to-cloud:destructive"} {
		if !catalog.HasScope(scope) {
			t.Fatalf("catalog lacks %s", scope)
		}
	}
	byCommand := map[string]scopecatalog.Scope{}
	for _, s := range catalog.Scopes {
		if s.Scenario == "scenario-to-cloud" {
			byCommand[s.Command] = s
		}
	}
	nonRead := []string{
		"secrets/set", "secrets/get", "secrets/delete",
		"deployment/recovery-points/capture", "deployment/recovery-points/restore", "deployment/rollback",
		"credential/rotate", "credential/revoke", "credential/recover", "credential/rotation-resume",
		"deployment/apply", "deployment/execute", "deployment/start", "deployment/delete", "redeploy",
		"publication/apply", "operation/cancel", "process/kill",
	}
	for _, command := range nonRead {
		s, ok := byCommand[command]
		if !ok {
			t.Fatalf("command %s is not governed by the manifest; governed: %v", command, keys(byCommand))
		}
		if s.Effect == scopecatalog.EffectRead {
			t.Fatalf("command %s must not be read-scoped, got %s", command, s.Value)
		}
	}
	// Invariant (lead decision, phase 18/20): destructive verbs are agent
	// run-eligible only when they require confirmation and carry the
	// destructive scope; exactly the verbs the QEMU qualification journey
	// needs are run-eligible; read verbs never require confirmation.
	m, err := clidoc.Parse(readManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	wantRunEligibleDestructive := map[string]bool{"deployment/apply": true, "deployment/rollback": true, "deployment/recovery-points/restore": true}
	m.Walk(func(path string, c clidoc.Command) {
		g := c.Governance
		if g.Effect == "read" && g.RequiresConfirmation {
			t.Fatalf("read command %s must not require confirmation", path)
		}
		if g.Effect == "destructive" && g.RunEligible {
			if !g.RequiresConfirmation {
				t.Fatalf("run-eligible destructive command %s must require confirmation", path)
			}
			if !wantRunEligibleDestructive[path] {
				t.Fatalf("destructive command %s is run-eligible but is not a QEMU-journey verb", path)
			}
			if s := byCommand[path]; s.Value != "scenario-to-cloud:destructive" || !s.RunEligible {
				t.Fatalf("catalog scope for %s = %+v, want run-eligible scenario-to-cloud:destructive", path, s)
			}
			delete(wantRunEligibleDestructive, path)
		}
	})
	if len(wantRunEligibleDestructive) != 0 {
		t.Fatalf("QEMU-journey verbs not run-eligible: %v", wantRunEligibleDestructive)
	}
	for _, command := range []string{"deployment/resolve", "deployment/plan", "deployment/health", "operation/wait", "operation/resume", "inspect/logs", "edge/status", "credential/list", "publication/status"} {
		s, ok := byCommand[command]
		if !ok || s.Effect != scopecatalog.EffectRead {
			t.Fatalf("command %s must be read-scoped, got %+v ok=%v", command, s, ok)
		}
	}
	var profile *scopecatalog.AuthenticationProfile
	for i := range catalog.AuthenticationProfiles {
		if catalog.AuthenticationProfiles[i].Scenario == "scenario-to-cloud" {
			profile = &catalog.AuthenticationProfiles[i]
		}
	}
	if profile == nil {
		t.Fatal("no authentication profile for scenario-to-cloud")
	}
	if len(profile.MissingDeclarations) != 0 {
		t.Fatalf("authentication profile findings: %v (CLI manifest and service.json must agree)", profile.MissingDeclarations)
	}
	if profile.Profile != "scenario_authenticator" || profile.DefaultMode != "personal_local" || len(profile.Capabilities) != 3 {
		t.Fatalf("profile = %+v", *profile)
	}
}

func keys(m map[string]scopecatalog.Scope) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TestManifestMatchesCommandTree [REQ:STC-P0-037] proves every group and
// command the manifest declares is dispatched by the built CLI (the group
// help lists it), so the Bridge's manifest-derived vocabulary cannot name a
// command the binary does not have.
func TestManifestMatchesCommandTree(t *testing.T) {
	m, err := clidoc.Parse(readManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	app := newTestApp(t)
	newParityServer(t) // groups need a reachable API before printing usage
	topHelp := captureStdout(t, func() {
		if err := app.Run([]string{"help"}); err != nil {
			t.Fatal(err)
		}
	})
	var check func(g clidoc.Group, parents []string)
	check = func(g clidoc.Group, parents []string) {
		if g.Flat {
			for _, c := range g.Commands {
				if !strings.Contains(topHelp, c.Name) {
					t.Fatalf("flat command %s not in top-level help:\n%s", c.Name, topHelp)
				}
			}
			return
		}
		path := append(append([]string(nil), parents...), g.Name)
		if len(parents) == 0 && !strings.Contains(topHelp, g.Name) {
			t.Fatalf("group %s not in top-level help:\n%s", g.Name, topHelp)
		}
		// A group invoked without a subcommand prints its own usage, which is
		// the surface the manifest must match.
		// Some nested groups return their usage as an error; both surfaces count.
		var usageErr error
		help := captureStdout(t, func() { usageErr = app.Run(path) })
		if usageErr != nil {
			help += "\n" + usageErr.Error()
		}
		for _, c := range g.Commands {
			if !strings.Contains(help, c.Name) {
				t.Fatalf("command %s %s not listed by its group help:\n%s", strings.Join(path, " "), c.Name, help)
			}
		}
		for _, child := range g.Groups {
			if !strings.Contains(help, child.Name) {
				t.Fatalf("subgroup %s not listed by %v help:\n%s", child.Name, path, help)
			}
			check(child, path)
		}
	}
	for _, g := range m.Groups {
		check(g, nil)
	}
}

// TestCLIDocMatchesManifest [REQ:STC-P0-037] proves docs/reference/cli-commands.md
// is generated from the manifest; regenerate with STC_WRITE_CLI_DOC=1.
func TestCLIDocMatchesManifest(t *testing.T) {
	m, err := clidoc.Parse(readManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	want := clidoc.Render(m)
	path := filepath.Join("..", "docs", "reference", "cli-commands.md")
	if os.Getenv("STC_WRITE_CLI_DOC") == "1" {
		if err := os.WriteFile(path, []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (regenerate with STC_WRITE_CLI_DOC=1): %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("%s is stale; regenerate with STC_WRITE_CLI_DOC=1 go test ./ -run TestCLIDocMatchesManifest", path)
	}
	for _, needle := range []string{"scenario-to-cloud deployment plan [deployment_id]", "| 124 |", "deployment_selector_ambiguous", "scenario-to-cloud operation resume <operation_id>"} {
		if !strings.Contains(want, needle) {
			t.Fatalf("generated doc lacks %q", needle)
		}
	}
}
