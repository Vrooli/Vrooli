package delivery

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
)

type routedTestStorageProvider struct {
	newCalls int
	storage  Storage
}

func (p *routedTestStorageProvider) ProviderKey() string { return "test" }

func (p *routedTestStorageProvider) New(context.Context, StorageSettings) (Storage, error) {
	p.newCalls++
	return p.storage, nil
}

type routedTestStorage struct{}

func (routedTestStorage) TestConnection(context.Context, string) error { return nil }
func (routedTestStorage) PresignGet(context.Context, string, string, time.Duration) (string, error) {
	return "https://leased.example/presigned", nil
}
func (routedTestStorage) PresignPut(context.Context, string, string, time.Duration, string) (string, map[string]string, error) {
	return "", nil, nil
}
func (routedTestStorage) HeadObject(context.Context, string, string) (string, int64, string, error) {
	return "", 0, "", nil
}

type expiringRequestResolver struct {
	routed *database.RoutedDB
	clock  *scheduletest.FakeClock
	calls  int
}

func (r *expiringRequestResolver) ResolveRequestStore(ctx context.Context) (Store, error) {
	r.calls++
	store, err := (strictRoutedStore{db: r.routed}).ResolveRequestStore(ctx)
	if err != nil {
		return nil, err
	}
	// The bound pool remains usable for this operation, but a second
	// PoolForContext call would now observe the expired lease.
	r.clock.Advance(2 * time.Minute)
	return store, nil
}

func seedStorageSettings(t *testing.T, dbPath string, bucket string) {
	t.Helper()
	db, err := openSQLiteDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	createStorageSettingsTable(t, db)
	if _, err := db.Exec(`INSERT INTO download_storage_settings
		(id, bundle_key, provider, bucket, region, force_path_style, signed_url_ttl_seconds, created_at, updated_at)
		VALUES (1, 'business-suite', 'test', ?, 'test', FALSE, 900, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, bucket); err != nil {
		t.Fatal(err)
	}
}

func createStorageSettingsTable(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`CREATE TABLE download_storage_settings (
		id INTEGER PRIMARY KEY, bundle_key TEXT, provider TEXT, bucket TEXT,
		region TEXT, endpoint TEXT, force_path_style BOOLEAN, default_prefix TEXT,
		signed_url_ttl_seconds INTEGER, public_base_url TEXT,
		created_at TIMESTAMP, updated_at TIMESTAMP
	)`); err != nil {
		t.Fatal(err)
	}
}

func openSQLiteDB(path string) (*sql.DB, error) {
	return sql.Open("sqlite", path)
}

func TestStrictRoutedStorePresignUsesLeaseAndRefusesMissingOrExpiredLease(t *testing.T) {
	primaryPath := filepath.Join(t.TempDir(), "primary.db")
	primary, err := openSQLiteDB(primaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := primary.Exec(`CREATE TABLE download_storage_settings (
		id INTEGER PRIMARY KEY, bundle_key TEXT, provider TEXT, bucket TEXT,
		region TEXT, endpoint TEXT, force_path_style BOOLEAN, default_prefix TEXT,
		signed_url_ttl_seconds INTEGER, public_base_url TEXT,
		created_at TIMESTAMP, updated_at TIMESTAMP
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := primary.Exec(`INSERT INTO download_storage_settings
		(id, bundle_key, provider, bucket, region, force_path_style, signed_url_ttl_seconds, created_at, updated_at)
		VALUES (1, 'business-suite', 'test', 'primary-bucket', 'test', FALSE, 900, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	_ = primary.Close()

	clock := scheduletest.New(time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC))
	routed, err := database.OpenWithClock(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: primaryPath}, clock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = routed.Close() })
	testPath := filepath.Join(t.TempDir(), "test.db")
	seedStorageSettings(t, testPath, "leased-bucket")

	provider := &routedTestStorageProvider{storage: routedTestStorage{}}
	service := NewServiceWithRequestStore(NewStrictRoutedStore(routed), provider)
	ctx := database.WithTestMode(context.Background())
	if _, err := service.PresignGetArtifact(ctx, "business-suite", Artifact{Bucket: "primary-bucket", ObjectKey: "asset.bin"}); err == nil {
		t.Fatal("missing lease reached primary storage settings or provider")
	}
	if provider.newCalls != 0 {
		t.Fatalf("missing lease constructed provider %d times, want 0", provider.newCalls)
	}
	if got := routed.LeaseStats().PrimaryDuringTestModeRequests; got != 0 {
		t.Fatalf("missing lease reached primary: %d requests", got)
	}
	if err := routed.InstallTestPool(context.Background(), testPath, "lease", time.Minute); err != nil {
		t.Fatal(err)
	}
	url, err := service.PresignGetArtifact(ctx, "business-suite", Artifact{Bucket: "leased-bucket", ObjectKey: "asset.bin"})
	if err != nil || url != "https://leased.example/presigned" {
		t.Fatalf("leased presign = %q, error = %v", url, err)
	}
	if provider.newCalls != 1 {
		t.Fatalf("provider construction calls = %d, want 1", provider.newCalls)
	}

	clock.Advance(2 * time.Minute)
	if _, err := service.PresignGetArtifact(ctx, "business-suite", Artifact{Bucket: "primary-bucket", ObjectKey: "asset.bin"}); err == nil {
		t.Fatal("expired lease reached primary settings or provider")
	}
	if provider.newCalls != 1 {
		t.Fatalf("expired lease constructed provider %d times, want 1", provider.newCalls)
	}

	if got := routed.LeaseStats().PrimaryDuringTestModeRequests; got != 0 {
		t.Fatalf("expired lease reached primary: %d requests", got)
	}
}

func TestRequestStoreBindingDoesNotReResolveAfterLeaseExpires(t *testing.T) {
	routed, clock := openDeliveryRoutedDB(t)
	createStorageSettingsTable(t, routed.Primary())
	testPath := filepath.Join(t.TempDir(), "test.db")
	seedStorageSettings(t, testPath, "leased-bucket")
	if err := routed.InstallTestPool(context.Background(), testPath, "lease", time.Minute); err != nil {
		t.Fatal(err)
	}

	resolver := &expiringRequestResolver{routed: routed, clock: clock}
	service := NewServiceWithRequestStore(resolver)
	settings, err := service.GetSettings(database.WithTestMode(context.Background()), "business-suite")
	if err != nil {
		t.Fatalf("bound settings lookup after lease expiry = %v", err)
	}
	if settings == nil || settings.Bucket != "leased-bucket" {
		t.Fatalf("bound settings = %+v, want leased-bucket", settings)
	}
	if resolver.calls != 1 {
		t.Fatalf("request resolver calls = %d, want one bound pool", resolver.calls)
	}
}
