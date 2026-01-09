package orchestrator

import (
	"fmt"
	"os"
	"strings"

	"test-genie/internal/orchestrator/phases"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

// requirementsSyncDecision makes the requirements-sync choice explicit so the
// Go orchestrator mirrors the legacy bash behavior and remains easy to audit.
type requirementsSyncDecision struct {
	Execute bool
	Forced  bool
	Reason  string
}

func newRequirementsSyncDecision(cfg *workspacepkg.Config, plan *phasePlan, results []PhaseExecutionResult) requirementsSyncDecision {
	if plan == nil || len(plan.Definitions) == 0 {
		return requirementsSyncDecision{Reason: "phase plan unavailable"}
	}
	if len(plan.Selected) == 0 {
		return requirementsSyncDecision{Reason: "no phases selected"}
	}
	if len(results) == 0 {
		return requirementsSyncDecision{Reason: "no phase results recorded"}
	}

	if cfg != nil && cfg.Requirements.Sync != nil && !*cfg.Requirements.Sync {
		return requirementsSyncDecision{Reason: "requirements sync disabled via testing.json"}
	}

	if disabledByEnv("TESTING_REQUIREMENTS_SYNC") {
		return requirementsSyncDecision{Reason: "TESTING_REQUIREMENTS_SYNC disabled"}
	}

	forced := envBool("TESTING_REQUIREMENTS_SYNC_FORCE")
	missing, skipped := summarizePhaseCoverage(plan.Definitions, results)

	var gatingReason string
	if len(missing) > 0 {
		gatingReason = fmt.Sprintf("missing required phases: %s", strings.Join(missing, ", "))
	} else if len(skipped) > 0 {
		gatingReason = fmt.Sprintf("required phases skipped: %s", strings.Join(skipped, ", "))
	}

	if gatingReason != "" && !forced {
		return requirementsSyncDecision{Reason: gatingReason}
	}

	return requirementsSyncDecision{
		Execute: true,
		Forced:  forced && gatingReason != "",
		Reason:  gatingReason,
	}
}

func summarizePhaseCoverage(defs []phases.Definition, results []PhaseExecutionResult) (missing []string, skipped []string) {
	if len(defs) == 0 {
		return nil, nil
	}
	resultLookup := make(map[string]PhaseExecutionResult, len(results))
	for _, result := range results {
		key := normalizePhaseName(result.Name)
		if key == "" {
			continue
		}
		resultLookup[key] = result
	}
	for _, def := range defs {
		if def.Optional {
			continue
		}
		key := def.Name.Key()
		result, exists := resultLookup[key]
		if !exists {
			missing = append(missing, def.Name.String())
			continue
		}
		status := strings.ToLower(strings.TrimSpace(result.Status))
		if isSkippedStatus(status) {
			skipped = append(skipped, def.Name.String())
		}
	}
	return missing, skipped
}

func isSkippedStatus(status string) bool {
	switch status {
	case "skipped", "missing", "not_executable", "not_run":
		return true
	default:
		return false
	}
}

func disabledByEnv(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false
	}
	value = strings.ToLower(value)
	return value == "0" || value == "false" || value == "no"
}

func envBool(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return false
	}
	value = strings.ToLower(value)
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
