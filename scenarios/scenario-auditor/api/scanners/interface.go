package scanners

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

// ScannerType identifies the type of scanner
type ScannerType string

const (
	ScannerGosec    ScannerType = "gosec"
	ScannerGitleaks ScannerType = "gitleaks"
	ScannerSemgrep  ScannerType = "semgrep"
	ScannerTrivy    ScannerType = "trivy"
	ScannerBandit   ScannerType = "bandit"
	ScannerESLint   ScannerType = "eslint-security"
	ScannerCustom   ScannerType = "custom-patterns"
)

// Severity levels for vulnerabilities
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// ScanOptions configures a security scan
type ScanOptions struct {
	Path            string          `json:"path"`
	ScanType        string          `json:"scan_type"` // quick, full, targeted
	TargetedChecks  []string        `json:"targeted_checks,omitempty"`
	IncludePatterns []string        `json:"include_patterns,omitempty"`
	ExcludePatterns []string        `json:"exclude_patterns,omitempty"`
	Timeout         time.Duration   `json:"timeout,omitempty"`
	MaxConcurrency  int             `json:"max_concurrency,omitempty"`
	CustomRules     []CustomRule    `json:"custom_rules,omitempty"`
	EnabledScanners []ScannerType   `json:"enabled_scanners,omitempty"`
	Context         context.Context `json:"-"`
	Progress        func(ScannerType, *ScanResult)
}

// CustomRule defines a custom security rule
type CustomRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Pattern     string   `json:"pattern"`
	FileTypes   []string `json:"file_types"`
	Severity    Severity `json:"severity"`
	Category    string   `json:"category"`
	Description string   `json:"description"`
	CWE         int      `json:"cwe,omitempty"`
}

// Finding represents a security vulnerability or issue
type Finding struct {
	ID             string                 `json:"id"`
	ScannerType    ScannerType            `json:"scanner_type"`
	RuleID         string                 `json:"rule_id"`
	Severity       Severity               `json:"severity"`
	Confidence     string                 `json:"confidence"` // high, medium, low
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	Category       string                 `json:"category"`
	FilePath       string                 `json:"file_path"`
	LineNumber     int                    `json:"line_number"`
	ColumnNumber   int                    `json:"column_number,omitempty"`
	EndLine        int                    `json:"end_line,omitempty"`
	CodeSnippet    string                 `json:"code_snippet,omitempty"`
	Recommendation string                 `json:"recommendation"`
	References     []string               `json:"references,omitempty"`
	CWE            int                    `json:"cwe,omitempty"`
	OWASP          string                 `json:"owasp,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ScanResult contains the results from a security scan
type ScanResult struct {
	ScanID       string        `json:"scan_id"`
	ScannerType  ScannerType   `json:"scanner_type"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	ScannedPath  string        `json:"scanned_path"`
	Findings     []Finding     `json:"findings"`
	FilesScanned int           `json:"files_scanned"`
	LinesScanned int           `json:"lines_scanned"`
	ErrorCount   int           `json:"error_count,omitempty"`
	Errors       []string      `json:"errors,omitempty"`
	ToolVersion  string        `json:"tool_version,omitempty"`
	RulesVersion string        `json:"rules_version,omitempty"`
}

// Scanner interface that all security scanners must implement
type Scanner interface {
	// Scan performs a security scan on the specified path
	Scan(opts ScanOptions) (*ScanResult, error)

	// IsAvailable checks if the scanner tool is installed and available
	IsAvailable() bool

	// GetType returns the scanner type
	GetType() ScannerType

	// GetVersion returns the scanner tool version
	GetVersion() (string, error)

	// GetSupportedLanguages returns the languages this scanner supports
	GetSupportedLanguages() []string

	// GetDefaultRules returns the default rules/checks for this scanner
	GetDefaultRules() []CustomRule
}

// ScanOrchestrator manages multiple scanners and aggregates results
type ScanOrchestrator struct {
	scanners        map[ScannerType]Scanner
	enabledScanners []ScannerType
	logger          Logger
}

// NewScanOrchestrator creates a new scan orchestrator
func NewScanOrchestrator(logger Logger) *ScanOrchestrator {
	return &ScanOrchestrator{
		scanners: make(map[ScannerType]Scanner),
		logger:   logger,
	}
}

// RegisterScanner adds a scanner to the orchestrator
func (so *ScanOrchestrator) RegisterScanner(scanner Scanner) error {
	if !scanner.IsAvailable() {
		so.logger.Warn("Scanner %s is not available (tool not installed)", scanner.GetType())
		return nil // Don't error, just skip unavailable scanners
	}

	version, err := scanner.GetVersion()
	if err != nil {
		so.logger.Warn("Failed to get version for scanner %s: %v", scanner.GetType(), err)
	} else {
		so.logger.Info("Registered scanner %s v%s", scanner.GetType(), version)
	}

	so.scanners[scanner.GetType()] = scanner
	so.enabledScanners = append(so.enabledScanners, scanner.GetType())
	return nil
}

// Scan performs a comprehensive scan using all registered scanners
func (so *ScanOrchestrator) Scan(opts ScanOptions) (*AggregatedScanResult, error) {
	startTime := time.Now()

	// Determine which scanners to use
	scannersToUse := so.enabledScanners
	if len(opts.EnabledScanners) > 0 {
		scannersToUse = opts.EnabledScanners
	}

	// Filter based on scan type
	if opts.ScanType == "quick" {
		// For quick scans, only use fast scanners
		scannersToUse = so.filterQuickScanners(scannersToUse)
	}

	results := make([]*ScanResult, 0)
	allFindings := make([]Finding, 0)
	totalFilesScanned := 0
	totalLinesScanned := 0

	// Run each scanner
	for _, scannerType := range scannersToUse {
		if opts.Context != nil {
			select {
			case <-opts.Context.Done():
				return nil, opts.Context.Err()
			default:
			}
		}
		scanner, exists := so.scanners[scannerType]
		if !exists {
			so.logger.Warn("Scanner %s not registered", scannerType)
			continue
		}

		so.logger.Info("Running %s scanner...", scannerType)
		if opts.Progress != nil {
			opts.Progress(scannerType, nil)
		}
		result, err := scanner.Scan(opts)
		if err != nil {
			so.logger.Error("Scanner %s failed: %v", scannerType, err)
			continue
		}

		results = append(results, result)
		allFindings = append(allFindings, result.Findings...)
		totalFilesScanned += result.FilesScanned
		totalLinesScanned += result.LinesScanned

		so.logger.Info("%s found %d issues", scannerType, len(result.Findings))
		if opts.Progress != nil {
			opts.Progress(scannerType, result)
		}
	}

	// Deduplicate findings
	dedupedFindings := so.deduplicateFindings(allFindings)

	// Sort by severity
	sortedFindings := so.sortFindingsBySeverity(dedupedFindings)

	return &AggregatedScanResult{
		ScanID:         generateScanID(),
		StartTime:      startTime,
		EndTime:        time.Now(),
		Duration:       time.Since(startTime),
		ScannedPath:    opts.Path,
		ScanType:       opts.ScanType,
		Findings:       sortedFindings,
		FilesScanned:   totalFilesScanned,
		LinesScanned:   totalLinesScanned,
		ScannerResults: results,
		Statistics:     so.calculateStatistics(sortedFindings),
	}, nil
}

// AggregatedScanResult combines results from multiple scanners
type AggregatedScanResult struct {
	ScanID         string         `json:"scan_id"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	Duration       time.Duration  `json:"duration"`
	ScannedPath    string         `json:"scanned_path"`
	ScanType       string         `json:"scan_type"`
	Findings       []Finding      `json:"findings"`
	FilesScanned   int            `json:"files_scanned"`
	LinesScanned   int            `json:"lines_scanned"`
	ScannerResults []*ScanResult  `json:"scanner_results"`
	Statistics     ScanStatistics `json:"statistics"`
}

// ScanStatistics provides summary statistics for a scan
type ScanStatistics struct {
	TotalFindings      int            `json:"total_findings"`
	BySeverity         map[string]int `json:"by_severity"`
	ByCategory         map[string]int `json:"by_category"`
	ByScanner          map[string]int `json:"by_scanner"`
	TopVulnerableFiles []FileStats    `json:"top_vulnerable_files"`
}

// FileStats provides statistics for a specific file
type FileStats struct {
	FilePath      string `json:"file_path"`
	FindingCount  int    `json:"finding_count"`
	CriticalCount int    `json:"critical_count"`
	HighCount     int    `json:"high_count"`
}

// Helper functions for the orchestrator
func (so *ScanOrchestrator) filterQuickScanners(scanners []ScannerType) []ScannerType {
	quickScanners := []ScannerType{
		ScannerGitleaks, // Fast secret scanning
		ScannerCustom,   // Fast pattern matching
	}

	filtered := make([]ScannerType, 0)
	for _, s := range scanners {
		for _, q := range quickScanners {
			if s == q {
				filtered = append(filtered, s)
				break
			}
		}
	}
	return filtered
}

func (so *ScanOrchestrator) deduplicateFindings(findings []Finding) []Finding {
	seen := make(map[string]bool)
	deduped := make([]Finding, 0)

	for _, finding := range findings {
		// Create a unique key for the finding
		key := finding.FilePath + ":" + strconv.Itoa(finding.LineNumber) + ":" + finding.RuleID
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, finding)
		}
	}

	return deduped
}

func (so *ScanOrchestrator) sortFindingsBySeverity(findings []Finding) []Finding {
	// Simple severity-based sorting (would use sort.Slice in production)
	critical := make([]Finding, 0)
	high := make([]Finding, 0)
	medium := make([]Finding, 0)
	low := make([]Finding, 0)
	info := make([]Finding, 0)

	for _, f := range findings {
		switch f.Severity {
		case SeverityCritical:
			critical = append(critical, f)
		case SeverityHigh:
			high = append(high, f)
		case SeverityMedium:
			medium = append(medium, f)
		case SeverityLow:
			low = append(low, f)
		case SeverityInfo:
			info = append(info, f)
		}
	}

	sorted := make([]Finding, 0, len(findings))
	sorted = append(sorted, critical...)
	sorted = append(sorted, high...)
	sorted = append(sorted, medium...)
	sorted = append(sorted, low...)
	sorted = append(sorted, info...)

	return sorted
}

func (so *ScanOrchestrator) calculateStatistics(findings []Finding) ScanStatistics {
	stats := ScanStatistics{
		TotalFindings: len(findings),
		BySeverity:    make(map[string]int),
		ByCategory:    make(map[string]int),
		ByScanner:     make(map[string]int),
	}

	fileStats := make(map[string]*FileStats)

	for _, f := range findings {
		// Count by severity
		stats.BySeverity[string(f.Severity)]++

		// Count by category
		stats.ByCategory[f.Category]++

		// Count by scanner
		stats.ByScanner[string(f.ScannerType)]++

		// Track file statistics
		if _, exists := fileStats[f.FilePath]; !exists {
			fileStats[f.FilePath] = &FileStats{
				FilePath: f.FilePath,
			}
		}
		fileStats[f.FilePath].FindingCount++
		if f.Severity == SeverityCritical {
			fileStats[f.FilePath].CriticalCount++
		} else if f.Severity == SeverityHigh {
			fileStats[f.FilePath].HighCount++
		}
	}

	// Get top vulnerable files
	for _, fs := range fileStats {
		stats.TopVulnerableFiles = append(stats.TopVulnerableFiles, *fs)
	}

	// Sort top files by finding count (simplified)
	if len(stats.TopVulnerableFiles) > 10 {
		stats.TopVulnerableFiles = stats.TopVulnerableFiles[:10]
	}

	return stats
}

// Logger interface for logging
type Logger interface {
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// Helper function to generate unique scan IDs
func generateScanID() string {
	return time.Now().Format("20060102-150405") + "-" + generateRandomString(6)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// MarshalJSON custom marshaler for AggregatedScanResult
func (r *AggregatedScanResult) MarshalJSON() ([]byte, error) {
	type Alias AggregatedScanResult
	return json.Marshal(&struct {
		Duration string `json:"duration"`
		*Alias
	}{
		Duration: r.Duration.String(),
		Alias:    (*Alias)(r),
	})
}
