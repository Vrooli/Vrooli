package supervision

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func installedExactFixture(t *testing.T) (*EffortService, *Repository, *ExactVerifier, ExactPolicy, ed25519.PrivateKey, context.Context) {
	t.Helper()
	s, r, _, p, key, _, _ := exactFixture(t)
	if _, err := r.db.Exec(`DELETE FROM keyring_exact_authority`); err != nil {
		t.Fatal(err)
	}
	p.Operations = []string{"effort.reconcile", "receipt.read"}
	p.ReadEligibility = ExactStandingReadEligibility
	v := s.InstalledExactVerifier()
	v.now = r.now
	owner := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Verified: true, Subject: p.Owner, Scopes: []string{"agent-manager:write", "agent-manager:read"}, ExpiresAt: v.now().Add(time.Hour)})
	return s, r, v, p, key, owner
}
func exactSetupState(t *testing.T, r *Repository) string {
	t.Helper()
	state := exactReadState(t, r)
	for _, q := range []string{`SELECT COALESCE(group_concat(slot||instance||owner),'') FROM keyring_exact_installation`, `SELECT COALESCE(group_concat(grant_id||policy_digest||state||approved_by||approved_at),'') FROM (SELECT * FROM keyring_exact_policy_setup ORDER BY grant_id)`} {
		var raw string
		if err := r.db.QueryRowContext(context.Background(), q).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		state += raw
	}
	return state
}
func exactActivateRequest(t *testing.T, receipt *ExactSetupReceipt, key ed25519.PrivateKey) ExactSetupRequest {
	t.Helper()
	raw, _ := json.Marshal(receipt.ActivationChallenge)
	return ExactSetupRequest{Action: "activate", Grant: receipt.Grant, ExpectedEpoch: receipt.Epoch, PolicyDigest: receipt.PolicyDigest, KeyProof: base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, raw))}
}
func TestExactInstallationStagesActivatesRecoversAndRevokes(t *testing.T) {
	s, r, v, p, key, owner := installedExactFixture(t)
	if v.repo != s.repo {
		t.Fatal("ordinary service factory crossed store")
	}
	if _, err := v.policy(owner, r.db, p.ID); err == nil {
		t.Fatal("uninstalled authority")
	}
	staged, err := v.Setup(owner, ExactSetupRequest{Action: "stage", Policy: &p})
	if err != nil || staged.State != "staged" || staged.ApprovedBy != p.Owner || staged.ActivationChallenge == nil {
		t.Fatal("stage", err)
	}
	if _, err = v.policy(owner, r.db, p.ID); err == nil {
		t.Fatal("stage enabled operation")
	}
	before := exactSetupState(t, r)
	recovered, err := v.Setup(owner, ExactSetupRequest{Action: "status", Grant: p.ID})
	if err != nil || !reflect.DeepEqual(staged, recovered) || before != exactSetupState(t, r) {
		t.Fatal("unknown stage acknowledgment recovery", err)
	}
	replay, err := v.Setup(owner, ExactSetupRequest{Action: "stage", Policy: &p})
	if err != nil || !reflect.DeepEqual(staged, replay) || before != exactSetupState(t, r) {
		t.Fatal("duplicate stage approval", err)
	}
	activate := exactActivateRequest(t, staged, key)
	bad := activate
	bad.KeyProof = base64.RawURLEncoding.EncodeToString(make([]byte, 64))
	if _, err = v.Setup(owner, bad); err == nil || before != exactSetupState(t, r) {
		t.Fatal("key possession refusal has effects")
	}
	active, err := v.Setup(owner, activate)
	if err != nil || active.State != "active" {
		t.Fatal("activate", err)
	}
	if _, err = v.policy(owner, r.db, p.ID); err != nil {
		t.Fatal("installed verifier unavailable", err)
	}
	before = exactSetupState(t, r)
	if _, err = v.Setup(owner, activate); err != nil || before != exactSetupState(t, r) {
		t.Fatal("activation replay", err)
	}
	revoke := ExactSetupRequest{Action: "revoke", Grant: p.ID, ExpectedEpoch: p.Epoch, PolicyDigest: staged.PolicyDigest}
	if out, err := v.Setup(owner, revoke); err != nil || out.State != "revoked" {
		t.Fatal("revoke", err)
	}
	before = exactSetupState(t, r)
	if _, err = v.Setup(owner, activate); err == nil || before != exactSetupState(t, r) {
		t.Fatal("old activation revived revoked authority")
	}
	replay, err = v.Setup(owner, ExactSetupRequest{Action: "stage", Policy: &p})
	if err != nil || replay.State != "revoked" || before != exactSetupState(t, r) {
		t.Fatal("stage replay revived authority", err)
	}
}
func TestExactInstallationRejectsInvalidAuthorityAndPoliciesWithoutEffects(t *testing.T) {
	for _, name := range []string{"agent", "unverified", "foreign owner", "zero expiry", "expired", "missing capability", "bad operation", "legacy reader", "legacy source", "bad eligibility", "wrong epoch", "bad key", "long deadline", "missing effort", "foreign thin authority"} {
		t.Run(name, func(t *testing.T) {
			_, r, v, p, _, ctx := installedExactFixture(t)
			pr, _ := identity.PrincipalFromContext(ctx)
			switch name {
			case "agent":
				pr.Kind = identity.ActorAgent
			case "unverified":
				pr.Verified = false
			case "foreign owner":
				pr.Subject = "foreign-owner"
			case "zero expiry":
				pr.ExpiresAt = time.Time{}
			case "expired":
				pr.ExpiresAt = v.now()
			case "missing capability":
				pr.Scopes = nil
			case "bad operation":
				p.Operations = []string{"run.create"}
			case "legacy reader":
				p.Reader = "legacy"
			case "legacy source":
				p.ReceiptSource = &ExactReceiptSource{Grant: p.ID}
			case "bad eligibility":
				p.ReadEligibility = ""
			case "wrong epoch":
				p.Epoch = 2
			case "bad key":
				p.PublicKey = nil
			case "long deadline":
				p.Deadline = v.now().Add(31 * 24 * time.Hour).Unix()
			case "missing effort":
				p.EffortRef = "effort:missing"
			case "foreign thin authority":
				p.AuthorityRef = "other"
			}
			ctx = identity.WithPrincipal(ctx, pr)
			before := exactSetupState(t, r)
			if out, err := v.Setup(ctx, ExactSetupRequest{Action: "stage", Policy: &p}); err == nil || out != nil || before != exactSetupState(t, r) {
				t.Fatal("invalid setup had effects", err)
			}
		})
	}
}
func TestExactInstallationExpiryAfterFenceAndBeforeCommitRollsBack(t *testing.T) {
	for _, boundary := range []int{3, 5} {
		t.Run(string(rune('0'+boundary)), func(t *testing.T) {
			_, r, v, p, _, ctx := installedExactFixture(t)
			pr, _ := identity.PrincipalFromContext(ctx)
			calls := 0
			start := r.now()
			v.now = func() time.Time {
				calls++
				if calls >= boundary {
					return pr.ExpiresAt
				}
				return start
			}
			before := exactSetupState(t, r)
			if out, err := v.Setup(ctx, ExactSetupRequest{Action: "stage", Policy: &p}); err == nil || out != nil || before != exactSetupState(t, r) {
				t.Fatal("expired setup committed", err, calls)
			}
		})
	}
}
func TestExactInstallationRotationCannotAdoptForeignOrOrphanAuthority(t *testing.T) {
	_, r, v, p, key, ctx := installedExactFixture(t)
	staged, err := v.Setup(ctx, ExactSetupRequest{Action: "stage", Policy: &p})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = v.Setup(ctx, exactActivateRequest(t, staged, key)); err != nil {
		t.Fatal(err)
	}
	before := exactSetupState(t, r)
	next := p
	next.Epoch++
	next.PublicKey, _, _ = ed25519.GenerateKey(rand.Reader)
	if _, err = v.Setup(ctx, ExactSetupRequest{Action: "stage", Policy: &next, ExpectedEpoch: 1}); err == nil || before != exactSetupState(t, r) {
		t.Fatal("key changed without version increment")
	}
	next.KeyVersion++
	staged, err = v.Setup(ctx, ExactSetupRequest{Action: "stage", Policy: &next, ExpectedEpoch: 1})
	if err != nil || staged.Epoch != 2 {
		t.Fatal("qualified rotation", err)
	}
	if _, err = v.policy(ctx, r.db, p.ID); err == nil {
		t.Fatal("rotation stage active")
	}
	foreign := next
	foreign.ID = "orphan"
	raw, _ := json.Marshal(foreign)
	if _, err = r.db.Exec(`INSERT INTO keyring_exact_authority(id,policy_json) VALUES(?,?)`, foreign.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	foreign.Epoch = 1
	before = exactSetupState(t, r)
	if _, err = v.Setup(ctx, ExactSetupRequest{Action: "stage", Policy: &foreign}); err == nil || before != exactSetupState(t, r) {
		t.Fatal("orphan authority overwritten")
	}
}
