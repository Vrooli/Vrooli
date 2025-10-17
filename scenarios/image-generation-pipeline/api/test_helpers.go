// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalLogger *log.Logger
	cleanup        func()
}

// setupTestLogger initializes the global logger for testing
func setupTestLogger() func() {
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[test] ")
	return func() {
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	}
}

// TestDatabase manages test database connection
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use test database URL from environment or default
	testDBURL := os.Getenv("TEST_POSTGRES_URL")
	if testDBURL == "" {
		testDBURL = "postgres://postgres:postgres@localhost:5432/image_gen_test?sslmode=disable"
	}

	testDB, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Try to ping the database
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test: database not reachable: %v", err)
		return nil
	}

	// Store original db and replace with test db
	originalDB := db
	db = testDB

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			// Clean up test data
			testDB.Exec("DELETE FROM image_generations")
			testDB.Exec("DELETE FROM campaigns")
			testDB.Exec("DELETE FROM brands")

			// Restore original db and close test db
			db = originalDB
			testDB.Close()
		},
	}
}

// TestBrand provides a pre-configured brand for testing
type TestBrand struct {
	Brand   *Brand
	Cleanup func()
}

// setupTestBrand creates a test brand with sample data
func setupTestBrand(t *testing.T, name string) *TestBrand {
	description := fmt.Sprintf("Test brand: %s", name)
	guidelines := "Test brand guidelines"
	logoURL := "https://example.com/logo.png"

	brand := &Brand{
		ID:          uuid.New().String(),
		Name:        name,
		Description: &description,
		Guidelines:  &guidelines,
		Colors:      []string{"#FF0000", "#00FF00", "#0000FF"},
		Fonts:       []string{"Arial", "Helvetica"},
		LogoURL:     &logoURL,
	}

	colorsJSON, _ := json.Marshal(brand.Colors)
	fontsJSON, _ := json.Marshal(brand.Fonts)

	if db != nil {
		_, err := db.Exec(`
			INSERT INTO brands (id, name, description, guidelines, colors, fonts, logo_url, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, brand.ID, brand.Name, brand.Description, brand.Guidelines, colorsJSON, fontsJSON, brand.LogoURL)

		if err != nil {
			t.Fatalf("Failed to create test brand: %v", err)
		}
	}

	return &TestBrand{
		Brand: brand,
		Cleanup: func() {
			if db != nil {
				db.Exec("DELETE FROM brands WHERE id = $1", brand.ID)
			}
		},
	}
}

// TestCampaign provides a pre-configured campaign for testing
type TestCampaign struct {
	Campaign *Campaign
	Cleanup  func()
}

// setupTestCampaign creates a test campaign with sample data
func setupTestCampaign(t *testing.T, name string, brandID string) *TestCampaign {
	description := fmt.Sprintf("Test campaign: %s", name)
	budget := 10000.0

	campaign := &Campaign{
		ID:          uuid.New().String(),
		Name:        name,
		BrandID:     brandID,
		Description: &description,
		Status:      "active",
		Budget:      &budget,
		TeamMembers: []string{"user1", "user2"},
	}

	teamMembersJSON, _ := json.Marshal(campaign.TeamMembers)

	if db != nil {
		_, err := db.Exec(`
			INSERT INTO campaigns (id, name, brand_id, description, status, budget, team_members, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		`, campaign.ID, campaign.Name, campaign.BrandID, campaign.Description,
		   campaign.Status, campaign.Budget, teamMembersJSON)

		if err != nil {
			t.Fatalf("Failed to create test campaign: %v", err)
		}
	}

	return &TestCampaign{
		Campaign: campaign,
		Cleanup: func() {
			if db != nil {
				db.Exec("DELETE FROM campaigns WHERE id = $1", campaign.ID)
			}
		},
	}
}

// TestImageGeneration provides a test image generation record
type TestImageGeneration struct {
	Generation *ImageGeneration
	Cleanup    func()
}

// setupTestImageGeneration creates a test image generation record
func setupTestImageGeneration(t *testing.T, campaignID string) *TestImageGeneration {
	voiceBrief := "Test voice brief"
	imageURL := "https://example.com/image.png"
	qualityScore := 0.95

	generation := &ImageGeneration{
		ID:           uuid.New().String(),
		CampaignID:   campaignID,
		Prompt:       "Test prompt for image generation",
		VoiceBrief:   &voiceBrief,
		Status:       "completed",
		ImageURL:     &imageURL,
		QualityScore: &qualityScore,
		Metadata:     map[string]interface{}{"test": "data"},
	}

	metadataJSON, _ := json.Marshal(generation.Metadata)

	if db != nil {
		_, err := db.Exec(`
			INSERT INTO image_generations (id, campaign_id, prompt, voice_brief, status, image_url, quality_score, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`, generation.ID, generation.CampaignID, generation.Prompt, generation.VoiceBrief,
		   generation.Status, generation.ImageURL, generation.QualityScore, metadataJSON)

		if err != nil {
			t.Fatalf("Failed to create test image generation: %v", err)
		}
	}

	return &TestImageGeneration{
		Generation: generation,
		Cleanup: func() {
			if db != nil {
				db.Exec("DELETE FROM image_generations WHERE id = $1", generation.ID)
			}
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	URLVars     map[string]string
	QueryParams map[string]string
	Headers     map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyReader *bytes.Reader

	if req.Body != nil {
		var bodyBytes []byte
		var err error

		switch v := req.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(v)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to marshal request body: %v", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else {
		bodyReader = bytes.NewReader([]byte{})
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

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
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
	return w, httpReq, nil
}

// assertJSONResponse validates JSON response structure and content
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	// Validate expected fields
	if expectedFields != nil {
		for key, expectedValue := range expectedFields {
			actualValue, exists := response[key]
			if !exists {
				t.Errorf("Expected field '%s' not found in response", key)
				continue
			}

			if expectedValue != nil && actualValue != expectedValue {
				t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}

	return response
}

// assertJSONArray validates that response contains an array and returns it
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}
}

// mockHTTPServer creates a mock HTTP server for testing external service calls
type MockHTTPServer struct {
	Server   *httptest.Server
	Requests []*http.Request
	Responses []MockResponse
}

type MockResponse struct {
	StatusCode int
	Body       interface{}
	Headers    map[string]string
}

// newMockHTTPServer creates a new mock HTTP server
func newMockHTTPServer(responses []MockResponse) *MockHTTPServer {
	mock := &MockHTTPServer{
		Requests:  []*http.Request{},
		Responses: responses,
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Store request for inspection
		bodyBytes, _ := ioutil.ReadAll(r.Body)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		mock.Requests = append(mock.Requests, r)

		// Return response
		if len(mock.Responses) > 0 {
			resp := mock.Responses[0]
			if len(mock.Responses) > 1 {
				mock.Responses = mock.Responses[1:]
			}

			// Set headers
			for key, value := range resp.Headers {
				w.Header().Set(key, value)
			}
			w.Header().Set("Content-Type", "application/json")

			// Set status code
			w.WriteHeader(resp.StatusCode)

			// Write body
			if resp.Body != nil {
				json.NewEncoder(w).Encode(resp.Body)
			}
		} else {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}
	}))

	return mock
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// GenerateGenerationRequest creates a test generation request
func (g *TestDataGenerator) GenerateGenerationRequest(campaignID string) GenerationRequest {
	return GenerationRequest{
		CampaignID: campaignID,
		Prompt:     "Test image generation prompt",
		Style:      "photographic",
		Dimensions: "1024x1024",
		Metadata:   map[string]interface{}{"test": "data"},
	}
}

// GenerateVoiceBriefRequest creates a test voice brief request
func (g *TestDataGenerator) GenerateVoiceBriefRequest() VoiceBriefRequest {
	return VoiceBriefRequest{
		AudioData: "base64-encoded-audio-data",
		Format:    "wav",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}
