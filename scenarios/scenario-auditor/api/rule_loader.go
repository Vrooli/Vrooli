package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// Violation represents a rule violation
type Violation struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// RuleInfo represents metadata extracted from a rule file
type RuleInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Standard    string `json:"standard"`
	FilePath    string `json:"file_path"`
	Enabled     bool   `json:"enabled"`
}

// Check method for RuleInfo - delegates to the actual rule implementation
func (r RuleInfo) Check(content string, filepath string) ([]Violation, error) {
	// Look up the rule implementation in the registry
	impl, exists := RuleRegistry[r.ID]
	if !exists {
		// No implementation found, return empty violations
		return []Violation{}, nil
	}
	
	// Execute the rule
	return impl.Check(content, filepath)
}

// LoadRulesFromFiles scans the rules directory and extracts rule metadata
func LoadRulesFromFiles() (map[string]RuleInfo, error) {
	rules := make(map[string]RuleInfo)
	
	// Get the rules directory path
	vrooliRoot := os.Getenv("VROOLI_ROOT")
	if vrooliRoot == "" {
		vrooliRoot = os.Getenv("HOME") + "/Vrooli"
	}
	rulesDir := filepath.Join(vrooliRoot, "scenarios", "scenario-auditor", "rules")
	
	// Walk through all subdirectories
	categories := []string{"api", "cli", "config", "test", "ui"}
	
	for _, category := range categories {
		categoryDir := filepath.Join(rulesDir, category)
		
		// Check if directory exists
		if _, err := os.Stat(categoryDir); os.IsNotExist(err) {
			continue
		}
		
		// Read all .go files in the category
		files, err := ioutil.ReadDir(categoryDir)
		if err != nil {
			continue
		}
		
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".go") || file.Name() == "types.go" {
				continue
			}
			
			filePath := filepath.Join(categoryDir, file.Name())
			rule, err := extractRuleMetadata(filePath, category)
			if err != nil {
				continue
			}
			
			if rule.ID != "" {
				rules[rule.ID] = rule
			}
		}
	}
	
	return rules, nil
}

// extractRuleMetadata parses a rule file and extracts metadata from comments
func extractRuleMetadata(filePath string, defaultCategory string) (RuleInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return RuleInfo{}, err
	}
	defer file.Close()
	
	rule := RuleInfo{
		Category: defaultCategory,
		FilePath: filePath,
		Enabled:  true,
	}
	
	scanner := bufio.NewScanner(file)
	inComment := false
	commentLines := []string{}
	
	// Read the file to find the metadata comment block
	for scanner.Scan() {
		line := scanner.Text()
		
		if strings.HasPrefix(line, "/*") {
			inComment = true
			continue
		}
		
		if inComment {
			if strings.HasPrefix(line, "*/") {
				inComment = false
				// Parse the collected comment lines
				rule = parseCommentBlock(commentLines, rule)
				break
			}
			commentLines = append(commentLines, line)
		}
		
		// Stop after we've processed the first comment block
		if !inComment && len(commentLines) > 0 {
			break
		}
	}
	
	// Generate ID from filename if not set
	if rule.ID == "" {
		baseName := strings.TrimSuffix(filepath.Base(filePath), ".go")
		rule.ID = baseName
	}
	
	return rule, nil
}

// parseCommentBlock parses the metadata from comment lines
func parseCommentBlock(lines []string, rule RuleInfo) RuleInfo {
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Parse key: value pairs
		if strings.HasPrefix(line, "Rule:") {
			rule.Name = strings.TrimSpace(strings.TrimPrefix(line, "Rule:"))
		} else if strings.HasPrefix(line, "Description:") {
			rule.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		} else if strings.HasPrefix(line, "Reason:") {
			rule.Reason = strings.TrimSpace(strings.TrimPrefix(line, "Reason:"))
		} else if strings.HasPrefix(line, "Category:") {
			rule.Category = strings.TrimSpace(strings.TrimPrefix(line, "Category:"))
		} else if strings.HasPrefix(line, "Severity:") {
			rule.Severity = strings.TrimSpace(strings.TrimPrefix(line, "Severity:"))
		} else if strings.HasPrefix(line, "Standard:") {
			rule.Standard = strings.TrimSpace(strings.TrimPrefix(line, "Standard:"))
		}
	}
	
	return rule
}

// GetRuleCategories returns all available rule categories
func GetRuleCategories() map[string]RuleCategory {
	return map[string]RuleCategory{
		"api": {
			ID:          "api",
			Name:        "API Standards",
			Description: "Rules for API design, implementation, and resource management",
		},
		"cli": {
			ID:          "cli",
			Name:        "CLI & Structure",
			Description: "Rules for command-line interfaces and code structure",
		},
		"config": {
			ID:          "config",
			Name:        "Configuration",
			Description: "Rules for configuration management and environment variables",
		},
		"test": {
			ID:          "test",
			Name:        "Testing",
			Description: "Rules for test coverage and quality",
		},
		"ui": {
			ID:          "ui",
			Name:        "User Interface",
			Description: "Rules for UI components and frontend code",
		},
	}
}

// ConvertRuleInfoToRule converts RuleInfo to the Rule struct used by the API
func ConvertRuleInfoToRule(info RuleInfo) Rule {
	return Rule{
		ID:          info.ID,
		Name:        info.Name,
		Description: info.Description,
		Category:    info.Category,
		Severity:    info.Severity,
		Enabled:     info.Enabled,
		Standard:    info.Standard,
	}
}