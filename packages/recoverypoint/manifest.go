package recoverypoint

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/packages/cloudrelease"
)

// FormatVersion is the on-disk recovery point format.
const FormatVersion = 1

// ManifestFile is the manifest name inside a recovery point directory.
const ManifestFile = "recovery-point.json"

// Binding kinds.
const (
	KindSQL         = "sql"
	KindFiles       = "files"
	KindApplication = "application"
)

// Provider names the package ships. A registry may add more (the package-lane
// drill registers an in-process SQLite provider under "sqlite").
const (
	ProviderPostgres         = "postgres"
	ProviderObjectStore      = "object_store"
	ProviderApplicationHooks = "application_hooks"
)

// Write-quiescence outcomes recorded per binding.
const (
	QuiescenceQuiesced     = "quiesced"
	QuiescenceSnapshotSafe = "snapshot_safe"
	QuiescenceNotDeclared  = "not_declared"
)

// Consistency modes recorded per binding.
const (
	ModeDatabaseNative   = "database_native"
	ModeObjectInventory  = "object_inventory"
	ModeApplicationHooks = "application_hooks"
)

// Migration postures (storage-steer categories) recorded on a recovery point.
const (
	PostureGreenfield          = "greenfield"
	PostureGreenfieldWithData  = "greenfield_with_data"
	PostureProductionEvolution = "production_evolution"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// Hook is a declared application argv (never a shell string).
type Hook struct {
	Tool string   `json:"tool"`
	Argv []string `json:"argv"`
}

// Binding is one declared persistent-data binding as the engine sees it.
type Binding struct {
	ID       string `json:"id"`
	Owner    string `json:"owner,omitempty"`
	Kind     string `json:"kind"`
	Provider string `json:"provider"`
	// Locator is provider-specific: a database name for SQL providers, an
	// absolute directory for the object store.
	Locator        string `json:"locator"`
	MigrationOwner string `json:"migration_owner,omitempty"`
	// Quiesce and Release are the declared application hooks entered around
	// the capture. Absent hooks are recorded as write_quiescence not_declared.
	Quiesce *Hook `json:"quiesce,omitempty"`
	Release *Hook `json:"release,omitempty"`
}

// Validate checks the binding shape.
func (b Binding) Validate() error {
	if !identifierPattern.MatchString(b.ID) {
		return newError(CodeInvalidArgument, "binding id %q is not a valid identifier", b.ID)
	}
	switch b.Kind {
	case KindSQL, KindFiles, KindApplication:
	default:
		return newError(CodeInvalidArgument, "binding %s: kind %q is not sql, files or application", b.ID, b.Kind)
	}
	if strings.TrimSpace(b.Provider) == "" {
		return newError(CodeInvalidArgument, "binding %s: provider is required", b.ID)
	}
	if strings.TrimSpace(b.Locator) == "" {
		return newError(CodeInvalidArgument, "binding %s: locator is required", b.ID)
	}
	return nil
}

// Inventory is the count and content checksum of one binding's data.
type Inventory struct {
	Count    int64  `json:"count"`
	Checksum string `json:"checksum"`
	// Comparable is false when the provider's artifact bytes are not
	// reproducible (a pg_dump header carries a timestamp), so only Count and
	// application invariants may be compared after a restore.
	Comparable bool `json:"comparable"`
}

// ConsistencyRecord is what the consistency boundary reported for a binding.
type ConsistencyRecord struct {
	Binding         string `json:"binding"`
	Mode            string `json:"mode"`
	WriteQuiescence string `json:"write_quiescence"`
	Token           string `json:"token"`
}

// Artifact is one sealed provider artifact inside the directory.
type Artifact struct {
	Binding     string `json:"binding"`
	File        string `json:"file"`
	Bytes       int64  `json:"bytes"`
	SHA256      string `json:"sha256"`
	PlainSHA256 string `json:"plain_sha256"`
	PlainBytes  int64  `json:"plain_bytes"`
}

// Refs binds the recovery point to the release it was captured under.
type Refs struct {
	SchemaVersion         string   `json:"schema_version"`
	ConfigurationDigest   string   `json:"configuration_digest"`
	CredentialVersionRefs []string `json:"credential_version_refs"`
}

// Manifest is recovery-point.json.
type Manifest struct {
	FormatVersion    int                  `json:"format_version"`
	ID               string               `json:"id"`
	DeploymentID     string               `json:"deployment_id"`
	Bindings         []Binding            `json:"bindings"`
	Refs             Refs                 `json:"refs"`
	ConsistencyToken string               `json:"consistency_token"`
	Consistency      []ConsistencyRecord  `json:"consistency"`
	CapturedAt       time.Time            `json:"captured_at"`
	Provider         string               `json:"provider"`
	ProviderRef      string               `json:"provider_ref,omitempty"`
	Checksums        map[string]Inventory `json:"checksums"`
	Artifacts        []Artifact           `json:"artifacts"`
	Encrypted        bool                 `json:"encrypted"`
	KeyRef           string               `json:"key_ref"`
	RetentionPolicy  string               `json:"retention_policy"`
	Protected        bool                 `json:"protected"`
	MigrationPosture string               `json:"migration_posture"`
	Digest           string               `json:"digest"`
}

// BindingIDs returns the captured binding ids in manifest order.
func (m Manifest) BindingIDs() []string {
	ids := make([]string, 0, len(m.Bindings))
	for _, b := range m.Bindings {
		ids = append(ids, b.ID)
	}
	return ids
}

// ComputeDigest returns the sha256 over the canonical JSON of the manifest
// with Digest cleared.
func (m Manifest) ComputeDigest() (string, error) {
	m.Digest = ""
	return cloudrelease.CanonicalDigest(m)
}

// WriteManifest stamps the digest and writes the manifest atomically.
func WriteManifest(dir string, m Manifest) (Manifest, error) {
	digest, err := m.ComputeDigest()
	if err != nil {
		return m, newError(CodeCaptureFailed, "digest manifest: %v", err)
	}
	m.Digest = digest
	if err := writeJSONAtomic(filepath.Join(dir, ManifestFile), m); err != nil {
		return m, newError(CodeCaptureFailed, "write manifest: %v", err)
	}
	return m, nil
}

// ReadManifest reads and verifies the manifest digest. A missing file is
// os.ErrNotExist; anything unparsable or with a wrong digest is corrupt.
func ReadManifest(dir string) (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, ManifestFile)) //nolint:gosec // caller-owned recovery point directory
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Manifest{}, err
		}
		return Manifest{}, newError(CodeRecoveryPointCorrupt, "read manifest: %v", err)
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return Manifest{}, newError(CodeRecoveryPointCorrupt, "manifest is not valid JSON: %v", err)
	}
	if m.FormatVersion != FormatVersion {
		return Manifest{}, newError(CodeRecoveryPointCorrupt, "manifest format_version %d is not %d", m.FormatVersion, FormatVersion)
	}
	digest, err := m.ComputeDigest()
	if err != nil {
		return Manifest{}, newError(CodeRecoveryPointCorrupt, "digest manifest: %v", err)
	}
	if digest != m.Digest {
		return Manifest{}, newError(CodeRecoveryPointCorrupt, "manifest digest mismatch").withDetail("recorded_digest", m.Digest).withDetail("computed_digest", digest)
	}
	return m, nil
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp-" + randomSuffix()
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	if err := platform.AtomicReplace(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func randomSuffix() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}
