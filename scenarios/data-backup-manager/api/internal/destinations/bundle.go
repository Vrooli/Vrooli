package destinations

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BundleSchemaVersion identifies the on-disk vrooli-backup-destination.json
// layout. Bump when the manifest shape changes incompatibly.
const BundleSchemaVersion = 1

// Bundle file names written into the operator-facing bundle root.
const (
	BundleReadmeFile   = "README.txt"
	BundleRecoveryFile = "RECOVERY.txt"
	BundleManifestFile = "vrooli-backup-destination.json"
)

// BundleWriter is the destination-owned seam that turns a bare filesystem
// folder into a self-describing Vrooli backup bundle: README.txt, RECOVERY.txt,
// a non-secret JSON manifest, and the repositories/<slug>.kopia directory that
// holds the vanilla kopia repository. S3 backends skip this seam entirely (no
// filesystem bundle files).
//
// seam: BundleWriter materializes the filesystem destination bundle. Production
// wires FSBundleWriter (bundle.go); destinations service tests wire
// mocks.FakeBundleWriter so CreateDestination is exercised without touching the
// real filesystem.
type BundleWriter interface {
	// PrepareRepository creates the bundle root and the repository directory
	// (repositoryPath, expected to be RepositoryPathFor(bundleRoot, name)).
	// It MUST refuse a bundle root that is already a kopia repository at its
	// root (a kopia.repository file present directly under bundleRoot), so the
	// operator never silently nests one repository inside another.
	PrepareRepository(ctx context.Context, bundleRoot, repositoryPath string) error
	// WriteMetadata writes README.txt, RECOVERY.txt and the JSON manifest into
	// the bundle root. It is idempotent only when the on-disk content matches;
	// pre-existing files with conflicting content fail with an actionable error
	// rather than overwriting unknown data.
	WriteMetadata(ctx context.Context, meta BundleMetadata) error
	// RefreshMetadata updates only Vrooli-owned recovery metadata for an
	// existing destination whose identity has already been verified. It is
	// separate from WriteMetadata so create remains fail-closed around unknown
	// files while migrations can repair a stale credential reference.
	RefreshMetadata(ctx context.Context, meta BundleMetadata) error
	// RebindMetadata repairs generated absolute-path projections after the
	// destination owner has verified the same bundle at a relocated mountpoint.
	RebindMetadata(ctx context.Context, meta BundleMetadata) error
	ValidateRelocation(ctx context.Context, meta BundleMetadata) error
	ValidatePendingCreate(ctx context.Context, meta BundleMetadata) error
	// ReadExisting inspects an already-created filesystem bundle without
	// modifying it. Import must never create a repository in a non-empty path.
	ReadExisting(ctx context.Context, bundleRoot string) (ExistingBundle, error)
}

// ExistingBundle is the non-secret, repository-anchored identity read from an
// existing DBM filesystem bundle.
type ExistingBundle struct {
	Metadata           BundleMetadata
	RepositoryUniqueID string
}

// BundleMetadata is the non-secret information rendered into the bundle files.
// It never carries a passphrase, token, or other secret value — only a
// credential-authority identity/field reference.
type BundleMetadata struct {
	DestinationID       string
	Name                string
	Backend             string
	BundleRoot          string
	RelativePath        string
	RepositoryPath      string
	RepositoryUniqueID  string
	EncryptionAlgorithm string
	SecretRef           string
	CreatedAt           time.Time
	Host                string
	User                string
}

// bundleManifest is the JSON shape persisted to vrooli-backup-destination.json.
type bundleManifest struct {
	SchemaVersion       int      `json:"schema_version"`
	Kind                string   `json:"kind"`
	Producer            string   `json:"producer"`
	DestinationID       string   `json:"destination_id"`
	DestinationName     string   `json:"destination_name"`
	Backend             string   `json:"backend"`
	BundleRoot          string   `json:"bundle_root"`
	RelativePath        string   `json:"relative_path,omitempty"`
	RepositoryPath      string   `json:"repository_path"`
	RepositoryUniqueID  string   `json:"repository_unique_id,omitempty"`
	EncryptionAlgorithm string   `json:"encryption_algorithm"`
	SecretRef           string   `json:"secret_ref"`
	CreatedAt           string   `json:"created_at"`
	Host                string   `json:"host,omitempty"`
	User                string   `json:"user,omitempty"`
	RestoreHints        []string `json:"restore_hints"`
}

// FSBundleWriter is the production BundleWriter backed by the local filesystem.
type FSBundleWriter struct {
	// DirPerm / FilePerm default to 0o700 / 0o600 when zero so a backup bundle
	// on a removable drive is not world-readable where the OS honours modes.
	DirPerm  os.FileMode
	FilePerm os.FileMode
}

// Compile-time guarantee.
var _ BundleWriter = (*FSBundleWriter)(nil)

func (w *FSBundleWriter) dirPerm() os.FileMode {
	if w.DirPerm != 0 {
		return w.DirPerm
	}
	return 0o700
}

func (w *FSBundleWriter) filePerm() os.FileMode {
	if w.FilePerm != 0 {
		return w.FilePerm
	}
	return 0o600
}

func (w *FSBundleWriter) PrepareRepository(_ context.Context, bundleRoot, repositoryPath string) error {
	bundleRoot = filepath.Clean(bundleRoot)
	// Refuse to wrap a folder that is itself a kopia repository: a destination
	// bundle root must be a plain directory, with the repository nested under
	// repositories/<slug>.kopia.
	for _, marker := range []string{"kopia.repository.f", "kopia.repository"} {
		if _, err := os.Stat(filepath.Join(bundleRoot, marker)); err == nil {
			return ErrInvalidDestination{Field: "location", Reason: "bundle root already contains a kopia repository at its root; choose an empty folder or a dedicated subdirectory"}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect bundle-root repository marker: %w", err)
		}
	}
	if err := os.MkdirAll(bundleRoot, w.dirPerm()); err != nil {
		return fmt.Errorf("create bundle root %q: %w", bundleRoot, err)
	}
	if err := os.MkdirAll(repositoryPath, w.dirPerm()); err != nil {
		return fmt.Errorf("create repository dir %q: %w", repositoryPath, err)
	}
	entries, err := os.ReadDir(repositoryPath)
	if err != nil {
		return fmt.Errorf("inspect repository directory %q: %w", repositoryPath, err)
	}
	if len(entries) != 0 {
		return ErrInvalidDestination{Field: "repository_path", Reason: "repository directory contains existing content; refusing to create or adopt it without a durable provisioning intent"}
	}
	return nil
}

func (w *FSBundleWriter) WriteMetadata(_ context.Context, meta BundleMetadata) error {
	uniqueID, err := repositoryUniqueID(meta.RepositoryPath)
	if err != nil {
		return fmt.Errorf("read Kopia repository identity for bundle: %w", err)
	}
	meta.RepositoryUniqueID = uniqueID
	files := map[string][]byte{
		BundleReadmeFile:   []byte(renderReadme(meta)),
		BundleRecoveryFile: []byte(renderRecovery(meta)),
	}
	manifestBytes, err := renderManifest(meta)
	if err != nil {
		return err
	}
	files[BundleManifestFile] = manifestBytes

	for name, content := range files {
		if err := w.writeIfAbsentOrMatching(filepath.Join(meta.BundleRoot, name), content); err != nil {
			return err
		}
	}
	return nil
}

// RefreshMetadata repairs generated recovery metadata without replacing the
// repository or unrelated files. The manifest is the identity anchor: if it
// is absent, malformed, or belongs to another destination, refuse the update
// rather than guessing which files are safe to overwrite.
func (w *FSBundleWriter) RefreshMetadata(_ context.Context, meta BundleMetadata) error {
	manifestPath := filepath.Join(meta.BundleRoot, BundleManifestFile)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read destination manifest for metadata refresh: %w", err)
	}
	var existing bundleManifest
	if err := json.Unmarshal(manifestBytes, &existing); err != nil {
		return fmt.Errorf("decode destination manifest for metadata refresh: %w", err)
	}
	if err := validateManifestIdentity(existing, meta); err != nil {
		return err
	}

	// Preserve all existing non-secret metadata. This migration owns only the
	// credential-authority reference in the manifest.
	existing.SecretRef = meta.SecretRef
	updatedManifest, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("encode refreshed destination manifest: %w", err)
	}
	updatedManifest = append(updatedManifest, '\n')
	if err := w.replaceManagedFile(manifestPath, updatedManifest); err != nil {
		return err
	}

	// RECOVERY.txt is the operator's executable procedure and is generated
	// from the same metadata. Refresh it only after the identity check above.
	recoveryPath := filepath.Join(meta.BundleRoot, BundleRecoveryFile)
	if err := w.replaceManagedFile(recoveryPath, []byte(renderRecovery(meta))); err != nil {
		return err
	}
	return nil
}

// RebindMetadata updates only generated path fields after the caller has
// independently verified stable volume identity and the existing repository.
// The destination ID, name, backend, and volume-relative path remain immutable.
func (w *FSBundleWriter) RebindMetadata(_ context.Context, meta BundleMetadata) error {
	manifestPath := filepath.Join(meta.BundleRoot, BundleManifestFile)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read destination manifest for path rebind: %w", err)
	}
	var existing bundleManifest
	if err := json.Unmarshal(manifestBytes, &existing); err != nil {
		return fmt.Errorf("decode destination manifest for path rebind: %w", err)
	}
	if existing.Kind != "vrooli.data-backup-manager.destination" || existing.Producer != "data-backup-manager" || existing.DestinationID != meta.DestinationID || existing.DestinationName != meta.Name || existing.Backend != meta.Backend || existing.RelativePath != meta.RelativePath {
		return fmt.Errorf("refusing path rebind: destination manifest identity or relative locator differs")
	}
	uniqueID, err := repositoryUniqueID(meta.RepositoryPath)
	if err != nil {
		return fmt.Errorf("read rebound Kopia repository identity: %w", err)
	}
	if existing.RepositoryUniqueID != "" && existing.RepositoryUniqueID != uniqueID {
		return fmt.Errorf("refusing path rebind: Kopia repository identity differs from the bundle anchor")
	}
	if existing.RepositoryUniqueID == "" && (filepath.Clean(existing.BundleRoot) != filepath.Clean(meta.BundleRoot) || filepath.Clean(existing.RepositoryPath) != filepath.Clean(meta.RepositoryPath)) {
		return fmt.Errorf("refusing relocation: legacy bundle has no repository identity anchor")
	}
	existing.RepositoryUniqueID = uniqueID
	existing.BundleRoot = meta.BundleRoot
	existing.RepositoryPath = meta.RepositoryPath
	updated, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rebound destination manifest: %w", err)
	}
	updated = append(updated, '\n')
	if err := w.replaceManagedFile(manifestPath, updated); err != nil {
		return err
	}
	if err := w.replaceManagedFile(filepath.Join(meta.BundleRoot, BundleReadmeFile), []byte(renderReadme(meta))); err != nil {
		return err
	}
	if err := w.replaceManagedFile(filepath.Join(meta.BundleRoot, BundleRecoveryFile), []byte(renderRecovery(meta))); err != nil {
		return err
	}
	return nil
}

// ValidateRelocation is the read-only half of path recovery. It accepts stale
// absolute path projections only when the stable destination identity and
// relative locator still match the manifest at the newly resolved bundle root.
func (w *FSBundleWriter) ValidateRelocation(_ context.Context, meta BundleMetadata) error {
	manifestPath := filepath.Join(meta.BundleRoot, BundleManifestFile)
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read destination manifest for path validation: %w", err)
	}
	var existing bundleManifest
	if err := json.Unmarshal(manifestBytes, &existing); err != nil {
		return fmt.Errorf("decode destination manifest for path validation: %w", err)
	}
	if existing.Kind != "vrooli.data-backup-manager.destination" || existing.Producer != "data-backup-manager" || existing.DestinationID != meta.DestinationID || existing.DestinationName != meta.Name || existing.Backend != meta.Backend || existing.RelativePath != meta.RelativePath {
		return fmt.Errorf("destination manifest identity or relative locator differs")
	}
	uniqueID, err := repositoryUniqueID(meta.RepositoryPath)
	if err != nil {
		return fmt.Errorf("read Kopia repository identity: %w", err)
	}
	if existing.RepositoryUniqueID != "" && existing.RepositoryUniqueID != uniqueID {
		return fmt.Errorf("Kopia repository identity differs from the bundle anchor")
	}
	if existing.RepositoryUniqueID == "" && (filepath.Clean(existing.BundleRoot) != filepath.Clean(meta.BundleRoot) || filepath.Clean(existing.RepositoryPath) != filepath.Clean(meta.RepositoryPath)) {
		return fmt.Errorf("legacy bundle has no repository identity anchor for relocation")
	}
	return nil
}

// ValidatePendingCreate requires the strict repository identity anchor needed
// before recovering a created-but-not-ready DBM attempt. Unlike generic legacy
// relocation validation, a matching path is insufficient when the manifest
// does not already record Kopia's unique repository ID.
func (w *FSBundleWriter) ValidatePendingCreate(_ context.Context, meta BundleMetadata) error {
	manifestPath := filepath.Join(meta.BundleRoot, BundleManifestFile)
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		return fmt.Errorf("inspect destination manifest for pending-create validation: %w", err)
	}
	if !manifestInfo.Mode().IsRegular() {
		return fmt.Errorf("pending destination manifest is not a regular file")
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read destination manifest for pending-create validation: %w", err)
	}
	var existing bundleManifest
	if err := json.Unmarshal(manifestBytes, &existing); err != nil {
		return fmt.Errorf("decode destination manifest for pending-create validation: %w", err)
	}
	if existing.Kind != "vrooli.data-backup-manager.destination" || existing.Producer != "data-backup-manager" || existing.DestinationID != meta.DestinationID || existing.DestinationName != meta.Name || existing.Backend != meta.Backend || existing.RelativePath != meta.RelativePath {
		return fmt.Errorf("pending destination manifest identity or relative locator differs")
	}
	if strings.TrimSpace(existing.RepositoryUniqueID) == "" {
		return fmt.Errorf("pending destination manifest has no Kopia repository identity anchor")
	}
	uniqueID, err := repositoryUniqueID(meta.RepositoryPath)
	if err != nil {
		return fmt.Errorf("read pending Kopia repository identity: %w", err)
	}
	if existing.RepositoryUniqueID != uniqueID {
		return fmt.Errorf("pending Kopia repository identity differs from the manifest anchor")
	}
	createdAt := time.Time{}
	if existing.CreatedAt != "(unknown)" {
		createdAt, err = time.Parse(time.RFC3339Nano, existing.CreatedAt)
		if err != nil {
			return fmt.Errorf("decode pending destination creation time: %w", err)
		}
	}
	anchored := BundleMetadata{
		DestinationID: existing.DestinationID, Name: existing.DestinationName, Backend: existing.Backend,
		BundleRoot: existing.BundleRoot, RelativePath: existing.RelativePath, RepositoryPath: existing.RepositoryPath,
		RepositoryUniqueID: existing.RepositoryUniqueID, EncryptionAlgorithm: existing.EncryptionAlgorithm,
		SecretRef: existing.SecretRef, CreatedAt: createdAt,
	}
	for name, expected := range map[string]string{BundleReadmeFile: renderReadme(anchored), BundleRecoveryFile: renderRecovery(anchored)} {
		path := filepath.Join(meta.BundleRoot, name)
		info, statErr := os.Lstat(path)
		if os.IsNotExist(statErr) {
			continue // Missing generated files can be finalized after identity proof.
		}
		if statErr != nil {
			return fmt.Errorf("inspect generated pending metadata %q: %w", path, statErr)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("pending destination file %q is not a regular file", name)
		}
		actual, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read generated pending metadata %q: %w", path, readErr)
		}
		if string(actual) != expected {
			return fmt.Errorf("pending destination file %q differs from its manifest-anchored generated content", name)
		}
	}
	return nil
}

// ReadExisting verifies the generated manifest and Kopia repository marker at
// an existing bundle root. It is read-only and rejects legacy bundles without
// the repository identity anchor needed to distinguish a moved repository from
// a different one at the same path.
func (w *FSBundleWriter) ReadExisting(_ context.Context, bundleRoot string) (ExistingBundle, error) {
	bundleRoot, err := filepath.Abs(filepath.Clean(strings.TrimSpace(bundleRoot)))
	if err != nil || bundleRoot == "." {
		return ExistingBundle{}, fmt.Errorf("resolve existing destination bundle root: %w", err)
	}
	rootInfo, err := os.Lstat(bundleRoot)
	if err != nil {
		return ExistingBundle{}, fmt.Errorf("inspect existing destination bundle root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return ExistingBundle{}, fmt.Errorf("existing destination bundle root must be a real directory")
	}
	manifestPath := filepath.Join(bundleRoot, BundleManifestFile)
	manifestInfo, err := os.Lstat(manifestPath)
	if err != nil {
		return ExistingBundle{}, fmt.Errorf("inspect existing destination manifest: %w", err)
	}
	if !manifestInfo.Mode().IsRegular() {
		return ExistingBundle{}, fmt.Errorf("existing destination manifest must be a regular file")
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return ExistingBundle{}, fmt.Errorf("read existing destination manifest: %w", err)
	}
	var manifest bundleManifest
	decoder := json.NewDecoder(strings.NewReader(string(manifestBytes)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return ExistingBundle{}, fmt.Errorf("decode existing destination manifest: %w", err)
	}
	if manifest.SchemaVersion != BundleSchemaVersion || manifest.Kind != "vrooli.data-backup-manager.destination" || manifest.Producer != "data-backup-manager" || manifest.Backend != string(BackendFilesystem) {
		return ExistingBundle{}, fmt.Errorf("existing destination bundle is not a supported DBM filesystem bundle")
	}
	if !ValidSlug(manifest.DestinationName) || strings.TrimSpace(manifest.DestinationID) == "" || strings.TrimSpace(manifest.RelativePath) == "" || strings.TrimSpace(manifest.RepositoryUniqueID) == "" {
		return ExistingBundle{}, fmt.Errorf("existing destination bundle is missing its stable identity anchor")
	}
	repositoryPath := RepositoryPathFor(bundleRoot, manifest.DestinationName)
	uniqueID, err := repositoryUniqueID(repositoryPath)
	if err != nil {
		return ExistingBundle{}, fmt.Errorf("inspect existing Kopia repository: %w", err)
	}
	if uniqueID != manifest.RepositoryUniqueID {
		return ExistingBundle{}, fmt.Errorf("existing Kopia repository does not match the bundle identity anchor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, manifest.CreatedAt)
	if err != nil {
		return ExistingBundle{}, fmt.Errorf("existing destination created_at is invalid: %w", err)
	}
	meta := BundleMetadata{
		DestinationID: manifest.DestinationID, Name: manifest.DestinationName,
		Backend: manifest.Backend, BundleRoot: bundleRoot,
		RelativePath: manifest.RelativePath, RepositoryPath: repositoryPath,
		RepositoryUniqueID: uniqueID, EncryptionAlgorithm: manifest.EncryptionAlgorithm,
		SecretRef: manifest.SecretRef, CreatedAt: createdAt, Host: manifest.Host, User: manifest.User,
	}
	return ExistingBundle{Metadata: meta, RepositoryUniqueID: uniqueID}, nil
}

func repositoryUniqueID(repositoryPath string) (string, error) {
	markers := []string{filepath.Join(repositoryPath, "kopia.repository.f"), filepath.Join(repositoryPath, "kopia.repository")}
	var marker string
	for _, candidate := range markers {
		if info, err := os.Lstat(candidate); err == nil {
			if !info.Mode().IsRegular() {
				return "", fmt.Errorf("repository marker %q is not a regular file", candidate)
			}
			if marker != "" {
				return "", fmt.Errorf("multiple Kopia repository markers are present")
			}
			marker = candidate
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect %q: %w", candidate, err)
		}
	}
	if marker == "" {
		return "", fmt.Errorf("repository marker is missing")
	}
	contents, err := os.ReadFile(marker)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", marker, err)
	}
	var identity struct {
		UniqueID string `json:"uniqueID"`
	}
	if err := json.Unmarshal(contents, &identity); err != nil {
		return "", fmt.Errorf("decode %q: %w", marker, err)
	}
	if strings.TrimSpace(identity.UniqueID) == "" {
		return "", fmt.Errorf("repository marker has no uniqueID")
	}
	return identity.UniqueID, nil
}

func validateManifestIdentity(existing bundleManifest, meta BundleMetadata) error {
	if existing.Kind != "vrooli.data-backup-manager.destination" || existing.Producer != "data-backup-manager" {
		return fmt.Errorf("refusing metadata refresh: destination manifest is not owned by data-backup-manager")
	}
	want := map[string]string{
		"destination_id":   meta.DestinationID,
		"destination_name": meta.Name,
		"backend":          meta.Backend,
		"bundle_root":      meta.BundleRoot,
		"repository_path":  meta.RepositoryPath,
	}
	got := map[string]string{
		"destination_id":   existing.DestinationID,
		"destination_name": existing.DestinationName,
		"backend":          existing.Backend,
		"bundle_root":      existing.BundleRoot,
		"repository_path":  existing.RepositoryPath,
	}
	for field, expected := range want {
		if got[field] != expected {
			return fmt.Errorf("refusing metadata refresh: manifest %s %q does not match destination %q", field, got[field], expected)
		}
	}
	return nil
}

func (w *FSBundleWriter) replaceManagedFile(path string, content []byte) error {
	mode := w.filePerm()
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect managed metadata %q: %w", path, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vrooli-metadata-*")
	if err != nil {
		return fmt.Errorf("create managed metadata temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set managed metadata permissions: %w", err)
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write managed metadata: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync managed metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close managed metadata: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace managed metadata %q: %w", path, err)
	}
	return nil
}

// writeIfAbsentOrMatching writes content when the file is absent, no-ops when
// the existing content is byte-identical, and errors when it differs — so a
// create retry is idempotent but unknown pre-existing data is never clobbered.
func (w *FSBundleWriter) writeIfAbsentOrMatching(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	switch {
	case err == nil:
		if string(existing) == string(content) {
			return nil
		}
		return ErrInvalidDestination{
			Field:  "location",
			Reason: fmt.Sprintf("%s already exists with different content; refusing to overwrite", filepath.Base(path)),
		}
	case os.IsNotExist(err):
		if writeErr := os.WriteFile(path, content, w.filePerm()); writeErr != nil {
			return fmt.Errorf("write %q: %w", path, writeErr)
		}
		return nil
	default:
		return fmt.Errorf("stat %q: %w", path, err)
	}
}

func renderReadme(meta BundleMetadata) string {
	return fmt.Sprintf(`Vrooli Data Backup Manager — Backup Destination
================================================

This folder is a Vrooli Data Backup Manager backup destination.

  Destination name : %s
  Backend          : %s
  Created          : %s
  Volume-relative path: %s

The actual backups live in an ENCRYPTED kopia repository under:

  %s

DO NOT edit, move, or delete files inside that repository directory by hand.
Doing so can corrupt every backup it holds.

To restore, use the Data Backup Manager app, or plain kopia plus this
destination's passphrase. See %s in this folder for standalone recovery steps.

Non-secret metadata about this destination is in %s. No passphrase, token, or
credential is stored anywhere in this folder.
`,
		meta.Name,
		meta.Backend,
		formatCreated(meta.CreatedAt),
		emptyDash(meta.RelativePath),
		meta.RepositoryPath,
		BundleRecoveryFile,
		BundleManifestFile,
	)
}

func renderRecovery(meta BundleMetadata) string {
	secretRef := meta.SecretRef
	if secretRef == "" {
		// #nosec G101 -- human-readable fallback message for the recovery doc, not a credential.
		secretRef = "(see the Vrooli credential authority; no credential reference was recorded)"
	}
	return fmt.Sprintf(`Vrooli Data Backup Manager — Standalone Recovery
================================================

You can restore this destination WITHOUT Vrooli, using plain kopia and the
repository passphrase. The repository is a vanilla kopia repository; Vrooli adds
no proprietary layer.

What you need:
  1. The kopia CLI (https://kopia.io), version 0.23.0 or newer.
  2. The repository passphrase. It is NOT stored on this drive. It lives in the
     Vrooli credential authority under:

         %s

     On a replacement host, import it with:
       printf '%%s' "$PASSPHRASE" | vrooli credentials provision --identity "%s" --field "repository-passphrase"

The destination path inside the volume is: %s
After a replug, confirm the volume UUID/serial before resolving that path at
the current mountpoint.

Steps (filesystem repository):
  1. Connect to the repository:
       kopia repository connect filesystem --path "%s"
     (kopia will prompt for the passphrase.)
  2. List snapshots:
       kopia snapshot list --all
  3. Restore a snapshot into a NEW empty directory:
       kopia snapshot restore <snapshot-id> /path/to/empty/restore/dir

Never restore over an existing non-empty directory you care about.

Encryption algorithm reported at creation: %s
`,
		secretRef,
		credentialIdentity(secretRef),
		emptyDash(meta.RelativePath),
		meta.RepositoryPath,
		emptyDash(meta.EncryptionAlgorithm),
	)
}

func renderManifest(meta BundleMetadata) ([]byte, error) {
	m := bundleManifest{
		SchemaVersion:       BundleSchemaVersion,
		Kind:                "vrooli.data-backup-manager.destination",
		Producer:            "data-backup-manager",
		DestinationID:       meta.DestinationID,
		DestinationName:     meta.Name,
		Backend:             meta.Backend,
		BundleRoot:          meta.BundleRoot,
		RelativePath:        meta.RelativePath,
		RepositoryPath:      meta.RepositoryPath,
		RepositoryUniqueID:  meta.RepositoryUniqueID,
		EncryptionAlgorithm: meta.EncryptionAlgorithm,
		SecretRef:           meta.SecretRef,
		CreatedAt:           formatCreated(meta.CreatedAt),
		Host:                meta.Host,
		User:                meta.User,
		RestoreHints: []string{
			fmt.Sprintf("kopia repository connect filesystem --path %q", meta.RepositoryPath),
			"kopia snapshot list --all",
			"kopia snapshot restore <snapshot-id> <empty-target-dir>",
		},
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render manifest: %w", err)
	}
	return append(b, '\n'), nil
}

func formatCreated(t time.Time) string {
	if t.IsZero() {
		return "(unknown)"
	}
	return t.UTC().Format(time.RFC3339)
}

func emptyDash(s string) string {
	if s == "" {
		return "(unknown)"
	}
	return s
}

func credentialIdentity(ref string) string {
	identity, _, ok := strings.Cut(strings.TrimSpace(ref), ":")
	if !ok || identity == "" {
		return "<identity>"
	}
	return identity
}
