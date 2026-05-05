package heartbeat

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"prompt-manager/store"
)

// TestBuildHeartbeatAttributionEnv_HappyPath asserts the helper produces a
// well-formed (key, value) pair: key matches the canonical env-var name and
// value base64-decodes into a complete AttributionInfo with the expected
// agent-member shape.
func TestBuildHeartbeatAttributionEnv_HappyPath(t *testing.T) {
	key, value := buildHeartbeatAttributionEnv("marketing-crew", "researcher")

	if key != "VROOLI_PROMPT_MANAGER_ATTRIBUTION" {
		t.Errorf("key = %q, want VROOLI_PROMPT_MANAGER_ATTRIBUTION", key)
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}

	var info store.AttributionInfo
	if err := json.Unmarshal(decoded, &info); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}

	if info.Kind != store.KnowledgeKindAgentMember {
		t.Errorf("Kind = %q, want agent-member", info.Kind)
	}
	if info.SpawnOrigin != store.SpawnOriginHeartbeat {
		t.Errorf("SpawnOrigin = %q, want heartbeat", info.SpawnOrigin)
	}
	if info.MemberID == nil || *info.MemberID != "researcher" {
		t.Errorf("MemberID = %v, want researcher", info.MemberID)
	}
	if info.TeamID == nil || *info.TeamID != "marketing-crew" {
		t.Errorf("TeamID = %v, want marketing-crew", info.TeamID)
	}
	if info.RunID != nil {
		t.Errorf("RunID = %v, want nil (heartbeat-spawn leaves run_id unset; canon: RUNTIME_ATTRIBUTION.md § Env-var bridge)", info.RunID)
	}
	if info.SourceSkillID != nil {
		t.Errorf("SourceSkillID = %v, want nil (no skill mediates a heartbeat spawn)", info.SourceSkillID)
	}
}

// TestBuildHeartbeatAttributionEnv_PassesValidator asserts the produced
// payload is accepted by the API-side validateAttribution. This is the
// load-bearing round-trip: spawner-emitted attribution MUST be a payload the
// API will accept on the spawned agent's first knowledge-add.
func TestBuildHeartbeatAttributionEnv_PassesValidator(t *testing.T) {
	_, value := buildHeartbeatAttributionEnv("marketing-crew", "researcher")
	info, err := parseAttributionHeader(value)
	if err != nil {
		t.Fatalf("parseAttributionHeader: %v", err)
	}
	if err := validateAttribution(info, "marketing-crew"); err != nil {
		t.Errorf("spawner-built payload must pass API validation, got %v", err)
	}
}

// TestBuildHeartbeatAttributionEnv_RejectedByValidatorOnTeamMismatch asserts
// the team_id baked into the payload IS cross-checked against the URL team
// id by the API. A spawner that builds attribution for marketing-crew must
// not be silently accepted when its CLI posts to /teams/monetization/...
// (handler returns 400 team_mismatch).
func TestBuildHeartbeatAttributionEnv_RejectedByValidatorOnTeamMismatch(t *testing.T) {
	_, value := buildHeartbeatAttributionEnv("marketing-crew", "researcher")
	info, err := parseAttributionHeader(value)
	if err != nil {
		t.Fatalf("parseAttributionHeader: %v", err)
	}
	err = validateAttribution(info, "monetization")
	if err == nil {
		t.Fatal("expected team_mismatch error when URL team differs from attribution team")
	}
	// Sanity-check the error message routes to operator-actionable text.
	if !strings.Contains(err.Error(), "team_mismatch") {
		t.Errorf("err = %v, want mention of team_mismatch", err)
	}
}

// TestBuildHeartbeatAttributionEnv_StableAcrossCalls asserts the helper is
// pure: same inputs → same output bytes. This matters for the CLI's
// passthrough invariant — the encoded value must be deterministic so
// snapshot tests, log greps, and operator debugging see a stable string.
func TestBuildHeartbeatAttributionEnv_StableAcrossCalls(t *testing.T) {
	key1, val1 := buildHeartbeatAttributionEnv("marketing-crew", "researcher")
	key2, val2 := buildHeartbeatAttributionEnv("marketing-crew", "researcher")
	if key1 != key2 {
		t.Errorf("key drifts across calls: %q vs %q", key1, key2)
	}
	if val1 != val2 {
		t.Errorf("value drifts across calls: %q vs %q", val1, val2)
	}
}

// TestSpawnAttributionEnvVarMatchesCLI is the load-bearing drift detector
// between the spawner-side env-var name and the CLI-side env-var name. They
// MUST be byte-equal — drift here silently breaks attribution propagation,
// causing every heartbeat-spawned agent to fall back to operator-direct.
//
// The CLI side is at scenarios/prompt-manager/cli/internal/attribution.EnvVar.
// We can't import that package from the API side (CLI imports API, not the
// reverse), so the value is mirrored here as a literal and checked against
// the API-side constant. The CLI side's test (attribution_test.go in the
// CLI package) checks against the same literal — so any drift surfaces in
// at least one of the two suites.
func TestSpawnAttributionEnvVarMatchesCLI(t *testing.T) {
	const cliEnvVar = "VROOLI_PROMPT_MANAGER_ATTRIBUTION"
	if attributionEnvVar != cliEnvVar {
		t.Errorf("attributionEnvVar = %q, must match cli/internal/attribution.EnvVar = %q",
			attributionEnvVar, cliEnvVar)
	}
}
