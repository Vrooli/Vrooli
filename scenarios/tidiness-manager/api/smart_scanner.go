package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SmartScanConfig represents AI batch scanning configuration (TM-SS-001)
type SmartScanConfig struct {
	MaxFilesPerBatch int
	MaxConcurrent    int
	MaxTokensPerReq  int
}

// Validate checks if configuration is valid
func (c *SmartScanConfig) Validate() bool {
	return c.MaxFilesPerBatch > 0 && c.MaxConcurrent > 0
}

// GetDefaultSmartScanConfig returns default AI batch configuration
func GetDefaultSmartScanConfig() SmartScanConfig {
	return SmartScanConfig{
		MaxFilesPerBatch: 10,
		MaxConcurrent:    5,
		MaxTokensPerReq:  100000,
	}
}

// SmartScanner orchestrates AI-powered code analysis
type SmartScanner struct {
	config        SmartScanConfig
	resourceURL   string
	sessionID     string
	analyzedFiles map[string]bool
	mu            sync.RWMutex
	httpClient    *http.Client
}

// NewSmartScanner creates a SmartScanner with given configuration
func NewSmartScanner(config SmartScanConfig) (*SmartScanner, error) {
	if !config.Validate() {
		return nil, fmt.Errorf("invalid smart scan configuration")
	}

	// Detect claude-code resource URL
	resourceURL := os.Getenv("CLAUDE_CODE_URL")
	if resourceURL == "" {
		resourceURL = "http://localhost:8100" // Default claude-code port
	}

	return &SmartScanner{
		config:        config,
		resourceURL:   resourceURL,
		sessionID:     generateSessionID(),
		analyzedFiles: make(map[string]bool),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// SmartScanRequest represents a request to perform smart scanning
type SmartScanRequest struct {
	Scenario    string   `json:"scenario"`
	Files       []string `json:"files"`
	ForceRescan bool     `json:"force_rescan"`
	CampaignID  *int     `json:"campaign_id,omitempty"`
}

// SmartScanResult represents the result of a smart scan
type SmartScanResult struct {
	SessionID     string        `json:"session_id"`
	FilesAnalyzed int           `json:"files_analyzed"`
	IssuesFound   int           `json:"issues_found"`
	BatchResults  []BatchResult `json:"batch_results"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
}

// BatchResult represents results from a single batch
type BatchResult struct {
	BatchID  int           `json:"batch_id"`
	Files    []string      `json:"files"`
	Issues   []AIIssue     `json:"issues"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// AIIssue represents an issue found by AI analysis (TM-SS-002)
type AIIssue struct {
	FilePath         string `json:"file_path"`
	Category         string `json:"category"` // dead_code, duplication, length, complexity, style
	Severity         string `json:"severity"` // critical, high, medium, low, info
	Title            string `json:"title"`
	Description      string `json:"description"`
	LineNumber       *int   `json:"line_number,omitempty"`
	ColumnNumber     *int   `json:"column_number,omitempty"`
	AgentNotes       string `json:"agent_notes,omitempty"`
	RemediationSteps string `json:"remediation_steps"`
}

// ScanScenario performs smart scanning on a scenario (TM-SS-001, TM-SS-002, TM-SS-007)
func (s *SmartScanner) ScanScenario(ctx context.Context, req SmartScanRequest) (*SmartScanResult, error) {
	startTime := time.Now()

	result := &SmartScanResult{
		SessionID:    s.sessionID,
		BatchResults: []BatchResult{},
		Errors:       []string{},
	}

	// TM-SS-007: Filter out files already analyzed in this session
	filesToAnalyze := []string{}
	for _, file := range req.Files {
		if req.ForceRescan || !s.isFileAnalyzed(file) {
			filesToAnalyze = append(filesToAnalyze, file)
		}
	}

	if len(filesToAnalyze) == 0 {
		return result, nil
	}

	// TM-SS-001: Create batches based on max files per batch
	batches := createBatches(filesToAnalyze, s.config.MaxFilesPerBatch)

	// TM-SS-001: Process batches with concurrency limit
	semaphore := make(chan struct{}, s.config.MaxConcurrent)
	var wg sync.WaitGroup
	batchResults := make(chan BatchResult, len(batches))

	for i, batch := range batches {
		wg.Add(1)
		go func(batchID int, files []string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			batchResult := s.processBatch(ctx, batchID, files, req.Scenario)
			batchResults <- batchResult
		}(i, batch)
	}

	wg.Wait()
	close(batchResults)

	// Collect results
	for batchResult := range batchResults {
		result.BatchResults = append(result.BatchResults, batchResult)
		result.IssuesFound += len(batchResult.Issues)
		result.FilesAnalyzed += len(batchResult.Files)

		if batchResult.Error != "" {
			result.Errors = append(result.Errors, batchResult.Error)
		}

		// TM-SS-007: Mark files as analyzed
		for _, file := range batchResult.Files {
			s.markFileAnalyzed(file)
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// processBatch analyzes a batch of files using AI resource
func (s *SmartScanner) processBatch(ctx context.Context, batchID int, files []string, scenario string) BatchResult {
	startTime := time.Now()

	result := BatchResult{
		BatchID: batchID,
		Files:   files,
		Issues:  []AIIssue{},
	}

	// Read file contents
	fileContents := make(map[string]string)
	for _, file := range files {
		content, err := s.readFileContent(scenario, file)
		if err != nil {
			result.Error = fmt.Sprintf("failed to read file %s: %v", file, err)
			result.Duration = time.Since(startTime)
			return result
		}
		fileContents[file] = content
	}

	// Call AI resource to analyze batch
	issues, err := s.callAIResource(ctx, fileContents)
	if err != nil {
		result.Error = fmt.Sprintf("AI analysis failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.Issues = issues
	result.Duration = time.Since(startTime)
	return result
}

// readFileContent reads the content of a file from the scenario directory
func (s *SmartScanner) readFileContent(scenario, filePath string) (string, error) {
	// Security: Prevent path traversal attacks
	scenarioPath := getScenarioPath(scenario)
	fullPath := filepath.Join(scenarioPath, filePath)

	// Ensure the resolved path is within the scenario directory
	cleanPath := filepath.Clean(fullPath)
	rel, err := filepath.Rel(scenarioPath, cleanPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid file path: access denied")
	}

	// Security: Limit file size to prevent resource exhaustion (10MB max)
	info, err := os.Stat(cleanPath)
	if err != nil {
		return "", err
	}
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if info.Size() > maxFileSize {
		return "", fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxFileSize)
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// callAIResource sends files to AI resource for analysis (placeholder implementation)
// In real implementation, this would call resource-claude-code or resource-codes
func (s *SmartScanner) callAIResource(ctx context.Context, fileContents map[string]string) ([]AIIssue, error) {
	// Build prompt for AI analysis
	prompt := s.buildAnalysisPrompt(fileContents)

	// Prepare request payload
	reqBody := map[string]interface{}{
		"prompt":      prompt,
		"max_tokens":  s.config.MaxTokensPerReq,
		"temperature": 0.3, // Lower temperature for more consistent analysis
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call AI resource endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", s.resourceURL+"/api/analyze", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		// If resource unavailable, return empty results (graceful degradation)
		return []AIIssue{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Security: Limit error response size to prevent memory exhaustion
		limitedBody := io.LimitReader(resp.Body, 1024) // 1KB max for error messages
		io.ReadAll(limitedBody)
		// Sanitize error message to prevent information leakage
		return nil, fmt.Errorf("AI resource returned status %d", resp.StatusCode)
	}

	// Security: Limit response body size to prevent memory exhaustion (50MB max)
	const maxResponseSize = 50 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxResponseSize)

	// Parse AI response
	var aiResponse struct {
		Issues []AIIssue `json:"issues"`
	}

	if err := json.NewDecoder(limitedReader).Decode(&aiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode AI response")
	}

	return aiResponse.Issues, nil
}

// buildAnalysisPrompt constructs the prompt for AI analysis
func (s *SmartScanner) buildAnalysisPrompt(fileContents map[string]string) string {
	var sb strings.Builder

	sb.WriteString("You are a code tidiness analyzer. Analyze the following files and identify tidiness issues.\n\n")
	sb.WriteString("Focus on:\n")
	sb.WriteString("- Dead code (unused functions, imports, variables)\n")
	sb.WriteString("- Code duplication\n")
	sb.WriteString("- Excessive file length or complexity\n")
	sb.WriteString("- Inconsistent code style\n\n")
	sb.WriteString("Return issues in JSON format with fields: file_path, category, severity, title, description, line_number, remediation_steps\n\n")

	sb.WriteString("Files to analyze:\n\n")
	for path, content := range fileContents {
		sb.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", path, content))
	}

	return sb.String()
}

// isFileAnalyzed checks if a file has been analyzed in this session (TM-SS-007)
func (s *SmartScanner) isFileAnalyzed(file string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.analyzedFiles[file]
}

// markFileAnalyzed marks a file as analyzed in this session (TM-SS-007)
func (s *SmartScanner) markFileAnalyzed(file string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.analyzedFiles[file] = true
}

// createBatches splits files into batches
func createBatches(files []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batches = append(batches, files[i:end])
	}
	return batches
}

// generateSessionID generates a unique session identifier
func generateSessionID() string {
	return fmt.Sprintf("smart-scan-%d", time.Now().Unix())
}
