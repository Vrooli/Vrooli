package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestVPSInspectPlanAndApply(t *testing.T) {
	// [REQ:STC-P0-006] Remote status and logs retrieval
	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS: &ManifestVPS{
				Host:    "203.0.113.10",
				Port:    22,
				User:    "root",
				Workdir: "/root/Vrooli",
			},
		},
		Scenario: ManifestScenario{ID: "landing-page-business-suite"},
		Dependencies: ManifestDependencies{
			Scenarios: []string{"landing-page-business-suite"},
			Resources: []string{"postgres"},
		},
		Bundle: ManifestBundle{IncludePackages: true, IncludeAutoheal: true},
		Ports:  ManifestPorts{UI: 3000, API: 3001, WS: 3002},
		Edge:   ManifestEdge{Domain: "example.com", Caddy: ManifestCaddy{Enabled: true}},
	}

	opts, err := (VPSInspectOptionsIn{TailLines: 123}).Normalize(manifest)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}

	plan, err := BuildVPSInspectPlan(manifest, opts)
	if err != nil {
		t.Fatalf("BuildVPSInspectPlan: %v", err)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("expected 3 steps, got: %+v", plan)
	}
	if !strings.Contains(plan.Steps[2].Command, "--tail 123") {
		t.Fatalf("expected tail override, got: %s", plan.Steps[2].Command)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	runner := fakeSSHRunner{responses: map[string]SSHResult{
		"cd '/root/Vrooli' && vrooli scenario status 'landing-page-business-suite' --json": {ExitCode: 0, Stdout: `{"status":"healthy"}`},
		"cd '/root/Vrooli' && vrooli resource status --json":                               {ExitCode: 0, Stdout: `{"resources":[]}`},
		"cd '/root/Vrooli' && vrooli scenario logs 'landing-page-business-suite' --tail 123": {ExitCode: 0, Stdout: "hello\nworld"},
	}}

	result := RunVPSInspect(ctx, manifest, opts, runner)
	if !result.OK {
		t.Fatalf("expected OK, got: %+v", result)
	}
	if string(result.ScenarioStatus) != `{"status":"healthy"}` {
		t.Fatalf("unexpected scenario status: %s", string(result.ScenarioStatus))
	}
	if !strings.Contains(result.ScenarioLogs, "hello") {
		t.Fatalf("unexpected logs: %q", result.ScenarioLogs)
	}
}
