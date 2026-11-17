package vroolicli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	configpkg "github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/tuning"
)

// AgentSessionRow is one live session as `vrooli agent list` shows it.
type AgentSessionRow struct {
	Session     string   `json:"session"`
	LeaseStatus string   `json:"lease_status,omitempty"`
	BindingID   string   `json:"binding_id,omitempty"`
	Harness     string   `json:"harness"`
	Agent       string   `json:"agent"`
	Tree        string   `json:"tree"`
	Scope       string   `json:"scope"`
	PID         int      `json:"pid"`
	Age         string   `json:"age"`
	Claims      []string `json:"claims,omitempty"`
	Frozen      string   `json:"frozen"`
	Heartbeat   string   `json:"last_heartbeat"`
	// State is derived: live, idle or orphaned on top of what the kernel
	// agreed to (ghost, agentless, unrecorded, undetermined).
	State string `json:"state"`
	// KernelState is the kernel's answer before owner and activity.
	KernelState string `json:"kernel_state"`
	ScopeHeld   bool   `json:"scope_held"`
	// Owner is what the session belongs to and whether it is still there.
	Owner AgentOwnerView `json:"owner"`
	// LastActivity is the newest heartbeat or terminal input/output seen.
	LastActivity string `json:"last_activity,omitempty"`
	// Tasks is what a scope holds when no lease names it.
	Tasks int64 `json:"tasks,omitempty"`
	// MemoryBytes and SwapBytes prefer exact cgroup v2 charged-byte readings,
	// then fall back to per-process VmRSS/VmSwap sums. Source distinguishes
	// exact cgroup totals from process sums that can double-count shared pages.
	MemoryBytes  *int64 `json:"memory_bytes"`
	MemoryStatus string `json:"memory_status"`
	MemorySource string `json:"memory_source,omitempty"`
	SwapBytes    *int64 `json:"swap_bytes"`
	SwapStatus   string `json:"swap_status"`
	SwapSource   string `json:"swap_source,omitempty"`
	UsageDetail  string `json:"usage_detail,omitempty"`
	// Reapable and ReapReason are the reap policy's verdict on this row
	// (agentsession.ReapDecision with the default --older-than), so an
	// orphan kept as an operator terminal is visible without running reap.
	Reapable   bool   `json:"reapable"`
	ReapReason string `json:"reap_reason"`
}

// AgentListSummary separates complete readings, partial process sums, and
// unavailable rows so the scalar complete totals never hide a partial read.
type AgentListSummary struct {
	ByOwnerKind map[string]map[string]int `json:"by_owner_kind"`
	Sessions    int                       `json:"sessions"`
	// Complete scalar totals mix source bases; use by-source maps to
	// distinguish cgroup charged RAM from per-process sums when both exist.
	MemoryBytes                int64            `json:"memory_bytes"`
	MemoryBytesBySource        map[string]int64 `json:"memory_bytes_by_source"`
	MemoryPartial              int              `json:"memory_partial"`
	MemoryPartialBytes         int64            `json:"memory_partial_bytes"`
	MemoryPartialBySource      map[string]int64 `json:"memory_partial_bytes_by_source"`
	MemoryUnknown              int              `json:"memory_unknown"`
	SwapBytes                  int64            `json:"swap_bytes"`
	SwapBytesBySource          map[string]int64 `json:"swap_bytes_by_source"`
	SwapPartial                int              `json:"swap_partial"`
	SwapPartialBytes           int64            `json:"swap_partial_bytes"`
	SwapPartialBySource        map[string]int64 `json:"swap_partial_bytes_by_source"`
	SwapUnknown                int              `json:"swap_unknown"`
	Reapable                   int              `json:"reapable"`
	ReapableMemoryBytes        int64            `json:"reapable_memory_bytes"`
	ReapableMemoryPartialBytes int64            `json:"reapable_memory_partial_bytes"`
	ReapableSwapBytes          int64            `json:"reapable_swap_bytes"`
	ReapableSwapPartialBytes   int64            `json:"reapable_swap_partial_bytes"`
}

// AgentOwnerView is the owner block of a row.
type AgentOwnerView struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref,omitempty"`
	// Alive is true or false when the owner's authority answered, null when
	// it could not be asked or the kind has no authority.
	Alive     *bool                 `json:"alive"`
	Liveness  string                `json:"liveness"`
	Detail    string                `json:"detail,omitempty"`
	CheckedAt string                `json:"checked_at"`
	Source    string                `json:"source,omitempty"`
	Evidence  agentsession.Evidence `json:"evidence"`
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
	idleAfter := fs.Duration("idle-after", tuning.AgentSessionIdleAfter(), "report a live session idle after this long with no heartbeat or terminal activity")
	sessionID := fs.String("session-id", "", "inspect one exact lease, including expired records, without mutation")
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
	var lister agentSessionLister
	var reconciler agentsession.Reconciler
	var closeStore func()
	if strings.TrimSpace(*sessionID) != "" {
		leases, err := exactAgentSession(home, *sessionID)
		if err != nil {
			return err
		}
		lister = func() ([]scenarioruntime.EditorLease, error) { return leases, nil }
		// Exact historical inspection must remain read-only: no owner backfill
		// is performed while resolving an expired or legacy lease.
		reconciler = agentsession.LiveReconciler(home, nil, *idleAfter)
		if len(leases) == 1 {
			scopeUnit := (agentsession.Session{Scope: leases[0].Scope}).ScopeUnit()
			reconciler.Census = exactAgentScopeCensus(scopeUnit)
		}
		closeStore = func() {}
	} else {
		lister = func() ([]scenarioruntime.EditorLease, error) {
			return maintenance.NewController(ctx.Root, home).ListAgentSessions()
		}
		reconciler, closeStore = liveAgentReconciler(home, *idleAfter)
	}
	defer closeStore()
	return renderAgentListFiltered(context.Background(), ctx.Stdout, lister, reconciler, *jsonOut || ctx.Globals.JSON, *sessionID)
}

func exactAgentSession(home, sessionID string) ([]scenarioruntime.EditorLease, error) {
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home, ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("open runtime registry read-only: %w", err)
	}
	defer store.Close()
	lease, err := store.GetEditorLease(context.Background(), sessionID)
	if err != nil {
		return nil, fmt.Errorf("read exact agent session %s: %w", sessionID, err)
	}
	return []scenarioruntime.EditorLease{lease}, nil
}

func exactAgentScopeCensus(scopeUnit string) func() ([]agentscope.Entry, error) {
	return func() ([]agentscope.Entry, error) {
		if scopeUnit == "" {
			return nil, nil
		}
		entries, err := agentsession.LiveCensus()
		if err != nil {
			return nil, err
		}
		return filterAgentScopeCensus(entries, scopeUnit), nil
	}
}

func filterAgentScopeCensus(entries []agentscope.Entry, scopeUnit string) []agentscope.Entry {
	filtered := make([]agentscope.Entry, 0, 1)
	for _, entry := range entries {
		if entry.Name == scopeUnit {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// liveAgentReconciler binds the reconciliation to this host: the kernel
// census, the process table, the owners' authorities (read-only), and the
// registry for backfilling owners on leases recorded before owners existed.
func liveAgentReconciler(home string, idleAfter time.Duration) (agentsession.Reconciler, func()) {
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		return agentsession.LiveReconciler(home, nil, idleAfter), func() {}
	}
	return agentsession.LiveReconciler(home, store, idleAfter), func() { _ = store.Close() }
}

// renderAgentList reconciles the lease table against the kernel and the
// owners, then renders it.
//
// The lease table alone is not the answer to "who is on this host". It lags
// the kernel in both directions: a lease outlives its process until the
// registry sweep proves the pid dead, and a scope outlives its session when
// that session left long-running children behind. On 2026-09-04 this command
// showed 38 sessions while systemd held 42 scopes and the cgroup tree held
// 39, three of the rows named scopes that had never existed, and none of the
// seven scopes holding the scenario fleet appeared at all. On 2026-10-07 it
// showed 89 scopes, every one "live", 42 of them helpers of a desktop app that
// had exited three weeks earlier: without an owner, a leak and a legitimate
// long-lived agent looked the same.
//
// So every row carries a state the kernel agreed to, an owner and whether the
// owner is still there, and a scope the kernel holds that no lease names is
// shown rather than hidden.
func renderAgentList(ctx context.Context, w io.Writer, lister agentSessionLister, reconciler agentsession.Reconciler, asJSON bool) error {
	return renderAgentListFiltered(ctx, w, lister, reconciler, asJSON, "")
}

func renderAgentListFiltered(ctx context.Context, w io.Writer, lister agentSessionLister, reconciler agentsession.Reconciler, asJSON bool, onlySessionID string) error {
	if w == nil {
		w = io.Discard
	}
	leases, err := lister()
	if err != nil {
		return fmt.Errorf("list agent sessions: %w", err)
	}
	sessions := reconciler.Sessions(ctx, leases)
	if onlySessionID != "" {
		filtered := sessions[:0]
		for _, session := range sessions {
			if session.SessionID == onlySessionID {
				filtered = append(filtered, session)
			}
		}
		sessions = filtered
	}
	now := time.Now().UTC()
	if reconciler.Now != nil {
		now = reconciler.Now().UTC()
	}
	rows := make([]AgentSessionRow, 0, len(sessions))
	summary := AgentListSummary{
		ByOwnerKind: agentsession.CountByOwnerKind(sessions), Sessions: len(sessions),
		MemoryBytesBySource: map[string]int64{}, MemoryPartialBySource: map[string]int64{},
		SwapBytesBySource: map[string]int64{}, SwapPartialBySource: map[string]int64{},
	}
	for _, session := range sessions {
		row := agentSessionRow(session, now)
		row.Reapable, row.ReapReason = agentsession.ReapDecision(session, agentsession.ReapPolicy{}, now)
		summary.add(row)
		rows = append(rows, row)
	}
	if asJSON {
		return cliout.WriteJSON(w, map[string]any{"sessions": rows, "summary": summary})
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(w, "No live agent sessions are recorded in the runtime registry.")
		return nil
	}
	table := make([][]string, 0, len(rows))
	for _, row := range rows {
		owner := row.Owner.Kind
		if row.Owner.Ref != "" {
			owner += ":" + shortRef(row.Owner.Ref)
		}
		table = append(table, []string{row.Session, row.LeaseStatus, row.Agent, row.State, owner, row.Owner.Liveness, row.Tree, row.Scope, fmt.Sprint(row.PID), row.Age, usageText(row.MemoryBytes, row.MemoryStatus, row.MemorySource), usageText(row.SwapBytes, row.SwapStatus, row.SwapSource), strings.Join(row.Claims, ","), row.Frozen})
	}
	return cliout.RenderTable(w, []string{"Session", "Lease", "Agent", "State", "Owner", "Owner State", "Tree", "Scope", "PID", "Age", "RAM (source)", "Swap (source)", "Claims", "Frozen"}, table)
}

func usageText(n *int64, status, source string) string {
	value := optionalBytes(n, status)
	if source == "" {
		return value
	}
	return value + " (" + source + ")"
}

func (s *AgentListSummary) add(row AgentSessionRow) {
	if row.MemoryBytes == nil || (row.MemoryStatus != agentsession.UsageMeasured && row.MemoryStatus != agentsession.UsagePartial) {
		s.MemoryUnknown++
	} else if row.MemoryStatus == agentsession.UsagePartial {
		s.MemoryPartial++
		s.MemoryPartialBytes += *row.MemoryBytes
		if row.MemorySource != "" {
			if s.MemoryPartialBySource == nil {
				s.MemoryPartialBySource = map[string]int64{}
			}
			s.MemoryPartialBySource[row.MemorySource] += *row.MemoryBytes
		}
	} else {
		s.MemoryBytes += *row.MemoryBytes
		if row.MemorySource != "" {
			if s.MemoryBytesBySource == nil {
				s.MemoryBytesBySource = map[string]int64{}
			}
			s.MemoryBytesBySource[row.MemorySource] += *row.MemoryBytes
		}
	}
	if row.SwapBytes == nil || (row.SwapStatus != agentsession.UsageMeasured && row.SwapStatus != agentsession.UsagePartial) {
		s.SwapUnknown++
	} else if row.SwapStatus == agentsession.UsagePartial {
		s.SwapPartial++
		s.SwapPartialBytes += *row.SwapBytes
		if row.SwapSource != "" {
			if s.SwapPartialBySource == nil {
				s.SwapPartialBySource = map[string]int64{}
			}
			s.SwapPartialBySource[row.SwapSource] += *row.SwapBytes
		}
	} else {
		s.SwapBytes += *row.SwapBytes
		if row.SwapSource != "" {
			if s.SwapBytesBySource == nil {
				s.SwapBytesBySource = map[string]int64{}
			}
			s.SwapBytesBySource[row.SwapSource] += *row.SwapBytes
		}
	}
	if row.Reapable {
		s.Reapable++
		if row.MemoryBytes != nil && row.MemoryStatus == agentsession.UsageMeasured {
			s.ReapableMemoryBytes += *row.MemoryBytes
		} else if row.MemoryBytes != nil && row.MemoryStatus == agentsession.UsagePartial {
			s.ReapableMemoryPartialBytes += *row.MemoryBytes
		}
		if row.SwapBytes != nil && row.SwapStatus == agentsession.UsageMeasured {
			s.ReapableSwapBytes += *row.SwapBytes
		} else if row.SwapBytes != nil && row.SwapStatus == agentsession.UsagePartial {
			s.ReapableSwapPartialBytes += *row.SwapBytes
		}
	}
}

func agentSessionRow(session agentsession.Session, now time.Time) AgentSessionRow {
	row := AgentSessionRow{
		MemoryBytes: session.Usage.MemoryBytes, MemoryStatus: session.Usage.MemoryStatus,
		MemorySource: session.Usage.MemorySource,
		SwapBytes:    session.Usage.SwapBytes, SwapStatus: session.Usage.SwapStatus,
		SwapSource: session.Usage.SwapSource, UsageDetail: session.Usage.Detail,
		Session: session.SessionID, Harness: session.Harness, Agent: session.Agent, Tree: session.Tree, Scope: session.Scope,
		LeaseStatus: session.LeaseStatus, BindingID: session.BindingID, ScopeHeld: session.ScopeHeld,
		PID: session.PID, Claims: session.Claims, Frozen: session.Frozen, State: session.State, KernelState: session.KernelState,
		Tasks: session.Tasks,
		Owner: AgentOwnerView{
			Kind: session.Owner.Kind, Ref: session.Owner.Ref, Alive: session.Liveness.Alive(), Liveness: session.Liveness.State,
			Detail: session.Liveness.Detail, CheckedAt: session.Liveness.CheckedAt.UTC().Format(time.RFC3339),
			Source: session.Owner.Source, Evidence: session.Owner.Evidence,
		},
	}
	if !session.StartedAt.IsZero() {
		row.Age = now.Sub(session.StartedAt).Round(time.Second).String()
	}
	if session.Leased {
		row.Heartbeat = session.LastHeartbeat.UTC().Format(time.RFC3339)
	}
	if !session.LastActivity.IsZero() {
		row.LastActivity = session.LastActivity.UTC().Format(time.RFC3339)
	}
	return row
}

// shortRef keeps a table cell readable; JSON carries the full ref.
func shortRef(ref string) string {
	const keep = 36
	if len(ref) <= keep {
		return ref
	}
	return "…" + ref[len(ref)-keep+1:]
}
