package phases

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	workspacepkg "test-genie/internal/orchestrator/workspace"
)

type scenarioStatusReport struct {
	Diagnostics scenarioDiagnostics `json:"diagnostics"`
	Insights    scenarioInsights    `json:"insights"`
}

type scenarioDiagnostics struct {
	UISmoke      *uiSmokeDiagnostics               `json:"ui_smoke"`
	HealthChecks map[string]healthCheckDiagnostics `json:"health_checks"`
}

type uiSmokeDiagnostics struct {
	Status     string           `json:"status"`
	Message    string           `json:"message"`
	DurationMs float64          `json:"duration_ms"`
	Bundle     *uiBundleSummary `json:"bundle"`
}

type uiBundleSummary struct {
	Fresh  bool   `json:"fresh"`
	Reason string `json:"reason"`
}

type healthCheckDiagnostics struct {
	Status       string                                 `json:"status"`
	Available    bool                                   `json:"available"`
	Dependencies map[string]healthCheckDependencyStatus `json:"dependencies"`
}

type healthCheckDependencyStatus struct {
	Connected bool `json:"connected"`
}

type scenarioInsights struct {
	Resources resourceInsights `json:"resources"`
	Stack     stackInsights    `json:"stack"`
}

type resourceInsights struct {
	Items []resourceTelemetry `json:"items"`
}

type resourceTelemetry struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Running  bool   `json:"running"`
	Healthy  bool   `json:"healthy"`
	Enabled  bool   `json:"enabled"`
}

type stackInsights struct {
	Detection stackDetection `json:"detection"`
}

type stackDetection struct {
	API componentDetection `json:"api"`
	UI  componentDetection `json:"ui"`
}

type componentDetection struct {
	Language string `json:"language"`
	Path     string `json:"path"`
}

func fetchScenarioStatus(ctx context.Context, env workspacepkg.Environment, logWriter io.Writer) (*scenarioStatusReport, error) {
	if err := EnsureCommandAvailable("vrooli"); err != nil {
		return nil, fmt.Errorf("vrooli CLI unavailable: %w", err)
	}
	appRoot := env.AppRoot
	if strings.TrimSpace(appRoot) == "" {
		appRoot = workspacepkg.AppRootFromScenario(env.ScenarioDir)
	}
	logPhaseStep(logWriter, "collecting scenario status via 'vrooli scenario status %s --json'", env.ScenarioName)
	output, err := phaseCommandCapture(ctx, appRoot, nil, "vrooli", "scenario", "status", env.ScenarioName, "--json")
	if err != nil {
		return nil, fmt.Errorf("vrooli scenario status failed: %w", err)
	}
	var report scenarioStatusReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		return nil, fmt.Errorf("failed to parse scenario status JSON: %w", err)
	}
	return &report, nil
}
