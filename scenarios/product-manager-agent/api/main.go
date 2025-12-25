package main

import (
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "product-manager-agent",
	}) {
		return // Process was re-exec'd after rebuild
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
	return database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
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
