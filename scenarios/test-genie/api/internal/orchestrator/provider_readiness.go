package orchestrator

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/packages/artifactpaths"

	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerreadiness"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

type providerReadinessPlan struct {
	Active   []phases.Definition
	Blocked  map[string]providerreadiness.Outcome
	Outcomes []providerreadiness.Outcome
	Stages   []PreparationStage
	releases []func()
}

func (p *providerReadinessPlan) releaseProviders() {
	for index := len(p.releases) - 1; index >= 0; index-- {
		p.releases[index]()
	}
	p.releases = nil
}

const defaultProviderReadinessConcurrency = 4

// providerLeaseWaitTimeout bounds the hand-off from readiness probes to the
// execution phases. A server-owned run has a lifetime context that intentionally
// survives client disconnects, so waiting on a provider lease with that context
// alone could leave a run in preparing forever when another owner disappears.
// A bounded wait turns that infrastructure condition into an explicit,
// evidence-bearing provider-unavailable outcome.
var providerLeaseWaitTimeout = 2 * time.Minute

// slowProviderCheckThreshold is the point at which a provider readiness check
// stops looking like a health probe and starts looking like a cold start.
//
// It is deliberately generous. The alarm exists to make a pathological case
// visible — a 41-minute start went unnoticed — not to flag ordinary variance,
// and an alarm that fires on ordinary variance is one that gets ignored.
const slowProviderCheckThreshold = 2 * time.Minute

func providerReadinessConcurrency() int {
	value := strings.TrimSpace(os.Getenv("TEST_GENIE_PROVIDER_READINESS_CONCURRENCY"))
	if strings.EqualFold(value, "serial") || value == "0" || strings.EqualFold(value, "false") {
		return 1
	}
	if n, err := strconv.Atoi(value); err == nil && n > 0 {
		return n
	}
	return defaultProviderReadinessConcurrency
}

type synchronizedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (w *synchronizedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.w == nil {
		return len(p), nil
	}
	return w.w.Write(p)
}

type providerReadinessResult struct {
	def      phases.Definition
	outcome  providerreadiness.Outcome
	duration time.Duration
}

func (o *SuiteOrchestrator) checkProviderReadiness(
	ctx context.Context,
	env workspacepkg.Environment,
	defs []phases.Definition,
	logWriter io.Writer,
	emit ExecutionEventCallback,
) providerReadinessPlan {
	active := make([]phases.Definition, 0, len(defs))
	blocked := make(map[string]providerreadiness.Outcome)
	outcomes := make([]providerreadiness.Outcome, 0, len(defs))
	stages := make([]PreparationStage, 0, len(defs))
	manager := o.readiness
	if manager == nil {
		manager = providerreadiness.NewManager()
	}
	// Enable the staleness gate against this repo. Without a root the gate is
	// inert, which is the fail-open default: a provider is never restarted on
	// the strength of a root we could not resolve.
	if manager.RepoRoot == "" {
		manager.RepoRoot = o.repoRoot()
	}
	// The cooldown window spans runs, so its ledger has to outlive this one.
	if manager.Ledger == nil {
		if artifactRoot, err := artifactpaths.ScenarioRoot("test-genie"); err == nil {
			manager.Ledger = providerreadiness.NewRestartLedgerAt(
				artifactpaths.ScenarioPath(artifactRoot, artifactpaths.CoverageRoot, "runtime", "provider-restarts.json"))
		}
	}

	results := make([]providerReadinessResult, len(defs))
	var providerLocks sync.Map
	serialLog := &synchronizedWriter{w: logWriter}
	worker := func(index int) {
		def := defs[index]
		policy := def.Policy
		if policy.IsZero() {
			policy = phasepolicy.FromLegacyCatalog(def.Optional, false)
		}
		provider := strings.TrimSpace(def.ProviderScenario)
		started := time.Now()
		if provider != "" {
			lockValue, _ := providerLocks.LoadOrStore(provider, &sync.Mutex{})
			lockValue.(*sync.Mutex).Lock()
			defer lockValue.(*sync.Mutex).Unlock()
		}
		// A server-owned run intentionally survives client disconnects. Bound
		// both the provider probe and its per-provider ownership wait so a
		// vanished owner cannot strand preparation indefinitely.
		checkCtx, cancelCheck := context.WithTimeout(ctx, providerLeaseWaitTimeout)
		defer cancelCheck()
		outcome := manager.Check(checkCtx, providerreadiness.Input{
			Phase:            def.Name.String(),
			ProviderScenario: provider,
			TargetScenario:   env.ScenarioName,
			TargetPath:       env.ScenarioDir,
			Policy:           policy,
			Timeout:          def.Timeout,
		}, serialLog)
		results[index] = providerReadinessResult{def: def, outcome: outcome, duration: time.Since(started)}
	}
	limit := providerReadinessConcurrency()
	semaphore := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for index := range defs {
		if limit == 1 {
			worker(index)
			continue
		}
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			worker(index)
		}(index)
	}
	wg.Wait()
	for _, result := range results {
		def, outcome := result.def, result.outcome
		provider := strings.TrimSpace(def.ProviderScenario)
		if provider != "" {
			stages = append(stages, PreparationStage{Name: "provider_check", Parent: "provider_readiness", Subject: provider, Status: string(outcome.Status), DurationMilliseconds: result.duration.Milliseconds()})
			emitPreparationProgress(emit, "provider_readiness", fmt.Sprintf("checked %s: %s in %s", provider, outcome.Status, result.duration.Round(time.Millisecond)))
			// A readiness check is a health probe against an already-running
			// provider; it should take seconds. One template-manager start took
			// 41 MINUTES and passed unnoticed, because a slow check is
			// indistinguishable from a fast one in a duration column nobody
			// reads. Say it out loud instead.
			if result.duration >= slowProviderCheckThreshold {
				emitPreparationProgress(emit, "provider_readiness", fmt.Sprintf(
					"SLOW provider check: %s took %s (threshold %s) — the provider is probably starting up rather than responding; this is provider uptime, not a Test Genie scheduling problem",
					provider, result.duration.Round(time.Second), slowProviderCheckThreshold))
			}
		}
		if def.ProviderScenario != "" || outcome.Status != providerreadiness.OutcomeReady {
			outcomes = append(outcomes, outcome)
		}
		if outcome.BlocksExecution() {
			blocked[def.Name.Key()] = outcome
			continue
		}
		active = append(active, def)
	}

	// Hold providers from the end of readiness through the last phase RPC. A
	// concurrent run may still inspect other providers, but it cannot restart
	// one while this run is consuming it. Sort acquisition to avoid cross-run
	// lock-order deadlocks when suites use several providers.
	providers := make([]string, 0, len(active))
	seen := map[string]bool{}
	for _, def := range active {
		provider := strings.TrimSpace(def.ProviderScenario)
		if provider != "" && !seen[provider] {
			seen[provider] = true
			providers = append(providers, provider)
		}
	}
	sort.Strings(providers)
	releases := make([]func(), 0, len(providers))
	leaseCtx, cancelLeaseWait := context.WithTimeout(ctx, providerLeaseWaitTimeout)
	defer cancelLeaseWait()
	for _, provider := range providers {
		release, err := manager.LeaseProviderContext(leaseCtx, provider)
		if err != nil {
			// No phase may execute after losing its lifetime context while
			// waiting for provider ownership. Release every hold acquired so
			// far instead of stranding it behind this abandoned preparation.
			for index := len(releases) - 1; index >= 0; index-- {
				releases[index]()
			}
			for _, def := range active {
				outcome := providerreadiness.Outcome{Phase: def.Name.String(), ProviderScenario: def.ProviderScenario,
					Status: providerreadiness.OutcomeUnreachable, Err: err,
					Message: fmt.Sprintf("provider ownership preparation canceled: %v", err)}
				blocked[def.Name.Key()] = outcome
				for index := range outcomes {
					if outcomes[index].Phase == def.Name.String() {
						outcomes[index] = outcome
					}
				}
			}
			return providerReadinessPlan{Blocked: blocked, Outcomes: outcomes, Stages: stages}
		}
		releases = append(releases, release)
	}
	return providerReadinessPlan{Active: active, Blocked: blocked, Outcomes: outcomes, Stages: stages, releases: releases}
}

func (o *SuiteOrchestrator) newProviderReadinessPhaseResult(def phases.Definition, runLogDir string, outcome providerreadiness.Outcome) PhaseExecutionResult {
	logPath := phaseLogPath(runLogDir, def.Name)
	obs := providerreadiness.ResultObservation(outcome)
	if f, err := os.Create(logPath); err == nil {
		fmt.Fprintf(f, "[PROVIDER_READINESS] %s\n", obs.String())
		if remediation := providerreadiness.Remediation(outcome); strings.TrimSpace(remediation) != "" {
			fmt.Fprintf(f, "Remediation: %s\n", remediation)
		}
		_ = f.Close()
	}
	displayLogPath := logPath
	if rel, err := filepath.Rel(o.projectRoot, logPath); err == nil {
		displayLogPath = rel
	}
	status := phaseStatusProviderUnavailable
	if outcome.SkipsWithoutFailure() {
		status = phaseStatusSkipped
	}
	return PhaseExecutionResult{
		Name:            def.Name.String(),
		Status:          status,
		DurationSeconds: 0,
		LogPath:         displayLogPath,
		Error:           outcome.ErrorString(),
		Classification:  providerreadiness.FailureClass(outcome),
		Remediation:     providerreadiness.Remediation(outcome),
		Observations:    []phases.Observation{obs},
	}
}

// repoRoot resolves the repository root from the scenarios root the
// orchestrator was constructed with. Returning "" disables the provider
// staleness gate, which is the safe direction.
func (o *SuiteOrchestrator) repoRoot() string {
	if o == nil || strings.TrimSpace(o.scenariosRoot) == "" {
		return ""
	}
	abs, err := filepath.Abs(o.scenariosRoot)
	if err != nil {
		return ""
	}
	return filepath.Dir(abs)
}
