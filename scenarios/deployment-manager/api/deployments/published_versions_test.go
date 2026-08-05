package deployments

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func publishedVersionRows() *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{"id", "profile_id", "platform", "version", "git_commit_hash", "artifact_id", "deployment_id", "release_id", "published_at"}).
		AddRow(1, "p1", "linux-x64", "1.0.0", "abc", int64(7), "d1", "r1", now)
}

func TestSQLPublishedVersionsRepositoryPersistsAndQueriesHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLPublishedVersionsRepository(db)
	record := &PublishedVersion{ProfileID: "p1", Platform: "linux-x64", Version: "1.0.0"}
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO published_versions")).WithArgs("p1", "linux-x64", "1.0.0", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	if err := repo.RecordPublish(context.Background(), record); err != nil || record.ID != 1 || record.PublishedAt.IsZero() {
		t.Fatalf("record = %#v, %v", record, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT DISTINCT ON (platform)")).WithArgs("p1").WillReturnRows(publishedVersionRows())
	latest, err := repo.GetLatestByProfile(context.Background(), "p1")
	if err != nil || len(latest) != 1 || latest[0].ArtifactID != 7 {
		t.Fatalf("latest = %#v, %v", latest, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, platform, version, git_commit_hash, artifact_id, deployment_id, release_id, published_at")).WithArgs("p1", "linux-x64", 50).WillReturnRows(publishedVersionRows())
	history, err := repo.GetHistory(context.Background(), "p1", "linux-x64", 0)
	if err != nil || len(history) != 1 || history[0].GitCommitHash != "abc" {
		t.Fatalf("history = %#v, %v", history, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, profile_id, platform, version, git_commit_hash, artifact_id, deployment_id, release_id, published_at")).WithArgs("p1", 10).WillReturnRows(publishedVersionRows())
	if history, err := repo.GetHistory(context.Background(), "p1", "", 10); err != nil || len(history) != 1 {
		t.Fatalf("unfiltered history = %#v, %v", history, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPublishedVersionNullInt64(t *testing.T) {
	if value := nullInt64(0); value.Valid {
		t.Fatal("zero artifact should be null")
	}
	if value := nullInt64(3); !value.Valid || value.Int64 != 3 {
		t.Fatalf("non-zero artifact = %#v", value)
	}
}
