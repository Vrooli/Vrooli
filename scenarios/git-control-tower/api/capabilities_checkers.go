package main

import (
	"context"
	"log"
	"net/http"

	"github.com/vrooli/api-core/discovery"
)

// ScenarioChecker checks whether a scenario is running by resolving its URL
// via the Vrooli CLI and pinging its /health endpoint.
type ScenarioChecker struct {
	Slug     string
	Client   *http.Client
	Resolver *discovery.Resolver
}

// Check probes the scenario's health endpoint.
func (c *ScenarioChecker) Check(ctx context.Context) (CapabilityStatus, string) {
	var (
		url string
		err error
	)
	if c.Resolver != nil {
		url, err = c.Resolver.ResolveScenarioURLDefault(ctx, c.Slug)
	} else {
		url, err = discovery.ResolveScenarioURLDefault(ctx, c.Slug)
	}
	if err != nil {
		if discovery.IsScenarioNotRunning(err) {
			return StatusUnavailable, c.Slug + " is not running"
		}
		log.Printf("WARNING: capability check for %s failed: %v", c.Slug, err)
		return StatusUnavailable, c.Slug + " is not running or could not be reached"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url+"/health", nil)
	if err != nil {
		log.Printf("WARNING: capability health check for %s failed: %v", c.Slug, err)
		return StatusUnavailable, c.Slug + " is not responding"
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return StatusUnavailable, c.Slug + " is not responding"
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StatusUnavailable, c.Slug + " is not responding"
	}

	return StatusAvailable, "running at " + url
}
