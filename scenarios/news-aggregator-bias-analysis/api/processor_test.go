// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestNewFeedProcessor tests the feed processor creation
func TestNewFeedProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

		if fp == nil {
			t.Fatal("Expected feed processor to be created")
		}

		if fp.db == nil {
			t.Error("Expected database to be set")
		}

		if fp.ollamaURL != "http://localhost:11434" {
			t.Errorf("Expected ollamaURL to be set to http://localhost:11434, got %s", fp.ollamaURL)
		}
	})

	t.Run("WithRedisClient", func(t *testing.T) {
		// Create a mock Redis client (nil is acceptable for testing)
		fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

		if fp == nil {
			t.Fatal("Expected feed processor to be created")
		}
	})
}

// TestFetchRSSFeed tests RSS feed fetching
func TestFetchRSSFeed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_ValidRSS", func(t *testing.T) {
		// Create a test RSS server
		rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Feed</title>
		<link>https://example.com</link>
		<description>Test RSS Feed</description>
		<item>
			<title>Test Article</title>
			<link>https://example.com/article1</link>
			<description>Test article description</description>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
			<guid>https://example.com/article1</guid>
		</item>
	</channel>
</rss>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(rssContent))
		}))
		defer server.Close()

		rss, err := fp.FetchRSSFeed(server.URL)
		if err != nil {
			t.Fatalf("Failed to fetch RSS: %v", err)
		}

		if rss == nil {
			t.Fatal("Expected RSS to be parsed")
		}

		if rss.Channel.Title != "Test Feed" {
			t.Errorf("Expected title 'Test Feed', got %s", rss.Channel.Title)
		}

		if len(rss.Channel.Items) != 1 {
			t.Errorf("Expected 1 item, got %d", len(rss.Channel.Items))
		}

		if rss.Channel.Items[0].Title != "Test Article" {
			t.Errorf("Expected item title 'Test Article', got %s", rss.Channel.Items[0].Title)
		}
	})

	t.Run("Error_InvalidURL", func(t *testing.T) {
		_, err := fp.FetchRSSFeed("http://invalid-url-that-does-not-exist.test")
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("Error_InvalidXML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("invalid xml content"))
		}))
		defer server.Close()

		_, err := fp.FetchRSSFeed(server.URL)
		if err == nil {
			t.Error("Expected error for invalid XML")
		}
	})

	t.Run("Error_HTTPError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}))
		defer server.Close()

		rss, err := fp.FetchRSSFeed(server.URL)
		// Should still return RSS even with error status
		if err != nil {
			t.Logf("Got expected error: %v", err)
		}

		if rss != nil {
			t.Log("RSS parsing attempted despite HTTP error")
		}
	})
}

// TestProcessFeed tests feed processing logic
func TestProcessFeed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_ProcessValidFeed", func(t *testing.T) {
		// Create a test RSS server
		rssContent := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Feed</title>
		<link>https://example.com</link>
		<description>Test RSS Feed</description>
		<item>
			<title>Process Test Article</title>
			<link>https://example.com/process-test</link>
			<description>Article for processing test</description>
			<pubDate>Mon, 02 Jan 2006 15:04:05 MST</pubDate>
		</item>
	</channel>
</rss>`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/rss+xml")
			w.Write([]byte(rssContent))
		}))
		defer server.Close()

		testFeed := Feed{
			ID:         1,
			Name:       "Test Feed",
			URL:        server.URL,
			Category:   "test",
			BiasRating: "center",
			Active:     true,
		}

		// Process the feed (this will attempt to call Ollama, which may fail)
		err := fp.ProcessFeed(testFeed)
		// We don't fail the test if Ollama is unavailable
		if err != nil {
			t.Logf("Feed processing encountered error (expected if Ollama unavailable): %v", err)
		}

		// Give time for goroutines to complete
		time.Sleep(100 * time.Millisecond)

		// Check if article was stored (it may not be if Ollama is unavailable)
		var count int
		env.DB.QueryRow("SELECT COUNT(*) FROM articles WHERE url = $1",
			"https://example.com/process-test").Scan(&count)

		if count > 0 {
			t.Log("Article successfully stored")
			// Clean up
			env.DB.Exec("DELETE FROM articles WHERE url = $1", "https://example.com/process-test")
		} else {
			t.Log("Article not stored (likely due to Ollama unavailability)")
		}
	})

	t.Run("Error_InvalidFeedURL", func(t *testing.T) {
		testFeed := Feed{
			ID:   2,
			Name: "Invalid Feed",
			URL:  "http://invalid-feed-url.test",
		}

		err := fp.ProcessFeed(testFeed)
		if err == nil {
			t.Error("Expected error for invalid feed URL")
		}
	})
}

// TestProcessAllFeeds tests processing all active feeds
func TestProcessAllFeeds(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_NoFeeds", func(t *testing.T) {
		err := fp.ProcessAllFeeds()
		if err != nil {
			t.Errorf("Expected no error with no feeds, got: %v", err)
		}
	})

	t.Run("Success_WithFeeds", func(t *testing.T) {
		// Create test feeds
		feeds := GenerateTestFeeds(t, env, 2)
		defer CleanupTestFeeds(feeds)

		// Mark feeds as inactive to prevent actual processing
		for _, feed := range feeds {
			env.DB.Exec("UPDATE feeds SET active = false WHERE id = $1", feed.Feed.ID)
		}

		err := fp.ProcessAllFeeds()
		// Should succeed even if no active feeds
		if err != nil {
			t.Logf("ProcessAllFeeds returned error: %v", err)
		}
	})
}

// TestArticleExists tests the articleExists helper
func TestArticleExists(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_ArticleExists", func(t *testing.T) {
		testArticle := createTestArticle(t, env, "Exists Test", "Test Source")
		defer testArticle.Cleanup()

		exists := fp.articleExists(testArticle.Article.URL)
		if !exists {
			t.Error("Expected article to exist")
		}
	})

	t.Run("Success_ArticleDoesNotExist", func(t *testing.T) {
		exists := fp.articleExists("https://example.com/nonexistent-article")
		if exists {
			t.Error("Expected article to not exist")
		}
	})
}

// TestStoreArticle tests article storage
func TestStoreArticle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_NewArticle", func(t *testing.T) {
		article := Article{
			ID:               "store-test-1",
			Title:            "Store Test Article",
			URL:              "https://example.com/store-test-1",
			Source:           "Test Source",
			PublishedAt:      time.Now(),
			Summary:          "Test summary",
			BiasScore:        0.0,
			BiasAnalysis:     "Test analysis",
			PerspectiveCount: 1,
			FetchedAt:        time.Now(),
		}

		err := fp.storeArticle(article)
		if err != nil {
			t.Fatalf("Failed to store article: %v", err)
		}

		defer env.DB.Exec("DELETE FROM articles WHERE id = $1", article.ID)

		// Verify storage
		var title string
		err = env.DB.QueryRow("SELECT title FROM articles WHERE id = $1", article.ID).Scan(&title)
		if err != nil {
			t.Fatalf("Failed to retrieve stored article: %v", err)
		}

		if title != "Store Test Article" {
			t.Errorf("Expected title 'Store Test Article', got %s", title)
		}
	})

	t.Run("Success_UpdateExisting", func(t *testing.T) {
		article := Article{
			ID:               "store-test-2",
			Title:            "Original Title",
			URL:              "https://example.com/store-test-2",
			Source:           "Test Source",
			PublishedAt:      time.Now(),
			Summary:          "Original summary",
			BiasScore:        0.0,
			BiasAnalysis:     "Original analysis",
			FetchedAt:        time.Now(),
		}

		// Store initial article
		err := fp.storeArticle(article)
		if err != nil {
			t.Fatalf("Failed to store initial article: %v", err)
		}

		defer env.DB.Exec("DELETE FROM articles WHERE id = $1", article.ID)

		// Update article
		article.Summary = "Updated summary"
		article.BiasScore = 50.0

		err = fp.storeArticle(article)
		if err != nil {
			t.Fatalf("Failed to update article: %v", err)
		}

		// Verify update
		var summary string
		var biasScore float64
		err = env.DB.QueryRow("SELECT summary, bias_score FROM articles WHERE id = $1",
			article.ID).Scan(&summary, &biasScore)
		if err != nil {
			t.Fatalf("Failed to retrieve updated article: %v", err)
		}

		if summary != "Updated summary" {
			t.Errorf("Expected summary 'Updated summary', got %s", summary)
		}

		if biasScore != 50.0 {
			t.Errorf("Expected bias_score 50.0, got %.2f", biasScore)
		}
	})
}

// TestGetActiveFeeds tests retrieving active feeds
func TestGetActiveFeeds(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_NoFeeds", func(t *testing.T) {
		feeds, err := fp.getActiveFeeds()
		if err != nil {
			t.Fatalf("Failed to get active feeds: %v", err)
		}

		if feeds == nil {
			t.Error("Expected feeds slice, got nil")
		}
	})

	t.Run("Success_WithActiveFeeds", func(t *testing.T) {
		// Create active feed
		activeFeed := createTestFeed(t, env, "Active Feed", "https://example.com/active.rss")
		defer activeFeed.Cleanup()

		// Create inactive feed
		inactiveFeed := createTestFeed(t, env, "Inactive Feed", "https://example.com/inactive.rss")
		defer inactiveFeed.Cleanup()

		env.DB.Exec("UPDATE feeds SET active = false WHERE id = $1", inactiveFeed.Feed.ID)

		feeds, err := fp.getActiveFeeds()
		if err != nil {
			t.Fatalf("Failed to get active feeds: %v", err)
		}

		foundActive := false
		foundInactive := false

		for _, feed := range feeds {
			if feed.Name == "Active Feed" {
				foundActive = true
			}
			if feed.Name == "Inactive Feed" {
				foundInactive = true
			}
		}

		if !foundActive {
			t.Error("Expected to find active feed")
		}

		if foundInactive {
			t.Error("Should not find inactive feed")
		}
	})
}

// TestFetchArticlesByTopic tests fetching articles by topic
func TestFetchArticlesByTopic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_NoArticles", func(t *testing.T) {
		articles, err := fp.fetchArticlesByTopic("nonexistent-topic")
		if err != nil {
			t.Fatalf("Failed to fetch articles: %v", err)
		}

		if len(articles) != 0 {
			t.Errorf("Expected 0 articles, got %d", len(articles))
		}
	})

	t.Run("Success_WithArticles", func(t *testing.T) {
		testArticle := createTestArticle(t, env, "Climate Change Article", "Test Source")
		defer testArticle.Cleanup()

		articles, err := fp.fetchArticlesByTopic("Climate")
		if err != nil {
			t.Fatalf("Failed to fetch articles: %v", err)
		}

		if len(articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(articles))
		}

		if articles[0].Title != "Climate Change Article" {
			t.Errorf("Expected title 'Climate Change Article', got %s", articles[0].Title)
		}
	})

	t.Run("Success_CaseInsensitive", func(t *testing.T) {
		testArticle := createTestArticle(t, env, "POLITICS Article", "Test Source")
		defer testArticle.Cleanup()

		articles, err := fp.fetchArticlesByTopic("politics")
		if err != nil {
			t.Fatalf("Failed to fetch articles: %v", err)
		}

		if len(articles) != 1 {
			t.Errorf("Expected 1 article, got %d", len(articles))
		}
	})
}

// TestCategorizeBias tests bias categorization
func TestCategorizeBias(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	tests := []struct {
		score    float64
		expected string
	}{
		{-100, "left"},
		{-50, "left"},
		{-34, "left"},
		{-33, "center"},
		{0, "center"},
		{33, "center"},
		{34, "right"},
		{50, "right"},
		{100, "right"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Score_%.0f", tt.score), func(t *testing.T) {
			category := fp.categorizeBias(tt.score)
			if category != tt.expected {
				t.Errorf("For score %.0f, expected category %s, got %s",
					tt.score, tt.expected, category)
			}
		})
	}
}

// TestSummarizePerspective tests perspective summarization
func TestSummarizePerspective(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	fp := NewFeedProcessor(env.DB, "http://localhost:11434", nil)

	t.Run("Success_NoArticles", func(t *testing.T) {
		summary := fp.summarizePerspective([]Article{})
		if summary != "No articles from this perspective" {
			t.Errorf("Expected 'No articles from this perspective', got %s", summary)
		}
	})

	t.Run("Success_WithArticles", func(t *testing.T) {
		articles := []Article{
			{Summary: "Summary 1"},
			{Summary: "Summary 2"},
			{Summary: "Summary 3"},
		}

		summary := fp.summarizePerspective(articles)
		if summary == "" {
			t.Error("Expected non-empty summary")
		}

		if !contains(summary, "Summary 1") {
			t.Error("Expected summary to contain 'Summary 1'")
		}
	})

	t.Run("Success_LimitToThree", func(t *testing.T) {
		articles := []Article{
			{Summary: "Summary 1"},
			{Summary: "Summary 2"},
			{Summary: "Summary 3"},
			{Summary: "Summary 4"},
			{Summary: "Summary 5"},
		}

		summary := fp.summarizePerspective(articles)

		// Should only include first 3
		if contains(summary, "Summary 4") || contains(summary, "Summary 5") {
			t.Error("Summary should be limited to 3 articles")
		}
	})
}

// TestHelperFunctions tests utility helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("GetString", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}

		if getString(m, "key1") != "value1" {
			t.Error("Expected 'value1'")
		}

		if getString(m, "key2") != "" {
			t.Error("Expected empty string for non-string value")
		}

		if getString(m, "nonexistent") != "" {
			t.Error("Expected empty string for nonexistent key")
		}
	})

	t.Run("GetBool", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": true,
			"key2": false,
			"key3": "not a bool",
		}

		if !getBool(m, "key1") {
			t.Error("Expected true")
		}

		if getBool(m, "key2") {
			t.Error("Expected false")
		}

		if getBool(m, "key3") {
			t.Error("Expected false for non-bool value")
		}

		if getBool(m, "nonexistent") {
			t.Error("Expected false for nonexistent key")
		}
	})

	t.Run("GetStringSlice", func(t *testing.T) {
		m := map[string]interface{}{
			"key1": []interface{}{"a", "b", "c"},
			"key2": []interface{}{"a", 123, "c"},
			"key3": "not a slice",
		}

		result := getStringSlice(m, "key1")
		if len(result) != 3 {
			t.Errorf("Expected 3 items, got %d", len(result))
		}

		result = getStringSlice(m, "key2")
		if len(result) != 2 {
			t.Errorf("Expected 2 items (non-strings filtered), got %d", len(result))
		}

		result = getStringSlice(m, "key3")
		if len(result) != 0 {
			t.Errorf("Expected 0 items for non-slice, got %d", len(result))
		}
	})
}

// Helper function for tests
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
