package app

import (
	"test-genie/internal/app/httpserver"
	"test-genie/internal/app/runtime"
)

// Server exposes the HTTP transport to callers without leaking the transport package details.
type Server = httpserver.Server

// NewServer wires runtime configuration, dependencies, and HTTP transport seams.
func NewServer() (*httpserver.Server, error) {
	cfg, err := runtime.LoadConfig()
	if err != nil {
		return nil, err
	}

	deps, err := runtime.BuildDependencies(cfg)
	if err != nil {
		return nil, err
	}

	httpCfg := httpserver.Config{
		Port:        cfg.Port,
		ServiceName: "Test Genie API",
	}
	httpDeps := httpserver.Dependencies{
		DB:           deps.DB,
		SuiteQueue:   deps.SuiteRequests,
		Executions:   deps.ExecutionRepo,
		ExecutionSvc: deps.ExecutionService,
		Scenarios:    deps.ScenarioService,
		PhaseCatalog: deps.PhaseCatalog,
	}

	return httpserver.New(httpCfg, httpDeps)
}
