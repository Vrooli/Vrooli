package versionledger

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"react-component-library/internal/components"
)

// LedgerAuditEntry is the verification result for one authored release file.
type LedgerAuditEntry struct {
	Path   string `json:"path"`
	Asset  string `json:"asset"`
	Status string `json:"status"`
}

// LedgerAudit is a read-only report used before repairing or pruning the
// release hash ledger. Derived dependencies.json rows are excluded by policy.
type LedgerAudit struct {
	Matching []LedgerAuditEntry `json:"matching"`
	Mutated  []LedgerAuditEntry `json:"mutated"`
	Missing  []LedgerAuditEntry `json:"missing"`
}

type LedgerRepair struct {
	Pruned          []string           `json:"pruned"`
	Rerecorded      []string           `json:"rerecorded"`
	AcceptedCurrent []string           `json:"acceptedCurrent"`
	Unresolved      []LedgerAuditEntry `json:"unresolved"`
	Applied         bool               `json:"applied"`
}

// LedgerRepairOptions controls explicitly authorized ledger repairs.
type LedgerRepairOptions struct {
	// AcceptCurrent records the current authored bytes when they differ from
	// the ledger and durable mirror. Callers must supply independent provenance
	// evidence before enabling this option.
	AcceptCurrent bool
	// PruneMissing removes authored-source rows whose files were deliberately
	// removed after an independent claim-retirement review. It is opt-in so a
	// missing file remains an unresolved defect during ordinary reconciliation.
	PruneMissing bool
}

// AuditReleaseHashLedger checks authored-source entries without changing the
// ledger or the working tree. root is the repository root.
func AuditReleaseHashLedger(root string) (LedgerAudit, error) {
	return auditReleaseHashLedger(root, nil)
}

// AuditReleaseHashLedgerWithDatabase also verifies intentionally evicted
// released files against their durable SQLite mirror. An evicted version is
// absent from the authored tree by design, so treating its source path as an
// ordinary missing file would turn a successful retention operation into a
// false ledger defect.
func AuditReleaseHashLedgerWithDatabase(root string, db *sql.DB) (LedgerAudit, error) {
	return auditReleaseHashLedger(root, db)
}

func auditReleaseHashLedger(root string, db *sql.DB) (LedgerAudit, error) {
	ledgerPath := filepath.Join(root, "scenarios", "react-component-library", "library", "released-version-hashes.json")
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		return LedgerAudit{}, fmt.Errorf("read release hash ledger: %w", err)
	}
	var ledger struct {
		Entries []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		return LedgerAudit{}, fmt.Errorf("decode release hash ledger: %w", err)
	}
	var audit LedgerAudit
	for _, entry := range ledger.Entries {
		if !components.IsAuthoredReleaseFile(entry.Path) {
			continue
		}
		result := LedgerAuditEntry{Path: filepath.ToSlash(entry.Path), Asset: ledgerAsset(entry.Path)}
		path := filepath.Join(root, "scenarios", "react-component-library", "library", filepath.FromSlash(entry.Path))
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			if db != nil {
				status, found, statusErr := ledgerVersionStatus(db, entry.Path)
				if statusErr != nil {
					return LedgerAudit{}, statusErr
				}
				if found && (strings.EqualFold(status, "released") || strings.EqualFold(status, "retired") || strings.EqualFold(status, "archived")) {
					presence, presenceErr := ledgerVersionPresence(db, entry.Path)
					if presenceErr != nil {
						return LedgerAudit{}, presenceErr
					}
					if strings.EqualFold(presence, "evicted") {
						mirror, mirrorFound, mirrorErr := ledgerMirror(db, entry.Path)
						if mirrorErr != nil {
							return LedgerAudit{}, mirrorErr
						}
						if mirrorFound {
							sum := sha256.Sum256(mirror)
							if hex.EncodeToString(sum[:]) == entry.SHA256 {
								result.Status = "matching"
								audit.Matching = append(audit.Matching, result)
								continue
							}
							result.Status = "mutated"
							audit.Mutated = append(audit.Mutated, result)
							continue
						}
					}
				}
			}
			result.Status = "missing"
			audit.Missing = append(audit.Missing, result)
			continue
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != entry.SHA256 {
			result.Status = "mutated"
			audit.Mutated = append(audit.Mutated, result)
			continue
		}
		result.Status = "matching"
		audit.Matching = append(audit.Matching, result)
	}
	return audit, nil
}

func ledgerVersionPresence(db *sql.DB, path string) (string, error) {
	asset, version, ok := ledgerVersion(path)
	if !ok {
		return "", nil
	}
	var presence string
	err := db.QueryRow(`SELECT presence FROM component_versions WHERE library_id=? AND version=?`, "react-component-library:"+asset, version).Scan(&presence)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return presence, err
}

func ledgerAsset(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range parts {
		if part == "versions" && i > 0 && i+1 < len(parts) {
			return parts[i-1] + "@" + parts[i+1]
		}
	}
	return "__corpus__.released-version-hashes"
}

// RepairReleaseHashLedger removes only entries whose owning version is gone
// or retired, and re-records only files whose current bytes match the durable
// mirror. All other defects are returned for human evidence review.
func RepairReleaseHashLedger(root string, db *sql.DB, apply bool) (LedgerRepair, error) {
	return RepairReleaseHashLedgerWithOptions(root, db, apply, LedgerRepairOptions{})
}

// RepairReleaseHashLedgerWithOptions performs the safe repair flow and, when
// explicitly requested, accepts current authored bytes whose provenance is
// established outside the mirror (for example, a reviewed repository commit).
func RepairReleaseHashLedgerWithOptions(root string, db *sql.DB, apply bool, options LedgerRepairOptions) (LedgerRepair, error) {
	ledgerPath := filepath.Join(root, "scenarios/react-component-library/library/released-version-hashes.json")
	data, err := os.ReadFile(ledgerPath)
	if err != nil {
		return LedgerRepair{}, err
	}
	var ledger struct {
		SchemaVersion int `json:"schemaVersion"`
		Entries       []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		return LedgerRepair{}, err
	}
	repair := LedgerRepair{Applied: apply}
	kept := ledger.Entries[:0]
	for _, entry := range ledger.Entries {
		if !components.IsAuthoredReleaseFile(entry.Path) {
			kept = append(kept, entry)
			continue
		}
		status, found, queryErr := ledgerVersionStatus(db, entry.Path)
		if queryErr != nil {
			return LedgerRepair{}, queryErr
		}
		if !found || status == "retired" {
			repair.Pruned = append(repair.Pruned, filepath.ToSlash(entry.Path))
			continue
		}
		path := filepath.Join(root, "scenarios/react-component-library/library", filepath.FromSlash(entry.Path))
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			if options.PruneMissing {
				repair.Pruned = append(repair.Pruned, filepath.ToSlash(entry.Path))
				continue
			}
			repair.Unresolved = append(repair.Unresolved, LedgerAuditEntry{Path: filepath.ToSlash(entry.Path), Asset: ledgerAsset(entry.Path), Status: "missing"})
			kept = append(kept, entry)
			continue
		}
		sum := sha256.Sum256(raw)
		current := hex.EncodeToString(sum[:])
		if current == entry.SHA256 {
			kept = append(kept, entry)
			continue
		}
		mirror, mirrorFound, queryErr := ledgerMirror(db, entry.Path)
		if queryErr != nil {
			return LedgerRepair{}, queryErr
		}
		if mirrorFound && string(mirror) == string(raw) {
			entry.SHA256 = current
			repair.Rerecorded = append(repair.Rerecorded, filepath.ToSlash(entry.Path))
		} else if options.AcceptCurrent {
			entry.SHA256 = current
			repair.AcceptedCurrent = append(repair.AcceptedCurrent, filepath.ToSlash(entry.Path))
		} else {
			repair.Unresolved = append(repair.Unresolved, LedgerAuditEntry{Path: filepath.ToSlash(entry.Path), Asset: ledgerAsset(entry.Path), Status: "mutated"})
		}
		kept = append(kept, entry)
	}
	if apply && (len(repair.Pruned) > 0 || len(repair.Rerecorded) > 0 || len(repair.AcceptedCurrent) > 0) {
		ledger.Entries = kept
		updated, err := json.MarshalIndent(ledger, "", "  ")
		if err != nil {
			return LedgerRepair{}, err
		}
		updated = append(updated, '\n')
		tmp := ledgerPath + ".repair.tmp"
		if err := os.WriteFile(tmp, updated, 0o600); err != nil {
			return LedgerRepair{}, err
		}
		if err := os.Rename(tmp, ledgerPath); err != nil {
			_ = os.Remove(tmp)
			return LedgerRepair{}, err
		}
	}
	return repair, nil
}

func ledgerVersionStatus(db *sql.DB, path string) (string, bool, error) {
	asset, version, ok := ledgerVersion(path)
	if !ok {
		return "", false, nil
	}
	var status string
	err := db.QueryRow(`SELECT status FROM component_versions WHERE library_id=? AND version=?`, "react-component-library:"+asset, version).Scan(&status)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return status, err == nil, err
}

func ledgerMirror(db *sql.DB, path string) ([]byte, bool, error) {
	asset, version, ok := ledgerVersion(path)
	if !ok {
		return nil, false, nil
	}
	file := filepath.Base(path)
	var content []byte
	err := db.QueryRow(`SELECT f.content FROM component_version_files f JOIN component_versions v ON v.id=f.version_id WHERE v.library_id=? AND v.version=? AND f.path=?`, "react-component-library:"+asset, version, file).Scan(&content)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	return content, err == nil, err
}

func ledgerVersion(path string) (string, string, bool) {
	parts := strings.Split(filepath.ToSlash(path), "/")
	if len(parts) < 4 || parts[0] == "" || parts[2] != "versions" {
		return "", "", false
	}
	return parts[1], parts[3], true
}
