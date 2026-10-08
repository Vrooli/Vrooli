// This file reports weighted non-cache token spend for a run and its children.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// defaultTierWeights weight non-cache tokens by model tier (D11): premium
// supervision models cost about ten economical delivery tokens.
var defaultTierWeights = map[string]float64{"sol": 10, "luna": 1}

// tokenRow is one run's contribution to weighted spend.
type tokenRow struct {
	RunID           string  `json:"run_id"`
	Status          string  `json:"status"`
	Model           string  `json:"model"`
	TokensKnown     bool    `json:"tokens_known"`
	NonCacheTokens  int64   `json:"non_cache_tokens"`
	CacheReadTokens int64   `json:"cache_read_tokens"`
	Weight          float64 `json:"weight"`
	Weighted        float64 `json:"weighted_tokens"`
}

// tokenReport is the weighted spend of one run and, optionally, its children.
type tokenReport struct {
	Rows          []tokenRow         `json:"runs"`
	Weights       map[string]float64 `json:"weights"`
	TotalWeighted float64            `json:"total_weighted_tokens"`
	TotalNonCache int64              `json:"total_non_cache_tokens"`
	Unknown       []string           `json:"usage_unknown_runs"`
}

// parseTierWeights reads "sol=10,luna=1" into tier weights.
func parseTierWeights(raw string) (map[string]float64, error) {
	weights := map[string]float64{}
	for tier, weight := range defaultTierWeights {
		weights[tier] = weight
	}
	if strings.TrimSpace(raw) == "" {
		return weights, nil
	}
	for _, pair := range strings.Split(raw, ",") {
		tier, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		weight, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if !ok || strings.TrimSpace(tier) == "" || err != nil || weight < 0 {
			return nil, fmt.Errorf("invalid --weights entry %q (want tier=number)", pair)
		}
		weights[strings.ToLower(strings.TrimSpace(tier))] = weight
	}
	return weights, nil
}

// tierWeight matches a model name to the first tier it contains; an
// unrecognized model weighs 1 so it is never silently free.
func tierWeight(model string, weights map[string]float64) float64 {
	model = strings.ToLower(model)
	tiers := make([]string, 0, len(weights))
	for tier := range weights {
		tiers = append(tiers, tier)
	}
	sort.Strings(tiers)
	for _, tier := range tiers {
		if strings.Contains(model, tier) {
			return weights[tier]
		}
	}
	return 1
}

func (r *tokenReport) add(row tokenRow, weights map[string]float64) {
	row.Weight = tierWeight(row.Model, weights)
	row.Weighted = float64(row.NonCacheTokens) * row.Weight
	if !row.TokensKnown {
		r.Unknown = append(r.Unknown, row.RunID)
	}
	r.Rows = append(r.Rows, row)
	r.TotalWeighted += row.Weighted
	r.TotalNonCache += row.NonCacheTokens
}

// weightedTokens is the one token-weighting path: `run tokens` and the epoch
// check's spend trigger both read it.
func (a *App) weightedTokens(ids []string, weights map[string]float64) (*tokenReport, error) {
	report := &tokenReport{Weights: weights, Unknown: []string{}}
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
		fmt.Printf("%s  %-10s %-12s non-cache=%d cache-read=%d x%g = %.0f%s\n", row.RunID, row.Status, row.Model, row.NonCacheTokens, row.CacheReadTokens, row.Weight, row.Weighted, known)
	}
	fmt.Printf("Total weighted non-cache tokens: %.0f (non-cache %d across %d run(s))\n", report.TotalWeighted, report.TotalNonCache, len(report.Rows))
	return nil
}
