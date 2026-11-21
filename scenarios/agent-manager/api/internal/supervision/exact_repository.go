package supervision

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Semantic identity excludes proof signature, expiry and raw JSON ordering.
// It retains account/executor/grant/epoch and every native request commitment.
type exactMutation struct {
	Purpose, Instance, Audience, Issuer                      string
	KeyVersion                                               int
	Grant                                                    string
	Epoch                                                    int
	Initiator, Executor, Operation, Target, Payload, Request string
}

func (a *exactAuthorization) digest() string {
	e := a.envelope
	b, _ := json.Marshal(exactMutation{e.Purpose, e.Instance, e.Audience, e.Issuer, e.KeyVersion, e.Grant, e.Epoch, e.Initiator, e.Executor, e.Operation, e.Target, e.Payload, e.Request})
	return "keyring-exact-v1:" + string(b)
}
func (r *Repository) reconcileExact(ctx context.Context, req *pb.ReconcileEffortMetadataRequest, a *exactAuthorization) (out *pb.EffortEnrollment, err error) {
	defer func() { err = exactTransactionError(err) }()
	if a == nil || a.verifier.repo != r || a.validate(ctx, req) != nil {
		return nil, ErrExactAuthority
	}
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Acquire the same-store writer fence before reading authority, receipt and
	// domain state. A committed revoke wins before this decision; busy is not
	// acknowledged revocation. The fence changes no policy or authority.
	result, err := tx.ExecContext(ctx, `UPDATE keyring_exact_authority SET revoked=revoked WHERE id=?`, a.policy.ID)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return nil, ErrExactAuthority
	}
	policy, err := a.verifier.policy(ctx, tx, a.policy.ID)
	if err != nil || !reflect.DeepEqual(policy, a.policy) || a.validate(ctx, req) != nil {
		return nil, ErrExactAuthority
	}
	var raw, obraw string
	if err = tx.QueryRowContext(ctx, `SELECT enrollment_json,observation_json FROM supervision_efforts WHERE effort_ref=?`, policy.EffortRef).Scan(&raw, &obraw); err != nil {
		return nil, ErrExactAuthority
	}
	old := &pb.EffortEnrollment{}
	if effortRead.Unmarshal([]byte(raw), old) != nil || !exactThin(old, policy, false) {
		return nil, ErrExactAuthority
	}
	key, digest := "reconcile-metadata:"+req.IdempotencyKey, a.digest()
	var stored, receipt string
	err = tx.QueryRowContext(ctx, `SELECT request_digest,result_json FROM supervision_effort_transitions WHERE operation_key=?`, key).Scan(&stored, &receipt)
	if err == nil {
		if stored != digest {
			return nil, ErrConflict
		}
		out := &pb.EffortEnrollment{}
		if effortRead.Unmarshal([]byte(receipt), out) != nil || !exactThin(out, policy, false) {
			return nil, ErrExactAuthority
		}
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return out, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if old.Revision >= 9007199254740991 {
		return nil, ErrExactAuthority
	}
	if old.Revision != req.ExpectedRevision {
		return nil, ErrConflict
	}
	next := proto.Clone(old).(*pb.EffortEnrollment)
	next.DisplayName = req.Enrollment.DisplayName
	if validateEffortEnrollment(next) != nil {
		return nil, ErrExactAuthority
	}
	next.Revision++
	next.UpdatedAt = timestamppb.New(a.verifier.now().UTC())
	b, err := effortJSON.Marshal(next)
	if err != nil {
		return nil, err
	}
	if a.validate(ctx, req) != nil {
		return nil, ErrExactAuthority
	}
	if err = r.saveEffortTx(ctx, tx, next, old.Revision, key, digest, b, []byte(obraw)); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return next, nil
}

// ExactReceipt is a bounded immutable historical result, never a board read.
type ExactReceipt struct {
	OperationKey     string `json:"operationKey"`
	EffortRef        string `json:"effortRef"`
	Revision         uint64 `json:"revision"`
	DisplayName      string `json:"displayName"`
	OccurredAt       string `json:"occurredAt"`
	ResultHash       string `json:"resultHash"`
	OriginalExecutor string `json:"originalExecutor"`
	OriginalPayload  string `json:"originalPayload"`
}

// ReadExactReceipt is invoked only after an independent human read check.
// Owner comparison uses immutable recorded initiation, not the new executor.
func (r *Repository) readExactReceipt(ctx context.Context, key, ref, owner string) (*ExactReceipt, error) {
	if !boundedExactText(key, 180) || !boundedExactText(owner, 160) || !exactRef.MatchString(ref) {
		return nil, ErrExactAuthority
	}
	var digest, raw, at string
	err := r.db.QueryRowContext(ctx, `SELECT request_digest,result_json,occurred_at FROM supervision_effort_transitions WHERE operation_key=?`, key).Scan(&digest, &raw, &at)
	if err != nil {
		return nil, ErrExactAuthority
	}
	const prefix = "keyring-exact-v1:"
	if len(digest) <= len(prefix) || digest[:len(prefix)] != prefix {
		return nil, ErrExactAuthority
	}
	var m exactMutation
	if json.Unmarshal([]byte(digest[len(prefix):]), &m) != nil || m.Initiator != owner || m.Operation != "effort.reconcile" || m.Target != exactTarget(ref) || key != "reconcile-metadata:"+m.Request {
		return nil, ErrExactAuthority
	}
	return exactReceiptResult(key, ref, raw, at, m)
}

func exactReceiptResult(key, ref, raw, at string, m exactMutation) (*ExactReceipt, error) {
	e := &pb.EffortEnrollment{}
	if effortRead.Unmarshal([]byte(raw), e) != nil || e.EffortRef != ref {
		return nil, ErrExactAuthority
	}
	return &ExactReceipt{key, ref, e.Revision, e.DisplayName, at, bytesDigest([]byte(raw)), m.Executor, m.Payload}, nil
}
