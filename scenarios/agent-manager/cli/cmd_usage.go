// This file holds the weighted token spend `run report` and the epoch check
// read, and `usage reindex` (rebuilds durable invocation evidence).
package main

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"agent-manager/cli/internal/support"

	"github.com/vrooli/cli-core/agentcatalog"
	"github.com/vrooli/cli-core/cliapp"
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

// weightedTokens is the one token-weighting path: `run report` and the epoch
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
		report.add(tokenRow{
			RunID: runID, Status: status, Model: accounting.GetModel(), TokensKnown: accounting.GetTokensKnown(),
			NonCacheTokens: accounting.GetNonCacheTokens(), CacheReadTokens: accounting.GetCacheReadTokens(),
		}, weights)
	}
	return report, nil
}

func (a *App) usageCommands() []cliapp.Command {
	reindex := support.Command("reindex", "Rebuild durable invocation evidence for one run or a filtered corpus [--full] [--json]", a.usageReindex)
	reindex.Usage = "agent-manager usage reindex [<run-id>] [--from RFC3339] [--to RFC3339] [--status s] [--profile-id UUID] [--tag-prefix p] [--limit n] [--full] [--json]"
	return []cliapp.Command{reindex}
}

// printSpend prints the weighted non-cache token spend rows `run report` shows.
func printSpend(report *tokenReport) {
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
}

// usageReindex rebuilds durable invocation evidence from retained events for
// one run or a filtered corpus. By default it refreshes: a run is rebuilt only
// when its events advanced or its classifier version is stale. --full rebuilds
// every selected run.
func (a *App) usageReindex(args []string) error {
	fs := flag.NewFlagSet("usage reindex", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	from := fs.String("from", "", "RFC3339 inclusive start (run end time)")
	to := fs.String("to", "", "RFC3339 exclusive end (run end time)")
	status := fs.String("status", "", "run status")
	profileID := fs.String("profile-id", "", "agent profile UUID")
	tagPrefix := fs.String("tag-prefix", "", "tag prefix")
	limit := fs.String("limit", "", "maximum runs (default 1000)")
	full := fs.Bool("full", false, "rebuild every selected run, not only runs whose events advanced")
	var runID string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		runID, args = args[0], args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	values := reindexQuery(runID, map[string]string{"from": *from, "to": *to, "status": *status, "profile_id": *profileID, "tag_prefix": *tagPrefix, "limit": *limit}, *full)
	body, err := a.services.Runs.ReplayInvocationCorpus(values)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Println(string(body))
	return nil
}

func reindexQuery(runID string, filters map[string]string, full bool) url.Values {
	values := url.Values{}
	if runID != "" {
		values.Set("run_id", runID)
	}
	for key, value := range filters {
		if value != "" {
			values.Set(key, value)
		}
	}
	if !full {
		values.Set("mode", "refresh")
	}
	return values
}
