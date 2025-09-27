package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/jung-kurt/gofpdf"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Story struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Content        string            `json:"content"`
	AgeGroup       string            `json:"age_group"`
	Theme          string            `json:"theme"`
	StoryLength    string            `json:"story_length"`
	ReadingTime    int               `json:"reading_time_minutes"`
	CharacterNames []string          `json:"character_names"`
	PageCount      int               `json:"page_count"`
	CreatedAt      time.Time         `json:"created_at"`
	TimesRead      int               `json:"times_read"`
	LastRead       *time.Time        `json:"last_read"`
	IsFavorite     bool              `json:"is_favorite"`
	Illustrations  map[string]string `json:"illustrations,omitempty"` // Page number to illustration mapping
}

type GenerateStoryRequest struct {
	AgeGroup       string   `json:"age_group"`
	Theme          string   `json:"theme"`
	Length         string   `json:"length"`
	CharacterNames []string `json:"character_names"`
}

type ReadingSession struct {
	ID        string    `json:"id"`
	StoryID   string    `json:"story_id"`
	StartedAt time.Time `json:"started_at"`
	PagesRead int       `json:"pages_read"`
}

var db *sql.DB

// Simple in-memory cache for frequently accessed stories
type StoryCache struct {
	mu      sync.RWMutex
	stories map[string]*Story
	maxSize int
}

var storyCache = &StoryCache{
	stories: make(map[string]*Story),
	maxSize: 20, // Cache up to 20 most recent stories
}

func (c *StoryCache) Get(id string) (*Story, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	story, ok := c.stories[id]
	return story, ok
}

func (c *StoryCache) Set(id string, story *Story) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple eviction: if cache is full, remove oldest entry
	if len(c.stories) >= c.maxSize {
		// Remove a random entry (simple strategy)
		for k := range c.stories {
			delete(c.stories, k)
			break
		}
	}
	c.stories[id] = story
}

func (c *StoryCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.stories, id)
}

func main() {
	// Protect against direct execution - must be run through lifecycle system
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start bedtime-story-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	godotenv.Load()

	// Initialize database
	initDB()
	defer db.Close()

	// Setup routes
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Story endpoints
	router.HandleFunc("/api/v1/stories", getStoriesHandler).Methods("GET")
	router.HandleFunc("/api/v1/stories/generate", generateStoryHandler).Methods("POST")
	router.HandleFunc("/api/v1/stories/{id}", getStoryHandler).Methods("GET")
	router.HandleFunc("/api/v1/stories/{id}/favorite", toggleFavoriteHandler).Methods("POST")
	router.HandleFunc("/api/v1/stories/{id}/read", startReadingHandler).Methods("POST")
	router.HandleFunc("/api/v1/stories/{id}/export", exportStoryHandler).Methods("GET")
	router.HandleFunc("/api/v1/stories/{id}", deleteStoryHandler).Methods("DELETE")

	// Theme endpoints
	router.HandleFunc("/api/v1/themes", getThemesHandler).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})

	handler := c.Handler(router)

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	log.Printf("Bedtime Story API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func initDB() {
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
			log.Fatal("❌ Missing database configuration. Provide POSTGRES_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}

		postgresURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	var err error
	db, err = sql.Open("postgres", postgresURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
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
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
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
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	// Run database migration to ensure illustrations column exists
	runDatabaseMigration()
}

func runDatabaseMigration() {
	// Check if illustrations column exists and add if not
	var columnExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'stories' 
			AND column_name = 'illustrations'
		)
	`).Scan(&columnExists)

	if err != nil {
		log.Printf("⚠️  Failed to check illustrations column: %v", err)
		return
	}

	if !columnExists {
		log.Println("📦 Adding illustrations column to stories table...")
		_, err := db.Exec(`
			ALTER TABLE stories 
			ADD COLUMN illustrations JSONB DEFAULT '{}'::jsonb
		`)

		if err != nil {
			log.Printf("⚠️  Failed to add illustrations column: %v", err)
		} else {
			log.Println("✅ Illustrations column added successfully")
		}
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "bedtime-story-generator",
		"timestamp": time.Now(),
	})
}

func generateStoryHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR: Invalid request body: %v", err)
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.AgeGroup == "" {
		req.AgeGroup = "6-8"
	}
	if req.Theme == "" {
		req.Theme = "Adventure"
	}
	if req.Length == "" {
		req.Length = "medium"
	}

	// Validate age group
	validAgeGroups := map[string]bool{"3-5": true, "6-8": true, "9-12": true}
	if !validAgeGroups[req.AgeGroup] {
		log.Printf("ERROR: Invalid age group: %s", req.AgeGroup)
		http.Error(w, fmt.Sprintf("Invalid age group '%s'. Valid options: 3-5, 6-8, 9-12", req.AgeGroup), http.StatusBadRequest)
		return
	}

	// Validate theme
	validThemes := map[string]bool{
		"Adventure": true, "Animals": true, "Bedtime": true, "Dinosaurs": true,
		"Fairy Tales": true, "Fantasy": true, "Forest": true, "Friendship": true,
		"Magic": true, "Ocean": true, "Space": true,
	}
	if !validThemes[req.Theme] {
		log.Printf("ERROR: Invalid theme: %s", req.Theme)
		http.Error(w, fmt.Sprintf("Invalid theme '%s'. Valid options: Adventure, Animals, Bedtime, Dinosaurs, Fairy Tales, Fantasy, Forest, Friendship, Magic, Ocean, Space", req.Theme), http.StatusBadRequest)
		return
	}

	// Validate length
	validLengths := map[string]bool{"short": true, "medium": true, "long": true}
	if !validLengths[req.Length] {
		log.Printf("ERROR: Invalid length: %s", req.Length)
		http.Error(w, fmt.Sprintf("Invalid length '%s'. Valid options: short, medium, long", req.Length), http.StatusBadRequest)
		return
	}

	// Generate story using Ollama
	log.Printf("Generating story: age_group=%s, theme=%s, length=%s", req.AgeGroup, req.Theme, req.Length)
	story, err := generateStoryWithOllama(req)
	if err != nil {
		log.Printf("ERROR: Story generation failed: %v", err)
		http.Error(w, "Failed to generate story. Please try again.", http.StatusInternalServerError)
		return
	}
	log.Printf("Story generated successfully: %s", story.Title)

	// Generate illustrations for the story
	illustrations := generateIllustrations(story.Content, req.Theme)
	story.Illustrations = illustrations
	log.Printf("Generated %d illustrations for story", len(illustrations))

	// Save to database
	id := uuid.New().String()
	readingTime := calculateReadingTime(req.Length)
	pageCount := calculatePageCount(story.Content)

	charactersJSON, _ := json.Marshal(req.CharacterNames)
	illustrationsJSON, _ := json.Marshal(illustrations)

	_, err = db.Exec(`
		INSERT INTO stories (id, title, content, age_group, theme, story_length, 
			reading_time_minutes, page_count, character_names, illustrations)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, story.Title, story.Content, req.AgeGroup, req.Theme, req.Length,
		readingTime, pageCount, string(charactersJSON), string(illustrationsJSON))

	if err != nil {
		http.Error(w, "Failed to save story: "+err.Error(), http.StatusInternalServerError)
		return
	}

	story.ID = id
	story.ReadingTime = readingTime
	story.PageCount = pageCount
	story.CreatedAt = time.Now()
	story.TimesRead = 0
	story.IsFavorite = false

	// Cache the newly generated story
	storyCache.Set(id, story)
	log.Printf("Cached newly generated story: %s", id)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(story)
}

func generateStoryWithOllama(req GenerateStoryRequest) (*Story, error) {
	// Build the prompt
	// Optimize prompt for faster generation - very concise
	prompt := fmt.Sprintf(`Write a %s bedtime story about %s for age %s.%s
Format: TITLE: [title]\nSTORY:\n## Page 1\n[story with ## Page N markers, calm and soothing]`,
		req.Length, req.Theme, req.AgeGroup,
		getCharacterPrompt(req.CharacterNames))

	// Use Ollama HTTP API
	// Try OLLAMA_URL first (full URL), then OLLAMA_BASE_URL, then OLLAMA_HOST with port
	ollamaHost := os.Getenv("OLLAMA_URL")
	if ollamaHost == "" {
		ollamaHost = os.Getenv("OLLAMA_BASE_URL")
	}
	if ollamaHost == "" {
		host := os.Getenv("OLLAMA_HOST")
		port := os.Getenv("OLLAMA_PORT")
		if host != "" && port != "" {
			ollamaHost = fmt.Sprintf("http://%s:%s", host, port)
		} else {
			ollamaHost = "http://localhost:11434"
		}
	}

	// Use faster model for better performance
	model := os.Getenv("STORY_MODEL")
	if model == "" {
		model = "llama3.2:1b" // Much smaller and faster 1B model
	}

	// Adjust token limit based on story length - reduced for speed
	tokenLimit := 300 // short - reduced from 500
	if req.Length == "medium" {
		tokenLimit = 500 // reduced from 800
	} else if req.Length == "long" {
		tokenLimit = 800 // reduced from 1200
	}

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature":    0.6, // Lower for more focused output
			"num_predict":    tokenLimit,
			"top_k":          20,   // Reduced from 40 for speed
			"top_p":          0.85, // Slightly lower for speed
			"repeat_penalty": 1.1,
			"seed":           42, // Consistent seed for caching benefit
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// Log the request for debugging
	log.Printf("Calling Ollama API at %s/api/generate with model %s", ollamaHost, model)

	resp, err := http.Post(ollamaHost+"/api/generate", "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("ERROR: Failed to call Ollama API at %s: %v", ollamaHost, err)
		return nil, fmt.Errorf("failed to call Ollama API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Ollama API returned status %d", resp.StatusCode)
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	var ollamaResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResponse); err != nil {
		log.Printf("ERROR: Failed to decode Ollama response: %v", err)
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Check for error in response
	if errMsg, hasError := ollamaResponse["error"].(string); hasError {
		log.Printf("ERROR: Ollama returned error: %s", errMsg)
		return nil, fmt.Errorf("Ollama error: %s", errMsg)
	}

	response, ok := ollamaResponse["response"].(string)
	if !ok {
		log.Printf("ERROR: Invalid response format from Ollama: %+v", ollamaResponse)
		return nil, fmt.Errorf("invalid response from Ollama")
	}

	// Parse the response
	title, content := parseStoryResponse(response)

	return &Story{
		Title:          title,
		Content:        content,
		AgeGroup:       req.AgeGroup,
		Theme:          req.Theme,
		StoryLength:    req.Length,
		CharacterNames: req.CharacterNames,
		CreatedAt:      time.Now(),
	}, nil
}

func parseStoryResponse(response string) (title, content string) {
	lines := strings.Split(response, "\n")

	for i, line := range lines {
		if strings.HasPrefix(line, "TITLE:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "TITLE:"))
		} else if strings.HasPrefix(line, "STORY:") {
			content = strings.Join(lines[i+1:], "\n")
			content = strings.TrimSpace(content)
			break
		}
	}

	// Fallback if parsing fails
	if title == "" {
		title = "A Magical Bedtime Story"
	}
	if content == "" {
		content = response
	}

	return title, content
}

// generateIllustrations creates simple emoji-based illustrations for story pages
func generateIllustrations(content string, theme string) map[string]string {
	illustrations := make(map[string]string)

	// Theme-based emoji sets
	themeEmojis := map[string][]string{
		"adventure": {"🏔️", "🗺️", "🎒", "⛺", "🌟", "🔦", "🧭", "🏕️"},
		"animals":   {"🐻", "🦊", "🐰", "🦉", "🦌", "🐿️", "🦋", "🐝"},
		"fantasy":   {"🏰", "🧙", "🦄", "🐉", "✨", "🔮", "🧚", "👑"},
		"space":     {"🚀", "🌟", "🌙", "🪐", "⭐", "🛸", "👽", "🌌"},
		"ocean":     {"🐠", "🐙", "🐢", "🦈", "🌊", "🐚", "🦀", "🏝️"},
		"forest":    {"🌲", "🦫", "🦌", "🍄", "🦉", "🐻", "🌳", "🌿"},
		"magic":     {"✨", "🎩", "🔮", "🪄", "💫", "🌟", "🦄", "🧚"},
		"dinosaur":  {"🦕", "🦖", "🦴", "🌋", "🌿", "🥚", "👣", "🏔️"},
	}

	// Default emoji set if theme not found
	defaultEmojis := []string{"🌟", "🌙", "✨", "💫", "🌈", "☁️", "🎈", "🎀"}

	// Select emoji set based on theme
	themeKey := strings.ToLower(theme)
	emojis, exists := themeEmojis[themeKey]
	if !exists {
		// Try to find partial match
		for key, value := range themeEmojis {
			if strings.Contains(themeKey, key) || strings.Contains(key, themeKey) {
				emojis = value
				break
			}
		}
		if emojis == nil {
			emojis = defaultEmojis
		}
	}

	// Generate a simple illustration for each page
	pages := strings.Split(content, "## Page")
	pageNum := 0

	for i, page := range pages {
		if strings.TrimSpace(page) == "" {
			continue
		}

		// Create a simple emoji pattern
		var illustration strings.Builder

		// Add decorative border
		illustration.WriteString("╔════════════════╗\n")
		illustration.WriteString("║                ║\n")

		// Add 2-3 rows of themed emojis
		for row := 0; row < 2; row++ {
			illustration.WriteString("║  ")
			for col := 0; col < 3; col++ {
				emojiIdx := (pageNum + row*3 + col) % len(emojis)
				illustration.WriteString(emojis[emojiIdx])
				illustration.WriteString("  ")
			}
			illustration.WriteString(" ║\n")
		}

		illustration.WriteString("║                ║\n")
		illustration.WriteString("╚════════════════╝")

		// Store illustration for this page
		if i == 0 {
			illustrations["cover"] = illustration.String()
		} else {
			illustrations[fmt.Sprintf("page_%d", pageNum)] = illustration.String()
		}
		pageNum++
	}

	return illustrations
}

func getAgeDescription(ageGroup string) string {
	switch ageGroup {
	case "3-5":
		return "ages 3-5 (very simple vocabulary, short sentences, repetition)"
	case "6-8":
		return "ages 6-8 (simple vocabulary, clear plot, relatable characters)"
	case "9-12":
		return "ages 9-12 (richer vocabulary, complex plot, character development)"
	default:
		return "ages 6-8"
	}
}

func getLengthDescription(length string) string {
	switch length {
	case "short":
		return "3-5 minute read (about 300-500 words)"
	case "medium":
		return "8-10 minute read (about 800-1200 words)"
	case "long":
		return "12-15 minute read (about 1500-2000 words)"
	default:
		return "8-10 minute read"
	}
}

func getCharacterPrompt(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return fmt.Sprintf("\n- Include these character names: %s", strings.Join(names, ", "))
}

func calculateReadingTime(length string) int {
	switch length {
	case "short":
		return 5
	case "medium":
		return 10
	case "long":
		return 15
	default:
		return 10
	}
}

func calculatePageCount(content string) int {
	// Count ## Page markers in content
	count := strings.Count(content, "## Page")
	if count == 0 {
		// Estimate based on word count
		words := len(strings.Fields(content))
		return (words / 200) + 1 // Roughly 200 words per page
	}
	return count
}

func getStoriesHandler(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, title, content, age_group, theme, story_length, 
			reading_time_minutes, page_count, character_names::text, 
			created_at, times_read, last_read, is_favorite,
			COALESCE(illustrations::text, '{}') as illustrations
		FROM stories
		ORDER BY created_at DESC
		LIMIT 50`

	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		var charactersJSON, illustrationsJSON string
		err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.AgeGroup, &s.Theme,
			&s.StoryLength, &s.ReadingTime, &s.PageCount, &charactersJSON,
			&s.CreatedAt, &s.TimesRead, &s.LastRead, &s.IsFavorite, &illustrationsJSON)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(charactersJSON), &s.CharacterNames)
		json.Unmarshal([]byte(illustrationsJSON), &s.Illustrations)
		stories = append(stories, s)
	}

	json.NewEncoder(w).Encode(stories)
}

func getStoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Check cache first
	if cachedStory, ok := storyCache.Get(id); ok {
		log.Printf("Cache hit for story: %s", id)
		json.NewEncoder(w).Encode(cachedStory)
		return
	}

	var s Story
	var charactersJSON, illustrationsJSON string
	err := db.QueryRow(`
		SELECT id, title, content, age_group, theme, story_length,
			reading_time_minutes, page_count, character_names::text,
			created_at, times_read, last_read, is_favorite,
			COALESCE(illustrations::text, '{}') as illustrations
		FROM stories WHERE id = $1`, id).Scan(
		&s.ID, &s.Title, &s.Content, &s.AgeGroup, &s.Theme,
		&s.StoryLength, &s.ReadingTime, &s.PageCount, &charactersJSON,
		&s.CreatedAt, &s.TimesRead, &s.LastRead, &s.IsFavorite, &illustrationsJSON)

	if err == sql.ErrNoRows {
		http.Error(w, "Story not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.Unmarshal([]byte(charactersJSON), &s.CharacterNames)
	json.Unmarshal([]byte(illustrationsJSON), &s.Illustrations)

	// Add to cache for future requests
	storyCache.Set(id, &s)
	log.Printf("Cached story: %s", id)

	json.NewEncoder(w).Encode(s)
}

func toggleFavoriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var isFavorite bool
	err := db.QueryRow(`
		UPDATE stories 
		SET is_favorite = NOT is_favorite, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING is_favorite`, id).Scan(&isFavorite)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Story not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"is_favorite": isFavorite,
	})
}

func startReadingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	storyID := vars["id"]

	sessionID := uuid.New().String()
	_, err := db.Exec(`
		INSERT INTO reading_sessions (id, story_id, started_at)
		VALUES ($1, $2, $3)`, sessionID, storyID, time.Now())

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"session_id": sessionID,
		"story_id":   storyID,
	})
}

func deleteStoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := db.Exec("DELETE FROM stories WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove from cache if present
	storyCache.Delete(id)
	log.Printf("Removed story from cache: %s", id)

	w.WriteHeader(http.StatusNoContent)
}

func exportStoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Try to get from cache first
	if cachedStory, found := storyCache.Get(id); found {
		sendPDFExport(w, cachedStory)
		return
	}

	// Fetch from database if not in cache
	var story Story
	var charactersJSON, illustrationsJSON sql.NullString
	err := db.QueryRow(`
		SELECT id, title, content, age_group, theme, story_length, 
		       reading_time_minutes, character_names, page_count, 
		       created_at, times_read, last_read, is_favorite, illustrations
		FROM stories 
		WHERE id = $1`, id).Scan(
		&story.ID, &story.Title, &story.Content, &story.AgeGroup,
		&story.Theme, &story.StoryLength, &story.ReadingTime,
		&charactersJSON, &story.PageCount, &story.CreatedAt,
		&story.TimesRead, &story.LastRead, &story.IsFavorite, &illustrationsJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Story not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Parse JSON fields
	if charactersJSON.Valid {
		json.Unmarshal([]byte(charactersJSON.String), &story.CharacterNames)
	}
	if illustrationsJSON.Valid {
		json.Unmarshal([]byte(illustrationsJSON.String), &story.Illustrations)
	}

	sendPDFExport(w, &story)
}

func sendPDFExport(w http.ResponseWriter, story *Story) {
	// Create a proper PDF document using gofpdf
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Add fonts and set defaults
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 24)

	// Title
	pdf.SetTextColor(107, 70, 193) // Purple color
	pdf.CellFormat(190, 15, story.Title, "0", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Metadata box
	pdf.SetFont("Arial", "", 12)
	pdf.SetTextColor(75, 85, 99)    // Gray color
	pdf.SetFillColor(243, 244, 246) // Light gray background

	// Draw metadata box
	y := pdf.GetY()
	pdf.Rect(10, y, 190, 30, "F")
	pdf.SetXY(15, y+5)
	pdf.Cell(0, 6, fmt.Sprintf("Age Group: %s", story.AgeGroup))
	pdf.Ln(6)
	pdf.SetX(15)
	pdf.Cell(0, 6, fmt.Sprintf("Theme: %s", story.Theme))
	pdf.Ln(6)
	pdf.SetX(15)
	pdf.Cell(0, 6, fmt.Sprintf("Reading Time: %d minutes | Pages: %d", story.ReadingTime, story.PageCount))
	pdf.Ln(15)

	// Story content
	pdf.SetFont("Arial", "", 11)
	pdf.SetTextColor(0, 0, 0) // Black text

	// Split content into paragraphs
	paragraphs := strings.Split(story.Content, "\n\n")
	for _, paragraph := range paragraphs {
		// Handle page headers (## Page X)
		if strings.HasPrefix(paragraph, "##") {
			pdf.AddPage()
			pdf.SetFont("Arial", "B", 14)
			pdf.Cell(0, 10, strings.TrimPrefix(paragraph, "## "))
			pdf.Ln(10)
			pdf.SetFont("Arial", "", 11)
			continue
		}

		// Remove markdown bold syntax
		cleanParagraph := strings.ReplaceAll(paragraph, "**", "")
		cleanParagraph = strings.TrimSpace(cleanParagraph)

		if cleanParagraph != "" {
			// Calculate line height based on content length
			lines := pdf.SplitLines([]byte(cleanParagraph), 180)
			for _, line := range lines {
				pdf.Cell(0, 6, string(line))
				pdf.Ln(6)
			}
			pdf.Ln(4) // Extra space between paragraphs
		}
	}

	// Footer
	pdf.Ln(10)
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(156, 163, 175) // Light gray
	pdf.Cell(0, 6, fmt.Sprintf("Generated on %s by Bedtime Story Generator", story.CreatedAt.Format("January 2, 2006")))
	pdf.Ln(5)
	pdf.Cell(0, 6, "© Vrooli - Creating magical moments for children")

	// Generate PDF to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		log.Printf("Error generating PDF: %v", err)
		http.Error(w, "Error generating PDF", http.StatusInternalServerError)
		return
	}

	// Set headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.pdf\"",
		strings.ReplaceAll(story.Title, " ", "_")))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

	w.Write(buf.Bytes())
	log.Printf("Exported story as PDF: %s", story.ID)
}

func getThemesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT name, description, emoji, color
		FROM story_themes
		ORDER BY name`)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var themes []map[string]string
	for rows.Next() {
		var name, description, emoji, color string
		rows.Scan(&name, &description, &emoji, &color)
		themes = append(themes, map[string]string{
			"name":        name,
			"description": description,
			"emoji":       emoji,
			"color":       color,
		})
	}

	json.NewEncoder(w).Encode(themes)
}
