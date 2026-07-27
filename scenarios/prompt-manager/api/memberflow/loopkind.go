// loopkind.go holds every rule that depends on a member's declared
// LoopKind. They are grouped here rather than scattered through
// validation.go because they share one generating idea: a loop must be
// able to say where its memory lives between heartbeats, and the only
// kind whose memory has no natural home is a sweep.
//
// Adoption order matters and is encoded in the severities. The declare
// rules (loop_kind_missing, loop_kind_intake_mismatch) come first; the
// enforce rules (sweep_without_ledger, ledger_shape_invalid) are gated
// behind an explicit LoopKind, so they cannot fire on a member that has
// not declared one. That predicate — not a release sequence — is what
// keeps a team from being failed for a gap it cannot yet express.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md § Loop kinds.
package memberflow

import (
	"fmt"
	"strings"
)

// ledgerPrefixMarker identifies the coverage-ledger topic family by its
// canonical naming shape, `<subject>-visited/<id>`. Registered in
// docs/agent-system/TOPICS.md § Topic families.
const ledgerPrefixMarker = "-visited/"

// isLedgerPrefix reports whether a prefix follows the coverage-ledger
// naming shape.
func isLedgerPrefix(prefix string) bool {
	return strings.Contains(prefix, ledgerPrefixMarker)
}

// selfReadPrefixes returns the member's own read-side prefixes, across
// every read declaration kind. A coverage ledger is defined by the writer
// also reading it, so this is the set the ledger rules test against.
func selfReadPrefixes(t Topics) []string {
	var out []string
	for _, in := range t.Intake {
		out = append(out, in.Prefix)
	}
	for _, r := range t.RequiredRead {
		out = append(out, r.Prefix)
	}
	for _, e := range t.EvidenceConsumed {
		out = append(out, e.Prefix)
	}
	return out
}

// declaredLedgers returns the member's output prefixes that form a valid
// coverage ledger: written by this member and read back by the same
// member. Self-reference is the defining property, not an accident.
func declaredLedgers(t Topics) []string {
	reads := selfReadPrefixes(t)
	var out []string
	for _, o := range t.Output {
		if o.DestinationKind != DestinationKnowledge {
			continue
		}
		for _, r := range reads {
			if Overlap(r, o.Prefix) {
				out = append(out, o.Prefix)
				break
			}
		}
	}
	return out
}

// ruleLoopKindMissing — the member declares no loop kind.
//
// Severity is warning during adoption. A member cannot be checked for the
// obligations its loop shape implies until it states that shape, and the
// value is a judgment call owned by the member's team — inferring 27 of
// them and calling the result declared is how a declaration layer becomes
// fiction. Promote to error once every member carries a value.
func ruleLoopKindMissing(members []MemberTopics) []Finding {
	var out []Finding
	for _, m := range members {
		if !m.Exists || m.Topics.IsEmpty() && m.Topics.LoopKind == "" {
			// No topics.json at all, or a positively-empty
			// declaration: other rules own that case.
			continue
		}
		if m.Topics.LoopKind != "" {
			continue
		}
		out = append(out, Finding{
			Rule:     "loop_kind_missing",
			Severity: SeverityWarning,
			Member:   m.Ref,
			Detail:   "member declares no loop_kind; the loop's memory location is unstated, so ledger obligations cannot be checked (queue | reactive | sweep | generative)",
		})
	}
	return out
}

// ruleLoopKindInvalid — the declared loop kind is not a known value.
func ruleLoopKindInvalid(members []MemberTopics) []Finding {
	var out []Finding
	for _, m := range members {
		k := m.Topics.LoopKind
		if k == "" || k.Valid() {
			continue
		}
		out = append(out, Finding{
			Rule:     "loop_kind_invalid",
			Severity: SeverityError,
			Member:   m.Ref,
			Detail:   fmt.Sprintf("loop_kind %q is not one of queue | reactive | sweep | generative", string(k)),
		})
	}
	return out
}

// ruleLoopKindIntakeMismatch — the member drains an intake but does not
// declare itself queue-driven.
//
// This is the rule that makes the new field self-checking: it tests the
// declaration against data that already exists, so loop_kind cannot
// silently drift from the flow it claims to describe. A member that
// drains a topic keeps its memory in that topic's unrouted set by
// construction; any other loop_kind is a contradiction rather than a
// preference.
func ruleLoopKindIntakeMismatch(members []MemberTopics) []Finding {
	var out []Finding
	for _, m := range members {
		if len(m.Topics.Intake) == 0 || m.Topics.LoopKind == "" {
			continue
		}
		if m.Topics.LoopKind == LoopQueue {
			continue
		}
		out = append(out, Finding{
			Rule:     "loop_kind_intake_mismatch",
			Severity: SeverityError,
			Member:   m.Ref,
			Detail: fmt.Sprintf(
				"member declares %d intake prefix(es) but loop_kind is %q; a member that drains an intake is queue-driven — its memory is the unrouted set",
				len(m.Topics.Intake), string(m.Topics.LoopKind)),
		})
	}
	return out
}

// ruleSweepWithoutLedger — a sweep declares no coverage ledger.
//
// A sweep iterates a standing population that never empties. Nothing
// marks a target done, so with no ledger the member re-picks the head of
// its priority ladder every heartbeat and never reaches the tail. The
// ledger is the output prefix the member also reads back.
func ruleSweepWithoutLedger(members []MemberTopics) []Finding {
	var out []Finding
	for _, m := range members {
		if m.Topics.LoopKind != LoopSweep {
			continue
		}
		if len(declaredLedgers(m.Topics)) > 0 {
			continue
		}
		out = append(out, Finding{
			Rule:     "sweep_without_ledger",
			Severity: SeverityError,
			Member:   m.Ref,
			Detail:   "loop_kind is sweep but no coverage ledger is declared; add a `<subject>-visited/<id>` prefix to both output[] and required_read[] so the loop remembers which targets it has reached",
		})
	}
	return out
}

// ruleLedgerShapeInvalid — a ledger-shaped output is not read back by its
// writer.
//
// A `<subject>-visited/*` output that nobody reads is a write-only record:
// it looks like coverage memory and functions as none. The rule is
// deliberately independent of loop_kind so a mis-shaped ledger is caught
// even on a member whose loop kind is still undeclared.
func ruleLedgerShapeInvalid(members []MemberTopics) []Finding {
	var out []Finding
	for _, m := range members {
		reads := selfReadPrefixes(m.Topics)
		for _, o := range m.Topics.Output {
			if !isLedgerPrefix(o.Prefix) {
				continue
			}
			read := false
			for _, r := range reads {
				if Overlap(r, o.Prefix) {
					read = true
					break
				}
			}
			if read {
				continue
			}
			out = append(out, Finding{
				Rule:     "ledger_shape_invalid",
				Severity: SeverityError,
				Member:   m.Ref,
				Prefix:   o.Prefix,
				Detail:   fmt.Sprintf("ledger output %q is not read back by its writer; a coverage ledger the member never reads cannot inform target selection", o.Prefix),
			})
		}
	}
	return out
}

// nonBlank drops empty and whitespace-only entries so a declaration of
// `["", " "]` does not read as a named population.
func nonBlank(list []string) []string {
	var out []string
	for _, s := range list {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// ruleSweepPopulationMissing — a sweep does not name what it iterates.
//
// Warning, not error: a sweep without a declared population is correct,
// just unmeasurable. Naming a graph-resident population is what turns
// ledger entries into a coverage figure (visited over total). Populations
// that are not graph-resident — a time window over recent runs, say —
// legitimately omit it, which is why this never hardens to an error.
func ruleSweepPopulationMissing(members []MemberTopics) []Finding {
	var out []Finding
	for _, m := range members {
		if m.Topics.LoopKind != LoopSweep {
			continue
		}
		if len(nonBlank(m.Topics.Population)) > 0 {
			continue
		}
		out = append(out, Finding{
			Rule:     "sweep_population_missing",
			Severity: SeverityWarning,
			Member:   m.Ref,
			Detail:   "loop_kind is sweep but no population is named; coverage stays unmeasurable (declare a graph node type, or omit deliberately when the population is not graph-resident)",
		})
	}
	return out
}

// A self-read output is NOT by itself a defect. Drafting this file, a
// `self_reference_not_ledger` warning was written on the theory that an output
// escaping orphan_output via its own writer's read was laundering orphan
// status. Run against the real store it produced 18 findings, and the pattern
// it flagged turned out to be legitimate almost everywhere: `outcome-target-record`,
// `goal-portfolio-record`, and `vision-walk-record` are members re-reading their
// own durable records for continuity between heartbeats. That is a third valid
// shape alongside ledger and orphan, so the rule was retracted rather than
// shipped as alarm noise. `ledger_shape_invalid` already covers the real defect
// (a ledger-shaped output nobody reads back), and `orphan_output`'s detail text
// was corrected — it claimed "no peer-member consumer" while its consumer set
// counted the writer itself.
