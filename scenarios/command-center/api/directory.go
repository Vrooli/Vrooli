package main

import (
	"context"
	"os"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// resolveScenarioBaseURL returns a runtime address for a declared scenario.
// Explicit endpoint configuration is useful for isolated deployments; the
// normal path delegates to api-core/discovery so lifecycle-assigned ports are
// re-resolved after a restart instead of being captured at process startup.
func resolveScenarioBaseURL(scenario, baseURLEnv, portEnv string) func() string {
	resolver := discovery.DefaultResolver()
	return resolveScenarioBaseURLUsing(scenario, baseURLEnv, portEnv, resolver.ResolveScenarioURLDefault)
}

// resolveScenarioBaseURLUsing keeps the scenario resolver boundary injectable
// for offline owner tests while sharing the production env/discovery/port policy.
func resolveScenarioBaseURLUsing(scenario, baseURLEnv, portEnv string, resolve func(context.Context, string) (string, error)) func() string {
	return func() string {
		if v := os.Getenv(baseURLEnv); v != "" {
			return v
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		url, err := resolve(ctx, scenario)
		if err == nil {
			return url
		}
		if v := os.Getenv(portEnv); v != "" {
			return "http://localhost:" + v
		}
		return ""
	}
}
