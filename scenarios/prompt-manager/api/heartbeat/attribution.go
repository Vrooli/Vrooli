package heartbeat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"prompt-manager/store"
	"strings"
)

// attributionHeaderName is the HTTP header carrying structured
// attribution from CLI / SDK / UI clients to prompt-manager. Mirrored
// in scenarios/prompt-manager/cli/internal/attribution/attribution.go
// (HeaderName) and documented in docs/agent-system/RUNTIME_ATTRIBUTION.md
// § HTTP header.
const attributionHeaderName = "X-Vrooli-Attribution"

// callerNoteMaxLen caps the freeform CallerNote field per the canon
// (RUNTIME_ATTRIBUTION.md § Optional caller_note). Over-cap requests
// return HTTP 400.
const callerNoteMaxLen = 256

// runIDDisplayPrefix is the number of leading characters of a run UUID
// shown in the derived `investigation:<run_id-prefix>` caller. 8 is
// the canonical short-uuid length used elsewhere in the codebase.
const runIDDisplayPrefix = 8

// parseAttributionHeader decodes the X-Vrooli-Attribution header
// payload (base64-JSON) into a store.AttributionInfo. Errors carry
// HTTP-friendly messages suitable for inclusion in a 400 response.
//
// Returns ErrAttributionRequired when the header is empty so callers
// can distinguish missing-header from malformed-header cases (the
// canon treats both as 400 but the error message differs).
func parseAttributionHeader(raw string) (store.AttributionInfo, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return store.AttributionInfo{}, errAttributionRequired
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return store.AttributionInfo{}, fmt.Errorf("attribution: invalid base64: %w", err)
	}
	var info store.AttributionInfo
	if err := json.Unmarshal(decoded, &info); err != nil {
		return store.AttributionInfo{}, fmt.Errorf("attribution: invalid JSON: %w", err)
	}
	return info, nil
}

// errAttributionRequired is returned by parseAttributionHeader when
// the header is missing. Callers branch on it to produce a more
// actionable error message than a generic parse failure.
var errAttributionRequired = fmt.Errorf("X-Vrooli-Attribution header is required for mutating writes (canon: docs/agent-system/RUNTIME_ATTRIBUTION.md)")

// validateAttribution enforces the structural and per-kind rules
// from RUNTIME_ATTRIBUTION.md § structured-attribution payload and
// § Conflict policy. urlTeamID is the team id from the request path
// (mux.Vars["id"]); used for the team_mismatch check.
//
// Returns nil if the attribution is well-formed and consistent with
// the URL; otherwise an error suitable for an HTTP 400 response.
func validateAttribution(info store.AttributionInfo, urlTeamID string) error {
	if !isKnownKind(info.Kind) {
		return fmt.Errorf("attribution: unknown kind %q (allowed: %s)", info.Kind, strings.Join(store.KnowledgeKinds, ", "))
	}
	if info.Kind == store.KnowledgeKindLegacy {
		// `legacy` is produced exclusively by the migration tool;
		// clients cannot write legacy entries post-cutoff.
		return fmt.Errorf("attribution: kind %q is reserved for the migration tool, not for live writes", store.KnowledgeKindLegacy)
	}
	if !isKnownSpawnOrigin(info.SpawnOrigin) {
		return fmt.Errorf("attribution: unknown spawn_origin %q (allowed: %s)", info.SpawnOrigin, strings.Join(store.SpawnOrigins, ", "))
	}

	// Per-kind required-field rules.
	switch info.Kind {
	case store.KnowledgeKindAgentMember:
		if info.MemberID == nil || strings.TrimSpace(*info.MemberID) == "" {
			return fmt.Errorf("attribution: kind=agent-member requires member_id")
		}
		if info.TeamID == nil || strings.TrimSpace(*info.TeamID) == "" {
			return fmt.Errorf("attribution: kind=agent-member requires team_id")
		}
		// run_id is required for agent-member EXCEPT when spawn_origin=heartbeat:
		// at heartbeat-spawn time prompt-manager constructs attribution before
		// agent-manager assigns the run UUID (Environment is fixed at
		// CreateRunRequest construction; the run_id only exists in the
		// CreateRunResponse). Permitting null run_id for spawn_origin=heartbeat
		// resolves that chicken-and-egg; future strengthening will overlay
		// run_id from VROOLI_AGENT_IDENTITY_TOKEN claims at request time.
		// Canon: docs/agent-system/RUNTIME_ATTRIBUTION.md § Env-var bridge.
		if info.SpawnOrigin != store.SpawnOriginHeartbeat {
			if info.RunID == nil || strings.TrimSpace(*info.RunID) == "" {
				return fmt.Errorf("attribution: kind=agent-member with spawn_origin=%q requires run_id", info.SpawnOrigin)
			}
		}
	case store.KnowledgeKindWriterSkill:
		if info.SourceSkillID == nil || strings.TrimSpace(*info.SourceSkillID) == "" {
			return fmt.Errorf("attribution: kind=writer-skill requires source_skill_id")
		}
		if info.TeamID == nil || strings.TrimSpace(*info.TeamID) == "" {
			return fmt.Errorf("attribution: kind=writer-skill requires team_id (the target team)")
		}
	case store.KnowledgeKindInvestigation:
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

// deriveCaller computes the human-readable caller string from
// validated attribution per RUNTIME_ATTRIBUTION.md § Derived display.
// urlTeamID is used only as a fallback when info.TeamID is nil
// (writer-skill etc. always has team_id; agent-member always has it).
//
// validateAttribution must run before deriveCaller — derivations
// assume per-kind required fields are populated.
func deriveCaller(info store.AttributionInfo, urlTeamID string) string {
	switch info.Kind {
	case store.KnowledgeKindAgentMember:
		return fmt.Sprintf("%s/%s", derefStr(info.TeamID, urlTeamID), derefStr(info.MemberID, ""))
	case store.KnowledgeKindWriterSkill:
		return fmt.Sprintf("skill:%s", derefStr(info.SourceSkillID, ""))
	case store.KnowledgeKindOperatorDirect:
		return "operator"
	case store.KnowledgeKindExternal:
		return "external"
	case store.KnowledgeKindInvestigation:
		runID := derefStr(info.RunID, "")
		if len(runID) > runIDDisplayPrefix {
			runID = runID[:runIDDisplayPrefix]
		}
		return fmt.Sprintf("investigation:%s", runID)
	case store.KnowledgeKindLegacy:
		// Live handlers reject `legacy` in validateAttribution; this
		// branch covers only the migration path's read-back display.
		return "legacy"
	default:
		return info.Kind
	}
}

// isKnownKind returns true if k is a member of store.KnowledgeKinds.
func isKnownKind(k string) bool {
	for _, allowed := range store.KnowledgeKinds {
		if k == allowed {
			return true
		}
	}
	return false
}

// isKnownSpawnOrigin returns true if o is a member of store.SpawnOrigins.
func isKnownSpawnOrigin(o string) bool {
	for _, allowed := range store.SpawnOrigins {
		if o == allowed {
			return true
		}
	}
	return false
}

// derefStr dereferences a string pointer, returning fallback when nil
// or empty.
func derefStr(p *string, fallback string) string {
	if p == nil {
		return fallback
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return fallback
	}
	return v
}
