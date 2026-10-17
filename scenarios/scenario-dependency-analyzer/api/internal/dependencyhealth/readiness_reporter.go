package dependencyhealth

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

const dependencyReadinessBinding = "scenario-dependency-analyzer.deps.readiness"

type ReadinessReporter interface {
	Report(context.Context, string, *scenariovalidationv1.ValidateScenarioResponse, time.Time) error
}

func newReadinessReporter(ctx context.Context) ReadinessReporter {
	if strings.TrimSpace(os.Getenv("SCENARIO_DEPENDENCY_ANALYZER_DEPLOYMENT_MANAGER_READINESS_TOKEN")) == "" {
		return nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("SCENARIO_DEPENDENCY_ANALYZER_DEPLOYMENT_MANAGER_READINESS_URL")), "/")
	if baseURL == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
		if err != nil {
			return &configurationError{err: err}
		}
		baseURL = strings.TrimRight(strings.TrimSpace(resolved), "/")
	}
	reporter, err := readinessreporter.NewFromEnv(
		"SCENARIO_DEPENDENCY_ANALYZER_DEPLOYMENT_MANAGER_READINESS",
		baseURL,
		"dependencies-governed",
		dependencyReadinessBinding,
		"scenario-dependency-analyzer",
	)
	if err != nil {
		return &configurationError{err: err}
	}
	return reporter
}

type configurationError struct{ err error }

func (e *configurationError) Report(context.Context, string, *scenariovalidationv1.ValidateScenarioResponse, time.Time) error {
	if e == nil || e.err == nil {
		return nil
	}
	return e.err
}
