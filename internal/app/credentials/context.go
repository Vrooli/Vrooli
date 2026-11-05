package credentials

import "time"

// ListOptions controls credential inventory output.
type ListOptions struct{ Format string }

// CredentialSelectorOptions identifies one credential without carrying CLI
// parser state into the service.
type CredentialSelectorOptions struct {
	Identity string
	Field    string
	Format   string
	Yes      bool
}

// DoctorOptions controls credential-provider diagnosis.
type DoctorOptions struct {
	Format      string
	CheckWrites bool
}

// RecoveryExportOptions selects the entries written to a recovery bundle.
type RecoveryExportOptions struct {
	Entries []string
	Output  string
	All     bool
	Format  string
}

// RecoveryBundleOptions identifies an existing recovery bundle.
type RecoveryBundleOptions struct {
	Input  string
	Format string
}

// KeyringOptions identifies a keyring and output format.
type KeyringOptions struct {
	Path   string
	Format string
}

// KeyringRepairOptions adds explicit backup-repair policy.
type KeyringRepairOptions struct {
	Path                 string
	Format               string
	RetireBackup         string
	OfferRetireOlderThan time.Duration
}

// BreakGlassOptions contains already-validated break-glass command input.
type BreakGlassOptions struct {
	Operation   string
	AccountID   string
	Audience    string
	Purpose     string
	Target      string
	Scopes      string
	Scope       string
	OperatorID  string
	MachineID   string
	NodeID      string
	PlanHash    string
	OperationID string
	TTL         time.Duration
	Format      string
}

// StoreCopyOptions describes one encrypted-store copy operation.
type StoreCopyOptions struct {
	Sink                      string
	ObjectStoreCredentialID   string
	ObjectStoreRegion         string
	ObjectStoreEndpoint       string
	ObjectStoreAccessKeyField string
	ObjectStoreSecretKeyField string
	ObjectStoreSessionField   string
	Configured                bool
	Format                    string
}

// StoreCopyConfigureOptions describes persisted copy configuration.
type StoreCopyConfigureOptions struct {
	Sink                      string
	Interval                  time.Duration
	ObjectStoreCredentialID   string
	ObjectStoreRegion         string
	ObjectStoreEndpoint       string
	ObjectStoreAccessKeyField string
	ObjectStoreSecretKeyField string
	ObjectStoreSessionField   string
	Enabled                   bool
	Format                    string
}

// ExtensionOptions controls browser native-messaging registration for the
// Secrets Manager extension. It contains public installation metadata only.
type ExtensionOptions struct {
	Operation   string
	HostPath    string
	ExtensionID string
	Browser     string
	Format      string
	Yes         bool
}
