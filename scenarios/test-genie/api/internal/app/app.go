package app

import (
	"test-genie/internal/app/httpserver"
	"test-genie/internal/app/runtime"
)

// Server exposes the HTTP transport to callers without leaking the transport package details.
type Server = httpserver.Server

var (
	loadConfig        = runtime.LoadConfig
	buildDependencies = runtime.BuildDependencies
	newHTTPServer     = httpserver.New
)

// NewServer wires runtime configuration, dependencies, and HTTP transport seams.
func NewServer() (*httpserver.Server, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}

	deps, err := buildDependencies(cfg)
	if err != nil {
		return nil, err
	}

	httpCfg := httpserver.Config{
		Port:        cfg.Port,
		ServiceName: "Test Genie API",
	}
	httpDeps := httpserver.Dependencies{
		DB:                         deps.DB,
		SuiteQueue:                 deps.SuiteRequests,
		Executions:                 deps.ExecutionHistory,
		ExecutionSvc:               deps.ExecutionService,
		ExecutionPlanner:           deps.ExecutionPlanner,
		Scenarios:                  deps.ScenarioService,
		PhaseCatalog:               deps.PhaseCatalog,
		AgentService:               deps.AgentService,
		FixService:                 deps.FixService,
		RequirementsImproveService: deps.RequirementsImproveService,
		RequirementsSyncer:         deps.RequirementsSyncer,
		ToolRegistry:               deps.ToolRegistry,
		ToolHandler:                deps.ToolHandler,
	}

	return newHTTPServer(httpCfg, httpDeps)
}
