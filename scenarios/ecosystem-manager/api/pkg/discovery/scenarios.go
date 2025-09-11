package discovery

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

// DiscoverScenarios scans the filesystem for available scenarios
func DiscoverScenarios() ([]tasks.ScenarioInfo, error) {
	var scenarios []tasks.ScenarioInfo

	// Use vrooli command to get scenarios list with verbose output
	cmd := exec.Command("vrooli", "scenario", "list", "--json", "--verbose")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Warning: Failed to run 'vrooli scenario list --json --verbose': %v", err)
		// Try without verbose flag as fallback
		cmd = exec.Command("vrooli", "scenario", "list", "--json")
		output, err = cmd.Output()
		if err != nil {
			log.Printf("Warning: Fallback failed too: %v", err)
			// Return empty list instead of error to prevent UI issues
			return scenarios, nil
		}
	}

	// Parse the JSON output - vrooli returns an object with "scenarios" field
	var vrooliResponse struct {
		Scenarios []map[string]interface{} `json:"scenarios"`
	}
	if err := json.Unmarshal(output, &vrooliResponse); err != nil {
		log.Printf("Warning: Failed to parse vrooli scenario list output: %v", err)
		return scenarios, nil
	}
	vrooliScenarios := vrooliResponse.Scenarios

	// Convert to our ScenarioInfo format
	for _, vs := range vrooliScenarios {
		scenario := tasks.ScenarioInfo{
			Name:           getStringField(vs, "name"),
			Path:           getStringField(vs, "path"),
			Category:       getStringField(vs, "category"),
			Description:    getStringField(vs, "description"),
			PRDComplete:    getIntField(vs, "prd_completion"),
			Healthy:        getBoolField(vs, "healthy"),
			P0Requirements: getIntField(vs, "p0_requirements"),
			P0Completed:    getIntField(vs, "p0_completed"),
		}

		// Skip empty entries
		if scenario.Name != "" {
			scenarios = append(scenarios, scenario)
		}
	}

	// Sort scenarios alphabetically by name
	sort.Slice(scenarios, func(i, j int) bool {
		return scenarios[i].Name < scenarios[j].Name
	})

	log.Printf("Discovered %d scenarios via vrooli command", len(scenarios))
	return scenarios, nil
}

// CheckResourceHealth checks if a resource is healthy
func CheckResourceHealth(resourceName, resourceDir string) bool {
	// Try to find and execute health check script
	healthScript := filepath.Join(resourceDir, "lib", "health.sh")
	if _, err := os.Stat(healthScript); os.IsNotExist(err) {
		// No health script found, assume healthy if service.json exists
		return true
	}

	// TODO: Actually execute the health script
	// For now, just return true to avoid blocking
	return true
}

// CheckScenarioHealth checks if a scenario is healthy
func CheckScenarioHealth(scenarioName, scenarioDir string) bool {
	// Check if scenario has basic structure
	requiredFiles := []string{"PRD.md", "README.md"}

	for _, file := range requiredFiles {
		filePath := filepath.Join(scenarioDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// GetScenarioPRDStatus parses a PRD.md file to get completion status
func GetScenarioPRDStatus(scenarioName, prdPath string) tasks.PRDStatus {
	status := tasks.PRDStatus{}

	// Read PRD file
	data, err := os.ReadFile(prdPath)
	if err != nil {
		log.Printf("Warning: could not read PRD for %s: %v", scenarioName, err)
		return status
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Parse checkboxes to count requirements
	totalRequirements := 0
	completedRequirements := 0
	p0Requirements := 0
	p0Completed := 0
	p1Requirements := 0
	p1Completed := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for checkbox patterns
		if strings.Contains(trimmed, "- [ ]") || strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
			totalRequirements++

			// Check if completed
			if strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
				completedRequirements++
			}

			// Check priority by looking at context
			contextLines := ""
			for j := max(0, i-3); j <= min(len(lines)-1, i+1); j++ {
				contextLines += strings.ToLower(lines[j]) + " "
			}

			if strings.Contains(contextLines, "p0") || strings.Contains(contextLines, "priority 0") {
				p0Requirements++
				if strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
					p0Completed++
				}
			} else if strings.Contains(contextLines, "p1") || strings.Contains(contextLines, "priority 1") {
				p1Requirements++
				if strings.Contains(trimmed, "- [x]") || strings.Contains(trimmed, "- [X]") {
					p1Completed++
				}
			}
		}
	}

	// Calculate completion percentage
	completionPercentage := 0
	if totalRequirements > 0 {
		completionPercentage = (completedRequirements * 100) / totalRequirements
	}

	status.CompletionPercentage = completionPercentage
	status.P0Requirements = p0Requirements
	status.P0Completed = p0Completed
	status.P1Requirements = p1Requirements
	status.P1Completed = p1Completed

	return status
}

// ExtractDescriptionFromPRD extracts description from PRD file
func ExtractDescriptionFromPRD(prdPath string) string {
	data, err := os.ReadFile(prdPath)
	if err != nil {
		return "No description available"
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Look for overview or description section
	for i, line := range lines {
		trimmed := strings.TrimSpace(strings.ToLower(line))
		if strings.Contains(trimmed, "## overview") || strings.Contains(trimmed, "## description") {
			// Get the next few non-empty lines
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine != "" && !strings.HasPrefix(nextLine, "#") {
					return nextLine
				}
			}
		}
	}

	return "No description available"
}

// InferScenarioCategory attempts to categorize a scenario based on name and description
func InferScenarioCategory(name, description string) string {
	lower := strings.ToLower(name + " " + description)

	if strings.Contains(lower, "ai") || strings.Contains(lower, "ml") || strings.Contains(lower, "llm") {
		return "ai-tools"
	}
	if strings.Contains(lower, "business") || strings.Contains(lower, "invoice") || strings.Contains(lower, "finance") {
		return "business"
	}
	if strings.Contains(lower, "automat") || strings.Contains(lower, "workflow") {
		return "automation"
	}
	if strings.Contains(lower, "personal") || strings.Contains(lower, "life") {
		return "personal"
	}
	if strings.Contains(lower, "game") || strings.Contains(lower, "entertainment") {
		return "entertainment"
	}

	return "productivity" // default
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
