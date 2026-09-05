package main

import (
	"context"
	"log"
	"net/http"

	"portal/internal/modules"
	internalsearch "portal/internal/search"
	"portal/internal/server"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	chatH "portal/handlers/chat"
	healthH "portal/handlers/health"
	integrationsH "portal/handlers/integrations"
	messageH "portal/handlers/message"
	searchH "portal/handlers/search"
	internalchat "portal/internal/chat"
)

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "portal"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "portal",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db.Primary(), modules.AllSchemas()...); err != nil {
		log.Fatalf("schema initialization failed: %v", err)
	}

	clk := schedule.System()
	integrationRegistry := integrationsH.NewRegistry(db, clk)
	chatRepo := internalchat.NewSQLiteRepository(db, clk)
	chatService := internalchat.NewService(chatRepo)
	searchService := internalsearch.NewService(internalsearch.Config{
		Chat:     chatService,
		Registry: integrationRegistry,
		Clock:    clk,
	})
	srv := server.New(
		server.Deps{Clock: clk, Logger: log.Default()},
		chatH.Module(db, clk),
		healthH.Module(db, "portal-api", "1.0.0"),
		integrationsH.Module(integrationRegistry),
		messageH.Module(db, clk, searchService),
		searchH.Module(searchService),
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
