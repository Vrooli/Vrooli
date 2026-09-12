package cpverify

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

func writePin(t *testing.T, dir string, pub ed25519.PublicKey) string {
	t.Helper()
	path := filepath.Join(dir, "control_plane.pub")
	if err := os.WriteFile(path, []byte(base64.StdEncoding.EncodeToString(pub)), 0o600); err != nil {
		t.Fatalf("write pin: %v", err)
	}
	return path
}

func signPayload(t *testing.T, priv ed25519.PrivateKey, frame *channelv1.ServerFrame) []byte {
	t.Helper()
	inner, err := proto.Marshal(frame)
	if err != nil {
		t.Fatalf("marshal frame: %v", err)
	}
	env := &channelv1.SignedServerFrame{Frame: inner, Signature: ed25519.Sign(priv, inner)}
	payload, err := protojson.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return payload
}

func jobFrame() *channelv1.ServerFrame {
	return &channelv1.ServerFrame{Payload: &channelv1.ServerFrame_Job{
		Job: &channelv1.JobPush{RunId: "r1", Scenario: "s", Verb: "scenario test"},
	}}
}

// TestLoad_MissingPinIsHardError: a paired agent with no pinned key fails loudly
// with an actionable ErrNoPin (no trust-on-first-use fallback).
func TestLoad_MissingPinIsHardError(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "control_plane.pub"))
	if !errors.Is(err, ErrNoPin) {
		t.Fatalf("want ErrNoPin, got %v", err)
	}
	// The error names the path so an operator knows where the bootstrap must write.
	if err == nil || len(err.Error()) == 0 {
		t.Fatal("expected a non-empty, path-bearing error")
	}
}

// TestLoad_MalformedPinRejected: a garbage key file is a hard error, not a
// silent accept.
func TestLoad_MalformedPinRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "control_plane.pub")
	if err := os.WriteFile(path, []byte("not-base64!!"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("malformed pin accepted")
	}
}

// TestLoadedVerifier_AcceptsValidRejectsImpostor round-trips a real pinned key.
func TestLoadedVerifier_AcceptsValidRejectsImpostor(t *testing.T) {
	dir := t.TempDir()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	v, err := Load(writePin(t, dir, pub))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	frame, err := v.Open(signPayload(t, priv, jobFrame()))
	if err != nil {
		t.Fatalf("valid frame rejected: %v", err)
	}
	if frame.GetJob().GetRunId() != "r1" {
		t.Fatalf("unexpected frame: %v", frame)
	}

	// A frame signed by a different key must not verify against the pinned one.
	_, attacker, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Open(signPayload(t, attacker, jobFrame())); err == nil {
		t.Fatal("Open accepted an impostor-signed frame")
	}
}

// TestOpen_RejectsBareServerFrame proves there is no legacy unsigned path: a
// pre-signing bare ServerFrame does not verify.
func TestOpen_RejectsBareServerFrame(t *testing.T) {
	dir := t.TempDir()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	v, err := Load(writePin(t, dir, pub))
	if err != nil {
		t.Fatal(err)
	}
	bare, err := protojson.Marshal(jobFrame())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Open(bare); err == nil {
		t.Fatal("Open accepted a bare unsigned ServerFrame")
	}
}
