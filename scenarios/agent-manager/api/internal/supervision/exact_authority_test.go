package supervision

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func exactFixture(t *testing.T) (*EffortService, *Repository, *ExactVerifier, ExactPolicy, ed25519.PrivateKey, context.Context, *pb.ReconcileEffortMetadataRequest) {
	t.Helper()
	r, db := testRepository(t)
	s := NewEffortService(r, nil, EffortDiscoveryConfig{})
	v := NewExactVerifier(r, "fixture-native-1")
	v.now = r.now
	if err := r.EnsureExactAuthoritySchema(context.Background()); err != nil {
		t.Fatal(err)
	}
	public, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	p := ExactPolicy{ID: "fixture-grant", Owner: "fixture-owner", Instance: "fixture-native-1", Issuer: "fixture-keyring", KeyVersion: 1, Epoch: 1, Deadline: r.now().Add(time.Hour).Unix(), EffortRef: "effort:fixture", AuthorizedBy: "fixture-owner", AuthorityRef: "fixture-approval", Profile: ExactProfile, PublicKey: public}
	raw, _ := json.Marshal(p)
	if _, err = db.Exec(`INSERT INTO keyring_exact_authority(id,policy_json) VALUES(?,?)`, p.ID, string(raw)); err != nil {
		t.Fatal(err)
	}
	e := &pb.EffortEnrollment{EffortRef: p.EffortRef, DisplayName: "before", Revision: 1, AuthorizedBy: p.AuthorizedBy, AuthorityRef: p.AuthorityRef}
	if err = r.SaveEffort(context.Background(), e, &pb.EffortBoardRow{}, 0, "seed", "seed"); err != nil {
		t.Fatal(err)
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Verified: true, Subject: uuid.NewString(), ExpiresAt: r.now().Add(time.Hour)})
	req := &pb.ReconcileEffortMetadataRequest{Enrollment: proto.Clone(e).(*pb.EffortEnrollment), ExpectedRevision: 1, IdempotencyKey: "exact-first", Authority: pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	req.Enrollment.DisplayName = "after"
	return s, r, v, p, key, ctx, req
}
func exactSigned(t *testing.T, v *ExactVerifier, p ExactPolicy, key ed25519.PrivateKey, ctx context.Context, req *pb.ReconcileEffortMetadataRequest, change func(*ExactEnvelope)) (string, string) {
	t.Helper()
	principal, _ := identity.PrincipalFromContext(ctx)
	action, err := protojson.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	e := ExactEnvelope{1, ExactPurpose, p.Instance, "agent-manager", p.Issuer, p.KeyVersion, action, p.ID, p.Epoch, p.Owner, principal.Subject, "effort.reconcile", exactTarget(p.EffortRef), effortDigest(req), req.IdempotencyKey, v.now().Add(30 * time.Second).Unix()}
	if change != nil {
		change(&e)
	}
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, raw))
}
func exactAuthorize(t *testing.T, v *ExactVerifier, p ExactPolicy, key ed25519.PrivateKey, ctx context.Context, req *pb.ReconcileEffortMetadataRequest) (context.Context, EffortActor) {
	t.Helper()
	raw, sig := exactSigned(t, v, p, key, ctx, req, nil)
	sealed, err := v.Verify(ctx, req, raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	actor, ok, err := ExactEffortActor(sealed, p.EffortRef)
	if !ok || err != nil {
		t.Fatal(err)
	}
	return sealed, actor
}
func TestExactAuthorityCompatibilityProvenanceAndReplay(t *testing.T) {
	s, r, v, p, key, ctx, req := exactFixture(t)
	sealed, actor := exactAuthorize(t, v, p, key, ctx, req)
	if principal, _ := identity.PrincipalFromContext(sealed); len(principal.Scopes) != 0 || principal.Grant != nil || principal.Owner != "" {
		t.Fatal("proof widened native principal")
	}
	for name, call := range map[string]func() error{
		"lost context": func() error { _, err := s.ReconcileMetadata(ctx, req, actor); return err },
		"lost actor": func() error {
			_, err := s.ReconcileMetadata(sealed, req, EffortActor{ID: actor.ID, Operator: true})
			return err
		},
		"downgraded actor": func() error {
			bad := actor
			bad.Delegated = true
			_, err := s.ReconcileMetadata(sealed, req, bad)
			return err
		},
		"changed principal": func() error {
			pr, _ := identity.PrincipalFromContext(sealed)
			pr.Owner = "other"
			_, err := s.ReconcileMetadata(identity.WithPrincipal(sealed, pr), req, actor)
			return err
		},
		"different operation": func() error {
			_, err := authn.RequireOperation(sealed, "agent-manager:write", "agent.manage", exactTarget(p.EffortRef))
			return err
		},
	} {
		if err := call(); err == nil {
			t.Fatal(name, "accepted")
		}
	}
	old, _, _ := r.GetEffort(ctx, p.EffortRef)
	if old.Revision != 1 {
		t.Fatal("rejection changed revision")
	}
	first, err := s.ReconcileMetadata(sealed, req, actor)
	if err != nil || first.Revision != 2 {
		t.Fatal(first, err)
	}
	// Lost acknowledgement, new proof and reordered action use one receipt.
	raw, sig := exactSigned(t, v, p, key, ctx, req, func(e *ExactEnvelope) {
		e.Expires++
		e.Action = json.RawMessage(`{"authority":"WATCH_AUTHORITY_OPERATOR","idempotencyKey":"exact-first","expectedRevision":"1","enrollment":{"authorityRef":"fixture-approval","authorizedBy":"fixture-owner","revision":"1","displayName":"after","effortRef":"effort:fixture"}}`)
	})
	renewed, err := v.Verify(ctx, req, raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	renewedActor, _, _ := ExactEffortActor(renewed, p.EffortRef)
	replay, err := s.ReconcileMetadata(renewed, req, renewedActor)
	if err != nil || !proto.Equal(first, replay) {
		t.Fatal("semantic retry", err)
	}
	// Actor B cannot adopt A's mutation identity, even under the same grant.
	pr, _ := identity.PrincipalFromContext(ctx)
	pr.Subject = uuid.NewString()
	otherCtx := identity.WithPrincipal(context.Background(), pr)
	other, otherActor := exactAuthorize(t, v, p, key, otherCtx, req)
	if _, err = s.ReconcileMetadata(other, req, otherActor); !errors.Is(err, ErrConflict) {
		t.Fatal("actor replay", err)
	}
	next := proto.Clone(req).(*pb.ReconcileEffortMetadataRequest)
	next.ExpectedRevision = 2
	next.Enrollment.Revision = 2
	next.IdempotencyKey = "exact-second"
	next.Enrollment.DisplayName = "second"
	other, otherActor = exactAuthorize(t, v, p, key, otherCtx, next)
	if _, err = s.ReconcileMetadata(other, next, otherActor); err != nil {
		t.Fatal("second executor under one grant", err)
	}
	// Current epoch and committed revocation gate even old mutation retries.
	if err = r.RevokeExactAuthority(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReconcileMetadata(sealed, req, actor); err == nil {
		t.Fatal("revoke did not block retained sealed request")
	}
	human := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Verified: true, Subject: p.Owner, Scopes: []string{"agent-manager:read"}})
	receipt, err := s.ReadExactReceipt(human, "reconcile-metadata:"+req.IdempotencyKey, p.EffortRef, p.Owner)
	if err != nil || receipt.Revision != 2 {
		t.Fatal("immutable receipt after revoke", receipt, err)
	}
	if _, err = s.ReadExactReceipt(otherCtx, "reconcile-metadata:"+req.IdempotencyKey, p.EffortRef, p.Owner); err == nil {
		t.Fatal("new agent borrowed human read")
	}
}
func TestExactAuthorityRejectsWrongBindingsAndThinErasure(t *testing.T) {
	s, r, v, p, key, ctx, req := exactFixture(t)
	for name, change := range map[string]func(*ExactEnvelope){
		"purpose": func(e *ExactEnvelope) { e.Purpose = "receipt.read" }, "instance": func(e *ExactEnvelope) { e.Instance = "other" }, "audience": func(e *ExactEnvelope) { e.Audience = "other" }, "issuer": func(e *ExactEnvelope) { e.Issuer = "other" }, "key version": func(e *ExactEnvelope) { e.KeyVersion++ }, "epoch": func(e *ExactEnvelope) { e.Epoch++ }, "initiator": func(e *ExactEnvelope) { e.Initiator = "other" }, "executor": func(e *ExactEnvelope) { e.Executor = uuid.NewString() }, "operation": func(e *ExactEnvelope) { e.Operation = "run.create" }, "target": func(e *ExactEnvelope) { e.Target = "agent-manager:*" }, "payload": func(e *ExactEnvelope) { e.Payload = "wrong" }, "request": func(e *ExactEnvelope) { e.Request = "other" }, "expired": func(e *ExactEnvelope) { e.Expires = v.now().Unix() }, "long lifetime": func(e *ExactEnvelope) { e.Expires = v.now().Add(61 * time.Second).Unix() }, "decoded action": func(e *ExactEnvelope) { e.Action = json.RawMessage(`{}`) },
	} {
		t.Run(name, func(t *testing.T) {
			raw, sig := exactSigned(t, v, p, key, ctx, req, change)
			if _, err := v.Verify(ctx, req, raw, sig); err == nil {
				t.Fatal("accepted wrong binding")
			}
		})
	}
	raw, sig := exactSigned(t, v, p, key, ctx, req, nil)
	if _, err := v.Verify(ctx, req, raw, sig[:len(sig)-1]); err == nil {
		t.Fatal("invalid signature")
	}
	if _, err := v.Verify(context.Background(), req, raw, sig); err == nil {
		t.Fatal("missing identity")
	}
	sealed, actor := exactAuthorize(t, v, p, key, ctx, req)
	old, ob, _ := r.GetEffort(ctx, p.EffortRef)
	old.DestinationRef = "existing-metadata"
	old.Revision++
	if err := r.SaveEffort(ctx, old, ob, 1, "other-change", "other-change"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReconcileMetadata(sealed, req, actor); err == nil {
		t.Fatal("thin proof erased stored metadata")
	}
	current, _, _ := r.GetEffort(ctx, p.EffortRef)
	if current.DestinationRef != "existing-metadata" || current.Revision != 2 {
		t.Fatal("thin rejection effects")
	}
}

func TestExactSiteRevisionAndUnicodeBounds(t *testing.T) {
	_, _, v, p, key, ctx, req := exactFixture(t)
	req.Enrollment.Revision = 2 // Site permits a distinct enrollment revision.
	raw, sig := exactSigned(t, v, p, key, ctx, req, nil)
	if _, err := v.Verify(ctx, req, raw, sig); err != nil {
		t.Fatal("valid Site revision rejected", err)
	}
	if boundedExactText(strings.Repeat("🗝", 61), 120) {
		t.Fatal("UTF-16 Site length bound exceeded")
	}
	req.ExpectedRevision = 9007199254740992
	raw, sig = exactSigned(t, v, p, key, ctx, req, nil)
	if _, err := v.Verify(ctx, req, raw, sig); err == nil {
		t.Fatal("unsafe Site revision accepted")
	}
	if _, err := v.Verify(ctx, req, strings.Repeat("a", 21337), sig); err == nil {
		t.Fatal("oversized proof accepted")
	}
}

func TestExactCurrentPolicyChangesRefuseBeforeReceiptOrEffect(t *testing.T) {
	for name, change := range map[string]func(*ExactPolicy){
		"epoch":       func(p *ExactPolicy) { p.Epoch++ },
		"issuer":      func(p *ExactPolicy) { p.Issuer = "rotated-issuer" },
		"key version": func(p *ExactPolicy) { p.KeyVersion++ },
		"key":         func(p *ExactPolicy) { public, _, _ := ed25519.GenerateKey(rand.Reader); p.PublicKey = public },
		"deadline":    func(p *ExactPolicy) { p.Deadline = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			s, r, v, p, key, ctx, req := exactFixture(t)
			sealed, actor := exactAuthorize(t, v, p, key, ctx, req)
			change(&p)
			raw, _ := json.Marshal(p)
			if _, err := r.db.ExecContext(ctx, `UPDATE keyring_exact_authority SET policy_json=? WHERE id=?`, string(raw), p.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := s.ReconcileMetadata(sealed, req, actor); err == nil {
				t.Fatal("current authority change accepted")
			}
			old, _, err := r.GetEffort(ctx, p.EffortRef)
			if err != nil || old.Revision != 1 {
				t.Fatal("refusal changed domain", err)
			}
			var count int
			if err = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM supervision_effort_transitions WHERE operation_key LIKE 'reconcile-metadata:%'`).Scan(&count); err != nil || count != 0 {
				t.Fatal("refusal changed receipt", count, err)
			}
		})
	}
}

func exactReadFixture(t *testing.T) (*EffortService, *Repository, *ExactVerifier, ExactPolicy, ed25519.PrivateKey, context.Context, ExactReceiptRequest) {
	t.Helper()
	s, r, v, p, key, ctx, req := exactFixture(t)
	p.Operations = []string{"effort.reconcile", "receipt.read"}
	p.ReadEligibility = ExactStandingReadEligibility
	encoded, _ := json.Marshal(p)
	if _, err := r.db.Exec(`UPDATE keyring_exact_authority SET policy_json=? WHERE id=?`, string(encoded), p.ID); err != nil {
		t.Fatal(err)
	}
	sealed, actor := exactAuthorize(t, v, p, key, ctx, req)
	if _, err := s.ReconcileMetadata(sealed, req, actor); err != nil {
		t.Fatal(err)
	}
	original, _ := identity.PrincipalFromContext(ctx)
	reader := identity.Principal{Kind: identity.ActorAgent, Verified: true, Subject: uuid.NewString(), ExpiresAt: v.now().Add(time.Hour)}
	return s, r, v, p, key, identity.WithPrincipal(context.Background(), reader), ExactReceiptRequest{"reconcile-metadata:" + req.IdempotencyKey, p.EffortRef, original.Subject, effortDigest(req)}
}

func exactReadSigned(t *testing.T, v *ExactVerifier, p ExactPolicy, key ed25519.PrivateKey, ctx context.Context, req ExactReceiptRequest, change func(*ExactEnvelope)) (string, string) {
	t.Helper()
	pr, _ := identity.PrincipalFromContext(ctx)
	action, _ := json.Marshal(req)
	e := ExactEnvelope{1, ExactReadPurpose, p.Instance, "agent-manager", p.Issuer, p.KeyVersion, action, p.ID, p.Epoch, p.Owner, pr.Subject, "receipt.read", exactTarget(req.EffortRef), req.commitment(), req.OperationKey, v.now().Add(30 * time.Second).Unix()}
	if change != nil {
		change(&e)
	}
	raw, _ := json.Marshal(e)
	return base64.RawURLEncoding.EncodeToString(raw), base64.RawURLEncoding.EncodeToString(ed25519.Sign(key, raw))
}
func exactReadState(t *testing.T, r *Repository) string {
	t.Helper()
	var effects, receipts, policies string
	for _, item := range []struct {
		query string
		out   *string
	}{
		{`SELECT COALESCE(group_concat(effort_ref||enrollment_json||observation_json),'') FROM (SELECT * FROM supervision_efforts ORDER BY effort_ref)`, &effects},
		{`SELECT COALESCE(group_concat(operation_key||request_digest||result_json||occurred_at),'') FROM (SELECT * FROM supervision_effort_transitions ORDER BY operation_key)`, &receipts},
		{`SELECT COALESCE(group_concat(id||policy_json||revoked),'') FROM (SELECT * FROM keyring_exact_authority ORDER BY id)`, &policies},
	} {
		if err := r.db.QueryRowContext(context.Background(), item.query).Scan(item.out); err != nil {
			t.Fatal(err)
		}
	}
	return effects + receipts + policies
}
func TestExactReceiptFutureReaderUsesCurrentReadAuthorityWithoutMutation(t *testing.T) {
	s, r, v, p, key, ctx, req := exactReadFixture(t)
	// Fresh reader uses the same original standing grant without new approval.
	before := exactReadState(t, r)
	challenge, err := v.ReceiptChallenge(ctx, p.ID, req)
	if err != nil || challenge.Purpose != ExactReadPurpose || challenge.Executor != principalSubject(ctx) || challenge.Payload != req.commitment() {
		t.Fatal("read challenge", err)
	}
	raw, sig := exactReadSigned(t, v, p, key, ctx, req, nil)
	sealed, err := v.VerifyReceipt(ctx, req, raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.ReadExactReceiptProof(sealed, req)
	if err != nil || out.Revision != 2 || out.OriginalExecutor != req.OriginalExecutor || out.OriginalPayload != req.OriginalPayload || out.OriginalExecutor == principalSubject(ctx) {
		t.Fatal("immutable original receipt", out, err)
	}
	if _, err = authn.RequireOperation(sealed, "agent-manager:write", "effort.reconcile", exactTarget(req.EffortRef)); err == nil {
		t.Fatal("read proof authorized mutation")
	}
	if _, err = s.ReadExactReceiptProof(ctx, req); err == nil {
		t.Fatal("unsealed reader accepted")
	}
	if before != exactReadState(t, r) {
		t.Fatal("read changed domain/receipts/authority")
	}
	// Fresh proof can read the same immutable receipt, without a second receipt.
	raw, sig = exactReadSigned(t, v, p, key, ctx, req, func(e *ExactEnvelope) { e.Expires++ })
	sealed, err = v.VerifyReceipt(ctx, req, raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.ReadExactReceiptProof(sealed, req)
	if err != nil || !reflect.DeepEqual(out, again) {
		t.Fatal("fresh read changed receipt", err)
	}
}
func TestExactReceiptWrongBindingsAndOriginalIdentityRefuseZeroEffects(t *testing.T) {
	for _, name := range []string{"owner", "key", "target", "reader", "purpose", "operation", "original executor", "original payload", "original key", "missing receipt", "mutation grant", "source grant", "future receipt epoch", "legacy reader", "legacy source", "expires", "unknown action"} {
		t.Run(name, func(t *testing.T) {
			s, r, v, p, key, ctx, req := exactReadFixture(t)
			change := func(e *ExactEnvelope) {}
			switch name {
			case "owner":
				change = func(e *ExactEnvelope) { e.Initiator = "other-owner" }
			case "key":
				_, key, _ = ed25519.GenerateKey(rand.Reader)
			case "target":
				change = func(e *ExactEnvelope) { e.Target = "agent-manager:effort/other" }
			case "reader":
				change = func(e *ExactEnvelope) { e.Executor = uuid.NewString() }
			case "purpose":
				change = func(e *ExactEnvelope) { e.Purpose = ExactPurpose }
			case "operation":
				change = func(e *ExactEnvelope) { e.Operation = "effort.reconcile" }
			case "original executor":
				req.OriginalExecutor = uuid.NewString()
			case "original payload":
				req.OriginalPayload = strings.Repeat("0", 64)
			case "original key":
				req.OperationKey = "reconcile-metadata:seed"
			case "missing receipt":
				req.OperationKey = "reconcile-metadata:absent"
			case "mutation grant":
				p.Operations = nil
			case "source grant":
				p.ID = "another-grant"
				encoded, _ := json.Marshal(p)
				if _, err := r.db.Exec(`INSERT INTO keyring_exact_authority(id,policy_json) VALUES(?,?)`, p.ID, string(encoded)); err != nil {
					t.Fatal(err)
				}
			case "future receipt epoch":
				var digest string
				_ = r.db.QueryRowContext(ctx, `SELECT request_digest FROM supervision_effort_transitions WHERE operation_key=?`, req.OperationKey).Scan(&digest)
				var mutation exactMutation
				_ = json.Unmarshal([]byte(strings.TrimPrefix(digest, "keyring-exact-v1:")), &mutation)
				mutation.Epoch = p.Epoch + 1
				encoded, _ := json.Marshal(mutation)
				if _, err := r.db.Exec(`UPDATE supervision_effort_transitions SET request_digest=? WHERE operation_key=?`, "keyring-exact-v1:"+string(encoded), req.OperationKey); err != nil {
					t.Fatal(err)
				}
			case "legacy reader":
				p.Reader = uuid.NewString()
			case "legacy source":
				p.ReceiptSource = &ExactReceiptSource{p.ID, p.Epoch, p.Issuer, p.KeyVersion}
			case "expires":
				change = func(e *ExactEnvelope) { e.Expires = v.now().Unix() }
			case "unknown action":
				change = func(e *ExactEnvelope) { e.Action = json.RawMessage(`{"unexpected":true}`) }
			}
			policy, _ := json.Marshal(p)
			if _, err := r.db.Exec(`UPDATE keyring_exact_authority SET policy_json=? WHERE id=?`, string(policy), p.ID); err != nil {
				t.Fatal(err)
			}
			before := exactReadState(t, r)
			raw, sig := exactReadSigned(t, v, p, key, ctx, req, change)
			sealed, err := v.VerifyReceipt(ctx, req, raw, sig)
			if err == nil {
				_, err = s.ReadExactReceiptProof(sealed, req)
			}
			if err == nil {
				t.Fatal("wrong read accepted")
			}
			if before != exactReadState(t, r) {
				t.Fatal("rejection changed persistent state")
			}
		})
	}
}
func TestExactReceiptCurrentPolicyAndProvenanceRefuseZeroEffects(t *testing.T) {
	for _, name := range []string{"revoked", "epoch", "key", "issuer", "key version", "owner", "reader", "deadline", "operations", "source", "context", "principal", "request"} {
		t.Run(name, func(t *testing.T) {
			s, r, v, p, key, ctx, req := exactReadFixture(t)
			raw, sig := exactReadSigned(t, v, p, key, ctx, req, nil)
			sealed, err := v.VerifyReceipt(ctx, req, raw, sig)
			if err != nil {
				t.Fatal(err)
			}
			switch name {
			case "revoked":
				if err = r.RevokeExactAuthority(ctx, p.ID); err != nil {
					t.Fatal(err)
				}
			case "epoch":
				p.Epoch++
			case "key":
				p.PublicKey, _, _ = ed25519.GenerateKey(rand.Reader)
			case "issuer":
				p.Issuer = "rotated"
			case "key version":
				p.KeyVersion++
			case "owner":
				p.Owner = "other-owner"
			case "reader":
				p.Reader = uuid.NewString()
			case "deadline":
				p.Deadline = 0
			case "operations":
				p.Operations = []string{}
			case "source":
				p.ReadEligibility = "unqualified"
			case "context":
				sealed = ctx
			case "principal":
				pr, _ := identity.PrincipalFromContext(ctx)
				pr.Subject = uuid.NewString()
				sealed = identity.WithPrincipal(sealed, pr)
			case "request":
				req.OriginalPayload = strings.Repeat("0", 64)
			}
			policy, _ := json.Marshal(p)
			if _, err = r.db.Exec(`UPDATE keyring_exact_authority SET policy_json=? WHERE id=?`, string(policy), p.ID); err != nil {
				t.Fatal(err)
			}
			before := exactReadState(t, r)
			if _, err = s.ReadExactReceiptProof(sealed, req); err == nil {
				t.Fatal("changed read authority accepted")
			}
			if before != exactReadState(t, r) {
				t.Fatal("rejected read changed state")
			}
		})
	}
}

func principalSubject(ctx context.Context) string {
	p, _ := identity.PrincipalFromContext(ctx)
	return p.Subject
}
func TestExactStandingGrantFreshReadersRotationAndImmutableMutation(t *testing.T) {
	s, r, v, p, key, ctx, req := exactReadFixture(t)
	originalOwner := p.Owner
	for i := 0; i < 3; i++ {
		reader := identity.Principal{Kind: identity.ActorAgent, Verified: true, Subject: uuid.NewString(), ExpiresAt: v.now().Add(time.Hour)}
		ctx = identity.WithPrincipal(context.Background(), reader)
		raw, sig := exactReadSigned(t, v, p, key, ctx, req, nil)
		sealed, err := v.VerifyReceipt(ctx, req, raw, sig)
		if err != nil {
			t.Fatal(err)
		}
		before := exactReadState(t, r)
		if out, err := s.ReadExactReceiptProof(sealed, req); err != nil || out.OriginalExecutor != req.OriginalExecutor {
			t.Fatal("same grant fresh reader", err)
		}
		if before != exactReadState(t, r) {
			t.Fatal("reader caused approval/state change")
		}
	}
	var count int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM keyring_exact_authority`).Scan(&count)
	if count != 1 {
		t.Fatal("per-reader policy created", count)
	}
	p.Epoch++
	p.KeyVersion++
	p.Issuer = "rotated-existing-issuer"
	p.PublicKey, key, _ = ed25519.GenerateKey(rand.Reader)
	encoded, _ := json.Marshal(p)
	if _, err := r.db.Exec(`UPDATE keyring_exact_authority SET policy_json=? WHERE id=?`, string(encoded), p.ID); err != nil {
		t.Fatal(err)
	}
	before := exactReadState(t, r)
	raw, sig := exactReadSigned(t, v, p, key, ctx, req, nil)
	sealed, err := v.VerifyReceipt(ctx, req, raw, sig)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.ReadExactReceiptProof(sealed, req)
	if err != nil || out.OriginalPayload != req.OriginalPayload {
		t.Fatal("historical original under current rotated authority", err)
	}
	originalCtx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Verified: true, Subject: req.OriginalExecutor, ExpiresAt: v.now().Add(time.Hour)})
	mutation := &pb.ReconcileEffortMetadataRequest{Enrollment: &pb.EffortEnrollment{EffortRef: p.EffortRef, DisplayName: "after", Revision: 1, AuthorizedBy: p.AuthorizedBy, AuthorityRef: p.AuthorityRef}, ExpectedRevision: 1, IdempotencyKey: strings.TrimPrefix(req.OperationKey, "reconcile-metadata:"), Authority: pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}
	mutationCtx, actor := exactAuthorize(t, v, p, key, originalCtx, mutation)
	if _, err = s.ReconcileMetadata(mutationCtx, mutation, actor); !errors.Is(err, ErrConflict) {
		t.Fatal("rotation adopted immutable mutation", err)
	}
	if p.Owner != originalOwner || before != exactReadState(t, r) {
		t.Fatal("rotation read/retry rewrote original receipt")
	}
	if err = r.RevokeExactAuthority(ctx, p.ID); err != nil {
		t.Fatal(err)
	}
	before = exactReadState(t, r)
	if _, err = s.ReadExactReceiptProof(sealed, req); err == nil {
		t.Fatal("revoked standing grant read")
	}
	if before != exactReadState(t, r) {
		t.Fatal("revoked read effects")
	}
}
