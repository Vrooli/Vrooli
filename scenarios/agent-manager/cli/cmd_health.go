// `agent-manager health <subcommand>` — persisted health audit CLI
// surface: current snapshots and audit history reads.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdHealth(args []string) error {
	if len(args) == 0 {
		return a.healthHelp()
	}
	switch args[0] {
	case "models":
		return a.healthModels(args[1:])
	case "runners":
		return a.healthRunners(args[1:])
	case "audit":
		return a.healthAudit(args[1:])
	case "help", "-h", "--help":
		return a.healthHelp()
	default:
		return fmt.Errorf("unknown health subcommand: %s\n\nRun 'agent-manager health help' for usage", args[0])
	}
}

func (a *App) healthHelp() error {
	fmt.Println(`Usage: agent-manager health <subcommand> [options]

Persisted health snapshots and audit history.

Subcommands:
  models     Current snapshot of every (runner, model) pair
  runners    Current snapshot of every runner
  audit      Paginated audit history (model_health_audit by default;
             use --scope=runner for runner_health_audit)

Audit options:
  --scope=model|runner   Audit table to query (default: model)
  --runner=<type>        Filter by runner_type
  --model=<id>           Filter by model_id (model scope only)
  --status=<state>       Filter by status: ok|unknown|failed
  --since=<RFC3339>      Lower bound on timestamp
  --until=<RFC3339>      Upper bound on timestamp
  --limit=<n>            Max rows (default 100)

Output:
  --json   Output the raw server response`)
	return nil
}

func (a *App) healthModels(args []string) error {
	fs := flag.NewFlagSet("health models", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.HealthAudit.GetModels()
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var resp struct {
		Models []struct {
			Runner      string    `json:"runner"`
			Model       string    `json:"model"`
			Status      string    `json:"status"`
			LastChecked time.Time `json:"last_checked"`
			Reason      string    `json:"reason,omitempty"`
			Message     string    `json:"message,omitempty"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Models) == 0 {
		fmt.Println("No model health observations recorded yet.")
		return nil
	}
	sort.Slice(resp.Models, func(i, j int) bool {
		if resp.Models[i].Runner != resp.Models[j].Runner {
			return resp.Models[i].Runner < resp.Models[j].Runner
		}
		return resp.Models[i].Model < resp.Models[j].Model
	})
	fmt.Printf("%-15s  %-30s  %-9s  %-19s  %s\n", "RUNNER", "MODEL", "STATUS", "LAST CHECKED", "REASON")
	for _, m := range resp.Models {
		fmt.Printf("%-15s  %-30s  %-9s  %-19s  %s\n",
			trim(m.Runner, 15),
			trim(m.Model, 30),
			m.Status,
			m.LastChecked.UTC().Format("2006-01-02 15:04:05"),
			m.Reason,
		)
	}
	return nil
}

func (a *App) healthRunners(args []string) error {
	fs := flag.NewFlagSet("health runners", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.services.HealthAudit.GetRunners()
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var resp struct {
		Runners []struct {
			Runner      string    `json:"runner"`
			Status      string    `json:"status"`
			LastChecked time.Time `json:"last_checked"`
			Reason      string    `json:"reason,omitempty"`
			Message     string    `json:"message,omitempty"`
		} `json:"runners"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Runners) == 0 {
		fmt.Println("No runner health observations recorded yet.")
		return nil
	}
	sort.Slice(resp.Runners, func(i, j int) bool { return resp.Runners[i].Runner < resp.Runners[j].Runner })
	fmt.Printf("%-15s  %-9s  %-19s  %s\n", "RUNNER", "STATUS", "LAST CHECKED", "REASON")
	for _, r := range resp.Runners {
		fmt.Printf("%-15s  %-9s  %-19s  %s\n",
			trim(r.Runner, 15),
			r.Status,
			r.LastChecked.UTC().Format("2006-01-02 15:04:05"),
			r.Reason,
		)
	}
	return nil
}

func (a *App) healthAudit(args []string) error {
	fs := flag.NewFlagSet("health audit", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	scope := fs.String("scope", "model", "audit table to query (model|runner)")
	runner := fs.String("runner", "", "filter by runner_type")
	model := fs.String("model", "", "filter by model_id")
	status := fs.String("status", "", "filter by status (ok|unknown|failed)")
	since := fs.String("since", "", "lower bound (RFC3339)")
	until := fs.String("until", "", "upper bound (RFC3339)")
	limit := fs.Int("limit", 100, "max rows")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	q := AuditQuery{
		Scope:  *scope,
		Runner: *runner,
		Model:  *model,
		Status: *status,
		Since:  *since,
		Until:  *until,
		Limit:  *limit,
	}
	body, err := a.services.HealthAudit.QueryAudit(q)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(body)
		return nil
	}
	var resp struct {
		Rows []struct {
			Timestamp   time.Time `json:"timestamp"`
			RunnerType  string    `json:"runnerType"`
			ModelID     string    `json:"modelId,omitempty"`
			Status      string    `json:"status"`
			Reason      string    `json:"reason,omitempty"`
			Message     string    `json:"message,omitempty"`
			TriggeredBy string    `json:"triggeredBy"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}
	if len(resp.Rows) == 0 {
		fmt.Println("No audit rows match the query.")
		return nil
	}
	if *scope == "model" {
		fmt.Printf("%-19s  %-15s  %-30s  %-9s  %-15s  %s\n", "WHEN (UTC)", "RUNNER", "MODEL", "STATUS", "REASON", "TRIGGERED BY")
		for _, r := range resp.Rows {
			fmt.Printf("%-19s  %-15s  %-30s  %-9s  %-15s  %s\n",
				r.Timestamp.UTC().Format("2006-01-02 15:04:05"),
				trim(r.RunnerType, 15), trim(r.ModelID, 30), r.Status, trim(r.Reason, 15), r.TriggeredBy)
		}
	} else {
		fmt.Printf("%-19s  %-15s  %-9s  %-15s  %s\n", "WHEN (UTC)", "RUNNER", "STATUS", "REASON", "TRIGGERED BY")
		for _, r := range resp.Rows {
			fmt.Printf("%-19s  %-15s  %-9s  %-15s  %s\n",
				r.Timestamp.UTC().Format("2006-01-02 15:04:05"),
				trim(r.RunnerType, 15), r.Status, trim(r.Reason, 15), r.TriggeredBy)
		}
	}
	return nil
}
