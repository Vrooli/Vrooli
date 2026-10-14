package finitesigning

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/vrooli/api-core/effortauthority"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"testing"
	"time"
)

type fixtureStore struct {
	record provisionedRecord
	reads  int
	calls  []string
}

func (s *fixtureStore) Require(id credentialauthority.Identity, field string) (string, error) {
	s.reads++
	s.calls = append(s.calls, string(id)+":"+field)
	b, _ := json.Marshal(s.record)
	return string(b), nil
}
func signingFixture(t *testing.T) (*Signer, *fixtureStore, effortauthority.Policy, effortauthority.Intent) {
	t.Helper()
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC()
	d := effortauthority.Digest("fixture")
	p := effortauthority.Policy{ID: "fixture", Owner: "verified-owner", Client: "fixture-client", ClientKey: pub, Epoch: 1, Repository: t.TempDir(), Effort: "fixture-effort", Revision: "1", ContentDigest: d, Team: "fixture-team", TeamDigest: d, BindingDigest: d, Members: []string{"lead"}, Profiles: map[string]string{"profile": d}, Scopes: []string{"agent-manager:write"}, Effects: []string{"run.create"}, NotBefore: now.Add(-time.Minute), Deadline: now.Add(time.Hour), MaxStarts: 2, MaxConcurrent: 1, MaxTurns: 4, MaxToolCalls: 5, MaxRunSeconds: 60, TotalTurns: 8, TotalToolCalls: 10, TotalRunSeconds: 120, SurviveLogout: true}
	store := &fixtureStore{record: provisionedRecord{Version: 1, Purpose: Purpose, HandleID: "fixture-handle", Client: p.Client, PolicyDigest: effortauthority.Digest(p), PublicKey: base64.RawStdEncoding.EncodeToString(pub), PrivateKey: base64.RawStdEncoding.EncodeToString(key)}}
	s, e := newSigner(store, Descriptor{Identity: "finite-effort/fixture", HandleID: "fixture-handle", PublicKey: pub, Policy: p})
	if e != nil {
		t.Fatal(e)
	}
	i := effortauthority.Intent{Audience: effortauthority.Audience, Method: "POST", Endpoint: "/api/v1/runs", PolicyID: p.ID, Epoch: p.Epoch, Client: p.Client, Repository: p.Repository, Effort: p.Effort, Revision: p.Revision, ContentDigest: p.ContentDigest, Team: p.Team, Member: "lead", Profile: "profile", ProfileDigest: d, Effect: "run.create", IdempotencyKey: "fixture-one", InputDigest: d, Turns: 4, ToolCalls: 5, RunSeconds: 60}
	return s, store, p, i
}
func TestFinitePurposePositiveAndCurrentRevocation(t *testing.T) {
	s, store, p, i := signingFixture(t)
	ctx := context.Background()
	if s.CheckPolicy(ctx, p) != nil {
		t.Fatal("purpose read")
	}
	proof, e := s.SignPolicyProof(ctx, p, effortauthority.Proof{Intent: i, Nonce: "offered", IssuedAt: time.Now().Add(time.Hour)})
	if e != nil || proof.Nonce == "offered" || len(proof.Signature) != 64 {
		t.Fatal("typed positive", e)
	}
	unsigned := proof
	unsigned.Signature = nil
	raw, _ := json.Marshal(unsigned)
	if !ed25519.Verify(p.ClientKey, raw, proof.Signature) {
		t.Fatal("incompatible proof carrier")
	}
	store.record.Revoked = true
	if _, e = s.SignPolicyProof(ctx, p, effortauthority.Proof{Intent: i}); e == nil {
		t.Fatal("revoked purpose signed")
	}
	if store.calls[0] != "finite-effort/fixture:"+Field {
		t.Fatal("wrong authority source")
	}
}
func TestFinitePurposeWrongIntentHasZeroCredentialReads(t *testing.T) {
	for _, mode := range []string{"audience", "client", "effect", "input", "budget", "parent", "revision"} {
		t.Run(mode, func(t *testing.T) {
			s, store, p, i := signingFixture(t)
			switch mode {
			case "audience":
				i.Audience = "receipt"
			case "client":
				i.Client = "other"
			case "effect":
				i.Effect = "run.child"
			case "input":
				i.InputDigest = "bad"
			case "budget":
				i.Turns++
			case "parent":
				i.ParentRunID = "parent"
			case "revision":
				i.Revision = "2"
			}
			if _, e := s.SignPolicyProof(context.Background(), p, effortauthority.Proof{Intent: i}); e == nil || store.reads != 0 {
				t.Fatal("wrong typed intent reached custody", e, store.reads)
			}
		})
	}
}
func TestFinitePurposeRefusesReceiptOrWrongHandleAndSnapshotMutation(t *testing.T) {
	s, store, p, i := signingFixture(t)
	for _, mode := range []string{"purpose", "handle", "public", "policy", "key"} {
		old := store.record
		switch mode {
		case "purpose":
			store.record.Purpose = "receipt"
		case "handle":
			store.record.HandleID = "other"
		case "public":
			store.record.PublicKey = "bad"
		case "policy":
			store.record.PolicyDigest = effortauthority.Digest("other")
		case "key":
			store.record.PrivateKey = "bad"
		}
		if _, e := s.SignPolicyProof(context.Background(), p, effortauthority.Proof{Intent: i}); e == nil {
			t.Fatal(mode)
		}
		store.record = old
	}
	p.Profiles["profile"] = "mutated"
	if s.CheckPolicy(context.Background(), p) == nil {
		t.Fatal("mutable policy accepted")
	}
}
func TestFinitePurposeDisabledConstructorHasNoAuthorityRead(t *testing.T) {
	if _, e := New(Config{}); e == nil {
		t.Fatal("nil authority")
	}
}

func TestFinitePurposeCorruptSeedRefusesEligibilityAndSigning(t *testing.T) {
	s, store, p, i := signingFixture(t)
	raw, e := base64.RawStdEncoding.DecodeString(store.record.PrivateKey)
	if e != nil {
		t.Fatal(e)
	}
	raw[0] ^= 1
	store.record.PrivateKey = base64.RawStdEncoding.EncodeToString(raw)
	clear(raw)
	if s.CheckPolicy(context.Background(), p) == nil {
		t.Fatal("corrupt key published eligibility")
	}
	if _, e = s.SignPolicyProof(context.Background(), p, effortauthority.Proof{Intent: i}); e == nil {
		t.Fatal("corrupt seed signed")
	}
}
