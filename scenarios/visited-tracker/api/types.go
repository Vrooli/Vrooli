package main

import (
	"time"

	"github.com/google/uuid"
)

// Campaign represents a file tracking campaign with visit history and staleness metrics
type Campaign struct {
	ID                 uuid.UUID              `json:"id"`
	Name               string                 `json:"name"`
	FromAgent          string                 `json:"from_agent"`
	Description        *string                `json:"description,omitempty"`
	Patterns           []string               `json:"patterns"`
	Location           *string                `json:"location,omitempty"`
	Tag                *string                `json:"tag,omitempty"`
	Notes              *string                `json:"notes,omitempty"`
	MaxFiles           int                    `json:"max_files,omitempty"`
	ExcludePatterns    []string               `json:"exclude_patterns,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
	Status             string                 `json:"status"`
	Metadata           map[string]interface{} `json:"metadata"`
	TrackedFiles       []TrackedFile          `json:"tracked_files"`
	Visits             []Visit                `json:"visits"`
	StructureSnapshots []StructureSnapshot    `json:"structure_snapshots"`
	// Computed fields (not stored)
	TotalFiles      int     `json:"total_files,omitempty"`
	VisitedFiles    int     `json:"visited_files,omitempty"`
	CoveragePercent float64 `json:"coverage_percent,omitempty"`
}

// TrackedFile represents a file being tracked with visit counts and staleness metrics
type TrackedFile struct {
	ID             uuid.UUID              `json:"id"`
	FilePath       string                 `json:"file_path"`
	AbsolutePath   string                 `json:"absolute_path"`
	VisitCount     int                    `json:"visit_count"`
	FirstSeen      time.Time              `json:"first_seen"`
	LastVisited    *time.Time             `json:"last_visited,omitempty"`
	LastModified   time.Time              `json:"last_modified"`
	ContentHash    *string                `json:"content_hash,omitempty"`
	SizeBytes      int64                  `json:"size_bytes"`
	StalenessScore float64                `json:"staleness_score"`
	Deleted        bool                   `json:"deleted"`
	Notes          *string                `json:"notes,omitempty"`
	PriorityWeight float64                `json:"priority_weight,omitempty"`
	Excluded       bool                   `json:"excluded,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// Visit represents a single file visit event
type Visit struct {
	ID             uuid.UUID              `json:"id"`
	FileID         uuid.UUID              `json:"file_id"`
	Timestamp      time.Time              `json:"timestamp"`
	Context        *string                `json:"context,omitempty"`
	Agent          *string                `json:"agent,omitempty"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	DurationMs     *int                   `json:"duration_ms,omitempty"`
	Findings       map[string]interface{} `json:"findings,omitempty"`
}

// StructureSnapshot captures the state of tracked files at a point in time
type StructureSnapshot struct {
	ID           uuid.UUID              `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	TotalFiles   int                    `json:"total_files"`
	NewFiles     []string               `json:"new_files"`
	DeletedFiles []string               `json:"deleted_files"`
	MovedFiles   map[string]string      `json:"moved_files"`
	SnapshotData map[string]interface{} `json:"snapshot_data"`
}

// CreateCampaignRequest contains parameters for creating a new campaign
type CreateCampaignRequest struct {
	Name            string                 `json:"name"`
	FromAgent       string                 `json:"from_agent"`
	Description     *string                `json:"description,omitempty"`
	Patterns        []string               `json:"patterns"`
	Location        *string                `json:"location,omitempty"`
	Tag             *string                `json:"tag,omitempty"`
	Notes           *string                `json:"notes,omitempty"`
	MaxFiles        int                    `json:"max_files,omitempty"`
	ExcludePatterns []string               `json:"exclude_patterns,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// VisitRequest contains parameters for recording file visits
type VisitRequest struct {
	Files          interface{}            `json:"files"` // Can be []string or []FileVisit
	Context        *string                `json:"context,omitempty"`
	Agent          *string                `json:"agent,omitempty"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	FileNotes      map[string]string      `json:"file_notes,omitempty"` // Map of file_path -> note
}

// FileVisit represents a single file visit with optional context
type FileVisit struct {
	Path    string  `json:"path"`
	Context *string `json:"context,omitempty"`
}

// StructureSyncRequest contains parameters for syncing campaign file structure
type StructureSyncRequest struct {
	Files         []string               `json:"files,omitempty"`
	Structure     map[string]interface{} `json:"structure,omitempty"`
	Patterns      []string               `json:"patterns,omitempty"`
	RemoveDeleted bool                   `json:"remove_deleted,omitempty"`
}

// SyncResult contains the results of a file structure sync operation
type SyncResult struct {
	Added      int       `json:"added"`
	Moved      int       `json:"moved"`
	Removed    int       `json:"removed"`
	SnapshotID uuid.UUID `json:"snapshot_id"`
	Total      int       `json:"total"`
}

// AdjustVisitRequest contains parameters for manually adjusting visit counts
type AdjustVisitRequest struct {
	FileID string `json:"file_id"`
	Action string `json:"action"` // "increment" or "decrement"
}

// BulkExcludeRequest contains parameters for bulk excluding files
type BulkExcludeRequest struct {
	Files    []string `json:"files"`
	Reason   *string  `json:"reason,omitempty"`
	Excluded bool     `json:"excluded"`
}
