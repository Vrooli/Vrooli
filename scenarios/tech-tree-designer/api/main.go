package main

import (
	"context"
	"log"
	"net/http"

	"tech-tree-designer/internal/modules"
	"tech-tree-designer/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	graphH "tech-tree-designer/handlers/graph"
	healthH "tech-tree-designer/handlers/health"
	ontologyH "tech-tree-designer/handlers/ontology"
	planningH "tech-tree-designer/handlers/planning"
	graphdomain "tech-tree-designer/internal/graph"
	ontologydomain "tech-tree-designer/internal/ontology"
	planningdomain "tech-tree-designer/internal/planning"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tech-tree-designer/v1/graph"
)

type graphScenarioSource struct {
	graph *graphdomain.Service
}

func (s graphScenarioSource) ScenarioGraph(ctx context.Context) (*graphv1.TechTreeGraph, error) {
	return s.graph.Describe(ctx, graphdomain.SourceRequest{})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "tech-tree-designer"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "tech-tree-designer",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	planningService := planningdomain.NewService(
		planningdomain.NewSQLiteRepository(db),
		planningdomain.NewCompilerValidator(""),
		planningdomain.NewFilesystemMaterializer(""),
	)
	graphService := graphdomain.NewServiceWithPlanned(
		graphdomain.NewSDASource(graphdomain.NewSDAClient(nil, nil)),
		planningService,
	)
	ontologyService := ontologydomain.NewServiceWithScenarioSource(
		ontologydomain.NewSQLiteRepository(db),
		graphScenarioSource{graph: graphService},
	)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "tech-tree-designer-api", "1.0.0"),
		graphH.Module(graphService),
		ontologyH.Module(ontologyService),
		planningH.Module(planningService),
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
		Cleanup: func(ctx context.Context) error { return db.Close() },
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
