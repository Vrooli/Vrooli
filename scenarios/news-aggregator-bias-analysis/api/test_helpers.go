// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for tests with minimal output
func setupTestLogger() func() {
	originalOutput := log.Writer()
	// Disable logging during tests to reduce noise
	log.SetOutput(ioutil.Discard)
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	Router     *mux.Router
	Cleanup    func()
	TempDir    string
	TestServer *httptest.Server
}

// setupTestDatabase creates an isolated test database
func setupTestDatabase(t *testing.T) *sql.DB {
	// Use in-memory or test database
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	if postgresURL == "" {
		// Default test database
		postgresURL = "postgres://test:test@localhost:5432/news_test?sslmode=disable"
	}

	testDB, err := sql.Open("postgres", postgresURL)
	if err != nil {
		t.Skipf("Skipping database test: %v", err)
		return nil
	}

	// Verify connection
	if err := testDB.Ping(); err != nil {
		t.Skipf("Skipping database test - cannot connect: %v", err)
		return nil
	}

	// Clean up test data
	cleanupTestData(testDB)

	return testDB
}

// cleanupTestData removes all test data from database
func cleanupTestData(testDB *sql.DB) {
	tables := []string{"article_interactions", "fact_checks", "fact_check_summaries", "articles", "feeds", "user_preferences", "topics"}
	for _, table := range tables {
		testDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}
}

// setupTestEnvironment creates a complete test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	cleanup := setupTestLogger()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}

	// Set global db for handlers
	db = testDB

	router := mux.NewRouter()

	// Register all handlers
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/articles", getArticlesHandler).Methods("GET")
	router.HandleFunc("/articles/{id}", getArticleHandler).Methods("GET")
	router.HandleFunc("/feeds", getFeedsHandler).Methods("GET")
	router.HandleFunc("/feeds", addFeedHandler).Methods("POST")
	router.HandleFunc("/feeds/{id}", updateFeedHandler).Methods("PUT")
	router.HandleFunc("/feeds/{id}", deleteFeedHandler).Methods("DELETE")
	router.HandleFunc("/refresh", refreshFeedsHandler).Methods("POST")
	router.HandleFunc("/analyze/{id}", analyzeBiasHandler).Methods("POST")
	router.HandleFunc("/perspectives/{topic}", getPerspectivesHandler).Methods("GET")
	router.HandleFunc("/perspectives/aggregate", aggregatePerspectivesHandler).Methods("POST")

	testServer := httptest.NewServer(router)

	return &TestEnvironment{
		DB:         testDB,
		Router:     router,
		TestServer: testServer,
		Cleanup: func() {
			testServer.Close()
			cleanupTestData(testDB)
			testDB.Close()
			cleanup()
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	QueryParams map[string]string
	URLVars     map[string]string
}

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonBytes)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for k, v := range req.QueryParams {
			q.Add(k, v)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	// Set URL vars for mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Set content type for JSON bodies
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return result
}

// assertErrorResponse validates error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMsg string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	if expectedMsg != "" && !bytes.Contains(w.Body.Bytes(), []byte(expectedMsg)) {
		t.Errorf("Expected error message to contain %q, got: %s", expectedMsg, w.Body.String())
	}
}

// TestArticle creates a test article for use in tests
type TestArticle struct {
	Article *Article
	Cleanup func()
}

// createTestArticle inserts a test article into the database
func createTestArticle(t *testing.T, env *TestEnvironment, title, source string) *TestArticle {
	article := &Article{
		ID:               fmt.Sprintf("test-%d", time.Now().UnixNano()),
		Title:            title,
		URL:              fmt.Sprintf("https://example.com/article/%d", time.Now().UnixNano()),
		Source:           source,
		PublishedAt:      time.Now(),
		Summary:          "Test article summary",
		BiasScore:        0.0,
		BiasAnalysis:     "Test bias analysis",
		PerspectiveCount: 1,
		FetchedAt:        time.Now(),
	}

	query := `
		INSERT INTO articles (id, title, url, source, published_at, summary,
		                      bias_score, bias_analysis, perspective_count, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := env.DB.Exec(query, article.ID, article.Title, article.URL, article.Source,
		article.PublishedAt, article.Summary, article.BiasScore, article.BiasAnalysis,
		article.PerspectiveCount, article.FetchedAt)

	if err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}

	return &TestArticle{
		Article: article,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM articles WHERE id = $1", article.ID)
		},
	}
}

// TestFeed creates a test feed for use in tests
type TestFeed struct {
	Feed    *Feed
	Cleanup func()
}

// createTestFeed inserts a test feed into the database
func createTestFeed(t *testing.T, env *TestEnvironment, name, url string) *TestFeed {
	feed := &Feed{
		Name:       name,
		URL:        url,
		Category:   "test",
		BiasRating: "center",
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO feeds (name, url, category, bias_rating, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	err := env.DB.QueryRow(query, feed.Name, feed.URL, feed.Category,
		feed.BiasRating, feed.Active, feed.CreatedAt, feed.UpdatedAt).Scan(&feed.ID)

	if err != nil {
		t.Fatalf("Failed to create test feed: %v", err)
	}

	return &TestFeed{
		Feed: feed,
		Cleanup: func() {
			env.DB.Exec("DELETE FROM feeds WHERE id = $1", feed.ID)
		},
	}
}

// assertArticleFields validates article fields
func assertArticleFields(t *testing.T, article map[string]interface{}, expectedTitle string) {
	if title, ok := article["title"].(string); !ok || title != expectedTitle {
		t.Errorf("Expected title %q, got %v", expectedTitle, article["title"])
	}

	requiredFields := []string{"id", "url", "source", "published_at", "summary"}
	for _, field := range requiredFields {
		if _, ok := article[field]; !ok {
			t.Errorf("Article missing required field: %s", field)
		}
	}
}

// assertFeedFields validates feed fields
func assertFeedFields(t *testing.T, feed map[string]interface{}, expectedName string) {
	if name, ok := feed["name"].(string); !ok || name != expectedName {
		t.Errorf("Expected name %q, got %v", expectedName, feed["name"])
	}

	requiredFields := []string{"id", "url", "category", "bias_rating", "active"}
	for _, field := range requiredFields {
		if _, ok := feed[field]; !ok {
			t.Errorf("Feed missing required field: %s", field)
		}
	}
}
