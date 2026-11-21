package vps

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reach/sshadapter"
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

// inspectProbes are the three typed reads an inspection issues.
func inspectProbes(manifest domain.CloudManifest, opts InspectOptions) []probe {
	targetScenario := manifest.Scenario.ID
	return []probe{
		verb("scenario_status", "scenario status", targetScenario, "--json"),
		verb("resource_status", "resource status", "--json"),
		verb("scenario_logs", "scenario logs", targetScenario, "--tail", strconv.Itoa(opts.TailLines)),
	}
}

// BuildInspectPlan creates a plan of steps for VPS inspection. The displayed
// command is the typed argv rendered through the SSH adapter's quoting rule;
// it is a preview, never what is executed.
func BuildInspectPlan(manifest domain.CloudManifest, opts InspectOptions) (InspectPlan, error) {
	workdir := manifest.Target.VPS.Workdir
	titles := map[string][2]string{
		"scenario_status": {"Scenario status", "Fetch the typed scenario status for the target scenario."},
		"resource_status": {"Resource status", "Fetch the typed resource status for the mini install."},
		"scenario_logs":   {"Scenario logs", "Fetch bounded logs output for the target scenario."},
	}
	var steps []domain.VPSPlanStep
	for _, p := range inspectProbes(manifest, opts) {
		steps = append(steps, domain.VPSPlanStep{
			ID:          p.id,
			Title:       titles[p.id][0],
			Description: titles[p.id][1],
			Command:     sshadapter.RemoteCommand(workdir, p.cmd.Argv()[1:]),
		})
	}
	return InspectPlan{Steps: steps}, nil
}

// RunInspect performs VPS inspection through the prober.
func RunInspect(ctx context.Context, manifest domain.CloudManifest, opts InspectOptions, prober Prober) InspectResult {
	answers := make(map[string]reach.Result, 3)
	for _, p := range inspectProbes(manifest, opts) {
		res, err := prober.Reach.Exec(ctx, prober.Target, p.cmd)
		if err != nil {
			return InspectResult{OK: false, Error: coalesce(res.Stderr, err.Error()), Timestamp: time.Now().UTC().Format(time.RFC3339)}
		}
		answers[p.id] = res
	}
	return InspectResult{
		OK:             true,
		ScenarioStatus: json.RawMessage(answers["scenario_status"].Stdout),
		ResourceStatus: json.RawMessage(answers["resource_status"].Stdout),
		ScenarioLogs:   answers["scenario_logs"].Stdout,
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
