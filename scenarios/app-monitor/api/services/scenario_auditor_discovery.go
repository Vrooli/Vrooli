package services

import (
	"context"
	"fmt"

	"app-monitor-api/logger"

	"github.com/vrooli/api-core/discovery"
)

func (s *AppService) locateScenarioAuditorAPIPort(ctx context.Context) (int, error) {
	if port, err := discovery.ResolveScenarioPort(ctx, "scenario-auditor", "API_PORT"); err == nil && port > 0 {
		return port, nil
	} else if err != nil {
		logger.Warn("failed to resolve scenario-auditor port via CLI", err)
	}

	apps, err := s.GetAppsFromOrchestrator(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect scenarios: %w", err)
	}

	for _, candidate := range apps {
		name := normalizeLower(candidate.ScenarioName)
		if name == "" {
			name = normalizeLower(candidate.ID)
		}
		if name != "scenario-auditor" {
			continue
		}

		port := resolvePort(candidate.PortMappings, []string{"api", "api_port", "API", "API_PORT"})
		if port > 0 {
			return port, nil
		}
	}

	return 0, fmt.Errorf("scenario-auditor is not running or API port not found (checked %d scenarios)", len(apps))
}
