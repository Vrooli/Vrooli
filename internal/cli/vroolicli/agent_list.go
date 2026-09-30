package vroolicli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	configpkg "github.com/vrooli/vrooli/internal/config"
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
	// State is what the kernel agreed to: live, ghost, orphan, unrecorded or
	// undetermined. The lease table alone lags the kernel in both directions.
	State string `json:"state"`
	// Tasks is what a scope holds when no lease names it.
	Tasks int64 `json:"tasks,omitempty"`
}

// agentSessionLister is the registry read the command depends on; tests
// inject leases directly.
type agentSessionLister func() ([]scenarioruntime.EditorLease, error)

// scopeFrozenFn answers whether a scope is frozen; "" means unknown.
type scopeFrozenFn func(scope string) string

func (app *App) runAgentList(ctx *AppContext, args []string) error {
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
	home, err := configpkg.HomeDir()
	if err != nil {
		return err
	}
	lister := func() ([]scenarioruntime.EditorLease, error) {
		return maintenance.NewController(ctx.Root, home).ListAgentSessions()
	}
	return renderAgentList(ctx.Stdout, lister, liveScopeFrozen, liveKernelCensus, pidRunning, time.Now().UTC(), *jsonOut || ctx.Globals.JSON)
}

// renderAgentList is the pure rendering step, reconciled against the kernel.
//
// The lease table alone is not the answer to "who is on this host". It lags
// the kernel in both directions: a lease outlives its process until the
// registry sweep proves the pid dead, and a scope outlives its session when
// that session left long-running children behind. On 2026-09-04 this command
// showed 38 sessions while systemd held 42 scopes and the cgroup tree held
// 39, three of the rows named scopes that had never existed, and none of the
// seven scopes holding the scenario fleet appeared at all.
//
// So every row carries a state the kernel agreed to, and a scope the kernel
// holds that no lease names is shown rather than hidden.
func renderAgentList(w io.Writer, lister agentSessionLister, frozen scopeFrozenFn, census kernelCensusFn, alive pidAliveFn, now time.Time, asJSON bool) error {
	if w == nil {
		w = io.Discard
	}
	leases, err := lister()
	if err != nil {
		return fmt.Errorf("list agent sessions: %w", err)
	}
	held, censusErr := census()
	named := make(map[string]bool, len(leases))
	rows := make([]AgentSessionRow, 0, len(leases))
	for _, lease := range leases {
		named[strings.TrimSuffix(lease.Scope, ".scope")] = true
		rows = append(rows, AgentSessionRow{
			Session: lease.SessionID, Harness: lease.Harness, Agent: lease.Agent, Tree: lease.WorkingDir, Scope: lease.Scope, PID: lease.PID,
			Age: now.Sub(lease.CreatedAt).Round(time.Second).String(), Claims: lease.Claims, Frozen: frozen(lease.Scope),
			Heartbeat: lease.LastHeartbeatAt.UTC().Format(time.RFC3339),
			State:     leaseState(lease, held, alive, censusErr),
		})
	}
	// A scope the kernel holds that no lease names is a session this command
	// would otherwise hide. It is the shape that consumed the agent slice.
	for _, entry := range held {
		if named[strings.TrimSuffix(entry.Name, ".scope")] {
			continue
		}
		state := agentStateUnrecorded
		if entry.Agentless {
			state = agentStateOrphan
		}
		rows = append(rows, AgentSessionRow{
			Session: "(no lease)", Scope: entry.Name, Tasks: entry.Tasks,
			Frozen: frozen(entry.Name), State: state,
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
		table = append(table, []string{row.Session, row.Agent, row.State, row.Tree, row.Scope, fmt.Sprint(row.PID), row.Age, strings.Join(row.Claims, ","), row.Frozen})
	}
	return cliout.RenderTable(w, []string{"Session", "Agent", "State", "Tree", "Scope", "PID", "Age", "Claims", "Frozen"}, table)
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

// Session states, each one the kernel agreed to.
const (
	// agentStateLive is a lease whose process is running.
	agentStateLive = "live"
	// agentStateGhost is a lease whose process is gone. The registry expires
	// it on proof of death, so a ghost is a row waiting for the next sweep,
	// not a session.
	agentStateGhost = "ghost"
	// agentStateOrphan is a scope holding processes with no coding agent
	// among them: the session ended and left long-running children.
	agentStateOrphan = "orphan"
	// agentStateUnrecorded is a scope the kernel holds that no lease names.
	agentStateUnrecorded = "unrecorded"
	// agentStateUndetermined is what a kernel that could not be read gets. A
	// census that cannot run must never make a session look healthy.
	agentStateUndetermined = "undetermined"
)

// kernelCensusFn reads the session scopes the agent slice holds.
type kernelCensusFn func() ([]agentscope.Entry, error)

// pidAliveFn reports whether a process is still running.
type pidAliveFn func(pid int) bool

// leaseState reconciles one lease with the kernel.
func leaseState(lease scenarioruntime.EditorLease, held []agentscope.Entry, alive pidAliveFn, censusErr error) string {
	if alive != nil && lease.PID > 0 && !alive(lease.PID) {
		return agentStateGhost
	}
	if censusErr != nil {
		return agentStateUndetermined
	}
	for _, entry := range held {
		if strings.TrimSuffix(entry.Name, ".scope") != strings.TrimSuffix(lease.Scope, ".scope") {
			continue
		}
		if entry.Agentless {
			return agentStateOrphan
		}
		return agentStateLive
	}
	// The lease names a scope the kernel does not hold. Either the scope was
	// reaped after the session left it, or it was never created.
	return agentStateUnrecorded
}

// liveKernelCensus reads this host's agent slice.
func liveKernelCensus() ([]agentscope.Entry, error) {
	path, err := platformgo.SliceCgroup(agentscope.Slice)
	if err != nil {
		return nil, err
	}
	return agentscope.NewReader().Census(agentscope.Ref(path))
}

// pidRunning asks the operating system whether a pid is alive.
func pidRunning(pid int) bool { return platformgo.IsPIDRunning(pid) }
