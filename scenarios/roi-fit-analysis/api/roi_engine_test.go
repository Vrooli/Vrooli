package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

)

// TestNewROIAnalysisEngine tests engine creation
func TestNewROIAnalysisEngine(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		mockDB := setupMockDatabase(t)
		engine := NewROIAnalysisEngine(mockDB)

		if engine == nil {
			t.Fatal("Expected engine to be created")
		}

		if engine.db != mockDB {
			t.Error("Database not set correctly")
		}

		if engine.httpClient == nil {
			t.Error("HTTP client not initialized")
		}

		if engine.ollamaClient == nil {
			t.Error("Ollama client not initialized")
		}

		if engine.ctx == nil {
			t.Error("Context not initialized")
		}

		if engine.cancel == nil {
			t.Error("Cancel function not initialized")
		}

		// Clean up context
		if engine.cancel != nil {
			engine.cancel()
		}
	})

	t.Run("NilDatabase", func(t *testing.T) {
		engine := NewROIAnalysisEngine(nil)

		if engine == nil {
			t.Fatal("Expected engine to be created even with nil database")
		}

		if engine.db != nil {
			t.Error("Database should be nil")
		}

		if engine.cancel != nil {
			engine.cancel()
		}
	})
}

// TestMockOllamaClient tests the mock Ollama client
func TestMockOllamaClient(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DefaultMockResponse", func(t *testing.T) {
		client := &MockOllamaClient{}
		response, err := client.Generate("test prompt", "llama3.2")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if response == "" {
			t.Error("Expected response, got empty string")
		}

		// Verify it's valid JSON
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(response), &result); err != nil {
			t.Errorf("Expected valid JSON response, got error: %v", err)
		}
	})

	t.Run("CustomMockResponse", func(t *testing.T) {
		expectedResponse := `{"custom": "response"}`
		client := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return expectedResponse, nil
			},
		}

		response, err := client.Generate("test", "model")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if response != expectedResponse {
			t.Errorf("Expected '%s', got '%s'", expectedResponse, response)
		}
	})

	t.Run("MockError", func(t *testing.T) {
		expectedError := fmt.Errorf("mock error")
		client := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return "", expectedError
			},
		}

		_, err := client.Generate("test", "model")
		if err == nil {
			t.Error("Expected error, got nil")
		}

		if err.Error() != expectedError.Error() {
			t.Errorf("Expected error '%v', got '%v'", expectedError, err)
		}
	})
}

// TestROIAnalysisResult tests ROI analysis result structure
func TestROIAnalysisResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateMockResult", func(t *testing.T) {
		result := createMockROIAnalysisResult()

		if result == nil {
			t.Fatal("Expected result to be created")
		}

		if result.ROIPercentage <= 0 {
			t.Error("ROI percentage should be positive")
		}

		if result.PaybackMonths <= 0 {
			t.Error("Payback months should be positive")
		}

		if result.RiskLevel == "" {
			t.Error("Risk level should be set")
		}

		if len(result.Recommendations) == 0 {
			t.Error("Should have recommendations")
		}
	})

	t.Run("ValidateROIPercentage", func(t *testing.T) {
		result := createMockROIAnalysisResult()

		if result.ROIPercentage < 0 || result.ROIPercentage > 10000 {
			t.Errorf("ROI percentage should be reasonable, got %.2f", result.ROIPercentage)
		}
	})

	t.Run("ValidateRiskLevel", func(t *testing.T) {
		result := createMockROIAnalysisResult()

		validRiskLevels := map[string]bool{
			"low":    true,
			"medium": true,
			"high":   true,
		}

		if !validRiskLevels[result.RiskLevel] {
			t.Errorf("Risk level should be low/medium/high, got '%s'", result.RiskLevel)
		}
	})
}

// TestMarketResearchResult tests market research result structure
func TestMarketResearchResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateMockResult", func(t *testing.T) {
		result := createMockMarketResearchResult()

		if result == nil {
			t.Fatal("Expected result to be created")
		}

		if result.MarketSize <= 0 {
			t.Error("Market size should be positive")
		}

		if result.MarketGrowthRate < 0 {
			t.Error("Growth rate should not be negative")
		}

		if result.PESTEL == nil {
			t.Error("PESTEL analysis should be included")
		}

		if len(result.MarketTrends) == 0 {
			t.Error("Should have market trends")
		}
	})

	t.Run("PESTELStructure", func(t *testing.T) {
		result := createMockMarketResearchResult()

		pestel := result.PESTEL
		if pestel == nil {
			t.Fatal("PESTEL should not be nil")
		}

		// At least one factor should be populated
		totalFactors := len(pestel.Political) + len(pestel.Economic) + len(pestel.Social) +
			len(pestel.Technological) + len(pestel.Environmental) + len(pestel.Legal)

		if totalFactors == 0 {
			t.Error("PESTEL analysis should have at least one factor")
		}
	})
}

// TestFinancialCalculationResult tests financial calculation result
func TestFinancialCalculationResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateMockResult", func(t *testing.T) {
		result := createMockFinancialResult()

		if result == nil {
			t.Fatal("Expected result to be created")
		}

		if result.InitialInvestment <= 0 {
			t.Error("Initial investment should be positive")
		}

		if len(result.MonthlyRevenue) == 0 {
			t.Error("Should have revenue projections")
		}

		if len(result.CashFlowProjection) == 0 {
			t.Error("Should have cash flow projections")
		}

		if result.BreakevenMonth <= 0 {
			t.Error("Breakeven month should be positive")
		}
	})

	t.Run("RevenueProjectionLength", func(t *testing.T) {
		result := createMockFinancialResult()

		if len(result.MonthlyRevenue) != 24 {
			t.Errorf("Expected 24 months of revenue projection, got %d", len(result.MonthlyRevenue))
		}

		if len(result.CashFlowProjection) != 24 {
			t.Errorf("Expected 24 months of cash flow projection, got %d", len(result.CashFlowProjection))
		}
	})

	t.Run("CostStructure", func(t *testing.T) {
		result := createMockFinancialResult()

		if len(result.CostStructure) == 0 {
			t.Error("Cost structure should be populated")
		}

		totalCosts := 0.0
		for _, cost := range result.CostStructure {
			if cost < 0 {
				t.Error("Individual costs should not be negative")
			}
			totalCosts += cost
		}

		if totalCosts <= 0 {
			t.Error("Total costs should be positive")
		}
	})
}

// TestCompetitiveAnalysisResult tests competitive analysis result
func TestCompetitiveAnalysisResult(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateMockResult", func(t *testing.T) {
		result := createMockCompetitiveResult()

		if result == nil {
			t.Fatal("Expected result to be created")
		}

		if len(result.DirectCompetitors) == 0 {
			t.Error("Should have at least one direct competitor")
		}

		if result.CompetitiveScore < 0 || result.CompetitiveScore > 100 {
			t.Errorf("Competitive score should be 0-100, got %.2f", result.CompetitiveScore)
		}
	})

	t.Run("CompetitorStructure", func(t *testing.T) {
		result := createMockCompetitiveResult()

		if len(result.DirectCompetitors) > 0 {
			comp := result.DirectCompetitors[0]

			if comp.Name == "" {
				t.Error("Competitor name should not be empty")
			}

			if comp.MarketShare < 0 || comp.MarketShare > 100 {
				t.Errorf("Market share should be 0-100, got %.2f", comp.MarketShare)
			}

			validThreatLevels := map[string]bool{
				"low":    true,
				"medium": true,
				"high":   true,
			}

			if !validThreatLevels[comp.ThreatLevel] {
				t.Errorf("Threat level should be low/medium/high, got '%s'", comp.ThreatLevel)
			}
		}
	})
}

// TestExecutiveSummary tests executive summary structure
func TestExecutiveSummary(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateMockSummary", func(t *testing.T) {
		summary := createMockExecutiveSummary()

		if summary == nil {
			t.Fatal("Expected summary to be created")
		}

		if summary.OverallRating == "" {
			t.Error("Overall rating should be set")
		}

		if summary.InvestmentAdvice == "" {
			t.Error("Investment advice should be set")
		}

		if len(summary.KeySuccessFactors) == 0 {
			t.Error("Should have key success factors")
		}

		if len(summary.MajorRisks) == 0 {
			t.Error("Should have major risks")
		}

		if len(summary.ActionItems) == 0 {
			t.Error("Should have action items")
		}
	})

	t.Run("ValidateRating", func(t *testing.T) {
		summary := createMockExecutiveSummary()

		validRatings := map[string]bool{
			"Excellent": true,
			"Good":      true,
			"Fair":      true,
			"Poor":      true,
		}

		if !validRatings[summary.OverallRating] {
			t.Errorf("Overall rating should be Excellent/Good/Fair/Poor, got '%s'", summary.OverallRating)
		}
	})
}

// TestOllamaClient tests Ollama client creation
func TestOllamaClient(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NewClient", func(t *testing.T) {
		client := NewOllamaClient()

		if client == nil {
			t.Fatal("Expected client to be created")
		}

		if client.baseURL == "" {
			t.Error("Base URL should be set")
		}

		if client.httpClient == nil {
			t.Error("HTTP client should be initialized")
		}
	})
}

// TestComprehensiveAnalysisRequest tests request structure validation
func TestComprehensiveAnalysisRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateValidRequest", func(t *testing.T) {
		req := createTestComprehensiveRequest()

		if req.Idea == "" {
			t.Error("Idea should not be empty")
		}

		if req.Budget <= 0 {
			t.Error("Budget should be positive")
		}

		if req.Timeline == "" {
			t.Error("Timeline should not be empty")
		}

		if len(req.Skills) == 0 {
			t.Error("Should have at least one skill")
		}
	})

	t.Run("MarshallToJSON", func(t *testing.T) {
		req := createTestComprehensiveRequest()

		data, err := json.Marshal(req)
		if err != nil {
			t.Errorf("Failed to marshal request: %v", err)
		}

		var unmarshalled ComprehensiveAnalysisRequest
		if err := json.Unmarshal(data, &unmarshalled); err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if unmarshalled.Idea != req.Idea {
			t.Error("Idea not preserved after JSON round-trip")
		}

		if unmarshalled.Budget != req.Budget {
			t.Error("Budget not preserved after JSON round-trip")
		}
	})
}

// TestPerformComprehensiveAnalysis tests the complete analysis workflow
func TestPerformComprehensiveAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success_WithMockOllama", func(t *testing.T) {
		mockClient := &MockOllamaClient{}
		engine := &ROIAnalysisEngine{
			db:           nil,
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, err := engine.PerformComprehensiveAnalysis(&req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result, got nil")
		}

		if result.AnalysisID == "" {
			t.Error("Analysis ID should be set")
		}

		if result.Success != true {
			t.Errorf("Expected success=true, got %v", result.Success)
		}

		if result.ROIAnalysis == nil {
			t.Error("ROI analysis should be present")
		}

		if result.MarketResearch == nil {
			t.Error("Market research should be present")
		}

		if result.FinancialMetrics == nil {
			t.Error("Financial metrics should be present")
		}

		if result.Competitive == nil {
			t.Error("Competitive analysis should be present")
		}

		if result.Summary == nil {
			t.Error("Executive summary should be present")
		}
	})

	t.Run("Error_InvalidOllamaResponse", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return "invalid json", nil
			},
		}

		engine := &ROIAnalysisEngine{
			db:           nil,
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, err := engine.PerformComprehensiveAnalysis(&req)

		if err != nil {
			t.Errorf("Should not return error on partial failure: %v", err)
		}

		if result == nil {
			t.Fatal("Result should still be returned")
		}

		// Analysis should fail but still return a result
		if result.Success {
			t.Error("Expected success=false due to JSON parse errors")
		}

		if result.Error == "" {
			t.Error("Error message should be set")
		}
	})

	t.Run("Error_OllamaGenerationFailure", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return "", fmt.Errorf("ollama generation failed")
			},
		}

		engine := &ROIAnalysisEngine{
			db:           nil,
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, err := engine.PerformComprehensiveAnalysis(&req)

		if err != nil {
			t.Errorf("Should not return error: %v", err)
		}

		if result.Success {
			t.Error("Expected success=false due to generation failures")
		}
	})

	t.Run("Timing_ExecutionTime", func(t *testing.T) {
		mockClient := &MockOllamaClient{}
		engine := &ROIAnalysisEngine{
			db:           nil,
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, _ := engine.PerformComprehensiveAnalysis(&req)

		// Execution time may be 0 for very fast mocked operations
		if result.ExecutionTime < 0 {
			t.Error("Execution time should not be negative")
		}
	})
}

// TestPerformROIAnalysis tests individual ROI analysis workflow
func TestPerformROIAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return `{
					"roi_percentage": 75.5,
					"npv": 100000,
					"irr": 18.5,
					"payback_months": 16,
					"risk_level": "medium",
					"confidence_score": 72.0,
					"breakeven_point": 45000,
					"estimated_revenue": 175000,
					"estimated_costs": 75000,
					"profit_margin": 57.1,
					"recommendations": ["Test recommendation"]
				}`, nil
			},
		}

		engine := &ROIAnalysisEngine{
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, err := engine.performROIAnalysis(&req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result")
		}

		if result.ROIPercentage != 75.5 {
			t.Errorf("Expected ROI 75.5, got %.2f", result.ROIPercentage)
		}

		if result.RiskLevel != "medium" {
			t.Errorf("Expected risk level 'medium', got '%s'", result.RiskLevel)
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return "not json", nil
			},
		}

		engine := &ROIAnalysisEngine{
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		_, err := engine.performROIAnalysis(&req)

		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})
}

// TestPerformMarketResearch tests market research workflow
func TestPerformMarketResearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return `{
					"market_size": 5000000000,
					"market_growth_rate": 12.5,
					"target_market_size": 500000000,
					"market_saturation": "medium",
					"customer_demand": "high",
					"market_trends": ["AI adoption"],
					"entry_barriers": ["Capital"],
					"opportunities": ["Market gap"],
					"threats": ["Competition"],
					"pestel_analysis": {
						"political": ["Stable"],
						"economic": ["Growing"],
						"social": ["Digital"],
						"technological": ["AI"],
						"environmental": ["Sustainable"],
						"legal": ["Privacy"]
					},
					"key_insights": ["Insight"],
					"data_sources": ["Source"]
				}`, nil
			},
		}

		engine := &ROIAnalysisEngine{
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, err := engine.performMarketResearch(&req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result.MarketSize != 5000000000 {
			t.Errorf("Expected market size 5B, got %.2f", result.MarketSize)
		}

		if result.PESTEL == nil {
			t.Error("PESTEL analysis should be included")
		}
	})
}

// TestPerformFinancialCalculations tests financial calculations
func TestPerformFinancialCalculations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return `{
					"initial_investment": 50000,
					"monthly_burn_rate": 8000,
					"monthly_revenue_forecast": [5000, 10000],
					"cash_flow_projection": [-3000, 2000],
					"breakeven_month": 6,
					"funding_needed": 60000,
					"runway_months": 7,
					"cost_structure": {"dev": 30000},
					"revenue_streams": {"subs": 50000},
					"key_assumptions": ["10% growth"],
					"sensitivity_analysis": {"best": 100000}
				}`, nil
			},
		}

		engine := &ROIAnalysisEngine{
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, err := engine.performFinancialCalculations(&req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result.BreakevenMonth != 6 {
			t.Errorf("Expected breakeven month 6, got %d", result.BreakevenMonth)
		}
	})
}

// TestPerformCompetitiveAnalysis tests competitive analysis
func TestPerformCompetitiveAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return `{
					"direct_competitors": [{
						"name": "Competitor A",
						"market_share": 25.0,
						"strengths": ["Brand"],
						"weaknesses": ["Price"],
						"pricing_model": "subscription",
						"threat_level": "medium"
					}],
					"indirect_competitors": [],
					"competitive_advantages": ["Innovation"],
					"weaknesses": ["Resources"],
					"market_position": "challenger",
					"pricing_strategy": "value-based",
					"differentiation_opportunities": ["AI"],
					"threat_level": "medium",
					"competitive_score": 68.5
				}`, nil
			},
		}

		engine := &ROIAnalysisEngine{
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		result, err := engine.performCompetitiveAnalysis(&req)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(result.DirectCompetitors) == 0 {
			t.Error("Should have direct competitors")
		}

		if result.CompetitiveScore != 68.5 {
			t.Errorf("Expected competitive score 68.5, got %.2f", result.CompetitiveScore)
		}
	})
}

// TestGenerateExecutiveSummary tests executive summary generation
func TestGenerateExecutiveSummary(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		mockClient := &MockOllamaClient{
			GenerateFunc: func(prompt, model string) (string, error) {
				return `{
					"overall_rating": "Good",
					"investment_advice": "Proceed with caution",
					"key_success_factors": ["PMF", "Execution"],
					"major_risks": ["Competition", "Timing"],
					"timeline_assessment": "6-9 months",
					"final_recommendation": "Recommended",
					"action_items": ["Validate", "MVP"],
					"next_steps": ["Research", "Prototype"]
				}`, nil
			},
		}

		engine := &ROIAnalysisEngine{
			ollamaClient: mockClient,
		}

		req := createTestComprehensiveRequest()
		response := &ComprehensiveAnalysisResponse{
			ROIAnalysis:      createMockROIAnalysisResult(),
			MarketResearch:   createMockMarketResearchResult(),
			FinancialMetrics: createMockFinancialResult(),
			Competitive:      createMockCompetitiveResult(),
		}

		result, err := engine.generateExecutiveSummary(&req, response)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result.OverallRating != "Good" {
			t.Errorf("Expected rating 'Good', got '%s'", result.OverallRating)
		}

		if len(result.KeySuccessFactors) == 0 {
			t.Error("Should have key success factors")
		}
	})
}

// Benchmark tests for ROI engine
func BenchmarkNewROIAnalysisEngine(b *testing.B) {
	mockDB := &sql.DB{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine := NewROIAnalysisEngine(mockDB)
		if engine.cancel != nil {
			engine.cancel()
		}
	}
}

func BenchmarkCreateMockResults(b *testing.B) {
	b.Run("ROIAnalysis", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			createMockROIAnalysisResult()
		}
	})

	b.Run("MarketResearch", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			createMockMarketResearchResult()
		}
	})

	b.Run("FinancialCalc", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			createMockFinancialResult()
		}
	})

	b.Run("Competitive", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			createMockCompetitiveResult()
		}
	})
}

func BenchmarkPerformComprehensiveAnalysis(b *testing.B) {
	mockClient := &MockOllamaClient{}
	engine := &ROIAnalysisEngine{
		db:           nil,
		ollamaClient: mockClient,
	}

	req := createTestComprehensiveRequest()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.PerformComprehensiveAnalysis(&req)
	}
}
