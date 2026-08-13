package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"react-component-library/internal/modules"
	"react-component-library/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	adoptionsH "react-component-library/handlers/adoptions"
	catalogH "react-component-library/handlers/catalog"
	componentsH "react-component-library/handlers/components"
	componentTestsH "react-component-library/handlers/componenttests"
	depsH "react-component-library/handlers/deps"
	healthH "react-component-library/handlers/health"
	inventoryH "react-component-library/handlers/inventory"
	previewH "react-component-library/handlers/preview"
	themesH "react-component-library/handlers/themes"
	versionsH "react-component-library/handlers/versions"
	workflowsH "react-component-library/handlers/workflows"
	capabilitiesH "react-component-library/internal/capabilities"

	"react-component-library/internal/uimanifest"

	adoptionsInternal "react-component-library/internal/adoptions"
	componentsInternal "react-component-library/internal/components"
	depsInternal "react-component-library/internal/deps"
	experienceInternal "react-component-library/internal/experience"
	themesInternal "react-component-library/internal/themes"
	versionledgerInternal "react-component-library/internal/versionledger"
	workflowsInternal "react-component-library/internal/workflows"
)

// componentsDepsObserver is the bridge from the components indexer's
// UpsertObserver seam to the deps service's SyncForComponent. Parses
// the component's @deps header field (req DC-001) and re-syncs the
// dep declarations table for that component. Errors are returned so
// the indexer surfaces a clear "header malformed" signal in the
// IndexComponents response; a successful re-sync with zero
// declarations clears any prior rows for the component.
type componentsDepsObserver struct {
	svc    depsInternal.Service
	logger *log.Logger
}

func (o *componentsDepsObserver) Observe(ctx context.Context, c componentsInternal.Component, in componentsInternal.IndexManifestInput) error {
	var declarations []depsInternal.DeclarationFields
	for _, v := range in.Versions {
		parsed, err := depsInternal.ParseHeaderField(v.Headers["deps"])
		if err != nil {
			return fmt.Errorf("parse @deps for %s@%s: %w", c.LibraryID, v.Version, err)
		}
		for _, d := range parsed {
			d.Version = v.Version
			declarations = append(declarations, d)
		}
	}
	return o.svc.SyncForComponent(ctx, depsInternal.SyncInput{
		ComponentID:  c.ID,
		LibraryID:    c.LibraryID,
		Declarations: declarations,
	})
}

// sqliteDSN resolves the SQLite database file path and wraps it in a DSN
// with the canonical pragma string. Resolution order:
//
//  1. SQLITE_PATH env — the canonical override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. storage.NewResolver(ProfileAuto) — the storage-steer-mandated
//     filesystem-safe-by-default location.
//
// The pragmas mirror agent-inbox; tweak in lockstep with
// internal/testutil/db.NewSQLite so production and tests open files the
// same way.
// fileRootPath is the request-scoped file-store seam. It keeps the primary
// SQLite file on the normal data root while allowing validation requests to
// resolve any file-backed path through the leased test root. Every file store
// must use RoutedRoots.Pick(ctx, class) rather than retaining a live root.
func fileRootPath(ctx context.Context, roots *filerouting.RoutedRoots, class storage.Class, rel string) (string, error) {
	root, err := roots.Pick(ctx, class)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, rel), nil
}

func sqliteDSN(roots *filerouting.RoutedRoots) (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return sqliteFileDSN(path)
	}
	if path := strings.TrimSpace(os.Getenv("SQLITE_DB")); path != "" {
		return sqliteFileDSN(path)
	}

	path, err := fileRootPath(context.Background(), roots, storage.ClassData, "react-component-library.db")
	if err != nil {
		return "", fmt.Errorf("resolve react-component-library db path through routed roots: %w", err)
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

// scenarioStorageRoots resolves every storage class once at startup. Any
// request-scoped file operation must select its class through RoutedRoots so
// test-genie's leased isolation root is honored instead of the live tree.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("react-component-library")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve react-component-library storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "react-component-library"}) {
		return
	}

	primaryFileRoots, err := scenarioStorageRoots()
	if err != nil {
		log.Fatalf("file storage configuration failed: %v", err)
	}
	fileRoots := filerouting.New(primaryFileRoots)

	dsn, err := sqliteDSN(fileRoots)
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
	primaryDB := db.Primary()

	if err := componentsInternal.EnsureSchemaMigrations(context.Background(), primaryDB); err != nil {
		log.Fatalf("schema migration failed: %v", err)
	}
	if err := adoptionsInternal.EnsureSchemaMigrations(context.Background(), primaryDB); err != nil {
		log.Fatalf("adoption schema migration failed: %v", err)
	}
	if err := workflowsInternal.EnsureSchemaMigrations(context.Background(), primaryDB); err != nil {
		log.Fatalf("assisted workflow schema migration failed: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), primaryDB, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	sourceRoot, err := componentsH.DefaultSourceRoot()
	if err != nil {
		log.Fatalf("components source root: %v", err)
	}
	componentsSvc, componentsRepo := componentsH.BuildService(primaryDB, schedule.System(), sourceRoot)
	versionLedger := versionledgerInternal.NewRepository(primaryDB, sourceRoot)
	if err := versionLedger.Rebuild(context.Background()); err != nil {
		log.Fatalf("rebuild version ledger: %v", err)
	}

	scenariosRoot, err := adoptionsH.DefaultScenariosRoot()
	if err != nil {
		log.Fatalf("adoptions scenarios root: %v", err)
	}
	componentsInternal.SetServiceJSONReader(componentsSvc, componentsInternal.NewFSServiceJSONReader(scenariosRoot))
	adoptionsSvc, scenariosReader := adoptionsH.BuildService(primaryDB, schedule.System(), adoptionsH.LibraryFromComponents(componentsSvc), scenariosRoot)
	if tokenReader, ok := scenariosReader.(adoptionsInternal.ScenarioTokenNamespaceReader); ok {
		adoptionsInternal.SetTokenNamespaceReader(adoptionsSvc, tokenReader)
	} else {
		log.Fatalf("adoptions scenarios reader does not expose token namespace resolution")
	}
	componentsInternal.SetScenarioSourceReader(componentsSvc, scenariosReader)

	// Install the swarm-manager drift reporter so Refresh files a
	// `fix` backlog item when an adoption first transitions to
	// behind/modified. CLI-only per [feedback_skills_use_cli_never_api.md];
	// disable by setting RCL_DRIFT_REPORTER=off.
	if strings.TrimSpace(os.Getenv("RCL_DRIFT_REPORTER")) != "off" {
		reporter := adoptionsInternal.NewSwarmManagerCLIReporter(nil)
		if path := strings.TrimSpace(os.Getenv("RCL_SWARM_MANAGER_BIN")); path != "" {
			reporter.BinaryPath = path
		}
		adoptionsInternal.SetDriftReporter(adoptionsSvc, reporter, log.Default())
	}

	versionsResolver := versionsH.AdoptionResolverFromService(adoptionsSvc, scenariosReader)
	versionsSvc := versionsH.BuildService(primaryDB, schedule.System(), versionsResolver)

	// Wire post-save versions recording. Listener errors are logged
	// inside the adapter; UpdateContent does not fail when recording
	// fails (the file is already on disk).
	componentsInternal.SetContentChangeListener(componentsSvc, &versionsH.ListenerAdapter{
		Service: versionsSvc,
		Logger:  log.Default(),
	})

	// Wire the deps domain (req 10). Shares the adoptions scenarios
	// root so the package.json reader walks the same tree adoption
	// refresh does. depsObserver re-syncs declarations after every
	// successful components Upsert so the registry stays in step with
	// the on-disk @deps headers.
	depsSvc := depsH.BuildService(primaryDB, depsInternal.NewFSPackageJSONReader(scenariosRoot))
	depsObserver := &componentsDepsObserver{svc: depsSvc, logger: log.Default()}
	adoptionsInternal.SetValidationGates(adoptionsSvc, depsSvc, componentsSvc)

	// Wire the themes domain (req 12). Same scenariosRoot as adoptions
	// + deps so the DESIGN.md reader walks the same tree. Seed the
	// built-in themes table on first boot; idempotent on re-runs.
	themesSvc := themesH.BuildService(primaryDB, themesInternal.NewFSDesignMDReader(scenariosRoot))
	if err := themesSvc.EnsureBuiltinsSeeded(context.Background()); err != nil {
		log.Fatalf("seed built-in themes: %v", err)
	}
	// Suggestions consume the same InventoryService.ScanScenario implementation
	// as ui-health rather than maintaining a second filesystem interpretation.
	inventoryScanner := inventoryH.NewConnectHandler(inventoryH.Deps{
		Logger:        log.Default(),
		Adoptions:     inventoryH.AdoptionsServiceAdapter{Service: adoptionsSvc},
		ManifestLoad:  uimanifest.NewFSLoader(filepath.Dir(scenariosRoot)),
		ScenariosRoot: scenariosRoot,
	})

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		capabilitiesH.Module(),
		adoptionsH.ModuleFromService(
			adoptionsSvc,
			log.Default(),
			adoptionsH.WithResolver(
				adoptionsH.BuildResolver(filepath.Dir(scenariosRoot)),
				&adoptionsH.IndexedSlotReader{Components: componentsSvc},
				adoptionsH.LibraryFromComponents(componentsSvc),
			),
			adoptionsH.WithSuggestions(componentsSvc, depsSvc, inventoryScanner, scenariosRoot),
		),
		componentsH.ModuleFromService(componentsSvc, componentsRepo, sourceRoot, log.Default(), componentsH.WithIndexObserver(depsObserver), componentsH.WithExperienceReader(experienceInternal.NewReader(filepath.Dir(scenariosRoot))), componentsH.WithVersionLedger(versionLedger)),
		componentTestsH.Module(primaryDB, componentsSvc, sourceRoot, log.Default()),
		catalogH.Module(filepath.Dir(scenariosRoot), primaryDB),
		depsH.ModuleFromService(depsSvc, log.Default()),
		healthH.Module(primaryDB, "react-component-library-api", "1.0.0"),
		inventoryH.Module(log.Default(), scenariosRoot, inventoryH.AdoptionsServiceAdapter{Service: adoptionsSvc}, uimanifest.NewFSLoader(filepath.Dir(scenariosRoot))),
		previewH.ModuleWithDepsAtRoot(componentsSvc, depsSvc, log.Default(), filepath.Dir(scenariosRoot)),
		themesH.ModuleFromService(themesSvc, log.Default()),
		versionsH.ModuleWithLedger(primaryDB, schedule.System(), versionsResolver, log.Default(), versionLedger),
		workflowsH.ModuleWithReadiness(primaryDB, schedule.System(), workflowsInternal.NewAgentManagerDispatcher(), workflowsInternal.NewPromotionReadinessReader(componentsSvc, adoptionsSvc), log.Default()),
	)

	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/", srv.Handler())
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// The catalog provider executes one browser-backed report per latest
		// asset. Keep the response open for the descriptor's 300s phase budget
		// plus a bounded transport margin so a complete aggregate result is not
		// truncated into an opaque unexpected EOF.
		WriteTimeout: 6 * time.Minute,
		Cleanup:      func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
