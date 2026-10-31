// Package cpverify is the node-agent's half of control-plane → node mutual auth:
// it holds the control-plane public key the node pinned at pairing
// (<state-dir>/control_plane.pub) and verifies every pushed ServerFrame's
// signature against it BEFORE the agent acts on the frame. A frame that is
// unsigned, mis-signed, or signed by a different key is rejected — so a node
// never executes a job or provisioning command pushed by an impostor control
// plane (SECURITY.md boundary 2). This is the READ half of the pin whose WRITE
// half is `pair redeem` persisting the key returned at pairing.
//
// The envelope + signing contract MUST match the control plane's api
// internal/channelsign byte-for-byte (the same way nodecred mirrors nodeauth):
//
//   - the SSE `data:` payload is a protojson-encoded channel.SignedServerFrame
//   - the signature is Ed25519 over the EXACT `frame` bytes it carries
//   - `frame` is a proto-serialised channel.ServerFrame
//
// Change both sides together.
package cpverify

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// ErrNoPin is returned by Load when the pinned key file is absent. It is a hard,
// actionable failure: a paired agent MUST have pinned the control-plane key at
// bootstrap (`pair redeem` writes it), and there is deliberately no
// trust-on-first-use fallback.
var ErrNoPin = errors.New("control-plane public key not pinned")

// Verifier holds the pinned control-plane public key and verifies signed frames.
type Verifier struct {
	pub ed25519.PublicKey
}

// Load reads the pinned control-plane public key from path (base64 Ed25519 — the
// exact string `pair redeem` returns and persists). A missing file is ErrNoPin
// (wrapped with the path so the operator knows where the bootstrap must write
// it); a malformed file is a hard error. There is no fallback: an agent that
// cannot load its pin must not dial.
func Load(path string) (*Verifier, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- path is the agent's own pinned control-plane key path, not user input.
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("%w at %q", ErrNoPin, path)
	case err != nil:
		return nil, fmt.Errorf("cpverify: read pinned key %q: %w", path, err)
	}
	pub, err := decodeKey(string(raw))
	if err != nil {
		return nil, fmt.Errorf("cpverify: pinned key %q: %w", path, err)
	}
	return &Verifier{pub: pub}, nil
}

// NewVerifier builds a Verifier from an already-decoded key (tests / in-process
// wiring). It validates the key size.
func NewVerifier(pub ed25519.PublicKey) (*Verifier, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("cpverify: public key must be %d bytes, got %d", ed25519.PublicKeySize, len(pub))
	}
	return &Verifier{pub: append(ed25519.PublicKey(nil), pub...)}, nil
}

func decodeKey(s string) (ed25519.PublicKey, error) {
	dec, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("must be standard base64: %w", err)
	}
	if len(dec) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("must be a %d-byte Ed25519 key, got %d", ed25519.PublicKeySize, len(dec))
	}
	return ed25519.PublicKey(dec), nil
}

// Open verifies payload against the pinned key and returns the inner
// ServerFrame. Any failure — unparseable envelope, or a signature that does not
// verify against the pinned key — returns a nil frame and a non-nil error; the
// caller MUST NOT act on the frame in that case.
func (v *Verifier) Open(payload []byte) (*channelv1.ServerFrame, error) {
	var env channelv1.SignedServerFrame
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &env); err != nil {
		return nil, fmt.Errorf("cpverify: unmarshal envelope: %w", err)
	}
	if !ed25519.Verify(v.pub, env.GetFrame(), env.GetSignature()) {
		return nil, errors.New("cpverify: signature does not verify against the pinned control-plane key")
	}
	var frame channelv1.ServerFrame
	if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(env.GetFrame(), &frame); err != nil {
		return nil, fmt.Errorf("cpverify: unmarshal frame: %w", err)
	}
	return &frame, nil
}
