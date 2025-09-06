package pipeline

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ValidationResult contains the validation outcome
type ValidationResult struct {
	Success          bool              `json:"success"`
	Errors           []string          `json:"errors"`
	Warnings         []string          `json:"warnings"`
	TestedComponents []string          `json:"tested_components"`
	ValidationScore  int               `json:"validation_score"`
	DeploymentReady  bool              `json:"deployment_ready"`
	Iterations       int               `json:"iterations"`
}

// ValidationMetrics tracks validation phase metrics
type ValidationMetrics struct {
	Iterations      int      `json:"iterations"`
	ErrorsFixed     int      `json:"errors_fixed"`
	ValidationScore int      `json:"validation_score"`
	TestsPassed     []string `json:"tests_passed"`
	TestsFailed     []string `json:"tests_failed"`
}

// Validator handles scenario validation
type Validator struct {
	vrooliRoot string
	tempDir    string
}

// NewValidator creates a new validator
func NewValidator() *Validator {
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = os.Getenv("HOME") + "/Vrooli"
	}
	
	return &Validator{
		vrooliRoot: vrooliRoot,
		tempDir:    "/tmp",
	}
}

// runValidationPhase validates and fixes the generated scenario
func (p *Pipeline) runValidationPhase(req GenerationRequest, files map[string]string) (ValidationResult, *ValidationMetrics, error) {
	startTime := time.Now()
	
	result := ValidationResult{
		Success:          false,
		Errors:           []string{},
		Warnings:         []string{},
		TestedComponents: []string{},
		ValidationScore:  0,
		DeploymentReady:  false,
		Iterations:       0,
	}
	
	metrics := &ValidationMetrics{
		Iterations:  0,
		ErrorsFixed: 0,
		TestsPassed: []string{},
		TestsFailed: []string{},
	}
	
	// Create temporary directory for validation
	tempDir := fmt.Sprintf("/tmp/scenario-validation-%s-%d", req.Name, time.Now().Unix())
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return result, metrics, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up after validation
	
	log.Printf("Starting validation in %s", tempDir)
	
	// Write files to temp directory
	if err := p.writeFilesToDisk(files, tempDir); err != nil {
		return result, metrics, fmt.Errorf("failed to write files: %w", err)
	}
	
	// Run validation iterations
	for iteration := 1; iteration <= req.Iterations.Validation; iteration++ {
		metrics.Iterations = iteration
		log.Printf("Validation iteration %d/%d", iteration, req.Iterations.Validation)
		
		// Run validation tests
		validationOutput, err := p.runValidationTests(tempDir, req)
		
		if err == nil {
			// Validation passed!
			result.Success = true
			result.ValidationScore = 100
			result.DeploymentReady = true
			result.TestedComponents = p.extractTestedComponents(validationOutput)
			metrics.TestsPassed = append(metrics.TestsPassed, "all")
			
			log.Printf("✅ Validation passed on iteration %d", iteration)
			break
		}
		
		// Parse validation errors
		errors := p.parseValidationErrors(validationOutput, err)
		result.Errors = errors
		
		log.Printf("Validation iteration %d found %d errors", iteration, len(errors))
		
		// Try to fix errors if we have more iterations
		if iteration < req.Iterations.Validation && len(errors) > 0 {
			log.Printf("Attempting to fix %d validation errors", len(errors))
			
			// Generate fixes using Claude
			fixes, fixErr := p.generateFixes(files, errors, req)
			if fixErr != nil {
				log.Printf("Failed to generate fixes: %v", fixErr)
				continue
			}
			
			// Apply fixes
			for filename, content := range fixes {
				files[filename] = content
				metrics.ErrorsFixed++
			}
			
			// Rewrite files with fixes
			if err := p.writeFilesToDisk(files, tempDir); err != nil {
				log.Printf("Failed to write fixed files: %v", err)
				continue
			}
			
			// Add delay between fix attempts
			time.Sleep(2 * time.Second)
		}
	}
	
	// Calculate final validation score
	if result.Success {
		result.ValidationScore = 100
		metrics.ValidationScore = 100
	} else {
		// Partial credit based on what works
		score := 100
		score -= len(result.Errors) * 10
		if score < 0 {
			score = 0
		}
		result.ValidationScore = score
		metrics.ValidationScore = score
	}
	
	log.Printf("Validation phase completed in %v with score %d", time.Since(startTime), result.ValidationScore)
	
	return result, metrics, nil
}

// writeFilesToDisk writes generated files to a directory
func (p *Pipeline) writeFilesToDisk(files map[string]string, baseDir string) error {
	for filename, content := range files {
		fullPath := filepath.Join(baseDir, filename)
		
		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		
		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
		
		// Make scripts executable
		if strings.HasSuffix(filename, ".sh") {
			os.Chmod(fullPath, 0755)
		}
	}
	
	return nil
}

// runValidationTests runs the actual validation tests
func (p *Pipeline) runValidationTests(tempDir string, req GenerationRequest) (string, error) {
	// Try different validation approaches
	
	// 1. Check if service.json exists and is valid
	serviceFile := filepath.Join(tempDir, "service.json")
	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		return "", fmt.Errorf("service.json not found")
	}
	
	// 2. Check file structure
	requiredDirs := []string{}
	if _, err := os.Stat(filepath.Join(tempDir, "api")); err == nil {
		requiredDirs = append(requiredDirs, "api")
	}
	if _, err := os.Stat(filepath.Join(tempDir, "ui")); err == nil {
		requiredDirs = append(requiredDirs, "ui")
	}
	
	// 3. Try to run test script if it exists
	testScript := filepath.Join(tempDir, "test.sh")
	if _, err := os.Stat(testScript); err == nil {
		cmd := exec.Command("bash", testScript)
		cmd.Dir = tempDir
		output, err := cmd.CombinedOutput()
		
		if err != nil {
			return string(output), fmt.Errorf("test script failed: %w\nOutput: %s", err, output)
		}
		
		return string(output), nil
	}
	
	// 4. Basic validation passed
	return "Basic validation checks passed", nil
}

// parseValidationErrors extracts errors from validation output
func (p *Pipeline) parseValidationErrors(output string, err error) []string {
	errors := []string{}
	
	// Add the main error if present
	if err != nil {
		errors = append(errors, err.Error())
	}
	
	// Look for common error patterns in output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for error indicators
		if strings.Contains(strings.ToLower(line), "error") ||
		   strings.Contains(strings.ToLower(line), "failed") ||
		   strings.Contains(line, "❌") ||
		   strings.Contains(line, "FAILED") {
			
			// Skip generic error lines
			if !strings.Contains(line, "Error:") || len(line) > 20 {
				errors = append(errors, line)
			}
		}
	}
	
	// Deduplicate errors
	seen := make(map[string]bool)
	unique := []string{}
	for _, err := range errors {
		if !seen[err] {
			seen[err] = true
			unique = append(unique, err)
		}
	}
	
	return unique
}

// generateFixes attempts to fix validation errors using Claude
func (p *Pipeline) generateFixes(files map[string]string, errors []string, req GenerationRequest) (map[string]string, error) {
	// Build a prompt for Claude to fix the errors
	codeBlockStart := "```"
	codeBlockEnd := "```"
	
	prompt := fmt.Sprintf("# Scenario Validation - Error Fixes Required\n\n"+
		"The generated scenario has validation errors that need to be fixed.\n\n"+
		"## Scenario: %s\n\n"+
		"## Validation Errors:\n%s\n\n"+
		"## Current Files:\n%s\n\n"+
		"## Requirements:\n"+
		"1. Analyze the validation errors\n"+
		"2. Generate ONLY the files that need to be fixed\n"+
		"3. Ensure fixes address the specific errors\n"+
		"4. Maintain compatibility with other files\n\n"+
		"Generate the fixed files using the format:\n"+
		"%sfilename\n"+
		"fixed content\n"+
		"%s\n\n"+
		"Only include files that need changes to fix the errors.",
		req.Name,
		strings.Join(errors, "\n- "),
		p.summarizeFiles(files),
		codeBlockStart,
		codeBlockEnd)
	
	// Call Claude for fixes
	fixOutput, err := p.claude.Chat(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fixes: %w", err)
	}
	
	// Parse fixed files
	fixes := p.claude.ParseCodeBlocks(fixOutput)
	
	log.Printf("Generated %d file fixes for validation errors", len(fixes))
	
	return fixes, nil
}

// extractTestedComponents identifies what was successfully tested
func (p *Pipeline) extractTestedComponents(output string) []string {
	components := []string{}
	
	outputLower := strings.ToLower(output)
	
	if strings.Contains(outputLower, "service.json") {
		components = append(components, "service-config")
	}
	if strings.Contains(outputLower, "database") || strings.Contains(outputLower, "schema") {
		components = append(components, "database")
	}
	if strings.Contains(outputLower, "api") {
		components = append(components, "api")
	}
	if strings.Contains(outputLower, "ui") || strings.Contains(outputLower, "interface") {
		components = append(components, "ui")
	}
	if strings.Contains(outputLower, "test") {
		components = append(components, "tests")
	}
	
	if len(components) == 0 {
		components = append(components, "basic-structure")
	}
	
	return components
}

// summarizeFiles creates a summary of current files for the fix prompt
func (p *Pipeline) summarizeFiles(files map[string]string) string {
	summary := []string{}
	
	for filename, content := range files {
		lines := strings.Split(content, "\n")
		preview := ""
		if len(lines) > 0 {
			preview = lines[0]
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
		}
		
		summary = append(summary, fmt.Sprintf("- %s (%d lines): %s", 
			filename, len(lines), preview))
		
		// Limit to 20 files in summary
		if len(summary) >= 20 {
			summary = append(summary, fmt.Sprintf("... and %d more files", len(files)-20))
			break
		}
	}
	
	return strings.Join(summary, "\n")
}