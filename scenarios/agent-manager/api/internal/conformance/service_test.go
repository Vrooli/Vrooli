package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

type failingPermissionPosture struct{ err error }

func (p failingPermissionPosture) ReadinessError(context.Context) error { return p.err }

func TestValidateReportsDependencyAndRoleProfileContract(t *testing.T) {
	repo := t.TempDir()
	root := filepath.Join(repo, "scenarios", "consumer")
	if err := os.MkdirAll(filepath.Join(root, ".vrooli", "agent-profiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, ".vrooli", "service.json"), `{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"profiles":{"reconcile":true,"mode":"update_if_unmodified","sources":[".vrooli/agent-profiles/default.json"]}}}}}}`)
	write(filepath.Join(root, ".vrooli", "agent-profiles", "default.json"), `{"profileKey":"consumer/default","roleRef":"code.default"}`)
	copyRoleCatalog(t, repo)
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %#v, want clean profile", report.Findings)
	}

	write(filepath.Join(root, ".vrooli", "agent-profiles", "default.json"), `{"profileKey":"other/default","runnerType":"codex"}`)
	report, err = (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].Code != CodeProfileLegacy {
		t.Fatalf("findings = %#v, want legacy finding", report.Findings)
	}
}

func TestDeclaredAgentManagerConsumersSatisfyProfileContract(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := Service{RepoRoot: repoRoot}
	var consumers []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Dependencies struct {
				Scenarios map[string]struct {
					Enabled *bool `json:"enabled"`
				} `json:"scenarios"`
			} `json:"dependencies"`
		}
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		dep, ok := manifest.Dependencies.Scenarios["agent-manager"]
		if !ok || (dep.Enabled != nil && !*dep.Enabled) {
			continue
		}
		consumers = append(consumers, filepath.Base(filepath.Dir(filepath.Dir(path))))
	}
	sort.Strings(consumers)
	if len(consumers) == 0 {
		t.Fatal("found no enabled Agent Manager consumers")
	}
	for _, scenario := range consumers {
		t.Run(scenario, func(t *testing.T) {
			report, err := service.Validate(scenario, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Findings) != 0 {
				t.Fatalf("findings = %#v", report.Findings)
			}
		})
	}
}

func TestValidateDistinguishesMissingAndDisabledDependency(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer", ".vrooli")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "service.json")
	if err := os.WriteFile(path, []byte(`{"dependencies":{"scenarios":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	service := Service{RepoRoot: repo}
	report, err := service.Validate("consumer", "")
	if err != nil || len(report.Findings) != 1 || report.Findings[0].Code != CodeDependencyMissing {
		t.Fatalf("report=%#v err=%v", report, err)
	}
	if err := os.WriteFile(path, []byte(`{"dependencies":{"scenarios":{"agent-manager":{"enabled":false}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err = service.Validate("consumer", "")
	if err != nil || len(report.Findings) != 1 || report.Findings[0].Code != CodeDependencyDisabled {
		t.Fatalf("report=%#v err=%v", report, err)
	}
}

func TestValidateReportsAnOrphanProfileAlongsideMissingDependency(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer", ".vrooli")
	if err := os.MkdirAll(filepath.Join(root, "agent-profiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "service.json"), []byte(`{"dependencies":{"scenarios":{}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "agent-profiles", "default.json"), []byte(`{"profileKey":"consumer/default","roleRef":"code.default"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 2 || report.Findings[0].Code != CodeDependencyMissing || report.Findings[1].Code != CodeProfileOrphan {
		t.Fatalf("findings = %#v", report.Findings)
	}
}

func TestValidateRejectsEscapingScenarioPaths(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	outside := t.TempDir()
	if _, err := (Service{RepoRoot: repo}).Validate("", outside); err == nil {
		t.Fatal("Validate accepted an explicit path outside the repository scenarios root")
	}
	if _, err := (Service{RepoRoot: repo}).Validate("../outside", ""); err == nil {
		t.Fatal("Validate accepted a non-canonical scenario slug")
	}
}

func TestValidateReportsUndeclaredProfileSourcesAndDirectSpawnAsBlocking(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	if err := os.MkdirAll(filepath.Join(root, ".vrooli", "agent-profiles"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "agent-profiles", "orphan.json"), []byte(`{"profileKey":"consumer/orphan","roleRef":"code.default"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "spawn.go"), []byte(`package api; import "os/exec"; func f() { exec.Command("codex") }`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 2 || report.Findings[0].Code != CodeDirectSpawnBypass || report.Findings[0].Severity != "SEVERITY_ERROR" || report.Findings[1].Code != CodeProfileOrphan {
		t.Fatalf("findings = %#v", report.Findings)
	}
}

func TestValidateReportsUnreadyGlobalPermissionPosture(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer", ".vrooli")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "service.json"), []byte(`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := (Service{RepoRoot: repo, PermissionPosture: failingPermissionPosture{err: errors.New("hard enforcement is stale")}}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].Code != CodePermissionPosture || report.Findings[0].Severity != "SEVERITY_ERROR" {
		t.Fatalf("findings = %#v", report.Findings)
	}
}

func copyRoleCatalog(t *testing.T, repo string) {
	t.Helper()
	source := filepath.Join("..", "..", "..", "config", "role-policy-catalog.json")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(repo, "scenarios", "agent-manager", "config", "role-policy-catalog.json")
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
