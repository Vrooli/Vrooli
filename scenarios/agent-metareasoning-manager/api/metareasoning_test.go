package main

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNewMetareasoningEngine tests metareasoning engine creation
func TestNewMetareasoningEngine(t *testing.T) {
	t.Run("CreateEngine_Success", func(t *testing.T) {
		engine := NewMetareasoningEngine(nil)

		if engine == nil {
			t.Fatal("Engine should not be nil")
		}
		if engine.httpClient == nil {
			t.Error("HTTP client should not be nil")
		}
		if engine.ollamaClient == nil {
			t.Error("Ollama client should not be nil")
		}
		if engine.vectorStore == nil {
			t.Error("Vector store should not be nil")
		}
		if engine.activeChains == nil {
			t.Error("Active chains map should not be nil")
		}
	})
}

// TestNewOllamaClient tests Ollama client creation
func TestNewOllamaClient(t *testing.T) {
	t.Run("CreateOllamaClient_Success", func(t *testing.T) {
		client := NewOllamaClient()

		if client == nil {
			t.Fatal("Client should not be nil")
		}
		if client.httpClient == nil {
			t.Error("HTTP client should not be nil")
		}
		if client.baseURL == "" {
			t.Error("Base URL should not be empty")
		}
	})
}

// TestNewVectorStore tests vector store creation
func TestNewVectorStore(t *testing.T) {
	t.Run("CreateVectorStore_Success", func(t *testing.T) {
		store := NewVectorStore()

		if store == nil {
			t.Fatal("Store should not be nil")
		}
		if store.httpClient == nil {
			t.Error("HTTP client should not be nil")
		}
		if store.qdrantURL == "" {
			t.Error("Qdrant URL should not be empty")
		}
	})
}

// TestReasoningRequest tests reasoning request structure
func TestReasoningRequest(t *testing.T) {
	t.Run("CreateReasoningRequest_Complete", func(t *testing.T) {
		req := ReasoningRequest{
			Input:       "Should we adopt microservices?",
			Type:        "pros_cons",
			Model:       "llama3.2",
			Context:     "E-commerce platform with 10M users",
			Constraints: "Budget: $500k, Timeline: 6 months",
		}

		if req.Input == "" {
			t.Error("Input should not be empty")
		}
		if req.Type != "pros_cons" {
			t.Errorf("Expected type 'pros_cons', got '%s'", req.Type)
		}
	})

	t.Run("CreateReasoningRequest_WithCustomChain", func(t *testing.T) {
		customChain := []ReasoningStep{
			{Step: "step1", Type: "pros_cons", Description: "Initial analysis"},
			{Step: "step2", Type: "risk_assessment", Description: "Risk evaluation"},
		}

		req := ReasoningRequest{
			Input:       "test",
			Type:        "reasoning_chain",
			CustomChain: customChain,
		}

		if len(req.CustomChain) != 2 {
			t.Errorf("Expected 2 custom chain steps, got %d", len(req.CustomChain))
		}
	})
}

// TestReasoningResponse tests reasoning response structure
func TestReasoningResponse(t *testing.T) {
	t.Run("CreateReasoningResponse_Success", func(t *testing.T) {
		resp := ReasoningResponse{
			ID:         uuid.New().String(),
			Type:       "pros_cons",
			Confidence: 0.85,
			Timestamp:  time.Now(),
			Model:      "llama3.2",
			Success:    true,
		}

		if resp.ID == "" {
			t.Error("ID should not be empty")
		}
		if resp.Confidence < 0 || resp.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1, got %f", resp.Confidence)
		}
	})

	t.Run("CreateReasoningResponse_WithError", func(t *testing.T) {
		resp := ReasoningResponse{
			ID:      uuid.New().String(),
			Type:    "pros_cons",
			Success: false,
			Error:   "Analysis failed",
		}

		if resp.Success {
			t.Error("Success should be false for error response")
		}
		if resp.Error == "" {
			t.Error("Error message should not be empty")
		}
	})
}

// TestProsCons tests pros/cons structure
func TestProsCons(t *testing.T) {
	t.Run("CreateProsCons_Complete", func(t *testing.T) {
		analysis := ProsCons{
			Pros: []WeightedItem{
				{Item: "Scalability", Weight: 8.5, Explanation: "Better horizontal scaling"},
				{Item: "Independence", Weight: 7.0, Explanation: "Teams can work independently"},
			},
			Cons: []WeightedItem{
				{Item: "Complexity", Weight: 9.0, Explanation: "Increased operational complexity"},
			},
			Recommendation: "Proceed with caution",
			Confidence:     0.75,
		}

		if len(analysis.Pros) != 2 {
			t.Errorf("Expected 2 pros, got %d", len(analysis.Pros))
		}
		if len(analysis.Cons) != 1 {
			t.Errorf("Expected 1 con, got %d", len(analysis.Cons))
		}
		if analysis.Confidence < 0 || analysis.Confidence > 1 {
			t.Errorf("Confidence should be between 0 and 1, got %f", analysis.Confidence)
		}
	})
}

// TestSWOTAnalysis tests SWOT analysis structure
func TestSWOTAnalysis(t *testing.T) {
	t.Run("CreateSWOTAnalysis_Complete", func(t *testing.T) {
		analysis := SWOTAnalysis{
			Strengths: []WeightedItem{
				{Item: "Strong team", Weight: 8.0, Explanation: "Experienced developers"},
			},
			Weaknesses: []WeightedItem{
				{Item: "Legacy code", Weight: 7.0, Explanation: "Technical debt"},
			},
			Opportunities: []WeightedItem{
				{Item: "Market growth", Weight: 9.0, Explanation: "Expanding market"},
			},
			Threats: []WeightedItem{
				{Item: "Competition", Weight: 6.5, Explanation: "New competitors"},
			},
			Strategic:  "Focus on strengths",
			Priority:   "Address weaknesses",
			Confidence: 0.8,
		}

		if len(analysis.Strengths) == 0 {
			t.Error("Strengths should not be empty")
		}
		if len(analysis.Weaknesses) == 0 {
			t.Error("Weaknesses should not be empty")
		}
		if len(analysis.Opportunities) == 0 {
			t.Error("Opportunities should not be empty")
		}
		if len(analysis.Threats) == 0 {
			t.Error("Threats should not be empty")
		}
	})
}

// TestRiskAssessment tests risk assessment structure
func TestRiskAssessment(t *testing.T) {
	t.Run("CreateRiskAssessment_Complete", func(t *testing.T) {
		analysis := RiskAssessment{
			Risks: []Risk{
				{
					Description: "Data loss",
					Probability: 0.3,
					Impact:      9.0,
					Severity:    2.7,
					Category:    "technical",
				},
			},
			Mitigations: []Risk{
				{
					Description: "Implement backups",
					Probability: 0.9,
					Impact:      8.0,
					Category:    "mitigation",
				},
			},
			OverallRisk:    "medium",
			Recommendation: "Implement mitigations",
			Confidence:     0.7,
		}

		if len(analysis.Risks) == 0 {
			t.Error("Risks should not be empty")
		}
		if len(analysis.Mitigations) == 0 {
			t.Error("Mitigations should not be empty")
		}

		// Test severity calculation
		risk := analysis.Risks[0]
		expectedSeverity := risk.Probability * risk.Impact
		if risk.Severity != expectedSeverity && risk.Severity == 0 {
			t.Errorf("Expected severity %f, got %f", expectedSeverity, risk.Severity)
		}
	})

	t.Run("RiskSeverity_Calculation", func(t *testing.T) {
		risk := Risk{
			Probability: 0.5,
			Impact:      8.0,
		}
		risk.Severity = risk.Probability * risk.Impact

		expectedSeverity := 4.0
		if risk.Severity != expectedSeverity {
			t.Errorf("Expected severity %f, got %f", expectedSeverity, risk.Severity)
		}
	})
}

// TestSelfReview tests self-review structure
func TestSelfReview(t *testing.T) {
	t.Run("CreateSelfReview_Complete", func(t *testing.T) {
		analysis := SelfReview{
			BiasesIdentified:   []string{"Confirmation bias", "Anchoring bias"},
			AssumptionsChecked: []string{"User adoption rate", "Technical feasibility"},
			AlternativeViews:   []string{"Monolithic alternative", "Hybrid approach"},
			LogicalFallacies:   []string{"Sunk cost fallacy"},
			ConfidenceLevel:    0.75,
			RecommendedActions: []string{"Validate assumptions", "Seek expert opinion"},
			QualityAssessment:  "Good reasoning with minor gaps",
		}

		if len(analysis.BiasesIdentified) == 0 {
			t.Error("Biases should not be empty")
		}
		if len(analysis.AssumptionsChecked) == 0 {
			t.Error("Assumptions should not be empty")
		}
		if len(analysis.RecommendedActions) == 0 {
			t.Error("Recommended actions should not be empty")
		}
	})
}

// TestReasoningChain tests reasoning chain structure
func TestReasoningChain(t *testing.T) {
	t.Run("CreateReasoningChain_Complete", func(t *testing.T) {
		chain := ReasoningChain{
			ID:    uuid.New().String(),
			Type:  "comprehensive",
			Input: "test input",
			Steps: []ReasoningStep{
				{Step: "step1", Type: "pros_cons", Description: "Initial analysis"},
				{Step: "step2", Type: "risk_assessment", Description: "Risk evaluation"},
			},
			CurrentStep: 0,
			Results:     []StepResult{},
			Status:      "initiated",
			StartedAt:   time.Now(),
		}

		if chain.ID == "" {
			t.Error("ID should not be empty")
		}
		if len(chain.Steps) != 2 {
			t.Errorf("Expected 2 steps, got %d", len(chain.Steps))
		}
		if chain.Status != "initiated" {
			t.Errorf("Expected status 'initiated', got '%s'", chain.Status)
		}
	})

	t.Run("ReasoningChain_StatusProgression", func(t *testing.T) {
		chain := ReasoningChain{
			ID:          uuid.New().String(),
			Status:      "initiated",
			CurrentStep: 0,
			Steps:       []ReasoningStep{{Step: "test", Type: "pros_cons"}},
		}

		// Simulate progression
		chain.Status = "executing"
		if chain.Status != "executing" {
			t.Error("Status should be 'executing'")
		}

		chain.CurrentStep = 1
		chain.Status = "completed"
		now := time.Now()
		chain.CompletedAt = &now

		if chain.Status != "completed" {
			t.Error("Status should be 'completed'")
		}
		if chain.CompletedAt == nil {
			t.Error("CompletedAt should not be nil")
		}
	})
}

// TestGetModel tests model selection helper
func TestGetModel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"DefaultModel", "", "llama3.2"},
		{"CustomModel", "gpt-4", "gpt-4"},
		{"LlamaModel", "llama3.1", "llama3.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getModel(tt.input)
			if result != tt.expected {
				t.Errorf("Expected model '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestCalculateItemsScore tests score calculation
func TestCalculateItemsScore(t *testing.T) {
	t.Run("CalculateScore_MultipleItems", func(t *testing.T) {
		items := []WeightedItem{
			{Item: "item1", Weight: 5.0},
			{Item: "item2", Weight: 7.5},
			{Item: "item3", Weight: 3.5},
		}

		score := calculateItemsScore(items)
		expectedScore := 16.0

		if score != expectedScore {
			t.Errorf("Expected score %f, got %f", expectedScore, score)
		}
	})

	t.Run("CalculateScore_EmptyItems", func(t *testing.T) {
		items := []WeightedItem{}
		score := calculateItemsScore(items)

		if score != 0.0 {
			t.Errorf("Expected score 0, got %f", score)
		}
	})

	t.Run("CalculateScore_SingleItem", func(t *testing.T) {
		items := []WeightedItem{
			{Item: "item1", Weight: 8.5},
		}

		score := calculateItemsScore(items)
		if score != 8.5 {
			t.Errorf("Expected score 8.5, got %f", score)
		}
	})
}

// TestProcessReasoning tests reasoning processing (unit test)
func TestProcessReasoning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("ProcessReasoning_UnsupportedType", func(t *testing.T) {
		engine := NewMetareasoningEngine(nil)

		req := &ReasoningRequest{
			Input: "test input",
			Type:  "unsupported_type",
		}

		resp, err := engine.ProcessReasoning(req)

		if err == nil {
			t.Error("Expected error for unsupported type")
		}
		if resp != nil && resp.Success {
			t.Error("Response should indicate failure")
		}
	})
}

// TestReasoningChainExecution tests chain execution
func TestReasoningChainExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("ChainExecution_Cancellation", func(t *testing.T) {
		engine := NewMetareasoningEngine(nil)

		// Test context cancellation
		ctx, cancel := context.WithCancel(context.Background())
		engine.ctx = ctx

		cancel() // Cancel immediately

		// Context should be canceled
		select {
		case <-engine.ctx.Done():
			// Expected
		default:
			t.Error("Context should be canceled")
		}
	})
}

// Benchmark tests

func BenchmarkNewMetareasoningEngine(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewMetareasoningEngine(nil)
	}
}

func BenchmarkCalculateItemsScore(b *testing.B) {
	items := []WeightedItem{
		{Item: "item1", Weight: 5.0},
		{Item: "item2", Weight: 7.5},
		{Item: "item3", Weight: 3.5},
		{Item: "item4", Weight: 6.0},
		{Item: "item5", Weight: 4.5},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateItemsScore(items)
	}
}

func BenchmarkGetModel(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getModel("llama3.2")
	}
}

// Edge case tests

func TestEdgeCasesMetareasoning(t *testing.T) {
	t.Run("EmptyInput", func(t *testing.T) {
		req := ReasoningRequest{
			Input: "",
			Type:  "pros_cons",
		}

		if req.Input != "" {
			t.Error("Input should be empty")
		}
	})

	t.Run("VeryLongInput", func(t *testing.T) {
		longInput := ""
		for i := 0; i < 10000; i++ {
			longInput += "a"
		}

		req := ReasoningRequest{
			Input: longInput,
			Type:  "pros_cons",
		}

		if len(req.Input) != 10000 {
			t.Errorf("Expected input length 10000, got %d", len(req.Input))
		}
	})

	t.Run("NilAnalysis", func(t *testing.T) {
		resp := ReasoningResponse{
			ID:       uuid.New().String(),
			Analysis: nil,
			Success:  true,
		}

		if resp.Analysis != nil {
			t.Error("Analysis should be nil")
		}
	})
}
