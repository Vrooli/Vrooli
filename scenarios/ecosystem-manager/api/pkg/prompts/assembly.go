package prompts

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ecosystem-manager/api/pkg/tasks"
	"gopkg.in/yaml.v3"
)

// Assembler handles dynamic prompt composition
// DESIGN DECISION: Dynamic prompt loading is intentional and provides several benefits:
// 1. Hot-reload capability - prompt changes take effect immediately without restart
// 2. Multi-scenario collaboration - other scenarios can modify shared prompt sections
// 3. Development flexibility - prompts can be refined during active development
// 4. Version control - prompt evolution is tracked alongside code changes
// 5. Modular composition - sections can be mixed/matched for different operations
type Assembler struct {
	PromptsConfig tasks.PromptsConfig
	PromptsDir    string
}

// NewAssembler creates a new prompt assembler
func NewAssembler(promptsDir string) (*Assembler, error) {
	assembler := &Assembler{
		PromptsDir: promptsDir,
	}
	
	// Load prompts configuration
	configFile := filepath.Join(promptsDir, "sections.yaml")
	if err := assembler.loadConfig(configFile); err != nil {
		return nil, fmt.Errorf("failed to load prompts config: %v", err)
	}
	
	return assembler, nil
}

// loadConfig loads the prompts configuration from YAML
func (a *Assembler) loadConfig(configFile string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	
	if err := yaml.Unmarshal(data, &a.PromptsConfig); err != nil {
		return err
	}
	
	log.Println("Loaded prompts configuration for", len(a.PromptsConfig.Operations), "operations")
	return nil
}

// SelectPromptAssembly chooses the appropriate configuration for a task
func (a *Assembler) SelectPromptAssembly(taskType, operation string) (tasks.OperationConfig, error) {
	operationKey := fmt.Sprintf("%s-%s", taskType, operation)
	config, exists := a.PromptsConfig.Operations[operationKey]
	if !exists {
		return tasks.OperationConfig{}, fmt.Errorf("no configuration found for operation: %s", operationKey)
	}
	return config, nil
}

// GeneratePromptSections creates the section list for a task with conditional filtering
func (a *Assembler) GeneratePromptSections(task tasks.TaskItem) ([]string, error) {
	operationConfig, err := a.SelectPromptAssembly(task.Type, task.Operation)
	if err != nil {
		return nil, err
	}
	
	// Start with base sections
	allSections := make([]string, len(a.PromptsConfig.BaseSections))
	copy(allSections, a.PromptsConfig.BaseSections)
	
	// Apply conditional filtering to base sections
	filteredBase := []string{}
	for _, section := range allSections {
		// Skip operational sections that aren't relevant
		if task.Operation == "generator" && strings.Contains(section, "improver-specific") {
			continue // Skip improver sections for generators
		}
		if task.Operation == "improver" && strings.Contains(section, "generator-specific") {
			continue // Skip generator sections for improvers
		}
		
		// Skip type-specific sections that don't match
		if task.Type == "resource" && strings.Contains(section, "scenario-specific") {
			continue // Skip scenario sections for resources
		}
		if task.Type == "scenario" && strings.Contains(section, "resource-specific") {
			continue // Skip resource sections for scenarios
		}
		
		filteredBase = append(filteredBase, section)
	}
	
	// Add operation-specific sections with same filtering
	filteredAdditional := []string{}
	for _, section := range operationConfig.AdditionalSections {
		// Apply same filtering logic
		if task.Type == "resource" && strings.Contains(section, "scenario-specific") {
			log.Printf("Skipping scenario-specific section for resource task: %s", section)
			continue
		}
		if task.Type == "scenario" && strings.Contains(section, "resource-specific") {
			log.Printf("Skipping resource-specific section for scenario task: %s", section)
			continue
		}
		
		filteredAdditional = append(filteredAdditional, section)
	}
	
	// Combine filtered sections
	allSections = append(filteredBase, filteredAdditional...)
	
	log.Printf("Task %s (%s-%s): Using %d sections (filtered from %d)", 
		task.ID, task.Type, task.Operation, 
		len(allSections), 
		len(a.PromptsConfig.BaseSections) + len(operationConfig.AdditionalSections))
	
	return allSections, nil
}

// resolveIncludes recursively resolves {{INCLUDE: path}} directives in content
func (a *Assembler) resolveIncludes(content string, basePath string, depth int) (string, error) {
	// Prevent infinite recursion
	if depth > 10 {
		return content, fmt.Errorf("include depth exceeded (max 10)")
	}
	
	// Pattern to match {{INCLUDE: path}} directives
	includePattern := regexp.MustCompile(`\{\{INCLUDE:\s*([^\}]+)\}\}`)
	
	// Track errors
	var resolveErrors []string
	
	// Replace all includes
	resolved := includePattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the path from the match
		submatches := includePattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			resolveErrors = append(resolveErrors, fmt.Sprintf("Invalid include directive: %s", match))
			return match
		}
		
		includePath := strings.TrimSpace(submatches[1])
		
		// Resolve the full path
		var fullPath string
		if filepath.IsAbs(includePath) {
			fullPath = includePath
		} else {
			fullPath = filepath.Join(basePath, includePath)
		}
		
		// Ensure .md extension
		if !strings.HasSuffix(fullPath, ".md") {
			fullPath += ".md"
		}
		
		// Read the included file
		includeContent, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("Warning: Failed to include %s: %v", includePath, err)
			resolveErrors = append(resolveErrors, fmt.Sprintf("Failed to include %s: %v", includePath, err))
			return fmt.Sprintf("<!-- INCLUDE FAILED: %s -->", includePath)
		}
		
		// Recursively resolve includes in the included content
		resolvedInclude, err := a.resolveIncludes(string(includeContent), filepath.Dir(fullPath), depth+1)
		if err != nil {
			log.Printf("Warning: Failed to resolve includes in %s: %v", includePath, err)
			return string(includeContent) // Return unresolved content
		}
		
		return resolvedInclude
	})
	
	// Return error if any includes failed
	if len(resolveErrors) > 0 {
		log.Printf("Include resolution warnings: %v", resolveErrors)
		// Don't fail completely, just log warnings
	}
	
	return resolved, nil
}

// AssemblePrompt reads and concatenates prompt sections into a full prompt
func (a *Assembler) AssemblePrompt(sections []string) (string, error) {
	var promptBuilder strings.Builder
	
	promptBuilder.WriteString("# Ecosystem Manager Task Execution\n\n")
	promptBuilder.WriteString("You are executing a task for the Vrooli Ecosystem Manager.\n\n")
	promptBuilder.WriteString("---\n\n")
	
	for i, section := range sections {
		// Convert section path to file path
		var filePath string
		
		// Check if section already has .md extension
		if strings.HasSuffix(section, ".md") {
			filePath = filepath.Join(a.PromptsDir, section)
		} else {
			filePath = filepath.Join(a.PromptsDir, section + ".md")
		}
		
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Try without the prompts prefix if it's an absolute path
			if filepath.IsAbs(section) {
				filePath = section
				if !strings.HasSuffix(filePath, ".md") {
					filePath += ".md"
				}
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					log.Printf("Warning: Section file not found: %s", section)
					continue // Skip missing sections with warning
				}
			} else {
				log.Printf("Warning: Section file not found: %s", section)
				continue // Skip missing sections with warning
			}
		}
		
		// Read the file content
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read section %s: %v", section, err)
		}
		
		// Resolve any {{INCLUDE:}} directives in the content
		resolvedContent, err := a.resolveIncludes(string(content), filepath.Dir(filePath), 0)
		if err != nil {
			log.Printf("Warning: Include resolution had issues in %s: %v", section, err)
			// Continue with partially resolved content
			resolvedContent = string(content)
		}
		
		// Add section header for clarity
		sectionName := filepath.Base(section)
		sectionName = strings.TrimSuffix(sectionName, ".md")
		promptBuilder.WriteString(fmt.Sprintf("## Section %d: %s\n\n", i+1, sectionName))
		promptBuilder.WriteString(resolvedContent)
		promptBuilder.WriteString("\n\n---\n\n")
	}
	
	return promptBuilder.String(), nil
}

// AssemblePromptForTask generates a complete prompt for a specific task
func (a *Assembler) AssemblePromptForTask(task tasks.TaskItem) (string, error) {
	sections, err := a.GeneratePromptSections(task)
	if err != nil {
		return "", fmt.Errorf("failed to generate sections: %v", err)
	}
	
	prompt, err := a.AssemblePrompt(sections)
	if err != nil {
		return "", fmt.Errorf("failed to assemble prompt: %v", err)
	}
	
	// Add task-specific context to the prompt
	var taskContext strings.Builder
	taskContext.WriteString("\n\n## Task Context\n\n")
	taskContext.WriteString(fmt.Sprintf("**Task ID**: %s\n", task.ID))
	taskContext.WriteString(fmt.Sprintf("**Title**: %s\n", task.Title))
	taskContext.WriteString(fmt.Sprintf("**Type**: %s\n", task.Type))
	taskContext.WriteString(fmt.Sprintf("**Operation**: %s\n", task.Operation))
	taskContext.WriteString(fmt.Sprintf("**Category**: %s\n", task.Category))
	taskContext.WriteString(fmt.Sprintf("**Priority**: %s\n", task.Priority))
	
	if task.Requirements != nil && len(task.Requirements) > 0 {
		taskContext.WriteString("\n### Requirements\n")
		requirementsJSON, _ := json.MarshalIndent(task.Requirements, "", "  ")
		taskContext.WriteString(string(requirementsJSON))
	}
	
	if len(task.Notes) > 0 {
		taskContext.WriteString(fmt.Sprintf("\n### Notes\n%s\n", task.Notes))
	}
	
	return prompt + taskContext.String(), nil
}

// GetOperationNames returns all available operation names
func (a *Assembler) GetOperationNames() []string {
	var names []string
	for key := range a.PromptsConfig.Operations {
		names = append(names, key)
	}
	return names
}

// GetPromptsConfig returns the current prompts configuration
func (a *Assembler) GetPromptsConfig() tasks.PromptsConfig {
	return a.PromptsConfig
}