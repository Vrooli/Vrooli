package hub

// Every decision gets a page. The operator answers on a phone with none of the
// requester's context, and a page that shows the situation (a before and
// after, the options side by side, the numbers) is what makes a decision
// answerable cold. An ask sent without one gets the Hub's page made from its
// cover (cover_page.go), so it never reaches the operator bare; that page can
// only repeat the cover, so the check still speaks. It is advice and never
// blocks: the ask is still sent.

// WarnExplainerMissing advises a page for a decision sent without one, and
// WarnPacketMissing a packet for an agent's decision sent without one.
const (
	WarnExplainerMissing = "explainer_missing"
	WarnPacketMissing    = "packet_missing"
)

// checkPacketPath advises an agent's decision (a direct ask) that names no
// packet. Whoever receives the answer starts a worker from the packet; without
// one it has to re-derive what the asker knew. A scenario's event decision is
// carried out by its own code, so it is not advised.
func checkPacketPath(spec AskSpec) []PacketWarning {
	if spec.Packet != "" || spec.SourceEventType != "" {
		return nil
	}
	return []PacketWarning{advice(WarnPacketMissing, "packet",
		"no packet: write the brief a worker needs to carry out each answer (work/<ID>/BRIEF.md, see docs/agent-system/LEVELS.md) "+
			"and send it with --packet <path>, so whoever gets the answer starts the work without re-deriving it")}
}

// checkShape returns the advice to add an explainer. It says nothing when the
// ask has the sender's own page.
func checkShape(hasExplainer bool) []PacketWarning {
	if hasExplainer {
		return nil
	}
	return []PacketWarning{advice(WarnExplainerMissing, "attachments",
		"no page of its own: the operator gets the Hub's plain page, which only repeats your question and options. "+
			"Make a real one from a starter (ls \"$VROOLI_ROOT/scenarios/notification-hub/skills/notification-hub/examples/\") that shows the situation and what happens on approval, "+
			"and send it with --explainer page.html; it replaces the Hub's page (skill section \"Make the page\")")}
}
