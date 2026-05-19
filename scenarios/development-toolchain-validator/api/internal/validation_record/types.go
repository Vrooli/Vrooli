// Package validation_record owns the append-only history of completed
// validation runs (OT-P0-006). Every terminal run lands here. The
// canonical operational state for in-flight runs lives in
// internal/validation_run/ (Phase 4).
package validation_record

import (
	"fmt"
	"time"
)

// TupleKind classifies what was validated.
type TupleKind int

const (
	TupleKindUnspecified TupleKind = 0
	TupleKindSkill       TupleKind = 1
	TupleKindTool        TupleKind = 2
)

// Verdict enumerates terminal outcomes.
type Verdict int

const (
	VerdictUnspecified        Verdict = 0
	VerdictPass               Verdict = 1
	VerdictUnexpectedMutation Verdict = 2
	VerdictRunFailure         Verdict = 3
	VerdictToolFailure        Verdict = 4
)

// Record is the domain shape for an append-only validation record.
type Record struct {
	ID         string
	TupleKind  TupleKind
	SubjectID  string
	GoldenSlug string

	StartedAt  time.Time
	EndedAt    time.Time
	DurationMS int64

	TokensUsed   int64
	CostUSDMicro int64

	Verdict Verdict

	DiffHash      string
	DiffPathCount int32

	AgentManagerRunID string

	ManifestTemplateVersionAtRun string
	ManifestSkillVersionAtRun    string

	ErrorMessage string
}

// AppendInput is the explicit DTO Service.Append accepts. ID and
// timestamps are server-owned.
type AppendInput struct {
	TupleKind     TupleKind
	SubjectID     string
	GoldenSlug    string
	StartedAt     time.Time
	EndedAt       time.Time
	TokensUsed    int64
	CostUSDMicro  int64
	Verdict       Verdict
	DiffHash      string
	DiffPathCount int32

	AgentManagerRunID            string
	ManifestTemplateVersionAtRun string
	ManifestSkillVersionAtRun    string
	ErrorMessage                 string
}

// ListFilter narrows ListRecords queries. Zero values mean "any".
type ListFilter struct {
	GoldenSlug string
	SubjectID  string
	TupleKind  TupleKind
}

// ListResult is the cursor-paginated response shape.
type ListResult struct {
	Records       []Record
	NextPageToken string
}

// ErrRecordNotFound is the typed sentinel returned by Get when no row
// matches.
type ErrRecordNotFound struct {
	ID string
}

func (e ErrRecordNotFound) Error() string {
	return fmt.Sprintf("validation record %q not found", e.ID)
}

// ErrInvalidRecord is the typed sentinel returned when input validation
// fails.
type ErrInvalidRecord struct {
	Field  string
	Reason string
}

func (e ErrInvalidRecord) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
