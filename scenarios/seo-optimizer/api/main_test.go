package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Disable lifecycle check for tests
	setupTestSEOProcessor()
	m.Run()
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	setupTestSEOProcessor()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertHealthResponse(t, w)
	})

	t.Run("CORS_Preflight", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, corsMiddleware(healthHandler))
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusOK, w.Code)
		}

		// Verify CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Missing or incorrect CORS allow-origin header")
		}
	})
}

// TestSEOAuditHandler tests the SEO audit endpoint
func TestSEOAuditHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	setupTestSEOProcessor()

	t.Run("Success", func(t *testing.T) {
		// Create test server to audit
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Test Page for SEO</title>
					<meta name="description" content="This is a test page for SEO audit testing">
					<meta name="viewport" content="width=device-width">
				</head>
				<body>
					<h1>Main Heading</h1>
					<p>This is test content with multiple words to test word count.</p>
					<img src="test.jpg" alt="Test image">
					<a href="/internal">Internal link</a>
					<a href="https://example.com">External link</a>
				</body>
				</html>
			`))
		}))
		defer testServer.Close()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/seo-audit",
			Body: SEOAuditRequest{
				URL:   testServer.URL,
				Depth: 3,
			},
		}

		w, err := makeHTTPRequest(req, seoAuditHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response SEOAuditResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Validate response structure
		if response.URL != testServer.URL {
			t.Errorf("Expected URL %s, got %s", testServer.URL, response.URL)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		if response.Score <= 0 {
			t.Error("Expected positive SEO score")
		}

		// Validate technical audit
		if response.TechnicalAudit.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", response.TechnicalAudit.StatusCode)
		}

		if response.TechnicalAudit.MetaTags.Title == "" {
			t.Error("Expected title to be extracted")
		}

		// Validate content audit
		if response.ContentAudit.WordCount == 0 {
			t.Error("Expected non-zero word count")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddMethodNotAllowed("/api/seo-audit", "GET").
			AddMethodNotAllowed("/api/seo-audit", "PUT").
			AddMethodNotAllowed("/api/seo-audit", "DELETE").
			AddInvalidJSON("POST", "/api/seo-audit").
			AddEmptyField("POST", "/api/seo-audit", "URL", SEOAuditRequest{URL: "", Depth: 3}).
			Build()

		RunErrorPatternTests(t, patterns, seoAuditHandler)
	})

	t.Run("InvalidURL", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/seo-audit",
			Body: SEOAuditRequest{
				URL:   "http://invalid-domain-that-does-not-exist-12345.com",
				Depth: 3,
			},
		}

		w, err := makeHTTPRequest(req, seoAuditHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should still return success with error status
		var response SEOAuditResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Status != "error" {
			t.Errorf("Expected status 'error' for invalid URL, got '%s'", response.Status)
		}

		if len(response.Issues) == 0 {
			t.Error("Expected issues to be reported for invalid URL")
		}
	})

	t.Run("DefaultDepth", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><head><title>Test</title></head><body>Test</body></html>"))
		}))
		defer testServer.Close()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/seo-audit",
			Body: SEOAuditRequest{
				URL: testServer.URL,
				// Depth not specified, should default to 3
			},
		}

		w, err := makeHTTPRequest(req, seoAuditHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response SEOAuditResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Status != "success" {
			t.Errorf("Expected success with default depth, got '%s'", response.Status)
		}
	})
}

// TestContentOptimizeHandler tests the content optimization endpoint
func TestContentOptimizeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	setupTestSEOProcessor()

	t.Run("Success", func(t *testing.T) {
		content := `
			This is a comprehensive guide about SEO optimization.
			SEO is critical for online visibility.
			Effective SEO strategies can improve your search rankings.
			Remember to focus on quality content and user experience.
			SEO best practices evolve over time.
		`

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/content-optimize",
			Body: ContentOptimizeRequest{
				Content:        content,
				TargetKeywords: "SEO, optimization, search rankings",
				ContentType:    "blog",
			},
		}

		w, err := makeHTTPRequest(req, contentOptimizeHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response ContentOptimizationResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Validate content analysis
		if response.ContentAnalysis.WordCount == 0 {
			t.Error("Expected non-zero word count")
		}

		if response.ContentAnalysis.SentenceCount == 0 {
			t.Error("Expected non-zero sentence count")
		}

		// Validate keyword analysis
		if len(response.KeywordAnalysis) == 0 {
			t.Error("Expected keyword analysis for target keywords")
		}

		// Validate score
		if response.Score < 0 || response.Score > 100 {
			t.Errorf("Expected score between 0-100, got %d", response.Score)
		}

		// Validate recommendations exist
		if len(response.Recommendations) == 0 {
			t.Error("Expected recommendations to be generated")
		}
	})

	t.Run("DefaultContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/content-optimize",
			Body: ContentOptimizeRequest{
				Content:        "Short content for testing.",
				TargetKeywords: "test",
				// ContentType not specified
			},
		}

		w, err := makeHTTPRequest(req, contentOptimizeHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response ContentOptimizationResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.ContentType != "general" {
			t.Errorf("Expected default content type 'general', got '%s'", response.ContentType)
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddMethodNotAllowed("/api/content-optimize", "GET").
			AddMethodNotAllowed("/api/content-optimize", "PUT").
			AddInvalidJSON("POST", "/api/content-optimize").
			AddEmptyField("POST", "/api/content-optimize", "Content", ContentOptimizeRequest{
				Content:        "",
				TargetKeywords: "test",
			}).
			Build()

		RunErrorPatternTests(t, patterns, contentOptimizeHandler)
	})

	t.Run("EmptyKeywords", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/content-optimize",
			Body: ContentOptimizeRequest{
				Content:        "This is test content.",
				TargetKeywords: "",
			},
		}

		w, err := makeHTTPRequest(req, contentOptimizeHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response ContentOptimizationResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Should still work with empty keywords
		if response.ContentAnalysis.WordCount == 0 {
			t.Error("Expected content to be analyzed even without keywords")
		}
	})
}

// TestKeywordResearchHandler tests the keyword research endpoint
func TestKeywordResearchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	setupTestSEOProcessor()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/keyword-research",
			Body: KeywordResearchRequest{
				SeedKeyword:    "digital marketing",
				TargetLocation: "United States",
				Language:       "en",
			},
		}

		w, err := makeHTTPRequest(req, keywordResearchHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response KeywordResearchResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Validate response
		if response.SeedKeyword != "digital marketing" {
			t.Errorf("Expected seed keyword 'digital marketing', got '%s'", response.SeedKeyword)
		}

		if response.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", response.Status)
		}

		if len(response.Keywords) == 0 {
			t.Error("Expected keyword suggestions")
		}

		if len(response.RelatedTerms) == 0 {
			t.Error("Expected related terms")
		}

		if len(response.LongTail) == 0 {
			t.Error("Expected long-tail keywords")
		}

		if len(response.Questions) == 0 {
			t.Error("Expected question keywords")
		}

		// Validate keyword suggestion structure
		for _, kw := range response.Keywords {
			if kw.Keyword == "" {
				t.Error("Keyword suggestion should have a keyword")
			}
			if kw.Volume == "" {
				t.Error("Keyword suggestion should have volume")
			}
			if kw.Competition == "" {
				t.Error("Keyword suggestion should have competition")
			}
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/keyword-research",
			Body: KeywordResearchRequest{
				SeedKeyword: "test",
				// TargetLocation and Language not specified
			},
		}

		w, err := makeHTTPRequest(req, keywordResearchHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response KeywordResearchResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if response.Status != "success" {
			t.Error("Expected success with default values")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddMethodNotAllowed("/api/keyword-research", "GET").
			AddMethodNotAllowed("/api/keyword-research", "DELETE").
			AddInvalidJSON("POST", "/api/keyword-research").
			AddEmptyField("POST", "/api/keyword-research", "SeedKeyword", KeywordResearchRequest{
				SeedKeyword: "",
			}).
			Build()

		RunErrorPatternTests(t, patterns, keywordResearchHandler)
	})
}

// TestCompetitorAnalysisHandler tests the competitor analysis endpoint
func TestCompetitorAnalysisHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	setupTestSEOProcessor()

	t.Run("Success", func(t *testing.T) {
		// Create two test servers
		yourServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`
				<html>
				<head><title>Your Site</title></head>
				<body><h1>Content</h1><p>Your content here with multiple words.</p></body>
				</html>
			`))
		}))
		defer yourServer.Close()

		competitorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`
				<html>
				<head><title>Competitor Site</title></head>
				<body><h1>Better Content</h1><p>Competitor content with more comprehensive information.</p></body>
				</html>
			`))
		}))
		defer competitorServer.Close()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitor-analysis",
			Body: CompetitorAnalysisRequest{
				YourURL:       yourServer.URL,
				CompetitorURL: competitorServer.URL,
				AnalysisType:  "comprehensive",
			},
		}

		w, err := makeHTTPRequest(req, competitorAnalysisHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response CompetitorAnalysisResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Validate response
		if response.YourURL != yourServer.URL {
			t.Errorf("Expected your URL %s, got %s", yourServer.URL, response.YourURL)
		}

		if response.CompetitorURL != competitorServer.URL {
			t.Errorf("Expected competitor URL %s, got %s", competitorServer.URL, response.CompetitorURL)
		}

		// Validate comparison structure
		if response.Comparison.SEOScore.Winner == "" {
			t.Error("Expected SEO score comparison to have a winner")
		}

		// Validate opportunities and threats
		if len(response.Opportunities) == 0 {
			t.Error("Expected opportunities to be identified")
		}

		if len(response.Threats) == 0 {
			t.Error("Expected threats to be identified")
		}

		if len(response.Recommendations) == 0 {
			t.Error("Expected recommendations to be generated")
		}

		// Validate overall score
		if response.OverallScore.Overall < 0 || response.OverallScore.Overall > 100 {
			t.Errorf("Expected overall score between 0-100, got %d", response.OverallScore.Overall)
		}
	})

	t.Run("DefaultAnalysisType", func(t *testing.T) {
		yourServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><head><title>Test</title></head><body>Test</body></html>"))
		}))
		defer yourServer.Close()

		competitorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><head><title>Test</title></head><body>Test</body></html>"))
		}))
		defer competitorServer.Close()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitor-analysis",
			Body: CompetitorAnalysisRequest{
				YourURL:       yourServer.URL,
				CompetitorURL: competitorServer.URL,
				// AnalysisType not specified
			},
		}

		w, err := makeHTTPRequest(req, competitorAnalysisHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		var response CompetitorAnalysisResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Should default to comprehensive analysis
		if len(response.Opportunities) == 0 {
			t.Error("Expected comprehensive analysis with default type")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddMethodNotAllowed("/api/competitor-analysis", "GET").
			AddMethodNotAllowed("/api/competitor-analysis", "PUT").
			AddInvalidJSON("POST", "/api/competitor-analysis").
			AddEmptyField("POST", "/api/competitor-analysis", "YourURL", CompetitorAnalysisRequest{
				YourURL:       "",
				CompetitorURL: "http://example.com",
			}).
			AddEmptyField("POST", "/api/competitor-analysis", "CompetitorURL", CompetitorAnalysisRequest{
				YourURL:       "http://example.com",
				CompetitorURL: "",
			}).
			Build()

		RunErrorPatternTests(t, patterns, competitorAnalysisHandler)
	})

	t.Run("BothURLsMissing", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitor-analysis",
			Body: CompetitorAnalysisRequest{
				YourURL:       "",
				CompetitorURL: "",
			},
		}

		w, err := makeHTTPRequest(req, competitorAnalysisHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestCORSMiddleware tests CORS functionality
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testHandler := corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("OptionsRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/test",
		}

		w, err := makeHTTPRequest(req, testHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusOK, w.Code)
		}

		// Verify CORS headers
		headers := []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
		}

		for _, header := range headers {
			if w.Header().Get(header) == "" {
				t.Errorf("Missing CORS header: %s", header)
			}
		}
	})

	t.Run("PostRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
		}

		w, err := makeHTTPRequest(req, testHandler)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// CORS headers should still be present
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Missing CORS allow-origin header on POST request")
		}
	})
}

// TestGetEnv tests environment variable retrieval
func TestGetEnv(t *testing.T) {
	t.Run("WithValue", func(t *testing.T) {
		key := "TEST_ENV_VAR_UNIQUE"
		expectedValue := "test_value"
		t.Setenv(key, expectedValue)

		result := getEnv(key, "default")
		if result != expectedValue {
			t.Errorf("Expected %s, got %s", expectedValue, result)
		}
	})

	t.Run("WithDefault", func(t *testing.T) {
		key := "NON_EXISTENT_ENV_VAR_12345"
		defaultValue := "default_value"

		result := getEnv(key, defaultValue)
		if result != defaultValue {
			t.Errorf("Expected default %s, got %s", defaultValue, result)
		}
	})

	t.Run("EmptyValue", func(t *testing.T) {
		key := "EMPTY_ENV_VAR"
		defaultValue := "default"
		t.Setenv(key, "")

		result := getEnv(key, defaultValue)
		if result != defaultValue {
			t.Errorf("Expected default %s for empty env var, got %s", defaultValue, result)
		}
	})
}
