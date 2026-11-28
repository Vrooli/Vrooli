package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type memoryCorpus struct {
	mu        sync.Mutex
	knowledge map[string][]KnowledgeEntry
	handoffs  map[string][]HandoffEntry
	attempts  map[string][]HeartbeatAttempt
	drafts    map[string][]BugDraft
	closures  map[string][]KnowledgeClosure
}

func newMemoryCorpus() *memoryCorpus {
	return &memoryCorpus{knowledge: map[string][]KnowledgeEntry{}, handoffs: map[string][]HandoffEntry{}, attempts: map[string][]HeartbeatAttempt{}, drafts: map[string][]BugDraft{}, closures: map[string][]KnowledgeClosure{}}
}

func sourceLedgerScope(teamID string) string {
	teamID = strings.TrimSpace(teamID)
	if strings.HasPrefix(teamID, "team:") {
		return teamID
	}
	return "team:" + teamID
}

const (
	sourceLedgerKnowledgeKind       = "prompt-manager.team-knowledge"
	sourceLedgerHandoffKind         = "prompt-manager.team-handoff"
	sourceLedgerHandoffSnapshotKind = "prompt-manager.team-handoff-snapshot"
	sourceLedgerBugDraftKind        = "prompt-manager.team-bug-draft"
	// sourceLedgerKnowledgeClosureKind records that a knowledge entry is
	// resolved. The ledger is append-only, so closure is a separate record
	// keyed by knowledge id rather than a mutation of the entry.
	sourceLedgerKnowledgeClosureKind = "prompt-manager.team-knowledge-closure"
)

// Team corpus listing reads source-ledger pages of teamCorpusPageSize (the
// ledger's maximum) and stops after maxTeamCorpusPages so one oversized scope
// cannot turn a listing into an unbounded walk.
const (
	teamCorpusPageSize = 500
	maxTeamCorpusPages = 200
)

// Knowledge closure reasons. duplicate-of and routed-to require Ref.
const (
	KnowledgeCloseFixed       = "fixed"
	KnowledgeCloseStale       = "stale"
	KnowledgeCloseDuplicateOf = "duplicate-of"
	KnowledgeCloseRoutedTo    = "routed-to"
)

// KnowledgeClosure records that a team-corpus entry is resolved.
type KnowledgeClosure struct {
	KnowledgeID string `json:"knowledge_id"`
	Reason      string `json:"reason"`
	Ref         string `json:"ref,omitempty"`
	Note        string `json:"note,omitempty"`
	Actor       string `json:"actor,omitempty"`
	At          string `json:"at,omitempty"`
}

// KnowledgeListOptions filters a team-corpus listing.
type KnowledgeListOptions struct {
	Topic         string
	TopicPrefix   string
	Last          int
	IncludeClosed bool
}

// KnowledgeListResult is a filtered team-corpus listing. Total counts matches
// before Last was applied; Truncated reports that the page bound was reached.
type KnowledgeListResult struct {
	Entries   []KnowledgeEntry
	Total     int
	Truncated bool
}

// ValidateKnowledgeClosure normalizes and validates a closure request.
func ValidateKnowledgeClosure(closure *KnowledgeClosure) error {
	closure.KnowledgeID = strings.TrimSpace(closure.KnowledgeID)
	closure.Reason = strings.TrimSpace(strings.ToLower(closure.Reason))
	closure.Ref = strings.TrimSpace(closure.Ref)
	if closure.KnowledgeID == "" {
		return fmt.Errorf("knowledge id is required")
	}
	switch closure.Reason {
	case KnowledgeCloseFixed, KnowledgeCloseStale:
	case KnowledgeCloseDuplicateOf, KnowledgeCloseRoutedTo:
		if closure.Ref == "" {
			return fmt.Errorf("reason %s requires ref", closure.Reason)
		}
		if closure.Reason == KnowledgeCloseDuplicateOf && closure.Ref == closure.KnowledgeID {
			return fmt.Errorf("an entry cannot be a duplicate of itself")
		}
	default:
		return fmt.Errorf("reason must be one of fixed, stale, duplicate-of, routed-to")
	}
	return nil
}

func decodeKnowledgeClosure(body string) (KnowledgeClosure, bool) {
	var closure KnowledgeClosure
	if err := json.Unmarshal([]byte(body), &closure); err != nil || strings.TrimSpace(closure.KnowledgeID) == "" {
		return KnowledgeClosure{}, false
	}
	return closure, true
}

func encodeKnowledge(entry KnowledgeEntry) (string, error) {
	data, err := json.Marshal(entry)
	return string(data), err
}

func decodeKnowledge(body string) (KnowledgeEntry, bool) {
	var entry KnowledgeEntry
	if err := json.Unmarshal([]byte(body), &entry); err != nil || entry.Topic == "" && entry.Content == "" {
		return KnowledgeEntry{}, false
	}
	return entry, true
}

func encodeHandoff(entry HandoffEntry) (string, error) {
	data, err := json.Marshal(entry)
	return string(data), err
}

func decodeHandoff(body string) (HandoffEntry, bool) {
	var entry HandoffEntry
	if err := json.Unmarshal([]byte(body), &entry); err != nil {
		return HandoffEntry{}, false
	}
	return entry, true
}
