package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
)

// ScanFilesOptions mirrors the request body's `options` object.
type ScanFilesOptions struct {
	PII       bool `json:"pii"`
	Secrets   bool `json:"secrets"`
	Watchlist bool `json:"watchlist"`
}

// ScanFilesRequest is the JSON body accepted by POST /security/scan-files.
type ScanFilesRequest struct {
	Files   []string         `json:"files"`
	Options ScanFilesOptions `json:"options"`
}

// ScanFileFinding is the on-wire shape of a single finding in scan-files
// responses. Note: this is intentionally slimmer than SecurityVulnerability so
// GCT and similar callers don't have to handle legacy fields.
type ScanFileFinding struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Pattern  string `json:"pattern"`
	Snippet  string `json:"snippet"`
}

// ScanFilesMetrics summarizes a file-list scan.
type ScanFilesMetrics struct {
	FilesScanned  int `json:"files_scanned"`
	FilesSkipped  int `json:"files_skipped"`
	FindingsCount int `json:"findings_count"`
	DurationMs    int `json:"duration_ms"`
}

// ScanFilesResult is the unified response shape for both the synchronous
// endpoint and the polling endpoint.
type ScanFilesResult struct {
	ScanID             string            `json:"scan_id"`
	Status             string            `json:"status"` // "complete" | "partial" | "running"
	WatchlistAvailable bool              `json:"watchlist_available"`
	Findings           []ScanFileFinding `json:"findings"`
	Metrics            ScanFilesMetrics  `json:"metrics"`
}

const (
	scanFilesMaxFiles    = 500
	scanFilesFileTimeout = 2 * time.Second
	scanFilesSyncTimeout = 10 * time.Second
)

// activeFileScans tracks in-flight scans so the polling endpoint can return
// partial results for a scan that exceeded the synchronous timeout.
var (
	activeFileScans   = map[string]*ScanFilesResult{}
	activeFileScansMu sync.RWMutex
)

func storeActiveFileScan(result *ScanFilesResult) {
	activeFileScansMu.Lock()
	defer activeFileScansMu.Unlock()
	activeFileScans[result.ScanID] = result
}

func snapshotActiveFileScan(id string) *ScanFilesResult {
	activeFileScansMu.RLock()
	defer activeFileScansMu.RUnlock()
	r, ok := activeFileScans[id]
	if !ok {
		return nil
	}
	// Shallow copy + copy the findings slice so callers can marshal without
	// racing the background goroutine appending to it. Use make so the JSON
	// encoder emits `[]` instead of `null` when there are zero findings.
	snap := *r
	snap.Findings = make([]ScanFileFinding, len(r.Findings))
	copy(snap.Findings, r.Findings)
	return &snap
}

// scanFileList runs PII/secret/watchlist matching against the given files,
// enforcing per-file timeouts, allowlist exemptions, and file-size caps.
// The returned ScanFilesResult is the same pointer that gets written into
// `target`, allowing a background goroutine to update status/findings while
// the synchronous handler observes the same state.
func scanFileList(ctx context.Context, database *database.RoutedDB, files []string, opts ScanFilesOptions, target *ScanFilesResult) {
	start := time.Now()

	rules, err := loadAllowlistRules(ctx, database, true)
	if err != nil {
		if logger != nil {
			logger.Warning("failed to load allowlist rules: %v", err)
		}
		rules = nil
	}

	var watchlistEntries []watchlistValue
	watchlistAvailable := watchlistKeyAvailable()
	if opts.Watchlist && watchlistAvailable {
		watchlistEntries, err = loadWatchlistValues(ctx, database)
		if err != nil {
			if logger != nil {
				logger.Warning("failed to load watchlist values: %v", err)
			}
			watchlistEntries = nil
		}
	}

	activeFileScansMu.Lock()
	target.WatchlistAvailable = watchlistAvailable
	activeFileScansMu.Unlock()

	scanOpts := scanFileOptions{
		includeSecrets: opts.Secrets,
		includePII:     opts.PII,
		runAST:         false,
	}

	var persistFindings []SecurityVulnerability

	for _, filePath := range files {
		select {
		case <-ctx.Done():
			activeFileScansMu.Lock()
			target.Status = "partial"
			activeFileScansMu.Unlock()
			break
		default:
		}

		if shouldExcludeFile(rules, filePath, "*") || matchesAnyAllowlistAllType(rules, filePath) {
			activeFileScansMu.Lock()
			target.Metrics.FilesSkipped++
			activeFileScansMu.Unlock()
			continue
		}

		if info, err := os.Stat(filePath); err == nil {
			if info.IsDir() {
				activeFileScansMu.Lock()
				target.Metrics.FilesSkipped++
				activeFileScansMu.Unlock()
				continue
			}
			if info.Size() > MaxScenarioWalkFileSize {
				activeFileScansMu.Lock()
				target.Findings = append(target.Findings, ScanFileFinding{
					File:     filePath,
					Line:     0,
					Type:     "scan_skip",
					Severity: "info",
					Snippet:  "skipped: too_large",
				})
				activeFileScansMu.Unlock()
				continue
			}
		}

		fileCtx, cancel := context.WithTimeout(ctx, scanFilesFileTimeout)
		type scanOutcome struct {
			vulns []SecurityVulnerability
			err   error
		}
		done := make(chan scanOutcome, 1)
		go func() {
			v, err := scanFileForVulnerabilitiesOpts(filePath, "file", filePath, scanOpts)
			done <- scanOutcome{v, err}
		}()

		select {
		case out := <-done:
			cancel()
			if out.err != nil {
				activeFileScansMu.Lock()
				target.Metrics.FilesSkipped++
				activeFileScansMu.Unlock()
				continue
			}

			fileFindings := make([]ScanFileFinding, 0, len(out.vulns))
			rulePatternFindings := 0
			for _, v := range out.vulns {
				if shouldExcludeFile(rules, filePath, v.Type) {
					continue
				}
				rulePatternFindings++
				fileFindings = append(fileFindings, ScanFileFinding{
					File:     v.FilePath,
					Line:     v.LineNumber,
					Type:     v.Type,
					Severity: v.Severity,
					Pattern:  v.Title,
					Snippet:  snippetFromCode(v.Code, v.LineNumber),
				})
				persistFindings = append(persistFindings, v)
			}

			if opts.Watchlist && watchlistAvailable && len(watchlistEntries) > 0 {
				wFindings := matchWatchlistEntries(filePath, watchlistEntries)
				for _, wf := range wFindings {
					if shouldExcludeFile(rules, filePath, wf.Type) {
						continue
					}
					fileFindings = append(fileFindings, wf)
				}
			}

			activeFileScansMu.Lock()
			target.Metrics.FilesScanned++
			target.Findings = append(target.Findings, fileFindings...)
			target.Metrics.FindingsCount = len(target.Findings)
			activeFileScansMu.Unlock()

		case <-fileCtx.Done():
			cancel()
			activeFileScansMu.Lock()
			target.Findings = append(target.Findings, ScanFileFinding{
				File:     filePath,
				Line:     0,
				Type:     "scan_skip",
				Severity: "info",
				Snippet:  "skipped: timeout",
			})
			activeFileScansMu.Unlock()
		}
	}

	activeFileScansMu.Lock()
	target.Metrics.DurationMs = int(time.Since(start).Milliseconds())
	target.Metrics.FindingsCount = len(target.Findings)
	if target.Status != "partial" {
		target.Status = "complete"
	}
	activeFileScansMu.Unlock()

	if database != nil && len(persistFindings) > 0 {
		metrics := ScanMetrics{
			FilesScanned:    target.Metrics.FilesScanned,
			FilesSkipped:    target.Metrics.FilesSkipped,
			TotalScanTimeMs: target.Metrics.DurationMs,
		}
		persistCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		riskScore := calculateRiskScore(persistFindings)
		if _, err := persistSecurityScan(persistCtx, target.ScanID, "", "file-list", "", metrics, riskScore, persistFindings); err != nil && logger != nil {
			logger.Info("failed to persist scan-files run: %v", err)
		}
	}
}

// matchesAnyAllowlistAllType reports whether any enabled allowlist rule with
// `excluded_types = {'*'}` matches the file — which means the file should be
// skipped entirely (counted as files_skipped, not producing findings).
func matchesAnyAllowlistAllType(rules []AllowlistRule, filePath string) bool {
	normalized := strings.ReplaceAll(filePath, "\\", "/")
	base := normalized
	if idx := strings.LastIndex(normalized, "/"); idx >= 0 {
		base = normalized[idx+1:]
	}
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		hasStar := false
		for _, t := range rule.ExcludedTypes {
			if t == "*" {
				hasStar = true
				break
			}
		}
		if !hasStar {
			continue
		}
		if patternMatches(rule.PathPattern, normalized, base) {
			return true
		}
	}
	return false
}

// matchWatchlistEntries scans the file for literal (case-insensitive) matches
// of watchlist values and returns findings. Each matching line is reported.
func matchWatchlistEntries(filePath string, entries []watchlistValue) []ScanFileFinding {
	if len(entries) == 0 {
		return nil
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}
	lines := bytes.Split(content, []byte("\n"))
	var findings []ScanFileFinding
	for idx, line := range lines {
		lower := bytes.ToLower(line)
		for _, entry := range entries {
			needle := bytes.ToLower(entry.Value)
			if len(needle) == 0 {
				continue
			}
			if bytes.Contains(lower, needle) {
				findings = append(findings, ScanFileFinding{
					File:     filePath,
					Line:     idx + 1,
					Type:     fmt.Sprintf("watchlist_%s", entry.ValueType),
					Severity: "high",
					Pattern:  fmt.Sprintf("watchlist: %s", entry.Label),
					Snippet:  string(line),
				})
				break
			}
		}
	}
	return findings
}

// snippetFromCode returns the single line that triggered the finding from a
// multi-line `Code` field produced by extractCodeSnippet. The matching line
// sits at index min(lineNum-1, codeSnippetContextLines) because
// extractCodeSnippet clamps its window start at zero.
func snippetFromCode(code string, lineNum int) string {
	if code == "" {
		return ""
	}
	parts := strings.Split(code, "\n")
	idx := lineNum - 1
	if idx > codeSnippetContextLines {
		idx = codeSnippetContextLines
	}
	if idx < 0 {
		idx = 0
	}
	if idx >= len(parts) {
		idx = len(parts) - 1
	}
	return parts[idx]
}

// ScanFilesHandlers owns the scan-files HTTP surface.
type ScanFilesHandlers struct {
	db     *database.RoutedDB
	logger *Logger
}

// NewScanFilesHandlers returns a configured handler set.
func NewScanFilesHandlers(db *database.RoutedDB, logger *Logger) *ScanFilesHandlers {
	return &ScanFilesHandlers{db: db, logger: logger}
}

// RegisterRoutes mounts the scan-files endpoints under /security.
func (h *ScanFilesHandlers) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/scan-files", h.ScanFiles).Methods(http.MethodPost)
	r.HandleFunc("/scan-runs/{id}", h.GetScanRun).Methods(http.MethodGet)
}

// ScanFiles runs a synchronous-with-timeout scan over a caller-provided file
// list. If the scan exceeds the sync budget, a partial result is returned and
// the scan continues in the background; callers can poll GetScanRun for the
// final state using the returned scan_id.
func (h *ScanFilesHandlers) ScanFiles(w http.ResponseWriter, r *http.Request) {
	var req ScanFilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if len(req.Files) == 0 {
		http.Error(w, "files is required", http.StatusBadRequest)
		return
	}
	if len(req.Files) > scanFilesMaxFiles {
		http.Error(w, fmt.Sprintf("too many files (max %d)", scanFilesMaxFiles), http.StatusBadRequest)
		return
	}

	// Default to running all pattern families when callers don't specify.
	if !req.Options.PII && !req.Options.Secrets && !req.Options.Watchlist {
		req.Options.PII = true
		req.Options.Secrets = true
		req.Options.Watchlist = true
	}

	result := &ScanFilesResult{
		ScanID:             uuid.New().String(),
		Status:             "running",
		WatchlistAvailable: watchlistKeyAvailable(),
		Findings:           []ScanFileFinding{},
	}
	storeActiveFileScan(result)

	scanCtx, cancelScan := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		scanFileList(scanCtx, h.db, req.Files, req.Options, result)
	}()

	timer := time.NewTimer(scanFilesSyncTimeout)
	defer timer.Stop()

	select {
	case <-done:
		cancelScan()
		writeJSON(w, http.StatusOK, snapshotActiveFileScan(result.ScanID))
	case <-timer.C:
		// Mark partial; keep goroutine running so polling endpoint can return
		// the eventual complete result. The scan ctx stays open until the
		// goroutine finishes.
		activeFileScansMu.Lock()
		if result.Status == "running" {
			result.Status = "partial"
		}
		activeFileScansMu.Unlock()
		go func() {
			<-done
			cancelScan()
		}()
		writeJSON(w, http.StatusOK, snapshotActiveFileScan(result.ScanID))
	}
}

// GetScanRun returns the current state of a file-list scan by id. It first
// checks the in-memory map of active scans, then falls back to the persisted
// scan runs table for older scans that have completed and been evicted.
func (h *ScanFilesHandlers) GetScanRun(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	if strings.TrimSpace(id) == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	if snap := snapshotActiveFileScan(id); snap != nil {
		writeJSON(w, http.StatusOK, snap)
		return
	}
	if h.db == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	stored, err := loadScanFilesResult(r.Context(), h.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to load scan: %v", err), http.StatusInternalServerError)
		return
	}
	if stored == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, stored)
}

// loadScanFilesResult reconstructs a ScanFilesResult from persisted scan
// rows. Returns nil (no error) when the scan id isn't present.
func loadScanFilesResult(ctx context.Context, database *database.RoutedDB, scanID string) (*ScanFilesResult, error) {
	if database == nil {
		return nil, nil
	}
	var (
		runID       string
		riskScore   sql.NullInt64
		durationMs  sql.NullInt64
		metadataRaw []byte
		filesScan   sql.NullInt64
		filesSkip   sql.NullInt64
	)
	row := database.QueryRowContext(ctx, `
		SELECT id, risk_score, duration_ms, metadata, files_scanned, files_skipped
		FROM security_scan_runs
		WHERE scan_id = $1
		ORDER BY completed_at DESC
		LIMIT 1
	`, scanID)
	if err := row.Scan(&runID, &riskScore, &durationMs, &metadataRaw, &filesScan, &filesSkip); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	_ = riskScore

	vulnData, err := loadVulnerabilitiesForScan(ctx, runID)
	if err != nil {
		return nil, err
	}

	findings := make([]ScanFileFinding, 0, len(vulnData.vulnerabilities))
	for _, v := range vulnData.vulnerabilities {
		findings = append(findings, ScanFileFinding{
			File:     v.FilePath,
			Line:     v.LineNumber,
			Type:     v.Type,
			Severity: v.Severity,
			Pattern:  v.Title,
			Snippet:  snippetFromCode(v.Code, v.LineNumber),
		})
	}

	metrics := ScanFilesMetrics{
		FilesScanned:  int(filesScan.Int64),
		FilesSkipped:  int(filesSkip.Int64),
		FindingsCount: len(findings),
		DurationMs:    int(durationMs.Int64),
	}
	if len(metadataRaw) > 0 {
		var parsed ScanMetrics
		if err := json.Unmarshal(metadataRaw, &parsed); err == nil {
			if metrics.DurationMs == 0 {
				metrics.DurationMs = parsed.TotalScanTimeMs
			}
		}
	}

	return &ScanFilesResult{
		ScanID:             scanID,
		Status:             "complete",
		WatchlistAvailable: watchlistKeyAvailable(),
		Findings:           findings,
		Metrics:            metrics,
	}, nil
}
