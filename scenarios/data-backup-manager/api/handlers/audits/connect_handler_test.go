package audits

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	internalaudits "data-backup-manager/internal/audits"
	"data-backup-manager/internal/audits/mocks"

	auditsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits"
)

// TestAuditsService_Contract exercises each AuditsService RPC against the
// handler backed by a fake service, asserting the domain→wire translation
// (status enum, inventories, comparison, timestamps) and typed-error codes.
func TestAuditsService_Contract(t *testing.T) {
	ctx := context.Background()

	t.Run("RunSnapshotAudit maps status, flags, inventories, comparison", func(t *testing.T) {
		svc := &mocks.FakeService{RunOut: internalaudits.Audit{
			ID:                 "a-1",
			TargetID:           "t1",
			DestinationID:      "dst-1",
			SnapshotID:         "snap-1",
			Status:             internalaudits.AuditCompleted,
			IncludeContentHash: true,
			IncludeSQLiteCheck: true,
			Restorable:         true,
			SnapshotTime:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			RequestedAt:        time.Unix(1700000000, 0).UTC(),
			Live:               &internalaudits.InventorySummary{Files: 5, PathListSHA256: "lp"},
			Snapshot: &internalaudits.InventorySummary{
				Files: 5, PathListSHA256: "lp",
				SQLite: []internalaudits.SqliteInventory{{Path: "events.db", IntegrityStatus: "ok", TableCount: 3}},
			},
			Comparison: &internalaudits.AuditComparison{Matches: true},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.RunSnapshotAudit(ctx, connect.NewRequest(&auditsv1.RunSnapshotAuditRequest{
			TargetId: "t1", DestinationId: "dst-1", SnapshotId: "snap-1",
			IncludeContentHash: true, IncludeSqliteChecks: true,
		}))
		if err != nil {
			t.Fatalf("RunSnapshotAudit: %v", err)
		}
		got := resp.Msg.Audit
		if got.Status != auditsv1.AuditStatus_AUDIT_STATUS_COMPLETED {
			t.Errorf("status = %v, want COMPLETED", got.Status)
		}
		if !got.Restorable {
			t.Errorf("restorable = false, want true")
		}
		if got.Comparison == nil || !got.Comparison.Matches {
			t.Errorf("comparison not mapped: %+v", got.Comparison)
		}
		if got.Snapshot == nil || len(got.Snapshot.Sqlite) != 1 || got.Snapshot.Sqlite[0].Path != "events.db" {
			t.Errorf("snapshot sqlite not mapped: %+v", got.Snapshot)
		}
		if got.SnapshotTime == nil {
			t.Errorf("snapshot_time not mapped")
		}
		if len(svc.RunCalls) != 1 || !svc.RunCalls[0].IncludeContentHash {
			t.Fatalf("expected 1 run call with content hash, got %+v", svc.RunCalls)
		}
	})

	t.Run("GetAudit returns the record", func(t *testing.T) {
		svc := &mocks.FakeService{GetOut: internalaudits.Audit{ID: "a-9", Status: internalaudits.AuditRunning}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.GetAudit(ctx, connect.NewRequest(&auditsv1.GetAuditRequest{Id: "a-9"}))
		if err != nil {
			t.Fatalf("GetAudit: %v", err)
		}
		if resp.Msg.Audit.Status != auditsv1.AuditStatus_AUDIT_STATUS_RUNNING {
			t.Errorf("status = %v, want RUNNING", resp.Msg.Audit.Status)
		}
	})

	t.Run("ListAudits maps each record", func(t *testing.T) {
		svc := &mocks.FakeService{ListOut: []internalaudits.Audit{
			{ID: "a-1", Status: internalaudits.AuditCompleted},
			{ID: "a-2", Status: internalaudits.AuditFailed},
		}}
		h := NewConnectHandler(Deps{Service: svc})
		resp, err := h.ListAudits(ctx, connect.NewRequest(&auditsv1.ListAuditsRequest{}))
		if err != nil {
			t.Fatalf("ListAudits: %v", err)
		}
		if len(resp.Msg.Audits) != 2 {
			t.Fatalf("audits = %d, want 2", len(resp.Msg.Audits))
		}
	})

	t.Run("not-found maps to CodeNotFound", func(t *testing.T) {
		svc := &mocks.FakeService{GetErr: internalaudits.ErrAuditNotFound{ID: "x"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.GetAudit(ctx, connect.NewRequest(&auditsv1.GetAuditRequest{Id: "x"}))
		if connect.CodeOf(err) != connect.CodeNotFound {
			t.Errorf("code = %v, want NotFound", connect.CodeOf(err))
		}
	})

	t.Run("invalid argument maps to CodeInvalidArgument", func(t *testing.T) {
		svc := &mocks.FakeService{RunErr: internalaudits.ErrInvalidAudit{Field: "target_id", Reason: "required"}}
		h := NewConnectHandler(Deps{Service: svc})
		_, err := h.RunSnapshotAudit(ctx, connect.NewRequest(&auditsv1.RunSnapshotAuditRequest{
			TargetId: "t", DestinationId: "d", SnapshotId: "s",
		}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("code = %v, want InvalidArgument", connect.CodeOf(err))
		}
	})
}
