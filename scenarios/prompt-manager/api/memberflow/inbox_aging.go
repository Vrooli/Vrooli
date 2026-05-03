// Inbox-aging validation rules: stalled_drain and piling_inbox.
//
// These rules are warnings, never errors, and depend on real team-knowledge
// queue depth + age. The pure-Go Validate() in validation.go cannot synthesize
// them because it has no I/O; we layer them on top via a KnowledgeQuery
// dependency that the API supplies in production and tests stub out.
//
// DOC: docs/agent-system/TOPICS_SCHEMA.md
package memberflow

import (
	"fmt"
	"time"
)

const (
	defaultStalledAfter = 7 * 24 * time.Hour
	defaultPilingAt     = 50
)

// InboxEntry is the minimal projection of a team-knowledge entry that the
// aging rules need. It is intentionally smaller than store.KnowledgeEntry to
// avoid a circular import between memberflow and store.
type InboxEntry struct {
	ID    string
	Topic string
	At    time.Time
}

// KnowledgeQuery returns unrouted entries whose Topic falls under the given
// prefix (matched with the same Overlap semantics as the topics data layer).
// Implementations should scope to a single team.
//
// `prefix` here is a topic-prefix in the topics.json sense (e.g.
// "research-inbox/audience/*" or "research-inbox/audience"). The query is
// expected to interpret a trailing /* as a wildcard.
type KnowledgeQuery interface {
	ListUnrouted(team string, prefix string) ([]InboxEntry, error)
}

// InboxAgingOptions configures the threshold behaviour for the inbox-aging
// rules. Zero values are replaced with defaults.
type InboxAgingOptions struct {
	StalledAfter time.Duration
	PilingAt     int
	Now          time.Time
}

func (o InboxAgingOptions) normalised() InboxAgingOptions {
	out := o
	if out.StalledAfter <= 0 {
		out.StalledAfter = defaultStalledAfter
	}
	if out.PilingAt <= 0 {
		out.PilingAt = defaultPilingAt
	}
	if out.Now.IsZero() {
		out.Now = time.Now()
	}
	return out
}

// EnrichWithDrainStatus runs the stalled_drain and piling_inbox rules for
// every intake declaration on every member. Returns an empty slice when
// `q` is nil — callers without a knowledge backend get a clean signal that
// the warnings are unavailable, not a misleading "all clean."
func EnrichWithDrainStatus(members []MemberTopics, q KnowledgeQuery, opts InboxAgingOptions) []Finding {
	if q == nil {
		return nil
	}
	o := opts.normalised()

	var findings []Finding
	for _, m := range members {
		for _, in := range m.Topics.Intake {
			entries, err := q.ListUnrouted(m.Ref.Team, in.Prefix)
			if err != nil {
				// A query error becomes a single warning per intake so
				// operators see the failure without blocking the report.
				findings = append(findings, Finding{
					Rule:     "drain_status_unavailable",
					Severity: SeverityWarning,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("knowledge-query error: %v", err),
				})
				continue
			}
			if len(entries) == 0 {
				continue
			}
			if len(entries) >= o.PilingAt {
				findings = append(findings, Finding{
					Rule:     "piling_inbox",
					Severity: SeverityWarning,
					Member:   m.Ref,
					Prefix:   in.Prefix,
					Detail:   fmt.Sprintf("%d unrouted entries under %q (threshold %d)", len(entries), in.Prefix, o.PilingAt),
				})
			}
			oldest := oldestEntry(entries)
			if !oldest.At.IsZero() {
				age := o.Now.Sub(oldest.At)
				if age >= o.StalledAfter {
					findings = append(findings, Finding{
						Rule:     "stalled_drain",
						Severity: SeverityWarning,
						Member:   m.Ref,
						Prefix:   in.Prefix,
						Detail:   fmt.Sprintf("oldest unrouted entry under %q is %s old (threshold %s)", in.Prefix, age.Round(time.Hour), o.StalledAfter),
					})
				}
			}
		}
	}
	return findings
}

func oldestEntry(entries []InboxEntry) InboxEntry {
	out := entries[0]
	for _, e := range entries[1:] {
		if e.At.IsZero() {
			continue
		}
		if out.At.IsZero() || e.At.Before(out.At) {
			out = e
		}
	}
	return out
}

// MergeFindings appends extra findings to a ValidationResult and updates the
// summary counts. Used by the API handlers to layer inbox-aging warnings on
// top of the pure-Go validation result.
func MergeFindings(r ValidationResult, extra []Finding) ValidationResult {
	if len(extra) == 0 {
		return r
	}
	r.Findings = append(r.Findings, extra...)
	for _, f := range extra {
		switch f.Severity {
		case SeverityError:
			r.Errors++
		case SeverityWarning:
			r.Warnings++
		}
	}
	sortFindings(r.Findings)
	return r
}
