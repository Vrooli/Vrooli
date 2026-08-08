package app

import (
	"path/filepath"

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
		RepoRoot:            filepath.Dir(cfg.ScenariosRoot),
		DB:                  deps.DB,
		HealthDB:            deps.HealthDB,
		Executions:          deps.ExecutionHistory,
		ExecutionPlanner:    deps.ExecutionPlanner,
		RunManager:          deps.RunManager,
		Scenarios:           deps.ScenarioService,
		PhaseCatalog:        deps.PhaseCatalog,
		AgentService:        deps.AgentService,
		RemediationService:  deps.RemediationService,
		RemediationLauncher: deps.RemediationLauncher,
		RequirementsSyncer:  deps.RequirementsSyncer,
		PlaybooksClaims:     deps.PlaybooksClaims,
		EligibilityService:  deps.EligibilityService,
		RunsService:         deps.RunsService,
		ValidationService:   deps.ValidationService,
		StartBackground:     deps.StartBackground,
		SweepStatus:         deps.SweepStatus,
	}

	return newHTTPServer(httpCfg, httpDeps)
}
