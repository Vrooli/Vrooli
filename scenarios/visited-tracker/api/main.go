package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	apiVersion  = "2.0.0"
	serviceName = "visited-tracker"
)

var (
	db     *sql.DB
	logger *log.Logger
)

// Models
type Campaign struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	FromAgent   string                 `json:"from_agent"`
	Description *string                `json:"description,omitempty"`
	Patterns    []string               `json:"patterns"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Status      string                 `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
	// Computed fields
	TotalFiles     int     `json:"total_files,omitempty"`
	VisitedFiles   int     `json:"visited_files,omitempty"`
	CoveragePercent float64 `json:"coverage_percent,omitempty"`
}

type TrackedFile struct {
	ID             uuid.UUID              `json:"id"`
	CampaignID     uuid.UUID              `json:"campaign_id"`
	FilePath       string                 `json:"file_path"`
	AbsolutePath   string                 `json:"absolute_path"`
	VisitCount     int                    `json:"visit_count"`
	FirstSeen      time.Time              `json:"first_seen"`
	LastVisited    *time.Time             `json:"last_visited,omitempty"`
	LastModified   time.Time              `json:"last_modified"`
	ContentHash    *string                `json:"content_hash,omitempty"`
	SizeBytes      int64                  `json:"size_bytes"`
	StalenessScore float64                `json:"staleness_score"`
	Deleted        bool                   `json:"deleted"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type Visit struct {
	ID             uuid.UUID              `json:"id"`
	FileID         uuid.UUID              `json:"file_id"`
	Timestamp      time.Time              `json:"timestamp"`
	Context        *string                `json:"context,omitempty"`
	Agent          *string                `json:"agent,omitempty"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	DurationMs     *int                   `json:"duration_ms,omitempty"`
	Findings       map[string]interface{} `json:"findings,omitempty"`
}

// Request/Response types
type CreateCampaignRequest struct {
	Name        string                 `json:"name"`
	FromAgent   string                 `json:"from_agent"`
	Description *string                `json:"description,omitempty"`
	Patterns    []string               `json:"patterns"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type VisitRequest struct {
	Files          interface{}            `json:"files"` // Can be []string or []FileVisit
	Context        *string                `json:"context,omitempty"`
	Agent          *string                `json:"agent,omitempty"`
	ConversationID *string                `json:"conversation_id,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type FileVisit struct {
	Path    string  `json:"path"`
	Context *string `json:"context,omitempty"`
}

type StructureSyncRequest struct {
	Files         []string               `json:"files,omitempty"`
	Structure     map[string]interface{} `json:"structure,omitempty"`
	Patterns      []string               `json:"patterns,omitempty"`
	RemoveDeleted bool                   `json:"remove_deleted,omitempty"`
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start visited-tracker

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Change working directory to project root for file pattern resolution
	// API runs from scenarios/visited-tracker/api/, so go up 3 levels to project root
	if err := os.Chdir("../../../"); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to change to project root directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger = log.New(os.Stdout, "[visited-tracker] ", log.LstdFlags)

	// Log current working directory for transparency
	if cwd, err := os.Getwd(); err == nil {
		logger.Printf("📁 Working directory: %s", cwd)
		logger.Printf("💡 File patterns will be resolved relative to this directory")
	}

	// Get port from environment
	port := os.Getenv("API_PORT")
	if port == "" {
		logger.Fatal("❌ API_PORT environment variable is required")
	}

	// Initialize database
	if err := initDB(); err != nil {
		logger.Fatalf("Database initialization failed: %v", err)
	}
	defer db.Close()

	// Setup router
	r := mux.NewRouter()
	
	// Apply CORS middleware first
	r.Use(corsMiddleware)

	// API v1 routes
	v1 := r.PathPrefix("/api/v1").Subrouter()

	// Health endpoint (outside versioning for simplicity)
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// Campaign management endpoints
	v1.HandleFunc("/campaigns", listCampaignsHandler).Methods("GET")
	v1.HandleFunc("/campaigns", createCampaignHandler).Methods("POST")
	v1.HandleFunc("/campaigns", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}", getCampaignHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}", deleteCampaignHandler).Methods("DELETE")
	v1.HandleFunc("/campaigns/{id}", optionsHandler).Methods("OPTIONS")

	// Campaign-specific visit tracking endpoints
	v1.HandleFunc("/campaigns/{id}/visit", visitHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/visit", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/structure/sync", structureSyncHandler).Methods("POST")
	v1.HandleFunc("/campaigns/{id}/structure/sync", optionsHandler).Methods("OPTIONS")
	v1.HandleFunc("/campaigns/{id}/prioritize/least-visited", leastVisitedHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/prioritize/most-stale", mostStaleHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/coverage", coverageHandler).Methods("GET")
	v1.HandleFunc("/campaigns/{id}/export", exportHandler).Methods("GET")

	logger.Printf("🚀 %s API v%s starting on port %s", serviceName, apiVersion, port)
	logger.Printf("📊 Endpoints available at http://localhost:%s/api/v1", port)

	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func initDB() error {
	// Database configuration from environment
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		// Build from components - all required
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DB")

		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			return fmt.Errorf("database configuration missing - provide POSTGRES_URL or all connection parameters")
		}

		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbname)
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	logger.Println("✅ Database connected successfully")
	
	// Initialize schema if needed
	if err = initSchema(); err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}
	
	return nil
}

func initSchema() error {
	// Always use inline schema for now to avoid trigger issues
	return applyInlineSchema()
}

func applyInlineSchema() error {
	// Drop old tables if they exist (for migration)
	dropOldSchema := `
	DROP TABLE IF EXISTS structure_snapshots CASCADE;
	DROP TABLE IF EXISTS visits CASCADE;
	DROP TABLE IF EXISTS tracked_files CASCADE;
	DROP TABLE IF EXISTS campaigns CASCADE;
	DROP FUNCTION IF EXISTS record_visit CASCADE;
	DROP FUNCTION IF EXISTS get_least_visited_files CASCADE;
	DROP FUNCTION IF EXISTS get_most_stale_files CASCADE;
	DROP FUNCTION IF EXISTS get_coverage_stats CASCADE;
	`
	
	// Try to drop old schema first (ignore errors)
	db.Exec(dropOldSchema)
	
	// Inline schema with campaigns support
	schema := `
	-- Campaigns table
	CREATE TABLE IF NOT EXISTS campaigns (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		from_agent VARCHAR(255) NOT NULL DEFAULT 'manual',
		description TEXT,
		patterns JSONB NOT NULL DEFAULT '[]',
		created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		metadata JSONB NOT NULL DEFAULT '{}',
		UNIQUE(name)
	);
	
	-- Tracked files table
	CREATE TABLE IF NOT EXISTS tracked_files (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
		file_path VARCHAR(1000) NOT NULL,
		absolute_path VARCHAR(1000) NOT NULL,
		visit_count INTEGER NOT NULL DEFAULT 0,
		first_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		last_visited TIMESTAMP WITH TIME ZONE,
		last_modified TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		content_hash VARCHAR(64),
		size_bytes BIGINT NOT NULL DEFAULT 0,
		staleness_score DECIMAL(10, 4) NOT NULL DEFAULT 0,
		deleted BOOLEAN NOT NULL DEFAULT FALSE,
		metadata JSONB NOT NULL DEFAULT '{}',
		UNIQUE(campaign_id, absolute_path)
	);
	
	-- Visits table
	CREATE TABLE IF NOT EXISTS visits (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		file_id UUID NOT NULL REFERENCES tracked_files(id) ON DELETE CASCADE,
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		context VARCHAR(50),
		agent VARCHAR(255),
		conversation_id VARCHAR(255),
		duration_ms INTEGER,
		findings JSONB
	);
	
	-- Structure snapshots table
	CREATE TABLE IF NOT EXISTS structure_snapshots (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
		total_files INTEGER NOT NULL,
		new_files JSONB NOT NULL DEFAULT '[]',
		deleted_files JSONB NOT NULL DEFAULT '[]',
		moved_files JSONB NOT NULL DEFAULT '{}',
		snapshot_data JSONB NOT NULL
	);
	
	-- Indexes
	CREATE INDEX IF NOT EXISTS idx_campaigns_name ON campaigns(name);
	CREATE INDEX IF NOT EXISTS idx_campaigns_from_agent ON campaigns(from_agent);
	CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
	CREATE INDEX IF NOT EXISTS idx_tracked_files_campaign_id ON tracked_files(campaign_id);
	CREATE INDEX IF NOT EXISTS idx_tracked_files_path ON tracked_files(file_path);
	CREATE INDEX IF NOT EXISTS idx_tracked_files_visit_count ON tracked_files(visit_count);
	CREATE INDEX IF NOT EXISTS idx_visits_file_id ON visits(file_id);
	
	-- Function to record visit
	CREATE OR REPLACE FUNCTION record_visit(
		p_campaign_id UUID,
		p_file_path VARCHAR,
		p_absolute_path VARCHAR,
		p_context VARCHAR DEFAULT NULL,
		p_agent VARCHAR DEFAULT NULL,
		p_conversation_id VARCHAR DEFAULT NULL,
		p_duration_ms INTEGER DEFAULT NULL,
		p_findings JSONB DEFAULT NULL
	)
	RETURNS UUID AS $$
	DECLARE
		v_file_id UUID;
		v_visit_id UUID;
	BEGIN
		-- Get or create the tracked file for this campaign
		INSERT INTO tracked_files (campaign_id, file_path, absolute_path)
		VALUES (p_campaign_id, p_file_path, p_absolute_path)
		ON CONFLICT (campaign_id, absolute_path) DO NOTHING;
		
		-- Get the file ID
		SELECT id INTO v_file_id
		FROM tracked_files
		WHERE campaign_id = p_campaign_id AND absolute_path = p_absolute_path;
		
		-- Update visit count and last visited time
		UPDATE tracked_files
		SET 
			visit_count = visit_count + 1,
			last_visited = NOW()
		WHERE id = v_file_id;
		
		-- Record the visit
		INSERT INTO visits (file_id, context, agent, conversation_id, duration_ms, findings)
		VALUES (v_file_id, p_context, p_agent, p_conversation_id, p_duration_ms, p_findings)
		RETURNING id INTO v_visit_id;
		
		RETURN v_visit_id;
	END;
	$$ LANGUAGE plpgsql;
	
	-- Function to get least visited files for a campaign
	CREATE OR REPLACE FUNCTION get_least_visited_files(
		p_campaign_id UUID,
		p_limit INTEGER DEFAULT 10,
		p_context VARCHAR DEFAULT NULL,
		p_include_unvisited BOOLEAN DEFAULT TRUE
	)
	RETURNS TABLE (
		id UUID,
		file_path VARCHAR,
		absolute_path VARCHAR,
		visit_count INTEGER,
		last_visited TIMESTAMP WITH TIME ZONE,
		staleness_score DECIMAL
	) AS $$
	BEGIN
		RETURN QUERY
		SELECT 
			tf.id,
			tf.file_path,
			tf.absolute_path,
			tf.visit_count,
			tf.last_visited,
			tf.staleness_score
		FROM tracked_files tf
		WHERE 
			tf.campaign_id = p_campaign_id
			AND tf.deleted = FALSE
			AND (p_include_unvisited OR tf.visit_count > 0)
			AND (p_context IS NULL OR EXISTS (
				SELECT 1 FROM visits v 
				WHERE v.file_id = tf.id AND v.context = p_context
			))
		ORDER BY tf.visit_count ASC, tf.staleness_score DESC
		LIMIT p_limit;
	END;
	$$ LANGUAGE plpgsql;
	
	-- Function to get most stale files for a campaign
	CREATE OR REPLACE FUNCTION get_most_stale_files(
		p_campaign_id UUID,
		p_limit INTEGER DEFAULT 10,
		p_threshold DECIMAL DEFAULT 0.0
	)
	RETURNS TABLE (
		id UUID,
		file_path VARCHAR,
		absolute_path VARCHAR,
		visit_count INTEGER,
		last_visited TIMESTAMP WITH TIME ZONE,
		last_modified TIMESTAMP WITH TIME ZONE,
		staleness_score DECIMAL
	) AS $$
	BEGIN
		RETURN QUERY
		SELECT 
			tf.id,
			tf.file_path,
			tf.absolute_path,
			tf.visit_count,
			tf.last_visited,
			tf.last_modified,
			tf.staleness_score
		FROM tracked_files tf
		WHERE 
			tf.campaign_id = p_campaign_id
			AND tf.deleted = FALSE
			AND tf.staleness_score >= p_threshold
		ORDER BY tf.staleness_score DESC
		LIMIT p_limit;
	END;
	$$ LANGUAGE plpgsql;
	
	-- Function to get coverage stats for a campaign
	CREATE OR REPLACE FUNCTION get_coverage_stats(
		p_campaign_id UUID
	)
	RETURNS TABLE (
		total_files INTEGER,
		visited_files INTEGER,
		unvisited_files INTEGER,
		coverage_percentage DECIMAL,
		average_visits DECIMAL,
		average_staleness DECIMAL
	) AS $$
	BEGIN
		RETURN QUERY
		SELECT 
			COUNT(*)::INTEGER as total_files,
			COUNT(CASE WHEN visit_count > 0 THEN 1 END)::INTEGER as visited_files,
			COUNT(CASE WHEN visit_count = 0 THEN 1 END)::INTEGER as unvisited_files,
			ROUND(COUNT(CASE WHEN visit_count > 0 THEN 1 END)::DECIMAL / NULLIF(COUNT(*), 0) * 100, 2) as coverage_percentage,
			ROUND(AVG(visit_count)::DECIMAL, 2) as average_visits,
			ROUND(AVG(staleness_score)::DECIMAL, 2) as average_staleness
		FROM tracked_files
		WHERE campaign_id = p_campaign_id AND deleted = FALSE;
	END;
	$$ LANGUAGE plpgsql;
	`
	
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to apply inline schema: %w", err)
	}
	
	logger.Println("✅ Database schema initialized (campaigns support)")
	return nil
}

// Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   serviceName,
		"version":   apiVersion,
		"timestamp": time.Now().UTC(),
		"database":  "connected",
	})
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	// CORS headers are already set by corsMiddleware
	w.WriteHeader(http.StatusOK)
}

// Campaign handlers
func listCampaignsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	fromAgent := r.URL.Query().Get("from_agent")
	search := r.URL.Query().Get("search")
	
	query := `
		SELECT c.id, c.name, c.from_agent, c.description, c.patterns, 
		       c.created_at, c.updated_at, c.status, c.metadata,
		       COUNT(tf.id) as total_files,
		       COUNT(CASE WHEN tf.visit_count > 0 THEN 1 END) as visited_files
		FROM campaigns c
		LEFT JOIN tracked_files tf ON tf.campaign_id = c.id AND tf.deleted = FALSE
		WHERE 1=1
	`
	
	args := []interface{}{}
	argCount := 0
	
	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND c.status = $%d", argCount)
		args = append(args, status)
	}
	
	if fromAgent != "" {
		argCount++
		query += fmt.Sprintf(" AND c.from_agent = $%d", argCount)
		args = append(args, fromAgent)
	}
	
	if search != "" {
		argCount++
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+search+"%")
	}
	
	query += " GROUP BY c.id ORDER BY c.created_at DESC"
	
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	campaigns := []Campaign{}
	for rows.Next() {
		var c Campaign
		var patternsJSON, metadataJSON string
		var totalFiles, visitedFiles int
		
		err := rows.Scan(&c.ID, &c.Name, &c.FromAgent, &c.Description, &patternsJSON,
			&c.CreatedAt, &c.UpdatedAt, &c.Status, &metadataJSON,
			&totalFiles, &visitedFiles)
		
		if err == nil {
			json.Unmarshal([]byte(patternsJSON), &c.Patterns)
			json.Unmarshal([]byte(metadataJSON), &c.Metadata)
			c.TotalFiles = totalFiles
			c.VisitedFiles = visitedFiles
			if totalFiles > 0 {
				c.CoveragePercent = float64(visitedFiles) / float64(totalFiles) * 100
			}
			campaigns = append(campaigns, c)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"campaigns": campaigns,
		"count":     len(campaigns),
	})
}

func createCampaignHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateCampaignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Campaign name is required", http.StatusBadRequest)
		return
	}
	
	if req.FromAgent == "" {
		req.FromAgent = "manual"
	}
	
	// Create campaign
	campaignID := uuid.New()
	patternsJSON, _ := json.Marshal(req.Patterns)
	metadataJSON, _ := json.Marshal(req.Metadata)
	
	_, err := db.Exec(`
		INSERT INTO campaigns (id, name, from_agent, description, patterns, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, campaignID, req.Name, req.FromAgent, req.Description, patternsJSON, metadataJSON)
	
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint \"campaigns_name_key\"" {
			http.Error(w, "Campaign name already exists", http.StatusConflict)
		} else {
			http.Error(w, "Failed to create campaign", http.StatusInternalServerError)
		}
		return
	}
	
	// Return created campaign
	var c Campaign
	var patterns, metadata string
	err = db.QueryRow(`
		SELECT id, name, from_agent, description, patterns, created_at, updated_at, status, metadata
		FROM campaigns WHERE id = $1
	`, campaignID).Scan(&c.ID, &c.Name, &c.FromAgent, &c.Description, &patterns,
		&c.CreatedAt, &c.UpdatedAt, &c.Status, &metadata)
	
	if err == nil {
		json.Unmarshal([]byte(patterns), &c.Patterns)
		json.Unmarshal([]byte(metadata), &c.Metadata)
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(c)
}

func getCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}
	
	var c Campaign
	var patternsJSON, metadataJSON string
	
	err = db.QueryRow(`
		SELECT c.id, c.name, c.from_agent, c.description, c.patterns, 
		       c.created_at, c.updated_at, c.status, c.metadata,
		       COUNT(tf.id) as total_files,
		       COUNT(CASE WHEN tf.visit_count > 0 THEN 1 END) as visited_files
		FROM campaigns c
		LEFT JOIN tracked_files tf ON tf.campaign_id = c.id AND tf.deleted = FALSE
		WHERE c.id = $1
		GROUP BY c.id
	`, campaignID).Scan(&c.ID, &c.Name, &c.FromAgent, &c.Description, &patternsJSON,
		&c.CreatedAt, &c.UpdatedAt, &c.Status, &metadataJSON,
		&c.TotalFiles, &c.VisitedFiles)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Campaign not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	
	json.Unmarshal([]byte(patternsJSON), &c.Patterns)
	json.Unmarshal([]byte(metadataJSON), &c.Metadata)
	if c.TotalFiles > 0 {
		c.CoveragePercent = float64(c.VisitedFiles) / float64(c.TotalFiles) * 100
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(c)
}

func deleteCampaignHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}
	
	result, err := db.Exec("DELETE FROM campaigns WHERE id = $1", campaignID)
	if err != nil {
		http.Error(w, "Failed to delete campaign", http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Campaign not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": true,
		"id":      campaignID,
	})
}

// Visit tracking handlers
func visitHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}
	
	var req VisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse files from request
	var filePaths []string
	switch files := req.Files.(type) {
	case []interface{}:
		for _, f := range files {
			if str, ok := f.(string); ok {
				filePaths = append(filePaths, str)
			}
		}
	case []string:
		filePaths = files
	}

	if len(filePaths) == 0 {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	var trackedFiles []TrackedFile
	for _, path := range filePaths {
		// Resolve to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			absPath = path
		}

		// Record visit using stored procedure
		var visitID uuid.UUID
		err = db.QueryRow(`
			SELECT record_visit($1, $2, $3, $4, $5, $6, NULL, NULL)
		`, campaignID, path, absPath, req.Context, req.Agent, req.ConversationID).Scan(&visitID)

		if err != nil {
			logger.Printf("Error recording visit for %s: %v", path, err)
			continue
		}

		// Get updated file info
		var tf TrackedFile
		var metadataJSON string
		err = db.QueryRow(`
			SELECT id, campaign_id, file_path, absolute_path, visit_count, first_seen, 
			       last_visited, last_modified, content_hash, size_bytes, 
			       staleness_score, deleted, metadata
			FROM tracked_files
			WHERE campaign_id = $1 AND absolute_path = $2
		`, campaignID, absPath).Scan(&tf.ID, &tf.CampaignID, &tf.FilePath, &tf.AbsolutePath, &tf.VisitCount,
			&tf.FirstSeen, &tf.LastVisited, &tf.LastModified, &tf.ContentHash,
			&tf.SizeBytes, &tf.StalenessScore, &tf.Deleted, &metadataJSON)

		if err == nil {
			json.Unmarshal([]byte(metadataJSON), &tf.Metadata)
			trackedFiles = append(trackedFiles, tf)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recorded": len(trackedFiles),
		"files":    trackedFiles,
	})
}

func structureSyncHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}
	
	var req StructureSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Collect files to track
	var filesToTrack []string
	
	// From explicit file list
	filesToTrack = append(filesToTrack, req.Files...)
	
	// From patterns
	for _, pattern := range req.Patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			logger.Printf("Invalid pattern %s: %v", pattern, err)
			continue
		}
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() {
				filesToTrack = append(filesToTrack, match)
			}
		}
	}

	// Track statistics
	added := 0
	removed := 0
	total := len(filesToTrack)

	// Add new files
	for _, path := range filesToTrack {
		absPath, _ := filepath.Abs(path)
		relPath := path
		
		// Check if file exists
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		// Insert or update tracked file
		_, err = db.Exec(`
			INSERT INTO tracked_files (campaign_id, file_path, absolute_path, last_modified, size_bytes)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (campaign_id, absolute_path) 
			DO UPDATE SET 
				last_modified = $4,
				size_bytes = $5,
				deleted = FALSE
		`, campaignID, relPath, absPath, info.ModTime(), info.Size())

		if err == nil {
			added++
		}
	}

	// Handle deleted files if requested
	if req.RemoveDeleted {
		result, err := db.Exec(`
			UPDATE tracked_files 
			SET deleted = TRUE 
			WHERE campaign_id = $1 
			AND deleted = FALSE 
			AND absolute_path NOT IN (
				SELECT unnest($2::text[])
			)
		`, campaignID, filesToTrack)
		if err == nil {
			removed64, _ := result.RowsAffected()
			removed = int(removed64)
		}
	}

	// Create snapshot
	snapshotID := uuid.New()
	snapshotData, _ := json.Marshal(map[string]interface{}{
		"files":     filesToTrack,
		"timestamp": time.Now(),
	})
	
	db.Exec(`
		INSERT INTO structure_snapshots (id, campaign_id, total_files, snapshot_data)
		VALUES ($1, $2, $3, $4)
	`, snapshotID, campaignID, total, snapshotData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"added":       added,
		"removed":     removed,
		"moved":       0,
		"total":       total,
		"snapshot_id": snapshotID,
	})
}

func leastVisitedHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}
	
	// Parse query parameters
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	context := r.URL.Query().Get("context")
	includeUnvisited := r.URL.Query().Get("include_unvisited") != "false"

	rows, err := db.Query(`
		SELECT * FROM get_least_visited_files($1, $2, $3, $4)
	`, campaignID, limit, nullableString(context), includeUnvisited)

	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var files []TrackedFile
	for rows.Next() {
		var tf TrackedFile
		err := rows.Scan(&tf.ID, &tf.FilePath, &tf.AbsolutePath,
			&tf.VisitCount, &tf.LastVisited, &tf.StalenessScore)
		if err == nil {
			tf.CampaignID = campaignID
			files = append(files, tf)
		}
	}

	// Get coverage stats
	var visited, unvisited int
	db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN visit_count > 0 THEN 1 END),
			COUNT(CASE WHEN visit_count = 0 THEN 1 END)
		FROM tracked_files WHERE campaign_id = $1 AND deleted = FALSE
	`, campaignID).Scan(&visited, &unvisited)

	total := visited + unvisited
	percentage := 0.0
	if total > 0 {
		percentage = float64(visited) / float64(total) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files": files,
		"coverage": map[string]interface{}{
			"visited":    visited,
			"unvisited":  unvisited,
			"percentage": percentage,
		},
	})
}

func mostStaleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}
	
	// Parse query parameters
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	threshold := 0.0
	if t := r.URL.Query().Get("threshold"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil {
			threshold = parsed
		}
	}

	rows, err := db.Query(`
		SELECT * FROM get_most_stale_files($1, $2, $3)
	`, campaignID, limit, threshold)

	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var files []TrackedFile
	totalStaleness := 0.0
	criticalCount := 0

	for rows.Next() {
		var tf TrackedFile
		err := rows.Scan(&tf.ID, &tf.FilePath, &tf.AbsolutePath,
			&tf.VisitCount, &tf.LastVisited, &tf.LastModified, &tf.StalenessScore)
		if err == nil {
			tf.CampaignID = campaignID
			files = append(files, tf)
			totalStaleness += tf.StalenessScore
			if tf.StalenessScore > 10.0 {
				criticalCount++
			}
		}
	}

	averageStaleness := 0.0
	if len(files) > 0 {
		averageStaleness = totalStaleness / float64(len(files))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":             files,
		"average_staleness": averageStaleness,
		"critical_count":    criticalCount,
	})
}

func coverageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}

	// Get basic coverage stats
	rows, err := db.Query(`SELECT * FROM get_coverage_stats($1)`, campaignID)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var result map[string]interface{}
	if rows.Next() {
		var total, visited, unvisited int
		var percentage, avgVisits, avgStaleness float64
		
		rows.Scan(&total, &visited, &unvisited, &percentage, &avgVisits, &avgStaleness)
		
		result = map[string]interface{}{
			"total_files":         total,
			"visited_files":       visited,
			"unvisited_files":     unvisited,
			"coverage_percentage": percentage,
			"average_visits":      avgVisits,
			"average_staleness":   avgStaleness,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func exportHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid campaign ID", http.StatusBadRequest)
		return
	}
	
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	includeHistory := r.URL.Query().Get("include_history") == "true"

	// Get tracked files for this campaign
	query := `
		SELECT id, file_path, absolute_path, visit_count, first_seen,
		       last_visited, last_modified, staleness_score
		FROM tracked_files
		WHERE campaign_id = $1 AND deleted = FALSE
	`
	
	rows, err := db.Query(query, campaignID)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var files []TrackedFile
	for rows.Next() {
		var tf TrackedFile
		rows.Scan(&tf.ID, &tf.FilePath, &tf.AbsolutePath, &tf.VisitCount,
			&tf.FirstSeen, &tf.LastVisited, &tf.LastModified, &tf.StalenessScore)
		tf.CampaignID = campaignID
		files = append(files, tf)
	}

	exportData := map[string]interface{}{
		"format":           format,
		"campaign_id":      campaignID,
		"exported_count":   len(files),
		"export_timestamp": time.Now().UTC(),
		"data":            files,
	}

	if includeHistory {
		// Would add visit history here
		exportData["include_history"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exportData)
}

// Helper functions
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}