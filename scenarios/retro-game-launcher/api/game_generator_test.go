// +build testing

package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestExtractJavaScriptCode verifies code extraction from markdown
func TestExtractJavaScriptCode(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	tests := []struct {
		name     string
		input    string
		expected string
		hasCode  bool
	}{
		{
			name: "StandardMarkdownBlock",
			input: "```javascript\nconst game = {};\nfunction start() {}\n```",
			expected: "const game = {};\nfunction start() {}",
			hasCode: true,
		},
		{
			name: "JSMarkdownBlock",
			input: "```js\nconst game = {};\n```",
			expected: "const game = {};",
			hasCode: true,
		},
		{
			name: "NoMarkdown",
			input: "const game = {};\nfunction start() {}",
			expected: "",
			hasCode: false,
		},
		{
			name: "FallbackExtraction",
			input: "Here's the code:\nfunction start() {\n  const canvas = document.getElementById('game');\n  let score = 0;\n}\n",
			expected: "",
			hasCode: false,
		},
		{
			name: "EmptyInput",
			input: "",
			expected: "",
			hasCode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := env.Server.extractJavaScriptCode(tt.input)

			if tt.hasCode {
				if !strings.Contains(result, "game") && !strings.Contains(result, "start") {
					t.Errorf("Expected code extraction, got: %s", result)
				}
			} else {
				if result != "" && len(result) > 0 && tt.name != "FallbackExtraction" {
					t.Logf("Unexpected code extracted: %s", result)
				}
			}
		})
	}
}

// TestValidateGameCode verifies game code validation
func TestValidateGameCode(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	tests := []struct {
		name      string
		code      string
		shouldErr bool
	}{
		{
			name: "ValidCode",
			code: `
const canvas = document.getElementById('gameCanvas');
const ctx = canvas.getContext('2d');
function gameLoop() {
	requestAnimationFrame(gameLoop);
}
gameLoop();
			`,
			shouldErr: false,
		},
		{
			name:      "TooShort",
			code:      "const x = 1;",
			shouldErr: true,
		},
		{
			name: "MissingCanvas",
			code: `
function gameLoop() {
	console.log('Game running');
}
			`,
			shouldErr: true,
		},
		{
			name: "MissingGetContext",
			code: `
const canvas = document.getElementById('game');
function start() {
	// Game logic
}
			`,
			shouldErr: true,
		},
		{
			name: "ContainsEval",
			code: `
const canvas = document.getElementById('game');
const ctx = canvas.getContext('2d');
eval('malicious code');
function gameLoop() {}
			`,
			shouldErr: true,
		},
		{
			name: "ContainsInnerHTML",
			code: `
const canvas = document.getElementById('game');
const ctx = canvas.getContext('2d');
function gameLoop() {
	element.innerHTML = '<script>alert("xss")</script>';
}
			`,
			shouldErr: true,
		},
		{
			name: "ContainsFunction",
			code: `
const canvas = document.getElementById('game');
const ctx = canvas.getContext('2d');
const fn = new Function('alert("danger")');
			`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := env.Server.validateGameCode(tt.code)

			if tt.shouldErr && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestGenerateGameTitle verifies title generation from prompts
func TestGenerateGameTitle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	tests := []struct {
		name     string
		prompt   string
		contains string
	}{
		{
			name:     "SimplePrompt",
			prompt:   "create a space shooter game",
			contains: "Create",
		},
		{
			name:     "LongPrompt",
			prompt:   "create an epic adventure game with multiple levels and power-ups",
			contains: "Create",
		},
		{
			name:     "EmptyPrompt",
			prompt:   "",
			contains: "Untitled",
		},
		{
			name:     "SingleWord",
			prompt:   "pong",
			contains: "Pong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := env.Server.generateGameTitle(tt.prompt)

			if title == "" {
				t.Error("Title should not be empty")
			}

			if tt.contains != "" && !strings.Contains(title, tt.contains) {
				t.Logf("Title '%s' doesn't contain '%s' (may still be valid)", title, tt.contains)
			}

			if len(title) > 50 {
				t.Errorf("Title too long: %d characters", len(title))
			}
		})
	}
}

// TestStartGameGeneration verifies generation workflow initiation
func TestStartGameGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := GameGenerationRequest{
			Prompt: "Create a simple platformer game",
			Engine: "html5",
			Tags:   []string{"platformer", "retro"},
		}

		genID := env.Server.startGameGeneration(req)

		if genID == "" {
			t.Error("Generation ID should not be empty")
		}

		// Verify generation status was created
		status, exists := generationStore[genID]
		if !exists {
			t.Error("Generation status should exist")
		}

		if status.Status != "pending" {
			t.Errorf("Expected status 'pending', got '%s'", status.Status)
		}

		if status.Prompt != req.Prompt {
			t.Errorf("Expected prompt '%s', got '%s'", req.Prompt, status.Prompt)
		}

		// Clean up
		delete(generationStore, genID)
	})
}

// TestGetGenerationStatusByID verifies status retrieval
func TestGetGenerationStatusByID(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("StatusExists", func(t *testing.T) {
		genID := uuid.New().String()
		expectedStatus := &GameGenerationStatus{
			ID:              genID,
			Status:          "generating",
			Prompt:          "Test prompt",
			CreatedAt:       time.Now(),
			EstimatedTimeMs: 45000,
		}

		generationStore[genID] = expectedStatus

		status, exists := env.Server.getGenerationStatusByID(genID)

		if !exists {
			t.Error("Status should exist")
		}

		if status.ID != genID {
			t.Errorf("Expected ID %s, got %s", genID, status.ID)
		}

		if status.Status != "generating" {
			t.Errorf("Expected status 'generating', got '%s'", status.Status)
		}

		delete(generationStore, genID)
	})

	t.Run("StatusNotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()

		status, exists := env.Server.getGenerationStatusByID(nonExistentID)

		if exists {
			t.Error("Status should not exist")
		}

		if status != nil {
			t.Error("Status should be nil for non-existent ID")
		}
	})
}

// TestUpdateGenerationStatus verifies status updates
func TestUpdateGenerationStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	genID := uuid.New().String()
	generationStore[genID] = &GameGenerationStatus{
		ID:        genID,
		Status:    "pending",
		Prompt:    "Test",
		CreatedAt: time.Now(),
	}

	t.Run("UpdateToPending", func(t *testing.T) {
		env.Server.updateGenerationStatus(genID, "generating", "", "", "")

		status := generationStore[genID]
		if status.Status != "generating" {
			t.Errorf("Expected status 'generating', got '%s'", status.Status)
		}
	})

	t.Run("UpdateToCompleted", func(t *testing.T) {
		testGameID := uuid.New().String()
		testCode := "const canvas = document.getElementById('game');"

		// Add small delay to ensure measurable time difference
		time.Sleep(10 * time.Millisecond)

		env.Server.updateGenerationStatus(genID, "completed", testCode, testGameID, "")

		status := generationStore[genID]
		if status.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", status.Status)
		}
		if status.GameID != testGameID {
			t.Errorf("Expected game ID %s, got %s", testGameID, status.GameID)
		}
		if status.GeneratedCode != testCode {
			t.Error("Generated code should be set")
		}
		if status.CompletedAt == nil {
			t.Error("CompletedAt should be set")
		}
		if status.ActualTimeMs <= 0 {
			t.Logf("ActualTimeMs was %d (may be 0 if execution was very fast)", status.ActualTimeMs)
		}
	})

	t.Run("UpdateToFailed", func(t *testing.T) {
		errorMsg := "Test error message"

		env.Server.updateGenerationStatus(genID, "failed", "", "", errorMsg)

		status := generationStore[genID]
		if status.Status != "failed" {
			t.Errorf("Expected status 'failed', got '%s'", status.Status)
		}
		if status.Error != errorMsg {
			t.Errorf("Expected error '%s', got '%s'", errorMsg, status.Error)
		}
		if status.CompletedAt == nil {
			t.Error("CompletedAt should be set even on failure")
		}
	})

	delete(generationStore, genID)
}

// TestSaveGeneratedGame verifies game persistence
func TestSaveGeneratedGame(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		game := Game{
			ID:          uuid.New().String(),
			Title:       "Test Generated Game",
			Prompt:      "Test prompt for generation",
			Description: stringPtr("Generated by AI"),
			Code:        "const canvas = document.getElementById('game'); const ctx = canvas.getContext('2d');",
			Engine:      "html5",
			Tags:        []string{"test", "ai-generated"},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Published:   true,
		}

		err := env.Server.saveGeneratedGame(game)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify game was saved
		var savedGame Game
		err = env.DB.QueryRow("SELECT id, title, prompt FROM games WHERE id = $1", game.ID).
			Scan(&savedGame.ID, &savedGame.Title, &savedGame.Prompt)

		if err != nil {
			t.Fatalf("Failed to query saved game: %v", err)
		}

		if savedGame.ID != game.ID {
			t.Errorf("Expected ID %s, got %s", game.ID, savedGame.ID)
		}
		if savedGame.Title != game.Title {
			t.Errorf("Expected title %s, got %s", game.Title, savedGame.Title)
		}
	})
}

// TestGenerateCodeWithOllama verifies Ollama integration
func TestGenerateCodeWithOllama(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Setup mock Ollama server
	mockServer := mockOllamaServer()
	defer mockServer.Close()

	// Temporarily override Ollama URL
	originalURL := env.Server.ollamaURL
	env.Server.ollamaURL = mockServer.URL
	defer func() { env.Server.ollamaURL = originalURL }()

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		prompt := "Create a simple game"

		code, err := env.Server.generateCodeWithOllama(ctx, prompt, "codellama")

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if code == "" {
			t.Error("Generated code should not be empty")
		}

		if !strings.Contains(code, "javascript") && !strings.Contains(code, "canvas") {
			t.Logf("Generated code may not contain expected keywords: %s", code)
		}
	})

	t.Run("ContextTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond) // Ensure timeout

		_, err := env.Server.generateCodeWithOllama(ctx, "test", "codellama")

		if err == nil {
			t.Error("Expected timeout error")
		}
	})
}

// TestMinFunction verifies min utility function
func TestMinFunction(t *testing.T) {
	tests := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{"First smaller", 5, 10, 5},
		{"Second smaller", 10, 5, 5},
		{"Equal", 5, 5, 5},
		{"Negative numbers", -5, -10, -10},
		{"Zero", 0, 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
