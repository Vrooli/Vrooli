// Package persistence provides data access for the unified recording service.
//
// DOC: docs/architecture/recording.md#persistence-layer
package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/domain"
)

// Repository defines the persistence interface for the unified recording service.
// It handles both timeline entries (actions + page events) and session metadata.
type Repository interface {
	// === Session lifecycle ===

	// CreateSession persists a new recording session.
	CreateSession(ctx context.Context, session *domain.RecordingSession) error

	// GetSession retrieves a session by ID.
	GetSession(ctx context.Context, sessionID string) (*domain.RecordingSession, error)

	// CloseSession marks a session as closed.
	CloseSession(ctx context.Context, sessionID string, closedAt time.Time) error

	// ListSessions returns sessions with optional filtering by profile ID.
	ListSessions(ctx context.Context, profileID *string, limit, offset int) ([]*domain.RecordingSession, error)

	// DeleteSession removes a session and all its timeline entries.
	DeleteSession(ctx context.Context, sessionID string) error

	// === Timeline persistence ===

	// SaveTimelineEntry persists a single timeline entry (action or page event).
	SaveTimelineEntry(ctx context.Context, entry *UnifiedTimelineEntry) error

	// SaveTimelineEntries persists multiple entries in a batch.
	SaveTimelineEntries(ctx context.Context, entries []*UnifiedTimelineEntry) error

	// GetTimelineEntry retrieves a single entry by ID.
	GetTimelineEntry(ctx context.Context, entryID uuid.UUID) (*UnifiedTimelineEntry, error)

	// GetTimeline returns timeline entries matching the query.
	GetTimeline(ctx context.Context, query TimelineQuery) (*TimelineResponse, error)

	// CountTimelineEntries returns the total entry count for a session.
	CountTimelineEntries(ctx context.Context, sessionID string) (int, error)

	// === Cleanup ===

	// DeleteSessionEntries removes all timeline entries for a session.
	DeleteSessionEntries(ctx context.Context, sessionID string) error

	// PruneOldSessions removes sessions older than the given time.
	PruneOldSessions(ctx context.Context, olderThan time.Time) (int, error)
}
