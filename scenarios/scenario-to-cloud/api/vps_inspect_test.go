package main

import (
	"context"
	"errors"
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
		Ports:  ManifestPorts{"ui": 3000, "api": 3001, "ws": 3002},
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

	// Commands now include PATH setup for SSH non-interactive sessions
	pathPrefix := `export PATH="$HOME/.local/bin:$HOME/bin:/usr/local/bin:$PATH" && `
	runner := &FakeSSHRunner{Responses: map[string]SSHResult{
		pathPrefix + "cd '/root/Vrooli' && vrooli scenario status 'landing-page-business-suite' --json":   {ExitCode: 0, Stdout: `{"status":"healthy"}`},
		pathPrefix + "cd '/root/Vrooli' && vrooli resource status --json":                                 {ExitCode: 0, Stdout: `{"resources":[]}`},
		pathPrefix + "cd '/root/Vrooli' && vrooli scenario logs 'landing-page-business-suite' --tail 123": {ExitCode: 0, Stdout: "hello\nworld"},
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

func TestVPSInspectOptionsNormalize(t *testing.T) {
	// [REQ:STC-P0-006] Options validation for inspect
	manifest := CloudManifest{
		Version:  "1.0.0",
		Scenario: ManifestScenario{ID: "test"},
	}

	tests := []struct {
		name      string
		in        VPSInspectOptionsIn
		wantLines int
		wantErr   bool
	}{
		{
			name:      "default when zero",
			in:        VPSInspectOptionsIn{TailLines: 0},
			wantLines: 200,
		},
		{
			name:      "custom value within range",
			in:        VPSInspectOptionsIn{TailLines: 500},
			wantLines: 500,
		},
		{
			name:      "minimum valid",
			in:        VPSInspectOptionsIn{TailLines: 1},
			wantLines: 1,
		},
		{
			name:      "maximum valid",
			in:        VPSInspectOptionsIn{TailLines: 2000},
			wantLines: 2000,
		},
		{
			name:    "below minimum",
			in:      VPSInspectOptionsIn{TailLines: -1},
			wantErr: true,
		},
		{
			name:    "above maximum",
			in:      VPSInspectOptionsIn{TailLines: 2001},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := tt.in.Normalize(manifest)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got opts: %+v", opts)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if opts.TailLines != tt.wantLines {
				t.Errorf("TailLines = %d, want %d", opts.TailLines, tt.wantLines)
			}
		})
	}
}

func TestBuildVPSInspectPlanSteps(t *testing.T) {
	// [REQ:STC-P0-006] Inspect plan contains correct steps
	manifest := CloudManifest{
		Version: "1.0.0",
		Target: ManifestTarget{
			Type: "vps",
			VPS: &ManifestVPS{
				Host:    "10.0.0.1",
				Port:    22,
				User:    "admin",
				Workdir: "/opt/vrooli",
			},
		},
		Scenario: ManifestScenario{ID: "my-app"},
	}

	opts := VPSInspectOptions{TailLines: 50}
	plan, err := BuildVPSInspectPlan(manifest, opts)
	if err != nil {
		t.Fatalf("BuildVPSInspectPlan: %v", err)
	}

	expectedSteps := []struct {
		id       string
		contains string
	}{
		{"scenario_status", "vrooli scenario status"},
		{"resource_status", "vrooli resource status"},
		{"scenario_logs", "vrooli scenario logs"},
	}

	if len(plan.Steps) != len(expectedSteps) {
		t.Fatalf("expected %d steps, got %d", len(expectedSteps), len(plan.Steps))
	}

	for i, exp := range expectedSteps {
		step := plan.Steps[i]
		if step.ID != exp.id {
			t.Errorf("step %d: expected ID %q, got %q", i, exp.id, step.ID)
		}
		if !strings.Contains(step.Command, exp.contains) {
			t.Errorf("step %d: expected command to contain %q, got: %s", i, exp.contains, step.Command)
		}
		if step.Title == "" {
			t.Errorf("step %d: missing title", i)
		}
		if step.Description == "" {
			t.Errorf("step %d: missing description", i)
		}
	}

	// Verify scenario ID is properly quoted in commands
	if !strings.Contains(plan.Steps[0].Command, "'my-app'") {
		t.Errorf("expected quoted scenario ID in status command: %s", plan.Steps[0].Command)
	}
	if !strings.Contains(plan.Steps[2].Command, "'my-app'") {
		t.Errorf("expected quoted scenario ID in logs command: %s", plan.Steps[2].Command)
	}

	// Verify tail lines are in logs command
	if !strings.Contains(plan.Steps[2].Command, "--tail 50") {
		t.Errorf("expected --tail 50 in logs command: %s", plan.Steps[2].Command)
	}
}

func TestRunVPSInspectHandlesErrors(t *testing.T) {
	// [REQ:STC-P0-006] Inspect handles SSH errors gracefully
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
		Scenario: ManifestScenario{ID: "test-app"},
	}
	opts := VPSInspectOptions{TailLines: 100}

	tests := []struct {
		name          string
		failOn        string
		expectedError string
	}{
		{
			name:          "status command fails",
			failOn:        "status",
			expectedError: "status failed",
		},
		{
			name:          "resource command fails",
			failOn:        "resource",
			expectedError: "resource failed",
		},
		{
			name:          "logs command fails",
			failOn:        "logs",
			expectedError: "logs failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			pathPrefix := `export PATH="$HOME/.local/bin:$HOME/bin:/usr/local/bin:$PATH" && `
			responses := map[string]SSHResult{
				pathPrefix + "cd '/root/Vrooli' && vrooli scenario status 'test-app' --json": {ExitCode: 0, Stdout: `{}`},
				pathPrefix + "cd '/root/Vrooli' && vrooli resource status --json":           {ExitCode: 0, Stdout: `{}`},
				pathPrefix + "cd '/root/Vrooli' && vrooli scenario logs 'test-app' --tail 100": {ExitCode: 0, Stdout: "logs"},
			}
			errs := map[string]error{}

			// Set up failure based on test case
			switch tt.failOn {
			case "status":
				errs[pathPrefix+"cd '/root/Vrooli' && vrooli scenario status 'test-app' --json"] = errors.New(tt.expectedError)
			case "resource":
				errs[pathPrefix+"cd '/root/Vrooli' && vrooli resource status --json"] = errors.New(tt.expectedError)
			case "logs":
				errs[pathPrefix+"cd '/root/Vrooli' && vrooli scenario logs 'test-app' --tail 100"] = errors.New(tt.expectedError)
			}

			runner := &FakeSSHRunner{Responses: responses, Errs: errs}
			result := RunVPSInspect(ctx, manifest, opts, runner)

			if result.OK {
				t.Fatal("expected not OK")
			}
			if result.Error == "" {
				t.Fatal("expected error message")
			}
		})
	}
}

func TestRunVPSInspectTimestamp(t *testing.T) {
	// [REQ:STC-P0-006] Inspect result includes valid timestamp
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

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
		Scenario: ManifestScenario{ID: "test"},
	}
	opts := VPSInspectOptions{TailLines: 10}

	pathPrefix := `export PATH="$HOME/.local/bin:$HOME/bin:/usr/local/bin:$PATH" && `
	runner := &FakeSSHRunner{Responses: map[string]SSHResult{
		pathPrefix + "cd '/root/Vrooli' && vrooli scenario status 'test' --json": {ExitCode: 0, Stdout: `{}`},
		pathPrefix + "cd '/root/Vrooli' && vrooli resource status --json":        {ExitCode: 0, Stdout: `{}`},
		pathPrefix + "cd '/root/Vrooli' && vrooli scenario logs 'test' --tail 10": {ExitCode: 0, Stdout: ""},
	}}

	// Subtract 1 second from before to allow for rounding differences in RFC3339 format
	before := time.Now().UTC().Add(-time.Second)
	result := RunVPSInspect(ctx, manifest, opts, runner)
	after := time.Now().UTC().Add(time.Second)

	if result.Timestamp == "" {
		t.Fatal("expected timestamp in result")
	}

	ts, err := time.Parse(time.RFC3339, result.Timestamp)
	if err != nil {
		t.Fatalf("invalid timestamp format: %v", err)
	}

	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %s not within expected range [%s, %s]", result.Timestamp, before.Format(time.RFC3339), after.Format(time.RFC3339))
	}
}
