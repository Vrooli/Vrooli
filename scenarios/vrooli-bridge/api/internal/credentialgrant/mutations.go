package credentialgrant

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type sqlBeginner interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}
type mutationRecord struct {
	Receipt   MutationReceipt
	Operation string
}

var ErrMutationConflict = errors.New("credential grant request ID conflict")
var ErrGrantRevisionConflict = errors.New("credential grant revision conflict")

const grantColumns = "id,node_id,logical_id,field,class,retention,generation,revision,granted_at,revoked_at,acked_generation,receipt_at,receipt_accepted,receipt_reason,purge_state,purge_receipt_at,purge_accepted,purge_reason"

func (r *SQLiteRepository) CreateWithReceipt(ctx context.Context, grant Grant, binding MutationBinding) (Grant, MutationReceipt, error) {
	tx, err := r.begin(ctx)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	defer tx.Rollback()
	prior, found, err := readMutation(ctx, tx, binding.RequestID)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if found {
		if !mutationMatches(prior, binding, binding.Subject) {
			return Grant{}, MutationReceipt{}, ErrMutationConflict
		}
		grant, err = readGrant(ctx, tx, prior.Receipt.GrantID)
		if err != nil {
			return Grant{}, MutationReceipt{}, err
		}
		prior.Receipt.Replayed = true
		return grant, prior.Receipt, nil
	}
	if err = validateTargetTx(ctx, tx, binding); err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	var exists int
	err = tx.QueryRowContext(ctx, "SELECT 1 FROM credential_grants WHERE node_id=? AND logical_id=? AND field=?", grant.NodeID, grant.LogicalID, grant.Field).Scan(&exists)
	if err == nil {
		return Grant{}, MutationReceipt{}, ErrGrantRevisionConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Grant{}, MutationReceipt{}, err
	}
	if grant.ID == "" {
		grant.ID = uuid.NewString()
	}
	if grant.GrantedAt.IsZero() {
		grant.GrantedAt = r.now().UTC()
	}
	if grant.Revision <= 0 {
		grant.Revision = 1
	}
	if grant.Generation <= 0 {
		grant.Generation = 1
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO credential_grants (id,node_id,logical_id,field,class,retention,generation,revision,granted_at,revoked_at,acked_generation,receipt_at,receipt_accepted,receipt_reason,purge_state,purge_receipt_at,purge_accepted,purge_reason) VALUES (?,?,?,?,?,?,?,?,?,'',?,'',?,?,?,'',?,?)`, grant.ID, grant.NodeID, grant.LogicalID, grant.Field, grant.Class, grant.Retention, grant.Generation, grant.Revision, grant.GrantedAt.UTC().Format(time.RFC3339Nano), grant.AckedGeneration, grant.ReceiptAccepted, grant.ReceiptReason, grant.PurgeState, grant.PurgeAccepted, grant.PurgeReason)
	if err != nil {
		return Grant{}, MutationReceipt{}, fmt.Errorf("create credential grant: %w", err)
	}
	receipt := MutationReceipt{RequestID: binding.RequestID, ActorID: binding.ActorID, NodeID: binding.NodeID, EnrollmentGeneration: binding.EnrollmentGeneration, Subject: CreateGrantSubject, ExpectedRevision: binding.ExpectedRevision, ResultRevision: grant.Revision, PayloadDigest: binding.PayloadDigest, GrantID: grant.ID, Outcome: "created", DeliveryState: "pending", CreatedAt: r.now().UTC()}
	if err = writeMutation(ctx, tx, receipt, CreateGrantSubject); err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return Grant{}, MutationReceipt{}, fmt.Errorf("commit credential grant creation: %w", err)
	}
	return grant, receipt, nil
}

func (r *SQLiteRepository) RevokeWithReceipt(ctx context.Context, id string, binding MutationBinding) (Grant, MutationReceipt, error) {
	tx, err := r.begin(ctx)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	defer tx.Rollback()
	prior, found, err := readMutation(ctx, tx, binding.RequestID)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if found {
		if !mutationMatches(prior, binding, binding.Subject) || prior.Receipt.GrantID != id {
			return Grant{}, MutationReceipt{}, ErrMutationConflict
		}
		grant, err := readGrant(ctx, tx, id)
		if err != nil {
			return Grant{}, MutationReceipt{}, err
		}
		prior.Receipt.Replayed = true
		return grant, prior.Receipt, nil
	}
	if err = validateTargetTx(ctx, tx, binding); err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	grant, err := readGrant(ctx, tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Grant{}, MutationReceipt{}, sql.ErrNoRows
	}
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if grant.NodeID != binding.NodeID {
		return Grant{}, MutationReceipt{}, sql.ErrNoRows
	}
	if grant.Revision != binding.ExpectedRevision || !grant.RevokedAt.IsZero() {
		return Grant{}, MutationReceipt{}, ErrGrantRevisionConflict
	}
	now := r.now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE credential_grants SET revoked_at=?,purge_state='pending',revision=revision+1 WHERE id=? AND node_id=? AND revision=? AND COALESCE(revoked_at,'')=''`, now.Format(time.RFC3339Nano), id, binding.NodeID, binding.ExpectedRevision)
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if affected != 1 {
		return Grant{}, MutationReceipt{}, ErrGrantRevisionConflict
	}
	grant.RevokedAt = now
	grant.PurgeState = "pending"
	grant.Revision++
	receipt := MutationReceipt{RequestID: binding.RequestID, ActorID: binding.ActorID, NodeID: binding.NodeID, EnrollmentGeneration: binding.EnrollmentGeneration, Subject: binding.Subject, ExpectedRevision: binding.ExpectedRevision, ResultRevision: grant.Revision, PayloadDigest: binding.PayloadDigest, GrantID: id, Outcome: "revoked", DeliveryState: "pending", CreatedAt: now}
	if err = writeMutation(ctx, tx, receipt, RevokeGrantSubject); err != nil {
		return Grant{}, MutationReceipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return Grant{}, MutationReceipt{}, fmt.Errorf("commit credential grant revocation: %w", err)
	}
	return grant, receipt, nil
}

func (r *SQLiteRepository) SetMutationDeliveryState(ctx context.Context, requestID, state string) error {
	if state != "dispatched" && state != "retryable" {
		return fmt.Errorf("invalid credential mutation delivery state %q", state)
	}
	result, err := r.db.ExecContext(ctx, `UPDATE credential_grant_mutation_receipts SET delivery_state=? WHERE request_id=? AND delivery_state IN ('pending','dispatched','retryable')`, state, requestID)
	if err != nil {
		return fmt.Errorf("record credential mutation delivery state: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("credential mutation receipt is missing or already terminal")
	}
	return nil
}

func (r *SQLiteRepository) begin(ctx context.Context) (*sql.Tx, error) {
	db, ok := r.db.(sqlBeginner)
	if !ok {
		return nil, errors.New("credential grant durable transaction support is unavailable")
	}
	return db.BeginTx(ctx, &sql.TxOptions{})
}

func validateTargetTx(ctx context.Context, tx *sql.Tx, b MutationBinding) error {
	var generation int64
	var revokedAt, kind string
	err := tx.QueryRowContext(ctx, "SELECT enrollment_generation,revoked_at,kind FROM nodes WHERE id=?", b.NodeID).Scan(&generation, &revokedAt, &kind)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrStaleNodeGeneration{NodeID: b.NodeID}
	}
	if err != nil {
		return fmt.Errorf("read Bridge target binding: %w", err)
	}
	if generation <= 0 || generation != b.EnrollmentGeneration || strings.TrimSpace(revokedAt) != "" {
		return ErrStaleNodeGeneration{NodeID: b.NodeID}
	}
	if strings.HasPrefix(b.Subject, CreateGrantSubject) && kind == "control_plane" {
		return errors.New("control-plane node cannot receive credential grants")
	}
	return nil
}

func readMutation(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, requestID string) (mutationRecord, bool, error) {
	var m mutationRecord
	var created string
	err := q.QueryRowContext(ctx, `SELECT actor_id,request_id,operation,node_id,enrollment_generation,subject,expected_revision,result_revision,payload_digest,grant_id,outcome,delivery_state,created_at FROM credential_grant_mutation_receipts WHERE request_id=?`, requestID).Scan(&m.Receipt.ActorID, &m.Receipt.RequestID, &m.Operation, &m.Receipt.NodeID, &m.Receipt.EnrollmentGeneration, &m.Receipt.Subject, &m.Receipt.ExpectedRevision, &m.Receipt.ResultRevision, &m.Receipt.PayloadDigest, &m.Receipt.GrantID, &m.Receipt.Outcome, &m.Receipt.DeliveryState, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return mutationRecord{}, false, nil
	}
	if err != nil {
		return mutationRecord{}, false, err
	}
	m.Receipt.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return m, true, err
}

func mutationMatches(m mutationRecord, b MutationBinding, subject string) bool {
	return m.Receipt.ActorID == b.ActorID && m.Receipt.RequestID == b.RequestID && m.Receipt.NodeID == b.NodeID && m.Receipt.EnrollmentGeneration == b.EnrollmentGeneration && m.Receipt.Subject == subject && m.Receipt.ExpectedRevision == b.ExpectedRevision && m.Receipt.PayloadDigest == b.PayloadDigest
}

func writeMutation(ctx context.Context, tx *sql.Tx, r MutationReceipt, operation string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO credential_grant_mutation_receipts (actor_id,request_id,operation,node_id,enrollment_generation,subject,expected_revision,result_revision,payload_digest,grant_id,outcome,delivery_state,created_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`, r.ActorID, r.RequestID, operation, r.NodeID, r.EnrollmentGeneration, r.Subject, r.ExpectedRevision, r.ResultRevision, r.PayloadDigest, r.GrantID, r.Outcome, r.DeliveryState, r.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func readGrant(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id string) (Grant, error) {
	var g Grant
	var class, retention, granted, revoked, receipt, purgeReceipt string
	err := q.QueryRowContext(ctx, "SELECT "+grantColumns+" FROM credential_grants WHERE id=?", id).Scan(&g.ID, &g.NodeID, &g.LogicalID, &g.Field, &class, &retention, &g.Generation, &g.Revision, &granted, &revoked, &g.AckedGeneration, &receipt, &g.ReceiptAccepted, &g.ReceiptReason, &g.PurgeState, &purgeReceipt, &g.PurgeAccepted, &g.PurgeReason)
	if err != nil {
		return Grant{}, err
	}
	g.Class = Class(class)
	g.Retention = Retention(retention)
	if g.GrantedAt, err = time.Parse(time.RFC3339Nano, granted); err != nil {
		return Grant{}, err
	}
	if revoked != "" {
		if g.RevokedAt, err = time.Parse(time.RFC3339Nano, revoked); err != nil {
			return Grant{}, err
		}
	}
	if receipt != "" {
		if g.ReceiptAt, err = time.Parse(time.RFC3339Nano, receipt); err != nil {
			return Grant{}, err
		}
	}
	if purgeReceipt != "" {
		if g.PurgeReceiptAt, err = time.Parse(time.RFC3339Nano, purgeReceipt); err != nil {
			return Grant{}, err
		}
	}
	return g, nil
}
