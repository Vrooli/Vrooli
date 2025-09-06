package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// =============================================================================
// MODELS
// =============================================================================

type Person struct {
	ID                      string                 `json:"id" db:"id"`
	CreatedAt               time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at" db:"updated_at"`
	FullName                string                 `json:"full_name" db:"full_name"`
	DisplayName             *string                `json:"display_name" db:"display_name"`
	Nicknames               []string               `json:"nicknames" db:"nicknames"`
	Pronouns                *string                `json:"pronouns" db:"pronouns"`
	Emails                  []string               `json:"emails" db:"emails"`
	Phones                  []string               `json:"phones" db:"phones"`
	ScenarioAuthenticatorID *string                `json:"scenario_authenticator_id" db:"scenario_authenticator_id"`
	Metadata                map[string]interface{} `json:"metadata" db:"metadata"`
	Tags                    []string               `json:"tags" db:"tags"`
	Notes                   *string                `json:"notes" db:"notes"`
	SocialProfiles          map[string]interface{} `json:"social_profiles" db:"social_profiles"`
	ComputedSignals         map[string]interface{} `json:"computed_signals" db:"computed_signals"`
	DeletedAt               *time.Time             `json:"deleted_at" db:"deleted_at"`
}

type Relationship struct {
	ID                     string                 `json:"id" db:"id"`
	CreatedAt              time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at" db:"updated_at"`
	FromPersonID           string                 `json:"from_person_id" db:"from_person_id"`
	ToPersonID             string                 `json:"to_person_id" db:"to_person_id"`
	RelationshipType       string                 `json:"relationship_type" db:"relationship_type"`
	Strength               *float64               `json:"strength" db:"strength"`
	RecencyScore           *float64               `json:"recency_score" db:"recency_score"`
	LastContactDate        *time.Time             `json:"last_contact_date" db:"last_contact_date"`
	IntroducedByPersonID   *string                `json:"introduced_by_person_id" db:"introduced_by_person_id"`
	IntroductionDate       *time.Time             `json:"introduction_date" db:"introduction_date"`
	IntroductionContext    *string                `json:"introduction_context" db:"introduction_context"`
	SharedInterests        []string               `json:"shared_interests" db:"shared_interests"`
	AffinityScore          *float64               `json:"affinity_score" db:"affinity_score"`
	Metadata               map[string]interface{} `json:"metadata" db:"metadata"`
	Notes                  *string                `json:"notes" db:"notes"`
}

type Organization struct {
	ID        string                 `json:"id" db:"id"`
	CreatedAt time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
	Name      string                 `json:"name" db:"name"`
	Type      string                 `json:"type" db:"type"`
	Industry  *string                `json:"industry" db:"industry"`
	Website   *string                `json:"website" db:"website"`
	Metadata  map[string]interface{} `json:"metadata" db:"metadata"`
	Notes     *string                `json:"notes" db:"notes"`
	Tags      []string               `json:"tags" db:"tags"`
}

type SocialAnalytics struct {
	ID                         string                 `json:"id" db:"id"`
	PersonID                   string                 `json:"person_id" db:"person_id"`
	ComputedAt                 time.Time              `json:"computed_at" db:"computed_at"`
	OverallClosenessScore      *float64               `json:"overall_closeness_score" db:"overall_closeness_score"`
	SocialCentralityScore      *float64               `json:"social_centrality_score" db:"social_centrality_score"`
	MutualConnectionCount      *int                   `json:"mutual_connection_count" db:"mutual_connection_count"`
	AvgResponseLatencyHours    *int                   `json:"avg_response_latency_hours" db:"avg_response_latency_hours"`
	CommunicationFrequencyScore *float64              `json:"communication_frequency_score" db:"communication_frequency_score"`
	LastInteractionDays        *int                   `json:"last_interaction_days" db:"last_interaction_days"`
	TopSharedInterests         []string               `json:"top_shared_interests" db:"top_shared_interests"`
	AffinityVector             map[string]interface{} `json:"affinity_vector" db:"affinity_vector"`
	MaintenancePriorityScore   *float64               `json:"maintenance_priority_score" db:"maintenance_priority_score"`
	ComputationVersion         *string                `json:"computation_version" db:"computation_version"`
	Metadata                   map[string]interface{} `json:"metadata" db:"metadata"`
}

type CreatePersonRequest struct {
	FullName                string                 `json:"full_name" binding:"required"`
	DisplayName             *string                `json:"display_name"`
	Nicknames               []string               `json:"nicknames"`
	Pronouns                *string                `json:"pronouns"`
	Emails                  []string               `json:"emails"`
	Phones                  []string               `json:"phones"`
	ScenarioAuthenticatorID *string                `json:"scenario_authenticator_id"`
	Metadata                map[string]interface{} `json:"metadata"`
	Tags                    []string               `json:"tags"`
	Notes                   *string                `json:"notes"`
	SocialProfiles          map[string]interface{} `json:"social_profiles"`
}

type CreateRelationshipRequest struct {
	FromPersonID        string                 `json:"from_person_id" binding:"required"`
	ToPersonID          string                 `json:"to_person_id" binding:"required"`
	RelationshipType    string                 `json:"relationship_type" binding:"required"`
	Strength            *float64               `json:"strength"`
	LastContactDate     *time.Time             `json:"last_contact_date"`
	SharedInterests     []string               `json:"shared_interests"`
	IntroductionContext *string                `json:"introduction_context"`
	Metadata            map[string]interface{} `json:"metadata"`
	Notes               *string                `json:"notes"`
}

type SearchRequest struct {
	Query  string            `json:"query"`
	Filters map[string]interface{} `json:"filters"`
	Limit  *int              `json:"limit"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
	Version   string `json:"version"`
}

// =============================================================================
// DATABASE
// =============================================================================

var db *sql.DB

func initDB() {
	var err error
	
	// Load environment variables
	godotenv.Load()
	
	// Database connection string
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "password")
	dbname := getEnv("DB_NAME", "postgres")
	
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	// Test the connection
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	log.Println("Connected to PostgreSQL database")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// =============================================================================
// HANDLERS
// =============================================================================

func healthCheck(c *gin.Context) {
	// Test database connection
	dbStatus := "healthy"
	if err := db.Ping(); err != nil {
		dbStatus = "unhealthy"
	}
	
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Database:  dbStatus,
		Version:   "1.0.0",
	})
}

// =============================================================================
// PERSON HANDLERS
// =============================================================================

func getPersons(c *gin.Context) {
	limit := 50 // Default limit
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	
	search := c.Query("search")
	tags := c.Query("tags")
	
	query := `
		SELECT id, created_at, updated_at, full_name, display_name, nicknames, pronouns, 
		       emails, phones, scenario_authenticator_id, metadata, tags, notes, 
		       social_profiles, computed_signals, deleted_at
		FROM persons 
		WHERE deleted_at IS NULL`
	
	args := []interface{}{}
	argCount := 0
	
	if search != "" {
		argCount++
		query += fmt.Sprintf(` AND (full_name ILIKE $%d OR display_name ILIKE $%d OR $%d = ANY(emails))`, argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
	}
	
	if tags != "" {
		tagList := strings.Split(tags, ",")
		for _, tag := range tagList {
			argCount++
			query += fmt.Sprintf(` AND $%d = ANY(tags)`, argCount)
			args = append(args, strings.TrimSpace(tag))
		}
	}
	
	query += fmt.Sprintf(` ORDER BY updated_at DESC LIMIT $%d`, argCount+1)
	args = append(args, limit)
	
	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var persons []Person
	for rows.Next() {
		var p Person
		var metadataBytes, socialProfilesBytes, computedSignalsBytes []byte
		var nicknamesString, emailsString, phonesString, tagsString string
		
		err := rows.Scan(
			&p.ID, &p.CreatedAt, &p.UpdatedAt, &p.FullName, &p.DisplayName,
			&nicknamesString, &p.Pronouns, &emailsString, &phonesString,
			&p.ScenarioAuthenticatorID, &metadataBytes, &tagsString, &p.Notes,
			&socialProfilesBytes, &computedSignalsBytes, &p.DeletedAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		// Parse arrays and JSON
		if err := parseArrayString(nicknamesString, &p.Nicknames); err != nil {
			p.Nicknames = []string{}
		}
		if err := parseArrayString(emailsString, &p.Emails); err != nil {
			p.Emails = []string{}
		}
		if err := parseArrayString(phonesString, &p.Phones); err != nil {
			p.Phones = []string{}
		}
		if err := parseArrayString(tagsString, &p.Tags); err != nil {
			p.Tags = []string{}
		}
		
		if len(metadataBytes) > 0 {
			json.Unmarshal(metadataBytes, &p.Metadata)
		}
		if len(socialProfilesBytes) > 0 {
			json.Unmarshal(socialProfilesBytes, &p.SocialProfiles)
		}
		if len(computedSignalsBytes) > 0 {
			json.Unmarshal(computedSignalsBytes, &p.ComputedSignals)
		}
		
		persons = append(persons, p)
	}
	
	c.JSON(http.StatusOK, gin.H{"persons": persons, "count": len(persons)})
}

func getPerson(c *gin.Context) {
	id := c.Param("id")
	
	query := `
		SELECT id, created_at, updated_at, full_name, display_name, nicknames, pronouns,
		       emails, phones, scenario_authenticator_id, metadata, tags, notes,
		       social_profiles, computed_signals, deleted_at
		FROM persons 
		WHERE id = $1 AND deleted_at IS NULL`
	
	var p Person
	var metadataBytes, socialProfilesBytes, computedSignalsBytes []byte
	var nicknamesString, emailsString, phonesString, tagsString string
	
	err := db.QueryRow(query, id).Scan(
		&p.ID, &p.CreatedAt, &p.UpdatedAt, &p.FullName, &p.DisplayName,
		&nicknamesString, &p.Pronouns, &emailsString, &phonesString,
		&p.ScenarioAuthenticatorID, &metadataBytes, &tagsString, &p.Notes,
		&socialProfilesBytes, &computedSignalsBytes, &p.DeletedAt,
	)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Person not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	// Parse arrays and JSON
	parseArrayString(nicknamesString, &p.Nicknames)
	parseArrayString(emailsString, &p.Emails)
	parseArrayString(phonesString, &p.Phones)
	parseArrayString(tagsString, &p.Tags)
	
	if len(metadataBytes) > 0 {
		json.Unmarshal(metadataBytes, &p.Metadata)
	}
	if len(socialProfilesBytes) > 0 {
		json.Unmarshal(socialProfilesBytes, &p.SocialProfiles)
	}
	if len(computedSignalsBytes) > 0 {
		json.Unmarshal(computedSignalsBytes, &p.ComputedSignals)
	}
	
	c.JSON(http.StatusOK, p)
}

func createPerson(c *gin.Context) {
	var req CreatePersonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	id := uuid.New().String()
	now := time.Now()
	
	// Convert arrays and maps to JSON for storage
	metadataJSON, _ := json.Marshal(req.Metadata)
	socialProfilesJSON, _ := json.Marshal(req.SocialProfiles)
	
	query := `
		INSERT INTO persons (id, created_at, updated_at, full_name, display_name, nicknames, 
		                    pronouns, emails, phones, scenario_authenticator_id, metadata, 
		                    tags, notes, social_profiles)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id`
	
	var createdID string
	err := db.QueryRow(query, id, now, now, req.FullName, req.DisplayName,
		arrayToPostgresArray(req.Nicknames), req.Pronouns,
		arrayToPostgresArray(req.Emails), arrayToPostgresArray(req.Phones),
		req.ScenarioAuthenticatorID, metadataJSON,
		arrayToPostgresArray(req.Tags), req.Notes, socialProfilesJSON).Scan(&createdID)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"id":      createdID,
		"message": "Person created successfully",
	})
}

// =============================================================================
// RELATIONSHIP HANDLERS
// =============================================================================

func getRelationships(c *gin.Context) {
	personID := c.Query("person_id")
	relationshipType := c.Query("type")
	
	query := `
		SELECT id, created_at, updated_at, from_person_id, to_person_id, relationship_type,
		       strength, recency_score, last_contact_date, introduced_by_person_id,
		       introduction_date, introduction_context, shared_interests, affinity_score,
		       metadata, notes
		FROM relationships
		WHERE 1=1`
	
	args := []interface{}{}
	argCount := 0
	
	if personID != "" {
		argCount++
		query += fmt.Sprintf(` AND (from_person_id = $%d OR to_person_id = $%d)`, argCount, argCount)
		args = append(args, personID)
	}
	
	if relationshipType != "" {
		argCount++
		query += fmt.Sprintf(` AND relationship_type = $%d`, argCount)
		args = append(args, relationshipType)
	}
	
	query += ` ORDER BY strength DESC NULLS LAST, updated_at DESC LIMIT 100`
	
	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var relationships []Relationship
	for rows.Next() {
		var r Relationship
		var metadataBytes []byte
		var sharedInterestsString string
		
		err := rows.Scan(
			&r.ID, &r.CreatedAt, &r.UpdatedAt, &r.FromPersonID, &r.ToPersonID,
			&r.RelationshipType, &r.Strength, &r.RecencyScore, &r.LastContactDate,
			&r.IntroducedByPersonID, &r.IntroductionDate, &r.IntroductionContext,
			&sharedInterestsString, &r.AffinityScore, &metadataBytes, &r.Notes,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		parseArrayString(sharedInterestsString, &r.SharedInterests)
		if len(metadataBytes) > 0 {
			json.Unmarshal(metadataBytes, &r.Metadata)
		}
		
		relationships = append(relationships, r)
	}
	
	c.JSON(http.StatusOK, gin.H{"relationships": relationships, "count": len(relationships)})
}

func createRelationship(c *gin.Context) {
	var req CreateRelationshipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Validate that persons exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM persons WHERE id IN ($1, $2) AND deleted_at IS NULL",
		req.FromPersonID, req.ToPersonID).Scan(&count)
	if err != nil || count != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "One or both persons not found"})
		return
	}
	
	id := uuid.New().String()
	now := time.Now()
	metadataJSON, _ := json.Marshal(req.Metadata)
	
	query := `
		INSERT INTO relationships (id, created_at, updated_at, from_person_id, to_person_id,
		                          relationship_type, strength, last_contact_date, shared_interests,
		                          introduction_context, metadata, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`
	
	var createdID string
	err = db.QueryRow(query, id, now, now, req.FromPersonID, req.ToPersonID,
		req.RelationshipType, req.Strength, req.LastContactDate,
		arrayToPostgresArray(req.SharedInterests), req.IntroductionContext,
		metadataJSON, req.Notes).Scan(&createdID)
	
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"id":      createdID,
		"message": "Relationship created successfully",
	})
}

// =============================================================================
// ANALYTICS HANDLERS
// =============================================================================

func getSocialAnalytics(c *gin.Context) {
	personID := c.Query("person_id")
	
	query := `
		SELECT id, person_id, computed_at, overall_closeness_score, social_centrality_score,
		       mutual_connection_count, avg_response_latency_hours, communication_frequency_score,
		       last_interaction_days, top_shared_interests, affinity_vector,
		       maintenance_priority_score, computation_version, metadata
		FROM social_analytics`
	
	args := []interface{}{}
	if personID != "" {
		query += ` WHERE person_id = $1`
		args = append(args, personID)
	}
	query += ` ORDER BY computed_at DESC`
	
	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var analytics []SocialAnalytics
	for rows.Next() {
		var sa SocialAnalytics
		var affinityVectorBytes, metadataBytes []byte
		var topSharedInterestsString string
		
		err := rows.Scan(
			&sa.ID, &sa.PersonID, &sa.ComputedAt, &sa.OverallClosenessScore,
			&sa.SocialCentralityScore, &sa.MutualConnectionCount, &sa.AvgResponseLatencyHours,
			&sa.CommunicationFrequencyScore, &sa.LastInteractionDays, &topSharedInterestsString,
			&affinityVectorBytes, &sa.MaintenancePriorityScore, &sa.ComputationVersion,
			&metadataBytes,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		parseArrayString(topSharedInterestsString, &sa.TopSharedInterests)
		if len(affinityVectorBytes) > 0 {
			json.Unmarshal(affinityVectorBytes, &sa.AffinityVector)
		}
		if len(metadataBytes) > 0 {
			json.Unmarshal(metadataBytes, &sa.Metadata)
		}
		
		analytics = append(analytics, sa)
	}
	
	c.JSON(http.StatusOK, gin.H{"analytics": analytics, "count": len(analytics)})
}

// =============================================================================
// SEARCH HANDLERS
// =============================================================================

func searchContacts(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	limit := 20
	if req.Limit != nil && *req.Limit > 0 && *req.Limit <= 100 {
		limit = *req.Limit
	}
	
	// Simple search implementation - could be enhanced with full-text search
	query := `
		SELECT id, full_name, display_name, emails, tags, notes
		FROM persons
		WHERE deleted_at IS NULL
		AND (full_name ILIKE $1 OR display_name ILIKE $1 OR notes ILIKE $1 OR $2 = ANY(emails))
		ORDER BY 
			CASE WHEN full_name ILIKE $1 THEN 1
			     WHEN display_name ILIKE $1 THEN 2
			     WHEN $2 = ANY(emails) THEN 3
			     ELSE 4 END,
			updated_at DESC
		LIMIT $3`
	
	searchTerm := "%" + req.Query + "%"
	rows, err := db.Query(query, searchTerm, req.Query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	
	var results []map[string]interface{}
	for rows.Next() {
		var id, fullName string
		var displayName, notes *string
		var emailsString, tagsString string
		
		err := rows.Scan(&id, &fullName, &displayName, &emailsString, &tagsString, &notes)
		if err != nil {
			continue
		}
		
		var emails, tags []string
		parseArrayString(emailsString, &emails)
		parseArrayString(tagsString, &tags)
		
		result := map[string]interface{}{
			"id":           id,
			"full_name":    fullName,
			"display_name": displayName,
			"emails":       emails,
			"tags":         tags,
			"notes":        notes,
		}
		results = append(results, result)
	}
	
	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}

// =============================================================================
// UTILITY FUNCTIONS
// =============================================================================

func parseArrayString(s string, dest *[]string) error {
	s = strings.Trim(s, "{}")
	if s == "" {
		*dest = []string{}
		return nil
	}
	
	// Split by comma and clean up each element
	parts := strings.Split(s, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.Trim(strings.TrimSpace(part), "\"")
	}
	*dest = result
	return nil
}

func arrayToPostgresArray(arr []string) string {
	if len(arr) == 0 {
		return "{}"
	}
	
	// Escape quotes and build array string
	escaped := make([]string, len(arr))
	for i, s := range arr {
		escaped[i] = `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return "{" + strings.Join(escaped, ",") + "}"
}

// =============================================================================
// MAIN
// =============================================================================

func main() {
	// Initialize database
	initDB()
	defer db.Close()
	
	// Initialize Gin router
	r := gin.Default()
	
	// CORS middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	
	// Health check
	r.GET("/health", healthCheck)
	
	// API routes
	api := r.Group("/api/v1")
	{
		// Person routes
		api.GET("/contacts", getPersons)
		api.GET("/contacts/:id", getPerson)
		api.POST("/contacts", createPerson)
		
		// Relationship routes
		api.GET("/relationships", getRelationships)
		api.POST("/relationships", createRelationship)
		
		// Analytics routes
		api.GET("/analytics", getSocialAnalytics)
		
		// Search routes
		api.POST("/search", searchContacts)
	}
	
	// Start server
	port := getEnv("API_PORT", "8080")
	log.Printf("Contact Book API server starting on port %s", port)
	r.Run(":" + port)
}