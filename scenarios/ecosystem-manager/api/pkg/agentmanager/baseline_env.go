package agentmanager

import (
	"os"
	"strings"
)

// shadowRoutingEnvKeys are the ambient Baseline-Modes routing variables an
// orchestrator forwards verbatim into the agent-manager runs it spawns, so the
// coding agent's nested CLI calls inherit the same shadow target the
// orchestrator itself is running under (plan P1.5 §137: "orchestrators pass
// VROOLI_SHADOW_SCENARIOS into spawned runs through CreateRun.Environment, so
// nested CLI calls inherit both Case-A and Case-B routing").
//
// Every key is VROOLI_-prefixed so the forwarded map always satisfies
// agent-manager's validateCustomEnvironment allowlist (VROOLI_ prefix, ≤20
// entries, ≤4096B total). Today this is the single named-scenarios var; the
// slice exists so adding a future routing var is a one-line change with no
// caller churn.
var shadowRoutingEnvKeys = []string{EnvShadowScenarios}

// EnvShadowScenarios is the ambient variable naming the scenarios whose nested
// CLI calls route to their shadow instance (a comma-separated slug list). It is
// the SSOT for the env key, shared by the process-forwarding path
// (AmbientShadowEnv) and the engagement layer (WithEngagedShadowScenario), so
// the name is defined once.
const EnvShadowScenarios = "VROOLI_SHADOW_SCENARIOS"

// WithEngagedShadowScenario returns base with scenario unioned into the
// VROOLI_SHADOW_SCENARIOS comma-list, so a coding agent spawned for an active
// Baseline Modes engagement routes that scenario's nested CLI calls to its
// shadow instance — while preserving any scenarios the process already forwards
// (AmbientShadowEnv). A blank scenario returns base unchanged (possibly nil, so
// the result assigns straight to the proto Environment map).
func WithEngagedShadowScenario(base map[string]string, scenario string) map[string]string {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return base
	}
	existing := ""
	if base != nil {
		existing = base[EnvShadowScenarios]
	}
	out := make(map[string]string, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out[EnvShadowScenarios] = unionCSV(existing, scenario)
	return out
}

// unionCSV merges add into the comma-separated csv, de-duplicating and trimming,
// preserving first-seen order.
func unionCSV(csv, add string) string {
	seen := make(map[string]struct{})
	var ordered []string
	appendUnique := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, dup := seen[s]; dup {
			return
		}
		seen[s] = struct{}{}
		ordered = append(ordered, s)
	}
	for _, part := range strings.Split(csv, ",") {
		appendUnique(part)
	}
	appendUnique(add)
	return strings.Join(ordered, ",")
}

// AmbientShadowEnv collects the Baseline-Modes shadow-routing variables present
// in the current process environment, returning a map suitable for
// ExecuteRequest.Environment.
//
// When ecosystem-manager runs inside an active baseline engagement, the
// engagement sets VROOLI_SHADOW_SCENARIOS in EM's environment; forwarding it
// onto every spawned run makes the coding agent's nested `vrooli`/scenario CLI
// calls auto-route to the shadow instance instead of live. When no engagement is
// active the var is unset and this returns nil, so the run inherits no extra env
// (exactly the pre-Baseline-Modes behavior — this is safe to call unconditionally).
func AmbientShadowEnv() map[string]string {
	return collectShadowEnv(os.Getenv)
}

// collectShadowEnv is AmbientShadowEnv with an injectable lookup so the
// propagation contract is unit-testable without mutating the process
// environment. It returns nil (not an empty map) when no routing var is set, so
// callers can assign the result straight to a proto map field without
// materializing an empty one.
func collectShadowEnv(getenv func(string) string) map[string]string {
	var env map[string]string
	for _, k := range shadowRoutingEnvKeys {
		v := strings.TrimSpace(getenv(k))
		if v == "" {
			continue
		}
		if env == nil {
			env = make(map[string]string, len(shadowRoutingEnvKeys))
		}
		env[k] = v
	}
	return env
}
