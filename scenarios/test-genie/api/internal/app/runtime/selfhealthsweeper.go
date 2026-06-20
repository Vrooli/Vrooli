package runtime

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/selfhealth"
	"test-genie/internal/selfhealthsnapshots"
)

// selfHealthSweepRollup is the stable, timestamp-free content fingerprinted into
// a snapshot digest (and stored as payload_json). Identical content → identical
// digest → the sweeper skips the row, so idle periods don't accumulate.
type selfHealthSweepRollup struct {
	WindowDays     int                     `json:"window_days"`
	RunCount       int                     `json:"run_count"`
	Availability   float64                 `json:"availability"`
	HardViolations int                     `json:"hard_violations"`
	MetricsAdopted int                     `json:"metrics_adopted"`
	ProvidersTotal int                     `json:"providers_total"`
	Phases         []selfHealthPhasePoint  `json:"phases"`
	Providers      []selfHealthProviderPnt `json:"providers"`
}

type selfHealthPhasePoint struct {
	Phase        string  `json:"phase"`
	Availability float64 `json:"availability"`
	FailureRate  float64 `json:"failure_rate"`
	Adopted      int     `json:"metrics_adopted"`
}

type selfHealthProviderPnt struct {
	Provider  string `json:"provider"`
	Reachable bool   `json:"reachable"`
	Adopted   bool   `json:"metrics_adopted"`
	Hard      bool   `json:"hard_violation"`
}

// newSelfHealthRollupBuilder returns the sweeper's RollupBuilder: it composes the
// compute-on-read reliability ledger with a live conformance scan into one
// timestamp-free rollup. Both are the same sources GetSelfHealth reads.
func newSelfHealthRollupBuilder(ledgerSource *execution.SuiteExecutionRepository, repoRoot string) selfhealthsnapshots.RollupBuilder {
	return func(ctx context.Context) (selfhealthsnapshots.Rollup, error) {
		ledger, err := selfhealth.NewBuilder(ledgerSource, selfhealth.DefaultWindow).Build(ctx, selfhealth.DefaultPhaseMeta())
		if err != nil {
			return selfhealthsnapshots.Rollup{}, err
		}
		report := selfhealth.ConformanceScanner{RepoRoot: repoRoot, Target: selfhealth.DefaultScanTarget}.Scan(ctx)

		payload := selfHealthSweepRollup{
			WindowDays:     ledger.WindowDays,
			RunCount:       ledger.RunCount,
			Availability:   ledger.Availability,
			ProvidersTotal: len(report.Providers),
		}
		for _, p := range ledger.Phases {
			payload.Phases = append(payload.Phases, selfHealthPhasePoint{
				Phase:        p.Phase,
				Availability: p.Availability,
				FailureRate:  p.FailureRate,
				Adopted:      p.MetricsAdopted,
			})
		}
		sort.Slice(payload.Phases, func(i, j int) bool { return payload.Phases[i].Phase < payload.Phases[j].Phase })

		for _, pr := range report.Providers {
			hard := pr.HasHardViolation()
			if hard {
				payload.HardViolations++
			}
			if pr.MetricsAdopted {
				payload.MetricsAdopted++
			}
			payload.Providers = append(payload.Providers, selfHealthProviderPnt{
				Provider:  pr.Provider,
				Reachable: pr.Reachable,
				Adopted:   pr.MetricsAdopted,
				Hard:      hard,
			})
		}
		sort.Slice(payload.Providers, func(i, j int) bool { return payload.Providers[i].Provider < payload.Providers[j].Provider })

		return selfhealthsnapshots.Rollup{
			WindowDays:     payload.WindowDays,
			RunCount:       payload.RunCount,
			Availability:   payload.Availability,
			HardViolations: payload.HardViolations,
			MetricsAdopted: payload.MetricsAdopted,
			ProvidersTotal: payload.ProvidersTotal,
			Payload:        payload,
		}, nil
	}
}

// startSelfHealthSweeper launches the background snapshot sweeper under a
// process-lifetime context unless disabled. It is env-tuned:
//   - TEST_GENIE_SELFHEALTH_SWEEP_DISABLED=true  → do not start
//   - TEST_GENIE_SELFHEALTH_SWEEP_INTERVAL=<dur> → tick interval (default 1h)
//   - TEST_GENIE_SELFHEALTH_SWEEP_START_JITTER=<dur> → initial delay (default 0)
//
// The sweeper is read-only-then-single-write and idempotent (digest dedup), so
// a background context is safe; it exits on process termination.
func startSelfHealthSweeper(repo selfhealthsnapshots.SnapshotRepository, build selfhealthsnapshots.RollupBuilder) {
	if isTruthy(os.Getenv("TEST_GENIE_SELFHEALTH_SWEEP_DISABLED")) {
		log.Printf("[test-genie] self-health snapshot sweeper disabled via env")
		return
	}
	interval := parseDurationEnv("TEST_GENIE_SELFHEALTH_SWEEP_INTERVAL", time.Hour)
	jitter := parseDurationEnv("TEST_GENIE_SELFHEALTH_SWEEP_START_JITTER", 0)
	sweeper, err := selfhealthsnapshots.NewSweeper(selfhealthsnapshots.SweeperConfig{
		Repository:    repo,
		Build:         build,
		Interval:      interval,
		InitialJitter: jitter,
	})
	if err != nil {
		log.Printf("[test-genie] self-health snapshot sweeper not started: %v", err)
		return
	}
	go sweeper.RunLoop(context.Background())
}

func isTruthy(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	default:
		return false
	}
}

func parseDurationEnv(name string, fallback time.Duration) time.Duration {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil && d >= 0 {
		return d
	}
	// Tolerate a bare integer count of seconds.
	if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	return fallback
}

// repoRootFromScenariosRoot derives the repository root (the parent of the
// scenarios directory) so the conformance scan can load each provider's spec.
func repoRootFromScenariosRoot(scenariosRoot string) string {
	return filepath.Dir(scenariosRoot)
}
