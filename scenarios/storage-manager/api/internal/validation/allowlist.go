package validation

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	corestorage "github.com/vrooli/api-core/storage"
)

const (
	migrationAllowlistPath  = "scenarios/storage-manager/config/storage-migration-allowlist.json"
	filesystemAllowlistPath = "scenarios/storage-manager/config/filesystem-contract-allowlist.json"
)

type migrationAllowlist struct {
	Version int                       `json:"version"`
	Entries []migrationAllowlistEntry `json:"entries"`
}

type migrationAllowlistEntry struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	Code string `json:"code"`
}

type filesystemAllowlist struct {
	Version int                        `json:"version"`
	Entries []filesystemAllowlistEntry `json:"entries"`
}

type filesystemAllowlistEntry struct {
	Kind          string `json:"kind"`
	ID            string `json:"id"`
	Code          string `json:"code"`
	Owner         string `json:"owner"`
	Reason        string `json:"reason"`
	Scope         string `json:"scope"`
	ReviewTrigger string `json:"review_trigger"`
}

type filesystemException struct {
	Owner         string
	Reason        string
	Scope         string
	ReviewTrigger string
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

func loadFilesystemAllowlist(repoRoot string) (map[string]filesystemException, error) {
	allowed := make(map[string]filesystemException)
	if strings.TrimSpace(repoRoot) == "" {
		return allowed, nil
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, filesystemAllowlistPath))
	if errors.Is(err, os.ErrNotExist) {
		return allowed, nil
	}
	if err != nil {
		return nil, err
	}
	var file filesystemAllowlist
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	for _, entry := range file.Entries {
		kind := corestorage.OwnerKind(strings.TrimSpace(entry.Kind))
		id := strings.TrimSpace(entry.ID)
		code := strings.TrimSpace(entry.Code)
		owner := strings.TrimSpace(entry.Owner)
		reason := strings.TrimSpace(entry.Reason)
		scope := strings.TrimSpace(entry.Scope)
		review := strings.TrimSpace(entry.ReviewTrigger)
		if kind == "" || id == "" || code == "" || owner == "" || reason == "" || scope == "" || review == "" {
			continue
		}
		allowed[allowlistKey(kind, id, code)] = filesystemException{Owner: owner, Reason: reason, Scope: scope, ReviewTrigger: review}
	}
	return allowed, nil
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

func (s *Service) applyFilesystemAllowlist(report Report) Report {
	if len(s.filesystemAllowlist) == 0 {
		return report
	}
	for i := range report.Findings {
		finding := &report.Findings[i]
		if !strings.HasPrefix(finding.Code, "FILESYSTEM_") {
			continue
		}
		exception, ok := s.filesystemAllowlist[allowlistKey(report.OwnerKind, report.OwnerID, finding.Code)]
		if !ok {
			continue
		}
		finding.Severity = SeverityInfo
		finding.Remediation = fmt.Sprintf("Reviewed exception owned by %s (scope: %s). Reason: %s. Re-review when %s.", exception.Owner, exception.Scope, exception.Reason, exception.ReviewTrigger)
	}
	report.Status = reportStatus(report.Findings)
	return report
}
