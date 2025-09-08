package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// RICE score calculation handler
func (app *App) riceScoreHandler(w http.ResponseWriter, r *http.Request) {
	var features []Feature
	if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate RICE scores for all features
	for i := range features {
		features[i].Score = app.calculateRICE(&features[i])
		features[i].Priority = app.determinePriority(features[i].Score)
	}

	// Sort by score
	sortedFeatures := app.sortFeaturesByRICE(features)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prioritized_features": sortedFeatures,
		"total": len(sortedFeatures),
	})
}

// Prioritize features handler
func (app *App) prioritizeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Features []Feature `json:"features"`
		Strategy string    `json:"strategy"` // rice, value, effort
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Apply prioritization strategy
	var prioritized []Feature
	switch req.Strategy {
	case "value":
		prioritized = app.prioritizeByValue(req.Features)
	case "effort":
		prioritized = app.prioritizeByEffort(req.Features)
	default:
		prioritized = app.sortFeaturesByRICE(req.Features)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prioritized)
}

// Roadmap handler
func (app *App) roadmapHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		roadmap, err := app.fetchCurrentRoadmap()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(roadmap)
		
	case "POST":
		var roadmap Roadmap
		if err := json.NewDecoder(r.Body).Decode(&roadmap); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		
		roadmap.CreatedAt = time.Now()
		roadmap.Version = 1
		
		if err := app.storeRoadmap(&roadmap); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(roadmap)
		
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Generate roadmap handler
func (app *App) generateRoadmapHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FeatureIDs []string  `json:"feature_ids"`
		StartDate  time.Time `json:"start_date"`
		Duration   int       `json:"duration_months"`
		Capacity   int       `json:"team_capacity"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Fetch features
	features, err := app.fetchFeaturesByIDs(req.FeatureIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate roadmap
	roadmap := app.generateRoadmap(features, req.StartDate, req.Duration, req.Capacity)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roadmap)
}

// Sprint planning handler
func (app *App) sprintPlanHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamSize int `json:"team_size"`
		Velocity int `json:"velocity"`
		Duration int `json:"duration_weeks"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate capacity
	capacity := req.TeamSize * req.Velocity * req.Duration

	// Get available features
	features, err := app.fetchAvailableFeatures()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Optimize sprint
	sprintPlan, err := app.optimizeSprint(capacity, features)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sprintPlan)
}

// Current sprint handler
func (app *App) currentSprintHandler(w http.ResponseWriter, r *http.Request) {
	sprint, err := app.fetchCurrentSprint()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sprint)
}

// Market analysis handler
func (app *App) marketAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProductName string `json:"product_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	analysis, err := app.analyzeMarket(req.ProductName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// Competitor analysis handler
func (app *App) competitorAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompetitorName string `json:"competitor_name"`
		Depth          string `json:"depth"` // quick, standard, deep
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	analysis, err := app.analyzeCompetitor(req.CompetitorName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// Feedback analysis handler
func (app *App) feedbackAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FeedbackItems []FeedbackItem `json:"feedback_items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	analysis, err := app.analyzeFeedback(req.FeedbackItems)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// ROI calculation handler
func (app *App) roiCalculationHandler(w http.ResponseWriter, r *http.Request) {
	var feature Feature
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	calculation, err := app.calculateROI(&feature)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calculation)
}

// Decision analysis handler
func (app *App) decisionAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	var decision Decision
	if err := json.NewDecoder(r.Body).Decode(&decision); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	analysis, err := app.analyzeDecision(&decision)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// Dashboard handler
func (app *App) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get metrics
	metrics, err := app.fetchDashboardMetrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get recent features
	recentFeatures, _ := app.fetchRecentFeatures(5)

	// Get current sprint
	currentSprint, _ := app.fetchCurrentSprint()

	// Get roadmap
	roadmap, _ := app.fetchCurrentRoadmap()

	dashboard := Dashboard{
		Metrics:        metrics,
		RecentFeatures: recentFeatures,
		CurrentSprint:  currentSprint,
		Roadmap:        roadmap,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// Helper methods for prioritization
func (app *App) calculateRICE(feature *Feature) float64 {
	if feature.Effort == 0 {
		feature.Effort = 1 // Prevent division by zero
	}
	return (float64(feature.Reach) * float64(feature.Impact) * feature.Confidence) / float64(feature.Effort)
}

func (app *App) determinePriority(score float64) string {
	if score > 100 {
		return "CRITICAL"
	} else if score > 50 {
		return "HIGH"
	} else if score > 20 {
		return "MEDIUM"
	}
	return "LOW"
}

func (app *App) prioritizeByValue(features []Feature) []Feature {
	// Sort by impact * reach
	for i := range features {
		features[i].Score = float64(features[i].Impact * features[i].Reach)
	}
	return app.sortFeaturesByScore(features)
}

func (app *App) prioritizeByEffort(features []Feature) []Feature {
	// Sort by effort (ascending)
	for i := 0; i < len(features)-1; i++ {
		for j := 0; j < len(features)-i-1; j++ {
			if features[j].Effort > features[j+1].Effort {
				features[j], features[j+1] = features[j+1], features[j]
			}
		}
	}
	return features
}

func (app *App) sortFeaturesByScore(features []Feature) []Feature {
	// Sort by score (descending)
	for i := 0; i < len(features)-1; i++ {
		for j := 0; j < len(features)-i-1; j++ {
			if features[j].Score < features[j+1].Score {
				features[j], features[j+1] = features[j+1], features[j]
			}
		}
	}
	return features
}

func (app *App) generateRoadmap(features []Feature, startDate time.Time, durationMonths, capacity int) *Roadmap {
	roadmap := &Roadmap{
		ID:        fmt.Sprintf("rm-%d", time.Now().Unix()),
		Name:      fmt.Sprintf("Product Roadmap %s", startDate.Format("Jan 2006")),
		StartDate: startDate,
		EndDate:   startDate.AddDate(0, durationMonths, 0),
		Features:  []string{},
		CreatedAt: time.Now(),
		Version:   1,
	}

	// Distribute features across milestones
	monthlyCapacity := capacity / durationMonths
	currentMonth := 0
	currentCapacity := 0

	for _, feature := range features {
		if currentCapacity+feature.Effort > monthlyCapacity {
			currentMonth++
			currentCapacity = 0
			if currentMonth >= durationMonths {
				break
			}
		}
		
		roadmap.Features = append(roadmap.Features, feature.ID)
		currentCapacity += feature.Effort
		
		// Create milestone every quarter
		if currentMonth%3 == 0 && currentMonth > 0 {
			milestone := Milestone{
				ID:          fmt.Sprintf("ms-%d-%d", time.Now().Unix(), currentMonth),
				Name:        fmt.Sprintf("Q%d Milestone", currentMonth/3),
				Date:        startDate.AddDate(0, currentMonth, 0),
				Description: fmt.Sprintf("Quarter %d deliverables", currentMonth/3),
				Features:    roadmap.Features[len(roadmap.Features)-10:], // Last 10 features
			}
			roadmap.Milestones = append(roadmap.Milestones, milestone)
		}
	}

	return roadmap
}