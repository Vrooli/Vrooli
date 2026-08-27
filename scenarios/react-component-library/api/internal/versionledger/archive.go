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
)

const archiveSchemaVersion = 1

var archiveTables = []string{"component_versions", "component_version_files", "version_ledger"}
var archiveIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

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
	LibraryID string `json:"library_id"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	Expected  string `json:"expected_sha256"`
	Actual    string `json:"actual_sha256"`
	Reason    string `json:"reason"`
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
		for i, column := range archive.Columns {
			quoted[i] = column
		}
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

func (r *Repository) Doctor(ctx context.Context) ([]DoctorIssue, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, library_id, version, source_path FROM component_versions WHERE presence = 'evicted' ORDER BY library_id, version`)
	if err != nil {
		return nil, err
	}
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
