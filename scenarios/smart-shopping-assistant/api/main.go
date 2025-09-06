package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	router *mux.Router
	port   string
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
}

type ShoppingResearchRequest struct {
	ProfileID         string  `json:"profile_id"`
	Query             string  `json:"query"`
	BudgetMax         float64 `json:"budget_max,omitempty"`
	IncludeAlternatives bool   `json:"include_alternatives"`
	GiftRecipientID   string  `json:"gift_recipient_id,omitempty"`
}

type Product struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Category        string                 `json:"category"`
	Description     string                 `json:"description"`
	CurrentPrice    float64                `json:"current_price"`
	OriginalPrice   float64                `json:"original_price,omitempty"`
	AffiliateLink   string                 `json:"affiliate_link,omitempty"`
	Features        map[string]interface{} `json:"features,omitempty"`
	ReviewsSummary  *ReviewsSummary        `json:"reviews_summary,omitempty"`
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
	CurrentTrend    string  `json:"current_trend"`
	BestTimeToWait  bool    `json:"best_time_to_wait"`
	PredictedDrop   float64 `json:"predicted_drop,omitempty"`
	HistoricalLow   float64 `json:"historical_low"`
	HistoricalHigh  float64 `json:"historical_high"`
}

type ShoppingResearchResponse struct {
	Products        []Product       `json:"products"`
	Alternatives    []Alternative   `json:"alternatives"`
	PriceAnalysis   PriceInsights   `json:"price_analysis"`
	Recommendations []string        `json:"recommendations"`
	AffiliateLinks  []AffiliateLink `json:"affiliate_links"`
}

type AffiliateLink struct {
	ProductID    string  `json:"product_id"`
	Retailer     string  `json:"retailer"`
	URL          string  `json:"url"`
	Commission   float64 `json:"commission"`
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
	ActiveAlerts    []PriceAlert   `json:"active_alerts"`
	TrackedProducts []Product      `json:"tracked_products"`
	RecentChanges   []PriceChange  `json:"recent_changes"`
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

func NewServer() *Server {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3300"
	}

	s := &Server{
		router: mux.NewRouter(),
		port:   port,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	
	// Shopping research endpoints
	s.router.HandleFunc("/api/v1/shopping/research", s.handleShoppingResearch).Methods("POST")
	s.router.HandleFunc("/api/v1/shopping/tracking/{profile_id}", s.handleGetTracking).Methods("GET")
	s.router.HandleFunc("/api/v1/shopping/tracking", s.handleCreateTracking).Methods("POST")
	s.router.HandleFunc("/api/v1/shopping/pattern-analysis", s.handlePatternAnalysis).Methods("POST")
	
	// Profile management
	s.router.HandleFunc("/api/v1/profiles", s.handleGetProfiles).Methods("GET")
	s.router.HandleFunc("/api/v1/profiles", s.handleCreateProfile).Methods("POST")
	s.router.HandleFunc("/api/v1/profiles/{id}", s.handleGetProfile).Methods("GET")
	
	// Price alerts
	s.router.HandleFunc("/api/v1/alerts", s.handleGetAlerts).Methods("GET")
	s.router.HandleFunc("/api/v1/alerts", s.handleCreateAlert).Methods("POST")
	s.router.HandleFunc("/api/v1/alerts/{id}", s.handleDeleteAlert).Methods("DELETE")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Service:   "smart-shopping-assistant",
		Version:   "1.0.0",
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
	
	// TODO: Implement actual shopping research logic
	// For now, return mock data
	response := ShoppingResearchResponse{
		Products: []Product{
			{
				ID:            "mock-product-1",
				Name:          "Sample Product",
				Category:      "Electronics",
				Description:   "A great product matching your search",
				CurrentPrice:  99.99,
				OriginalPrice: 129.99,
				AffiliateLink: "https://example.com/product?ref=smartshop",
				ReviewsSummary: &ReviewsSummary{
					AverageRating: 4.5,
					TotalReviews:  1250,
					Pros:          []string{"Great quality", "Good value"},
					Cons:          []string{"Limited colors"},
				},
			},
		},
		Alternatives: []Alternative{
			{
				Product: Product{
					ID:           "mock-alt-1",
					Name:         "Generic Alternative",
					Category:     "Electronics",
					CurrentPrice: 59.99,
				},
				AlternativeType: "generic",
				SavingsAmount:   40.00,
				Reason:          "Similar features at lower price point",
			},
		},
		PriceAnalysis: PriceInsights{
			CurrentTrend:   "stable",
			BestTimeToWait: false,
			HistoricalLow:  89.99,
			HistoricalHigh: 149.99,
		},
		Recommendations: []string{
			"Current price is near historical low",
			"Consider the generic alternative for 40% savings",
		},
		AffiliateLinks: []AffiliateLink{
			{
				ProductID:  "mock-product-1",
				Retailer:   "Example Store",
				URL:        "https://example.com/product?ref=smartshop",
				Commission: 4.0,
			},
		},
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
				Description:      "Buy coffee in bulk",
				Potential:        25.00,
				Action:           "Purchase 3-month supply",
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
	server := NewServer()
	server.Start()
}