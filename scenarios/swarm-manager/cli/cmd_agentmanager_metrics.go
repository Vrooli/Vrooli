package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/cli-core/cliutil"
)

type SandboxAdoption struct {
	Breakdown             []SandboxAdoptionRow `json:"breakdown"`
	RunsWithProvenance    float64              `json:"runs_with_provenance"`
	RunsWithoutProvenance float64              `json:"runs_without_provenance"`
	AttributionRate       float64              `json:"attribution_rate"`
}

type SandboxAdoptionRow struct {
	RunMode      string  `json:"run_mode"`
	SandboxMode  string  `json:"sandbox_mode"`
	ManualReview string  `json:"manual_review"`
	Count        float64 `json:"count"`
}

func (a *App) cmdAgentManagerSandboxAdoption(args []string) error {
	fs := flag.NewFlagSet("agent-manager metrics sandbox-adoption", flag.ContinueOnError)
	format := fs.String("format", "human", "Output format: human or json")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := agentManagerMetrics()
	if err != nil {
		return err
	}
	adoption := parseSandboxAdoptionMetrics(body)
	if *jsonOut || *format == "json" {
		encoded, err := json.MarshalIndent(adoption, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
	printSandboxAdoptionHuman(adoption)
	return nil
}

func agentManagerMetrics() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	base, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return nil, fmt.Errorf("resolve agent-manager URL: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/metrics", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("agent-manager /metrics returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func parseSandboxAdoptionMetrics(body []byte) SandboxAdoption {
	var out SandboxAdoption
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "agent_manager_sandbox_adoption_total{"):
			if row := parseSandboxAdoptionRow(line); row != nil {
				out.Breakdown = append(out.Breakdown, *row)
			}
		case strings.HasPrefix(line, "agent_manager_runs_with_provenance_total"):
			out.RunsWithProvenance = prometheusValue(line)
		case strings.HasPrefix(line, "agent_manager_runs_without_provenance_total"):
			out.RunsWithoutProvenance = prometheusValue(line)
		}
	}
	if total := out.RunsWithProvenance + out.RunsWithoutProvenance; total > 0 {
		out.AttributionRate = out.RunsWithProvenance / total
	}
	sort.Slice(out.Breakdown, func(i, j int) bool {
		a, b := out.Breakdown[i], out.Breakdown[j]
		return a.RunMode+a.SandboxMode+a.ManualReview < b.RunMode+b.SandboxMode+b.ManualReview
	})
	return out
}

func parseSandboxAdoptionRow(line string) *SandboxAdoptionRow {
	open, close := strings.Index(line, "{"), strings.Index(line, "}")
	if open < 0 || close < open {
		return nil
	}
	row := &SandboxAdoptionRow{}
	for _, pair := range strings.Split(line[open+1:close], ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			continue
		}
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		switch strings.TrimSpace(parts[0]) {
		case "run_mode":
			row.RunMode = value
		case "sandbox_mode":
			row.SandboxMode = value
		case "manual_review":
			row.ManualReview = value
		}
	}
	row.Count = prometheusValue(line[close+1:])
	return row
}

func prometheusValue(value string) float64 {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0
	}
	var result float64
	_, _ = fmt.Sscanf(fields[len(fields)-1], "%f", &result)
	return result
}

func printSandboxAdoptionHuman(a SandboxAdoption) {
	fmt.Println("Sandbox-default rollout adoption:")
	for _, row := range a.Breakdown {
		fmt.Printf("  %s / %s / %s: %.0f\n", row.RunMode, row.SandboxMode, row.ManualReview, row.Count)
	}
	fmt.Printf("Provenance attribution: %.1f%% (%.0f with, %.0f without)\n", a.AttributionRate*100, a.RunsWithProvenance, a.RunsWithoutProvenance)
}
