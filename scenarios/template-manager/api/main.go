package main

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/catalog"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/modules"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/monitor"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/server"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/templateengine"
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/validationrunner"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	searchregister "github.com/vrooli/searchregister-go"
	_ "modernc.org/sqlite"

	debtH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/debt"
	guidanceH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/guidance"
	healthH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/health"
	lifecycleH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/lifecycle"
	measuresH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/measures"
	monitorH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/monitor"
	registryH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/registry"
	resourceTemplateH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/resource_template"
	validationH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/validation"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "template-manager"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "template-manager",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	catalogRepo := catalog.NewSQLiteRepository(db)
	if err := syncScenarioTemplateRegistry(context.Background(), catalogRepo); err != nil {
		log.Fatalf("scenario template registry synchronization failed: %v", err)
	}
	// Keep Search Hub's durable registry as a mirror of this scenario's
	// search.json SSOT. Registration is deliberately best-effort: template
	// management remains healthy when the optional federation layer is down,
	// and the next lifecycle start retries the mirror.
	go searchregister.Register(context.Background(), searchregister.Config{
		ScenarioID:     "template-manager",
		SearchFilePath: filepath.Join("..", ".vrooli", "search.json"),
		Logger:         log.Default(),
	})
	validationRunner, err := validationrunner.NewEngineRunner("")
	if err != nil {
		log.Fatalf("template engine initialization failed: %v", err)
	}
	validationService := validationrunner.NewService(catalogRepo, validationRunner)
	monitorService := monitor.NewService(catalogRepo, validationService, monitor.ConfigFromEnv(), log.Default())
	monitorService.Start(context.Background())
	measuresModule, err := measuresH.Module(catalogRepo, time.Now)
	if err != nil {
		log.Fatalf("measures module init failed: %v", err)
	}
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "template-manager-api", "1.0.0"),
		registryH.Module(db, log.Default()),
		validationH.Module(db, log.Default()),
		debtH.Module(db, log.Default()),
		guidanceH.Module(log.Default()),
		lifecycleH.Module(log.Default(), validationRunner.Engine, validationService),
		measuresModule,
		monitorH.Module(monitorService),
		resourceTemplateH.Module(log.Default()),
		validationH.ScenarioValidationModule(catalogRepo, log.Default()),
	)

	// Top-level mux that mounts the API handler plus, when in development
	// mode, the dev-only RoutingService used by test-genie to install a
	// runtime test DB pool without restarting this scenario.
	rootMux := http.NewServeMux()
	devrouting.Register(rootMux, db)

	rootMux.Handle("/", srv.Handler())

	// apihttp.TestModeMiddleware reads X-Vrooli-Test-Mode: 1 and marks the
	// request context so *database.RoutedDB routes the call to the
	// installed test pool. Self-disables in production mode.
	handler := apihttp.TestModeMiddleware(rootMux)

	if err := apiserver.Run(apiserver.Config{
		Handler: handler,
		// Deep template validation starts a real scenario through Test Genie and
		// waits for its server-owned lifecycle result. The api-core default 30s
		// response deadline would sever the RPC before an actionable failure or
		// phase result can be returned.
		WriteTimeout: 20 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			monitorService.Stop()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func syncScenarioTemplateRegistry(ctx context.Context, repo catalog.Repository) error {
	engine, err := templateengine.New("")
	if err != nil {
		return err
	}
	infos, err := engine.ListTemplates(ctx)
	if err != nil {
		return err
	}
	templates := make([]catalog.ScenarioTemplate, 0, len(infos))
	for _, info := range infos {
		templates = append(templates, catalog.ScenarioTemplate{
			ID:           info.Name,
			Version:      info.Manifest.Version,
			ManifestPath: filepath.ToSlash(filepath.Join("templates", "scenarios", info.Name, "template.json")),
			SourcePath:   filepath.ToSlash(filepath.Join("templates", "scenarios", info.Name)),
			UpdatedAt:    time.Now().UTC(),
		})
	}
	return repo.SyncScenarioTemplates(ctx, templates)
}
