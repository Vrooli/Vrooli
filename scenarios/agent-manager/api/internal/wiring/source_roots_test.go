package wiring

import (
	"agent-manager/internal/runreport"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/rolepolicy"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestRetainedReceiptAndSupervisionUseCanonicalSource(t *testing.T) {
	fixture := repocontracttest.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	fixture.WriteScenarioStub(t, "agent-manager")
	source := filepath.Join(fixture.Root, "scenarios", "agent-manager")
	retained := t.TempDir()
	for _, key := range []string{"PROJECT_ROOT", "VROOLI_SOURCE_ROOT", "VROOLI_ROOT", "AGENT_MANAGER_EFFORT_ROOT", "AGENT_MANAGER_ROLE_POLICY_CATALOG_PATH", "AGENT_MANAGER_PERMISSION_POLICY_CATALOG_PATH"} {
		t.Setenv(key, "")
	}
	t.Setenv("SCENARIO_NAME", "agent-manager")
	t.Setenv("VROOLI_SCENARIO", "agent-manager")
	t.Setenv("SCENARIO_PATH", source)
	t.Setenv("VROOLI_SCENARIO_DIR", source)
	runtime := t.TempDir()
	t.Setenv("AGENT_MANAGER_RUNTIME_ROOT", runtime)
	declaration := filepath.Join(source, ".vrooli", "vrooli-events", "receipt-capture.json")
	repocontracttest.WriteFile(t, declaration, `{"policies":[{"targetScenario":"canonical-target"}]}`)
	t.Chdir(retained)
	path, err := receiptCaptureDeclarationPath()
	if err != nil || path != declaration {
		t.Fatalf("receipt path=%q err=%v", path, err)
	}
	targets, err := receiptCaptureTargets()
	if err != nil || len(targets) != 1 || targets[0] != "canonical-target" {
		t.Fatalf("targets=%v err=%v", targets, err)
	}
	effort, err := effortDiscoveryConfig()
	if err != nil || effort.ProjectRoot != fixture.Root {
		t.Fatalf("effort=%+v err=%v", effort, err)
	}
	contract, err := repocontract.LoadDefault(fixture.Root)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := contract.RuntimeHomeEntry(string(filepath.Separator), repocontract.HomeKeyPlanArtifacts)
	if err != nil {
		t.Fatal(err)
	}
	if effort.Root != filepath.Join(runtime, entry.RelPath, "efforts") {
		t.Fatalf("effort state moved to source: %q", effort.Root)
	}
	if got := rolepolicy.ResolvePath(); got != filepath.Join(source, "config", "role-policy-catalog.json") {
		t.Fatalf("role catalog=%q", got)
	}
	if got := permissionpolicy.ResolvePath(); got != filepath.Join(source, "config", "permission-policy-catalog.json") {
		t.Fatalf("permission catalog=%q", got)
	}
	explicitEfforts := t.TempDir()
	t.Setenv("AGENT_MANAGER_EFFORT_ROOT", explicitEfforts)
	effort, err = effortDiscoveryConfig()
	if err != nil || effort.Root != explicitEfforts || effort.ProjectRoot != fixture.Root {
		t.Fatalf("explicit effort=%+v err=%v", effort, err)
	}
	t.Setenv("PROJECT_ROOT", filepath.Join(fixture.Root, "missing"))
	if _, err = receiptCaptureDeclarationPath(); err == nil {
		t.Fatal("receipt accepted invalid root")
	}
	if _, err = effortDiscoveryConfig(); err == nil {
		t.Fatal("supervision accepted invalid root")
	}
	if rolepolicy.ResolvePath() != "" || permissionpolicy.ResolvePath() != "" {
		t.Fatal("invalid root fell through to relative catalog")
	}
	if cwd, _ := os.Getwd(); cwd != retained {
		t.Fatal("lookup changed runtime working directory")
	}
}

func TestInvalidDeclarationSourceIsUnavailableNotPolicyAbsent(t *testing.T) {
	summary, decisive := receiptCaptureAvailability(context.Background(), nil, nil, fmt.Errorf("invalid root"))
	if !decisive || summary.State != runreport.AvailabilityUnavailable {
		t.Fatalf("summary=%+v decisive=%v", summary, decisive)
	}
	for _, freshness := range catalogFreshnessReader("")(context.Background()) {
		if freshness.Status != "invalid" || freshness.PolicyPath != "" {
			t.Fatalf("invalid root read relative catalog: %+v", freshness)
		}
	}
}
