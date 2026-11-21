package channelsign

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// edSigner is a minimal Signer over a raw Ed25519 private key so the vectors are
// deterministic (a fixed seed), independent of cpkeys' file persistence.
type edSigner struct{ priv ed25519.PrivateKey }

func (s edSigner) Sign(msg []byte) []byte { return ed25519.Sign(s.priv, msg) }

// deterministicKey builds a fixed keypair from a 32-byte seed so signatures are
// reproducible test vectors.
func deterministicKey(seedByte byte) (ed25519.PublicKey, edSigner) {
	seed := bytes.Repeat([]byte{seedByte}, ed25519.SeedSize)
	priv := ed25519.NewKeyFromSeed(seed)
	return priv.Public().(ed25519.PublicKey), edSigner{priv: priv}
}

func sampleFrame() *channelv1.ServerFrame {
	return &channelv1.ServerFrame{
		Payload: &channelv1.ServerFrame_Job{
			Job: &channelv1.JobPush{
				RunId:          "run-1",
				Scenario:       "web-search",
				Verb:           "scenario test",
				Args:           []string{"web-search"},
				TimeoutSeconds: 30,
			},
		},
	}
}

// TestMarshal_SignsExactFrameBytes pins the wire contract: the signature is
// Ed25519 over the EXACT proto-serialised ServerFrame bytes the envelope carries
// — the property the agent relies on when it verifies over received bytes.
func TestMarshal_SignsExactFrameBytes(t *testing.T) {
	pub, signer := deterministicKey(0x01)
	frame := sampleFrame()

	payload, err := Marshal(signer, frame)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var env channelv1.SignedServerFrame
	if err := protojson.Unmarshal(payload, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	wantFrame, err := proto.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	if !bytes.Equal(env.GetFrame(), wantFrame) {
		t.Fatalf("envelope frame bytes = %x, want %x", env.GetFrame(), wantFrame)
	}
	wantSig := ed25519.Sign(signer.priv, wantFrame)
	if !bytes.Equal(env.GetSignature(), wantSig) {
		t.Fatalf("signature = %x, want %x", env.GetSignature(), wantSig)
	}
	if !ed25519.Verify(pub, env.GetFrame(), env.GetSignature()) {
		t.Fatal("signature does not verify against the signer's public key")
	}
}

// TestOpen_RoundTrip: a validly signed frame opens back to the original.
func TestOpen_RoundTrip(t *testing.T) {
	pub, signer := deterministicKey(0x02)
	payload, err := Marshal(signer, sampleFrame())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	frame, err := Open(pub, payload)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := frame.GetJob().GetRunId(); got != "run-1" {
		t.Fatalf("run id = %q, want run-1", got)
	}
}

// TestOpen_RejectsWrongKey: a frame signed by one key never verifies against a
// different pinned key (the impostor-control-plane case).
func TestOpen_RejectsWrongKey(t *testing.T) {
	_, signer := deterministicKey(0x03)
	otherPub, _ := deterministicKey(0x04)
	payload, err := Marshal(signer, sampleFrame())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if _, err := Open(otherPub, payload); err == nil {
		t.Fatal("Open accepted a frame signed by a different key")
	}
}

// TestOpen_RejectsTamperedFrame: mutating the signed bytes invalidates the
// signature.
func TestOpen_RejectsTamperedFrame(t *testing.T) {
	pub, signer := deterministicKey(0x05)
	payload, err := Marshal(signer, sampleFrame())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var env channelv1.SignedServerFrame
	if err := protojson.Unmarshal(payload, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	// Flip a byte of the signed frame, keeping the original signature.
	tampered := append([]byte(nil), env.GetFrame()...)
	tampered[0] ^= 0xFF
	env.Frame = tampered
	bad, err := protojson.Marshal(&env)
	if err != nil {
		t.Fatalf("re-marshal envelope: %v", err)
	}
	if _, err := Open(pub, bad); err == nil {
		t.Fatal("Open accepted a tampered frame")
	}
}

// TestOpen_RejectsUnsignedEnvelope: an envelope with no signature (an attacker
// simply omitting it) is rejected.
func TestOpen_RejectsUnsignedEnvelope(t *testing.T) {
	pub, _ := deterministicKey(0x06)
	inner, err := proto.Marshal(sampleFrame())
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	env := &channelv1.SignedServerFrame{Frame: inner} // no signature
	payload, err := protojson.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if _, err := Open(pub, payload); err == nil {
		t.Fatal("Open accepted an unsigned envelope")
	}
}

// TestOpen_RejectsGarbage: a non-envelope payload (e.g. a bare ServerFrame from a
// pre-signing control plane) is rejected, not silently acted on.
func TestOpen_RejectsGarbage(t *testing.T) {
	pub, _ := deterministicKey(0x07)
	if _, err := Open(pub, []byte("not json")); err == nil {
		t.Fatal("Open accepted non-JSON payload")
	}
}

// TestMarshal_NilSigner is a guard: a push path that forgot to wire the CP key
// fails loudly rather than emitting an unsigned frame every node would drop.
func TestMarshal_NilSigner(t *testing.T) {
	if _, err := Marshal(nil, sampleFrame()); err == nil {
		t.Fatal("Marshal accepted a nil signer")
	}
}
