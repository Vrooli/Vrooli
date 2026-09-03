package vroolicli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
)

// AgentSessionRow is one live session as `vrooli agent list` shows it.
type AgentSessionRow struct {
	Session   string   `json:"session"`
	Harness   string   `json:"harness"`
	Agent     string   `json:"agent"`
	Tree      string   `json:"tree"`
	Scope     string   `json:"scope"`
	PID       int      `json:"pid"`
	Age       string   `json:"age"`
	Claims    []string `json:"claims,omitempty"`
	Frozen    string   `json:"frozen"`
	Heartbeat string   `json:"last_heartbeat"`
}

// agentSessionLister is the registry read the command depends on; tests
// inject leases directly.
type agentSessionLister func() ([]scenarioruntime.EditorLease, error)

// scopeFrozenFn answers whether a scope is frozen; "" means unknown.
type scopeFrozenFn func(scope string) string

func (app *App) runAgentList(ctx *CommandContext, args []string) error {
	fs := commandtree.NewFlagSet("vrooli agent list")
	fs.SetOutput(ctx.Stderr)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return clipolicy.UsageErrorf("agent list", "unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	lister := func() ([]scenarioruntime.EditorLease, error) {
		return maintenance.NewController(ctx.Root, home).ListAgentSessions()
	}
	return renderAgentList(ctx.Stdout, lister, liveScopeFrozen, time.Now().UTC(), *jsonOut || ctx.Globals.JSON)
}

// renderAgentList is the pure rendering step: session, harness, tree, scope,
// pid, age, claims, frozen.
func renderAgentList(w io.Writer, lister agentSessionLister, frozen scopeFrozenFn, now time.Time, asJSON bool) error {
	if w == nil {
		w = io.Discard
	}
	leases, err := lister()
	if err != nil {
		return fmt.Errorf("list agent sessions: %w", err)
	}
	rows := make([]AgentSessionRow, 0, len(leases))
	for _, lease := range leases {
		rows = append(rows, AgentSessionRow{
			Session: lease.SessionID, Harness: lease.Harness, Agent: lease.Agent, Tree: lease.WorkingDir, Scope: lease.Scope, PID: lease.PID,
			Age: now.Sub(lease.CreatedAt).Round(time.Second).String(), Claims: lease.Claims, Frozen: frozen(lease.Scope),
			Heartbeat: lease.LastHeartbeatAt.UTC().Format(time.RFC3339),
		})
	}
	if asJSON {
		return cliout.WriteJSON(w, map[string]any{"sessions": rows})
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(w, "No live agent sessions are recorded in the runtime registry.")
		return nil
	}
	table := make([][]string, 0, len(rows))
	for _, row := range rows {
		table = append(table, []string{row.Session, row.Agent, row.Tree, row.Scope, fmt.Sprint(row.PID), row.Age, strings.Join(row.Claims, ","), row.Frozen})
	}
	return cliout.RenderTable(w, []string{"Session", "Agent", "Tree", "Scope", "PID", "Age", "Claims", "Frozen"}, table)
}

// liveScopeFrozen reads cgroup.freeze for a session scope: a unit name is
// resolved through the user manager the way thaw does, a cgroup ref is read
// directly; anything else is reported as unknown rather than guessed.
func liveScopeFrozen(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" || scope == "none" {
		return "n/a"
	}
	ctx, cancel := context.WithTimeout(context.Background(), thawAPITimeout)
	defer cancel()
	ref, err := agentScopeRef(ctx, scope, shell.OSRunner{})
	if err != nil || ref.Kind != platformgo.ScopeKindCgroup {
		return "n/a"
	}
	frozen, err := platformgo.ScopeFrozen(ref)
	if err != nil {
		return "unknown"
	}
	if frozen {
		return "yes"
	}
	return "no"
}
