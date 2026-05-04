// Package attribution constructs the X-Vrooli-Attribution header value
// the prompt-manager CLI sends on every mutating API call.
//
// The contract — header name, env-var bridge, payload shape, conflict
// policy — lives in docs/agent-system/RUNTIME_ATTRIBUTION.md (canon).
// This package implements the requesting-side half (CLI → API). The
// receiving side lives in api/heartbeat/handlers.go (P3.4).
//
// Design note: the env-var IS the header value. When
// VROOLI_PROMPT_MANAGER_ATTRIBUTION is set (by an agent-manager-spawned
// prompt-manager process; see P3.5), the CLI forwards its value
// verbatim — no decode-and-re-encode, no field reconstruction. This is
// load-bearing: it keeps the CLI a pure passthrough so a future
// payload-shape change requires no CLI update.
package attribution

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// HeaderName is the HTTP header that carries structured attribution
// from any client (CLI, SDK, UI) to prompt-manager's API. Mirrors the
// canon in docs/agent-system/RUNTIME_ATTRIBUTION.md § HTTP header.
const HeaderName = "X-Vrooli-Attribution"

// EnvVar is the per-process env var an agent-manager-spawned CLI reads
// to inherit attribution from its spawner. See RUNTIME_ATTRIBUTION.md
// § Env-var bridge for the lifecycle.
const EnvVar = "VROOLI_PROMPT_MANAGER_ATTRIBUTION"

// Attribution kinds. Closed vocabulary; mirrors api/store/models.go
// (KnowledgeKind*). Drift between the CLI and API constants surfaces
// as test failures in attribution_test.go.
const (
	KindAgentMember    = "agent-member"
	KindWriterSkill    = "writer-skill"
	KindOperatorDirect = "operator-direct"
	KindExternal       = "external"
	KindLegacy         = "legacy"
	KindInvestigation  = "investigation"
)

// Spawn origins. Closed vocabulary; mirrors api/store/models.go
// (SpawnOrigin*).
const (
	SpawnOriginHeartbeat     = "heartbeat"
	SpawnOriginOperatorCLI   = "operator-cli"
	SpawnOriginSwarmTask     = "swarm-task"
	SpawnOriginVisionWalk    = "vision-walk"
	SpawnOriginInvestigation = "investigation"
	SpawnOriginLegacy        = "legacy"
	SpawnOriginUnknown       = "unknown"
)

// Info is the structured attribution payload. Wire-compatible with
// api/store/models.go::AttributionInfo (the API-side struct is
// canonical; this is the CLI mirror). Pointer fields marshal as JSON
// null when nil — the canon preserves null over omission so every
// payload has the same field set regardless of kind.
type Info struct {
	Kind          string  `json:"kind"`
	MemberID      *string `json:"member_id"`
	TeamID        *string `json:"team_id"`
	RunID         *string `json:"run_id"`
	SpawnOrigin   string  `json:"spawn_origin"`
	SourceSkillID *string `json:"source_skill_id"`
}

// HeaderValue returns the X-Vrooli-Attribution header value for the
// current process.
//
//   - When VROOLI_PROMPT_MANAGER_ATTRIBUTION is set, the value is
//     returned verbatim (passthrough — see package doc).
//   - Otherwise, the CLI constructs operator-direct attribution and
//     base64-encodes it.
//
// HeaderValue never returns an error: a malformed env-var value is
// still forwarded verbatim and rejected by the API (HTTP 400). This
// matches the canon — the CLI is a passthrough; the API is the
// validator. A panic-free fallback to operator-direct on encode
// failure is impossible (json.Marshal on a fixed-shape struct cannot
// fail), so any encode failure indicates a programming bug and is
// surfaced via panic in Encode().
func HeaderValue() string {
	if v := strings.TrimSpace(os.Getenv(EnvVar)); v != "" {
		return v
	}
	return Encode(OperatorDirect())
}

// OperatorDirect returns the default attribution for a human at the
// CLI: kind=operator-direct, spawn_origin=operator-cli, no
// agent/team/run/skill context.
func OperatorDirect() Info {
	return Info{
		Kind:        KindOperatorDirect,
		SpawnOrigin: SpawnOriginOperatorCLI,
	}
}

// Encode marshals info to canonical JSON and base64-encodes the
// result. The returned string is suitable as the X-Vrooli-Attribution
// header value. Panics on json.Marshal failure — the Info shape is
// fully serializable, so any failure indicates a programming bug.
func Encode(info Info) string {
	payload, err := json.Marshal(info)
	if err != nil {
		panic(fmt.Sprintf("attribution: marshal Info: %v", err))
	}
	return base64.StdEncoding.EncodeToString(payload)
}

// HeaderMap returns a header map suitable for cliutil.HTTPClient's
// SetHeaderSource. Adapter helper; bare HeaderValue() is preferred
// for direct callers.
func HeaderMap() map[string]string {
	return map[string]string{HeaderName: HeaderValue()}
}
