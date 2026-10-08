package credentialauthoritysigning

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/api-core/receiptsigning"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type memoryStore struct{ values map[string]string }

func (s *memoryStore) Require(identity credentialauthority.Identity, field string) (string, error) {
	value, ok := s.values[string(identity)+"/"+field]
	if !ok {
		return "", credentialauthority.ErrUnconfigured
	}
	return value, nil
}

func (s *memoryStore) Put(identity credentialauthority.Identity, field, value string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[string(identity)+"/"+field] = value
	return nil
}

func TestSignerRotatesWithoutBreakingHistoricalVerification(t *testing.T) {
	identity := credentialauthority.Identity("vrooli/prompt-manager/experiment-receipts")
	store := &memoryStore{}
	signer, err := New(Config{Identity: identity, Field: "key-ring", Store: store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Health(context.Background()); err == nil {
		t.Fatal("Health() reported an unconfigured signer as ready")
	} else if !errors.Is(err, credentialauthority.ErrUnconfigured) {
		t.Fatalf("Health() error = %v, want unconfigured error", err)
	}
	if _, err := signer.Rotate(context.Background()); err != nil {
		t.Fatalf("initial Rotate() error = %v", err)
	}

	canonical := []byte(`{"receipt":"audit"}`)
	first, err := signer.Sign(context.Background(), receiptsigning.PurposeExperimentAuditReceipt, canonical)
	if err != nil {
		t.Fatalf("first Sign() error = %v", err)
	}
	if first.Algorithm != receiptsigning.AlgorithmCredentialAuthorityEd25519 || first.KeyID != string(identity)+":v1" {
		t.Fatalf("first envelope = %#v", first)
	}
	if strings.Contains(first.Signature, "privateKey") || strings.Contains(first.Signature, "key-ring") {
		t.Fatal("signature envelope leaked authority key-ring metadata")
	}
	if err := signer.Verify(context.Background(), first, canonical); err != nil {
		t.Fatalf("first Verify() error = %v", err)
	}
	if _, err := signer.Rotate(context.Background()); err != nil {
		t.Fatalf("second Rotate() error = %v", err)
	}
	second, err := signer.Sign(context.Background(), receiptsigning.PurposeExperimentHoldoutReceipt, canonical)
	if err != nil {
		t.Fatalf("second Sign() error = %v", err)
	}
	if second.KeyID != string(identity)+":v2" {
		t.Fatalf("second key ID = %q, want v2", second.KeyID)
	}
	if err := signer.Verify(context.Background(), first, canonical); err != nil {
		t.Fatalf("historical Verify() error = %v", err)
	}
	if err := signer.Verify(context.Background(), second, canonical); err != nil {
		t.Fatalf("rotated Verify() error = %v", err)
	}
	if err := signer.Verify(context.Background(), second, []byte("tampered")); err == nil {
		t.Fatal("Verify() accepted tampered canonical bytes")
	}
	second.Purpose = receiptsigning.PurposeExperimentAuditReceipt
	if err := signer.Verify(context.Background(), second, canonical); err == nil {
		t.Fatal("Verify() accepted a purpose replay")
	}
	health, err := signer.Health(context.Background())
	if err != nil || !health.Ready || !health.Production || !health.RotationOK || health.KeyID != string(identity)+":v2" {
		t.Fatalf("Health() = %#v, %v", health, err)
	}
}

func TestSignerRejectsMalformedAuthorityKeyRing(t *testing.T) {
	identity := credentialauthority.Identity("vrooli/prompt-manager/experiment-receipts")
	store := &memoryStore{values: map[string]string{
		string(identity) + "/key-ring": `{"version":1,"active":"v1","keys":[{"id":"v1","privateKey":"bad","publicKey":"bad"}]}`,
	}}
	signer, err := New(Config{Identity: identity, Store: store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signer.Health(context.Background()); err == nil {
		t.Fatal("Health() accepted malformed key ring")
	}
}

func TestCompatibleSignerUsesLegacyVerifierOnlyForHistoricalAlgorithm(t *testing.T) {
	primary := receiptsigning.NewDevelopmentSigner()
	legacy := &recordingVerifier{}
	compatible := CompatibleSigner{Primary: primary, Legacy: legacy}
	canonical := []byte("canonical")
	newEnvelope, err := compatible.Sign(context.Background(), receiptsigning.PurposeExperimentAuditReceipt, canonical)
	if err != nil {
		t.Fatal(err)
	}
	if err := compatible.Verify(context.Background(), newEnvelope, canonical); err != nil {
		t.Fatalf("new Verify() error = %v", err)
	}
	legacyEnvelope := receiptsigning.SignatureEnvelope{Version: receiptsigning.EnvelopeVersionV1, Purpose: receiptsigning.PurposeExperimentAuditReceipt, Algorithm: receiptsigning.AlgorithmVaultTransit, KeyID: "legacy:v1", Digest: receiptsigning.Digest(canonical), Signature: "legacy"}
	if err := compatible.Verify(context.Background(), legacyEnvelope, canonical); err != nil {
		t.Fatalf("legacy Verify() error = %v", err)
	}
	if !legacy.called {
		t.Fatal("historical envelope did not route to the legacy verifier")
	}
}

type recordingVerifier struct{ called bool }

func (v *recordingVerifier) Sign(context.Context, receiptsigning.Purpose, []byte) (receiptsigning.SignatureEnvelope, error) {
	return receiptsigning.SignatureEnvelope{}, errors.New("not a signer")
}

func (v *recordingVerifier) Verify(context.Context, receiptsigning.SignatureEnvelope, []byte) error {
	v.called = true
	return nil
}

func (v *recordingVerifier) Health(context.Context) (receiptsigning.Health, error) {
	return receiptsigning.Health{Ready: true}, nil
}
