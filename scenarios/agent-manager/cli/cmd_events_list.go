// `agent-manager events <subcommand>` — typed event log read CLI.
//
// "list" reads the persisted typed-operational event store via
// /api/v1/events. The existing run-scoped WebSocket streamer (used by
// `agent-manager run watch`) is the right path for live event tails;
// this command is for historical / cross-run queries.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdEvents(args []string) error {
	if len(args) == 0 {
		return a.eventsHelp()
	}
	switch args[0] {
	case "list":
		return a.eventsList(args[1:])
	case "help", "-h", "--help":
		return a.eventsHelp()
	default:
		return fmt.Errorf("unknown events subcommand: %s\n\nRun 'agent-manager events help' for usage", args[0])
	}
}

func (a *App) eventsHelp() error {
	fmt.Println(`Usage: agent-manager events <subcommand> [options]

Typed-operational event log queries. Live per-run streaming is on
'agent-manager run watch'; this command is for historical queries.

Subcommands:
  list   Query typed events with optional filters

list options:
  --run=<id>     Filter to one run id (UUID)
  --type=<name>  Filter to one event type (e.g., model.fallback.attempted)
  --since=<ts>   RFC3339 lower bound on timestamp (with --type only)
  --limit=<n>    Max rows (default 100, max 1000)
  --json         Output the raw server response`)
	return nil
}

func (a *App) eventsList(args []string) error {
	fs := flag.NewFlagSet("events list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	run := fs.String("run", "", "run id (UUID)")
	typ := fs.String("type", "", "typed event type")
	since := fs.String("since", "", "RFC3339 lower bound on timestamp")
	limit := fs.Int("limit", 100, "max rows")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	q := EventsQuery{Run: *run, Type: *typ, Since: *since, Limit: *limit}
	body, err := a.services.Events.List(q)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var resp struct {
		Events []struct {
			ID            string      `json:"id"`
			RunID         string      `json:"run_id"`
			Sequence      int64       `json:"sequence"`
			EventType     string      `json:"event_type"`
			SchemaVersion int         `json:"schema_version"`
			Timestamp     time.Time   `json:"timestamp"`
			Payload       interface{} `json:"payload"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Events) == 0 {
		fmt.Println("No events match the query.")
		return nil
	}
	fmt.Printf("%-19s  %-32s  %-30s  %s\n", "WHEN (UTC)", "RUN", "EVENT TYPE", "PAYLOAD")
	for _, e := range resp.Events {
		payloadJSON, _ := json.Marshal(e.Payload)
		fmt.Printf("%-19s  %-32s  %-30s  %s\n",
			e.Timestamp.UTC().Format("2006-01-02 15:04:05"),
			e.RunID,
			trim(e.EventType, 30),
			string(payloadJSON),
		)
	}
	return nil
}
