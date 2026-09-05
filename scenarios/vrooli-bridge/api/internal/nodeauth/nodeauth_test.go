package nodeauth

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

// fakeStore is the credential-lookup seam double. A node is present iff it is in
// keys AND not in revoked — mirroring "active credential only".
type fakeStore struct {
	keys    map[string]ed25519.PublicKey
	revoked map[string]bool
	err     error
}

func (f *fakeStore) ActivePublicKey(_ context.Context, nodeID string) (ed25519.PublicKey, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	if f.revoked[nodeID] {
		return nil, false, nil
	}
	pub, ok := f.keys[nodeID]
	return pub, ok, nil
}

func newNode(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	return pub, priv
}

func fixedClock(ts time.Time) Option { return WithClock(func() time.Time { return ts }) }

// [REQ:BRG-P0-002] A correctly-signed, fresh proof from a node with an active
// credential verifies — the happy path of node→control-plane mutual auth.
func TestVerify_AcceptsValidProof(t *testing.T) {
	pub, priv := newNode(t)
	now := time.Unix(1_700_000_000, 0)
	store := &fakeStore{keys: map[string]ed25519.PublicKey{"node-1": pub}}
	v := NewVerifier(store, fixedClock(now))

	sig := ed25519.Sign(priv, SigningPayload("node-1", now))
	if err := v.Verify(context.Background(), "node-1", now, sig); err != nil {
		t.Fatalf("valid proof rejected: %v", err)
	}
}

// [REQ:BRG-P0-002] A rogue node with no stored credential is rejected — it
// cannot enrol itself by merely presenting a self-generated key.
func TestVerify_RejectsUnknownNode(t *testing.T) {
	_, priv := newNode(t)
	now := time.Unix(1_700_000_000, 0)
	v := NewVerifier(&fakeStore{keys: map[string]ed25519.PublicKey{}}, fixedClock(now))

	sig := ed25519.Sign(priv, SigningPayload("rogue", now))
	if err := v.Verify(context.Background(), "rogue", now, sig); err != ErrUnknownNode {
		t.Fatalf("got %v, want ErrUnknownNode", err)
	}
}

// [REQ:BRG-P0-002] Revocation severs auth immediately: a node whose credential
// is revoked fails verification even with a perfectly valid signature.
func TestVerify_RejectsRevokedNode(t *testing.T) {
	pub, priv := newNode(t)
	now := time.Unix(1_700_000_000, 0)
	store := &fakeStore{
		keys:    map[string]ed25519.PublicKey{"node-1": pub},
		revoked: map[string]bool{"node-1": true},
	}
	v := NewVerifier(store, fixedClock(now))

	sig := ed25519.Sign(priv, SigningPayload("node-1", now))
	if err := v.Verify(context.Background(), "node-1", now, sig); err != ErrUnknownNode {
		t.Fatalf("got %v, want ErrUnknownNode (revoked == no active credential)", err)
	}
}

// [REQ:BRG-P0-002] A signature minted for one node cannot be replayed as
// another: the node id is bound into the signed payload.
func TestVerify_RejectsCrossNodeReplay(t *testing.T) {
	pubA, privA := newNode(t)
	pubB, _ := newNode(t)
	now := time.Unix(1_700_000_000, 0)
	store := &fakeStore{keys: map[string]ed25519.PublicKey{"node-a": pubA, "node-b": pubB}}
	v := NewVerifier(store, fixedClock(now))

	// node-a signs as itself, attacker presents it claiming to be node-b.
	sigA := ed25519.Sign(privA, SigningPayload("node-a", now))
	if err := v.Verify(context.Background(), "node-b", now, sigA); err != ErrBadSignature {
		t.Fatalf("got %v, want ErrBadSignature", err)
	}
}

// [REQ:BRG-P0-002] A captured signature cannot be replayed indefinitely: a
// timestamp outside the skew window is rejected in both directions.
func TestVerify_RejectsStaleAndFutureProofs(t *testing.T) {
	pub, priv := newNode(t)
	now := time.Unix(1_700_000_000, 0)
	store := &fakeStore{keys: map[string]ed25519.PublicKey{"node-1": pub}}
	v := NewVerifier(store, fixedClock(now), WithMaxClockSkew(time.Minute))

	for _, ts := range []time.Time{now.Add(-2 * time.Minute), now.Add(2 * time.Minute)} {
		sig := ed25519.Sign(priv, SigningPayload("node-1", ts))
		if err := v.Verify(context.Background(), "node-1", ts, sig); err != ErrStaleProof {
			t.Fatalf("ts %v: got %v, want ErrStaleProof", ts, err)
		}
	}
}

// [REQ:BRG-P0-002] A tampered signature is rejected.
func TestVerify_RejectsTamperedSignature(t *testing.T) {
	pub, priv := newNode(t)
	now := time.Unix(1_700_000_000, 0)
	store := &fakeStore{keys: map[string]ed25519.PublicKey{"node-1": pub}}
	v := NewVerifier(store, fixedClock(now))

	sig := ed25519.Sign(priv, SigningPayload("node-1", now))
	sig[0] ^= 0xFF
	if err := v.Verify(context.Background(), "node-1", now, sig); err != ErrBadSignature {
		t.Fatalf("got %v, want ErrBadSignature", err)
	}
}

// [REQ:BRG-P0-002] The header and SSE-token encodings round-trip to the same
// proof, so the same signature authenticates a Connect call and a dial-out.
func TestProofEncodings_RoundTrip(t *testing.T) {
	pub, priv := newNode(t)
	now := time.Unix(1_700_000_000, 0)
	store := &fakeStore{keys: map[string]ed25519.PublicKey{"node-1": pub}}
	v := NewVerifier(store, fixedClock(now))
	sig := ed25519.Sign(priv, SigningPayload("node-1", now))

	tokenProof, err := ParseToken(EncodeToken("node-1", now, sig))
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if err := v.VerifyProof(context.Background(), tokenProof); err != nil {
		t.Fatalf("token proof rejected: %v", err)
	}

	sigB64 := base64.StdEncoding.EncodeToString(sig)
	headerProof, err := ParseHeaders("node-1", "1700000000", sigB64)
	if err != nil {
		t.Fatalf("parse headers: %v", err)
	}
	if err := v.VerifyProof(context.Background(), headerProof); err != nil {
		t.Fatalf("header proof rejected: %v", err)
	}
}

func TestParseToken_RejectsMalformed(t *testing.T) {
	if _, err := ParseToken("only.two"); err != ErrMalformedAuth {
		t.Fatalf("got %v, want ErrMalformedAuth", err)
	}
}
