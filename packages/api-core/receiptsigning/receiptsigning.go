package receiptsigning

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	EnvelopeVersionV1              = "vrooli.receipt-signature.v1"
	AlgorithmHMACSHA256Development = "hmac-sha256-development"
	AlgorithmVaultTransit          = "vault-transit"

	PurposeExperimentAuditReceipt   Purpose = "experiment-audit-receipt-v1"
	PurposeExperimentHoldoutReceipt Purpose = "experiment-holdout-receipt-v1"
)

// Purpose domain-separates a signature so evidence of one kind cannot be
// replayed as another kind of receipt.
type Purpose string

func (p Purpose) Valid() bool {
	return p == PurposeExperimentAuditReceipt || p == PurposeExperimentHoldoutReceipt
}

// SignatureEnvelope is stored alongside evidence, never instead of evidence.
// KeyID is the Vault Transit key version or equivalent immutable verifier id.
type SignatureEnvelope struct {
	Version   string  `json:"version"`
	Purpose   Purpose `json:"purpose"`
	Algorithm string  `json:"algorithm"`
	KeyID     string  `json:"keyId"`
	Digest    string  `json:"digest"`
	Signature string  `json:"signature"`
}

func (e SignatureEnvelope) Validate() error {
	if e.Version != EnvelopeVersionV1 {
		return fmt.Errorf("unsupported receipt signature envelope version %q", e.Version)
	}
	if !e.Purpose.Valid() {
		return fmt.Errorf("unsupported receipt signing purpose %q", e.Purpose)
	}
	if strings.TrimSpace(e.Algorithm) == "" || strings.TrimSpace(e.KeyID) == "" || strings.TrimSpace(e.Signature) == "" {
		return fmt.Errorf("receipt signature envelope requires algorithm, key ID, and signature")
	}
	if _, err := ParseDigest(e.Digest); err != nil {
		return err
	}
	return nil
}

// Digest canonical bytes using the sole receipt digest encoding. Callers must
// construct canonical bytes before calling this function; this keeps the
// platform contract independent of any scenario's JSON model.
func Digest(canonical []byte) string {
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func ParseDigest(value string) ([]byte, error) {
	if !strings.HasPrefix(value, "sha256:") {
		return nil, fmt.Errorf("receipt digest must use sha256 prefix")
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	if err != nil || len(decoded) != sha256.Size {
		return nil, fmt.Errorf("receipt digest must contain a SHA-256 value")
	}
	return decoded, nil
}

// Health is intentionally operational rather than configuration-derived.
// Ready is true only when the provider can perform the required trusted work.
type Health struct {
	Ready       bool   `json:"ready"`
	Provider    string `json:"provider"`
	KeyID       string `json:"keyId,omitempty"`
	Production  bool   `json:"production"`
	RotationOK  bool   `json:"rotationOk"`
	Description string `json:"description,omitempty"`
}

// ReceiptSigner owns signature custody. Implementations must reject unsupported
// purposes and must never expose key material through this interface.
type ReceiptSigner interface {
	Sign(context.Context, Purpose, []byte) (SignatureEnvelope, error)
	Verify(context.Context, SignatureEnvelope, []byte) error
	Health(context.Context) (Health, error)
}

// RequireHealthy verifies a signer is truly ready for the requested trust tier.
func RequireHealthy(ctx context.Context, signer ReceiptSigner, production bool) (Health, error) {
	if signer == nil {
		return Health{}, fmt.Errorf("receipt signer is unavailable")
	}
	health, err := signer.Health(ctx)
	if err != nil {
		return Health{}, fmt.Errorf("receipt signer health: %w", err)
	}
	if !health.Ready || (production && !health.Production) {
		return Health{}, fmt.Errorf("receipt signer is not ready for %s use", map[bool]string{true: "production", false: "development"}[production])
	}
	return health, nil
}
