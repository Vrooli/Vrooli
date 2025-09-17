package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type StandardsViolation struct {
	ID             string `json:"id"`
	ScenarioName   string `json:"scenario_name"`
	Type           string `json:"type"`
	Severity       string `json:"severity"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	FilePath       string `json:"file_path"`
	LineNumber     int    `json:"line_number"`
	CodeSnippet    string `json:"code_snippet,omitempty"`
	Recommendation string `json:"recommendation"`
	Standard       string `json:"standard"`
	DiscoveredAt   string `json:"discovered_at"`
}

type StandardsCheckResult struct {
	CheckID      string              `json:"check_id"`
	Status       string              `json:"status"`
	ScanType     string              `json:"scan_type"`
	StartedAt    string              `json:"started_at"`
	CompletedAt  string              `json:"completed_at"`
	Duration     float64             `json:"duration_seconds"`
	FilesScanned int                 `json:"files_scanned"`
	Violations   []StandardsViolation `json:"violations"`
	Statistics   map[string]int      `json:"statistics"`
	Message      string              `json:"message"`
	ScenarioName string              `json:"scenario_name,omitempty"`
}

// Standards compliance check handler
func enhancedStandardsCheckHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	logger := NewLogger()

	w.Header().Set("Content-Type", "application/json")

	// Parse request body for check options
	var checkRequest struct {
		Type           string   `json:"type"`            // "quick", "full", or "targeted"
		Standards      []string `json:"standards"`       // Specific standards to check
	}
	if err := json.NewDecoder(r.Body).Decode(&checkRequest); err != nil {
		checkRequest.Type = "full" // Default to full check
	}

	// Validate check type
	if checkRequest.Type != "quick" && checkRequest.Type != "full" && checkRequest.Type != "targeted" {
		checkRequest.Type = "full"
	}

	startTime := time.Now()

	// Determine scan path
	var scanPath string
	var scenarioCount int = 1

	if scenarioName == "all" {
		scanPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios")
		// Count scenarios for reporting
		scenarios, _ := getVrooliScenarios()
		if scenarios != nil && scenarios.Scenarios != nil {
			scenarioCount = len(scenarios.Scenarios)
		}
	} else {
		scanPath = filepath.Join(os.Getenv("HOME"), "Vrooli", "scenarios", scenarioName)
		if _, err := os.Stat(scanPath); os.IsNotExist(err) {
			HTTPError(w, "Scenario not found", http.StatusNotFound, nil)
			return
		}
	}

	logger.Info(fmt.Sprintf("Starting %s standards compliance check on %s", checkRequest.Type, scenarioName))

	// Perform the standards check
	violations, filesScanned, err := performStandardsCheck(scanPath, checkRequest.Type, checkRequest.Standards)
	if err != nil {
		logger.Error("Standards compliance check failed", err)
		HTTPError(w, "Standards check failed", http.StatusInternalServerError, err)
		return
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Calculate statistics
	stats := map[string]int{
		"total":    len(violations),
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
	}

	for _, violation := range violations {
		stats[violation.Severity]++
	}

	// Build response
	response := StandardsCheckResult{
		CheckID:      fmt.Sprintf("standards-%d", time.Now().Unix()),
		Status:       "completed",
		ScanType:     checkRequest.Type,
		StartedAt:    startTime.Format(time.RFC3339),
		CompletedAt:  endTime.Format(time.RFC3339),
		Duration:     duration.Seconds(),
		FilesScanned: filesScanned,
		Violations:   violations,
		Statistics:   stats,
	}

	if scenarioName == "all" {
		response.Message = fmt.Sprintf("Standards compliance check completed across %d scenarios. Found %d violations.", scenarioCount, len(violations))
	} else {
		response.ScenarioName = scenarioName
		response.Message = fmt.Sprintf("Standards compliance check completed for %s. Found %d violations.", scenarioName, len(violations))
	}

	// Store violations in memory store
	standardsStore.StoreViolations(scenarioName, violations)
	logger.Info(fmt.Sprintf("Stored %d standards violations in memory for %s", len(violations), scenarioName))
	
	logger.Info(fmt.Sprintf("Standards compliance check completed: %d violations found", len(violations)))
	json.NewEncoder(w).Encode(response)
}

// performStandardsCheck performs the actual standards compliance checking
func performStandardsCheck(scanPath, checkType string, specificStandards []string) ([]StandardsViolation, int, error) {
	var violations []StandardsViolation
	filesScanned := 0

	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}

		// Skip directories and non-relevant files
		if info.IsDir() {
			// Skip common directories we don't want to scan
			if shouldSkipDirectory(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Only scan relevant file types
		if !isRelevantFile(path) {
			return nil
		}

		filesScanned++

		// Read file content
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Get relative path for display
		relativePath := getRelativePath(path, scanPath)
		scenarioName := extractScenarioName(path)

		// Perform various standards checks
		fileViolations := []StandardsViolation{}

		// Check 1: Lifecycle protection in main.go files
		if filepath.Base(path) == "main.go" && strings.Contains(string(content), "func main()") {
			if violation := checkLifecycleProtection(content, relativePath, scenarioName); violation != nil {
				fileViolations = append(fileViolations, *violation)
			}
		}

		// Check 2: Test file coverage for Go files
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			if violation := checkTestFileCoverage(path, relativePath, scenarioName); violation != nil {
				fileViolations = append(fileViolations, *violation)
			}
		}

		// Check 3: Lightweight main.go structure
		if filepath.Base(path) == "main.go" {
			if violations := checkMainGoStructure(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		// Check 4: Hardcoded ports and URLs
		if violations := checkHardcodedValues(content, relativePath, scenarioName); violations != nil {
			fileViolations = append(fileViolations, violations...)
		}

		// Check 5: Versioned endpoints (for Go API files)
		if strings.Contains(string(content), "http.HandleFunc") || strings.Contains(string(content), "mux.") {
			if violations := checkVersionedEndpoints(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		// Check 6: HTTP status codes (for Go API files)
		if strings.HasSuffix(path, ".go") {
			if violations := checkHTTPStatusCodes(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		// Check 7: Content-Type headers (for Go API files)
		if strings.HasSuffix(path, ".go") && (strings.Contains(string(content), "http.ResponseWriter") || strings.Contains(string(content), "w.Write")) {
			if violations := checkContentTypeHeaders(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		// Check 8: Environment variable naming
		if strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".sh") || strings.HasSuffix(path, ".json") {
			if violations := checkEnvironmentVariableNaming(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		// Check 9: Structured logging patterns
		if strings.HasSuffix(path, ".go") {
			if violations := checkStructuredLogging(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		// Check 10: Resource leak detection (CRITICAL for port exhaustion)
		if strings.HasSuffix(path, ".go") {
			if violations := checkResourceLeaks(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		// Check 11: Health check implementation
		if strings.HasSuffix(path, ".go") && strings.Contains(string(content), "health") {
			if violations := checkHealthCheckImplementation(content, relativePath, scenarioName); violations != nil {
				fileViolations = append(fileViolations, violations...)
			}
		}

		violations = append(violations, fileViolations...)
		return nil
	})

	return violations, filesScanned, err
}

// checkLifecycleProtection checks if main.go has the required lifecycle protection
func checkLifecycleProtection(content []byte, filePath, scenarioName string) *StandardsViolation {
	contentStr := string(content)
	
	// Check if the lifecycle protection exists
	lifecyclePattern := `VROOLI_LIFECYCLE_MANAGED.*!=.*"true"`
	matched, _ := regexp.MatchString(lifecyclePattern, contentStr)
	
	if !matched {
		return &StandardsViolation{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			Type:           "lifecycle_protection",
			Severity:       "high",
			Title:          "Missing Lifecycle Protection",
			Description:    "main.go file lacks required lifecycle protection check",
			FilePath:       filePath,
			LineNumber:     findMainFunctionLine(contentStr),
			Recommendation: `Add lifecycle protection at the start of main(): if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" { /* error handling */ }`,
			Standard:       "Vrooli Lifecycle Management",
			DiscoveredAt:   time.Now().Format(time.RFC3339),
		}
	}
	
	return nil
}

// checkTestFileCoverage checks if a Go source file has a corresponding test file
func checkTestFileCoverage(filePath, relativePath, scenarioName string) *StandardsViolation {
	// Skip if this is already a test file or not in main directories
	if strings.HasSuffix(filePath, "_test.go") || 
	   strings.Contains(filePath, "/vendor/") ||
	   strings.Contains(filePath, "/node_modules/") {
		return nil
	}

	// Generate expected test file path
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	testFile := filepath.Join(dir, strings.TrimSuffix(base, ".go")+"_test.go")

	// Check if test file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		return &StandardsViolation{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			Type:           "test_coverage",
			Severity:       "medium",
			Title:          "Missing Test File",
			Description:    fmt.Sprintf("Go source file %s has no corresponding test file", base),
			FilePath:       relativePath,
			LineNumber:     1,
			Recommendation: fmt.Sprintf("Create test file: %s_test.go", strings.TrimSuffix(base, ".go")),
			Standard:       "Test Coverage",
			DiscoveredAt:   time.Now().Format(time.RFC3339),
		}
	}

	return nil
}

// checkMainGoStructure checks if main.go is lightweight and well-structured
func checkMainGoStructure(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Check if main function is too long (more than 50 lines is suspicious)
	mainStart := findMainFunctionLine(contentStr)
	if mainStart > 0 {
		braceCount := 0
		mainLength := 0
		inMain := false

		for i := mainStart - 1; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			
			if strings.Contains(line, "func main()") {
				inMain = true
			}
			
			if inMain {
				mainLength++
				braceCount += strings.Count(line, "{")
				braceCount -= strings.Count(line, "}")
				
				// End of main function
				if braceCount == 0 && mainLength > 1 {
					break
				}
			}
		}

		if mainLength > 50 {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "main_structure",
				Severity:       "medium",
				Title:          "Main Function Too Large",
				Description:    fmt.Sprintf("main() function is %d lines long, consider refactoring", mainLength),
				FilePath:       filePath,
				LineNumber:     mainStart,
				Recommendation: "Move business logic to separate functions or packages. main() should primarily handle initialization and orchestration.",
				Standard:       "Code Organization",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
	}

	return violations
}

// checkHardcodedValues checks for hardcoded ports, URLs, and other configuration
func checkHardcodedValues(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Patterns to detect hardcoded values
	patterns := []struct {
		regex       *regexp.Regexp
		title       string
		description string
		severity    string
	}{
		{
			regexp.MustCompile(`:\d{4,5}[^0-9]`), // Hardcoded ports
			"Hardcoded Port Number",
			"Found hardcoded port number, should use environment variable",
			"high",
		},
		{
			regexp.MustCompile(`https?://localhost`), // Hardcoded localhost URLs
			"Hardcoded Localhost URL",
			"Found hardcoded localhost URL, should use environment variable",
			"high",
		},
		{
			regexp.MustCompile(`https?://[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`), // Hardcoded IPs
			"Hardcoded IP Address",
			"Found hardcoded IP address in URL, should use environment variable",
			"high",
		},
		{
			regexp.MustCompile(`"password":\s*"[^"]+"`), // Hardcoded passwords
			"Potential Hardcoded Password",
			"Found potential hardcoded password in configuration",
			"critical",
		},
	}

	for lineNum, line := range lines {
		// Skip comments
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}

		for _, pattern := range patterns {
			if pattern.regex.MatchString(line) {
				violations = append(violations, StandardsViolation{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					Type:           "hardcoded_values",
					Severity:       pattern.severity,
					Title:          pattern.title,
					Description:    pattern.description,
					FilePath:       filePath,
					LineNumber:     lineNum + 1,
					CodeSnippet:    strings.TrimSpace(line),
					Recommendation: "Replace hardcoded value with environment variable or configuration parameter",
					Standard:       "Configuration Management",
					DiscoveredAt:   time.Now().Format(time.RFC3339),
				})
			}
		}
	}

	return violations
}

// checkVersionedEndpoints checks if API endpoints use versioning
func checkVersionedEndpoints(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Look for endpoint definitions
	endpointPattern := regexp.MustCompile(`(?:HandleFunc|Handle|GET|POST|PUT|DELETE)\s*\(\s*["\']([^"\']+)["\']`)

	for lineNum, line := range lines {
		matches := endpointPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				endpoint := match[1]
				
				// Check if endpoint has version prefix
				versionPattern := regexp.MustCompile(`^/(?:api/)?v\d+/`)
				if !versionPattern.MatchString(endpoint) && !strings.HasPrefix(endpoint, "/health") {
					violations = append(violations, StandardsViolation{
						ID:             uuid.New().String(),
						ScenarioName:   scenarioName,
						Type:           "endpoint_versioning",
						Severity:       "medium",
						Title:          "Unversioned API Endpoint",
						Description:    fmt.Sprintf("API endpoint '%s' lacks version prefix", endpoint),
						FilePath:       filePath,
						LineNumber:     lineNum + 1,
						CodeSnippet:    strings.TrimSpace(line),
						Recommendation: fmt.Sprintf("Add version prefix like '/api/v1%s'", endpoint),
						Standard:       "API Versioning",
						DiscoveredAt:   time.Now().Format(time.RFC3339),
					})
				}
			}
		}
	}

	return violations
}

// Helper functions
func shouldSkipDirectory(path string) bool {
	skipDirs := []string{
		"vendor", "node_modules", ".git", "dist", "build", 
		".next", ".nuxt", "target", "bin", "obj",
	}
	
	for _, skip := range skipDirs {
		if strings.Contains(path, "/"+skip+"/") || strings.HasSuffix(path, "/"+skip) {
			return true
		}
	}
	return false
}

func isRelevantFile(path string) bool {
	relevantExts := []string{".go", ".js", ".ts", ".py", ".java", ".yaml", ".yml", ".json"}
	
	for _, ext := range relevantExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func getRelativePath(fullPath, basePath string) string {
	if rel, err := filepath.Rel(basePath, fullPath); err == nil {
		return rel
	}
	return fullPath
}

func extractScenarioName(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		if part == "scenarios" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return "unknown"
}

func findMainFunctionLine(content string) int {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.Contains(line, "func main()") {
			return i + 1
		}
	}
	return 1
}

// checkHTTPStatusCodes checks for proper HTTP status code usage
func checkHTTPStatusCodes(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Patterns that indicate HTTP handlers
	if !strings.Contains(contentStr, "http.ResponseWriter") {
		return violations
	}

	for lineNum, line := range lines {
		// Check for generic status writes without proper codes
		if strings.Contains(line, "w.WriteHeader(") {
			// Good: w.WriteHeader(http.StatusOK)
			// Bad: w.WriteHeader(200)
			if matched, _ := regexp.MatchString(`w\.WriteHeader\(\d+\)`, line); matched {
				violations = append(violations, StandardsViolation{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					Type:           "http_status_codes",
					Severity:       "medium",
					Title:          "Hardcoded HTTP Status Code",
					Description:    "Use http.Status constants instead of numeric codes",
					FilePath:       filePath,
					LineNumber:     lineNum + 1,
					CodeSnippet:    strings.TrimSpace(line),
					Recommendation: "Replace with http.StatusOK, http.StatusBadRequest, etc.",
					Standard:       "API Design",
					DiscoveredAt:   time.Now().Format(time.RFC3339),
				})
			}
		}

		// Check for http.Error usage without proper status codes
		if strings.Contains(line, "http.Error(") && !strings.Contains(line, "http.Status") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "http_status_codes",
				Severity:       "medium",
				Title:          "HTTP Error Without Status Constant",
				Description:    "http.Error should use http.Status constants",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Use http.Error(w, \"message\", http.StatusBadRequest)",
				Standard:       "API Design",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
	}

	return violations
}

// checkContentTypeHeaders checks for proper Content-Type header setting
func checkContentTypeHeaders(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	hasResponseWriter := strings.Contains(contentStr, "http.ResponseWriter")
	hasJSONResponse := strings.Contains(contentStr, "json.NewEncoder") || strings.Contains(contentStr, "json.Marshal")
	hasContentTypeSet := strings.Contains(contentStr, `"Content-Type"`)

	if hasResponseWriter && hasJSONResponse && !hasContentTypeSet {
		violations = append(violations, StandardsViolation{
			ID:             uuid.New().String(),
			ScenarioName:   scenarioName,
			Type:           "content_type_headers",
			Severity:       "medium",
			Title:          "Missing Content-Type Header",
			Description:    "JSON responses should set Content-Type header",
			FilePath:       filePath,
			LineNumber:     1,
			Recommendation: `Add w.Header().Set("Content-Type", "application/json")`,
			Standard:       "API Design",
			DiscoveredAt:   time.Now().Format(time.RFC3339),
		})
	}

	// Check for responses without explicit content type
	for lineNum, line := range lines {
		if strings.Contains(line, "w.Write(") && !strings.Contains(contentStr[:strings.Index(contentStr, line)], "Content-Type") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "content_type_headers",
				Severity:       "low",
				Title:          "Response Without Content-Type",
				Description:    "HTTP responses should specify Content-Type",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Set appropriate Content-Type header before writing response",
				Standard:       "API Design",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
	}

	return violations
}

// checkEnvironmentVariableNaming checks for consistent environment variable naming
func checkEnvironmentVariableNaming(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Pattern for environment variables
	envPattern := regexp.MustCompile(`os\.Getenv\("([^"]+)"\)|getenv\("([^"]+)"\)|\$\{([^}]+)\}|\$([A-Z_][A-Z0-9_]*)`)

	for lineNum, line := range lines {
		matches := envPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			var envVar string
			for i := 1; i < len(match); i++ {
				if match[i] != "" {
					envVar = match[i]
					break
				}
			}

			if envVar != "" {
				// Check naming convention: UPPER_CASE_WITH_UNDERSCORES
				if !regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(envVar) {
					violations = append(violations, StandardsViolation{
						ID:             uuid.New().String(),
						ScenarioName:   scenarioName,
						Type:           "env_var_naming",
						Severity:       "low",
						Title:          "Environment Variable Naming Convention",
						Description:    fmt.Sprintf("Environment variable '%s' doesn't follow UPPER_CASE convention", envVar),
						FilePath:       filePath,
						LineNumber:     lineNum + 1,
						CodeSnippet:    strings.TrimSpace(line),
						Recommendation: "Use UPPER_CASE_WITH_UNDERSCORES naming for environment variables",
						Standard:       "Configuration Management",
						DiscoveredAt:   time.Now().Format(time.RFC3339),
					})
				}
			}
		}
	}

	return violations
}

// checkStructuredLogging checks for proper logging patterns
func checkStructuredLogging(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Check if this is main.go - it should have Logger setup
	if strings.HasSuffix(filePath, "main.go") {
		hasLoggerStruct := strings.Contains(contentStr, "type Logger struct")
		hasNewLogger := strings.Contains(contentStr, "func NewLogger(")
		// Check for service name prefix in logger initialization
		// Look for patterns like [scenario-name] or "[scenario-name]"
		hasServicePrefix := false
		for _, line := range lines {
			if strings.Contains(line, "log.New(") && 
			   (strings.Contains(line, "["+scenarioName+"]") || 
			    strings.Contains(line, `"[`+scenarioName+`]"`)) {
				hasServicePrefix = true
				break
			}
		}
		
		if !hasLoggerStruct || !hasNewLogger {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "structured_logging",
				Severity:       "high",
				Title:          "Missing Logger Setup",
				Description:    "Main.go should define a Logger struct with NewLogger() function",
				FilePath:       filePath,
				LineNumber:     1,
				CodeSnippet:    "",
				Recommendation: "Add Logger struct with Info() and Error() methods, initialized with service name prefix",
				Standard:       "Logging",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		} else if !hasServicePrefix {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "structured_logging",
				Severity:       "medium",
				Title:          "Missing Service Name in Logger",
				Description:    "Logger should include service name prefix",
				FilePath:       filePath,
				LineNumber:     1,
				CodeSnippet:    "",
				Recommendation: fmt.Sprintf(`Use log.New(os.Stdout, "[%s] ", log.LstdFlags|log.Lshortfile)`, scenarioName),
				Standard:       "Logging",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
	}

	// Track if we're in the lifecycle protection block
	inLifecycleBlock := false
	lifecycleBlockStart := -1
	
	for lineNum, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Detect start of lifecycle protection block
		if strings.Contains(line, "VROOLI_LIFECYCLE_MANAGED") && strings.Contains(line, "os.Getenv") {
			inLifecycleBlock = true
			lifecycleBlockStart = lineNum
		}
		
		// Detect end of lifecycle block (usually os.Exit or closing brace after several lines)
		if inLifecycleBlock && (strings.Contains(line, "os.Exit(") || 
			(lineNum > lifecycleBlockStart+10 && trimmedLine == "}")) {
			inLifecycleBlock = false
		}
		
		// Skip checks if we're in the lifecycle protection block
		// fmt.Fprintf(os.Stderr is allowed in lifecycle block
		if inLifecycleBlock {
			continue
		}
		
		// Check for fmt.Println usage (should use structured logging)
		if strings.Contains(line, "fmt.Println(") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "structured_logging",
				Severity:       "medium",
				Title:          "Unstructured Logging",
				Description:    "Use structured logging instead of fmt.Println",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Use logger.Info() from your Logger instance",
				Standard:       "Logging",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
		
		// Check for fmt.Printf (except fmt.Fprintf(os.Stderr which is used in lifecycle)
		if strings.Contains(line, "fmt.Printf(") && !strings.Contains(line, "fmt.Fprintf") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "structured_logging",
				Severity:       "medium",
				Title:          "Unstructured Logging",
				Description:    "Use structured logging instead of fmt.Printf",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Use logger.Info() or logger.Error() from your Logger instance",
				Standard:       "Logging",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
		
		// Check for log.Println (should use logger instance)
		if strings.Contains(line, "log.Println(") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "structured_logging",
				Severity:       "medium",
				Title:          "Direct log.Println Usage",
				Description:    "Use logger instance instead of direct log.Println",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Use logger.Info() from your Logger instance",
				Standard:       "Logging",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}

		// Check for proper error logging
		if strings.Contains(line, "log.Fatal(") && !strings.Contains(line, "err") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "structured_logging",
				Severity:       "high",
				Title:          "Unsafe Fatal Logging",
				Description:    "log.Fatal should include error context",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: `Include error details: log.Fatalf("Failed to %s: %v", action, err)`,
				Standard:       "Logging",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
		
		// Check for raw log.Printf outside of Logger methods
		// Skip if line is inside a Logger method (rough heuristic)
		if strings.Contains(line, "log.Printf(") && !strings.Contains(line, "l.Printf") {
			// Try to check if we're inside a Logger method
			isInLoggerMethod := false
			for i := lineNum - 1; i >= 0 && i > lineNum-20; i-- {
				if strings.Contains(lines[i], "func (l *Logger)") {
					isInLoggerMethod = true
					break
				}
				if strings.Contains(lines[i], "func ") && !strings.Contains(lines[i], "func (") {
					// We hit a different function, stop looking
					break
				}
			}
			
			if !isInLoggerMethod {
				violations = append(violations, StandardsViolation{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					Type:           "structured_logging",
					Severity:       "medium",
					Title:          "Direct log.Printf Usage",
					Description:    "Use logger instance instead of direct log.Printf",
					FilePath:       filePath,
					LineNumber:     lineNum + 1,
					CodeSnippet:    strings.TrimSpace(line),
					Recommendation: "Use logger.Info() or logger.Error() from your Logger instance",
					Standard:       "Logging",
					DiscoveredAt:   time.Now().Format(time.RFC3339),
				})
			}
		}
	}

	return violations
}

// checkResourceLeaks checks for potential resource leaks (THE BIG ONE!)
func checkResourceLeaks(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	// Track function scopes to check for proper cleanup
	var inFunction bool

	for lineNum, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Track function boundaries
		if strings.HasPrefix(trimmedLine, "func ") {
			inFunction = true
		}
		if trimmedLine == "}" && inFunction {
			inFunction = false
		}

		// Check 1: HTTP clients without timeouts (CRITICAL for port leaks)
		if strings.Contains(line, "http.Client{") && !strings.Contains(line, "Timeout") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "resource_leaks",
				Severity:       "critical",
				Title:          "HTTP Client Without Timeout",
				Description:    "HTTP clients without timeouts can cause connection leaks and port exhaustion",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Set Timeout: &http.Client{Timeout: 30 * time.Second}",
				Standard:       "Resource Management",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}

		// Check 2: HTTP responses without Body.Close()
		if strings.Contains(line, "http.Get(") || strings.Contains(line, "client.Do(") {
			// Look ahead to see if there's a defer resp.Body.Close()
			hasDefer := false
			for i := lineNum; i < len(lines) && i < lineNum+10; i++ {
				if strings.Contains(lines[i], "defer") && strings.Contains(lines[i], "Body.Close()") {
					hasDefer = true
					break
				}
			}
			if !hasDefer {
				violations = append(violations, StandardsViolation{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					Type:           "resource_leaks",
					Severity:       "critical",
					Title:          "HTTP Response Body Not Closed",
					Description:    "HTTP response bodies must be closed to prevent connection leaks",
					FilePath:       filePath,
					LineNumber:     lineNum + 1,
					CodeSnippet:    strings.TrimSpace(line),
					Recommendation: "Add: defer resp.Body.Close() after checking for errors",
					Standard:       "Resource Management",
					DiscoveredAt:   time.Now().Format(time.RFC3339),
				})
			}
		}

		// Check 3: File operations without defer Close()
		if strings.Contains(line, "os.Open(") || strings.Contains(line, "os.Create(") {
			hasDefer := false
			for i := lineNum; i < len(lines) && i < lineNum+10; i++ {
				if strings.Contains(lines[i], "defer") && strings.Contains(lines[i], ".Close()") {
					hasDefer = true
					break
				}
			}
			if !hasDefer {
				violations = append(violations, StandardsViolation{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					Type:           "resource_leaks",
					Severity:       "high",
					Title:          "File Handle Not Closed",
					Description:    "File handles must be closed to prevent resource leaks",
					FilePath:       filePath,
					LineNumber:     lineNum + 1,
					CodeSnippet:    strings.TrimSpace(line),
					Recommendation: "Add: defer file.Close() after checking for errors",
					Standard:       "Resource Management",
					DiscoveredAt:   time.Now().Format(time.RFC3339),
				})
			}
		}

		// Check 4: Database rows without Close()
		if strings.Contains(line, ".Query(") && !strings.Contains(line, "QueryRow") {
			hasDefer := false
			for i := lineNum; i < len(lines) && i < lineNum+10; i++ {
				if strings.Contains(lines[i], "defer") && strings.Contains(lines[i], "rows.Close()") {
					hasDefer = true
					break
				}
			}
			if !hasDefer {
				violations = append(violations, StandardsViolation{
					ID:             uuid.New().String(),
					ScenarioName:   scenarioName,
					Type:           "resource_leaks",
					Severity:       "high",
					Title:          "Database Rows Not Closed",
					Description:    "Database rows must be closed to prevent connection leaks",
					FilePath:       filePath,
					LineNumber:     lineNum + 1,
					CodeSnippet:    strings.TrimSpace(line),
					Recommendation: "Add: defer rows.Close() after checking for errors",
					Standard:       "Resource Management",
					DiscoveredAt:   time.Now().Format(time.RFC3339),
				})
			}
		}

		// Check 5: Goroutines without context or cancellation
		if strings.Contains(line, "go func(") && !strings.Contains(line, "context") {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "resource_leaks",
				Severity:       "medium",
				Title:          "Goroutine Without Context",
				Description:    "Goroutines should use context for proper cancellation",
				FilePath:       filePath,
				LineNumber:     lineNum + 1,
				CodeSnippet:    strings.TrimSpace(line),
				Recommendation: "Pass context.Context and check for cancellation",
				Standard:       "Resource Management",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
	}

	return violations
}

// checkHealthCheckImplementation validates health check endpoints
func checkHealthCheckImplementation(content []byte, filePath, scenarioName string) []StandardsViolation {
	var violations []StandardsViolation
	contentStr := string(content)

	hasHealthHandler := strings.Contains(contentStr, "healthHandler") || strings.Contains(contentStr, "/health")
	
	if hasHealthHandler {
		hasTimestamp := strings.Contains(contentStr, "timestamp")
		hasStatus := strings.Contains(contentStr, "status")
		
		if !hasTimestamp || !hasStatus {
			violations = append(violations, StandardsViolation{
				ID:             uuid.New().String(),
				ScenarioName:   scenarioName,
				Type:           "health_check",
				Severity:       "medium",
				Title:          "Incomplete Health Check",
				Description:    "Health checks should include status and timestamp",
				FilePath:       filePath,
				LineNumber:     1,
				Recommendation: "Include status and timestamp in health check response",
				Standard:       "Health Monitoring",
				DiscoveredAt:   time.Now().Format(time.RFC3339),
			})
		}
	}

	return violations
}

// Get standards violations handler
func getStandardsViolationsHandler(w http.ResponseWriter, r *http.Request) {
	scenario := r.URL.Query().Get("scenario")
	
	w.Header().Set("Content-Type", "application/json")
	
	// Get violations from memory store
	violations := standardsStore.GetViolations(scenario)
	
	response := map[string]interface{}{
		"violations": violations,
		"count":      len(violations),
		"timestamp":  time.Now().Format(time.RFC3339),
		"scenario":   scenario,
	}
	
	// Add a note if no violations found
	if len(violations) == 0 {
		response["note"] = "No standards violations found. Run a standards compliance check to detect violations."
	}
	
	json.NewEncoder(w).Encode(response)
}