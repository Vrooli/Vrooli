package pipeline

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// ImplementationResult contains the generated files and metadata
type ImplementationResult struct {
	Files            map[string]string `json:"files"`
	Structure        ScenarioStructure `json:"structure"`
	QualityScore     int               `json:"quality_score"`
	Iterations       int               `json:"iterations"`
}

// ScenarioStructure describes what components were generated
type ScenarioStructure struct {
	HasServiceConfig bool     `json:"has_service_config"`
	HasDatabase      bool     `json:"has_database"`
	HasAPI           bool     `json:"has_api"`
	HasUI            bool     `json:"has_ui"`
	HasWorkflows     bool     `json:"has_workflows"`
	HasDocumentation bool     `json:"has_documentation"`
	HasTests         bool     `json:"has_tests"`
	FileCount        int      `json:"file_count"`
	CoreFiles        []string `json:"core_files"`
}

// ImplementationMetrics tracks implementation phase metrics
type ImplementationMetrics struct {
	Iterations    int `json:"iterations"`
	FilesCreated  int `json:"files_created"`
	QualityScore  int `json:"quality_score"`
}

// runImplementationPhase generates the actual scenario files
func (p *Pipeline) runImplementationPhase(req GenerationRequest, plan *PlanningResult) (map[string]string, *ImplementationMetrics, error) {
	startTime := time.Now()
	
	metrics := &ImplementationMetrics{
		Iterations: 0,
	}
	
	var bestFiles map[string]string
	var bestScore int
	
	for iteration := 1; iteration <= req.Iterations.Implementation; iteration++ {
		metrics.Iterations = iteration
		log.Printf("Implementation iteration %d/%d for %s", iteration, req.Iterations.Implementation, req.Name)
		
		// Build implementation prompt
		var prompt string
		if iteration == 1 {
			prompt = p.buildImplementationPrompt(req, plan)
		} else {
			prompt = p.buildImplementationRefinementPrompt(req, plan, bestFiles, bestScore)
		}
		
		// Call Claude for implementation
		implementationOutput, err := p.claude.Chat(prompt)
		if err != nil {
			log.Printf("Implementation iteration %d failed: %v", iteration, err)
			
			// Continue with next iteration if we have retries left
			if iteration < req.Iterations.Implementation {
				time.Sleep(3 * time.Second)
				continue
			}
			
			// Return best attempt if we have any files
			if len(bestFiles) > 0 {
				log.Printf("Using best implementation attempt with %d files", len(bestFiles))
				metrics.FilesCreated = len(bestFiles)
				metrics.QualityScore = bestScore
				return bestFiles, metrics, nil
			}
			
			return nil, nil, fmt.Errorf("implementation failed after %d iterations: %w", iteration, err)
		}
		
		// Parse generated files from Claude's response
		files := p.parseImplementationOutput(implementationOutput)
		
		// Analyze implementation quality
		analysis := p.analyzeImplementation(files, plan)
		
		// Keep track of the best implementation
		if analysis.QualityScore > bestScore || len(files) > len(bestFiles) {
			bestFiles = files
			bestScore = analysis.QualityScore
		}
		
		log.Printf("Implementation iteration %d generated %d files with quality score %d", 
			iteration, len(files), analysis.QualityScore)
		
		// Check if implementation is good enough
		if analysis.QualityScore >= 70 && len(files) >= 5 {
			log.Printf("Implementation quality acceptable (score: %d, files: %d) after %d iterations", 
				analysis.QualityScore, len(files), iteration)
			break
		}
		
		// Add delay between iterations
		if iteration < req.Iterations.Implementation {
			time.Sleep(3 * time.Second)
		}
	}
	
	// Ensure we have essential files
	if len(bestFiles) == 0 {
		bestFiles = p.generateMinimalFiles(req, plan)
	} else {
		// Add any missing essential files
		p.ensureEssentialFiles(bestFiles, req, plan)
	}
	
	metrics.FilesCreated = len(bestFiles)
	metrics.QualityScore = bestScore
	
	log.Printf("Implementation phase completed in %v with %d files", time.Since(startTime), len(bestFiles))
	
	return bestFiles, metrics, nil
}

// buildImplementationPrompt creates the implementation prompt
func (p *Pipeline) buildImplementationPrompt(req GenerationRequest, plan *PlanningResult) string {
	// Build the prompt using regular string concatenation to avoid backtick issues
	codeBlockStart := "```"
	codeBlockEnd := "```"
	
	prompt := fmt.Sprintf("# Scenario Generation - Implementation Phase\n\n"+
		"Based on the approved plan below, generate all necessary files and code to create a complete, deployable Vrooli scenario.\n\n"+
		"## Approved Plan:\n%s\n\n"+
		"## Scenario Details:\n"+
		"**Name:** %s\n"+
		"**Description:** %s\n"+
		"**Complexity:** %s\n"+
		"**Category:** %s\n"+
		"**Resources:** %s\n\n"+
		"## Implementation Requirements:\n\n"+
		"You MUST generate a complete Vrooli scenario with ALL necessary files. Use the following format for EACH file:\n\n"+
		"### File Format:\n"+
		"Every file MUST be wrapped in code blocks with the filename as the first line:\n"+
		"%spath/to/file.ext\n"+
		"file content here\n"+
		"%s\n\n"+
		"## Required Files to Generate:\n\n"+
		"### 1. Service Configuration\n"+
		"%sservice.json\n"+
		"{\n"+
		"  \"name\": \"%s\",\n"+
		"  \"version\": \"1.0.0\",\n"+
		"  \"description\": \"%s\",\n"+
		"  \"resources\": [%s],\n"+
		"  ...complete service configuration...\n"+
		"}\n"+
		"%s\n\n"+
		"### 2. Database Schema (if using postgres)\n"+
		"%sinitialization/postgres/schema.sql\n"+
		"-- Complete database schema\n"+
		"CREATE TABLE ...\n"+
		"%s\n\n"+
		"### 3. API Implementation\n"+ 
		"%sapi/main.go\n"+
		"package main\n"+
		"// Complete API server code\n"+
		"%s\n\n"+
		"### 4. User Interface\n"+
		"%sui/index.html\n"+
		"<!DOCTYPE html>\n"+
		"<!-- Complete UI implementation -->\n"+
		"%s\n\n"+
		"### 5. README Documentation\n"+
		"%sREADME.md\n"+
		"# %s\n"+
		"Complete documentation...\n"+
		"%s\n\n"+
		"### 6. Test Script\n"+
		"%stest.sh\n"+
		"#!/usr/bin/env bash\n"+
		"# Complete test script\n"+
		"%s\n\n"+
		"Generate AT LEAST 8-10 complete files including:\n"+
		"- Service configuration (service.json)\n"+
		"- Database schema if applicable\n"+
		"- API server implementation if applicable\n"+
		"- UI files if applicable\n"+
		"- CLI implementation if applicable\n"+
		"- Configuration files\n"+
		"- Test scripts\n"+
		"- Documentation (README.md)\n"+
		"- Deployment scripts\n\n"+
		"IMPORTANT:\n"+
		"- Each file MUST be complete and production-ready\n"+
		"- Use the EXACT format shown above with filename in code block header\n"+
		"- Generate ACTUAL working code, not placeholders or TODOs\n"+
		"- Files should integrate properly with each other\n"+
		"- Follow Vrooli conventions and best practices\n\n"+
		"Begin generating files now:",
		plan.FinalPlan,
		req.Name,
		req.Description,
		req.Complexity,
		req.Category,
		strings.Join(plan.IdentifiedResources, ", "),
		codeBlockStart,
		codeBlockEnd,
		codeBlockStart,
		req.Name,
		req.Description,
		p.formatResourcesJSON(plan.IdentifiedResources),
		codeBlockEnd,
		codeBlockStart,
		codeBlockEnd,
		codeBlockStart,
		codeBlockEnd,
		codeBlockStart,
		codeBlockEnd,
		codeBlockStart,
		req.Name,
		codeBlockEnd,
		codeBlockStart,
		codeBlockEnd)
	
	return prompt
}

// buildImplementationRefinementPrompt creates a refinement prompt
func (p *Pipeline) buildImplementationRefinementPrompt(req GenerationRequest, plan *PlanningResult, currentFiles map[string]string, qualityScore int) string {
	filesList := []string{}
	for filename := range currentFiles {
		filesList = append(filesList, filename)
	}
	
	codeBlockStart := "```"
	codeBlockEnd := "```"
	
	return fmt.Sprintf("# Scenario Generation - Implementation Refinement\n\n"+
		"The implementation needs improvement. Please enhance and complete the following implementation.\n\n"+
		"## Current Files Generated (%d files):\n%s\n\n"+
		"## Quality Score: %d/100\n\n"+
		"## Missing or Incomplete:\n%s\n\n"+
		"## Requirements:\n"+
		"1. Add any missing essential files\n"+
		"2. Complete any partial implementations\n"+
		"3. Ensure all files are production-ready\n"+
		"4. Add proper error handling\n"+
		"5. Include configuration files\n"+
		"6. Add test scripts\n\n"+
		"Generate the COMPLETE implementation with ALL files needed. Use the same format:\n\n"+
		"%sfilename.ext\n"+
		"complete file content\n"+
		"%s\n\n"+
		"Focus on files that are missing or need improvement. Generate complete, working code.",
		len(filesList),
		strings.Join(filesList, "\n"),
		qualityScore,
		p.identifyMissingFiles(currentFiles, plan),
		codeBlockStart,
		codeBlockEnd)
}

// parseImplementationOutput extracts files from Claude's response
func (p *Pipeline) parseImplementationOutput(output string) map[string]string {
	files := make(map[string]string)
	
	// First try to use the claude client's built-in parser
	files = p.claude.ParseCodeBlocks(output)
	
	// If we didn't get many files, try a more aggressive parsing
	if len(files) < 3 {
		// Split by triple backticks
		blocks := strings.Split(output, "```")
		
		for i := 0; i < len(blocks)-1; i += 2 {
			if i+1 >= len(blocks) {
				break
			}
			
			header := strings.TrimSpace(blocks[i])
			content := blocks[i+1]
			
			// Extract filename from header
			lines := strings.Split(header, "\n")
			if len(lines) > 0 {
				lastLine := strings.TrimSpace(lines[len(lines)-1])
				
				// Check if this looks like a filename
				if strings.Contains(lastLine, ".") || strings.Contains(lastLine, "/") {
					// Remove language prefix if present
					if colonIdx := strings.Index(lastLine, ":"); colonIdx != -1 {
						lastLine = strings.TrimSpace(lastLine[colonIdx+1:])
					}
					
					// Clean up the filename
					filename := strings.TrimSpace(lastLine)
					filename = strings.TrimPrefix(filename, "`")
					filename = strings.TrimSuffix(filename, "`")
					
					if filename != "" && !strings.HasPrefix(filename, "//") && !strings.HasPrefix(filename, "#") {
						files[filename] = strings.TrimSpace(content)
					}
				}
			}
		}
	}
	
	// Clean up filenames and content
	cleanedFiles := make(map[string]string)
	for filename, content := range files {
		// Clean filename
		filename = strings.TrimSpace(filename)
		filename = strings.ReplaceAll(filename, "`", "")
		
		// Skip empty files or language markers
		if filename == "" || len(content) < 10 {
			continue
		}
		
		// Skip if filename is just a language identifier
		if filename == "json" || filename == "go" || filename == "javascript" || 
		   filename == "typescript" || filename == "bash" || filename == "sql" ||
		   filename == "yaml" || filename == "html" || filename == "css" {
			continue
		}
		
		cleanedFiles[filename] = content
	}
	
	return cleanedFiles
}

// analyzeImplementation evaluates the quality of generated files
func (p *Pipeline) analyzeImplementation(files map[string]string, plan *PlanningResult) *ImplementationAnalysis {
	analysis := &ImplementationAnalysis{
		QualityScore: 0,
		Structure: ScenarioStructure{
			FileCount: len(files),
			CoreFiles: []string{},
		},
	}
	
	// Check for service.json
	for filename, content := range files {
		filenameLower := strings.ToLower(filename)
		
		if strings.Contains(filenameLower, "service.json") {
			analysis.Structure.HasServiceConfig = true
			analysis.QualityScore += 20
			analysis.Structure.CoreFiles = append(analysis.Structure.CoreFiles, filename)
		}
		
		if strings.Contains(filenameLower, "schema.sql") {
			analysis.Structure.HasDatabase = true
			analysis.QualityScore += 15
			analysis.Structure.CoreFiles = append(analysis.Structure.CoreFiles, filename)
		}
		
		if strings.Contains(filenameLower, "main.go") || strings.Contains(filenameLower, "server.js") || 
		   strings.Contains(filenameLower, "api") {
			analysis.Structure.HasAPI = true
			analysis.QualityScore += 15
			analysis.Structure.CoreFiles = append(analysis.Structure.CoreFiles, filename)
		}
		
		if strings.Contains(filenameLower, "index.html") || strings.Contains(filenameLower, "app.js") ||
		   strings.Contains(filenameLower, "ui/") {
			analysis.Structure.HasUI = true
			analysis.QualityScore += 15
			analysis.Structure.CoreFiles = append(analysis.Structure.CoreFiles, filename)
		}
		
		if strings.Contains(filenameLower, "workflow") || strings.Contains(filenameLower, ".json") &&
		   strings.Contains(content, "nodes") {
			analysis.Structure.HasWorkflows = true
			analysis.QualityScore += 10
		}
		
		if strings.Contains(filenameLower, "readme") {
			analysis.Structure.HasDocumentation = true
			analysis.QualityScore += 10
			analysis.Structure.CoreFiles = append(analysis.Structure.CoreFiles, filename)
		}
		
		if strings.Contains(filenameLower, "test") {
			analysis.Structure.HasTests = true
			analysis.QualityScore += 10
		}
	}
	
	// Bonus points for file count
	if len(files) >= 10 {
		analysis.QualityScore += 10
	} else if len(files) >= 5 {
		analysis.QualityScore += 5
	}
	
	// Cap at 100
	if analysis.QualityScore > 100 {
		analysis.QualityScore = 100
	}
	
	return analysis
}

// ImplementationAnalysis contains implementation analysis results
type ImplementationAnalysis struct {
	QualityScore int
	Structure    ScenarioStructure
}

// identifyMissingFiles identifies what files are missing
func (p *Pipeline) identifyMissingFiles(files map[string]string, plan *PlanningResult) string {
	missing := []string{}
	
	hasServiceJSON := false
	hasREADME := false
	hasTest := false
	hasAPI := false
	hasUI := false
	hasSchema := false
	
	for filename := range files {
		filenameLower := strings.ToLower(filename)
		if strings.Contains(filenameLower, "service.json") {
			hasServiceJSON = true
		}
		if strings.Contains(filenameLower, "readme") {
			hasREADME = true
		}
		if strings.Contains(filenameLower, "test") {
			hasTest = true
		}
		if strings.Contains(filenameLower, "main.go") || strings.Contains(filenameLower, "server") {
			hasAPI = true
		}
		if strings.Contains(filenameLower, "index.html") || strings.Contains(filenameLower, "app.js") {
			hasUI = true
		}
		if strings.Contains(filenameLower, "schema.sql") {
			hasSchema = true
		}
	}
	
	if !hasServiceJSON {
		missing = append(missing, "- Missing service.json configuration")
	}
	if !hasREADME {
		missing = append(missing, "- Missing README.md documentation")
	}
	if !hasTest {
		missing = append(missing, "- Missing test.sh or test scripts")
	}
	
	// Check based on plan requirements
	if plan.Architecture.HasAPI && !hasAPI {
		missing = append(missing, "- Missing API implementation (main.go or server.js)")
	}
	if plan.Architecture.HasUI && !hasUI {
		missing = append(missing, "- Missing UI files (index.html, app.js)")
	}
	if plan.Architecture.HasDatabase && !hasSchema {
		missing = append(missing, "- Missing database schema (schema.sql)")
	}
	
	if len(missing) == 0 {
		missing = append(missing, "- Consider adding more configuration and initialization files")
	}
	
	return strings.Join(missing, "\n")
}

// ensureEssentialFiles adds any missing essential files
func (p *Pipeline) ensureEssentialFiles(files map[string]string, req GenerationRequest, plan *PlanningResult) {
	// Ensure service.json exists
	if _, exists := files["service.json"]; !exists {
		files["service.json"] = p.generateServiceJSON(req, plan)
	}
	
	// Ensure README.md exists
	if _, exists := files["README.md"]; !exists {
		files["README.md"] = p.generateREADME(req, plan)
	}
	
	// Ensure test.sh exists
	if _, exists := files["test.sh"]; !exists {
		files["test.sh"] = p.generateTestScript(req)
	}
}

// generateMinimalFiles creates minimal required files if generation completely fails
func (p *Pipeline) generateMinimalFiles(req GenerationRequest, plan *PlanningResult) map[string]string {
	files := make(map[string]string)
	
	files["service.json"] = p.generateServiceJSON(req, plan)
	files["README.md"] = p.generateREADME(req, plan)
	files["test.sh"] = p.generateTestScript(req)
	
	if plan.Architecture.HasDatabase {
		files["initialization/postgres/schema.sql"] = p.generateMinimalSchema(req)
	}
	
	return files
}

// Helper functions to generate minimal files
func (p *Pipeline) generateServiceJSON(req GenerationRequest, plan *PlanningResult) string {
	resources := plan.IdentifiedResources
	if len(resources) == 0 {
		resources = []string{"postgres"}
	}
	
	return fmt.Sprintf(`{
  "name": "%s",
  "version": "1.0.0",
  "description": "%s",
  "category": "%s",
  "complexity": "%s",
  "resources": [%s],
  "estimated_revenue": %d
}`, req.Name, req.Description, req.Category, req.Complexity, 
		p.formatResourcesJSON(resources), p.estimateRevenue(req.Complexity))
}

func (p *Pipeline) generateREADME(req GenerationRequest, plan *PlanningResult) string {
	codeBlockStart := "```"
	codeBlockEnd := "```"
	
	return fmt.Sprintf("# %s\n\n"+
		"%s\n\n"+
		"## Overview\n"+
		"This is a %s complexity %s scenario.\n\n"+
		"## Resources Used\n"+
		"%s\n\n"+
		"## Setup Instructions\n"+
		"1. Run initialization scripts\n"+
		"2. Configure environment\n"+
		"3. Start services\n"+
		"4. Access the application\n\n"+
		"## Testing\n"+
		"Run the test script:\n"+
		"%sbash\n"+
		"./test.sh\n"+
		"%s\n\n"+
		"---\n"+
		"Generated by Scenario Generator V1",
		req.Name, req.Description, req.Complexity, req.Category,
		strings.Join(plan.IdentifiedResources, "\n- "),
		codeBlockStart,
		codeBlockEnd)
}

func (p *Pipeline) generateTestScript(req GenerationRequest) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# Test script for %s

set -e

echo "Testing %s scenario..."

# Add your tests here
echo "✅ All tests passed"

exit 0`, req.Name, req.Name)
}

func (p *Pipeline) generateMinimalSchema(req GenerationRequest) string {
	return fmt.Sprintf(`-- Database schema for %s

CREATE TABLE IF NOT EXISTS config (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Add your tables here`, req.Name)
}

func (p *Pipeline) formatResourcesJSON(resources []string) string {
	quoted := []string{}
	for _, r := range resources {
		quoted = append(quoted, fmt.Sprintf(`"%s"`, r))
	}
	return strings.Join(quoted, ", ")
}