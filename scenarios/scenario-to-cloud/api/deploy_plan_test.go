package main

import (
	"context"
	"strings"
	"testing"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/vps"
)

func TestBuildDeployPlanRoutesThroughTargetOwnerVerbs(t *testing.T) {
	// [REQ:STC-P0-005] Start resources + scenario and verify HTTPS
	// [REQ:STC-P0-011] Caddy + Let's Encrypt configured and verified
	// [REQ:STC-P0-028] Activation and routing are target owner verbs
	manifest := domain.CloudManifest{
		Version: "1.0.0",
		Target: domain.ManifestTarget{
			Type: "vps",
			VPS: &domain.ManifestVPS{
				Host:    "203.0.113.10",
				Port:    22,
				User:    "root",
				Workdir: "/root/Vrooli",
			},
		},
		Scenario: domain.ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: domain.ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite", "vrooli-autoheal"},
			Resources: []string{"postgres"},
		},
		Bundle: domain.ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  domain.ManifestPorts{"ui": 3000, "api": 3001, "metrics": 3002},
		Edge:   domain.ManifestEdge{Domain: "example.com", Caddy: domain.ManifestCaddy{Enabled: true}},
	}

	plan, err := vps.BuildDeployPlan(manifest)
	if err != nil {
		t.Fatalf("vps.BuildDeployPlan: %v", err)
	}

	got := map[string]domain.VPSPlanStep{}
	for _, step := range plan {
		got[step.ID] = step
	}
	for _, id := range []string{execplan.OpRuntimeStartDeps, execplan.OpWorkloadStop, execplan.OpReleaseActivate, execplan.OpEdgeRouteApply, execplan.OpVerifyReadiness} {
		if _, ok := got[id]; !ok {
			t.Fatalf("expected action %s in runtime plan: %+v", id, plan)
		}
	}
	if _, ok := got[execplan.OpWorkloadStart]; ok {
		t.Fatalf("runtime plan activates through the target owner; workload.start belongs to the start scope: %+v", plan)
	}
	if cmd := got[execplan.OpEdgeRouteApply].Command; !strings.Contains(cmd, "route-apply") || !strings.Contains(cmd, "--spec") || !strings.Contains(cmd, "b64:") {
		t.Fatalf("edge routing must be the target owner's route-apply verb with a typed spec: %s", cmd)
	}
	// An ad hoc runtime plan has no built release beside it: the activation
	// preview renders the owner's refusal instead of a shell overlay, and the
	// typed inputs still pin the listener ports for the owner verb.
	if cmd := got[execplan.OpReleaseActivate].Command; !strings.Contains(cmd, "not part of a built release") || strings.Contains(cmd, "export ") {
		t.Fatalf("activation preview without a built release must render the refusal, never an exported environment: %s", cmd)
	}
	executable, err := vps.BuildDeployExecutablePlan(context.Background(), manifest, execplan.ScopeRuntime)
	if err != nil {
		t.Fatalf("BuildDeployExecutablePlan: %v", err)
	}
	if activate := executable.Action(execplan.OpReleaseActivate); activate == nil || activate.Inputs["ports"] != "api=3001,metrics=3002,ui=3000" || activate.Inputs["strategy"] != execplan.StrategyMaintenance {
		t.Fatalf("activation must carry the pinned ports and the selected strategy: %+v", executable.Action(execplan.OpReleaseActivate))
	}
	if cmd := got[execplan.OpWorkloadStop].Command; !strings.Contains(cmd, "process.stop.scoped") || strings.Contains(cmd, "pkill") {
		t.Fatalf("workload.stop must be the scoped lifecycle stop: %s", cmd)
	}
	if cmd := got[execplan.OpRuntimeStartDeps].Command; !strings.Contains(cmd, "resource") || !strings.Contains(cmd, "postgres") {
		t.Fatalf("dependencies must start through the resource owner: %s", cmd)
	}
	if cmd := got[execplan.OpVerifyReadiness].Command; !strings.Contains(cmd, "https://example.com/health") || !strings.Contains(cmd, "release") || !strings.Contains(cmd, "list") {
		t.Fatalf("readiness must prove the active pointer and the public path: %s", cmd)
	}
	for _, step := range plan {
		for _, fragment := range []string{"&& rm", "pkill", "kill -9", "ufw allow", "apt-get"} {
			if strings.Contains(step.Command, fragment) {
				t.Fatalf("step %s preview carries a private shell path %q: %s", step.ID, fragment, step.Command)
			}
		}
	}
}
