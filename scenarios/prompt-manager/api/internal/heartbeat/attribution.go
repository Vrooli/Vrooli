package heartbeat

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/teamattribution"
	"prompt-manager/internal/store"
)

// The shared descriptive header contract is separate from caller authority.
const attributionHeaderName = teamattribution.HeaderName

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
	return teamattribution.Parse(raw)
}

// errAttributionRequired is returned by parseAttributionHeader when
// the header is missing. Callers branch on it to produce a more
// actionable error message than a generic parse failure.
var errAttributionRequired = teamattribution.ErrRequired

// validateAttribution enforces the structural and per-kind rules
// from RUNTIME_ATTRIBUTION.md § structured-attribution payload and
// § Conflict policy. urlTeamID is the team id from the request path
// (mux.Vars["id"]); used for the team_mismatch check.
//
// Returns nil if the attribution is well-formed and consistent with
// the URL; otherwise an error suitable for an HTTP 400 response.
func validateAttribution(info store.AttributionInfo, urlTeamID string) error {
	return teamattribution.Validate(info, urlTeamID)
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
func isKnownKind(kind string) bool { return teamattribution.IsKnownKind(kind) }

// isKnownSpawnOrigin returns true if o is a member of store.SpawnOrigins.
func isKnownSpawnOrigin(origin string) bool { return teamattribution.IsKnownSpawnOrigin(origin) }

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
