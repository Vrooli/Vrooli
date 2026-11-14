package skillreads

// Read is one record of a skill being served through POST /skills/read.
//
// Why the server records this rather than the CLI: `skill read` resolves many
// identifiers in one call, and non-CLI callers (UI, MCP, other scenarios) reach
// the same endpoint. Recording at the handler counts every consumer exactly
// once per resolved skill.
type Read struct {
	ID      string `json:"id"`
	At      string `json:"at"` // RFC3339
	SkillID string `json:"skillId"`
	// Caller is the same best-effort attribution string discovery records, so
	// the two logs join on equal terms.
	Caller string `json:"caller,omitempty"`
	// CallerKind is the attribution kind alone (agent-member, operator-direct,
	// writer-skill, investigation) without the member or skill suffix. It is
	// what separates demand from audit traffic: an optimizer reading a skill to
	// audit it must not inflate that skill's demand signal.
	CallerKind string `json:"callerKind,omitempty"`
	// AgentRunID is the agent-manager run this read belongs to, when the caller
	// presented an identity token. Empty for operator sessions.
	AgentRunID string `json:"agentRunId,omitempty"`
	// ViaDiscovery is true when a discover call by the same caller returned this
	// skill shortly before the read. It converts "discovery offered it" into
	// "discovery offered it and the agent took it".
	ViaDiscovery bool `json:"viaDiscovery,omitempty"`
	// DiscoveryCallID names the call that surfaced it, when ViaDiscovery.
	DiscoveryCallID string `json:"discoveryCallId,omitempty"`
}
