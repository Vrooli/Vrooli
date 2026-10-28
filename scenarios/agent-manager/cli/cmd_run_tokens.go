// This file reports weighted non-cache token spend for a run and its children.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliutil"
)

// Shared operator heuristic; actual billing is owned separately.
var defaultTierWeights = agentcatalog.DefaultTokenWeights()

// tokenRow is one run's contribution to weighted spend.
type tokenRow struct {
	RunID           string   `json:"run_id"`
	Status          string   `json:"status"`
	Model           string   `json:"model"`
	TokensKnown     bool     `json:"tokens_known"`
	NonCacheTokens  int64    `json:"non_cache_tokens"`
	CacheReadTokens int64    `json:"cache_read_tokens"`
	Weight          *float64 `json:"weight"`
	Weighted        *float64 `json:"weighted_tokens"`
}

// tokenReport is the weighted spend of one run and, optionally, its children.
type tokenReport struct {
	PolicyVersion string             `json:"weight_policy_version"`
	Rows          []tokenRow         `json:"runs"`
	Weights       map[string]float64 `json:"weights"`
	TotalWeighted float64            `json:"total_weighted_tokens"`
	TotalNonCache int64              `json:"total_non_cache_tokens"`
	Complete      bool               `json:"weighted_complete"`
	WeightUnknown []string           `json:"weight_unknown_runs"`
	Unknown       []string           `json:"usage_unknown_runs"`
}

// parseTierWeights reads "sol=10,luna=1" into tier weights.
func parseTierWeights(raw string) (map[string]float64, error) {
	return agentcatalog.ParseTokenWeights(raw, defaultTierWeights)
}

func (r *tokenReport) add(row tokenRow, weights map[string]float64) {
	weight, known := agentcatalog.TokenWeight(row.Model, weights)
	if known {
		row.Weight = &weight
		weighted := float64(row.NonCacheTokens) * weight
		row.Weighted = &weighted
		r.TotalWeighted += weighted
	}
	if !row.TokensKnown {
		r.Unknown = append(r.Unknown, row.RunID)
	}
	if !known {
		r.WeightUnknown = append(r.WeightUnknown, row.RunID)
	}
	if !row.TokensKnown || !known {
		r.Complete = false
	}
	r.Rows = append(r.Rows, row)
	r.TotalNonCache += row.NonCacheTokens
}

// weightedTokens is the one token-weighting path: `run tokens` and the epoch
// check's spend trigger both read it.
func (a *App) weightedTokens(ids []string, weights map[string]float64) (*tokenReport, error) {
	report := &tokenReport{PolicyVersion: agentcatalog.TokenWeightPolicyVersion, Weights: weights, Complete: true, Unknown: []string{}, WeightUnknown: []string{}}
	for _, runID := range ids {
		accounting, err := a.services.Runs.Accounting(runID)
		if err != nil {
			return nil, fmt.Errorf("accounting for %s: %w", runID, err)
		}
		_, run, _ := a.services.Runs.Get(runID)
		status := ""
		if run != nil {
			status = formatEnumValue(run.Status, "RUN_STATUS_", "_")
		}
		report.add(tokenRow{RunID: runID, Status: status, Model: accounting.GetModel(), TokensKnown: accounting.GetTokensKnown(),
			NonCacheTokens: accounting.GetNonCacheTokens(), CacheReadTokens: accounting.GetCacheReadTokens()}, weights)
	}
	return report, nil
}

// runTokens prints weighted non-cache tokens for a run, and with --children
// for its direct child runs too: the orchestrator's and supervisor's spend unit.
func (a *App) runTokens(args []string) error {
	fs := flag.NewFlagSet("run tokens", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	children := fs.Bool("children", false, "Include the run's direct child runs")
	rawWeights := fs.String("weights", "", "Tier weights by model-name substring, e.g. sol=10,luna=1 (defaults shown)")
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id, args = args[0], args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run tokens <id> [--children] [--weights sol=10,luna=1] [--json]")
	}
	weights, err := parseTierWeights(*rawWeights)
	if err != nil {
		return err
	}
	ids := []string{id}
	if *children {
		_, kids, err := a.services.Runs.ListChildren(id, 1000)
		if err != nil {
			return err
		}
		for _, kid := range kids {
			ids = append(ids, kid.GetId())
		}
	}
	report, err := a.weightedTokens(ids, weights)
	if err != nil {
		return err
	}
	if *jsonOutput {
		data, err := json.Marshal(report)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(data)
		return nil
	}
	for _, row := range report.Rows {
		known := ""
		if !row.TokensKnown {
			known = " (usage not final)"
		}
		if row.Weight == nil {
			fmt.Printf("%s  %-10s %-12s non-cache=%d weight=unknown\n", row.RunID, row.Status, row.Model, row.NonCacheTokens)
			continue
		}
		fmt.Printf("%s  %-10s %-12s non-cache=%d cache-read=%d x%g = %.0f%s\n", row.RunID, row.Status, row.Model, row.NonCacheTokens, row.CacheReadTokens, *row.Weight, *row.Weighted, known)
	}
	fmt.Printf("Known subtotal weighted non-cache tokens: %.0f (non-cache %d across %d run(s))\n", report.TotalWeighted, report.TotalNonCache, len(report.Rows))
	return nil
}
