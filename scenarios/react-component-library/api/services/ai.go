package services

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/scenarios/react-component-library/models"
)

type AIService struct {
	// Could add configuration for different AI providers
}

func NewAIService() *AIService {
	return &AIService{}
}

// GenerateComponent uses AI to generate a React component
func (s *AIService) GenerateComponent(req models.ComponentGenerationRequest) (*models.ComponentGenerationResponse, error) {
	// Build the prompt for claude-code
	prompt := s.buildGenerationPrompt(req)
	
	// Call claude-code CLI
	code, err := s.callClaudeCode("generate", prompt)
	if err != nil {
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	// Parse the generated response
	response, err := s.parseGeneratedComponent(code, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated component: %w", err)
	}

	return response, nil
}

// ImproveComponent uses AI to suggest improvements for an existing component
func (s *AIService) ImproveComponent(componentID uuid.UUID, req models.ComponentImprovementRequest) (*models.ComponentImprovementResponse, error) {
	// First, get the component (this would require access to ComponentService)
	// For now, we'll simulate this
	componentCode := "const Button = ({ onClick, children }) => <button onClick={onClick}>{children}</button>;"

	// Build improvement prompt
	prompt := s.buildImprovementPrompt(componentCode, req)

	// Call claude-code CLI
	suggestions, err := s.callClaudeCode("improve", prompt)
	if err != nil {
		return nil, fmt.Errorf("AI improvement analysis failed: %w", err)
	}

	// Parse improvement suggestions
	response, err := s.parseImprovementSuggestions(suggestions, req)
	if err != nil {
		return nil, fmt.Errorf("failed to parse improvement suggestions: %w", err)
	}

	return response, nil
}

// buildGenerationPrompt creates a detailed prompt for component generation
func (s *AIService) buildGenerationPrompt(req models.ComponentGenerationRequest) string {
	var prompt strings.Builder
	
	prompt.WriteString("Generate a React component with the following requirements:\n\n")
	prompt.WriteString(fmt.Sprintf("Description: %s\n\n", req.Description))
	
	if len(req.Requirements) > 0 {
		prompt.WriteString("Specific Requirements:\n")
		for _, requirement := range req.Requirements {
			prompt.WriteString(fmt.Sprintf("- %s\n", requirement))
		}
		prompt.WriteString("\n")
	}
	
	if req.Category != "" {
		prompt.WriteString(fmt.Sprintf("Category: %s component\n\n", req.Category))
	}
	
	if req.AccessibilityLevel != "" {
		prompt.WriteString(fmt.Sprintf("Accessibility Level: WCAG %s compliance required\n\n", req.AccessibilityLevel))
	}
	
	if len(req.StylePreferences) > 0 {
		prompt.WriteString("Style Preferences:\n")
		for key, value := range req.StylePreferences {
			prompt.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
		}
		prompt.WriteString("\n")
	}
	
	if len(req.Dependencies) > 0 {
		prompt.WriteString("Allowed Dependencies:\n")
		for _, dep := range req.Dependencies {
			prompt.WriteString(fmt.Sprintf("- %s\n", dep))
		}
		prompt.WriteString("\n")
	}
	
	prompt.WriteString("Please provide:\n")
	prompt.WriteString("1. Complete React component code using modern hooks\n")
	prompt.WriteString("2. TypeScript interfaces for props\n")
	prompt.WriteString("3. Basic usage example\n")
	prompt.WriteString("4. Brief explanation of the component\n\n")
	prompt.WriteString("Format your response as valid React/TypeScript code with comments.")
	
	return prompt.String()
}

// buildImprovementPrompt creates a prompt for component improvement analysis
func (s *AIService) buildImprovementPrompt(code string, req models.ComponentImprovementRequest) string {
	var prompt strings.Builder
	
	prompt.WriteString("Analyze the following React component and suggest improvements:\n\n")
	prompt.WriteString("```typescript\n")
	prompt.WriteString(code)
	prompt.WriteString("\n```\n\n")
	
	if len(req.Focus) > 0 {
		prompt.WriteString("Focus Areas:\n")
		for _, focus := range req.Focus {
			switch focus {
			case models.ImprovementFocusAccessibility:
				prompt.WriteString("- Accessibility: Ensure WCAG compliance, proper ARIA attributes, keyboard navigation\n")
			case models.ImprovementFocusPerformance:
				prompt.WriteString("- Performance: Optimize re-renders, memory usage, bundle size\n")
			case models.ImprovementFocusCodeQuality:
				prompt.WriteString("- Code Quality: Improve maintainability, readability, TypeScript usage\n")
			case models.ImprovementFocusSecurity:
				prompt.WriteString("- Security: Check for XSS vulnerabilities, input sanitization\n")
			}
		}
		prompt.WriteString("\n")
	}
	
	prompt.WriteString("Provide:\n")
	prompt.WriteString("1. Specific improvement suggestions with reasoning\n")
	prompt.WriteString("2. Code examples for each suggestion\n")
	prompt.WriteString("3. Estimated impact (low/medium/high)\n")
	prompt.WriteString("4. Implementation effort (easy/moderate/complex)\n")
	
	if req.Apply {
		prompt.WriteString("5. Complete improved component code\n")
	}
	
	return prompt.String()
}

// callClaudeCode executes claude-code CLI with the given prompt
func (s *AIService) callClaudeCode(operation, prompt string) (string, error) {
	// In real implementation, this would call the actual claude-code CLI
	// For now, we'll return mock responses
	
	switch operation {
	case "generate":
		return s.generateMockComponent(prompt)
	case "improve":
		return s.generateMockImprovement(prompt)
	default:
		return "", fmt.Errorf("unsupported operation: %s", operation)
	}
}

// generateMockComponent returns a mock generated component
func (s *AIService) generateMockComponent(prompt string) (string, error) {
	// This would be replaced with actual claude-code CLI call
	// cmd := exec.Command("resource-claude-code", "generate", "--prompt", prompt)
	// output, err := cmd.Output()
	
	// Mock response for demonstration
	mockResponse := `
// Generated React Button Component
import React from 'react';

interface ButtonProps {
  children: React.ReactNode;
  onClick?: () => void;
  variant?: 'primary' | 'secondary' | 'outline';
  size?: 'small' | 'medium' | 'large';
  disabled?: boolean;
  ariaLabel?: string;
}

const Button: React.FC<ButtonProps> = ({ 
  children, 
  onClick, 
  variant = 'primary',
  size = 'medium',
  disabled = false,
  ariaLabel 
}) => {
  const baseClasses = 'inline-flex items-center justify-center font-medium rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 transition-colors';
  
  const variantClasses = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500',
    secondary: 'bg-gray-600 text-white hover:bg-gray-700 focus:ring-gray-500',
    outline: 'border border-gray-300 text-gray-700 hover:bg-gray-50 focus:ring-blue-500'
  };
  
  const sizeClasses = {
    small: 'px-3 py-2 text-sm',
    medium: 'px-4 py-2 text-sm',
    large: 'px-6 py-3 text-base'
  };
  
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={ariaLabel}
      className={` + "`" + `${baseClasses} ${variantClasses[variant]} ${sizeClasses[size]} ${
        disabled ? 'opacity-50 cursor-not-allowed' : ''
      }` + "`" + `}
    >
      {children}
    </button>
  );
};

// Usage Example:
// <Button variant="primary" size="medium" onClick={() => console.log('Clicked!')}>
//   Click Me
// </Button>

export default Button;
`
	return mockResponse, nil
}

// generateMockImprovement returns mock improvement suggestions
func (s *AIService) generateMockImprovement(prompt string) (string, error) {
	// Mock response for demonstration
	mockResponse := `
Improvement Analysis:

1. ACCESSIBILITY IMPROVEMENTS:
   - Add proper ARIA attributes for better screen reader support
   - Implement keyboard navigation with Enter and Space keys
   - Ensure color contrast meets WCAG AA standards
   Impact: High | Effort: Easy

2. PERFORMANCE OPTIMIZATIONS:
   - Use React.memo to prevent unnecessary re-renders
   - Optimize className string concatenation
   - Consider using CSS modules for better tree-shaking
   Impact: Medium | Effort: Moderate

3. CODE QUALITY ENHANCEMENTS:
   - Add prop validation with more specific TypeScript types
   - Implement proper error boundaries
   - Add unit tests with React Testing Library
   Impact: Medium | Effort: Moderate

4. SECURITY CONSIDERATIONS:
   - Sanitize any dynamic content passed as children
   - Validate onClick handler to prevent XSS
   Impact: Low | Effort: Easy

Improved Component Code:
[Would include the improved component code here if apply=true]
`
	return mockResponse, nil
}

// parseGeneratedComponent parses the AI-generated component response
func (s *AIService) parseGeneratedComponent(aiResponse string, req models.ComponentGenerationRequest) (*models.ComponentGenerationResponse, error) {
	// Extract component name from the code (simple regex or string parsing)
	componentName := s.extractComponentName(aiResponse)
	if componentName == "" {
		componentName = "GeneratedComponent"
	}

	// Generate props schema from the TypeScript interface
	propsSchema := s.extractPropsSchema(aiResponse)

	// Extract example usage
	exampleUsage := s.extractExampleUsage(aiResponse)

	// Generate explanation
	explanation := fmt.Sprintf("Generated %s component based on: %s", componentName, req.Description)

	response := &models.ComponentGenerationResponse{
		GeneratedCode: aiResponse,
		ComponentName: componentName,
		PropsSchema:   propsSchema,
		Explanation:   explanation,
		Dependencies:  req.Dependencies,
		ExampleUsage:  exampleUsage,
	}

	return response, nil
}

// parseImprovementSuggestions parses the AI improvement suggestions
func (s *AIService) parseImprovementSuggestions(aiResponse string, req models.ComponentImprovementRequest) (*models.ComponentImprovementResponse, error) {
	// Parse the improvement suggestions from AI response
	suggestions := []models.ImprovementSuggestion{
		{
			Type:        models.ImprovementFocusAccessibility,
			Title:       "Improve ARIA Support",
			Description: "Add proper ARIA attributes and keyboard navigation",
			Impact:      "high",
			Effort:      "easy",
		},
		{
			Type:        models.ImprovementFocusPerformance,
			Title:       "Optimize Re-renders",
			Description: "Use React.memo to prevent unnecessary re-renders",
			Impact:      "medium",
			Effort:      "moderate",
		},
	}

	// Extract improved code if apply was requested
	var improvedCode string
	if req.Apply {
		improvedCode = s.extractImprovedCode(aiResponse)
	}

	// Calculate estimated impact
	estimatedImpact := map[string]float64{
		"accessibility_score": 15.0,
		"performance_score":   10.0,
		"code_quality_score":  8.0,
	}

	response := &models.ComponentImprovementResponse{
		Suggestions:     suggestions,
		ImprovedCode:    improvedCode,
		Applied:         req.Apply,
		EstimatedImpact: estimatedImpact,
	}

	return response, nil
}

// Helper functions for parsing AI responses

func (s *AIService) extractComponentName(code string) string {
	// Simple extraction - look for "const ComponentName" or "function ComponentName"
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "const ") && strings.Contains(line, ":") {
			parts := strings.Split(line, " ")
			if len(parts) >= 2 {
				name := strings.TrimSuffix(parts[1], ":")
				if name != "" && strings.Title(name) == name { // Check if PascalCase
					return name
				}
			}
		}
	}
	return "GeneratedComponent"
}

func (s *AIService) extractPropsSchema(code string) json.RawMessage {
	// Mock props schema extraction
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"children": map[string]string{
				"type":        "node",
				"description": "Child elements to render",
			},
			"onClick": map[string]string{
				"type":        "function",
				"description": "Click handler function",
			},
		},
		"required": []string{"children"},
	}
	
	schemaBytes, _ := json.Marshal(schema)
	return schemaBytes
}

func (s *AIService) extractExampleUsage(code string) string {
	// Look for usage examples in comments
	lines := strings.Split(code, "\n")
	var exampleLines []string
	inExample := false
	
	for _, line := range lines {
		if strings.Contains(line, "Usage Example") || strings.Contains(line, "Example:") {
			inExample = true
			continue
		}
		if inExample && strings.HasPrefix(strings.TrimSpace(line), "//") {
			exampleLines = append(exampleLines, strings.TrimPrefix(strings.TrimSpace(line), "// "))
		} else if inExample && strings.TrimSpace(line) == "" {
			continue
		} else if inExample {
			break
		}
	}
	
	if len(exampleLines) > 0 {
		return strings.Join(exampleLines, "\n")
	}
	
	return "<GeneratedComponent />"
}

func (s *AIService) extractImprovedCode(response string) string {
	// Look for improved code section
	if strings.Contains(response, "Improved Component Code:") {
		parts := strings.Split(response, "Improved Component Code:")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

// callActualClaudeCode would be used in production to call the real claude-code CLI
func (s *AIService) callActualClaudeCode(operation, prompt string) (string, error) {
	cmd := exec.Command("resource-claude-code", operation, "--prompt", prompt)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}