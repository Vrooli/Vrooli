package main

import (
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

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

// App represents the application context
type App struct {
	DB          *sql.DB
	RedisClient *redis.Client
	OllamaURL   string
	QdrantURL   string
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start product-manager-agent

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT
		if port == "" {
			log.Fatal("❌ API_PORT or PORT environment variable is required")
		}
	}

	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Redis
	redisClient := initRedis()
	defer redisClient.Close()

	// Ollama URL - REQUIRED, no defaults
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}

	// Qdrant URL - REQUIRED, no defaults
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		log.Fatal("❌ QDRANT_URL environment variable is required")
	}

	// Create app instance
	app := &App{
		DB:          db,
		RedisClient: redisClient,
		OllamaURL:   ollamaURL,
		QdrantURL:   qdrantURL,
	}

	// Setup routes
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", corsMiddleware(app.healthHandler))

	// Feature management
	mux.HandleFunc("/api/features", corsMiddleware(app.featuresHandler))
	mux.HandleFunc("/api/features/prioritize", corsMiddleware(app.prioritizeHandler))
	mux.HandleFunc("/api/features/rice", corsMiddleware(app.riceScoreHandler))

	// Roadmap management
	mux.HandleFunc("/api/roadmap", corsMiddleware(app.roadmapHandler))
	mux.HandleFunc("/api/roadmap/generate", corsMiddleware(app.generateRoadmapHandler))

	// Sprint planning
	mux.HandleFunc("/api/sprint/plan", corsMiddleware(app.sprintPlanHandler))
	mux.HandleFunc("/api/sprint/current", corsMiddleware(app.currentSprintHandler))

	// Analysis endpoints
	mux.HandleFunc("/api/market/analyze", corsMiddleware(app.marketAnalysisHandler))
	mux.HandleFunc("/api/competitor/analyze", corsMiddleware(app.competitorAnalysisHandler))
	mux.HandleFunc("/api/feedback/analyze", corsMiddleware(app.feedbackAnalysisHandler))
	mux.HandleFunc("/api/roi/calculate", corsMiddleware(app.roiCalculationHandler))
	mux.HandleFunc("/api/decision/analyze", corsMiddleware(app.decisionAnalysisHandler))

	// Dashboard
	mux.HandleFunc("/api/dashboard", corsMiddleware(app.dashboardHandler))

	fmt.Printf("Product Manager API server starting on port %s\n", port)
	fmt.Printf("Ollama URL: %s\n", app.OllamaURL)
	fmt.Printf("Database connected\n")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func initDB() (*sql.DB, error) {
	// Database configuration - support both POSTGRES_URL and individual components
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")

		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			log.Fatal("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
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
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	return db, nil
}

func initRedis() *redis.Client {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("Failed to parse Redis URL, using defaults: %v", err)
		opt = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	_, err = client.Ping(ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
	}

	return client
}

// getEnv removed to prevent hardcoded defaults

func corsMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}

func (app *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	dbHealthy := true
	if app.DB != nil {
		if err := app.DB.Ping(); err != nil {
			dbHealthy = false
		}
	} else {
		dbHealthy = false
	}

	// Check Redis connection
	redisHealthy := true
	if app.RedisClient != nil {
		ctx := context.Background()
		if _, err := app.RedisClient.Ping(ctx).Result(); err != nil {
			redisHealthy = false
		}
	} else {
		redisHealthy = false
	}

	status := "healthy"
	if !dbHealthy || !redisHealthy {
		status = "degraded"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"service":   "product-manager-api",
		"database":  dbHealthy,
		"redis":     redisHealthy,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (app *App) featuresHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		app.getFeatures(w, r)
	case "POST":
		app.createFeature(w, r)
	case "PUT":
		app.updateFeature(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (app *App) getFeatures(w http.ResponseWriter, r *http.Request) {
	features, err := app.fetchFeatures()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(features)
}

func (app *App) createFeature(w http.ResponseWriter, r *http.Request) {
	var feature Feature
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate RICE score
	feature.Score = app.calculateRICE(&feature)
	feature.CreatedAt = time.Now()
	feature.UpdatedAt = time.Now()

	// Store in database
	if err := app.storeFeature(&feature); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feature)
}

func (app *App) updateFeature(w http.ResponseWriter, r *http.Request) {
	var feature Feature
	if err := json.NewDecoder(r.Body).Decode(&feature); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Recalculate RICE score
	feature.Score = app.calculateRICE(&feature)
	feature.UpdatedAt = time.Now()

	// Update in database
	if err := app.updateFeatureDB(&feature); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feature)
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	dashboard := map[string]interface{}{
		"metrics": map[string]interface{}{
			"active_features":    24,
			"sprint_progress":    87,
			"avg_priority_score": 4.2,
			"projected_roi":      125000,
		},
		"recent_decisions": []string{
			"Migrate to TypeScript",
			"Implement WebSocket support",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

func prioritizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var features []Feature
	if err := json.NewDecoder(r.Body).Decode(&features); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate RICE scores
	for i := range features {
		score := float64(features[i].Reach*features[i].Impact) * features[i].Confidence / float64(features[i].Effort)
		features[i].Score = score

		// Assign priority based on score
		if score > 1000 {
			features[i].Priority = "CRITICAL"
		} else if score > 500 {
			features[i].Priority = "HIGH"
		} else if score > 100 {
			features[i].Priority = "MEDIUM"
		} else {
			features[i].Priority = "LOW"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(features)
}
