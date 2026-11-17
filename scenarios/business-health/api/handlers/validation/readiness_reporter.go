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

const businessHealthReadinessPrefix = "BUSINESS_HEALTH_DEPLOYMENT_MANAGER_READINESS"

type ReadinessReporter interface {
	Report(context.Context, string, *scenariovalidationv1.ValidateScenarioResponse, time.Time) error
}

func newReadinessReporter(ctx context.Context) ReadinessReporter {
	if strings.TrimSpace(os.Getenv(businessHealthReadinessPrefix+"_TOKEN")) == "" {
		return nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv(businessHealthReadinessPrefix+"_URL")), "/")
	if baseURL != "" {
		return newReadinessReporterFor(baseURL)
	}
	return newReadinessReporterWithResolver()
}

func newReadinessReporterFor(baseURL string) ReadinessReporter {
	reporter, err := readinessreporter.NewFromEnv(baseURLPrefix(), baseURL, "requirements-cover-sale", "business-health.requirements.readiness", "business-health")
	if err != nil {
		return readinessConfigurationError{err: err}
	}
	return reporter
}

func newReadinessReporterWithResolver() ReadinessReporter {
	reporter, err := readinessreporter.NewFromEnvResolver(baseURLPrefix(), func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "deployment-manager")
	}, "requirements-cover-sale", "business-health.requirements.readiness", "business-health")
	if err != nil {
		return readinessConfigurationError{err: err}
	}
	return reporter
}

func baseURLPrefix() string { return businessHealthReadinessPrefix }

type readinessConfigurationError struct{ err error }

func (e readinessConfigurationError) Report(context.Context, string, *scenariovalidationv1.ValidateScenarioResponse, time.Time) error {
	if e.err == nil {
		return errors.New("business-health readiness reporter is unavailable")
	}
	return e.err
}
