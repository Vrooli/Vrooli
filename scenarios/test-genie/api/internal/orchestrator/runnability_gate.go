package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/targetruntime"
	workspacepkg "test-genie/internal/orchestrator/workspace"
	"test-genie/internal/selfidentity"
)

// runnabilityResolver is the single policy that decides RUN/RUN_DEGRADED/SKIP
// per phase. Stateless and pure; held as a value so tests can swap a fake.
var runnabilityResolver runnability.Resolver = runnability.NewResolver()

// resolveRunContext builds the phase-independent runnability.RunContext for one
// suite execution: who the target is relative to us, which surfaces are live,
// whether the target is routed-eligible, and which resources are available.
//
// It performs no lifecycle mutation — only reads. live carries the already-
// probed surfaces (so callers that probed once don't re-probe); routedEligible
// is supplied by the caller because computing it requires the (cached) auditor
// scan the playbooks phase owns.
func resolveRunContext(
	env workspacepkg.Environment,
	live targetruntime.URLs,
	routedEligible bool,
	routedReason string,
	resources map[string]bool,
) runnability.RunContext {
	return runnability.RunContext{
		TargetIsSelf:   selfidentity.Is(strings.TrimSpace(env.ScenarioName)),
		RoutedEligible: routedEligible,
		RoutedReason:   routedReason,
		LiveSurfaces: runnability.Surfaces{
			UI:  strings.TrimSpace(firstNonEmpty(env.UIURL, live.UI)) != "",
			API: strings.TrimSpace(firstNonEmpty(env.APIURL, live.API)) != "",
		},
		Resources: resources,
	}
}

// resolvePhaseVerdict resolves the runnability verdict for one phase definition
// against a run context, using the package resolver.
func resolvePhaseVerdict(def phases.Definition, rc runnability.RunContext) runnability.Verdict {
	caps := def.Capabilities
	if strings.TrimSpace(caps.Phase) == "" {
		caps.Phase = def.Name.String()
	}
	return runnabilityResolver.Resolve(caps, rc)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// newSkippedPhaseResult builds the execution result for a phase the runnability
// gate decided not to run. It writes a small skip log for traceability and
// records the verdict reason/remediation so downstream surfaces (CLI printer,
// requirements sync, the report JSON) can explain the skip honestly.
func (o *SuiteOrchestrator) newSkippedPhaseResult(def phases.Definition, runLogDir string, v runnability.Verdict) PhaseExecutionResult {
	logPath := filepath.Join(runLogDir, def.Name.String()+".log")
	if f, err := os.Create(logPath); err == nil {
		fmt.Fprintf(f, "[SKIP] %s\n", v.Reason)
		if strings.TrimSpace(v.Remediation) != "" {
			fmt.Fprintf(f, "Remediation: %s\n", v.Remediation)
		}
		_ = f.Close()
	}
	displayLogPath := logPath
	if rel, err := filepath.Rel(o.projectRoot, logPath); err == nil {
		displayLogPath = rel
	}
	return PhaseExecutionResult{
		Name:               def.Name.String(),
		Status:             phaseStatusSkipped,
		DurationSeconds:    0,
		LogPath:            displayLogPath,
		Remediation:        v.Remediation,
		RunnabilityVerdict: v.Kind.String(),
		RunnabilityReason:  v.Reason,
		Observations:       []phases.Observation{phases.NewSkipObservation(v.Reason)},
	}
}

// mergeRunnabilityObservations prepends a degraded-run notice (if any) to the
// phase's existing pre-run observations so the operator sees why a less-
// preferred path was taken before the phase output.
func mergeRunnabilityObservations(v runnability.Verdict, existing []phases.Observation) []phases.Observation {
	if v.Kind != runnability.VerdictRunDegraded || strings.TrimSpace(v.Reason) == "" {
		return existing
	}
	notice := phases.NewWarningObservation("runnability (degraded): " + v.Reason)
	return append([]phases.Observation{notice}, existing...)
}

// annotatePhaseRunnability records the gate verdict on a result for a phase that
// actually ran (run or run_degraded), so the per-phase verdict travels in the
// report even when the phase executed normally.
func annotatePhaseRunnability(result *PhaseExecutionResult, v runnability.Verdict) {
	if result == nil {
		return
	}
	result.RunnabilityVerdict = v.Kind.String()
	if v.Kind == runnability.VerdictRunDegraded {
		result.RunnabilityReason = v.Reason
	}
}

// computeSuiteVerdict derives the tri-state suite verdict from the per-phase
// results and the selected definitions (for Optional lookup). FAIL dominates;
// otherwise a non-optional skip degrades the run to PARTIAL; an optional skip
// stays PASS.
func computeSuiteVerdict(results []PhaseExecutionResult, defs []phases.Definition) string {
	optional := make(map[string]bool, len(defs))
	for _, d := range defs {
		optional[d.Name.Key()] = d.Optional
	}
	partial := false
	for _, r := range results {
		switch strings.ToLower(strings.TrimSpace(r.Status)) {
		case phaseStatusFailed:
			return SuiteVerdictFail
		case phaseStatusSkipped:
			if !optional[normalizePhaseName(r.Name)] {
				partial = true
			}
		}
	}
	if partial {
		return SuiteVerdictPartial
	}
	return SuiteVerdictPass
}
