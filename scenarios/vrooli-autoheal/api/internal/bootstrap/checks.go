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
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/checks/infra"
	"vrooli-autoheal/internal/checks/system"
	"vrooli-autoheal/internal/checks/vrooli"
	"vrooli-autoheal/internal/platform"
)

// Default targets for infrastructure checks - defined here in bootstrap
// so check implementations stay pure and don't embed operational defaults.
const (
	DefaultNetworkTarget = "8.8.8.8:53" // Google DNS for connectivity check
	DefaultDNSDomain     = "google.com" // Reliable domain for DNS resolution check
)

// RegisterDefaultChecks adds all standard health checks to the registry.
// This centralizes check registration, keeping main.go focused on server setup.
// Platform capabilities are passed to checks that need them for runtime decisions.
func RegisterDefaultChecks(registry *checks.Registry, caps *platform.Capabilities) {
	// Infrastructure checks - explicit targets make dependencies clear
	registry.Register(infra.NewNetworkCheck(DefaultNetworkTarget))
	registry.Register(infra.NewDNSCheck(DefaultDNSDomain, caps))
	registry.Register(infra.NewDockerCheck(caps))
	registry.Register(infra.NewCloudflaredCheck(caps))
	registry.Register(infra.NewRDPCheck(caps))
	registry.Register(infra.NewNTPCheck(caps))
	registry.Register(infra.NewResolvedCheck(caps))
	registry.Register(infra.NewCertificateCheck())
	registry.Register(infra.NewDisplayManagerCheck(caps))

	// Vrooli API check - monitors the central orchestration layer
	registry.Register(vrooli.NewAPICheck())

	// Vrooli resource checks - core infrastructure services
	registry.Register(vrooli.NewResourceCheck("postgres"))
	registry.Register(vrooli.NewResourceCheck("redis"))
	registry.Register(vrooli.NewResourceCheck("ollama"))
	registry.Register(vrooli.NewResourceCheck("qdrant"))
	registry.Register(vrooli.NewResourceCheck("searxng"))
	registry.Register(vrooli.NewResourceCheck("browserless"))

	// System checks - host-level health monitoring
	registry.Register(system.NewDiskCheck())
	registry.Register(system.NewInodeCheck())
	registry.Register(system.NewSwapCheck())
	registry.Register(system.NewZombieCheck())
	registry.Register(system.NewPortCheck())
	registry.Register(system.NewClaudeCacheCheck())
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
