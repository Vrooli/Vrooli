package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OllamaRequest struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	Stream    bool   `json:"stream"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

type OllamaResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int64     `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int64     `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

type GameGenerationStatus struct {
	ID               string    `json:"id"`
	Status           string    `json:"status"` // "pending", "generating", "validating", "completed", "failed"
	Prompt           string    `json:"prompt"`
	GeneratedCode    string    `json:"generated_code,omitempty"`
	GameID           string    `json:"game_id,omitempty"`
	Error            string    `json:"error,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	EstimatedTimeMs  int       `json:"estimated_time_ms"`
	ActualTimeMs     int       `json:"actual_time_ms,omitempty"`
}

// In-memory store for generation status (in production, use Redis or database)
var generationStore = make(map[string]*GameGenerationStatus)

func (s *APIServer) generateGameWithAI(ctx context.Context, req GameGenerationRequest, generationID string) error {
	// Update status to generating
	if status, exists := generationStore[generationID]; exists {
		status.Status = "generating"
	}

	// Prepare prompt for Ollama
	prompt := fmt.Sprintf(`Create a retro JavaScript game based on this prompt: %s

Generate complete, playable game code using HTML5 Canvas. Include:
1. Game loop with requestAnimationFrame
2. Player controls (keyboard input)
3. Score system with display
4. Game over condition
5. Retro pixel art style with simple shapes
6. Sound effects using Web Audio API (optional)

Requirements:
- Use only vanilla JavaScript (no libraries)
- Game should fit in a 800x600 canvas
- Include clear win/lose conditions
- Make it fun and engaging
- Use bright, contrasting colors for retro feel

Return ONLY the complete JavaScript code wrapped in ```javascript``` tags. No explanations or additional text.`, req.Prompt)

	// Generate game code with Ollama
	generatedCode, err := s.generateCodeWithOllama(ctx, prompt, "codellama")
	if err != nil {
		s.updateGenerationStatus(generationID, "failed", "", "", err.Error())
		return err
	}

	// Extract JavaScript code from markdown
	jsCode := s.extractJavaScriptCode(generatedCode)
	if jsCode == "" {
		err := errors.New("failed to extract valid JavaScript code from AI response")
		s.updateGenerationStatus(generationID, "failed", "", "", err.Error())
		return err
	}

	// Update status to validating
	if status, exists := generationStore[generationID]; exists {
		status.Status = "validating"
		status.GeneratedCode = jsCode
	}

	// Basic code validation
	if err := s.validateGameCode(jsCode); err != nil {
		s.updateGenerationStatus(generationID, "failed", jsCode, "", err.Error())
		return err
	}

	// Create game title from prompt (first few words)
	title := s.generateGameTitle(req.Prompt)
	
	// Save game to database
	gameID := uuid.New().String()
	game := Game{
		ID:          gameID,
		Title:       title,
		Prompt:      req.Prompt,
		Description: &req.Prompt,
		Code:        jsCode,
		Engine:      req.Engine,
		AuthorID:    req.AuthorID,
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Published:   true,
	}

	err = s.saveGeneratedGame(game)
	if err != nil {
		s.updateGenerationStatus(generationID, "failed", jsCode, "", err.Error())
		return err
	}

	// Generate assets in background (optional)
	go s.generateGameAssets(ctx, gameID, title)

	// Mark as completed
	s.updateGenerationStatus(generationID, "completed", jsCode, gameID, "")

	return nil
}

func (s *APIServer) generateCodeWithOllama(ctx context.Context, prompt, model string) (string, error) {
	reqBody := OllamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
		Options: map[string]interface{}{
			"temperature": 0.7,
			"top_p":       0.9,
			"top_k":       40,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama request failed: %s", string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}

	return ollamaResp.Response, nil
}

func (s *APIServer) extractJavaScriptCode(response string) string {
	// Extract code from markdown blocks
	codeBlockRegex := regexp.MustCompile("```(?:javascript|js)\\s*\\n([\\s\\S]*?)\\n```")
	matches := codeBlockRegex.FindStringSubmatch(response)
	
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback: look for any code-like content
	lines := strings.Split(response, "\n")
	var codeLines []string
	inCode := false

	for _, line := range lines {
		if strings.Contains(line, "function") || strings.Contains(line, "const") || strings.Contains(line, "let") || strings.Contains(line, "var") {
			inCode = true
		}
		if inCode {
			codeLines = append(codeLines, line)
		}
		if strings.Contains(line, "}") && len(codeLines) > 10 {
			break
		}
	}

	if len(codeLines) > 5 {
		return strings.Join(codeLines, "\n")
	}

	return ""
}

func (s *APIServer) validateGameCode(code string) error {
	// Basic validation checks
	if len(code) < 100 {
		return errors.New("generated code is too short")
	}

	// Check for essential game elements
	essentialElements := []string{
		"canvas",
		"getContext",
		"function",
	}

	for _, element := range essentialElements {
		if !strings.Contains(code, element) {
			return fmt.Errorf("missing essential game element: %s", element)
		}
	}

	// Check for basic security (no eval, no external scripts)
	dangerousPatterns := []string{
		"eval(",
		"Function(",
		"<script",
		"document.write",
		"innerHTML",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(code, pattern) {
			return fmt.Errorf("code contains potentially dangerous pattern: %s", pattern)
		}
	}

	return nil
}

func (s *APIServer) generateGameTitle(prompt string) string {
	words := strings.Fields(prompt)
	if len(words) == 0 {
		return "Untitled Game"
	}

	// Take first 3-5 words and capitalize
	titleWords := words[:min(len(words), 4)]
	for i, word := range titleWords {
		if len(word) > 0 {
			titleWords[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	title := strings.Join(titleWords, " ")
	if len(title) > 50 {
		title = title[:47] + "..."
	}

	return title
}

func (s *APIServer) saveGeneratedGame(game Game) error {
	tagsJSON, _ := json.Marshal(game.Tags)

	query := `
		INSERT INTO games (id, title, prompt, description, code, engine, 
		                  author_id, tags, created_at, updated_at, published)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := s.db.Exec(query,
		game.ID, game.Title, game.Prompt, game.Description, game.Code,
		game.Engine, game.AuthorID, tagsJSON, game.CreatedAt, game.UpdatedAt, game.Published,
	)

	return err
}

func (s *APIServer) generateGameAssets(ctx context.Context, gameID, title string) {
	// Generate simple game assets using AI (optional feature)
	prompt := fmt.Sprintf("Generate retro pixel art assets for game: %s\nReturn as base64 encoded 32x32 sprite data.", title)
	
	assets, err := s.generateCodeWithOllama(ctx, prompt, "llama3.2")
	if err != nil {
		// Asset generation is optional, don't fail the entire game creation
		return
	}

	// Save assets to database or file system (implementation depends on requirements)
	query := `UPDATE games SET thumbnail_url = $1 WHERE id = $2`
	s.db.Exec(query, assets, gameID)
}

func (s *APIServer) updateGenerationStatus(id, status, code, gameID, errorMsg string) {
	if genStatus, exists := generationStore[id]; exists {
		genStatus.Status = status
		genStatus.GeneratedCode = code
		genStatus.GameID = gameID
		genStatus.Error = errorMsg
		
		if status == "completed" || status == "failed" {
			now := time.Now()
			genStatus.CompletedAt = &now
			genStatus.ActualTimeMs = int(now.Sub(genStatus.CreatedAt).Milliseconds())
		}
	}
}

func (s *APIServer) startGameGeneration(req GameGenerationRequest) string {
	generationID := uuid.New().String()
	
	status := &GameGenerationStatus{
		ID:              generationID,
		Status:          "pending",
		Prompt:          req.Prompt,
		CreatedAt:       time.Now(),
		EstimatedTimeMs: 45000, // 45 seconds estimate
	}
	
	generationStore[generationID] = status
	
	// Start generation in background
	go func() {
		ctx := context.Background()
		s.generateGameWithAI(ctx, req, generationID)
	}()
	
	return generationID
}

func (s *APIServer) getGenerationStatusByID(id string) (*GameGenerationStatus, bool) {
	status, exists := generationStore[id]
	return status, exists
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}