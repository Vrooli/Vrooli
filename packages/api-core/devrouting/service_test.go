package devrouting_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/projectmeta"
	"github.com/vrooli/api-core/storage"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
)

func writeMode(t *testing.T, mode string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{"mode":"` + mode + `"}`
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	projectmeta.SetStartDirForTesting(dir)
}

func openRouted(t *testing.T) *database.RoutedDB {
	t.Helper()
	r, err := database.Open(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    filepath.Join(t.TempDir(), "primary.db"),
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r
}

func TestRegister_DevMode_MountsHandler(t *testing.T) {
	writeMode(t, "development")
	t.Setenv(apihttp.TestModeForceEnableEnv, "")

	db := openRouted(t)
	mux := http.NewServeMux()
	if !devrouting.Register(mux, db) {
		t.Fatalf("Register returned false in development mode")
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := routing_v1connect.NewRoutingServiceClient(http.DefaultClient, srv.URL)

	// ClearTestPool on an empty slot is a no-op success.
	if _, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: "lease-a"})); err != nil {
		t.Fatalf("ClearTestPool: %v", err)
	}

	// InstallTestPool with a fresh sqlite path.
	dsn := filepath.Join(t.TempDir(), "test.db")
	resp, err := client.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn, LeaseId: "lease-a"}))
	if err != nil {
		t.Fatalf("InstallTestPool: %v", err)
	}
	if resp.Msg.GetActiveLeaseId() != "lease-a" {
		t.Fatalf("active_lease_id = %q, want lease-a", resp.Msg.GetActiveLeaseId())
	}
	if !db.HasTestPool() {
		t.Fatalf("expected test pool installed")
	}

	// Conflicting install under a different lease → AlreadyExists.
	dsn2 := filepath.Join(t.TempDir(), "test2.db")
	_, err = client.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn2, LeaseId: "lease-b"}))
	if err == nil {
		t.Fatalf("expected AlreadyExists, got nil")
	}
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("expected CodeAlreadyExists, got %v", connect.CodeOf(err))
	}

	// Clearing with the wrong lease → FailedPrecondition.
	_, err = client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: "lease-b"}))
	if err == nil {
		t.Fatalf("expected FailedPrecondition on wrong-lease clear, got nil")
	}
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %v", connect.CodeOf(err))
	}

	if _, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: "lease-a"})); err != nil {
		t.Fatalf("ClearTestPool 2: %v", err)
	}
	if db.HasTestPool() {
		t.Fatalf("expected test pool cleared")
	}
}

func TestRegister_ProductionMode_NoOp(t *testing.T) {
	writeMode(t, "production")
	t.Setenv(apihttp.TestModeForceEnableEnv, "")

	db := openRouted(t)
	mux := http.NewServeMux()
	if devrouting.Register(mux, db) {
		t.Fatalf("Register returned true in production mode")
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := routing_v1connect.NewRoutingServiceClient(http.DefaultClient, srv.URL)
	// Path is not mounted → request fails (404 / unknown procedure).
	_, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: "lease-a"}))
	if err == nil {
		t.Fatalf("expected error calling unmounted ClearTestPool in production, got nil")
	}
}

func TestRegister_ForceEnableOverridesProduction(t *testing.T) {
	writeMode(t, "production")
	t.Setenv(apihttp.TestModeForceEnableEnv, "1")

	db := openRouted(t)
	mux := http.NewServeMux()
	if !devrouting.Register(mux, db) {
		t.Fatalf("Register returned false despite force-enable")
	}
}

func TestRegisterWithFileRootsOwnsLeasedRoots(t *testing.T) {
	writeMode(t, "development")
	db := openRouted(t)
	primary := filepath.Join(t.TempDir(), "primary-config")
	if err := os.MkdirAll(primary, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(primary, "seed.txt"), []byte("seed"), 0o644); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{ConfigDir: primary, DataDir: filepath.Join(t.TempDir(), "data"), CacheDir: filepath.Join(t.TempDir(), "cache"), LogsDir: filepath.Join(t.TempDir(), "logs"), StateDir: filepath.Join(t.TempDir(), "state")})
	mux := http.NewServeMux()
	if !devrouting.RegisterWithFileRoots(mux, db, roots) {
		t.Fatal("RegisterWithFileRoots returned false")
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := routing_v1connect.NewRoutingServiceClient(http.DefaultClient, srv.URL)
	leaseID := "lease-files"
	if _, err := client.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "test.db"), LeaseId: leaseID})); err != nil {
		t.Fatalf("InstallTestPool: %v", err)
	}
	testConfig, err := roots.Pick(database.WithTestMode(context.Background()), storage.ClassConfig)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(testConfig, "seed.txt")); err != nil {
		t.Fatalf("expected copied config seed: %v", err)
	}
	roots.RecordWrite(database.WithTestMode(context.Background()))
	cleared, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatalf("ClearTestPool: %v", err)
	}
	if cleared.Msg.GetStats().GetTestRootWrites() != 1 || cleared.Msg.GetStats().GetPrimaryRootWritesDuringTestMode() != 0 {
		t.Fatalf("unexpected file stats: %+v", cleared.Msg.GetStats())
	}
	if _, err := os.Stat(testConfig); !os.IsNotExist(err) {
		t.Fatalf("leased test root remains: %v", err)
	}
}

func TestRegisterFileRootsOwnsLeasedRootsWithoutDatabase(t *testing.T) {
	writeMode(t, "development")
	primary := filepath.Join(t.TempDir(), "primary-config")
	if err := os.MkdirAll(primary, 0o755); err != nil {
		t.Fatal(err)
	}
	roots := filerouting.New(storage.Paths{ConfigDir: primary, DataDir: filepath.Join(t.TempDir(), "data"), CacheDir: filepath.Join(t.TempDir(), "cache"), LogsDir: filepath.Join(t.TempDir(), "logs"), StateDir: filepath.Join(t.TempDir(), "state")})
	mux := http.NewServeMux()
	if !devrouting.RegisterFileRoots(mux, roots) {
		t.Fatal("RegisterFileRoots returned false")
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := routing_v1connect.NewRoutingServiceClient(http.DefaultClient, srv.URL)
	leaseID := "file-only"
	installed, err := client.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatalf("InstallTestPool: %v", err)
	}
	if !installed.Msg.GetFileRootsInstalled() {
		t.Fatal("file roots were not installed")
	}
	roots.RecordWrite(database.WithTestMode(context.Background()))
	cleared, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatalf("ClearTestPool: %v", err)
	}
	if got := cleared.Msg.GetStats().GetTestRootWrites(); got != 1 {
		t.Fatalf("test root writes = %d, want 1", got)
	}
}

func TestRegisterMountsConnectServiceSubtreeWhenRouterSupportsMount(t *testing.T) {
	t.Setenv(apihttp.TestModeForceEnableEnv, "1")
	db := openRouted(t)
	router := &mountRecordingMux{}

	if !devrouting.Register(router, db) {
		t.Fatal("Register returned false")
	}
	if router.mountPath != "/vrooli.dev_routing.v1.routing.RoutingService/" {
		t.Fatalf("mounted path = %q", router.mountPath)
	}
	if router.mounted == nil {
		t.Fatal("expected Connect handler to be mounted")
	}
	if router.handleCalls != 0 {
		t.Fatalf("Handle called %d times; mounted routers must use Mount", router.handleCalls)
	}
}

type mountRecordingMux struct {
	mountPath   string
	mounted     http.Handler
	handleCalls int
}

func (m *mountRecordingMux) Handle(string, http.Handler) {
	m.handleCalls++
}

func (m *mountRecordingMux) Mount(path string, handler http.Handler) {
	m.mountPath = path
	m.mounted = handler
}
