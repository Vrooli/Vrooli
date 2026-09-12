package domain

import "time"

// RecoveryPointSchemaVersion is stamped on every recovery point record.
const RecoveryPointSchemaVersion = 1

// Data binding kinds and the providers that own their consistency boundary.
const (
	DataBindingKindSQL         = "sql"
	DataBindingKindFiles       = "files"
	DataBindingKindApplication = "application"

	BackupProviderPostgres         = "postgres"
	BackupProviderSQLite           = "sqlite"
	BackupProviderObjectStore      = "object_store"
	BackupProviderApplicationHooks = "application_hooks"
)

// Migration postures (storage-steer categories). The posture is selected from
// the actual predecessor state and recorded on the recovery point so the
// restore side knows which schema story the data carries.
const (
	MigrationPostureGreenfield          = "greenfield"
	MigrationPostureGreenfieldWithData  = "greenfield_with_data"
	MigrationPostureProductionEvolution = "production_evolution"
)

// Write-quiescence outcomes recorded per binding on a recovery point.
const (
	WriteQuiescenceQuiesced     = "quiesced"
	WriteQuiescenceSnapshotSafe = "snapshot_safe"
	WriteQuiescenceNotDeclared  = "not_declared"
)

// Restore outcomes.
const (
	RestoreOutcomeSucceeded = "succeeded"
	RestoreOutcomeFailed    = "failed"
	RestoreOutcomeRefused   = "refused"
)

// DataHook is a declared argv hook (never a shell string).
type DataHook struct {
	Tool string   `json:"tool"`
	Argv []string `json:"argv"`
}

// DataBinding is one declared persistent-data set as the recovery owner sees
// it: the closure's persistent_data entry resolved to a provider and a
// locator the provider understands. Locator is a database name for SQL
// bindings or an absolute directory for object stores; never a host path
// for a database.
type DataBinding struct {
	ID             string `json:"id"`
	Owner          string `json:"owner,omitempty"`
	Kind           string `json:"kind"`
	Provider       string `json:"provider"`
	Locator        string `json:"locator"`
	MigrationOwner string `json:"migration_owner,omitempty"`
	// Quiesce and Release are the declared application hooks entered around
	// the capture. When absent the recovery point records the limitation as
	// write_quiescence not_declared (snapshot_safe for database-native
	// providers).
	Quiesce *DataHook `json:"quiesce,omitempty"`
	Release *DataHook `json:"release,omitempty"`
	// ProviderRef is the backup owner's reference for this binding (for
	// example the data-backup-manager target id), empty when unregistered.
	ProviderRef string `json:"provider_ref,omitempty"`
}

// BindingChecksum is the count and content checksum of one binding.
type BindingChecksum struct {
	Count    int64  `json:"count"`
	Checksum string `json:"checksum"`
	// Comparable is false when the provider artifact is not byte-reproducible
	// (a pg_dump carries a timestamp): only Count and application invariants
	// may then be compared after a restore.
	Comparable bool `json:"comparable"`
}

// ConsistencyRecord is what the consistency boundary reported for a binding.
type ConsistencyRecord struct {
	Binding         string `json:"binding"`
	Mode            string `json:"mode"`
	WriteQuiescence string `json:"write_quiescence"`
	Token           string `json:"token"`
}

// RecoveryPoint is one immutable, encrypted, checksummed capture bound to the
// release, schema, configuration and credential versions it was taken under.
// It carries references only: the recovery key is a reference resolvable
// outside the target's failure domain and never material.
type RecoveryPoint struct {
	ID                    string                     `json:"id"`
	SchemaVersionRecord   int                        `json:"schema_version_record"`
	DeploymentID          string                     `json:"deployment_id"`
	OperationID           string                     `json:"operation_id,omitempty"`
	BindingIDs            []string                   `json:"binding_ids"`
	Bindings              []DataBinding              `json:"bindings"`
	SchemaVersion         string                     `json:"schema_version"`
	ConfigurationDigest   string                     `json:"configuration_digest"`
	ReleaseDigest         string                     `json:"release_digest"`
	CredentialVersionRefs []string                   `json:"credential_version_refs"`
	ConsistencyToken      string                     `json:"consistency_token"`
	Consistency           []ConsistencyRecord        `json:"consistency"`
	CapturedAt            time.Time                  `json:"captured_at"`
	Provider              string                     `json:"provider"`
	ProviderRef           string                     `json:"provider_ref,omitempty"`
	Checksums             map[string]BindingChecksum `json:"checksums"`
	Encrypted             bool                       `json:"encrypted"`
	RecoveryKeyRef        string                     `json:"recovery_key_ref"`
	RetentionPolicy       string                     `json:"retention_policy"`
	Protected             bool                       `json:"protected"`
	// ProtectedBy names what holds the point (release digest, operation id).
	ProtectedBy      []string  `json:"protected_by,omitempty"`
	MigrationPosture string    `json:"migration_posture"`
	Location         string    `json:"location"`
	ManifestDigest   string    `json:"manifest_digest"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// InvariantResult is one application-level or inventory invariant checked
// after a restore.
type InvariantResult struct {
	Binding  string `json:"binding"`
	Check    string `json:"check"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	Passed   bool   `json:"passed"`
}

// RestoreReceipt records one restore attempt with its measured recovery
// time (RTO) and recovery-point age (RPO) at the moment the restore started.
type RestoreReceipt struct {
	ID               string            `json:"id"`
	RecoveryPointID  string            `json:"recovery_point_id"`
	DeploymentID     string            `json:"deployment_id"`
	TargetRef        string            `json:"target_ref"`
	StartedAt        time.Time         `json:"started_at"`
	CompletedAt      time.Time         `json:"completed_at"`
	MeasuredRTO      time.Duration     `json:"measured_rto_ns"`
	RecoveryPointAge time.Duration     `json:"recovery_point_age_ns"`
	InvariantResults []InvariantResult `json:"invariant_results"`
	Outcome          string            `json:"outcome"`
	ErrorCode        string            `json:"error_code,omitempty"`
	ErrorMessage     string            `json:"error_message,omitempty"`
	// Budget verdicts are recorded against certification/budgets.json.
	RTOBudgetSeconds int64 `json:"rto_budget_seconds,omitempty"`
	RPOBudgetSeconds int64 `json:"rpo_budget_seconds,omitempty"`
	WithinBudgets    bool  `json:"within_budgets"`
}

// RollbackVerdict is the schema-aware admission decision for a code
// rollback. Compatible rollbacks keep the data in place; incompatible ones
// carry a typed forward-repair or restore plan instead of a bare refusal.
type RollbackVerdict struct {
	Compatible     bool   `json:"compatible"`
	SchemaStrategy string `json:"schema_strategy"`
	CurrentSchema  string `json:"current_schema"`
	TargetSchema   string `json:"target_schema"`
	ReasonCode     string `json:"reason_code,omitempty"`
	Reason         string `json:"reason,omitempty"`
	// Plan is present only when the rollback is refused.
	Plan *RecoveryPlan `json:"plan,omitempty"`
}

// RecoveryPlan is the typed alternative offered when rollback is refused.
type RecoveryPlan struct {
	Kind            string   `json:"kind"`
	RecoveryPointID string   `json:"recovery_point_id,omitempty"`
	Preconditions   []string `json:"preconditions"`
	Steps           []string `json:"steps"`
}

// Recovery plan kinds.
const (
	RecoveryPlanForwardRepair = "forward_repair"
	RecoveryPlanRestore       = "restore"
)
