// Package graphsweeper keeps the unified graph store current with a default-ON,
// freshness-gated ingest loop that never overloads the upstream fact sources
// (proto-health, code-facts). It gates re-ingest on a per-scenario tree digest,
// caps concurrency, bounds each cycle by a wall-clock budget, single-flights
// cycles, and trips a circuit breaker on repeated upstream failure — retaining
// last-good edges rather than zeroing a scenario's contribution mid-outage.
package graphsweeper

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the sweeper's tunable control surface. Defaults are conservative and
// derived from the Phase 0 cold/warm fleet-build measurements (see
// docs/perf/2026-06-22-sda-graph-ingest.md).
type Config struct {
	Enabled          bool
	Interval         time.Duration
	StartJitter      time.Duration
	Concurrency      int
	CycleBudget      time.Duration
	BreakerThreshold int
	BreakerCooldown  time.Duration
	RepoRoot         string
	ScenariosRoot    string
}

// Defaults returns the conservative baseline configuration.
func Defaults() Config {
	return Config{
		Enabled:          true,
		Interval:         30 * time.Minute,
		StartJitter:      90 * time.Second,
		Concurrency:      3,
		CycleBudget:      10 * time.Minute,
		BreakerThreshold: 4,
		BreakerCooldown:  5 * time.Minute,
	}
}

// LoadConfig builds a Config from SDA_GRAPH_SWEEP_* env knobs over the defaults.
func LoadConfig(repoRoot, scenariosRoot string) Config {
	cfg := Defaults()
	cfg.RepoRoot = repoRoot
	cfg.ScenariosRoot = scenariosRoot

	cfg.Enabled = boolEnv("SDA_GRAPH_SWEEP_ENABLED", cfg.Enabled)
	cfg.Interval = durationEnv("SDA_GRAPH_SWEEP_INTERVAL", cfg.Interval)
	cfg.StartJitter = durationEnv("SDA_GRAPH_SWEEP_START_JITTER", cfg.StartJitter)
	cfg.Concurrency = intEnv("SDA_GRAPH_SWEEP_CONCURRENCY", cfg.Concurrency, 1, 32)
	cfg.CycleBudget = durationEnv("SDA_GRAPH_SWEEP_CYCLE_BUDGET", cfg.CycleBudget)
	cfg.BreakerThreshold = intEnv("SDA_GRAPH_SWEEP_BREAKER_THRESHOLD", cfg.BreakerThreshold, 1, 1000)
	cfg.BreakerCooldown = durationEnv("SDA_GRAPH_SWEEP_BREAKER_COOLDOWN", cfg.BreakerCooldown)
	return cfg
}

func boolEnv(name string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	switch raw {
	case "":
		return fallback
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func intEnv(name string, fallback, min, max int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return clampInt(fallback, min, max)
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return clampInt(fallback, min, max)
	}
	return clampInt(parsed, min, max)
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
