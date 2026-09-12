// Package channelsign is the single choke point through which every
// control-plane → node push is signed before it is written to the node's SSE
// stream (SECURITY.md boundary 2, DECISIONS.md 2026-06-18). It serialises a
// channel.ServerFrame, signs the exact bytes with the control plane's long-lived
// Ed25519 identity key (internal/cpkeys), wraps them in a SignedServerFrame
// envelope, and returns the protojson payload the presence hub pushes.
//
// Keeping this in one place is the whole point: the node rejects any frame that
// does not verify against the pinned control-plane key (agent internal/cpverify),
// so a push path that forgot to sign would simply be dropped by every node. This
// package makes "all pushes are signed" a compile-time seam the two channel push
// adapters (queue + provision) share, rather than a rule each remembers.
package channelsign

import (
	"crypto/ed25519"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// Signer signs the serialized frame bytes with the control plane's identity key.
// *cpkeys.Keypair satisfies it; a test can substitute any Ed25519 key.
type Signer interface {
	// Sign returns the Ed25519 signature over msg.
	Sign(msg []byte) []byte
}

// Marshal serialises frame, signs the exact bytes with signer, wraps them in a
// SignedServerFrame envelope, and returns the protojson payload written to the
// node's SSE stream as one `data:` event. The signature covers the precise
// `frame` bytes carried on the wire so the node verifies over what it received,
// never a re-serialisation.
func Marshal(signer Signer, frame *channelv1.ServerFrame) ([]byte, error) {
	if signer == nil {
		return nil, fmt.Errorf("channelsign: nil signer (control-plane identity key not wired)")
	}
	inner, err := proto.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("channelsign: marshal frame: %w", err)
	}
	env := &channelv1.SignedServerFrame{
		Frame:     inner,
		Signature: signer.Sign(inner),
	}
	payload, err := protojson.Marshal(env)
	if err != nil {
		return nil, fmt.Errorf("channelsign: marshal envelope: %w", err)
	}
	return payload, nil
}

// Open is the symmetric verify half of Marshal: it decodes the envelope, checks
// the signature against pub, and returns the inner ServerFrame. It is the exact
// contract the node-agent implements in its own module (agent internal/cpverify,
// which cannot import this package); it lives here too so the control plane can
// assert its own signed frames round-trip in tests. Any failure — unparseable
// envelope, wrong-size key, or a signature that does not verify — returns a nil
// frame and a non-nil error, so a caller can never act on an unverified frame.
func Open(pub ed25519.PublicKey, payload []byte) (*channelv1.ServerFrame, error) {
	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("channelsign: bad public key size %d (want %d)", len(pub), ed25519.PublicKeySize)
	}
	var env channelv1.SignedServerFrame
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(payload, &env); err != nil {
		return nil, fmt.Errorf("channelsign: unmarshal envelope: %w", err)
	}
	if !ed25519.Verify(pub, env.GetFrame(), env.GetSignature()) {
		return nil, errors.New("channelsign: signature does not verify against the control-plane key")
	}
	var frame channelv1.ServerFrame
	if err := (proto.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(env.GetFrame(), &frame); err != nil {
		return nil, fmt.Errorf("channelsign: unmarshal frame: %w", err)
	}
	return &frame, nil
}
