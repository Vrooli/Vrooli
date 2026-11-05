package vroolicli

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	proofRunID            = "am-run-123"
	proofHarnessSessionID = "host-session-456"
	proofWCID             = "wc-789"
	proofUnit             = "vrooli-agent-run-123.scope"
	proofPath             = "/user.slice/user-1000.slice/user@1000.service/vrooli.slice/vrooli-agents.slice/vrooli-agent-run-123.scope"
)

type proofLeaseReader struct {
	leases    []scenarioruntime.EditorLease
	calls     int
	selectors []string
	cancel    context.CancelFunc
}

func (r *proofLeaseReader) GetEditorLease(_ context.Context, sessionID string) (scenarioruntime.EditorLease, error) {
	r.calls++
	r.selectors = append(r.selectors, sessionID)
	if r.cancel != nil && r.calls == 1 {
		r.cancel()
	}
	index := r.calls - 1
	if index >= len(r.leases) {
		index = len(r.leases) - 1
	}
	if index < 0 {
		return scenarioruntime.EditorLease{}, errors.New("missing fixture lease")
	}
	return r.leases[index], nil
}

type proofHost struct {
	memberUnit string
	members    []int
	fact       agentsession.ProcessFact
	started    time.Time
	memberErr  error
}

func (h *proofHost) Process(int) (agentsession.ProcessFact, error) { return h.fact, nil }
func (h *proofHost) StartedAt(int) (time.Time, error)              { return h.started, nil }
func (h *proofHost) ScopeMembers(unit string) ([]int, error) {
	if unit != h.memberUnit {
		return nil, errors.New("unexpected scope unit")
	}
	return h.members, h.memberErr
}

func proofFixture() (identityBindingDeps, *proofLeaseReader, *proofHost, time.Time) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	created := now.Add(-time.Hour)
	evidence, _ := json.Marshal(agentsession.Evidence{
		ReadFromPID: 4086, WebConsoleSessionID: proofWCID,
		WebConsoleStateRoot: "/run/user/1000/web-console",
	})
	lease := scenarioruntime.EditorLease{
		SessionID: proofHarnessSessionID, Harness: "codex", Agent: "codex", PID: 4086,
		HostBootID: "boot-1", Scope: "vrooli-agent-run-123", Status: scenarioruntime.EditorLeaseActive,
		CreatedAt: created, LastHeartbeatAt: now.Add(-30 * 24 * time.Hour),
		Owner: scenarioruntime.EditorLeaseOwner{
			Kind: agentsession.KindWebConsoleSession, Ref: proofWCID,
			Source: agentsession.SourceLaunch, EvidenceJSON: string(evidence),
		},
	}
	reader := &proofLeaseReader{leases: []scenarioruntime.EditorLease{lease, lease}}
	host := &proofHost{
		memberUnit: proofUnit, members: []int{4086},
		fact:    agentsession.ProcessFact{PID: 4086, PPID: 100, StartTime: "385828408"},
		started: created.Add(-time.Second),
	}
	deps := identityBindingDeps{
		lease: reader, host: host,
		observe: func(pid int) (cliutil.ProcessBinding, error) {
			return cliutil.ProcessBinding{PID: pid, UID: 1000, Start: "385828408", Boot: "boot-1", Cgroup: "0::" + proofPath}, nil
		},
		webConsole: func(_ context.Context, _, sessionID string) (agentsession.WebConsoleSession, error) {
			return agentsession.WebConsoleSession{ID: sessionID}, nil
		},
		now: func() time.Time { return now },
		scopePath: func(unit string) (string, error) {
			if unit != proofUnit {
				return "", errors.New("unexpected normalized scope")
			}
			return proofPath, nil
		},
	}
	return deps, reader, host, now
}

// TestProveAgentIdentityBindingUsesHarnessNotAgent guards the claude-specific
// field distinction: a claude session records Harness "claude-code" but Agent
// "claude". The proof validates against Harness, so the authorize request must
// send harness_kind=lease.Harness. Sending Agent ("claude") produced a spurious
// lease_identity_mismatch that blocked claude sessions from requesting
// authorization at all. (codex/grok never hit this because Harness==Agent.)
func TestProveAgentIdentityBindingUsesHarnessNotAgent(t *testing.T) {
	makeClaudeDeps := func() identityBindingDeps {
		deps, reader, _, _ := proofFixture()
		for i := range reader.leases {
			reader.leases[i].Harness = "claude-code"
			reader.leases[i].Agent = "claude"
		}
		return deps
	}
	// The harness kind the fixed CLI sends (lease.Harness) binds.
	if _, err := proveAgentIdentityBinding(context.Background(), makeClaudeDeps(), proofRunID, proofHarnessSessionID, proofWCID, "claude-code"); err != nil {
		t.Fatalf("valid claude binding (harness_kind=claude-code) rejected: %v", err)
	}
	// The agent value (the old bug) must be rejected, not silently accepted.
	if _, err := proveAgentIdentityBinding(context.Background(), makeClaudeDeps(), proofRunID, proofHarnessSessionID, proofWCID, "claude"); err == nil || err.Error() != "lease_identity_mismatch" {
		t.Fatalf("harness_kind=claude (the old bug) must fail lease_identity_mismatch, got: %v", err)
	}
}

func TestProveAgentIdentityBindingAcceptsExactLaunchJoinAndStaleHeartbeat(t *testing.T) {
	deps, _, _, _ := proofFixture()
	got, err := proveAgentIdentityBinding(context.Background(), deps, proofRunID, proofHarnessSessionID, proofWCID, "codex")
	if err != nil {
		t.Fatalf("proof rejected exact live launch binding: %v", err)
	}
	if got.RunID != proofRunID || got.HarnessSessionID != proofHarnessSessionID || got.WebConsoleSessionID != proofWCID || got.OwnerKind != agentsession.KindWebConsoleSession || got.Binding.PID != 4086 || got.Binding.Cgroup != "0::"+proofPath {
		t.Fatalf("proof did not preserve the exact joined binding: %+v", got)
	}
	if len(deps.lease.(*proofLeaseReader).selectors) != 2 || deps.lease.(*proofLeaseReader).selectors[0] != proofHarnessSessionID || deps.lease.(*proofLeaseReader).selectors[1] != proofHarnessSessionID {
		t.Fatalf("lease was not selected by host session on both reads: %q", deps.lease.(*proofLeaseReader).selectors)
	}
}

func TestProveAgentIdentityBindingSeparatesRunAndHostSessionSelectors(t *testing.T) {
	deps, reader, _, _ := proofFixture()
	got, err := proveAgentIdentityBinding(context.Background(), deps, "different-am-run", proofHarnessSessionID, "", "codex")
	if err != nil {
		t.Fatalf("proof required AM run id to equal host lease id: %v", err)
	}
	if got.RunID != "different-am-run" || got.HarnessSessionID != proofHarnessSessionID || got.WebConsoleSessionID != proofWCID {
		t.Fatalf("proof tuple collapsed run and host-session identities: %+v", got)
	}
	if len(reader.selectors) != 2 || reader.selectors[0] != proofHarnessSessionID {
		t.Fatalf("lease selector = %q, want host session %q", reader.selectors, proofHarnessSessionID)
	}
}

func TestProveAgentIdentityBindingPreservesNonWebConsoleLeaseWithoutLocator(t *testing.T) {
	deps, _, _, _ := proofFixture()
	reader := deps.lease.(*proofLeaseReader)
	for i := range reader.leases {
		owner := scenarioruntime.EditorLeaseOwner{
			Kind: agentsession.KindTerminal, Ref: "tty:/dev/pts/7", Source: agentsession.SourceLaunch,
		}
		evidence, _ := json.Marshal(agentsession.Evidence{ReadFromPID: reader.leases[i].PID, Terminal: "/dev/pts/7"})
		owner.EvidenceJSON = string(evidence)
		reader.leases[i].Owner = owner
	}
	webConsoleCalls := 0
	deps.webConsole = func(context.Context, string, string) (agentsession.WebConsoleSession, error) {
		webConsoleCalls++
		return agentsession.WebConsoleSession{}, errors.New("non-Web Console owner must not be probed")
	}
	got, err := proveAgentIdentityBinding(context.Background(), deps, "", proofHarnessSessionID, "", "codex")
	if err != nil || got.RunID != "" || got.HarnessSessionID != proofHarnessSessionID || got.WebConsoleSessionID != "" || got.OwnerKind != agentsession.KindTerminal {
		t.Fatalf("non-Web Console launch was not preserved as an ownerless proof: proof=%+v err=%v", got, err)
	}
	if webConsoleCalls != 0 {
		t.Fatalf("non-Web Console owner triggered %d Web Console probes", webConsoleCalls)
	}

	// Older or ownerless active leases can still use the existing ordinary
	// identity path. They prove the exact host process, but make no Web Console
	// ownership claim and therefore must not trigger a Web Console lookup.
	for i := range reader.leases {
		reader.leases[i].Owner = scenarioruntime.EditorLeaseOwner{}
	}
	reader.calls = 0
	got, err = proveAgentIdentityBinding(context.Background(), deps, "", proofHarnessSessionID, "", "codex")
	if err != nil || got.WebConsoleSessionID != "" || got.OwnerKind != "unknown" {
		t.Fatalf("ownerless lease was not preserved without a Web Console claim: proof=%+v err=%v", got, err)
	}
	if webConsoleCalls != 0 {
		t.Fatalf("ownerless lease triggered %d Web Console probes", webConsoleCalls)
	}
}

func TestIdentityBindingRetriesTransientSessionReadWithoutWeakeningProof(t *testing.T) {
	for _, tc := range []struct {
		name        string
		first       agentsession.WebConsoleSession
		firstErr    error
		wantCalls   int
		wantSuccess bool
	}{
		{name: "transient", firstErr: errors.New("temporary owner read failure"), wantCalls: 2, wantSuccess: true},
		{name: "missing", firstErr: agentsession.ErrNotFound, wantCalls: 1},
		{name: "wrong-session", first: agentsession.WebConsoleSession{ID: "different"}, wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, _, _, _ := proofFixture()
			calls := 0
			deps.webConsole = func(_ context.Context, _, sessionID string) (agentsession.WebConsoleSession, error) {
				calls++
				if calls == 1 {
					return tc.first, tc.firstErr
				}
				return agentsession.WebConsoleSession{ID: sessionID}, nil
			}
			_, err := proveAgentIdentityBinding(context.Background(), deps, "", proofHarnessSessionID, proofWCID, "codex")
			if (err == nil) != tc.wantSuccess || calls != tc.wantCalls {
				t.Fatalf("proof err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestProveAgentIdentityBindingRefusesUnverifiedJoinWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*identityBindingDeps, *proofLeaseReader, *proofHost)
	}{
		{name: "missing lease", setup: func(_ *identityBindingDeps, r *proofLeaseReader, _ *proofHost) { r.leases = nil }},
		{name: "backfilled owner", setup: func(d *identityBindingDeps, _ *proofLeaseReader, _ *proofHost) {
			r := d.lease.(*proofLeaseReader)
			r.leases[0].Owner.Source = agentsession.SourceBackfill
			r.leases[1].Owner.Source = agentsession.SourceBackfill
		}},
		{name: "wrong web console session", setup: func(d *identityBindingDeps, _ *proofLeaseReader, _ *proofHost) {
			d.webConsole = func(context.Context, string, string) (agentsession.WebConsoleSession, error) {
				return agentsession.WebConsoleSession{ID: "other"}, nil
			}
		}},
		{name: "wrong process id", setup: func(d *identityBindingDeps, _ *proofLeaseReader, _ *proofHost) {
			d.observe = func(int) (cliutil.ProcessBinding, error) {
				return cliutil.ProcessBinding{PID: 999, UID: 1000, Start: "385828408", Boot: "boot-1", Cgroup: "0::" + proofPath}, nil
			}
		}},
		{name: "scope member absent", setup: func(_ *identityBindingDeps, _ *proofLeaseReader, h *proofHost) { h.members = []int{1} }},
		{name: "invalid scope name", setup: func(d *identityBindingDeps, _ *proofLeaseReader, _ *proofHost) {
			r := d.lease.(*proofLeaseReader)
			r.leases[0].Scope = "outside/other"
			r.leases[1].Scope = "outside/other"
		}},
		{name: "wrong exact cgroup", setup: func(d *identityBindingDeps, _ *proofLeaseReader, _ *proofHost) {
			d.observe = func(pid int) (cliutil.ProcessBinding, error) {
				return cliutil.ProcessBinding{PID: pid, UID: 1000, Start: "385828408", Boot: "boot-1", Cgroup: "0::/user.slice/other.scope"}, nil
			}
		}},
		{name: "boot mismatch", setup: func(d *identityBindingDeps, _ *proofLeaseReader, _ *proofHost) {
			d.observe = func(pid int) (cliutil.ProcessBinding, error) {
				return cliutil.ProcessBinding{PID: pid, UID: 1000, Start: "385828408", Boot: "boot-2", Cgroup: "0::" + proofPath}, nil
			}
		}},
		{name: "process born after lease", setup: func(_ *identityBindingDeps, _ *proofLeaseReader, h *proofHost) {
			h.started = time.Date(2026, 10, 18, 11, 30, 0, 0, time.UTC)
		}},
		{name: "lease changed during proof", setup: func(d *identityBindingDeps, _ *proofLeaseReader, _ *proofHost) {
			r := d.lease.(*proofLeaseReader)
			changed := r.leases[1]
			changed.PID++
			r.leases[1] = changed
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, reader, _, _ := proofFixture()
			tc.setup(&deps, reader, deps.host.(*proofHost))
			if _, err := proveAgentIdentityBinding(context.Background(), deps, proofRunID, proofHarnessSessionID, proofWCID, "codex"); err == nil {
				t.Fatal("unverified binding was accepted")
			}
			if reader.calls == 0 || reader.calls > 2 {
				t.Fatalf("unexpected lease read count after refusal: %d", reader.calls)
			}
		})
	}
}

func TestProveAgentIdentityBindingRefusesChangedKernelBindingBetweenReads(t *testing.T) {
	deps, _, _, _ := proofFixture()
	observations := 0
	deps.observe = func(pid int) (cliutil.ProcessBinding, error) {
		observations++
		binding := cliutil.ProcessBinding{PID: pid, UID: 1000, Start: "385828408", Boot: "boot-1", Cgroup: "0::" + proofPath}
		if observations == 2 {
			binding.UID = 1001
		}
		return binding, nil
	}
	if _, err := proveAgentIdentityBinding(context.Background(), deps, proofRunID, proofHarnessSessionID, proofWCID, "codex"); err == nil || !strings.Contains(err.Error(), "process_changed") {
		t.Fatalf("changed process binding was accepted: %v", err)
	}
	if observations != 2 {
		t.Fatalf("expected both kernel observations, got %d", observations)
	}
}

func TestProveAgentIdentityBindingHonorsCancellationBetweenReads(t *testing.T) {
	deps, reader, _, _ := proofFixture()
	ctx, cancel := context.WithCancel(context.Background())
	reader.cancel = cancel
	if _, err := proveAgentIdentityBinding(ctx, deps, proofRunID, proofHarnessSessionID, proofWCID, "codex"); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("canceled proof did not fail closed: %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("continued proof after cancellation: lease reads=%d", reader.calls)
	}
}

func TestAgentCommandRecognizesIdentityBinding(t *testing.T) {
	app := &App{}
	ctx := &AppContext{Stderr: &strings.Builder{}}
	err := app.runAgentCommand(ctx, []string{"identity-binding", "--json"})
	if err == nil || !strings.Contains(err.Error(), "--harness-session-id") || strings.Contains(err.Error(), "unsupported agent subcommand") {
		t.Fatalf("identity-binding did not reach its selector validation: %v", err)
	}
}

func TestRegisteredLeafPathsIncludesIdentityBinding(t *testing.T) {
	paths, err := RegisteredLeafPaths()
	if err != nil {
		t.Fatalf("registered leaf paths: %v", err)
	}
	for _, path := range paths {
		if path == "agent identity-binding" {
			return
		}
	}
	t.Fatal("agent identity-binding is invocable but absent from registered command paths")
}

type exactProofLeases map[string]scenarioruntime.EditorLease

func (r exactProofLeases) GetEditorLease(_ context.Context, id string) (scenarioruntime.EditorLease, error) {
	if lease, ok := r[id]; ok {
		return lease, nil
	}
	return scenarioruntime.EditorLease{}, scenarioruntime.ErrNotFound
}

func TestAttachedRunProofCorrelatesExactLeaseAndProcess(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		legacy, badPID, conflict bool
	}{
		{name: "canonical host key"}, {name: "existing run key", legacy: true}, {name: "wrong process", legacy: true, badPID: true}, {name: "conflicting canonical lease", legacy: true, conflict: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deps, reader, _, _ := proofFixture()
			lease := reader.leases[0]
			leases := exactProofLeases{}
			if tc.legacy {
				lease.SessionID = proofRunID
			}
			leases[lease.SessionID] = lease
			if tc.conflict {
				conflict := lease
				conflict.SessionID = proofHarnessSessionID
				conflict.PID++
				leases[conflict.SessionID] = conflict
			}
			deps.lease = leases
			pid := lease.PID
			if tc.badPID {
				pid++
			}
			proof, err := proveAgentRunBinding(context.Background(), deps, proofRunID, proofHarnessSessionID, proofWCID, "codex", pid)
			if tc.badPID || tc.conflict {
				if err == nil {
					t.Fatal("run mismatch borrowed another lease")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if proof.ProofVersion != cliutil.AgentSessionBindingProofVersion || proof.HarnessSessionID != proofHarnessSessionID || proof.LeaseSessionID != lease.SessionID || proof.Binding.PID != pid {
				t.Fatalf("proof conflated run anchor and actual host lease: %+v", proof)
			}
		})
	}
}
