package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func aiStageIdeasHandler(c *gin.Context) {
	tree, err := resolveTreeContext(c)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "Unable to resolve tech tree"})
		return
	}

	var payload struct {
		SectorID string `json:"sector_id"`
		Prompt   string `json:"prompt"`
		Count    int    `json:"count"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(payload.SectorID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sector_id is required"})
		return
	}
	count := payload.Count
	if count <= 0 {
		count = 3
	}
	if count > 5 {
		count = 5
	}

	ctx := c.Request.Context()
	sector, err := fetchSectorByID(ctx, payload.SectorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sector not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sector"})
		return
	}
	if sector.TreeID != tree.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sector does not belong to selected tech tree"})
		return
	}

	ideas := []StageIdea{}
	if aiIdeas, aiErr := generateIdeasWithOpenRouter(ctx, sector, payload.Prompt, count); aiErr == nil && len(aiIdeas) > 0 {
		ideas = aiIdeas
	} else {
		ideas = generateHeuristicStageIdeas(sector, payload.Prompt, count)
		if aiErr != nil {
			log.Printf("openrouter suggestions unavailable, using heuristics: %v", aiErr)
		}
	}

	c.JSON(http.StatusOK, gin.H{"ideas": ideas, "tree": tree})
}

func generateIdeasWithOpenRouter(ctx context.Context, sector *Sector, hint string, count int) ([]StageIdea, error) {
	modelPrompt := buildStageIdeaPrompt(sector, hint, count)
	if _, err := exec.LookPath("resource-openrouter"); err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	requestCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	model := os.Getenv("TECH_TREE_OPENROUTER_MODEL")
	if model == "" {
		model = "openai/gpt-4o-mini"
	}
	cmd := exec.CommandContext(requestCtx, "resource-openrouter", "infer",
		"--model", model,
		"--max-tokens", "640",
		"--prompt", modelPrompt,
	)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("resource-openrouter error: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	jsonPayload := extractJSONArray(string(output))
	if jsonPayload == "" {
		return nil, errors.New("model response missing JSON array")
	}
	var rawIdeas []struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		StageType   string   `json:"stage_type"`
		Scenarios   []string `json:"scenarios"`
	}
	if err := json.Unmarshal([]byte(jsonPayload), &rawIdeas); err != nil {
		return nil, err
	}
	ideas := make([]StageIdea, 0, len(rawIdeas))
	for _, idea := range rawIdeas {
		stageType := normalizeStageType(idea.StageType)
		ideas = append(ideas, StageIdea{
			Name:               strings.TrimSpace(idea.Name),
			Description:        strings.TrimSpace(idea.Description),
			StageType:          stageType,
			SuggestedScenarios: idea.Scenarios,
			Confidence:         0.72,
			StrategicRationale: "Generated via OpenRouter",
		})
	}
	if len(ideas) == 0 {
		return nil, errors.New("model returned zero ideas")
	}
	return ideas, nil
}

func buildStageIdeaPrompt(sector *Sector, hint string, count int) string {
	var builder strings.Builder
	builder.WriteString("You are the strategic intelligence layer for Vrooli. Return a JSON array (no prose) with stage ideas for the tech tree. \n")
	builder.WriteString("Each element must include name, description, stage_type, and scenarios (array).\n")
	builder.WriteString(fmt.Sprintf("Sector: %s (%s). Current progress %.0f%%.\n", sector.Name, sector.Category, sector.ProgressPercentage))
	if hint != "" {
		builder.WriteString("Focus: " + hint + "\n")
	}
	builder.WriteString(fmt.Sprintf("Respond with exactly %d ideas.", count))
	return builder.String()
}

func extractJSONArray(text string) string {
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return ""
}

func generateHeuristicStageIdeas(sector *Sector, hint string, count int) []StageIdea {
	categoryKey := strings.ToLower(strings.TrimSpace(sector.Category))
	templates, ok := stageIdeaTemplates[categoryKey]
	if !ok || len(templates) == 0 {
		templates = []ideaTemplate{
			{Name: "Capability Fusion Node", StageType: "integration", Description: "Connects adjacent sectors to unlock compounding leverage", Scenarios: []string{"graph-studio", "ecosystem-manager"}},
			{Name: "Insight Amplifier", StageType: "analytics", Description: "Turns raw telemetry into compounding intelligence", Scenarios: []string{"system-monitor", "product-manager-agent"}},
		}
	}
	ideas := make([]StageIdea, 0, count)
	for len(ideas) < count {
		template := templates[len(ideas)%len(templates)]
		description := template.Description
		if hint != "" {
			description = fmt.Sprintf("%s\nFocus: %s", template.Description, hint)
		}
		ideas = append(ideas, StageIdea{
			Name:               fmt.Sprintf("%s • %s", sector.Name, template.Name),
			Description:        description,
			StageType:          normalizeStageType(template.StageType),
			SuggestedScenarios: append([]string{}, template.Scenarios...),
			Confidence:         0.61,
			StrategicRationale: fmt.Sprintf("Strengthens %s sector flywheel", sector.Name),
		})
	}
	return ideas
}
