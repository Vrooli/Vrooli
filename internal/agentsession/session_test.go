package agentsession

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

var now = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func TestUnknownMemberClassificationCannotMakeAReapCandidate(t *testing.T) {
	for _, leased := range []bool{false, true} {
		r, lease := scenarioHelperFixture(nil, scenarioruntime.StatusStopped)
		r.Census = func() ([]agentscope.Entry, error) {
			return []agentscope.Entry{{Name: "vrooli-agent-helper.scope", MembershipError: "permission denied"}}, nil
		}
		var leases []scenarioruntime.EditorLease
		if leased {
			leases = []scenarioruntime.EditorLease{lease}
		}
		sessions := r.Sessions(context.Background(), leases)
		if len(sessions) != 1 || sessions[0].State != StateUndetermined || len(Plan(sessions, ReapPolicy{}, now)) != 0 {
			t.Fatalf("uncertain membership entered reap policy: leased=%v sessions=%+v", leased, sessions)
		}
	}
}

func scenarioHelperFixture(store OwnerStore, status string) (Reconciler, scenarioruntime.EditorLease) {
	host := &fakeHost{
		facts: map[int]ProcessFact{7: {PID: 7, PPID: 1}},
		envs:  map[int]map[string]string{7: {"VROOLI_RUNTIME_INSTANCE_ID": "inst-owner"}},
		names: map[int]string{7: "opencode"}, members: map[string][]int{"vrooli-agent-helper.scope": {7}},
	}
	r := Reconciler{
		Census: func() ([]agentscope.Entry, error) {
			return []agentscope.Entry{{Name: "vrooli-agent-helper.scope"}}, nil
		},
		Alive: func(int) bool { return true }, Host: host, Store: store, Now: func() time.Time { return now },
		Probes: Probes{ScenarioInstance: func(context.Context, string) (scenarioruntime.Instance, error) {
			return scenarioruntime.Instance{InstanceID: "inst-owner", Status: status, StoppedAt: &now}, nil
		}},
	}
	lease := scenarioruntime.EditorLease{
		SessionID: "helper", PID: 7, Scope: "vrooli-agent-helper", CreatedAt: now.Add(-24 * time.Hour),
		Owner: scenarioruntime.EditorLeaseOwner{Kind: KindUnknown, Source: SourceLaunch},
	}
	return r, lease
}

func TestReconcilerIdentifiesUnknownScenarioHelpersAndKeepsTheLiveGeneration(t *testing.T) {
	for _, status := range []string{scenarioruntime.StatusStopped, scenarioruntime.StatusRunning} {
		t.Run(status, func(t *testing.T) {
			store := newFakeOwnerStore()
			r, lease := scenarioHelperFixture(store, status)
			session := r.Sessions(context.Background(), []scenarioruntime.EditorLease{lease})[0]
			if session.Owner.Kind != KindScenarioInstance || session.Owner.Ref != "inst-owner" || session.Owner.Source != SourceBackfill {
				t.Fatalf("owner=%+v", session.Owner)
			}
			if got := store.backfills[lease.SessionID]; got.Kind != KindScenarioInstance || got.Ref != "inst-owner" {
				t.Fatalf("persisted owner=%+v", got)
			}
			reap, reason := ReapDecision(session, ReapPolicy{}, now)
			if reap != (status == scenarioruntime.StatusStopped) {
				t.Fatalf("state=%s reap=%t reason=%s", session.State, reap, reason)
			}
		})
	}
}

type ownerCASStore struct {
	*fakeOwnerStore
	readErr error
}

func (s ownerCASStore) BackfillEditorLeaseOwner(context.Context, string, scenarioruntime.EditorLeaseOwner) (bool, error) {
	return false, nil
}

func (s ownerCASStore) GetEditorLease(_ context.Context, id string) (scenarioruntime.EditorLease, error) {
	return scenarioruntime.EditorLease{SessionID: id, Owner: scenarioruntime.EditorLeaseOwner{Kind: KindTerminal, Ref: "/dev/pts/9", Source: SourceLaunch}}, s.readErr
}

func TestReconcilerRespectsConcurrentOwnerRecordingAndReadUncertainty(t *testing.T) {
	for _, readErr := range []error{nil, errors.New("registry unavailable")} {
		store := ownerCASStore{fakeOwnerStore: newFakeOwnerStore(), readErr: readErr}
		r, lease := scenarioHelperFixture(store, scenarioruntime.StatusStopped)
		session := r.Sessions(context.Background(), []scenarioruntime.EditorLease{lease})[0]
		if reap, reason := ReapDecision(session, ReapPolicy{}, now); reap {
			t.Fatalf("conflicting or unverified owner was reapable: %s", reason)
		}
		if readErr == nil && session.Owner.Kind != KindTerminal {
			t.Fatalf("recorded owner lost: %+v", session.Owner)
		}
		if readErr != nil && session.Owner.Kind != KindUnknown {
			t.Fatalf("unverified owner=%+v", session.Owner)
		}
	}
}

type runtimeOwnerStore struct{ *fakeOwnerStore }

type refiningOwnerStore struct {
	*fakeOwnerStore
	refines int
}

func (s *refiningOwnerStore) RefineBackfilledEditorLeaseOwner(_ context.Context, _ string, _ int, _, _ scenarioruntime.EditorLeaseOwner) (bool, error) {
	s.refines++
	return true, nil
}

func TestReconcilerRefinesOnlyTheMatchingTokenOnlyBackfill(t *testing.T) {
	for _, tc := range []struct {
		name, source, tag string
		refine            bool
	}{
		{"ambient-token backfill", SourceBackfill, "", true},
		{"recorded launch owner", SourceLaunch, "", false},
		{"explicit managed runner", SourceBackfill, "managed-run", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &refiningOwnerStore{fakeOwnerStore: newFakeOwnerStore()}
			r, lease := scenarioHelperFixture(store, scenarioruntime.StatusStopped)
			env := r.Host.(*fakeHost).envs[7]
			env[envIdentityToken] = identityToken(`{"run_id":"maintainer"}`)
			env["OPENCODE_AGENT_TAG"] = tc.tag
			lease.Owner = Owner{Kind: KindAgentManagerRun, Ref: "maintainer", Source: tc.source, Evidence: Evidence{RunID: "maintainer", RunIDSource: "identity_token"}}.LeaseOwner()
			session := r.Sessions(context.Background(), []scenarioruntime.EditorLease{lease})[0]
			if (store.refines == 1) != tc.refine || (session.Owner.Kind == KindScenarioInstance) != tc.refine {
				t.Fatalf("refines=%d owner=%+v", store.refines, session.Owner)
			}
		})
	}
}

func (s runtimeOwnerStore) GetInstance(_ context.Context, id string) (scenarioruntime.Instance, error) {
	return scenarioruntime.Instance{InstanceID: id, Status: scenarioruntime.StatusRunning}, nil
}

func TestLiveReconcilerUsesTheSelectedRegistryForScenarioOwner(t *testing.T) {
	r := LiveReconciler(t.TempDir(), runtimeOwnerStore{newFakeOwnerStore()}, 0)
	got := Check(context.Background(), Owner{Kind: KindScenarioInstance, Ref: "inst-selected"}, r.Probes, now)
	if got.State != OwnerAlive {
		t.Fatalf("selected runtime registry was not used: %+v", got)
	}
}

func TestDeriveState(t *testing.T) {
	gone := Liveness{State: OwnerGone}
	alive := Liveness{State: OwnerAlive}
	unknown := Liveness{State: OwnerUnknown}
	desktop := Owner{Kind: KindDesktopApp, Ref: "/data"}
	terminal := Owner{Kind: KindTerminal, Ref: "/dev/pts/1"}
	recent, stale := now.Add(-time.Hour), now.Add(-8*24*time.Hour)
	cases := []struct {
		name      string
		kernel    string
		held      bool
		owner     Owner
		liveness  Liveness
		activity  time.Time
		wantState string
	}{
		{"known owner gone with a held scope is orphaned", StateLive, true, desktop, gone, recent, StateOrphaned},
		{"agentless scope whose owner is gone is orphaned", StateAgentless, true, desktop, gone, recent, StateOrphaned},
		{"owner gone but scope not held stays as the kernel says", StateUnrecorded, false, desktop, gone, recent, StateUnrecorded},
		{"owner alive stays live", StateLive, true, desktop, alive, recent, StateLive},
		{"owner unknown is never orphaned", StateLive, true, desktop, unknown, recent, StateLive},
		{"terminal owner is never orphaned", StateLive, true, terminal, gone, recent, StateLive},
		{"quiet live session is idle", StateLive, true, terminal, Liveness{State: OwnerNotApplicable}, stale, StateIdle},
		{"orphaned wins over idle", StateLive, true, desktop, gone, stale, StateOrphaned},
		{"no observable activity is not idle", StateLive, true, terminal, Liveness{State: OwnerNotApplicable}, time.Time{}, StateLive},
		{"ghost stays ghost", StateGhost, true, desktop, gone, stale, StateGhost},
		{"undetermined stays undetermined", StateUndetermined, false, desktop, gone, stale, StateUndetermined},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeriveState(tc.kernel, tc.held, tc.owner, tc.liveness, tc.activity, now, DefaultIdleAfter); got != tc.wantState {
				t.Fatalf("DeriveState = %s, want %s", got, tc.wantState)
			}
		})
	}
}

// The promised default idle period is seven days.
func TestDefaultIdleAfterIsSevenDays(t *testing.T) {
	if DefaultIdleAfter != 168*time.Hour {
		t.Fatalf("DefaultIdleAfter = %s, want 168h", DefaultIdleAfter)
	}
}

// Forty-two helpers of one desktop app that has exited, with leases recorded
// before owners existed: each is backfilled from its own environment, the
// app is asked about once, and every one is orphaned.
func TestReconcilerBackfillsAndOrphansDesktopHelpersOfAnExitedApp(t *testing.T) {
	data := "/home/op/.config/web-console-desktop/runtime"
	env := map[string]string{"VROOLI_DESKTOP_MODE": "true", "VROOLI_DATA": data}
	host := &fakeHost{
		facts: map[int]ProcessFact{}, envs: map[int]map[string]string{}, names: map[int]string{},
		members: map[string][]int{},
	}
	var leases []scenarioruntime.EditorLease
	var census []agentscope.Entry
	for i, pid := range []int{501, 502, 503} {
		scope := "vrooli-agent-opencode-" + string(rune('a'+i))
		host.facts[pid] = ProcessFact{PID: pid, PPID: 1, StartTime: "100"}
		host.envs[pid] = env
		host.names[pid] = "opencode"
		host.members[scope+".scope"] = []int{pid}
		leases = append(leases, scenarioruntime.EditorLease{SessionID: scope, Agent: "opencode", PID: pid, Scope: scope, CreatedAt: now.Add(-21 * 24 * time.Hour), LastHeartbeatAt: now.Add(-21 * 24 * time.Hour)})
		census = append(census, agentscope.Entry{Name: scope + ".scope", Tasks: 9})
	}
	calls := &probeCalls{}
	store := newFakeOwnerStore()
	r := Reconciler{
		Census: func() ([]agentscope.Entry, error) { return census, nil },
		Alive:  func(int) bool { return true },
		Host:   host,
		Probes: Probes{DesktopAppProcesses: func(dir string) ([]int, error) {
			calls.inc(&calls.desktop)
			if dir != data {
				t.Fatalf("asked about %s", dir)
			}
			return nil, nil
		}},
		Store: store,
		Now:   func() time.Time { return now },
	}
	sessions := r.Sessions(context.Background(), leases)
	if len(sessions) != 3 {
		t.Fatalf("sessions = %d, want 3", len(sessions))
	}
	for _, s := range sessions {
		if s.State != StateOrphaned || s.Owner.Kind != KindDesktopApp || s.Owner.Ref != data || s.Owner.Source != SourceBackfill {
			t.Fatalf("session %s = state %s owner %+v, want an orphaned backfilled desktop app", s.SessionID, s.State, s.Owner)
		}
		if got := store.backfills[s.SessionID]; got.Kind != KindDesktopApp || got.Ref != data {
			t.Fatalf("backfill of %s = %+v", s.SessionID, got)
		}
	}
	if calls.desktop != 1 {
		t.Fatalf("the app was asked about %d times, want once for all its helpers", calls.desktop)
	}
}

// A recorded owner is the authority; the session's current environment does
// not replace it, and a recorded owner is not backfilled again.
func TestReconcilerKeepsTheRecordedOwner(t *testing.T) {
	host := &fakeHost{
		facts: map[int]ProcessFact{7: {PID: 7, PPID: 1}}, envs: map[int]map[string]string{7: {"CODEX_AGENT_TAG": "x"}},
		names: map[int]string{7: "codex"}, members: map[string][]int{"vrooli-agent-c.scope": {7}},
	}
	store := newFakeOwnerStore()
	lease := scenarioruntime.EditorLease{
		SessionID: "c", PID: 7, Scope: "vrooli-agent-c", CreatedAt: now.Add(-time.Hour), LastHeartbeatAt: now,
		Owner: scenarioruntime.EditorLeaseOwner{Kind: KindTerminal, Ref: "/dev/pts/4", Source: SourceLaunch},
	}
	r := Reconciler{
		Census: func() ([]agentscope.Entry, error) { return []agentscope.Entry{{Name: "vrooli-agent-c.scope"}}, nil },
		Host:   host, Store: store, Now: func() time.Time { return now },
	}
	sessions := r.Sessions(context.Background(), []scenarioruntime.EditorLease{lease})
	if got := sessions[0]; got.Owner.Kind != KindTerminal || got.Owner.Source != SourceLaunch || got.State != StateLive {
		t.Fatalf("session = %+v", got)
	}
	if len(store.backfills) != 0 {
		t.Fatalf("a recorded owner was backfilled: %+v", store.backfills)
	}
}

// A live web-console session's reported origin is folded into the recorded
// evidence, because web-console stops answering once the session is archived.
func TestReconcilerRecordsTheObservedWebConsoleOrigin(t *testing.T) {
	lease := scenarioruntime.EditorLease{
		SessionID: "w", PID: 9, Scope: "vrooli-agent-w", CreatedAt: now.Add(-time.Hour), LastHeartbeatAt: now,
		Owner: scenarioruntime.EditorLeaseOwner{Kind: KindWebConsoleSession, Ref: "wc-1", Source: SourceLaunch, EvidenceJSON: `{"web_console_session_id":"wc-1"}`},
	}
	store := newFakeOwnerStore()
	r := Reconciler{
		Census: func() ([]agentscope.Entry, error) { return []agentscope.Entry{{Name: "vrooli-agent-w.scope"}}, nil },
		Probes: Probes{WebConsoleSession: func(context.Context, string, string) (WebConsoleSession, error) {
			return WebConsoleSession{Origin: OriginUI}, nil
		}},
		Store: store, Now: func() time.Time { return now },
	}
	sessions := r.Sessions(context.Background(), []scenarioruntime.EditorLease{lease})
	if sessions[0].Owner.Evidence.WebConsoleOrigin != OriginUI {
		t.Fatalf("evidence = %+v, want origin ui", sessions[0].Owner.Evidence)
	}
	refreshed, ok := OwnerFromLease(store.refreshes["w"])
	if !ok || refreshed.Evidence.WebConsoleOrigin != OriginUI {
		t.Fatalf("refresh = %+v, want the observed origin persisted", store.refreshes["w"])
	}
}

// A scope no lease names is classified from its own members and is never
// persisted (there is no row), and its age comes from its oldest member.
func TestReconcilerClassifiesAScopeNoLeaseNames(t *testing.T) {
	started := now.Add(-48 * time.Hour)
	host := &fakeHost{
		facts:   map[int]ProcessFact{11: {PID: 11, PPID: 1}, 12: {PID: 12, PPID: 11}},
		envs:    map[int]map[string]string{11: {"VROOLI_AGENT_IDENTITY_TOKEN": identityToken(`{"run_id":"run-x"}`)}},
		names:   map[int]string{11: "claude", 12: "node"},
		started: map[int]time.Time{11: started, 12: now},
		members: map[string][]int{"vrooli-agent-z.scope": {12, 11}},
		memory:  map[int][2]int64{11: {4096, 1024}, 12: {8192, 0}},
	}
	store := newFakeOwnerStore()
	r := Reconciler{
		Census: func() ([]agentscope.Entry, error) {
			return []agentscope.Entry{{Name: "vrooli-agent-z.scope", Tasks: 4}}, nil
		},
		Host: host, Store: store, Now: func() time.Time { return now },
		Probes: Probes{RunStatus: func(context.Context, string) (string, error) { return "RUN_STATUS_FAILED", nil }},
	}
	sessions := r.Sessions(context.Background(), nil)
	if len(sessions) != 1 {
		t.Fatalf("sessions = %+v", sessions)
	}
	got := sessions[0]
	if got.Owner.Kind != KindAgentManagerRun || got.Owner.Ref != "run-x" || got.Owner.Source != SourceInferred || got.Owner.Evidence.ReadFromPID != 11 {
		t.Fatalf("owner = %+v, want the run read from the coding agent's own environment", got.Owner)
	}
	if got.State != StateOrphaned || !got.StartedAt.Equal(started) {
		t.Fatalf("state %s started %s, want orphaned started %s", got.State, got.StartedAt, started)
	}
	if len(store.backfills) != 0 {
		t.Fatal("a scope with no lease was backfilled")
	}
	if got.Usage.MemoryStatus != UsageMeasured || *got.Usage.MemoryBytes != 12288 || *got.Usage.SwapBytes != 1024 {
		t.Fatalf("usage = %+v, want the members' summed VmRSS and VmSwap", got.Usage)
	}
}

// Terminal activity keeps a session out of idle even when its lease never
// heartbeats (the exec branch cannot).
func TestReconcilerTerminalActivityKeepsASessionLive(t *testing.T) {
	host := &fakeHost{
		facts: map[int]ProcessFact{5: {PID: 5, PPID: 1, TTY: "/dev/pts/5"}}, envs: map[int]map[string]string{5: {}},
		names: map[int]string{5: "claude"}, members: map[string][]int{"vrooli-agent-t.scope": {5}},
		activity: map[string]time.Time{"/dev/pts/5": now.Add(-time.Minute)},
	}
	old := now.Add(-30 * 24 * time.Hour)
	r := Reconciler{
		Census: func() ([]agentscope.Entry, error) { return []agentscope.Entry{{Name: "vrooli-agent-t.scope"}}, nil },
		Host:   host, Now: func() time.Time { return now },
	}
	lease := scenarioruntime.EditorLease{SessionID: "t", PID: 5, Scope: "vrooli-agent-t", CreatedAt: old, LastHeartbeatAt: old}
	if got := r.Sessions(context.Background(), []scenarioruntime.EditorLease{lease})[0]; got.State != StateLive || got.Owner.Kind != KindTerminal {
		t.Fatalf("session = state %s owner %+v, want a live terminal session", got.State, got.Owner)
	}
	host.activity = map[string]time.Time{}
	if got := r.Sessions(context.Background(), []scenarioruntime.EditorLease{lease})[0]; got.State != StateIdle {
		t.Fatalf("a month without activity = %s, want idle", got.State)
	}
}
