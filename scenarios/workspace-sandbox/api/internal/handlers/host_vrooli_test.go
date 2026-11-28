package handlers

import (
	"strings"
	"testing"
)

func TestValidateHostVrooliScenarioRequest(t *testing.T) {
	valid := hostVrooliScenarioRequest{Action: "start", Name: "prompt-manager"}
	if err := validateHostVrooliScenarioRequest(valid); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	for _, req := range []hostVrooliScenarioRequest{
		{Action: "setup", Name: "prompt-manager"},
		{Action: "start", Name: "../prompt-manager"},
		{Action: "start", Name: "Prompt_Manager"},
	} {
		if err := validateHostVrooliScenarioRequest(req); err == nil {
			t.Fatalf("invalid request accepted: %+v", req)
		}
	}
}

func TestHostLifecycleEnvStripsSandboxIdentity(t *testing.T) {
	env := hostLifecycleEnv([]string{
		"PATH=/usr/bin",
		"VROOLI_SANDBOX_ID=sbx",
		"VROOLI_SANDBOX_MERGED=/tmp/merged",
		"VROOLI_SANDBOX_SCOPE=scenarios/demo",
		"VROOLI_ROOT=/repo",
	})
	joined := strings.Join(env, "\n")
	for _, forbidden := range []string{"VROOLI_SANDBOX_ID=", "VROOLI_SANDBOX_MERGED=", "VROOLI_SANDBOX_SCOPE="} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("sandbox env leaked into host lifecycle env: %s", joined)
		}
	}
	for _, want := range []string{"PATH=/usr/bin", "VROOLI_ROOT=/repo"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in env %s", want, joined)
		}
	}
}

func TestValidateHostVrooliCLIRefreshRequest(t *testing.T) {
	for _, kind := range []string{"scenario", "resource"} {
		valid := hostVrooliCLIRefreshRequest{
			Kind: kind, Name: "scenario-qa", RunningExecutable: "/home/agent/.vrooli/bin/scenario-qa",
		}
		if err := validateHostVrooliCLIRefreshRequest(valid); err != nil {
			t.Fatalf("valid %s request rejected: %v", kind, err)
		}
	}
	valid := hostVrooliCLIRefreshRequest{Kind: "scenario", Name: "scenario-qa", RunningExecutable: "/home/agent/.vrooli/bin/scenario-qa"}
	for _, req := range []hostVrooliCLIRefreshRequest{
		{Kind: "arbitrary", Name: "scenario-qa", RunningExecutable: valid.RunningExecutable},
		{Kind: "scenario", Name: "../scenario-qa", RunningExecutable: valid.RunningExecutable},
		{Kind: "scenario", Name: "scenario-qa", RunningExecutable: "relative/scenario-qa"},
		{Kind: "scenario", Name: "scenario-qa", RunningExecutable: "/tmp/../scenario-qa"},
	} {
		if err := validateHostVrooliCLIRefreshRequest(req); err == nil {
			t.Fatalf("invalid request accepted: %+v", req)
		}
	}
}
