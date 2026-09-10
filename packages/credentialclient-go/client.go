// Package credentialclient is the typed client boundary for scenario
// consumers. Use it when a scenario needs the authoritative inventory,
// subscriptions, or remote-node transport; control-plane-adjacent Go code
// should use packages/credential-authority-go for direct in-process access.
// Neither binding exposes a credential value in metadata responses.
package credentialclient

import (
	"context"
	"time"

	"github.com/vrooli/vrooli/internal/credentialspec"
)

type CredentialRef struct {
	Version   string `json:"version,omitempty"`
	Resource  string `json:"resource"`
	Env       string `json:"env"`
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	// Owner and SourceRef identify the declaration owner and its canonical
	// source location. They are metadata only; neither field can reveal a
	// credential value.
	Owner                string                               `json:"owner,omitempty"`
	SourceRef            string                               `json:"source_ref,omitempty"`
	Kind                 string                               `json:"kind,omitempty"`
	Provider             string                               `json:"provider,omitempty"`
	AppliesWhen          *credentialspec.Applicability        `json:"applies_when,omitempty"`
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
	// Provenance retains every declaration that contributed this shared
	// authority address. Resource and scenario manifests may describe the same
	// address from different process boundaries; collapsing them must not erase
	// the owner, source, or consumer-specific requirement.
	Provenance []CredentialProvenance `json:"provenance,omitempty"`
	Label      string                 `json:"label,omitempty"`
	// Description is the operator-facing purpose of the credential. It is
	// carried separately from Label because a credential card must show both
	// the name and the reason the value is being asked for.
	Description  string `json:"description,omitempty"`
	ObtainURL    string `json:"obtain_url,omitempty"`
	Provisioning string `json:"provisioning,omitempty"`
	DerivedFrom  string `json:"derived_from,omitempty"`
	Required     bool   `json:"required"`
}

type CredentialProvenance struct {
	Version              string                         `json:"version,omitempty"`
	Owner                string                         `json:"owner"`
	SourceRef            string                         `json:"source_ref"`
	Kind                 string                         `json:"kind,omitempty"`
	Provider             string                         `json:"provider,omitempty"`
	AppliesWhen          *credentialspec.Applicability  `json:"applies_when,omitempty"`
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
	ObtainURL            string                         `json:"obtain_url,omitempty"`
	Provisioning         string                         `json:"provisioning,omitempty"`
	DerivedFrom          string                         `json:"derived_from,omitempty"`
	Required             bool                           `json:"required"`
	Consumers            []CredentialConsumerProvenance `json:"consumers,omitempty"`
}

// CredentialConsumerProvenance keeps the runtime obligation attached to the
// declaration that introduced it. A shared address can have multiple owners
// and delivery modes; flattening these to consumer names loses the source and
// requiredness needed by contextual projections.
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

type CredentialStatus struct {
	Identity       string `json:"identity"`
	Field          string `json:"field"`
	Configured     bool   `json:"configured"`
	Provider       string `json:"provider"`
	ProviderState  string `json:"provider_state"`
	ProviderDetail string `json:"provider_detail,omitempty"`
}

type ProvisionRequest struct {
	Identity string
	Field    string
	Value    string
}

type ProvisionResponse struct {
	Identity string `json:"identity"`
	Field    string `json:"field"`
	Provider string `json:"provider"`
	Status   string `json:"status"`
}

// ExposureMode makes plaintext handling an explicit part of the runtime
// contract. Brokered and typed uses keep values inside an authority boundary;
// runtime injection intentionally hands one value to one receiving process.
type ExposureMode string

const (
	ExposureBrokered         ExposureMode = "brokered"
	ExposureRuntimeInjection ExposureMode = "runtime_injection"
)

type HydrationRequest struct {
	Identity string
	Field    string
	Env      string
	Target   map[string]string
	// LeaseTTL bounds how long the caller may deliver this value to a
	// receiving process. The lifecycle owner must revoke the lease when that
	// process stops; the default is deliberately short for callers that omit it.
	LeaseTTL time.Duration
}

type HydrationResponse struct {
	Identity     string       `json:"identity"`
	Field        string       `json:"field"`
	ExposureMode ExposureMode `json:"exposure_mode"`
	Injected     bool         `json:"injected"`
	LeaseID      string       `json:"lease_id,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at,omitempty"`
}

// HydrationProvider is optional because broker-only consumers do not need to
// receive raw material. Implementations must document the receiving process
// and must never persist or log the target map.
type HydrationProvider interface {
	Hydrate(context.Context, HydrationRequest) (HydrationResponse, error)
}

// HydrationRevocationRequest is issued by the lifecycle owner after the
// receiving process is stopped. ProcessRunning is retained because a lease
// can stop future delivery without erasing a value already copied into a
// running process environment.
type HydrationRevocationRequest struct {
	LeaseID        string
	ProcessRunning bool
}

// HydrationRevocationResponse makes the exposure boundary explicit. A
// revoked lease always stops future delivery. If the process was already
// running, its inherited environment remains an exposure until that process
// exits and the response says so without pretending revocation erased it.
type HydrationRevocationResponse struct {
	LeaseID               string `json:"lease_id"`
	Status                string `json:"status"`
	FutureDeliveryStopped bool   `json:"future_delivery_stopped"`
	ProcessExposure       string `json:"process_exposure"`
}

// HydrationRevocationProvider is optional for broker-only clients. Runtime
// injection consumers must retain the lease and call this method from their
// control-plane lifecycle owner during stop, failure, and restart cleanup.
type HydrationRevocationProvider interface {
	RevokeHydration(context.Context, HydrationRevocationRequest) (HydrationRevocationResponse, error)
}

type ProviderDiagnosis struct {
	Platform    string `json:"platform"`
	Adapter     string `json:"adapter"`
	Backend     string `json:"backend"`
	Condition   string `json:"condition"`
	Available   bool   `json:"available"`
	Writable    bool   `json:"writable"`
	Explanation string `json:"explanation,omitempty"`
	Fix         string `json:"fix,omitempty"`
}

type RecoveryStatus struct {
	ReceiptExists            bool     `json:"receipt_exists"`
	ExportedAt               string   `json:"exported_at"`
	EntryCount               int      `json:"entry_count"`
	Uncovered                []string `json:"uncovered"`
	RequiredAbsent           []string `json:"required_absent,omitempty"`
	Basis                    string   `json:"basis"`
	ManagedInstancesIncluded bool     `json:"managed_instances_included"`
	Status                   string   `json:"status"`
	AgeSeconds               int64    `json:"age_seconds,omitempty"`
	FreshnessReason          string   `json:"freshness_reason,omitempty"`
}

type DoctorResponse struct {
	Provider                 ProviderDiagnosis `json:"provider"`
	Credentials              []CredentialRef   `json:"credentials"`
	CredentialCount          int               `json:"credential_count"`
	DeclarationSiteCount     int               `json:"declaration_site_count"`
	InventoryBasis           string            `json:"inventory_basis"`
	ManagedInstancesIncluded bool              `json:"managed_instances_included"`
	Recovery                 RecoveryStatus    `json:"recovery"`
}

// InventoryResponse is the metadata-only inventory shared by credential
// surfaces. Values are intentionally absent. Counts are explicit about their
// basis so declaration-site and distinct-address answers cannot be conflated.
type InventoryResponse struct {
	Credentials              []CredentialRef `json:"credentials"`
	CredentialCount          int             `json:"credential_count"`
	DeclarationSiteCount     int             `json:"declaration_site_count"`
	InventoryBasis           string          `json:"inventory_basis"`
	ManagedInstancesIncluded bool            `json:"managed_instances_included"`
	Uncovered                []string        `json:"uncovered"`
	RequiredAbsent           []string        `json:"required_absent,omitempty"`
}

// InventoryProvider is implemented by transports that can return the
// authoritative metadata inventory. It is optional to preserve compatibility
// with narrow test doubles and older transports.
type InventoryProvider interface {
	Inventory(context.Context) (InventoryResponse, error)
}

type RecoveryExportRequest struct {
	Entries    []CredentialRef
	Passphrase string
	OutputPath string
}

type RecoveryExportResponse struct {
	Path       string   `json:"path"`
	EntryCount int      `json:"entry_count"`
	Entries    []string `json:"entries"`
}

type RecoveryVerifyRequest struct {
	InputPath  string
	Passphrase string
}

type RecoveryVerifyResponse struct {
	Version int      `json:"version"`
	Entries []string `json:"entries"`
}

type RecoveryRestoreRequest struct {
	InputPath  string
	Passphrase string
}

type StoreStatus struct {
	Path        string `json:"path"`
	Initialized bool   `json:"initialized"`
	Unlocked    bool   `json:"unlocked"`
	Entries     int    `json:"entries"`
	Active      bool   `json:"active"`
}

type KeyringReport struct {
	Path          string          `json:"path"`
	Format        string          `json:"format,omitempty"`
	Assessed      bool            `json:"assessed"`
	Loadable      bool            `json:"loadable"`
	Repaired      int             `json:"repaired"`
	Verdict       string          `json:"verdict,omitempty"`
	VerdictReason string          `json:"verdict_reason,omitempty"`
	Backups       []KeyringBackup `json:"backups,omitempty"`
}

type KeyringBackup struct {
	Path       string `json:"path"`
	ModifiedAt string `json:"modified_at"`
	AgeSeconds int64  `json:"age_seconds"`
}

type Client interface {
	Provision(context.Context, ProvisionRequest) (ProvisionResponse, error)
	// Resolve returns a credential only to the requesting in-process consumer;
	// callers must keep the value in memory and never persist or log it.
	Resolve(context.Context, string, string) (string, error)
	Delete(context.Context, string, string) error
	Status(context.Context, string, string) (CredentialStatus, error)
	List(context.Context) ([]CredentialRef, error)
	Doctor(context.Context) (DoctorResponse, error)
	KeyringInspect(context.Context, string) (KeyringReport, error)
	KeyringRepair(context.Context, string) (KeyringReport, error)
	RecoveryExport(context.Context, RecoveryExportRequest) (RecoveryExportResponse, error)
	RecoveryVerify(context.Context, RecoveryVerifyRequest) (RecoveryVerifyResponse, error)
	RecoveryRestore(context.Context, RecoveryRestoreRequest) error
	StoreStatus(context.Context) (StoreStatus, error)
}

type ErrTransportUnavailable struct{ Transport string }

func (e ErrTransportUnavailable) Error() string {
	return e.Transport + " credential transport is unavailable"
}
