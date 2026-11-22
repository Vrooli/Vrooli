package relationships

import "errors"

var (
	ErrInvalid  = errors.New("invalid content relationship")
	ErrNotFound = errors.New("relationship target not found")
	ErrConflict = errors.New("primary family relationship already exists")
)

type Relation struct {
	ID, SourceDraftID, SourceRevisionID, TargetDraftID, TargetRevisionID, ProjectID, Kind string
	Primary                                                                               bool
}

type Collection struct{ ID, ProjectID, Name string }
type CollectionItem struct{ CollectionID, DraftID, RevisionID string }

// Impact is a read-only inventory. It reports references that would be
// affected by a proposed action; it never performs the action or cascades.
type Impact struct {
	DraftID               string
	Operation             string
	Relationships         int64
	CollectionMemberships int64
	Attachments           int64
	HasDraftSlot          bool
	HasApproval           bool
}
