package cases

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	chainforensics "github.com/vrooli/chain-forensics"
)

type sqlBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type revisionHistoryPayload struct {
	Version           string              `json:"version"`
	Additions         RevisionAdditions   `json:"additions"`
	PreviousCase      chainforensics.Case `json:"previous_case"`
	PreviousCreatedAt time.Time           `json:"previous_case_created_at"`
}

// Revise appends typed attribution and evidence material to a case. The
// caller's read timestamp is an optimistic concurrency token. The case update
// and immutable revision receipt commit in one database transaction.
func (r *sqliteRepository) Revise(ctx context.Context, caseID string, expected time.Time, additions RevisionAdditions) (Record, CaseRevision, error) {
	if strings.TrimSpace(caseID) == "" || expected.IsZero() {
		return Record{}, CaseRevision{}, ErrInvalid{Field: "expected_updated_at", Reason: "case id and the exact case updated_at value are required"}
	}
	if strings.TrimSpace(additions.Summary) == "" && len(additions.Sources)+len(additions.Claims)+len(additions.Occurrences)+len(additions.Edges)+len(additions.EdgeRevisions)+len(additions.Gaps)+len(additions.Counterexamples)+len(additions.Disagreements)+len(additions.ResolvedGaps) == 0 {
		return Record{}, CaseRevision{}, ErrInvalid{Field: "additions", Reason: "at least one typed evidence addition is required"}
	}
	current, err := r.Get(ctx, caseID)
	if err != nil {
		return Record{}, CaseRevision{}, err
	}
	if !current.UpdatedAt.Equal(expected.UTC()) {
		return Record{}, CaseRevision{}, ErrStaleRevision{ID: caseID}
	}
	previousCaseBytes, err := json.Marshal(current.Case)
	if err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("encode prior case snapshot: %w", err)
	}
	var previousCase chainforensics.Case
	if err := json.Unmarshal(previousCaseBytes, &previousCase); err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("copy prior case snapshot: %w", err)
	}
	previousCreatedAt := current.CreatedAt
	usedEdgeIDs, err := r.usedEdgeIDs(ctx, caseID, current.Case)
	if err != nil {
		return Record{}, CaseRevision{}, err
	}
	if strings.TrimSpace(additions.Summary) != "" {
		current.Case.Summary = additions.Summary
	}
	if len(additions.ResolvedGaps) > 0 {
		present := make(map[string]struct{}, len(current.Case.Gaps))
		for _, gap := range current.Case.Gaps {
			present[gap] = struct{}{}
		}
		requested := make(map[string]struct{}, len(additions.ResolvedGaps))
		for _, gap := range additions.ResolvedGaps {
			if strings.TrimSpace(gap) == "" {
				return Record{}, CaseRevision{}, ErrInvalid{Field: "resolved_gaps", Reason: "resolved gaps must be non-empty exact matches"}
			}
			if _, duplicate := requested[gap]; duplicate {
				return Record{}, CaseRevision{}, ErrInvalid{Field: "resolved_gaps", Reason: fmt.Sprintf("gap %q was listed more than once", gap)}
			}
			if _, exists := present[gap]; !exists {
				return Record{}, CaseRevision{}, ErrInvalid{Field: "resolved_gaps", Reason: fmt.Sprintf("gap %q does not exactly match an open case gap", gap)}
			}
			requested[gap] = struct{}{}
		}
		open := current.Case.Gaps[:0]
		for _, gap := range current.Case.Gaps {
			if _, resolved := requested[gap]; !resolved {
				open = append(open, gap)
			}
		}
		current.Case.Gaps = open
	}
	existingEdges := usedEdgeIDs
	currentEdgeIDs := make(map[string]struct{}, len(current.Case.Edges))
	currentEdgeCounts := make(map[string]int, len(current.Case.Edges))
	for _, edge := range current.Case.Edges {
		currentEdgeIDs[edge.ID] = struct{}{}
		currentEdgeCounts[edge.ID]++
	}
	for _, edge := range additions.Edges {
		if _, exists := existingEdges[edge.ID]; exists {
			return Record{}, CaseRevision{}, ErrInvalid{Field: "edges", Reason: fmt.Sprintf("edge id %q already exists; assign a new id to preserve prior grading", edge.ID)}
		}
		existingEdges[edge.ID] = struct{}{}
	}
	superseded := make(map[string]struct{}, len(additions.EdgeRevisions))
	replacements := make([]chainforensics.Edge, 0, len(additions.EdgeRevisions))
	for _, revision := range additions.EdgeRevisions {
		if _, exists := currentEdgeIDs[revision.SupersedesEdgeID]; !exists {
			return Record{}, CaseRevision{}, ErrInvalid{Field: "edge_revisions", Reason: fmt.Sprintf("superseded edge %q does not exist", revision.SupersedesEdgeID)}
		}
		if currentEdgeCounts[revision.SupersedesEdgeID] != 1 {
			return Record{}, CaseRevision{}, ErrInvalid{Field: "edge_revisions", Reason: fmt.Sprintf("superseded edge id %q is ambiguous in the current case", revision.SupersedesEdgeID)}
		}
		if _, alreadySuperseded := superseded[revision.SupersedesEdgeID]; alreadySuperseded {
			return Record{}, CaseRevision{}, ErrInvalid{Field: "edge_revisions", Reason: fmt.Sprintf("edge %q was listed for supersession more than once", revision.SupersedesEdgeID)}
		}
		if _, idUsed := existingEdges[revision.Replacement.ID]; idUsed {
			return Record{}, CaseRevision{}, ErrInvalid{Field: "edge_revisions", Reason: fmt.Sprintf("replacement edge id %q must be new and unique", revision.Replacement.ID)}
		}
		superseded[revision.SupersedesEdgeID] = struct{}{}
		existingEdges[revision.Replacement.ID] = struct{}{}
		replacements = append(replacements, revision.Replacement)
	}
	previous := current.UpdatedAt
	current.Case.Sources = append(current.Case.Sources, additions.Sources...)
	current.Case.Claims = append(current.Case.Claims, additions.Claims...)
	current.Case.Occurrences = append(current.Case.Occurrences, additions.Occurrences...)
	if len(superseded) > 0 {
		kept := current.Case.Edges[:0]
		for _, edge := range current.Case.Edges {
			if _, remove := superseded[edge.ID]; !remove {
				kept = append(kept, edge)
			}
		}
		current.Case.Edges = kept
	}
	current.Case.Edges = append(current.Case.Edges, additions.Edges...)
	current.Case.Edges = append(current.Case.Edges, replacements...)
	current.Case.Gaps = append(current.Case.Gaps, additions.Gaps...)
	current.Case.Counterexamples = append(current.Case.Counterexamples, additions.Counterexamples...)
	current.Case.Disagreements = append(current.Case.Disagreements, additions.Disagreements...)
	if err := chainforensics.Validate(current.Case); err != nil {
		return Record{}, CaseRevision{}, ErrInvalid{Field: "additions", Reason: err.Error()}
	}
	document, err := json.Marshal(current.Case)
	if err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("encode revised case: %w", err)
	}
	additionJSON, err := json.Marshal(revisionHistoryPayload{Version: "chain-forensics.case-revision/v1", Additions: additions, PreviousCase: previousCase, PreviousCreatedAt: previousCreatedAt})
	if err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("encode revision receipt: %w", err)
	}
	beg, ok := r.db.(sqlBeginner)
	if !ok {
		return Record{}, CaseRevision{}, errors.New("case revision requires transactional database support")
	}
	tx, err := beg.BeginTx(ctx, nil)
	if err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("begin case revision: %w", err)
	}
	defer tx.Rollback()
	now := r.clock.Now().UTC()
	if !now.After(previous) {
		now = previous.Add(time.Nanosecond)
	}
	stamp := now.Format(timestampLayout)
	expectedStamp := previous.UTC().Format(timestampLayout)
	result, err := tx.ExecContext(ctx, `UPDATE forensic_cases SET document_json = ?, updated_at = ? WHERE id = ? AND updated_at = ?`, string(document), stamp, caseID, expectedStamp)
	if err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("update revised case: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("read revision update result: %w", err)
	}
	if rows != 1 {
		return Record{}, CaseRevision{}, ErrStaleRevision{ID: caseID}
	}
	revision := CaseRevision{ID: uuid.NewString(), CaseID: caseID, PreviousUpdatedAt: previous, AppliedAt: now, Additions: additions, PreviousCase: previousCase, PreviousCreatedAt: previousCreatedAt}
	_, err = tx.ExecContext(ctx, `INSERT INTO forensic_case_revisions(id, case_id, previous_updated_at, applied_at, additions_json) VALUES (?, ?, ?, ?, ?)`, revision.ID, caseID, expectedStamp, stamp, string(additionJSON))
	if err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("persist case revision receipt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Record{}, CaseRevision{}, fmt.Errorf("commit case revision: %w", err)
	}
	current.UpdatedAt = now
	return current, revision, nil
}

func (r *sqliteRepository) ListRevisions(ctx context.Context, caseID string) ([]CaseRevision, error) {
	if _, err := r.Get(ctx, caseID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, case_id, previous_updated_at, applied_at, additions_json FROM forensic_case_revisions WHERE case_id = ? ORDER BY applied_at, id`, caseID)
	if err != nil {
		return nil, fmt.Errorf("list case revisions: %w", err)
	}
	defer rows.Close()
	var revisions []CaseRevision
	for rows.Next() {
		var item CaseRevision
		var previous, applied, raw string
		if err := rows.Scan(&item.ID, &item.CaseID, &previous, &applied, &raw); err != nil {
			return nil, fmt.Errorf("scan case revision: %w", err)
		}
		item.PreviousUpdatedAt, err = time.Parse(timestampLayout, previous)
		if err != nil {
			return nil, fmt.Errorf("parse revision previous timestamp: %w", err)
		}
		item.AppliedAt, err = time.Parse(timestampLayout, applied)
		if err != nil {
			return nil, fmt.Errorf("parse revision applied timestamp: %w", err)
		}
		var payload revisionHistoryPayload
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return nil, fmt.Errorf("decode revision additions: %w", err)
		}
		if payload.Version == "" {
			// Read receipts written before full prior-case snapshots were added.
			if err := json.Unmarshal([]byte(raw), &payload.Additions); err != nil {
				return nil, fmt.Errorf("decode legacy revision additions: %w", err)
			}
		} else if payload.Version != "chain-forensics.case-revision/v1" {
			return nil, fmt.Errorf("decode revision additions: unsupported receipt version %q", payload.Version)
		}
		item.Additions = payload.Additions
		item.PreviousCase = payload.PreviousCase
		item.PreviousCreatedAt = payload.PreviousCreatedAt
		revisions = append(revisions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate case revisions: %w", err)
	}
	return revisions, nil
}

func (r *sqliteRepository) usedEdgeIDs(ctx context.Context, caseID string, current chainforensics.Case) (map[string]struct{}, error) {
	used := make(map[string]struct{}, len(current.Edges))
	addEdges := func(edges []chainforensics.Edge) {
		for _, edge := range edges {
			if edge.ID != "" {
				used[edge.ID] = struct{}{}
			}
		}
	}
	addEdges(current.Edges)
	rows, err := r.db.QueryContext(ctx, `SELECT additions_json FROM forensic_case_revisions WHERE case_id = ?`, caseID)
	if err != nil {
		return nil, fmt.Errorf("read prior case revisions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, fmt.Errorf("scan prior case revision: %w", err)
		}
		var payload revisionHistoryPayload
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			return nil, fmt.Errorf("decode prior case revision: %w", err)
		}
		if payload.Version == "" {
			if err := json.Unmarshal([]byte(raw), &payload.Additions); err != nil {
				return nil, fmt.Errorf("decode legacy prior case revision: %w", err)
			}
		} else if payload.Version != "chain-forensics.case-revision/v1" {
			return nil, fmt.Errorf("decode prior case revision: unsupported receipt version %q", payload.Version)
		}
		addEdges(payload.PreviousCase.Edges)
		addEdges(payload.Additions.Edges)
		for _, revision := range payload.Additions.EdgeRevisions {
			if revision.Replacement.ID != "" {
				used[revision.Replacement.ID] = struct{}{}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate prior case revisions: %w", err)
	}
	return used, nil
}
