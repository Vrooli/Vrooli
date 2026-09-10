package operatorcapability

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

const EvidenceSchemaVersion = "1.0.0"

type VerificationStage string

const (
	VerificationStorage        VerificationStage = "storage"
	VerificationAuthentication VerificationStage = "authentication"
	VerificationAuthorization  VerificationStage = "authorization"
	VerificationOperation      VerificationStage = "operation"
	VerificationRecovery       VerificationStage = "recovery"
)

// FreshAt reports whether a receipt is still usable at the supplied
// observation time. It deliberately does not use dashboard refresh time.
func (e EvidenceReference) FreshAt(now time.Time) bool {
	if e.ObservedAt.IsZero() || e.ExpiresAt.IsZero() {
		return false
	}
	return !now.Before(e.ObservedAt) && now.Before(e.ExpiresAt) && e.Verified
}

type EffectClass string

const (
	EffectReadOnly     EffectClass = "read_only"
	EffectBoundedWrite EffectClass = "bounded_write"
)

type EffectBudget struct {
	Class         EffectClass `json:"class"`
	MaxOperations int         `json:"max_operations"`
	CleanupPolicy string      `json:"cleanup_policy,omitempty"`
}

type VerificationRequest struct {
	CapabilityID    string                 `json:"capability_id"`
	CredentialRef   *CredentialEvidenceRef `json:"credential_ref,omitempty"`
	TargetID        string                 `json:"target_id"`
	Environment     string                 `json:"environment,omitempty"`
	AccountIdentity string                 `json:"account_identity,omitempty"`
	Operation       string                 `json:"operation"`
	Context         map[string]string      `json:"context,omitempty"`
	Effect          EffectBudget           `json:"effect"`
	Timeout         time.Duration          `json:"-"`
}

func (r VerificationRequest) Validate() error {
	for field, value := range map[string]string{
		"capability_id": r.CapabilityID, "target_id": r.TargetID, "operation": r.Operation,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
		if len(value) > maxActionTextBytes || strings.IndexByte(value, 0) >= 0 {
			return fmt.Errorf("%s is invalid", field)
		}
	}
	if r.CredentialRef != nil {
		if strings.TrimSpace(r.CredentialRef.LogicalID) == "" || strings.TrimSpace(r.CredentialRef.Field) == "" || strings.TrimSpace(r.CredentialRef.Version) == "" {
			return errors.New("credential_ref requires logical_id, field, and opaque version")
		}
	}
	if len(r.Context) > maxMetadataEntries {
		return errors.New("verification context has too many entries")
	}
	for key, value := range r.Context {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n") || strings.ContainsAny(value, "\x00\r\n") {
			return errors.New("verification context contains invalid metadata")
		}
		if sensitiveMetadataKey(key) {
			return fmt.Errorf("verification context cannot carry secret metadata %q", key)
		}
		if len(key) > maxDescriptorTextBytes || len(value) > maxDescriptorTextBytes {
			return errors.New("verification context metadata is too large")
		}
	}
	if r.Effect.Class != EffectReadOnly && r.Effect.Class != EffectBoundedWrite {
		return errors.New("verification effect class must be read_only or bounded_write")
	}
	if r.Effect.MaxOperations < 0 || r.Effect.MaxOperations > 16 {
		return errors.New("verification effect budget is outside the supported bound")
	}
	if r.Effect.Class == EffectBoundedWrite && r.Effect.MaxOperations == 0 {
		return errors.New("bounded_write verification requires a positive effect budget")
	}
	if r.Timeout < 0 || r.Timeout > 15*time.Minute {
		return errors.New("verification timeout is outside the supported bound")
	}
	return nil
}

func sensitiveMetadataKey(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, term := range []string{"secret", "password", "passphrase", "token", "private_key", "credential_value"} {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

// VerificationProvider is an optional owner adapter. It receives only typed,
// value-free context; the owner resolves protected material through its own
// authority boundary and returns redacted evidence.
type VerificationProvider interface {
	Verify(context.Context, VerificationRequest) ([]EvidenceReference, error)
}

// Verify executes one registered owner verification adapter and validates its
// receipt against the request and the capability's evidence contract.
func (r *Registry) Verify(ctx context.Context, request VerificationRequest) ([]EvidenceReference, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	provider, ok := r.Provider(request.CapabilityID)
	if !ok {
		return nil, fmt.Errorf("capability %q is not registered", request.CapabilityID)
	}
	descriptor := provider.Descriptor()
	if !descriptor.Lifecycle.Verify {
		return nil, fmt.Errorf("capability %q does not support verification", descriptor.ID)
	}
	verifier, ok := provider.(VerificationProvider)
	if !ok {
		return nil, fmt.Errorf("capability %q has no owner verification adapter", descriptor.ID)
	}
	if request.Timeout == 0 {
		request.Timeout = 30 * time.Second
	}
	verifyCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	receipts, err := verifier.Verify(verifyCtx, request)
	if err != nil {
		return nil, fmt.Errorf("verify capability %q: %w", descriptor.ID, err)
	}
	if len(receipts) == 0 || len(receipts) > 32 {
		return nil, errors.New("verification returned an unsupported number of evidence receipts")
	}
	for index := range receipts {
		receipt := &receipts[index]
		if receipt.CapabilityID == "" {
			receipt.CapabilityID = request.CapabilityID
		}
		if receipt.CapabilityID != request.CapabilityID || receipt.TargetID != request.TargetID || receipt.Operation != request.Operation {
			return nil, errors.New("verification evidence is bound to a different capability, target, or operation")
		}
		if request.CredentialRef != nil {
			if receipt.CredentialRef == nil || receipt.CredentialRef.LogicalID != request.CredentialRef.LogicalID || receipt.CredentialRef.Field != request.CredentialRef.Field || receipt.CredentialRef.Version != request.CredentialRef.Version {
				return nil, errors.New("verification evidence is not bound to the requested credential version")
			}
		}
		if receipt.EffectClass == "" {
			receipt.EffectClass = string(request.Effect.Class)
		}
		if receipt.EffectsUsed > request.Effect.MaxOperations {
			return nil, errors.New("verification exceeded its declared effect budget")
		}
		if request.Effect.Class == EffectReadOnly && receipt.EffectsUsed != 0 {
			return nil, errors.New("read_only verification reported an external effect")
		}
		if request.Effect.Class == EffectBoundedWrite && request.Effect.CleanupPolicy != "" && !receipt.CleanupCompleted {
			return nil, errors.New("bounded verification did not complete its cleanup policy")
		}
		if err := receipt.Validate(); err != nil {
			return nil, err
		}
		if err := descriptor.Evidence.ValidateReference(*receipt); err != nil {
			return nil, err
		}
	}
	return receipts, nil
}

func (c EvidenceContract) ValidateReference(receipt EvidenceReference) error {
	if len(c.Kinds) > 0 {
		matched := false
		for _, kind := range c.Kinds {
			if kind == receipt.Kind {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("evidence kind %q is not accepted by the capability", receipt.Kind)
		}
	}
	for _, field := range c.RequiredFields {
		if !evidenceFieldPresent(receipt, field) {
			return fmt.Errorf("evidence is missing required field %q", field)
		}
	}
	if c.SecretFree && (sensitiveMetadataKey(receipt.ArtifactIdentity) || sensitiveMetadataKey(receipt.NextAction)) {
		return errors.New("secret-free evidence contains sensitive metadata")
	}
	return nil
}

func evidenceFieldPresent(receipt EvidenceReference, field string) bool {
	switch field {
	case "capability_id":
		return receipt.CapabilityID != ""
	case "credential_ref":
		return receipt.CredentialRef != nil
	case "target_id":
		return receipt.TargetID != ""
	case "environment":
		return receipt.Environment != ""
	case "account_identity":
		return receipt.AccountIdentity != ""
	case "operation":
		return receipt.Operation != ""
	case "status":
		return receipt.Status != ""
	case "kind":
		return receipt.Kind != ""
	case "artifact_identity":
		return receipt.ArtifactIdentity != ""
	case "checksum":
		return receipt.Checksum != ""
	case "source_generation":
		return receipt.SourceGeneration != ""
	case "coverage":
		return len(receipt.Coverage) > 0
	case "verified":
		return receipt.Verified
	case "observed_at":
		return !receipt.ObservedAt.IsZero()
	case "expires_at":
		return !receipt.ExpiresAt.IsZero()
	case "artifact_refs":
		return len(receipt.ArtifactRefs) > 0
	default:
		return false
	}
}

// ValidateProbeURL permits only an explicit external HTTPS endpoint. Owner
// adapters must call this before making a provider probe so a manifest cannot
// turn verification into an internal-network or credential-forwarding proxy.
func ValidateProbeURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("verification endpoint must be an HTTPS URL without credentials or fragments")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".localhost") {
		return errors.New("verification endpoint cannot target a local hostname")
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
		return errors.New("verification endpoint cannot target a local or private address")
	}
	return nil
}
