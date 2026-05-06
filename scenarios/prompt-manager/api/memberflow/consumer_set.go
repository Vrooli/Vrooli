// consumerSet is the single place in memberflow that knows what counts as
// "a consumer" of a topic prefix. The orphan_output rule (and, post-Phase
// 1.6, any other rule that needs to ask "does anyone read this prefix?")
// delegates the decision here.
//
// Three-pillar design context (see docs/agent-system/PRIMITIVES.md):
// Pillar 1 (declared graph) is the validator-visible view of who reads /
// writes what. The set aggregates every read-side declaration on every
// member's topics.json: `intake[]` (drain), `required_read[]`
// (heartbeat-prompt context), and `evidence_consumed[]` (decision
// rationale). Holding this aggregation in one type is the whole point —
// future consumer kinds become a one-line addition to buildConsumerSet
// rather than a scatter of conditionals across rule functions.
//
// What is NOT a consumer: `external_producers[]` documents who writes
// into a member's intake from outside the team graph; it is a
// producer-side anchor, not a read claim, and is intentionally excluded
// from the set so orphan_output's semantics stay clean.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md (declaration shapes),
// docs/agent-system/PRIMITIVES.md (three-pillar architecture).
package memberflow

import "strings"

// consumerSource names the topics.json field that contributed a consumer
// entry. Useful for diagnostic/finding context in future phases (e.g.,
// telling the operator whether a prefix is read via intake vs.
// required_read), and as a registry of every consumer kind the validator
// understands.
//
// Adding a new consumer kind: add a constant here, register a contributor
// in buildConsumerSet, and the rest of the validator picks it up
// automatically.
type consumerSource string

const (
	// consumerSourceIntake — the prefix appears on a member's `intake[]`.
	// The member drains the prefix on every heartbeat.
	consumerSourceIntake consumerSource = "intake"

	// consumerSourceRequiredRead — the prefix appears on a member's
	// `required_read[]`. The heartbeat builder renders it into the
	// agent prompt's "Required Memory" section every tick. The member
	// is not draining; it just needs the prefix's recent state visible.
	consumerSourceRequiredRead consumerSource = "required_read"

	// consumerSourceEvidence — the prefix appears on a member's
	// `evidence_consumed[]`. The member cites entries from this prefix
	// when authoring or contributing to specific decisions; the
	// for_decisions[] list on the entry names which decisions consume
	// it.
	consumerSourceEvidence consumerSource = "evidence_consumed"
)

// consumerEntry is one consumer claim on a topic prefix: a single
// (member, prefix, source) triple. The set holds many of these and
// answers Overlaps queries against the full collection.
type consumerEntry struct {
	Member MemberRef
	Prefix string
	Source consumerSource
}

// consumerSet aggregates every consumer claim across all members. The
// only public operation is Overlaps; construction and the entry shape
// are package-private so the abstraction's invariants stay local to this
// file.
//
// Zero value is a valid empty set (Overlaps returns false for any prefix).
type consumerSet struct {
	entries []consumerEntry
}

// buildConsumerSet collects consumer claims from every member's
// topics.json declarations and returns the populated set. The aggregation
// covers every read-side declaration kind: intake[] (drain),
// required_read[] (heartbeat-prompt context), and evidence_consumed[]
// (decision-rationale evidence).
//
// Members are visited in input order; within a member, entries are
// appended in this order — intake → required_read → evidence_consumed —
// matching topics.json field ordering for stable, diff-friendly
// downstream output. Empty / whitespace-only prefixes are skipped —
// shape-level Topics.Validate already rejects them, but silently skipping
// keeps this contributor robust if a caller bypasses the loader.
func buildConsumerSet(members []MemberTopics) consumerSet {
	var s consumerSet
	for _, m := range members {
		for _, in := range m.Topics.Intake {
			s.add(m.Ref, in.Prefix, consumerSourceIntake)
		}
		for _, r := range m.Topics.RequiredRead {
			s.add(m.Ref, r.Prefix, consumerSourceRequiredRead)
		}
		for _, e := range m.Topics.EvidenceConsumed {
			s.add(m.Ref, e.Prefix, consumerSourceEvidence)
		}
	}
	return s
}

// add registers one consumer claim on the set. Empty prefixes are
// silently dropped so a malformed declaration does not corrupt overlap
// results.
func (s *consumerSet) add(ref MemberRef, prefix string, source consumerSource) {
	if strings.TrimSpace(prefix) == "" {
		return
	}
	s.entries = append(s.entries, consumerEntry{
		Member: ref,
		Prefix: prefix,
		Source: source,
	})
}

// Overlaps reports whether any registered consumer claim covers the
// supplied output prefix, using memberflow.Overlap semantics
// (wildcard-aware: "foo/*" overlaps "foo/bar", and vice versa).
//
// An empty `prefix` argument never overlaps anything; callers should
// have rejected that at shape-validation time, but the guard keeps this
// method total.
func (s consumerSet) Overlaps(prefix string) bool {
	if strings.TrimSpace(prefix) == "" {
		return false
	}
	for _, e := range s.entries {
		if Overlap(e.Prefix, prefix) {
			return true
		}
	}
	return false
}
