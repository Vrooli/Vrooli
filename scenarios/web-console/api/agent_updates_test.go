package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/vrooli/packages/capabilityprobe"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
)

func TestAgentUpdateRequiresMatchingFreshTargetEvidence(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	base := failedAgentUpdate("bridge-node:one", "codex", "codex", "")
	rep := harnessUpdateReport{Name: "codex", State: agentStateUpdateReady, PrimaryAction: "update", Installed: "0.165.0", Latest: "0.166.0", Pinned: "0.165.0", CommandPath: "/bin/codex", OS: "linux", Arch: "amd64", CheckedAt: now}
	probe := capabilityprobe.Observation{ID: "codex", State: capabilityprobe.Ready, Path: "/bin/codex", Version: "codex-cli 0.165.0", ProbedAt: now}
	cases := []struct {
		name   string
		change func(*harnessUpdateReport, *capabilityprobe.Observation)
		want   string
	}{
		{"matching", func(*harnessUpdateReport, *capabilityprobe.Observation) {}, agentStateUpdateReady},
		{"found copy remains available without a managed action", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) {
			r.State, r.PrimaryAction, r.InstallState, r.SignInState, r.UpdateState = agentStateFound, "", "found", "unknown", "blocked(found_copy)"
		}, agentStateFound},
		{"different executable", func(_ *harnessUpdateReport, p *capabilityprobe.Observation) { p.Path = "/other/codex" }, agentStateCheckFailed},
		{"different installed version", func(_ *harnessUpdateReport, p *capabilityprobe.Observation) { p.Version = "0.164.0" }, agentStateCheckFailed},
		{"stale capability", func(_ *harnessUpdateReport, p *capabilityprobe.Observation) { p.ProbedAt = now.Add(-3 * time.Minute) }, agentStateCheckFailed},
		{"future observation", func(_ *harnessUpdateReport, p *capabilityprobe.Observation) { p.ProbedAt = now.Add(time.Second) }, agentStateCheckFailed},
		{"old resource check", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) { r.CheckedAt = time.Time{} }, agentStateCheckFailed},
		{"different platform", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) { r.Arch = "arm64" }, agentStateCheckFailed},
		{"prerelease", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) { r.Latest = "0.166.0-rc.1" }, agentStateCheckFailed},
		{"missing upstream", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) { r.Latest = "" }, agentStateCheckFailed},
		{"ready but not behind", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) { r.Latest = "0.165.0" }, agentStateCheckFailed},
		{"unknown state", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) { r.State = "maybe" }, agentStateCheckFailed},
		{"current", func(r *harnessUpdateReport, _ *capabilityprobe.Observation) {
			r.State, r.PrimaryAction, r.Latest = agentStateUpToDate, "", "0.165.0"
		}, agentStateUpToDate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, p := rep, probe
			tc.change(&r, &p)
			got := projectAgentUpdate(base, r, p, "linux", "amd64", now)
			if got.State != tc.want {
				t.Fatalf("state %q, want %q: %+v", got.State, tc.want, got)
			}
			if tc.want == agentStateFound && (got.InstallState != "found" || got.SignInState != "unknown" || got.UpdateState != "blocked(found_copy)") {
				t.Fatalf("three-fact state lost in API projection: %+v", got)
			}
			if tc.want == agentStateCheckFailed && (got.PrimaryAction != agentActionRetry || got.Status != upstreamcheck.StatusUnknown) {
				t.Fatalf("a failed check offers retry and no update indicator: %+v", got)
			}
			if tc.want == agentStateUpdateReady && !got.ProbeVerified {
				t.Fatal("matching installed tool must be verified")
			}
		})
	}
}

func TestDecodeGrokSignInStateRequiresTheResourceOwnedContract(t *testing.T) {
	cases := []struct {
		name string
		data string
		want string
	}{
		{"signed out", `{"success":true,"sign_in_state":"signed_out"}`, "signed_out"},
		{"signed in", `{"success":true,"sign_in_state":"signed_in"}`, "signed_in"},
		{"resource unknown", `{"success":true,"sign_in_state":"unknown"}`, "unknown"},
		{"invalid state", `{"success":true,"sign_in_state":"expired"}`, "unknown"},
		{"failed status", `{"success":false,"sign_in_state":"signed_out"}`, "unknown"},
		{"unrecognized fields", `{"success":true,"sign_in_state":"signed_out","token":"secret"}`, "unknown"},
		{"trailing data", `{"success":true,"sign_in_state":"signed_out"}{"token":"secret"}`, "unknown"},
		{"invalid JSON", `not-json`, "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decodeGrokSignInState([]byte(tc.data)); got != tc.want {
				t.Fatalf("state = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRemoteGrokSignInStateDoesNotTrustUpdateReportFields(t *testing.T) {
	for _, claimed := range []string{"signed_in", "signed_out", "expired", "unknown"} {
		if got := remoteGrokSignInState(claimed); got != "unknown" {
			t.Errorf("remote report claim %q produced sign-in state %q, want unknown without an owner probe", claimed, got)
		}
	}
}

func TestReadGrokSignInStateUsesSafeUnknownFallbacks(t *testing.T) {
	if got := readGrokSignInState(context.Background(), func(context.Context) ([]byte, error) {
		return []byte(`{"success":true,"sign_in_state":"signed_out"}`), nil
	}); got != "signed_out" {
		t.Fatalf("resource signed-out state = %q", got)
	}
	if got := readGrokSignInState(context.Background(), func(context.Context) ([]byte, error) {
		return nil, errors.New("resource command unavailable")
	}); got != "unknown" {
		t.Fatalf("resource command failure state = %q, want unknown", got)
	}
}

// The API's PATH can select the managed link while a session's shell selects an
// upstream copy. The report names both, and either one is the same install.
func TestProbeOfTheManagedCopyConfirmsAShadowedReport(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	rep := harnessUpdateReport{Name: "opencode", State: agentStateShadowed, PrimaryAction: "use_managed_copy", Installed: "1.18.35", ManagedVersion: "1.19.4", ManagedPath: "/home/o/.local/bin/opencode", Latest: "1.19.5", CommandPath: "/home/o/.opencode/bin/opencode", OS: "linux", Arch: "amd64", CheckedAt: now}
	probe := capabilityprobe.Observation{State: capabilityprobe.Ready, Path: "/home/o/.local/bin/opencode", Version: "1.19.4", ProbedAt: now}
	got := projectAgentUpdate(failedAgentUpdate("local", "opencode", "opencode", ""), rep, probe, "linux", "amd64", now)
	if got.State != agentStateShadowed || got.PrimaryAction != "use_managed_copy" || !got.ProbeVerified || got.Installed != "1.18.35" {
		t.Fatalf("got %+v", got)
	}
}

// The API's PATH puts the operator shims first. A shim runs the install it
// fronts, so its version confirms the report; another version does not.
func TestProbeThroughTheOperatorShimConfirmsTheReport(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	rep := harnessUpdateReport{Name: "claude-code", State: agentStateUpToDate, Installed: "2.1.316", ManagedVersion: "2.1.316", ManagedPath: "/home/o/.local/bin/claude", Latest: "2.1.316", CommandPath: "/home/o/.local/bin/claude", OS: "linux", Arch: "amd64", CheckedAt: now}
	probe := capabilityprobe.Observation{State: capabilityprobe.Ready, Path: "/home/o/.vrooli/shims/claude", Version: "2.1.316 (Claude Code)", ProbedAt: now}
	got := projectAgentUpdate(failedAgentUpdate("local", "claude", "claude-code", ""), rep, probe, "linux", "amd64", now)
	if got.State != agentStateUpToDate || !got.ProbeVerified {
		t.Fatalf("shim probe should confirm: %+v", got)
	}
	probe.Version = "2.1.300 (Claude Code)"
	got = projectAgentUpdate(failedAgentUpdate("local", "claude", "claude-code", ""), rep, probe, "linux", "amd64", now)
	if got.State != agentStateCheckFailed || got.ProbeVerified {
		t.Fatalf("a different version through the shim must not confirm: %+v", got)
	}
	probe.Version, probe.Path = "2.1.316 (Claude Code)", "/opt/elsewhere/shims/claude"
	got = projectAgentUpdate(failedAgentUpdate("local", "claude", "claude-code", ""), rep, probe, "linux", "amd64", now)
	if got.State != agentStateCheckFailed {
		t.Fatalf("a path outside the operator shims must not confirm: %+v", got)
	}
}

// A repair state is file evidence on the target, so it survives a probe that has
// not caught up. A version claim does not.
func TestRepairStatesDoNotDependOnAFreshProbe(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	for _, state := range []string{agentStateMissing, agentStateShadowed, agentStateUnlinked, agentStateNeedsSetup} {
		rep := harnessUpdateReport{Name: "agy", State: state, OperatorCommand: "cmd", Reason: "why", OS: "linux", Arch: "amd64", CheckedAt: now, RollbackVersion: "1.0.0", NewerThanReviewed: true, Checks: []harnessUpdateCheck{{ID: "ownership"}}}
		got := projectAgentUpdate(failedAgentUpdate("local", "agy", "antigravity", ""), rep, capabilityprobe.Observation{}, "linux", "amd64", now)
		if got.State != state || got.OperatorCommand != "cmd" || got.Reason != "why" || got.RollbackVersion != "1.0.0" || !got.NewerThanReviewed || len(got.Checks) != 1 {
			t.Fatalf("%s: got %+v", state, got)
		}
	}
}

type agentUpdateRelayFake struct {
	request *relayv1.RelayCallRequest
	token   string
	fail    bool
}

func (f *agentUpdateRelayFake) Call(_ context.Context, req *connect.Request[relayv1.RelayCallRequest]) (*connect.Response[relayv1.RelayCallResponse], error) {
	f.request, f.token = req.Msg, req.Header().Get("Authorization")
	if f.fail {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("not permitted"))
	}
	return connect.NewResponse(&relayv1.RelayCallResponse{Outcome: relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED, Data: []byte(`{"success":true}`)}), nil
}

func (*agentUpdateRelayFake) Reconcile(context.Context, *connect.Request[relayv1.RelayReconcileRequest]) (*connect.Response[relayv1.RelayReconcileResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
}

func TestRemoteAgentUpdateUsesReadOnlyRelayOnSelectedNode(t *testing.T) {
	fake := &agentUpdateRelayFake{}
	_, handler := relayconnect.NewRelayServiceHandler(fake)
	server := httptest.NewServer(handler)
	defer server.Close()
	target := targetConnection{Target: targetmodel.Target{NodeID: "node-one"}, BaseURL: server.URL, OwnerToken: "Bearer test"}
	data, err := (&Server{}).remoteAgentUpdate(context.Background(), target, "codex")
	if err != nil || string(data) != `{"success":true}` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	if fake.request.GetNodeId() != "node-one" || fake.request.GetCommand() != "resource update-check" || fake.request.GetScenario() != "vrooli" || len(fake.request.GetArgs()) != 2 || fake.request.GetArgs()[0] != "codex" || fake.token != "Bearer test" {
		t.Fatalf("wrong relay request: %+v, token %q", fake.request, fake.token)
	}
	fake.fail = true
	if _, err := (&Server{}).remoteAgentUpdate(context.Background(), target, "codex"); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("permission failure not preserved: %v", err)
	}
}

func TestAgentUpdateCacheCoalescesPerTarget(t *testing.T) {
	var c agentUpdateCache
	var calls atomic.Int32
	start := make(chan struct{})
	load := func(context.Context) (agentUpdate, error) {
		calls.Add(1)
		<-start
		return agentUpdate{Status: upstreamcheck.StatusBehind}, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.read(context.Background(), "one\x00codex", load); err != nil {
				t.Error(err)
			}
		}()
	}
	for calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(start)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("coalesced calls = %d, want 1", calls.Load())
	}
	_, err := c.read(context.Background(), "two\x00codex", func(context.Context) (agentUpdate, error) {
		calls.Add(1)
		return agentUpdate{Status: upstreamcheck.StatusUnknown}, nil
	})
	if err != nil || calls.Load() != 2 {
		t.Fatalf("distinct target must load independently, calls=%d err=%v", calls.Load(), err)
	}
}

func TestInvalidateDropsOneAgentAndARunningCheckIsNotStored(t *testing.T) {
	var c agentUpdateCache
	load := func(state string) func(context.Context) (agentUpdate, error) {
		return func(context.Context) (agentUpdate, error) { return agentUpdate{State: state}, nil }
	}
	key := "local\x00codex\x00linux"
	other := "local\x00grok\x00linux"
	if _, err := c.read(context.Background(), key, load("old")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.read(context.Background(), other, load("other")); err != nil {
		t.Fatal(err)
	}
	started, finish := make(chan struct{}), make(chan struct{})
	done := make(chan struct{})
	c.invalidate("local", "codex")
	go func() {
		defer close(done)
		_, _ = c.read(context.Background(), key, func(context.Context) (agentUpdate, error) {
			close(started)
			<-finish
			return agentUpdate{State: "before-the-action"}, nil
		})
	}()
	<-started
	c.invalidate("local", "codex")
	close(finish)
	<-done
	got, err := c.read(context.Background(), key, load("after-the-action"))
	if err != nil || got.State != "after-the-action" {
		t.Fatalf("got %+v, %v: a check begun before the action must not be cached", got, err)
	}
	if got, _ := c.read(context.Background(), other, load("reloaded")); got.State != "other" {
		t.Fatalf("another agent was invalidated: %+v", got)
	}
}

func TestAgentResourceUsesCanonicalAliases(t *testing.T) {
	for id, want := range map[string]string{"claude": "claude-code", "agy": "antigravity", "codex": "codex"} {
		if got, ok := agentResource(id); !ok || got != want {
			t.Fatalf("%s maps to %s, %v", id, got, ok)
		}
	}
	if _, ok := agentResource("unknown"); ok {
		t.Fatal("unknown agent accepted")
	}
}
