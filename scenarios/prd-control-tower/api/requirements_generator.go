package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// RequirementsGenerateRequest represents a request to generate requirements from PRD
type RequirementsGenerateRequest struct {
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
	Context    string `json:"context,omitempty"` // Additional context for AI generation
	Model      string `json:"model,omitempty"`   // Override OpenRouter model
}

// RequirementsGenerateResponse represents the result of requirements generation
type RequirementsGenerateResponse struct {
	EntityType       string   `json:"entity_type"`
	EntityName       string   `json:"entity_name"`
	Success          bool     `json:"success"`
	Message          string   `json:"message,omitempty"`
	RequirementCount int      `json:"requirement_count"`
	P0Count          int      `json:"p0_count"`
	P1Count          int      `json:"p1_count"`
	P2Count          int      `json:"p2_count"`
	FilesCreated     []string `json:"files_created"`
	Model            string   `json:"model,omitempty"`
	GeneratedAt      string   `json:"generated_at"`
}

// GeneratedRequirement represents a single requirement from AI generation
type GeneratedRequirement struct {
	ID          string                      `json:"id"`
	Title       string                      `json:"title"`
	Description string                      `json:"description"`
	PRDRef      string                      `json:"prd_ref"`
	Criticality string                      `json:"criticality"`
	Category    string                      `json:"category"`
	Status      string                      `json:"status"`
	Validation  GeneratedValidation         `json:"validation"`
	Dependencies []string                   `json:"dependencies,omitempty"`
}

// GeneratedValidation represents validation phases for a requirement
type GeneratedValidation struct {
	Unit        []string `json:"unit,omitempty"`
	Integration []string `json:"integration,omitempty"`
	Performance []string `json:"performance,omitempty"`
	Business    []string `json:"business,omitempty"`
}

// AIRequirementsResponse represents the structured response from AI
type AIRequirementsResponse struct {
	Requirements []GeneratedRequirement `json:"requirements"`
}

func handleRequirementsGenerate(w http.ResponseWriter, r *http.Request) {
	var req RequirementsGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	req.EntityType = strings.TrimSpace(req.EntityType)
	req.EntityName = strings.TrimSpace(req.EntityName)

	if !isValidEntityType(req.EntityType) {
		respondJSON(w, http.StatusBadRequest, RequirementsGenerateResponse{
			Success: false,
			Message: "Invalid entity type. Must be 'scenario' or 'resource'",
		})
		return
	}
	if req.EntityName == "" {
		respondJSON(w, http.StatusBadRequest, RequirementsGenerateResponse{
			Success: false,
			Message: "entity_name is required",
		})
		return
	}

	response, status := executeRequirementsGenerate(req)
	respondJSON(w, status, response)
}

func executeRequirementsGenerate(req RequirementsGenerateRequest) (RequirementsGenerateResponse, int) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return RequirementsGenerateResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to get Vrooli root: %v", err),
		}, http.StatusInternalServerError
	}

	// Step 1: Extract operational targets from PRD
	targets, err := extractOperationalTargets(req.EntityType, req.EntityName)
	if err != nil {
		return RequirementsGenerateResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to extract operational targets from PRD: %v", err),
		}, http.StatusBadRequest
	}

	if len(targets) == 0 {
		return RequirementsGenerateResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    "No operational targets found in PRD. Ensure PRD has 🎯 Operational Targets section with P0/P1/P2 items.",
		}, http.StatusBadRequest
	}

	openrouterURL := os.Getenv("RESOURCE_OPENROUTER_URL")
	if openrouterURL == "" {
		openrouterURL = "https://openrouter.ai/api/v1"
	}

	// Step 2: Initial AI generation pass
	prompt := buildRequirementsPrompt(req.EntityType, req.EntityName, targets, req.Context)
	aiResponse, usedModel, err := openRouterChatCompletionWithTimeout(openrouterURL, prompt, req.Model, 8000, 300)
	if err != nil {
		return RequirementsGenerateResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("AI generation failed: %v", err),
		}, http.StatusInternalServerError
	}

	requirements, err := parseAIRequirementsResponse(aiResponse)
	if err != nil {
		return RequirementsGenerateResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to parse AI response: %v", err),
			Model:      usedModel,
		}, http.StatusInternalServerError
	}

	// Step 3: Check for uncovered P0/P1 targets and generate iteratively
	uncoveredTargets := findUncoveredTargets(targets, requirements, true) // P0/P1 only
	iterationCount := 0
	maxIterations := 3 // Prevent infinite loops

	for len(uncoveredTargets) > 0 && iterationCount < maxIterations {
		iterationCount++

		// Build focused prompt for uncovered targets
		followUpPrompt := buildFollowUpRequirementsPrompt(req.EntityType, req.EntityName, uncoveredTargets, requirements, req.Context)
		followUpResponse, _, err := openRouterChatCompletionWithTimeout(openrouterURL, followUpPrompt, req.Model, 4000, 180)
		if err != nil {
			// Log but don't fail - we have partial results
			break
		}

		newReqs, err := parseAIRequirementsResponse(followUpResponse)
		if err != nil {
			break
		}

		// Merge new requirements, avoiding duplicates
		requirements = mergeRequirements(requirements, newReqs)

		// Check again for uncovered targets
		uncoveredTargets = findUncoveredTargets(targets, requirements, true)
	}

	// Step 4: Write requirements files
	var entityDir string
	if req.EntityType == EntityTypeScenario {
		entityDir = "scenarios"
	} else {
		entityDir = "resources"
	}
	requirementsDir := filepath.Join(vrooliRoot, entityDir, req.EntityName, "requirements")

	filesCreated, err := writeRequirementsFiles(requirementsDir, requirements, req.EntityName, targets)
	if err != nil {
		return RequirementsGenerateResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to write requirements files: %v", err),
			Model:      usedModel,
		}, http.StatusInternalServerError
	}

	// Count by criticality
	p0, p1, p2 := 0, 0, 0
	for _, req := range requirements {
		switch strings.ToUpper(req.Criticality) {
		case "P0":
			p0++
		case "P1":
			p1++
		case "P2":
			p2++
		}
	}

	// Calculate coverage
	coveredTargets := len(targets) - len(findUncoveredTargets(targets, requirements, false))
	message := fmt.Sprintf("Generated %d requirements covering %d/%d operational targets", len(requirements), coveredTargets, len(targets))
	if iterationCount > 0 {
		message += fmt.Sprintf(" (%d follow-up iterations)", iterationCount)
	}

	return RequirementsGenerateResponse{
		EntityType:       req.EntityType,
		EntityName:       req.EntityName,
		Success:          true,
		Message:          message,
		RequirementCount: len(requirements),
		P0Count:          p0,
		P1Count:          p1,
		P2Count:          p2,
		FilesCreated:     filesCreated,
		Model:            usedModel,
		GeneratedAt:      time.Now().Format(time.RFC3339),
	}, http.StatusOK
}

// findUncoveredTargets returns targets that don't have any requirements linked to them
func findUncoveredTargets(targets []OperationalTarget, requirements []GeneratedRequirement, p0p1Only bool) []OperationalTarget {
	// Build set of covered target IDs
	coveredIDs := make(map[string]bool)
	for _, req := range requirements {
		coveredIDs[req.PRDRef] = true
	}

	var uncovered []OperationalTarget
	for _, target := range targets {
		if coveredIDs[target.ID] {
			continue
		}
		// If p0p1Only, skip P2 targets
		if p0p1Only && strings.ToUpper(target.Criticality) == "P2" {
			continue
		}
		uncovered = append(uncovered, target)
	}
	return uncovered
}

// buildFollowUpRequirementsPrompt creates a focused prompt for uncovered targets
func buildFollowUpRequirementsPrompt(entityType, entityName string, uncoveredTargets []OperationalTarget, existingReqs []GeneratedRequirement, context string) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert requirements engineer. Generate requirements for the SPECIFIC operational targets listed below.\n\n")
	prompt.WriteString(fmt.Sprintf("Entity: %s/%s\n\n", entityType, entityName))

	prompt.WriteString("## UNCOVERED Operational Targets (MUST generate requirements for ALL of these)\n\n")
	for _, t := range uncoveredTargets {
		prompt.WriteString(fmt.Sprintf("- %s | %s | %s\n", t.ID, t.Title, t.Notes))
	}
	prompt.WriteString("\n")

	// Show existing requirement IDs to avoid duplicates
	if len(existingReqs) > 0 {
		prompt.WriteString("## Already Generated Requirement IDs (DO NOT duplicate these)\n")
		for _, req := range existingReqs {
			prompt.WriteString(fmt.Sprintf("- %s (for %s)\n", req.ID, req.PRDRef))
		}
		prompt.WriteString("\n")
	}

	// Find the highest existing requirement number for each criticality
	maxNums := map[string]int{"P0": 0, "P1": 0, "P2": 0}
	for _, req := range existingReqs {
		crit := strings.ToUpper(req.Criticality)
		// Extract number from ID like REQ-P0-005
		parts := strings.Split(req.ID, "-")
		if len(parts) >= 3 {
			var num int
			fmt.Sscanf(parts[2], "%d", &num)
			if num > maxNums[crit] {
				maxNums[crit] = num
			}
		}
	}

	prompt.WriteString(fmt.Sprintf("## Starting Requirement Numbers\n"))
	prompt.WriteString(fmt.Sprintf("- P0: Start from REQ-P0-%03d\n", maxNums["P0"]+1))
	prompt.WriteString(fmt.Sprintf("- P1: Start from REQ-P1-%03d\n", maxNums["P1"]+1))
	prompt.WriteString(fmt.Sprintf("- P2: Start from REQ-P2-%03d\n\n", maxNums["P2"]+1))

	if strings.TrimSpace(context) != "" {
		prompt.WriteString("## Context\n")
		prompt.WriteString(context + "\n\n")
	}

	prompt.WriteString(`## Task

Generate requirements ONLY for the uncovered targets listed above. Return valid JSON:

{
  "requirements": [
    {
      "id": "REQ-{CRITICALITY}-{NUMBER}",
      "title": "Short descriptive title",
      "description": "Detailed description",
      "prd_ref": "<EXACT-TARGET-ID-FROM-ABOVE>",
      "criticality": "P0|P1|P2",
      "category": "core|api|cli|ui|integration|performance|security",
      "status": "draft",
      "validation": {
        "unit": ["Test case 1"],
        "integration": ["Integration scenario"],
        "performance": ["Performance criteria"],
        "business": ["Business criteria"]
      },
      "dependencies": []
    }
  ]
}

CRITICAL:
- Generate at least ONE requirement for EACH uncovered target
- The prd_ref MUST be copied EXACTLY from the target ID (first value before |) - do NOT modify or slugify it
- Use the starting numbers provided above to avoid ID collisions
- Return ONLY JSON, no explanations
`)

	return prompt.String()
}

// mergeRequirements combines existing and new requirements, avoiding duplicates
func mergeRequirements(existing, new []GeneratedRequirement) []GeneratedRequirement {
	existingIDs := make(map[string]bool)
	existingPRDRefs := make(map[string]bool)
	for _, req := range existing {
		existingIDs[req.ID] = true
		existingPRDRefs[req.PRDRef] = true
	}

	merged := append([]GeneratedRequirement{}, existing...)
	for _, req := range new {
		// Skip if we already have this ID or already covered this target
		if existingIDs[req.ID] {
			continue
		}
		// Allow multiple requirements per target, just not duplicate IDs
		merged = append(merged, req)
		existingIDs[req.ID] = true
	}
	return merged
}

func buildRequirementsPrompt(entityType, entityName string, targets []OperationalTarget, context string) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert requirements engineer. Generate detailed requirements from operational targets.\n\n")
	prompt.WriteString(fmt.Sprintf("Entity Type: %s\n", entityType))
	prompt.WriteString(fmt.Sprintf("Entity Name: %s\n\n", entityName))

	prompt.WriteString("## Operational Targets from PRD\n\n")

	// Group targets by criticality
	p0Targets := []OperationalTarget{}
	p1Targets := []OperationalTarget{}
	p2Targets := []OperationalTarget{}

	for _, target := range targets {
		switch strings.ToUpper(target.Criticality) {
		case "P0":
			p0Targets = append(p0Targets, target)
		case "P1":
			p1Targets = append(p1Targets, target)
		case "P2":
			p2Targets = append(p2Targets, target)
		}
	}

	writeTargetGroup := func(header string, targets []OperationalTarget) {
		if len(targets) == 0 {
			return
		}
		prompt.WriteString(header + "\n")
		for _, t := range targets {
			prompt.WriteString(fmt.Sprintf("- %s | %s | %s\n", t.ID, t.Title, t.Notes))
		}
		prompt.WriteString("\n")
	}

	writeTargetGroup("### P0 - Critical", p0Targets)
	writeTargetGroup("### P1 - Important", p1Targets)
	writeTargetGroup("### P2 - Nice to Have", p2Targets)

	if strings.TrimSpace(context) != "" {
		prompt.WriteString("## Additional Context\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n")
		prompt.WriteString(context)
		prompt.WriteString("\n")
		prompt.WriteString("=" + strings.Repeat("=", 70) + "\n\n")
	}

	prompt.WriteString(`## Task

Generate a JSON array of requirements. Each operational target may produce one or more requirements.

Requirements:
- Generate 1-3 requirements per operational target (more for complex targets)
- Each requirement must have a unique ID following the pattern: REQ-{CRITICALITY}-{3-digit-number} (e.g., REQ-P0-001)
- CRITICAL: The prd_ref field MUST contain the EXACT target ID from the list above (the first value before the | in each target line)
- Include specific, testable validation criteria for each phase
- Dependencies should reference other requirement IDs if applicable

## Output Format

Return ONLY valid JSON (no markdown code fences, no explanations):

{
  "requirements": [
    {
      "id": "REQ-P0-001",
      "title": "Short descriptive title",
      "description": "Detailed description of what this requirement entails",
      "prd_ref": "<EXACT-TARGET-ID-FROM-ABOVE>",
      "criticality": "P0",
      "category": "core|api|cli|ui|integration|performance|security",
      "status": "draft",
      "validation": {
        "unit": ["Test case 1", "Test case 2"],
        "integration": ["Integration test scenario"],
        "performance": ["Performance criteria if applicable"],
        "business": ["Business validation criteria"]
      },
      "dependencies": ["REQ-P0-002"]
    }
  ]
}

CRITICAL INSTRUCTIONS:
- Return ONLY the JSON object, no additional text
- Ensure valid JSON syntax
- The prd_ref MUST be copied EXACTLY from the target ID (first value before |) - do NOT modify, slugify, or create new IDs
- Generate requirements for ALL operational targets listed above
`)

	return prompt.String()
}

func parseAIRequirementsResponse(response string) ([]GeneratedRequirement, error) {
	// Clean up response - remove markdown code fences if present
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) >= 3 {
			// Find start and end of JSON
			start := 0
			for i, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "```") {
					start = i + 1
					break
				}
			}
			end := len(lines) - 1
			for i := len(lines) - 1; i >= 0; i-- {
				if strings.TrimSpace(lines[i]) == "```" {
					end = i
					break
				}
			}
			if start < end {
				response = strings.Join(lines[start:end], "\n")
			}
		}
	}

	// Try to find JSON object in response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	if jsonStart >= 0 && jsonEnd > jsonStart {
		response = response[jsonStart : jsonEnd+1]
	}

	var aiResp AIRequirementsResponse
	if err := json.Unmarshal([]byte(response), &aiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w\nResponse was: %s", err, truncateString(response, 500))
	}

	if len(aiResp.Requirements) == 0 {
		return nil, fmt.Errorf("no requirements found in AI response")
	}

	return aiResp.Requirements, nil
}

func writeRequirementsFiles(requirementsDir string, requirements []GeneratedRequirement, entityName string, targets []OperationalTarget) ([]string, error) {
	// Create requirements directory if it doesn't exist
	if err := os.MkdirAll(requirementsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create requirements directory: %w", err)
	}

	var filesCreated []string

	// Group requirements by target for folder organization
	targetToReqs := make(map[string][]GeneratedRequirement)
	for _, req := range requirements {
		targetToReqs[req.PRDRef] = append(targetToReqs[req.PRDRef], req)
	}

	// Create module folders and files
	moduleIndex := 1
	moduleMeta := []map[string]any{}

	for _, target := range targets {
		reqs, ok := targetToReqs[target.ID]
		if !ok || len(reqs) == 0 {
			continue
		}

		// Create folder name from target
		folderName := fmt.Sprintf("%02d-%s", moduleIndex, sanitizeFolderName(target.Title))
		moduleDir := filepath.Join(requirementsDir, folderName)

		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			return filesCreated, fmt.Errorf("failed to create module directory %s: %w", folderName, err)
		}

		// Create module.json
		moduleFile := filepath.Join(moduleDir, "module.json")
		moduleData := map[string]any{
			"module_id":    fmt.Sprintf("MOD-%s-%03d", strings.ToUpper(target.Criticality), moduleIndex),
			"title":        target.Title,
			"priority":     target.Criticality,
			"description":  target.Notes,
			"prd_ref":      target.ID,
			"requirements": convertRequirementsToJSON(reqs),
		}

		moduleJSON, err := json.MarshalIndent(moduleData, "", "  ")
		if err != nil {
			return filesCreated, fmt.Errorf("failed to marshal module.json: %w", err)
		}

		if err := os.WriteFile(moduleFile, moduleJSON, 0644); err != nil {
			return filesCreated, fmt.Errorf("failed to write module.json: %w", err)
		}
		filesCreated = append(filesCreated, moduleFile)

		// Track module for index.json
		moduleMeta = append(moduleMeta, map[string]any{
			"id":       moduleData["module_id"],
			"path":     folderName,
			"priority": target.Criticality,
			"title":    target.Title,
		})

		moduleIndex++
	}

	// Create index.json with imports array for the loader
	indexFile := filepath.Join(requirementsDir, "index.json")
	imports := make([]string, 0, len(moduleMeta))
	for _, meta := range moduleMeta {
		imports = append(imports, meta["path"].(string)+"/module.json")
	}
	indexData := map[string]any{
		"_metadata": map[string]any{
			"generated_at":    time.Now().Format(time.RFC3339),
			"generator":       "prd-control-tower",
			"entity":          entityName,
			"schema_version":  "1.0",
		},
		"imports": imports,
		"modules": moduleMeta,
	}

	indexJSON, err := json.MarshalIndent(indexData, "", "  ")
	if err != nil {
		return filesCreated, fmt.Errorf("failed to marshal index.json: %w", err)
	}

	if err := os.WriteFile(indexFile, indexJSON, 0644); err != nil {
		return filesCreated, fmt.Errorf("failed to write index.json: %w", err)
	}
	filesCreated = append(filesCreated, indexFile)

	// Create README.md
	readmeFile := filepath.Join(requirementsDir, "README.md")
	readmeContent := generateRequirementsReadme(entityName, requirements, targets)
	if err := os.WriteFile(readmeFile, []byte(readmeContent), 0644); err != nil {
		return filesCreated, fmt.Errorf("failed to write README.md: %w", err)
	}
	filesCreated = append(filesCreated, readmeFile)

	return filesCreated, nil
}

func convertRequirementsToJSON(reqs []GeneratedRequirement) []map[string]any {
	result := make([]map[string]any, len(reqs))
	for i, req := range reqs {
		validation := []map[string]any{}

		addValidation := func(phase string, items []string) {
			for _, item := range items {
				validation = append(validation, map[string]any{
					"type":   "test",
					"ref":    "",
					"phase":  phase,
					"status": "pending",
					"notes":  item,
				})
			}
		}

		addValidation("unit", req.Validation.Unit)
		addValidation("integration", req.Validation.Integration)
		addValidation("performance", req.Validation.Performance)
		addValidation("business", req.Validation.Business)

		result[i] = map[string]any{
			"id":          req.ID,
			"title":       req.Title,
			"description": req.Description,
			"prd_ref":     req.PRDRef,
			"category":    req.Category,
			"status":      req.Status,
			"validation":  validation,
		}

		if len(req.Dependencies) > 0 {
			result[i]["dependencies"] = req.Dependencies
		}
	}
	return result
}

func generateRequirementsReadme(entityName string, requirements []GeneratedRequirement, targets []OperationalTarget) string {
	var readme strings.Builder

	readme.WriteString(fmt.Sprintf("# Requirements Registry: %s\n\n", entityName))

	readme.WriteString("## Overview\n\n")
	readme.WriteString("This directory contains the requirements definitions for this scenario, organized by operational target.\n\n")

	readme.WriteString("## Structure\n\n")
	readme.WriteString("```\n")
	readme.WriteString("requirements/\n")
	readme.WriteString("├── index.json           # Module registry\n")
	readme.WriteString("├── README.md            # This file\n")
	readme.WriteString("└── XX-target-name/      # Per-target module folders\n")
	readme.WriteString("    └── module.json      # Requirements for that target\n")
	readme.WriteString("```\n\n")

	readme.WriteString("## Operational Targets\n\n")
	readme.WriteString("Requirements are linked to PRD operational targets using the `prd_ref` field.\n\n")

	// Summary table
	readme.WriteString("| Priority | Target ID | Title | Requirements |\n")
	readme.WriteString("|----------|-----------|-------|-------------|\n")

	targetReqCount := make(map[string]int)
	for _, req := range requirements {
		targetReqCount[req.PRDRef]++
	}

	for _, target := range targets {
		count := targetReqCount[target.ID]
		readme.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n", target.Criticality, target.ID, truncateString(target.Title, 40), count))
	}
	readme.WriteString("\n")

	readme.WriteString("## Auto-sync Behavior\n\n")
	readme.WriteString("Requirements are linked to PRD operational targets via the `prd_ref` field. When operational targets are updated in the PRD:\n")
	readme.WriteString("1. New targets should have corresponding requirements added\n")
	readme.WriteString("2. Removed targets should have their requirements reviewed for relevance\n")
	readme.WriteString("3. Run `prd-control-tower requirements validate` to check for linkage issues\n\n")

	readme.WriteString("## Validation Commands\n\n")
	readme.WriteString("```bash\n")
	readme.WriteString("# Validate requirements against PRD\n")
	readme.WriteString(fmt.Sprintf("prd-control-tower requirements validate %s\n\n", entityName))
	readme.WriteString("# Check PRD quality (includes requirements validation)\n")
	readme.WriteString(fmt.Sprintf("prd-control-tower prd validate %s\n", entityName))
	readme.WriteString("```\n\n")

	readme.WriteString("## Test Tagging\n\n")
	readme.WriteString("Tag tests with requirement IDs to track coverage:\n")
	readme.WriteString("```go\n")
	readme.WriteString("// [REQ:REQ-P0-001] Test file compression functionality\n")
	readme.WriteString("func TestFileCompression(t *testing.T) { ... }\n")
	readme.WriteString("```\n\n")

	readme.WriteString("---\n\n")
	readme.WriteString(fmt.Sprintf("*Generated by prd-control-tower on %s*\n", time.Now().Format("2006-01-02")))

	return readme.String()
}

func sanitizeFolderName(name string) string {
	// Convert to lowercase, replace spaces and special chars with dashes
	name = strings.ToLower(name)
	// Remove common noise words and patterns
	name = regexp.MustCompile(`\s*\([^)]*\)\s*`).ReplaceAllString(name, "") // Remove parenthetical content
	name = regexp.MustCompile(`\*[^*]*\*`).ReplaceAllString(name, "")      // Remove *italics*
	name = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(name, "-")    // Replace non-alphanumeric with dash
	name = regexp.MustCompile(`-+`).ReplaceAllString(name, "-")            // Collapse multiple dashes
	name = strings.Trim(name, "-")                                          // Trim leading/trailing dashes

	// Truncate to reasonable length
	if len(name) > 40 {
		name = name[:40]
		// Don't end with a dash
		name = strings.TrimRight(name, "-")
	}

	return name
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// RequirementsValidateResponse represents validation results for requirements
type RequirementsValidateResponse struct {
	EntityType       string               `json:"entity_type"`
	EntityName       string               `json:"entity_name"`
	Status           string               `json:"status"`
	Message          string               `json:"message,omitempty"`
	RequirementCount int                  `json:"requirement_count"`
	TargetCount      int                  `json:"target_count"`
	LinkedCount      int                  `json:"linked_count"`
	UnlinkedTargets  []string             `json:"unlinked_targets,omitempty"`
	Violations       []StandardsViolation `json:"violations"`
	GeneratedAt      string               `json:"generated_at"`
}

func handleRequirementsValidate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	useCache := parseBoolQuery(r, "use_cache", false)
	response := executeRequirementsValidate(entityType, entityName, useCache)
	respondJSON(w, http.StatusOK, response)
}

// RequirementsFixRequest represents a request to fix requirements coverage
type RequirementsFixRequest struct {
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
	Context    string `json:"context,omitempty"`
	Model      string `json:"model,omitempty"`
}

// RequirementsFixResponse represents the result of requirements fix operation
type RequirementsFixResponse struct {
	EntityType          string   `json:"entity_type"`
	EntityName          string   `json:"entity_name"`
	Success             bool     `json:"success"`
	Message             string   `json:"message,omitempty"`
	TargetsFixed        int      `json:"targets_fixed"`
	RequirementsAdded   int      `json:"requirements_added"`
	TotalRequirements   int      `json:"total_requirements"`
	RemainingViolations int      `json:"remaining_violations"`
	FilesModified       []string `json:"files_modified"`
	Model               string   `json:"model,omitempty"`
	FixedAt             string   `json:"fixed_at"`
}

func handleRequirementsFix(w http.ResponseWriter, r *http.Request) {
	var req RequirementsFixRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	req.EntityType = strings.TrimSpace(req.EntityType)
	req.EntityName = strings.TrimSpace(req.EntityName)

	if !isValidEntityType(req.EntityType) {
		respondJSON(w, http.StatusBadRequest, RequirementsFixResponse{
			Success: false,
			Message: "Invalid entity type. Must be 'scenario' or 'resource'",
		})
		return
	}
	if req.EntityName == "" {
		respondJSON(w, http.StatusBadRequest, RequirementsFixResponse{
			Success: false,
			Message: "entity_name is required",
		})
		return
	}

	response, status := executeRequirementsFix(req)
	respondJSON(w, status, response)
}

func executeRequirementsFix(req RequirementsFixRequest) (RequirementsFixResponse, int) {
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return RequirementsFixResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to get Vrooli root: %v", err),
		}, http.StatusInternalServerError
	}

	// Step 1: Get current validation state
	validation := executeRequirementsValidate(req.EntityType, req.EntityName, false)

	// If no requirements exist, suggest using generate instead
	if validation.Status == "missing" {
		return RequirementsFixResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    "No requirements found. Use 'requirements generate' first to create initial requirements.",
		}, http.StatusBadRequest
	}

	// If already healthy, nothing to fix
	if validation.Status == "healthy" || len(validation.UnlinkedTargets) == 0 {
		return RequirementsFixResponse{
			EntityType:          req.EntityType,
			EntityName:          req.EntityName,
			Success:             true,
			Message:             "All P0/P1 targets already have requirements. No fix needed.",
			TargetsFixed:        0,
			RequirementsAdded:   0,
			TotalRequirements:   validation.RequirementCount,
			RemainingViolations: 0,
			FixedAt:             time.Now().Format(time.RFC3339),
		}, http.StatusOK
	}

	// Step 2: Load existing requirements and targets
	targets, err := extractOperationalTargets(req.EntityType, req.EntityName)
	if err != nil {
		return RequirementsFixResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to extract operational targets: %v", err),
		}, http.StatusInternalServerError
	}

	existingGroups, err := loadRequirementsForEntity(req.EntityType, req.EntityName)
	if err != nil {
		return RequirementsFixResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to load existing requirements: %v", err),
		}, http.StatusInternalServerError
	}

	// Convert existing requirements to GeneratedRequirement format
	existingReqs := convertRecordsToGenerated(flattenRequirements(existingGroups))

	// Step 3: Find uncovered P0/P1 targets
	uncoveredTargets := findUncoveredTargets(targets, existingReqs, true)
	if len(uncoveredTargets) == 0 {
		return RequirementsFixResponse{
			EntityType:          req.EntityType,
			EntityName:          req.EntityName,
			Success:             true,
			Message:             "All P0/P1 targets already covered.",
			TargetsFixed:        0,
			RequirementsAdded:   0,
			TotalRequirements:   len(existingReqs),
			RemainingViolations: 0,
			FixedAt:             time.Now().Format(time.RFC3339),
		}, http.StatusOK
	}

	// Step 4: Generate requirements for uncovered targets
	openrouterURL := os.Getenv("RESOURCE_OPENROUTER_URL")
	if openrouterURL == "" {
		openrouterURL = "https://openrouter.ai/api/v1"
	}

	prompt := buildFollowUpRequirementsPrompt(req.EntityType, req.EntityName, uncoveredTargets, existingReqs, req.Context)
	aiResponse, usedModel, err := openRouterChatCompletionWithTimeout(openrouterURL, prompt, req.Model, 4000, 180)
	if err != nil {
		return RequirementsFixResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("AI generation failed: %v", err),
		}, http.StatusInternalServerError
	}

	newReqs, err := parseAIRequirementsResponse(aiResponse)
	if err != nil {
		return RequirementsFixResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to parse AI response: %v", err),
			Model:      usedModel,
		}, http.StatusInternalServerError
	}

	// Step 5: Merge and write
	allReqs := mergeRequirements(existingReqs, newReqs)

	var entityDir string
	if req.EntityType == EntityTypeScenario {
		entityDir = "scenarios"
	} else {
		entityDir = "resources"
	}
	requirementsDir := filepath.Join(vrooliRoot, entityDir, req.EntityName, "requirements")

	filesModified, err := writeRequirementsFiles(requirementsDir, allReqs, req.EntityName, targets)
	if err != nil {
		return RequirementsFixResponse{
			EntityType: req.EntityType,
			EntityName: req.EntityName,
			Success:    false,
			Message:    fmt.Sprintf("Failed to write requirements: %v", err),
			Model:      usedModel,
		}, http.StatusInternalServerError
	}

	// Step 6: Re-validate to check remaining issues
	postValidation := executeRequirementsValidate(req.EntityType, req.EntityName, false)
	remainingViolations := len(postValidation.Violations)

	targetsFixed := len(uncoveredTargets) - len(postValidation.UnlinkedTargets)
	if targetsFixed < 0 {
		targetsFixed = 0
	}

	return RequirementsFixResponse{
		EntityType:          req.EntityType,
		EntityName:          req.EntityName,
		Success:             true,
		Message:             fmt.Sprintf("Fixed %d uncovered targets, added %d requirements", targetsFixed, len(newReqs)),
		TargetsFixed:        targetsFixed,
		RequirementsAdded:   len(newReqs),
		TotalRequirements:   len(allReqs),
		RemainingViolations: remainingViolations,
		FilesModified:       filesModified,
		Model:               usedModel,
		FixedAt:             time.Now().Format(time.RFC3339),
	}, http.StatusOK
}

// convertRecordsToGenerated converts RequirementRecord slice to GeneratedRequirement slice
func convertRecordsToGenerated(records []RequirementRecord) []GeneratedRequirement {
	result := make([]GeneratedRequirement, len(records))
	for i, r := range records {
		result[i] = GeneratedRequirement{
			ID:          r.ID,
			Title:       r.Title,
			Description: r.Description,
			PRDRef:      r.PRDRef,
			Criticality: r.Criticality,
			Category:    r.Category,
			Status:      r.Status,
		}
	}
	return result
}

func executeRequirementsValidate(entityType, entityName string, useCache bool) RequirementsValidateResponse {
	// Get full quality report which includes requirements validation
	report, _ := buildQualityReport(entityType, entityName, useCache)

	// Filter violations to only requirements-related ones
	var reqViolations []StandardsViolation
	allViolations := buildStandardsViolationsFromReport(report)

	reqRuleIDs := map[string]bool{
		"prd_missing_requirements":       true,
		"requirements_readme":            true,
		"prd_operational_target_linkage": true,
		"prd_requirements_without_targets": true,
		"prd_prd_ref_integrity":          true,
	}

	for _, v := range allViolations {
		if reqRuleIDs[v.RuleID] {
			reqViolations = append(reqViolations, v)
		}
	}

	// Count linked targets
	linkedCount := 0
	var unlinkedTargets []string
	for _, issue := range report.TargetLinkageIssues {
		unlinkedTargets = append(unlinkedTargets, fmt.Sprintf("%s: %s", issue.Criticality, issue.Title))
	}
	linkedCount = report.TargetCount - len(report.TargetLinkageIssues)

	status := "healthy"
	message := ""
	if !report.HasRequirements {
		status = "missing"
		message = "Requirements directory not found. Run 'requirements generate' first."
	} else if len(reqViolations) > 0 {
		status = "needs_attention"
		message = fmt.Sprintf("Found %d requirements-related issue(s)", len(reqViolations))
	} else {
		message = "All requirements properly linked to operational targets"
	}

	return RequirementsValidateResponse{
		EntityType:       entityType,
		EntityName:       entityName,
		Status:           status,
		Message:          message,
		RequirementCount: report.RequirementCount,
		TargetCount:      report.TargetCount,
		LinkedCount:      linkedCount,
		UnlinkedTargets:  unlinkedTargets,
		Violations:       reqViolations,
		GeneratedAt:      time.Now().Format(time.RFC3339),
	}
}
