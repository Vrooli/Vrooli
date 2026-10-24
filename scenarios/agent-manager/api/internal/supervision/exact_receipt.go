package supervision

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"reflect"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

const ExactReadPurpose = "keyring.receipt-read.v1"

// Read authority is separately installed and pins the original mutation source.
// Neither a mutation grant nor operator identity implies this authority.
type ExactReceiptSource struct {
	Grant      string `json:"grant"`
	Epoch      int    `json:"epoch"`
	Issuer     string `json:"issuer"`
	KeyVersion int    `json:"keyVersion"`
}

// Fixed field order is the receipt-read v1 action commitment. This is canonical
// JSON of this typed value, not a protobuf commitment or a mutation action.
type ExactReceiptRequest struct {
	OperationKey     string `json:"operationKey"`
	EffortRef        string `json:"effortRef"`
	OriginalExecutor string `json:"originalExecutor"`
	OriginalPayload  string `json:"originalPayload"`
}

func (p ExactPolicy) allows(operation string) bool {
	if p.Operations == nil {
		return operation == "effort.reconcile"
	}
	found := false
	for _, op := range p.Operations {
		if op != "effort.reconcile" && op != "receipt.read" {
			return false
		}
		if op == operation {
			found = true
		}
	}
	return found
}

var exactHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (r ExactReceiptRequest) valid(p ExactPolicy) bool {
	_, err := uuid.Parse(r.OriginalExecutor)
	return p.allows("receipt.read") && p.ReadEligibility == ExactStandingReadEligibility && p.Reader == "" && p.ReceiptSource == nil && r.EffortRef == p.EffortRef && boundedExactText(r.OperationKey, 180) && len(r.OperationKey) > len("reconcile-metadata:") && r.OperationKey[:len("reconcile-metadata:")] == "reconcile-metadata:" && boundedExactText(r.OperationKey[len("reconcile-metadata:"):], 160) && err == nil && exactHash.MatchString(r.OriginalPayload)
}
func (r ExactReceiptRequest) commitment() string { b, _ := json.Marshal(r); return bytesDigest(b) }

// DecodeExactReceiptRequest rejects unknown fields and trailing data. The
// decoded typed value is canonically marshaled for the action commitment.
func DecodeExactReceiptRequest(raw []byte) (ExactReceiptRequest, error) {
	var req ExactReceiptRequest
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if len(raw) > ExactMaxBytes || d.Decode(&req) != nil || d.Decode(&struct{}{}) != io.EOF {
		return req, ErrExactAuthority
	}
	return req, nil
}
func (v *ExactVerifier) ReceiptChallenge(ctx context.Context, grant string, req ExactReceiptRequest) (*ExactChallenge, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || p.Kind != identity.ActorAgent || v == nil || v.repo == nil {
		return nil, ErrExactAuthority
	}
	policy, err := v.policy(ctx, v.repo.db, grant)
	if err != nil || !req.valid(policy) || p.Owner != "" && p.Owner != policy.Owner || !p.ExpiresAt.IsZero() && !v.now().Before(p.ExpiresAt) {
		return nil, ErrExactAuthority
	}
	if _, err := uuid.Parse(p.Subject); err != nil {
		return nil, ErrExactAuthority
	}
	action, _ := json.Marshal(req)
	return &ExactChallenge{1, ExactReadPurpose, policy.Instance, "agent-manager", grant, policy.Epoch, p.Subject, "receipt.read", exactTarget(policy.EffortRef), req.commitment(), req.OperationKey, v.now().Unix(), action}, nil
}
func (v *ExactVerifier) VerifyReceipt(ctx context.Context, req ExactReceiptRequest, signed, signature string) (context.Context, error) {
	if v == nil || v.repo == nil || len(signed) > ((ExactMaxBytes+2)/3)*4 || len(signature) > 128 {
		return ctx, ErrExactAuthority
	}
	raw, err := base64.RawURLEncoding.DecodeString(signed)
	if err != nil || len(raw) > ExactMaxBytes {
		return ctx, ErrExactAuthority
	}
	sig, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return ctx, ErrExactAuthority
	}
	var e ExactEnvelope
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil {
		return ctx, ErrExactAuthority
	}
	canonical, err := json.Marshal(e)
	if err != nil || !bytes.Equal(canonical, raw) {
		return ctx, ErrExactAuthority
	}
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || p.Kind != identity.ActorAgent {
		return ctx, ErrExactAuthority
	}
	if _, err := uuid.Parse(p.Subject); err != nil {
		return ctx, ErrExactAuthority
	}
	policy, err := v.policy(ctx, v.repo.db, e.Grant)
	action, _ := json.Marshal(req)
	if err != nil || !req.valid(policy) || e.Version != 1 || e.Purpose != ExactReadPurpose || e.Instance != policy.Instance || e.Audience != "agent-manager" || e.Issuer != policy.Issuer || e.KeyVersion != policy.KeyVersion || e.Epoch != policy.Epoch || e.Initiator != policy.Owner || p.Owner != "" && p.Owner != policy.Owner || e.Executor != p.Subject || e.Operation != "receipt.read" || e.Target != exactTarget(req.EffortRef) || e.Payload != req.commitment() || e.Request != req.OperationKey || !bytes.Equal(e.Action, action) || !ed25519.Verify(policy.PublicKey, raw, sig) {
		return ctx, ErrExactAuthority
	}
	a := &exactReceiptAuthorization{v, policy, e, cloneExactPrincipal(p), req}
	sealed := authn.WithRequestOperationAuthority(ctx, a)
	if a.validate(sealed, req) != nil {
		return ctx, ErrExactAuthority
	}
	return sealed, nil
}

type exactReceiptAuthorization struct {
	verifier  *ExactVerifier
	policy    ExactPolicy
	envelope  ExactEnvelope
	principal identity.Principal
	request   ExactReceiptRequest
}

func (a *exactReceiptAuthorization) validate(ctx context.Context, req ExactReceiptRequest) error {
	value, ok := authn.RequestOperationAuthorityFromContext(ctx)
	p, verified := identity.PrincipalFromContext(ctx)
	now := a.verifier.now()
	expires := time.Unix(a.envelope.Expires, 0)
	if !ok || value != a || !verified || !reflect.DeepEqual(p, a.principal) || req != a.request || !now.Before(expires) || expires.After(now.Add(time.Minute)) || expires.After(time.Unix(a.policy.Deadline, 0)) || !p.ExpiresAt.IsZero() && !now.Before(p.ExpiresAt) {
		return ErrExactAuthority
	}
	return nil
}
func (a *exactReceiptAuthorization) AuthorizeOperation(ctx context.Context, p identity.Principal, capability, operation, target string) error {
	if capability != "agent-manager:read" || operation != "receipt.read" || target != a.envelope.Target || !reflect.DeepEqual(p, a.principal) {
		return ErrExactAuthority
	}
	return a.validate(ctx, a.request)
}
func (s *EffortService) ReadExactReceiptProof(ctx context.Context, req ExactReceiptRequest) (out *ExactReceipt, err error) {
	defer func() { err = exactTransactionError(err) }()
	value, ok := authn.RequestOperationAuthorityFromContext(ctx)
	a, sealed := value.(*exactReceiptAuthorization)
	repo, native := s.repo.(*Repository)
	if !ok || !sealed || !native || a.verifier.repo != repo || a.validate(ctx, req) != nil {
		return nil, ErrExactAuthority
	}
	if _, err = authn.RequireOperation(ctx, "agent-manager:read", "receipt.read", exactTarget(req.EffortRef)); err != nil {
		return nil, ErrExactAuthority
	}
	conn, err := repo.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// Same-store ordering with current read-grant revoke. Rollback even successful
	// reads: no persisted fence, receipt, revision or authority change.
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
	var digest, raw, at string
	if tx.QueryRowContext(ctx, `SELECT request_digest,result_json,occurred_at FROM supervision_effort_transitions WHERE operation_key=?`, req.OperationKey).Scan(&digest, &raw, &at) != nil {
		return nil, ErrExactAuthority
	}
	const prefix = "keyring-exact-v1:"
	if len(digest) <= len(prefix) || digest[:len(prefix)] != prefix {
		return nil, ErrExactAuthority
	}
	var m exactMutation
	if json.Unmarshal([]byte(digest[len(prefix):]), &m) != nil || m.Purpose != ExactPurpose || m.Instance != policy.Instance || m.Audience != "agent-manager" || m.Initiator != policy.Owner || m.Executor != req.OriginalExecutor || m.Payload != req.OriginalPayload || m.Operation != "effort.reconcile" || m.Target != exactTarget(req.EffortRef) || req.OperationKey != "reconcile-metadata:"+m.Request || m.Grant != policy.ID || m.Epoch < 1 || m.Epoch > policy.Epoch || m.KeyVersion < 1 || m.KeyVersion > policy.KeyVersion || !boundedExactText(m.Issuer, 160) {
		return nil, ErrExactAuthority
	}
	return exactReceiptResult(req.OperationKey, req.EffortRef, raw, at, m)
}

func (s *EffortService) ReadExactReceipt(ctx context.Context, key, ref, owner string) (*ExactReceipt, error) {
	p, err := authn.RequireIndependentHuman(ctx)
	if err != nil || p.Subject != owner {
		return nil, ErrExactAuthority
	}
	if _, err = authn.RequireCapability(ctx, "agent-manager:read"); err != nil {
		return nil, ErrExactAuthority
	}
	repo, ok := s.repo.(*Repository)
	if !ok {
		return nil, ErrExactAuthority
	}
	return repo.readExactReceipt(ctx, key, ref, owner)
}
