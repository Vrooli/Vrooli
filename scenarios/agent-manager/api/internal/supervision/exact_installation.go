package supervision

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/vrooli/api-core/authn"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

// Normal wiring exposes protected setup, never an installed key or grant. All
// current instance/owner/activation decisions remain in the owning SQL store.
func NewInstalledExactVerifier(r *Repository) *ExactVerifier {
	return &ExactVerifier{repo: r, now: time.Now, installed: true}
}
func (v *ExactVerifier) nativeInstallation(ctx context.Context, q exactQuery) (string, string, error) {
	if !v.installed {
		if !boundedExactText(v.instance, 160) {
			return "", "", ErrExactAuthority
		}
		return v.instance, "", nil
	}
	var instance, owner string
	if q.QueryRowContext(ctx, `SELECT instance,owner FROM keyring_exact_installation WHERE slot=1`).Scan(&instance, &owner) != nil || !boundedExactText(instance, 160) || !boundedExactText(owner, 160) {
		return "", "", ErrExactAuthority
	}
	return instance, owner, nil
}

type ExactSetupRequest struct {
	Action        string       `json:"action"`
	Policy        *ExactPolicy `json:"policy,omitempty"`
	Grant         string       `json:"grant,omitempty"`
	ExpectedEpoch int          `json:"expectedEpoch"`
	PolicyDigest  string       `json:"policyDigest,omitempty"`
	KeyProof      string       `json:"keyProof,omitempty"`
}
type ExactActivationChallenge struct {
	Version      int    `json:"v"`
	Purpose      string `json:"purpose"`
	Instance     string `json:"instance"`
	Owner        string `json:"owner"`
	Grant        string `json:"grant"`
	Epoch        int    `json:"epoch"`
	PolicyDigest string `json:"policyDigest"`
}
type ExactSetupReceipt struct {
	Grant               string                    `json:"grant"`
	Epoch               int                       `json:"epoch"`
	PolicyDigest        string                    `json:"policyDigest"`
	State               string                    `json:"state"`
	Instance            string                    `json:"instance"`
	ApprovedBy          string                    `json:"approvedBy"`
	ApprovedAt          int64                     `json:"approvedAt"`
	ActivationChallenge *ExactActivationChallenge `json:"activationChallenge,omitempty"`
}

func exactPolicyDigest(p ExactPolicy) string { b, _ := json.Marshal(p); return bytesDigest(b) }
func (v *ExactVerifier) validateSetupPolicy(p ExactPolicy, owner string) error {
	now := v.now().Unix()
	if p.Owner != owner || !boundedExactText(p.ID, 160) || !boundedExactText(p.Owner, 160) || !boundedExactText(p.Instance, 160) || !boundedExactText(p.Issuer, 160) || !boundedExactText(p.AuthorizedBy, 160) || !boundedExactText(p.AuthorityRef, 160) || len(p.EffortRef) > 160 || !exactRef.MatchString(p.EffortRef) || p.Profile != ExactProfile || len(p.PublicKey) != ed25519.PublicKeySize || p.Epoch < 1 || p.Epoch > 9007199254740991 || p.KeyVersion < 1 || p.KeyVersion > 9007199254740991 || p.Deadline <= now || p.Deadline-now > 30*24*60*60 || len(p.Operations) == 0 || len(p.Operations) > 2 || p.Reader != "" || p.ReceiptSource != nil {
		return ErrExactAuthority
	}
	seen := map[string]bool{}
	for _, op := range p.Operations {
		if seen[op] || (op != "effort.reconcile" && op != "receipt.read") {
			return ErrExactAuthority
		}
		seen[op] = true
	}
	if seen["receipt.read"] {
		if p.ReadEligibility != ExactStandingReadEligibility {
			return ErrExactAuthority
		}
	} else if p.ReadEligibility != "" {
		return ErrExactAuthority
	}
	if !v.installed && p.Instance != v.instance {
		return ErrExactAuthority
	}
	return nil
}

// Setup requires independently verified current human authority. The client
// supplies public bindings only; setup never imports/generates a private key,
// creates native identities, exchanges credentials or grants broader run execution authority.
func (v *ExactVerifier) Setup(ctx context.Context, req ExactSetupRequest) (out *ExactSetupReceipt, err error) {
	defer func() { err = exactTransactionError(err) }()
	if v == nil || v.repo == nil {
		return nil, ErrExactAuthority
	}
	owner, e := authn.RequireIndependentHuman(ctx)
	if e != nil || owner.ExpiresAt.IsZero() || !v.now().Before(owner.ExpiresAt) {
		return nil, ErrExactAuthority
	}
	capability := "agent-manager:write"
	if req.Action == "status" {
		capability = "agent-manager:read"
	}
	if _, e = authn.RequireCapability(ctx, capability); e != nil {
		return nil, ErrExactAuthority
	}
	if req.Action != "stage" && req.Action != "activate" && req.Action != "revoke" && req.Action != "status" {
		return nil, ErrExactAuthority
	}
	if req.Action == "stage" {
		if req.Policy == nil || req.Grant != "" || req.PolicyDigest != "" || req.KeyProof != "" || req.ExpectedEpoch < 0 || v.validateSetupPolicy(*req.Policy, owner.Subject) != nil {
			return nil, ErrExactAuthority
		}
	} else if req.Action == "status" {
		if req.Policy != nil || !boundedExactText(req.Grant, 160) || req.KeyProof != "" || req.ExpectedEpoch < 0 || (req.PolicyDigest != "" && !exactHash.MatchString(req.PolicyDigest)) {
			return nil, ErrExactAuthority
		}
	} else if req.Policy != nil || !boundedExactText(req.Grant, 160) || req.ExpectedEpoch < 1 || !exactHash.MatchString(req.PolicyDigest) || (req.Action != "activate" && req.KeyProof != "") {
		return nil, ErrExactAuthority
	}
	conn, err := v.repo.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// One store serializes setup, current authority, proof admission and revoke.
	if _, err = tx.ExecContext(ctx, `UPDATE keyring_exact_installation SET instance=instance WHERE slot=1`); err != nil {
		return nil, err
	}
	if !v.now().Before(owner.ExpiresAt) {
		return nil, ErrExactAuthority
	}
	var instance, installedOwner string
	err = tx.QueryRowContext(ctx, `SELECT instance,owner FROM keyring_exact_installation WHERE slot=1`).Scan(&instance, &installedOwner)
	if errors.Is(err, sql.ErrNoRows) {
		if req.Action != "stage" {
			return nil, ErrExactAuthority
		}
		instance, installedOwner = req.Policy.Instance, owner.Subject
		if _, err = tx.ExecContext(ctx, `INSERT INTO keyring_exact_installation(slot,instance,owner) VALUES(1,?,?)`, instance, installedOwner); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if installedOwner != owner.Subject {
		return nil, ErrExactAuthority
	}
	grant := req.Grant
	if req.Action == "stage" {
		grant = req.Policy.ID
		if req.Policy.Instance != instance {
			return nil, ErrExactAuthority
		}
	}
	var raw, state, digest, approvedBy string
	var revoked, epoch int
	var approvedAt int64
	err = tx.QueryRowContext(ctx, `SELECT a.policy_json,a.revoked,s.state,s.policy_digest,s.approved_by,s.approved_at FROM keyring_exact_authority a JOIN keyring_exact_policy_setup s ON s.grant_id=a.id WHERE a.id=?`, grant).Scan(&raw, &revoked, &state, &digest, &approvedBy, &approvedAt)
	var current ExactPolicy
	exists := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if !exists {
		var count int
		if tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM keyring_exact_authority WHERE id=?`, grant).Scan(&count) != nil || count != 0 {
			return nil, ErrExactAuthority
		}
	}
	if exists {
		if json.Unmarshal([]byte(raw), &current) != nil || current.Owner != owner.Subject || current.Instance != instance || exactPolicyDigest(current) != digest {
			return nil, ErrExactAuthority
		}
		epoch = current.Epoch
	}
	if req.Action == "stage" {
		next := *req.Policy
		nextDigest := exactPolicyDigest(next)
		// Unknown-outcome same-policy replay reads the original approval/state. It
		// never restages/re-enables a revoked grant or creates another approval.
		if exists && digest == nextDigest && req.ExpectedEpoch == epoch-1 {
			if approvedBy != owner.Subject {
				return nil, ErrExactAuthority
			}
		} else {
			if req.ExpectedEpoch != epoch || next.Epoch != epoch+1 {
				return nil, ErrExactAuthority
			}
			if exists && (next.Owner != current.Owner || next.Instance != current.Instance || next.EffortRef != current.EffortRef || next.AuthorizedBy != current.AuthorizedBy || next.AuthorityRef != current.AuthorityRef || next.Profile != current.Profile || next.KeyVersion < current.KeyVersion || (!reflect.DeepEqual(next.PublicKey, current.PublicKey) || next.Issuer != current.Issuer) && next.KeyVersion <= current.KeyVersion) {
				return nil, ErrExactAuthority
			}
			// Stage only against the existing exact thin owner-reviewed effort. Never
			// seed/edit it or discard metadata to make installation eligible.
			var enrollment string
			if tx.QueryRowContext(ctx, `SELECT enrollment_json FROM supervision_efforts WHERE effort_ref=?`, next.EffortRef).Scan(&enrollment) != nil {
				return nil, ErrExactAuthority
			}
			if !exactStoredThin(enrollment, next) {
				return nil, ErrExactAuthority
			}
			encoded, _ := json.Marshal(next)
			if _, err = tx.ExecContext(ctx, `INSERT INTO keyring_exact_authority(id,policy_json,revoked) VALUES(?,?,1) ON CONFLICT(id) DO UPDATE SET policy_json=excluded.policy_json,revoked=1`, grant, string(encoded)); err != nil {
				return nil, err
			}
			state, digest, approvedBy, approvedAt = "staged", nextDigest, owner.Subject, v.now().Unix()
			current = next
			epoch = next.Epoch
			revoked = 1
			if _, err = tx.ExecContext(ctx, `INSERT INTO keyring_exact_policy_setup(grant_id,policy_digest,state,approved_by,approved_at) VALUES(?,?,?,?,?) ON CONFLICT(grant_id) DO UPDATE SET policy_digest=excluded.policy_digest,state=excluded.state,approved_by=excluded.approved_by,approved_at=excluded.approved_at`, grant, digest, state, approvedBy, approvedAt); err != nil {
				return nil, err
			}
		}
	} else {
		if !exists || (req.Action != "status" && (req.ExpectedEpoch != epoch || req.PolicyDigest != digest)) || (req.Action == "status" && ((req.ExpectedEpoch != 0 && req.ExpectedEpoch != epoch) || (req.PolicyDigest != "" && req.PolicyDigest != digest))) {
			return nil, ErrExactAuthority
		}
		switch req.Action {
		case "activate":
			if state != "staged" && state != "active" || state == "active" && revoked != 0 || v.validateSetupPolicy(current, owner.Subject) != nil {
				return nil, ErrExactAuthority
			}
			challenge := ExactActivationChallenge{1, "keyring.native-activation.v1", instance, owner.Subject, grant, epoch, digest}
			canonical, _ := json.Marshal(challenge)
			proof, e := base64.RawURLEncoding.DecodeString(req.KeyProof)
			if e != nil || len(proof) != ed25519.SignatureSize || !ed25519.Verify(current.PublicKey, canonical, proof) {
				return nil, ErrExactAuthority
			}
			var enrollment string
			if tx.QueryRowContext(ctx, `SELECT enrollment_json FROM supervision_efforts WHERE effort_ref=?`, current.EffortRef).Scan(&enrollment) != nil || !exactStoredThin(enrollment, current) {
				return nil, ErrExactAuthority
			}
			if _, err = tx.ExecContext(ctx, `UPDATE keyring_exact_authority SET revoked=0 WHERE id=?`, grant); err != nil {
				return nil, err
			}
			state = "active"
			revoked = 0
			if _, err = tx.ExecContext(ctx, `UPDATE keyring_exact_policy_setup SET state='active' WHERE grant_id=?`, grant); err != nil {
				return nil, err
			}
		case "revoke":
			if _, err = tx.ExecContext(ctx, `UPDATE keyring_exact_authority SET revoked=1 WHERE id=?`, grant); err != nil {
				return nil, err
			}
			state = "revoked"
			revoked = 1
			if _, err = tx.ExecContext(ctx, `UPDATE keyring_exact_policy_setup SET state='revoked' WHERE grant_id=?`, grant); err != nil {
				return nil, err
			}
		}
	}
	if state == "active" && revoked != 0 {
		state = "revoked"
	}
	out = &ExactSetupReceipt{Grant: grant, Epoch: epoch, PolicyDigest: digest, State: state, Instance: instance, ApprovedBy: approvedBy, ApprovedAt: approvedAt}
	if state == "staged" {
		out.ActivationChallenge = &ExactActivationChallenge{1, "keyring.native-activation.v1", instance, owner.Subject, grant, epoch, digest}
	}
	if !v.now().Before(owner.ExpiresAt) {
		return nil, ErrExactAuthority
	}
	if (req.Action == "stage" || req.Action == "activate") && v.validateSetupPolicy(current, owner.Subject) != nil {
		return nil, ErrExactAuthority
	}
	if req.Action == "status" {
		return out, nil
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func exactStoredThin(raw string, p ExactPolicy) bool {
	e := &pb.EffortEnrollment{}
	return effortRead.Unmarshal([]byte(raw), e) == nil && exactThin(e, p, false)
}

// InstalledExactVerifier shares the exact service-owned repository. Private
// seals retain pointer identity so proofs cannot cross an unrelated store.
func (s *EffortService) InstalledExactVerifier() *ExactVerifier {
	if s == nil {
		return nil
	}
	r, ok := s.repo.(*Repository)
	if !ok {
		return nil
	}
	return NewInstalledExactVerifier(r)
}
