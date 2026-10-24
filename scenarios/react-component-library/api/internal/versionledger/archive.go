package versionledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const archiveSchemaVersion = 1

var (
	archiveTables     = []string{"component_versions", "component_version_files", "version_ledger"}
	archiveIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

type archiveTable struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

type archiveBody struct {
	SchemaVersion int                     `json:"schema_version"`
	Tables        map[string]archiveTable `json:"tables"`
}

type archiveEnvelope struct {
	archiveBody
	Checksum string `json:"checksum"`
}

type ArchiveSummary struct {
	Path          string         `json:"path"`
	SchemaVersion int            `json:"schema_version"`
	RowCounts     map[string]int `json:"row_counts"`
	Checksum      string         `json:"checksum"`
}

type DoctorIssue struct {
	LibraryID      string                `json:"library_id"`
	Version        string                `json:"version"`
	Path           string                `json:"path"`
	Expected       string                `json:"expected_sha256"`
	Actual         string                `json:"actual_sha256"`
	Reason         string                `json:"reason"`
	LedgerMetadata *DoctorLedgerMetadata `json:"ledger_metadata,omitempty"`
}

type DoctorLedgerMetadata struct {
	CreatedAt      string `json:"created_at"`
	ReleasedAt     string `json:"released_at"`
	RetiredAt      string `json:"retired_at"`
	LifecycleState string `json:"lifecycle_state"`
	FileCount      int    `json:"file_count"`
}

func (r *Repository) ExportArchive(ctx context.Context, path string) (ArchiveSummary, error) {
	body := archiveBody{SchemaVersion: archiveSchemaVersion, Tables: map[string]archiveTable{}}
	for _, table := range archiveTables {
		rows, err := r.exportTable(ctx, table)
		if err != nil {
			return ArchiveSummary{}, err
		}
		body.Tables[table] = rows
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return ArchiveSummary{}, fmt.Errorf("encode ledger archive: %w", err)
	}
	envelope := archiveEnvelope{archiveBody: body, Checksum: digestArchive(encoded)}
	output, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return ArchiveSummary{}, fmt.Errorf("encode ledger archive envelope: %w", err)
	}
	output = append(output, '\n')
	absolute, err := filepath.Abs(path)
	if err != nil {
		return ArchiveSummary{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		return ArchiveSummary{}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(absolute), ".rcl-ledger-archive-")
	if err != nil {
		return ArchiveSummary{}, err
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if _, err := temporary.Write(output); err != nil {
		_ = temporary.Close()
		return ArchiveSummary{}, err
	}
	if err := temporary.Close(); err != nil {
		return ArchiveSummary{}, err
	}
	if err := os.Rename(temporaryName, absolute); err != nil {
		return ArchiveSummary{}, err
	}
	return archiveSummary(absolute, envelope), nil
}

func (r *Repository) exportTable(ctx context.Context, table string) (archiveTable, error) {
	if !archiveIdentifier.MatchString(table) {
		return archiveTable{}, fmt.Errorf("invalid archive table %q", table)
	}
	rows, err := r.db.QueryContext(ctx, `SELECT * FROM `+table)
	if err != nil {
		return archiveTable{}, fmt.Errorf("export %s: %w", table, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return archiveTable{}, err
	}
	archive := archiveTable{Columns: columns}
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return archiveTable{}, err
		}
		for i, value := range values {
			if bytes, ok := value.([]byte); ok {
				values[i] = string(bytes)
			}
		}
		archive.Rows = append(archive.Rows, values)
	}
	return archive, rows.Err()
}

func (r *Repository) ImportArchive(ctx context.Context, path string, overwrite bool) (ArchiveSummary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ArchiveSummary{}, err
	}
	var envelope archiveEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ArchiveSummary{}, fmt.Errorf("decode ledger archive: %w", err)
	}
	body, err := json.Marshal(envelope.archiveBody)
	if err != nil || digestArchive(body) != envelope.Checksum {
		return ArchiveSummary{}, fmt.Errorf("ledger archive checksum mismatch")
	}
	if envelope.SchemaVersion != archiveSchemaVersion {
		return ArchiveSummary{}, fmt.Errorf("unsupported ledger archive schema version %d", envelope.SchemaVersion)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ArchiveSummary{}, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, table := range archiveTables {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&count); err != nil {
			return ArchiveSummary{}, err
		}
		if count > 0 && !overwrite {
			return ArchiveSummary{}, fmt.Errorf("refusing archive import into non-empty table %s; pass overwrite", table)
		}
	}
	if overwrite {
		for i := len(archiveTables) - 1; i >= 0; i-- {
			if _, err := tx.ExecContext(ctx, `DELETE FROM `+archiveTables[i]); err != nil {
				return ArchiveSummary{}, err
			}
		}
	}
	for _, table := range archiveTables {
		archive := envelope.Tables[table]
		for _, column := range archive.Columns {
			if !archiveIdentifier.MatchString(column) {
				return ArchiveSummary{}, fmt.Errorf("invalid archive column %q", column)
			}
		}
		quoted := make([]string, len(archive.Columns))
		copy(quoted, archive.Columns)
		placeholders := strings.TrimRight(strings.Repeat("?,", len(quoted)), ",")
		query := `INSERT INTO ` + table + ` (` + strings.Join(quoted, ",") + `) VALUES (` + placeholders + `)`
		for _, row := range archive.Rows {
			if len(row) != len(quoted) {
				return ArchiveSummary{}, fmt.Errorf("archive row width mismatch for %s", table)
			}
			if _, err := tx.ExecContext(ctx, query, row...); err != nil {
				return ArchiveSummary{}, fmt.Errorf("import %s: %w", table, err)
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return ArchiveSummary{}, err
	}
	absolute, _ := filepath.Abs(path)
	return archiveSummary(absolute, envelope), nil
}

// PublishedReleaseBundle is the deliberately narrow recovery artifact for
// the Button 2.2.4 release. The revision and hashes are pinned here because
// the ledger has no prior row from which to establish identity.
type PublishedReleaseBundle struct {
	SchemaVersion  int                          `json:"schema_version"`
	LibraryID      string                       `json:"library_id"`
	Version        string                       `json:"version"`
	Status         string                       `json:"status"`
	SourceRevision string                       `json:"source_revision"`
	PublishedAt    string                       `json:"published_at"`
	Files          []PublishedReleaseBundleFile `json:"files"`
}

type PublishedReleaseBundleFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256"`
	Entry   bool   `json:"entry"`
}

const (
	button224LibraryID   = "react-component-library:Button"
	button224Version     = "2.2.4"
	button224Revision    = "50f54d5a89ebd8da93099475dbd1263eeb01af0a"
	button224PublishedAt = "2026-08-31T05:19:38Z"
	button224EntryHash   = "251d5b0e187b741451bce8910b9578905a8d9d07d04932dbbd32877c52b002dd"
	button224StoryHash   = "671eb434656cce4baaef21c248fbc19e96866eda2c332590d43ae476d8f965ba"
)

// RecoverButton224PublishedRelease validates the complete attested bundle before any
// database mutation, inserts only the missing release and mirror rows, and
// leaves filesystem materialization to the canonical components materializer.
func (r *Repository) RecoverButton224PublishedRelease(ctx context.Context, path string) (ArchiveSummary, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ArchiveSummary{}, "", err
	}
	var bundle PublishedReleaseBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return ArchiveSummary{}, "", fmt.Errorf("decode published release bundle: %w", err)
	}
	return r.restoreValidatedRelease(ctx, path, data, bundle, button224Attestation())
}

func (r *Repository) restoreValidatedRelease(ctx context.Context, path string, data []byte, bundle PublishedReleaseBundle, attestation releaseAttestation) (ArchiveSummary, string, error) {
	files, err := validatePublishedReleaseBundle(bundle, attestation)
	if err != nil {
		return ArchiveSummary{}, "", err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ArchiveSummary{}, "", err
	}
	defer func() { _ = tx.Rollback() }()
	var componentID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM components WHERE library_id = ?`, attestation.libraryID).Scan(&componentID); err != nil {
		return ArchiveSummary{}, "", fmt.Errorf("find canonical component %s: %w", attestation.libraryID, err)
	}
	var versions, ledger int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM component_versions WHERE library_id = ? AND version = ?`, attestation.libraryID, attestation.version).Scan(&versions); err != nil {
		return ArchiveSummary{}, "", err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM version_ledger WHERE library_id = ? AND version = ?`, attestation.libraryID, attestation.version).Scan(&ledger); err != nil {
		return ArchiveSummary{}, "", err
	}
	if versions != 0 {
		return ArchiveSummary{}, "", fmt.Errorf("refusing recovery: %s@%s is already present", attestation.libraryID, attestation.version)
	}
	if ledger > 1 {
		return ArchiveSummary{}, "", fmt.Errorf("refusing recovery: ambiguous ledger rows for %s@%s", attestation.libraryID, attestation.version)
	}
	if ledger == 1 {
		var libraryID, version, createdAt, releasedAt, retiredAt, lifecycle string
		var fileCount int
		if err := tx.QueryRowContext(ctx, `SELECT library_id, version, created_at, released_at, retired_at, lifecycle_state, file_count FROM version_ledger WHERE library_id = ? AND version = ?`, attestation.libraryID, attestation.version).Scan(&libraryID, &version, &createdAt, &releasedAt, &retiredAt, &lifecycle, &fileCount); err != nil {
			return ArchiveSummary{}, "", fmt.Errorf("read orphan release ledger row: %w", err)
		}
		if libraryID != attestation.libraryID || version != attestation.version || lifecycle != "released" || createdAt != attestation.publishedAt || releasedAt != attestation.publishedAt || retiredAt != "" || fileCount != len(files) {
			return ArchiveSummary{}, "", fmt.Errorf("refusing recovery: orphan ledger metadata does not match attested release %s@%s", attestation.libraryID, attestation.version)
		}
	}
	entry := files["Button.tsx"]
	versionID := uuid.NewString()
	entryPath := filepath.ToSlash(filepath.Join("components", "Button", "versions", attestation.version, "Button.tsx"))
	if _, err := tx.ExecContext(ctx, `INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence) VALUES (?, ?, ?, ?, 'released', ?, ?, ?, '', ?, ?, ?, 'evicted')`, versionID, componentID, attestation.libraryID, attestation.version, entryPath, entry.Content, entry.SHA256, time.Now().UTC().Format(time.RFC3339Nano), bundle.PublishedAt, bundle.PublishedAt); err != nil {
		return ArchiveSummary{}, "", fmt.Errorf("insert recovered version: %w", err)
	}
	for _, name := range []string{"Button.tsx", "story.tsx"} {
		file := files[name]
		isEntry := 0
		if file.Entry {
			isEntry = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry, slot) VALUES (?, ?, ?, ?, ?, '')`, versionID, file.Path, file.Content, file.SHA256, isEntry); err != nil {
			return ArchiveSummary{}, "", fmt.Errorf("insert recovered file %s: %w", name, err)
		}
	}
	if ledger == 0 {
		if _, err := tx.ExecContext(ctx, `INSERT INTO version_ledger (library_id, version, created_at, released_at, lifecycle_state, file_count) VALUES (?, ?, ?, ?, 'released', 2)`, attestation.libraryID, attestation.version, bundle.PublishedAt, bundle.PublishedAt); err != nil {
			return ArchiveSummary{}, "", fmt.Errorf("insert recovered ledger row: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return ArchiveSummary{}, "", err
	}
	ledgerRowsAdded := 0
	if ledger == 0 {
		ledgerRowsAdded = 1
	}
	return ArchiveSummary{Path: path, SchemaVersion: bundle.SchemaVersion, RowCounts: map[string]int{"component_versions": 1, "component_version_files": 2, "version_ledger": ledgerRowsAdded}, Checksum: digestArchive(data)}, componentID, nil
}

type releaseAttestation struct {
	libraryID, version, revision, publishedAt string
	hashes                                    map[string]string
}

func button224Attestation() releaseAttestation {
	return releaseAttestation{libraryID: button224LibraryID, version: button224Version, revision: button224Revision, publishedAt: button224PublishedAt, hashes: map[string]string{"Button.tsx": button224EntryHash, "story.tsx": button224StoryHash}}
}

func validatePublishedReleaseBundle(bundle PublishedReleaseBundle, attestation releaseAttestation) (map[string]PublishedReleaseBundleFile, error) {
	if bundle.SchemaVersion != 1 || bundle.LibraryID != attestation.libraryID || bundle.Version != attestation.version || bundle.Status != "released" {
		return nil, fmt.Errorf("bundle identity/status must be released %s@%s (schema 1)", attestation.libraryID, attestation.version)
	}
	if bundle.SourceRevision != attestation.revision || bundle.PublishedAt != attestation.publishedAt {
		return nil, fmt.Errorf("bundle source revision/publication time does not match the attested release")
	}
	if len(bundle.Files) != 2 {
		return nil, fmt.Errorf("bundle must contain exactly Button.tsx and story.tsx")
	}
	byPath := make(map[string]PublishedReleaseBundleFile, len(bundle.Files))
	for _, file := range bundle.Files {
		clean := filepath.Clean(filepath.FromSlash(file.Path))
		if file.Path == "" || strings.Contains(file.Path, `\`) || !filepath.IsLocal(file.Path) || clean == "." || filepath.ToSlash(clean) != file.Path {
			return nil, fmt.Errorf("unsafe bundle file path %q", file.Path)
		}
		if _, exists := byPath[file.Path]; exists {
			return nil, fmt.Errorf("duplicate bundle file path %q", file.Path)
		}
		want, ok := attestation.hashes[file.Path]
		if !ok {
			return nil, fmt.Errorf("unexpected bundle file path %q", file.Path)
		}
		actual := fmt.Sprintf("%x", sha256.Sum256([]byte(file.Content)))
		if file.SHA256 != want || actual != file.SHA256 {
			return nil, fmt.Errorf("bundle hash mismatch for %s", file.Path)
		}
		if (file.Path == "Button.tsx") != file.Entry {
			return nil, fmt.Errorf("bundle entry marker mismatch for %s", file.Path)
		}
		byPath[file.Path] = file
	}
	if _, ok := byPath["Button.tsx"]; !ok {
		return nil, fmt.Errorf("bundle is missing Button.tsx")
	}
	if _, ok := byPath["story.tsx"]; !ok {
		return nil, fmt.Errorf("bundle is missing story.tsx")
	}
	return byPath, nil
}

func (r *Repository) Doctor(ctx context.Context) ([]DoctorIssue, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, library_id, version, source_path FROM component_versions WHERE presence = 'evicted' ORDER BY library_id, version`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type evictedVersion struct {
		id, libraryID, version, sourcePath string
	}
	var evicted []evictedVersion
	for rows.Next() {
		var version evictedVersion
		if err := rows.Scan(&version.id, &version.libraryID, &version.version, &version.sourcePath); err != nil {
			rows.Close()
			return nil, err
		}
		evicted = append(evicted, version)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var issues []DoctorIssue
	for _, version := range evicted {
		mirror, err := r.db.QueryContext(ctx, `SELECT path, content, content_sha256 FROM component_version_files WHERE version_id = ? ORDER BY path`, version.id)
		if err != nil {
			return nil, err
		}
		checked := 0
		for mirror.Next() {
			var path, content, expected string
			if err := mirror.Scan(&path, &content, &expected); err != nil {
				mirror.Close()
				return nil, err
			}
			actual := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
			if actual != expected {
				issues = append(issues, DoctorIssue{LibraryID: version.libraryID, Version: version.version, Path: path, Expected: expected, Actual: actual, Reason: "mirror content hash mismatch"})
			}
			checked++
		}
		if err := mirror.Err(); err != nil {
			mirror.Close()
			return nil, err
		}
		mirror.Close()
		if checked == 0 {
			issues = append(issues, DoctorIssue{LibraryID: version.libraryID, Version: version.version, Path: filepath.ToSlash(filepath.Dir(version.sourcePath)), Reason: "evicted version has no file mirror"})
		}
	}
	ledgerRows, err := r.db.QueryContext(ctx, `
SELECT ledger.library_id, ledger.version
     , ledger.created_at, ledger.released_at, ledger.retired_at
     , ledger.lifecycle_state, ledger.file_count
FROM version_ledger AS ledger
LEFT JOIN component_versions AS component
  ON component.library_id = ledger.library_id AND component.version = ledger.version
WHERE component.library_id IS NULL
ORDER BY ledger.library_id, ledger.version`)
	if err != nil {
		return nil, err
	}
	for ledgerRows.Next() {
		var libraryID, version string
		metadata := &DoctorLedgerMetadata{}
		if err := ledgerRows.Scan(&libraryID, &version, &metadata.CreatedAt, &metadata.ReleasedAt, &metadata.RetiredAt, &metadata.LifecycleState, &metadata.FileCount); err != nil {
			ledgerRows.Close()
			return nil, err
		}
		issues = append(issues, DoctorIssue{
			LibraryID:      libraryID,
			Version:        version,
			Reason:         "version ledger row has no component version",
			LedgerMetadata: metadata,
		})
	}
	if err := ledgerRows.Err(); err != nil {
		ledgerRows.Close()
		return nil, err
	}
	if err := ledgerRows.Close(); err != nil {
		return nil, err
	}
	return issues, nil
}

func archiveSummary(path string, envelope archiveEnvelope) ArchiveSummary {
	counts := make(map[string]int, len(envelope.Tables))
	for table, value := range envelope.Tables {
		counts[table] = len(value.Rows)
	}
	return ArchiveSummary{Path: path, SchemaVersion: envelope.SchemaVersion, RowCounts: counts, Checksum: envelope.Checksum}
}

func digestArchive(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
