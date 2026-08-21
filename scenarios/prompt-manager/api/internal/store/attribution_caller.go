package store

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// AttributionHeaderName is the transport header carrying the structured
// attribution payload (canon: docs/agent-system/RUNTIME_ATTRIBUTION.md).
const AttributionHeaderName = "X-Vrooli-Attribution"

// AgentIdentityHeaderName carries the opaque agent-manager identity token.
// Presence proves the caller was spawned by agent-manager; the token itself is
// verified by agent-manager, not here.
const AgentIdentityHeaderName = "X-Agent-Identity-Token"

// CallerAttribution is the derived, best-effort answer to "who made this
// request". Kind is the closed-vocabulary attribution kind on its own; Caller
// is the display form that also carries the member or source-skill id.
//
// Kind is the field that separates demand from audit traffic. A skill read by
// an agent-member doing its lane's work is demand; the same skill read by an
// optimizer auditing it is not, and counting them together makes a selection
// ladder keyed on usage reinforce its own choices.
type CallerAttribution struct {
	Caller     string
	Kind       string
	MemberID   string
	TeamID     string
	RunID      string
	HasAgentID bool
}

// CallerFromRequest derives caller attribution from the structured attribution
// header when present and decodable, and falls back to the User-Agent so a
// caller that predates attribution still lands in a distinguishable bucket.
//
// This mirrors the derivation the discovery telemetry already applies, so the
// discovery-call log and the skill-read log join on equal terms.
func CallerFromRequest(r *http.Request) CallerAttribution {
	out := CallerAttribution{}
	if r == nil {
		return out
	}
	out.HasAgentID = strings.TrimSpace(r.Header.Get(AgentIdentityHeaderName)) != ""

	if raw := strings.TrimSpace(r.Header.Get(AttributionHeaderName)); raw != "" {
		if decoded, err := base64.StdEncoding.DecodeString(raw); err == nil {
			var info AttributionInfo
			if json.Unmarshal(decoded, &info) == nil && info.Kind != "" {
				out.Kind = info.Kind
				caller := info.Kind
				if info.MemberID != nil && *info.MemberID != "" {
					out.MemberID = *info.MemberID
					caller += "/" + *info.MemberID
				} else if info.SourceSkillID != nil && *info.SourceSkillID != "" {
					caller += "/" + *info.SourceSkillID
				}
				if info.TeamID != nil {
					out.TeamID = *info.TeamID
				}
				if info.RunID != nil {
					out.RunID = *info.RunID
				}
				out.Caller = caller
				return out
			}
		}
		out.Caller = "attribution"
		return out
	}

	if ua := strings.TrimSpace(r.UserAgent()); ua != "" {
		out.Caller = ua
	}
	return out
}
