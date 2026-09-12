package restores

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	internalrestores "data-backup-manager/internal/restores"
	"data-backup-manager/internal/restores/mocks"

	restoresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores"
)

// TestRestoresService_Contract exercises each RestoresService RPC against the
// handler backed by a fake service, asserting the domain→wire translation
// (mode/status enums, timestamps) and typed-error codes.
func TestRestoresService_Contract(t *testing.T) {
	ctx := context.Background()
	verifiedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("RestoreTarget maps mode, status, and location", func(t *testing.T) {
		svc := &mocks.FakeService{RestoreOut: internalrestores.Restore{
			ID:            "r-1",
			TargetID:      "t1",
			DestinationID: "dst-1",
			SnapshotID:    "snap-1",
			Mode:          internalrestores.ModeRestore,
			Status:        internalrestores.RestoreRestored,
			Location:      "/restore/dest",
			RequestedAt:   time.Unix(1700000000, 0).UTC(),
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.RestoreTarget(ctx, connect.NewRequest(&restoresv1.RestoreTargetRequest{
			TargetId: "t1", DestinationId: "dst-1", SnapshotId: "snap-1", Location: "/restore/dest",
		}))
		if err != nil {
			t.Fatalf("RestoreTarget: %v", err)
		}
		got := resp.Msg.Restore
		if got.Status != restoresv1.RestoreStatus_RESTORE_STATUS_RESTORED {
			t.Errorf("status = %v, want RESTORED", got.Status)
		}
		if got.Mode != restoresv1.RestoreMode_RESTORE_MODE_RESTORE {
			t.Errorf("mode = %v, want RESTORE", got.Mode)
		}
		if got.Location != "/restore/dest" {
			t.Errorf("location = %q", got.Location)
		}
		if len(svc.RestoreTargetCalls) != 1 {
			t.Fatalf("expected 1 RestoreTarget call, got %d", len(svc.RestoreTargetCalls))
		}
	})

	t.Run("VerifyTarget maps verified status and last_verified_at", func(t *testing.T) {
		svc := &mocks.FakeService{VerifyOut: internalrestores.Restore{
			ID:             "r-2",
			TargetID:       "t1",
			DestinationID:  "dst-1",
			SnapshotID:     "snap-1",
			Mode:           internalrestores.ModeVerify,
			Status:         internalrestores.RestoreVerified,
			Checksum:       "abc123",
			LastVerifiedAt: verifiedAt,
			RequestedAt:    time.Unix(1700000000, 0).UTC(),
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.VerifyTarget(ctx, connect.NewRequest(&restoresv1.VerifyTargetRequest{
			TargetId: "t1", DestinationId: "dst-1", SnapshotId: "snap-1",
		}))
		if err != nil {
			t.Fatalf("VerifyTarget: %v", err)
		}
		got := resp.Msg.Restore
		if got.Status != restoresv1.RestoreStatus_RESTORE_STATUS_VERIFIED {
			t.Errorf("status = %v, want VERIFIED", got.Status)
		}
		if got.Mode != restoresv1.RestoreMode_RESTORE_MODE_VERIFY {
			t.Errorf("mode = %v, want VERIFY", got.Mode)
		}
		if got.Checksum != "abc123" {
			t.Errorf("checksum = %q", got.Checksum)
		}
		if got.LastVerifiedAt == nil {
			t.Error("last_verified_at must be set for verified restore")
		}
	})

	t.Run("VerifyTarget maps failed status without last_verified_at", func(t *testing.T) {
		svc := &mocks.FakeService{VerifyOut: internalrestores.Restore{
			ID:     "r-3",
			Mode:   internalrestores.ModeVerify,
			Status: internalrestores.RestoreFailed,
			Error:  "verify: content mismatch",
			// LastVerifiedAt is zero — this is the safety invariant.
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.VerifyTarget(ctx, connect.NewRequest(&restoresv1.VerifyTargetRequest{
			TargetId: "t1", DestinationId: "dst-1", SnapshotId: "snap-bad",
		}))
		if err != nil {
			t.Fatalf("VerifyTarget: %v", err)
		}
		got := resp.Msg.Restore
		if got.Status != restoresv1.RestoreStatus_RESTORE_STATUS_FAILED {
			t.Errorf("status = %v, want FAILED", got.Status)
		}
		if got.LastVerifiedAt != nil {
			t.Errorf("last_verified_at must be nil for failed verify, got %v", got.LastVerifiedAt)
		}
	})

	t.Run("GetRestore surfaces not-found", func(t *testing.T) {
		svc := &mocks.FakeService{GetErr: internalrestores.ErrRestoreNotFound{ID: "x"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetRestore(ctx, connect.NewRequest(&restoresv1.GetRestoreRequest{Id: "x"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Fatalf("code = %v, want not_found", connect.CodeOf(err))
		}
	})

	t.Run("ListRestores returns multiple records", func(t *testing.T) {
		svc := &mocks.FakeService{ListOut: []internalrestores.Restore{
			{ID: "r-1", Status: internalrestores.RestoreRestored, Mode: internalrestores.ModeRestore},
			{ID: "r-2", Status: internalrestores.RestoreVerified, Mode: internalrestores.ModeVerify, Checksum: "xyz"},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.ListRestores(ctx, connect.NewRequest(&restoresv1.ListRestoresRequest{}))
		if err != nil {
			t.Fatalf("ListRestores: %v", err)
		}
		if len(resp.Msg.Restores) != 2 {
			t.Fatalf("expected 2 restores, got %d", len(resp.Msg.Restores))
		}
		if resp.Msg.Restores[0].Status != restoresv1.RestoreStatus_RESTORE_STATUS_RESTORED {
			t.Errorf("first status = %v", resp.Msg.Restores[0].Status)
		}
		if resp.Msg.Restores[1].Status != restoresv1.RestoreStatus_RESTORE_STATUS_VERIFIED {
			t.Errorf("second status = %v", resp.Msg.Restores[1].Status)
		}
	})

	t.Run("RestoreTarget surfaces invalid-argument", func(t *testing.T) {
		svc := &mocks.FakeService{RestoreErr: internalrestores.ErrInvalidRestore{Field: "target_id", Reason: "required"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.RestoreTarget(ctx, connect.NewRequest(&restoresv1.RestoreTargetRequest{
			TargetId: "t1", DestinationId: "dst-1", SnapshotId: "snap-1", Location: "/loc",
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("code = %v, want invalid_argument", connect.CodeOf(err))
		}
	})
}
