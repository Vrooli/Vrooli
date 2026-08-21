// Package health provides the /health endpoint.
//
// Built on api-core/health for the standardized response schema
// (status / dependencies / metrics) but plumbed through the local
// database.Pinger seam so handler tests can substitute a fake without
// opening the on-disk SQLite file.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code-facts/internal/database"
)

// Deps wires the seams the health handler needs. Service and Version
// are reported in the response envelope; Pinger backs the "database"
// dependency check.
type Deps struct {
	Pinger       database.Pinger
	Service      string
	Version      string
	CacheMetrics func(context.Context) (map[string]any, error)
}

// NewHandler returns a handler that reports overall health, service
// metadata, and the connectivity of the database dependency. The check
// is registered as Critical: a failed ping flips the response to
// status="unhealthy" with HTTP 503.
func NewHandler(d Deps) http.HandlerFunc {
	startedAt := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		resp := healthResponse{
			Status:        "healthy",
			Service:       d.Service,
			Timestamp:     now.UTC().Format(time.RFC3339),
			Readiness:     true,
			Version:       d.Version,
			UptimeSeconds: now.Sub(startedAt).Seconds(),
			Dependencies:  map[string]dependencyStatus{},
			Metrics:       runtimeMetrics(startedAt, now),
		}
		start := time.Now()
		err := d.Pinger.PingContext(r.Context())
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		dbStatus := dependencyStatus{Connected: err == nil, LatencyMS: &latencyMs}
		if err != nil {
			dbStatus.Error = err.Error()
			resp.Status = "unhealthy"
			resp.Readiness = false
		}
		resp.Dependencies["database"] = dbStatus
		if d.CacheMetrics != nil {
			if metrics, err := d.CacheMetrics(r.Context()); err == nil {
				for key, value := range metrics {
					resp.Metrics[key] = value
				}
				if value, ok := metrics["last_indexed_at"].(string); ok {
					resp.LastIndexedAt = value
				}
				if value, ok := metrics["indexed_count"].(int64); ok {
					resp.IndexedCount = value
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if !resp.Readiness {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type healthResponse struct {
	Status        string                      `json:"status"`
	Service       string                      `json:"service"`
	Timestamp     string                      `json:"timestamp"`
	Readiness     bool                        `json:"readiness"`
	Version       string                      `json:"version,omitempty"`
	UptimeSeconds float64                     `json:"uptime_seconds,omitempty"`
	Dependencies  map[string]dependencyStatus `json:"dependencies,omitempty"`
	Metrics       map[string]any              `json:"metrics,omitempty"`
	LastIndexedAt string                      `json:"last_indexed_at,omitempty"`
	IndexedCount  int64                       `json:"indexed_count,omitempty"`
}

type dependencyStatus struct {
	Connected bool     `json:"connected"`
	LatencyMS *float64 `json:"latency_ms,omitempty"`
	Error     string   `json:"error,omitempty"`
	Database  string   `json:"database,omitempty"`
}

func runtimeMetrics(startedAt, now time.Time) map[string]any {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	metrics := map[string]any{
		"goroutines":     runtime.NumGoroutine(),
		"heap_mb":        float64(mem.HeapAlloc) / 1024 / 1024,
		"uptime_seconds": now.Sub(startedAt).Seconds(),
	}
	if payload, err := os.ReadFile("/proc/self/statm"); err == nil { // #nosec G304 -- fixed Linux process accounting path.
		fields := strings.Fields(string(payload))
		if len(fields) > 1 {
			if pages, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
				metrics["rss_mb"] = float64(pages*int64(os.Getpagesize())) / 1024 / 1024
			}
		}
	}
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err == nil {
		metrics["cpu_seconds"] = timevalSeconds(usage.Utime) + timevalSeconds(usage.Stime)
		// Linux reports ru_maxrss in KiB.
		metrics["rss_high_water_mb"] = float64(usage.Maxrss) / 1024
	}
	return metrics
}

func timevalSeconds(value syscall.Timeval) float64 {
	return float64(value.Sec) + float64(value.Usec)/1_000_000
}
