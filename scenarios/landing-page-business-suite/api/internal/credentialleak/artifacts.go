package credentialleak

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxArtifactFiles = 1000
const maxArtifactTotalBytes = 64 << 20

type ArtifactFinding struct {
	File           string `json:"file,omitempty"`
	Family         string `json:"family"`
	Channel        string `json:"channel"`
	Representation string `json:"representation"`
}

type ArtifactFileResult struct {
	File   string `json:"file"`
	Bytes  int64  `json:"bytes"`
	Status string `json:"status"`
}

type ArtifactScanResult struct {
	Status        string               `json:"status"`
	FilesScanned  int                  `json:"files_scanned"`
	BytesScanned  int64                `json:"bytes_scanned"`
	Files         []ArtifactFileResult `json:"files,omitempty"`
	Findings      []ArtifactFinding    `json:"findings,omitempty"`
	FailureReason string               `json:"failure_reason,omitempty"`
}

// ScanArtifactDirectory recursively scans native Test Genie artifacts and
// returns metadata only. It rejects symlinks and oversized files rather than
// silently skipping or truncating them.
func (s *Scanner) ScanArtifactDirectory(root string) ArtifactScanResult {
	result := ArtifactScanResult{Status: "failed"}
	if s == nil || len(s.needles) == 0 {
		result.FailureReason = "scanner_uninitialized"
		return result
	}
	if strings.TrimSpace(root) == "" {
		result.FailureReason = "artifact_directory_required"
		return result
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		result.FailureReason = "artifact_directory_unavailable"
		return result
	}
	info, err := os.Lstat(abs)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		result.FailureReason = "artifact_directory_unavailable"
		return result
	}
	filesSeen := 0
	err = filepath.WalkDir(abs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("artifact_walk_failed")
		}
		if path == abs {
			return nil
		}
		rel, relErr := filepath.Rel(abs, path)
		if relErr != nil {
			return fmt.Errorf("artifact_path_invalid")
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("artifact_symlink_refused")
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("artifact_non_regular_file_refused")
		}
		fileInfo, infoErr := entry.Info()
		if infoErr != nil {
			return fmt.Errorf("artifact_metadata_unavailable")
		}
		if fileInfo.Size() > maxObservationBytes {
			return fmt.Errorf("artifact_scan_bound_exceeded")
		}
		if filesSeen >= maxArtifactFiles {
			return fmt.Errorf("artifact_file_count_exceeded")
		}
		filesSeen++
		if fileInfo.Size() > maxArtifactTotalBytes-result.BytesScanned {
			return fmt.Errorf("artifact_total_bound_exceeded")
		}
		file, openedInfo, openErr := openVerifiedArtifact(path, fileInfo, os.Open)
		if openErr != nil {
			return openErr
		}
		data, readErr := io.ReadAll(io.LimitReader(file, maxObservationBytes+1))
		afterInfo, statErr := file.Stat()
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			return fmt.Errorf("artifact_read_failed")
		}
		if statErr != nil || afterInfo.Size() != openedInfo.Size() || !afterInfo.ModTime().Equal(openedInfo.ModTime()) {
			return fmt.Errorf("artifact_changed_during_scan")
		}
		findings, scanErr := s.Scan(Observation{Channel: "test-genie-artifact", Data: data})
		if scanErr != nil {
			return fmt.Errorf("artifact_scan_bound_exceeded")
		}
		pathFindings, pathErr := s.Scan(Observation{Channel: "test-genie-artifact-path", Data: []byte(filepath.ToSlash(rel))})
		if pathErr != nil {
			return fmt.Errorf("artifact_path_scan_failed")
		}
		result.FilesScanned++
		result.BytesScanned += int64(len(data))
		fileStatus := "clean"
		if len(findings)+len(pathFindings) > 0 {
			fileStatus = "leak_detected"
		}
		if fileStatus == "clean" {
			result.Files = append(result.Files, ArtifactFileResult{File: filepath.ToSlash(rel), Bytes: int64(len(data)), Status: fileStatus})
		}
		for _, finding := range findings {
			result.Findings = append(result.Findings, ArtifactFinding{Family: finding.Family, Channel: finding.Channel, Representation: finding.Representation})
		}
		for _, finding := range pathFindings {
			result.Findings = append(result.Findings, ArtifactFinding{Family: finding.Family, Channel: finding.Channel, Representation: finding.Representation})
		}
		return nil
	})
	if err != nil {
		result.Files = nil
		result.Findings = nil
		result.FailureReason = err.Error()
		return result
	}
	if result.FilesScanned == 0 {
		result.FailureReason = "artifact_set_empty"
		return result
	}
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].File < result.Files[j].File })
	sort.Slice(result.Findings, func(i, j int) bool {
		left, right := result.Findings[i], result.Findings[j]
		if left.Family != right.Family {
			return left.Family < right.Family
		}
		if left.Channel != right.Channel {
			return left.Channel < right.Channel
		}
		return left.Representation < right.Representation
	})
	if len(result.Findings) > 0 {
		result.Status = "leak_detected"
		return result
	}
	result.Status = "clean"
	return result
}

// openVerifiedArtifact binds the walk-time identity to the opened descriptor and
// rechecks the pathname after opening. The opener parameter keeps the race
// boundary deterministic in tests without weakening the production check.
func openVerifiedArtifact(path string, walkedInfo fs.FileInfo, openFile func(string) (*os.File, error)) (*os.File, fs.FileInfo, error) {
	file, err := openFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("artifact_read_failed")
	}
	openedInfo, statErr := file.Stat()
	pathInfo, pathStatErr := os.Lstat(path)
	if statErr != nil || pathStatErr != nil || !openedInfo.Mode().IsRegular() || pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(walkedInfo, openedInfo) || !os.SameFile(openedInfo, pathInfo) {
		_ = file.Close()
		return nil, nil, fmt.Errorf("artifact_changed_during_scan")
	}
	return file, openedInfo, nil
}

// ScanTestGenieArtifactDirectory requires both native result files so an empty
// or unrelated directory can never be mistaken for proof that Test Genie
// artifacts are secret-free.
func (s *Scanner) ScanTestGenieArtifactDirectory(root string) ArtifactScanResult {
	result := s.ScanArtifactDirectory(root)
	if result.FailureReason != "" {
		return result
	}
	seen := make(map[string]bool, len(result.Files))
	for _, file := range result.Files {
		seen[strings.ToLower(filepath.Base(file.File))] = true
	}
	// Leaking files are intentionally omitted from Files. Their filenames are
	// still inspected via a separate metadata walk, never included in receipts.
	if len(result.Findings) > 0 {
		if !artifactNamePresent(root, "phase.result.json") || !artifactNamePresent(root, "findings.report.json") {
			result.Status, result.FailureReason = "failed", "required_test_genie_artifact_missing"
			return result
		}
	} else if !seen["phase.result.json"] || !seen["findings.report.json"] {
		result.Status, result.FailureReason = "failed", "required_test_genie_artifact_missing"
	}
	return result
}

func artifactNamePresent(root, name string) bool {
	found := false
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.Type()&os.ModeSymlink != 0 {
			return filepath.SkipAll
		}
		if !entry.IsDir() && strings.EqualFold(entry.Name(), name) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found
}
