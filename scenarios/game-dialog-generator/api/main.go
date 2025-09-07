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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/go-resty/resty/v2"
)

// Data structures for the Game Dialog Generator
type Character struct {
	ID                uuid.UUID              `json:"id" db:"id"`
	Name              string                 `json:"name" db:"name"`
	PersonalityTraits map[string]interface{} `json:"personality_traits" db:"personality_traits"`
	BackgroundStory   string                 `json:"background_story" db:"background_story"`
	SpeechPatterns    map[string]interface{} `json:"speech_patterns" db:"speech_patterns"`
	Relationships     map[string]interface{} `json:"relationships" db:"relationships"`
	VoiceProfile      map[string]interface{} `json:"voice_profile" db:"voice_profile"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`
}

type Scene struct {
	ID           uuid.UUID                `json:"id" db:"id"`
	ProjectID    uuid.UUID                `json:"project_id" db:"project_id"`
	Name         string                   `json:"name" db:"name"`
	Context      string                   `json:"context" db:"context"`
	Setting      map[string]interface{}   `json:"setting" db:"setting"`
	Mood         string                   `json:"mood" db:"mood"`
	Participants pq.StringArray           `json:"participants" db:"participants"`
	CreatedAt    time.Time                `json:"created_at" db:"created_at"`
}

type Project struct {
	ID           uuid.UUID                `json:"id" db:"id"`
	Name         string                   `json:"name" db:"name"`
	Description  string                   `json:"description" db:"description"`
	Characters   pq.StringArray           `json:"characters" db:"characters"`
	Settings     map[string]interface{}   `json:"settings" db:"settings"`
	ExportFormat string                   `json:"export_format" db:"export_format"`
	CreatedAt    time.Time                `json:"created_at" db:"created_at"`
}

type DialogLine struct {
	ID          uuid.UUID `json:"id" db:"id"`
	CharacterID uuid.UUID `json:"character_id" db:"character_id"`
	SceneID     uuid.UUID `json:"scene_id" db:"scene_id"`
	Content     string    `json:"content" db:"content"`
	Emotion     string    `json:"emotion" db:"emotion"`
	AudioURL    string    `json:"audio_url,omitempty" db:"audio_file_path"`
	GeneratedAt time.Time `json:"generated_at" db:"generated_at"`
}

// Request/Response structures
type DialogGenerationRequest struct {
	CharacterID    string                 `json:"character_id"`
	SceneContext   string                 `json:"scene_context"`
	PreviousDialog []string               `json:"previous_dialog,omitempty"`
	EmotionState   string                 `json:"emotion_state,omitempty"`
	Constraints    map[string]interface{} `json:"constraints,omitempty"`
}

type DialogGenerationResponse struct {
	Dialog                   string  `json:"dialog"`
	Emotion                  string  `json:"emotion"`
	CharacterConsistencyScore float64 `json:"character_consistency_score"`
	AudioURL                 string  `json:"audio_url,omitempty"`
}

type BatchDialogRequest struct {
	SceneID        string                   `json:"scene_id"`
	DialogRequests []DialogGenerationRequest `json:"dialog_requests"`
	ExportFormat   string                   `json:"export_format,omitempty"`
}

type BatchDialogResponse struct {
	DialogSet          []DialogGenerationResponse `json:"dialog_set"`
	ExportFileURL      string                     `json:"export_file_url,omitempty"`
	GenerationMetrics  map[string]interface{}     `json:"generation_metrics"`
}

// Global variables
var (
	db           *sql.DB
	restClient   *resty.Client
	ollamaURL    string
	qdrantURL    string
)

// Database initialization
func initDB() {
	var err error
	
	// Get database connection details from environment
	dbHost := getEnvWithDefault("POSTGRES_HOST", "localhost")
	dbPort := getEnvWithDefault("POSTGRES_PORT", "5432")
	dbUser := getEnvWithDefault("POSTGRES_USER", "postgres")
	dbPassword := getEnvWithDefault("POSTGRES_PASSWORD", "password")
	dbName := getEnvWithDefault("POSTGRES_DB", "game_dialog_generator")
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	log.Println("🌿 Connected to PostgreSQL database successfully!")
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Initialize external service clients
func initClients() {
	restClient = resty.New()
	ollamaURL = getEnvWithDefault("OLLAMA_URL", "http://localhost:11434")
	qdrantURL = getEnvWithDefault("QDRANT_URL", "http://localhost:6333")
	
	log.Printf("🎮 Initialized clients - Ollama: %s, Qdrant: %s", ollamaURL, qdrantURL)
}

// Health check handler
func healthHandler(c *gin.Context) {
	status := gin.H{
		"status":    "healthy",
		"service":   "game-dialog-generator",
		"theme":     "jungle-platformer",
		"timestamp": time.Now().UTC(),
		"resources": gin.H{
			"database": "connected",
			"ollama":   "available",
			"qdrant":   "available",
		},
	}
	
	// Test database connection
	if err := db.Ping(); err != nil {
		status["resources"].(gin.H)["database"] = "disconnected"
		status["status"] = "degraded"
	}
	
	// Test Ollama connection
	resp, err := restClient.R().Get(ollamaURL + "/api/version")
	if err != nil || resp.StatusCode() != 200 {
		status["resources"].(gin.H)["ollama"] = "unavailable"
		status["status"] = "degraded"
	}
	
	// Test Qdrant connection
	resp, err = restClient.R().Get(qdrantURL + "/health")
	if err != nil || resp.StatusCode() != 200 {
		status["resources"].(gin.H)["qdrant"] = "unavailable"
		status["status"] = "degraded"
	}
	
	c.JSON(http.StatusOK, status)
}

// Character management handlers
func createCharacterHandler(c *gin.Context) {
	var character Character
	
	if err := c.ShouldBindJSON(&character); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character data", "details": err.Error()})
		return
	}
	
	character.ID = uuid.New()
	character.CreatedAt = time.Now().UTC()
	character.UpdatedAt = time.Now().UTC()
	
	// Convert maps to JSON for storage
	personalityJSON, _ := json.Marshal(character.PersonalityTraits)
	speechJSON, _ := json.Marshal(character.SpeechPatterns)
	relationshipsJSON, _ := json.Marshal(character.Relationships)
	voiceJSON, _ := json.Marshal(character.VoiceProfile)
	
	query := `
		INSERT INTO characters (id, name, personality_traits, background_story, speech_patterns, relationships, voice_profile, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	
	_, err := db.Exec(query, character.ID, character.Name, personalityJSON, character.BackgroundStory,
		speechJSON, relationshipsJSON, voiceJSON, character.CreatedAt, character.UpdatedAt)
	
	if err != nil {
		log.Printf("Error creating character: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create character"})
		return
	}
	
	// Generate embeddings for character personality and store in Qdrant
	go func() {
		generateCharacterEmbedding(character)
	}()
	
	c.JSON(http.StatusCreated, gin.H{
		"success":      true,
		"character_id": character.ID,
		"message":      "🌿 Character created in the jungle adventure!",
	})
}

func getCharacterHandler(c *gin.Context) {
	characterID := c.Param("id")
	
	var character Character
	var personalityJSON, speechJSON, relationshipsJSON, voiceJSON []byte
	
	query := `
		SELECT id, name, personality_traits, background_story, speech_patterns, relationships, voice_profile, created_at, updated_at
		FROM characters WHERE id = $1`
	
	err := db.QueryRow(query, characterID).Scan(
		&character.ID, &character.Name, &personalityJSON, &character.BackgroundStory,
		&speechJSON, &relationshipsJSON, &voiceJSON, &character.CreatedAt, &character.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Character not found in the jungle"})
		return
	} else if err != nil {
		log.Printf("Error getting character: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve character"})
		return
	}
	
	// Parse JSON fields
	json.Unmarshal(personalityJSON, &character.PersonalityTraits)
	json.Unmarshal(speechJSON, &character.SpeechPatterns)
	json.Unmarshal(relationshipsJSON, &character.Relationships)
	json.Unmarshal(voiceJSON, &character.VoiceProfile)
	
	c.JSON(http.StatusOK, character)
}

func listCharactersHandler(c *gin.Context) {
	query := `
		SELECT id, name, personality_traits, background_story, speech_patterns, relationships, voice_profile, created_at, updated_at
		FROM characters ORDER BY created_at DESC`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error listing characters: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list characters"})
		return
	}
	defer rows.Close()
	
	var characters []Character
	
	for rows.Next() {
		var character Character
		var personalityJSON, speechJSON, relationshipsJSON, voiceJSON []byte
		
		err := rows.Scan(
			&character.ID, &character.Name, &personalityJSON, &character.BackgroundStory,
			&speechJSON, &relationshipsJSON, &voiceJSON, &character.CreatedAt, &character.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning character: %v", err)
			continue
		}
		
		// Parse JSON fields
		json.Unmarshal(personalityJSON, &character.PersonalityTraits)
		json.Unmarshal(speechJSON, &character.SpeechPatterns)
		json.Unmarshal(relationshipsJSON, &character.Relationships)
		json.Unmarshal(voiceJSON, &character.VoiceProfile)
		
		characters = append(characters, character)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"characters": characters,
		"count":      len(characters),
		"message":    "🌿 Characters from the jungle adventure!",
	})
}

// Dialog generation handlers
func generateDialogHandler(c *gin.Context) {
	var req DialogGenerationRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dialog request", "details": err.Error()})
		return
	}
	
	// Get character information
	characterID, err := uuid.Parse(req.CharacterID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character ID"})
		return
	}
	
	character, err := getCharacterFromDB(characterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Character not found in jungle"})
		return
	}
	
	// Generate dialog using Ollama
	dialog, emotion, consistencyScore, err := generateDialogWithOllama(character, req)
	if err != nil {
		log.Printf("Error generating dialog: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate dialog", "details": err.Error()})
		return
	}
	
	response := DialogGenerationResponse{
		Dialog:                   dialog,
		Emotion:                  emotion,
		CharacterConsistencyScore: consistencyScore,
	}
	
	// TODO: Add voice synthesis here if Whisper is available
	
	c.JSON(http.StatusOK, response)
}

func batchGenerateDialogHandler(c *gin.Context) {
	var req BatchDialogRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid batch dialog request", "details": err.Error()})
		return
	}
	
	var dialogSet []DialogGenerationResponse
	startTime := time.Now()
	
	for _, dialogReq := range req.DialogRequests {
		characterID, err := uuid.Parse(dialogReq.CharacterID)
		if err != nil {
			continue // Skip invalid character IDs
		}
		
		character, err := getCharacterFromDB(characterID)
		if err != nil {
			continue // Skip characters not found
		}
		
		dialog, emotion, consistencyScore, err := generateDialogWithOllama(character, dialogReq)
		if err != nil {
			log.Printf("Error in batch generation: %v", err)
			continue // Skip failed generations
		}
		
		dialogSet = append(dialogSet, DialogGenerationResponse{
			Dialog:                   dialog,
			Emotion:                  emotion,
			CharacterConsistencyScore: consistencyScore,
		})
	}
	
	metrics := map[string]interface{}{
		"total_requested": len(req.DialogRequests),
		"successful":      len(dialogSet),
		"duration_ms":     time.Since(startTime).Milliseconds(),
		"avg_consistency": calculateAverageConsistency(dialogSet),
	}
	
	response := BatchDialogResponse{
		DialogSet:          dialogSet,
		GenerationMetrics:  metrics,
	}
	
	c.JSON(http.StatusOK, response)
}

// Helper functions for dialog generation
func getCharacterFromDB(characterID uuid.UUID) (Character, error) {
	var character Character
	var personalityJSON, speechJSON, relationshipsJSON, voiceJSON []byte
	
	query := `
		SELECT id, name, personality_traits, background_story, speech_patterns, relationships, voice_profile, created_at, updated_at
		FROM characters WHERE id = $1`
	
	err := db.QueryRow(query, characterID).Scan(
		&character.ID, &character.Name, &personalityJSON, &character.BackgroundStory,
		&speechJSON, &relationshipsJSON, &voiceJSON, &character.CreatedAt, &character.UpdatedAt,
	)
	
	if err != nil {
		return character, err
	}
	
	// Parse JSON fields
	json.Unmarshal(personalityJSON, &character.PersonalityTraits)
	json.Unmarshal(speechJSON, &character.SpeechPatterns)
	json.Unmarshal(relationshipsJSON, &character.Relationships)
	json.Unmarshal(voiceJSON, &character.VoiceProfile)
	
	return character, nil
}

func generateDialogWithOllama(character Character, req DialogGenerationRequest) (string, string, float64, error) {
	// Build prompt for Ollama
	prompt := buildCharacterDialogPrompt(character, req)
	
	// Call Ollama API
	ollamaRequest := map[string]interface{}{
		"model":  "llama3.2",
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.8,
			"max_tokens":  200,
		},
	}
	
	resp, err := restClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(ollamaRequest).
		Post(ollamaURL + "/api/generate")
	
	if err != nil {
		return "", "", 0.0, fmt.Errorf("ollama request failed: %v", err)
	}
	
	if resp.StatusCode() != 200 {
		return "", "", 0.0, fmt.Errorf("ollama returned status %d", resp.StatusCode())
	}
	
	// Parse Ollama response
	var ollamaResponse map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &ollamaResponse); err != nil {
		return "", "", 0.0, fmt.Errorf("failed to parse ollama response: %v", err)
	}
	
	dialog, ok := ollamaResponse["response"].(string)
	if !ok {
		return "", "", 0.0, fmt.Errorf("no response from ollama")
	}
	
	// Extract emotion and calculate consistency score
	emotion := extractEmotionFromDialog(dialog, req.EmotionState)
	consistencyScore := calculateCharacterConsistency(character, dialog)
	
	return strings.TrimSpace(dialog), emotion, consistencyScore, nil
}

func buildCharacterDialogPrompt(character Character, req DialogGenerationRequest) string {
	prompt := fmt.Sprintf(`You are roleplaying as %s, a character in a video game with the following traits:

Character: %s
Background: %s
Personality Traits: %v
Speech Patterns: %v

Scene Context: %s
Current Emotion: %s
Previous Dialog: %v

Generate a single line of dialog that this character would say in this situation. The dialog should:
1. Stay true to the character's personality and speech patterns
2. Fit the scene context and current emotional state
3. Be appropriate for a video game
4. Be natural and engaging

Dialog:`, 
		character.Name,
		character.Name,
		character.BackgroundStory,
		character.PersonalityTraits,
		character.SpeechPatterns,
		req.SceneContext,
		req.EmotionState,
		req.PreviousDialog,
	)
	
	return prompt
}

func extractEmotionFromDialog(dialog, defaultEmotion string) string {
	// Simple emotion detection based on punctuation and keywords
	dialog = strings.ToLower(dialog)
	
	if strings.Contains(dialog, "!") && (strings.Contains(dialog, "great") || strings.Contains(dialog, "awesome")) {
		return "excited"
	}
	if strings.Contains(dialog, "?") {
		return "curious"
	}
	if strings.Contains(dialog, "...") || strings.Contains(dialog, "sigh") {
		return "thoughtful"
	}
	if strings.Contains(dialog, "no!") || strings.Contains(dialog, "stop") {
		return "angry"
	}
	
	if defaultEmotion != "" {
		return defaultEmotion
	}
	
	return "neutral"
}

func calculateCharacterConsistency(character Character, dialog string) float64 {
	// Simple consistency scoring based on character traits
	score := 0.7 // Base score
	
	dialog = strings.ToLower(dialog)
	
	// Check personality trait alignment
	if traits, ok := character.PersonalityTraits["brave"]; ok {
		if braveScore, ok := traits.(float64); ok && braveScore > 0.7 {
			if strings.Contains(dialog, "fight") || strings.Contains(dialog, "brave") || strings.Contains(dialog, "courage") {
				score += 0.1
			}
		}
	}
	
	if traits, ok := character.PersonalityTraits["humorous"]; ok {
		if humorScore, ok := traits.(float64); ok && humorScore > 0.6 {
			if strings.Contains(dialog, "haha") || strings.Contains(dialog, "joke") || strings.Contains(dialog, "funny") {
				score += 0.1
			}
		}
	}
	
	// Ensure score is between 0 and 1
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}
	
	return score
}

func calculateAverageConsistency(dialogSet []DialogGenerationResponse) float64 {
	if len(dialogSet) == 0 {
		return 0.0
	}
	
	total := 0.0
	for _, dialog := range dialogSet {
		total += dialog.CharacterConsistencyScore
	}
	
	return total / float64(len(dialogSet))
}

// Qdrant integration for character embeddings
func generateCharacterEmbedding(character Character) {
	// Create character description for embedding
	description := fmt.Sprintf("%s: %s. Personality: %v. Speech: %v", 
		character.Name, character.BackgroundStory, character.PersonalityTraits, character.SpeechPatterns)
	
	// Generate embedding using Ollama's nomic-embed-text model
	embeddingRequest := map[string]interface{}{
		"model":  "nomic-embed-text",
		"prompt": description,
	}
	
	resp, err := restClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(embeddingRequest).
		Post(ollamaURL + "/api/embeddings")
	
	if err != nil {
		log.Printf("Failed to generate embedding for character %s: %v", character.Name, err)
		return
	}
	
	var embeddingResponse map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &embeddingResponse); err != nil {
		log.Printf("Failed to parse embedding response: %v", err)
		return
	}
	
	embedding, ok := embeddingResponse["embedding"].([]interface{})
	if !ok {
		log.Printf("No embedding in response")
		return
	}
	
	// Store embedding in Qdrant
	storeCharacterEmbedding(character.ID.String(), embedding, character)
}

func storeCharacterEmbedding(characterID string, embedding []interface{}, character Character) {
	// Convert embedding to float64 slice
	var embeddingVector []float64
	for _, v := range embedding {
		if f, ok := v.(float64); ok {
			embeddingVector = append(embeddingVector, f)
		}
	}
	
	// Prepare Qdrant payload
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      characterID,
				"vector":  embeddingVector,
				"payload": map[string]interface{}{
					"name":               character.Name,
					"personality_traits": character.PersonalityTraits,
					"background_story":   character.BackgroundStory,
					"type":               "character",
				},
			},
		},
	}
	
	resp, err := restClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Put(qdrantURL + "/collections/characters/points")
	
	if err != nil {
		log.Printf("Failed to store character embedding: %v", err)
		return
	}
	
	if resp.StatusCode() != 200 {
		log.Printf("Qdrant returned status %d for character embedding", resp.StatusCode())
		return
	}
	
	log.Printf("🌿 Stored embedding for character %s in the jungle vector space!", character.Name)
}

// Project management handlers (basic implementation)
func createProjectHandler(c *gin.Context) {
	var project Project
	
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project data"})
		return
	}
	
	project.ID = uuid.New()
	project.CreatedAt = time.Now().UTC()
	
	settingsJSON, _ := json.Marshal(project.Settings)
	
	query := `
		INSERT INTO projects (id, name, description, characters, settings, export_format, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err := db.Exec(query, project.ID, project.Name, project.Description, 
		project.Characters, settingsJSON, project.ExportFormat, project.CreatedAt)
	
	if err != nil {
		log.Printf("Error creating project: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"project_id": project.ID,
		"message":    "🌿 Game project created in the jungle!",
	})
}

func listProjectsHandler(c *gin.Context) {
	query := `SELECT id, name, description, characters, settings, export_format, created_at FROM projects ORDER BY created_at DESC`
	
	rows, err := db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}
	defer rows.Close()
	
	var projects []Project
	for rows.Next() {
		var project Project
		var settingsJSON []byte
		
		err := rows.Scan(&project.ID, &project.Name, &project.Description, 
			&project.Characters, &settingsJSON, &project.ExportFormat, &project.CreatedAt)
		if err != nil {
			continue
		}
		
		json.Unmarshal(settingsJSON, &project.Settings)
		projects = append(projects, project)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
		"count":    len(projects),
	})
}

func main() {
	// Load environment variables
	godotenv.Load()
	
	// Initialize database and clients
	initDB()
	initClients()
	
	// Initialize Gin router
	r := gin.Default()
	
	// Add CORS middleware
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
	
	// Health check endpoint
	r.GET("/health", healthHandler)
	
	// Character management routes
	api := r.Group("/api/v1")
	{
		// Characters
		api.POST("/characters", createCharacterHandler)
		api.GET("/characters", listCharactersHandler)
		api.GET("/characters/:id", getCharacterHandler)
		api.GET("/characters/:id/personality", func(c *gin.Context) {
			// Reuse the getCharacterHandler logic but return only personality data
			characterID := c.Param("id")
			character, err := getCharacterFromDB(uuid.MustParse(characterID))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Character not found"})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{
				"personality_traits":     character.PersonalityTraits,
				"speech_patterns":       character.SpeechPatterns,
				"relationship_dynamics": character.Relationships,
			})
		})
		
		// Dialog generation
		api.POST("/dialog/generate", generateDialogHandler)
		api.POST("/dialog/batch", batchGenerateDialogHandler)
		
		// Projects
		api.POST("/projects", createProjectHandler)
		api.GET("/projects", listProjectsHandler)
	}
	
	// Get port from environment
	port := getEnvWithDefault("API_PORT", "8080")
	
	log.Printf("🌿 Game Dialog Generator API starting on port %s with jungle adventure theme! 🎮", port)
	log.Printf("🔧 Health check: http://localhost:%s/health", port)
	log.Printf("📖 API docs: http://localhost:%s/api/v1/characters", port)
	
	r.Run(":" + port)
}