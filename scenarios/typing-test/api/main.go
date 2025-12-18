package main

import (
	"github.com/vrooli/api-core/preflight"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Score struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Score      int       `json:"score"`
	WPM        int       `json:"wpm"`
	Accuracy   int       `json:"accuracy"`
	MaxCombo   int       `json:"maxCombo"`
	Difficulty string    `json:"difficulty"`
	Mode       string    `json:"mode"`
	CreatedAt  time.Time `json:"createdAt"`
}

type LeaderboardResponse struct {
	Scores []Score `json:"scores"`
	Period string  `json:"period"`
}

type StatsRequest struct {
	SessionID string `json:"sessionId"`
	WPM       int    `json:"wpm"`
	Accuracy  int    `json:"accuracy"`
	Text      string `json:"text"`
}

type CoachingRequest struct {
	UserStats  StatsRequest `json:"userStats"`
	Difficulty string       `json:"difficulty"`
}

type AdaptiveTextRequest struct {
	UserID           string   `json:"userId"`
	Difficulty       string   `json:"difficulty"`
	TargetWords      []string `json:"targetWords"`
	ProblemChars     []string `json:"problemChars"`
	UserLevel        string   `json:"userLevel"`
	TextLength       string   `json:"textLength"`
	PreviousMistakes []struct {
		Word       string `json:"word"`
		Char       string `json:"char"`
		Position   string `json:"position"`
		ErrorCount int    `json:"errorCount"`
	} `json:"previousMistakes"`
}

type AdaptiveTextResponse struct {
	Text       string `json:"text"`
	WordCount  int    `json:"wordCount"`
	Difficulty string `json:"difficulty"`
	IsAdaptive bool   `json:"isAdaptive"`
	Timestamp  string `json:"timestamp"`
}

var db *sql.DB
var typingProcessor *TypingProcessor

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "typing-test",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

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
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	// Connect to database
	var err error
	db, err = sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}
	defer db.Close()

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
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	// Initialize database
	initDB()

	// Initialize TypingProcessor
	typingProcessor = NewTypingProcessor(db)

	// Setup routes
	router := mux.NewRouter()

	// API routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/leaderboard", getLeaderboard).Methods("GET")
	router.HandleFunc("/api/submit-score", submitScore).Methods("POST")
	router.HandleFunc("/api/stats", submitStats).Methods("POST")
	router.HandleFunc("/api/coaching", getCoaching).Methods("POST")
	router.HandleFunc("/api/practice-text", getPracticeText).Methods("GET")
	router.HandleFunc("/api/adaptive-text", getAdaptiveText).Methods("POST")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	log.Printf("🎮 Typing Test API running on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initDB() {
	// Check if tables exist from the proper schema.sql initialization
	// If not, we'll create minimal tables for basic functionality
	var tableExists bool

	// Check if the proper schema is already loaded (from initialization/postgres/schema.sql)
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'players')").Scan(&tableExists)
	if err != nil {
		log.Printf("Warning: Failed to check schema: %v", err)
	}

	if tableExists {
		// Proper schema exists, just add any missing basic compatibility
		log.Printf("✅ Using existing PostgreSQL schema with full features")

		// Ensure compatibility columns exist in scores table for our simplified API
		compatQuery := `
        DO $$ 
        BEGIN
            -- Add name column to scores if it doesn't exist (for backwards compatibility)
            IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                          WHERE table_name = 'scores' AND column_name = 'name') THEN
                ALTER TABLE scores ADD COLUMN name VARCHAR(50);
            END IF;
        END $$;
        `

		if _, err := db.Exec(compatQuery); err != nil {
			log.Printf("Warning: Failed to add compatibility columns: %v", err)
		}
		return
	}

	// Fallback: create minimal tables if proper schema wasn't loaded
	log.Printf("⚠️  Creating minimal schema - consider running proper schema.sql for full features")

	query := `
    CREATE TABLE IF NOT EXISTS scores (
        id SERIAL PRIMARY KEY,
        name VARCHAR(50) NOT NULL,
        score INTEGER NOT NULL,
        wpm INTEGER NOT NULL,
        accuracy INTEGER NOT NULL,
        max_combo INTEGER DEFAULT 0,
        difficulty VARCHAR(20) DEFAULT 'easy',
        mode VARCHAR(20) DEFAULT 'classic',
        user_id VARCHAR(100),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    CREATE TABLE IF NOT EXISTS typing_sessions (
        id SERIAL PRIMARY KEY,
        session_id VARCHAR(100) UNIQUE NOT NULL,
        total_chars INTEGER DEFAULT 0,
        correct_chars INTEGER DEFAULT 0,
        total_time INTEGER DEFAULT 0,
        texts_completed INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE TABLE IF NOT EXISTS game_sessions (
        id SERIAL PRIMARY KEY,
        session_id VARCHAR(100) UNIQUE NOT NULL,
        session_data JSONB,
        started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        duration_seconds INTEGER DEFAULT 0
    );

    CREATE INDEX IF NOT EXISTS idx_scores_created_at ON scores(created_at DESC);
    CREATE INDEX IF NOT EXISTS idx_scores_score ON scores(score DESC);
    CREATE INDEX IF NOT EXISTS idx_sessions_session_id ON typing_sessions(session_id);
    `

	if _, err := db.Exec(query); err != nil {
		log.Printf("Warning: Failed to initialize minimal database: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connectivity
	dbConnected := false
	var dbLatency *float64
	var dbError interface{}

	if db != nil {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err == nil {
			dbConnected = true
			latency := float64(time.Since(start).Milliseconds())
			dbLatency = &latency
			dbError = nil
		} else {
			dbError = map[string]interface{}{
				"code":      "DB_PING_FAILED",
				"message":   err.Error(),
				"category":  "resource",
				"retryable": true,
			}
		}
	}

	status := "healthy"
	if !dbConnected {
		status = "degraded"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "typing-test-api",
		"timestamp": time.Now().Format(time.RFC3339),
		"readiness": dbConnected, // Ready when DB is connected
		"version":   "1.0.0",
		"dependencies": map[string]interface{}{
			"database": map[string]interface{}{
				"connected":  dbConnected,
				"latency_ms": dbLatency,
				"error":      dbError,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

func getLeaderboard(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "all"
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = uuid.New().String()
	}

	// Use TypingProcessor to get leaderboard
	ctx := context.Background()
	entries, err := typingProcessor.ManageLeaderboard(ctx, period, userID)
	if err != nil {
		http.Error(w, "Failed to fetch leaderboard", http.StatusInternalServerError)
		return
	}

	// Convert LeaderboardEntry to Score for compatibility
	scores := []Score{}
	for _, entry := range entries {
		scores = append(scores, Score{
			ID:        entry.Rank,
			Name:      entry.Name,
			Score:     entry.Score,
			WPM:       entry.WPM,
			Accuracy:  int(entry.Accuracy),
			CreatedAt: entry.Date,
		})
	}

	response := LeaderboardResponse{
		Scores: scores,
		Period: period,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func submitScore(w http.ResponseWriter, r *http.Request) {
	var score Score
	if err := json.NewDecoder(r.Body).Decode(&score); err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use TypingProcessor to add score
	ctx := context.Background()
	err := typingProcessor.AddScore(ctx, score)
	if err != nil {
		log.Printf("Error saving score: %v", err)
		http.Error(w, fmt.Sprintf("Failed to save score: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("New score submitted: %s - %d points", score.Name, score.Score)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"scoreId": score.ID,
	})
}

func submitStats(w http.ResponseWriter, r *http.Request) {
	var stats StatsRequest
	if err := json.NewDecoder(r.Body).Decode(&stats); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to SessionStats for processing
	sessionStats := SessionStats{
		SessionID:       stats.SessionID,
		WPM:             stats.WPM,
		Accuracy:        float64(stats.Accuracy),
		CharactersTyped: len(stats.Text),
		ErrorCount:      int(float64(len(stats.Text)) * (100 - float64(stats.Accuracy)) / 100),
		TimeSpent:       60, // Default session time
		Difficulty:      "medium",
		TextCompleted:   true,
	}

	// Process stats using TypingProcessor
	ctx := context.Background()
	result, err := typingProcessor.ProcessStats(ctx, sessionStats)
	if err != nil {
		log.Printf("Failed to process stats: %v", err)
		http.Error(w, "Failed to process stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getCoaching(w http.ResponseWriter, r *http.Request) {
	var request CoachingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert to SessionStats for coaching
	sessionStats := SessionStats{
		SessionID:       request.UserStats.SessionID,
		WPM:             request.UserStats.WPM,
		Accuracy:        float64(request.UserStats.Accuracy),
		CharactersTyped: len(request.UserStats.Text),
		TimeSpent:       60,
		Difficulty:      request.Difficulty,
		TextCompleted:   true,
	}

	// Get coaching from TypingProcessor
	ctx := context.Background()
	coaching := typingProcessor.ProvideCoaching(ctx, sessionStats, request.Difficulty)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coaching)
}

func getPracticeText(w http.ResponseWriter, r *http.Request) {
	difficulty := r.URL.Query().Get("difficulty")
	if difficulty == "" {
		difficulty = "easy"
	}

	// In production, this would fetch from database or n8n workflow
	texts := map[string][]string{
		"easy": {
			"The cat sat on the mat and looked at the moon",
			"A bird in hand is worth two in the bush today",
			"Time flies when you are having fun with friends",
		},
		"medium": {
			"The extraordinary symphony echoed through the vast cathedral halls magnificently",
			"Quantum mechanics revolutionizes our understanding of subatomic particle behavior",
			"Artificial intelligence transforms industries through automated decision-making processes",
		},
		"hard": {
			"Pseudopseudohypoparathyroidism represents complex endocrinological dysfunction requiring specialized treatment",
			"Cryptographically secure pseudorandom generators utilize mathematical algorithms ensuring unpredictability",
			"Neuroplasticity demonstrates the brain's remarkable capacity for reorganization throughout life",
		},
	}

	selectedTexts := texts[difficulty]
	if selectedTexts == nil {
		selectedTexts = texts["easy"]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"texts":      selectedTexts,
		"difficulty": difficulty,
	})
}

func getRecommendedDifficulty(wpm, accuracy int) string {
	if wpm > 60 && accuracy > 95 {
		return "hard"
	} else if wpm > 40 && accuracy > 90 {
		return "medium"
	}
	return "easy"
}

func getAdaptiveText(w http.ResponseWriter, r *http.Request) {
	var request AdaptiveTextRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults if not provided
	if request.Difficulty == "" {
		request.Difficulty = "medium"
	}
	if request.UserLevel == "" {
		request.UserLevel = "intermediate"
	}
	if request.TextLength == "" {
		request.TextLength = "medium"
	}

	// Use TypingProcessor to generate adaptive text
	ctx := context.Background()
	response := typingProcessor.GenerateAdaptiveText(ctx, request)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
