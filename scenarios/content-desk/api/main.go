package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"content-desk/internal/modules"
	"content-desk/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	artifactsH "content-desk/handlers/artifacts"
	campaignsH "content-desk/handlers/campaigns"
	claimsH "content-desk/handlers/claims"
	healthH "content-desk/handlers/health"
	ledgerH "content-desk/handlers/ledger"
	posttypesH "content-desk/handlers/posttypes"
	reviewH "content-desk/handlers/review"
	searchH "content-desk/handlers/search"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. File writers must select their class through fileRootPath so a
// test-mode request uses the lease-owned root instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("content-desk")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve content-desk storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

// fileRootPath is the template's mandatory file-store seam. Domain stores
// compose their relative paths from it rather than retaining startup root
// strings, so X-Vrooli-Test-Mode is honored independently per request.
func fileRootPath(ctx context.Context, roots *filerouting.RoutedRoots, class storage.Class, rel string) (string, error) {
	root, err := roots.Pick(ctx, class)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "content-desk"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "content-desk",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		artifactsH.Module(db),
		campaignsH.Module(db),
		claimsH.Module(db),
		healthH.Module(db, "content-desk-api", "1.0.0"),
		ledgerH.Module(db),
		posttypesH.Module(db),
		reviewH.Module(db),
		searchH.Module(db),
	)

	// Content Desk owns this descriptor and its live corpus. Search Hub receives
	// only the transport mapping; inability to reach it never blocks editorial
	// reads or lifecycle startup.
	registerCtx, cancelSearchRegistration := context.WithCancel(context.Background())
	defer cancelSearchRegistration()
	go searchregister.Register(registerCtx, searchregister.Config{
		ScenarioID:     "content-desk",
		SearchFilePath: filepath.Join("..", ".vrooli", "search.json"),
		Logger:         log.Default(),
	})

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
