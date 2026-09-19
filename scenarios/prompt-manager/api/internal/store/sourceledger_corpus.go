package store

import (
	"encoding/json"
	"strings"
	"sync"
)

type memoryCorpus struct {
	mu        sync.Mutex
	knowledge map[string][]KnowledgeEntry
	handoffs  map[string][]HandoffEntry
	attempts  map[string][]HeartbeatAttempt
	drafts    map[string][]BugDraft
}

func newMemoryCorpus() *memoryCorpus {
	return &memoryCorpus{knowledge: map[string][]KnowledgeEntry{}, handoffs: map[string][]HandoffEntry{}, attempts: map[string][]HeartbeatAttempt{}, drafts: map[string][]BugDraft{}}
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
)

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
