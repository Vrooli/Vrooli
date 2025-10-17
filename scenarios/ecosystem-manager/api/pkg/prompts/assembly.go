package prompts

import (
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

// SectionDetail captures metadata for each assembled prompt section.
type SectionDetail struct {
	Index        int      `json:"index"`
	Key          string   `json:"key"`
	Title        string   `json:"title"`
	RelativePath string   `json:"relative_path"`
	Includes     []string `json:"includes,omitempty"`
	Content      string   `json:"content"`
}

// PromptAssembly represents the fully assembled prompt and its section breakdown.
type PromptAssembly struct {
	Prompt   string          `json:"prompt"`
	Sections []SectionDetail `json:"sections"`
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

	// Helper function to determine if a section should be skipped
	shouldSkipSection := func(section string, taskType string, taskOperation string) bool {
		// Skip operation-specific sections that don't match
		if taskOperation == "generator" && strings.Contains(section, "improver-specific") {
			log.Printf("Skipping improver-specific section for generator task: %s", section)
			return true
		}
		if taskOperation == "improver" && strings.Contains(section, "generator-specific") {
			log.Printf("Skipping generator-specific section for improver task: %s", section)
			return true
		}

		// Skip type-specific sections that don't match
		if taskType == "resource" && strings.Contains(section, "scenario-specific") {
			log.Printf("Skipping scenario-specific section for resource task: %s", section)
			return true
		}
		if taskType == "scenario" && strings.Contains(section, "resource-specific") {
			log.Printf("Skipping resource-specific section for scenario task: %s", section)
			return true
		}

		return false
	}

	// Filter base sections
	filteredSections := []string{}
	for _, section := range a.PromptsConfig.BaseSections {
		if !shouldSkipSection(section, task.Type, task.Operation) {
			filteredSections = append(filteredSections, section)
		}
	}

	// Filter and add operation-specific sections
	for _, section := range operationConfig.AdditionalSections {
		if !shouldSkipSection(section, task.Type, task.Operation) {
			filteredSections = append(filteredSections, section)
		}
	}

	// Use the filtered sections
	allSections := filteredSections

	log.Printf("Task %s (%s-%s): Using %d sections (filtered from %d)",
		task.ID, task.Type, task.Operation,
		len(allSections),
		len(a.PromptsConfig.BaseSections)+len(operationConfig.AdditionalSections))

	// Log the actual sections being used for debugging
	log.Printf("Sections for task %s: %v", task.ID, allSections)

	return allSections, nil
}

// isCriticalInclude determines if an include path is critical and must not fail
func (a *Assembler) isCriticalInclude(includePath string) bool {
	// Critical includes are core sections that all operations need
	criticalPaths := []string{
		"shared/core/",
		"shared/methodologies/",
		"shared/operational/",
		"patterns/prd-essentials",
	}

	for _, critical := range criticalPaths {
		if strings.Contains(includePath, critical) {
			return true
		}
	}
	return false
}

// resolveIncludes recursively resolves {{INCLUDE: path}} directives in content
func (a *Assembler) resolveIncludes(content string, basePath string, depth int, includeAccumulator *[]string) (string, error) {
	// Prevent infinite recursion
	if depth > 10 {
		return content, fmt.Errorf("include depth exceeded (max 10)")
	}

	// Pattern to match {{INCLUDE: path}} directives
	includePattern := regexp.MustCompile(`\{\{INCLUDE:\s*([^\}]+)\}\}`)

	// Track errors separately for critical and non-critical
	var criticalErrors []string
	var warningErrors []string

	// Replace all includes
	resolved := includePattern.ReplaceAllStringFunc(content, func(match string) string {
		// Extract the path from the match
		submatches := includePattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			errorMsg := fmt.Sprintf("Invalid include directive: %s", match)
			criticalErrors = append(criticalErrors, errorMsg)
			return match
		}

		includePath := strings.TrimSpace(submatches[1])
		isCritical := a.isCriticalInclude(includePath)

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
			errorMsg := fmt.Sprintf("Failed to include %s: %v", includePath, err)
			if isCritical {
				log.Printf("CRITICAL: %s", errorMsg)
				criticalErrors = append(criticalErrors, errorMsg)
				return fmt.Sprintf("<!-- CRITICAL INCLUDE FAILED: %s -->", includePath)
			} else {
				log.Printf("Warning: %s", errorMsg)
				warningErrors = append(warningErrors, errorMsg)
				return fmt.Sprintf("<!-- INCLUDE FAILED: %s -->", includePath)
			}
		}

		if includeAccumulator != nil {
			if rel, err := filepath.Rel(a.PromptsDir, fullPath); err == nil {
				*includeAccumulator = append(*includeAccumulator, filepath.ToSlash(rel))
			} else {
				*includeAccumulator = append(*includeAccumulator, includePath)
			}
		}

		// Recursively resolve includes in the included content
		resolvedInclude, err := a.resolveIncludes(string(includeContent), filepath.Dir(fullPath), depth+1, includeAccumulator)
		if err != nil {
			errorMsg := fmt.Sprintf("Failed to resolve includes in %s: %v", includePath, err)
			if isCritical {
				log.Printf("CRITICAL: %s", errorMsg)
				criticalErrors = append(criticalErrors, errorMsg)
				return string(includeContent) // Return unresolved content
			} else {
				log.Printf("Warning: %s", errorMsg)
				return string(includeContent) // Return unresolved content
			}
		}

		return resolvedInclude
	})

	// Log warnings if any non-critical includes failed
	if len(warningErrors) > 0 {
		log.Printf("Include resolution warnings: %v", warningErrors)
	}

	// Return error if any critical includes failed
	if len(criticalErrors) > 0 {
		return resolved, fmt.Errorf("critical include failures: %v", criticalErrors)
	}

	return resolved, nil
}

// AssemblePrompt reads and concatenates prompt sections into a full prompt
func (a *Assembler) AssemblePrompt(sections []string) (PromptAssembly, error) {
	var promptBuilder strings.Builder
	var sectionDetails []SectionDetail

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
			filePath = filepath.Join(a.PromptsDir, section+".md")
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
			return PromptAssembly{}, fmt.Errorf("failed to read section %s: %v", section, err)
		}

		// Resolve any {{INCLUDE:}} directives in the content
		includeAccumulator := []string{}
		resolvedContent, err := a.resolveIncludes(string(content), filepath.Dir(filePath), 0, &includeAccumulator)
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

		relPath := section
		if filepath.IsAbs(filePath) {
			if rel, err := filepath.Rel(a.PromptsDir, filePath); err == nil {
				relPath = filepath.ToSlash(rel)
			} else {
				relPath = filepath.ToSlash(filePath)
			}
		} else {
			if !strings.HasSuffix(relPath, ".md") {
				relPath = relPath + ".md"
			}
			relPath = filepath.ToSlash(relPath)
		}

		if len(includeAccumulator) > 0 {
			includeAccumulator = dedupeStrings(includeAccumulator)
		}

		sectionDetails = append(sectionDetails, SectionDetail{
			Index:        i,
			Key:          section,
			Title:        sectionName,
			RelativePath: relPath,
			Includes:     includeAccumulator,
			Content:      resolvedContent,
		})
	}

	return PromptAssembly{
		Prompt:   promptBuilder.String(),
		Sections: sectionDetails,
	}, nil
}

func dedupeStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		if _, exists := seen[v]; exists {
			continue
		}
		seen[v] = struct{}{}
		result = append(result, v)
	}
	return result
}

// AssemblePromptForTask generates a complete prompt for a specific task
func (a *Assembler) AssemblePromptForTask(task tasks.TaskItem) (PromptAssembly, error) {
	// Reload config to pick up changes (hot-reload capability)
	configFile := filepath.Join(a.PromptsDir, "sections.yaml")
	if err := a.loadConfig(configFile); err != nil {
		log.Printf("Warning: Failed to reload config, using cached version: %v", err)
	}

	sections, err := a.GeneratePromptSections(task)
	if err != nil {
		return PromptAssembly{}, fmt.Errorf("failed to generate sections: %v", err)
	}

	assembly, err := a.AssemblePrompt(sections)
	if err != nil {
		return PromptAssembly{}, fmt.Errorf("failed to assemble prompt: %v", err)
	}

	// Add task-specific context to the prompt
	var taskContext strings.Builder
	taskContext.WriteString("\n\n## Task Context\n\n")
	taskContext.WriteString(fmt.Sprintf("**Task ID**: %s\n", task.ID))
	taskContext.WriteString(fmt.Sprintf("**Title**: %s\n", task.Title))
	taskContext.WriteString(fmt.Sprintf("**Type**: %s\n", task.Type))
	taskContext.WriteString(fmt.Sprintf("**Operation**: %s\n", task.Operation))
	taskContext.WriteString(fmt.Sprintf("**Priority**: %s\n", task.Priority))

	if len(task.Notes) > 0 {
		taskContext.WriteString(fmt.Sprintf("\n### Notes\n%s\n", task.Notes))
	}

	assembly.Prompt = assembly.Prompt + taskContext.String()
	assembly.Sections = append(assembly.Sections, SectionDetail{
		Index:        len(assembly.Sections),
		Key:          "task-context",
		Title:        "task_context",
		RelativePath: "(generated)",
		Content:      taskContext.String(),
	})

	return assembly, nil
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
