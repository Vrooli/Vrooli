package planworkshop

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/storage"
)

const (
	legacyMigrationMarker  = "legacy-workshop-history-migration-v1.json"
	legacyMigrationVersion = "v2"
	legacyBackupDirectory  = "legacy-workshop-history-backups"
)

// LegacyMigrationError records a source artifact that needs an operator's
// attention. It is intentionally data, not a fatal migration error: corrupt
// historical evidence must stay inspectable and must not prevent clean items
// from entering the new operator model.
type LegacyMigrationError struct {
	SourcePath string `json:"source_path"`
	Message    string `json:"message"`
}

// LegacyMigrationReport is the durable, one-shot cutover record. The source
// workshops remain read-only evidence; unaccepted state additionally receives
// an immutable backup before it is declared archived from active use.
type LegacyMigrationReport struct {
	Version            string                   `json:"version"`
	CompletedAt        string                   `json:"completed_at"`
	Entries            []LegacyHistoryReference `json:"entries"`
	ArchivedUnaccepted int                      `json:"archived_unaccepted"`
	Errors             []LegacyMigrationError   `json:"errors,omitempty"`
}

// MigrateLegacyHistory performs the Plan Workshop cutover migration. It does
// not rewrite historical workshop files or manufacture plan acceptance. Every
// unaccepted legacy workshop is backed up and logically archived in the report;
// queueing remains guarded by the fresh acceptance gate.
func MigrateLegacyHistory(dataRoot string, now time.Time) (LegacyMigrationReport, error) {
	marker := filepath.Join(dataRoot, "plan-workshops", legacyMigrationMarker)
	var existing LegacyMigrationReport
	if found, err := storage.ReadJSON(marker, &existing); err != nil {
		return LegacyMigrationReport{}, err
	} else if found {
		switch existing.Version {
		case legacyMigrationVersion:
			return existing, nil
		case "v1":
			// v1 only counted valid JSON files. Rebuild once so its historical
			// record gains backups, archived-state accounting, and corrupt-file
			// visibility; original source files remain untouched.
		default:
			return LegacyMigrationReport{}, fmt.Errorf("unsupported legacy workshop migration marker %q", existing.Version)
		}
	}

	report := LegacyMigrationReport{Version: legacyMigrationVersion, CompletedAt: now.UTC().Format(time.RFC3339Nano)}
	for _, kind := range []string{"idea", "research", "fix", "execute", "chore"} {
		items, err := os.ReadDir(filepath.Join(dataRoot, kind))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return LegacyMigrationReport{}, fmt.Errorf("read legacy %s items: %w", kind, err)
		}
		for _, item := range items {
			if !item.IsDir() {
				continue
			}
			sourcePath := filepath.ToSlash(filepath.Join(kind, item.Name(), "workshop"))
			workshopDir := filepath.Join(dataRoot, filepath.FromSlash(sourcePath))
			rounds, errors, exists, err := inspectLegacyWorkshop(workshopDir, sourcePath)
			if err != nil {
				return LegacyMigrationReport{}, err
			}
			report.Errors = append(report.Errors, errors...)
			if !exists || rounds == 0 {
				continue
			}
			entry := LegacyHistoryReference{SourcePath: sourcePath, RoundCount: rounds, ArchivedAt: report.CompletedAt}
			accepted, acceptanceErr := legacyPlanAccepted(filepath.Join(dataRoot, kind, item.Name(), "spec.json"))
			if acceptanceErr != nil {
				report.Errors = append(report.Errors, LegacyMigrationError{SourcePath: filepath.ToSlash(filepath.Join(kind, item.Name(), "spec.json")), Message: acceptanceErr.Error()})
			}
			if !accepted {
				backupPath := filepath.ToSlash(filepath.Join("plan-workshops", legacyBackupDirectory, sourcePath))
				if err := backupLegacyWorkshop(workshopDir, filepath.Join(dataRoot, filepath.FromSlash(backupPath))); err != nil {
					return LegacyMigrationReport{}, fmt.Errorf("backup %s: %w", sourcePath, err)
				}
				entry.BackupPath = backupPath
				entry.ArchivedUnaccepted = true
				report.ArchivedUnaccepted++
			}
			report.Entries = append(report.Entries, entry)
		}
	}
	sort.Slice(report.Entries, func(i, j int) bool { return report.Entries[i].SourcePath < report.Entries[j].SourcePath })
	sort.Slice(report.Errors, func(i, j int) bool { return report.Errors[i].SourcePath < report.Errors[j].SourcePath })
	if err := storage.WriteJSONAtomic(marker, report); err != nil {
		return LegacyMigrationReport{}, err
	}
	return report, nil
}

func inspectLegacyWorkshop(dir, sourcePath string) (rounds int, errors []LegacyMigrationError, exists bool, err error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil, false, nil
	}
	if err != nil {
		return 0, nil, false, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "round-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		rounds++
		contents, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			errors = append(errors, LegacyMigrationError{SourcePath: filepath.ToSlash(filepath.Join(sourcePath, entry.Name())), Message: readErr.Error()})
			continue
		}
		if !json.Valid(contents) {
			errors = append(errors, LegacyMigrationError{SourcePath: filepath.ToSlash(filepath.Join(sourcePath, entry.Name())), Message: "invalid JSON historical round"})
		}
	}
	return rounds, errors, true, nil
}

func legacyPlanAccepted(specPath string) (bool, error) {
	contents, err := os.ReadFile(specPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var document struct {
		PlanAcceptance *struct {
			Actor           string `json:"actor"`
			PlanContentHash string `json:"plan_content_hash"`
			SubjectVersion  string `json:"subject_version"`
		} `json:"plan_acceptance"`
	}
	if err := json.Unmarshal(contents, &document); err != nil {
		return false, fmt.Errorf("invalid backlog spec JSON")
	}
	if document.PlanAcceptance == nil {
		return false, nil
	}
	return strings.TrimSpace(document.PlanAcceptance.Actor) != "" &&
		strings.TrimSpace(document.PlanAcceptance.PlanContentHash) != "" &&
		strings.TrimSpace(document.PlanAcceptance.SubjectVersion) != "", nil
}

func backupLegacyWorkshop(source, destination string) error {
	if _, err := os.Stat(destination); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	temporary := destination + ".staging"
	if err := os.RemoveAll(temporary); err != nil {
		return err
	}
	if err := copyTree(source, temporary); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o750); err != nil {
		return err
	}
	return os.Rename(temporary, destination)
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported historical workshop file %q", path)
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
