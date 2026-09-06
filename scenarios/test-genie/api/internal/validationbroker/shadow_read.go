package validationbroker

import (
	"context"
	"time"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"test-genie/internal/storage/sqliteutil"
)

// ShadowComparison is immutable historical evidence retained for bounded
// audit. New validation work no longer writes this compatibility projection.
type ShadowComparison struct {
	ComparisonID         string
	SourceKind           string
	SourceID             string
	ReceiptID            string
	ReceiptRevision      uint64
	LegacyState          string
	ReceiptState         validationv1.ReceiptState
	Matched              bool
	ReasonCode           string
	LegacyEvidenceCount  int
	ReceiptEvidenceCount int
	ObservedAt           time.Time
}

// ListShadowComparisons is the read-only historical evidence boundary. The
// table is retained so archived cutover evidence remains inspectable without
// restoring the retired migration writers.
func (r *Repository) ListShadowComparisons(ctx context.Context, limit int) ([]ShadowComparison, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT comparison_id, source_kind, source_id, receipt_id, receipt_revision, legacy_state, receipt_state, matched, reason_code, legacy_evidence_count, receipt_evidence_count, observed_at
        FROM validation_shadow_comparisons ORDER BY observed_at DESC, comparison_id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ShadowComparison, 0, limit)
	for rows.Next() {
		var item ShadowComparison
		var observedAt string
		if err := rows.Scan(&item.ComparisonID, &item.SourceKind, &item.SourceID, &item.ReceiptID, &item.ReceiptRevision, &item.LegacyState, &item.ReceiptState, &item.Matched, &item.ReasonCode, &item.LegacyEvidenceCount, &item.ReceiptEvidenceCount, &observedAt); err != nil {
			return nil, err
		}
		item.ObservedAt, err = sqliteutil.ParseTimestamp(observedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}
