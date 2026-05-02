// Adapter from prompt-manager/store.FileTeamStore to memberflow.KnowledgeQuery.
//
// The topics data layer uses topic-prefix syntax with /* wildcards (e.g.
// "research-inbox/*"); the team-store knowledge log uses a literal HasPrefix
// match. This adapter strips the wildcard suffix before delegating, and maps
// the timestamp string to time.Time.
//
// DOC: docs/agent-system/drafts/topics-schema.md

package main

import (
	"context"
	"strings"
	"time"

	"prompt-manager/memberflow"
	"prompt-manager/store"
)

type teamKnowledgeQuery struct {
	store *store.FileTeamStore
}

func newTeamKnowledgeQuery(s *store.FileTeamStore) memberflow.KnowledgeQuery {
	if s == nil {
		return nil
	}
	return &teamKnowledgeQuery{store: s}
}

// ListUnrouted returns team-knowledge entries whose Topic falls under the
// given topic-prefix. Wildcard suffix is stripped: "research-inbox/*" ->
// HasPrefix "research-inbox/".
func (q *teamKnowledgeQuery) ListUnrouted(team string, prefix string) ([]memberflow.InboxEntry, error) {
	literal := strings.TrimSuffix(prefix, "/*")
	if literal == prefix && !strings.HasSuffix(literal, "/") {
		// Exact-topic prefix. team_store.GetKnowledge HasPrefix matches both
		// the topic and any descendants — for an exact match we still want
		// HasPrefix because the inbox convention nests slugs under the
		// signal type (`research-inbox/audience/foo`).
	} else if !strings.HasSuffix(literal, "/") {
		literal = literal + "/"
	}

	entries, err := q.store.GetKnowledge(context.Background(), team, "", literal, 0)
	if err != nil {
		return nil, err
	}
	out := make([]memberflow.InboxEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, memberflow.InboxEntry{
			ID:    e.ID,
			Topic: e.Topic,
			At:    parseRFC3339(e.At),
		})
	}
	return out, nil
}

func parseRFC3339(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
