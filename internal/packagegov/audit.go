package packagegov

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type AuditReport struct {
	Validation ValidationReport  `json:"validation"`
	Issues     []ValidationIssue `json:"issues"`
	ScanStats  ScanStats         `json:"scan_stats"`
}

type ScanStats struct {
	FilesVisited    int            `json:"files_visited"`
	FilesScanned    int            `json:"files_scanned"`
	FilesSkipped    int            `json:"files_skipped"`
	BytesScanned    int64          `json:"bytes_scanned"`
	SkippedByReason map[string]int `json:"skipped_by_reason,omitempty"`
	BudgetExceeded  bool           `json:"budget_exceeded"`
}

func (s *ScanStats) addSkip(reason string) {
	s.FilesSkipped++
	if s.SkippedByReason == nil {
		s.SkippedByReason = make(map[string]int)
	}
	s.SkippedByReason[reason]++
}

func (s *ScanStats) merge(other ScanStats) {
	s.FilesVisited += other.FilesVisited
	s.FilesScanned += other.FilesScanned
	s.FilesSkipped += other.FilesSkipped
	s.BytesScanned += other.BytesScanned
	s.BudgetExceeded = s.BudgetExceeded || other.BudgetExceeded
	for reason, count := range other.SkippedByReason {
		if s.SkippedByReason == nil {
			s.SkippedByReason = make(map[string]int)
		}
		s.SkippedByReason[reason] += count
	}
}

type scanPolicy struct {
	MaxFileBytes  int64
	MaxTotalBytes int64
}

var (
	defaultScanPolicy = scanPolicy{
		MaxFileBytes:  4 << 20,
		MaxTotalBytes: 256 << 20,
	}
	scanOpenFile          = os.Open
	allowedScanExtensions = map[string]struct{}{
		".md":   {},
		".mdx":  {},
		".txt":  {},
		".json": {},
		".yaml": {},
		".yml":  {},
		".toml": {},
		".go":   {},
		".ts":   {},
		".tsx":  {},
		".js":   {},
		".sh":   {},
	}
	skippedScanExtensions = map[string]string{
		".db":       "runtime-data",
		".sqlite":   "runtime-data",
		".sqlite3":  "runtime-data",
		".wal":      "runtime-data",
		".shm":      "runtime-data",
		".png":      "binary-media",
		".jpg":      "binary-media",
		".jpeg":     "binary-media",
		".gif":      "binary-media",
		".webp":     "binary-media",
		".ico":      "binary-media",
		".pdf":      "binary-doc",
		".zip":      "archive",
		".gz":       "archive",
		".tgz":      "archive",
		".tar":      "archive",
		".xz":       "archive",
		".7z":       "archive",
		".exe":      "executable",
		".dll":      "executable",
		".so":       "executable",
		".dylib":    "executable",
		".appimage": "executable",
	}
)

func Audit(root string, filter string) (AuditReport, error) {
	validation, err := Validate(root, filter)
	if err != nil {
		return AuditReport{}, err
	}

	var issues []ValidationIssue
	var stats ScanStats
	for _, path := range []string{
		filepath.Join(root, "packages"),
		filepath.Join(root, "scenarios"),
		filepath.Join(root, "templates"),
		filepath.Join(root, "docs"),
		filepath.Join(root, "pnpm-workspace.yaml"),
	} {
		scanIssues, scanStats, err := scanDocsDrift(path)
		if err != nil {
			return AuditReport{}, err
		}
		issues = append(issues, scanIssues...)
		stats.merge(scanStats)
	}

	return AuditReport{
		Validation: validation,
		Issues:     normalizeIssues(append(validation.Issues, issues...)),
		ScanStats:  stats,
	}, nil
}

func scanDocsDrift(root string) ([]ValidationIssue, ScanStats, error) {
	return scanDocsDriftWithPolicy(root, defaultScanPolicy)
}

func scanDocsDriftWithPolicy(root string, policy scanPolicy) ([]ValidationIssue, ScanStats, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ScanStats{}, nil
		}
		return nil, ScanStats{}, err
	}
	if !info.IsDir() {
		var stats ScanStats
		issues, err := scanSingleFile(root, policy, &stats)
		return issues, stats, err
	}

	var issues []ValidationIssue
	var stats ScanStats
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				stats.addSkip("permission-denied")
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}
		if d.IsDir() {
			if reason, skip := shouldSkipScanDir(d.Name()); skip {
				stats.addSkip(reason)
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		fileIssues, err := scanSingleFile(path, policy, &stats)
		if err != nil {
			return err
		}
		issues = append(issues, fileIssues...)
		return nil
	})
	return issues, stats, err
}

func scanSingleFile(path string, policy scanPolicy, stats *ScanStats) ([]ValidationIssue, error) {
	stats.FilesVisited++
	slashPath := filepath.ToSlash(path)
	if reason, skip := shouldSkipScanFile(slashPath); skip {
		stats.addSkip(reason)
		return nil, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsPermission(err) {
			stats.addSkip("permission-denied")
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	if policy.MaxFileBytes > 0 && info.Size() > policy.MaxFileBytes {
		stats.BudgetExceeded = true
		stats.addSkip("file-byte-budget")
		return nil, nil
	}
	if policy.MaxTotalBytes > 0 && stats.BytesScanned+info.Size() > policy.MaxTotalBytes {
		stats.BudgetExceeded = true
		stats.addSkip("total-byte-budget")
		return nil, nil
	}

	content, scanned, binary, err := readBoundedTextFile(path, policy.MaxFileBytes)
	if err != nil {
		if os.IsPermission(err) {
			stats.addSkip("permission-denied")
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if binary {
		stats.addSkip("binary-content")
		return nil, nil
	}
	stats.FilesScanned++
	stats.BytesScanned += scanned

	var issues []ValidationIssue
	if strings.Contains(content, "refresh-shared-package.sh") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "legacy-refresh-helper-reference",
			Message:  "legacy refresh helper is still referenced",
			Path:     path,
		})
	}
	guidanceCandidate := isGovernanceGuidanceCandidate(slashPath)
	if guidanceCandidate && strings.Contains(content, "workspace:*") &&
		!strings.Contains(content, "must not use `workspace:*`") &&
		!strings.Contains(content, "Do NOT use `\"workspace:*\"`") &&
		!strings.Contains(content, "Do NOT use `workspace:*`") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "workspace-star-guidance",
			Message:  "workspace:* guidance conflicts with scenario isolation policy",
			Path:     path,
		})
	}
	if guidanceCandidate && strings.Contains(content, "pnpm workspace") &&
		!strings.Contains(content, "do not join the root pnpm workspace") &&
		!strings.Contains(content, "do not join the pnpm workspace") &&
		!strings.Contains(content, "do not use the pnpm workspace") &&
		!strings.Contains(content, "must not use the pnpm workspace") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "pnpm-workspace-guidance",
			Message:  "pnpm workspace guidance conflicts with scenario isolation policy",
			Path:     path,
		})
	}
	if guidanceCandidate && strings.Contains(content, "go.work") &&
		!strings.Contains(content, "GOWORK=off") &&
		!strings.Contains(content, "does not depend on a repo-wide `go.work`") &&
		!strings.Contains(content, "must not depend on a repo-wide `go.work`") &&
		!strings.Contains(content, "no dependency on repo-level go.work") {
		issues = append(issues, ValidationIssue{
			Severity: "error",
			Code:     "go-work-guidance",
			Message:  "go.work guidance conflicts with governed Go package adoption",
			Path:     path,
		})
	}
	return issues, nil
}

func readBoundedTextFile(path string, maxBytes int64) (string, int64, bool, error) {
	file, err := scanOpenFile(path)
	if err != nil {
		return "", 0, false, err
	}
	defer file.Close()

	limit := maxBytes
	if limit <= 0 {
		limit = 4 << 20
	}
	reader := bufio.NewReader(io.LimitReader(file, limit))
	var builder strings.Builder
	buffer := make([]byte, 32*1024)
	var scanned int64
	for {
		n, readErr := reader.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			if bytes.IndexByte(chunk, 0) >= 0 {
				return "", scanned + int64(n), true, nil
			}
			builder.Write(chunk)
			scanned += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", scanned, false, readErr
		}
	}
	return builder.String(), scanned, false, nil
}

func shouldSkipScanDir(name string) (string, bool) {
	name = strings.TrimSpace(name)
	switch name {
	case "data":
		return "runtime-data-dir", true
	case "node_modules", "dist", "dist-electron", "build", "bundle", "bin", "coverage", "generated", "testdata", "logs", "artifacts", ".turbo", ".vite":
		return "output-dir", true
	}
	if strings.HasPrefix(name, ".git") {
		return "vcs-dir", true
	}
	if strings.Contains(strings.ToLower(name), "backup") {
		return "backup-dir", true
	}
	return "", false
}

func shouldSkipScanFile(path string) (string, bool) {
	lower := strings.ToLower(path)
	switch {
	case strings.Contains(path, "/docs/plans/"):
		return "plan-doc", true
	case strings.Contains(path, "/testdata/"):
		return "testdata", true
	case strings.HasSuffix(path, ".lock"), strings.HasSuffix(path, "pnpm-lock.yaml"):
		return "lockfile", true
	case strings.HasSuffix(path, "SKILL.md"):
		return "skill-doc", true
	case strings.HasSuffix(path, "_test.go"):
		return "test-code", true
	case strings.HasSuffix(path, ".pb.go"), strings.HasSuffix(path, ".pb.ts"):
		return "generated-proto", true
	case strings.HasSuffix(lower, ".db-wal"), strings.HasSuffix(lower, ".db-shm"):
		return "runtime-data", true
	}
	ext := strings.ToLower(filepath.Ext(path))
	if reason, ok := skippedScanExtensions[ext]; ok {
		return reason, true
	}
	if _, ok := allowedScanExtensions[ext]; !ok {
		return "extension-not-allowlisted", true
	}
	return "", false
}

func isGovernanceGuidanceCandidate(path string) bool {
	path = filepath.ToSlash(path)
	if strings.HasPrefix(path, "docs/") || strings.Contains(path, "/docs/") && !strings.Contains(path, "/scenarios/") {
		return true
	}
	if strings.HasPrefix(path, "packages/") || strings.Contains(path, "/packages/") {
		return true
	}
	if strings.HasPrefix(path, "templates/") || strings.Contains(path, "/templates/") {
		return true
	}
	if (strings.HasPrefix(path, "scenarios/") || strings.Contains(path, "/scenarios/")) && strings.HasSuffix(path, "/README.md") {
		return true
	}
	return false
}
