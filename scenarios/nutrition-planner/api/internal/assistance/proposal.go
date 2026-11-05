// Package assistance contains deterministic boundaries around optional model
// work. Proposals are data awaiting review; they are never recipe truth.
package assistance

import (
	"errors"
	"fmt"
	"strings"
)

const CurrentSchemaVersion = 1

type Change struct {
	Field string
	Value string
}

type Proposal struct {
	SchemaVersion       int
	SourceIDs           []string
	BaseDraftRevision   int64
	Changes             []Change
	EvidenceReferences  []string
	UnresolvedQuestions []string
	ValidationWarnings  []string
}

type Review struct {
	Valid   bool
	Errors  []string
	Changes []Change
}

var allowedFields = map[string]bool{
	"name":         true,
	"notes":        true,
	"sourceUrl":    true,
	"originalText": true,
}

func Validate(p Proposal) Review {
	review := Review{Valid: true}
	if p.SchemaVersion != CurrentSchemaVersion {
		review.Errors = append(review.Errors, fmt.Sprintf("unsupported proposal schema version %d", p.SchemaVersion))
	}
	if p.BaseDraftRevision < 0 {
		review.Errors = append(review.Errors, "base draft revision must not be negative")
	}
	seen := map[string]bool{}
	for _, change := range p.Changes {
		field := strings.TrimSpace(change.Field)
		if !allowedFields[field] {
			review.Errors = append(review.Errors, fmt.Sprintf("unsupported proposed field %q", field))
			continue
		}
		if seen[field] {
			review.Errors = append(review.Errors, fmt.Sprintf("duplicate proposed field %q", field))
			continue
		}
		seen[field] = true
		review.Changes = append(review.Changes, Change{Field: field, Value: change.Value})
	}
	review.Valid = len(review.Errors) == 0
	return review
}

var ErrStaleDraft = errors.New("proposal base draft revision is stale")

func Apply(p Proposal, currentRevision int64) (map[string]string, error) {
	review := Validate(p)
	if !review.Valid {
		return nil, errors.New(strings.Join(review.Errors, "; "))
	}
	if p.BaseDraftRevision != currentRevision {
		return nil, ErrStaleDraft
	}
	changes := make(map[string]string, len(review.Changes))
	for _, change := range review.Changes {
		changes[change.Field] = change.Value
	}
	return changes, nil
}
