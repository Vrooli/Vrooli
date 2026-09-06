package validationbroker

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"test-genie/internal/dbexec"
	"test-genie/internal/storage/sqliteutil"

	"github.com/google/uuid"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ReceiptSchemaVersion = 1

var (
	ErrNotFound          = errors.New("validation receipt not found")
	ErrInvalidIntent     = errors.New("invalid validation intent")
	ErrIdempotencyKey    = errors.New("validation idempotency key was reused with different intent")
	ErrInvalidTransition = errors.New("invalid validation receipt transition")
	ErrConcurrentUpdate  = errors.New("validation receipt changed concurrently")
)

type AdmissionKind string

const (
	AdmissionNew            AdmissionKind = "new"
	AdmissionAttachedActive AdmissionKind = "attached_active"
	AdmissionReusedTerminal AdmissionKind = "reused_terminal"
	AdmissionIdempotent     AdmissionKind = "idempotent"
)

type Admission struct {
	Kind    AdmissionKind
	Receipt *validationv1.ValidationReceipt
}

type Repository struct {
	db  dbexec.Executor
	now func() time.Time
}

// ReceiptRepository is the engine-independent domain boundary used by the
// service. SQLite is the current adapter; callers never depend on its schema.
type ReceiptRepository interface {
	Admit(context.Context, *validationv1.ValidationIntent) (Admission, error)
	Get(context.Context, string) (*validationv1.ValidationReceipt, error)
	List(context.Context, ListFilter) ([]*validationv1.ValidationReceipt, int, error)
	Transition(context.Context, string, validationv1.ReceiptState, func(*validationv1.ValidationReceipt) error) (*validationv1.ValidationReceipt, error)
	PropagateTerminal(context.Context, *validationv1.ValidationReceipt) ([]string, error)
	History(context.Context, string) ([]TransitionRecord, error)
	ActiveProducers(context.Context) ([]ProducerWork, error)
}

type TransitionRecord struct {
	Revision   uint64
	From       validationv1.ReceiptState
	To         validationv1.ReceiptState
	ReasonCode validationv1.ValidationReasonCode
	Receipt    *validationv1.ValidationReceipt
	CreatedAt  time.Time
}

type ProducerWork struct {
	Receipt *validationv1.ValidationReceipt
	Intent  *validationv1.ValidationIntent
}

func NewRepository(db dbexec.Executor) *Repository {
	return &Repository{db: db, now: time.Now}
}

func (r *Repository) Admit(ctx context.Context, input *validationv1.ValidationIntent) (Admission, error) {
	intent, err := normalizeIntent(input)
	if err != nil {
		return Admission{}, err
	}
	intentFingerprint, err := fingerprintIntent(intent, false)
	if err != nil {
		return Admission{}, err
	}
	executionKey, err := fingerprintIntent(intent, true)
	if err != nil {
		return Admission{}, err
	}
	if intent.GetReusePolicy().GetMode() == validationv1.ReuseMode_REUSE_MODE_NEVER || intent.GetConcurrencyPolicy().GetMode() == validationv1.ConcurrencyMode_CONCURRENCY_MODE_EXCLUSIVE {
		sum := sha256.Sum256([]byte(executionKey + "\x00" + intent.GetIntentId()))
		executionKey = "sha256:" + hex.EncodeToString(sum[:])
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Admission{}, err
	}
	defer tx.Rollback()

	if existing, storedFingerprint, getErr := getByIdempotency(ctx, tx, intent.GetCallerScenario(), intent.GetIdempotencyKey()); getErr == nil {
		if storedFingerprint != intentFingerprint {
			return Admission{}, ErrIdempotencyKey
		}
		return Admission{Kind: AdmissionIdempotent, Receipt: existing}, tx.Commit()
	} else if !errors.Is(getErr, ErrNotFound) {
		return Admission{}, getErr
	}

	compatible, compatibility, findErr := findCompatible(ctx, tx, intent, executionKey, r.now().UTC())
	if findErr != nil && !errors.Is(findErr, ErrNotFound) {
		return Admission{}, findErr
	}
	now := r.now().UTC()
	receipt := &validationv1.ValidationReceipt{
		SchemaVersion: ReceiptSchemaVersion,
		ReceiptId:     uuid.NewString(),
		IntentId:      intent.GetIntentId(),
		State:         validationv1.ReceiptState_RECEIPT_STATE_ADMITTED,
		ReasonCode:    validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE,
		CreatedAt:     timestamppb.New(now),
		UpdatedAt:     timestamppb.New(now),
		Revision:      1,
	}
	kind, producer := AdmissionNew, true
	if compatible != nil {
		receipt.LineageId = compatible.GetLineageId()
		receipt.Compatibility = compatibility
		receipt.Children = cloneChildren(compatible.GetChildren())
		if terminal(compatible.GetState()) {
			kind, producer = AdmissionReusedTerminal, false
			receipt.State = compatible.GetState()
			receipt.AchievedStrength = compatible.GetAchievedStrength()
			receipt.AdmittedIdentity = cloneIdentity(compatible.GetAdmittedIdentity())
			receipt.ObservedIdentity = cloneIdentity(compatible.GetObservedIdentity())
			receipt.Evidence = cloneEvidence(compatible.GetEvidence())
			receipt.ReasonCode = compatible.GetReasonCode()
			receipt.Detail = "reused compatible terminal receipt " + compatible.GetReceiptId()
			receipt.TerminalAt = compatible.GetTerminalAt()
		} else {
			kind, producer = AdmissionAttachedActive, false
			receipt.State = validationv1.ReceiptState_RECEIPT_STATE_ATTACHED
			receipt.AdmittedIdentity = cloneIdentity(compatible.GetAdmittedIdentity())
			receipt.Detail = "attached to compatible active receipt " + compatible.GetReceiptId()
		}
	} else {
		receipt.LineageId = uuid.NewString()
		receipt.Compatibility = &validationv1.CompatibilityDecision{
			Kind:         validationv1.CompatibilityKind_COMPATIBILITY_KIND_NEW_WORK,
			ExecutionKey: executionKey,
			ReasonCode:   validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE,
		}
		receipt.AdmittedIdentity = cloneIdentity(intent.GetExpectedIdentity())
	}
	if err := insertReceipt(ctx, tx, intent, receipt, intentFingerprint, executionKey, producer); err != nil {
		return Admission{}, err
	}
	if err := tx.Commit(); err != nil {
		return Admission{}, err
	}
	return Admission{Kind: kind, Receipt: receipt}, nil
}

func (r *Repository) Get(ctx context.Context, receiptID string) (*validationv1.ValidationReceipt, error) {
	var payload []byte
	if err := r.db.QueryRowContext(ctx, `SELECT receipt_proto FROM validation_receipts WHERE receipt_id = ?`, strings.TrimSpace(receiptID)).Scan(&payload); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return decodeReceipt(payload)
}

type ListFilter struct {
	CallerScenario    string
	CallerExecutionID string
	PlanID            string
	State             validationv1.ReceiptState
	Limit             int
	Offset            int
}

func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*validationv1.ValidationReceipt, int, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `SELECT receipt_proto FROM validation_receipts WHERE (? = '' OR caller_scenario = ?) AND (? = '' OR caller_execution_id = ?) AND (? = '' OR plan_id = ?) AND (? = 0 OR state = ?) ORDER BY created_at DESC, receipt_id DESC LIMIT ? OFFSET ?`,
		filter.CallerScenario, filter.CallerScenario,
		filter.CallerExecutionID, filter.CallerExecutionID,
		filter.PlanID, filter.PlanID,
		filter.State, filter.State, limit+1, filter.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var receipts []*validationv1.ValidationReceipt
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, 0, err
		}
		receipt, err := decodeReceipt(payload)
		if err != nil {
			return nil, 0, err
		}
		receipts = append(receipts, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	next := 0
	if len(receipts) > limit {
		receipts = receipts[:limit]
		next = filter.Offset + limit
	}
	return receipts, next, nil
}

func ParsePageToken(token string) (int, error) {
	if strings.TrimSpace(token) == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid page token")
	}
	return offset, nil
}

// Transition applies one compare-and-swap state transition. Repeating an
// already-applied transition is idempotent; every other edge must be present
// in the closed receipt state machine.
func (r *Repository) Transition(ctx context.Context, receiptID string, next validationv1.ReceiptState, mutate func(*validationv1.ValidationReceipt) error) (*validationv1.ValidationReceipt, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var payload []byte
	var revision uint64
	var producer int
	err = tx.QueryRowContext(ctx, `SELECT receipt_proto, revision, is_producer FROM validation_receipts WHERE receipt_id = ?`, strings.TrimSpace(receiptID)).Scan(&payload, &revision, &producer)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	receipt, err := decodeReceipt(payload)
	if err != nil {
		return nil, err
	}
	if receipt.GetState() == next && (mutate == nil || terminal(next)) {
		return receipt, tx.Commit()
	}
	if receipt.GetState() != next && !CanTransition(receipt.GetState(), next) {
		return nil, fmt.Errorf("%w: %s to %s", ErrInvalidTransition, receipt.GetState(), next)
	}
	if mutate != nil {
		if err := mutate(receipt); err != nil {
			return nil, err
		}
	}
	receipt.State = next
	receipt.Revision = revision + 1
	now := r.now().UTC()
	receipt.UpdatedAt = timestamppb.New(now)
	if terminal(next) {
		receipt.TerminalAt = timestamppb.New(now)
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(receipt)
	if err != nil {
		return nil, err
	}
	if terminal(next) {
		producer = 0
	}
	result, err := tx.ExecContext(ctx, `UPDATE validation_receipts SET state = ?, is_producer = ?, receipt_proto = ?, updated_at = ?, revision = ? WHERE receipt_id = ? AND revision = ?`, next, producer, encoded, sqliteutil.FormatTimestamp(now), receipt.GetRevision(), receipt.GetReceiptId(), revision)
	if err != nil {
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows != 1 {
		return nil, ErrConcurrentUpdate
	}
	if err := insertTransition(ctx, tx, receipt, validationv1.ReceiptState(payloadState(payload)), next, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return receipt, nil
}

func (r *Repository) History(ctx context.Context, receiptID string) ([]TransitionRecord, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT revision, from_state, to_state, reason_code, receipt_proto, created_at FROM validation_receipt_transitions WHERE receipt_id = ? ORDER BY revision`, strings.TrimSpace(receiptID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []TransitionRecord
	for rows.Next() {
		var record TransitionRecord
		var payload []byte
		var created string
		if err := rows.Scan(&record.Revision, &record.From, &record.To, &record.ReasonCode, &payload, &created); err != nil {
			return nil, err
		}
		record.Receipt, err = decodeReceipt(payload)
		if err != nil {
			return nil, err
		}
		record.CreatedAt, err = sqliteutil.ParseTimestamp(created)
		if err != nil {
			return nil, err
		}
		history = append(history, record)
	}
	return history, rows.Err()
}

func (r *Repository) ActiveProducers(ctx context.Context) ([]ProducerWork, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT receipt_proto, intent_proto FROM validation_receipts WHERE is_producer = 1 AND state IN (?, ?, ?, ?, ?) ORDER BY created_at, receipt_id`,
		validationv1.ReceiptState_RECEIPT_STATE_ADMITTED,
		validationv1.ReceiptState_RECEIPT_STATE_ATTACHED,
		validationv1.ReceiptState_RECEIPT_STATE_QUEUED,
		validationv1.ReceiptState_RECEIPT_STATE_RUNNING,
		validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var work []ProducerWork
	for rows.Next() {
		var receiptPayload, intentPayload []byte
		if err := rows.Scan(&receiptPayload, &intentPayload); err != nil {
			return nil, err
		}
		receipt, err := decodeReceipt(receiptPayload)
		if err != nil {
			return nil, err
		}
		var intent validationv1.ValidationIntent
		if err := proto.Unmarshal(intentPayload, &intent); err != nil {
			return nil, fmt.Errorf("decode validation intent: %w", err)
		}
		work = append(work, ProducerWork{Receipt: receipt, Intent: &intent})
	}
	return work, rows.Err()
}

// PropagateTerminal gives every attached observer its own durable terminal
// receipt while preserving the observer's identity and compatibility decision.
// Evidence is copied, never aliased to the producer row.
func (r *Repository) PropagateTerminal(ctx context.Context, producer *validationv1.ValidationReceipt) ([]string, error) {
	if producer == nil || !terminal(producer.GetState()) {
		return nil, nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT receipt_id, receipt_proto, revision FROM validation_receipts WHERE lineage_id = ? AND receipt_id <> ? AND state = ?`, producer.GetLineageId(), producer.GetReceiptId(), validationv1.ReceiptState_RECEIPT_STATE_ATTACHED)
	if err != nil {
		return nil, err
	}
	type attachedRow struct {
		id       string
		receipt  *validationv1.ValidationReceipt
		revision uint64
	}
	var attached []attachedRow
	for rows.Next() {
		var id string
		var payload []byte
		var revision uint64
		if err := rows.Scan(&id, &payload, &revision); err != nil {
			rows.Close()
			return nil, err
		}
		receipt, err := decodeReceipt(payload)
		if err != nil {
			rows.Close()
			return nil, err
		}
		attached = append(attached, attachedRow{id: id, receipt: receipt, revision: revision})
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	now := r.now().UTC()
	updated := make([]string, 0, len(attached))
	for _, row := range attached {
		receipt := row.receipt
		receipt.State = producer.GetState()
		receipt.AchievedStrength = producer.GetAchievedStrength()
		receipt.ObservedIdentity = cloneIdentity(producer.GetObservedIdentity())
		receipt.Evidence = cloneEvidence(producer.GetEvidence())
		receipt.Children = cloneChildren(producer.GetChildren())
		receipt.Retry = cloneRetry(producer.GetRetry())
		receipt.Degradation = cloneDegradation(producer.GetDegradation())
		receipt.ReasonCode = producer.GetReasonCode()
		receipt.Detail = "lineage producer " + producer.GetReceiptId() + ": " + producer.GetDetail()
		receipt.UpdatedAt = timestamppb.New(now)
		receipt.TerminalAt = timestamppb.New(now)
		receipt.Revision = row.revision + 1
		payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(receipt)
		if err != nil {
			return nil, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE validation_receipts SET state = ?, receipt_proto = ?, updated_at = ?, revision = ? WHERE receipt_id = ? AND revision = ?`, receipt.GetState(), payload, sqliteutil.FormatTimestamp(now), receipt.GetRevision(), row.id, row.revision)
		if err != nil {
			return nil, err
		}
		count, err := result.RowsAffected()
		if err != nil || count != 1 {
			if err != nil {
				return nil, err
			}
			return nil, ErrConcurrentUpdate
		}
		if err := insertTransition(ctx, tx, receipt, validationv1.ReceiptState_RECEIPT_STATE_ATTACHED, receipt.GetState(), now); err != nil {
			return nil, err
		}
		updated = append(updated, row.id)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

// CanTransition is the executable receipt state machine. It is exported so
// adapters and architecture tests use the same decision instead of copying a
// switch over lifecycle strings.
func CanTransition(from, to validationv1.ReceiptState) bool {
	allowed := map[validationv1.ReceiptState]map[validationv1.ReceiptState]bool{
		validationv1.ReceiptState_RECEIPT_STATE_ADMITTED: {
			validationv1.ReceiptState_RECEIPT_STATE_ATTACHED:   true,
			validationv1.ReceiptState_RECEIPT_STATE_QUEUED:     true,
			validationv1.ReceiptState_RECEIPT_STATE_RUNNING:    true,
			validationv1.ReceiptState_RECEIPT_STATE_CANCELLED:  true,
			validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED: true,
		},
		validationv1.ReceiptState_RECEIPT_STATE_ATTACHED: {
			validationv1.ReceiptState_RECEIPT_STATE_RUNNING:    true,
			validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED:  true,
			validationv1.ReceiptState_RECEIPT_STATE_FAILED:     true,
			validationv1.ReceiptState_RECEIPT_STATE_DEGRADED:   true,
			validationv1.ReceiptState_RECEIPT_STATE_CANCELLED:  true,
			validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED: true,
		},
		validationv1.ReceiptState_RECEIPT_STATE_QUEUED: {
			validationv1.ReceiptState_RECEIPT_STATE_RUNNING:    true,
			validationv1.ReceiptState_RECEIPT_STATE_FAILED:     true,
			validationv1.ReceiptState_RECEIPT_STATE_DEGRADED:   true,
			validationv1.ReceiptState_RECEIPT_STATE_CANCELLED:  true,
			validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED: true,
		},
		validationv1.ReceiptState_RECEIPT_STATE_RUNNING: {
			validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING: true,
			validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED:     true,
			validationv1.ReceiptState_RECEIPT_STATE_FAILED:        true,
			validationv1.ReceiptState_RECEIPT_STATE_DEGRADED:      true,
			validationv1.ReceiptState_RECEIPT_STATE_CANCELLED:     true,
		},
		validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING: {
			validationv1.ReceiptState_RECEIPT_STATE_QUEUED:     true,
			validationv1.ReceiptState_RECEIPT_STATE_CANCELLED:  true,
			validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED: true,
		},
	}
	return allowed[from][to]
}

func normalizeIntent(input *validationv1.ValidationIntent) (*validationv1.ValidationIntent, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: intent is required", ErrInvalidIntent)
	}
	intent := proto.Clone(input).(*validationv1.ValidationIntent)
	// Historical records used an attribution map for an execution-affecting input.
	// Normalize them at this single intake seam; all current writers use the field.
	legacyPrior := strings.TrimSpace(intent.GetCallerAttributes()["baseline_name"])
	intent.BehavioralPrior = strings.TrimSpace(intent.GetBehavioralPrior())
	if intent.BehavioralPrior != "" && legacyPrior != "" && intent.BehavioralPrior != legacyPrior {
		return nil, fmt.Errorf("%w: conflicting behavioral prior identities", ErrInvalidIntent)
	}
	if intent.BehavioralPrior == "" {
		intent.BehavioralPrior = legacyPrior
	}
	delete(intent.CallerAttributes, "baseline_name")
	if intent.EvidencePolicy == nil {
		return nil, fmt.Errorf("%w: evidence policy is required", ErrInvalidIntent)
	}
	if len(intent.GetPhases()) > 0 && (intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_CERTIFICATION || intent.GetRequiredStrength() >= validationv1.ValidationStrength_VALIDATION_STRENGTH_COMPREHENSIVE) {
		return nil, fmt.Errorf("%w: explicit phase selection cannot claim comprehensive or certification strength", ErrInvalidIntent)
	}
	for i, phase := range intent.Phases {
		intent.Phases[i] = strings.TrimSpace(phase)
		if intent.Phases[i] == "" {
			return nil, fmt.Errorf("%w: phase names must not be empty", ErrInvalidIntent)
		}
	}
	sort.Strings(intent.Phases)
	if intent.GetSchemaVersion() == 0 {
		intent.SchemaVersion = ReceiptSchemaVersion
	}
	if intent.GetSchemaVersion() != ReceiptSchemaVersion {
		return nil, fmt.Errorf("%w: unsupported schema version %d", ErrInvalidIntent, intent.GetSchemaVersion())
	}
	intent.IntentId = strings.TrimSpace(intent.GetIntentId())
	if intent.IntentId == "" {
		intent.IntentId = uuid.NewString()
	}
	intent.IdempotencyKey = strings.TrimSpace(intent.GetIdempotencyKey())
	intent.CallerScenario = strings.TrimSpace(intent.GetCallerScenario())
	if intent.IdempotencyKey == "" || intent.CallerScenario == "" {
		return nil, fmt.Errorf("%w: caller_scenario and idempotency_key are required", ErrInvalidIntent)
	}
	if len(intent.GetTargets()) == 0 {
		return nil, fmt.Errorf("%w: at least one target is required", ErrInvalidIntent)
	}
	seenTargets := map[string]struct{}{}
	for _, target := range intent.GetTargets() {
		if target == nil || target.GetKind() == commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_UNSPECIFIED || strings.TrimSpace(target.GetId()) == "" {
			return nil, fmt.Errorf("%w: every target requires kind and id", ErrInvalidIntent)
		}
		key := fmt.Sprintf("%d:%s:%s", target.GetKind(), strings.TrimSpace(target.GetId()), strings.TrimSpace(target.GetRoot()))
		if _, duplicate := seenTargets[key]; duplicate {
			return nil, fmt.Errorf("%w: duplicate target %s", ErrInvalidIntent, key)
		}
		seenTargets[key] = struct{}{}
	}
	if intent.GetPurpose() == validationv1.ValidationPurpose_VALIDATION_PURPOSE_UNSPECIFIED || intent.GetRequiredStrength() == validationv1.ValidationStrength_VALIDATION_STRENGTH_UNSPECIFIED {
		return nil, fmt.Errorf("%w: purpose and required_strength are required", ErrInvalidIntent)
	}
	if intent.GetExpectedIdentity() == nil || intent.GetExpectedIdentity().GetSchemaVersion() == 0 || strings.TrimSpace(intent.GetExpectedIdentity().GetIdentity()) == "" {
		return nil, fmt.Errorf("%w: expected content identity is required", ErrInvalidIntent)
	}
	if len(intent.GetContentInputs()) == 0 {
		return nil, fmt.Errorf("%w: at least one content input root is required", ErrInvalidIntent)
	}
	primaryRoots := 0
	seenInputRoots := map[string]struct{}{}
	for _, root := range intent.GetContentInputs() {
		if root == nil || strings.TrimSpace(root.GetName()) == "" || strings.TrimSpace(root.GetRoot()) == "" || len(root.GetSelections()) == 0 {
			return nil, fmt.Errorf("%w: every content input requires name, root, and selections", ErrInvalidIntent)
		}
		if _, exists := seenInputRoots[root.GetName()]; exists {
			return nil, fmt.Errorf("%w: duplicate content input root %q", ErrInvalidIntent, root.GetName())
		}
		seenInputRoots[root.GetName()] = struct{}{}
		if !root.GetDependency() {
			primaryRoots++
		}
		for _, selection := range root.GetSelections() {
			if selection == nil || strings.TrimSpace(selection.GetGlob()) == "" || strings.HasPrefix(strings.TrimSpace(selection.GetGlob()), "!") {
				return nil, fmt.Errorf("%w: content input selections require a non-negated glob", ErrInvalidIntent)
			}
		}
	}
	if primaryRoots != 1 {
		return nil, fmt.Errorf("%w: exactly one content input must be primary", ErrInvalidIntent)
	}
	if intent.GetReusePolicy() == nil || intent.GetReusePolicy().GetMode() == validationv1.ReuseMode_REUSE_MODE_UNSPECIFIED {
		return nil, fmt.Errorf("%w: reuse policy is required", ErrInvalidIntent)
	}
	if intent.GetConcurrencyPolicy() == nil || intent.GetConcurrencyPolicy().GetMode() == validationv1.ConcurrencyMode_CONCURRENCY_MODE_UNSPECIFIED {
		return nil, fmt.Errorf("%w: concurrency policy is required", ErrInvalidIntent)
	}
	if intent.GetDeadlinePolicy() == nil || intent.GetDeadlinePolicy().GetMaximumAttempts() == 0 {
		return nil, fmt.Errorf("%w: deadline policy with positive maximum_attempts is required", ErrInvalidIntent)
	}
	sort.Slice(intent.Targets, func(i, j int) bool {
		left, right := intent.Targets[i], intent.Targets[j]
		if left.GetKind() != right.GetKind() {
			return left.GetKind() < right.GetKind()
		}
		if left.GetId() != right.GetId() {
			return left.GetId() < right.GetId()
		}
		return left.GetRoot() < right.GetRoot()
	})
	sort.Strings(intent.EvidencePolicy.RequiredEvidenceKinds)
	sort.Slice(intent.ContentInputs, func(i, j int) bool {
		if intent.ContentInputs[i].GetDependency() != intent.ContentInputs[j].GetDependency() {
			return !intent.ContentInputs[i].GetDependency()
		}
		return intent.ContentInputs[i].GetName() < intent.ContentInputs[j].GetName()
	})
	return intent, nil
}

func fingerprintIntent(intent *validationv1.ValidationIntent, execution bool) (string, error) {
	value := proto.Clone(intent).(*validationv1.ValidationIntent)
	value.IntentId = ""
	value.IdempotencyKey = ""
	if identity := value.ExpectedIdentity; identity != nil {
		value.ExpectedIdentity = &validationv1.SourceIdentity{SchemaVersion: identity.GetSchemaVersion(), Identity: identity.GetIdentity()}
	}
	if execution {
		value.CallerScenario = ""
		value.CallerExecutionId = ""
		value.PlanId = ""
		value.PhaseId = ""
		value.CallerAttributes = nil
	}
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func getByIdempotency(ctx context.Context, tx *sql.Tx, caller, key string) (*validationv1.ValidationReceipt, string, error) {
	var payload, intentPayload []byte
	err := tx.QueryRowContext(ctx, `SELECT receipt_proto, intent_proto FROM validation_receipts WHERE caller_scenario = ? AND idempotency_key = ?`, caller, key).Scan(&payload, &intentPayload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	// Compare retained intent under today's semantic rules without rewriting
	// historical evidence or breaking an observer's durable idempotency key.
	stored := &validationv1.ValidationIntent{}
	if err := proto.Unmarshal(intentPayload, stored); err != nil {
		return nil, "", err
	}
	stored, err = normalizeIntent(stored)
	if err != nil {
		return nil, "", err
	}
	fingerprint, err := fingerprintIntent(stored, false)
	if err != nil {
		return nil, "", err
	}
	receipt, err := decodeReceipt(payload)
	return receipt, fingerprint, err
}

func findCompatible(ctx context.Context, tx *sql.Tx, intent *validationv1.ValidationIntent, key string, now time.Time) (*validationv1.ValidationReceipt, *validationv1.CompatibilityDecision, error) {
	mode := intent.GetReusePolicy().GetMode()
	if mode == validationv1.ReuseMode_REUSE_MODE_NEVER {
		return nil, nil, ErrNotFound
	}
	states := []validationv1.ReceiptState{}
	if mode == validationv1.ReuseMode_REUSE_MODE_ATTACH_ACTIVE || mode == validationv1.ReuseMode_REUSE_MODE_ATTACH_OR_TERMINAL {
		states = append(states, validationv1.ReceiptState_RECEIPT_STATE_ADMITTED, validationv1.ReceiptState_RECEIPT_STATE_QUEUED, validationv1.ReceiptState_RECEIPT_STATE_RUNNING, validationv1.ReceiptState_RECEIPT_STATE_RETRY_PENDING)
	}
	if mode == validationv1.ReuseMode_REUSE_MODE_COMPATIBLE_TERMINAL || mode == validationv1.ReuseMode_REUSE_MODE_ATTACH_OR_TERMINAL {
		states = append(states, validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, validationv1.ReceiptState_RECEIPT_STATE_DEGRADED)
	}
	for _, state := range states {
		var payload []byte
		err := tx.QueryRowContext(ctx, `SELECT receipt_proto FROM validation_receipts WHERE execution_key = ? AND state = ? ORDER BY is_producer DESC, created_at DESC LIMIT 1`, key, state).Scan(&payload)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		receipt, err := decodeReceipt(payload)
		if err != nil {
			return nil, nil, err
		}
		if terminal(state) && intent.GetReusePolicy().GetMaximumAge() != nil {
			age := intent.GetReusePolicy().GetMaximumAge().AsDuration()
			if age <= 0 || receipt.GetTerminalAt() == nil || now.Sub(receipt.GetTerminalAt().AsTime()) > age {
				continue
			}
		}
		kind := validationv1.CompatibilityKind_COMPATIBILITY_KIND_ATTACHED_ACTIVE
		if terminal(state) {
			kind = validationv1.CompatibilityKind_COMPATIBILITY_KIND_REUSED_TERMINAL
		}
		return receipt, &validationv1.CompatibilityDecision{Kind: kind, CompatibleReceiptId: receipt.GetReceiptId(), ExecutionKey: key, ReasonCode: validationv1.ValidationReasonCode_VALIDATION_REASON_CODE_NONE}, nil
	}
	return nil, nil, ErrNotFound
}

func insertReceipt(ctx context.Context, tx *sql.Tx, intent *validationv1.ValidationIntent, receipt *validationv1.ValidationReceipt, intentFingerprint, executionKey string, producer bool) error {
	intentBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(intent)
	if err != nil {
		return err
	}
	receiptBytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(receipt)
	if err != nil {
		return err
	}
	producerInt := 0
	if producer {
		producerInt = 1
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO validation_receipts (receipt_id, lineage_id, intent_id, caller_scenario, caller_execution_id, plan_id, idempotency_key, intent_fingerprint, execution_key, state, is_producer, intent_proto, receipt_proto, created_at, updated_at, revision) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, receipt.GetReceiptId(), receipt.GetLineageId(), intent.GetIntentId(), intent.GetCallerScenario(), intent.GetCallerExecutionId(), intent.GetPlanId(), intent.GetIdempotencyKey(), intentFingerprint, executionKey, receipt.GetState(), producerInt, intentBytes, receiptBytes, sqliteutil.FormatTimestamp(receipt.GetCreatedAt().AsTime()), sqliteutil.FormatTimestamp(receipt.GetUpdatedAt().AsTime()), receipt.GetRevision())
	if err != nil {
		return err
	}
	return insertTransition(ctx, tx, receipt, validationv1.ReceiptState_RECEIPT_STATE_UNSPECIFIED, receipt.GetState(), receipt.GetCreatedAt().AsTime())
}

func insertTransition(ctx context.Context, tx *sql.Tx, receipt *validationv1.ValidationReceipt, from, to validationv1.ReceiptState, at time.Time) error {
	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(receipt)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO validation_receipt_transitions (receipt_id, revision, from_state, to_state, reason_code, receipt_proto, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, receipt.GetReceiptId(), receipt.GetRevision(), from, to, receipt.GetReasonCode(), payload, sqliteutil.FormatTimestamp(at))
	return err
}

func payloadState(payload []byte) int32 {
	receipt, err := decodeReceipt(payload)
	if err != nil {
		return int32(validationv1.ReceiptState_RECEIPT_STATE_UNSPECIFIED)
	}
	return int32(receipt.GetState())
}

func decodeReceipt(payload []byte) (*validationv1.ValidationReceipt, error) {
	var receipt validationv1.ValidationReceipt
	if err := proto.Unmarshal(payload, &receipt); err != nil {
		return nil, fmt.Errorf("decode validation receipt: %w", err)
	}
	return &receipt, nil
}

func terminal(state validationv1.ReceiptState) bool {
	switch state {
	case validationv1.ReceiptState_RECEIPT_STATE_SUCCEEDED, validationv1.ReceiptState_RECEIPT_STATE_FAILED, validationv1.ReceiptState_RECEIPT_STATE_DEGRADED, validationv1.ReceiptState_RECEIPT_STATE_CANCELLED, validationv1.ReceiptState_RECEIPT_STATE_SUPERSEDED:
		return true
	default:
		return false
	}
}

func cloneIdentity(value *validationv1.SourceIdentity) *validationv1.SourceIdentity {
	if value == nil {
		return nil
	}
	return proto.Clone(value).(*validationv1.SourceIdentity)
}

func cloneChildren(values []*validationv1.ChildOperation) []*validationv1.ChildOperation {
	out := make([]*validationv1.ChildOperation, 0, len(values))
	for _, value := range values {
		out = append(out, proto.Clone(value).(*validationv1.ChildOperation))
	}
	return out
}

func cloneEvidence(values []*validationv1.EvidenceReference) []*validationv1.EvidenceReference {
	out := make([]*validationv1.EvidenceReference, 0, len(values))
	for _, value := range values {
		out = append(out, proto.Clone(value).(*validationv1.EvidenceReference))
	}
	return out
}

func cloneRetry(value *validationv1.RetryDisposition) *validationv1.RetryDisposition {
	if value == nil {
		return nil
	}
	return proto.Clone(value).(*validationv1.RetryDisposition)
}

func cloneDegradation(value *validationv1.Degradation) *validationv1.Degradation {
	if value == nil {
		return nil
	}
	return proto.Clone(value).(*validationv1.Degradation)
}
