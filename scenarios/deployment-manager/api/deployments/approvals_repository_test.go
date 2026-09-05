package deployments

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestApprovalsRepositoryCRUDAndConfiguration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLApprovalsRepository(db)
	now := time.Now()
	approval := &DeploymentApproval{ID: "a1", ProfileID: "p", GitCommitHash: "c", Platform: "linux", Status: ApprovalStatusPending, CreatedAt: now, UpdatedAt: now}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE deployment_approvals")).WithArgs("p", "linux", "c").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO deployment_approvals")).WithArgs("a1", "p", "c", "linux", ApprovalStatusPending, sqlmock.AnyArg(), now).WillReturnResult(sqlmock.NewResult(1, 1))
	if err := repo.Create(context.Background(), approval); err != nil {
		t.Fatalf("create: %v", err)
	}
	rows := approvalRows(map[string]string{"linux": ApprovalStatusPending})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, git_commit_hash, platform, status,")).WithArgs("a1").WillReturnRows(rows)
	if got, err := repo.Get(context.Background(), "a1"); err != nil || got.Platform != "linux" {
		t.Fatalf("get: %+v %v", got, err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE deployment_approvals")).WithArgs("a1", ApprovalStatusApproved, "reviewer", sqlmock.AnyArg(), "looks good").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.UpdateDecision(context.Background(), "a1", ApprovalStatusApproved, "reviewer", "looks good"); err != nil {
		t.Fatalf("update: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, git_commit_hash, platform, status,")).WithArgs("p", "c").WillReturnRows(approvalRows(map[string]string{"linux": ApprovalStatusApproved}))
	if got, err := repo.ListByCommit(context.Background(), "p", "c"); err != nil || len(got) != 1 {
		t.Fatalf("list commit: %+v %v", got, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, git_commit_hash, platform, status,")).WithArgs("p", 50).WillReturnRows(approvalRows(map[string]string{"linux": ApprovalStatusApproved}))
	if got, err := repo.ListByProfile(context.Background(), "p", 0); err != nil || len(got) != 1 {
		t.Fatalf("list profile: %+v %v", got, err)
	}
	if got := nullString(""); got.Valid || nullString("x").String != "x" {
		t.Fatal("unexpected nullable conversion")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApprovalsRepositoryRequiredTargetsAndStale(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLApprovalsRepository(db)
	ctx := context.Background()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM profile_required_platforms")).WithArgs("p").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profile_required_platforms")).WithArgs("p", "linux").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profile_required_platforms")).WithArgs("p", "windows").WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	if err := repo.SetRequiredPlatforms(ctx, "p", []string{"linux", "windows"}); err != nil {
		t.Fatalf("set platforms: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT platform FROM profile_required_platforms")).WithArgs("p").WillReturnRows(requiredRows([]string{"linux", "windows"}))
	if got, err := repo.GetRequiredPlatforms(ctx, "p"); err != nil || len(got) != 2 {
		t.Fatalf("get platforms: %+v %v", got, err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE deployment_approvals")).WithArgs("p", "linux", "new").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.MarkStale(ctx, "p", "linux", "new"); err != nil {
		t.Fatalf("mark stale: %v", err)
	}
	targets := []RequiredTarget{{Ramp: "desktop", Platform: "linux", OS: "linux", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST}}
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM profile_required_targets")).WithArgs("p").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profile_required_targets")).WithArgs("p", "desktop", "linux", "linux", int32(commonv1.DeviceKind_DEVICE_KIND_HOST)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := repo.SetRequiredTargets(ctx, "p", targets); err != nil {
		t.Fatalf("set targets: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT ramp, platform, os, device_kind FROM profile_required_targets")).WithArgs("p").WillReturnRows(sqlmock.NewRows([]string{"ramp", "platform", "os", "device_kind"}).AddRow("desktop", "linux", "linux", int32(commonv1.DeviceKind_DEVICE_KIND_HOST)))
	gotTargets, err := repo.GetRequiredTargets(ctx, "p")
	if err != nil || len(gotTargets) != 1 || !sameTarget(targets[0], &commonv1.EvidenceTarget{Ramp: "desktop", Platform: "linux", Os: "linux", DeviceKind: commonv1.DeviceKind_DEVICE_KIND_HOST}) {
		t.Fatalf("targets: %+v %v", gotTargets, err)
	}
	if sameTarget(targets[0], &commonv1.EvidenceTarget{Ramp: "other"}) {
		t.Fatal("different target should not match")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCheckReleaseGate(t *testing.T) {
	tests := []struct {
		name          string
		required      []string
		approvals     map[string]string
		wantReady     bool
		wantReason    string
		wantPlatforms int
	}{
		{name: "no required platforms", wantReady: false, wantReason: "no_required_platforms_configured"},
		{name: "one platform missing", required: []string{"linux"}, wantReady: false, wantReason: "platforms_not_approved", wantPlatforms: 1},
		{name: "one platform rejected", required: []string{"linux"}, approvals: map[string]string{"linux": ApprovalStatusRejected}, wantReady: false, wantReason: "platforms_not_approved", wantPlatforms: 1},
		{name: "one platform stale", required: []string{"linux"}, approvals: map[string]string{"linux": ApprovalStatusStale}, wantReady: false, wantReason: "platforms_not_approved", wantPlatforms: 1},
		{name: "all approved", required: []string{"linux", "windows"}, approvals: map[string]string{"linux": ApprovalStatusApproved, "windows": ApprovalStatusApproved}, wantReady: true, wantPlatforms: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()

			requiredQuery := mock.ExpectQuery(regexp.QuoteMeta("SELECT platform FROM profile_required_platforms WHERE profile_id = $1 ORDER BY platform"))
			requiredQuery.WithArgs("profile-1").WillReturnRows(requiredRows(tt.required))
			approvalQuery := mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, git_commit_hash, platform, status,"))
			approvalQuery.WithArgs("profile-1", "commit-1").WillReturnRows(approvalRows(tt.approvals))

			repo := NewSQLApprovalsRepository(db)
			got, err := repo.CheckReleaseGate(context.Background(), "profile-1", "commit-1")
			if err != nil {
				t.Fatal(err)
			}
			if got.Ready != tt.wantReady || got.Reason != tt.wantReason {
				t.Fatalf("gate = %+v, want ready=%t reason=%q", got, tt.wantReady, tt.wantReason)
			}
			if len(got.Platforms) != tt.wantPlatforms {
				t.Fatalf("platforms = %d, want %d", len(got.Platforms), tt.wantPlatforms)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func requiredRows(platforms []string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"platform"})
	for _, platform := range platforms {
		rows.AddRow(platform)
	}
	return rows
}

func approvalRows(approvals map[string]string) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{
		"id", "profile_id", "git_commit_hash", "platform", "status",
		"approved_by", "approved_at", "notes", "validation_id", "created_at", "updated_at",
	})
	now := time.Now()
	for platform, status := range approvals {
		var approvedBy, notes, validationID sql.NullString
		var approvedAt sql.NullTime
		if status == ApprovalStatusApproved {
			approvedBy = sql.NullString{String: "reviewer", Valid: true}
			approvedAt = sql.NullTime{Time: now, Valid: true}
		}
		rows.AddRow("approval-"+platform, "profile-1", "commit-1", platform, status, approvedBy, approvedAt, notes, validationID, now, now)
	}
	return rows
}
