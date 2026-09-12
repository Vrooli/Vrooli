package validation

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	readinessreporter "github.com/vrooli/vrooli/packages/proto/readinessreporter"
)

const securityHealthReadinessPrefix = "SECURITY_HEALTH_DEPLOYMENT_MANAGER_READINESS"

type ReadinessReporter interface {
	Report(context.Context, string, *scenariovalidationv1.ValidateScenarioResponse, time.Time) error
}

type readinessReporterGroup []ReadinessReporter

func (g readinessReporterGroup) Report(ctx context.Context, scenario string, response *scenariovalidationv1.ValidateScenarioResponse, observedAt time.Time) error {
	for _, reporter := range g {
		if reporter != nil {
			if err := reporter.Report(ctx, scenario, response, observedAt); err != nil {
				return err
			}
		}
	}
	return nil
}

func newReadinessReporter(ctx context.Context) ReadinessReporter {
	if strings.TrimSpace(os.Getenv(securityHealthReadinessPrefix+"_TOKEN")) == "" {
		return nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(securityHealthReadinessPrefix+"_URL")), "/")
	if baseURL == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
		if err != nil {
			return readinessConfigurationError{err: err}
		}
		baseURL = strings.TrimRight(strings.TrimSpace(resolved), "/")
	}
	return readinessReporterGroup{
		newReadinessReporterFor(baseURL, "security-secrets-clean", "security-health.validate.scenario"),
		newReadinessReporterFor(baseURL, "enforcement-paths-gate", "security-health.enforcement.readiness"),
	}
}

func newReadinessReporterFor(baseURL, criterion, binding string) ReadinessReporter {
	reporter, err := readinessreporter.NewFromEnv(securityHealthReadinessPrefix, baseURL, criterion, binding, "security-health")
	if err != nil {
		return readinessConfigurationError{err: err}
	}
	return reporter
}

type readinessConfigurationError struct{ err error }

func (e readinessConfigurationError) Report(context.Context, string, *scenariovalidationv1.ValidateScenarioResponse, time.Time) error {
	if e.err == nil {
		return errors.New("security-health readiness reporter is unavailable")
	}
	return e.err
}
