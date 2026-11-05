package gates

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

type queryContexter interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func validateReleasedVersionImmutableWithDB(ctx context.Context, root string, db queryContexter) (Result, error) {
	if db == nil {
		return Result{}, fmt.Errorf("component index database is not configured")
	}
	rows, err := db.QueryContext(ctx, `SELECT v.status, v.source_path, v.content_sha256 FROM component_versions v WHERE v.status = 'released'`)
	if err != nil {
		return validateReleasedVersionHashLedger(root)
	}
	defer rows.Close()
	result := Result{}
	for rows.Next() {
		var status, sourcePath, recorded string
		if err := rows.Scan(&status, &sourcePath, &recorded); err != nil {
			return Result{}, err
		}
		if status != "released" || filepath.Base(sourcePath) == "dependencies.json" {
			continue
		}
		path := filepath.Join(root, "scenarios", "react-component-library", "library", sourcePath)
		if isSupplementalImplementation(path) {
			continue
		}
		result.Inspected++
		raw, err := os.ReadFile(path)
		if err != nil {
			result.Findings = append(result.Findings, Finding{Code: "catalog.released_version_immutable", AssetID: implementationName(path), File: filepath.Join("library", sourcePath), Message: fmt.Sprintf("released source cannot be read: %v", err), Remediation: "Restore the released source or remove its database row through the governed retirement path.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning"})
			continue
		}
		sum := sha256.Sum256(raw)
		current := hex.EncodeToString(sum[:])
		withoutTerminalLF := sha256.Sum256(bytes.TrimSuffix(raw, []byte("\n")))
		if recorded != "" && recorded != current && recorded != hex.EncodeToString(withoutTerminalLF[:]) {
			result.Findings = append(result.Findings, Finding{Code: "catalog.released_version_immutable", AssetID: implementationName(path), File: filepath.Join("library", sourcePath), Message: fmt.Sprintf("released source changed after release: recorded %s, current %s", shortHash(recorded), shortHash(current)), Remediation: "Revert the released source and publish the change as a new version.", DocsRef: "docs/concepts/ARCHITECTURE.md#versioning"})
		}
	}
	if err := rows.Err(); err != nil {
		return Result{}, err
	}
	if result.Inspected == 0 {
		return validateReleasedVersionHashLedger(root)
	}
	return nonEmpty(result, "released-version-immutable"), nil
}

type releasedVersionHashLedger struct {
	SchemaVersion int `json:"schemaVersion"`
	Entries       []struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"entries"`
}
