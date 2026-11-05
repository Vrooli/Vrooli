// Package health owns SearXNG-specific probe logic that should not live in CLI
// wiring.
//
// The /healthz endpoint is liveness-only: it reports 200 while every
// search engine behind the instance is suspended or broken. This package adds
// the missing engine-level signal by running a canary query through
// /search?format=json (whose payload carries unresponsive_engines) and
// supplementing it with the unauthenticated /stats/errors introspection
// endpoint.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/vrooli/vrooli/internal/tuning"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// Classification buckets for engine-level health.
const (
	StatusHealthy  = "healthy"
	StatusDegraded = "degraded"
	StatusCritical = "critical"
)

// DefaultCanaryQuery is a deliberately generic query that every general engine
// should answer.
const DefaultCanaryQuery = "current world news"

// HTTPClient is the narrow HTTP contract used by the probe so tests can
// inject fakes.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// EngineIssue describes one engine the instance reported as unresponsive for
// the canary query.
type EngineIssue struct {
	Engine string `json:"engine"`
	Reason string `json:"reason"`
}

// Report summarizes engine-level health for one canary probe.
type Report struct {
	Status              string              `json:"status"`
	ResponsiveEngines   []string            `json:"responsive_engines"`
	UnresponsiveEngines []EngineIssue       `json:"unresponsive_engines"`
	ResultCount         int                 `json:"result_count"`
	ErrorStats          map[string][]string `json:"error_stats,omitempty"`
}

// Classify maps a responsive-engine count to a health bucket: two or more
// distinct engines answering means the metasearch layer still has redundancy,
// exactly one means every query rides a single backend, zero means search is
// effectively down even though the process is "healthy".
func Classify(responsiveEngines int) string {
	switch {
	case responsiveEngines >= 2:
		return StatusHealthy
	case responsiveEngines == 1:
		return StatusDegraded
	default:
		return StatusCritical
	}
}

// Probe runs the canary query and assembles an engine-level health report.
// The /stats/errors supplement is best-effort: failures there never fail the
// probe.
func Probe(ctx context.Context, client HTTPClient, baseURL, query string) (Report, error) {
	if client == nil {
		client = &http.Client{Timeout: tuning.PlatformSupportRequestTimeout()}
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return Report{}, fmt.Errorf("SearXNG base URL is required")
	}
	if strings.TrimSpace(query) == "" {
		query = DefaultCanaryQuery
	}

	report, err := probeSearch(ctx, client, baseURL, query)
	if err != nil {
		return Report{}, err
	}
	report.ErrorStats = probeErrorStats(ctx, client, baseURL)
	return report, nil
}

func probeSearch(ctx context.Context, client HTTPClient, baseURL, query string) (Report, error) {
	endpoint := baseURL + "/search?q=" + url.QueryEscape(query) + "&format=json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Report{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Report{}, fmt.Errorf("canary search request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Report{}, fmt.Errorf("canary search returned status %d", resp.StatusCode)
	}

	var payload struct {
		Results []struct {
			Engine  string   `json:"engine"`
			Engines []string `json:"engines"`
		} `json:"results"`
		// Entries are [engine, reason] pairs (a third "suspended" element may
		// be present on some versions).
		UnresponsiveEngines [][]json.RawMessage `json:"unresponsive_engines"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Report{}, fmt.Errorf("decode canary search response: %w", err)
	}

	responsive := map[string]struct{}{}
	for _, result := range payload.Results {
		if engine := strings.TrimSpace(result.Engine); engine != "" {
			responsive[engine] = struct{}{}
		}
		for _, engine := range result.Engines {
			if engine = strings.TrimSpace(engine); engine != "" {
				responsive[engine] = struct{}{}
			}
		}
	}

	report := Report{ResultCount: len(payload.Results)}
	for engine := range responsive {
		report.ResponsiveEngines = append(report.ResponsiveEngines, engine)
	}
	sort.Strings(report.ResponsiveEngines)

	for _, entry := range payload.UnresponsiveEngines {
		issue := EngineIssue{}
		if len(entry) > 0 {
			_ = json.Unmarshal(entry[0], &issue.Engine)
		}
		if len(entry) > 1 {
			_ = json.Unmarshal(entry[1], &issue.Reason)
		}
		if issue.Engine != "" {
			report.UnresponsiveEngines = append(report.UnresponsiveEngines, issue)
		}
	}
	sort.Slice(report.UnresponsiveEngines, func(i, j int) bool {
		return report.UnresponsiveEngines[i].Engine < report.UnresponsiveEngines[j].Engine
	})

	report.Status = Classify(len(report.ResponsiveEngines))
	return report, nil
}

// probeErrorStats reads /stats/errors and condenses it into engine → error
// classnames. The endpoint shape is a map keyed by engine name whose values
// are lists of error records; only exception_classname/log_message are kept.
func probeErrorStats(ctx context.Context, client HTTPClient, baseURL string) map[string][]string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/stats/errors", nil)
	if err != nil {
		return nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var payload map[string][]struct {
		ExceptionClassname string `json:"exception_classname"`
		LogMessage         string `json:"log_message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil
	}

	stats := map[string][]string{}
	for engine, records := range payload {
		seen := map[string]struct{}{}
		for _, record := range records {
			label := strings.TrimSpace(record.ExceptionClassname)
			if label == "" {
				label = strings.TrimSpace(record.LogMessage)
			}
			if label == "" {
				continue
			}
			if _, dup := seen[label]; dup {
				continue
			}
			seen[label] = struct{}{}
			stats[engine] = append(stats[engine], label)
		}
		sort.Strings(stats[engine])
	}
	if len(stats) == 0 {
		return nil
	}
	return stats
}
