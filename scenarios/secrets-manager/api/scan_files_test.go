package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestScanFileList_FindsPII(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "handler.go", "package main\n\nvar email = \"alice@example.com\"\n")

	result := &ScanFilesResult{ScanID: "test-1", Status: "running", Findings: []ScanFileFinding{}}
	storeActiveFileScan(result)
	defer func() {
		activeFileScansMu.Lock()
		delete(activeFileScans, result.ScanID)
		activeFileScansMu.Unlock()
	}()

	scanFileList(context.Background(), nil, []string{file}, ScanFilesOptions{PII: true, Secrets: false}, result)

	if result.Status != "complete" {
		t.Errorf("expected status complete, got %q", result.Status)
	}
	if result.Metrics.FilesScanned != 1 {
		t.Errorf("expected files_scanned=1, got %d", result.Metrics.FilesScanned)
	}
	found := false
	for _, f := range result.Findings {
		if f.Type == "pii_email" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected pii_email finding, got %+v", result.Findings)
	}
}

func TestScanFileList_SkipsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	// Write something slightly larger than the cap.
	big := strings.Repeat("x", int(MaxScenarioWalkFileSize)+100)
	file := writeTempFile(t, dir, "big.txt", big)

	result := &ScanFilesResult{ScanID: "test-2", Status: "running", Findings: []ScanFileFinding{}}
	storeActiveFileScan(result)
	defer func() {
		activeFileScansMu.Lock()
		delete(activeFileScans, result.ScanID)
		activeFileScansMu.Unlock()
	}()

	scanFileList(context.Background(), nil, []string{file}, ScanFilesOptions{PII: true}, result)

	foundSkip := false
	for _, f := range result.Findings {
		if f.Type == "scan_skip" && strings.Contains(f.Snippet, "too_large") {
			foundSkip = true
			break
		}
	}
	if !foundSkip {
		t.Errorf("expected scan_skip finding for oversized file, got %+v", result.Findings)
	}
}

func TestScanFileList_EmptyFindingsForCleanFile(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "clean.go", "package main\n\nfunc main() {}\n")

	result := &ScanFilesResult{ScanID: "test-3", Status: "running", Findings: []ScanFileFinding{}}
	storeActiveFileScan(result)
	defer func() {
		activeFileScansMu.Lock()
		delete(activeFileScans, result.ScanID)
		activeFileScansMu.Unlock()
	}()

	scanFileList(context.Background(), nil, []string{file}, ScanFilesOptions{PII: true, Secrets: true}, result)

	if result.Metrics.FilesScanned != 1 {
		t.Errorf("expected files_scanned=1, got %d", result.Metrics.FilesScanned)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected no findings, got %+v", result.Findings)
	}
}

func TestMatchWatchlistEntries(t *testing.T) {
	dir := t.TempDir()
	file := writeTempFile(t, dir, "notes.txt", "secret code: abc-42-xyz\nother line\nABC-42-XYZ appears here too\n")

	entries := []watchlistValue{
		{ID: "1", Label: "my code", ValueType: "custom", Value: []byte("abc-42-xyz")},
	}

	findings := matchWatchlistEntries(file, entries)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings (case-insensitive), got %d: %+v", len(findings), findings)
	}
	for _, f := range findings {
		if f.Type != "watchlist_custom" {
			t.Errorf("expected watchlist_custom type, got %s", f.Type)
		}
	}
}

func TestSnippetFromCode(t *testing.T) {
	// extractCodeSnippet returns up to codeSnippetContextLines (2) lines
	// above the matching line, then the matching line, then up to 2 below.
	// Caller passes the original 1-based line number; snippetFromCode must
	// return the matching line, not always the first line of the block.
	cases := []struct {
		name    string
		code    string
		lineNum int
		want    string
	}{
		{"line1_no_above_context", "first\nsecond\nthird", 1, "first"},
		{"line2_one_line_above", "first\nsecond\nthird", 2, "second"},
		{"line3_two_lines_above", "first\nsecond\nthird\nfourth", 3, "third"},
		{"line5_clamped_to_index_2", "above2\nabove1\nmatching\nbelow1\nbelow2", 5, "matching"},
		{"single_line_block", "only", 7, "only"},
		{"empty_code", "", 4, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := snippetFromCode(tc.code, tc.lineNum); got != tc.want {
				t.Errorf("snippetFromCode(%q, %d) = %q, want %q", tc.code, tc.lineNum, got, tc.want)
			}
		})
	}
}

func TestSnapshotActiveFileScan_EmptyFindingsNotNull(t *testing.T) {
	// Regression: the JSON response must emit `findings: []` rather than
	// `findings: null` when the scan produced (or filtered out) every match.
	result := &ScanFilesResult{
		ScanID:   "snap-empty-1",
		Status:   "complete",
		Findings: []ScanFileFinding{},
	}
	storeActiveFileScan(result)
	defer func() {
		activeFileScansMu.Lock()
		delete(activeFileScans, result.ScanID)
		activeFileScansMu.Unlock()
	}()

	snap := snapshotActiveFileScan(result.ScanID)
	if snap == nil {
		t.Fatal("expected snapshot, got nil")
	}
	if snap.Findings == nil {
		t.Fatal("expected non-nil empty findings slice, got nil")
	}
	encoded, err := json.Marshal(snap)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"findings":[]`) {
		t.Errorf("expected findings: [] in JSON, got %s", string(encoded))
	}
}

func TestScanFileList_HomeDirLineStartAttribution(t *testing.T) {
	// Regression: pii_home_dir previously matched the trailing \n of the
	// preceding line (\s covers newlines), so a path starting at column 1 of
	// line N was reported on line N-1 with line N-1's snippet. Assert that a
	// line-1 / column-1 home-dir path is attributed to line 1 and that one on
	// a later line is attributed to its actual line.
	dir := t.TempDir()
	content := "/home/alice/data starts here\n" +
		"unrelated line\n" +
		"/Users/bob/projects follows\n"
	file := writeTempFile(t, dir, "paths.txt", content)

	result := &ScanFilesResult{ScanID: "home-dir-1", Status: "running", Findings: []ScanFileFinding{}}
	storeActiveFileScan(result)
	defer func() {
		activeFileScansMu.Lock()
		delete(activeFileScans, result.ScanID)
		activeFileScansMu.Unlock()
	}()

	scanFileList(context.Background(), nil, []string{file}, ScanFilesOptions{PII: true}, result)

	contentLines := strings.Split(content, "\n")
	var homeDirFindings []ScanFileFinding
	for _, f := range result.Findings {
		if f.Type == "pii_home_dir" {
			homeDirFindings = append(homeDirFindings, f)
		}
	}
	if len(homeDirFindings) != 2 {
		t.Fatalf("expected 2 pii_home_dir findings, got %d: %+v", len(homeDirFindings), result.Findings)
	}
	seen := map[int]bool{}
	for _, f := range homeDirFindings {
		seen[f.Line] = true
		if f.Line < 1 || f.Line > len(contentLines) {
			t.Errorf("finding line %d out of range", f.Line)
			continue
		}
		if f.Snippet != contentLines[f.Line-1] {
			t.Errorf("snippet/line mismatch: line=%d snippet=%q expected %q",
				f.Line, f.Snippet, contentLines[f.Line-1])
		}
	}
	if !seen[1] || !seen[3] {
		t.Errorf("expected findings on lines 1 and 3, got %+v", homeDirFindings)
	}
}

func TestScanFileList_SnippetMatchesLineNumber(t *testing.T) {
	// Regression: with multiple findings on different lines, each snippet
	// must contain the actual matched-line text, not the first line of the
	// extracted context block.
	dir := t.TempDir()
	content := "package main\n\nvar email = \"alice@example.com\"\nvar phone = \"555-123-4567\"\n"
	file := writeTempFile(t, dir, "leaks.go", content)

	result := &ScanFilesResult{ScanID: "snippet-1", Status: "running", Findings: []ScanFileFinding{}}
	storeActiveFileScan(result)
	defer func() {
		activeFileScansMu.Lock()
		delete(activeFileScans, result.ScanID)
		activeFileScansMu.Unlock()
	}()

	scanFileList(context.Background(), nil, []string{file}, ScanFilesOptions{PII: true}, result)

	if len(result.Findings) == 0 {
		t.Fatalf("expected findings, got none")
	}
	contentLines := strings.Split(content, "\n")
	for _, f := range result.Findings {
		if f.Type == "scan_skip" {
			continue
		}
		if f.Line < 1 || f.Line > len(contentLines) {
			t.Errorf("finding line %d out of range", f.Line)
			continue
		}
		expected := contentLines[f.Line-1]
		if f.Snippet != expected {
			t.Errorf("snippet/line mismatch: type=%s line=%d snippet=%q expected line text %q",
				f.Type, f.Line, f.Snippet, expected)
		}
	}
}
