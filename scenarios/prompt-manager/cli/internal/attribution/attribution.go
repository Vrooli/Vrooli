// Package attribution constructs the X-Vrooli-Attribution header value
// the prompt-manager CLI sends on every mutating API call.
//
// The contract — header name, env-var bridge, payload shape, conflict
// policy — lives in docs/agent-system/RUNTIME_ATTRIBUTION.md (canon).
// This package implements the requesting-side half (CLI → API). The
// receiving side lives in api/heartbeat/handlers.go.
//
// Design note: the env-var IS the header value. When
// VROOLI_PROMPT_MANAGER_ATTRIBUTION is set (by an agent-manager-spawned
// prompt-manager process — the heartbeat executor populates it on every
// CreateRunRequest.Environment), the CLI forwards its value verbatim —
// no decode-and-re-encode, no field reconstruction. This is load-bearing:
// it keeps the CLI a pure passthrough so a future payload-shape change
// requires no CLI update.
package attribution

import (
	"fmt"
	"github.com/vrooli/cli-core/teamattribution"
	"os"
	"strings"
	"sync"
)

// HeaderName is the HTTP header that carries structured attribution
// from any client (CLI, SDK, UI) to prompt-manager's API. Mirrors the
// canon in docs/agent-system/RUNTIME_ATTRIBUTION.md § HTTP header.
const HeaderName = teamattribution.HeaderName

// EnvVar is the per-process env var an agent-manager-spawned CLI reads
// to inherit attribution from its spawner. See RUNTIME_ATTRIBUTION.md
// § Env-var bridge for the lifecycle.
const EnvVar = "VROOLI_PROMPT_MANAGER_ATTRIBUTION"

const (
	writerSkillEnvVar = "VROOLI_PROMPT_MANAGER_WRITER_SKILL"
	writerTeamEnvVar  = "VROOLI_PROMPT_MANAGER_WRITER_TEAM"
)

var writerAttributionMu sync.Mutex

// Compatibility names delegate to the shared descriptive wire contract.
const (
	KindAgentMember    = teamattribution.KindAgentMember
	KindWriterSkill    = teamattribution.KindWriterSkill
	KindOperatorDirect = teamattribution.KindOperatorDirect
	KindExternal       = teamattribution.KindExternal
	KindLegacy         = teamattribution.KindLegacy
	KindInvestigation  = teamattribution.KindInvestigation
)

// Spawn origins share the receiver vocabulary.
const (
	SpawnOriginHeartbeat     = teamattribution.SpawnOriginHeartbeat
	SpawnOriginOperatorCLI   = teamattribution.SpawnOriginOperatorCLI
	SpawnOriginSwarmTask     = teamattribution.SpawnOriginSwarmTask
	SpawnOriginVisionWalk    = teamattribution.SpawnOriginVisionWalk
	SpawnOriginInvestigation = teamattribution.SpawnOriginInvestigation
	SpawnOriginLegacy        = teamattribution.SpawnOriginLegacy
	SpawnOriginUnknown       = teamattribution.SpawnOriginUnknown
)

// Info preserves the existing CLI name for the shared wire DTO.
type Info = teamattribution.Info

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
	if skill := strings.TrimSpace(os.Getenv(writerSkillEnvVar)); skill != "" {
		if team := strings.TrimSpace(os.Getenv(writerTeamEnvVar)); team != "" {
			return WriterSkillHeaderValue(skill, team)
		}
	}
	if v := strings.TrimSpace(os.Getenv(EnvVar)); v != "" {
		return v
	}
	return Encode(OperatorDirect())
}

// WriterSkillHeaderValue overlays writer-skill attribution on the current
// process identity. Writer skills commonly write to a destination team's
// inbox while running inside an originating team's heartbeat. The destination
// team belongs in attribution.team_id for the API URL; member and run fields
// remain the originating identity used for lineage joins.
func WriterSkillHeaderValue(sourceSkillID, targetTeamID string) string {
	raw := strings.TrimSpace(os.Getenv(EnvVar))
	if raw == "" {
		raw = Encode(OperatorDirect())
	}

	info, err := teamattribution.Parse(raw)
	if err != nil {
		return raw
	}
	info = teamattribution.WriterSkill(info, sourceSkillID, targetTeamID)
	return Encode(info)
}

// WithWriterSkill scopes the writer-skill overlay to one CLI command. The
// standard HTTP header callback reads it lazily, so generated Connect clients
// and REST requests receive the same destination-aware attribution.
func WithWriterSkill(sourceSkillID, targetTeamID string, fn func() error) error {
	if strings.TrimSpace(sourceSkillID) == "" || strings.TrimSpace(targetTeamID) == "" {
		return fmt.Errorf("writer-skill attribution requires source skill and target team")
	}
	writerAttributionMu.Lock()
	defer writerAttributionMu.Unlock()

	oldSkill, hadSkill := os.LookupEnv(writerSkillEnvVar)
	oldTeam, hadTeam := os.LookupEnv(writerTeamEnvVar)
	if err := os.Setenv(writerSkillEnvVar, sourceSkillID); err != nil {
		return fmt.Errorf("set writer skill attribution: %w", err)
	}
	if err := os.Setenv(writerTeamEnvVar, targetTeamID); err != nil {
		restoreEnv(writerSkillEnvVar, oldSkill, hadSkill)
		return fmt.Errorf("set writer team attribution: %w", err)
	}
	defer func() {
		restoreEnv(writerSkillEnvVar, oldSkill, hadSkill)
		restoreEnv(writerTeamEnvVar, oldTeam, hadTeam)
	}()
	return fn()
}

func restoreEnv(key, value string, existed bool) {
	if existed {
		_ = os.Setenv(key, value)
		return
	}
	_ = os.Unsetenv(key)
}

func stringPtr(value string) *string {
	return &value
}

// OperatorDirect returns the default attribution for a human at the
// CLI: kind=operator-direct, spawn_origin=operator-cli, no
// agent/team/run/skill context.
func OperatorDirect() Info { return teamattribution.OperatorDirect() }

// Encode marshals info to canonical JSON and base64-encodes the
// result. The returned string is suitable as the X-Vrooli-Attribution
// header value. Panics on json.Marshal failure — the Info shape is
// fully serializable, so any failure indicates a programming bug.
func Encode(info Info) string { return teamattribution.Encode(info) }

// HeaderMap returns a header map suitable for cliutil.HTTPClient's
// SetHeaderSource. Adapter helper; bare HeaderValue() is preferred
// for direct callers.
func HeaderMap() map[string]string {
	return map[string]string{HeaderName: HeaderValue()}
}
