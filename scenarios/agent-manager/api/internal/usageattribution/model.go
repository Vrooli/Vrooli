// Package usageattribution explains provider quota consumption
// (USAGE-INTELLIGENCE T-5 and T-6). Attribute splits each window's observed
// used-percent increase across the usage that ran in it, with the token to
// quota conversion recalibrated per window and an explicit unattributed
// remainder. Recommend derives ranked, deterministic findings from the same
// attribution and durable run facts.
package usageattribution

import (
	"context"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/transcriptindex"
	"agent-manager/internal/usageforecast"
)

// Sources of a turn.
const (
	SourceIndexed      = "indexed"       // operator session read by the Transcript Indexer
	SourceAgentManager = "agent_manager" // run Agent Manager launched
	SourceMixed        = "mixed"
)

// Turn is usage one run reported at At: one provider report, or the reports
// of one session folded into a short bucket. Tokens are its contribution
// under runsignal.AccountUsage.
type Turn struct {
	At     time.Time
	RunID  string
	Runner string
	Model  string
	Tokens domain.UsageCounters
	// Reports counts provider usage reports, one per model request (Codex
	// token_count, Claude assistant message), not conversational turns: a
	// Codex turn makes many requests.
	Reports int
	Facts   transcriptindex.TurnFacts
}

// Prompt is the request size the turn read: input plus cached and cache
// creation tokens.
func (t Turn) Prompt() int64 {
	return int64(t.Tokens.InputTokens) + int64(t.Tokens.CacheReadTokens) + int64(t.Tokens.CacheCreationTokens)
}

// Run carries the dimensions a turn is grouped by.
type Run struct {
	ID          string
	Label       string
	Source      string
	Runner      string
	ProfileID   string
	ProfileName string
	// Workload is kind:key for Agent Manager runs; operator sessions have none.
	Workload string
	// Session groups a run with its sub-agents and forks: the web-console
	// session, the top-level transcript session, or the root run.
	Session      string
	SessionLabel string
	ParentRunID  string
	// Child marks a sub-agent sidechain, fork or child run.
	Child     bool
	Status    string
	Succeeded bool
	// Parked is an Agent Manager run waiting on an await or wake.
	Parked bool
}

// Episode is one persisted friction episode.
type Episode struct {
	RunID           string
	EpisodeID       string
	At              time.Time
	Pattern         string
	Fingerprint     string
	RepeatedElement string
	Owner           string
	// Turns is the provider turns the episode spans; its cost is their share
	// of the run's turns.
	Turns  int
	Tokens int64
}

// Outcome is one finished Agent Manager run, for R-model comparisons.
type Outcome struct {
	RunID     string
	Runner    string
	Model     string
	ProfileID string
	Workload  string
	Succeeded bool
	At        time.Time
	// Tokens is the run's total from the one accounting function.
	Tokens domain.UsageCounters
}

// Store reads the durable facts attribution and recommendations use.
type Store interface {
	// UsageTurns returns the usage reported in [from, to] by the durable
	// transcript usage rows: operator sessions and Agent Manager runs, each
	// report counted once.
	UsageTurns(ctx context.Context, from, to time.Time) ([]Turn, error)
	UsageRuns(ctx context.Context, runIDs []string) (map[string]Run, error)
	UsageEpisodes(ctx context.Context, from, to time.Time) ([]Episode, error)
	UsageOutcomes(ctx context.Context, from, to time.Time) ([]Outcome, error)
}

// Price is a model's API price per token. Under a subscription it is only a
// prior for the relative quota cost of token categories and models; the
// absolute scale is calibrated per window.
type Price struct {
	Input, Output, CacheRead, CacheCreation float64
	Known                                   bool
}

// Prices looks up a model's price; it returns Known=false when unpriced.
type Prices func(runner, model string) Price

// fallbackPrice prices unpriced models: API ratios cached:input:creation:output
// of 1:20:25:100 at a mid-tier (terra-like) level.
var fallbackPrice = Price{CacheRead: 5e-8, Input: 1e-6, CacheCreation: 1.25e-6, Output: 5e-6}

// tierPriors scale fallbackPrice by model tier when the price book has no
// price for a subscription model (the runner resources resolve them to
// unpriced subscription names). Codex tiers are the E-2 per-model fit on
// this host (cached tokens per 1%: sol ≈50M, terra ≈100M, luna ≈520M); Claude
// tiers follow Anthropic's list-price ratios. Only the ratios matter: the
// scale is calibrated per window.
var tierPriors = []struct {
	marker   string
	relative float64
}{{"sol", 2}, {"terra", 1}, {"luna", 0.2}, {"opus", 1}, {"sonnet", 0.5}, {"haiku", 0.05}}

// TierPrice is the prior price of an unpriced model.
func TierPrice(model string) Price {
	model = strings.ToLower(model)
	for _, tier := range tierPriors {
		if strings.Contains(model, tier.marker) {
			p := fallbackPrice
			p.CacheRead, p.Input, p.CacheCreation, p.Output = p.CacheRead*tier.relative, p.Input*tier.relative, p.CacheCreation*tier.relative, p.Output*tier.relative
			p.Known = true
			return p
		}
	}
	return Price{}
}

// normalized fills what a price book often omits: cached reads at a tenth
// of input and cache writes at 1.25× input, the providers' common ratios.
func (p Price) normalized() Price {
	if !p.Known || p.Input <= 0 {
		return fallbackPrice
	}
	if p.CacheRead <= 0 {
		p.CacheRead = p.Input / 10
	}
	if p.CacheCreation <= 0 {
		p.CacheCreation = p.Input * 1.25
	}
	return p
}

// cost is the turn's API-price cost; it is proportional to its
// cached-input-equivalent tokens.
func (p Price) cost(tokens domain.UsageCounters) float64 {
	return p.CacheRead*float64(tokens.CacheReadTokens) + p.Input*float64(tokens.InputTokens) +
		p.CacheCreation*float64(tokens.CacheCreationTokens) + p.Output*float64(tokens.OutputTokens)
}

// Window is one provider subscription window with its step series.
type Window struct {
	Provider    string
	Pool        string
	Name        string
	Minutes     int64
	Start       time.Time
	ResetAt     time.Time
	LastAt      time.Time
	UsedPercent float64
	Series      []usageforecast.Point
}

// FromForecast adapts a forecast window.
func FromForecast(w *usageforecast.Window) Window {
	return Window{Provider: w.Provider, Pool: w.Pool, Name: w.Window, Minutes: w.WindowMinutes, Start: w.Start, ResetAt: w.ResetAt, LastAt: w.LastAt, UsedPercent: w.Max(), Series: w.Series()}
}

// RunnerFor names the runner whose transcripts explain a provider's quota.
func RunnerFor(provider string) string {
	switch provider {
	case "openai":
		return string(domain.RunnerTypeCodex)
	case "anthropic":
		return string(domain.RunnerTypeClaudeCode)
	}
	return ""
}

// estimatedImageTokens is the typical input cost of one viewed image (R-1:
// Codex 32px patches ×1.2, Claude w×h/750; a 1280×800 screenshot is
// ≈1.2–1.4k tokens). It explains part of input tokens; it is never a charge.
const estimatedImageTokens = 1300

// EstimatedImageTokens estimates the input tokens images contributed.
func EstimatedImageTokens(facts transcriptindex.TurnFacts) int64 {
	return int64(facts.ImagesIn) * estimatedImageTokens
}
