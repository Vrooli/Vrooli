package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/ssh"
)

// InspectRequest is the request body for VPS inspection.
type InspectRequest struct {
	Manifest domain.CloudManifest `json:"manifest"`
	Options  InspectOptionsIn     `json:"options,omitempty"`
}

// InspectOptionsIn is the input options for inspection.
type InspectOptionsIn struct {
	TailLines int `json:"tail_lines,omitempty"`
}

// InspectOptions is the normalized inspection options.
type InspectOptions struct {
	TailLines int `json:"tail_lines"`
}

// Normalize validates and normalizes the input options.
func (in InspectOptionsIn) Normalize(manifest domain.CloudManifest) (InspectOptions, error) {
	lines := in.TailLines
	if lines == 0 {
		lines = 200
	}
	if lines < 1 || lines > 2000 {
		return InspectOptions{}, fmt.Errorf("tail_lines must be between 1 and 2000")
	}
	_ = manifest
	return InspectOptions{TailLines: lines}, nil
}

// String returns a string representation of the options.
func (o InspectOptions) String() string {
	return "tail_lines=" + strconv.Itoa(o.TailLines)
}

// InspectPlan describes the inspection steps.
type InspectPlan struct {
	Steps []domain.VPSPlanStep `json:"steps"`
}

// InspectResult is the result of a VPS inspection.
type InspectResult struct {
	OK             bool            `json:"ok"`
	ScenarioStatus json.RawMessage `json:"scenario_status,omitempty"`
	ResourceStatus json.RawMessage `json:"resource_status,omitempty"`
	ScenarioLogs   string          `json:"scenario_logs,omitempty"`
	Error          string          `json:"error,omitempty"`
	Timestamp      string          `json:"timestamp"`
}

// BuildInspectPlan creates a plan of steps for VPS inspection.
func BuildInspectPlan(manifest domain.CloudManifest, opts InspectOptions) (InspectPlan, error) {
	cfg := ssh.ConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID

	steps := []domain.VPSPlanStep{
		{
			ID:          "scenario_status",
			Title:       "Scenario status",
			Description: "Fetch `vrooli scenario status --json` for the target scenario.",
			Command:     ssh.LocalSSHCommand(cfg, ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario status %s --json", ssh.QuoteSingle(targetScenario)))),
		},
		{
			ID:          "resource_status",
			Title:       "Resource status",
			Description: "Fetch `vrooli resource status --json` for the mini install.",
			Command:     ssh.LocalSSHCommand(cfg, ssh.VrooliCommand(workdir, "vrooli resource status --json")),
		},
		{
			ID:          "scenario_logs",
			Title:       "Scenario logs",
			Description: "Fetch bounded logs output for the target scenario.",
			Command:     ssh.LocalSSHCommand(cfg, ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario logs %s --tail %d", ssh.QuoteSingle(targetScenario), opts.TailLines))),
		},
	}

	return InspectPlan{Steps: steps}, nil
}

// RunInspect performs VPS inspection.
func RunInspect(ctx context.Context, manifest domain.CloudManifest, opts InspectOptions, sshRunner ssh.Runner) InspectResult {
	cfg := ssh.ConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	targetScenario := manifest.Scenario.ID

	statusRes, err := sshRunner.Run(ctx, cfg, ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario status %s --json", ssh.QuoteSingle(targetScenario))))
	if err != nil {
		return InspectResult{OK: false, Error: coalesce(statusRes.Stderr, err.Error()), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	resourceRes, err := sshRunner.Run(ctx, cfg, ssh.VrooliCommand(workdir, "vrooli resource status --json"))
	if err != nil {
		return InspectResult{OK: false, Error: coalesce(resourceRes.Stderr, err.Error()), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	logsRes, err := sshRunner.Run(ctx, cfg, ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario logs %s --tail %d", ssh.QuoteSingle(targetScenario), opts.TailLines)))
	if err != nil {
		return InspectResult{OK: false, Error: coalesce(logsRes.Stderr, err.Error()), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return InspectResult{
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
