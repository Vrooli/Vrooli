package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Data models
type Campaign struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	BrandID      string    `json:"brand_id" db:"brand_id"`
	Description  *string   `json:"description" db:"description"`
	Status       string    `json:"status" db:"status"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	StartDate    *time.Time `json:"start_date" db:"start_date"`
	EndDate      *time.Time `json:"end_date" db:"end_date"`
	Budget       *float64  `json:"budget" db:"budget"`
	TeamMembers  []string  `json:"team_members" db:"team_members"`
}

type Brand struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description" db:"description"`
	Guidelines  *string   `json:"guidelines" db:"guidelines"`
	Colors      []string  `json:"colors" db:"colors"`
	Fonts       []string  `json:"fonts" db:"fonts"`
	LogoURL     *string   `json:"logo_url" db:"logo_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type ImageGeneration struct {
	ID           string    `json:"id" db:"id"`
	CampaignID   string    `json:"campaign_id" db:"campaign_id"`
	Prompt       string    `json:"prompt" db:"prompt"`
	VoiceBrief   *string   `json:"voice_brief" db:"voice_brief"`
	Status       string    `json:"status" db:"status"`
	ImageURL     *string   `json:"image_url" db:"image_url"`
	QualityScore *float64  `json:"quality_score" db:"quality_score"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	CompletedAt  *time.Time `json:"completed_at" db:"completed_at"`
	Metadata     map[string]interface{} `json:"metadata" db:"metadata"`
}

type GenerationRequest struct {
	CampaignID  string                 `json:"campaign_id"`
	Prompt      string                 `json:"prompt"`
	VoiceBrief  *string               `json:"voice_brief,omitempty"`
	Style       string                 `json:"style"`
	Dimensions  string                 `json:"dimensions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type VoiceBriefRequest struct {
	AudioData string `json:"audio_data"` // base64 encoded
	Format    string `json:"format"`     // wav, mp3, etc.
}

// Service configuration
type Config struct {
	Port              string
	PostgresURL       string
	N8NBaseURL        string
	WindmillBaseURL   string
	ComfyUIBaseURL    string
	WhisperBaseURL    string
	MinioEndpoint     string
	MinioAccessKey    string
	MinioSecretKey    string
	QdrantURL         string
}

func loadConfig() *Config {
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
	
	// Port configuration - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}
	
	return &Config{
		Port:              port,
		PostgresURL:       postgresURL,
		N8NBaseURL:        getEnv("N8N_BASE_URL", "http://localhost:5678"),
		WindmillBaseURL:   getEnv("WINDMILL_BASE_URL", "http://localhost:8000"),
		ComfyUIBaseURL:    getEnv("COMFYUI_BASE_URL", "http://localhost:8188"),
		WhisperBaseURL:    getEnv("WHISPER_BASE_URL", "http://localhost:9000"),
		MinioEndpoint:     getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:    getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:    getEnv("MINIO_SECRET_KEY", "minioadmin"),
		QdrantURL:         getEnv("QDRANT_URL", "http://localhost:6333"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Database connection
var db *sql.DB

func initDB(config *Config) error {
	var err error
	db, err = sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
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
	log.Printf("🎨 Database URL configured")
	
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

		// Add random jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * rand.Float64())
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
		return fmt.Errorf("❌ Database connection failed after %d attempts: %w", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

// HTTP handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "image-generation-pipeline",
		"version":   "1.0.0",
		"timestamp": time.Now().Unix(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func campaignsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getCampaigns(w, r)
	case http.MethodPost:
		createCampaign(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getCampaigns(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, brand_id, description, status, created_at, updated_at, 
		       start_date, end_date, budget, team_members
		FROM campaigns 
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		var c Campaign
		var teamMembersJSON []byte
		
		err := rows.Scan(&c.ID, &c.Name, &c.BrandID, &c.Description, &c.Status,
			&c.CreatedAt, &c.UpdatedAt, &c.StartDate, &c.EndDate, &c.Budget, &teamMembersJSON)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan error: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Parse team members JSON
		if len(teamMembersJSON) > 0 {
			json.Unmarshal(teamMembersJSON, &c.TeamMembers)
		}
		
		campaigns = append(campaigns, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(campaigns)
}

func createCampaign(w http.ResponseWriter, r *http.Request) {
	var campaign Campaign
	if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	campaign.ID = uuid.New().String()
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()

	teamMembersJSON, _ := json.Marshal(campaign.TeamMembers)

	_, err := db.Exec(`
		INSERT INTO campaigns (id, name, brand_id, description, status, created_at, updated_at, 
		                      start_date, end_date, budget, team_members)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, campaign.ID, campaign.Name, campaign.BrandID, campaign.Description, campaign.Status,
		campaign.CreatedAt, campaign.UpdatedAt, campaign.StartDate, campaign.EndDate, 
		campaign.Budget, teamMembersJSON)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(campaign)
}

func brandsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getBrands(w, r)
	case http.MethodPost:
		createBrand(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getBrands(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, description, guidelines, colors, fonts, logo_url, created_at, updated_at
		FROM brands 
		ORDER BY name ASC
	`)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		var b Brand
		var colorsJSON, fontsJSON []byte
		
		err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.Guidelines, 
			&colorsJSON, &fontsJSON, &b.LogoURL, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan error: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Parse JSON arrays
		if len(colorsJSON) > 0 {
			json.Unmarshal(colorsJSON, &b.Colors)
		}
		if len(fontsJSON) > 0 {
			json.Unmarshal(fontsJSON, &b.Fonts)
		}
		
		brands = append(brands, b)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(brands)
}

func createBrand(w http.ResponseWriter, r *http.Request) {
	var brand Brand
	if err := json.NewDecoder(r.Body).Decode(&brand); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	brand.ID = uuid.New().String()
	brand.CreatedAt = time.Now()
	brand.UpdatedAt = time.Now()

	colorsJSON, _ := json.Marshal(brand.Colors)
	fontsJSON, _ := json.Marshal(brand.Fonts)

	_, err := db.Exec(`
		INSERT INTO brands (id, name, description, guidelines, colors, fonts, logo_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, brand.ID, brand.Name, brand.Description, brand.Guidelines, 
		colorsJSON, fontsJSON, brand.LogoURL, brand.CreatedAt, brand.UpdatedAt)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(brand)
}

func generateImageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create generation record
	generation := ImageGeneration{
		ID:         uuid.New().String(),
		CampaignID: req.CampaignID,
		Prompt:     req.Prompt,
		VoiceBrief: req.VoiceBrief,
		Status:     "processing",
		CreatedAt:  time.Now(),
		Metadata:   req.Metadata,
	}

	metadataJSON, _ := json.Marshal(generation.Metadata)

	_, err := db.Exec(`
		INSERT INTO image_generations (id, campaign_id, prompt, voice_brief, status, created_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, generation.ID, generation.CampaignID, generation.Prompt, generation.VoiceBrief,
		generation.Status, generation.CreatedAt, metadataJSON)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Trigger n8n workflow for image generation
	go triggerImageGeneration(generation, req)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      generation.ID,
		"status":  "processing",
		"message": "Image generation started",
	})
}

func triggerImageGeneration(generation ImageGeneration, req GenerationRequest) {
	config := loadConfig()
	
	// Prepare n8n webhook payload
	payload := map[string]interface{}{
		"generation_id": generation.ID,
		"campaign_id":   req.CampaignID,
		"prompt":        req.Prompt,
		"style":         req.Style,
		"dimensions":    req.Dimensions,
		"metadata":      req.Metadata,
	}

	payloadJSON, _ := json.Marshal(payload)

	// Call n8n main workflow
	resp, err := http.Post(
		config.N8NBaseURL+"/webhook/image-generation",
		"application/json",
		bytes.NewBuffer(payloadJSON),
	)
	
	if err != nil {
		log.Printf("❌ Failed to trigger n8n workflow: %v", err)
		updateGenerationStatus(generation.ID, "failed", nil, nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ n8n workflow returned status %d", resp.StatusCode)
		updateGenerationStatus(generation.ID, "failed", nil, nil)
		return
	}

	log.Printf("✅ Image generation workflow triggered for %s", generation.ID)
}

func updateGenerationStatus(id, status string, imageURL *string, qualityScore *float64) {
	var completedAt *time.Time
	if status == "completed" || status == "failed" {
		now := time.Now()
		completedAt = &now
	}

	_, err := db.Exec(`
		UPDATE image_generations 
		SET status = $1, image_url = $2, quality_score = $3, completed_at = $4, updated_at = $5
		WHERE id = $6
	`, status, imageURL, qualityScore, completedAt, time.Now(), id)

	if err != nil {
		log.Printf("❌ Failed to update generation status: %v", err)
	}
}

func processVoiceBriefHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VoiceBriefRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Forward to Whisper service
	config := loadConfig()
	transcript, err := processAudioWithWhisper(config.WhisperBaseURL, req.AudioData, req.Format)
	if err != nil {
		http.Error(w, fmt.Sprintf("Voice processing error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"transcript": transcript,
		"status":     "success",
	})
}

func processAudioWithWhisper(whisperURL, audioData, format string) (string, error) {
	payload := map[string]interface{}{
		"audio": audioData,
		"task":  "transcribe",
		"language": "auto",
	}

	payloadJSON, _ := json.Marshal(payload)

	resp, err := http.Post(
		whisperURL+"/asr",
		"application/json",
		bytes.NewBuffer(payloadJSON),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper service returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if text, ok := result["text"].(string); ok {
		return text, nil
	}

	return "", fmt.Errorf("unexpected response format from whisper service")
}

func generationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	campaignID := r.URL.Query().Get("campaign_id")
	status := r.URL.Query().Get("status")
	
	query := `
		SELECT id, campaign_id, prompt, voice_brief, status, image_url, 
		       quality_score, created_at, completed_at, metadata
		FROM image_generations
		WHERE 1=1
	`
	
	var args []interface{}
	argIndex := 1
	
	if campaignID != "" {
		query += fmt.Sprintf(" AND campaign_id = $%d", argIndex)
		args = append(args, campaignID)
		argIndex++
	}
	
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	
	query += " ORDER BY created_at DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var generations []ImageGeneration
	for rows.Next() {
		var g ImageGeneration
		var metadataJSON []byte
		
		err := rows.Scan(&g.ID, &g.CampaignID, &g.Prompt, &g.VoiceBrief, &g.Status,
			&g.ImageURL, &g.QualityScore, &g.CreatedAt, &g.CompletedAt, &metadataJSON)
		if err != nil {
			http.Error(w, fmt.Sprintf("Row scan error: %v", err), http.StatusInternalServerError)
			return
		}
		
		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &g.Metadata)
		}
		
		generations = append(generations, g)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(generations)
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start image-generation-pipeline

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	log.Println("🚀 Starting Image Generation Pipeline API...")

	config := loadConfig()

	// Initialize database
	if err := initDB(config); err != nil {
		log.Fatalf("❌ Database initialization failed: %v", err)
	}
	defer db.Close()

	// Setup routes
	r := mux.NewRouter()

	// API routes
	r.HandleFunc("/health", healthHandler).Methods("GET")
	r.HandleFunc("/api/campaigns", campaignsHandler).Methods("GET", "POST")
	r.HandleFunc("/api/brands", brandsHandler).Methods("GET", "POST")
	r.HandleFunc("/api/generate", generateImageHandler).Methods("POST")
	r.HandleFunc("/api/voice-brief", processVoiceBriefHandler).Methods("POST")
	r.HandleFunc("/api/generations", generationsHandler).Methods("GET")

	// CORS middleware
	corsHandler := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"*"}),
	)(r)

	// Start server
	log.Printf("✅ Server starting on port %s", config.Port)
	log.Printf("🔗 Health check: http://localhost:%s/health", config.Port)
	log.Printf("🔗 API endpoints: http://localhost:%s/api/", config.Port)

	if err := http.ListenAndServe(":"+config.Port, corsHandler); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}