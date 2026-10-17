package domain

import (
	"fmt"
	"strings"
	"time"
)

// CredentialDescriptor is the exact credential-authority address a deployment
// binds. LogicalID and Field are preserved byte-for-byte from the closure;
// the deployment never normalises them, because two descriptors that
// normalise to the same target are a refusal, not a merge.
type CredentialDescriptor struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
}

// Address renders the descriptor in the `logical_id:field` form shared with
// packages/credentialclient-go and the closure document.
func (d CredentialDescriptor) Address() string {
	return d.LogicalID + ":" + d.Field
}

// IsZero reports whether either half of the address is missing.
func (d CredentialDescriptor) IsZero() bool {
	return strings.TrimSpace(d.LogicalID) == "" || strings.TrimSpace(d.Field) == ""
}

// ParseCredentialDescriptor splits a `logical_id:field` address on its first
// colon. Logical ids are slash-namespaced and never contain a colon.
func ParseCredentialDescriptor(address string) (CredentialDescriptor, error) {
	address = strings.TrimSpace(address)
	idx := strings.Index(address, ":")
	if idx <= 0 || idx == len(address)-1 {
		return CredentialDescriptor{}, fmt.Errorf("credential descriptor %q must be logical_id:field", address)
	}
	return CredentialDescriptor{LogicalID: address[:idx], Field: address[idx+1:]}, nil
}

// CredentialClass selects the rotation behaviour a binding needs. It is
// distinct from the manifest secret class (per_install_generated, user_prompt,
// remote_fetch, infrastructure), which says where the first value comes from.
type CredentialClass string

const (
	CredentialClassGeneratedDatabasePassword CredentialClass = "generated_database_password"
	CredentialClassExternalAPICredential     CredentialClass = "external_api_credential" // #nosec G101 -- closed class vocabulary, not credential material.
	CredentialClassSigningKey                CredentialClass = "signing_key"
	CredentialClassMachineEnrollment         CredentialClass = "machine_enrollment_credential"
	CredentialClassEncryptionRecoveryKey     CredentialClass = "encryption_recovery_key"
	CredentialClassSharedDependency          CredentialClass = "shared_dependency_credential"
)

// CredentialClasses is the closed vocabulary.
var CredentialClasses = []CredentialClass{
	CredentialClassGeneratedDatabasePassword,
	CredentialClassExternalAPICredential,
	CredentialClassSigningKey,
	CredentialClassMachineEnrollment,
	CredentialClassEncryptionRecoveryKey,
	CredentialClassSharedDependency,
}

// Manifest secret classes (domain.BundleSecretPlan.Class).
const (
	SecretClassPerInstallGenerated = "per_install_generated"
	SecretClassUserPrompt          = "user_prompt"
	SecretClassRemoteFetch         = "remote_fetch"
	SecretClassInfrastructure      = "infrastructure"
)

// CredentialVersion identifies one materialised value without carrying it.
// Number is monotonic per binding; ContentRef is an opaque random token
// minted with the version. It is never derived from the value, so no retained
// surface can be used to confirm or brute-force a value.
type CredentialVersion struct {
	Number     int64     `json:"number"`
	ContentRef string    `json:"content_ref"`
	CreatedAt  time.Time `json:"created_at"`
	// ExpiresAt is provider- or target-supplied metadata. A missing value means
	// the owner has not declared an expiry policy; it must not be fabricated by
	// the cloud owner.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Expired reports whether the version is no longer valid at now.
func (v CredentialVersion) Expired(now time.Time) bool {
	return v.ExpiresAt != nil && !now.Before(v.ExpiresAt.UTC())
}

// RenewalDue reports whether the version is inside the owner's renewal window.
func (v CredentialVersion) RenewalDue(now time.Time, window time.Duration) bool {
	return v.ExpiresAt != nil && !v.Expired(now) && window > 0 && !now.Before(v.ExpiresAt.UTC().Add(-window))
}

// CredentialBindingState is the binding's standing on its deployment.
type CredentialBindingState string

const (
	CredentialBindingPlanned      CredentialBindingState = "planned"
	CredentialBindingMaterialized CredentialBindingState = "materialized"
	CredentialBindingRevoked      CredentialBindingState = "revoked"
)

// CredentialBinding is one descriptor bound to one deployment. It references
// versions and consumers; it never holds a value.
type CredentialBinding struct {
	ID           string               `json:"id"`
	DeploymentID string               `json:"deployment_id"`
	Descriptor   CredentialDescriptor `json:"descriptor"`
	Class        CredentialClass      `json:"class"`
	// SourceClass is the manifest secret class the first value came from.
	SourceClass string `json:"source_class"`
	// Target is the declared injection target (env name or file path). It is
	// what the collision check protects.
	Target BundleSecretTarget `json:"target"`
	// Version is the current active version; zero Number means nothing has
	// been materialised yet.
	Version CredentialVersion `json:"version"`
	// PreviousVersion is retained until its revocation completes.
	PreviousVersion *CredentialVersion `json:"previous_version,omitempty"`
	// ConsumerRefs are scenario/resource ids from the closure that read this
	// credential. A shared credential completes rotation only when each has
	// acknowledged the version.
	ConsumerRefs []string `json:"consumer_refs"`
	// GrantRef is the Bridge credential grant id when the target is reached
	// over the bridge transport.
	GrantRef string `json:"grant_ref,omitempty"`
	// RecoveryKeyRef is set on encryption/recovery-key bindings: the
	// `logical_id:field` reference the backup domain resolves the key by
	// (recoverypoint key_ref). It names the key; it never carries it.
	RecoveryKeyRef string                 `json:"recovery_key_ref,omitempty"`
	State          CredentialBindingState `json:"state"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// CredentialAck records that a consumer verified use of a version.
type CredentialAck struct {
	BindingID  string    `json:"binding_id"`
	Consumer   string    `json:"consumer"`
	Version    int64     `json:"version"`
	VerifiedAt time.Time `json:"verified_at"`
}

// CredentialVersionRef is the reference other owners (recovery points, break
// glass receipts) carry to a credential version. Reference only, never a value.
type CredentialVersionRef struct {
	BindingID string `json:"binding_id"`
	Version   int64  `json:"version"`
}

// CredentialOperationKind distinguishes the lifecycle operations that share
// the rotation ledger.
type CredentialOperationKind string

const (
	CredentialOperationRotate     CredentialOperationKind = "rotate"
	CredentialOperationRevoke     CredentialOperationKind = "revoke"
	CredentialOperationRecover    CredentialOperationKind = "recover"
	CredentialOperationBreakGlass CredentialOperationKind = "break_glass"
)

// CredentialRotationState is the lifecycle state machine from design section I.
type CredentialRotationState string

const (
	RotationPlanned              CredentialRotationState = "planned"
	RotationNewVersionCreated    CredentialRotationState = "new_version_created"
	RotationProviderPrepared     CredentialRotationState = "provider_prepared"
	RotationConsumersUpdated     CredentialRotationState = "consumers_updated"
	RotationVerified             CredentialRotationState = "verified"
	RotationPendingOperatorInput CredentialRotationState = "pending_operator_input"
	RotationOldVersionRevoked    CredentialRotationState = "old_version_revoked"
	RotationComplete             CredentialRotationState = "complete"
	RotationRecovering           CredentialRotationState = "recovering"
	RotationFailed               CredentialRotationState = "failed"
	// RotationRevocationIncomplete is the truthful state of a revoke whose
	// target could not be reached; the unreached list is retained.
	RotationRevocationIncomplete CredentialRotationState = "revocation_incomplete"
)

// Terminal reports whether no further transition is possible.
func (s CredentialRotationState) Terminal() bool {
	return s == RotationComplete || s == RotationFailed
}

// CredentialConsumerState is a consumer's standing within one operation.
type CredentialConsumerState string

const (
	ConsumerPending      CredentialConsumerState = "pending"
	ConsumerUpdated      CredentialConsumerState = "updated"
	ConsumerAcknowledged CredentialConsumerState = "acknowledged"
	ConsumerUnreachable  CredentialConsumerState = "unreachable"
	ConsumerFailed       CredentialConsumerState = "failed"
)

// CredentialConsumerProgress is one consumer's acknowledgement standing.
type CredentialConsumerProgress struct {
	Consumer  string                  `json:"consumer"`
	State     CredentialConsumerState `json:"state"`
	Version   int64                   `json:"version"`
	Reason    string                  `json:"reason,omitempty"`
	UpdatedAt time.Time               `json:"updated_at"`
}

// CredentialReceipt is one auditable step inside an operation. Details are
// metadata only.
type CredentialReceipt struct {
	Step             string         `json:"step"`
	State            string         `json:"state"`
	Outcome          string         `json:"outcome"`
	At               time.Time      `json:"at"`
	TargetReceiptRef string         `json:"target_receipt_ref,omitempty"`
	Details          map[string]any `json:"details,omitempty"`
	Limitations      []string       `json:"limitations,omitempty"`
}

// OperatorHandoff is the durable reference for a resumable operator-input
// step: the operation cannot finish automatically and says exactly what the
// operator must do and how to resume.
type OperatorHandoff struct {
	Reference   string    `json:"reference"`
	Provider    string    `json:"provider"`
	Instruction string    `json:"instruction"`
	ResumeWith  string    `json:"resume_with"`
	RequestedAt time.Time `json:"requested_at"`
}

// CredentialOperationError is the persisted failure of an operation.
type CredentialOperationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BreakGlassWindow bounds an emergency access grant.
type BreakGlassWindow struct {
	Scope           string     `json:"scope"`
	Operator        string     `json:"operator"`
	IssuedAt        time.Time  `json:"issued_at"`
	ExpiresAt       time.Time  `json:"expires_at"`
	PredecessorRef  string     `json:"predecessor_ref"`
	AutoRevokedAt   *time.Time `json:"auto_revoked_at,omitempty"`
	ConfirmationRef string     `json:"confirmation_ref"`
}

// CredentialRotation is one durable lifecycle operation over one binding.
type CredentialRotation struct {
	ID           string                       `json:"id"`
	DeploymentID string                       `json:"deployment_id"`
	BindingID    string                       `json:"binding_id"`
	Kind         CredentialOperationKind      `json:"kind"`
	FromVersion  int64                        `json:"from_version"`
	ToVersion    int64                        `json:"to_version"`
	State        CredentialRotationState      `json:"state"`
	Consumers    []CredentialConsumerProgress `json:"consumers"`
	// Unreached lists the nodes/consumers an incomplete operation could not
	// confirm; retained until each is confirmed.
	Unreached            []string                  `json:"unreached,omitempty"`
	PendingOperatorInput *OperatorHandoff          `json:"pending_operator_input,omitempty"`
	ResumeAfter          *time.Time                `json:"resume_after,omitempty"`
	BreakGlass           *BreakGlassWindow         `json:"break_glass,omitempty"`
	Receipts             []CredentialReceipt       `json:"receipts"`
	Error                *CredentialOperationError `json:"error,omitempty"`
	CreatedAt            time.Time                 `json:"created_at"`
	UpdatedAt            time.Time                 `json:"updated_at"`
	CompletedAt          *time.Time                `json:"completed_at,omitempty"`
}

// Incomplete reports whether the operation stopped short of its terminal
// success state for a reason the operator must act on.
func (r *CredentialRotation) Incomplete() bool {
	return r != nil && r.State != RotationComplete && r.State != RotationFailed
}
