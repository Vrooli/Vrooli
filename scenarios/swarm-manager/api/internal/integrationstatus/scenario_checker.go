package integrationstatus

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type URLResolver func(context.Context, string) (string, error)

// ScenarioChecker adapts a scenario's standard health endpoint into the
// provider's narrow seam. It never decides transition policy.
type ScenarioChecker struct {
	Scenario         string
	Required         bool
	DegradedBehavior string
	ResolveURL       URLResolver
	HTTPClient       *http.Client
	Now              func() time.Time
	FreshFor         time.Duration
}

func (c ScenarioChecker) Check(ctx context.Context) (Status, error) {
	now := c.now()
	status := Status{Required: c.Required, CheckedAt: now, DegradedBehavior: c.DegradedBehavior}
	baseURL, err := c.ResolveURL(ctx, c.Scenario)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		status.Availability = Unconfigured
		status.Diagnostic = "scenario URL is not configured"
		return status, nil
	}
	status.Configured = true
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
	if err != nil {
		return Status{}, fmt.Errorf("build health request: %w", err)
	}
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		status.Availability = Unavailable
		status.Diagnostic = "health endpoint is unreachable"
		return status, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		status.Availability = Unavailable
		status.Diagnostic = fmt.Sprintf("health endpoint returned HTTP %d", resp.StatusCode)
		return status, nil
	}
	status.Availability = Available
	if c.FreshFor > 0 {
		status.FreshUntil = now.Add(c.FreshFor)
	}
	return status, nil
}

func (c ScenarioChecker) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}
