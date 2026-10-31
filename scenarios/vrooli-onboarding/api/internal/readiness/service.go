// Package readiness owns the transport free readiness result and service seam.
// The onboarding API injects the existing host, credential, and completion
// probes here; this package deliberately has no HTTP or Connect dependency.
package readiness

import (
	"context"
	"encoding/json"

	"github.com/vrooli/vrooli/internal/credentialspec"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/servicecall"
)

type Credential struct {
	Version              string                               `json:"version,omitempty"`
	Resource             string                               `json:"resource"`
	LogicalID            string                               `json:"logical_id"`
	Field                string                               `json:"field"`
	Owner                string                               `json:"owner,omitempty"`
	SourceRef            string                               `json:"source_ref,omitempty"`
	Kind                 string                               `json:"kind,omitempty"`
	Provider             string                               `json:"provider,omitempty"`
	AppliesWhen          *credentialspec.Applicability        `json:"applies_when,omitempty"`
	Tiers                []string                             `json:"tiers,omitempty"`
	RequirementGroup     string                               `json:"requirement_group,omitempty"`
	ConsumerRefs         []string                             `json:"consumer_refs,omitempty"`
	CompanionSettings    []string                             `json:"companion_settings,omitempty"`
	CompanionCredentials []string                             `json:"companion_credentials,omitempty"`
	AcquisitionRef       string                               `json:"acquisition_ref,omitempty"`
	VerificationRef      string                               `json:"verification_ref,omitempty"`
	RecoveryRef          string                               `json:"recovery_ref,omitempty"`
	HelpRef              string                               `json:"help_ref,omitempty"`
	EvidencePolicy       string                               `json:"evidence_policy,omitempty"`
	ProviderVersion      string                               `json:"provider_version,omitempty"`
	MigrationDiagnostics []credentialspec.MigrationDiagnostic `json:"migration_diagnostics,omitempty"`
	Provenance           []CredentialProvenance               `json:"provenance,omitempty"`
	Label                string                               `json:"label"`
	Description          string                               `json:"description,omitempty"`
	Placeholder          string                               `json:"placeholder,omitempty"`
	ObtainURL            string                               `json:"obtain_url,omitempty"`
	Provisioning         string                               `json:"provisioning,omitempty"`
	DerivedFrom          string                               `json:"derived_from,omitempty"`
	Required             bool                                 `json:"required"`
	Status               string                               `json:"status"`
	Detail               string                               `json:"detail,omitempty"`
	// EvidenceStatus is deliberately separate from Status. Status answers
	// whether the value is present in the configured store; evidence answers
	// whether the owning provider has actually exercised and verified it.
	// A stored value must never be presented as exercised readiness.
	EvidenceStatus string `json:"evidence_status,omitempty"`
	EvidenceDetail string `json:"evidence_detail,omitempty"`
	// EvidenceNextAction is a stable, renderer-neutral action identifier. It
	// lets UI, CLI, and release consumers agree on what closes an evidence
	// gap without guessing from prose.
	EvidenceNextAction string `json:"evidence_next_action,omitempty"`
	// EvidenceCredentialVersion is the opaque active authority version used to
	// bind owner evidence. It is never a credential value or a reversible hash.
	EvidenceCredentialVersion string `json:"evidence_credential_version,omitempty"`
	// ProviderState and ProviderDetail describe the credential backend itself,
	// independently from whether this address is configured. This distinction
	// prevents an unavailable store from being presented as an ordinary missing
	// input and gives every surface the same safe remediation context.
	ProviderState  string `json:"provider_state,omitempty"`
	ProviderDetail string `json:"provider_detail,omitempty"`
}

type CredentialProvenance struct {
	Version              string                         `json:"version,omitempty"`
	Owner                string                         `json:"owner"`
	SourceRef            string                         `json:"source_ref"`
	Kind                 string                         `json:"kind,omitempty"`
	Provider             string                         `json:"provider,omitempty"`
	AppliesWhen          *credentialspec.Applicability  `json:"applies_when,omitempty"`
	Tiers                []string                       `json:"tiers,omitempty"`
	RequirementGroup     string                         `json:"requirement_group,omitempty"`
	ConsumerRefs         []string                       `json:"consumer_refs,omitempty"`
	CompanionSettings    []string                       `json:"companion_settings,omitempty"`
	CompanionCredentials []string                       `json:"companion_credentials,omitempty"`
	AcquisitionRef       string                         `json:"acquisition_ref,omitempty"`
	VerificationRef      string                         `json:"verification_ref,omitempty"`
	RecoveryRef          string                         `json:"recovery_ref,omitempty"`
	HelpRef              string                         `json:"help_ref,omitempty"`
	EvidencePolicy       string                         `json:"evidence_policy,omitempty"`
	ProviderVersion      string                         `json:"provider_version,omitempty"`
	Env                  string                         `json:"env,omitempty"`
	Label                string                         `json:"label,omitempty"`
	Description          string                         `json:"description,omitempty"`
	Placeholder          string                         `json:"placeholder,omitempty"`
	ObtainURL            string                         `json:"obtain_url,omitempty"`
	Provisioning         string                         `json:"provisioning,omitempty"`
	DerivedFrom          string                         `json:"derived_from,omitempty"`
	Required             bool                           `json:"required"`
	Consumers            []CredentialConsumerProvenance `json:"consumers,omitempty"`
}

type CredentialConsumerProvenance struct {
	LogicalID      string   `json:"logical_id,omitempty"`
	AddressPattern string   `json:"address_pattern,omitempty"`
	Field          string   `json:"field,omitempty"`
	Kind           string   `json:"kind"`
	Consumer       string   `json:"consumer"`
	SourceRef      string   `json:"source_ref"`
	Required       bool     `json:"required"`
	Reason         string   `json:"reason,omitempty"`
	Tiers          []string `json:"tiers,omitempty"`
}

type Item struct {
	Name        string `json:"name"`
	Category    string `json:"category,omitempty"`
	Status      string `json:"status"`
	Detail      string `json:"detail,omitempty"`
	Remediation string `json:"remediation,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Host struct {
	Item
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}

type RecoveryGap struct {
	Address     string `json:"address"`
	Description string `json:"description,omitempty"`
}

type Recovery struct {
	ReceiptExists         bool            `json:"receipt_exists"`
	ExportedAt            string          `json:"exported_at,omitempty"`
	EntryCount            int             `json:"entry_count"`
	Uncovered             []string        `json:"uncovered"`
	RequiredAbsent        []string        `json:"required_absent"`
	RequiredAbsentDetails []RecoveryGap   `json:"required_absent_details"`
	RootCopy              json.RawMessage `json:"root_copy,omitempty"`
	RootCopyIssues        []string        `json:"root_copy_issues"`
	Status                string          `json:"status,omitempty"`
	AgeSeconds            int64           `json:"age_seconds,omitempty"`
	FreshnessReason       string          `json:"freshness_reason,omitempty"`
}

type CompletionBlocker struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	Remediation string `json:"remediation"`
}

type Response struct {
	Target                string              `json:"target"`
	ConfigurationRevision string              `json:"configuration_revision"`
	ExpiresAt             string              `json:"expires_at"`
	Status                string              `json:"status"`
	Scenarios             []string            `json:"scenarios"`
	Resources             []string            `json:"resources"`
	Credentials           []Credential        `json:"credentials"`
	Hosts                 []Host              `json:"hosts"`
	Integrations          []Item              `json:"integrations"`
	CheckedAt             string              `json:"checked_at"`
	CredentialDiagnosis   json.RawMessage     `json:"credential_diagnosis,omitempty"`
	Recovery              Recovery            `json:"recovery"`
	Blockers              []CompletionBlocker `json:"blockers"`
	Degraded              []CompletionBlocker `json:"degraded"`
	DegradedDigest        string              `json:"degraded_digest,omitempty"`
	DegradedAcknowledged  bool                `json:"degraded_acknowledged"`
	ManagedKeyConfigured  bool                `json:"managed_key_configured"`
	TrustAnchorMatch      bool                `json:"trust_anchor_match"`
}

// Service keeps readiness evaluation and mutation independent of the wire
// protocol. It is also the seam used by the Connect handler and completion
// checks, so both surfaces consume the same verdict.
type Service struct {
	Evaluate    func(context.Context, string) (Response, error)
	Acknowledge func(context.Context, string) (AcknowledgeResult, error)
}

type AcknowledgeResult struct {
	Status          string
	ReadinessDigest string
	Degraded        []CompletionBlocker
}

func (s Service) Get(ctx context.Context, target string) (Response, error) {
	return servicecall.Invoke(s.Evaluate != nil, func() (Response, error) { return s.Evaluate(ctx, target) }, Response{})
}

func (s Service) Accept(ctx context.Context, digest string) (AcknowledgeResult, error) {
	return servicecall.Invoke(s.Acknowledge != nil, func() (AcknowledgeResult, error) { return s.Acknowledge(ctx, digest) }, AcknowledgeResult{})
}
