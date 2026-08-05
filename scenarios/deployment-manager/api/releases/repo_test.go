package releases

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLRepositoryStatusAndPlatformUpdates(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLRepository(db)
	ctx := context.Background()
	mock.ExpectExec(regexp.QuoteMeta("UPDATE releases SET status = $2, updated_at = $3 WHERE id = $1")).WithArgs("r1", StatusPublishing, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.UpdateStatus(ctx, "r1", StatusPublishing); err != nil {
		t.Fatalf("status: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE releases\n\t\t\tSET status = $2, updated_at = $3, published_at = COALESCE(published_at, $3)\n\t\t\tWHERE id = $1")).WithArgs("r1", StatusPublished, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.UpdateStatus(ctx, "r1", StatusPublished); err != nil {
		t.Fatalf("published status: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE releases SET verification_evidence = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1")).WithArgs("r1", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.SetVerificationEvidence(ctx, "r1", []VerificationItem{{Platform: "linux", ObservedVersion: "1.0"}}); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE release_platforms")).WithArgs("r1", "linux", PlatformStatusPublished, int64(42)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.MarkPlatformPublished(ctx, "r1", "linux", 42); err != nil {
		t.Fatalf("platform published: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE release_platforms")).WithArgs("r1", "linux", PlatformStatusFailed, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.MarkPlatformStatus(ctx, "r1", "linux", PlatformStatusFailed, "broken"); err != nil {
		t.Fatalf("platform failed: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE releases")).WithArgs("p1", "stable", "r2", StatusSuperseded, StatusPublished).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.MarkSuperseded(ctx, "p1", "stable", "r2"); err != nil {
		t.Fatalf("supersede: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseHelpers(t *testing.T) {
	if got := nullString(""); got.Valid || nullString("x").String != "x" {
		t.Fatal("unexpected null string helper")
	}
	if got := nullIntPtr(0); got.Valid || !nullIntPtr(2).Valid || nullIntPtr(2).Int32 != 2 {
		t.Fatal("unexpected null int helper")
	}
}

func releaseRows(id string) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{"id", "profile_id", "deployment_id", "profile_version", "git_commit_hash", "release_version", "channel", "status", "release_notes", "released_by", "promoted_from_release_id", "verification_evidence", "created_at", "published_at", "updated_at"}).
		AddRow(id, "p1", nil, 1, "abc", "1.0.0", "stable", StatusPending, nil, nil, nil, []byte(`[]`), now, nil, now)
}

func releasePlatformRows(id string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"release_id", "platform", "status", "approval_id", "lpbs_artifact_id", "published_at", "verified_at", "error"}).
		AddRow(id, "linux-x64", PlatformStatusPending, nil, nil, nil, nil, nil)
}

func TestSQLRepositoryInsertGetAndList(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLRepository(db)
	ctx := context.Background()
	release := &Release{ID: "r1", ProfileID: "p1", ProfileVersion: 1, GitCommitHash: "abc", ReleaseVersion: "1.0.0", Platforms: []ReleasePlatform{{Platform: "linux-x64"}}}
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO releases")).WithArgs("r1", "p1", sqlmock.AnyArg(), sqlmock.AnyArg(), "abc", "1.0.0", "stable", StatusPending, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO release_platforms")).WithArgs("r1", "linux-x64", PlatformStatusPending, sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := repo.Insert(ctx, release); err != nil || release.Channel != "stable" || release.Status != StatusPending {
		t.Fatalf("insert = %#v, %v", release, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, deployment_id, profile_version, git_commit_hash")).WithArgs("r1").WillReturnRows(releaseRows("r1"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT release_id, platform, status, approval_id")).WithArgs("r1").WillReturnRows(releasePlatformRows("r1"))
	got, err := repo.Get(ctx, "r1")
	if err != nil || got == nil || len(got.Platforms) != 1 {
		t.Fatalf("get = %#v, %v", got, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, deployment_id, profile_version, git_commit_hash")).WithArgs("p1", 50).WillReturnRows(releaseRows("r1"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT release_id, platform, status, approval_id")).WithArgs("r1").WillReturnRows(releasePlatformRows("r1"))
	list, err := repo.ListByProfile(ctx, "p1", 0)
	if err != nil || len(list) != 1 || list[0].ID != "r1" {
		t.Fatalf("list = %#v, %v", list, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
