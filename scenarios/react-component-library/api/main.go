package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	captureInternal "react-component-library/internal/designcapture"
	critiqueInternal "react-component-library/internal/designcritique"
	inferenceInternal "react-component-library/internal/designinference"

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

	activityH "react-component-library/handlers/activity"
	adoptionsH "react-component-library/handlers/adoptions"
	brandH "react-component-library/handlers/brand"
	catalogH "react-component-library/handlers/catalog"
	componentsH "react-component-library/handlers/components"
	componentTestsH "react-component-library/handlers/componenttests"
	depsH "react-component-library/handlers/deps"
	designsH "react-component-library/handlers/designs"
	healthH "react-component-library/handlers/health"
	inventoryH "react-component-library/handlers/inventory"
	libraryhealthH "react-component-library/handlers/libraryhealth"
	pageH "react-component-library/handlers/page"
	previewH "react-component-library/handlers/preview"
	reviewH "react-component-library/handlers/review"
	searchH "react-component-library/handlers/search"
	sketchH "react-component-library/handlers/sketch"
	templatesH "react-component-library/handlers/templates"
	themesH "react-component-library/handlers/themes"
	versionsH "react-component-library/handlers/versions"
	workflowsH "react-component-library/handlers/workflows"
	workspacesH "react-component-library/handlers/workspaces"
	capabilitiesH "react-component-library/internal/capabilities"

	"github.com/vrooli/api-core/uimanifest"
	"react-component-library/internal/availability"

	activityInternal "react-component-library/internal/activity"
	adoptionsInternal "react-component-library/internal/adoptions"
	brandInternal "react-component-library/internal/brand"
	catalogcoverageInternal "react-component-library/internal/catalogcoverage"
	componentsInternal "react-component-library/internal/components"
	componenttestsInternal "react-component-library/internal/componenttests"
	depsInternal "react-component-library/internal/deps"
	designsInternal "react-component-library/internal/designs"
	experienceInternal "react-component-library/internal/experience"
	libraryhealthInternal "react-component-library/internal/libraryhealth"
	reviewInternal "react-component-library/internal/review"
	searchInternal "react-component-library/internal/search"
	templatesInternal "react-component-library/internal/templates"
	themesInternal "react-component-library/internal/themes"
	versionledgerInternal "react-component-library/internal/versionledger"
	workflowsInternal "react-component-library/internal/workflows"
)

// workspaceActivityRecorder is the one adapter from the workspace aggregate's
// mutation seam to the append-only activity log. The workspace package owns
// what changed; activity owns who/when and the durable trail.
//
// It also coalesces a search rebuild after a mutation burst. Workspace saves
// autosave while a person edits, so rebuilding on every write would repeat the
// same work; one rebuild after the burst keeps the index current without
// turning typing into index churn.
type workspaceActivityRecorder struct {
	store  *activityInternal.Store
	search *searchInternal.Service
	mu     sync.Mutex
	timer  *time.Timer
}

func (r *workspaceActivityRecorder) Record(ctx context.Context, action, subjectID string, before, after []byte) error {
	if err := r.store.Record(ctx, activityInternal.Event{
		Actor:       "local",
		Action:      action,
		SubjectKind: "workspace",
		SubjectID:   subjectID,
		Summary:     action,
		BeforeJSON:  string(before),
		AfterJSON:   string(after),
	}); err != nil {
		return err
	}
	r.scheduleReindex()
	return nil
}

const workspaceReindexDelay = 5 * time.Second

func (r *workspaceActivityRecorder) scheduleReindex() {
	if r.search == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.timer != nil {
		r.timer.Stop()
	}
	r.timer = time.AfterFunc(workspaceReindexDelay, func() {
		count, _, err := r.search.Reindex(context.Background(), false)
		if err != nil {
			slog.Error("workspace search reindex failed", "error", err)
			return
		}
		slog.Info("workspace search reindex complete", "documents", count)
	})
}

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

// sqliteDSN returns the DSN for react-component-library's own database.
//
// The path comes from the routed roots rather than from storage.SQLitePath,
// because Test Genie may lease this scenario an isolated data root for the
// duration of a run; RoutedRoots.Pick is what honours that lease. The pragmas
// still come from the one owned seam.
func sqliteDSN(roots *filerouting.RoutedRoots) (string, error) {
	path, err := fileRootPath(context.Background(), roots, storage.ClassData, "react-component-library.db")
	if err != nil {
		return "", fmt.Errorf("resolve react-component-library db path through routed roots: %w", err)
	}
	return storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
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
	// Catalog gates and evidence capture are jobs, not serving-path work. Give
	// them an independent read-mostly pool so a cancelled or long-running gate
	// cannot occupy the connection used by the workbench and domain RPCs.
	jobDB, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 2,
		MaxIdleConns: 2,
	})
	if err != nil {
		log.Fatalf("job database connection failed: %v", err)
	}
	healthDB, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("health database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), primaryDB, modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}
	if err := designsInternal.Migrate(context.Background(), primaryDB); err != nil {
		log.Fatalf("designs schema migration failed: %v", err)
	}
	if err := templatesInternal.Migrate(context.Background(), primaryDB); err != nil {
		log.Fatalf("templates schema migration failed: %v", err)
	}
	// The workspace aggregate is the replacement composition owner. Migrate
	// imports legacy studio designs exactly once, guarded by
	// workspace_migrations, so a design deleted after import cannot reappear.
	if err := workspacesH.Migrate(context.Background(), primaryDB); err != nil {
		log.Fatalf("workspace schema migration failed: %v", err)
	}
	activityStore := activityInternal.NewSQLiteStore(primaryDB)
	healthProviders := libraryhealthInternal.DBProviders(primaryDB)
	capabilityRegistry := capabilitiesH.NewRegistry()
	healthProviders.Services = func(ctx context.Context) ([]libraryhealthInternal.Metric, error) {
		states := capabilityRegistry.Resolve(ctx)
		metrics := make([]libraryhealthInternal.Metric, 0, len(states))
		for _, state := range states {
			value := 0
			if state.Status == capabilitiesH.StatusAvailable {
				value = 1
			}
			metrics = append(metrics, libraryhealthInternal.Metric{
				Key: state.ID, Label: state.Name, Value: value, Total: 1, Detail: string(state.Status),
			})
		}
		return metrics, nil
	}
	if err := componentsInternal.EnsureMigrations(context.Background(), primaryDB); err != nil {
		log.Fatalf("components schema migration failed: %v", err)
	}
	if err := componenttestsInternal.NewSQLiteRepository(primaryDB).EnforcePayloadCeiling(context.Background()); err != nil {
		log.Fatalf("component test payload retention failed: %v", err)
	}

	sourceRoot, err := componentsH.DefaultSourceRoot()
	if err != nil {
		log.Fatalf("components source root: %v", err)
	}
	// The template shelf is reviewed JSON beside the library. A seed version
	// bump applies a reviewed content change exactly once; a plain restart is a
	// no-op. It runs after the legacy-design import so a curated id can never be
	// confused with an imported one.
	seedChanged := false
	if seeded, err := workspacesH.SeedFromFile(context.Background(), primaryDB, filepath.Join(sourceRoot, "workspace-seeds.json")); err != nil {
		slog.Error("workspace seed load failed", "error", err)
	} else if seeded.Inserted+seeded.Updated+seeded.Removed > 0 {
		seedChanged = true
		slog.Info("workspace seeds applied", "inserted", seeded.Inserted, "updated", seeded.Updated, "removed", seeded.Removed)
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
	var materializer componentsInternal.Materializer
	if candidate, ok := componentsSvc.(componentsInternal.Materializer); ok {
		materializer = candidate
	}
	presenceReconciler := versionledgerInternal.NewPresenceReconciler(versionLedger, componentsSvc, materializer)
	adoptionsInternal.SetPresenceReconciler(adoptionsSvc, presenceReconciler)

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
	// The registry is source-backed, so a fresh process must reconcile it before
	// handlers that resolve catalog ids (component tests, previews, and
	// adoptions) serve requests. Keep this on the same indexer and observer seam
	// as the explicit IndexComponents RPC; otherwise a persistent database can
	// silently remain stale after a checkout or a process restart.
	startupIndexer := componentsInternal.NewIndexer(componentsRepo, sourceRoot, nil)
	startupIndexer.SetUpsertObserver(depsObserver)
	if indexResult, indexErr := startupIndexer.Run(context.Background()); indexErr != nil {
		slog.Error("startup component index failed", "error", indexErr)
	} else if len(indexResult.Errors) > 0 {
		slog.Info("startup component index completed", "errors", len(indexResult.Errors), "indexed", indexResult.Indexed, "deleted", indexResult.Deleted)
		for _, indexErr := range indexResult.Errors {
			slog.Error("startup component index error", "error", indexErr)
		}
	}
	previewSvc := previewH.BuildServiceAtRoot(componentsSvc, depsSvc, filepath.Dir(scenariosRoot))
	adoptionsInternal.SetValidationGates(adoptionsSvc, depsSvc, componentsSvc)
	catalogEvidence := catalogcoverageInternal.NewEvidenceStore(primaryDB)
	adoptionsInternal.SetMaturityReader(adoptionsSvc, adoptionsInternal.NewCatalogMaturityReader(filepath.Dir(scenariosRoot), catalogEvidence))
	adoptionsInternal.SetContractCoverageReader(adoptionsSvc, adoptionsInternal.NewCatalogGateReader(catalogEvidence))

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

	compileAsset := func(ctx context.Context, id, version string) (string, error) {
		bundle, err := previewSvc.GetBundleVersion(ctx, id, version)
		return bundle.SHA256, err
	}
	availabilityFactory := func(ctx context.Context) (*availability.Snapshot, error) {
		return availability.NewSnapshot(ctx, componentsSvc, compileAsset)
	}
	// Search asks for availability once per document. Reuse one snapshot for a
	// bounded window instead of rebuilding the published-version graph 478
	// times during one reindex.
	var searchSnapshotMu sync.Mutex
	var searchSnapshot *availability.Snapshot
	var searchSnapshotAt time.Time
	searchAvailability := func(ctx context.Context, catalogID string) (bool, string, error) {
		searchSnapshotMu.Lock()
		if searchSnapshot == nil || time.Since(searchSnapshotAt) > time.Minute {
			snapshot, err := availabilityFactory(ctx)
			if err != nil {
				searchSnapshotMu.Unlock()
				return false, string(availability.Unavailable), err
			}
			searchSnapshot = snapshot
			searchSnapshotAt = time.Now()
		}
		snapshot := searchSnapshot
		searchSnapshotMu.Unlock()
		result := snapshot.Resolve(ctx, catalogID, snapshot.Latest(catalogID))
		return result.IsBuilt(), string(result.State), nil
	}
	// sourceRoot is <scenario>/library; the scenario root is its parent. Search
	// reads catalog/ from that root, not the repository's scenarios/ directory.
	scenarioRoot := filepath.Dir(sourceRoot)
	searchSvc := searchH.NewService(primaryDB, scenarioRoot, searchAvailability)
	// A seed change is a content change the index has no other way to observe:
	// the loader writes directly, not through the API recorder.
	if count, indexedAt, err := searchSvc.Reindex(context.Background(), seedChanged); err != nil {
		slog.Error("startup search index failed", "error", err)
	} else {
		slog.Info("startup search index complete", "documents", count, "indexed_at", indexedAt)
	}
	// Review projects ~14k findings from the evidence corpus. Warming it once at
	// boot keeps the first Review view fast and makes the projection cost part
	// of startup rather than a surprise on the first request.
	reviewSvc := reviewInternal.NewService(primaryDB, reviewInternal.NewEvidenceProjector(primaryDB))
	if _, err := reviewSvc.List(context.Background(), reviewInternal.Filter{}, 1, 0); err != nil {
		slog.Error("startup review projection failed", "error", err)
	} else {
		slog.Info("startup review projection complete")
	}
	workspaceRecorder := &workspaceActivityRecorder{store: activityStore, search: searchSvc}
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		capabilitiesH.Module(),
		activityH.ModuleFromService(activityStore),
		workspacesH.ModuleWithRecorderAndAssistant(primaryDB, workspaceRecorder,
			inferenceInternal.NewService(inferenceInternal.NewSQLiteRepository(primaryDB))),
		reviewH.ModuleFromService(reviewSvc),
		searchH.ModuleWithService(searchSvc),
		libraryhealthH.ModuleFromProviders(healthProviders),
		pageH.Module(filepath.Dir(scenariosRoot)),
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
		componentsH.ModuleFromService(componentsSvc, componentsRepo, sourceRoot, log.Default(), componentsH.WithIndexObserver(depsObserver), componentsH.WithExperienceReader(experienceInternal.NewReader(filepath.Dir(scenariosRoot))), componentsH.WithVersionLedger(versionLedger), componentsH.WithPreviewService(previewSvc), componentsH.WithPresenceReconciler(presenceReconciler)),
		componentTestsH.ModuleWithGeneratedFixture(primaryDB, componentsSvc, adoptionsSvc, sourceRoot, log.Default()),
		catalogH.ModuleWithCapture(filepath.Dir(scenariosRoot), jobDB.Primary(), componentsSvc, componentTestsH.NewBASCaptureExecutor(), compileAsset),
		depsH.ModuleFromService(depsSvc, log.Default()),
		brandH.Module(brandInternal.NewClient()),
		designsH.Module(primaryDB),
		templatesH.Module(primaryDB),
		healthH.Module(healthDB.Primary(), "react-component-library-api", "1.0.0"),
		inventoryH.Module(log.Default(), scenariosRoot, inventoryH.AdoptionsServiceAdapter{Service: adoptionsSvc}, uimanifest.NewFSLoader(filepath.Dir(scenariosRoot))),
		previewH.ModuleFromService(previewSvc, componentsSvc, log.Default(), filepath.Dir(scenariosRoot)),
		sketchH.ModuleWithCapture(filepath.Dir(scenariosRoot), log.Default(), sketchH.NewInventoryScannerAdapter(filepath.Dir(scenariosRoot), log.Default(), inventoryH.AdoptionsServiceAdapter{Service: adoptionsSvc}), fileRoots, availabilityFactory, componentsSvc, previewH.NewConnectHandler(previewH.Deps{Service: previewSvc, RepoRoot: filepath.Dir(scenariosRoot)}), sketchH.CaptureConfig{InferenceFor: func(ctx context.Context) (inferenceInternal.Repository, error) {
			pool, err := db.PoolForContext(ctx)
			if err != nil {
				return nil, err
			}
			return inferenceInternal.NewSQLiteRepository(pool), nil
		}, CritiquesFor: func(ctx context.Context) (critiqueInternal.Repository, error) {
			pool, err := db.PoolForContext(ctx)
			if err != nil {
				return nil, err
			}
			return critiqueInternal.NewSQLiteRepository(pool), nil
		}, RepositoryFor: func(ctx context.Context) (captureInternal.Repository, error) {
			pool, err := db.PoolForContext(ctx)
			if err != nil {
				return nil, err
			}
			return captureInternal.NewSQLiteRepository(pool), nil
		}, Dispatcher: captureInternal.BASDispatcher{BASBaseURL: componenttestsInternal.NewBASCaptureExecutor().BASBaseURL, RCLTargetBaseURL: selfBaseURL()}}),
		themesH.ModuleFromService(themesSvc, log.Default()),
		versionsH.ModuleWithLedger(primaryDB, schedule.System(), versionsResolver, log.Default(), versionLedger, componentsSvc),
		workflowsH.ModuleWithReadiness(primaryDB, schedule.System(), workflowsInternal.NewAgentManagerDispatcher(), workflowsInternal.NewPromotionReadinessReader(componentsSvc, adoptionsSvc), log.Default()),
	)

	rootMux := http.NewServeMux()
	devrouting.RegisterWithFileRoots(rootMux, db, fileRoots)
	rootMux.Handle("/", srv.Handler())
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// The catalog provider executes one browser-backed report per latest
		// asset. Keep the response open for the descriptor's 600s phase budget
		// plus a bounded transport margin so a complete aggregate result is not
		// truncated into an opaque unexpected EOF.
		WriteTimeout: 12 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			_ = healthDB.Close()
			_ = jobDB.Close()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// selfBaseURL is this API's own address. The browser loads a design capture's
// target from /design-captures/<id>/target.html, which only the API serves
// (the UI server does not proxy it). api-core's server listens on API_PORT.
func selfBaseURL() string {
	port := strings.TrimSpace(os.Getenv("API_PORT"))
	if port == "" {
		port = "8080"
	}
	return "http://127.0.0.1:" + port
}
