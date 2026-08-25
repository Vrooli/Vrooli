package projectstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

const (
	setupDirName        = "setup"
	projectKeyHash      = 12
	migrationLedgerName = "migrations.json"
)

// MigrationStatus is the durable state of one compatibility migration.
// Incomplete states are intentionally retryable; only complete is trusted by
// setup as proof that the migration has been verified.
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
	return MigrationLedger{SchemaVersion: 1, Migrations: map[string]MigrationRecord{}}
}

type Locator struct {
	home          string
	root          string
	key           string
	setupStateDir string
}

func NewLocator(home, root string) (Locator, error) {
	resolvedHome := strings.TrimSpace(home)
	if resolvedHome == "" {
		var err error
		resolvedHome, err = config.HomeDir()
		if err != nil {
			return Locator{}, fmt.Errorf("resolve home: %w", err)
		}
	}
	resolvedRoot := strings.TrimSpace(root)
	if resolvedRoot == "" {
		return Locator{}, fmt.Errorf("project root is required")
	}
	absRoot, err := filepath.Abs(resolvedRoot)
	if err == nil {
		resolvedRoot = absRoot
	}
	resolvedRoot = filepath.Clean(resolvedRoot)
	cleanHome := filepath.Clean(resolvedHome)
	key := projectKey(resolvedRoot)
	// Resolve the project-state directory from the runtime_home authority. The
	// contract is required; a load failure surfaces here (no fallback).
	projectStateDir, err := repocontract.RuntimeHomeScopedPath(cleanHome, repocontract.ScopedProjectState, map[string]string{"project_key": key})
	if err != nil {
		return Locator{}, fmt.Errorf("resolve project state dir: %w", err)
	}
	return Locator{
		home:          cleanHome,
		root:          resolvedRoot,
		key:           key,
		setupStateDir: filepath.Join(projectStateDir, setupDirName),
	}, nil
}

func MustLocator(home, root string) Locator {
	locator, err := NewLocator(home, root)
	if err != nil {
		panic(err)
	}
	return locator
}

func (l Locator) Home() string {
	return l.home
}

func (l Locator) Root() string {
	return l.root
}

func (l Locator) ProjectKey() string {
	return l.key
}

func (l Locator) SetupStateDir() string {
	return l.setupStateDir
}

// MigrationLedgerPath stores named compatibility migrations beside the other
// project-scoped setup state. It is deliberately not part of bootstrap marker
// payloads, whose setup_version meaning is unrelated.
func (l Locator) MigrationLedgerPath() string {
	return filepath.Join(l.SetupStateDir(), migrationLedgerName)
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
	if ledger.SchemaVersion != 1 {
		return MigrationLedger{}, fmt.Errorf("unsupported migration ledger schema_version %d", ledger.SchemaVersion)
	}
	if ledger.Migrations == nil {
		ledger.Migrations = map[string]MigrationRecord{}
	}
	return ledger, nil
}

func SaveMigrationLedger(l Locator, ledger MigrationLedger) error {
	if ledger.SchemaVersion == 0 {
		ledger.SchemaVersion = 1
	}
	if ledger.SchemaVersion != 1 {
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
	return config.WriteOwnedFileAtomic(l.MigrationLedgerPath(), append(data, '\n'), 0o644)
}

// ActiveSetupPath is the best-effort, versioned in-progress setup record.
// It is separate from terminal result files so interrupted setup can be
// diagnosed without changing the existing result contract.
func (l Locator) ActiveSetupPath() string {
	return filepath.Join(l.SetupStateDir(), "active-setup.json")
}

// BootstrapCompletePath is written by the elevated, non-interactive setup
// phase. It is intentionally distinct from configuration completion, which is
// owned by onboarding.
func (l Locator) BootstrapCompletePath() string {
	return filepath.Join(l.SetupStateDir(), ".bootstrap-complete")
}

func (l Locator) ConfigurationCompletePath() string {
	return filepath.Join(l.SetupStateDir(), ".configuration-complete")
}

func (l Locator) ResourcesPopulatedPath() string {
	return filepath.Join(l.SetupStateDir(), ".resources-populated")
}

func (l Locator) ResourcePopulatedPath(resource string) string {
	return filepath.Join(l.SetupStateDir(), "."+safeMarkerName(resource)+"-populated")
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

func projectKey(root string) string {
	base := safeProjectName(filepath.Base(root))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "project"
	}
	sum := sha256.Sum256([]byte(filepath.Clean(root)))
	return base + "-" + hex.EncodeToString(sum[:])[:projectKeyHash]
}

func safeProjectName(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
		if !ok {
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}
		b.WriteRune(r)
		lastDash = r == '-'
	}
	return strings.Trim(b.String(), "-")
}

func safeMarkerName(value string) string {
	name := safeProjectName(value)
	if name == "" {
		sum := sha256.Sum256([]byte(value))
		return "resource-" + hex.EncodeToString(sum[:])[:projectKeyHash]
	}
	return name
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
	payload := map[string]any{
		"completed_at":     time.Now().UTC().Format(time.RFC3339),
		"phase":            "configuration_complete",
		"project_key":      locator.ProjectKey(),
		"selection_digest": strings.TrimSpace(selectionDigest),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteOwnedFile(locator.ConfigurationCompletePath(), append(data, '\n'), 0o644)
}
