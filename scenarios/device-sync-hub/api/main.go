package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/clock"
	"device-sync-hub/internal/deviceauth"
	internaldevices "device-sync-hub/internal/devices"
	"device-sync-hub/internal/middleware"
	"device-sync-hub/internal/modules"
	internalrealtime "device-sync-hub/internal/realtime"
	"device-sync-hub/internal/server"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	devicesH "device-sync-hub/handlers/devices"
	healthH "device-sync-hub/handlers/health"
	realtimeH "device-sync-hub/handlers/realtime"
	transferH "device-sync-hub/handlers/transfer"
)

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The path scope is the variant-aware namespace (storage.ScenarioNamespace),
// not the bare slug: under a Baseline Modes shadow engagement the lifecycle
// injects VROOLI_STORAGE_NAMESPACE, so the shadow's SQLite file lands beside
// "<scenario>_shadow" and never shares live's database. Outside the lifecycle
// (local `go run`, tests) it falls back to the compile-time slug, so live paths
// are unchanged. This is why a generated scenario is shadow-safe with zero
// per-scenario work — see packages/api-core/storage/namespace.go.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("device-sync-hub")
	if err != nil {
		return "", fmt.Errorf("resolve device-sync-hub storage namespace: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: scenarioID},
		storage.ClassData,
		"device-sync-hub.db",
	)
	if err != nil {
		return "", fmt.Errorf("resolve device-sync-hub db path: %w", err)
	}
	return sqliteFileDSN(path)
}

func sqliteFileDSN(path string) (string, error) {
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "device-sync-hub"}) {
		return
	}

	dsn, err := sqliteDSN()
	if err != nil {
		log.Fatalf("sqlite configuration failed: %v", err)
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	// Owner identity, JWTs, and sessions are delegated to scenario-authenticator
	// over HTTP (docs/concepts/INTEGRATIONS.md). The endpoint is injected by the
	// lifecycle from the declared dependency. It is required and deliberately has
	// NO hardcoded default: silently defaulting an auth endpoint would mask a
	// misconfigured deployment and could point validation at an unintended host.
	// When unset the auth client is constructed with an empty base URL and fails
	// closed — owner-gated RPCs reject because no Identity can be injected.
	authServiceURL := strings.TrimSpace(os.Getenv("AUTH_SERVICE_URL"))
	if authServiceURL == "" {
		log.Print("scenario-authenticator endpoint is not configured; owner authentication will fail closed until it is set")
	}
	authClient := auth.NewClient(auth.Config{BaseURL: authServiceURL})

	clk := clock.System{}
	logger := log.Default()

	// The realtime hub is shared across domains: transfer emits item events
	// through it, the devices pairing path pushes approve banners, and the
	// devices read path overlays live online presence. It holds only in-memory
	// connection state, so a single instance per process is correct.
	hub := internalrealtime.NewHub(clk)

	// A device repository + authenticator + a read service back the cross-cutting
	// device-token needs: the deviceauth middleware resolves a presented token to
	// a TRUSTED device, and the transfer domain checks a directed item's target
	// against the trust group. These read over the same pool the devices routes
	// use; trust transitions (revoke) are seen immediately.
	devRepo := internaldevices.NewSQLiteRepository(db, clk)
	devAuthn := internaldevices.NewAuthenticator(devRepo)
	devTrustSvc := internaldevices.NewService(internaldevices.Config{
		Repo: devRepo, Clock: clk, Auth: authClient, Logger: logger,
	})

	// The transfer domain owns its blob store, retention purge, and quotas.
	transferWiring, err := transferH.New(db, clk, devTrustSvc, hub, logger)
	if err != nil {
		log.Fatalf("transfer module: %v", err)
	}

	srv := server.New(
		server.Deps{Clock: clk, Logger: logger},
		healthH.Module(db, "device-sync-hub-api", "1.0.0"),
		devicesH.Module(db, clk, authClient, hub, logger),
		transferWiring.Module,
		realtimeH.Module(hub, logger),
	)

	// Retention sweep: purge expired Held items and delivered Live items on an
	// interval. Cancelled on shutdown via the Cleanup hook below.
	purgeCtx, stopPurge := context.WithCancel(context.Background())
	go internaltransfer.RunPurgeLoop(purgeCtx, transferWiring.Service, internaltransfer.DefaultPurgeInterval, logger)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	// Two best-effort-inject auth middlewares wrap the API handler:
	//   - auth.Middleware validates an OWNER bearer token (Authorization header)
	//     and injects the owner Identity; devices owner-gated RPCs read it.
	//   - deviceauth.Middleware resolves a DEVICE token (X-Device-Token / ?token=)
	//     to a TRUSTED device and injects it; transfer + realtime read it.
	// Both inject when present and stay silent when absent, so each surface's
	// per-handler Require* gate is what fails closed. The two credentials are
	// independent: a device-to-device transfer call carries no owner JWT, and an
	// owner management call carries no device token.
	apiHandler := auth.Middleware(authClient, logger)(
		deviceauth.Middleware(devAuthn, logger)(srv.Handler()),
	)
	rootMux.Handle("/", apiHandler)

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	//
	// middleware.SecurityHeaders is the outermost wrap so every response —
	// including error and preflight responses — carries the hardening headers.
	handler := middleware.SecurityHeaders(apihttp.TestModeMiddleware(rootMux))

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error {
			stopPurge()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
