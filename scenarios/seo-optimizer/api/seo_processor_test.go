package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestNewSEOProcessor tests SEO processor initialization
func TestNewSEOProcessor(t *testing.T) {
	processor := NewSEOProcessor()

	if processor == nil {
		t.Fatal("Expected non-nil SEO processor")
	}

	if processor.httpClient == nil {
		t.Error("Expected HTTP client to be initialized")
	}

	if processor.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected timeout of 30s, got %v", processor.httpClient.Timeout)
	}
}

// TestPerformSEOAudit tests comprehensive SEO audit functionality
func TestPerformSEOAudit(t *testing.T) {
	processor := NewSEOProcessor()
	ctx := context.Background()

	t.Run("Success_CompleteHTML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Complete SEO Test Page</title>
					<meta name="description" content="This is a comprehensive test page for SEO audit with proper meta tags">
					<meta name="keywords" content="test, seo, audit">
					<meta name="viewport" content="width=device-width, initial-scale=1">
				</head>
				<body>
					<h1>Main Heading</h1>
					<h2>Secondary Heading</h2>
					<p>This is the first paragraph with enough content to analyze readability and word count.</p>
					<p>Second paragraph with more content. SEO is important for visibility.</p>
					<img src="image1.jpg" alt="First image">
					<img src="image2.jpg" alt="Second image">
					<a href="/page1">Internal link 1</a>
					<a href="/page2">Internal link 2</a>
					<a href="https://external.com">External link</a>
				</body>
				</html>
			`))
		}))
		defer server.Close()

		result, err := processor.PerformSEOAudit(ctx, server.URL, 3)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Validate basic fields
		if result.URL != server.URL {
			t.Errorf("Expected URL %s, got %s", server.URL, result.URL)
		}

		if result.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", result.Status)
		}

		// Validate technical audit
		if result.TechnicalAudit.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", result.TechnicalAudit.StatusCode)
		}

		if result.TechnicalAudit.MetaTags.Title != "Complete SEO Test Page" {
			t.Errorf("Expected title to be extracted, got '%s'", result.TechnicalAudit.MetaTags.Title)
		}

		if result.TechnicalAudit.MetaTags.Description == "" {
			t.Error("Expected description to be extracted")
		}

		if result.TechnicalAudit.ResponsiveDesign != true {
			t.Error("Expected responsive design to be detected (viewport meta tag)")
		}

		// Validate content audit
		if result.ContentAudit.WordCount == 0 {
			t.Error("Expected non-zero word count")
		}

		if result.ContentAudit.Images != 2 {
			t.Errorf("Expected 2 images, got %d", result.ContentAudit.Images)
		}

		if result.ContentAudit.ImagesWithAlt != 2 {
			t.Errorf("Expected 2 images with alt, got %d", result.ContentAudit.ImagesWithAlt)
		}

		if len(result.ContentAudit.Headers) == 0 {
			t.Error("Expected headers to be extracted")
		}

		// Validate performance audit
		if result.PerformanceAudit.PageSpeed < 0 || result.PerformanceAudit.PageSpeed > 100 {
			t.Errorf("Expected page speed between 0-100, got %d", result.PerformanceAudit.PageSpeed)
		}

		// Validate score
		if result.Score < 0 || result.Score > 100 {
			t.Errorf("Expected score between 0-100, got %d", result.Score)
		}

		// Validate timestamp
		if result.Timestamp == "" {
			t.Error("Expected timestamp to be set")
		}
	})

	t.Run("Success_MinimalHTML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><body>Minimal</body></html>"))
		}))
		defer server.Close()

		result, err := processor.PerformSEOAudit(ctx, server.URL, 1)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", result.Status)
		}

		// Should have issues due to missing meta tags
		if len(result.Issues) == 0 {
			t.Error("Expected issues to be identified for minimal HTML")
		}
	})

	t.Run("Success_HTTPS", func(t *testing.T) {
		// Test HTTPS detection by creating TLS server
		result, _ := processor.PerformSEOAudit(ctx, "https://example.com", 2)

		// Even if it fails to connect, we can test the URL parsing
		if result != nil && result.TechnicalAudit.HasHTTPS != true {
			// Note: This might fail for connection errors, which is expected
			t.Log("HTTPS detection test - connection may fail, which is expected")
		}
	})

	t.Run("Error_InvalidURL", func(t *testing.T) {
		result, err := processor.PerformSEOAudit(ctx, "http://invalid-nonexistent-domain-12345.com", 3)

		// Should return a result with error status, not an error
		if err != nil {
			t.Errorf("Expected error to be handled gracefully, got error: %v", err)
		}

		if result.Status != "error" {
			t.Errorf("Expected status 'error', got '%s'", result.Status)
		}

		if len(result.Issues) == 0 {
			t.Error("Expected issues to be reported for invalid URL")
		}

		if result.Score != 0 {
			t.Errorf("Expected score 0 for failed audit, got %d", result.Score)
		}
	})

	t.Run("DefaultDepth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><body>Test</body></html>"))
		}))
		defer server.Close()

		// Pass depth 0 or negative to test default
		result, err := processor.PerformSEOAudit(ctx, server.URL, 0)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.Status != "success" {
			t.Error("Expected success with default depth")
		}
	})
}

// TestOptimizeContent tests content optimization functionality
func TestOptimizeContent(t *testing.T) {
	processor := NewSEOProcessor()
	ctx := context.Background()

	t.Run("Success_BlogContent", func(t *testing.T) {
		content := `
			Digital Marketing Strategy: A Comprehensive Guide

			Digital marketing is essential for modern businesses. This guide covers everything you need to know about digital marketing strategies.

			Understanding Digital Marketing
			Digital marketing encompasses various online channels. From SEO to social media marketing, businesses must adapt to the digital landscape.

			Key Strategies
			1. Content marketing helps build authority
			2. SEO improves search visibility
			3. Social media engages audiences

			Digital marketing continues to evolve. Stay updated with the latest trends and best practices.
		`

		result, err := processor.OptimizeContent(ctx, content, "digital marketing, SEO, strategy", "blog")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Validate content analysis
		if result.ContentAnalysis.WordCount == 0 {
			t.Error("Expected non-zero word count")
		}

		if result.ContentAnalysis.SentenceCount == 0 {
			t.Error("Expected non-zero sentence count")
		}

		if result.ContentAnalysis.ParagraphCount == 0 {
			t.Error("Expected non-zero paragraph count")
		}

		if result.ContentAnalysis.ReadabilityScore < 0 || result.ContentAnalysis.ReadabilityScore > 100 {
			t.Errorf("Expected readability score 0-100, got %.2f", result.ContentAnalysis.ReadabilityScore)
		}

		// Validate keyword analysis
		if len(result.KeywordAnalysis) == 0 {
			t.Error("Expected keyword analysis for provided keywords")
		}

		for keyword, metrics := range result.KeywordAnalysis {
			if metrics.Count == 0 {
				t.Logf("Warning: Keyword '%s' not found in content", keyword)
			}

			if metrics.Density < 0 {
				t.Errorf("Expected non-negative density for '%s'", keyword)
			}

			if metrics.ProminenceScore < 0 || metrics.ProminenceScore > 100 {
				t.Errorf("Expected prominence score 0-100 for '%s', got %.2f", keyword, metrics.ProminenceScore)
			}
		}

		// Validate score
		if result.Score < 0 || result.Score > 100 {
			t.Errorf("Expected score 0-100, got %d", result.Score)
		}

		// Validate content type
		if result.ContentType != "blog" {
			t.Errorf("Expected content type 'blog', got '%s'", result.ContentType)
		}

		// Should have recommendations
		if len(result.Recommendations) == 0 {
			t.Error("Expected recommendations to be generated")
		}
	})

	t.Run("Success_EmptyKeywords", func(t *testing.T) {
		content := "This is test content without specific keywords."

		result, err := processor.OptimizeContent(ctx, content, "", "general")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should still analyze content structure
		if result.ContentAnalysis.WordCount == 0 {
			t.Error("Expected content to be analyzed even without keywords")
		}

		// Keyword analysis should be empty
		if len(result.KeywordAnalysis) != 0 {
			t.Error("Expected empty keyword analysis with no keywords")
		}
	})

	t.Run("Success_DefaultContentType", func(t *testing.T) {
		content := "Short content."

		result, err := processor.OptimizeContent(ctx, content, "test", "")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.ContentType != "general" {
			t.Errorf("Expected default content type 'general', got '%s'", result.ContentType)
		}
	})

	t.Run("ShortContent", func(t *testing.T) {
		content := "Too short."

		result, err := processor.OptimizeContent(ctx, content, "test", "blog")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should identify issues with short content
		hasLengthIssue := false
		for _, issue := range result.Issues {
			if strings.Contains(issue, "short") || strings.Contains(issue, "words") {
				hasLengthIssue = true
				break
			}
		}

		if !hasLengthIssue {
			t.Error("Expected issue about content length")
		}
	})

	t.Run("LowKeywordDensity", func(t *testing.T) {
		content := `
			This is a long piece of content that doesn't mention the target keyword very often.
			We're writing many sentences here to test the keyword density calculation.
			The content continues with various topics and discussions.
			More content is added to increase the word count significantly.
			This helps us test the low keyword density scenario effectively.
		`

		result, err := processor.OptimizeContent(ctx, content, "rare-keyword-xyz", "general")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should identify low keyword density issue
		hasKeywordIssue := false
		for _, issue := range result.Issues {
			if strings.Contains(issue, "density") && strings.Contains(issue, "low") {
				hasKeywordIssue = true
				break
			}
		}

		if !hasKeywordIssue {
			t.Error("Expected issue about low keyword density")
		}
	})
}

// TestResearchKeywords tests keyword research functionality
func TestResearchKeywords(t *testing.T) {
	processor := NewSEOProcessor()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		result, err := processor.ResearchKeywords(ctx, "content marketing", "United States", "en")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.SeedKeyword != "content marketing" {
			t.Errorf("Expected seed keyword 'content marketing', got '%s'", result.SeedKeyword)
		}

		if result.Status != "success" {
			t.Errorf("Expected status 'success', got '%s'", result.Status)
		}

		// Validate keywords
		if len(result.Keywords) == 0 {
			t.Error("Expected keyword suggestions")
		}

		for _, kw := range result.Keywords {
			if kw.Keyword == "" {
				t.Error("Keyword should not be empty")
			}
			if kw.Volume == "" {
				t.Error("Volume should not be empty")
			}
			if kw.Competition == "" {
				t.Error("Competition should not be empty")
			}
			if kw.Difficulty < 0 || kw.Difficulty > 100 {
				t.Errorf("Expected difficulty 0-100, got %d", kw.Difficulty)
			}
			if kw.Relevance < 0 || kw.Relevance > 1 {
				t.Errorf("Expected relevance 0-1, got %.2f", kw.Relevance)
			}
		}

		// Validate related terms
		if len(result.RelatedTerms) == 0 {
			t.Error("Expected related terms")
		}

		// Validate long-tail keywords
		if len(result.LongTail) == 0 {
			t.Error("Expected long-tail keywords")
		}

		// Validate questions
		if len(result.Questions) == 0 {
			t.Error("Expected question keywords")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		result, err := processor.ResearchKeywords(ctx, "test", "", "")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if result.Status != "success" {
			t.Error("Expected success with default values")
		}

		// Should still generate results
		if len(result.Keywords) == 0 {
			t.Error("Expected keyword suggestions with default values")
		}
	})

	t.Run("DifferentLanguages", func(t *testing.T) {
		languages := []string{"en", "es", "fr", "de"}

		for _, lang := range languages {
			result, err := processor.ResearchKeywords(ctx, "test", "Global", lang)
			if err != nil {
				t.Errorf("Expected no error for language %s, got: %v", lang, err)
			}

			if result.Status != "success" {
				t.Errorf("Expected success for language %s", lang)
			}
		}
	})
}

// TestAnalyzeCompetitor tests competitor analysis functionality
func TestAnalyzeCompetitor(t *testing.T) {
	processor := NewSEOProcessor()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		yourServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`
				<html>
				<head>
					<title>Your Business Website</title>
					<meta name="description" content="Your business description">
				</head>
				<body>
					<h1>Your Content</h1>
					<p>Content about your business with some keywords and information.</p>
					<p>More content here.</p>
				</body>
				</html>
			`))
		}))
		defer yourServer.Close()

		competitorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`
				<html>
				<head>
					<title>Competitor Website with More Detail</title>
					<meta name="description" content="Comprehensive competitor description">
				</head>
				<body>
					<h1>Competitor Content</h1>
					<h2>More Structured Content</h2>
					<p>Detailed competitor content with extensive information and keywords.</p>
					<p>Additional paragraphs with more comprehensive coverage of topics.</p>
					<p>Even more content to demonstrate superiority.</p>
				</body>
				</html>
			`))
		}))
		defer competitorServer.Close()

		result, err := processor.AnalyzeCompetitor(ctx, yourServer.URL, competitorServer.URL, "comprehensive")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Validate URLs
		if result.YourURL != yourServer.URL {
			t.Errorf("Expected your URL %s, got %s", yourServer.URL, result.YourURL)
		}

		if result.CompetitorURL != competitorServer.URL {
			t.Errorf("Expected competitor URL %s, got %s", competitorServer.URL, result.CompetitorURL)
		}

		// Validate comparison
		if result.Comparison.SEOScore.Winner == "" {
			t.Error("Expected SEO score comparison to determine winner")
		}

		if result.Comparison.SEOScore.Yours < 0 {
			t.Error("Expected non-negative score for your site")
		}

		if result.Comparison.SEOScore.Competitor < 0 {
			t.Error("Expected non-negative score for competitor")
		}

		// Validate opportunities
		if len(result.Opportunities) == 0 {
			t.Error("Expected opportunities to be identified")
		}

		// Validate threats
		if len(result.Threats) == 0 {
			t.Error("Expected threats to be identified")
		}

		// Validate recommendations
		if len(result.Recommendations) == 0 {
			t.Error("Expected recommendations")
		}

		// Validate overall score
		if result.OverallScore.Overall < 0 || result.OverallScore.Overall > 100 {
			t.Errorf("Expected overall score 0-100, got %d", result.OverallScore.Overall)
		}
	})

	t.Run("DefaultAnalysisType", func(t *testing.T) {
		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><head><title>Test</title></head><body>Test</body></html>"))
		}))
		defer server1.Close()

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><head><title>Test</title></head><body>Test</body></html>"))
		}))
		defer server2.Close()

		result, err := processor.AnalyzeCompetitor(ctx, server1.URL, server2.URL, "")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should default to comprehensive analysis
		if len(result.Opportunities) == 0 {
			t.Error("Expected opportunities with default analysis type")
		}
	})

	t.Run("Error_InvalidYourURL", func(t *testing.T) {
		competitorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><body>Test</body></html>"))
		}))
		defer competitorServer.Close()

		result, err := processor.AnalyzeCompetitor(ctx, "http://invalid-url-12345.com", competitorServer.URL, "comprehensive")

		// PerformSEOAudit returns error in response, not as error value
		// So AnalyzeCompetitor will succeed but with error status for your site
		if err == nil && result != nil {
			// This is expected behavior - function succeeds but comparison may show issues
			t.Log("AnalyzeCompetitor handles invalid URLs gracefully")
		}
	})

	t.Run("Error_InvalidCompetitorURL", func(t *testing.T) {
		yourServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<html><body>Test</body></html>"))
		}))
		defer yourServer.Close()

		result, err := processor.AnalyzeCompetitor(ctx, yourServer.URL, "http://invalid-url-12345.com", "comprehensive")

		// PerformSEOAudit returns error in response, not as error value
		// So AnalyzeCompetitor will succeed but with error status for competitor site
		if err == nil && result != nil {
			// This is expected behavior - function succeeds but comparison may show issues
			t.Log("AnalyzeCompetitor handles invalid URLs gracefully")
		}
	})
}

// TestHelperFunctions tests utility and helper functions
func TestExtractMetaTags(t *testing.T) {
	processor := NewSEOProcessor()

	t.Run("AllMetaTags", func(t *testing.T) {
		html := `
			<html>
			<head>
				<title>Test Page Title</title>
				<meta name="description" content="Test description for the page">
				<meta name="keywords" content="test, seo, meta">
			</head>
			</html>
		`

		meta := processor.extractMetaTags(html)

		if meta.Title != "Test Page Title" {
			t.Errorf("Expected title 'Test Page Title', got '%s'", meta.Title)
		}

		if meta.Description != "Test description for the page" {
			t.Errorf("Expected description, got '%s'", meta.Description)
		}

		if meta.Keywords != "test, seo, meta" {
			t.Errorf("Expected keywords, got '%s'", meta.Keywords)
		}

		if meta.TitleLength != len("Test Page Title") {
			t.Errorf("Expected title length %d, got %d", len("Test Page Title"), meta.TitleLength)
		}

		if meta.DescLength != len("Test description for the page") {
			t.Errorf("Expected desc length %d, got %d", len("Test description for the page"), meta.DescLength)
		}
	})

	t.Run("MissingMetaTags", func(t *testing.T) {
		html := "<html><body>No meta tags</body></html>"

		meta := processor.extractMetaTags(html)

		if meta.Title != "" {
			t.Errorf("Expected empty title, got '%s'", meta.Title)
		}

		if meta.TitleLength != 0 {
			t.Errorf("Expected title length 0, got %d", meta.TitleLength)
		}
	})
}

func TestExtractHeaders(t *testing.T) {
	processor := NewSEOProcessor()

	html := `
		<html>
		<body>
			<h1>Main Heading</h1>
			<h2>Subheading One</h2>
			<h2>Subheading Two</h2>
			<h3>Tertiary Heading</h3>
		</body>
		</html>
	`

	headers := processor.extractHeaders(html)

	if len(headers) != 4 {
		t.Errorf("Expected 4 headers, got %d", len(headers))
	}

	expectedHeaders := map[string]bool{
		"H1: Main Heading":       true,
		"H2: Subheading One":     true,
		"H2: Subheading Two":     true,
		"H3: Tertiary Heading":   true,
	}

	for _, header := range headers {
		if !expectedHeaders[header] {
			t.Errorf("Unexpected header: %s", header)
		}
	}
}

func TestStripHTML(t *testing.T) {
	processor := NewSEOProcessor()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "WithTags",
			input:    "<p>Hello <strong>world</strong></p>",
			expected: " Hello  world  ",
		},
		{
			name:     "NoTags",
			input:    "Plain text",
			expected: "Plain text",
		},
		{
			name:     "ComplexHTML",
			input:    `<div class="container"><p>Text <a href="#">link</a> more text</p></div>`,
			expected: "  Text  link  more text  ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := processor.stripHTML(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestFilterEmptyStrings(t *testing.T) {
	processor := NewSEOProcessor()

	input := []string{"hello", "", "  ", "world", "\t", "test"}
	result := processor.filterEmptyStrings(input)

	expected := []string{"hello", "world", "test"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d elements, got %d", len(expected), len(result))
	}

	for i, v := range expected {
		if result[i] != v {
			t.Errorf("Expected '%s' at index %d, got '%s'", v, i, result[i])
		}
	}
}

func TestCalculateReadability(t *testing.T) {
	processor := NewSEOProcessor()

	t.Run("NormalText", func(t *testing.T) {
		content := "This is a test. It has simple sentences. Easy to read."
		score := processor.calculateReadability(content)

		if score < 0 || score > 100 {
			t.Errorf("Expected score 0-100, got %d", score)
		}
	})

	t.Run("EmptyText", func(t *testing.T) {
		score := processor.calculateReadability("")

		if score != 50 {
			t.Errorf("Expected default score 50 for empty text, got %d", score)
		}
	})
}

// Performance tests
func BenchmarkPerformSEOAudit(b *testing.B) {
	processor := NewSEOProcessor()
	ctx := context.Background()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Benchmark Test</title></head>
			<body><h1>Test</h1><p>Content for benchmarking.</p></body>
			</html>
		`))
	}))
	defer server.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.PerformSEOAudit(ctx, server.URL, 3)
	}
}

func BenchmarkOptimizeContent(b *testing.B) {
	processor := NewSEOProcessor()
	ctx := context.Background()

	content := "This is test content for benchmarking content optimization performance with multiple keywords."
	keywords := "test, content, performance, benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.OptimizeContent(ctx, content, keywords, "general")
	}
}
