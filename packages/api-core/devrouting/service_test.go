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
	"github.com/vrooli/api-core/projectmeta"

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

	// ClearTestPool is a safe no-op call; succeeds.
	if _, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{})); err != nil {
		t.Fatalf("ClearTestPool: %v", err)
	}

	// InstallTestPool with a fresh sqlite path.
	dsn := filepath.Join(t.TempDir(), "test.db")
	if _, err := client.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn})); err != nil {
		t.Fatalf("InstallTestPool: %v", err)
	}
	if !db.HasTestPool() {
		t.Fatalf("expected test pool installed")
	}

	if _, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{})); err != nil {
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
	_, err := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{}))
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
