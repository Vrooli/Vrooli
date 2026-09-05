// Package heartbeat — spawn_attribution.go builds the structured
// attribution payload prompt-manager propagates into agent-manager-spawned
// processes via CreateRunRequest.Environment.
//
// This file is the spawner-side mirror of
// scenarios/prompt-manager/cli/internal/attribution — the CLI consumes the
// env-var value verbatim as the X-Vrooli-Attribution HTTP header. The
// canonical contract lives in docs/agent-system/RUNTIME_ATTRIBUTION.md
// § Env-var bridge: VROOLI_PROMPT_MANAGER_ATTRIBUTION.
//
// Design rationale:
//
//   - The CLI is a pure passthrough: it forwards the env-var value as the
//     header without decoding or reconstructing fields. So the spawner must
//     emit a complete payload (every field present, even if null).
//   - run_id is unavailable at CreateRunRequest construction time because
//     agent-manager assigns the run UUID after the request lands. Per
//     RUNTIME_ATTRIBUTION.md § Per-kind required fields, the validator
//     accepts a null run_id for kind=agent-member specifically when
//     spawn_origin=heartbeat — this resolves the chicken-and-egg between
//     request construction and run-id assignment. Future strengthening
//     (token-derived run_id) is documented under § Future strengthening.
//   - The env-var key MUST match cli/internal/attribution.EnvVar. Drift
//     surfaces as TestSpawnAttributionEnvVarMatchesCLI in the test file.
package heartbeat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"prompt-manager/internal/store"
)

// attributionEnvVar is the per-process env var an agent-manager-spawned
// CLI reads to inherit attribution from prompt-manager (its spawner). The
// CLI consumes this name from
// scenarios/prompt-manager/cli/internal/attribution.EnvVar; the two names
// MUST stay equal.
const attributionEnvVar = "VROOLI_PROMPT_MANAGER_ATTRIBUTION"

// buildHeartbeatAttributionEnv returns the (key, value) pair the heartbeat
// executor merges into CreateRunRequest.Environment so the spawned agent
// process inherits its structured attribution.
//
// The constructed AttributionInfo describes the team-member identity the
// spawned process embodies:
//
//   - Kind            = agent-member  (the process IS this member)
//   - MemberID        = agentID       (team-scoped member id)
//   - TeamID          = teamID        (the team the member belongs to)
//   - SpawnOrigin     = heartbeat
//   - RunID           = nil           (assigned post-CreateRun by
//     agent-manager; permitted by validator
//     for spawn_origin=heartbeat)
//   - SourceSkillID   = nil           (no writer-skill mediated this spawn)
//
// The value is the standard base64-encoded canonical-JSON form of the
// AttributionInfo. Identical encoding to the X-Vrooli-Attribution HTTP
// header so the CLI's role is pure passthrough.
//
// Panics on json.Marshal failure — the AttributionInfo shape is
// fully-serialisable, so any failure indicates a code-level breakage
// (e.g. adding an unsupported field type to the struct), not a runtime
// condition. Returning an error here would force every caller to handle
// an impossible failure mode.
func buildHeartbeatAttributionEnv(teamID, agentID string) (string, string) {
	// Copy strings so the AttributionInfo's pointer fields don't alias the
	// caller's variables. Belt-and-braces against a future caller mutating
	// teamID/agentID after the call returns.
	memberID := agentID
	teamIDCopy := teamID
	info := store.AttributionInfo{
		Kind:        store.KnowledgeKindAgentMember,
		MemberID:    &memberID,
		TeamID:      &teamIDCopy,
		SpawnOrigin: store.SpawnOriginHeartbeat,
	}
	payload, err := json.Marshal(info)
	if err != nil {
		panic(fmt.Sprintf("heartbeat: marshal AttributionInfo: %v", err))
	}
	return attributionEnvVar, base64.StdEncoding.EncodeToString(payload)
}
