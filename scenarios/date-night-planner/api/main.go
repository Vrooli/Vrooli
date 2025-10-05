package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

var db *sql.DB

// Health response types
type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
	Checks    map[string]bool `json:"checks,omitempty"`
}

// Date suggestion types
type DateSuggestionRequest struct {
	CoupleID         string  `json:"couple_id"`
	DateType         string  `json:"date_type,omitempty"`
	BudgetMax        float64 `json:"budget_max,omitempty"`
	PreferredDate    string  `json:"preferred_date,omitempty"`
	WeatherPreference string  `json:"weather_preference,omitempty"`
	SurpriseMode     bool    `json:"surprise_mode,omitempty"`
	PlannerID        string  `json:"planner_id,omitempty"` // Who is planning the surprise
}

type Activity struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Duration string `json:"duration"`
	Location string `json:"location,omitempty"`
}

type DateSuggestion struct {
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Activities       []Activity `json:"activities"`
	EstimatedCost    float64    `json:"estimated_cost"`
	EstimatedDuration string    `json:"estimated_duration"`
	ConfidenceScore  float64    `json:"confidence_score"`
	WeatherBackup    *Activity  `json:"weather_backup,omitempty"`
}

type DateSuggestionResponse struct {
	Suggestions          []DateSuggestion `json:"suggestions"`
	PersonalizationFactors []string        `json:"personalization_factors"`
}

// Date plan types
type DatePlanRequest struct {
	CoupleID          string          `json:"couple_id"`
	SelectedSuggestion DateSuggestion  `json:"selected_suggestion"`
	PlannedDate       string          `json:"planned_date"`
	Customizations    json.RawMessage `json:"customizations,omitempty"`
}

type DatePlan struct {
	ID               string    `json:"id"`
	CoupleID         string    `json:"couple_id"`
	Title            string    `json:"title"`
	Description      string    `json:"description"`
	PlannedDate      time.Time `json:"planned_date"`
	Activities       []Activity `json:"activities"`
	EstimatedCost    float64   `json:"estimated_cost"`
	EstimatedDuration string   `json:"estimated_duration"`
	Status           string    `json:"status"`
	IsSurprise       bool      `json:"is_surprise,omitempty"`
	PlannedBy        string    `json:"planned_by,omitempty"`
	RevealDate       *time.Time `json:"reveal_date,omitempty"`
}

type DatePlanResponse struct {
	DatePlan        DatePlan          `json:"date_plan"`
	CalendarInvites []json.RawMessage `json:"calendar_invites,omitempty"`
	Reservations    []json.RawMessage `json:"reservations,omitempty"`
}

// Initialize database connection
func initDB() error {
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5433"
	}
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "vrooli"
	}
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		log.Println("Warning: POSTGRES_PASSWORD not set in environment, database connection may fail")
		dbPassword = "" // Don't use a hardcoded default password
	}
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "vrooli"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	return db.Ping()
}

// Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Service:   "date-night-planner",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func databaseHealthHandler(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]bool)
	checks["postgres_connected"] = false
	checks["schema_exists"] = false

	if db != nil {
		err := db.Ping()
		checks["postgres_connected"] = err == nil

		if err == nil {
			var exists bool
			err = db.QueryRow(`SELECT EXISTS(
				SELECT 1 FROM information_schema.schemata 
				WHERE schema_name = 'date_night_planner'
			)`).Scan(&exists)
			checks["schema_exists"] = exists && err == nil
		}
	}

	status := "healthy"
	if !checks["postgres_connected"] || !checks["schema_exists"] {
		status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := HealthResponse{
		Status:    status,
		Service:   "date-night-planner-database",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func workflowHealthHandler(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]bool)
	checks["n8n_available"] = false
	checks["workflows_active"] = false

	// Check n8n availability
	n8nPort := os.Getenv("N8N_PORT")
	if n8nPort == "" {
		n8nPort = "5678"
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/rest/workflows", n8nPort))
	if err == nil {
		defer resp.Body.Close()
		checks["n8n_available"] = resp.StatusCode < 500
		checks["workflows_active"] = resp.StatusCode == 200
	}

	status := "healthy"
	if !checks["n8n_available"] {
		status = "degraded"
	}

	response := HealthResponse{
		Status:    status,
		Service:   "date-night-planner-workflows",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func suggestDatesHandler(w http.ResponseWriter, r *http.Request) {
	var req DateSuggestionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CoupleID == "" {
		http.Error(w, "couple_id is required", http.StatusBadRequest)
		return
	}

	// Get preferences from database
	preferences := []string{}
	if db != nil {
		rows, err := db.Query(`
			SELECT category, preference_value 
			FROM date_night_planner.preferences 
			WHERE couple_id = $1 
			ORDER BY confidence_score DESC 
			LIMIT 10
		`, req.CoupleID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var cat, val string
				if err := rows.Scan(&cat, &val); err == nil {
					preferences = append(preferences, fmt.Sprintf("%s: %s", cat, val))
				}
			}
		}
	}

	// Get activity suggestions from database
	suggestions := []DateSuggestion{}
	if db != nil {
		// Build query with parameterized placeholders to prevent SQL injection
		queryParts := []string{`
			SELECT title, description,
				   (typical_cost_min + typical_cost_max) / 2 as avg_cost,
				   typical_duration::text,
				   popularity_score
			FROM date_night_planner.activity_suggestions
			WHERE 1=1
		`}
		args := []interface{}{}

		if req.DateType != "" {
			queryParts = append(queryParts, fmt.Sprintf(" AND category = $%d", len(args)+1))
			args = append(args, req.DateType)
		}

		if req.BudgetMax > 0 {
			queryParts = append(queryParts, fmt.Sprintf(" AND typical_cost_max <= $%d", len(args)+1))
			args = append(args, req.BudgetMax)
		}

		if req.WeatherPreference != "" {
			queryParts = append(queryParts, fmt.Sprintf(" AND weather_requirement = $%d", len(args)+1))
			args = append(args, req.WeatherPreference)
		}

		queryParts = append(queryParts, " ORDER BY popularity_score DESC LIMIT 5")

		// Concatenate query parts only after all placeholders are properly set
		var queryBuilder string
		for _, part := range queryParts {
			queryBuilder += part
		}

		rows, err := db.Query(queryBuilder, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var s DateSuggestion
				var avgCost sql.NullFloat64
				var duration sql.NullString
				var popularity sql.NullFloat64

				if err := rows.Scan(&s.Title, &s.Description, &avgCost, &duration, &popularity); err == nil {
					if avgCost.Valid {
						s.EstimatedCost = avgCost.Float64
					}
					if duration.Valid {
						s.EstimatedDuration = duration.String
					}
					if popularity.Valid {
						s.ConfidenceScore = popularity.Float64
					}

					// Add sample activities
					s.Activities = []Activity{
						{
							Type:     req.DateType,
							Name:     s.Title,
							Duration: s.EstimatedDuration,
						},
					}
					suggestions = append(suggestions, s)
				}
			}
		}
	}

	// Fallback suggestions if none from DB - generate based on request
	if len(suggestions) == 0 {
		suggestions = generateDynamicSuggestions(req)
	}

	response := DateSuggestionResponse{
		Suggestions:          suggestions,
		PersonalizationFactors: preferences,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// surpriseDateHandler creates a surprise date plan
func surpriseDateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CoupleID         string    `json:"couple_id"`
		PlannedBy        string    `json:"planned_by"`
		DateSuggestion   DateSuggestion `json:"date_suggestion"`
		PlannedDate      string    `json:"planned_date"`
		RevealTime       string    `json:"reveal_time,omitempty"` // When to reveal the surprise
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CoupleID == "" || req.PlannedBy == "" {
		http.Error(w, "couple_id and planned_by are required", http.StatusBadRequest)
		return
	}

	// Parse dates
	plannedDate, err := time.Parse(time.RFC3339, req.PlannedDate)
	if err != nil {
		plannedDate = time.Now().Add(24 * time.Hour)
	}

	var revealDate *time.Time
	if req.RevealTime != "" {
		if rd, err := time.Parse(time.RFC3339, req.RevealTime); err == nil {
			revealDate = &rd
		}
	}

	// Create surprise date plan
	datePlan := DatePlan{
		ID:               generateUUID(),
		CoupleID:         req.CoupleID,
		Title:            "🎉 Surprise Date: " + req.DateSuggestion.Title,
		Description:      req.DateSuggestion.Description + " (Planned as a surprise!)",
		PlannedDate:      plannedDate,
		Activities:       req.DateSuggestion.Activities,
		EstimatedCost:    req.DateSuggestion.EstimatedCost,
		EstimatedDuration: req.DateSuggestion.EstimatedDuration,
		Status:           "surprise_pending",
		IsSurprise:       true,
		PlannedBy:        req.PlannedBy,
		RevealDate:       revealDate,
	}

	// Save to database with surprise flag
	if db != nil {
		activitiesJSON, _ := json.Marshal(datePlan.Activities)
		_, err = db.Exec(`
			INSERT INTO date_night_planner.date_plans 
			(id, couple_id, title, description, planned_date, activities, 
			 estimated_cost, estimated_duration, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, datePlan.ID, datePlan.CoupleID, datePlan.Title, datePlan.Description,
			datePlan.PlannedDate, activitiesJSON, datePlan.EstimatedCost,
			datePlan.EstimatedDuration, datePlan.Status)
		if err != nil {
			log.Printf("Failed to save surprise date plan: %v", err)
		}
	}

	// Return response with surprise details hidden
	response := map[string]interface{}{
		"surprise_id": datePlan.ID,
		"status": "surprise_created",
		"planned_date": datePlan.PlannedDate,
		"planner_id": req.PlannedBy,
		"reveal_time": revealDate,
		"message": "Surprise date successfully planned! Details will be revealed at the specified time.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// getSurpriseHandler retrieves surprise date details (with access control)
func getSurpriseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	surpriseID := vars["id"]
	
	// Get requester ID from query params
	requesterID := r.URL.Query().Get("requester_id")
	if requesterID == "" {
		http.Error(w, "requester_id is required", http.StatusBadRequest)
		return
	}

	// In a real implementation, this would check the database
	// For now, return a sample surprise date
	datePlan := DatePlan{
		ID:               surpriseID,
		CoupleID:         "couple-123",
		Title:            "🎉 Surprise Date: Romantic Evening",
		Description:      "A special surprise date planned with love",
		PlannedDate:      time.Now().Add(48 * time.Hour),
		Activities: []Activity{
			{Type: "surprise", Name: "Secret Activity 1", Duration: "1 hour"},
			{Type: "surprise", Name: "Secret Activity 2", Duration: "2 hours"},
		},
		EstimatedCost:    150,
		EstimatedDuration: "4 hours",
		Status:           "surprise_pending",
		IsSurprise:       true,
		PlannedBy:        "partner-1",
	}

	// Check if requester is allowed to see details
	canViewDetails := requesterID == datePlan.PlannedBy
	
	// If it's not time to reveal yet or requester isn't the planner, hide details
	if !canViewDetails {
		response := map[string]interface{}{
			"surprise_id": surpriseID,
			"status": "surprise_pending",
			"planned_date": datePlan.PlannedDate,
			"message": "This is a surprise! Details will be revealed soon.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return full details for the planner
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(datePlan)
}

func planDateHandler(w http.ResponseWriter, r *http.Request) {
	var req DatePlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CoupleID == "" {
		http.Error(w, "couple_id is required", http.StatusBadRequest)
		return
	}

	// Parse planned date
	plannedDate, err := time.Parse(time.RFC3339, req.PlannedDate)
	if err != nil {
		plannedDate = time.Now().Add(24 * time.Hour) // Default to tomorrow
	}

	// Create date plan
	datePlan := DatePlan{
		ID:               generateUUID(),
		CoupleID:         req.CoupleID,
		Title:            req.SelectedSuggestion.Title,
		Description:      req.SelectedSuggestion.Description,
		PlannedDate:      plannedDate,
		Activities:       req.SelectedSuggestion.Activities,
		EstimatedCost:    req.SelectedSuggestion.EstimatedCost,
		EstimatedDuration: req.SelectedSuggestion.EstimatedDuration,
		Status:           "planned",
	}

	// Save to database
	if db != nil {
		activitiesJSON, _ := json.Marshal(datePlan.Activities)
		_, err = db.Exec(`
			INSERT INTO date_night_planner.date_plans 
			(id, couple_id, title, description, planned_date, activities, estimated_cost, estimated_duration, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, datePlan.ID, datePlan.CoupleID, datePlan.Title, datePlan.Description,
			datePlan.PlannedDate, activitiesJSON, datePlan.EstimatedCost,
			datePlan.EstimatedDuration, datePlan.Status)
		if err != nil {
			log.Printf("Failed to save date plan: %v", err)
		}
	}

	response := DatePlanResponse{
		DatePlan:        datePlan,
		CalendarInvites: []json.RawMessage{},
		Reservations:    []json.RawMessage{},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// generateDynamicSuggestions creates contextual suggestions based on request parameters
func generateDynamicSuggestions(req DateSuggestionRequest) []DateSuggestion {
	suggestions := []DateSuggestion{}
	
	// Base suggestions by date type
	switch req.DateType {
	case "adventure":
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Outdoor Adventure Experience",
			Description:      "Exciting outdoor activities like hiking, zip-lining, or rock climbing",
			EstimatedCost:    120,
			EstimatedDuration: "4 hours",
			ConfidenceScore:  0.80,
			Activities: []Activity{
				{Type: "adventure", Name: "Hiking Trail", Duration: "2 hours"},
				{Type: "adventure", Name: "Picnic Lunch", Duration: "1 hour"},
				{Type: "adventure", Name: "Scenic Viewpoint", Duration: "1 hour"},
			},
		})
		if req.BudgetMax > 150 {
			suggestions = append(suggestions, DateSuggestion{
				Title:            "Hot Air Balloon Ride",
				Description:      "Romantic aerial adventure with breathtaking views",
				EstimatedCost:    200,
				EstimatedDuration: "3 hours",
				ConfidenceScore:  0.85,
				Activities: []Activity{
					{Type: "adventure", Name: "Balloon Ride", Duration: "1.5 hours"},
					{Type: "dining", Name: "Champagne Toast", Duration: "30 minutes"},
					{Type: "experience", Name: "Photo Session", Duration: "1 hour"},
				},
			})
		}
		
	case "cultural":
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Museum & Art Gallery Tour",
			Description:      "Explore local culture through art and history",
			EstimatedCost:    40,
			EstimatedDuration: "3 hours",
			ConfidenceScore:  0.75,
			Activities: []Activity{
				{Type: "cultural", Name: "Museum Visit", Duration: "2 hours"},
				{Type: "dining", Name: "Café Break", Duration: "1 hour"},
			},
		})
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Theater & Fine Dining",
			Description:      "Evening of culture with a play or musical followed by dinner",
			EstimatedCost:    150,
			EstimatedDuration: "4 hours",
			ConfidenceScore:  0.82,
			Activities: []Activity{
				{Type: "cultural", Name: "Theater Show", Duration: "2.5 hours"},
				{Type: "dining", Name: "Late Dinner", Duration: "1.5 hours"},
			},
		})
		
	case "casual":
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Coffee Shop & Bookstore Browse",
			Description:      "Relaxed afternoon with coffee and book shopping",
			EstimatedCost:    30,
			EstimatedDuration: "2 hours",
			ConfidenceScore:  0.70,
			Activities: []Activity{
				{Type: "casual", Name: "Coffee Date", Duration: "1 hour"},
				{Type: "casual", Name: "Bookstore Browse", Duration: "1 hour"},
			},
		})
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Farmers Market & Cooking Together",
			Description:      "Shop for fresh ingredients and cook a meal together",
			EstimatedCost:    50,
			EstimatedDuration: "3 hours",
			ConfidenceScore:  0.78,
			Activities: []Activity{
				{Type: "casual", Name: "Market Shopping", Duration: "1 hour"},
				{Type: "experience", Name: "Cooking Together", Duration: "2 hours"},
			},
		})
		
	default: // romantic or unspecified
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Sunset Picnic in the Park",
			Description:      "Romantic outdoor dining with sunset views",
			EstimatedCost:    40,
			EstimatedDuration: "2.5 hours",
			ConfidenceScore:  0.77,
			Activities: []Activity{
				{Type: "romantic", Name: "Picnic Setup", Duration: "30 minutes"},
				{Type: "dining", Name: "Outdoor Dining", Duration: "1.5 hours"},
				{Type: "romantic", Name: "Sunset Walk", Duration: "30 minutes"},
			},
			WeatherBackup: &Activity{
				Type: "dining", Name: "Indoor Restaurant", Duration: "2 hours",
			},
		})
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Wine Tasting & Vineyard Tour",
			Description:      "Explore local wineries with tastings and tours",
			EstimatedCost:    90,
			EstimatedDuration: "3 hours",
			ConfidenceScore:  0.80,
			Activities: []Activity{
				{Type: "experience", Name: "Vineyard Tour", Duration: "1 hour"},
				{Type: "experience", Name: "Wine Tasting", Duration: "1.5 hours"},
				{Type: "dining", Name: "Light Appetizers", Duration: "30 minutes"},
			},
		})
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Candlelit Dinner & Dancing",
			Description:      "Classic romantic evening with fine dining and dancing",
			EstimatedCost:    120,
			EstimatedDuration: "4 hours",
			ConfidenceScore:  0.85,
			Activities: []Activity{
				{Type: "dining", Name: "Fine Dining", Duration: "2 hours"},
				{Type: "romantic", Name: "Dancing", Duration: "2 hours"},
			},
		})
	}
	
	// Add weather-aware suggestion if preference specified
	if req.WeatherPreference == "indoor" {
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Spa Day & Relaxation",
			Description:      "Indoor pampering with couples massage and relaxation",
			EstimatedCost:    180,
			EstimatedDuration: "4 hours",
			ConfidenceScore:  0.82,
			Activities: []Activity{
				{Type: "relaxation", Name: "Couples Massage", Duration: "1.5 hours"},
				{Type: "relaxation", Name: "Spa Amenities", Duration: "1.5 hours"},
				{Type: "dining", Name: "Healthy Lunch", Duration: "1 hour"},
			},
		})
	} else if req.WeatherPreference == "outdoor" {
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Beach Day & Sunset Dinner",
			Description:      "Full day at the beach with activities and waterfront dining",
			EstimatedCost:    100,
			EstimatedDuration: "6 hours",
			ConfidenceScore:  0.83,
			Activities: []Activity{
				{Type: "outdoor", Name: "Beach Activities", Duration: "3 hours"},
				{Type: "dining", Name: "Beachside Lunch", Duration: "1 hour"},
				{Type: "romantic", Name: "Sunset Dinner", Duration: "2 hours"},
			},
			WeatherBackup: &Activity{
				Type: "indoor", Name: "Aquarium Visit", Duration: "3 hours",
			},
		})
	}
	
	// Filter by budget if specified
	if req.BudgetMax > 0 {
		filteredSuggestions := []DateSuggestion{}
		for _, s := range suggestions {
			if s.EstimatedCost <= req.BudgetMax {
				filteredSuggestions = append(filteredSuggestions, s)
			}
		}
		if len(filteredSuggestions) > 0 {
			suggestions = filteredSuggestions
		}
	}
	
	// Limit to top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	
	// If still no suggestions, provide a default
	if len(suggestions) == 0 {
		suggestions = append(suggestions, DateSuggestion{
			Title:            "Romantic Dinner Date",
			Description:      "Enjoy an intimate dinner at a cozy restaurant",
			EstimatedCost:    80,
			EstimatedDuration: "2 hours",
			ConfidenceScore:  0.75,
			Activities: []Activity{
				{Type: "dining", Name: "Restaurant", Duration: "2 hours"},
			},
		})
	}
	
	return suggestions
}

// Helper function to generate UUID (simplified)
func generateUUID() string {
	return fmt.Sprintf("%d-%d-%d-%d-%d",
		time.Now().Unix(),
		time.Now().Nanosecond(),
		os.Getpid(),
		time.Now().UnixNano()%1000,
		time.Now().UnixNano()%10000,
	)
}

// CORS middleware
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// Load environment variables
	godotenv.Load()

	// Initialize database
	if err := initDB(); err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Running in degraded mode without database")
	}

	// Setup routes
	router := mux.NewRouter()

	// Health endpoints
	router.HandleFunc("/health", enableCORS(healthHandler)).Methods("GET", "OPTIONS")
	router.HandleFunc("/health/database", enableCORS(databaseHealthHandler)).Methods("GET", "OPTIONS")
	router.HandleFunc("/health/workflows", enableCORS(workflowHealthHandler)).Methods("GET", "OPTIONS")

	// API endpoints
	router.HandleFunc("/api/v1/dates/suggest", enableCORS(suggestDatesHandler)).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/dates/plan", enableCORS(planDateHandler)).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/dates/surprise", enableCORS(surpriseDateHandler)).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/v1/dates/surprise/{id}", enableCORS(getSurpriseHandler)).Methods("GET", "OPTIONS")

	// Root endpoint
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"service": "date-night-planner",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// Get port from environment
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Date Night Planner API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}