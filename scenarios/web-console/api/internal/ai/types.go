package ai

import "time"

// DefaultProviderTimeout is the fallback timeout for AI provider HTTP calls.
// CROSS-LANGUAGE COUPLING: SQL defaults also use 30s.
const DefaultProviderTimeout = 30 * time.Second

// Config holds the configuration for a single AI provider.
type Config struct {
	Name          string `json:"name"`
	Enabled       bool   `json:"enabled"`
	Priority      int    `json:"priority"`
	TimeoutSec    int    `json:"timeout_sec"`
	MaxRetries    int    `json:"max_retries"`
	KeyConfigured bool   `json:"key_configured,omitempty"`
	KeySource     string `json:"key_source,omitempty"`
}

// Health tracks runtime health for a single AI provider.
type Health struct {
	Name         string  `json:"name"`
	Available    bool    `json:"available"`
	LastCheck    string  `json:"last_check,omitempty"`
	LastLatency  string  `json:"last_latency,omitempty"`
	ErrorCount   int64   `json:"error_count"`
	SuccessCount int64   `json:"success_count"`
	ErrorRate    float64 `json:"error_rate"`
}

// healthTracker is the in-memory health record shared by mem and SQL stores.
type healthTracker struct {
	available    bool
	lastCheck    time.Time
	lastLatency  time.Duration
	errorCount   int64
	successCount int64
}

func (h *healthTracker) snapshot(name string) Health {
	total := h.successCount + h.errorCount
	var errorRate float64
	if total > 0 {
		errorRate = float64(h.errorCount) / float64(total)
	}
	out := Health{
		Name:         name,
		Available:    h.available,
		ErrorCount:   h.errorCount,
		SuccessCount: h.successCount,
		ErrorRate:    errorRate,
	}
	if !h.lastCheck.IsZero() {
		out.LastCheck = h.lastCheck.UTC().Format(time.RFC3339)
		out.LastLatency = h.lastLatency.Truncate(time.Millisecond).String()
	}
	return out
}
