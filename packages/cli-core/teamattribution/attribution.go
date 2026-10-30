// Package teamattribution owns the descriptive team-knowledge header wire
// contract. These fields describe a writer; they never confer authorization.
// Semantics: docs/agent-system/RUNTIME_ATTRIBUTION.md.
package teamattribution

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

const HeaderName = "X-Vrooli-Attribution"
const (
	KindAgentMember          = "agent-member"
	KindWriterSkill          = "writer-skill"
	KindOperatorDirect       = "operator-direct"
	KindExternal             = "external"
	KindLegacy               = "legacy"
	KindInvestigation        = "investigation"
	SpawnOriginConversation  = "conversation"
	SpawnOriginHeartbeat     = "heartbeat"
	SpawnOriginOperatorCLI   = "operator-cli"
	SpawnOriginSwarmTask     = "swarm-task"
	SpawnOriginVisionWalk    = "vision-walk"
	SpawnOriginInvestigation = "investigation"
	SpawnOriginLegacy        = "legacy"
	SpawnOriginUnknown       = "unknown"
)

var Kinds = []string{KindAgentMember, KindWriterSkill, KindOperatorDirect, KindExternal, KindLegacy, KindInvestigation}
var SpawnOrigins = []string{SpawnOriginConversation, SpawnOriginHeartbeat, SpawnOriginOperatorCLI, SpawnOriginSwarmTask, SpawnOriginVisionWalk, SpawnOriginInvestigation, SpawnOriginLegacy, SpawnOriginUnknown}
var ErrRequired = fmt.Errorf("X-Vrooli-Attribution header is required for mutating writes (canon: docs/agent-system/RUNTIME_ATTRIBUTION.md)")

type Info struct {
	// Kind is the category of writer; required, closed vocabulary
	// (KnowledgeKind*). Unknown values are rejected by the API at write
	// time and surfaced as attribution_malformed by the validator.
	Kind string `json:"kind"`
	// MemberID is the team-scoped member id (matches
	// store/teams/<team>/members/<id>/) for agent-member and writer-skill
	// writes. Nil for operator-direct, external, legacy. Marshals as null.
	MemberID *string `json:"member_id"`
	// TeamID is the team id this write is attributed to; the API handler
	// rejects mismatches against the URL path's team id (HTTP 400
	// team_mismatch). Nil for operator-direct when the operator did not
	// specify a team scope. Marshals as null.
	TeamID *string `json:"team_id"`
	// RunID is the agent-manager run UUID this write is attributed to —
	// allows joining a write to its run lineage. Required for
	// agent-member and investigation; optional for writer-skill (when
	// invoked from a run); nil for operator-direct, external, legacy.
	// Marshals as null.
	RunID *string `json:"run_id"`
	// SpawnOrigin is informational — how the writer process was started;
	// required, closed vocabulary (SpawnOrigin*). Used by debt-curator
	// to spot patterns ("most undeclared writes come from vision-walk
	// spawns"); never gates writes.
	SpawnOrigin string `json:"spawn_origin"`
	// SourceSkillID names the skill that performed or mediated the
	// write. Required for writer-skill; optional for others (when a
	// skill mediated the write). Marshals as null.
	SourceSkillID *string `json:"source_skill_id"`
}

func Parse(raw string) (Info, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Info{}, ErrRequired
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return Info{}, fmt.Errorf("attribution: invalid base64: %w", err)
	}
	var info Info
	if err := json.Unmarshal(decoded, &info); err != nil {
		return Info{}, fmt.Errorf("attribution: invalid JSON: %w", err)
	}
	return info, nil
}

func Validate(info Info, urlTeamID string) error {
	if !IsKnownKind(info.Kind) {
		return fmt.Errorf("attribution: unknown kind %q (allowed: %s)", info.Kind, strings.Join(Kinds, ", "))
	}
	if info.Kind == KindLegacy {
		// `legacy` is produced exclusively by the migration tool;
		// clients cannot write legacy entries post-cutoff.
		return fmt.Errorf("attribution: kind %q is reserved for the migration tool, not for live writes", KindLegacy)
	}
	if !IsKnownSpawnOrigin(info.SpawnOrigin) {
		return fmt.Errorf("attribution: unknown spawn_origin %q (allowed: %s)", info.SpawnOrigin, strings.Join(SpawnOrigins, ", "))
	}

	// Per-kind required-field rules.
	switch info.Kind {
	case KindAgentMember:
		if info.MemberID == nil || strings.TrimSpace(*info.MemberID) == "" {
			return fmt.Errorf("attribution: kind=agent-member requires member_id")
		}
		if info.TeamID == nil || strings.TrimSpace(*info.TeamID) == "" {
			return fmt.Errorf("attribution: kind=agent-member requires team_id")
		}
		// run_id is required for agent-member EXCEPT for heartbeat or conversation spawns:
		// at heartbeat-spawn time prompt-manager constructs attribution before
		// agent-manager assigns the run UUID (Environment is fixed at
		// CreateRunRequest construction; the run_id only exists in the
		// CreateRunResponse). Permitting null run_id for spawn_origin=heartbeat
		// resolves that chicken-and-egg; future strengthening will overlay
		// run_id from VROOLI_AGENT_IDENTITY_TOKEN claims at request time.
		// Canon: docs/agent-system/RUNTIME_ATTRIBUTION.md § Env-var bridge.
		if info.SpawnOrigin != SpawnOriginHeartbeat && info.SpawnOrigin != SpawnOriginConversation {
			if info.RunID == nil || strings.TrimSpace(*info.RunID) == "" {
				return fmt.Errorf("attribution: kind=agent-member with spawn_origin=%q requires run_id", info.SpawnOrigin)
			}
		}
	case KindWriterSkill:
		if info.SourceSkillID == nil || strings.TrimSpace(*info.SourceSkillID) == "" {
			return fmt.Errorf("attribution: kind=writer-skill requires source_skill_id")
		}
		if info.TeamID == nil || strings.TrimSpace(*info.TeamID) == "" {
			return fmt.Errorf("attribution: kind=writer-skill requires team_id (the target team)")
		}
	case KindInvestigation:
		if info.RunID == nil || strings.TrimSpace(*info.RunID) == "" {
			return fmt.Errorf("attribution: kind=investigation requires run_id")
		}
	}

	// Cross-check: if attribution.team_id is set, it must match the
	// URL team. Per RUNTIME_ATTRIBUTION.md § Conflict policy, the
	// server never reconciles — only accepts or rejects.
	if info.TeamID != nil && strings.TrimSpace(*info.TeamID) != "" {
		if *info.TeamID != urlTeamID {
			return fmt.Errorf("attribution: team_mismatch — header team_id=%q, URL team_id=%q", *info.TeamID, urlTeamID)
		}
	}
	return nil
}

func IsKnownKind(kind string) bool          { return slices.Contains(Kinds, kind) }
func IsKnownSpawnOrigin(origin string) bool { return slices.Contains(SpawnOrigins, origin) }
func OperatorDirect() Info {
	return Info{Kind: KindOperatorDirect, SpawnOrigin: SpawnOriginOperatorCLI}
}
func WriterSkill(info Info, skill, team string) Info {
	info.Kind = KindWriterSkill
	skill = strings.TrimSpace(skill)
	team = strings.TrimSpace(team)
	info.SourceSkillID = &skill
	info.TeamID = &team
	return info
}
func Encode(info Info) string {
	payload, err := json.Marshal(info)
	if err != nil {
		panic(fmt.Sprintf("attribution: marshal Info: %v", err))
	}
	return base64.StdEncoding.EncodeToString(payload)
}
