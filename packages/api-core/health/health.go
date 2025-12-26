// Package health provides standardized health check endpoints for Vrooli scenarios.
//
// The package implements the Vrooli health response schema with automatic status
// computation, dependency checking, and runtime metrics collection.
//
// Minimal usage (reads SCENARIO_NAME from environment):
//
//	router.HandleFunc("/health", health.Handler())
//
// With database check:
//
//	router.HandleFunc("/health", health.Handler(health.DB(db)))
//
// With multiple dependencies:
//
//	router.HandleFunc("/health", health.Handler(
//	    health.DB(db),
//	    health.HTTP("ollama", "http://localhost:11434/api/tags"),
//	    health.HTTP("redis", "http://localhost:6379"),
//	))
//
// With explicit configuration:
//
//	router.HandleFunc("/health", health.New("my-scenario-api").
//	    Version("1.0.0").
//	    Check(health.DB(db), health.Critical).
//	    Check(health.HTTP("ollama", ollamaURL)).
//	    Handler())
package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/vrooli/api-core/scenario"
)

// Status values for health responses.
const (
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusUnhealthy = "unhealthy"
)

// Criticality determines how a check failure affects overall health status.
type Criticality int

const (
	// Optional means check failure results in degraded status (default).
	Optional Criticality = iota
	// Critical means check failure results in unhealthy status.
	Critical
)

// Response is the standardized health check response matching the Vrooli schema.
// This structure is validated by the CLI's health-validator.sh.
type Response struct {
	// Status is the overall health: healthy, degraded, or unhealthy.
	Status string `json:"status"`

	// Service identifies this service (e.g., "my-scenario-api").
	Service string `json:"service"`

	// Timestamp is when this check was performed (RFC3339 format).
	Timestamp string `json:"timestamp"`

	// Readiness indicates if the service can accept traffic.
	// True when status is healthy or degraded; false when unhealthy.
	Readiness bool `json:"readiness"`

	// Version is the service version (optional).
	Version string `json:"version,omitempty"`

	// Dependencies contains the status of downstream dependencies.
	Dependencies map[string]DependencyStatus `json:"dependencies,omitempty"`

	// Metrics contains runtime metrics (goroutines, heap, uptime).
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// DependencyStatus represents the health of a single dependency.
type DependencyStatus struct {
	// Connected indicates if the dependency is reachable.
	Connected bool `json:"connected"`

	// Latency is the check duration in milliseconds (optional).
	Latency *float64 `json:"latency_ms,omitempty"`

	// Error describes why the check failed (optional).
	Error string `json:"error,omitempty"`
}

// CheckResult is returned by a Checker after performing its health check.
type CheckResult struct {
	// Name identifies this dependency (e.g., "database", "redis").
	Name string

	// Connected indicates if the check passed.
	Connected bool

	// Latency is how long the check took.
	Latency time.Duration

	// Error is set if the check failed.
	Error error
}

// Checker performs a health check on a dependency.
type Checker interface {
	// Check performs the health check and returns the result.
	Check(ctx context.Context) CheckResult
}

// Builder constructs a health handler with configuration and checks.
type Builder struct {
	service   string
	version   string
	checks    []checkerEntry
	timeout   time.Duration
	startTime time.Time

	// For testing
	nowFunc func() time.Time
}

type checkerEntry struct {
	checker     Checker
	criticality Criticality
}

// Handler creates an http.HandlerFunc for the /health endpoint.
// It accepts optional Checker arguments for dependency checks.
// All checks are treated as Optional; use New().Check(c, Critical) for critical checks.
//
// Usage:
//
//	router.HandleFunc("/health", health.Handler())
//	router.HandleFunc("/health", health.Handler(health.DB(db)))
//	router.HandleFunc("/health", health.Handler(health.DB(db), health.HTTP("redis", url)))
func Handler(checks ...Checker) http.HandlerFunc {
	b := New()
	for _, c := range checks {
		b = b.Check(c, Optional)
	}
	return b.Handler()
}

// New creates a new health check builder.
// If no service name is provided, it's auto-detected from directory structure or SCENARIO_NAME env.
//
// Usage:
//
//	health.New()                    // auto-detect service name
//	health.New("custom-service")    // explicit service name
func New(service ...string) *Builder {
	svc := ""
	if len(service) > 0 {
		svc = service[0]
	}
	return &Builder{
		service:   svc,
		timeout:   5 * time.Second,
		startTime: time.Now(),
		nowFunc:   time.Now,
	}
}

// Version sets the service version string.
func (b *Builder) Version(v string) *Builder {
	b.version = v
	return b
}

// Check adds a health checker with the specified criticality.
// If c is nil, the check is skipped.
//
// Usage:
//
//	Check(health.DB(db), Critical)    // critical - failure = unhealthy
//	Check(health.DB(db), Optional)    // optional - failure = degraded
func (b *Builder) Check(c Checker, criticality Criticality) *Builder {
	if c == nil {
		return b
	}
	b.checks = append(b.checks, checkerEntry{
		checker:     c,
		criticality: criticality,
	})
	return b
}

// Timeout sets the maximum duration for all health checks.
// Default is 5 seconds.
func (b *Builder) Timeout(d time.Duration) *Builder {
	b.timeout = d
	return b
}

// Handler returns an http.HandlerFunc that serves health check responses.
func (b *Builder) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), b.timeout)
		defer cancel()

		resp := b.buildResponse(ctx)

		w.Header().Set("Content-Type", "application/json")
		if resp.Status == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// buildResponse constructs the health response by running all checks.
func (b *Builder) buildResponse(ctx context.Context) Response {
	now := b.nowFunc()

	resp := Response{
		Status:    StatusHealthy,
		Service:   b.resolveService(),
		Timestamp: now.UTC().Format(time.RFC3339),
		Readiness: true,
		Version:   b.version,
		Metrics:   b.collectMetrics(now),
	}

	if len(b.checks) > 0 {
		resp.Dependencies = b.runChecks(ctx, &resp)
	}

	return resp
}

// resolveService returns the service name, falling back to auto-detection.
func (b *Builder) resolveService() string {
	if b.service != "" {
		return b.service
	}
	return scenario.ServiceName()
}

// runChecks executes all registered health checks concurrently.
func (b *Builder) runChecks(ctx context.Context, resp *Response) map[string]DependencyStatus {
	deps := make(map[string]DependencyStatus)
	results := make(chan checkWithMeta, len(b.checks))

	var wg sync.WaitGroup
	for _, entry := range b.checks {
		wg.Add(1)
		go func(e checkerEntry) {
			defer wg.Done()
			result := e.checker.Check(ctx)
			results <- checkWithMeta{result: result, criticality: e.criticality}
		}(entry)
	}

	// Close results channel when all checks complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	hasCriticalFailure := false
	hasAnyFailure := false

	for meta := range results {
		result := meta.result
		status := DependencyStatus{
			Connected: result.Connected,
		}

		if result.Latency > 0 {
			latencyMs := float64(result.Latency.Microseconds()) / 1000.0
			status.Latency = &latencyMs
		}

		if result.Error != nil {
			status.Error = result.Error.Error()
		}

		// A check fails if it's not connected OR has an error
		if !result.Connected || result.Error != nil {
			hasAnyFailure = true
			if meta.criticality == Critical {
				hasCriticalFailure = true
			}
		}

		deps[result.Name] = status
	}

	// Compute overall status
	if hasCriticalFailure {
		resp.Status = StatusUnhealthy
		resp.Readiness = false
	} else if hasAnyFailure {
		resp.Status = StatusDegraded
	}

	return deps
}

type checkWithMeta struct {
	result      CheckResult
	criticality Criticality
}

// collectMetrics gathers runtime metrics.
func (b *Builder) collectMetrics(now time.Time) map[string]interface{} {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return map[string]interface{}{
		"goroutines":     runtime.NumGoroutine(),
		"heap_mb":        float64(mem.HeapAlloc) / 1024 / 1024,
		"uptime_seconds": now.Sub(b.startTime).Seconds(),
	}
}

// =============================================================================
// Built-in Checkers
// =============================================================================

// dbChecker checks database connectivity.
type dbChecker struct {
	db   *sql.DB
	name string
}

// DB creates a database health checker.
// The check performs a ping to verify connectivity.
// If db is nil, the dependency is reported as unhealthy ("not configured").
// Whether this makes the overall service unhealthy depends on the Criticality
// passed to Check().
func DB(db *sql.DB) Checker {
	return &dbChecker{db: db, name: "database"}
}

// DBNamed creates a database health checker with a custom name.
// Useful when checking multiple databases.
func DBNamed(name string, db *sql.DB) Checker {
	return &dbChecker{db: db, name: name}
}

func (c *dbChecker) Check(ctx context.Context) CheckResult {
	if c.db == nil {
		return CheckResult{
			Name:      c.name,
			Connected: false,
			Error:     errNotConfigured,
		}
	}
	start := time.Now()
	err := c.db.PingContext(ctx)
	return CheckResult{
		Name:      c.name,
		Connected: err == nil,
		Latency:   time.Since(start),
		Error:     err,
	}
}

var errNotConfigured = &notConfiguredError{}

type notConfiguredError struct{}

func (e *notConfiguredError) Error() string { return "not configured" }

// httpChecker checks HTTP endpoint availability.
type httpChecker struct {
	name   string
	url    string
	client *http.Client
}

// HTTP creates an HTTP endpoint health checker.
// The check performs a GET request and expects a 2xx response.
func HTTP(name, url string) Checker {
	return &httpChecker{
		name:   name,
		url:    url,
		client: &http.Client{Timeout: 3 * time.Second},
	}
}

func (c *httpChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return CheckResult{
			Name:      c.name,
			Connected: false,
			Latency:   time.Since(start),
			Error:     err,
		}
	}

	resp, err := c.client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Name:      c.name,
			Connected: false,
			Latency:   latency,
			Error:     err,
		}
	}
	defer resp.Body.Close()

	connected := resp.StatusCode >= 200 && resp.StatusCode < 300
	var checkErr error
	if !connected {
		checkErr = &httpError{statusCode: resp.StatusCode}
	}

	return CheckResult{
		Name:      c.name,
		Connected: connected,
		Latency:   latency,
		Error:     checkErr,
	}
}

type httpError struct {
	statusCode int
}

func (e *httpError) Error() string {
	return "unexpected status: " + http.StatusText(e.statusCode)
}
