package vroolicli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	configpkg "github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/tuning"
)

// `vrooli agent reap --orphaned` stops agent sessions whose owner is
// confirmed gone: an Agent Manager run that is terminal or unknown to Agent
// Manager, a programmatic web-console session that is archived or absent, a
// bundled desktop app whose runtime has exited. It never stops a terminal or
// unknown-owner session, an idle-only session, or an operator's web-console
// terminal; every stop is revalidated against the owner immediately before it
// happens and audited in the runtime registry.

// AgentReapRow is one orphaned session and what the reap did with it.
type AgentReapRow struct {
	Session  string         `json:"session"`
	Agent    string         `json:"agent,omitempty"`
	Scope    string         `json:"scope"`
	Age      string         `json:"age,omitempty"`
	Owner    AgentOwnerView `json:"owner"`
	Decision string         `json:"decision"`
	Reason   string         `json:"reason,omitempty"`
	Outcome  string         `json:"outcome"`
	Error    string         `json:"error,omitempty"`
	// Memory and swap are agentsession.MeasureUsage's figures, the same
	// ones `vrooli agent list` reports; null with a status when unknown.
	MemoryBytes  *int64 `json:"memory_bytes"`
	MemoryStatus string `json:"memory_status"`
	SwapBytes    *int64 `json:"swap_bytes"`
	SwapStatus   string `json:"swap_status"`
	Tasks        int64  `json:"tasks"`
	AuditID      string `json:"audit_id,omitempty"`
}

// AgentReapReport is the command's JSON document.
type AgentReapReport struct {
	DryRun    bool                      `json:"dry_run"`
	OlderThan string                    `json:"older_than"`
	Sessions  int                       `json:"sessions_seen"`
	Summary   map[string]map[string]int `json:"summary"`
	Stop      int                       `json:"stop"`
	Kept      int                       `json:"kept"`
	Freed     AgentReapTotals           `json:"memory"`
	Rows      []AgentReapRow            `json:"orphaned"`
}

// AgentReapTotals adds up what the stopped (or, in a dry run, eligible)
// sessions hold.
type AgentReapTotals struct {
	MemoryBytes int64 `json:"memory_bytes"`
	SwapBytes   int64 `json:"swap_bytes"`
	Tasks       int64 `json:"tasks"`
}

func (app *App) runAgentReap(ctx *AppContext, args []string) error {
	fs := commandtree.NewFlagSet("vrooli agent reap")
	fs.SetOutput(ctx.Stderr)
	orphaned := fs.Bool("orphaned", false, "stop sessions whose owner is confirmed gone (required: the only reap this command performs)")
	olderThan := fs.Duration("older-than", agentsession.DefaultReapOlderThan, "only stop sessions at least this old")
	dryRun := fs.Bool("dry-run", false, "list what would be stopped, with evidence, and stop nothing")
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return clipolicy.UsageErrorf("agent reap", "unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	if !*orphaned {
		return clipolicy.UsageErrorf("agent reap", "--orphaned is required: reap stops only sessions whose owner is confirmed gone")
	}
	if *olderThan < 0 {
		return clipolicy.UsageErrorf("agent reap", "--older-than must not be negative")
	}
	home, err := configpkg.HomeDir()
	if err != nil {
		return err
	}
	lister := func() ([]scenarioruntime.EditorLease, error) {
		return maintenance.NewController(ctx.Root, home).ListAgentSessions()
	}
	reconciler, closeStore := liveAgentReconciler(home, tuning.AgentSessionIdleAfter())
	defer closeStore()
	executor := agentsession.Executor{
		Stopper:   agentsession.NewPlatformStopper(),
		Initiator: "vrooli agent reap",
		Recheck: func(runCtx context.Context, session agentsession.Session) agentsession.Liveness {
			return agentsession.Check(runCtx, session.Owner, reconciler.Probes, time.Now())
		},
	}
	if store, ok := reconciler.Store.(agentsession.AuditSink); ok {
		executor.Audit = store
	}
	return runAgentReapWith(context.Background(), ctx.Stdout, lister, reconciler, executor, agentsession.ReapPolicy{OlderThan: *olderThan}, *dryRun, *jsonOut || ctx.Globals.JSON)
}

// runAgentReapWith is the testable body: reconcile, plan, execute, render.
func runAgentReapWith(ctx context.Context, w io.Writer, lister agentSessionLister, reconciler agentsession.Reconciler, executor agentsession.Executor, policy agentsession.ReapPolicy, dryRun, asJSON bool) error {
	if w == nil {
		w = io.Discard
	}
	leases, err := lister()
	if err != nil {
		return fmt.Errorf("list agent sessions: %w", err)
	}
	now := time.Now().UTC()
	if reconciler.Now != nil {
		now = reconciler.Now().UTC()
	}
	sessions := reconciler.Sessions(ctx, leases)
	plan := agentsession.Plan(sessions, policy, now)
	results := executor.Execute(ctx, plan, dryRun)

	olderThan := policy.OlderThan
	if olderThan <= 0 {
		olderThan = agentsession.DefaultReapOlderThan
	}
	report := AgentReapReport{DryRun: dryRun, OlderThan: olderThan.String(), Sessions: len(sessions), Summary: agentsession.CountByOwnerKind(sessions), Rows: make([]AgentReapRow, 0, len(results))}
	var failed int
	for _, result := range results {
		session := result.Candidate.Session
		row := AgentReapRow{
			Session: session.SessionID, Agent: session.Agent, Scope: session.Scope, Decision: result.Candidate.Decision,
			Reason: result.Candidate.Reason, Outcome: result.Outcome, Error: result.Error, AuditID: result.AuditID,
			MemoryBytes: result.Usage.MemoryBytes, MemoryStatus: result.Usage.MemoryStatus,
			SwapBytes: result.Usage.SwapBytes, SwapStatus: result.Usage.SwapStatus, Tasks: session.Tasks,
			Owner: agentSessionRow(session, now).Owner,
		}
		if !session.StartedAt.IsZero() {
			row.Age = now.Sub(session.StartedAt).Round(time.Second).String()
		}
		report.Rows = append(report.Rows, row)
		switch result.Outcome {
		case agentsession.OutcomeKept:
			report.Kept++
		case agentsession.OutcomeWouldStop, agentsession.OutcomeStopped:
			report.Stop++
			if row.MemoryBytes != nil {
				report.Freed.MemoryBytes += *row.MemoryBytes
			}
			if row.SwapBytes != nil {
				report.Freed.SwapBytes += *row.SwapBytes
			}
			report.Freed.Tasks += max(row.Tasks, 0)
		default:
			failed++
		}
	}
	if asJSON {
		if err := cliout.WriteJSON(w, report); err != nil {
			return err
		}
	} else {
		renderAgentReap(w, report)
	}
	if failed > 0 {
		return fmt.Errorf("agent reap: %d orphaned session(s) were not stopped", failed)
	}
	return nil
}

func renderAgentReap(w io.Writer, report AgentReapReport) {
	verb := "stopped"
	if report.DryRun {
		verb = "would stop"
	}
	_, _ = fmt.Fprintf(w, "%d session(s) seen; %d orphaned; %s %d (older than %s); kept %d\n",
		report.Sessions, len(report.Rows), verb, report.Stop, report.OlderThan, report.Kept)
	if len(report.Rows) == 0 {
		return
	}
	table := make([][]string, 0, len(report.Rows))
	for _, row := range report.Rows {
		owner := row.Owner.Kind
		if row.Owner.Ref != "" {
			owner += ":" + shortRef(row.Owner.Ref)
		}
		detail := row.Reason
		if row.Error != "" {
			detail = row.Error
		}
		table = append(table, []string{row.Session, row.Scope, owner, row.Age, row.Outcome, optionalBytes(row.MemoryBytes, row.MemoryStatus), optionalBytes(row.SwapBytes, row.SwapStatus), detail})
	}
	_ = cliout.RenderTable(w, []string{"Session", "Scope", "Owner", "Age", "Outcome", "Memory", "Swap", "Evidence"}, table)
	_, _ = fmt.Fprintf(w, "memory %s, swap %s, %d tasks held by sessions %s\n", humanBytes(report.Freed.MemoryBytes), humanBytes(report.Freed.SwapBytes), report.Freed.Tasks, verb)
}

// optionalBytes renders a figure, or its status when it is unknown.
func optionalBytes(n *int64, status string) string {
	if n == nil {
		return status
	}
	if status == agentsession.UsagePartial {
		return ">=" + humanBytes(*n)
	}
	return humanBytes(*n)
}

func humanBytes(n int64) string {
	if n < 0 {
		return "?"
	}
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
