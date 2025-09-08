package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Data models
type Item struct {
	ID          string                 `json:"id" db:"id"`
	ScenarioID  string                 `json:"scenario_id" db:"scenario_id"`
	ExternalID  string                 `json:"external_id" db:"external_id"`
	Title       string                 `json:"title" db:"title"`
	Description string                 `json:"description" db:"description"`
	Category    string                 `json:"category" db:"category"`
	Metadata    map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

type User struct {
	ID          string                 `json:"id" db:"id"`
	ScenarioID  string                 `json:"scenario_id" db:"scenario_id"`
	ExternalID  string                 `json:"external_id" db:"external_id"`
	Preferences map[string]interface{} `json:"preferences" db:"preferences"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

type UserInteraction struct {
	ID               string                 `json:"id" db:"id"`
	UserID           string                 `json:"user_id" db:"user_id"`
	ItemID           string                 `json:"item_id" db:"item_id"`
	InteractionType  string                 `json:"interaction_type" db:"interaction_type"`
	InteractionValue float64                `json:"interaction_value" db:"interaction_value"`
	Context          map[string]interface{} `json:"context" db:"context"`
	Timestamp        time.Time              `json:"timestamp" db:"timestamp"`
}

type Recommendation struct {
	ItemID      string  `json:"item_id"`
	ExternalID  string  `json:"external_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`
	Reason      string  `json:"reason"`
	Category    string  `json:"category"`
}

type SimilarItem struct {
	ItemID          string  `json:"item_id"`
	ExternalID      string  `json:"external_id"`
	Title           string  `json:"title"`
	SimilarityScore float64 `json:"similarity_score"`
	Category        string  `json:"category"`
}

// Request/Response structs
type IngestRequest struct {
	ScenarioID   string `json:"scenario_id" binding:"required"`
	Items        []Item `json:"items,omitempty"`
	Interactions []struct {
		UserID             string                 `json:"user_id" binding:"required"`
		ItemExternalID     string                 `json:"item_external_id" binding:"required"`
		InteractionType    string                 `json:"interaction_type" binding:"required"`
		InteractionValue   *float64               `json:"interaction_value,omitempty"`
		Context            map[string]interface{} `json:"context,omitempty"`
	} `json:"interactions,omitempty"`
}

type IngestResponse struct {
	Success              bool     `json:"success"`
	ItemsProcessed       int      `json:"items_processed"`
	InteractionsProcessed int     `json:"interactions_processed"`
	Errors               []string `json:"errors,omitempty"`
}

type RecommendRequest struct {
	UserID       string                 `json:"user_id" binding:"required"`
	ScenarioID   string                 `json:"scenario_id" binding:"required"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Limit        int                    `json:"limit,omitempty"`
	Algorithm    string                 `json:"algorithm,omitempty"`
	ExcludeItems []string               `json:"exclude_items,omitempty"`
}

type RecommendResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
	AlgorithmUsed   string          `json:"algorithm_used"`
	GeneratedAt     time.Time       `json:"generated_at"`
}

type SimilarRequest struct {
	ItemExternalID string `json:"item_external_id" binding:"required"`
	ScenarioID     string `json:"scenario_id" binding:"required"`
	Limit          int    `json:"limit,omitempty"`
	Threshold      float64 `json:"threshold,omitempty"`
}

type SimilarResponse struct {
	SimilarItems []SimilarItem `json:"similar_items"`
}

// Service struct
type RecommendationService struct {
	db          *sql.DB
	qdrantConn  *grpc.ClientConn
	qdrantClient pb.QdrantClient
}

func NewRecommendationService(db *sql.DB, qdrantConn *grpc.ClientConn) *RecommendationService {
	return &RecommendationService{
		db:          db,
		qdrantConn:  qdrantConn,
		qdrantClient: pb.NewQdrantClient(qdrantConn),
	}
}

// Database operations
func (s *RecommendationService) CreateItem(item *Item) error {
	item.ID = uuid.New().String()
	item.CreatedAt = time.Now()
	
	metadataJSON, _ := json.Marshal(item.Metadata)
	
	query := `
		INSERT INTO items (id, scenario_id, external_id, title, description, category, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (scenario_id, external_id) 
		DO UPDATE SET title=$4, description=$5, category=$6, metadata=$7
		RETURNING id
	`
	
	err := s.db.QueryRow(query, item.ID, item.ScenarioID, item.ExternalID, 
		item.Title, item.Description, item.Category, metadataJSON, item.CreatedAt).Scan(&item.ID)
	
	return err
}

func (s *RecommendationService) CreateUserInteraction(interaction *UserInteraction) error {
	interaction.ID = uuid.New().String()
	interaction.Timestamp = time.Now()
	
	// Ensure user exists
	userQuery := `
		INSERT INTO users (id, scenario_id, external_id, preferences, created_at)
		VALUES ($1, $2, $3, '{}', $4)
		ON CONFLICT (scenario_id, external_id) DO NOTHING
	`
	userID := uuid.New().String()
	s.db.Exec(userQuery, userID, "", interaction.UserID, time.Now())
	
	contextJSON, _ := json.Marshal(interaction.Context)
	
	query := `
		INSERT INTO user_interactions (id, user_id, item_id, interaction_type, interaction_value, context, timestamp)
		VALUES ($1, (SELECT id FROM users WHERE external_id = $2 LIMIT 1), 
		        (SELECT id FROM items WHERE external_id = $3 LIMIT 1), $4, $5, $6, $7)
	`
	
	_, err := s.db.Exec(query, interaction.ID, interaction.UserID, 
		interaction.ItemID, interaction.InteractionType, 
		interaction.InteractionValue, contextJSON, interaction.Timestamp)
	
	return err
}

// Generate simple embeddings (in production, would use actual ML model)
func (s *RecommendationService) generateEmbedding(text string) []float32 {
	// Simple hash-based embedding for demo purposes
	// In production, would use sentence transformers or similar
	embedding := make([]float32, 384) // Standard sentence transformer size
	
	words := strings.Fields(strings.ToLower(text))
	for i, word := range words {
		for j, char := range word {
			idx := (i + j + int(char)) % len(embedding)
			embedding[idx] += 0.1
		}
	}
	
	// Normalize
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	if norm > 0 {
		norm = float32(1.0 / (norm * norm))
		for i := range embedding {
			embedding[i] *= norm
		}
	}
	
	return embedding
}

func (s *RecommendationService) StoreItemEmbedding(item *Item) error {
	text := item.Title + " " + item.Description
	embedding := s.generateEmbedding(text)
	
	point := &pb.PointStruct{
		Id: &pb.PointId{
			PointIdOptions: &pb.PointId_Uuid{Uuid: item.ID},
		},
		Vectors: &pb.Vectors{
			VectorsOptions: &pb.Vectors_Vector{
				Vector: &pb.Vector{Data: embedding},
			},
		},
		Payload: map[string]*pb.Value{
			"item_id":     {Kind: &pb.Value_StringValue{StringValue: item.ID}},
			"scenario_id": {Kind: &pb.Value_StringValue{StringValue: item.ScenarioID}},
			"category":    {Kind: &pb.Value_StringValue{StringValue: item.Category}},
			"title":       {Kind: &pb.Value_StringValue{StringValue: item.Title}},
		},
	}
	
	_, err := s.qdrantClient.Upsert(context.Background(), &pb.UpsertPoints{
		CollectionName: "item-embeddings",
		Points:         []*pb.PointStruct{point},
	})
	
	return err
}

func (s *RecommendationService) GetSimilarItems(itemID, scenarioID string, limit int, threshold float64) ([]SimilarItem, error) {
	// Get the reference item's embedding
	searchResponse, err := s.qdrantClient.Scroll(context.Background(), &pb.ScrollPoints{
		CollectionName: "item-embeddings",
		Filter: &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "item_id",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{Keyword: itemID},
							},
						},
					},
				},
			},
		},
		Limit: &limit,
	})
	
	if err != nil || len(searchResponse.Result) == 0 {
		return nil, fmt.Errorf("reference item not found")
	}
	
	referenceVector := searchResponse.Result[0].Vectors.GetVector().Data
	
	// Search for similar items
	searchRequest := &pb.SearchPoints{
		CollectionName: "item-embeddings",
		Vector:         referenceVector,
		Limit:          uint64(limit),
		ScoreThreshold: &threshold,
		Filter: &pb.Filter{
			Must: []*pb.Condition{
				{
					ConditionOneOf: &pb.Condition_Field{
						Field: &pb.FieldCondition{
							Key: "scenario_id",
							Match: &pb.Match{
								MatchValue: &pb.Match_Keyword{Keyword: scenarioID},
							},
						},
					},
				},
			},
		},
		WithPayload: &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	}
	
	response, err := s.qdrantClient.Search(context.Background(), searchRequest)
	if err != nil {
		return nil, err
	}
	
	var items []SimilarItem
	for _, point := range response.Result {
		if point.Payload["item_id"].GetStringValue() == itemID {
			continue // Skip self
		}
		
		items = append(items, SimilarItem{
			ItemID:          point.Payload["item_id"].GetStringValue(),
			ExternalID:      point.Payload["item_id"].GetStringValue(), // Simplified
			Title:           point.Payload["title"].GetStringValue(),
			SimilarityScore: float64(point.Score),
			Category:        point.Payload["category"].GetStringValue(),
		})
	}
	
	return items, nil
}

func (s *RecommendationService) GetRecommendations(userID, scenarioID string, limit int, algorithm string, excludeItems []string) ([]Recommendation, error) {
	// Simplified collaborative filtering based on user interactions
	query := `
		SELECT DISTINCT i.id, i.external_id, i.title, i.description, i.category,
		       COUNT(ui.id) as interaction_count
		FROM items i
		JOIN user_interactions ui ON i.id = ui.item_id
		WHERE i.scenario_id = $1 
		  AND ui.user_id != (SELECT id FROM users WHERE external_id = $2 LIMIT 1)
		  AND i.external_id NOT IN (` + strings.Repeat("?,", len(excludeItems)-1) + "?)" +
		`GROUP BY i.id, i.external_id, i.title, i.description, i.category
		ORDER BY interaction_count DESC
		LIMIT $3
	`
	
	args := []interface{}{scenarioID, userID, limit}
	for _, item := range excludeItems {
		args = append(args, item)
	}
	
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var recommendations []Recommendation
	for rows.Next() {
		var r Recommendation
		var count int
		err := rows.Scan(&r.ItemID, &r.ExternalID, &r.Title, &r.Description, &r.Category, &count)
		if err != nil {
			continue
		}
		
		r.Confidence = float64(count) / 10.0 // Normalize confidence
		r.Reason = fmt.Sprintf("Popular with %d other users", count)
		recommendations = append(recommendations, r)
	}
	
	return recommendations, nil
}

// HTTP Handlers
func (s *RecommendationService) IngestHandler(c *gin.Context) {
	var req IngestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	var response IngestResponse
	var errors []string
	
	// Process items
	for _, item := range req.Items {
		item.ScenarioID = req.ScenarioID
		if err := s.CreateItem(&item); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to create item %s: %v", item.ExternalID, err))
			continue
		}
		
		// Store embedding
		if err := s.StoreItemEmbedding(&item); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to store embedding for item %s: %v", item.ExternalID, err))
		}
		
		response.ItemsProcessed++
	}
	
	// Process interactions
	for _, interaction := range req.Interactions {
		userInteraction := &UserInteraction{
			UserID:           interaction.UserID,
			ItemID:           interaction.ItemExternalID,
			InteractionType:  interaction.InteractionType,
			InteractionValue: 1.0,
			Context:          interaction.Context,
		}
		
		if interaction.InteractionValue != nil {
			userInteraction.InteractionValue = *interaction.InteractionValue
		}
		
		if err := s.CreateUserInteraction(userInteraction); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to create interaction: %v", err))
			continue
		}
		
		response.InteractionsProcessed++
	}
	
	response.Success = len(errors) == 0
	if len(errors) > 0 {
		response.Errors = errors
	}
	
	c.JSON(http.StatusOK, response)
}

func (s *RecommendationService) RecommendHandler(c *gin.Context) {
	var req RecommendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Algorithm == "" {
		req.Algorithm = "hybrid"
	}
	
	recommendations, err := s.GetRecommendations(req.UserID, req.ScenarioID, req.Limit, req.Algorithm, req.ExcludeItems)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	response := RecommendResponse{
		Recommendations: recommendations,
		AlgorithmUsed:   req.Algorithm,
		GeneratedAt:     time.Now(),
	}
	
	c.JSON(http.StatusOK, response)
}

func (s *RecommendationService) SimilarHandler(c *gin.Context) {
	var req SimilarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Threshold == 0 {
		req.Threshold = 0.7
	}
	
	// Get item ID from external ID
	var itemID string
	err := s.db.QueryRow("SELECT id FROM items WHERE scenario_id = $1 AND external_id = $2", 
		req.ScenarioID, req.ItemExternalID).Scan(&itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"})
		return
	}
	
	similarItems, err := s.GetSimilarItems(itemID, req.ScenarioID, req.Limit, req.Threshold)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	response := SimilarResponse{
		SimilarItems: similarItems,
	}
	
	c.JSON(http.StatusOK, response)
}

func (s *RecommendationService) HealthHandler(c *gin.Context) {
	// Check database connection
	if err := s.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"database": "disconnected",
			"error": err.Error(),
		})
		return
	}
	
	// Check Qdrant connection
	_, err := s.qdrantClient.HealthCheck(context.Background(), &pb.HealthCheckRequest{})
	qdrantStatus := "connected"
	if err != nil {
		qdrantStatus = "disconnected"
	}
	
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"database": "connected",
		"qdrant": qdrantStatus,
		"timestamp": time.Now(),
	})
}

// Setup database connection
func setupDatabase() (*sql.DB, error) {
	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components - REQUIRED, no defaults
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		password := os.Getenv("POSTGRES_PASSWORD")
		dbname := os.Getenv("POSTGRES_DB")
		
		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			return nil, fmt.Errorf("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			user, password, host, port, dbname)
	}
	
	db, err := sql.Open("postgres", postgresURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to open database connection: %v", err)
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
	return db, nil
}

// Setup Qdrant connection
func setupQdrant() (*grpc.ClientConn, error) {
	// Qdrant configuration - REQUIRED, no defaults
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		return nil, fmt.Errorf("❌ QDRANT_HOST environment variable is required")
	}
	
	port := os.Getenv("QDRANT_PORT")
	if port == "" {
		return nil, fmt.Errorf("❌ QDRANT_PORT environment variable is required")
	}
	
	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", host, port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %v", err)
	}
	
	return conn, nil
}

func main() {
	// Load environment variables
	godotenv.Load()
	
	// Setup database
	db, err := setupDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	// Setup Qdrant
	qdrantConn, err := setupQdrant()
	if err != nil {
		log.Printf("Warning: Failed to connect to Qdrant: %v", err)
	}
	if qdrantConn != nil {
		defer qdrantConn.Close()
	}
	
	// Create service
	service := NewRecommendationService(db, qdrantConn)
	
	// Setup Gin router
	r := gin.Default()
	
	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})
	
	// Routes
	v1 := r.Group("/api/v1")
	{
		recommendations := v1.Group("/recommendations")
		{
			recommendations.POST("/ingest", service.IngestHandler)
			recommendations.POST("/get", service.RecommendHandler)
			recommendations.POST("/similar", service.SimilarHandler)
		}
	}
	
	r.GET("/health", service.HealthHandler)
	r.GET("/docs", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"name": "Recommendation Engine API",
			"version": "1.0.0",
			"endpoints": map[string]interface{}{
				"health": "/health",
				"ingest": "POST /api/v1/recommendations/ingest",
				"recommend": "POST /api/v1/recommendations/get",
				"similar": "POST /api/v1/recommendations/similar",
			},
			"documentation": "https://github.com/vrooli/vrooli/scenarios/recommendation-engine/README.md",
		})
	})
	
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	
	log.Printf("🚀 Recommendation Engine API starting on port %s", port)
	log.Printf("📚 API Documentation: http://localhost:%s/docs", port)
	log.Printf("❤️  Health Check: http://localhost:%s/health", port)
	
	r.Run(":" + port)
}