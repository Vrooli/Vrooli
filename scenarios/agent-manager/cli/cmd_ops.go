// `agent-manager ops <subcommand>` — operational stats CLI surface,
// reading the typed-event aggregations from /api/v1/stats/operational.
//
// Default output is human-readable (per project convention
// `feedback_cli_default_human_output`); `--json` returns the raw
// server response.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdOps(args []string) error {
	if len(args) == 0 {
		return a.opsHelp()
	}
	switch args[0] {
	case "summary":
		return a.opsCategory("summary", args[1:])
	case "fallback":
		return a.opsFallback(args[1:])
	case "health":
		return a.opsCategory("health", args[1:])
	case "sandbox":
		return a.opsCategory("sandbox", args[1:])
	case "heartbeat":
		return a.opsCategory("heartbeat", args[1:])
	case "checkpoint":
		return a.opsCategory("checkpoint", args[1:])
	case "retry":
		return a.opsCategory("retry", args[1:])
	case "help", "-h", "--help":
		return a.opsHelp()
	default:
		return fmt.Errorf("unknown ops subcommand: %s\n\nRun 'agent-manager ops help' for usage", args[0])
	}
}

func (a *App) opsHelp() error {
	fmt.Println(`Usage: agent-manager ops <subcommand> [options]

Operational stats derived from the typed event log (fallbacks, health
transitions, sandbox operations, heartbeat misses, checkpoint failures,
retries).

Subcommands:
  summary      Bundled snapshot across every category
  fallback     Runner + model fallback insights
  health       Engine-derived health transitions (use 'health' command
               for the authoritative current snapshot)
  sandbox      Sandbox operation success/failure summary
  heartbeat    Heartbeat-miss counters
  checkpoint   Checkpoint-failure counters
  retry        Retry-attempt counters

Options:
  --json   Output the raw server response`)
	return nil
}

func (a *App) opsCategory(category string, args []string) error {
	fs := flag.NewFlagSet("ops "+category, flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Operational.GetOperational(category)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	// Default: pretty-print JSON. Per-category bespoke tables are a UI
	// concern; the CLI's job is to make the structure readable. The
	// dedicated `ops fallback` command renders a real table because
	// fallback is the headline view.
	fmt.Println(string(prettyPrintJSON(body)))
	return nil
}

func (a *App) opsFallback(args []string) error {
	fs := flag.NewFlagSet("ops fallback", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.Operational.GetFallback()
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	return renderFallback(body)
}

func renderFallback(body []byte) error {
	var resp struct {
		History struct {
			HasHistory          bool    `json:"has_history"`
			HistoryDays         float64 `json:"history_days"`
			MinSampleMeaningful int     `json:"min_sample_meaningful"`
		} `json:"history"`
		EventCount       int64          `json:"event_count"`
		RunnerAttempts   int            `json:"runner_attempts"`
		RunnerExhausted  int            `json:"runner_exhausted"`
		ModelAttempts    int            `json:"model_attempts"`
		ModelExhausted   int            `json:"model_exhausted"`
		RunnerByReason   map[string]int `json:"runner_by_reason"`
		ModelByReason    map[string]int `json:"model_by_reason"`
		ModelByPreset    map[string]int `json:"model_by_preset"`
		RunnerChainDepth map[string]int `json:"runner_chain_depth"`
		ModelChainDepth  map[string]int `json:"model_chain_depth"`
		RunnerByPair     []struct {
			From   string `json:"from"`
			To     string `json:"to"`
			Reason string `json:"reason"`
			Count  int    `json:"count"`
		} `json:"runner_by_pair"`
		ModelByPair []struct {
			From   string `json:"from"`
			To     string `json:"to"`
			Reason string `json:"reason"`
			Count  int    `json:"count"`
		} `json:"model_by_pair"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode fallback response: %w", err)
	}

	totalRunner := resp.RunnerAttempts + resp.RunnerExhausted
	totalModel := resp.ModelAttempts + resp.ModelExhausted
	if totalRunner+totalModel < resp.History.MinSampleMeaningful {
		fmt.Printf("History: %.1f days, %d events (sample size below honesty threshold of %d — numbers may be misleading)\n\n",
			resp.History.HistoryDays, resp.EventCount, resp.History.MinSampleMeaningful)
	} else {
		fmt.Printf("History: %.1f days, %d events\n\n", resp.History.HistoryDays, resp.EventCount)
	}

	fmt.Printf("Runner fallback: %d attempted, %d exhausted\n", resp.RunnerAttempts, resp.RunnerExhausted)
	if len(resp.RunnerByReason) > 0 {
		printSortedCounts("  by reason:", resp.RunnerByReason)
	}
	fmt.Println()

	fmt.Printf("Model fallback:  %d attempted, %d exhausted\n", resp.ModelAttempts, resp.ModelExhausted)
	if len(resp.ModelByReason) > 0 {
		printSortedCounts("  by reason:", resp.ModelByReason)
	}
	if len(resp.ModelByPreset) > 0 {
		printSortedCounts("  by preset:", resp.ModelByPreset)
	}
	fmt.Println()

	if len(resp.ModelByPair) > 0 {
		fmt.Println("Top model fallback pairs:")
		fmt.Printf("  %-30s  %-30s  %-20s  %s\n", "FROM", "TO", "REASON", "COUNT")
		for i, p := range resp.ModelByPair {
			if i >= 10 {
				break
			}
			fmt.Printf("  %-30s  %-30s  %-20s  %d\n", trim(p.From, 30), trim(p.To, 30), trim(p.Reason, 20), p.Count)
		}
	}
	return nil
}

func printSortedCounts(label string, m map[string]int) {
	type kv struct {
		k string
		v int
	}
	rows := make([]kv, 0, len(m))
	for k, v := range m {
		rows = append(rows, kv{k, v})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].v != rows[j].v {
			return rows[i].v > rows[j].v
		}
		return rows[i].k < rows[j].k
	})
	fmt.Println(label)
	for _, r := range rows {
		fmt.Printf("    %-30s %d\n", r.k, r.v)
	}
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
