package pipeline

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// PlanningResult contains the output of the planning phase
type PlanningResult struct {
	FinalPlan           string                 `json:"final_plan"`
	Architecture        ArchitectureDetails    `json:"architecture"`
	IdentifiedResources []string               `json:"identified_resources"`
	FileStructure       []string               `json:"file_structure"`
	Phases              []ImplementationPhase  `json:"phases"`
	QualityScore        int                    `json:"quality_score"`
	Iterations          int                    `json:"iterations"`
}

// ArchitectureDetails describes the system architecture
type ArchitectureDetails struct {
	Components   []string          `json:"components"`
	HasAPI       bool              `json:"has_api"`
	HasUI        bool              `json:"has_ui"`
	HasDatabase  bool              `json:"has_database"`
	HasWorkflows bool              `json:"has_workflows"`
	DataFlow     map[string]string `json:"data_flow"`
}

// ImplementationPhase represents a development phase
type ImplementationPhase struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Deliverables []string `json:"deliverables"`
}

// PlanMetrics tracks planning phase metrics
type PlanMetrics struct {
	Iterations          int      `json:"iterations"`
	IdentifiedResources []string `json:"identified_resources"`
	QualityScore        int      `json:"quality_score"`
}

// runPlanningPhase executes the planning phase with iterative refinement
func (p *Pipeline) runPlanningPhase(req GenerationRequest) (*PlanningResult, *PlanMetrics, error) {
	startTime := time.Now()
	
	result := &PlanningResult{
		IdentifiedResources: []string{},
		FileStructure:       []string{},
		Phases:             []ImplementationPhase{},
	}
	
	metrics := &PlanMetrics{
		Iterations: 0,
	}
	
	var currentPlan string
	var bestPlan string
	var bestScore int
	
	for iteration := 1; iteration <= req.Iterations.Planning; iteration++ {
		metrics.Iterations = iteration
		log.Printf("Planning iteration %d/%d for %s", iteration, req.Iterations.Planning, req.Name)
		
		// Generate or refine the plan
		var prompt string
		if iteration == 1 {
			prompt = p.buildInitialPlanningPrompt(req)
		} else {
			prompt = p.buildPlanRefinementPrompt(req, currentPlan, result.QualityScore)
		}
		
		// Log the planning attempt
		p.logGenerationStep(req.Name, fmt.Sprintf("planning_iteration_%d", iteration), prompt, nil, true, nil)
		
		// Call Claude for planning
		planOutput, err := p.claude.Chat(prompt)
		if err != nil {
			errorMsg := fmt.Sprintf("Planning iteration %d failed: %v", iteration, err)
			p.logGenerationStep(req.Name, fmt.Sprintf("planning_error_%d", iteration), prompt, nil, false, &errorMsg)
			
			// Continue with next iteration if we have retries left
			if iteration < req.Iterations.Planning {
				log.Printf("Planning iteration %d failed, retrying: %v", iteration, err)
				continue
			}
			return nil, nil, fmt.Errorf("planning failed after %d iterations: %w", iteration, err)
		}
		
		currentPlan = planOutput
		
		// Analyze the plan quality
		analysis := p.analyzePlan(planOutput)
		result.QualityScore = analysis.QualityScore
		
		// Keep track of the best plan
		if analysis.QualityScore > bestScore {
			bestPlan = planOutput
			bestScore = analysis.QualityScore
			result.Architecture = analysis.Architecture
			result.IdentifiedResources = analysis.Resources
			result.FileStructure = analysis.FileStructure
			result.Phases = analysis.Phases
		}
		
		// Log successful iteration
		responseStr := fmt.Sprintf("Quality Score: %d", analysis.QualityScore)
		p.logGenerationStep(req.Name, fmt.Sprintf("planning_complete_%d", iteration), prompt, &responseStr, true, nil)
		
		// Check if plan is good enough
		if analysis.QualityScore >= 75 {
			log.Printf("Plan quality acceptable (score: %d) after %d iterations", analysis.QualityScore, iteration)
			break
		}
		
		// Add delay between iterations to avoid rate limiting
		if iteration < req.Iterations.Planning {
			time.Sleep(2 * time.Second)
		}
	}
	
	// Use the best plan we found
	result.FinalPlan = bestPlan
	result.QualityScore = bestScore
	metrics.IdentifiedResources = result.IdentifiedResources
	metrics.QualityScore = bestScore
	
	log.Printf("Planning phase completed in %v with quality score %d", time.Since(startTime), bestScore)
	
	return result, metrics, nil
}

// buildInitialPlanningPrompt creates the initial planning prompt
func (p *Pipeline) buildInitialPlanningPrompt(req GenerationRequest) string {
	prompt := fmt.Sprintf(`# Scenario Generation - Initial Planning Phase

You are an expert Vrooli scenario architect. Analyze the following customer request and create a comprehensive implementation plan.

## Customer Request:
**Scenario Name:** %s
**Description:** %s
**Complexity:** %s
**Category:** %s
**Business Value:** $15,000 - $35,000

## Planning Requirements:

Create a detailed implementation plan that includes:

### 1. Architecture Overview
- System components and their relationships
- Data flow and processing pipeline  
- User interface requirements
- Integration points
- Technology stack decisions

### 2. Resource Requirements
- Required Vrooli resources (postgres, claude-code, etc.)
- External APIs or services needed
- Storage and processing requirements
- Estimated resource usage
- Resource initialization needs

### 3. Implementation Phases
- Break down into logical development phases (minimum 3-5 phases)
- Dependencies between components
- Risk assessment and mitigation
- Testing and validation strategy
- Estimated time for each phase

### 4. File Structure
- Complete directory structure
- Key files that need to be generated (minimum 10-15 files)
- Configuration and initialization files
- Documentation requirements
- Test files and scripts

### 5. Business Logic
- Core workflows and processes
- User journeys and interactions
- Data models and relationships
- Business rules and validation
- Error handling strategies

### 6. Technical Specifications
- API endpoints and data formats (if applicable)
- Database schema design (if using postgres)
- UI/UX component structure (if applicable)
- Deployment and monitoring setup
- Security considerations

### 7. Success Criteria
- How to validate the scenario works
- Performance benchmarks
- User acceptance criteria
- Business value metrics

Provide a structured, detailed plan that can guide the implementation phase. Focus on practical, deployable solutions that deliver real business value.

**Output Format:** Well-structured markdown with clear sections, bullet points, and actionable details. Be comprehensive and specific.`, 
		req.Name, req.Description, req.Complexity, req.Category)
	
	// Add any specific requirements from the prompt
	if req.Prompt != "" {
		prompt += fmt.Sprintf("\n\n## Additional Requirements from User:\n%s", req.Prompt)
	}
	
	// Add resource hints if provided
	if len(req.Resources) > 0 {
		prompt += fmt.Sprintf("\n\n## Suggested Resources:\n%s", strings.Join(req.Resources, ", "))
	}
	
	return prompt
}

// buildPlanRefinementPrompt creates a prompt to refine an existing plan
func (p *Pipeline) buildPlanRefinementPrompt(req GenerationRequest, currentPlan string, qualityScore int) string {
	return fmt.Sprintf(`# Scenario Generation - Plan Refinement

The initial plan needs improvement. Please refine the following plan to be more comprehensive and actionable.

## Current Plan:
%s

## Quality Analysis:
Current Quality Score: %d/100

## Areas Needing Improvement:
%s

## Refinement Requirements:
1. Add more specific technical details
2. Ensure all sections are comprehensive
3. Include concrete file paths and names
4. Add specific Vrooli resource configurations
5. Include error handling and edge cases
6. Add validation and testing strategies
7. Ensure the plan is actionable and complete

Please provide an improved, more detailed plan that addresses these gaps. Focus on making the plan concrete and implementable.

**Output Format:** Enhanced markdown with all sections expanded and detailed.`, 
		currentPlan, 
		qualityScore,
		p.identifyPlanGaps(currentPlan, qualityScore))
}

// analyzePlan evaluates the quality of a generated plan
func (p *Pipeline) analyzePlan(planContent string) *PlanAnalysis {
	analysis := &PlanAnalysis{
		QualityScore: 0,
		Architecture: ArchitectureDetails{
			Components: []string{},
		},
		Resources:     []string{},
		FileStructure: []string{},
		Phases:       []ImplementationPhase{},
	}
	
	planLower := strings.ToLower(planContent)
	
	// Check for architecture section
	if strings.Contains(planLower, "architecture") {
		analysis.QualityScore += 15
		analysis.Architecture.HasAPI = strings.Contains(planLower, "api")
		analysis.Architecture.HasUI = strings.Contains(planLower, "ui") || strings.Contains(planLower, "interface")
		analysis.Architecture.HasDatabase = strings.Contains(planLower, "database") || strings.Contains(planLower, "postgres")
		analysis.Architecture.HasWorkflows = strings.Contains(planLower, "workflow") || strings.Contains(planLower, "automation")
		
		// Extract components
		componentPattern := regexp.MustCompile(`(?i)(?:component|service|module):\s*([^\n]+)`)
		matches := componentPattern.FindAllStringSubmatch(planContent, -1)
		for _, match := range matches {
			if len(match) > 1 {
				analysis.Architecture.Components = append(analysis.Architecture.Components, strings.TrimSpace(match[1]))
			}
		}
	}
	
	// Check for resource requirements
	if strings.Contains(planLower, "resource") {
		analysis.QualityScore += 15
		
		// Extract Vrooli resources
		resources := []string{"postgres", "claude-code", "minio", "redis", "vault", "comfyui", "unstructured-io", "judge0", "browserless"}
		for _, resource := range resources {
			if strings.Contains(planLower, resource) {
				analysis.Resources = append(analysis.Resources, resource)
			}
		}
		
		// Default to postgres if no resources found
		if len(analysis.Resources) == 0 {
			analysis.Resources = []string{"postgres"}
		}
	}
	
	// Check for implementation phases
	if strings.Contains(planLower, "phase") || strings.Contains(planLower, "step") {
		analysis.QualityScore += 15
		
		// Extract phases
		phasePattern := regexp.MustCompile(`(?i)(?:phase|step)\s*\d*[:\s]+([^\n]+)`)
		matches := phasePattern.FindAllStringSubmatch(planContent, -1)
		for i, match := range matches {
			if len(match) > 1 && i < 10 { // Limit to 10 phases
				phase := ImplementationPhase{
					Name:        strings.TrimSpace(match[1]),
					Description: strings.TrimSpace(match[1]),
				}
				analysis.Phases = append(analysis.Phases, phase)
			}
		}
	}
	
	// Check for file structure
	if strings.Contains(planLower, "file") || strings.Contains(planLower, "structure") {
		analysis.QualityScore += 15
		
		// Extract file paths
		filePattern := regexp.MustCompile(`(?:\/[\w\-]+)+(?:\.[\w]+)?|[\w\-]+\.[\w]+`)
		matches := filePattern.FindAllString(planContent, -1)
		seen := make(map[string]bool)
		for _, match := range matches {
			if !seen[match] && len(analysis.FileStructure) < 30 { // Limit to 30 files
				analysis.FileStructure = append(analysis.FileStructure, match)
				seen[match] = true
			}
		}
	}
	
	// Check for business logic
	if strings.Contains(planLower, "business") || strings.Contains(planLower, "workflow") || strings.Contains(planLower, "process") {
		analysis.QualityScore += 10
	}
	
	// Check for technical specifications
	if strings.Contains(planLower, "endpoint") || strings.Contains(planLower, "schema") || strings.Contains(planLower, "api") {
		analysis.QualityScore += 10
	}
	
	// Check for validation/testing
	if strings.Contains(planLower, "test") || strings.Contains(planLower, "validation") || strings.Contains(planLower, "verify") {
		analysis.QualityScore += 10
	}
	
	// Check for documentation
	if strings.Contains(planLower, "document") || strings.Contains(planLower, "readme") {
		analysis.QualityScore += 5
	}
	
	// Check for error handling
	if strings.Contains(planLower, "error") || strings.Contains(planLower, "exception") || strings.Contains(planLower, "failure") {
		analysis.QualityScore += 5
	}
	
	// Ensure we don't exceed 100
	if analysis.QualityScore > 100 {
		analysis.QualityScore = 100
	}
	
	return analysis
}

// PlanAnalysis contains the analysis results of a plan
type PlanAnalysis struct {
	QualityScore int
	Architecture ArchitectureDetails
	Resources    []string
	FileStructure []string
	Phases       []ImplementationPhase
}

// identifyPlanGaps identifies what's missing from a plan
func (p *Pipeline) identifyPlanGaps(plan string, score int) string {
	gaps := []string{}
	planLower := strings.ToLower(plan)
	
	if !strings.Contains(planLower, "architecture") {
		gaps = append(gaps, "- Missing detailed architecture overview")
	}
	
	if !strings.Contains(planLower, "resource") {
		gaps = append(gaps, "- Missing resource requirements specification")
	}
	
	if !strings.Contains(planLower, "phase") && !strings.Contains(planLower, "step") {
		gaps = append(gaps, "- Missing implementation phases breakdown")
	}
	
	if !strings.Contains(planLower, "file") {
		gaps = append(gaps, "- Missing file structure definition")
	}
	
	if !strings.Contains(planLower, "test") && !strings.Contains(planLower, "validation") {
		gaps = append(gaps, "- Missing testing and validation strategy")
	}
	
	if !strings.Contains(planLower, "error") {
		gaps = append(gaps, "- Missing error handling considerations")
	}
	
	if len(gaps) == 0 {
		gaps = append(gaps, "- Plan needs more specific technical details and concrete examples")
	}
	
	return strings.Join(gaps, "\n")
}

// logGenerationStep logs a step in the generation process
func (p *Pipeline) logGenerationStep(scenarioName string, step string, prompt string, response *string, success bool, errorMsg *string) {
	// This is a simplified version - in production, store in database
	if success {
		log.Printf("[%s] %s: Success", scenarioName, step)
	} else if errorMsg != nil {
		log.Printf("[%s] %s: Error - %s", scenarioName, step, *errorMsg)
	}
}