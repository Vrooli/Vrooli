// Validator rule: topic_key_prefix_mismatch.
//
// Cross-checks every knowledge entry's `topic` field against the set of
// topic prefixes declared in `topics.json` (intake[].prefix and
// output[].prefix) across all members of a team. Surfaces entries whose
// topic key isn't covered by any declared prefix, which means either:
//  1. The producer wrote to a prefix it didn't declare in topics.json
//     (declaration drift), or
//  2. The prefix is a system / one-off topic that should be added to a
//     member's declarations or accepted as a documented exception.
//
// The rule is severity=warning, never error — a real entry under an
// undeclared prefix is data, not a structural fault that breaks the system.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md
package memberflow

import (
	"fmt"
	"sort"
	"strings"
)

// EnrichWithKeyPrefixMismatch finds knowledge entries whose topic key does
// not match any declared topic prefix on the entry's team. Returns an empty
// slice when q is nil — callers without a knowledge backend get no findings,
// matching the convention used by EnrichWithDrainStatus.
//
// Match semantics: an entry topic `T` matches a declared prefix `P` (with any
// trailing `/*` stripped) when `T == P` or `T` starts with `P + "/"`. This
// matches the same semantics the existing topics_knowledge_query.go uses.
func EnrichWithKeyPrefixMismatch(members []MemberTopics, q KnowledgeQuery) []Finding {
	if q == nil {
		return nil
	}

	// Index declared prefixes per team so each entry only checks against
	// its own team's declarations.
	prefixesByTeam := make(map[string]map[string]struct{})
	for _, m := range members {
		team := m.Ref.Team
		if _, ok := prefixesByTeam[team]; !ok {
			prefixesByTeam[team] = make(map[string]struct{})
		}
		for _, in := range m.Topics.Intake {
			prefixesByTeam[team][stripPrefixWildcard(in.Prefix)] = struct{}{}
		}
		for _, out := range m.Topics.Output {
			prefixesByTeam[team][stripPrefixWildcard(out.Prefix)] = struct{}{}
		}
	}

	teams := make([]string, 0, len(prefixesByTeam))
	for team := range prefixesByTeam {
		teams = append(teams, team)
	}
	sort.Strings(teams)

	var findings []Finding
	for _, team := range teams {
		prefixes := prefixesByTeam[team]
		entries, err := q.ListAll(team)
		if err != nil {
			findings = append(findings, Finding{
				Rule:     "topic_key_query_unavailable",
				Severity: SeverityWarning,
				Team:     team,
				Detail:   fmt.Sprintf("ListAll(%q) error: %v", team, err),
			})
			continue
		}
		// Grouped per topic family: one undeclared family produces an entry
		// per write, so a per-entry count reports how long the team has been
		// running rather than how many prefixes need declaring.
		grouper := &findingGrouper{}
		for _, e := range entries {
			if topicMatchesAnyPrefix(e.Topic, prefixes) {
				continue
			}
			grouper.Add(e.Topic, e.ID)
		}
		findings = append(findings, grouper.Emit(func(family, evidence string) Finding {
			return Finding{
				Rule:     "topic_key_prefix_mismatch",
				Severity: SeverityWarning,
				Team:     team,
				Prefix:   family,
				Detail:   fmt.Sprintf("%s under %q have no matching declared prefix in any member's topics.json on team %q", evidence, family, team),
			}
		})...)
	}
	return findings
}

func stripPrefixWildcard(p string) string {
	return strings.TrimSuffix(p, "/*")
}

// topicMatchesAnyPrefix reports whether a live topic key is covered by any
// declared prefix. Matching is segment-wise so that a `<name>` placeholder in
// a declaration covers one real segment; without that, a member declaring
// `review-evidence/<work-item-id>` appears to have declared nothing and every
// entry it writes is reported as undeclared drift.
func topicMatchesAnyPrefix(topic string, prefixes map[string]struct{}) bool {
	topicSegs := strings.Split(topic, "/")
	for p := range prefixes {
		if p == "" {
			continue
		}
		prefixSegs := strings.Split(p, "/")
		if len(prefixSegs) > len(topicSegs) {
			continue
		}
		matched := true
		for i, seg := range prefixSegs {
			if !topicSegmentsUnify(seg, topicSegs[i]) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
