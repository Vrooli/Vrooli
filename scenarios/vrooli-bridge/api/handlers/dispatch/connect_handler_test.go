package dispatch

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/audit"
	auditmocks "vrooli-bridge/internal/audit/mocks"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/clock"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/presence"
	internalqueue "vrooli-bridge/internal/queue"
	"vrooli-bridge/internal/registry"
	rmocks "vrooli-bridge/internal/registry/mocks"
	"vrooli-bridge/internal/runs"
	runsmocks "vrooli-bridge/internal/runs/mocks"
	tmocks "vrooli-bridge/internal/testutil/mocks"

	queueH "vrooli-bridge/handlers/queue"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

// fakeDispatch records the DispatchInput and returns a canned decision/err.
type fakeDispatch struct {
	gotInput dispatch.DispatchInput
	out      dispatch.Decision
	err      error
}

func (f *fakeDispatch) Dispatch(_ context.Context, in dispatch.DispatchInput) (dispatch.Decision, error) {
	f.gotInput = in
	return f.out, f.err
}

// [REQ:BRG-P0-004] DispatchJob is owner-gated: no identity → Unauthenticated.
func TestDispatchHandler_RequiresOwner(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeDispatch{}})
	_, err := h.DispatchJob(context.Background(), connect.NewRequest(&dispatchv1.DispatchJobRequest{NodeId: "n1", Verb: "scenario test"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P0-004] The X-Dry-Run header threads through to a dry-run dispatch,
// and the owner identity becomes the audit actor.
func TestDispatchHandler_DryRunHeader(t *testing.T) {
	fake := &fakeDispatch{out: dispatch.Decision{DryRun: true, Job: dispatch.Job{NodeID: "n1", Verb: "scenario test"}}}
	h := NewConnectHandler(Deps{Service: fake})

	req := connect.NewRequest(&dispatchv1.DispatchJobRequest{NodeId: "n1", Verb: "scenario test"})
	req.Header().Set("X-Dry-Run", "true")
	resp, err := h.DispatchJob(ownerCtx(), req)
	require.NoError(t, err)
	require.True(t, fake.gotInput.DryRun, "handler reads X-Dry-Run into the dispatch input")
	require.True(t, resp.Msg.DryRun)
	require.Equal(t, "owner-1", fake.gotInput.Actor)
}

// [REQ:BRG-P0-004] An allowlist rejection maps to PermissionDenied.
func TestDispatchHandler_RejectionMapsToPermissionDenied(t *testing.T) {
	fake := &fakeDispatch{err: dispatch.ErrVerbOutOfScope{Verb: "scenario test"}}
	h := NewConnectHandler(Deps{Service: fake})
	_, err := h.DispatchJob(ownerCtx(), connect.NewRequest(&dispatchv1.DispatchJobRequest{NodeId: "n1", Verb: "scenario test"}))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

// [REQ:BRG-P0-004][REQ:BRG-P0-008] End-to-end through the REAL service + the
// real seam adapters (adapter.go): a valid dispatch creates a run, audits it,
// and the typed JobPush lands on the node's held channel as a decodable
// ServerFrame — exercising the proto translation + channel push for real.
func TestDispatchHandler_EndToEndPushesTypedJob(t *testing.T) {
	clk := clock.System{}

	repo := rmocks.NewFakeRepository()
	repo.Seed(registry.Node{ID: "n1", Name: "office", OS: "linux", Arch: "amd64", Scopes: []string{"scenario test*"}})
	registrySvc := registry.NewService(repo)

	runsSvc := runs.NewService(runsmocks.NewFakeRepository(), tmocks.NewFakeClock(time.Now()))
	auditSink := &auditmocks.FakeSink{}

	hub := presence.NewHub(clk)
	conn := hub.Connect("n1")
	defer conn.Close()

	// Wire the dispatch service with the same adapters the production Module uses,
	// including the per-node scheduler on the push path (a free slot pushes the
	// job immediately, so the JobPush still lands on the channel below).
	scheduler := internalqueue.NewScheduler(queueH.NewChannelPusher(hub), queueH.NewAborter(runsSvc), clk, 0)
	svc := dispatch.NewService(
		nodeReaderAdapter{svc: registrySvc},
		hub,
		runControllerAdapter{svc: runsSvc},
		auditSinkAdapter{sink: auditSink},
		jobPusherAdapter{scheduler: scheduler},
	)
	h := NewConnectHandler(Deps{Service: svc})

	resp, err := h.DispatchJob(ownerCtx(), connect.NewRequest(&dispatchv1.DispatchJobRequest{
		NodeId: "n1", Scenario: "web-search", Verb: "scenario test", Args: []string{"web-search"},
	}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.RunId)

	select {
	case payload := <-conn.Out():
		var frame channelv1.ServerFrame
		require.NoError(t, protojson.Unmarshal(payload, &frame))
		job := frame.GetJob()
		require.NotNil(t, job, "the pushed frame carries a JobPush")
		require.Equal(t, resp.Msg.RunId, job.RunId)
		require.Equal(t, "scenario test", job.Verb)
		require.Equal(t, []string{"web-search"}, job.Args)
	case <-time.After(time.Second):
		t.Fatal("no job frame pushed to the node channel")
	}

	require.Len(t, auditSink.Appended(), 1)
	require.Equal(t, audit.OutcomeAccepted, auditSink.Appended()[0].Outcome)
}
