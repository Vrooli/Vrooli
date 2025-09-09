package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	_ "github.com/lib/pq"
)

type Server struct {
	db     *sql.DB
	router *mux.Router
}

type SchemaTable struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	ColumnsCount int      `json:"columns_count"`
	Columns      []Column `json:"columns"`
}

type Column struct {
	Name       string  `json:"name"`
	DataType   string  `json:"data_type"`
	IsNullable bool    `json:"is_nullable"`
	IsPrimary  bool    `json:"is_primary"`
	IsForeign  bool    `json:"is_foreign"`
	References *string `json:"references,omitempty"`
}

type Relationship struct {
	FromTable  string `json:"from_table"`
	FromColumn string `json:"from_column"`
	ToTable    string `json:"to_table"`
	ToColumn   string `json:"to_column"`
	Type       string `json:"type"`
}

type SchemaResponse struct {
	Success       bool           `json:"success"`
	DatabaseName  string         `json:"database_name"`
	SchemaName    string         `json:"schema_name"`
	Tables        []SchemaTable  `json:"tables"`
	Relationships []Relationship `json:"relationships"`
	Statistics    SchemaStats    `json:"statistics"`
	Timestamp     time.Time      `json:"timestamp"`
}

type SchemaStats struct {
	TotalTables        int `json:"total_tables"`
	TotalColumns       int `json:"total_columns"`
	TotalRelationships int `json:"total_relationships"`
	TotalIndexes       int `json:"total_indexes"`
}

type QueryGenerateRequest struct {
	NaturalLanguage    string `json:"natural_language"`
	DatabaseContext    string `json:"database_context"`
	IncludeExplanation bool   `json:"include_explanation"`
}

type QueryGenerateResponse struct {
	Success         bool     `json:"success"`
	SQL             string   `json:"sql"`
	Explanation     string   `json:"explanation,omitempty"`
	TablesUsed      []string `json:"tables_used"`
	QueryType       string   `json:"query_type"`
	Confidence      int      `json:"confidence"`
	SimilarQueries  []Query  `json:"similar_queries,omitempty"`
}

type Query struct {
	ID              string    `json:"id"`
	NaturalLanguage string    `json:"natural_language"`
	SQL             string    `json:"sql"`
	UsageCount      int       `json:"usage_count"`
	LastUsed        time.Time `json:"last_used,omitempty"`
}

type QueryExecuteRequest struct {
	SQL          string `json:"sql"`
	DatabaseName string `json:"database_name"`
	Limit        int    `json:"limit"`
}

type QueryExecuteResponse struct {
	Success       bool       `json:"success"`
	Columns       []string   `json:"columns"`
	Rows          [][]interface{} `json:"rows"`
	RowCount      int        `json:"row_count"`
	ExecutionTime float64    `json:"execution_time_ms"`
	Error         string     `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

func NewServer() (*Server, error) {
	// Database configuration - support both POSTGRES_URL and individual components
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		// Try to build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			return nil, fmt.Errorf("❌ Database configuration missing. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
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
	log.Printf("📋 Database URL configured")
	
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
		return nil, fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
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
	s.router.HandleFunc("/api/v1/schema/connect", s.handleSchemaConnect).Methods("POST")
	s.router.HandleFunc("/api/v1/schema/list", s.handleSchemaList).Methods("GET")
	s.router.HandleFunc("/api/v1/schema/export", s.handleSchemaExport).Methods("POST")
	s.router.HandleFunc("/api/v1/schema/diff", s.handleSchemaDiff).Methods("POST")
	s.router.HandleFunc("/api/v1/query/generate", s.handleQueryGenerate).Methods("POST")
	s.router.HandleFunc("/api/v1/query/execute", s.handleQueryExecute).Methods("POST")
	s.router.HandleFunc("/api/v1/query/history", s.handleQueryHistory).Methods("GET")
	s.router.HandleFunc("/api/v1/query/optimize", s.handleQueryOptimize).Methods("POST")
	s.router.HandleFunc("/api/v1/layout/save", s.handleLayoutSave).Methods("POST")
	s.router.HandleFunc("/api/v1/layout/list", s.handleLayoutList).Methods("GET")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	services := make(map[string]string)
	
	// Check database connection
	if err := s.db.Ping(); err != nil {
		services["database"] = "unhealthy"
	} else {
		services["database"] = "healthy"
	}
	
	// Check n8n availability (mock for now)
	services["n8n"] = "healthy"
	services["qdrant"] = "healthy"
	services["ollama"] = "healthy"

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Services:  services,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleSchemaConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnectionString string `json:"connection_string"`
		DatabaseName     string `json:"database_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.DatabaseName == "" {
		req.DatabaseName = "main"
	}

	// Query schema information
	tables, relationships, stats, err := s.getSchemaInfo(req.DatabaseName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := SchemaResponse{
		Success:       true,
		DatabaseName:  req.DatabaseName,
		SchemaName:    "public",
		Tables:        tables,
		Relationships: relationships,
		Statistics:    stats,
		Timestamp:     time.Now(),
	}

	// Save snapshot to database
	s.saveSchemaSnapshot(response)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleSchemaList(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT DISTINCT database_name, schema_name, tables_count, columns_count, snapshot_timestamp
		FROM db_explorer.schema_snapshots
		ORDER BY snapshot_timestamp DESC
		LIMIT 20
	`

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var schemas []map[string]interface{}
	for rows.Next() {
		var dbName, schemaName string
		var tablesCount, columnsCount int
		var timestamp time.Time

		if err := rows.Scan(&dbName, &schemaName, &tablesCount, &columnsCount, &timestamp); err != nil {
			continue
		}

		schemas = append(schemas, map[string]interface{}{
			"database_name": dbName,
			"schema_name":   schemaName,
			"tables_count":  tablesCount,
			"columns_count": columnsCount,
			"last_snapshot": timestamp,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"schemas": schemas,
	})
}

func (s *Server) handleSchemaExport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DatabaseName string `json:"database_name"`
		Format       string `json:"format"` // sql, markdown, json
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// For now, return a simple JSON export
	tables, relationships, stats, err := s.getSchemaInfo(req.DatabaseName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	export := map[string]interface{}{
		"database":      req.DatabaseName,
		"exported_at":   time.Now(),
		"format":        req.Format,
		"tables":        tables,
		"relationships": relationships,
		"statistics":    stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(export)
}

func (s *Server) handleSchemaDiff(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceDatabase string `json:"source"`
		TargetDatabase string `json:"target"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Mock diff for now
	diff := map[string]interface{}{
		"success": true,
		"source":  req.SourceDatabase,
		"target":  req.TargetDatabase,
		"differences": map[string]interface{}{
			"tables_added":   []string{"new_table1", "new_table2"},
			"tables_removed": []string{"old_table1"},
			"tables_modified": []map[string]interface{}{
				{
					"table":          "users",
					"columns_added":  []string{"last_login"},
					"columns_removed": []string{"deprecated_field"},
				},
			},
		},
		"summary": "3 tables added, 1 removed, 1 modified",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(diff)
}

func (s *Server) handleQueryGenerate(w http.ResponseWriter, r *http.Request) {
	var req QueryGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// For now, return a mock response
	// In production, this would call the n8n workflow
	response := QueryGenerateResponse{
		Success:     true,
		SQL:         "SELECT * FROM users WHERE created_at > NOW() - INTERVAL '7 days' LIMIT 100;",
		Explanation: "This query retrieves all user records created in the last 7 days",
		TablesUsed:  []string{"users"},
		QueryType:   "SELECT",
		Confidence:  85,
		SimilarQueries: []Query{
			{
				ID:              uuid.New().String(),
				NaturalLanguage: "Show recent users",
				SQL:             "SELECT * FROM users ORDER BY created_at DESC LIMIT 10;",
				UsageCount:      15,
			},
		},
	}

	// Save to query history
	s.saveQueryHistory(req.NaturalLanguage, response.SQL, req.DatabaseContext)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleQueryExecute(w http.ResponseWriter, r *http.Request) {
	var req QueryExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	startTime := time.Now()
	
	// Execute query (with safety checks in production)
	rows, err := s.db.Query(req.SQL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(QueryExecuteResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch rows
	var results [][]interface{}
	for rows.Next() && len(results) < req.Limit {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}
		results = append(results, values)
	}

	executionTime := time.Since(startTime).Milliseconds()

	response := QueryExecuteResponse{
		Success:       true,
		Columns:       columns,
		Rows:          results,
		RowCount:      len(results),
		ExecutionTime: float64(executionTime),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleQueryHistory(w http.ResponseWriter, r *http.Request) {
	databaseName := r.URL.Query().Get("database")
	if databaseName == "" {
		databaseName = "main"
	}

	query := `
		SELECT id, natural_language, generated_sql, execution_time_ms, 
		       result_count, query_type, user_feedback, created_at
		FROM db_explorer.query_history
		WHERE database_name = $1
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := s.db.Query(query, databaseName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var id, naturalLang, sql, queryType string
		var feedback sql.NullString
		var execTime, resultCount sql.NullInt64
		var createdAt time.Time

		if err := rows.Scan(&id, &naturalLang, &sql, &execTime, 
			&resultCount, &queryType, &feedback, &createdAt); err != nil {
			continue
		}

		item := map[string]interface{}{
			"id":               id,
			"natural_language": naturalLang,
			"sql":              sql,
			"query_type":       queryType,
			"created_at":       createdAt,
		}

		if execTime.Valid {
			item["execution_time_ms"] = execTime.Int64
		}
		if resultCount.Valid {
			item["result_count"] = resultCount.Int64
		}
		if feedback.Valid {
			item["user_feedback"] = feedback.String
		}

		history = append(history, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"history": history,
	})
}

func (s *Server) handleQueryOptimize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SQL          string `json:"sql"`
		DatabaseName string `json:"database_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Mock optimization suggestions
	response := map[string]interface{}{
		"success": true,
		"original_sql": req.SQL,
		"optimizations": []map[string]interface{}{
			{
				"type":        "INDEX",
				"severity":    "HIGH",
				"description": "Missing index on frequently queried column",
				"suggestion":  "CREATE INDEX idx_users_created_at ON users(created_at);",
			},
			{
				"type":        "QUERY",
				"severity":    "MEDIUM",
				"description": "SELECT * fetches unnecessary columns",
				"suggestion":  "Specify only required columns",
			},
		},
		"optimized_sql": "SELECT id, name, email FROM users WHERE created_at > NOW() - INTERVAL '7 days' LIMIT 100;",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleLayoutSave(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string          `json:"name"`
		DatabaseName string          `json:"database_name"`
		LayoutType   string          `json:"layout_type"`
		LayoutData   json.RawMessage `json:"layout_data"`
		IsShared     bool            `json:"is_shared"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := uuid.New()
	query := `
		INSERT INTO db_explorer.visualization_layouts 
		(id, name, database_name, layout_type, layout_data, is_shared)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.Exec(query, id, req.Name, req.DatabaseName, 
		req.LayoutType, req.LayoutData, req.IsShared)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      id.String(),
		"message": "Layout saved successfully",
	})
}

func (s *Server) handleLayoutList(w http.ResponseWriter, r *http.Request) {
	databaseName := r.URL.Query().Get("database")
	
	query := `
		SELECT id, name, layout_type, is_shared, created_at
		FROM db_explorer.visualization_layouts
		WHERE ($1 = '' OR database_name = $1)
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query, databaseName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var layouts []map[string]interface{}
	for rows.Next() {
		var id, name, layoutType string
		var isShared bool
		var createdAt time.Time

		if err := rows.Scan(&id, &name, &layoutType, &isShared, &createdAt); err != nil {
			continue
		}

		layouts = append(layouts, map[string]interface{}{
			"id":          id,
			"name":        name,
			"layout_type": layoutType,
			"is_shared":   isShared,
			"created_at":  createdAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"layouts": layouts,
	})
}

// Helper functions
func (s *Server) getSchemaInfo(databaseName string) ([]SchemaTable, []Relationship, SchemaStats, error) {
	// Mock implementation - in production, this would query the actual database
	tables := []SchemaTable{
		{
			Name:         "users",
			Type:         "BASE TABLE",
			ColumnsCount: 8,
			Columns: []Column{
				{Name: "id", DataType: "uuid", IsNullable: false, IsPrimary: true},
				{Name: "email", DataType: "varchar", IsNullable: false},
				{Name: "created_at", DataType: "timestamp", IsNullable: false},
			},
		},
		{
			Name:         "projects",
			Type:         "BASE TABLE",
			ColumnsCount: 6,
			Columns: []Column{
				{Name: "id", DataType: "uuid", IsNullable: false, IsPrimary: true},
				{Name: "name", DataType: "varchar", IsNullable: false},
				{Name: "owner_id", DataType: "uuid", IsNullable: false, IsForeign: true},
			},
		},
	}

	relationships := []Relationship{
		{
			FromTable:  "projects",
			FromColumn: "owner_id",
			ToTable:    "users",
			ToColumn:   "id",
			Type:       "foreign_key",
		},
	}

	stats := SchemaStats{
		TotalTables:        len(tables),
		TotalColumns:       14,
		TotalRelationships: len(relationships),
		TotalIndexes:       5,
	}

	return tables, relationships, stats, nil
}

func (s *Server) saveSchemaSnapshot(schema SchemaResponse) error {
	schemaJSON, _ := json.Marshal(schema)
	
	query := `
		INSERT INTO db_explorer.schema_snapshots 
		(database_name, schema_name, tables_count, columns_count, relationships_count, schema_data)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	_, err := s.db.Exec(query, 
		schema.DatabaseName,
		schema.SchemaName,
		schema.Statistics.TotalTables,
		schema.Statistics.TotalColumns,
		schema.Statistics.TotalRelationships,
		schemaJSON,
	)
	
	return err
}

func (s *Server) saveQueryHistory(naturalLanguage, sql, databaseName string) error {
	query := `
		INSERT INTO db_explorer.query_history 
		(natural_language, generated_sql, database_name, query_type)
		VALUES ($1, $2, $3, 'SELECT')
	`
	
	_, err := s.db.Exec(query, naturalLanguage, sql, databaseName)
	return err
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
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
	
	log.Printf("Database Schema Explorer API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}