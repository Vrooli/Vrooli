// Package bootstrap provides initialization helpers that wire up the application
// components during startup. This separates the "what gets registered" concern
// from the entry point.
//
// Responsibility: Orchestration layer - decides what checks get registered and
// with what configuration. Domain logic lives in the checks themselves.
package bootstrap

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/coverage"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

const (
	NetworkTargetEnv     = "AUTOHEAL_NETWORK_TARGET"
	DNSDomainEnv         = "AUTOHEAL_DNS_DOMAIN"
	ExternalDNSServerEnv = "AUTOHEAL_EXTERNAL_DNS_SERVER"
)

// Infrastructure targets belong to lifecycle configuration. Keeping them out
// of the binary allows deployments to choose a reachable resolver without
// rebuilding the health supervisor.
func configuredInfrastructureValue(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// RegisterDefaultChecks adds all standard health checks to the registry.
// This centralizes check registration, keeping main.go focused on server setup.
// Platform capabilities are passed to checks that need them for runtime decisions.
// Uses the default factory for check creation.
func RegisterDefaultChecks(registry *checks.Registry, caps *platform.Capabilities) {
	RegisterChecksWithFactory(registry, caps, NewDefaultCheckFactory())
}

// RegisterChecksFromConfig adds health checks using the user's monitoring configuration.
// This respects which scenarios and resources the user has configured for monitoring.
func RegisterChecksFromConfig(registry *checks.Registry, caps *platform.Capabilities, configMgr *userconfig.Manager, delivery coverage.DeliveryReader) (*SupervisionController, error) {
	factory := NewCheckFactoryFromConfigManager(configMgr)
	factory.deliveryReader = delivery
	// Scenario/resource membership is reconciled from the canonical supervision
	// set below. The persisted monitoring configuration is additive input to
	// that controller, not a second registration path.
	factory.criticalScenarios = nil
	factory.nonCriticalScenarios = nil
	factory.resources = nil
	RegisterChecksWithFactory(registry, caps, factory)
	controller := NewSupervisionController(registry, configMgr, NewSupervisionSource(checks.DefaultExecutor))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := controller.Refresh(ctx); err != nil {
		return nil, err
	}
	registry.Register(&supervisionSourceCheck{controller: controller})
	return controller, nil
}

// RegisterChecksWithFactory adds health checks to the registry using the provided factory.
// This enables testing check registration with mock factories.
// [REQ:TEST-SEAM-001]
func RegisterChecksWithFactory(registry *checks.Registry, caps *platform.Capabilities, factory CheckFactory) {
	started := time.Now()
	// Infrastructure checks
	sectionStarted := time.Now()
	for _, check := range factory.CreateInfrastructureChecks(caps) {
		registry.Register(check)
	}
	log.Printf("autoheal startup: infrastructure checks registered in %s", time.Since(sectionStarted))

	// System checks
	sectionStarted = time.Now()
	for _, check := range factory.CreateSystemChecks() {
		registry.Register(check)
	}
	log.Printf("autoheal startup: system checks registered in %s", time.Since(sectionStarted))

	// Vrooli checks (API, resources, scenarios, watchdog)
	sectionStarted = time.Now()
	for _, check := range factory.CreateVrooliChecks(caps) {
		registry.Register(check)
	}
	// These projections are themselves checks so a missing remediation path or
	// unreadable delivery source is visible in the same ordered-severity model.
	registry.Register(coverage.NewRemediationReachCheck(registry))
	delivery := coverage.UnavailableDeliveryReader
	if provider, ok := factory.(deliveryReaderProvider); ok && provider.DeliveryReader() != nil {
		delivery = provider.DeliveryReader()
	}
	registry.Register(coverage.NewDeliveryReachCheck(delivery))
	log.Printf("autoheal startup: Vrooli checks registered in %s (total=%s)", time.Since(sectionStarted), time.Since(started))
}

// ResultLoader is the interface for loading persisted results.
// Implemented by persistence.Store.
type ResultLoader interface {
	GetLatestResultPerCheck(ctx context.Context) ([]checks.Result, error)
}

// ResultSaver is the interface for persisting results.
// Implemented by persistence.Store.
type ResultSaver interface {
	SaveResult(ctx context.Context, result checks.Result) error
}

// PopulateRecentResults loads the latest results from the database into the registry.
// This pre-populates the in-memory state so the dashboard shows data immediately after restart.
func PopulateRecentResults(ctx context.Context, registry *checks.Registry, loader ResultLoader) error {
	results, err := loader.GetLatestResultPerCheck(ctx)
	if err != nil {
		return err
	}

	for _, result := range results {
		registry.SetResult(result)
	}

	log.Printf("pre-populated %d health check results from database", len(results))
	return nil
}

// ScheduleInitialTick runs the first health check tick asynchronously after a delay.
// This ensures fresh results are available shortly after startup without blocking server readiness.
func ScheduleInitialTick(registry *checks.Registry, saver ResultSaver, delay time.Duration) {
	go func() {
		time.Sleep(delay)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		log.Println("running initial health check tick...")
		results := registry.RunAll(ctx, true)

		// Persist results to database (fire and forget errors for startup)
		for _, result := range results {
			if err := saver.SaveResult(ctx, result); err != nil {
				log.Printf("warning: failed to save initial result for %s: %v", result.CheckID, err)
			}
		}

		log.Printf("initial tick completed: %d checks executed", len(results))
	}()
}

// deliveryReaderProvider is implemented by the factory that carries the
// notification-hub delivery projection; a fake factory need not.
type deliveryReaderProvider interface {
	DeliveryReader() coverage.DeliveryReader
}
