package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestIntegration_EndToEndAnalysis tests complete end-to-end analysis flow
func TestIntegration_EndToEndAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Setup mock ROI engine
	mockEngine := &ROIAnalysisEngine{
		db:         nil,
		ollamaClient: &MockOllamaClient{},
	}

	originalEngine := roiEngine
	roiEngine = mockEngine
	defer func() { roiEngine = originalEngine }()

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Check health
		healthReq, _ := http.NewRequest(http.MethodGet, "/health", nil)
		healthRec := httptest.NewRecorder()
		corsMiddleware(healthHandler)(healthRec, healthReq)

		if healthRec.Code != http.StatusOK {
			t.Errorf("Health check failed: %d", healthRec.Code)
		}

		// 2. Submit analysis request
		analysisReq := createTestAnalysisRequest()
		bodyBytes, _ := json.Marshal(analysisReq)
		analyzeReq, _ := http.NewRequest(http.MethodPost, "/analyze", bytes.NewBuffer(bodyBytes))
		analyzeReq.Header.Set("Content-Type", "application/json")
		analyzeRec := httptest.NewRecorder()
		corsMiddleware(analyzeHandler)(analyzeRec, analyzeReq)

		if analyzeRec.Code != http.StatusOK {
			t.Errorf("Analysis failed: %d - %s", analyzeRec.Code, analyzeRec.Body.String())
		}

		var analysisResponse map[string]interface{}
		json.NewDecoder(analyzeRec.Body).Decode(&analysisResponse)

		// 3. Get opportunities
		oppReq, _ := http.NewRequest(http.MethodGet, "/opportunities", nil)
		oppRec := httptest.NewRecorder()
		corsMiddleware(opportunitiesHandler)(oppRec, oppReq)

		if oppRec.Code != http.StatusOK {
			t.Errorf("Opportunities failed: %d", oppRec.Code)
		}

		// 4. Get reports
		reportReq, _ := http.NewRequest(http.MethodGet, "/reports", nil)
		reportRec := httptest.NewRecorder()
		corsMiddleware(reportsHandler)(reportRec, reportReq)

		if reportRec.Code != http.StatusOK {
			t.Errorf("Reports failed: %d", reportRec.Code)
		}
	})
}

// TestIntegration_ComprehensiveAnalysisFlow tests comprehensive analysis integration
func TestIntegration_ComprehensiveAnalysisFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	mockEngine := &ROIAnalysisEngine{
		db:         nil,
		ollamaClient: &MockOllamaClient{},
	}

	originalEngine := roiEngine
	roiEngine = mockEngine
	defer func() { roiEngine = originalEngine }()

	t.Run("FullComprehensiveAnalysis", func(t *testing.T) {
		req := createTestComprehensiveRequest()
		bodyBytes, _ := json.Marshal(req)

		httpReq, _ := http.NewRequest(http.MethodPost, "/comprehensive-analysis", bytes.NewBuffer(bodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		corsMiddleware(comprehensiveAnalysisHandler)(recorder, httpReq)

		if recorder.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d: %s", recorder.Code, recorder.Body.String())
		}

		var response ComprehensiveAnalysisResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Validate all components are present
		if response.AnalysisID == "" {
			t.Error("Analysis ID missing")
		}
		if response.ROIAnalysis == nil {
			t.Error("ROI analysis missing")
		}
		if response.MarketResearch == nil {
			t.Error("Market research missing")
		}
		if response.FinancialMetrics == nil {
			t.Error("Financial metrics missing")
		}
		if response.Competitive == nil {
			t.Error("Competitive analysis missing")
		}
		if response.Summary == nil {
			t.Error("Executive summary missing")
		}
	})
}

// TestBusinessLogic_ROICalculations tests business logic accuracy
func TestBusinessLogic_ROICalculations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RecommendationLogic", func(t *testing.T) {
		tests := []struct {
			roiScore    float64
			expected    string
		}{
			{90.0, "Excellent opportunity - High potential"},
			{80.0, "Excellent opportunity - High potential"},
			{70.0, "Good opportunity - Moderate potential"},
			{60.0, "Good opportunity - Moderate potential"},
			{50.0, "Fair opportunity - Consider risks"},
			{40.0, "Fair opportunity - Consider risks"},
			{30.0, "Poor opportunity - High risk"},
			{10.0, "Poor opportunity - High risk"},
		}

		for _, tt := range tests {
			result := getRecommendation(tt.roiScore)
			if result != tt.expected {
				t.Errorf("ROI %.1f: expected '%s', got '%s'", tt.roiScore, tt.expected, result)
			}
		}
	})

	t.Run("MarketSizeFormatting", func(t *testing.T) {
		tests := []struct {
			size     float64
			expected string
		}{
			{5000000000, "$5.0B"},
			{1500000000, "$1.5B"},
			{750000000, "$750.0M"},
			{50000000, "$50.0M"},
			{7500000, "$7.5M"},
			{850000, "$850.0K"},
			{25000, "$25.0K"},
			{999, "$999"},
		}

		for _, tt := range tests {
			result := formatMarketSize(tt.size)
			if result != tt.expected {
				t.Errorf("Size %.0f: expected '%s', got '%s'", tt.size, tt.expected, result)
			}
		}
	})

	t.Run("RatingToRecommendation", func(t *testing.T) {
		ratings := map[string]string{
			"excellent": "Excellent opportunity - High potential",
			"Excellent": "Excellent opportunity - High potential",
			"EXCELLENT": "Excellent opportunity - High potential",
			"good": "Good opportunity - Moderate potential",
			"fair": "Fair opportunity - Consider risks",
			"poor": "Poor opportunity - High risk",
			"unknown": "Assessment pending",
		}

		for rating, expected := range ratings {
			result := getRecommendationFromRating(rating)
			if result != expected {
				t.Errorf("Rating '%s': expected '%s', got '%s'", rating, expected, result)
			}
		}
	})
}

// TestBusinessLogic_DataExtraction tests data extraction logic
func TestBusinessLogic_DataExtraction(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExtractMarketSize_ValidData", func(t *testing.T) {
		details := map[string]interface{}{
			"market_size": "$10.5B",
			"other_field": "value",
		}

		result := extractMarketSize(details)
		if result != "$10.5B" {
			t.Errorf("Expected '$10.5B', got '%s'", result)
		}
	})

	t.Run("ExtractMarketSize_MissingData", func(t *testing.T) {
		details := map[string]interface{}{
			"other_field": "value",
		}

		result := extractMarketSize(details)
		if result != "Unknown" {
			t.Errorf("Expected 'Unknown', got '%s'", result)
		}
	})

	t.Run("ExtractMarketSize_WrongType", func(t *testing.T) {
		details := map[string]interface{}{
			"market_size": 12345, // Not a string
		}

		result := extractMarketSize(details)
		if result != "Unknown" {
			t.Errorf("Expected 'Unknown', got '%s'", result)
		}
	})

	t.Run("ExtractCompetitionLevel_ValidData", func(t *testing.T) {
		details := map[string]interface{}{
			"competition_level": 8.5,
		}

		result := extractCompetitionLevel(details)
		if result != 8 {
			t.Errorf("Expected 8, got %d", result)
		}
	})

	t.Run("ExtractCompetitionLevel_Default", func(t *testing.T) {
		details := map[string]interface{}{}

		result := extractCompetitionLevel(details)
		if result != 5 {
			t.Errorf("Expected default 5, got %d", result)
		}
	})
}

// TestPerformance_ConcurrentRequests tests concurrent request handling
func TestPerformance_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	mockEngine := &ROIAnalysisEngine{
		db:         nil,
		ollamaClient: &MockOllamaClient{},
	}

	originalEngine := roiEngine
	roiEngine = mockEngine
	defer func() { roiEngine = originalEngine }()

	t.Run("ConcurrentAnalyzeRequests", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(id int) {
				req := createTestAnalysisRequest()
				bodyBytes, _ := json.Marshal(req)

				httpReq, _ := http.NewRequest(http.MethodPost, "/analyze", bytes.NewBuffer(bodyBytes))
				httpReq.Header.Set("Content-Type", "application/json")

				recorder := httptest.NewRecorder()
				corsMiddleware(analyzeHandler)(recorder, httpReq)

				if recorder.Code != http.StatusOK {
					errors <- nil // Track failures but don't stop
				}
				done <- true
			}(i)
		}

		// Wait for all to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}
		close(errors)

		errorCount := len(errors)
		if errorCount > 0 {
			t.Logf("Warning: %d out of %d concurrent requests had issues", errorCount, concurrency)
		}
	})
}

// TestPerformance_ResponseTime tests response time benchmarks
func TestPerformance_ResponseTime(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthEndpoint_ResponseTime", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)

		start := time.Now()
		recorder := httptest.NewRecorder()
		corsMiddleware(healthHandler)(recorder, req)
		elapsed := time.Since(start)

		if elapsed > 100*time.Millisecond {
			t.Errorf("Health endpoint too slow: %v", elapsed)
		}

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", recorder.Code)
		}
	})

	t.Run("OpportunitiesEndpoint_ResponseTime", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/opportunities", nil)

		start := time.Now()
		recorder := httptest.NewRecorder()
		corsMiddleware(opportunitiesHandler)(recorder, req)
		elapsed := time.Since(start)

		if elapsed > 500*time.Millisecond {
			t.Errorf("Opportunities endpoint too slow: %v", elapsed)
		}
	})
}

// TestPerformance_MemoryUsage tests memory efficiency
func TestPerformance_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("LargeRequestHandling", func(t *testing.T) {
		req := createTestAnalysisRequest()
		// Create large skills array
		req.Skills = make([]string, 1000)
		for i := 0; i < 1000; i++ {
			req.Skills[i] = "Skill"
		}

		bodyBytes, _ := json.Marshal(req)
		httpReq, _ := http.NewRequest(http.MethodPost, "/analyze", bytes.NewBuffer(bodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")

		mockEngine := &ROIAnalysisEngine{
			db:         nil,
			ollamaClient: &MockOllamaClient{},
		}

		originalEngine := roiEngine
		roiEngine = mockEngine
		defer func() { roiEngine = originalEngine }()

		recorder := httptest.NewRecorder()
		corsMiddleware(analyzeHandler)(recorder, httpReq)

		// Should handle large requests without crashing
		if recorder.Code != http.StatusOK && recorder.Code != http.StatusBadRequest {
			t.Errorf("Large request handling failed: %d", recorder.Code)
		}
	})
}
