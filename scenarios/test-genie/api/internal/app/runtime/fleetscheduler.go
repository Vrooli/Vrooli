package runtime

import (
	"context"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/fleetscheduler"
	"test-genie/internal/runmanager"
)

// fleetRosterFromScenariosRoot returns a roster provider listing the first-level
// scenario directories under scenariosRoot (hidden entries skipped), so the
// fleet ledger can name scenarios that exist on disk but have no run in the
// window. It is best-effort: a read error yields an empty roster, and the ledger
// treats "no roster" as an honest unknown rather than asserting zero coverage.
func fleetRosterFromScenariosRoot(scenariosRoot string) func(ctx context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		entries, err := os.ReadDir(scenariosRoot)
		if err != nil {
			return nil, err
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := strings.TrimSpace(entry.Name())
			if name == "" || strings.HasPrefix(name, ".") {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		return names, nil
	}
}

// startFleetScheduler launches the priority-weighted background fleet scheduler
// under a process-lifetime context. It is DEFAULT-OFF (mirroring EM's
// DefaultImportanceAwareScheduling=false and the digest sweepers' disable envs):
// it starts ONLY when explicitly enabled, because each cycle launches real full
// suites across the fleet. Env tuning:
//   - TEST_GENIE_FLEET_SCHEDULER_ENABLED=true        → start (default: off)
//   - TEST_GENIE_FLEET_SCHEDULER_INTERVAL=<dur>      → tick cadence (default 6h)
//   - TEST_GENIE_FLEET_SCHEDULER_START_JITTER=<dur>  → initial delay (default 0)
//   - TEST_GENIE_FLEET_SCHEDULER_MAX_CONCURRENT=<n>  → simultaneous runs (default 1)
//   - TEST_GENIE_FLEET_SCHEDULER_MAX_PER_CYCLE=<n>   → runs launched per cycle (default 5)
//   - TEST_GENIE_FLEET_SCHEDULER_CYCLE_BUDGET=<dur>  → per-cycle wall-clock cap (default 0 = none)
//   - TEST_GENIE_FLEET_SCHEDULER_STALENESS_HORIZON=<dur> → staleness weighting window (default 168h)
//   - TEST_GENIE_FLEET_SCHEDULER_PRESET=<preset>     → suite shape (default "comprehensive")
func startFleetScheduler(runManager *runmanager.Manager) {
	if !isTruthy(os.Getenv("TEST_GENIE_FLEET_SCHEDULER_ENABLED")) {
		// Silent by default: the scheduler being off is the normal state, not a
		// condition worth logging on every boot.
		return
	}
	preset := os.Getenv("TEST_GENIE_FLEET_SCHEDULER_PRESET")
	if preset == "" {
		preset = "comprehensive"
	}
	scheduler, err := fleetscheduler.New(fleetscheduler.Config{
		Source:           fleetscheduler.NewCLIPrioritySource(0),
		Launcher:         fleetscheduler.NewManagerLauncher(runManager, preset),
		Interval:         parseDurationEnv("TEST_GENIE_FLEET_SCHEDULER_INTERVAL", 6*time.Hour),
		InitialJitter:    parseDurationEnv("TEST_GENIE_FLEET_SCHEDULER_START_JITTER", 0),
		MaxConcurrent:    parseIntEnv("TEST_GENIE_FLEET_SCHEDULER_MAX_CONCURRENT", 1),
		MaxRunsPerCycle:  parseIntEnv("TEST_GENIE_FLEET_SCHEDULER_MAX_PER_CYCLE", 5),
		CycleBudget:      parseDurationEnv("TEST_GENIE_FLEET_SCHEDULER_CYCLE_BUDGET", 0),
		StalenessHorizon: parseDurationEnv("TEST_GENIE_FLEET_SCHEDULER_STALENESS_HORIZON", 7*24*time.Hour),
	})
	if err != nil {
		log.Printf("[test-genie] fleet scheduler not started: %v", err)
		return
	}
	log.Printf("[test-genie] fleet scheduler ENABLED (preset=%s); cycling priority-ordered suites in the background", preset)
	go scheduler.RunLoop(context.Background())
}

func parseIntEnv(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return n
	}
	return fallback
}
