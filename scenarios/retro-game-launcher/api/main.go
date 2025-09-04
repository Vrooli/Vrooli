package main

import (
	"database/sql"
	"encoding/json"
	"log"
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
	db       *sql.DB
	n8nURL   string
	ollamaURL string
}

func main() {
	port := getEnv("API_PORT", getEnv("PORT", ""))

	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		postgresURL = "postgres://postgres:postgres@localhost:5432/retro_games?sslmode=disable"
	}

	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL == "" {
		n8nURL = "http://localhost:5678"
	}

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Connect to database
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	server := &APIServer{
		db:        db,
		n8nURL:    n8nURL,
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
	log.Printf("🤖 n8n: %s", n8nURL)
	log.Printf("🧠 Ollama: %s", ollamaURL)

	handler := corsHandler(router)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]interface{}{
			"database": s.checkDatabase(),
			"n8n": s.checkN8N(),
			"ollama": s.checkOllama(),
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

func (s *APIServer) checkN8N() string {
	resp, err := http.Get(s.n8nURL + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		return "unavailable"
	}
	return "healthy"
}

func (s *APIServer) checkOllama() string {
	resp, err := http.Get(s.ollamaURL + "/api/tags")
	if err != nil || resp.StatusCode != http.StatusOK {
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

	// Generate unique generation ID
	generationID := uuid.New().String()

	// For now, return immediately - in production this would trigger n8n workflow
	response := map[string]interface{}{
		"generation_id": generationID,
		"status": "started",
		"prompt": req.Prompt,
		"estimated_time": "30-60 seconds",
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
	w.WriteHeader(http.StatusNotImplemented)
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
