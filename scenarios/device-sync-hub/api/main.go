package main

import (
	"context"
	"log"
	"net/http"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/deviceauth"
	internaldevices "device-sync-hub/internal/devices"
	"device-sync-hub/internal/middleware"
	"device-sync-hub/internal/modules"
	internalrealtime "device-sync-hub/internal/realtime"
	"device-sync-hub/internal/server"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	devicesH "device-sync-hub/handlers/devices"
	healthH "device-sync-hub/handlers/health"
	identityH "device-sync-hub/handlers/identity"
	realtimeH "device-sync-hub/handlers/realtime"
	transferH "device-sync-hub/handlers/transfer"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "device-sync-hub"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "device-sync-hub",
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
	// (docs/concepts/INTEGRATIONS.md). The hub verifies owner tokens LOCALLY
	// against the authenticator's published RS256 key (JWKS); the authenticator's
	// API URL is resolved at runtime *by name* via api-core/discovery — no
	// AUTH_SERVICE_URL env var and no hardcoded port (discovery asks the
	// lifecycle for the live port). The authenticator is contacted only to fetch
	// the signing key (cached) and to revoke sessions, never per request, so a
	// momentarily-down authenticator never breaks an already-issued session.
	// One resolver, shared by the owner-token verifier and the identity
	// forwarder, resolves scenario-authenticator's URL by name at call time.
	authResolver := discovery.NewResolver(discovery.ResolverConfig{})
	authClient := auth.NewClient(auth.Config{Resolver: authResolver})

	clk := schedule.System()
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
		identityH.Module(authResolver, logger),
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
