package projectstate

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/config"
)

// MigrationStatus is the durable state of one compatibility migration.
// Incomplete states are intentionally retryable; only complete is trusted by
// setup as proof that the migration has been verified.
const migrationLedgerSchemaVersion = 1

const configurationCompletePhase = "configuration_complete"

type MigrationStatus string

const (
	MigrationPending     MigrationStatus = "pending"
	MigrationRunning     MigrationStatus = "running"
	MigrationComplete    MigrationStatus = "complete"
	MigrationFailed      MigrationStatus = "failed"
	MigrationInterrupted MigrationStatus = "interrupted"
)

type MigrationScope struct {
	Kind    string   `json:"kind"`
	Classes []string `json:"classes,omitempty"`
}

type MigrationExpectedIdentity struct {
	UID uint32 `json:"uid"`
	GID uint32 `json:"gid"`
}

type MigrationResult struct {
	Scanned  uint64 `json:"scanned"`
	Repaired uint64 `json:"repaired"`
	Skipped  uint64 `json:"skipped"`
	Failed   uint64 `json:"failed"`
	Duration int64  `json:"duration_ms"`
}

type MigrationRecord struct {
	AppliedThrough int                       `json:"applied_through"`
	Status         MigrationStatus           `json:"status"`
	Scope          MigrationScope            `json:"scope"`
	Expected       MigrationExpectedIdentity `json:"expected_identity"`
	StartedAt      string                    `json:"started_at,omitempty"`
	CompletedAt    string                    `json:"completed_at,omitempty"`
	Result         MigrationResult           `json:"result"`
	LastError      string                    `json:"last_error,omitempty"`
	Cursors        map[string]string         `json:"cursors,omitempty"`
	Completed      []string                  `json:"completed_classes,omitempty"`
}

type MigrationLedger struct {
	SchemaVersion int                        `json:"schema_version"`
	Migrations    map[string]MigrationRecord `json:"migrations"`
}

type ConfigurationCompletion struct {
	CompletedAt     string `json:"completed_at"`
	Phase           string `json:"phase"`
	ProjectKey      string `json:"project_key"`
	SelectionDigest string `json:"selection_digest"`
}

func NewMigrationLedger() MigrationLedger {
	return MigrationLedger{SchemaVersion: migrationLedgerSchemaVersion, Migrations: map[string]MigrationRecord{}}
}

func LoadMigrationLedger(l Locator) (MigrationLedger, error) {
	data, err := os.ReadFile(l.MigrationLedgerPath())
	if err != nil {
		if os.IsNotExist(err) {
			return NewMigrationLedger(), nil
		}
		return MigrationLedger{}, fmt.Errorf("read migration ledger: %w", err)
	}
	var ledger MigrationLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return MigrationLedger{}, fmt.Errorf("decode migration ledger: %w", err)
	}
	if ledger.SchemaVersion != migrationLedgerSchemaVersion {
		return MigrationLedger{}, fmt.Errorf("unsupported migration ledger schema_version %d", ledger.SchemaVersion)
	}
	if ledger.Migrations == nil {
		ledger.Migrations = map[string]MigrationRecord{}
	}
	return ledger, nil
}

func SaveMigrationLedger(l Locator, ledger MigrationLedger) error {
	if ledger.SchemaVersion == 0 {
		ledger.SchemaVersion = migrationLedgerSchemaVersion
	}
	if ledger.SchemaVersion != migrationLedgerSchemaVersion {
		return fmt.Errorf("unsupported migration ledger schema_version %d", ledger.SchemaVersion)
	}
	if ledger.Migrations == nil {
		ledger.Migrations = map[string]MigrationRecord{}
	}
	data, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return fmt.Errorf("encode migration ledger: %w", err)
	}
	if _, err := config.EnsureOwnedDir(l.SetupStateDir()); err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(l.MigrationLedgerPath(), append(data, '\n'), tuning.PermFile)
}

func (l Locator) HasBootstrapComplete() bool {
	return fileExists(l.BootstrapCompletePath())
}

func (l Locator) HasConfigurationComplete() bool {
	return fileExists(l.ConfigurationCompletePath())
}

func (l Locator) ReadConfigurationComplete() (ConfigurationCompletion, error) {
	data, err := os.ReadFile(l.ConfigurationCompletePath())
	if err != nil {
		return ConfigurationCompletion{}, err
	}
	var marker ConfigurationCompletion
	if err := json.Unmarshal(data, &marker); err != nil {
		return ConfigurationCompletion{}, fmt.Errorf("decode configuration completion marker: %w", err)
	}
	return marker, nil
}

func (l Locator) HasResourcesPopulated() bool {
	return fileExists(l.ResourcesPopulatedPath())
}

func (l Locator) HasResourcePopulated(resource string) bool {
	return fileExists(l.ResourcePopulatedPath(resource))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MarkConfigurationComplete records the second half of first run after
// onboarding has applied and validated the operator's choices. It never
// overwrites the bootstrap marker.
func MarkConfigurationComplete(home, root, selectionDigest string) error {
	locator, err := NewLocator(home, root)
	if err != nil {
		return err
	}
	if !locator.HasBootstrapComplete() {
		return fmt.Errorf("cannot mark configuration complete before bootstrap completion")
	}
	if _, err := config.EnsureOwnedDir(locator.SetupStateDir()); err != nil {
		return err
	}
	payload := ConfigurationCompletion{
		CompletedAt:     time.Now().UTC().Format(time.RFC3339),
		Phase:           configurationCompletePhase,
		ProjectKey:      locator.ProjectKey(),
		SelectionDigest: strings.TrimSpace(selectionDigest),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteOwnedFile(locator.ConfigurationCompletePath(), append(data, '\n'), tuning.PermFile)
}
