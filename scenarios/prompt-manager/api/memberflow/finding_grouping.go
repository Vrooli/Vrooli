// Per-defect grouping for rules that scan knowledge entries.
//
// Rules that walk `knowledge.jsonl` fire once per entry, so their finding
// count tracks how often a member ran rather than how much is wrong. One
// undeclared prefix on one member produced 25 `actual_writer_undeclared`
// errors; the fix was a single line of `topics.json`. Reported that way, a
// sweep reads as a large repair job and the operator cannot see how many
// distinct declarations actually need attention.
//
// These helpers collapse per-entry findings into one finding per
// (member, topic family), carrying the occurrence count and a sample entry
// so the evidence trail survives. Callers group at the emission site rather
// than post-hoc, so rules whose per-entry shape is deliberate — the
// external-write threshold, which exists to name exactly which entries
// crossed — keep it.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md
package memberflow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// isoDateSegment matches a bare `YYYY-MM-DD` path segment.
var isoDateSegment = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// generatedIDSegment matches store-generated ids such as `dec-1778803361775636366`
// and `knw-1781829060026014477`.
var generatedIDSegment = regexp.MustCompile(`^[a-z]+-\d{6,}$`)

// trailingISODate matches a `YYYY-MM-DD` suffix fused onto a segment with a
// hyphen instead of a path separator — the shape a member produces when it
// builds a topic key by string concatenation rather than by path join.
var trailingISODate = regexp.MustCompile(`-\d{4}-\d{2}-\d{2}$`)

// observedTopicFamily collapses a concrete written topic to the prefix family
// an operator would declare in `topics.json`. It truncates at the first
// variable-looking segment — an ISO date, a generated id, or a bare number —
// and returns the literal head plus `/*`.
//
//   - `initiative-portfolio-record/2026-05-14`                → `initiative-portfolio-record/*`
//   - `challenge-resolution-record/dec-1778803361775636366`   → `challenge-resolution-record/*`
//   - `friction-report/recurring-workaround/2026-06-15/toolchain-fallback`
//     → `friction-report/recurring-workaround/*`
//   - `contrarian-scan-2026-06-14`                            → `contrarian-scan-*`
//   - `quality-audit/audio-tools/api-steer`                   → unchanged
//
// The last two cases matter most. A malformed key that fuses the date onto the
// family with a hyphen collapses to its own `-*` family rather than to one
// group per day, which is what makes a month of the same typo read as one
// defect — and keeps it visibly distinct from the correctly-formed `/*` family,
// because they are two different problems.
//
// A topic with no variable-looking segment is returned unchanged: there is
// nothing to generalize, and inventing a wildcard would group unrelated topics.
func observedTopicFamily(topic string) string {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return ""
	}
	segments := strings.Split(topic, "/")
	for i, seg := range segments {
		if !isVariableSegment(seg) {
			continue
		}
		if i == 0 {
			// The whole topic is variable; nothing literal to anchor to.
			break
		}
		return strings.Join(segments[:i], "/") + "/*"
	}
	// No variable path segment. Check for a date fused onto the final segment.
	last := len(segments) - 1
	if loc := trailingISODate.FindStringIndex(segments[last]); loc != nil {
		segments[last] = segments[last][:loc[0]] + "-*"
		return strings.Join(segments, "/")
	}
	return topic
}

func isVariableSegment(seg string) bool {
	if seg == "" {
		return false
	}
	if isoDateSegment.MatchString(seg) || generatedIDSegment.MatchString(seg) {
		return true
	}
	return strings.IndexFunc(seg, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

// findingGroup accumulates the per-entry occurrences of one defect.
type findingGroup struct {
	family     string
	sampleID   string
	firstTopic string
	count      int
}

// findingGrouper collects per-entry occurrences keyed by topic family, then
// emits one finding per family in stable family order.
//
// Zero value is ready to use.
type findingGrouper struct {
	order  []string
	groups map[string]*findingGroup
}

// Add records one occurrence of a defect on `topic`, attributed to knowledge
// entry `entryID`. The first entry seen for a family becomes the sample cited
// in the collapsed finding.
func (g *findingGrouper) Add(topic, entryID string) {
	family := observedTopicFamily(topic)
	if g.groups == nil {
		g.groups = make(map[string]*findingGroup)
	}
	existing, ok := g.groups[family]
	if !ok {
		existing = &findingGroup{family: family, sampleID: entryID, firstTopic: topic}
		g.groups[family] = existing
		g.order = append(g.order, family)
	}
	existing.count++
}

// Empty reports whether any occurrence was recorded.
func (g *findingGrouper) Empty() bool { return len(g.groups) == 0 }

// Emit builds one finding per family. `build` receives the family prefix and a
// human-readable evidence clause naming the occurrence count and a sample
// entry, and returns the finished finding.
func (g *findingGrouper) Emit(build func(family, evidence string) Finding) []Finding {
	families := append([]string(nil), g.order...)
	sort.Strings(families)

	findings := make([]Finding, 0, len(families))
	for _, family := range families {
		group := g.groups[family]
		findings = append(findings, build(family, describeOccurrences(group)))
	}
	return findings
}

func describeOccurrences(group *findingGroup) string {
	if group.count == 1 {
		return fmt.Sprintf("1 entry (%q, topic %q)", group.sampleID, group.firstTopic)
	}
	return fmt.Sprintf("%d entries (e.g. %q, topic %q)", group.count, group.sampleID, group.firstTopic)
}
