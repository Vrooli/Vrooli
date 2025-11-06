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

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Server struct {
	db     *sql.DB
	router *mux.Router
}

type FeatureRequest struct {
	ID           string    `json:"id"`
	ScenarioID   string    `json:"scenario_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	CreatorID    *string   `json:"creator_id,omitempty"`
	VoteCount    int       `json:"vote_count"`
	CommentCount int       `json:"comment_count"`
	Position     int       `json:"position"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	UserVote     *int      `json:"user_vote,omitempty"`
}

type Vote struct {
	ID               string    `json:"id"`
	FeatureRequestID string    `json:"feature_request_id"`
	UserID           *string   `json:"user_id,omitempty"`
	SessionID        *string   `json:"session_id,omitempty"`
	Value            int       `json:"value"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateFeatureRequestRequest struct {
	ScenarioID  string   `json:"scenario_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type UpdateFeatureRequestRequest struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Status      *string  `json:"status,omitempty"`
	Priority    *string  `json:"priority,omitempty"`
	Position    *int     `json:"position,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

type VoteRequest struct {
	Value int `json:"value"`
}

type ListFeatureRequestsQuery struct {
	Status     string `json:"status,omitempty"`
	Sort       string `json:"sort,omitempty"`
	Page       int    `json:"page,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	SearchTerm string `json:"search,omitempty"`
}

func NewServer() (*Server, error) {
	godotenv.Load()

	// ALL database configuration MUST come from environment - no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		return nil, fmt.Errorf("missing required database configuration - ensure POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, and POSTGRES_DB are set")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool for better resilience
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📊 Connecting to: %s:%s/%s as user %s", dbHost, dbPort, dbName, dbUser)

	randSource := rand.New(rand.NewSource(time.Now().UnixNano()))

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
		jitter := time.Duration(randSource.Float64() * jitterRange)
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
		return nil, fmt.Errorf("database connection failed after %d attempts: %w", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	server := &Server{
		db:     db,
		router: mux.NewRouter(),
	}

	server.setupRoutes()
	return server, nil
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Feature request endpoints
	s.router.HandleFunc("/api/v1/scenarios/{scenario_id}/feature-requests", s.handleListFeatureRequests).Methods("GET")
	s.router.HandleFunc("/api/v1/feature-requests", s.handleCreateFeatureRequest).Methods("POST")
	s.router.HandleFunc("/api/v1/feature-requests/{id}", s.handleGetFeatureRequest).Methods("GET")
	s.router.HandleFunc("/api/v1/feature-requests/{id}", s.handleUpdateFeatureRequest).Methods("PUT")
	s.router.HandleFunc("/api/v1/feature-requests/{id}", s.handleDeleteFeatureRequest).Methods("DELETE")
	s.router.HandleFunc("/api/v1/feature-requests/{id}/vote", s.handleVote).Methods("POST")

	// Scenario management endpoints
	s.router.HandleFunc("/api/v1/scenarios", s.handleListScenarios).Methods("GET")
	s.router.HandleFunc("/api/v1/scenarios/{id}", s.handleGetScenario).Methods("GET")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListFeatureRequests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["scenario_id"]

	// Parse query parameters
	status := r.URL.Query().Get("status")
	sort := r.URL.Query().Get("sort")
	if sort == "" {
		sort = "votes"
	}

	query := `
		SELECT id, scenario_id, title, description, status, priority, 
		       creator_id, vote_count, comment_count, position, tags,
		       created_at, updated_at
		FROM feature_requests
		WHERE scenario_id = $1
	`

	args := []interface{}{scenarioID}
	argCount := 1

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	// Add sorting
	switch sort {
	case "date":
		query += " ORDER BY created_at DESC"
	case "priority":
		query += " ORDER BY priority DESC, vote_count DESC"
	default:
		query += " ORDER BY vote_count DESC"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query feature requests")
		return
	}
	defer rows.Close()

	requests := []FeatureRequest{}
	for rows.Next() {
		var fr FeatureRequest
		var tags sql.NullString

		err := rows.Scan(
			&fr.ID, &fr.ScenarioID, &fr.Title, &fr.Description,
			&fr.Status, &fr.Priority, &fr.CreatorID, &fr.VoteCount,
			&fr.CommentCount, &fr.Position, &tags,
			&fr.CreatedAt, &fr.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		if tags.Valid {
			json.Unmarshal([]byte(tags.String), &fr.Tags)
		}

		requests = append(requests, fr)
	}

	response := map[string]interface{}{
		"requests": requests,
		"total":    len(requests),
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateFeatureRequest(w http.ResponseWriter, r *http.Request) {
	var req CreateFeatureRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Title == "" || req.Description == "" || req.ScenarioID == "" {
		writeError(w, http.StatusBadRequest, "Title, description, and scenario_id are required")
		return
	}

	id := uuid.New().String()
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}

	tagsJSON, _ := json.Marshal(req.Tags)

	// Get user ID from auth header if available
	userID := getUserIDFromRequest(r)

	query := `
		INSERT INTO feature_requests (id, scenario_id, title, description, priority, creator_id, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := s.db.QueryRow(query, id, req.ScenarioID, req.Title, req.Description, priority, userID, string(tagsJSON)).
		Scan(&createdAt, &updatedAt)

	if err != nil {
		log.Printf("Error creating feature request: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to create feature request")
		return
	}

	response := map[string]interface{}{
		"id":         id,
		"success":    true,
		"created_at": createdAt,
	}

	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetFeatureRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var fr FeatureRequest
	var tags sql.NullString

	query := `
		SELECT id, scenario_id, title, description, status, priority,
		       creator_id, vote_count, comment_count, position, tags,
		       created_at, updated_at
		FROM feature_requests
		WHERE id = $1
	`

	err := s.db.QueryRow(query, id).Scan(
		&fr.ID, &fr.ScenarioID, &fr.Title, &fr.Description,
		&fr.Status, &fr.Priority, &fr.CreatorID, &fr.VoteCount,
		&fr.CommentCount, &fr.Position, &tags,
		&fr.CreatedAt, &fr.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Feature request not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get feature request")
		return
	}

	if tags.Valid {
		json.Unmarshal([]byte(tags.String), &fr.Tags)
	}

	// Get user's vote if authenticated
	if userID := getUserIDFromRequest(r); userID != nil {
		var vote int
		err := s.db.QueryRow("SELECT value FROM votes WHERE feature_request_id = $1 AND user_id = $2", id, *userID).Scan(&vote)
		if err == nil {
			fr.UserVote = &vote
		}
	}

	writeJSON(w, http.StatusOK, fr)
}

func (s *Server) handleUpdateFeatureRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req UpdateFeatureRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Build dynamic UPDATE query
	updates := []string{}
	args := []interface{}{}
	argCount := 0

	if req.Title != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("title = $%d", argCount))
		args = append(args, *req.Title)
	}

	if req.Description != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("description = $%d", argCount))
		args = append(args, *req.Description)
	}

	if req.Status != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("status = $%d", argCount))
		args = append(args, *req.Status)

		// Track status change
		go s.trackStatusChange(id, *req.Status, getUserIDFromRequest(r))
	}

	if req.Priority != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("priority = $%d", argCount))
		args = append(args, *req.Priority)
	}

	if req.Position != nil {
		argCount++
		updates = append(updates, fmt.Sprintf("position = $%d", argCount))
		args = append(args, *req.Position)
	}

	if len(req.Tags) > 0 {
		tagsJSON, _ := json.Marshal(req.Tags)
		argCount++
		updates = append(updates, fmt.Sprintf("tags = $%d", argCount))
		args = append(args, string(tagsJSON))
	}

	if len(updates) == 0 {
		writeError(w, http.StatusBadRequest, "No fields to update")
		return
	}

	argCount++
	args = append(args, id)

	query := fmt.Sprintf("UPDATE feature_requests SET %s WHERE id = $%d",
		joinStrings(updates, ", "), argCount)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update feature request")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Feature request not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleDeleteFeatureRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	result, err := s.db.Exec("DELETE FROM feature_requests WHERE id = $1", id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete feature request")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		writeError(w, http.StatusNotFound, "Feature request not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	featureRequestID := vars["id"]

	var req VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Value != 1 && req.Value != -1 {
		writeError(w, http.StatusBadRequest, "Vote value must be 1 or -1")
		return
	}

	userID := getUserIDFromRequest(r)
	sessionID := getSessionIDFromRequest(r)

	if userID == nil && sessionID == nil {
		sessionID = generateSessionID()
	}

	// Upsert vote
	query := `
		INSERT INTO votes (id, feature_request_id, user_id, session_id, value)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (feature_request_id, user_id) DO UPDATE SET value = $5
		ON CONFLICT (feature_request_id, session_id) DO UPDATE SET value = $5
	`

	voteID := uuid.New().String()
	_, err := s.db.Exec(query, voteID, featureRequestID, userID, sessionID, req.Value)
	if err != nil {
		log.Printf("Error creating/updating vote: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to record vote")
		return
	}

	// Get updated vote count
	var voteCount int
	err = s.db.QueryRow("SELECT vote_count FROM feature_requests WHERE id = $1", featureRequestID).Scan(&voteCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get vote count")
		return
	}

	response := map[string]interface{}{
		"vote_count": voteCount,
		"user_vote":  req.Value,
		"success":    true,
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, display_name, description, auth_config, created_at
		FROM scenarios
		ORDER BY display_name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query scenarios")
		return
	}
	defer rows.Close()

	scenarios := []map[string]interface{}{}
	for rows.Next() {
		var id, name, displayName string
		var description sql.NullString
		var authConfig string
		var createdAt time.Time

		err := rows.Scan(&id, &name, &displayName, &description, &authConfig, &createdAt)
		if err != nil {
			continue
		}

		scenario := map[string]interface{}{
			"id":           id,
			"name":         name,
			"display_name": displayName,
			"created_at":   createdAt,
		}

		if description.Valid {
			scenario["description"] = description.String
		}

		var authCfg map[string]interface{}
		json.Unmarshal([]byte(authConfig), &authCfg)
		scenario["auth_config"] = authCfg

		scenarios = append(scenarios, scenario)
	}

	writeJSON(w, http.StatusOK, scenarios)
}

func (s *Server) handleGetScenario(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var name, displayName string
	var description sql.NullString
	var authConfig string
	var createdAt time.Time

	query := `
		SELECT name, display_name, description, auth_config, created_at
		FROM scenarios
		WHERE id = $1 OR name = $1
	`

	err := s.db.QueryRow(query, id).Scan(&name, &displayName, &description, &authConfig, &createdAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "Scenario not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get scenario")
		return
	}

	scenario := map[string]interface{}{
		"id":           id,
		"name":         name,
		"display_name": displayName,
		"created_at":   createdAt,
	}

	if description.Valid {
		scenario["description"] = description.String
	}

	var authCfg map[string]interface{}
	json.Unmarshal([]byte(authConfig), &authCfg)
	scenario["auth_config"] = authCfg

	// Get feature request stats
	var totalRequests, openRequests, shippedRequests int
	s.db.QueryRow("SELECT COUNT(*) FROM feature_requests WHERE scenario_id = $1", id).Scan(&totalRequests)
	s.db.QueryRow("SELECT COUNT(*) FROM feature_requests WHERE scenario_id = $1 AND status = 'proposed'", id).Scan(&openRequests)
	s.db.QueryRow("SELECT COUNT(*) FROM feature_requests WHERE scenario_id = $1 AND status = 'shipped'", id).Scan(&shippedRequests)

	scenario["stats"] = map[string]int{
		"total_requests":   totalRequests,
		"open_requests":    openRequests,
		"shipped_requests": shippedRequests,
	}

	writeJSON(w, http.StatusOK, scenario)
}

func (s *Server) trackStatusChange(featureRequestID string, newStatus string, userID *string) {
	// Get current status
	var oldStatus string
	err := s.db.QueryRow("SELECT status FROM feature_requests WHERE id = $1", featureRequestID).Scan(&oldStatus)
	if err != nil || oldStatus == newStatus {
		return
	}

	// Insert status change record
	id := uuid.New().String()
	s.db.Exec(`
		INSERT INTO status_changes (id, feature_request_id, user_id, from_status, to_status)
		VALUES ($1, $2, $3, $4, $5)
	`, id, featureRequestID, userID, oldStatus, newStatus)
}

// Helper functions
func getUserIDFromRequest(r *http.Request) *string {
	// TODO: Integrate with scenario-authenticator
	userID := r.Header.Get("X-User-ID")
	if userID != "" {
		return &userID
	}
	return nil
}

func getSessionIDFromRequest(r *http.Request) *string {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID != "" {
		return &sessionID
	}
	return nil
}

func generateSessionID() *string {
	id := uuid.New().String()
	return &id
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start feature-request-voting

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.db.Close()

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(server.router)

	// API port is required from environment - no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ Missing required API_PORT environment variable")
	}

	log.Printf("📡 Feature voting API starting on port %s", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
// Test change
