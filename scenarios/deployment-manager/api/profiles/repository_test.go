package profiles

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func profileRows(id string) *sqlmock.Rows {
	now := time.Now()
	return sqlmock.NewRows([]string{"id", "name", "scenario", "tiers", "swaps", "secrets", "settings", "version", "created_at", "updated_at", "created_by", "updated_by"}).
		AddRow(id, "Demo", "demo", []byte(`["desktop"]`), []byte(`[]`), []byte(`[]`), []byte(`{}`), 1, now, now, "test", "test")
}

func TestSQLRepositoryCoreProfileOperations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLRepository(db)
	ctx := context.Background()
	getQuery := regexp.QuoteMeta("SELECT id, name, scenario, tiers, swaps, secrets, settings, version, created_at, updated_at, created_by, updated_by")
	mock.ExpectQuery(getQuery).WithArgs("p1").WillReturnRows(profileRows("p1"))
	got, err := repo.Get(ctx, "p1")
	if err != nil || got == nil || got.Scenario != "demo" {
		t.Fatalf("get: %+v %v", got, err)
	}
	mock.ExpectQuery(getQuery).WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if got, err := repo.Get(ctx, "missing"); err != nil || got != nil {
		t.Fatalf("missing get: %+v %v", got, err)
	}
	profile := &Profile{ID: "p1", Name: "Demo", Scenario: "demo", Tiers: []string{"desktop"}, Swaps: []Swap{}, Secrets: []string{}, Settings: map[string]interface{}{}}
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profiles")).WithArgs("p1", "Demo", "demo", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profile_versions")).WithArgs("p1", "Demo", "demo", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	if id, err := repo.Create(ctx, profile); err != nil || id != "p1" {
		t.Fatalf("create: %q %v", id, err)
	}
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM profiles")).WithArgs("p1").WillReturnResult(sqlmock.NewResult(0, 1))
	if deleted, err := repo.Delete(ctx, "p1"); err != nil || !deleted {
		t.Fatalf("delete: %v %v", deleted, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLRepositoryScenarioSwapsAndVersions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLRepository(db)
	ctx := context.Background()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT scenario, COALESCE(jsonb_array_length(tiers), 0)")).WithArgs("demo").WillReturnRows(sqlmock.NewRows([]string{"scenario", "tier_count"}).AddRow("demo", 2))
	if scenario, tiers, err := repo.GetScenarioAndTier(ctx, "demo"); err != nil || scenario != "demo" || tiers != 2 {
		t.Fatalf("scenario/tier: %q %d %v", scenario, tiers, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT scenario, COALESCE(jsonb_array_length(tiers), 0)")).WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if _, _, err := repo.GetScenarioAndTier(ctx, "missing"); err != ErrNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
	swap := Swap{From: "postgres", To: "sqlite", Reason: "local"}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, COALESCE(swaps, '[]'::jsonb)")).WithArgs("demo").WillReturnRows(sqlmock.NewRows([]string{"id", "swaps"}).AddRow("p1", []byte(`[]`)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE profiles")).WithArgs(sqlmock.AnyArg(), "p1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.AddSwap(ctx, "demo", swap); err != nil {
		t.Fatalf("add swap: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(swaps, '[]'::jsonb)")).WithArgs("demo").WillReturnRows(sqlmock.NewRows([]string{"swaps"}).AddRow([]byte(`[{"from":"postgres","to":"sqlite"}]`)))
	got, err := repo.GetSwaps(ctx, "demo")
	if err != nil || len(got) != 1 || got[0].To != "sqlite" {
		t.Fatalf("swaps: %+v %v", got, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLRepositoryListUpdateAndVersions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLRepository(db)
	ctx := context.Background()
	listQuery := regexp.QuoteMeta("SELECT id, name, scenario, tiers, swaps, secrets, settings, version, created_at, updated_at, created_by, updated_by")
	mock.ExpectQuery(listQuery).WillReturnRows(profileRows("p1"))
	profiles, err := repo.List(ctx)
	if err != nil || len(profiles) != 1 || profiles[0].ID != "p1" {
		t.Fatalf("list = %#v, %v", profiles, err)
	}
	mock.ExpectQuery(listQuery).WithArgs("p1").WillReturnRows(profileRows("p1"))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE profiles")).WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), 2, "p1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO profile_versions")).WithArgs("p1", 2, "Demo", "demo", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
	updated, err := repo.Update(ctx, "p1", map[string]interface{}{"tiers": []string{"desktop", "mobile"}, "settings": map[string]interface{}{"a": true}})
	if err != nil || updated == nil {
		t.Fatalf("update = %#v, %v", updated, err)
	}
	tiers, _ := updated.Tiers.([]string)
	if updated.Version != 2 || len(tiers) != 2 {
		t.Fatalf("updated profile = %#v", updated)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM profiles")).WithArgs("p1").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("p1"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT profile_id, version, name, scenario, tiers, swaps, secrets, settings, created_at, created_by, COALESCE(change_description, '')")).WithArgs("p1").WillReturnRows(sqlmock.NewRows([]string{"profile_id", "version", "name", "scenario", "tiers", "swaps", "secrets", "settings", "created_at", "created_by", "change_description"}).AddRow("p1", 1, "Demo", "demo", []byte(`[]`), []byte(`[]`), []byte(`[]`), []byte(`{}`), time.Now(), "system", "created"))
	versions, err := repo.GetVersions(ctx, "p1")
	if err != nil || len(versions) != 1 || versions[0].ChangeDescription != "created" {
		t.Fatalf("versions = %#v, %v", versions, err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id FROM profiles")).WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if versions, err := repo.GetVersions(ctx, "missing"); err != nil || len(versions) != 0 {
		t.Fatalf("missing versions = %#v, %v", versions, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLRepositoryAddSwapUpdatesExistingAndHandlesMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewSQLRepository(db)
	swap := Swap{From: "postgres", To: "sqlite", Reason: "updated"}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, COALESCE(swaps, '[]'::jsonb)")).WithArgs("p1").WillReturnRows(sqlmock.NewRows([]string{"id", "swaps"}).AddRow("p1", []byte(`[{"from":"postgres","to":"sqlite","reason":"old"}]`)))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE profiles")).WithArgs(sqlmock.AnyArg(), "p1").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.AddSwap(context.Background(), "p1", swap); err != nil {
		t.Fatalf("update existing swap: %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, COALESCE(swaps, '[]'::jsonb)")).WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if err := repo.AddSwap(context.Background(), "missing", swap); err != ErrNotFound {
		t.Fatalf("missing swap error = %v", err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(swaps, '[]'::jsonb)")).WithArgs("missing").WillReturnError(sql.ErrNoRows)
	if _, err := repo.GetSwaps(context.Background(), "missing"); err != ErrNotFound {
		t.Fatalf("missing GetSwaps error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
