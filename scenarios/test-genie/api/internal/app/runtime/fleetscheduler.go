package runtime

import (
	"context"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/fleetscheduler"
	"test-genie/internal/runmanager"
)

// fleetRosterEphemeralTTL prevents timestamp-named generated scenarios from
// becoming permanent coverage gaps after their short-lived experiment ends.
const fleetRosterEphemeralTTL = 30 * 24 * time.Hour

var timestampedScenarioDate = regexp.MustCompile(`(?:^|[-_])(20[0-9]{2})[-_]?([0-9]{2})[-_]?([0-9]{2})(?:$|[-_])`)

type fleetSchedulerOptions struct {
	enabled          bool
	preset           string
	interval         time.Duration
	initialJitter    time.Duration
	maxConcurrent    int
	maxRunsPerCycle  int
	cycleBudget      time.Duration
	stalenessHorizon time.Duration
}

func fleetSchedulerOptionsFromLookup(lookup func(string) string) fleetSchedulerOptions {
	options := fleetSchedulerOptions{
		enabled:          isTruthy(lookup("TEST_GENIE_FLEET_SCHEDULER_ENABLED")),
		preset:           lookup("TEST_GENIE_FLEET_SCHEDULER_PRESET"),
		interval:         parseDurationValue(lookup("TEST_GENIE_FLEET_SCHEDULER_INTERVAL"), 6*time.Hour),
		initialJitter:    parseDurationValue(lookup("TEST_GENIE_FLEET_SCHEDULER_START_JITTER"), 0),
		maxConcurrent:    parsePositiveInt(lookup("TEST_GENIE_FLEET_SCHEDULER_MAX_CONCURRENT"), 1),
		maxRunsPerCycle:  parsePositiveInt(lookup("TEST_GENIE_FLEET_SCHEDULER_MAX_PER_CYCLE"), 5),
		cycleBudget:      parseDurationValue(lookup("TEST_GENIE_FLEET_SCHEDULER_CYCLE_BUDGET"), 0),
		stalenessHorizon: parseDurationValue(lookup("TEST_GENIE_FLEET_SCHEDULER_STALENESS_HORIZON"), 7*24*time.Hour),
	}
	if options.preset == "" {
		options.preset = "comprehensive"
	}
	return options
}

// fleetRosterFromScenariosRoot returns a roster provider listing the first-level
// scenario directories under scenariosRoot (hidden entries skipped), so the
// fleet ledger can name scenarios that exist on disk but have no run in the
// window. It is best-effort: a read error yields an empty roster, and the ledger
// treats "no roster" as an honest unknown rather than asserting zero coverage.
func fleetRosterFromScenariosRoot(scenariosRoot string) func(ctx context.Context) ([]string, error) {
	return func(ctx context.Context) ([]string, error) {
		now := time.Now().UTC()
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
			if timestampedScenarioExpired(name, now) {
				continue
			}
			names = append(names, name)
		}
		sort.Strings(names)
		return names, nil
	}
}

func timestampedScenarioExpired(name string, now time.Time) bool {
	match := timestampedScenarioDate.FindStringSubmatch(name)
	if len(match) != 4 {
		return false
	}
	date, err := time.ParseInLocation("20060102", match[1]+match[2]+match[3], time.UTC)
	if err != nil {
		return false
	}
	return now.Sub(date) > fleetRosterEphemeralTTL
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
func runFleetScheduler(ctx context.Context, runManager *runmanager.Manager) {
	options := fleetSchedulerOptionsFromLookup(os.Getenv)
	if !options.enabled {
		// Silent by default: the scheduler being off is the normal state, not a
		// condition worth logging on every boot.
		return
	}
	preset := options.preset
	scheduler, err := fleetscheduler.New(fleetscheduler.Config{
		Source:           fleetscheduler.NewCLIPrioritySource(0),
		Launcher:         fleetscheduler.NewManagerLauncher(runManager, preset),
		Interval:         options.interval,
		InitialJitter:    options.initialJitter,
		MaxConcurrent:    options.maxConcurrent,
		MaxRunsPerCycle:  options.maxRunsPerCycle,
		CycleBudget:      options.cycleBudget,
		StalenessHorizon: options.stalenessHorizon,
	})
	if err != nil {
		log.Printf("[test-genie] fleet scheduler not started: %v", err)
		return
	}
	log.Printf("[test-genie] fleet scheduler ENABLED (preset=%s); cycling priority-ordered suites in the background", preset)
	scheduler.RunLoop(ctx)
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return n
	}
	return fallback
}
