package agentsession

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Session states. The first five are what the kernel agreed to; orphaned and
// idle are derived from the owner and from activity.
const (
	// StateLive is a session whose process is running inside a scope the
	// kernel holds, with an owner that is present or cannot be judged.
	StateLive = "live"
	// StateGhost is a lease whose process is gone; the registry expires it
	// on the next sweep.
	StateGhost = "ghost"
	// StateAgentless is a scope holding processes with no coding agent among
	// them: the agent exited and left long-running children. (Called
	// "orphan" before owners existed; renamed so it cannot be confused with
	// an orphaned owner.)
	StateAgentless = "agentless"
	// StateUnrecorded is a scope the kernel holds that no lease names, or a
	// lease naming a scope the kernel does not hold.
	StateUnrecorded = "unrecorded"
	// StateUndetermined is what an unreadable kernel gets.
	StateUndetermined = "undetermined"
	// StateOrphaned is a session whose owner kind is known and whose owner
	// its authority confirmed gone, while its scope still holds processes.
	StateOrphaned = "orphaned"
	// StateIdle is a live session with no activity for the idle period.
	// Idle is reported, never reaped.
	StateIdle = "idle"
)

// DefaultIdleAfter is how long a live session may show no activity before it
// is reported idle.
const DefaultIdleAfter = 7 * 24 * time.Hour

// Session is one agent session as the host holds it.
type Session struct {
	SessionID     string
	Harness       string
	Agent         string
	Tree          string
	Scope         string
	PID           int
	Claims        []string
	CreatedAt     time.Time
	LastHeartbeat time.Time
	// Leased says a registry lease names the session; a scope-only row has
	// no lease to persist an owner on.
	Leased bool
	// KernelState is the kernel-reconciled state before owner and activity.
	KernelState string
	// ScopeHeld says the kernel holds the session's scope right now.
	ScopeHeld bool
	Tasks     int64
	Frozen    string
	Owner     Owner
	Liveness  Liveness
	// StartedAt is the lease's creation, or the oldest member process's
	// start for a scope no lease names.
	StartedAt time.Time
	// LastActivity is the newest of the lease heartbeat and the controlling
	// terminal's last input or output; zero when nothing was observable.
	LastActivity time.Time
	// State is the derived state.
	State string
	// Usage is what the session's processes hold (MeasureUsage).
	Usage ScopeUsage
}

// ScopeUnit is the session's scope unit name ("vrooli-agent-….scope"), or ""
// when the scope is not a unit name.
func (s Session) ScopeUnit() string {
	scope := strings.TrimSpace(s.Scope)
	if scope == "" || scope == "none" || strings.Contains(scope, ":") {
		return ""
	}
	if !strings.HasSuffix(scope, agentscope.ScopeSuffix) {
		scope += agentscope.ScopeSuffix
	}
	if !agentscope.IsSessionScope(scope) {
		return ""
	}
	return scope
}

// HostFacts reads the host. The live implementation is NewHostFacts; tests
// inject fixtures.
type HostFacts interface {
	// Environment returns a process's environment.
	Environment(pid int) (map[string]string, error)
	// Process returns a process's parent, start stamp and controlling tty.
	Process(pid int) (ProcessFact, error)
	// ProcessName is the kernel's program name for a pid.
	ProcessName(pid int) (string, error)
	// StartedAt converts a process's start to wall time.
	StartedAt(pid int) (time.Time, error)
	// ScopeMembers lists the pids a session scope unit holds.
	ScopeMembers(scopeUnit string) ([]int, error)
	// TTYActivity is the last input or output on a terminal device.
	TTYActivity(tty string) (time.Time, error)
	// ProcessMemory is a process's resident and swapped bytes.
	ProcessMemory(pid int) (rss, swap int64, err error)
}

// OwnerStore persists owners on leases. A nil store reads without writing.
type OwnerStore interface {
	BackfillEditorLeaseOwner(ctx context.Context, sessionID string, owner scenarioruntime.EditorLeaseOwner) (bool, error)
	RefreshEditorLeaseOwnerEvidence(ctx context.Context, sessionID string, owner scenarioruntime.EditorLeaseOwner) (bool, error)
}

// Reconciler joins the lease table, the kernel and the owners.
type Reconciler struct {
	// Census reads the agent slice's scopes.
	Census func() ([]agentscope.Entry, error)
	// Alive reports whether a pid is running.
	Alive func(pid int) bool
	// Frozen reports a scope's freeze state ("yes", "no", "n/a", "unknown").
	Frozen func(scope string) string
	// Host reads process facts; nil leaves owners unknown.
	Host HostFacts
	// Probes ask owners' authorities.
	Probes Probes
	// Store persists backfilled owners; nil does not persist.
	Store OwnerStore
	// IdleAfter is the idle period; zero means DefaultIdleAfter.
	IdleAfter time.Duration
	// Now is the clock.
	Now func() time.Time
}

func (r Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

// Sessions reconciles leases with the kernel and classifies every session's
// owner, liveness and state. A scope the kernel holds that no lease names is
// included: it is the shape that consumed the agent slice.
func (r Reconciler) Sessions(ctx context.Context, leases []scenarioruntime.EditorLease) []Session {
	now := r.now()
	var held []agentscope.Entry
	var censusErr error
	if r.Census != nil {
		held, censusErr = r.Census()
	}
	heldByName := make(map[string]agentscope.Entry, len(held))
	for _, entry := range held {
		heldByName[strings.TrimSuffix(entry.Name, agentscope.ScopeSuffix)] = entry
	}
	named := make(map[string]bool, len(leases))
	sessions := make([]Session, 0, len(leases)+len(held))
	for _, lease := range leases {
		key := strings.TrimSuffix(lease.Scope, agentscope.ScopeSuffix)
		named[key] = true
		entry, isHeld := heldByName[key]
		session := Session{
			SessionID: lease.SessionID, Harness: lease.Harness, Agent: lease.Agent, Tree: lease.WorkingDir,
			Scope: lease.Scope, PID: lease.PID, Claims: lease.Claims, CreatedAt: lease.CreatedAt,
			LastHeartbeat: lease.LastHeartbeatAt, Leased: true, ScopeHeld: isHeld && censusErr == nil,
			StartedAt: lease.CreatedAt,
		}
		if isHeld {
			session.Tasks = entry.Tasks
		}
		session.KernelState = leaseKernelState(lease, entry, isHeld, r.Alive, censusErr)
		if owner, ok := OwnerFromLease(lease.Owner); ok {
			session.Owner = owner
		}
		sessions = append(sessions, session)
	}
	for _, entry := range held {
		if named[strings.TrimSuffix(entry.Name, agentscope.ScopeSuffix)] {
			continue
		}
		state := StateUnrecorded
		if entry.Agentless {
			state = StateAgentless
		}
		sessions = append(sessions, Session{SessionID: "(no lease)", Scope: entry.Name, Tasks: entry.Tasks, KernelState: state, ScopeHeld: true})
	}

	memo := map[string]Liveness{}
	for i := range sessions {
		s := &sessions[i]
		if r.Frozen != nil {
			s.Frozen = r.Frozen(s.Scope)
		}
		r.classify(ctx, s, memo, now)
	}
	return sessions
}

// leaseKernelState reconciles one lease with the kernel.
func leaseKernelState(lease scenarioruntime.EditorLease, entry agentscope.Entry, held bool, alive func(int) bool, censusErr error) string {
	if alive != nil && lease.PID > 0 && !alive(lease.PID) {
		return StateGhost
	}
	if censusErr != nil {
		return StateUndetermined
	}
	if !held {
		// The lease names a scope the kernel does not hold: it was reaped
		// after the session left it, or never created.
		return StateUnrecorded
	}
	if entry.Agentless {
		return StateAgentless
	}
	return StateLive
}

// classify fills the owner, liveness, activity and derived state.
func (r Reconciler) classify(ctx context.Context, s *Session, memo map[string]Liveness, now time.Time) {
	pid, members, membersErr := r.evidenceProcess(*s)
	s.Usage = MeasureUsage(r.Host, members, membersErr)
	var fact ProcessFact
	haveFact := false
	if r.Host != nil && pid > 0 {
		if f, err := r.Host.Process(pid); err == nil {
			fact, haveFact = f, true
		}
	}
	if s.Owner.Kind == "" {
		s.Owner = r.inferOwner(pid, fact)
		if s.Leased && s.Owner.Kind != KindUnknown && r.Store != nil {
			s.Owner.Source = SourceBackfill
			if _, err := r.Store.BackfillEditorLeaseOwner(ctx, s.SessionID, s.Owner.LeaseOwner()); err != nil {
				s.Owner.Source = SourceInferred
			}
		}
	}
	if s.StartedAt.IsZero() && r.Host != nil {
		s.StartedAt = r.oldestStart(members)
	}
	key := strings.Join([]string{s.Owner.Kind, s.Owner.Ref, s.Owner.Evidence.WebConsoleStateRoot, strconv.Itoa(s.Owner.Evidence.AppPID), s.Owner.Evidence.AppStartTime}, "\x00")
	liveness, cached := memo[key]
	if !cached {
		liveness = Check(ctx, s.Owner, r.Probes, now)
		memo[key] = liveness
	}
	s.Liveness = liveness
	if updated, changed := WithObservedOrigin(s.Owner, liveness); changed {
		s.Owner = updated
		if s.Leased && r.Store != nil {
			_, _ = r.Store.RefreshEditorLeaseOwnerEvidence(ctx, s.SessionID, s.Owner.LeaseOwner())
		}
	}
	s.LastActivity = latest(s.LastHeartbeat, s.CreatedAt)
	if haveFact && fact.TTY != "" && r.Host != nil {
		if at, err := r.Host.TTYActivity(fact.TTY); err == nil {
			s.LastActivity = latest(s.LastActivity, at)
		}
	}
	idleAfter := r.IdleAfter
	if idleAfter <= 0 {
		idleAfter = DefaultIdleAfter
	}
	s.State = DeriveState(s.KernelState, s.ScopeHeld, s.Owner, s.Liveness, s.LastActivity, now, idleAfter)
}

// evidenceProcess picks the process whose environment says who owns the
// session: the coding agent itself when the scope holds one, else the
// lease's pid while it runs, else any member of the scope.
func (r Reconciler) evidenceProcess(s Session) (int, []int, error) {
	if r.Host == nil {
		return 0, nil, nil
	}
	var members []int
	var membersErr error
	if unit := s.ScopeUnit(); unit != "" && s.ScopeHeld {
		members, membersErr = r.Host.ScopeMembers(unit)
	}
	for _, pid := range members {
		if name, err := r.Host.ProcessName(pid); err == nil && agentscope.Binaries[strings.ToLower(strings.TrimSpace(name))] {
			return pid, members, membersErr
		}
	}
	if s.PID > 0 && (r.Alive == nil || r.Alive(s.PID)) {
		return s.PID, members, membersErr
	}
	if len(members) > 0 {
		return members[0], members, membersErr
	}
	return 0, members, membersErr
}

func (r Reconciler) inferOwner(pid int, fact ProcessFact) Owner {
	if pid <= 0 {
		return Owner{Kind: KindUnknown, Source: SourceInferred}
	}
	env, err := r.Host.Environment(pid)
	if err != nil {
		return Owner{Kind: KindUnknown, Source: SourceInferred, Evidence: Evidence{ReadFromPID: pid}}
	}
	owner := Infer(env, fact.TTY)
	owner.Source = SourceInferred
	owner.Evidence.ReadFromPID = pid
	return owner
}

func (r Reconciler) oldestStart(pids []int) time.Time {
	var oldest time.Time
	for _, pid := range pids {
		at, err := r.Host.StartedAt(pid)
		if err != nil || at.IsZero() {
			continue
		}
		if oldest.IsZero() || at.Before(oldest) {
			oldest = at
		}
	}
	return oldest
}

// DeriveState combines the kernel's answer with the owner and activity.
//
// A ghost or an undetermined kernel is reported as such: there is nothing to
// judge. A session whose known owner is confirmed gone, while its scope still
// holds processes, is orphaned whatever its agent is doing. A live session
// with no activity for idleAfter is idle. Everything else keeps the kernel's
// state.
func DeriveState(kernelState string, scopeHeld bool, owner Owner, liveness Liveness, lastActivity, now time.Time, idleAfter time.Duration) string {
	switch kernelState {
	case StateGhost, StateUndetermined:
		return kernelState
	}
	if scopeHeld && owner.Known() && liveness.State == OwnerGone {
		return StateOrphaned
	}
	if kernelState == StateLive && idleAfter > 0 && !lastActivity.IsZero() && now.Sub(lastActivity) >= idleAfter {
		return StateIdle
	}
	return kernelState
}

func latest(times ...time.Time) time.Time {
	var out time.Time
	for _, t := range times {
		if t.After(out) {
			out = t
		}
	}
	return out
}

// CountByOwnerKind tallies sessions by owner kind and state, for summaries.
func CountByOwnerKind(sessions []Session) map[string]map[string]int {
	out := map[string]map[string]int{}
	for _, s := range sessions {
		kind := s.Owner.Kind
		if kind == "" {
			kind = KindUnknown
		}
		if out[kind] == nil {
			out[kind] = map[string]int{}
		}
		out[kind][s.State]++
	}
	return out
}

// SortedKinds lists the keys of a tally in a stable order.
func SortedKinds(tally map[string]map[string]int) []string {
	kinds := make([]string, 0, len(tally))
	for kind := range tally {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}
