package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/vps"
)

func setupPlanManifest(host string, port int, user, workdir, scenario string) domain.CloudManifest {
	return domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS:  &domain.ManifestVPS{Host: host, Port: port, User: user, Workdir: workdir},
		},
		Scenario: domain.ManifestScenario{ID: scenario},
	}
}

func writeFakeBundle(t *testing.T) string {
	t.Helper()
	bundlePath := filepath.Join(t.TempDir(), "mini-vrooli.tar.gz")
	if err := os.WriteFile(bundlePath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write bundle: %v", err)
	}
	return bundlePath
}

func TestBuildSetupPlanIncludesDeliverVerifyStageAndSetup(t *testing.T) {
	// [REQ:STC-P0-004] Install bundle and run setup
	// [REQ:STC-P0-028] Staging is the target owner's verb; activation is runtime scope
	bundlePath := writeFakeBundle(t)
	manifest := setupPlanManifest("203.0.113.10", 22, "root", "/root/Vrooli", "landing-page-business-suite")
	manifest.Dependencies = domain.ManifestDependencies{
		Scenarios: []string{"landing-page-business-suite", "vrooli-autoheal"},
		Resources: []string{"postgres"},
	}
	manifest.Bundle = domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true}
	manifest.Ports = domain.ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002}
	manifest.Edge = domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}}

	plan, err := vps.BuildSetupPlan(manifest, bundlePath)
	if err != nil {
		t.Fatalf("vps.BuildSetupPlan: %v", err)
	}
	got := map[string]domain.VPSPlanStep{}
	for _, step := range plan {
		got[step.ID] = step
	}
	for _, id := range []string{execplan.OpHostPrepare, execplan.OpEdgeFirewallAllow, execplan.OpDataInventory, execplan.OpReleaseDeliver, execplan.OpReleaseVerify, execplan.OpReleaseStage, execplan.OpConfigApply} {
		if _, ok := got[id]; !ok {
			t.Fatalf("expected action %s in plan, got %v", id, plan)
		}
	}
	if _, ok := got[execplan.OpReleaseActivate]; ok {
		t.Fatalf("install scope stages a release; activation belongs to the runtime scope: %v", plan)
	}
	if _, ok := got["cleanup_scenarios"]; ok {
		t.Fatalf("cleanup_scenarios must no longer exist; the release is staged and activated instead")
	}
	if !strings.HasPrefix(got[execplan.OpReleaseDeliver].Command, "scp ") {
		t.Fatalf("expected the deliver preview to be an scp command: %s", got[execplan.OpReleaseDeliver].Command)
	}
	if cmd := got[execplan.OpHostPrepare].Command; !strings.Contains(cmd, "apt.packages.ensure") || !strings.Contains(cmd, "cloud-target") || strings.Contains(cmd, "apt-get") {
		t.Fatalf("host.prepare must delegate to the privilege broker action, not run apt itself: %s", cmd)
	}
	if cmd := got[execplan.OpEdgeFirewallAllow].Command; !strings.Contains(cmd, "edge.ufw.allow") || strings.Contains(cmd, "ufw allow") {
		t.Fatalf("edge.firewall.allow must delegate to the privilege broker action: %s", cmd)
	}
	if cmd := got[execplan.OpConfigApply].Command; !strings.Contains(cmd, ".vrooli/bin/vrooli") || !strings.Contains(cmd, "setup") || !strings.Contains(cmd, "--environment") || !strings.Contains(cmd, "production") {
		t.Fatalf("expected config.apply preview to run production setup with the deployment-local binary: %s", got[execplan.OpConfigApply].Command)
	}
	if cmd := got[execplan.OpReleaseStage].Command; !strings.Contains(cmd, "release_manifest_missing") && !strings.Contains(cmd, "not part of a built release") {
		t.Fatalf("a bare bundle must render the staging refusal instead of a shell extraction: %s", cmd)
	}
	for _, step := range plan {
		if strings.Contains(step.Command, "-C '/root/Vrooli'") || strings.Contains(step.Command, "rm -rf") {
			t.Fatalf("step %s must never extract into or delete the live workdir: %s", step.ID, step.Command)
		}
	}
}

func TestBuildSetupPlanRequiresBundlePath(t *testing.T) {
	// [REQ:STC-P0-004] Install bundle and run setup - error case
	manifest := setupPlanManifest("203.0.113.10", 22, "root", "/root/Vrooli", "test-scenario")
	for _, path := range []string{"", "   "} {
		_, err := vps.BuildSetupPlan(manifest, path)
		if err == nil {
			t.Fatalf("expected error for bundle_path %q", path)
		}
		if !strings.Contains(err.Error(), "bundle_path") {
			t.Errorf("expected error about bundle_path, got: %v", err)
		}
	}
}

func TestBuildSetupPlanStepOrder(t *testing.T) {
	// [REQ:STC-P0-004] Plan steps must execute in the correct order
	bundlePath := writeFakeBundle(t)
	manifest := setupPlanManifest("203.0.113.10", 22, "root", "/root/Vrooli", "test-scenario")

	plan, err := vps.BuildSetupPlan(manifest, bundlePath)
	if err != nil {
		t.Fatalf("vps.BuildSetupPlan: %v", err)
	}
	expectedOrder := []string{execplan.OpHostPrepare, execplan.OpDataInventory, execplan.OpReleaseDeliver, execplan.OpReleaseVerify, execplan.OpReleaseStage, execplan.OpConfigApply}
	if len(plan) != len(expectedOrder) {
		t.Fatalf("expected %d steps, got %d: %+v", len(expectedOrder), len(plan), plan)
	}
	for i, step := range plan {
		if step.ID != expectedOrder[i] {
			t.Errorf("step %d: expected ID %q, got %q", i, expectedOrder[i], step.ID)
		}
		if step.Title == "" || step.Description == "" || step.Command == "" {
			t.Errorf("step %d (%s): missing title, description or command", i, step.ID)
		}
	}
}

func TestBuildSetupPlanUsesDeploymentLocalNativeCLIForSetup(t *testing.T) {
	bundlePath := writeFakeBundle(t)
	workdir := "/root/Vrooli"
	manifest := setupPlanManifest("203.0.113.10", 22, "root", workdir, "landing-page-business-suite")
	plan, err := vps.BuildSetupPlan(manifest, bundlePath)
	if err != nil {
		t.Fatalf("vps.BuildSetupPlan: %v", err)
	}
	var setupCmd string
	for _, step := range plan {
		if step.ID == execplan.OpConfigApply {
			setupCmd = step.Command
		}
	}
	wantBinary := shellutil.RemoteVrooliPath(workdir)
	if !strings.Contains(setupCmd, wantBinary) || !strings.Contains(setupCmd, "setup") || !strings.Contains(setupCmd, "--environment") || !strings.Contains(setupCmd, "production") {
		t.Fatalf("expected setup command to use deployment-local binary, got: %s", setupCmd)
	}
}

func TestBuildSetupPlanSSHConfig(t *testing.T) {
	// [REQ:STC-P0-004] Plan should use correct SSH configuration from manifest
	bundlePath := writeFakeBundle(t)
	tests := []struct {
		name, host, user string
		port             int
	}{
		{name: "standard config", host: "192.168.1.100", port: 22, user: "root"},
		{name: "custom port", host: "example.com", port: 2222, user: "deploy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := setupPlanManifest(tt.host, tt.port, tt.user, "/opt/vrooli", "test")
			plan, err := vps.BuildSetupPlan(manifest, bundlePath)
			if err != nil {
				t.Fatalf("vps.BuildSetupPlan: %v", err)
			}
			for _, step := range plan {
				if !strings.HasPrefix(step.Command, "ssh ") && !strings.HasPrefix(step.Command, "scp ") {
					continue
				}
				if !strings.Contains(step.Command, tt.host) || !strings.Contains(step.Command, tt.user+"@") {
					t.Errorf("step %s: expected %s@%s in command: %s", step.ID, tt.user, tt.host, step.Command)
				}
			}
		})
	}
}

func TestBuildSetupPlanAutohealConfig(t *testing.T) {
	// [REQ:STC-P0-010] Autoheal scope configured for mini-Vrooli
	bundlePath := writeFakeBundle(t)
	manifest := setupPlanManifest("203.0.113.10", 22, "root", "/root/Vrooli", "my-app")
	manifest.Dependencies = domain.ManifestDependencies{
		Scenarios: []string{"my-app", "vrooli-autoheal"},
		Resources: []string{"postgres", "redis"},
	}
	manifest.Bundle = domain.ManifestBundle{IncludeAutoheal: true}
	plan, err := vps.BuildSetupExecutablePlan(context.Background(), manifest, bundlePath)
	if err != nil {
		t.Fatalf("BuildSetupExecutablePlan: %v", err)
	}
	config := plan.Action(execplan.OpConfigApply)
	if config == nil {
		t.Fatal("expected config.apply action")
	}
	if config.Inputs["autoheal"] != "true" || config.Inputs["autoheal_scope_path"] != "/root/Vrooli/.vrooli/cloud/autoheal-scope.json" {
		t.Fatalf("config.apply must declare the typed autoheal scope delivery: %v", config.Inputs)
	}
	if config.Inputs["resources"] != "postgres,redis" || config.Inputs["scenarios"] != "my-app,vrooli-autoheal" {
		t.Fatalf("config.apply must carry the dependency snapshot: %v", config.Inputs)
	}
	for _, want := range []string{`"scenario_id": "my-app"`, "postgres", "redis", "vrooli-autoheal"} {
		if !strings.Contains(string(vps.AutohealScopeJSON(manifest)), want) {
			t.Errorf("autoheal scope document should contain %q", want)
		}
	}
}

func TestBuildFullPlanBindsPreservePathsThroughRecordedMappingsFirst(t *testing.T) {
	// [REQ:STC-P0-028] Legacy conversion: recorded mappings win, the heuristic carries only unmapped paths
	bundlePath := writeFakeBundle(t)
	manifest := setupPlanManifest("203.0.113.10", 22, "root", "/root/Vrooli", "landing-page-business-suite")
	manifest.Target.VPS.PreservePaths = []string{"scenarios/landing-page-business-suite/api/uploads", "scenarios/landing-page-business-suite/data"}
	manifest.Bundle = domain.ManifestBundle{Scenarios: []string{"landing-page-business-suite", "vrooli-autoheal"}}

	closure := &domain.Closure{SchemaVersion: "1", ScenarioID: "landing-page-business-suite", Digest: "sha256:fixture-lpbs", PersistentData: []domain.ClosurePersistentData{{ID: "uploads", Owner: "landing-page-business-suite", Binding: "dir:api/uploads", MigrationOwner: "scenario", DeclaredBy: "scenario:landing-page-business-suite"}}}
	plan, err := vps.CompilePlan(context.Background(), vps.PlanRequest{Manifest: manifest, BundlePath: bundlePath, Closure: closure, Scope: execplan.ScopeFull, Observations: execplan.Observations{PersistentDataBindings: []string{"uploads=landing-page-business-suite/api/uploads"}}})
	if err != nil {
		t.Fatalf("CompilePlan: %v", err)
	}
	inventory := plan.Action(execplan.OpDataInventory)
	activate := plan.Action(execplan.OpReleaseActivate)
	if inventory == nil || activate == nil {
		t.Fatalf("expected data.inventory and release.activate actions: %v", plan.ActionIDs())
	}
	if inventory.Inputs["legacy_preserve"] != "scenarios/landing-page-business-suite/api/uploads,scenarios/landing-page-business-suite/data" {
		t.Fatalf("expected preserve paths as data.inventory input, got %q", inventory.Inputs["legacy_preserve"])
	}
	if activate.Inputs["data_bindings"] != "uploads=landing-page-business-suite/api/uploads" {
		t.Fatalf("recorded mapping must bind the uploads directory: %q", activate.Inputs["data_bindings"])
	}
	if activate.Inputs["legacy_carry"] != "landing-page-business-suite/data" {
		t.Fatalf("only the unmapped preserve path may be carried by the heuristic: %q", activate.Inputs["legacy_carry"])
	}
	if activate.Inputs["legacy_root"] != "/root/Vrooli" {
		t.Fatalf("activation must know the in-place root it adopts legacy data from: %q", activate.Inputs["legacy_root"])
	}
	for _, action := range plan.Actions {
		for key, value := range action.Inputs {
			if strings.Contains(value, "rm -rf") || strings.Contains(value, "&&") {
				t.Fatalf("action %s input %s carries a shell fragment: %q", action.ID, key, value)
			}
		}
	}
}
