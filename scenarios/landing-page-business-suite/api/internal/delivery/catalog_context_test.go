package delivery

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
	_ "modernc.org/sqlite"
)

type contextCatalogStore struct {
	db  *sql.DB
	ctx context.Context
}

func (s *contextCatalogStore) Query(string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected context-free query")
}
func (s *contextCatalogStore) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	s.ctx = ctx
	return s.db.QueryContext(ctx, query, args...)
}
func (s *contextCatalogStore) QueryRow(string, ...any) *sql.Row { return nil }
func (s *contextCatalogStore) Exec(string, ...any) (sql.Result, error) {
	return nil, errors.New("unexpected exec")
}
func (s *contextCatalogStore) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, errors.New("unexpected exec")
}
func (s *contextCatalogStore) BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error) {
	return nil, errors.New("unexpected transaction")
}

func TestGetAssetByIDContextUsesExactCatalogIdentityAndRequestPool(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := &contextCatalogStore{db: db}
	service := NewCatalogService(store)
	ctx := context.WithValue(context.Background(), struct{}{}, "leased-test")
	mock.ExpectQuery("SELECT id, bundle_key, app_key, platform").
		WithArgs(int64(42), "bundle", "desktop", "linux").
		WillReturnRows(sqlmock.NewRows([]string{"id", "bundle_key", "app_key", "platform", "artifact_url", "artifact_source", "artifact_id", "release_version", "release_notes", "checksum", "requires_entitlement", "metadata"}).
			AddRow(42, "bundle", "desktop", "linux", "/downloads/desktop", "direct", nil, "1.0.0", "", "sha256:test", false, nil))

	asset, err := service.GetAssetByIDContext(ctx, "bundle", "desktop", "linux", 42)
	if err != nil || asset == nil || asset.ID != 42 || asset.Platform != "linux" {
		t.Fatalf("exact catalog lookup = %+v, error = %v", asset, err)
	}
	if store.ctx != ctx {
		t.Fatalf("catalog lookup context = %v, want request context", store.ctx)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetAssetByIDContextDoesNotFallbackOnIdentityMismatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	service := NewCatalogService(&contextCatalogStore{db: db})
	mock.ExpectQuery("SELECT id, bundle_key, app_key, platform").
		WithArgs(int64(42), "bundle", "desktop", "linux").
		WillReturnRows(sqlmock.NewRows([]string{"id", "bundle_key", "app_key", "platform", "artifact_url", "artifact_source", "artifact_id", "release_version", "release_notes", "checksum", "requires_entitlement", "metadata"}))

	if _, err := service.GetAssetByIDContext(context.Background(), "bundle", "desktop", "linux", 42); !errors.Is(err, ErrAssetNotFound) {
		t.Fatalf("mismatched exact row error = %v, want ErrAssetNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func openDeliveryRoutedDB(t *testing.T) (*database.RoutedDB, *scheduletest.FakeClock) {
	t.Helper()
	clock := scheduletest.New(time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC))
	db, err := database.OpenWithClock(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    filepath.Join(t.TempDir(), "primary.db"),
	}, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, clock
}

func createDeliveryAssetTable(t *testing.T, db *sql.DB, assetID int64, url string) {
	t.Helper()
	_, err := db.Exec(`CREATE TABLE download_assets (
		id INTEGER PRIMARY KEY, bundle_key TEXT NOT NULL, app_key TEXT NOT NULL,
		platform TEXT NOT NULL, artifact_url TEXT, artifact_source TEXT,
		artifact_id INTEGER, release_version TEXT, release_notes TEXT,
		checksum TEXT, requires_entitlement BOOLEAN, metadata BLOB
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO download_assets
		(id, bundle_key, app_key, platform, artifact_url, artifact_source, release_version, release_notes, checksum, requires_entitlement)
		VALUES (?, ?, ?, ?, ?, 'direct', '1.0.0', '', 'checksum', FALSE)`,
		assetID, "business-suite", "web-console", "linux", url)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetAssetContextUsesLeasedCatalogAndNeverFallsBack(t *testing.T) {
	db, clock := openDeliveryRoutedDB(t)
	createDeliveryAssetTable(t, db.Primary(), 1, "https://primary.example/asset")
	testPath := filepath.Join(t.TempDir(), "test.db")
	testDB, err := sql.Open("sqlite", testPath)
	if err != nil {
		t.Fatal(err)
	}
	createDeliveryAssetTable(t, testDB, 2, "https://leased.example/asset")
	_ = testDB.Close()
	if err := db.InstallTestPool(context.Background(), testPath, "delivery-lease", time.Minute); err != nil {
		t.Fatal(err)
	}

	service := NewCatalogService(NewRoutedCatalogStore(db))
	ctx := database.WithTestMode(context.Background())
	asset, err := service.GetAssetContext(ctx, "business-suite", "web-console", "linux")
	if err != nil || asset == nil || asset.ID != 2 || asset.ArtifactURL != "https://leased.example/asset" {
		t.Fatalf("leased asset = %+v, error = %v", asset, err)
	}

	clock.Advance(2 * time.Minute)
	if _, err := service.GetAssetContext(ctx, "business-suite", "web-console", "linux"); err == nil {
		t.Fatal("expired lease returned an asset instead of failing closed")
	}
	if got := db.LeaseStats().PrimaryDuringTestModeRequests; got != 0 {
		t.Fatalf("strict catalog lookup fell through to primary: %d requests", got)
	}
}

func TestGetAssetContextMissingLeaseDoesNotReadPrimary(t *testing.T) {
	db, _ := openDeliveryRoutedDB(t)
	createDeliveryAssetTable(t, db.Primary(), 1, "https://primary.example/asset")
	service := NewCatalogService(NewRoutedCatalogStore(db))
	if _, err := service.GetAssetContext(database.WithTestMode(context.Background()), "business-suite", "web-console", "linux"); err == nil {
		t.Fatal("missing lease returned the primary asset")
	}
	if got := db.LeaseStats().PrimaryDuringTestModeRequests; got != 0 {
		t.Fatalf("missing lease reached primary: %d requests", got)
	}
}
