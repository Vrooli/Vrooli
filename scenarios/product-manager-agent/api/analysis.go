package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Market research functionality
func (app *App) analyzeMarket(productName string) (*MarketAnalysis, error) {
	// Use Ollama to analyze market trends
	prompt := fmt.Sprintf(`Analyze the market for %s. Provide:
	1. Market size and growth rate
	2. Key competitors
	3. Target demographics
	4. Market opportunities
	5. Potential challenges
	Format as JSON.`, productName)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	analysis := &MarketAnalysis{
		ProductName: productName,
		Timestamp:   time.Now(),
	}

	if response, ok := result["response"].(string); ok {
		// Parse the AI response
		var aiAnalysis map[string]interface{}
		if err := json.Unmarshal([]byte(response), &aiAnalysis); err == nil {
			analysis.MarketSize = getString(aiAnalysis, "market_size")
			analysis.GrowthRate = getString(aiAnalysis, "growth_rate")
			analysis.Competitors = getStringSlice(aiAnalysis, "competitors")
			analysis.Demographics = getString(aiAnalysis, "demographics")
			analysis.Opportunities = getStringSlice(aiAnalysis, "opportunities")
			analysis.Challenges = getStringSlice(aiAnalysis, "challenges")
		}
	}

	// Store in database
	app.storeMarketAnalysis(analysis)

	return analysis, nil
}

// Competitor analysis
func (app *App) analyzeCompetitor(competitorName string) (*CompetitorAnalysis, error) {
	prompt := fmt.Sprintf(`Analyze the competitor %s. Provide:
	1. Key features of their product
	2. Pricing strategy
	3. Target market
	4. Strengths
	5. Weaknesses
	6. Market share estimate
	Format as JSON.`, competitorName)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	analysis := &CompetitorAnalysis{
		CompetitorName: competitorName,
		AnalyzedAt:     time.Now(),
	}

	if response, ok := result["response"].(string); ok {
		var aiAnalysis map[string]interface{}
		if err := json.Unmarshal([]byte(response), &aiAnalysis); err == nil {
			analysis.Features = getStringSlice(aiAnalysis, "features")
			analysis.Pricing = getString(aiAnalysis, "pricing")
			analysis.TargetMarket = getString(aiAnalysis, "target_market")
			analysis.Strengths = getStringSlice(aiAnalysis, "strengths")
			analysis.Weaknesses = getStringSlice(aiAnalysis, "weaknesses")
			analysis.MarketShare = getString(aiAnalysis, "market_share")
		}
	}

	// Store in database
	app.storeCompetitorAnalysis(analysis)

	return analysis, nil
}

// User feedback analysis
func (app *App) analyzeFeedback(feedbackItems []FeedbackItem) (*FeedbackAnalysis, error) {
	// Aggregate feedback for analysis
	feedbackText := ""
	for _, item := range feedbackItems {
		feedbackText += item.Content + "\n"
	}

	// Sentiment analysis
	sentimentPrompt := fmt.Sprintf(`Analyze the sentiment and key themes in this user feedback:
	%s
	
	Provide:
	1. Overall sentiment (positive/negative/neutral)
	2. Sentiment score (0-100)
	3. Key themes
	4. Feature requests
	5. Pain points
	Format as JSON.`, feedbackText)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": sentimentPrompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	analysis := &FeedbackAnalysis{
		TotalItems: len(feedbackItems),
		AnalyzedAt: time.Now(),
	}

	if response, ok := result["response"].(string); ok {
		var aiAnalysis map[string]interface{}
		if err := json.Unmarshal([]byte(response), &aiAnalysis); err == nil {
			analysis.Sentiment = getString(aiAnalysis, "sentiment")
			analysis.SentimentScore = getFloat(aiAnalysis, "sentiment_score")
			analysis.KeyThemes = getStringSlice(aiAnalysis, "themes")
			analysis.FeatureRequests = getStringSlice(aiAnalysis, "feature_requests")
			analysis.PainPoints = getStringSlice(aiAnalysis, "pain_points")
		}
	}

	return analysis, nil
}

// ROI calculation for features
func (app *App) calculateROI(feature *Feature) (*ROICalculation, error) {
	// Estimate revenue impact
	revenueImpact := float64(feature.Reach) * float64(feature.Impact) * 10 // Simple formula

	// Estimate cost
	costEstimate := float64(feature.Effort) * 1000 // $1000 per effort point

	// Calculate ROI
	roi := ((revenueImpact - costEstimate) / costEstimate) * 100

	// Payback period in months
	paybackPeriod := costEstimate / (revenueImpact / 12)

	calculation := &ROICalculation{
		FeatureID:      feature.ID,
		RevenueImpact:  revenueImpact,
		CostEstimate:   costEstimate,
		ROI:            roi,
		PaybackPeriod:  paybackPeriod,
		CalculatedAt:   time.Now(),
		Assumptions:    []string{"$10 per user impact", "$1000 per effort point"},
	}

	// Store calculation
	app.storeROICalculation(calculation)

	return calculation, nil
}

// Decision tree analysis
func (app *App) analyzeDecision(decision *Decision) (*DecisionAnalysis, error) {
	// Build context for AI analysis
	optionsText := ""
	for _, opt := range decision.Options {
		optionsText += fmt.Sprintf("- %s: %s\n", opt.Name, opt.Description)
	}

	prompt := fmt.Sprintf(`Analyze this product decision:
	Decision: %s
	Options:
	%s
	
	For each option, provide:
	1. Pros and cons
	2. Risk level (low/medium/high)
	3. Implementation complexity
	4. Estimated timeline
	5. Success probability
	6. Recommendation score (0-100)
	Format as JSON with an array of options.`, decision.Title, optionsText)

	payload := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"format": "json",
	}

	jsonData, _ := json.Marshal(payload)
	resp, err := http.Post(app.OllamaURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	analysis := &DecisionAnalysis{
		DecisionID: decision.ID,
		AnalyzedAt: time.Now(),
	}

	if response, ok := result["response"].(string); ok {
		var aiAnalysis map[string]interface{}
		if err := json.Unmarshal([]byte(response), &aiAnalysis); err == nil {
			if options, ok := aiAnalysis["options"].([]interface{}); ok {
				for i, opt := range options {
					if i < len(decision.Options) {
						if optMap, ok := opt.(map[string]interface{}); ok {
							decision.Options[i].Pros = getStringSlice(optMap, "pros")
							decision.Options[i].Cons = getStringSlice(optMap, "cons")
							decision.Options[i].RiskLevel = getString(optMap, "risk_level")
							decision.Options[i].Complexity = getString(optMap, "complexity")
							decision.Options[i].Timeline = getString(optMap, "timeline")
							decision.Options[i].SuccessProbability = getFloat(optMap, "success_probability")
							decision.Options[i].Score = getFloat(optMap, "recommendation_score")
						}
					}
				}
			}
		}
	}

	analysis.Options = decision.Options

	return analysis, nil
}

// Sprint planning optimization
func (app *App) optimizeSprint(capacity int, features []Feature) (*SprintPlan, error) {
	// Sort features by RICE score
	sortedFeatures := app.sortFeaturesByRICE(features)

	plan := &SprintPlan{
		Capacity:       capacity,
		PlannedAt:      time.Now(),
		Features:       []Feature{},
		TotalEffort:    0,
		EstimatedValue: 0,
	}

	// Greedy algorithm to fill sprint
	for _, feature := range sortedFeatures {
		if plan.TotalEffort+feature.Effort <= capacity {
			plan.Features = append(plan.Features, feature)
			plan.TotalEffort += feature.Effort
			plan.EstimatedValue += feature.Score * 1000 // Convert score to value
		}
	}

	// Calculate velocity and risk
	plan.Velocity = float64(plan.TotalEffort) / float64(len(plan.Features))
	plan.RiskLevel = app.calculateSprintRisk(plan.Features)

	// Store sprint plan
	app.storeSprintPlan(plan)

	return plan, nil
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getStringSlice(m map[string]interface{}, key string) []string {
	result := []string{}
	if v, ok := m[key].([]interface{}); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
	}
	return result
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func (app *App) calculateSprintRisk(features []Feature) string {
	avgConfidence := 0.0
	for _, f := range features {
		avgConfidence += f.Confidence
	}
	avgConfidence /= float64(len(features))

	if avgConfidence > 0.8 {
		return "low"
	} else if avgConfidence > 0.6 {
		return "medium"
	}
	return "high"
}

func (app *App) sortFeaturesByRICE(features []Feature) []Feature {
	// Calculate RICE scores
	for i := range features {
		features[i].Score = app.calculateRICE(&features[i])
	}

	// Sort by score (bubble sort for simplicity)
	for i := 0; i < len(features)-1; i++ {
		for j := 0; j < len(features)-i-1; j++ {
			if features[j].Score < features[j+1].Score {
				features[j], features[j+1] = features[j+1], features[j]
			}
		}
	}

	return features
}