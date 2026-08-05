package validation

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

const migrationAllowlistPath = "scenarios/storage-manager/config/storage-migration-allowlist.json"

type migrationAllowlist struct {
	Version int                       `json:"version"`
	Entries []migrationAllowlistEntry `json:"entries"`
}

type migrationAllowlistEntry struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Code string `json:"code"`
}

func loadMigrationAllowlist(repoRoot string) (map[string]struct{}, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return map[string]struct{}{}, nil
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, migrationAllowlistPath))
	if errors.Is(err, os.ErrNotExist) {
		return map[string]struct{}{}, nil
	}
	if err != nil {
		return nil, err
	}
	var file migrationAllowlist
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	allowed := make(map[string]struct{}, len(file.Entries))
	for _, entry := range file.Entries {
		kind := corestorage.OwnerKind(strings.TrimSpace(entry.Kind))
		id := strings.TrimSpace(entry.ID)
		code := strings.TrimSpace(entry.Code)
		if kind == "" || id == "" || code == "" {
			continue
		}
		allowed[allowlistKey(kind, id, code)] = struct{}{}
	}
	return allowed, nil
}

func allowlistKey(kind corestorage.OwnerKind, id, code string) string {
	return string(kind) + "\x00" + id + "\x00" + code
}

func (s *Service) applyMigrationAllowlist(report Report) Report {
	if len(s.allowlist) == 0 {
		return report
	}
	for i := range report.Findings {
		if _, ok := s.allowlist[allowlistKey(report.OwnerKind, report.OwnerID, report.Findings[i].Code)]; ok && report.Findings[i].Severity >= SeverityError {
			report.Findings[i].Severity = SeverityWarning
			report.Findings[i].Remediation = "Resolve this migration exception before removing its temporary allowlist entry."
		}
	}
	report.Status = reportStatus(report.Findings)
	return report
}
