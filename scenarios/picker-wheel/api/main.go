package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Wheel struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Options     []Option  `json:"options"`
	Theme       string    `json:"theme"`
	CreatedAt   time.Time `json:"created_at"`
	TimesUsed   int       `json:"times_used"`
}

type Option struct {
	Label  string  `json:"label"`
	Color  string  `json:"color"`
	Weight float64 `json:"weight"`
}

type SpinResult struct {
	Result    string    `json:"result"`
	WheelID   string    `json:"wheel_id"`
	SessionID string    `json:"session_id"`
	Timestamp time.Time `json:"timestamp"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

var db *sql.DB
var wheels []Wheel
var history []SpinResult

func initDB() {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Println("❌ Database configuration missing. Using in-memory fallback. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
			return
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	var err error
	db, err = sql.Open("postgres", postgresURL)
	if err != nil {
		log.Printf("Failed to open database connection: %v - using in-memory fallback", err)
		return
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📆 Database URL configured")

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Printf("❌ Database connection failed after %d attempts: %v - using in-memory fallback", maxRetries, pingErr)
		db = nil
		return
	}

	log.Println("🎉 Database connection pool established successfully!")
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start picker-wheel

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize database connection
	initDB()
	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	// Initialize in-memory data as fallback
	wheels = getDefaultWheels()
	history = []SpinResult{}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// API endpoints
	router.HandleFunc("/api/wheels", getWheelsHandler).Methods("GET")
	router.HandleFunc("/api/wheels", createWheelHandler).Methods("POST")
	router.HandleFunc("/api/wheels/{id}", getWheelHandler).Methods("GET")
	router.HandleFunc("/api/wheels/{id}", deleteWheelHandler).Methods("DELETE")
	router.HandleFunc("/api/history", getHistoryHandler).Methods("GET")
	router.HandleFunc("/api/spin", spinHandler).Methods("POST")

	// Enable CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	log.Printf("🎯 Picker Wheel API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// getEnv removed to prevent hardcoded defaults

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "healthy",
		Service: "picker-wheel-api",
		Version: "1.0.0",
	}
	json.NewEncoder(w).Encode(response)
}

func getWheelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		// Fallback to default wheels if no database
		json.NewEncoder(w).Encode(getDefaultWheels())
		return
	}

	rows, err := db.Query(`
        SELECT id, name, description, options, theme, times_used, created_at 
        FROM wheels 
        WHERE is_public = true 
        ORDER BY times_used DESC, created_at DESC
        LIMIT 20
    `)
	if err != nil {
		log.Printf("Error querying wheels: %v", err)
		// Return in-memory wheels on any database error
		wheels = getDefaultWheels()
		json.NewEncoder(w).Encode(wheels)
		return
	}
	defer rows.Close()

	var wheels []Wheel
	for rows.Next() {
		var wheel Wheel
		var optionsJSON []byte
		err := rows.Scan(&wheel.ID, &wheel.Name, &wheel.Description,
			&optionsJSON, &wheel.Theme, &wheel.TimesUsed, &wheel.CreatedAt)
		if err != nil {
			log.Printf("Error scanning wheel: %v", err)
			continue
		}

		if err := json.Unmarshal(optionsJSON, &wheel.Options); err != nil {
			log.Printf("Error unmarshalling options: %v", err)
			continue
		}

		wheels = append(wheels, wheel)
	}

	if len(wheels) == 0 {
		wheels = getDefaultWheels()
	}

	json.NewEncoder(w).Encode(wheels)
}

func createWheelHandler(w http.ResponseWriter, r *http.Request) {
	var wheel Wheel
	if err := json.NewDecoder(r.Body).Decode(&wheel); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	wheel.CreatedAt = time.Now()

	if db == nil {
		// Fallback to in-memory if no database
		wheel.ID = fmt.Sprintf("wheel_%d", time.Now().Unix())
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wheel)
		return
	}

	optionsJSON, err := json.Marshal(wheel.Options)
	if err != nil {
		http.Error(w, "Failed to marshal options", http.StatusInternalServerError)
		return
	}

	var id string
	err = db.QueryRow(`
        INSERT INTO wheels (name, description, options, theme, is_public) 
        VALUES ($1, $2, $3, $4, $5) 
        RETURNING id
    `, wheel.Name, wheel.Description, optionsJSON, wheel.Theme, false).Scan(&id)

	if err != nil {
		log.Printf("Error creating wheel in database: %v - falling back to in-memory", err)
		// Fallback to in-memory on database error
		wheel.ID = fmt.Sprintf("wheel_%d", time.Now().Unix())
		wheels = append(wheels, wheel)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wheel)
		return
	}

	wheel.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wheel)
}

func getWheelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, wheel := range wheels {
		if wheel.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(wheel)
			return
		}
	}

	http.Error(w, "Wheel not found", http.StatusNotFound)
}

func deleteWheelHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for i, wheel := range wheels {
		if wheel.ID == id {
			wheels = append(wheels[:i], wheels[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	http.Error(w, "Wheel not found", http.StatusNotFound)
}

func getHistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Try to get from database if available
	if db != nil {
		rows, err := db.Query(`
			SELECT sh.id, sh.wheel_id, sh.result, sh.session_id, sh.timestamp
			FROM spin_history sh
			ORDER BY sh.timestamp DESC
			LIMIT 100
		`)

		if err == nil {
			defer rows.Close()
			var dbHistory []SpinResult

			for rows.Next() {
				var result SpinResult
				var id string
				err := rows.Scan(&id, &result.WheelID, &result.Result, &result.SessionID, &result.Timestamp)
				if err == nil {
					dbHistory = append(dbHistory, result)
				}
			}

			if len(dbHistory) > 0 {
				json.NewEncoder(w).Encode(dbHistory)
				return
			}
		}
	}

	// Fallback to in-memory history
	json.NewEncoder(w).Encode(history)
}

func getDefaultWheels() []Wheel {
	return []Wheel{
		{
			ID:          "dinner-decider",
			Name:        "Dinner Decider",
			Description: "Can't decide what to eat?",
			Options: []Option{
				{Label: "Pizza 🍕", Color: "#FF6B6B", Weight: 1},
				{Label: "Sushi 🍱", Color: "#4ECDC4", Weight: 1},
				{Label: "Tacos 🌮", Color: "#FFD93D", Weight: 1},
				{Label: "Burger 🍔", Color: "#6C5CE7", Weight: 1},
			},
			Theme:     "food",
			CreatedAt: time.Now(),
			TimesUsed: 0,
		},
		{
			ID:          "yes-or-no",
			Name:        "Yes or No",
			Description: "Simple decision maker",
			Options: []Option{
				{Label: "YES! ✅", Color: "#4CAF50", Weight: 1},
				{Label: "NO ❌", Color: "#F44336", Weight: 1},
			},
			Theme:     "minimal",
			CreatedAt: time.Now(),
			TimesUsed: 0,
		},
	}
}

func spinHandler(w http.ResponseWriter, r *http.Request) {
	var spinRequest struct {
		WheelID   string   `json:"wheel_id"`
		SessionID string   `json:"session_id"`
		Options   []Option `json:"options"`
	}

	if err := json.NewDecoder(r.Body).Decode(&spinRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If no options provided, look up wheel by ID
	options := spinRequest.Options
	if len(options) == 0 && spinRequest.WheelID != "" {
		// First try database if available
		if db != nil {
			var optionsJSON []byte
			err := db.QueryRow("SELECT options FROM wheels WHERE id = $1", spinRequest.WheelID).Scan(&optionsJSON)
			if err == nil {
				var dbOptions []Option
				if err := json.Unmarshal(optionsJSON, &dbOptions); err == nil {
					options = dbOptions
				}
			}
		}
		// Fallback to in-memory wheels
		if len(options) == 0 {
			for _, wheel := range wheels {
				if wheel.ID == spinRequest.WheelID {
					options = wheel.Options
					break
				}
			}
		}
	}

	// Select random option based on weights
	var selectedOption string
	if len(options) > 0 {
		// Calculate total weight
		var totalWeight float64
		for _, option := range options {
			totalWeight += option.Weight
		}

		// Generate random number and select option
		randValue := rand.Float64() * totalWeight
		var currentWeight float64

		for _, option := range options {
			currentWeight += option.Weight
			if randValue <= currentWeight {
				selectedOption = option.Label
				break
			}
		}
	} else {
		selectedOption = "No options provided"
	}

	result := SpinResult{
		Result:    selectedOption,
		WheelID:   spinRequest.WheelID,
		SessionID: spinRequest.SessionID,
		Timestamp: time.Now(),
	}

	history = append(history, result)

	// Store in database if available
	if db != nil {
		_, err := db.Exec(`
			INSERT INTO spin_history (wheel_id, result, session_id, timestamp) 
			VALUES ($1, $2, $3, $4)
		`, spinRequest.WheelID, selectedOption, spinRequest.SessionID, result.Timestamp)

		if err != nil {
			log.Printf("Error storing spin history: %v", err)
		}

		// Also update times_used for the wheel
		_, err = db.Exec(`
			UPDATE wheels SET times_used = times_used + 1 
			WHERE id = $1
		`, spinRequest.WheelID)

		if err != nil {
			log.Printf("Error updating wheel usage count: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
