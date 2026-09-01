package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"switchboard/internal/capabilities"
	channelcore "switchboard/internal/channels"
	"switchboard/internal/channels/adapters"
	"switchboard/internal/dispatch"
	"switchboard/internal/egress"
	"switchboard/internal/gates"
	"switchboard/internal/ingress"
	"switchboard/internal/modules"
	"switchboard/internal/server"
	"switchboard/internal/threads"
	"switchboard/internal/trust"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/preflight"
	apiserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"

	capsH "switchboard/handlers/capabilities"
	channelsH "switchboard/handlers/channels"
	conformanceH "switchboard/handlers/conformance"
	healthH "switchboard/handlers/health"
)

// scenarioStorageRoots resolves all filesystem storage classes once at
// startup. Domain file stores receive the routed roots and select the
// request-appropriate class at their own storage seam.
func scenarioStorageRoots() (storage.Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return storage.Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace("switchboard")
	if err != nil {
		return storage.Paths{}, fmt.Errorf("resolve switchboard storage namespace: %w", err)
	}
	return resolver.Resolve(storage.Options{ScenarioID: scenarioID})
}

func main() {
	// Preflight checks must run first so the binary can re-exec itself
	// after a stale-source rebuild before any listeners are opened.
	if preflight.Run(preflight.Config{ScenarioName: "switchboard"}) {
		return
	}

	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		Scenario:     "switchboard",
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
	channelRegistry, err := channelcore.Load("../data/channels", adapters.NewAll()...)
	if err != nil {
		log.Fatalf("channel descriptor registry: %v", err)
	}
	agentManagerURL, _ := discovery.ResolveScenarioURLDefault(context.Background(), "agent-manager")
	authVerifier := owneridentity.NewClient(owneridentity.Config{Resolver: discovery.NewResolver(discovery.ResolverConfig{})})
	hostFacts := channelcore.HostFacts{
		"telegram_bot_token": strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")) != "",
		"slack_app":          strings.TrimSpace(os.Getenv("SLACK_APP_TOKEN")) != "" && strings.TrimSpace(os.Getenv("SLACK_BOT_TOKEN")) != "",
		"mac_node":           strings.EqualFold(strings.TrimSpace(os.Getenv("VROOLI_MAC_NODE_AVAILABLE")), "true"),
	}
	router := &egress.Router{Registry: channelRegistry}
	threadStore := threads.NewStore(db.Primary())
	gateStore := gates.NewStore(db.Primary(), schedule.System().Now)
	processor := &dispatch.Processor{Ingress: ingress.New(), Threads: threadStore, Runner: dispatch.AgentManagerRunner{BaseURL: agentManagerURL, Threads: threadStore, Send: router.Send}, Send: router.Send, Grant: trust.Grant{Scopes: []string{"read"}}}
	channelDeps := channelsH.ModuleDeps{Registry: channelRegistry, Facts: hostFacts, DB: db.Primary(), Egress: router, Processor: processor, Gates: gateStore, Identity: authVerifier}
	stopChannels := channelsH.Start(context.Background(), channelDeps, log.Printf)

	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthH.Module(db, "switchboard-api", "1.0.0"),
		capsH.Module(capabilities.NewRegistry()),
		channelsH.Module(channelDeps),
		conformanceH.Module(channelRegistry),
	)

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
			stopChannels()
			return db.Close()
		},
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
