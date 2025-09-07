package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"email-triage/models"
)

// RuleService handles triage rule management and AI generation
type RuleService struct {
	db        *sql.DB
	ollamaURL string
}

// NewRuleService creates a new RuleService instance
func NewRuleService(db *sql.DB, ollamaURL string) *RuleService {
	return &RuleService{
		db:        db,
		ollamaURL: ollamaURL,
	}
}

// GenerateRuleWithAI uses Ollama to generate email triage rules from natural language
func (rs *RuleService) GenerateRuleWithAI(description string) (*models.CreateRuleResponse, error) {
	// Create AI prompt for rule generation
	prompt := fmt.Sprintf(`You are an expert email management assistant. Generate specific email triage rule conditions based on this description: "%s"

Return a JSON response with the following structure:
{
  "conditions": [
    {
      "field": "sender|subject|body",
      "operator": "contains|equals|matches|starts_with|ends_with",
      "value": "specific value to match",
      "weight": 0.0-1.0
    }
  ],
  "confidence": 0.0-1.0,
  "reasoning": "brief explanation of the generated conditions"
}

Examples:
- "Archive newsletters" → conditions checking for promotional keywords, unsubscribe links
- "Forward VIP emails to assistant" → conditions checking for specific sender domains/addresses
- "Mark urgent emails with deadlines" → conditions checking for urgency keywords

Be specific and practical. Use multiple conditions with appropriate weights.`, description)

	// Call Ollama via CLI
	output, err := rs.callOllama(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI rule generation failed: %w", err)
	}

	// Parse AI response
	var aiResponse struct {
		Conditions []models.RuleCondition `json:"conditions"`
		Confidence float64                `json:"confidence"`
		Reasoning  string                 `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(output), &aiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Validate conditions
	if len(aiResponse.Conditions) == 0 {
		return nil, fmt.Errorf("AI generated no valid conditions")
	}

	// Estimate preview matches (simplified - would query database in production)
	previewMatches := rs.estimateMatches(aiResponse.Conditions)

	return &models.CreateRuleResponse{
		Success:             true,
		GeneratedConditions: aiResponse.Conditions,
		AIConfidence:        aiResponse.Confidence,
		PreviewMatches:      previewMatches,
	}, nil
}

// callOllama executes Ollama CLI command for AI inference
func (rs *RuleService) callOllama(prompt string) (string, error) {
	// Use the resource CLI pattern: resource-ollama generate
	cmd := exec.Command("bash", "-c", fmt.Sprintf(`echo '%s' | vrooli resource ollama generate --model phi3.5:3.8b --type reasoning --quiet`, strings.ReplaceAll(prompt, "'", "'\\''")))
	
	output, err := cmd.Output()
	if err != nil {
		// Fallback to direct ollama command if resource CLI unavailable
		cmd = exec.Command("ollama", "run", "phi3.5:3.8b", prompt)
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("ollama command failed: %w", err)
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// estimateMatches provides rough estimate of how many emails would match these conditions
func (rs *RuleService) estimateMatches(conditions []models.RuleCondition) int {
	// Simplified estimation - in production this would query the processed_emails table
	// For now, return a reasonable estimate based on condition types
	
	baseEstimate := 10 // Base estimate for specific conditions
	
	for _, condition := range conditions {
		switch condition.Field {
		case "sender":
			if condition.Operator == "contains" && strings.Contains(strings.ToLower(condition.Value.(string)), "newsletter") {
				baseEstimate += 50 // Newsletters are common
			} else {
				baseEstimate += 5 // Specific senders
			}
		case "subject":
			if strings.Contains(strings.ToLower(condition.Value.(string)), "urgent") {
				baseEstimate += 20 // Urgent emails are moderately common
			} else {
				baseEstimate += 10 // Subject-based rules
			}
		case "body":
			baseEstimate += 15 // Body content matching
		}
	}
	
	return baseEstimate
}

// CreateRule creates a new triage rule in the database
func (rs *RuleService) CreateRule(userID string, request *models.CreateRuleRequest) (*models.CreateRuleResponse, error) {
	var conditions []models.RuleCondition
	var aiConfidence float64
	
	if request.UseAIGeneration {
		// Generate conditions with AI
		aiResponse, err := rs.GenerateRuleWithAI(request.Description)
		if err != nil {
			return nil, fmt.Errorf("AI rule generation failed: %w", err)
		}
		conditions = aiResponse.GeneratedConditions
		aiConfidence = aiResponse.AIConfidence
	} else {
		// Use manual conditions
		conditions = request.ManualConditions
		aiConfidence = 1.0 // Manual rules have full confidence
	}
	
	// Serialize conditions and actions
	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize conditions: %w", err)
	}
	
	actionsJSON, err := json.Marshal(request.Actions)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize actions: %w", err)
	}
	
	// Insert rule into database
	ruleID := generateUUID()
	query := `
		INSERT INTO triage_rules (
			id, user_id, name, description, conditions, actions, 
			priority, enabled, created_by_ai, ai_confidence, 
			match_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`
	
	now := time.Now()
	_, err = rs.db.Exec(query,
		ruleID, userID, extractRuleName(request.Description), request.Description,
		conditionsJSON, actionsJSON, request.Priority, true,
		request.UseAIGeneration, aiConfidence, 0, now, now)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}
	
	// Estimate preview matches
	previewMatches := rs.estimateMatches(conditions)
	
	return &models.CreateRuleResponse{
		Success:             true,
		RuleID:              ruleID,
		GeneratedConditions: conditions,
		AIConfidence:        aiConfidence,
		PreviewMatches:      previewMatches,
	}, nil
}

// GetRulesByUser retrieves all rules for a specific user
func (rs *RuleService) GetRulesByUser(userID string) ([]*models.TriageRule, error) {
	query := `
		SELECT id, user_id, name, description, conditions, actions, 
			   priority, enabled, created_by_ai, ai_confidence, 
			   match_count, created_at, updated_at
		FROM triage_rules 
		WHERE user_id = $1 
		ORDER BY priority ASC, created_at DESC`
	
	rows, err := rs.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()
	
	var rules []*models.TriageRule
	
	for rows.Next() {
		var rule models.TriageRule
		err := rows.Scan(
			&rule.ID, &rule.UserID, &rule.Name, &rule.Description,
			&rule.Conditions, &rule.Actions, &rule.Priority, &rule.Enabled,
			&rule.CreatedByAI, &rule.AIConfidence, &rule.MatchCount,
			&rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		
		rules = append(rules, &rule)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}
	
	return rules, nil
}

// TestRuleAgainstEmail tests if a rule would match against a specific email
func (rs *RuleService) TestRuleAgainstEmail(rule *models.TriageRule, email *models.ProcessedEmail) (bool, float64, error) {
	// Parse rule conditions
	var conditions []models.RuleCondition
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return false, 0, fmt.Errorf("failed to parse rule conditions: %w", err)
	}
	
	totalWeight := 0.0
	matchedWeight := 0.0
	
	// Evaluate each condition
	for _, condition := range conditions {
		totalWeight += condition.Weight
		
		if rs.evaluateCondition(condition, email) {
			matchedWeight += condition.Weight
		}
	}
	
	// Calculate match confidence (weighted average)
	var confidence float64
	if totalWeight > 0 {
		confidence = matchedWeight / totalWeight
	}
	
	// Rule matches if confidence is above threshold (e.g., 0.6)
	matches := confidence >= 0.6
	
	return matches, confidence, nil
}

// evaluateCondition checks if a single condition matches an email
func (rs *RuleService) evaluateCondition(condition models.RuleCondition, email *models.ProcessedEmail) bool {
	var fieldValue string
	
	// Extract field value from email
	switch condition.Field {
	case "sender":
		fieldValue = strings.ToLower(email.SenderEmail)
	case "subject":
		fieldValue = strings.ToLower(email.Subject)
	case "body":
		fieldValue = strings.ToLower(email.FullBody)
	default:
		return false
	}
	
	// Get condition value as string
	conditionValue := strings.ToLower(fmt.Sprintf("%v", condition.Value))
	
	// Apply operator
	switch condition.Operator {
	case "contains":
		return strings.Contains(fieldValue, conditionValue)
	case "equals":
		return fieldValue == conditionValue
	case "starts_with":
		return strings.HasPrefix(fieldValue, conditionValue)
	case "ends_with":
		return strings.HasSuffix(fieldValue, conditionValue)
	case "matches":
		// Simple regex matching (simplified)
		return strings.Contains(fieldValue, conditionValue)
	default:
		return false
	}
}

// extractRuleName generates a rule name from description
func extractRuleName(description string) string {
	// Take first 50 characters and clean up
	name := description
	if len(name) > 50 {
		name = name[:47] + "..."
	}
	
	// Capitalize first letter
	if len(name) > 0 {
		name = strings.ToUpper(string(name[0])) + name[1:]
	}
	
	return name
}