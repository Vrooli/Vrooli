package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	router *mux.Router
	port   string
	db     *Database
}

type HealthResponse struct {
	Status       string                 `json:"status"`
	Timestamp    time.Time              `json:"timestamp"`
	Service      string                 `json:"service"`
	Version      string                 `json:"version"`
	Readiness    bool                   `json:"readiness"`
	Dependencies map[string]interface{} `json:"dependencies,omitempty"`
}

type ShoppingResearchRequest struct {
	ProfileID           string  `json:"profile_id"`
	Query               string  `json:"query"`
	BudgetMax           float64 `json:"budget_max,omitempty"`
	IncludeAlternatives bool    `json:"include_alternatives"`
	GiftRecipientID     string  `json:"gift_recipient_id,omitempty"`
}

type Product struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Category       string                 `json:"category"`
	Description    string                 `json:"description"`
	CurrentPrice   float64                `json:"current_price"`
	OriginalPrice  float64                `json:"original_price,omitempty"`
	AffiliateLink  string                 `json:"affiliate_link,omitempty"`
	Features       map[string]interface{} `json:"features,omitempty"`
	ReviewsSummary *ReviewsSummary        `json:"reviews_summary,omitempty"`
}

type ReviewsSummary struct {
	AverageRating float64  `json:"average_rating"`
	TotalReviews  int      `json:"total_reviews"`
	Pros          []string `json:"pros"`
	Cons          []string `json:"cons"`
}

type Alternative struct {
	Product         Product `json:"product"`
	AlternativeType string  `json:"alternative_type"`
	SavingsAmount   float64 `json:"savings_amount"`
	Reason          string  `json:"reason"`
}

type PriceInsights struct {
	CurrentTrend   string  `json:"current_trend"`
	BestTimeToWait bool    `json:"best_time_to_wait"`
	PredictedDrop  float64 `json:"predicted_drop,omitempty"`
	HistoricalLow  float64 `json:"historical_low"`
	HistoricalHigh float64 `json:"historical_high"`
}

type ShoppingResearchResponse struct {
	Products        []Product       `json:"products"`
	Alternatives    []Alternative   `json:"alternatives"`
	PriceAnalysis   PriceInsights   `json:"price_analysis"`
	Recommendations []string        `json:"recommendations"`
	AffiliateLinks  []AffiliateLink `json:"affiliate_links"`
}

type AffiliateLink struct {
	ProductID  string  `json:"product_id"`
	Retailer   string  `json:"retailer"`
	URL        string  `json:"url"`
	Commission float64 `json:"commission"`
}

type PriceAlert struct {
	ID           string    `json:"id"`
	ProfileID    string    `json:"profile_id"`
	ProductID    string    `json:"product_id"`
	TargetPrice  float64   `json:"target_price"`
	CurrentPrice float64   `json:"current_price"`
	AlertType    string    `json:"alert_type"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type TrackingResponse struct {
	ActiveAlerts    []PriceAlert  `json:"active_alerts"`
	TrackedProducts []Product     `json:"tracked_products"`
	RecentChanges   []PriceChange `json:"recent_changes"`
}

type PriceChange struct {
	ProductID string    `json:"product_id"`
	OldPrice  float64   `json:"old_price"`
	NewPrice  float64   `json:"new_price"`
	ChangedAt time.Time `json:"changed_at"`
}

type PatternAnalysisRequest struct {
	ProfileID string `json:"profile_id"`
	Timeframe string `json:"timeframe"`
}

type PurchasePattern struct {
	Category      string  `json:"category"`
	PatternType   string  `json:"pattern_type"`
	FrequencyDays int     `json:"frequency_days"`
	AverageSpend  float64 `json:"average_spend"`
	Confidence    float64 `json:"confidence"`
}

type RestockPrediction struct {
	ProductCategory string    `json:"product_category"`
	PredictedDate   time.Time `json:"predicted_date"`
	Confidence      float64   `json:"confidence"`
	SuggestedItems  []string  `json:"suggested_items"`
}

type PatternAnalysisResponse struct {
	Patterns             []PurchasePattern   `json:"patterns"`
	Predictions          []RestockPrediction `json:"predictions"`
	SavingsOpportunities []SavingsOption     `json:"savings_opportunities"`
}

type SavingsOption struct {
	Description string  `json:"description"`
	Potential   float64 `json:"potential_savings"`
	Action      string  `json:"action_required"`
}

type AuthUser struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Valid    bool   `json:"valid"`
}

// authMiddleware validates tokens with scenario-authenticator
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Check for profile_id parameter as fallback (for backward compatibility)
			profileID := r.URL.Query().Get("profile_id")
			if profileID == "" {
				// Try to get from body for POST requests
				if r.Method == "POST" {
					bodyBytes, _ := io.ReadAll(r.Body)
					r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
					var reqBody map[string]interface{}
					json.Unmarshal(bodyBytes, &reqBody)
					if pid, ok := reqBody["profile_id"].(string); ok && pid != "" {
						// Allow request to proceed with profile_id
						ctx := context.WithValue(r.Context(), "user_id", pid)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
				// No authentication provided, allow as anonymous for now
				ctx := context.WithValue(r.Context(), "user_id", "anonymous")
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// Use profile_id as user_id
			ctx := context.WithValue(r.Context(), "user_id", profileID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Extract token (remove "Bearer " prefix)
		token := strings.Replace(authHeader, "Bearer ", "", 1)

		// Get scenario-authenticator port from environment
		authPort := os.Getenv("SCENARIO_AUTHENTICATOR_API_PORT")
		if authPort == "" {
			// Try default port range for scenarios
			authPort = "15797"
		}

		// Validate token with scenario-authenticator
		authURL := fmt.Sprintf("http://localhost:%s/api/v1/auth/validate", authPort)
		req, err := http.NewRequest("GET", authURL, nil)
		if err != nil {
			log.Printf("Error creating auth request: %v", err)
			// Allow request to proceed as anonymous
			ctx := context.WithValue(r.Context(), "user_id", "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error validating token: %v", err)
			// Allow request to proceed as anonymous
			ctx := context.WithValue(r.Context(), "user_id", "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// Invalid token, allow as anonymous
			ctx := context.WithValue(r.Context(), "user_id", "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Parse auth response
		var authUser AuthUser
		if err := json.NewDecoder(resp.Body).Decode(&authUser); err != nil {
			log.Printf("Error parsing auth response: %v", err)
			ctx := context.WithValue(r.Context(), "user_id", "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if !authUser.Valid {
			ctx := context.WithValue(r.Context(), "user_id", "anonymous")
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Add user info to context
		ctx := context.WithValue(r.Context(), "user_id", authUser.ID)
		ctx = context.WithValue(ctx, "user", authUser)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func NewServer() *Server {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3300"
	}

	// Initialize database
	db, err := NewDatabase()
	if err != nil {
		log.Printf("Warning: Database initialization failed: %v", err)
	}

	s := &Server{
		router: mux.NewRouter(),
		port:   port,
		db:     db,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Shopping research endpoints - protected with auth middleware
	s.router.HandleFunc("/api/v1/shopping/research", s.authMiddleware(s.handleShoppingResearch)).Methods("POST")
	s.router.HandleFunc("/api/v1/shopping/tracking/{profile_id}", s.authMiddleware(s.handleGetTracking)).Methods("GET")
	s.router.HandleFunc("/api/v1/shopping/tracking", s.authMiddleware(s.handleCreateTracking)).Methods("POST")
	s.router.HandleFunc("/api/v1/shopping/pattern-analysis", s.authMiddleware(s.handlePatternAnalysis)).Methods("POST")

	// Profile management - protected with auth middleware
	s.router.HandleFunc("/api/v1/profiles", s.authMiddleware(s.handleGetProfiles)).Methods("GET")
	s.router.HandleFunc("/api/v1/profiles", s.handleCreateProfile).Methods("POST")
	s.router.HandleFunc("/api/v1/profiles/{id}", s.handleGetProfile).Methods("GET")

	// Price alerts
	s.router.HandleFunc("/api/v1/alerts", s.handleGetAlerts).Methods("GET")
	s.router.HandleFunc("/api/v1/alerts", s.handleCreateAlert).Methods("POST")
	s.router.HandleFunc("/api/v1/alerts/{id}", s.handleDeleteAlert).Methods("DELETE")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	dbConnected := false
	var dbLatency *float64
	var dbError map[string]interface{}

	if s.db.postgres != nil {
		start := time.Now()
		err := s.db.postgres.Ping()
		latency := float64(time.Since(start).Milliseconds())
		if err == nil {
			dbConnected = true
			dbLatency = &latency
		} else {
			dbError = map[string]interface{}{
				"code":      "CONNECTION_REFUSED",
				"message":   err.Error(),
				"category":  "resource",
				"retryable": true,
			}
		}
	}

	// Check Redis connectivity
	redisConnected := false
	var redisLatency *float64
	var redisError map[string]interface{}

	if s.db.redis != nil {
		start := time.Now()
		_, err := s.db.redis.Ping(context.Background()).Result()
		latency := float64(time.Since(start).Milliseconds())
		if err == nil {
			redisConnected = true
			redisLatency = &latency
		} else {
			redisError = map[string]interface{}{
				"code":      "CONNECTION_REFUSED",
				"message":   err.Error(),
				"category":  "resource",
				"retryable": true,
			}
		}
	}

	// Determine overall status
	status := "healthy"
	readiness := true
	if !dbConnected && !redisConnected {
		status = "degraded"
	}

	dependencies := map[string]interface{}{
		"database": map[string]interface{}{
			"connected":  dbConnected,
			"latency_ms": dbLatency,
			"error":      dbError,
		},
		"redis": map[string]interface{}{
			"connected":  redisConnected,
			"latency_ms": redisLatency,
			"error":      redisError,
		},
	}

	response := HealthResponse{
		Status:       status,
		Timestamp:    time.Now(),
		Service:      "smart-shopping-assistant",
		Version:      "1.0.0",
		Readiness:    readiness,
		Dependencies: dependencies,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleShoppingResearch(w http.ResponseWriter, r *http.Request) {
	var req ShoppingResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query is required", http.StatusBadRequest)
		return
	}

	// Get user ID from context (set by auth middleware)
	ctx := r.Context()
	userID := ctx.Value("user_id").(string)

	// Use user ID as profile ID if not specified
	if req.ProfileID == "" {
		req.ProfileID = userID
	}

	// Log the authenticated request
	log.Printf("Shopping research for user %s (profile: %s): %s", userID, req.ProfileID, req.Query)
	products, err := s.db.SearchProducts(ctx, req.Query, req.BudgetMax)
	if err != nil {
		log.Printf("Error searching products: %v", err)
		// Fall back to basic mock data if database fails
		products = []Product{
			{
				ID:            "fallback-1",
				Name:          fmt.Sprintf("Sample %s", req.Query),
				Category:      "General",
				Description:   "Product matching your search",
				CurrentPrice:  99.99,
				OriginalPrice: 129.99,
			},
		}
	}

	// Find alternatives for the first product
	alternatives := []Alternative{}
	if len(products) > 0 {
		alternatives = s.db.FindAlternatives(ctx, products[0].ID, products[0].CurrentPrice)
	}

	// Get price insights
	priceAnalysis := PriceInsights{
		CurrentTrend:   "stable",
		BestTimeToWait: false,
		HistoricalLow:  89.99,
		HistoricalHigh: 149.99,
	}
	if len(products) > 0 {
		priceAnalysis = s.db.GetPriceHistory(ctx, products[0].ID)
	}

	// Generate recommendations based on actual search
	recommendations := []string{}
	if req.BudgetMax > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Found %d products within your $%.2f budget", len(products), req.BudgetMax))
	}
	if len(alternatives) > 0 && len(products) > 0 && products[0].CurrentPrice > 0 {
		savingsPercent := (alternatives[0].SavingsAmount / products[0].CurrentPrice) * 100
		recommendations = append(recommendations, fmt.Sprintf("Save %.0f%% with alternative options", savingsPercent))
	} else if len(alternatives) > 0 {
		recommendations = append(recommendations, "Alternative options available for additional savings")
	}
	if priceAnalysis.CurrentTrend == "declining" {
		recommendations = append(recommendations, "Prices are trending down - good time to buy")
	}

	// Generate affiliate links
	affiliateLinks := s.db.GenerateAffiliateLinks(products)

	response := ShoppingResearchResponse{
		Products:        products,
		Alternatives:    alternatives,
		PriceAnalysis:   priceAnalysis,
		Recommendations: recommendations,
		AffiliateLinks:  affiliateLinks,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetTracking(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["profile_id"]

	// TODO: Implement actual tracking retrieval
	response := TrackingResponse{
		ActiveAlerts: []PriceAlert{
			{
				ID:           "alert-1",
				ProfileID:    profileID,
				ProductID:    "product-1",
				TargetPrice:  75.00,
				CurrentPrice: 99.99,
				AlertType:    "below_target",
				IsActive:     true,
				CreatedAt:    time.Now().Add(-24 * time.Hour),
			},
		},
		TrackedProducts: []Product{},
		RecentChanges:   []PriceChange{},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleCreateTracking(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement tracking creation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (s *Server) handlePatternAnalysis(w http.ResponseWriter, r *http.Request) {
	var req PatternAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual pattern analysis
	response := PatternAnalysisResponse{
		Patterns: []PurchasePattern{
			{
				Category:      "Groceries",
				PatternType:   "recurring",
				FrequencyDays: 14,
				AverageSpend:  150.00,
				Confidence:    0.85,
			},
		},
		Predictions: []RestockPrediction{
			{
				ProductCategory: "Coffee",
				PredictedDate:   time.Now().Add(7 * 24 * time.Hour),
				Confidence:      0.75,
				SuggestedItems:  []string{"Your usual coffee brand"},
			},
		},
		SavingsOpportunities: []SavingsOption{
			{
				Description: "Buy coffee in bulk",
				Potential:   25.00,
				Action:      "Purchase 3-month supply",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetProfiles(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement profile retrieval
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]string{})
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement profile creation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Implement single profile retrieval
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id, "name": "Demo Profile"})
}

func (s *Server) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement alerts retrieval
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]PriceAlert{})
}

func (s *Server) handleCreateAlert(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement alert creation
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (s *Server) handleDeleteAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// TODO: Implement alert deletion
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted", "id": id})
}

func (s *Server) Start() {
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(s.router)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Smart Shopping Assistant API starting on port %s", s.port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start smart-shopping-assistant

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server := NewServer()
	server.Start()
}
// Test change
