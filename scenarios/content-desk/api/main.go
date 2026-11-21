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
	boardH "content-desk/handlers/board"
	campaignsH "content-desk/handlers/campaigns"
	capabilitiesH "content-desk/handlers/capabilities"
	claimsH "content-desk/handlers/claims"
	healthH "content-desk/handlers/health"
	ledgerH "content-desk/handlers/ledger"
	posttypesH "content-desk/handlers/posttypes"
	reviewH "content-desk/handlers/review"
	searchH "content-desk/handlers/search"
	programruntime "content-desk/integrations/programruntime"
	"content-desk/internal/aisearch"
	internalartifacts "content-desk/internal/artifacts"
	internalcampaigns "content-desk/internal/campaigns"
	capabilitiesInternal "content-desk/internal/capabilities"
	internalclaims "content-desk/internal/claims"
	internalledger "content-desk/internal/ledger"
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

	// The capability catalog is owned by content-desk. Seed the canonical
	// 19-category catalog on startup; seeding inserts only absent records so
	// operator edits and observed qualifications survive restarts.
	if seeded, err := capabilitiesInternal.SeedCatalog(context.Background(), capabilitiesInternal.NewRepository(db)); err != nil {
		log.Fatalf("capability catalog seed failed: %v", err)
	} else if seeded.Inserted > 0 {
		log.Printf("capability catalog: seeded %d capability(ies), %d already present", seeded.Inserted, seeded.Existing)
	}
	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	// Search wiring: the scenario-owned .vrooli/search.json is the SSOT for the
	// provider descriptor and tuning. Start assembles the shared engine over the
	// authoritative editorial corpus (best-effort — a down Ollama/Qdrant degrades
	// to the lexical projection). The cancellable context is stopped on cleanup
	// so the engine's reconcile/sync goroutines exit with the process.
	searchCtx, cancelSearch := context.WithCancel(context.Background())
	searchJSONPath := filepath.Join("..", ".vrooli", "search.json")
	searchSource := aisearch.NewStoreSource(internalartifacts.NewSQLiteRepository(db), internalledger.NewSQLiteRepository(db)).
		WithCapabilityCatalog(capabilitiesInternal.NewRepository(db), nil, nil).
		WithCampaigns(internalcampaigns.NewSQLiteRepository(db)).
		WithClaims(internalclaims.NewLibrary(db, internalclaims.LocalRunner{}))
	searcher := aisearch.Start(searchCtx, searchSource, searchJSONPath, log.Default())

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		artifactsH.Module(db),
		boardH.Module(programruntime.NewClient()),
		campaignsH.Module(db),
		capabilitiesH.Module(db),
		claimsH.Module(db),
		healthH.Module(db, "content-desk-api", "1.0.0"),
		ledgerH.Module(db),
		posttypesH.Module(db),
		reviewH.Module(db),
		searchH.Module(db, searcher),
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
		Cleanup: func(ctx context.Context) error {
			cancelSearch()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
