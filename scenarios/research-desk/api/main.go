package main

import (
	"log"
	"os"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/schedule"
	apiserver "github.com/vrooli/api-core/server"

	"research-desk/handlers/activityapi"
	"research-desk/handlers/capabilitiesapi"
	"research-desk/handlers/healthapi"
	"research-desk/handlers/launchapi"
	"research-desk/handlers/operatorapi"
	"research-desk/internal/server"
)

func main() {
	// Preflight runs first so the binary can re-exec itself after a
	// stale-source rebuild before any listener opens.
	if preflight.Run(preflight.Config{ScenarioName: "research-desk"}) {
		return
	}

	w := newWiring(os.Getenv)
	srv := server.New(
		server.Deps{Clock: schedule.System(), Logger: log.Default()},
		healthapi.Module(healthapi.Deps{ResearchHome: w.researchHomeReadable, WebConsoleAccess: w.webConsoleAccess, Service: "research-desk-api", Version: w.version}),
		capabilitiesapi.Module(w.registry),
		activityapi.Module(activityapi.Deps{Reader: w.reader, Looks: w.writer, Routes: w.routes, Logger: log.Default()}),
		operatorapi.Module(operatorapi.Deps{ResearchDir: w.homeConfig.ResearchDir, Reader: w.reader, Writer: w.writer, Courier: w.courier, Clock: schedule.System()}),
		launchapi.Module(launchapi.Deps{ResearchDir: w.homeConfig.ResearchDir, Service: w.launch, Automatic: w.automaticLaunch, Logger: log.Default()}),
	)

	stop := w.start()

	authConfig, err := authn.FromEnvironment(os.Getenv)
	if err != nil {
		log.Fatalf("authentication configuration failed: %v", err)
	}
	var sharedAuth *authn.Config
	if authConfig.Enabled() {
		sharedAuth = &authConfig
	}
	if err := apiserver.Run(apiserver.Config{Handler: srv.Handler(), Authentication: sharedAuth, Cleanup: stop}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
