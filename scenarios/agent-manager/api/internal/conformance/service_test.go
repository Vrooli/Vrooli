package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type failingPermissionPosture struct{ err error }

func (p failingPermissionPosture) ReadinessError(context.Context) error { return p.err }

// validProfileJSON is a unified-layout profile source: schemaVersion-discriminated
// and role-only.
func validProfileJSON(key string) string {
	return `{"schemaVersion":"agent-profile/v1","profileKey":"` + key + `","roleRef":"code.default"}`
}

// validWorkflowJSON is a minimal but complete unified-layout workflow source with
// a single run node whose prompt placeholder is backed by a declared binding.
func validWorkflowJSON(owner, key string) string {
	return `{"schemaVersion":"agent-workflow/v1","owner":"` + owner + `","key":"` + key + `","version":"1.0.0",` +
		`"inputSchema":{"type":"object","additionalProperties":false},"outputSchema":{"type":"object","additionalProperties":false},` +
		`"entryNode":"start","nodes":[` +
		`{"id":"start","kind":"run","run":{"roleRef":"code.default","promptRef":{"skillId":"fixture-skill"}}},` +
		`{"id":"done","kind":"end","end":{"status":"succeeded"}}],` +
		`"edges":[{"from":"start","to":"done"}],` +
		`"budgets":{"wallTimeSeconds":60,"maxTurns":4,"maxTokens":1000,"maxChargeMicroUsd":1,"maxNodeAttempts":3,"maxChildren":2,"maxConcurrency":2,"maxRecursion":2,"maxRetries":2,"maxWaitSeconds":30}}`
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func countFindings(findings []Finding, code string) int {
	n := 0
	for _, f := range findings {
		if f.Code == code {
			n++
		}
	}
	return n
}

// TestValidateAcceptsUnifiedProfileAndWorkflowDeclarations proves a scenario on
// the new .vrooli/agent-manager/ layout with a valid profile and workflow both
// declared through config.declarations passes conformance cleanly.
func TestValidateAcceptsUnifiedProfileAndWorkflowDeclarations(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"),
		`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"declarations":{"reconcile":true,"profileMode":"update_if_unmodified","sources":[".vrooli/agent-manager/default.json",".vrooli/agent-manager/flow.json"]}}}}}}`)
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "default.json"), validProfileJSON("consumer/default"))
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "flow.json"), validWorkflowJSON("consumer", "consumer/flow"))
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %#v, want clean unified declarations", report.Findings)
	}
}

func TestValidateInlineWorkflowPromptBlocksMatureRung(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"),
		`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-manager/flow.json"]}}}}}}`)
	inline := strings.Replace(validWorkflowJSON("consumer", "consumer/flow"), `"promptRef":{"skillId":"fixture-skill"}`, `"promptTemplate":"Do work"`, 1)
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "flow.json"), inline)
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(report.Findings, CodeWorkflowInlinePrompt) {
		t.Fatalf("expected promptRef maturity finding, got %#v", report.Findings)
	}
}

// TestValidateRejectsLegacyLayoutDirectoriesAndBlocks proves the no-fallback
// cutover: files remaining under the retired directories and legacy config
// blocks are flagged as violations.
func TestValidateRejectsLegacyLayoutDirectoriesAndBlocks(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	// Legacy config blocks AND legacy-directory files still present.
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"),
		`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"profiles":{"reconcile":true,"sources":[".vrooli/agent-profiles/default.json"]},"workflows":{"sources":[".vrooli/agent-workflows/flow.json"]}}}}}}`)
	writeFile(t, filepath.Join(root, ".vrooli", "agent-profiles", "default.json"), validProfileJSON("consumer/default"))
	writeFile(t, filepath.Join(root, ".vrooli", "agent-workflows", "flow.json"), validWorkflowJSON("consumer", "consumer/flow"))
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if countFindings(report.Findings, CodeDeclarationLegacyBlock) != 2 {
		t.Fatalf("want two legacy-block findings, got %#v", report.Findings)
	}
	if countFindings(report.Findings, CodeDeclarationLegacyLayout) != 2 {
		t.Fatalf("want two legacy-layout findings, got %#v", report.Findings)
	}
}

// TestValidateReportsAllWorkflowDiagnostics proves the conformance scan surfaces
// every blocking workflow diagnostic (CEL compile errors and unbound prompt
// placeholders), not just the first.
func TestValidateReportsAllWorkflowDiagnostics(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	// A branch node with a syntactically broken edge condition and a run node
	// whose prompt references an undeclared placeholder.
	bad := `{"schemaVersion":"agent-workflow/v1","owner":"consumer","key":"consumer/flow","version":"1.0.0",` +
		`"inputSchema":{"type":"object","additionalProperties":false},"outputSchema":{"type":"object","additionalProperties":false},` +
		`"entryNode":"gate","nodes":[` +
		`{"id":"gate","kind":"branch","branch":{}},` +
		`{"id":"start","kind":"run","run":{"roleRef":"code.default","promptTemplate":"Do {{.missing}}"}},` +
		`{"id":"done","kind":"end","end":{"status":"succeeded"}}],` +
		`"edges":[{"from":"gate","to":"start","condition":"iteration <"},{"from":"gate","to":"done","condition":"true"},{"from":"start","to":"done"}],` +
		`"budgets":{"wallTimeSeconds":60,"maxTurns":4,"maxTokens":1000,"maxChargeMicroUsd":1,"maxNodeAttempts":3,"maxChildren":2,"maxConcurrency":2,"maxRecursion":2,"maxRetries":2,"maxWaitSeconds":30}}`
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"),
		`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-manager/flow.json"]}}}}}}`)
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "flow.json"), bad)
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if countFindings(report.Findings, CodeWorkflowInvalid) < 2 {
		t.Fatalf("want at least a CEL and a placeholder finding, got %#v", report.Findings)
	}
}

// TestValidateReportsOrphanUnifiedSources proves undeclared files under the
// unified directory are flagged and routed to the profile or workflow orphan
// code by their schemaVersion.
func TestValidateReportsOrphanUnifiedSources(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"),
		`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-manager/default.json"]}}}}}}`)
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "default.json"), validProfileJSON("consumer/default"))
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "orphan-flow.json"), validWorkflowJSON("consumer", "consumer/orphan"))
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if !hasFinding(report.Findings, CodeWorkflowOrphan) {
		t.Fatalf("undeclared unified workflow source not flagged: %#v", report.Findings)
	}
}

// TestValidateRejectsLegacyProfileFieldInUnifiedLayout keeps the role-only
// contract: a runner/model input in a unified-layout profile is a legacy-field
// violation.
func TestValidateRejectsLegacyProfileFieldInUnifiedLayout(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"),
		`{"dependencies":{"scenarios":{"agent-manager":{"enabled":true,"config":{"declarations":{"reconcile":true,"sources":[".vrooli/agent-manager/default.json"]}}}}}}`)
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "default.json"),
		`{"schemaVersion":"agent-profile/v1","profileKey":"consumer/default","runnerType":"codex"}`)
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 || report.Findings[0].Code != CodeProfileLegacy {
		t.Fatalf("findings = %#v, want a single legacy-field finding", report.Findings)
	}
}

// TestRealDeclaringScenariosCleanOnUnifiedLayout is the post-cutover real-boundary
// check: after the fleet migration, every scenario declaring Agent Manager
// resolves cleanly on the unified layout — no legacy config block, no legacy
// directory file, and no orphan declaration. This is the re-inverted Phase-5
// assertion of the transitional Phase-2 rejection check. At least one scenario
// must actually declare scenario-owned sources so the check has teeth.
func TestRealDeclaringScenariosCleanOnUnifiedLayout(t *testing.T) {
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", ".vrooli", "service.json"))
	if err != nil {
		t.Fatal(err)
	}
	service := Service{RepoRoot: repoRoot}
	legacyOrOrphan := []string{CodeDeclarationLegacyLayout, CodeDeclarationLegacyBlock, CodeProfileOrphan, CodeWorkflowOrphan}
	withSources := 0
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		var manifest struct {
			Dependencies struct {
				Scenarios map[string]struct {
					Enabled *bool           `json:"enabled"`
					Config  json.RawMessage `json:"config"`
				} `json:"scenarios"`
			} `json:"dependencies"`
		}
		if jsonErr := json.Unmarshal(data, &manifest); jsonErr != nil {
			t.Fatalf("parse %s: %v", path, jsonErr)
		}
		dep, ok := manifest.Dependencies.Scenarios["agent-manager"]
		if !ok || (dep.Enabled != nil && !*dep.Enabled) {
			continue
		}
		scenario := filepath.Base(filepath.Dir(filepath.Dir(path)))
		if declaresUnifiedSources(dep.Config) {
			withSources++
		}
		t.Run(scenario, func(t *testing.T) {
			report, valErr := service.Validate(scenario, "")
			if valErr != nil {
				t.Fatal(valErr)
			}
			for _, code := range legacyOrOrphan {
				if hasFinding(report.Findings, code) {
					t.Fatalf("scenario %s still produces %s on the unified layout: %#v", scenario, code, report.Findings)
				}
			}
		})
	}
	if withSources == 0 {
		t.Fatal("found no Agent Manager consumers declaring unified sources; the migration boundary check has nothing to prove")
	}
}

// declaresUnifiedSources reports whether a dependency config carries a
// config.declarations.sources list (a scenario that owns declarations, as
// opposed to one that declares the dependency only for portable runtime roles).
func declaresUnifiedSources(config json.RawMessage) bool {
	if len(config) == 0 {
		return false
	}
	var parsed struct {
		Declarations struct {
			Sources []string `json:"sources"`
		} `json:"declarations"`
	}
	if json.Unmarshal(config, &parsed) != nil {
		return false
	}
	return len(parsed.Declarations.Sources) > 0
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

func TestValidateReportsAnOrphanUnifiedSourceAlongsideMissingDependency(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer", ".vrooli")
	writeFile(t, filepath.Join(root, "service.json"), `{"dependencies":{"scenarios":{}}}`)
	writeFile(t, filepath.Join(root, "agent-manager", "default.json"), validProfileJSON("consumer/default"))
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 2 || !hasFinding(report.Findings, CodeDependencyMissing) || !hasFinding(report.Findings, CodeProfileOrphan) {
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

func TestValidateReportsUndeclaredUnifiedSourcesAndDirectSpawnAsBlocking(t *testing.T) {
	repo := t.TempDir()
	copyRoleCatalog(t, repo)
	root := filepath.Join(repo, "scenarios", "consumer")
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{"dependencies":{"scenarios":{"agent-manager":{"enabled":true}}}}`)
	writeFile(t, filepath.Join(root, ".vrooli", "agent-manager", "orphan.json"), validProfileJSON("consumer/orphan"))
	writeFile(t, filepath.Join(root, "api", "spawn.go"), `package api; import "os/exec"; func f() { exec.Command("codex") }`)
	report, err := (Service{RepoRoot: repo}).Validate("consumer", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 2 || !hasFinding(report.Findings, CodeDirectSpawnBypass) || !hasFinding(report.Findings, CodeProfileOrphan) {
		t.Fatalf("findings = %#v", report.Findings)
	}
}

func TestAgentManagerSourceHasNoDirectCodingAgentSpawn(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if matches := directSpawnBypasses(root); len(matches) != 0 {
		t.Fatalf("direct coding-agent spawn bypasses: %v", matches)
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
