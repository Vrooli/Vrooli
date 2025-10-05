// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := result["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", result["status"])
		}

		if service, ok := result["service"].(string); !ok || service != "news-aggregator-bias-analysis" {
			t.Errorf("Expected service 'news-aggregator-bias-analysis', got %v", result["service"])
		}
	})
}

// TestGetArticlesHandler tests the get articles endpoint
func TestGetArticlesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_NoArticles", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/articles",
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var articles []Article
		if err := json.Unmarshal(w.Body.Bytes(), &articles); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(articles) != 0 {
			t.Errorf("Expected 0 articles, got %d", len(articles))
		}
	})

	t.Run("Success_WithArticles", func(t *testing.T) {
		// Create test articles
		testArticle := createTestArticle(t, env, "Test Article 1", "Test Source")
		defer testArticle.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/articles",
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var articles []Article
		if err := json.Unmarshal(w.Body.Bytes(), &articles); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(articles))
		}

		if articles[0].Title != "Test Article 1" {
			t.Errorf("Expected title 'Test Article 1', got %s", articles[0].Title)
		}
	})

	t.Run("Success_WithCategoryFilter", func(t *testing.T) {
		// Insert article with category
		query := `
			INSERT INTO articles (id, title, url, source, category, published_at, summary, fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		articleID := fmt.Sprintf("test-cat-%d", time.Now().UnixNano())
		env.DB.Exec(query, articleID, "Category Test", "https://example.com/cat",
			"Test Source", "politics", time.Now(), "Summary", time.Now())
		defer env.DB.Exec("DELETE FROM articles WHERE id = $1", articleID)

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/articles",
			QueryParams: map[string]string{"category": "politics"},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var articles []Article
		if err := json.Unmarshal(w.Body.Bytes(), &articles); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(articles))
		}
	})

	t.Run("Success_WithSourceFilter", func(t *testing.T) {
		testArticle := createTestArticle(t, env, "Source Test", "BBC News")
		defer testArticle.Cleanup()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/articles",
			QueryParams: map[string]string{"source": "BBC News"},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var articles []Article
		if err := json.Unmarshal(w.Body.Bytes(), &articles); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(articles))
		}
	})

	t.Run("Success_WithLimit", func(t *testing.T) {
		// Create multiple test articles
		articles := GenerateTestArticles(t, env, 5, "Test Source")
		defer CleanupTestArticles(articles)

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/articles",
			QueryParams: map[string]string{"limit": "3"},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var resultArticles []Article
		if err := json.Unmarshal(w.Body.Bytes(), &resultArticles); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(resultArticles) > 3 {
			t.Errorf("Expected at most 3 articles, got %d", len(resultArticles))
		}
	})
}

// TestGetArticleHandler tests the get single article endpoint
func TestGetArticleHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testArticle := createTestArticle(t, env, "Single Article Test", "Test Source")
		defer testArticle.Cleanup()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/articles/" + testArticle.Article.ID,
			URLVars: map[string]string{"id": testArticle.Article.ID},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var article Article
		if err := json.Unmarshal(w.Body.Bytes(), &article); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if article.ID != testArticle.Article.ID {
			t.Errorf("Expected article ID %s, got %s", testArticle.Article.ID, article.ID)
		}

		if article.Title != "Single Article Test" {
			t.Errorf("Expected title 'Single Article Test', got %s", article.Title)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("GetArticle", env)

		patterns := NewTestScenarioBuilder().
			AddNonExistentArticle("/articles/{id}").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestGetFeedsHandler tests the get feeds endpoint
func TestGetFeedsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_NoFeeds", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/feeds",
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var feeds []Feed
		if err := json.Unmarshal(w.Body.Bytes(), &feeds); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// May have default feeds from schema
		if feeds == nil {
			t.Error("Expected feeds array, got nil")
		}
	})

	t.Run("Success_WithFeeds", func(t *testing.T) {
		testFeed := createTestFeed(t, env, "Test Feed", "https://example.com/feed.rss")
		defer testFeed.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/feeds",
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var feeds []Feed
		if err := json.Unmarshal(w.Body.Bytes(), &feeds); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		found := false
		for _, feed := range feeds {
			if feed.Name == "Test Feed" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Test feed not found in response")
		}
	})
}

// TestAddFeedHandler tests the add feed endpoint
func TestAddFeedHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		feedData := map[string]interface{}{
			"name":        "New Test Feed",
			"url":         fmt.Sprintf("https://example.com/feed-%d.rss", time.Now().UnixNano()),
			"category":    "technology",
			"bias_rating": "center",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/feeds",
			Body:   feedData,
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var feed Feed
		if err := json.Unmarshal(w.Body.Bytes(), &feed); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		defer env.DB.Exec("DELETE FROM feeds WHERE id = $1", feed.ID)

		if feed.Name != "New Test Feed" {
			t.Errorf("Expected name 'New Test Feed', got %s", feed.Name)
		}

		if !feed.Active {
			t.Error("Expected feed to be active")
		}

		if feed.ID == 0 {
			t.Error("Expected feed ID to be set")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("AddFeed", env)

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("/feeds", "POST").
			AddMissingRequiredFields("/feeds", "POST").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestUpdateFeedHandler tests the update feed endpoint
func TestUpdateFeedHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testFeed := createTestFeed(t, env, "Original Name", "https://example.com/original.rss")
		defer testFeed.Cleanup()

		updateData := map[string]interface{}{
			"name":        "Updated Name",
			"url":         testFeed.Feed.URL,
			"category":    "updated-category",
			"bias_rating": "left",
			"active":      false,
		}

		req := HTTPTestRequest{
			Method:  "PUT",
			Path:    "/feeds/" + strconv.Itoa(testFeed.Feed.ID),
			URLVars: map[string]string{"id": strconv.Itoa(testFeed.Feed.ID)},
			Body:    updateData,
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var feed Feed
		if err := json.Unmarshal(w.Body.Bytes(), &feed); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if feed.Name != "Updated Name" {
			t.Errorf("Expected name 'Updated Name', got %s", feed.Name)
		}

		if feed.Active {
			t.Error("Expected feed to be inactive")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("UpdateFeed", env)

		patterns := NewTestScenarioBuilder().
			AddInvalidID("/feeds/{id}", "PUT").
			AddNonExistentFeed("/feeds/{id}", "PUT").
			AddInvalidJSON("/feeds/{id}", "PUT").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestDeleteFeedHandler tests the delete feed endpoint
func TestDeleteFeedHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testFeed := createTestFeed(t, env, "To Delete", "https://example.com/delete.rss")

		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/feeds/" + strconv.Itoa(testFeed.Feed.ID),
			URLVars: map[string]string{"id": strconv.Itoa(testFeed.Feed.ID)},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		// Verify deletion
		var count int
		env.DB.QueryRow("SELECT COUNT(*) FROM feeds WHERE id = $1", testFeed.Feed.ID).Scan(&count)
		if count != 0 {
			t.Error("Feed was not deleted")
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("DeleteFeed", env)

		patterns := NewTestScenarioBuilder().
			AddNonExistentFeed("/feeds/{id}", "DELETE").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestRefreshFeedsHandler tests the refresh feeds endpoint
func TestRefreshFeedsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/refresh",
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := result["status"].(string); !ok || status != "refresh_triggered" {
			t.Errorf("Expected status 'refresh_triggered', got %v", result["status"])
		}
	})
}

// TestAnalyzeBiasHandler tests the analyze bias endpoint
func TestAnalyzeBiasHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Skip if OLLAMA_URL not configured
		if processor == nil {
			t.Skip("Skipping bias analysis test - processor not configured")
		}

		testArticle := createTestArticle(t, env, "Bias Analysis Test", "Test Source")
		defer testArticle.Cleanup()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/analyze/" + testArticle.Article.ID,
			URLVars: map[string]string{"id": testArticle.Article.ID},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := result["status"].(string); !ok || status != "analysis_completed" {
			t.Errorf("Expected status 'analysis_completed', got %v", result["status"])
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("AnalyzeBias", env)

		patterns := NewTestScenarioBuilder().
			AddNonExistentArticle("/analyze/{id}").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestGetPerspectivesHandler tests the get perspectives endpoint
func TestGetPerspectivesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test articles with topic in title
		testArticle := createTestArticle(t, env, "Climate Change Report", "Test Source")
		defer testArticle.Cleanup()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/perspectives/climate",
			URLVars: map[string]string{"topic": "climate"},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var perspectives []map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &perspectives); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(perspectives) != 1 {
			t.Errorf("Expected 1 perspective, got %d", len(perspectives))
		}
	})
}

// TestAggregatePerspectivesHandler tests the aggregate perspectives endpoint
func TestAggregatePerspectivesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create test articles with different bias scores
		query := `
			INSERT INTO articles (id, title, url, source, published_at, summary, bias_score, fetched_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		articles := []struct {
			id    string
			title string
			score float64
		}{
			{fmt.Sprintf("left-%d", time.Now().UnixNano()), "Immigration Left View", -50.0},
			{fmt.Sprintf("center-%d", time.Now().UnixNano()), "Immigration Center View", 0.0},
			{fmt.Sprintf("right-%d", time.Now().UnixNano()), "Immigration Right View", 50.0},
		}

		for _, a := range articles {
			env.DB.Exec(query, a.id, a.title, "https://example.com/"+a.id,
				"Test Source", time.Now(), "Summary", a.score, time.Now())
			defer env.DB.Exec("DELETE FROM articles WHERE id = $1", a.id)
		}

		requestData := map[string]interface{}{
			"topic":      "immigration",
			"time_range": "24 hours",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/perspectives/aggregate",
			Body:   requestData,
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, w, http.StatusOK)

		if topic, ok := result["topic"].(string); !ok || topic != "immigration" {
			t.Errorf("Expected topic 'immigration', got %v", result["topic"])
		}

		if perspectives, ok := result["perspectives"].(map[string]interface{}); !ok {
			t.Error("Expected perspectives object")
		} else {
			for _, group := range []string{"left", "center", "right"} {
				if _, ok := perspectives[group]; !ok {
					t.Errorf("Missing perspective group: %s", group)
				}
			}
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("AggregatePerspectives", env)

		patterns := NewTestScenarioBuilder().
			AddEmptyTopic("/perspectives/aggregate").
			AddInvalidJSON("/perspectives/aggregate", "POST").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyDatabase", func(t *testing.T) {
		endpoints := []string{"/articles", "/feeds"}
		for _, endpoint := range endpoints {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   endpoint,
			}

			w, err := makeHTTPRequest(env, req)
			if err != nil {
				t.Fatalf("Failed to make request to %s: %v", endpoint, err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Endpoint %s failed with status %d", endpoint, w.Code)
			}
		}
	})

	t.Run("LargeLimit", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/articles",
			QueryParams: map[string]string{"limit": "10000"},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInTopic", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/perspectives/" + "test%20topic%20%26%20more",
			URLVars: map[string]string{"topic": "test topic & more"},
		}

		w, err := makeHTTPRequest(env, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
