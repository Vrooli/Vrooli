package receiptsigning

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// DevelopmentSigner is deterministic only to make local tests and development
// restarts ergonomic. Its receipts are deliberately non-production eligible.
// It must never be used for production promotion authority.
type DevelopmentSigner struct{}

func NewDevelopmentSigner() *DevelopmentSigner { return &DevelopmentSigner{} }

func (s *DevelopmentSigner) Sign(_ context.Context, purpose Purpose, canonical []byte) (SignatureEnvelope, error) {
	if !purpose.Valid() {
		return SignatureEnvelope{}, fmt.Errorf("development signer: unsupported purpose %q", purpose)
	}
	digest := Digest(canonical)
	// This is intentionally public and deterministic: it is an ergonomics
	// provider for non-production validation, not a trust root.
	mac := hmac.New(sha256.New, []byte("vrooli-development-receipt-signer-v1-not-a-secret"))
	_, _ = mac.Write([]byte(string(purpose)))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(digest))
	return SignatureEnvelope{Version: EnvelopeVersionV1, Purpose: purpose, Algorithm: AlgorithmHMACSHA256Development, KeyID: "development-only", Digest: digest, Signature: base64.RawStdEncoding.EncodeToString(mac.Sum(nil))}, nil
}

func (s *DevelopmentSigner) Verify(ctx context.Context, envelope SignatureEnvelope, canonical []byte) error {
	if err := envelope.Validate(); err != nil {
		return err
	}
	if envelope.Algorithm != AlgorithmHMACSHA256Development || envelope.KeyID != "development-only" {
		return fmt.Errorf("development signer: envelope is not development signed")
	}
	if envelope.Digest != Digest(canonical) {
		return fmt.Errorf("receipt digest does not match canonical content")
	}
	expected, err := s.Sign(ctx, envelope.Purpose, canonical)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expected.Signature), []byte(envelope.Signature)) {
		return fmt.Errorf("receipt signature is invalid")
	}
	return nil
}

func (s *DevelopmentSigner) Health(context.Context) (Health, error) {
	return Health{Ready: true, Provider: "development-hmac", KeyID: "development-only", Production: false, RotationOK: false, Description: "development-only signer; receipts are never production eligible"}, nil
}
