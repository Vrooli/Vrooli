package runs

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"

	"data-backup-manager/internal/engine"
	internalruns "data-backup-manager/internal/runs"
	"data-backup-manager/internal/runs/mocks"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs"
)

var errVerifiedBoom = errors.New("verified lookup failed")

// fakeVerified satisfies VerifiedLookup for handler tests.
type fakeVerified struct {
	m   map[string]VerifiedInfo
	err error
}

func (f fakeVerified) LastVerifiedByTarget(context.Context) (map[string]VerifiedInfo, error) {
	return f.m, f.err
}

// TestRunsService_Contract exercises each RunsService RPC against the handler
// backed by a fake service, asserting the domain→wire translation (status and
// trigger enums, nested per-target outcomes) and typed-error codes.
func TestRunsService_Contract(t *testing.T) {
	ctx := context.Background()

	t.Run("TriggerRun maps status, trigger, and outcomes", func(t *testing.T) {
		svc := &mocks.FakeService{TriggerOut: internalruns.Run{
			ID: "run-1", PlanID: "plan-1", Trigger: internalruns.TriggerManual,
			Status: internalruns.RunPartialFailed, StartedAt: time.Unix(1700000000, 0).UTC(),
			Outcomes: []internalruns.TargetOutcome{
				{TargetID: "t1", DestinationID: "d1", Status: internalruns.OutcomeSucceeded, SnapshotID: "snap-1", Bytes: 42},
				{TargetID: "t2", DestinationID: "d1", Status: internalruns.OutcomeBlocked, Error: "cap"},
			},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.TriggerRun(ctx, connect.NewRequest(&runsv1.TriggerRunRequest{PlanId: "plan-1"}))
		if err != nil {
			t.Fatalf("TriggerRun: %v", err)
		}
		if svc.TriggeredPlan != "plan-1" {
			t.Fatalf("service got plan %q", svc.TriggeredPlan)
		}
		got := resp.Msg.Run
		if got.Status != runsv1.RunStatus_RUN_STATUS_PARTIAL_FAILED {
			t.Fatalf("status = %v", got.Status)
		}
		if got.Trigger != runsv1.TriggerSource_TRIGGER_SOURCE_MANUAL {
			t.Fatalf("trigger = %v", got.Trigger)
		}
		if len(got.Outcomes) != 2 || got.Outcomes[0].Status != runsv1.TargetOutcomeStatus_TARGET_OUTCOME_STATUS_SUCCEEDED {
			t.Fatalf("outcomes wrong: %+v", got.Outcomes)
		}
		if got.Outcomes[1].Status != runsv1.TargetOutcomeStatus_TARGET_OUTCOME_STATUS_BLOCKED {
			t.Fatalf("second outcome should be blocked: %+v", got.Outcomes[1])
		}
	})

	t.Run("GetRun surfaces not-found", func(t *testing.T) {
		svc := &mocks.FakeService{GetErr: internalruns.ErrRunNotFound{ID: "x"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetRun(ctx, connect.NewRequest(&runsv1.GetRunRequest{Id: "x"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
		}
	})

	t.Run("ListTargetStatus maps the rollup", func(t *testing.T) {
		svc := &mocks.FakeService{StatusOut: []internalruns.TargetStatus{
			{TargetID: "t1", LastRunStatus: internalruns.RunCompleted, LastSuccessAt: time.Unix(1700000000, 0).UTC()},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.ListTargetStatus(ctx, connect.NewRequest(&runsv1.ListTargetStatusRequest{}))
		if err != nil {
			t.Fatalf("ListTargetStatus: %v", err)
		}
		if len(resp.Msg.Statuses) != 1 || resp.Msg.Statuses[0].LastRunStatus != runsv1.RunStatus_RUN_STATUS_COMPLETED {
			t.Fatalf("status rollup wrong: %+v", resp.Msg.Statuses)
		}
	})

	t.Run("ListTargetStatus composes verified posture per target", func(t *testing.T) {
		verifiedAt := time.Unix(1700001111, 0).UTC()
		svc := &mocks.FakeService{StatusOut: []internalruns.TargetStatus{
			// t1: backed up AND verified.
			{TargetID: "t1", LastRunStatus: internalruns.RunCompleted, LastSuccessAt: time.Unix(1700000000, 0).UTC()},
			// t2: backed up but NEVER verified — the central distinction.
			{TargetID: "t2", LastRunStatus: internalruns.RunCompleted, LastSuccessAt: time.Unix(1700000000, 0).UTC()},
		}}
		verified := fakeVerified{m: map[string]VerifiedInfo{
			"t1": {LastVerifiedAt: verifiedAt, SnapshotID: "snap-verified"},
		}}
		h := NewConnectHandler(Deps{Service: svc, Verified: verified})
		resp, err := h.ListTargetStatus(ctx, connect.NewRequest(&runsv1.ListTargetStatusRequest{}))
		if err != nil {
			t.Fatalf("ListTargetStatus: %v", err)
		}
		byID := map[string]*runsv1.TargetStatus{}
		for _, s := range resp.Msg.Statuses {
			byID[s.TargetId] = s
		}
		t1 := byID["t1"]
		if t1 == nil || t1.LastVerifiedAt == nil || !t1.LastVerifiedAt.AsTime().Equal(verifiedAt) {
			t.Fatalf("t1 should carry last_verified_at %v: %+v", verifiedAt, t1)
		}
		if t1.LastVerifiedSnapshotId != "snap-verified" {
			t.Errorf("t1 verified snapshot = %q, want snap-verified", t1.LastVerifiedSnapshotId)
		}
		// The whole point: a backed-up-but-unverified target reports NO verify.
		t2 := byID["t2"]
		if t2 == nil || t2.LastVerifiedAt != nil || t2.LastVerifiedSnapshotId != "" {
			t.Fatalf("t2 is backed up but never verified; last_verified must be unset: %+v", t2)
		}
	})

	t.Run("verified-lookup error surfaces as an RPC error", func(t *testing.T) {
		svc := &mocks.FakeService{StatusOut: []internalruns.TargetStatus{{TargetID: "t1"}}}
		h := NewConnectHandler(Deps{Service: svc, Verified: fakeVerified{err: errVerifiedBoom}})
		_, err := h.ListTargetStatus(ctx, connect.NewRequest(&runsv1.ListTargetStatusRequest{}))
		if err == nil {
			t.Fatal("expected an error when the verified lookup fails")
		}
	})

	t.Run("BrowseSnapshot maps entries", func(t *testing.T) {
		svc := &mocks.FakeService{BrowseOut: []engine.SnapshotEntry{{Path: "a/b.txt", SizeBytes: 7}}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.BrowseSnapshot(ctx, connect.NewRequest(&runsv1.BrowseSnapshotRequest{DestinationId: "d1", SnapshotId: "s1"}))
		if err != nil {
			t.Fatalf("BrowseSnapshot: %v", err)
		}
		if len(resp.Msg.Entries) != 1 || resp.Msg.Entries[0].Path != "a/b.txt" {
			t.Fatalf("entries wrong: %+v", resp.Msg.Entries)
		}
	})
}
