// Package nodeauth is the control-plane half of mutual auth (SECURITY.md
// boundary 2): it proves an inbound node→control-plane call really comes from
// the paired node, not an impostor. The node signs a canonical, timestamped
// payload with the Ed25519 private key it generated at pairing; the control
// plane verifies the signature against the PUBLIC key it stored for that node
// (the pairing domain's node_credentials), checks the credential is still
// active (revocation severs auth instantly), and checks the timestamp is fresh
// (anti-replay). The control-plane→node direction is the mirror image and lives
// in cpkeys (the node pins the CP public key and verifies pushes).
//
// The signing scheme is deliberately transport-agnostic so the same proof rides
// a Connect-RPC header set OR an EventSource `?token=` query (SSE cannot set
// headers). Both reduce to (node_id, unix_ts, signature-over-payload).
package nodeauth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Request header names carrying the node's auth proof on Connect-RPC calls.
const (
	HeaderNode = "X-Bridge-Node"
	HeaderTS   = "X-Bridge-Ts"
	HeaderSig  = "X-Bridge-Sig"
)

// DefaultMaxClockSkew bounds how far a signed timestamp may be from the control
// plane's clock, in either direction. A signed payload older/newer than this is
// rejected, so a captured signature cannot be replayed indefinitely.
const DefaultMaxClockSkew = 5 * time.Minute

// Typed sentinels so the handler maps each failure to the right Connect code
// (all are Unauthenticated to the caller — they exist for logging/tests).
var (
	ErrMissingProof  = errors.New("nodeauth: missing node auth proof")
	ErrUnknownNode   = errors.New("nodeauth: no active credential for node")
	ErrStaleProof    = errors.New("nodeauth: auth timestamp outside allowed skew")
	ErrBadSignature  = errors.New("nodeauth: signature does not verify")
	ErrMalformedAuth = errors.New("nodeauth: malformed auth proof")
)

// CredentialStore is the seam nodeauth reads node public keys through. The
// pairing sqlite repository satisfies it; tests substitute a fake. It MUST
// return ok=false for an unknown OR revoked node so revocation severs auth.
type CredentialStore interface {
	ActivePublicKey(ctx context.Context, nodeID string) (ed25519.PublicKey, bool, error)
}

// SigningPayload is the canonical byte string a node signs to authenticate a
// call: the node id and the unix-second timestamp, newline-separated. Both
// sides MUST build it identically; binding the node id prevents a signature
// minted for one node being replayed as another.
func SigningPayload(nodeID string, ts time.Time) []byte {
	return []byte(nodeID + "\n" + strconv.FormatInt(ts.Unix(), 10))
}

// Verifier checks node auth proofs against the credential store with a bounded
// clock skew.
type Verifier struct {
	store CredentialStore
	now   func() time.Time
	skew  time.Duration
}

// Option customises a Verifier (clock, skew) for tests.
type Option func(*Verifier)

// WithClock overrides the time source.
func WithClock(now func() time.Time) Option { return func(v *Verifier) { v.now = now } }

// WithMaxClockSkew overrides the anti-replay window.
func WithMaxClockSkew(d time.Duration) Option { return func(v *Verifier) { v.skew = d } }

// NewVerifier constructs a Verifier over the given credential store.
func NewVerifier(store CredentialStore, opts ...Option) *Verifier {
	v := &Verifier{store: store, now: time.Now, skew: DefaultMaxClockSkew}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Verify checks that sig is a valid Ed25519 signature by nodeID over
// SigningPayload(nodeID, ts), that nodeID has an active credential, and that ts
// is within the allowed skew of now. It returns a typed sentinel on failure.
func (v *Verifier) Verify(ctx context.Context, nodeID string, ts time.Time, sig []byte) error {
	if nodeID == "" || len(sig) == 0 {
		return ErrMissingProof
	}
	if delta := v.now().Sub(ts); delta > v.skew || delta < -v.skew {
		return ErrStaleProof
	}
	pub, ok, err := v.store.ActivePublicKey(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("nodeauth: lookup credential: %w", err)
	}
	if !ok {
		return ErrUnknownNode
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, SigningPayload(nodeID, ts), sig) {
		return ErrBadSignature
	}
	return nil
}

// Proof is a parsed, not-yet-verified node auth proof.
type Proof struct {
	NodeID    string
	Timestamp time.Time
	Signature []byte
}

// VerifyProof verifies an already-parsed proof.
func (v *Verifier) VerifyProof(ctx context.Context, p Proof) error {
	return v.Verify(ctx, p.NodeID, p.Timestamp, p.Signature)
}

// ParseHeaders extracts a Proof from the X-Bridge-* Connect-RPC headers.
func ParseHeaders(nodeID, tsHeader, sigHeader string) (Proof, error) {
	if nodeID == "" || tsHeader == "" || sigHeader == "" {
		return Proof{}, ErrMissingProof
	}
	secs, err := strconv.ParseInt(strings.TrimSpace(tsHeader), 10, 64)
	if err != nil {
		return Proof{}, fmt.Errorf("%w: timestamp: %v", ErrMalformedAuth, err)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigHeader))
	if err != nil {
		return Proof{}, fmt.Errorf("%w: signature: %v", ErrMalformedAuth, err)
	}
	return Proof{NodeID: nodeID, Timestamp: time.Unix(secs, 0), Signature: sig}, nil
}

// EncodeToken renders a self-contained SSE dial-out token "<nodeID>.<unixTs>.
// <base64sig>" for the EventSource `?token=` path (which cannot set headers).
func EncodeToken(nodeID string, ts time.Time, sig []byte) string {
	return strings.Join([]string{
		nodeID,
		strconv.FormatInt(ts.Unix(), 10),
		base64.StdEncoding.EncodeToString(sig),
	}, ".")
}

// ParseToken parses a dial-out token produced by EncodeToken.
func ParseToken(token string) (Proof, error) {
	parts := strings.SplitN(strings.TrimSpace(token), ".", 3)
	if len(parts) != 3 {
		return Proof{}, ErrMalformedAuth
	}
	return ParseHeaders(parts[0], parts[1], parts[2])
}
