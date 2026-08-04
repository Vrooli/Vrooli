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
}

// BundleMetadata is the non-secret information rendered into the bundle files.
// It never carries a passphrase, token, or other secret value — only a
// credential-authority identity/field reference.
type BundleMetadata struct {
	DestinationID       string
	Name                string
	Backend             string
	BundleRoot          string
	RepositoryPath      string
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
	RepositoryPath      string   `json:"repository_path"`
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
	if _, err := os.Stat(filepath.Join(bundleRoot, "kopia.repository")); err == nil {
		return ErrInvalidDestination{
			Field:  "location",
			Reason: "bundle root already contains a kopia repository at its root; choose an empty folder or a dedicated subdirectory",
		}
	}
	if err := os.MkdirAll(bundleRoot, w.dirPerm()); err != nil {
		return fmt.Errorf("create bundle root %q: %w", bundleRoot, err)
	}
	if err := os.MkdirAll(repositoryPath, w.dirPerm()); err != nil {
		return fmt.Errorf("create repository dir %q: %w", repositoryPath, err)
	}
	return nil
}

func (w *FSBundleWriter) WriteMetadata(_ context.Context, meta BundleMetadata) error {
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
		RepositoryPath:      meta.RepositoryPath,
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
