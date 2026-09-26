package memberflow

import (
	"fmt"
	"sort"
	"strings"
)

// Generation of the `## Topic Catalog` table.
//
// Generating the graph alone is not enough. The table beside it restates the
// same topic set by hand, so the moment the graph becomes complete the table is
// stale and graph_topic_catalog_drift fires. That rule compares document
// against document, so it survives graph generation and is not one of the rules
// the graph work retires. Both surfaces have to come from the declarations in
// the same pass, or the two simply trade places: today the graph drifts from
// the declarations; generating only the graph makes the table drift from it.
//
// Status and purpose are authored per topic family in team.json::topicCatalog
// and are carried through unchanged. Writers, readers, and the topic set are
// derived.

// GenerateTopicCatalogTable renders the Markdown table from the same
// relationships the graph generator reads.
func GenerateTopicCatalogTable(in GenerateOperatingGraphInput) string {
	rels := BuildRuntimeOperatingRelationships(in.Runtime, in.TeamID)
	registry := DefaultOperatingRelationshipRegistry()

	writers := map[string]map[string]bool{}
	readers := map[string]map[string]bool{}
	add := func(set map[string]map[string]bool, topic, member string) {
		if topic == "" || member == "" {
			return
		}
		if set[topic] == nil {
			set[topic] = map[string]bool{}
		}
		set[topic][member] = true
	}
	for _, rel := range rels {
		if rel.Topic == "" || rel.Member == "" {
			continue
		}
		spec, ok := registry.Spec(registry.GraphKindForRuntime(rel.Kind))
		if !ok {
			continue
		}
		// A member -> topic shape is a write; topic -> member is a read.
		switch {
		case spec.GraphShape.FromKind == OperatingGraphNodeKindMember && spec.GraphShape.ToKind == OperatingGraphNodeKindTopic:
			add(writers, rel.Topic, rel.Member)
		case spec.GraphShape.FromKind == OperatingGraphNodeKindTopic && spec.GraphShape.ToKind == OperatingGraphNodeKindMember:
			add(readers, rel.Topic, rel.Member)
		}
	}

	topics := map[string]bool{}
	for topic := range writers {
		topics[topic] = true
	}
	for topic := range readers {
		topics[topic] = true
	}

	authored := topicCatalogByQualifiedTopic(in.Runtime.Contracts[in.TeamID])

	// The rows ARE the authored catalog: team.json::topicCatalog is the team's
	// own list of topic families, one entry each, with the status and purpose
	// only a human can write. Deriving the row set from declared prefixes
	// instead produces a row per SPELLING, and members spell one family several
	// ways. Writers and readers attach by prefix overlap, so a member declaring
	// the broad `friction-report/*` is credited on every friction-report row.
	// One row per topic the GRAPH draws, so the table and the diagram cover the
	// same set. Sourcing rows from team.json::topicCatalog instead leaves every
	// alternate spelling of a family drawn but undocumented, which the coverage
	// counter reports as graph-only topics even when no rule fires. Status and
	// purpose still come from the authored catalog, matched by prefix overlap,
	// so several spellings of one family share the one authored description.
	ordered := make([]string, 0, len(topics))
	for topic := range topics {
		ordered = append(ordered, topic)
	}
	sort.Strings(ordered)

	overlapping := func(set map[string]map[string]bool, row string) map[string]bool {
		out := map[string]bool{}
		for topic, members := range set {
			if topic == row || topicsOverlap(topic, row) {
				for member := range members {
					out[member] = true
				}
			}
		}
		return out
	}

	// Match a declared topic to its authored catalog entry by FAMILY, not by
	// exact string.
	//
	// The two files spell the same family differently and inconsistently:
	// topics.json may declare `action-audit/*` while team.json::topicCatalog
	// authors `action-audit/YYYY-MM-DD`, and which convention a team uses
	// varies per team and per file. An exact-key lookup misses every such pair
	// and reports the purpose as absent when it is authored and correct.
	// Reuse the rules' own family-aware lookup, so the generated table and the
	// validator agree by construction rather than by two similar helpers
	// drifting apart.
	lookup := func(topic string) (TopicCatalogEntry, bool) {
		return catalogEntryForTopic(authored, "", topic)
	}

	var b strings.Builder
	b.WriteString("| Topic family | Status | Owner / primary writer | Primary readers | Purpose |\n")
	b.WriteString("|---|---|---|---|---|\n")
	for _, topic := range ordered {
		entry, hasEntry := lookup(topic)
		status := strings.TrimSpace(entry.Status)
		if !hasEntry || status == "" {
			status = "live"
		}
		fmt.Fprintf(&b, "| `topic:%s` | %s | %s | %s | %s |\n",
			topic, status,
			joinSortedActors(overlapping(writers, topic)),
			joinSortedActors(overlapping(readers, topic)),
			strings.TrimSpace(entry.Purpose))
	}
	return b.String()
}

func joinSortedActors(set map[string]bool) string {
	if len(set) == 0 {
		// Empty rather than a dash: the actor resolver treats any non-empty
		// cell as a reference to resolve, so a placeholder glyph reports as an
		// unrecognised actor on every row that has no writer or no reader.
		return ""
	}
	out := make([]string, 0, len(set))
	for name := range set {
		// Typed tokens, so the actor resolver reads these the same way it reads
		// a hand-written cell. Bare member ids do not resolve and produce
		// graph_docs_unknown_actor on every row.
		out = append(out, "member:"+name)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
