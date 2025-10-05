package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalMode string
}

// setupTestLogger initializes Gin in test mode and returns cleanup function
func setupTestLogger() func() {
	originalMode := gin.Mode()
	gin.SetMode(gin.TestMode)

	// Suppress log output during tests
	log.SetOutput(io.Discard)

	return func() {
		gin.SetMode(originalMode)
		log.SetOutput(os.Stderr)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Router     *gin.Engine
	Cleanup    func()
}

// setupTestEnvironment creates an isolated test environment with mock dependencies
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	// Set test mode environment variable
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "8080")

	// Setup Gin router in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add basic middleware
	router.Use(gin.Recovery())

	return &TestEnvironment{
		Router: router,
		Cleanup: func() {
			// Cleanup test environment
			os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
			os.Unsetenv("API_PORT")
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader

	if req.Body != nil {
		var bodyBytes []byte
		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, _ = json.Marshal(v)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set headers
	if req.Headers != nil {
		for key, value := range req.Headers {
			httpReq.Header.Set(key, value)
		}
	}

	// Set default content type for POST/PUT requests with body
	if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set query parameters
	if req.QueryParams != nil {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Set(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields if provided
	if expectedFields != nil {
		for key, expectedValue := range expectedFields {
			actualValue, exists := response[key]
			if !exists {
				t.Errorf("Expected field '%s' not found in response", key)
				continue
			}

			if expectedValue != nil && actualValue != expectedValue {
				// Handle type conversion for numbers
				switch exp := expectedValue.(type) {
				case int:
					if act, ok := actualValue.(float64); ok {
						if int(act) != exp {
							t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
						}
					}
				case float64:
					if act, ok := actualValue.(float64); ok {
						if act != exp {
							t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
						}
					}
				default:
					if actualValue != expectedValue {
						t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
					}
				}
			}
		}
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON error response: %v. Response: %s", err, w.Body.String())
		return
	}

	errorMsg, exists := response["error"]
	if !exists {
		t.Error("Expected 'error' field in response")
		return
	}

	errorStr, ok := errorMsg.(string)
	if !ok {
		t.Errorf("Expected error to be a string, got %T", errorMsg)
		return
	}

	if expectedErrorSubstring != "" && !strings.Contains(errorStr, expectedErrorSubstring) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorSubstring, errorStr)
	}
}

// assertJSONArray validates that response contains an array
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, arrayField string) []interface{} {
	response := assertJSONResponse(t, w, expectedStatus, nil)
	if response == nil {
		return nil
	}

	array, ok := response[arrayField].([]interface{})
	if !ok {
		t.Errorf("Expected field '%s' to be an array, got %T", arrayField, response[arrayField])
		return nil
	}

	return array
}

// TestCharacter provides a pre-configured character for testing
type TestCharacter struct {
	Character Character
	Cleanup   func()
}

// createTestCharacter creates a test character with sample data
func createTestCharacter(t *testing.T, name string) *TestCharacter {
	character := Character{
		ID:   uuid.New(),
		Name: name,
		PersonalityTraits: map[string]interface{}{
			"brave":    0.8,
			"humorous": 0.6,
			"wise":     0.7,
		},
		BackgroundStory: fmt.Sprintf("%s is a brave jungle explorer", name),
		SpeechPatterns: map[string]interface{}{
			"formality": "casual",
			"tempo":     "moderate",
		},
		Relationships: map[string]interface{}{
			"allies": []string{"companion"},
		},
		VoiceProfile: map[string]interface{}{
			"pitch": "medium",
			"tone":  "confident",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	return &TestCharacter{
		Character: character,
		Cleanup: func() {
			// Cleanup if needed
		},
	}
}

// TestProject provides a pre-configured project for testing
type TestProject struct {
	Project Project
	Cleanup func()
}

// createTestProject creates a test project with sample data
func createTestProject(t *testing.T, name string) *TestProject {
	project := Project{
		ID:          uuid.New(),
		Name:        name,
		Description: fmt.Sprintf("Test project: %s", name),
		Characters:  []string{"char1", "char2"},
		Settings: map[string]interface{}{
			"theme": "jungle-adventure",
			"style": "action",
		},
		ExportFormat: "json",
		CreatedAt:    time.Now().UTC(),
	}

	return &TestProject{
		Project: project,
		Cleanup: func() {
			// Cleanup if needed
		},
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CharacterRequest creates a test character creation request
func (g *TestDataGenerator) CharacterRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
		"personality_traits": map[string]interface{}{
			"brave":    0.8,
			"humorous": 0.6,
		},
		"background_story": fmt.Sprintf("%s is a jungle explorer", name),
		"speech_patterns": map[string]interface{}{
			"formality": "casual",
		},
		"relationships": map[string]interface{}{
			"allies": []string{},
		},
		"voice_profile": map[string]interface{}{
			"pitch": "medium",
		},
	}
}

// DialogRequest creates a test dialog generation request
func (g *TestDataGenerator) DialogRequest(characterID string, sceneContext string) DialogGenerationRequest {
	return DialogGenerationRequest{
		CharacterID:  characterID,
		SceneContext: sceneContext,
		EmotionState: "neutral",
		PreviousDialog: []string{
			"Previous line of dialog",
		},
		Constraints: map[string]interface{}{
			"max_length": 100,
		},
	}
}

// BatchDialogRequest creates a test batch dialog request
func (g *TestDataGenerator) BatchDialogRequest(sceneID string, requests []DialogGenerationRequest) BatchDialogRequest {
	return BatchDialogRequest{
		SceneID:        sceneID,
		DialogRequests: requests,
		ExportFormat:   "json",
	}
}

// ProjectRequest creates a test project creation request
func (g *TestDataGenerator) ProjectRequest(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": fmt.Sprintf("Test project: %s", name),
		"characters":  []string{"char1", "char2"},
		"settings": map[string]interface{}{
			"theme": "jungle-adventure",
		},
		"export_format": "json",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// MockDatabase provides in-memory database mocking for tests
type MockDatabase struct {
	characters map[string]Character
	projects   map[string]Project
	dialogs    map[string]DialogLine
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		characters: make(map[string]Character),
		projects:   make(map[string]Project),
		dialogs:    make(map[string]DialogLine),
	}
}

// SaveCharacter saves a character to mock database
func (m *MockDatabase) SaveCharacter(char Character) error {
	m.characters[char.ID.String()] = char
	return nil
}

// GetCharacter retrieves a character from mock database
func (m *MockDatabase) GetCharacter(id string) (Character, error) {
	char, exists := m.characters[id]
	if !exists {
		return Character{}, fmt.Errorf("character not found")
	}
	return char, nil
}

// SaveProject saves a project to mock database
func (m *MockDatabase) SaveProject(proj Project) error {
	m.projects[proj.ID.String()] = proj
	return nil
}

// GetProject retrieves a project from mock database
func (m *MockDatabase) GetProject(id string) (Project, error) {
	proj, exists := m.projects[id]
	if !exists {
		return Project{}, fmt.Errorf("project not found")
	}
	return proj, nil
}
