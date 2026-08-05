package deployments

import (
	"context"
	"testing"
	"time"

	internalEvidence "deployment-manager/internal/evidence"
	"github.com/DATA-DOG/go-sqlmock"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type gateEvidenceRepo struct {
	verdicts []*commonv1.TargetVerdict
	commit   string
}

func (r gateEvidenceRepo) EnsureSchema(context.Context) error { return nil }
func (r gateEvidenceRepo) Save(context.Context, string, string, *commonv1.TargetVerdict) error {
	return nil
}

func (r gateEvidenceRepo) List(_ context.Context, _ string, commit string, _ int) ([]*commonv1.TargetVerdict, error) {
	if r.commit != "" && r.commit != commit {
		return nil, nil
	}
	return r.verdicts, nil
}

var _ internalEvidence.Repository = gateEvidenceRepo{}

func TestReleaseGateEvidenceContract(t *testing.T) {
	// [REQ:DM-P0-039] Desktop evidence must satisfy the release-gate contract.
	// [REQ:DM-P0-038] Gate decisions consume the shared evidence disposition.
	target := RequiredTarget{Ramp: "release", Platform: "linux", OS: "linux", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST}
	tests := []struct {
		name       string
		verdicts   []*commonv1.TargetVerdict
		approval   bool
		wantReady  bool
		wantReason string
	}{
		{name: "no required targets", wantReason: "no_required_targets_configured"},
		{name: "evidence missing", approval: true, wantReason: "target_evidence_missing"},
		{name: "evidence failed", approval: true, verdicts: []*commonv1.TargetVerdict{testVerdict(target, commonv1.Disposition_DISPOSITION_FAILED)}, wantReason: "target_evidence_failed"},
		{name: "approval missing", verdicts: []*commonv1.TargetVerdict{testVerdict(target, commonv1.Disposition_DISPOSITION_PASSED)}, wantReason: "approval_missing"},
		{name: "all satisfied", approval: true, verdicts: []*commonv1.TargetVerdict{testVerdict(target, commonv1.Disposition_DISPOSITION_PASSED)}, wantReady: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			mock.ExpectQuery(`SELECT ramp, platform, os, device_kind FROM profile_required_targets`).
				WithArgs("profile-1").WillReturnRows(func() *sqlmock.Rows {
				rows := sqlmock.NewRows([]string{"ramp", "platform", "os", "device_kind"})
				if tt.name != "no required targets" {
					rows.AddRow(target.Ramp, target.Platform, target.OS, int32(target.DeviceKind))
				}
				return rows
			}())
			if tt.name == "no required targets" {
				mock.ExpectQuery(`SELECT platform FROM profile_required_platforms`).WithArgs("profile-1").WillReturnRows(sqlmock.NewRows([]string{"platform"}))
			}
			rows := sqlmock.NewRows([]string{"id", "profile_id", "git_commit_hash", "platform", "status", "approved_by", "approved_at", "notes", "validation_id", "created_at", "updated_at"})
			if tt.approval {
				now := time.Now()
				rows.AddRow("approval-1", "profile-1", "commit-1", "linux", ApprovalStatusApproved, "reviewer", now, "", "", now, now)
			}
			mock.ExpectQuery(`SELECT id, profile_id, git_commit_hash, platform, status`).WithArgs("profile-1", "commit-1").WillReturnRows(rows)
			repo := NewSQLApprovalsRepository(db).WithEvidenceRepository(gateEvidenceRepo{verdicts: tt.verdicts})
			got, err := repo.CheckReleaseGate(context.Background(), "profile-1", "commit-1")
			if err != nil {
				t.Fatal(err)
			}
			if got.Ready != tt.wantReady || got.Reason != tt.wantReason {
				t.Fatalf("gate = %+v, want ready=%v reason=%q", got, tt.wantReady, tt.wantReason)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestReleaseGateDoesNotAcceptDifferentCommitEvidence(t *testing.T) {
	// [REQ:DM-P0-039] Evidence from a different commit cannot approve a release.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(`SELECT ramp, platform, os, device_kind FROM profile_required_targets`).WithArgs("profile-1").WillReturnRows(sqlmock.NewRows([]string{"ramp", "platform", "os", "device_kind"}).AddRow("release", "linux", "linux", 1))
	mock.ExpectQuery(`SELECT id, profile_id, git_commit_hash, platform, status`).WithArgs("profile-1", "commit-2").WillReturnRows(sqlmock.NewRows([]string{"id", "profile_id", "git_commit_hash", "platform", "status", "approved_by", "approved_at", "notes", "validation_id", "created_at", "updated_at"}).AddRow("approval-1", "profile-1", "commit-2", "linux", ApprovalStatusApproved, "reviewer", time.Now(), "", "", time.Now(), time.Now()))
	repo := NewSQLApprovalsRepository(db).WithEvidenceRepository(gateEvidenceRepo{commit: "commit-1", verdicts: []*commonv1.TargetVerdict{testVerdict(RequiredTarget{Ramp: "release", Platform: "linux", OS: "linux", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST}, commonv1.Disposition_DISPOSITION_PASSED)}})
	got, err := repo.CheckReleaseGate(context.Background(), "profile-1", "commit-2")
	if err != nil {
		t.Fatal(err)
	}
	if got.Ready || got.Reason != "target_evidence_missing" {
		t.Fatalf("evidence from a different commit must not satisfy the gate: %+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func testVerdict(target RequiredTarget, disposition commonv1.Disposition) *commonv1.TargetVerdict {
	return &commonv1.TargetVerdict{Target: &commonv1.EvidenceTarget{Ramp: target.Ramp, Platform: target.Platform, Os: target.OS, DeviceKind: target.DeviceKind}, Disposition: disposition, RunId: "run-1"}
}
