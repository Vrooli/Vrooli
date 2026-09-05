package validationbroker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"test-genie/internal/storage/sqliteutil"
)

// LegacyValidationRecord is the only supported legacy-to-receipt input. Each
// owner translates its historical row to this shape instead of teaching Test
// Genie about another scenario's storage engine.
type LegacyValidationRecord struct {
	SourceKind     string
	SourceID       string
	CallerScenario string
	TargetScenario string
	State          string
	Evidence       []*validationv1.EvidenceReference
}

type LegacyMigrationResult struct {
	Receipt  *validationv1.ValidationReceipt
	Migrated bool
	ReadOnly bool
	Reason   string
}

type LegacyMigrationAdapter struct {
	repo ReceiptRepository
	now  func() time.Time
}

func NewLegacyMigrationAdapter(repo ReceiptRepository) *LegacyMigrationAdapter {
	return &LegacyMigrationAdapter{repo: repo, now: time.Now}
}

func (a *LegacyMigrationAdapter) Migrate(ctx context.Context, record LegacyValidationRecord) (LegacyMigrationResult, error) {
	if a == nil || a.repo == nil {
		return LegacyMigrationResult{}, errors.New("legacy migration receipt repository is unavailable")
	}
	record.SourceKind = strings.TrimSpace(record.SourceKind)
	record.SourceID = strings.TrimSpace(record.SourceID)
	record.CallerScenario = strings.TrimSpace(record.CallerScenario)
	record.TargetScenario = strings.TrimSpace(record.TargetScenario)
	if record.SourceKind == "" || record.SourceID == "" || record.CallerScenario == "" || record.TargetScenario == "" {
		return LegacyMigrationResult{}, errors.New("legacy migration requires source kind, source id, caller scenario, and target scenario")
	}
	target, ok := legacyReceiptState(record.State)
	if !ok {
		return a.readOnlyProjection(record, "unmappable_legacy_state", "unsupported state "+strings.TrimSpace(record.State)), nil
	}
	// Historical active work cannot safely become a producer receipt: recovery
	// would mistake it for new work and execute it again. Preserve that evidence
	// as an explicit terminal projection until an owner supplies a reattachment
	// protocol.
	if !terminal(target) {
		return a.readOnlyProjection(record, "legacy_active_state_read_only", "active state "+strings.TrimSpace(record.State)+" cannot be reattached safely"), nil
	}
	key := "legacy:" + record.SourceKind + ":" + record.SourceID
	root := "scenarios/" + record.TargetScenario
	intent := &validationv1.ValidationIntent{
		SchemaVersion: ReceiptSchemaVersion, IntentId: key, IdempotencyKey: key,
		CallerScenario:    record.CallerScenario,
		Targets:           []*commonv1.ValidationTarget{{Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO, Id: record.TargetScenario, Root: root}},
		Purpose:           validationv1.ValidationPurpose_VALIDATION_PURPOSE_INVESTIGATION,
		RequiredStrength:  validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED,
		ReusePolicy:       &validationv1.ReusePolicy{Mode: validationv1.ReuseMode_REUSE_MODE_NEVER},
		ConcurrencyPolicy: &validationv1.ConcurrencyPolicy{Mode: validationv1.ConcurrencyMode_CONCURRENCY_MODE_EXCLUSIVE, MaximumParallelism: 1},
		ExpectedIdentity:  &validationv1.SourceIdentity{SchemaVersion: 1, Identity: key},
		ContentInputs:     []*validationv1.ContentInputRoot{{Name: "legacy-target", Root: root, Selections: []*validationv1.InputSelection{{Glob: "**", Required: false}}}},
		EvidencePolicy:    &validationv1.EvidencePolicy{},
		DeadlinePolicy:    &validationv1.DeadlinePolicy{MaximumAttempts: 1, QueueBudget: durationpb.New(time.Minute), ExecutionBudget: durationpb.New(time.Minute)},
		CallerAttributes:  map[string]string{"migration_source_kind": record.SourceKind, "migration_source_id": record.SourceID},
	}
	admission, err := a.repo.Admit(ctx, intent)
	if err != nil {
		return LegacyMigrationResult{}, err
	}
	receipt := admission.Receipt
	if admission.Kind == AdmissionIdempotent || terminal(receipt.GetState()) {
		return LegacyMigrationResult{Receipt: receipt, Migrated: true, Reason: "already_migrated"}, nil
	}
	if target == validationv1.ReceiptState_RECEIPT_STATE_ADMITTED {
		return LegacyMigrationResult{Receipt: receipt, Migrated: true, Reason: "migrated"}, nil
	}
	receipt, err = a.repo.Transition(ctx, receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_QUEUED, nil)
	if err != nil {
		return LegacyMigrationResult{}, err
	}
	if target == validationv1.ReceiptState_RECEIPT_STATE_QUEUED {
		return LegacyMigrationResult{Receipt: receipt, Migrated: true, Reason: "migrated"}, nil
	}
	receipt, err = a.repo.Transition(ctx, receipt.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_RUNNING, nil)
	if err != nil {
		return LegacyMigrationResult{}, err
	}
	if target == validationv1.ReceiptState_RECEIPT_STATE_RUNNING {
		return LegacyMigrationResult{Receipt: receipt, Migrated: true, Reason: "migrated"}, nil
	}
	receipt, err = a.repo.Transition(ctx, receipt.GetReceiptId(), target, func(value *validationv1.ValidationReceipt) error {
		value.Evidence = cloneEvidence(record.Evidence)
		value.AchievedStrength = validationv1.ValidationStrength_VALIDATION_STRENGTH_TARGETED
		value.Detail = "migrated from " + record.SourceKind + ":" + record.SourceID
		if target == validationv1.ReceiptState_RECEIPT_STATE_FAILED {
			value.ReasonCode = validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_REQUIRED_EVIDENCE_FAILED
		}
		return nil
	})
	if err != nil {
		return LegacyMigrationResult{}, err
	}
	return LegacyMigrationResult{Receipt: receipt, Migrated: true, Reason: "migrated"}, nil
}

func (a *LegacyMigrationAdapter) readOnlyProjection(record LegacyValidationRecord, reason, detail string) LegacyMigrationResult {
	now := a.now().UTC()
	return LegacyMigrationResult{
		ReadOnly: true,
		Reason:   reason,
		Receipt: &validationv1.ValidationReceipt{
			SchemaVersion: ReceiptSchemaVersion,
			ReceiptId:     "legacy-readonly:" + record.SourceKind + ":" + record.SourceID,
			IntentId:      "legacy:" + record.SourceKind + ":" + record.SourceID,
			State:         validationv1.ReceiptState_RECEIPT_STATE_DEGRADED,
			ReasonCode:    validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_INTERNAL,
			Detail:        "read-only legacy evidence: " + detail,
			Evidence:      cloneEvidence(record.Evidence),
			CreatedAt:     timestamppb.New(now), UpdatedAt: timestamppb.New(now), TerminalAt: timestamppb.New(now), Revision: 1,
		},
	}
}

func legacyReceiptState(state string) (validationv1.ReceiptState, bool) {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "admitted", "created", "pending":
		return validationv1.ReceiptState_RECEIPT_STATE_ADMITTED, true
	case "queued":
		return validationv1.ReceiptState_RECEIPT_STATE_QUEUED, true
	case "running", "in_progress":
		return validationv1.ReceiptState_RECEIPT_STATE_RUNNING, true
	case "passed", "succeeded", "complete", "completed":
		return validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, true
	case "failed", "error":
		return validationv1.ReceiptState_RECEIPT_STATE_FAILED, true
	case "degraded", "partial":
		return validationv1.ReceiptState_RECEIPT_STATE_DEGRADED, true
	case "cancelled", "canceled", "aborted":
		return validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, true
	default:
		return validationv1.ReceiptState_RECEIPT_STATE_UNSPECIFIED, false
	}
}

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

func (r *Repository) RecordShadowComparison(ctx context.Context, legacy LegacyValidationRecord, receipt *validationv1.ValidationReceipt) (ShadowComparison, error) {
	if receipt == nil || strings.TrimSpace(receipt.GetReceiptId()) == "" {
		return ShadowComparison{}, errors.New("shadow comparison requires a receipt")
	}
	legacyState, mapped := legacyReceiptState(legacy.State)
	reason := "state_equivalent"
	matched := mapped && legacyState == receipt.GetState()
	if !mapped {
		reason = "unmappable_legacy_state"
	} else if terminal(legacyState) != terminal(receipt.GetState()) {
		reason = "terminality_mismatch"
	} else if legacyState != receipt.GetState() {
		reason = "verdict_mismatch"
	} else if len(legacy.Evidence) != len(receipt.GetEvidence()) {
		matched = false
		reason = "evidence_count_mismatch"
	}
	material := fmt.Sprintf("%s\x00%s\x00%s\x00%d", legacy.SourceKind, legacy.SourceID, receipt.GetReceiptId(), receipt.GetRevision())
	sum := sha256.Sum256([]byte(material))
	now := r.now().UTC()
	comparison := ShadowComparison{
		ComparisonID: "shadow:" + hex.EncodeToString(sum[:]), SourceKind: strings.TrimSpace(legacy.SourceKind), SourceID: strings.TrimSpace(legacy.SourceID),
		ReceiptID: receipt.GetReceiptId(), ReceiptRevision: receipt.GetRevision(), LegacyState: strings.TrimSpace(legacy.State), ReceiptState: receipt.GetState(),
		Matched: matched, ReasonCode: reason, LegacyEvidenceCount: len(legacy.Evidence), ReceiptEvidenceCount: len(receipt.GetEvidence()), ObservedAt: now,
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO validation_shadow_comparisons
        (comparison_id, source_kind, source_id, receipt_id, receipt_revision, legacy_state, receipt_state, matched, reason_code, legacy_evidence_count, receipt_evidence_count, observed_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(source_kind, source_id, receipt_id, receipt_revision) DO NOTHING`,
		comparison.ComparisonID, comparison.SourceKind, comparison.SourceID, comparison.ReceiptID, comparison.ReceiptRevision, comparison.LegacyState, comparison.ReceiptState,
		comparison.Matched, comparison.ReasonCode, comparison.LegacyEvidenceCount, comparison.ReceiptEvidenceCount, sqliteutil.FormatTimestamp(now))
	return comparison, err
}

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
