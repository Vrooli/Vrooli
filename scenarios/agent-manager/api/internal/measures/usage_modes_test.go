package measures

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/usagemodes"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/usagemode"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
)

const modeTestPolicy = `{"runner":"codex","model_aliases":{"gpt-6.1-terra":{"canonical_model":"openai/gpt-6.1-terra"},"gpt-6-luna":{"canonical_model":"openai/gpt-6-luna"}},
"usage_tiers":{"model_map":{"gpt-6.1-terra":"gpt-6-luna"},"session_model":"gpt-6-luna"}}`

type modeTiers struct{}

func (modeTiers) Tiers(context.Context, string) (usagemode.Tiers, error) {
	return usagemode.LoadTiers([]byte(modeTestPolicy))
}

func modeHandler(t *testing.T) *Handler {
	t.Helper()
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	h := NewHandler(nil, func() time.Time { return now })
	h.SetUsageModes(&usagemodes.Service{Tiers: modeTiers{}, Now: func() time.Time { return now }, Path: filepath.Join(t.TempDir(), "mode.json")})
	return h
}

func withPrincipal(p identity.Principal) context.Context {
	return identity.WithPrincipal(context.Background(), p)
}

var (
	operator = identity.Principal{Kind: identity.ActorHuman, Subject: "operator@example.com", Verified: true, Scopes: []string{"agent-manager:write"}}
	agent    = identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Verified: true, Scopes: []string{"agent-manager:read", "agent-manager:write"}}
)

func startRequest() *connect.Request[measurepb.StartUsageModeRequest] {
	return connect.NewRequest(&measurepb.StartUsageModeRequest{Preset: "one-tier-down", Runner: "codex", Until: "6h"})
}

func TestUsageModeWritesRequireAVerifiedOperator(t *testing.T) {
	h := modeHandler(t)
	readOnly := identity.Principal{Kind: identity.ActorHuman, Subject: "viewer", Verified: true, Scopes: []string{"agent-manager:read"}}
	unverified := identity.Principal{Kind: identity.ActorHuman, Subject: "op", Scopes: []string{"agent-manager:write"}}
	service := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	for name, ctx := range map[string]context.Context{
		"no identity": context.Background(), "agent": withPrincipal(agent), "service": withPrincipal(service),
		"human without write": withPrincipal(readOnly), "unverified human": withPrincipal(unverified),
	} {
		if _, err := h.StartUsageMode(ctx, startRequest()); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s: start code = %v, want permission denied", name, connect.CodeOf(err))
		}
		if _, err := h.EndUsageMode(ctx, connect.NewRequest(&measurepb.EndUsageModeRequest{All: true})); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s: end code = %v, want permission denied", name, connect.CodeOf(err))
		}
	}
	listed, err := h.ListUsageModes(withPrincipal(agent), connect.NewRequest(&measurepb.ListUsageModesRequest{}))
	if err != nil || len(listed.Msg.GetModes()) != 0 {
		t.Fatalf("refused starts left modes: %v err=%v", listed, err)
	}
}

func TestOperatorStartsListsAndEndsAMode(t *testing.T) {
	h := modeHandler(t)
	started, err := h.StartUsageMode(withPrincipal(operator), startRequest())
	if err != nil {
		t.Fatal(err)
	}
	mode := started.Msg.GetMode()
	if mode.GetStartedBy() != "operator@example.com" || mode.GetScope().GetRunner() != "codex" || mode.GetUntilKind() != "time" {
		t.Fatalf("started mode = %+v", mode)
	}
	if got := mode.GetSettings()[0].GetModelMap()["gpt-6.1-terra"]; got != "gpt-6-luna" {
		t.Fatalf("mapped model = %q", got)
	}

	// An agent may read the mode, with its time left and the reach caveats.
	listed, err := h.ListUsageModes(withPrincipal(agent), connect.NewRequest(&measurepb.ListUsageModesRequest{}))
	if err != nil || len(listed.Msg.GetModes()) != 1 {
		t.Fatalf("list = %v err=%v", listed, err)
	}
	if left := listed.Msg.GetModes()[0].GetSecondsLeft(); left != 6*3600 {
		t.Fatalf("seconds left = %d", left)
	}
	if len(listed.Msg.GetReachNotes()) == 0 {
		t.Fatal("list must say what a mode cannot reach")
	}

	ended, err := h.EndUsageMode(withPrincipal(operator), connect.NewRequest(&measurepb.EndUsageModeRequest{Ids: []string{mode.GetId()}}))
	if err != nil || len(ended.Msg.GetEndedIds()) != 1 {
		t.Fatalf("end = %v err=%v", ended, err)
	}
	if after, _ := h.ListUsageModes(context.Background(), connect.NewRequest(&measurepb.ListUsageModesRequest{})); len(after.Msg.GetModes()) != 0 {
		t.Fatalf("mode still listed after end: %v", after)
	}
}

func TestStartRejectsAnInvalidRequestAsInvalidArgument(t *testing.T) {
	h := modeHandler(t)
	_, err := h.StartUsageMode(withPrincipal(operator), connect.NewRequest(&measurepb.StartUsageModeRequest{Preset: "one-tier-down", Runner: "codex"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing --until code = %v err=%v", connect.CodeOf(err), err)
	}
}

type fakeRebaser struct {
	calls  int
	dryRun bool
}

func (f *fakeRebaser) RebaseRuns(_ context.Context, modeID string, _ []string, dryRun bool) ([]RebaseResult, error) {
	f.calls++
	f.dryRun = dryRun
	return []RebaseResult{{RunID: "run-1", Status: "switched_at_next_resume", FromModel: "gpt-6.1-terra", ToModel: "gpt-6-luna", EstimatedRereadTokens: 90000}}, nil
}

func TestExtendAndRebaseRequireAVerifiedOperator(t *testing.T) {
	h := modeHandler(t)
	rebaser := &fakeRebaser{}
	h.SetRunRebaser(rebaser)
	for name, ctx := range map[string]context.Context{"no identity": context.Background(), "agent": withPrincipal(agent)} {
		if _, err := h.ExtendUsageMode(ctx, connect.NewRequest(&measurepb.ExtendUsageModeRequest{Id: "x", To: "2h"})); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s: extend code = %v, want permission denied", name, connect.CodeOf(err))
		}
		if _, err := h.RebaseUsageModeRuns(ctx, connect.NewRequest(&measurepb.RebaseUsageModeRunsRequest{ModeId: "x"})); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Errorf("%s: rebase code = %v, want permission denied", name, connect.CodeOf(err))
		}
	}
	if rebaser.calls != 0 {
		t.Fatal("a refused caller must never reach the rebase")
	}
}

func TestExtendKeepsTheModeAndRebaseReturnsPerRunResults(t *testing.T) {
	h := modeHandler(t)
	ctx := withPrincipal(operator)
	started, err := h.StartUsageMode(ctx, startRequest())
	if err != nil {
		t.Fatal(err)
	}
	id := started.Msg.GetMode().GetId()
	extended, err := h.ExtendUsageMode(ctx, connect.NewRequest(&measurepb.ExtendUsageModeRequest{Id: id, To: "2h"}))
	if err != nil {
		t.Fatal(err)
	}
	if extended.Msg.GetMode().GetId() != id || extended.Msg.GetMode().GetStartedAt() != started.Msg.GetMode().GetStartedAt() {
		t.Fatalf("extend replaced the mode: id %s -> %s, started %s -> %s", id, extended.Msg.GetMode().GetId(), started.Msg.GetMode().GetStartedAt(), extended.Msg.GetMode().GetStartedAt())
	}
	if extended.Msg.GetMode().GetExpiresAt() == started.Msg.GetMode().GetExpiresAt() {
		t.Fatal("extend did not move the end")
	}

	if _, err := h.RebaseUsageModeRuns(ctx, connect.NewRequest(&measurepb.RebaseUsageModeRunsRequest{ModeId: id})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("without a rebaser the control must say unavailable, got %v", connect.CodeOf(err))
	}
	rebaser := &fakeRebaser{}
	h.SetRunRebaser(rebaser)
	response, err := h.RebaseUsageModeRuns(ctx, connect.NewRequest(&measurepb.RebaseUsageModeRunsRequest{ModeId: id, DryRun: true}))
	if err != nil || !rebaser.dryRun || !response.Msg.GetDryRun() {
		t.Fatalf("dry run not passed through: %v %+v", err, response)
	}
	if got := response.Msg.GetResults(); len(got) != 1 || got[0].GetEstimatedRereadTokens() != 90000 || got[0].GetStatus() != "switched_at_next_resume" {
		t.Fatalf("results = %v", got)
	}
}
