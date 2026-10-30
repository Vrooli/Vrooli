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
	base := agentUpdate{TargetID: "bridge-node:one", AgentID: "codex", Status: upstreamcheck.StatusUnknown}
	rep := upstreamcheck.Report{Name: "codex", Installed: "0.165.0", Latest: "0.166.0", Pinned: "0.165.0", CommandPath: "/bin/codex", ExecutablePath: "/bin/codex", OS: "linux", Arch: "amd64", CheckedAt: now}
	probe := capabilityprobe.Observation{ID: "codex", State: capabilityprobe.Ready, Path: "/bin/codex", Version: "codex-cli 0.165.0", ProbedAt: now}
	cases := []struct {
		name   string
		change func(*upstreamcheck.Report, *capabilityprobe.Observation)
		want   upstreamcheck.Status
	}{
		{"matching", func(*upstreamcheck.Report, *capabilityprobe.Observation) {}, upstreamcheck.StatusBehind},
		{"matching managed shim", func(r *upstreamcheck.Report, p *capabilityprobe.Observation) {
			r.CommandPath = "/shims/codex"
			r.ExecutablePath = "/real/codex"
			p.Path = "/shims/codex"
		}, upstreamcheck.StatusBehind},
		{"different executable", func(_ *upstreamcheck.Report, p *capabilityprobe.Observation) { p.Path = "/other/codex" }, upstreamcheck.StatusUnknown},
		{"different installed version", func(_ *upstreamcheck.Report, p *capabilityprobe.Observation) { p.Version = "0.164.0" }, upstreamcheck.StatusUnknown},
		{"stale capability", func(_ *upstreamcheck.Report, p *capabilityprobe.Observation) { p.ProbedAt = now.Add(-3 * time.Minute) }, upstreamcheck.StatusUnknown},
		{"future observation", func(_ *upstreamcheck.Report, p *capabilityprobe.Observation) { p.ProbedAt = now.Add(time.Second) }, upstreamcheck.StatusUnknown},
		{"old resource check", func(r *upstreamcheck.Report, _ *capabilityprobe.Observation) { r.CheckedAt = time.Time{} }, upstreamcheck.StatusUnknown},
		{"different platform", func(r *upstreamcheck.Report, _ *capabilityprobe.Observation) { r.Arch = "arm64" }, upstreamcheck.StatusUnknown},
		{"prerelease", func(r *upstreamcheck.Report, _ *capabilityprobe.Observation) { r.Latest = "0.166.0-rc.1" }, upstreamcheck.StatusUnknown},
		{"missing upstream", func(r *upstreamcheck.Report, _ *capabilityprobe.Observation) { r.Latest = "" }, upstreamcheck.StatusUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, p := rep, probe
			tc.change(&r, &p)
			got := projectAgentUpdate(base, r, p, "linux", "amd64", now)
			if got.Status != tc.want {
				t.Fatalf("status %q, want %q: %+v", got.Status, tc.want, got)
			}
			if tc.want == upstreamcheck.StatusBehind && !got.ProbeVerified {
				t.Fatal("matching installed tool must be verified")
			}
		})
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
	if fake.request.GetNodeId() != "node-one" || fake.request.GetCommand() != "resource upstream-check" || fake.request.GetScenario() != "vrooli" || len(fake.request.GetArgs()) != 2 || fake.request.GetArgs()[0] != "codex" || fake.token != "Bearer test" {
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
