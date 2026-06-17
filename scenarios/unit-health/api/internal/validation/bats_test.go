package validation

import (
	"context"
	"path/filepath"
	"testing"

	"unit-health/internal/discovery"
	"unit-health/internal/executor"
)

func withBatsResolver(t *testing.T, bin string, ok bool) {
	t.Helper()
	orig := batsBinaryResolver
	t.Cleanup(func() { batsBinaryResolver = orig })
	batsBinaryResolver = func([]string) (string, bool) { return bin, ok }
}

func TestNormalizeLanguageBash(t *testing.T) {
	for _, in := range []string{"bash", "sh", "shell", "Bash"} {
		if got := normalizeLanguage(in, ""); got != "bash" {
			t.Errorf("normalizeLanguage(%q) = %q, want bash", in, got)
		}
	}
}

func TestResolveBashWorkspaceNoBatsTests(t *testing.T) {
	withBatsResolver(t, "bats", true)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "run.sh"), "#!/usr/bin/env bash\necho hi\n")
	ws, findings := resolveWorkspace("demo", discovery.Surface{ID: "cli", RootPath: root}, "bash", "now")
	if ws.Status != "degraded" {
		t.Errorf("status = %q, want degraded", ws.Status)
	}
	if !hasFinding(findings, codeTestFrameworkMissing) {
		t.Errorf("want %s, got %v", codeTestFrameworkMissing, codes(findings))
	}
}

func TestResolveBashWorkspaceBatsNotInstalled(t *testing.T) {
	withBatsResolver(t, "", false)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "test", "smoke.bats"), "@test \"ok\" { true; }\n")
	ws, findings := resolveWorkspace("demo", discovery.Surface{ID: "cli", RootPath: root}, "bash", "now")
	if ws.Status != "degraded" {
		t.Errorf("status = %q, want degraded", ws.Status)
	}
	if !hasFinding(findings, codeTestDependencyMissing) {
		t.Errorf("want %s, got %v", codeTestDependencyMissing, codes(findings))
	}
}

func TestResolveBashWorkspaceReady(t *testing.T) {
	withBatsResolver(t, "bats", true)
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "test", "smoke.bats"), "@test \"ok\" { true; }\n")
	ws, findings := resolveWorkspace("demo", discovery.Surface{ID: "cli", RootPath: root}, "bash", "now")
	if ws.Status != "ready" {
		t.Fatalf("status = %q (%s), want ready", ws.Status, ws.DegradedReason)
	}
	if ws.CanonicalFramework != "bats" || ws.TestCommand != "bats --recursive ." {
		t.Errorf("framework=%q cmd=%q; want bats / bats --recursive .", ws.CanonicalFramework, ws.TestCommand)
	}
	if len(findings) != 0 {
		t.Errorf("ready bash workspace should have no findings, got %v", codes(findings))
	}
}

// bashInventory builds an inventory with one shell (bats) surface.
func bashInventory(t *testing.T) discovery.Inventory {
	t.Helper()
	root := t.TempDir()
	cli := filepath.Join(root, "cli")
	writeFile(t, filepath.Join(cli, "run.sh"), "#!/usr/bin/env bash\necho hi\n")
	writeFile(t, filepath.Join(cli, "test", "smoke.bats"), "@test \"ok\" { true; }\n")
	return discovery.Inventory{
		Scenario: "demo", TargetKind: "scenario", RootPath: root,
		Surfaces: []discovery.Surface{{ID: "cli", Kind: "cli", Language: "bash", RootPath: cli, Status: "known"}},
	}
}

func TestExecuteBashSurfaceRunsBats(t *testing.T) {
	withBatsResolver(t, "bats", true)
	spec := loadSpec(t)
	svc := newService(fakeDiscoverer{inv: bashInventory(t)}, spec)
	svc.Executor = fakeExecutorByName{byName: map[string]executor.Result{
		"bash test": {Status: executor.StatusPassed},
	}}
	resp, err := svc.Validate(context.Background(), Request{Scenario: "demo", IncludeExecution: true})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	var ran bool
	for _, cr := range resp.CommandResults {
		if cr.Name == "bash test" && cr.Status == "passed" {
			ran = true
		}
	}
	if !ran {
		t.Errorf("expected the bats test command to run; results=%+v", resp.CommandResults)
	}
}
