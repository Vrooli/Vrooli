// Package apphealth reads the deployed scenario's own health endpoint body.
//
// A 2xx status code proves the application answers; it does not prove the
// application can do its job. A scenario that cannot reach its mail provider,
// its payment provider or any other declared dependency still answers 200
// with "status": "degraded" and a failing dependency. Discarding that body is
// how a deployment whose customer sign-in is broken reports clean.
//
// This package never fails a deployment. A failing dependency is an operator
// warning: the deployment is running, one declared capability is not. The
// report carries an Unavailable reason when the body could not be read or
// parsed, which is distinct from "every dependency passes" and must never be
// rendered as a pass.
package apphealth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// maxBodyBytes bounds the health body this package will read. A scenario
// health document is small; anything larger is treated as the reachable
// prefix, because an operator warning is not worth unbounded memory.
const maxBodyBytes = 64 * 1024

// DependencyStatus is the outcome of one declared application dependency.
type DependencyStatus string

const (
	// DependencyPass means the application reported the dependency working.
	DependencyPass DependencyStatus = "pass"
	// DependencyWarn means the application reported the dependency failing.
	// It is a warning rather than a failure because the application still
	// serves; only the capability behind the dependency is lost.
	DependencyWarn DependencyStatus = "warn"
)

// Dependency is one entry of the application's own health report.
type Dependency struct {
	Name   string           `json:"name"`
	Status DependencyStatus `json:"status"`
	Detail string           `json:"detail,omitempty"`
}

// Report is what the deployed application said about itself.
type Report struct {
	// Status is the application's own aggregate word for its state, verbatim
	// (for example "healthy" or "degraded"). It is advisory: Dependencies is
	// the actionable evidence.
	Status string `json:"status,omitempty"`
	// Dependencies is sorted by name so repeated observations of the same
	// state produce identical output.
	Dependencies []Dependency `json:"dependencies,omitempty"`
	// URL is the endpoint this report came from.
	URL string `json:"url,omitempty"`
	// FetchedAt is when the body was read.
	FetchedAt time.Time `json:"fetched_at,omitempty"`
	// Unavailable is true when the body could not be read or understood.
	// Callers must render this as "not observed", never as healthy.
	Unavailable bool `json:"unavailable,omitempty"`
	// Reason explains Unavailable in operator-readable terms.
	Reason string `json:"reason,omitempty"`
}

// Warnings returns the failing dependencies, sorted by name.
func (r Report) Warnings() []Dependency {
	var out []Dependency
	for _, dep := range r.Dependencies {
		if dep.Status == DependencyWarn {
			out = append(out, dep)
		}
	}
	return out
}

// Summary is a one-line operator-facing description of the report.
func (r Report) Summary() string {
	switch {
	case r.Unavailable:
		reason := strings.TrimSpace(r.Reason)
		if reason == "" {
			reason = "not observed"
		}
		return "application dependencies not observed: " + reason
	case len(r.Dependencies) == 0:
		return "the application reports no dependencies"
	}
	warnings := r.Warnings()
	if len(warnings) == 0 {
		return fmt.Sprintf("%d of %d application dependencies pass", len(r.Dependencies), len(r.Dependencies))
	}
	names := make([]string, 0, len(warnings))
	for _, dep := range warnings {
		names = append(names, dep.Name)
	}
	return fmt.Sprintf("%d of %d application dependencies failing: %s",
		len(warnings), len(r.Dependencies), strings.Join(names, ", "))
}

// unavailable builds a report that records why nothing could be observed.
func unavailable(url, reason string, at time.Time) Report {
	return Report{URL: url, FetchedAt: at.UTC(), Unavailable: true, Reason: reason}
}

// healthDocument is the scenario health contract this package understands.
// Both shapes are accepted: the dependency map every Vrooli scenario API
// serves at /health, and a flat check list, which readiness reports use.
type healthDocument struct {
	Status       string                      `json:"status"`
	Dependencies map[string]healthDependency `json:"dependencies"`
	Checks       []healthCheck               `json:"checks"`
}

type healthDependency struct {
	Connected *bool  `json:"connected"`
	Status    string `json:"status"`
	Error     string `json:"error"`
	Detail    string `json:"detail"`
}

type healthCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Error  string `json:"error"`
}

// Parse reads a scenario health body. An unparseable body is Unavailable, not
// a pass: an operator must be able to tell "nothing was observed" from
// "everything is fine".
func Parse(url string, body []byte, at time.Time) Report {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return unavailable(url, "the health endpoint returned an empty body", at)
	}
	var doc healthDocument
	if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
		return unavailable(url, "the health body is not JSON this producer understands", at)
	}
	report := Report{
		Status:    strings.TrimSpace(doc.Status),
		URL:       url,
		FetchedAt: at.UTC(),
	}
	for name, dep := range doc.Dependencies {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		report.Dependencies = append(report.Dependencies, Dependency{
			Name:   name,
			Status: dependencyStatus(dep.Connected, dep.Status),
			Detail: firstNonEmpty(dep.Error, dep.Detail),
		})
	}
	for _, check := range doc.Checks {
		name := strings.TrimSpace(check.Name)
		if name == "" {
			continue
		}
		report.Dependencies = append(report.Dependencies, Dependency{
			Name:   name,
			Status: dependencyStatus(nil, check.Status),
			Detail: firstNonEmpty(check.Error, check.Detail),
		})
	}
	sort.Slice(report.Dependencies, func(i, j int) bool {
		return report.Dependencies[i].Name < report.Dependencies[j].Name
	})
	if len(report.Dependencies) == 0 && report.Status == "" {
		return unavailable(url, "the health body declares neither a status nor any dependency", at)
	}
	return report
}

// dependencyStatus maps a scenario's own vocabulary onto pass or warn. An
// unrecognized word is a warning rather than a pass, so a scenario cannot
// earn a clean report by reporting something this producer cannot read.
func dependencyStatus(connected *bool, status string) DependencyStatus {
	if connected != nil && !*connected {
		return DependencyWarn
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "pass", "passed", "ok", "healthy", "up", "connected", "true":
		if connected != nil && !*connected {
			return DependencyWarn
		}
		return DependencyPass
	default:
		return DependencyWarn
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// Fetch reads one health endpoint. Transport and status-code problems are
// Unavailable, never a pass: the reachability verdict belongs to the
// readiness and transport checks, which observe it directly.
func Fetch(ctx context.Context, client *http.Client, url string, timeout time.Duration) Report {
	now := time.Now().UTC()
	url = strings.TrimSpace(url)
	if url == "" {
		return unavailable(url, "no application health URL is known for this deployment", now)
	}
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return unavailable(url, "the application health URL could not be requested", now)
	}
	resp, err := client.Do(req)
	if err != nil {
		return unavailable(url, "the application health endpoint could not be reached", now)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if readErr != nil {
		return unavailable(url, "the application health body could not be read", now)
	}
	report := Parse(url, body, now)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		// A non-2xx health response is the readiness check's verdict to give.
		// Keep whatever dependencies the body disclosed, but never let this
		// path look like a successful observation.
		if report.Unavailable {
			return unavailable(url, fmt.Sprintf("the application health endpoint returned %s", resp.Status), now)
		}
		report.Reason = fmt.Sprintf("the application health endpoint returned %s", resp.Status)
	}
	return report
}
