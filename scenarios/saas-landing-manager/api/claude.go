package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ClaudeCodeService handles Claude Code integration for intelligent deployment
type ClaudeCodeService struct {
	claudeCodeBinary string
	dbService        *DatabaseService
}

func NewClaudeCodeService(claudeCodeBinary string, dbService *DatabaseService) *ClaudeCodeService {
	if claudeCodeBinary == "" {
		claudeCodeBinary = ClaudeDefaultCLI
	}
	return &ClaudeCodeService{
		claudeCodeBinary: claudeCodeBinary,
		dbService:        dbService,
	}
}

func (ccs *ClaudeCodeService) DeployLandingPage(landingPageID, targetScenario string, req *DeployRequest) (*DeployResponse, error) {
	deploymentID := uuid.New().String()

	// Fetch landing page data
	landingPage, err := ccs.dbService.GetLandingPageByID(landingPageID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch landing page: %w", err)
	}

	if req.DeploymentMethod == "claude_agent" {
		// Spawn Claude Code agent for intelligent deployment
		agentSessionID, err := ccs.spawnClaudeAgent(targetScenario, landingPage)
		if err != nil {
			return nil, fmt.Errorf("failed to spawn Claude agent: %w", err)
		}

		return &DeployResponse{
			DeploymentID:        deploymentID,
			AgentSessionID:      agentSessionID,
			Status:              "agent_working",
			EstimatedCompletion: time.Now().Add(ClaudeAgentEstimatedDuration),
		}, nil
	}

	// Direct deployment
	err = ccs.directDeploy(targetScenario, landingPage)
	if err != nil {
		return nil, fmt.Errorf("direct deployment failed: %w", err)
	}

	return &DeployResponse{
		DeploymentID:        deploymentID,
		Status:              "completed",
		EstimatedCompletion: time.Now(),
	}, nil
}

func (ccs *ClaudeCodeService) spawnClaudeAgent(targetScenario string, landingPage *LandingPage) (string, error) {
	// Verify the Claude Code binary exists before attempting to execute
	if _, err := exec.LookPath(ccs.claudeCodeBinary); err != nil {
		return "", fmt.Errorf("claude code binary '%s' not found in PATH: %w", ccs.claudeCodeBinary, err)
	}

	// Create deployment prompt for Claude Code agent with actual landing page data
	prompt := fmt.Sprintf(`
		Deploy landing page to scenario %s.

		Landing Page Details:
		- Title: %s
		- Description: %s
		- Variant: %s
		- Template ID: %s

		Tasks:
		1. Create landing/ directory in target scenario if it doesn't exist
		2. Generate responsive HTML/CSS/JS files using the page title and description
		3. Set up A/B testing routing configuration for variant: %s
		4. Add SEO optimizations and meta tags
		5. Ensure mobile responsiveness and Core Web Vitals compliance
		6. Create backup of existing landing page if present

		Use the landing page data from the saas-landing-manager API endpoint:
		GET /api/v1/landing-pages/%s
	`, targetScenario, landingPage.Title, landingPage.Description, landingPage.Variant, landingPage.TemplateID, landingPage.Variant, landingPage.ID)

	// Execute Claude Code agent spawn
	cmd := exec.Command(ccs.claudeCodeBinary, "spawn", "--task", prompt)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to spawn claude agent: %w", err)
	}

	// Parse agent session ID from output (simplified)
	sessionID := strings.TrimSpace(string(output))
	if sessionID == "" {
		sessionID = uuid.New().String() // Fallback
	}

	return sessionID, nil
}

func (ccs *ClaudeCodeService) directDeploy(targetScenario string, landingPage *LandingPage) error {
	// Direct deployment using actual landing page data
	cleanTarget, err := validateScenarioPath(targetScenario)
	if err != nil {
		return fmt.Errorf("invalid target scenario path: %w", err)
	}
	scenarioPath := filepath.Join("..", "..", cleanTarget)
	landingPath := filepath.Join(scenarioPath, "landing")

	// Create landing directory
	if err := os.MkdirAll(landingPath, 0755); err != nil {
		return err
	}

	// Extract custom hero title from content if available
	heroTitle := "Revolutionary SaaS Solution"
	if landingPage.Content != nil {
		if customTitle, ok := landingPage.Content["hero_title"].(string); ok && customTitle != "" {
			heroTitle = customTitle
		}
	}

	// Use landing page data to create index.html
	indexHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <meta name="description" content="%s">
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
        .hero { text-align: center; padding: 50px 0; }
        .cta { background: #007bff; color: white; padding: 15px 30px; border: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="hero">
        <h1>%s</h1>
        <p>%s</p>
        <button class="cta">Get Started</button>
    </div>
</body>
</html>`, landingPage.Title, landingPage.Description, heroTitle, landingPage.Description)

	err = os.WriteFile(filepath.Join(landingPath, "index.html"), []byte(indexHTML), 0644)
	if err != nil {
		return err
	}

	log.Printf("Successfully deployed landing page '%s' (variant: %s) to %s", landingPage.Title, landingPage.Variant, targetScenario)
	return nil
}
