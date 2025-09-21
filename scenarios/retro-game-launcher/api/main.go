package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
)

type Game struct {
	ID           string    `json:"id" db:"id"`
	Title        string    `json:"title" db:"title"`
	Prompt       string    `json:"prompt" db:"prompt"`
	Description  *string   `json:"description" db:"description"`
	Code         string    `json:"code" db:"code"`
	Engine       string    `json:"engine" db:"engine"`
	AuthorID     *string   `json:"author_id" db:"author_id"`
	ParentGameID *string   `json:"parent_game_id" db:"parent_game_id"`
	IsRemix      bool      `json:"is_remix" db:"is_remix"`
	ThumbnailURL *string   `json:"thumbnail_url" db:"thumbnail_url"`
	PlayCount    int       `json:"play_count" db:"play_count"`
	RemixCount   int       `json:"remix_count" db:"remix_count"`
	Rating       *float64  `json:"rating" db:"rating"`
	Tags         []string  `json:"tags" db:"tags"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	Published    bool      `json:"published" db:"published"`
}

type User struct {
	ID                      string     `json:"id" db:"id"`
	Username                string     `json:"username" db:"username"`
	Email                   string     `json:"email" db:"email"`
	SubscriptionTier        string     `json:"subscription_tier" db:"subscription_tier"`
	GamesCreatedThisMonth   int        `json:"games_created_this_month" db:"games_created_this_month"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	LastLogin               *time.Time `json:"last_login" db:"last_login"`
}

type GameGenerationRequest struct {
	Prompt     string   `json:"prompt"`
	Engine     string   `json:"engine"`
	Tags       []string `json:"tags"`
	AuthorID   *string  `json:"author_id"`
	RemixFromID *string `json:"remix_from_id"`
}

type HighScore struct {
	ID               string    `json:"id" db:"id"`
	GameID           string    `json:"game_id" db:"game_id"`
	UserID           *string   `json:"user_id" db:"user_id"`
	Score            int       `json:"score" db:"score"`
	PlayTimeSeconds  *int      `json:"play_time_seconds" db:"play_time_seconds"`
	AchievedAt       time.Time `json:"achieved_at" db:"achieved_at"`
	Metadata         *string   `json:"metadata" db:"metadata"`
}

type APIServer struct {
	db        *sql.DB
	ollamaURL string
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start retro-game-launcher

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
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

	// Ollama URL - REQUIRED, no defaults
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
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
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")

	server := &APIServer{
		db:        db,
		ollamaURL: ollamaURL,
	}

	router := mux.NewRouter()

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Health check
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	// API routes
	api := router.PathPrefix("/api").Subrouter()
	
	// Games endpoints
	api.HandleFunc("/games", server.getGames).Methods("GET")
	api.HandleFunc("/games", server.createGame).Methods("POST")
	api.HandleFunc("/games/{id}", server.getGame).Methods("GET")
	api.HandleFunc("/games/{id}", server.updateGame).Methods("PUT")
	api.HandleFunc("/games/{id}", server.deleteGame).Methods("DELETE")
	api.HandleFunc("/games/{id}/play", server.recordPlay).Methods("POST")
	api.HandleFunc("/games/{id}/remix", server.createRemix).Methods("POST")
	
	// Game generation
	api.HandleFunc("/generate", server.generateGame).Methods("POST")
	api.HandleFunc("/generate/status/{id}", server.getGenerationStatus).Methods("GET")
	
	// High scores
	api.HandleFunc("/games/{id}/scores", server.getHighScores).Methods("GET")
	api.HandleFunc("/games/{id}/scores", server.submitScore).Methods("POST")
	
	// Search and discovery
	api.HandleFunc("/search/games", server.searchGames).Methods("GET")
	api.HandleFunc("/featured", server.getFeaturedGames).Methods("GET")
	api.HandleFunc("/trending", server.getTrendingGames).Methods("GET")
	
	// Template and prompt management
	api.HandleFunc("/templates", server.getPromptTemplates).Methods("GET")
	api.HandleFunc("/templates/{id}", server.getPromptTemplate).Methods("GET")

	// Users (basic endpoints)
	api.HandleFunc("/users", server.createUser).Methods("POST")
	api.HandleFunc("/users/{id}", server.getUser).Methods("GET")

	log.Printf("🚀 Retro Game Launcher API starting on port %s", port)
	log.Printf("🎮 Database: %s", postgresURL)
	log.Printf("🧠 Ollama: %s", ollamaURL)

	handler := corsHandler(router)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// getEnv removed to prevent hardcoded defaults

func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]interface{}{
			"database": s.checkDatabase(),
			"ollama":   s.checkOllama(),
		},
	}

	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) checkDatabase() string {
	if err := s.db.Ping(); err != nil {
		return "unhealthy"
	}
	return "healthy"
}


func (s *APIServer) checkOllama() string {
	resp, err := http.Get(s.ollamaURL + "/api/tags")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // Drain body to allow connection reuse
	
	if resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) getGames(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE published = true 
		ORDER BY created_at DESC 
		LIMIT 50`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var tagsJSON []byte
		
		err := rows.Scan(
			&game.ID, &game.Title, &game.Prompt, &game.Description,
			&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
			&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
			&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
			&game.UpdatedAt, &game.Published,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Parse tags JSON array
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &game.Tags)
		}

		games = append(games, game)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (s *APIServer) createGame(w http.ResponseWriter, r *http.Request) {
	var game Game
	if err := json.NewDecoder(r.Body).Decode(&game); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate UUID for new game
	game.ID = uuid.New().String()
	game.CreatedAt = time.Now()
	game.UpdatedAt = time.Now()

	tagsJSON, _ := json.Marshal(game.Tags)

	query := `
		INSERT INTO games (id, title, prompt, description, code, engine, 
		                  author_id, parent_game_id, is_remix, thumbnail_url,
		                  tags, created_at, updated_at, published)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, play_count, remix_count, rating`

	err := s.db.QueryRow(query,
		game.ID, game.Title, game.Prompt, game.Description, game.Code,
		game.Engine, game.AuthorID, game.ParentGameID, game.IsRemix,
		game.ThumbnailURL, tagsJSON, game.CreatedAt, game.UpdatedAt, game.Published,
	).Scan(&game.ID, &game.PlayCount, &game.RemixCount, &game.Rating)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(game)
}

func (s *APIServer) getGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE id = $1`

	var game Game
	var tagsJSON []byte

	err := s.db.QueryRow(query, gameID).Scan(
		&game.ID, &game.Title, &game.Prompt, &game.Description,
		&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
		&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
		&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
		&game.UpdatedAt, &game.Published,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse tags JSON array
	if len(tagsJSON) > 0 {
		json.Unmarshal(tagsJSON, &game.Tags)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func (s *APIServer) generateGame(w http.ResponseWriter, r *http.Request) {
	var req GameGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Prompt == "" {
		http.Error(w, "prompt is required", http.StatusBadRequest)
		return
	}
	if req.Engine == "" {
		req.Engine = "html5" // default engine
	}

	// Start game generation process
	generationID := s.startGameGeneration(req)

	response := map[string]interface{}{
		"generation_id":   generationID,
		"status":          "started",
		"prompt":          req.Prompt,
		"estimated_time":  "45-60 seconds",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

func (s *APIServer) recordPlay(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["id"]

	// Increment play count
	query := `UPDATE games SET play_count = play_count + 1 WHERE id = $1`
	_, err := s.db.Exec(query, gameID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "play recorded"})
}

func (s *APIServer) getFeaturedGames(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE published = true AND rating > 4.0
		ORDER BY rating DESC, play_count DESC 
		LIMIT 12`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var tagsJSON []byte
		
		err := rows.Scan(
			&game.ID, &game.Title, &game.Prompt, &game.Description,
			&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
			&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
			&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
			&game.UpdatedAt, &game.Published,
		)
		if err != nil {
			continue
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &game.Tags)
		}

		games = append(games, game)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

func (s *APIServer) searchGames(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		http.Error(w, "Search term required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, title, prompt, description, code, engine, author_id, 
		       parent_game_id, is_remix, thumbnail_url, play_count, 
		       remix_count, rating, tags, created_at, updated_at, published
		FROM games 
		WHERE published = true AND (
			title ILIKE $1 OR 
			description ILIKE $1 OR 
			prompt ILIKE $1 OR
			$2 = ANY(tags)
		)
		ORDER BY 
			CASE WHEN title ILIKE $1 THEN 1 ELSE 2 END,
			play_count DESC
		LIMIT 20`

	searchPattern := "%" + searchTerm + "%"
	rows, err := s.db.Query(query, searchPattern, searchTerm)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var games []Game
	for rows.Next() {
		var game Game
		var tagsJSON []byte
		
		err := rows.Scan(
			&game.ID, &game.Title, &game.Prompt, &game.Description,
			&game.Code, &game.Engine, &game.AuthorID, &game.ParentGameID,
			&game.IsRemix, &game.ThumbnailURL, &game.PlayCount,
			&game.RemixCount, &game.Rating, &tagsJSON, &game.CreatedAt,
			&game.UpdatedAt, &game.Published,
		)
		if err != nil {
			continue
		}

		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &game.Tags)
		}

		games = append(games, game)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

// Stub implementations for remaining endpoints
func (s *APIServer) updateGame(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) deleteGame(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) createRemix(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getGenerationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	generationID := vars["id"]

	status, exists := s.getGenerationStatusByID(generationID)
	if !exists {
		http.Error(w, "Generation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) getHighScores(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) submitScore(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getTrendingGames(w http.ResponseWriter, r *http.Request) {
	s.getFeaturedGames(w, r) // Use featured for now
}

func (s *APIServer) getPromptTemplates(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getPromptTemplate(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) createUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *APIServer) getUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
