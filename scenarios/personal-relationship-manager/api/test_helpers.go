
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger suppresses log output during tests
func setupTestLogger() func() {
	// Suppress logs during tests by redirecting to /dev/null
	original := os.Stdout
	os.Stdout = nil
	return func() {
		os.Stdout = original
	}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *TestDatabase {
	// Use environment variables for test database
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "test_personal_relationship_manager"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test database: %v", err)
		return nil
	}

	// Verify connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Initialize schema
	schemaSQL := `
-- Contacts table
CREATE TABLE IF NOT EXISTS contacts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    nickname VARCHAR(100),
    relationship_type VARCHAR(50),
    birthday DATE,
    anniversary DATE,
    email VARCHAR(255),
    phone VARCHAR(50),
    address TEXT,
    notes TEXT,
    interests TEXT[],
    tags TEXT[],
    favorite_color VARCHAR(50),
    clothing_size VARCHAR(20),
    photo_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Interactions table
CREATE TABLE IF NOT EXISTS interactions (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    interaction_type VARCHAR(50),
    interaction_date DATE NOT NULL,
    description TEXT,
    sentiment VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Gifts table
CREATE TABLE IF NOT EXISTS gifts (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    gift_name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10, 2),
    occasion VARCHAR(100),
    given_date DATE,
    status VARCHAR(50),
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    notes TEXT,
    purchase_link TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reminders table
CREATE TABLE IF NOT EXISTS reminders (
    id SERIAL PRIMARY KEY,
    contact_id INTEGER REFERENCES contacts(id) ON DELETE CASCADE,
    reminder_type VARCHAR(50) NOT NULL,
    reminder_date DATE NOT NULL,
    reminder_time TIME,
    message TEXT,
    sent BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`

	if _, err := testDB.Exec(schemaSQL); err != nil {
		testDB.Close()
		t.Skipf("Skipping test: failed to initialize schema: %v", err)
		return nil
	}

	// Clean up test data
	cleanup := func() {
		// Delete test data in reverse order of dependencies
		testDB.Exec("DELETE FROM reminders WHERE contact_id IN (SELECT id FROM contacts WHERE name LIKE 'Test%')")
		testDB.Exec("DELETE FROM gifts WHERE contact_id IN (SELECT id FROM contacts WHERE name LIKE 'Test%')")
		testDB.Exec("DELETE FROM interactions WHERE contact_id IN (SELECT id FROM contacts WHERE name LIKE 'Test%')")
		testDB.Exec("DELETE FROM contacts WHERE name LIKE 'Test%'")
		testDB.Close()
	}

	return &TestDatabase{
		DB:      testDB,
		Cleanup: cleanup,
	}
}

// TestContact provides a pre-configured contact for testing
type TestContact struct {
	Contact *Contact
	Cleanup func()
}

// setupTestContact creates a test contact with sample data
func setupTestContact(t *testing.T, db *sql.DB, name string) *TestContact {
	contact := &Contact{
		Name:             name,
		Nickname:         name + " Nick",
		RelationshipType: "friend",
		Birthday:         "1990-01-15",
		Email:            name + "@example.com",
		Phone:            "555-0100",
		Notes:            "Test contact for " + name,
		Interests:        []string{"reading", "hiking", "coffee"},
		Tags:             []string{"test", "friend"},
	}

	query := `INSERT INTO contacts (name, nickname, relationship_type, birthday, email, phone, notes, interests, tags)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	err := db.QueryRow(query, contact.Name, contact.Nickname, contact.RelationshipType,
		contact.Birthday, contact.Email, contact.Phone, contact.Notes,
		pq.Array(contact.Interests), pq.Array(contact.Tags)).Scan(&contact.ID)

	if err != nil {
		t.Fatalf("Failed to create test contact: %v", err)
	}

	return &TestContact{
		Contact: contact,
		Cleanup: func() {
			db.Exec("DELETE FROM reminders WHERE contact_id = $1", contact.ID)
			db.Exec("DELETE FROM gifts WHERE contact_id = $1", contact.ID)
			db.Exec("DELETE FROM interactions WHERE contact_id = $1", contact.ID)
			db.Exec("DELETE FROM contacts WHERE id = $1", contact.ID)
		},
	}
}

// setupTestInteraction creates a test interaction
func setupTestInteraction(t *testing.T, db *sql.DB, contactID int) *Interaction {
	interaction := &Interaction{
		ContactID:       contactID,
		InteractionType: "call",
		InteractionDate: "2024-01-15",
		Description:     "Test phone call",
		Sentiment:       "positive",
	}

	query := `INSERT INTO interactions (contact_id, interaction_type, interaction_date, description, sentiment)
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`

	err := db.QueryRow(query, interaction.ContactID, interaction.InteractionType,
		interaction.InteractionDate, interaction.Description, interaction.Sentiment).Scan(&interaction.ID)

	if err != nil {
		t.Fatalf("Failed to create test interaction: %v", err)
	}

	return interaction
}

// setupTestGift creates a test gift
func setupTestGift(t *testing.T, db *sql.DB, contactID int) *Gift {
	gift := &Gift{
		ContactID:   contactID,
		GiftName:    "Test Gift",
		Description: "A thoughtful test gift",
		Price:       50.00,
		Occasion:    "birthday",
		Status:      "idea",
		Notes:       "Test gift notes",
	}

	query := `INSERT INTO gifts (contact_id, gift_name, description, price, occasion, status, notes)
			  VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err := db.QueryRow(query, gift.ContactID, gift.GiftName, gift.Description,
		gift.Price, gift.Occasion, gift.Status, gift.Notes).Scan(&gift.ID)

	if err != nil {
		t.Fatalf("Failed to create test gift: %v", err)
	}

	return gift
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

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
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

			if expectedValue != nil && fmt.Sprint(actualValue) != fmt.Sprint(expectedValue) {
				t.Errorf("Expected field '%s' to be %v, got %v", key, expectedValue, actualValue)
			}
		}
	}

	return response
}

// assertJSONArray validates that response is an array
func assertJSONArray(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return nil
	}

	var response []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON array response: %v. Response: %s", err, w.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates error responses
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	// For this API, errors are plain text
	if w.Body.Len() == 0 {
		t.Error("Expected error message in response body")
	}
}

// TestDataGenerator provides utilities for generating test data
type TestDataGenerator struct{}

// CreateContactRequest creates a test contact creation request
func (g *TestDataGenerator) CreateContactRequest(name string) Contact {
	return Contact{
		Name:             name,
		Nickname:         name + " Nick",
		RelationshipType: "friend",
		Birthday:         "1990-05-15",
		Email:            name + "@example.com",
		Phone:            "555-0123",
		Notes:            "Generated test contact",
		Interests:        []string{"reading", "travel"},
		Tags:             []string{"test"},
	}
}

// CreateInteractionRequest creates a test interaction request
func (g *TestDataGenerator) CreateInteractionRequest(contactID int) Interaction {
	return Interaction{
		ContactID:       contactID,
		InteractionType: "call",
		InteractionDate: "2024-01-15",
		Description:     "Test interaction",
		Sentiment:       "positive",
	}
}

// CreateGiftRequest creates a test gift request
func (g *TestDataGenerator) CreateGiftRequest(contactID int) Gift {
	return Gift{
		ContactID:   contactID,
		GiftName:    "Test Gift",
		Description: "A test gift",
		Price:       50.00,
		Occasion:    "birthday",
		Status:      "idea",
		Notes:       "Test notes",
	}
}

// Global test data generator instance
var TestData = &TestDataGenerator{}

// MockRelationshipProcessor for testing without external dependencies
type MockRelationshipProcessor struct {
	db *sql.DB
}

// NewMockRelationshipProcessor creates a mock processor for testing
func NewMockRelationshipProcessor(db *sql.DB) *MockRelationshipProcessor {
	return &MockRelationshipProcessor{db: db}
}

// GetUpcomingBirthdays mock implementation
func (m *MockRelationshipProcessor) GetUpcomingBirthdays(ctx context.Context, daysAhead int) ([]BirthdayReminder, error) {
	return []BirthdayReminder{
		{
			ContactID: 1,
			Name:      "Test Contact",
			Birthday:  "2024-01-20",
			DaysUntil: 5,
			Message:   "Test birthday coming up",
			Urgency:   "medium",
		},
	}, nil
}

// EnrichContact mock implementation
func (m *MockRelationshipProcessor) EnrichContact(ctx context.Context, contactID int, name string) (*ContactEnrichment, error) {
	return &ContactEnrichment{
		ContactID:          contactID,
		Name:               name,
		SuggestedInterests: []string{"reading", "travel", "music"},
		PersonalityTraits:  []string{"friendly", "curious"},
		ConversationStarters: []string{
			"What's been interesting lately?",
			"Any new hobbies?",
		},
		EnrichmentNotes: "Mock enrichment data",
	}, nil
}

// SuggestGifts mock implementation
func (m *MockRelationshipProcessor) SuggestGifts(ctx context.Context, req GiftSuggestionRequest) (*GiftSuggestionResponse, error) {
	return &GiftSuggestionResponse{
		ContactID: req.ContactID,
		Occasion:  req.Occasion,
		Suggestions: []GiftSuggestion{
			{
				Name:           "Mock Gift 1",
				Description:    "A thoughtful mock gift",
				Price:          50,
				Store:          "Test Store",
				RelevanceScore: 0.9,
			},
		},
		GeneratedAt: "2024-01-15T10:00:00Z",
	}, nil
}

// AnalyzeRelationships mock implementation
func (m *MockRelationshipProcessor) AnalyzeRelationships(ctx context.Context, contactID int) (*RelationshipInsight, error) {
	return &RelationshipInsight{
		ContactID:            contactID,
		Name:                 "Test Contact",
		LastInteractionDays:  5,
		InteractionFrequency: "regular",
		OverallSentiment:     "positive",
		RecommendedActions:   []string{"Keep up the good work"},
		RelationshipScore:    75.5,
		TrendAnalysis:        "Relationship is stable",
	}, nil
}
