package findings

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SubjectEdge records the bounded, typed relationship between a durable
// finding and the subject run(s) that support it. Cohort membership is not an
// edge: callers must explicitly name the implicated or contextual run.
type SubjectEdge struct {
	FindingID    uuid.UUID `json:"findingId"`
	SubjectRunID uuid.UUID `json:"subjectRunId"`
	Relation     string    `json:"relation"`
	EvidenceRefs []string  `json:"evidenceRefs,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SubjectEdgeRepository interface {
	CreateEdge(context.Context, SubjectEdge) error
	ListEdges(context.Context, uuid.UUID) ([]SubjectEdge, error)
}

func (r *SQLiteRepository) CreateEdge(ctx context.Context, edge SubjectEdge) error {
	if edge.FindingID == uuid.Nil {
		return fmt.Errorf("finding id is required")
	}
	if edge.SubjectRunID == uuid.Nil {
		return fmt.Errorf("subject run id is required")
	}
	if strings.TrimSpace(edge.Relation) == "" {
		return fmt.Errorf("relation is required")
	}
	if edge.CreatedAt.IsZero() {
		edge.CreatedAt = time.Now().UTC()
	}
	refs, err := json.Marshal(edge.EvidenceRefs)
	if err != nil {
		return fmt.Errorf("marshal evidence refs: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT OR IGNORE INTO run_finding_subject_edges (finding_id, subject_run_id, relation, evidence_refs, created_at) VALUES (?, ?, ?, ?, ?)`, edge.FindingID.String(), edge.SubjectRunID.String(), strings.TrimSpace(edge.Relation), string(refs), edge.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (r *SQLiteRepository) ListEdges(ctx context.Context, findingID uuid.UUID) ([]SubjectEdge, error) {
	if findingID == uuid.Nil {
		return nil, fmt.Errorf("finding id is required")
	}
	rows, err := r.db.QueryxContext(ctx, `SELECT finding_id, subject_run_id, relation, evidence_refs, created_at FROM run_finding_subject_edges WHERE finding_id = ? ORDER BY created_at ASC, subject_run_id ASC`, findingID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []SubjectEdge
	for rows.Next() {
		var findingRaw, subjectRaw, relation, refsRaw, createdRaw string
		if err := rows.Scan(&findingRaw, &subjectRaw, &relation, &refsRaw, &createdRaw); err != nil {
			return nil, err
		}
		parsedFinding, err := uuid.Parse(findingRaw)
		if err != nil {
			return nil, fmt.Errorf("parse finding id: %w", err)
		}
		parsedSubject, err := uuid.Parse(subjectRaw)
		if err != nil {
			return nil, fmt.Errorf("parse subject run id: %w", err)
		}
		createdAt, err := time.Parse(time.RFC3339Nano, createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse edge timestamp: %w", err)
		}
		var refs []string
		if err := json.Unmarshal([]byte(refsRaw), &refs); err != nil {
			return nil, fmt.Errorf("parse evidence refs: %w", err)
		}
		result = append(result, SubjectEdge{FindingID: parsedFinding, SubjectRunID: parsedSubject, Relation: relation, EvidenceRefs: refs, CreatedAt: createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

var _ SubjectEdgeRepository = (*SQLiteRepository)(nil)
