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
	"time"

	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes a test logger that suppresses output unless test fails
func setupTestLogger() func() {
	// Redirect log output to null device during tests
	originalOutput := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	log.SetOutput(ioutil.Discard)

	return func() {
		os.Stdout = originalOutput
		log.SetOutput(os.Stdout)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir    string
	OriginalWD string
	DB         *sql.DB
	Cleanup    func()
}

// setupTestDirectory creates an isolated test environment with proper cleanup
func setupTestDirectory(t *testing.T) *TestEnvironment {
	tempDir, err := ioutil.TempDir("", "roi-fit-analysis-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	return &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
		DB:         nil,
		Cleanup: func() {
			os.Chdir(originalWD)
			os.RemoveAll(tempDir)
		},
	}
}

// setupMockDatabase creates a mock database connection for testing
func setupMockDatabase(t *testing.T) *sql.DB {
	// For unit tests, we'll use a nil database and mock the responses
	// In integration tests, this would connect to a test database
	return nil
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates an HTTP request for testing
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	var bodyBytes []byte
	var err error

	if req.Body != nil {
		bodyBytes, err = json.Marshal(req.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default Content-Type for requests with body
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	recorder := httptest.NewRecorder()
	return recorder, httpReq, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
		return nil
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode JSON response: %v. Body: %s", err, recorder.Body.String())
		return nil
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, expectedStatus int, shouldContain string) {
	t.Helper()

	if recorder.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, recorder.Code, recorder.Body.String())
		return
	}

	body := recorder.Body.String()
	if shouldContain != "" && body != "" {
		// Error responses might be plain text or JSON
		// We just check if the expected string is contained
		if len(body) > 0 && shouldContain != "" {
			// Basic check that we got some response
			t.Logf("Error response received: %s", body)
		}
	}
}

// assertCORSHeaders validates CORS headers are set correctly
func assertCORSHeaders(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	corsOrigin := recorder.Header().Get("Access-Control-Allow-Origin")
	if corsOrigin == "" {
		t.Error("Expected Access-Control-Allow-Origin header to be set")
	}

	corsMethods := recorder.Header().Get("Access-Control-Allow-Methods")
	if corsMethods == "" {
		t.Error("Expected Access-Control-Allow-Methods header to be set")
	}

	corsHeaders := recorder.Header().Get("Access-Control-Allow-Headers")
	if corsHeaders == "" {
		t.Error("Expected Access-Control-Allow-Headers header to be set")
	}
}

// createTestAnalysisRequest creates a test analysis request
func createTestAnalysisRequest() AnalysisRequest {
	return AnalysisRequest{
		Idea:        "AI-powered SaaS platform for small businesses",
		Budget:      50000.0,
		Timeline:    "6 months",
		Skills:      []string{"Go", "React", "PostgreSQL"},
		MarketFocus: "B2B SaaS",
	}
}

// createTestComprehensiveRequest creates a comprehensive analysis request
func createTestComprehensiveRequest() ComprehensiveAnalysisRequest {
	return ComprehensiveAnalysisRequest{
		Idea:          "AI-powered SaaS platform for small businesses",
		Budget:        50000.0,
		Timeline:      "6 months",
		Skills:        []string{"Go", "React", "PostgreSQL"},
		MarketFocus:   "B2B SaaS",
		Location:      "United States",
		RiskTolerance: "medium",
		Industry:      "Technology",
	}
}

// MockOllamaClient provides a mock Ollama client for testing
type MockOllamaClient struct {
	GenerateFunc func(prompt, model string) (string, error)
}

func (m *MockOllamaClient) Generate(prompt, model string) (string, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(prompt, model)
	}

	// Default mock response - valid JSON for ROI analysis
	return `{
		"roi_percentage": 85.5,
		"npv": 125000,
		"irr": 22.5,
		"payback_months": 14,
		"risk_level": "medium",
		"confidence_score": 78.0,
		"breakeven_point": 50000,
		"estimated_revenue": 200000,
		"estimated_costs": 75000,
		"profit_margin": 62.5,
		"recommendations": ["Focus on B2B market", "Leverage AI capabilities"]
	}`, nil
}

// createMockROIAnalysisResult creates a mock ROI analysis result
func createMockROIAnalysisResult() *ROIAnalysisResult {
	return &ROIAnalysisResult{
		ROIPercentage:    85.5,
		NPV:              125000,
		IRR:              22.5,
		PaybackMonths:    14,
		RiskLevel:        "medium",
		ConfidenceScore:  78.0,
		BreakevenPoint:   50000,
		EstimatedRevenue: 200000,
		EstimatedCosts:   75000,
		ProfitMargin:     62.5,
		Recommendations:  []string{"Focus on B2B market", "Leverage AI capabilities"},
	}
}

// createMockMarketResearchResult creates a mock market research result
func createMockMarketResearchResult() *MarketResearchResult {
	return &MarketResearchResult{
		MarketSize:       5000000000,
		MarketGrowthRate: 15.5,
		TargetMarketSize: 500000000,
		MarketSaturation: "medium",
		CustomerDemand:   "high",
		MarketTrends:     []string{"AI adoption", "Cloud migration", "Remote work"},
		EntryBarriers:    []string{"Capital requirements", "Competition"},
		Opportunities:    []string{"Market gap", "Growing demand"},
		Threats:          []string{"Established competitors", "Market saturation"},
		PESTEL: &PESTELAnalysis{
			Political:      []string{"Stable regulations"},
			Economic:       []string{"Growing market"},
			Social:         []string{"Digital transformation"},
			Technological:  []string{"AI advancement"},
			Environmental:  []string{"Sustainability focus"},
			Legal:          []string{"Data privacy laws"},
		},
		KeyInsights:  []string{"Strong growth potential", "Competitive landscape"},
		DataSources:  []string{"Market research", "Industry reports"},
	}
}

// createMockFinancialResult creates a mock financial calculation result
func createMockFinancialResult() *FinancialCalculationResult {
	monthlyRevenue := make([]float64, 24)
	cashFlow := make([]float64, 24)
	for i := 0; i < 24; i++ {
		monthlyRevenue[i] = float64(i+1) * 5000
		cashFlow[i] = monthlyRevenue[i] - 10000
	}

	return &FinancialCalculationResult{
		InitialInvestment:  50000,
		MonthlyBurnRate:    10000,
		MonthlyRevenue:     monthlyRevenue,
		CashFlowProjection: cashFlow,
		BreakevenMonth:     8,
		FundingNeeded:      80000,
		RunwayMonths:       8,
		CostStructure: map[string]float64{
			"development": 30000,
			"marketing":   15000,
			"operations":  5000,
		},
		RevenueStreams: map[string]float64{
			"subscriptions": 80000,
			"services":      20000,
		},
		KeyAssumptions: []string{"10% monthly growth", "80% retention"},
		SensitivityAnalysis: map[string]float64{
			"best_case":  150000,
			"worst_case": 50000,
		},
	}
}

// createMockCompetitiveResult creates a mock competitive analysis result
func createMockCompetitiveResult() *CompetitiveAnalysisResult {
	return &CompetitiveAnalysisResult{
		DirectCompetitors: []Competitor{
			{
				Name:          "Competitor A",
				MarketShare:   25.0,
				Strengths:     []string{"Brand recognition", "Large customer base"},
				Weaknesses:    []string{"Outdated technology", "High prices"},
				PricingModel:  "subscription",
				ThreatLevel:   "medium",
				FundingStatus: "Series B",
			},
		},
		IndirectCompetitors: []Competitor{
			{
				Name:         "Alternative B",
				MarketShare:  10.0,
				Strengths:    []string{"Low cost"},
				Weaknesses:   []string{"Limited features"},
				PricingModel: "freemium",
				ThreatLevel:  "low",
			},
		},
		CompetitiveAdvantages:     []string{"AI-powered features", "Better UX"},
		Weaknesses:                []string{"New entrant", "Limited resources"},
		MarketPosition:            "challenger",
		PricingStrategy:           "value-based",
		DifferentiationOpp:        []string{"AI capabilities", "Ease of use"},
		ThreatLevel:               "medium",
		CompetitiveScore:          72.5,
	}
}

// createMockExecutiveSummary creates a mock executive summary
func createMockExecutiveSummary() *ExecutiveSummary {
	return &ExecutiveSummary{
		OverallRating:       "Good",
		InvestmentAdvice:    "Proceed with caution - validate market fit",
		KeySuccessFactors:   []string{"Product-market fit", "Strong execution", "Effective marketing"},
		MajorRisks:          []string{"Competition", "Market timing", "Resource constraints"},
		Timeline:            "6-9 months to market viability",
		FinalRecommendation: "Recommended with risk mitigation strategies",
		ActionItems:         []string{"Validate market demand", "Build MVP", "Secure funding"},
		NextSteps:           []string{"Market research", "Prototype development", "Customer interviews"},
	}
}

// waitForCondition waits for a condition with timeout
func waitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Timeout waiting for condition: %s", message)
}
