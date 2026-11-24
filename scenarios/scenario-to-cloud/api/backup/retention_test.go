package backup

import (
	"context"
	"testing"
	"time"

	"scenario-to-cloud/domain"
)

// TestRetentionNeverDeletesProtectedPoints [REQ:STC-P0-030] proves P12-A06 at
// the planning level: points referenced by an active or retained release, a
// non-terminal operation or a pin are kept regardless of policy, and only
// unreferenced points are retired by keep_last or max_age.
func TestRetentionNeverDeletesProtectedPoints(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	active := point("rp-active", "v1", "r-active", now.Add(-72*time.Hour))
	retained := point("rp-retained", "v1", "r-prev", now.Add(-96*time.Hour))
	inflight := point("rp-op", "v1", "r-old", now.Add(-120*time.Hour))
	inflight.OperationID = "op-restore"
	pinned := point("rp-pinned", "v1", "r-old", now.Add(-200*time.Hour))
	fresh := point("rp-fresh", "v1", "r-old", now.Add(-time.Hour))
	stale := point("rp-stale", "v1", "r-old", now.Add(-48*time.Hour))
	ancient := point("rp-ancient", "v1", "r-old", now.Add(-300*time.Hour))
	refs := References{ActiveReleases: []string{"r-active"}, RetainedReleases: []string{"r-prev"}, NonTerminalOperations: []string{"op-restore"}, PinnedRecoveryPoints: []string{"rp-pinned"}}

	plan := Plan([]domain.RecoveryPoint{active, retained, inflight, pinned, fresh, stale, ancient}, refs, Policy{KeepLast: 1, MaxAge: 100 * time.Hour}, now)
	for _, id := range []string{"rp-active", "rp-retained", "rp-op", "rp-pinned"} {
		if len(plan.Protected[id]) == 0 {
			t.Fatalf("%s must be protected: %+v", id, plan)
		}
		for _, d := range plan.Delete {
			if d == id {
				t.Fatalf("%s must never be deleted: %+v", id, plan)
			}
		}
	}
	if plan.Protected["rp-active"][0] != "release:active:r-active" || plan.Protected["rp-op"][0] != "operation:op-restore" {
		t.Fatalf("holders = %+v", plan.Protected)
	}
	if len(plan.Delete) != 2 || plan.Delete[0] != "rp-ancient" || plan.Delete[1] != "rp-stale" {
		t.Fatalf("delete = %v (want rp-ancient by age, rp-stale by keep_last)", plan.Delete)
	}
	if len(plan.Keep) != 5 {
		t.Fatalf("keep = %v", plan.Keep)
	}
	// No policy keeps everything unprotected.
	if none := Plan([]domain.RecoveryPoint{fresh, stale, ancient}, References{}, Policy{}, now); len(none.Delete) != 0 || len(none.Keep) != 3 {
		t.Fatalf("empty policy = %+v", none)
	}
}

type retentionRepoStub struct {
	Repository
	points    []domain.RecoveryPoint
	updates   int
	updatedBy []string
}

func (r *retentionRepoStub) ListRecoveryPoints(context.Context, string) ([]domain.RecoveryPoint, error) {
	return r.points, nil
}

func (r *retentionRepoStub) SetRecoveryPointProtection(_ context.Context, _ string, _ bool, protectedBy []string) (*domain.RecoveryPoint, error) {
	r.updates++
	r.updatedBy = append([]string(nil), protectedBy...)
	return nil, nil
}

func (r *retentionRepoStub) DeleteRecoveryPoint(context.Context, string) error {
	return nil
}

func TestPruneWritesProtectionOnlyWhenOrderedHoldersChange(t *testing.T) {
	tests := []struct {
		name       string
		point      domain.RecoveryPoint
		refs       References
		wantWrites int
		wantBy     []string
	}{
		{
			name:  "nil and empty unprotected holders are equivalent",
			point: domain.RecoveryPoint{ID: "rp-empty"},
		},
		{
			name:  "same protected holders avoid a no-op write",
			point: domain.RecoveryPoint{ID: "rp-same", ReleaseDigest: "r1", Protected: true, ProtectedBy: []string{"release:active:r1"}},
			refs:  References{ActiveReleases: []string{"r1"}},
		},
		{
			name:       "changed holder is persisted",
			point:      domain.RecoveryPoint{ID: "rp-changed", ReleaseDigest: "r1", Protected: true, ProtectedBy: []string{"release:active:old"}},
			refs:       References{ActiveReleases: []string{"r1"}},
			wantWrites: 1,
			wantBy:     []string{"release:active:r1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &retentionRepoStub{points: []domain.RecoveryPoint{tt.point}}
			svc := &Service{Repo: repo, Now: func() time.Time { return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC) }}
			if _, err := svc.Prune(context.Background(), "deployment", tt.refs, Policy{}); err != nil {
				t.Fatalf("Prune: %v", err)
			}
			if repo.updates != tt.wantWrites {
				t.Fatalf("protection writes = %d, want %d", repo.updates, tt.wantWrites)
			}
			if len(tt.wantBy) > 0 && (len(repo.updatedBy) != len(tt.wantBy) || repo.updatedBy[0] != tt.wantBy[0]) {
				t.Fatalf("updated protected_by = %v, want %v", repo.updatedBy, tt.wantBy)
			}
		})
	}
}
