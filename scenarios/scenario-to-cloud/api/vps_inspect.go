package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type VPSInspectRequest struct {
	Manifest CloudManifest       `json:"manifest"`
	Options  VPSInspectOptionsIn `json:"options,omitempty"`
}

type VPSInspectOptionsIn struct {
	TailLines int `json:"tail_lines,omitempty"`
}

type VPSInspectOptions struct {
	TailLines int `json:"tail_lines"`
}

func (in VPSInspectOptionsIn) Normalize(manifest CloudManifest) (VPSInspectOptions, error) {
	lines := in.TailLines
	if lines == 0 {
		lines = 200
	}
	if lines < 1 || lines > 2000 {
		return VPSInspectOptions{}, fmt.Errorf("tail_lines must be between 1 and 2000")
	}
	_ = manifest
	return VPSInspectOptions{TailLines: lines}, nil
}

type VPSInspectPlan struct {
	Steps []VPSPlanStep `json:"steps"`
}

type VPSInspectResult struct {
	OK             bool            `json:"ok"`
	ScenarioStatus json.RawMessage `json:"scenario_status,omitempty"`
	ResourceStatus json.RawMessage `json:"resource_status,omitempty"`
	ScenarioLogs   string          `json:"scenario_logs,omitempty"`
	Error          string          `json:"error,omitempty"`
	Timestamp      string          `json:"timestamp"`
}

func BuildVPSInspectPlan(manifest CloudManifest, opts VPSInspectOptions) (VPSInspectPlan, error) {
	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID

	steps := []VPSPlanStep{
		{
			ID:          "scenario_status",
			Title:       "Scenario status",
			Description: "Fetch `vrooli scenario status --json` for the target scenario.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && vrooli scenario status %s --json", shellQuoteSingle(workdir), shellQuoteSingle(targetScenario))),
		},
		{
			ID:          "resource_status",
			Title:       "Resource status",
			Description: "Fetch `vrooli resource status --json` for the mini install.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && vrooli resource status --json", shellQuoteSingle(workdir))),
		},
		{
			ID:          "scenario_logs",
			Title:       "Scenario logs",
			Description: "Fetch bounded logs output for the target scenario.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && vrooli scenario logs %s --tail %d", shellQuoteSingle(workdir), shellQuoteSingle(targetScenario), opts.TailLines)),
		},
	}

	return VPSInspectPlan{Steps: steps}, nil
}

func RunVPSInspect(ctx context.Context, manifest CloudManifest, opts VPSInspectOptions, sshRunner SSHRunner) VPSInspectResult {
	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID

	statusRes, err := sshRunner.Run(ctx, cfg, fmt.Sprintf("cd %s && vrooli scenario status %s --json", shellQuoteSingle(workdir), shellQuoteSingle(targetScenario)))
	if err != nil {
		return VPSInspectResult{OK: false, Error: coalesce(statusRes.Stderr, err.Error()), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	resourceRes, err := sshRunner.Run(ctx, cfg, fmt.Sprintf("cd %s && vrooli resource status --json", shellQuoteSingle(workdir)))
	if err != nil {
		return VPSInspectResult{OK: false, Error: coalesce(resourceRes.Stderr, err.Error()), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	logsRes, err := sshRunner.Run(ctx, cfg, fmt.Sprintf("cd %s && vrooli scenario logs %s --tail %d", shellQuoteSingle(workdir), shellQuoteSingle(targetScenario), opts.TailLines))
	if err != nil {
		return VPSInspectResult{OK: false, Error: coalesce(logsRes.Stderr, err.Error()), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return VPSInspectResult{
		OK:             true,
		ScenarioStatus: json.RawMessage(statusRes.Stdout),
		ResourceStatus: json.RawMessage(resourceRes.Stdout),
		ScenarioLogs:   logsRes.Stdout,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
	}
}

func coalesce(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (o VPSInspectOptions) String() string {
	return "tail_lines=" + strconv.Itoa(o.TailLines)
}
