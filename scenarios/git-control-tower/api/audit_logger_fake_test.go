package main

import (
	"context"
	"time"
)

// FakeAuditLogger implements AuditLogger without a database.
// Use this in tests to verify audit logging behavior.
//
// [REQ:GCT-OT-P0-007] SQLite audit logging
type FakeAuditLogger struct {
	// Entries stores all logged entries.
	Entries []AuditEntry

	// Configured simulates whether logging is available.
	Configured bool

	// LogError is returned from Log if set.
	LogError error

	// QueryError is returned from Query if set.
	QueryError error
}

// NewFakeAuditLogger creates a new fake audit logger.
func NewFakeAuditLogger() *FakeAuditLogger {
	return &FakeAuditLogger{
		Entries:    []AuditEntry{},
		Configured: true,
	}
}

func (l *FakeAuditLogger) IsConfigured() bool {
	return l.Configured
}

func (l *FakeAuditLogger) Log(_ context.Context, entry AuditEntry) error {
	if l.LogError != nil {
		return l.LogError
	}
	if !l.Configured {
		return nil
	}

	// Auto-set timestamp if not provided
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	// Auto-increment ID
	entry.ID = int64(len(l.Entries) + 1)

	l.Entries = append(l.Entries, entry)
	return nil
}

func (l *FakeAuditLogger) Query(_ context.Context, req AuditQueryRequest) (*AuditQueryResponse, error) {
	if l.QueryError != nil {
		return nil, l.QueryError
	}

	filtered := l.filterEntries(req)
	total := len(filtered)
	filtered = paginateEntries(filtered, req.Offset, req.Limit)

	return &AuditQueryResponse{
		Entries:   filtered,
		Total:     total,
		Timestamp: time.Now().UTC(),
	}, nil
}

func (l *FakeAuditLogger) filterEntries(req AuditQueryRequest) []AuditEntry {
	var filtered []AuditEntry
	for _, e := range l.Entries {
		if req.Operation != "" && e.Operation != req.Operation {
			continue
		}
		if req.Branch != "" && e.Branch != req.Branch {
			continue
		}
		if !req.Since.IsZero() && e.Timestamp.Before(req.Since) {
			continue
		}
		if !req.Until.IsZero() && e.Timestamp.After(req.Until) {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func paginateEntries(entries []AuditEntry, offset, limit int) []AuditEntry {
	if offset > 0 {
		if offset >= len(entries) {
			return []AuditEntry{}
		}
		entries = entries[offset:]
	}
	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}
	return entries
}

// --- Test helpers ---

// WithUnconfigured sets the logger to unconfigured state.
func (l *FakeAuditLogger) WithUnconfigured() *FakeAuditLogger {
	l.Configured = false
	return l
}

// EntryCount returns the number of logged entries.
func (l *FakeAuditLogger) EntryCount() int {
	return len(l.Entries)
}

// LastEntry returns the most recently logged entry.
func (l *FakeAuditLogger) LastEntry() *AuditEntry {
	if len(l.Entries) == 0 {
		return nil
	}
	return &l.Entries[len(l.Entries)-1]
}

// HasOperation returns true if an operation of the given type was logged.
func (l *FakeAuditLogger) HasOperation(op AuditOperation) bool {
	for _, e := range l.Entries {
		if e.Operation == op {
			return true
		}
	}
	return false
}

// CountOperation returns the number of entries with the given operation type.
func (l *FakeAuditLogger) CountOperation(op AuditOperation) int {
	count := 0
	for _, e := range l.Entries {
		if e.Operation == op {
			count++
		}
	}
	return count
}
