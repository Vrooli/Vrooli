package config

// BehaviorConfig holds operator-tunable runtime behavior knobs for the
// workspace-sandbox API. Loaded at boot from `<scenarioDir>/.vrooli/config.json`
// under the top-level `behavior` key. The shape mirrors the JSON exactly.
//
// Distinction from PolicyConfig: PolicyConfig (in config.go) governs commit
// attribution + validation/teardown hooks — substrate-level rules wired into
// the sandbox machinery. BehaviorConfig governs the operator-facing knobs
// that an integrator might want to tune without rebuilding (the protected-
// mode git allowlist + rejection-message templates today; more in future).
//
// File precedent: see scenarios/{graph-studio,system-monitor,...}/.vrooli/config.json
// for the JSON file pattern. We extend the existing schema with a new
// top-level `behavior` key rather than introducing a parallel file.
type BehaviorConfig struct {
	// Protected groups protected-mode runtime guardrail tunables.
	Protected ProtectedBehaviorConfig `json:"protected"`
}

// ProtectedBehaviorConfig configures the API-side defaults for protected-mode
// guardrails. These defaults are applied by the workspace-sandbox API when
// the per-sandbox wire payload (types.ProtectedConfig) does not override.
//
// See scenarios/workspace-sandbox/api/internal/runtime/git_allowlist.go for
// the consumer.
type ProtectedBehaviorConfig struct {
	// GitAllowlist is the API-side default allowlist of `git` verbs the API
	// will allow when a sandbox's wire-level ProtectedConfig.GitAllowlist is
	// empty. Empty here means "no API-side default" — falls back to the
	// hardcoded read-only set (status, diff, log, show, rev-parse).
	GitAllowlist []string `json:"gitAllowlist,omitempty"`

	// GitDenyMessageTemplate overrides the rejection message rendered when a
	// blocked git verb is invoked. Supports {verb} and {allowlist}
	// placeholders. Empty means "use the hardcoded strong default".
	GitDenyMessageTemplate string `json:"gitDenyMessageTemplate,omitempty"`

	// GitNoVerbMessageTemplate overrides the rejection message rendered when
	// bare `git` is invoked (no verb). Supports {allowlist} placeholder. Empty
	// means "use the hardcoded short default".
	GitNoVerbMessageTemplate string `json:"gitNoVerbMessageTemplate,omitempty"`
}

// DefaultBehavior returns the zero-state defaults. Empty allowlist + empty
// templates mean "fall back to the hardcoded defaults in runtime.EvaluateProtectedGitAllowlist".
// The runtime defaults are the source of truth; this loader merely lets an
// operator override them via JSON.
func DefaultBehavior() BehaviorConfig {
	return BehaviorConfig{
		Protected: ProtectedBehaviorConfig{
			GitAllowlist: nil,
		},
	}
}
