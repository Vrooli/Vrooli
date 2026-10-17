package supervision

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const ExactProofHeader = "X-Keyring-Exact-Proof"
const ExactSignatureHeader = "X-Keyring-Exact-Signature"
const ExactMaxBytes = 16000
const ExactPurpose = "keyring.exact-action.v1"
const ExactStandingReadEligibility = "standing-account-v1"
const ExactProfile = "disposable-thin-fixture-v1"

var ErrExactAuthority = errors.New("exact-action authority refused")
var ErrExactRetryable = errors.New("exact-action transaction uncommitted; retry with fresh proof and original request identity")

// ExactPolicy is an owner-installed current-state record, not a run identity
// or execution authority. Normal startup creates empty schema, never grants.
type ExactPolicy struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	Instance     string `json:"instance"`
	Issuer       string `json:"issuer"`
	KeyVersion   int    `json:"keyVersion"`
	Epoch        int    `json:"epoch"`
	Deadline     int64  `json:"deadline"`
	EffortRef    string `json:"effortRef"`
	AuthorizedBy string `json:"authorizedBy"`
	AuthorityRef string `json:"authorityRef"`
	Profile      string `json:"profile"`
	PublicKey    []byte `json:"publicKey"`
	// Absent operations preserves mutation-only v1 policy compatibility.
	Operations      []string            `json:"operations"`
	ReadEligibility string              `json:"readEligibility"`
	Reader          string              `json:"reader"`
	ReceiptSource   *ExactReceiptSource `json:"receiptSource"`
}

// ExactEnvelope matches the accepted Site v1 canonical field order. Action
// remains signed JSON; only its decoded protobuf enters semantic effect identity.
type ExactEnvelope struct {
	Version    int             `json:"v"`
	Purpose    string          `json:"purpose"`
	Instance   string          `json:"instance"`
	Audience   string          `json:"audience"`
	Issuer     string          `json:"issuer"`
	KeyVersion int             `json:"keyVersion"`
	Action     json.RawMessage `json:"action"`
	Grant      string          `json:"grant"`
	Epoch      int             `json:"epoch"`
	Initiator  string          `json:"initiator"`
	Executor   string          `json:"executor"`
	Operation  string          `json:"operation"`
	Target     string          `json:"target"`
	Payload    string          `json:"payload"`
	Request    string          `json:"request"`
	Expires    int64           `json:"expires"`
}
type ExactChallenge struct {
	Version   int             `json:"v"`
	Purpose   string          `json:"purpose"`
	Instance  string          `json:"instance"`
	Audience  string          `json:"audience"`
	Grant     string          `json:"grant"`
	Epoch     int             `json:"epoch"`
	Executor  string          `json:"executor"`
	Operation string          `json:"operation"`
	Target    string          `json:"target"`
	Payload   string          `json:"payload"`
	Request   string          `json:"request"`
	IssuedAt  int64           `json:"issuedAt"`
	Action    json.RawMessage `json:"action"`
}
type ExactVerifier struct {
	repo      *Repository
	instance  string
	now       func() time.Time
	installed bool
}

func NewExactVerifier(r *Repository, instance string) *ExactVerifier {
	return &ExactVerifier{repo: r, instance: instance, now: time.Now}
}

// EnsureExactAuthoritySchema is separate protected installation; it is not
// called from production startup. No grant, key or identity is created here.
func (r *Repository) EnsureExactAuthoritySchema(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
-- Protected exact-action setup is inert until independent owner approval and
-- pinned-key activation. Schema installation creates no keys/grants/identities.
CREATE TABLE IF NOT EXISTS keyring_exact_authority(id TEXT PRIMARY KEY, policy_json TEXT NOT NULL, revoked INTEGER NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS keyring_exact_installation(slot INTEGER PRIMARY KEY CHECK(slot=1), instance TEXT NOT NULL, owner TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS keyring_exact_policy_setup(grant_id TEXT PRIMARY KEY, policy_digest TEXT NOT NULL, state TEXT NOT NULL CHECK(state IN ('staged','active','revoked')), approved_by TEXT NOT NULL, approved_at INTEGER NOT NULL);
`)
	return err
}
func (r *Repository) RevokeExactAuthority(ctx context.Context, id string) (err error) {
	defer func() { err = exactTransactionError(err) }()
	result, err := r.db.ExecContext(ctx, `UPDATE keyring_exact_authority SET revoked=1 WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrExactAuthority
	}
	return nil
}

type exactQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (v *ExactVerifier) policy(ctx context.Context, q exactQuery, id string) (ExactPolicy, error) {
	var raw string
	var revoked int
	var p ExactPolicy
	if v == nil || v.repo == nil {
		return p, ErrExactAuthority
	}
	if err := q.QueryRowContext(ctx, `SELECT policy_json,revoked FROM keyring_exact_authority WHERE id=?`, id).Scan(&raw, &revoked); err != nil {
		return p, ErrExactAuthority
	}
	instance, owner, err := v.nativeInstallation(ctx, q)
	if err != nil {
		return p, ErrExactAuthority
	}
	if json.Unmarshal([]byte(raw), &p) != nil || (owner != "" && p.Owner != owner) || p.ID != id || revoked != 0 || p.Instance != instance || p.Profile != ExactProfile || len(p.PublicKey) != ed25519.PublicKeySize || p.Epoch < 1 || p.KeyVersion < 1 || !boundedExactText(p.Owner, 160) || !boundedExactText(p.Issuer, 160) || !boundedExactText(p.ID, 160) || !boundedExactText(p.Instance, 160) || !boundedExactText(p.AuthorizedBy, 160) || !boundedExactText(p.AuthorityRef, 160) || !exactRef.MatchString(p.EffortRef) || !v.now().Before(time.Unix(p.Deadline, 0)) {
		return p, ErrExactAuthority
	}
	if v.installed {
		var digest, state string
		if q.QueryRowContext(ctx, `SELECT policy_digest,state FROM keyring_exact_policy_setup WHERE grant_id=?`, id).Scan(&digest, &state) != nil || state != "active" || digest != exactPolicyDigest(p) {
			return p, ErrExactAuthority
		}
	}
	return p, nil
}

var exactRef = regexp.MustCompile(`^effort:[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func boundedExactText(s string, max int) bool {
	if len(s) == 0 || !utf8.ValidString(s) || len(utf16.Encode([]rune(s))) > max {
		return false
	}
	for _, r := range s {
		if r <= 31 || r == 127 {
			return false
		}
	}
	return true
}
func exactThin(e *pb.EffortEnrollment, p ExactPolicy, request bool) bool {
	if e == nil || len(e.ProtoReflect().GetUnknown()) != 0 || e.EffortRef != p.EffortRef || len(e.EffortRef) > 160 || e.AuthorizedBy != p.AuthorizedBy || e.AuthorityRef != p.AuthorityRef || e.SupervisorRunId != "" || len(e.Subjects) != 0 || e.Withdrawn || e.WithdrawalReason != "" || e.DestinationRef != "" || e.TargetRevision != "" || e.SourceRevision != "" || e.Workspace != "" || e.WorkShape != "" || e.Revision < 1 || e.Revision > 9007199254740991 || !boundedExactText(e.DisplayName, 120) {
		return false
	}
	if request && e.UpdatedAt != nil {
		return false
	}
	return true
}
func exactRequest(req *pb.ReconcileEffortMetadataRequest, p ExactPolicy) bool {
	return req != nil && len(req.ProtoReflect().GetUnknown()) == 0 && req.Authority == pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR && exactThin(req.Enrollment, p, true) && req.ExpectedRevision > 0 && req.ExpectedRevision <= 9007199254740991 && boundedExactText(req.IdempotencyKey, 160)
}
func exactTarget(ref string) string {
	return "agent-manager:effort/" + strings.TrimPrefix(ref, "effort:")
}
func (v *ExactVerifier) Challenge(ctx context.Context, grant string, req *pb.ReconcileEffortMetadataRequest) (*ExactChallenge, error) {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || p.Kind != identity.ActorAgent || !p.ExpiresAt.IsZero() && !v.now().Before(p.ExpiresAt) {
		return nil, ErrExactAuthority
	}
	if _, err := uuid.Parse(p.Subject); err != nil {
		return nil, ErrExactAuthority
	}
	policy, err := v.policy(ctx, v.repo.db, grant)
	if err != nil || !policy.allows("effort.reconcile") || !exactRequest(req, policy) {
		return nil, ErrExactAuthority
	}
	if p.Owner != "" && p.Owner != policy.Owner {
		return nil, ErrExactAuthority
	}
	action, err := protojson.Marshal(req)
	if err != nil {
		return nil, ErrExactAuthority
	}
	return &ExactChallenge{1, ExactPurpose, policy.Instance, "agent-manager", grant, policy.Epoch, p.Subject, "effort.reconcile", exactTarget(policy.EffortRef), effortDigest(req), req.IdempotencyKey, v.now().Unix(), action}, nil
}

// Verify never changes the durable owner, Principal scopes or interactive grant.
// Invalid offered proofs are terminal; callers must not select legacy fallback.
func (v *ExactVerifier) Verify(ctx context.Context, req *pb.ReconcileEffortMetadataRequest, signed, signature string) (context.Context, error) {
	if v == nil || len(signed) > ((ExactMaxBytes+2)/3)*4 || len(signature) > 128 {
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
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(&e) != nil {
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
	if _, err = uuid.Parse(p.Subject); err != nil {
		return ctx, ErrExactAuthority
	}
	policy, err := v.policy(ctx, v.repo.db, e.Grant)
	if err != nil {
		return ctx, ErrExactAuthority
	}
	if !policy.allows("effort.reconcile") || e.Version != 1 || e.Purpose != ExactPurpose || e.Instance != policy.Instance || e.Audience != "agent-manager" || e.Issuer != policy.Issuer || e.KeyVersion != policy.KeyVersion || e.Epoch != policy.Epoch || e.Initiator != policy.Owner || e.Executor != p.Subject || p.Owner != "" && p.Owner != e.Initiator || e.Operation != "effort.reconcile" || e.Target != exactTarget(policy.EffortRef) || !exactRequest(req, policy) || e.Payload != effortDigest(req) || e.Request != req.IdempotencyKey || !ed25519.Verify(policy.PublicKey, raw, sig) {
		return ctx, ErrExactAuthority
	}
	action := &pb.ReconcileEffortMetadataRequest{}
	if protojson.Unmarshal(e.Action, action) != nil || !proto.Equal(action, req) {
		return ctx, ErrExactAuthority
	}
	a := &exactAuthorization{verifier: v, policy: policy, envelope: e, principal: cloneExactPrincipal(p), request: proto.Clone(req).(*pb.ReconcileEffortMetadataRequest)}
	ctx = authn.WithRequestOperationAuthority(ctx, a)
	if a.validate(ctx, req) != nil {
		return ctx, ErrExactAuthority
	}
	return ctx, nil
}

// The concrete authority and actor marker are private. Only Verify seals one.
type exactAuthorization struct {
	verifier  *ExactVerifier
	policy    ExactPolicy
	envelope  ExactEnvelope
	principal identity.Principal
	request   *pb.ReconcileEffortMetadataRequest
}

func (a *exactAuthorization) validate(ctx context.Context, req *pb.ReconcileEffortMetadataRequest) error {
	current, ok := authn.RequestOperationAuthorityFromContext(ctx)
	p, verified := identity.PrincipalFromContext(ctx)
	now := a.verifier.now()
	expiry := time.Unix(a.envelope.Expires, 0)
	if !ok || current != a || !verified || !reflect.DeepEqual(p, a.principal) || !proto.Equal(a.request, req) || !now.Before(expiry) || expiry.After(now.Add(time.Minute)) || expiry.After(time.Unix(a.policy.Deadline, 0)) || !p.ExpiresAt.IsZero() && !now.Before(p.ExpiresAt) {
		return ErrExactAuthority
	}
	return nil
}
func (a *exactAuthorization) AuthorizeOperation(ctx context.Context, p identity.Principal, capability, operation, target string) error {
	if capability != "agent-manager:write" || operation != a.envelope.Operation || target != a.envelope.Target || !reflect.DeepEqual(p, a.principal) {
		return ErrExactAuthority
	}
	return a.validate(ctx, a.request)
}
func ExactEffortActor(ctx context.Context, ref string) (EffortActor, bool, error) {
	value, present := authn.RequestOperationAuthorityFromContext(ctx)
	if !present {
		return EffortActor{}, false, nil
	}
	a, ok := value.(*exactAuthorization)
	if !ok || a.envelope.Target != exactTarget(ref) {
		return EffortActor{}, true, ErrExactAuthority
	}
	if _, err := authn.RequireOperation(ctx, "agent-manager:write", "effort.reconcile", exactTarget(ref)); err != nil {
		return EffortActor{}, true, err
	}
	return EffortActor{ID: a.envelope.Executor, ApprovedBy: a.envelope.Initiator, exact: a}, true, nil
}

func cloneExactPrincipal(p identity.Principal) identity.Principal {
	p.Scopes = append(p.Scopes[:0:0], p.Scopes...)
	p.Roles = append(p.Roles[:0:0], p.Roles...)
	p.Sources = append(p.Sources[:0:0], p.Sources...)
	if p.Grant != nil {
		g := *p.Grant
		g.Operations = append([]string(nil), g.Operations...)
		g.Targets = append([]string(nil), g.Targets...)
		g.Rules = append([]identity.OperationRule(nil), g.Rules...)
		for i := range g.Rules {
			g.Rules[i].Operations = append([]string(nil), g.Rules[i].Operations...)
			g.Rules[i].Targets = append([]string(nil), g.Rules[i].Targets...)
		}
		p.Grant = &g
	}
	return p
}

// SQLite busy/locked is a failed, uncommitted decision, never acknowledged
// revocation or permission to choose a new request ID. Preserve other errors.
func exactTransactionError(err error) error {
	if err == nil {
		return nil
	}
	var coded interface{ Code() int }
	if errors.As(err, &coded) && (coded.Code()&255 == 5 || coded.Code()&255 == 6) {
		return ErrExactRetryable
	}
	return err
}
