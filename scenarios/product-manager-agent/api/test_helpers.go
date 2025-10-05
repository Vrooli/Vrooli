package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Redirect log output to discard to reduce test noise
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(os.Stderr)
	}
}

// TestApp manages isolated test application instance
type TestApp struct {
	App     *App
	Cleanup func()
}

// setupTestApp creates a test application with mock dependencies
func setupTestApp(t *testing.T) *TestApp {
	t.Helper()

	// Create mock database connection
	db := setupMockDB(t)

	// Create mock Redis client
	redisClient := setupMockRedis(t)

	app := &App{
		DB:          db,
		RedisClient: redisClient,
		OllamaURL:   "http://localhost:11434",
		QdrantURL:   "http://localhost:6333",
	}

	cleanup := func() {
		if db != nil {
			db.Close()
		}
		if redisClient != nil {
			redisClient.Close()
		}
	}

	return &TestApp{
		App:     app,
		Cleanup: cleanup,
	}
}

// setupMockDB creates a mock database for testing
func setupMockDB(t *testing.T) *sql.DB {
	t.Helper()

	// Return nil - handlers should handle nil gracefully
	// In a real production test, you'd connect to a test PostgreSQL instance
	return nil
}

// setupMockRedis creates a mock Redis client for testing
func setupMockRedis(t *testing.T) *redis.Client {
	t.Helper()

	// Create a Redis client that won't actually connect
	// Tests will handle connection errors gracefully
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	return client
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(t *testing.T, app *App, req HTTPTestRequest) *httptest.ResponseRecorder {
	t.Helper()

	var body io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			body = bytes.NewBufferString(v)
		case []byte:
			body = bytes.NewBuffer(v)
		default:
			jsonData, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("Failed to marshal request body: %v", err)
			}
			body = bytes.NewBuffer(jsonData)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, body)

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()

	// Route the request to the appropriate handler
	switch {
	case req.Path == "/health":
		app.healthHandler(w, httpReq)
	case req.Path == "/api/features":
		app.featuresHandler(w, httpReq)
	case req.Path == "/api/features/prioritize":
		app.prioritizeHandler(w, httpReq)
	case req.Path == "/api/features/rice":
		app.riceScoreHandler(w, httpReq)
	case req.Path == "/api/roadmap":
		app.roadmapHandler(w, httpReq)
	case req.Path == "/api/roadmap/generate":
		app.generateRoadmapHandler(w, httpReq)
	case req.Path == "/api/sprint/plan":
		app.sprintPlanHandler(w, httpReq)
	case req.Path == "/api/sprint/current":
		app.currentSprintHandler(w, httpReq)
	case req.Path == "/api/market/analyze":
		app.marketAnalysisHandler(w, httpReq)
	case req.Path == "/api/competitor/analyze":
		app.competitorAnalysisHandler(w, httpReq)
	case req.Path == "/api/feedback/analyze":
		app.feedbackAnalysisHandler(w, httpReq)
	case req.Path == "/api/roi/calculate":
		app.roiCalculationHandler(w, httpReq)
	case req.Path == "/api/decision/analyze":
		app.decisionAnalysisHandler(w, httpReq)
	case req.Path == "/api/dashboard":
		app.dashboardHandler(w, httpReq)
	default:
		http.NotFound(w, httpReq)
	}

	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	if target != nil {
		if err := json.NewDecoder(w.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected error status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	// Ensure response has content
	if w.Body.Len() == 0 {
		t.Error("Expected error message in response body, got empty body")
	}
}

// createTestFeature creates a test feature with default values
func createTestFeature(name string) Feature {
	return Feature{
		ID:          fmt.Sprintf("test-%s-%d", name, time.Now().UnixNano()),
		Name:        name,
		Description: fmt.Sprintf("Test feature: %s", name),
		Reach:       1000,
		Impact:      4,
		Confidence:  0.8,
		Effort:      5,
		Priority:    "MEDIUM",
		Score:       0,
		Status:      "proposed",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// createTestFeatures creates multiple test features
func createTestFeatures(count int) []Feature {
	features := make([]Feature, count)
	for i := 0; i < count; i++ {
		features[i] = createTestFeature(fmt.Sprintf("feature-%d", i+1))
		// Vary the values to make tests more realistic
		features[i].Reach = 1000 * (i + 1)
		features[i].Impact = (i % 5) + 1
		features[i].Effort = (i % 10) + 1
	}
	return features
}

// createTestDecision creates a test decision with options
func createTestDecision(title string) Decision {
	return Decision{
		ID:          fmt.Sprintf("decision-%d", time.Now().UnixNano()),
		Title:       title,
		Description: fmt.Sprintf("Test decision: %s", title),
		Options: []DecisionOption{
			{
				Name:        "Option A",
				Description: "First option",
			},
			{
				Name:        "Option B",
				Description: "Second option",
			},
		},
		Status:    "pending",
		CreatedAt: time.Now(),
	}
}

// createTestFeedback creates test feedback items
func createTestFeedback(count int) []FeedbackItem {
	items := make([]FeedbackItem, count)
	feedbackTypes := []string{"bug", "feature_request", "improvement", "praise"}

	for i := 0; i < count; i++ {
		items[i] = FeedbackItem{
			ID:         fmt.Sprintf("feedback-%d", i+1),
			UserID:     fmt.Sprintf("user-%d", i+1),
			Content:    fmt.Sprintf("Test feedback content %d", i+1),
			Type:       feedbackTypes[i%len(feedbackTypes)],
			Sentiment:  "neutral",
			Priority:   i % 5,
			ReceivedAt: time.Now(),
		}
	}
	return items
}

// waitForCondition waits for a condition to become true
func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

// contextWithTimeout creates a context with timeout for tests
func contextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), timeout)
}
