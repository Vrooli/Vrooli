package hygiene

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// canonicalRootWorkspace is what a healed, valid root pnpm-workspace.yaml looks
// like. Tests reuse it to assert the valid case and idempotency.
func canonicalRootWorkspace(t *testing.T) string {
	t.Helper()
	healed, _ := healPnpmWorkspace(nil, true)
	return string(healed)
}

func runPnpmHygiene(t *testing.T, root string, fixSafe bool) Report {
	t.Helper()
	report, err := Service{Root: root, Home: root}.Run(Request{
		FailOn:            SeverityError,
		IncludePnpmConfig: true,
	}.withFixSafe(fixSafe))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return report
}

// withFixSafe is a tiny test helper so the table reads cleanly.
func (r Request) withFixSafe(v bool) Request {
	r.FixSafe = v
	return r
}

func findingCodes(report Report) map[string]Finding {
	out := map[string]Finding{}
	for _, f := range report.Findings {
		out[f.Code] = f
	}
	return out
}

func TestPnpmConfigDetectsStrippedCommentAndWrongSettings(t *testing.T) {
	root := t.TempDir()
	// Mirrors exactly what pnpm produced when it rewrote the file: no comment,
	// autoInstallPeers true, no public-hoist-pattern.
	writeFile(t, filepath.Join(root, "pnpm-workspace.yaml"), strings.Join([]string{
		"packages:",
		"  - packages/*",
		"",
		"autoInstallPeers: true",
		"",
		"link-workspace-packages: false",
		"",
		"packageManager: pnpm@10.14.0",
		"",
		"shared-workspace-lockfile: false",
		"",
	}, "\n"))
	// Redundant root .npmrc duplicating workspace keys.
	writeFile(t, filepath.Join(root, ".npmrc"), "link-workspace-packages=false\npublic-hoist-pattern[]=*eslint*\n")

	report := runPnpmHygiene(t, root, false)
	codes := findingCodes(report)

	for _, want := range []string{
		"pnpm_workspace_comment",
		"pnpm_workspace_autoInstallPeers",
		"pnpm_workspace_public_hoist",
		"pnpm_npmrc_redundant",
	} {
		if _, ok := codes[want]; !ok {
			t.Fatalf("missing finding %q; got %v", want, keys(codes))
		}
	}
	// Without --fix-safe, nothing should be written.
	if len(report.ConfigFixes) != 0 {
		t.Fatalf("ConfigFixes should be empty without --fix-safe, got %v", report.ConfigFixes)
	}
}

func TestPnpmConfigHealRestoresCanonicalAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pnpm-workspace.yaml")
	writeFile(t, path, "packages:\n  - packages/*\nautoInstallPeers: true\nlink-workspace-packages: false\npackageManager: pnpm@10.14.0\nshared-workspace-lockfile: false\n")

	report := runPnpmHygiene(t, root, true)
	if len(report.ConfigFixes) == 0 {
		t.Fatalf("expected a config fix to be recorded")
	}
	healed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read healed file: %v", err)
	}
	if !strings.Contains(string(healed), pnpmWorkspaceCommentMarker) {
		t.Fatalf("healed file missing comment marker:\n%s", healed)
	}
	if !strings.Contains(string(healed), "autoInstallPeers: false") {
		t.Fatalf("heal did not enforce autoInstallPeers: false:\n%s", healed)
	}
	if !strings.Contains(string(healed), `- "*eslint*"`) || !strings.Contains(string(healed), `- "*prettier*"`) {
		t.Fatalf("heal did not migrate public-hoist-pattern:\n%s", healed)
	}
	if string(healed) != canonicalRootWorkspace(t) {
		t.Fatalf("healed file is not canonical:\n%s", healed)
	}

	// A second heal pass must report no change (idempotent) and pass cleanly.
	report2 := runPnpmHygiene(t, root, true)
	if len(report2.ConfigFixes) != 0 {
		t.Fatalf("second heal should be a no-op, got %v", report2.ConfigFixes)
	}
	codes := findingCodes(report2)
	for code := range codes {
		if strings.HasPrefix(code, "pnpm_workspace_") {
			t.Fatalf("healed file still reports %q", code)
		}
	}
}

func TestPnpmConfigFlagsScenarioLeakAndHealsIt(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pnpm-workspace.yaml")
	writeFile(t, path, "packages:\n  - packages/*\n  - scenarios/*/ui\nautoInstallPeers: false\nlink-workspace-packages: false\nshared-workspace-lockfile: false\npackageManager: pnpm@10.14.0\npublic-hoist-pattern:\n  - \"*eslint*\"\n  - \"*prettier*\"\n")

	report := runPnpmHygiene(t, root, false)
	codes := findingCodes(report)
	leak, ok := codes["pnpm_workspace_scenario_leak"]
	if !ok {
		t.Fatalf("expected scenario leak finding, got %v", keys(codes))
	}
	if leak.Severity != SeverityError {
		t.Fatalf("scenario leak should be an error, got %q", leak.Severity)
	}

	// Heal must strip the scenario path.
	runPnpmHygiene(t, root, true)
	healed, _ := os.ReadFile(path)
	if strings.Contains(string(healed), "scenarios/") && !strings.Contains(string(healed), "# IMPORTANT: Do NOT add scenarios") {
		t.Fatalf("heal left a scenario path in packages:\n%s", healed)
	}
	// The only scenarios/ mention left should be inside the comment block.
	if strings.Contains(string(healed), "- scenarios/") {
		t.Fatalf("heal left a scenario list entry:\n%s", healed)
	}
}

func TestPnpmConfigPreservesUnmanagedKeys(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "pnpm-workspace.yaml")
	writeFile(t, path, "packages:\n  - packages/*\nonlyBuiltDependencies:\n  - esbuild\nautoInstallPeers: true\nlink-workspace-packages: false\nshared-workspace-lockfile: false\npackageManager: pnpm@10.14.0\n")

	runPnpmHygiene(t, root, true)
	healed, _ := os.ReadFile(path)
	if !strings.Contains(string(healed), "onlyBuiltDependencies:") || !strings.Contains(string(healed), "- esbuild") {
		t.Fatalf("heal dropped an unmanaged key (onlyBuiltDependencies):\n%s", healed)
	}
}

func TestPnpmConfigDetectsScenarioWorkspaceStar(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pnpm-workspace.yaml"), canonicalRootWorkspace(t))
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"),
		`{"name":"demo-ui","dependencies":{"@vrooli/api-base":"workspace:*"}}`)
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")

	report := runPnpmHygiene(t, root, false)
	codes := findingCodes(report)
	star, ok := codes["scenario_workspace_star"]
	if !ok {
		t.Fatalf("expected scenario_workspace_star finding, got %v", keys(codes))
	}
	if star.Severity != SeverityError {
		t.Fatalf("workspace:* should be an error, got %q", star.Severity)
	}
	if !containsString(star.Locations, "scenarios/demo/ui/package.json") {
		t.Fatalf("expected location scenarios/demo/ui/package.json, got %v", star.Locations)
	}
}

func TestPnpmConfigWarnsOnMissingScenarioLockfile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pnpm-workspace.yaml"), canonicalRootWorkspace(t))
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"),
		`{"name":"demo-ui","dependencies":{"@vrooli/api-base":"file:../../../packages/api-base"}}`)

	report := runPnpmHygiene(t, root, false)
	codes := findingCodes(report)
	lock, ok := codes["scenario_missing_lockfile"]
	if !ok {
		t.Fatalf("expected scenario_missing_lockfile finding, got %v", keys(codes))
	}
	if lock.Severity != SeverityWarning {
		t.Fatalf("missing lockfile should be a warning, got %q", lock.Severity)
	}
}

func TestPnpmConfigValidFilePasses(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pnpm-workspace.yaml"), canonicalRootWorkspace(t))
	// A clean isolated scenario: lockfile + workspace boundary present.
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"),
		`{"name":"demo-ui","dependencies":{"@vrooli/api-base":"file:../../../packages/api-base"}}`)
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "pnpm-workspace.yaml"), "packages:\n  - .\n")

	report := runPnpmHygiene(t, root, false)
	for _, f := range report.Findings {
		if strings.HasPrefix(f.Code, "pnpm_") || strings.HasPrefix(f.Code, "scenario_") {
			t.Fatalf("unexpected finding on a valid setup: %s - %s", f.Code, f.Message)
		}
	}
	var sawPnpm, sawScenario bool
	for _, c := range report.Checks {
		switch c.Name {
		case "pnpm_config":
			sawPnpm = true
			if !c.Passed {
				t.Fatalf("pnpm_config check failed: %s", c.Message)
			}
		case "scenario_pnpm":
			sawScenario = true
			if !c.Passed {
				t.Fatalf("scenario_pnpm check failed: %s", c.Message)
			}
		}
	}
	if !sawPnpm || !sawScenario {
		t.Fatalf("expected pnpm_config and scenario_pnpm checks, got %+v", report.Checks)
	}
}

func keys(m map[string]Finding) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestPnpmConfigWarnsOnMissingScenarioBoundary(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pnpm-workspace.yaml"), canonicalRootWorkspace(t))
	// Lockfile present but no pnpm-workspace.yaml boundary: a plain install
	// there would walk up and join the root workspace.
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "package.json"),
		`{"name":"demo-ui","dependencies":{"@vrooli/api-base":"file:../../../packages/api-base"}}`)
	writeFile(t, filepath.Join(root, "scenarios", "demo", "ui", "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")

	report := runPnpmHygiene(t, root, false)
	codes := findingCodes(report)
	boundary, ok := codes["scenario_missing_workspace_boundary"]
	if !ok {
		t.Fatalf("expected scenario_missing_workspace_boundary finding, got %v", keys(codes))
	}
	if boundary.Severity != SeverityWarning {
		t.Fatalf("missing boundary should be a warning, got %q", boundary.Severity)
	}
	if !containsString(boundary.Locations, "scenarios/demo/ui") {
		t.Fatalf("expected location scenarios/demo/ui, got %v", boundary.Locations)
	}
}

func TestPnpmConfigFlagsAndRemovesStrayRootLockfile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pnpm-workspace.yaml"), canonicalRootWorkspace(t))
	writeFile(t, filepath.Join(root, "pnpm-lock.yaml"), "lockfileVersion: '9.0'\n")

	// Without fix-safe: flagged as an error.
	report := runPnpmHygiene(t, root, false)
	codes := findingCodes(report)
	stray, ok := codes["pnpm_root_lockfile_stray"]
	if !ok {
		t.Fatalf("expected pnpm_root_lockfile_stray finding, got %v", keys(codes))
	}
	if stray.Severity != SeverityError {
		t.Fatalf("stray root lock should be an error, got %q", stray.Severity)
	}

	// With fix-safe: removed, no finding, fix recorded.
	report = runPnpmHygiene(t, root, true)
	if _, again := findingCodes(report)["pnpm_root_lockfile_stray"]; again {
		t.Fatal("stray root lockfile finding should clear after fix-safe heal")
	}
	if _, err := os.Stat(filepath.Join(root, "pnpm-lock.yaml")); !os.IsNotExist(err) {
		t.Fatal("fix-safe should remove the stray root pnpm-lock.yaml")
	}
	found := false
	for _, fix := range report.ConfigFixes {
		if strings.Contains(fix, "stray root pnpm-lock.yaml") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ConfigFixes entry for stray root lock removal, got %v", report.ConfigFixes)
	}
}
