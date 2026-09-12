package releases

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	internalreleases "deployment-manager/internal/releases"

	"github.com/DATA-DOG/go-sqlmock"
	_ "modernc.org/sqlite"
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
	mock.ExpectExec(regexp.QuoteMeta("UPDATE releases SET deployment_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1")).WithArgs("r1", "dep-1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.SetDeploymentID(ctx, "r1", "dep-1"); err != nil {
		t.Fatalf("deployment identity: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta("UPDATE releases SET verification_evidence = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1")).WithArgs("r1", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.SetVerificationEvidence(ctx, "r1", []VerificationItem{{Platform: "linux", ObservedVersion: "1.0"}}); err != nil {
		t.Fatalf("evidence: %v", err)
	}
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE releases
		SET readiness_goal_ref = $2, approved_at_commit = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`)).WithArgs("r1", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.SetReadinessApproval(ctx, "r1", "goal-1", "abc"); err != nil {
		t.Fatalf("readiness approval: %v", err)
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
	return releaseRowsWithEvidence(id, []byte(`[]`))
}

func releaseRowsWithEvidence(id string, evidence []byte) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{"id", "profile_id", "deployment_id", "profile_version", "git_commit_hash", "artifact_digest", "candidate_id", "destination_revision_id", "authorization_epoch", "idempotency_key", "readiness_review_key", "release_version", "channel", "status", "release_notes", "released_by", "promoted_from_release_id", "readiness_goal_ref", "approved_at_commit", "verification_evidence", "created_at", "published_at", "updated_at"}).
		AddRow(id, "p1", nil, 1, "abc", "sha256:test", "candidate-test", "destination-test", 1, "idem-test", "rr-test", "1.0.0", "stable", StatusPending, nil, nil, nil, nil, nil, evidence, now, nil, now)
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
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO releases")).WithArgs("r1", "p1", sqlmock.AnyArg(), sqlmock.AnyArg(), "abc", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), uint64(1), sqlmock.AnyArg(), sqlmock.AnyArg(), "1.0.0", "stable", StatusPending, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
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

func TestSQLRepositoryInsertRequiresRegisteredReleaseIdentities(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-insert-identity-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	repo := NewSQLRepository(db)
	err = repo.Insert(context.Background(), &Release{
		ID: "release-missing-identity", ProfileID: "p1", GitCommitHash: "commit-1", ArtifactDigest: "sha256:artifact",
		CandidateID: "candidate-missing", DestinationRevisionID: "destination-missing", ReleaseVersion: "1.0.0", Channel: "stable",
	})
	if err == nil {
		t.Fatal("release accepted unregistered candidate and destination identities")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM releases WHERE id = 'release-missing-identity'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("release row persisted after identity validation failure: %d", count)
	}
}

func TestSQLRepositoryRejectsMalformedVerificationEvidence(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	repo := NewSQLRepository(db)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, deployment_id, profile_version, git_commit_hash")).WithArgs("r-corrupt").WillReturnRows(
		releaseRowsWithEvidence("r-corrupt", []byte(`{"broken":`)),
	)
	if _, err := repo.Get(context.Background(), "r-corrupt"); err == nil {
		t.Fatal("malformed persisted verification evidence was accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
