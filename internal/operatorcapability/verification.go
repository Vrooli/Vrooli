package operatorcapability

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"
)

const EvidenceSchemaVersion = "1.0.0"

// Verification is deliberately bounded at the registry boundary. Providers
// may still apply their own lower limit, but one onboarding request cannot
// fan out into an unbounded number of simultaneous external probes.
const maxConcurrentVerifications = 4

const (
	verificationDefaultAttempts = 3
	verificationMaxAttempts     = 3
	verificationInitialBackoff  = 100 * time.Millisecond
	verificationMaxBackoff      = 2 * time.Second
)

// VerificationError is the provider-neutral failure envelope. Providers may
// classify a failure without exposing a secret or forcing consumers to infer
// whether an unknown result is safe to retry.
type VerificationError struct {
	Code       string
	Retryable  bool
	RetryAfter time.Duration
	NextAction string
	Cause      error
}

// VerificationReport is the machine-readable result emitted by the control
// plane for a verification attempt. Successful attempts may be represented by
// the legacy evidence array, but failures use this envelope so every client
// can preserve retry and remediation semantics without parsing error text.
type VerificationReport struct {
	Evidence          []EvidenceReference `json:"evidence,omitempty"`
	ErrorCode         string              `json:"error_code,omitempty"`
	Retryable         bool                `json:"retryable,omitempty"`
	RetryAfterSeconds int64               `json:"retry_after_seconds,omitempty"`
	NextAction        string              `json:"next_action,omitempty"`
}

func (e *VerificationError) Error() string {
	if e == nil {
		return "verification failed"
	}
	message := "verification failed"
	if strings.TrimSpace(e.Code) != "" {
		message += " (" + e.Code + ")"
	}
	if e.NextAction != "" {
		message += "; next_action=" + e.NextAction
	}
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *VerificationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func classifyVerificationError(err error) error {
	if err == nil {
		return nil
	}
	var classified *VerificationError
	if errors.As(err, &classified) {
		return classified
	}
	code := "provider_verification_failed"
	retryable := true
	nextAction := "retry-capability-verification"
	if errors.Is(err, context.DeadlineExceeded) {
		code = "verification_timeout"
	}
	return &VerificationError{Code: code, Retryable: retryable, NextAction: nextAction, Cause: err}
}

type VerificationStage string

const (
	VerificationStorage        VerificationStage = "storage"
	VerificationAuthentication VerificationStage = "authentication"
	VerificationAuthorization  VerificationStage = "authorization"
	VerificationOperation      VerificationStage = "operation"
	VerificationRecovery       VerificationStage = "recovery"
)

func validVerificationStage(stage VerificationStage) bool {
	switch stage {
	case VerificationStorage, VerificationAuthentication, VerificationAuthorization, VerificationOperation, VerificationRecovery:
		return true
	default:
		return false
	}
}

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
	CapabilityID           string                 `json:"capability_id"`
	CredentialRef          *CredentialEvidenceRef `json:"credential_ref,omitempty"`
	TargetID               string                 `json:"target_id"`
	Environment            string                 `json:"environment,omitempty"`
	AccountIdentity        string                 `json:"account_identity,omitempty"`
	Operation              string                 `json:"operation"`
	Context                map[string]string      `json:"context,omitempty"`
	ContextDigest          string                 `json:"context_digest,omitempty"`
	CatalogRevision        string                 `json:"catalog_revision,omitempty"`
	ConfigurationRevision  string                 `json:"configuration_revision,omitempty"`
	ProviderAdapterVersion string                 `json:"provider_adapter_version,omitempty"`
	Effect                 EffectBudget           `json:"effect"`
	Timeout                time.Duration          `json:"-"`
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
	if r.ContextDigest != "" && !validDigest(r.ContextDigest) {
		return errors.New("context_digest must be a lowercase SHA-256 digest")
	}
	for field, value := range map[string]string{
		"catalog_revision":         valueOrEmpty(r.CatalogRevision),
		"configuration_revision":   valueOrEmpty(r.ConfigurationRevision),
		"provider_adapter_version": valueOrEmpty(r.ProviderAdapterVersion),
	} {
		if value != "" && (len(value) > maxDescriptorTextBytes || strings.ContainsAny(value, "\r\n\x00")) {
			return fmt.Errorf("%s is invalid", field)
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

func valueOrEmpty(value string) string { return strings.TrimSpace(value) }

// VerificationContextDigest creates a stable, non-secret binding for the
// value-free context supplied to an owner adapter. It is intentionally empty
// when there is no context, so legacy providers remain compatible for the
// context-free verification path.
func VerificationContextDigest(values map[string]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	hash := sha256.New()
	_, _ = hash.Write([]byte("vrooli-verification-context/v1\n"))
	for _, key := range keys {
		_, _ = fmt.Fprintf(hash, "%d:%s=%d:%s\n", len(key), key, len(values[key]), values[key])
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && strings.ToLower(value) == value
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
	contextDigest := VerificationContextDigest(request.Context)
	if request.ContextDigest != "" && request.ContextDigest != contextDigest {
		return nil, errors.New("context_digest does not match the verification context")
	}
	request.ContextDigest = contextDigest
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
	if err := r.acquireVerificationSlot(verifyCtx); err != nil {
		return nil, fmt.Errorf("queue capability verification %q: %w", descriptor.ID, err)
	}
	defer r.releaseVerificationSlot()
	receipts, err := verifier.Verify(verifyCtx, request)
	if err != nil {
		return nil, fmt.Errorf("verify capability %q: %w", descriptor.ID, classifyVerificationError(err))
	}
	if len(receipts) == 0 || len(receipts) > 32 {
		return nil, errors.New("verification returned an unsupported number of evidence receipts")
	}
	for index := range receipts {
		receipt := &receipts[index]
		if receipt.CapabilityID == "" {
			receipt.CapabilityID = request.CapabilityID
		}
		if receipt.Owner == "" {
			receipt.Owner = descriptor.Owner
		}
		if receipt.Owner != descriptor.Owner {
			return nil, errors.New("verification evidence is owned by a different provider")
		}
		if receipt.CapabilityID != request.CapabilityID || receipt.TargetID != request.TargetID || receipt.Operation != request.Operation || receipt.Environment != request.Environment || receipt.AccountIdentity != request.AccountIdentity {
			return nil, errors.New("verification evidence is bound to a different capability, target, or operation")
		}
		if request.ContextDigest != "" && receipt.ContextDigest != request.ContextDigest {
			return nil, errors.New("verification evidence is not bound to the requested context")
		}
		if request.CatalogRevision != "" && receipt.CatalogRevision != request.CatalogRevision {
			return nil, errors.New("verification evidence is not bound to the requested catalog revision")
		}
		if request.ConfigurationRevision != "" && receipt.ConfigurationRevision != request.ConfigurationRevision {
			return nil, errors.New("verification evidence is not bound to the requested configuration revision")
		}
		if request.ProviderAdapterVersion != "" && receipt.ProviderAdapterVersion != request.ProviderAdapterVersion {
			return nil, errors.New("verification evidence is not bound to the requested provider adapter version")
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
		if len(descriptor.Evidence.Stages) > 0 {
			if !validVerificationStage(receipt.Stage) {
				return nil, fmt.Errorf("verification evidence for capability %q is missing a valid stage", descriptor.ID)
			}
			stageAccepted := false
			for _, stage := range descriptor.Evidence.Stages {
				if receipt.Stage == stage {
					stageAccepted = true
					break
				}
			}
			if !stageAccepted {
				return nil, fmt.Errorf("verification stage %q is not accepted by capability %q", receipt.Stage, descriptor.ID)
			}
		}
		if err := descriptor.Evidence.ValidateReference(*receipt); err != nil {
			return nil, err
		}
		if strings.TrimSpace(descriptor.Evidence.Freshness) != "" && !receipt.FreshAt(time.Now().UTC()) {
			return nil, fmt.Errorf("verification evidence for capability %q is stale or missing an expiry", descriptor.ID)
		}
	}
	return receipts, nil
}

// VerifyWithRetry performs a bounded retry loop for read-only verification.
// Bounded-write verification is intentionally single-attempt: a lost response
// must be reconciled by the owner rather than replayed by a generic caller.
func (r *Registry) VerifyWithRetry(ctx context.Context, request VerificationRequest) ([]EvidenceReference, error) {
	attempts := verificationDefaultAttempts
	if request.Effect.Class == EffectBoundedWrite {
		attempts = 1
	}
	if request.Timeout == 0 {
		request.Timeout = 30 * time.Second
	}
	overallCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	for attempt := 1; attempt <= attempts; attempt++ {
		receipts, err := r.Verify(overallCtx, request)
		if err == nil {
			return receipts, nil
		}
		var verificationErr *VerificationError
		if !errors.As(err, &verificationErr) || !verificationErr.Retryable || attempt == attempts {
			return nil, err
		}
		delay := verificationErr.RetryAfter
		if delay <= 0 {
			delay = verificationInitialBackoff << (attempt - 1)
		}
		if delay > verificationMaxBackoff {
			delay = verificationMaxBackoff
		}
		timer := time.NewTimer(delay)
		select {
		case <-overallCtx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return nil, fmt.Errorf("verification retry budget exhausted: %w", overallCtx.Err())
		case <-timer.C:
		}
	}
	return nil, errors.New("verification retry budget exhausted")
}

func (r *Registry) acquireVerificationSlot(ctx context.Context) error {
	if r == nil || r.verificationSlots == nil {
		return nil
	}
	select {
	case r.verificationSlots <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Registry) releaseVerificationSlot() {
	if r == nil || r.verificationSlots == nil {
		return
	}
	<-r.verificationSlots
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
	case "stage":
		return validVerificationStage(receipt.Stage)
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
	case "owner":
		return receipt.Owner != ""
	case "context_digest":
		return validDigest(receipt.ContextDigest)
	case "catalog_revision":
		return receipt.CatalogRevision != ""
	case "configuration_revision":
		return receipt.ConfigurationRevision != ""
	case "provider_adapter_version":
		return receipt.ProviderAdapterVersion != ""
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
