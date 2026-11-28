package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test-only helpers moved from production files by the private-function debt lens.
// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use test database URL from environment or default
	testDBURL := os.Getenv("TEST_DATABASE_URL")
	if testDBURL == "" {
		testDBURL = os.Getenv("DATABASE_URL")
	}

	if testDBURL == "" {
		t.Skip("DATABASE_URL not set, skipping database tests")
	}

	cfg := &Config{
		DatabaseURL: testDBURL,
	}

	db, err := NewDatabase(cfg, logger)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}

	return &TestDatabase{
		db: db,
		Cleanup: func() {
			db.Close()
		},
	}
}

// setupTestChatbot creates a test chatbot with sample data
func setupTestChatbot(t *testing.T, db *Database, name string) *TestChatbot {
	chatbot := &Chatbot{
		ID:            uuid.New().String(),
		Name:          name,
		Description:   fmt.Sprintf("Test chatbot: %s", name),
		Personality:   "friendly and helpful",
		KnowledgeBase: "Test knowledge base",
		ModelConfig: map[string]interface{}{
			"model":       "llama2",
			"temperature": 0.7,
		},
		WidgetConfig: map[string]interface{}{
			"theme":    "light",
			"position": "bottom-right",
		},
		EscalationConfig: map[string]interface{}{
			"enabled": true,
		},
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save chatbot to database if provided
	if db != nil {
		err := db.CreateChatbot(chatbot)
		if err != nil {
			t.Fatalf("Failed to create test chatbot: %v", err)
		}
	}

	return &TestChatbot{
		Chatbot: chatbot,
		Cleanup: func() {
			if db != nil {
				db.DeleteChatbot(chatbot.ID)
			}
		},
	}
}

// skipIfOllamaUnavailable skips the test if Ollama is not available
func skipIfOllamaUnavailable(t *testing.T) {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		t.Skip("OLLAMA_URL not set, skipping Ollama-dependent tests")
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(ollamaURL)
	if err != nil {
		t.Skipf("Ollama not available at %s: %v", ollamaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Skipf("Ollama returned unexpected status: %d", resp.StatusCode)
	}
}

// assertContains checks if a string contains a substring
func assertContains(t *testing.T, haystack, needle string) {
	if !strings.Contains(haystack, needle) {
		t.Errorf("Expected string to contain '%s', got '%s'", needle, haystack)
	}
}
