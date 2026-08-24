// Package credentialclient is the typed client boundary for credential
// consumers. It keeps scenario code independent of the bootstrap CLI while
// allowing the same request to use local authority, desktop IPC, or remote
// SSH transport.
package credentialclient

import "context"

type CredentialRef struct {
	Resource  string `json:"resource"`
	Env       string `json:"env"`
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Label     string `json:"label,omitempty"`
	Required  bool   `json:"required"`
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
	Basis                    string   `json:"basis"`
	ManagedInstancesIncluded bool     `json:"managed_instances_included"`
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
