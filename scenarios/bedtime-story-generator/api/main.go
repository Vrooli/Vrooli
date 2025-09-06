package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

type Story struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Content           string    `json:"content"`
	AgeGroup          string    `json:"age_group"`
	Theme             string    `json:"theme"`
	StoryLength       string    `json:"story_length"`
	ReadingTime       int       `json:"reading_time_minutes"`
	CharacterNames    []string  `json:"character_names"`
	PageCount         int       `json:"page_count"`
	CreatedAt         time.Time `json:"created_at"`
	TimesRead         int       `json:"times_read"`
	LastRead          *time.Time `json:"last_read"`
	IsFavorite        bool      `json:"is_favorite"`
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

func main() {
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
	
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "20000"
	}
	
	log.Printf("Bedtime Story API starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func initDB() {
	var err error
	
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	
	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	
	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	
	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "vrooli"
	}
	
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	log.Println("Database connected successfully")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"service": "bedtime-story-generator",
		"timestamp": time.Now(),
	})
}

func generateStoryHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	
	// Generate story using Ollama
	story, err := generateStoryWithOllama(req)
	if err != nil {
		http.Error(w, "Failed to generate story: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Save to database
	id := uuid.New().String()
	readingTime := calculateReadingTime(req.Length)
	pageCount := calculatePageCount(story.Content)
	
	charactersJSON, _ := json.Marshal(req.CharacterNames)
	
	_, err = db.Exec(`
		INSERT INTO stories (id, title, content, age_group, theme, story_length, 
			reading_time_minutes, page_count, character_names)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		id, story.Title, story.Content, req.AgeGroup, req.Theme, req.Length,
		readingTime, pageCount, string(charactersJSON))
	
	if err != nil {
		http.Error(w, "Failed to save story: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	story.ID = id
	story.ReadingTime = readingTime
	story.PageCount = pageCount
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(story)
}

func generateStoryWithOllama(req GenerateStoryRequest) (*Story, error) {
	// Build the prompt
	ageDescription := getAgeDescription(req.AgeGroup)
	lengthDescription := getLengthDescription(req.Length)
	
	prompt := fmt.Sprintf(`You are a creative children's story writer. Generate a bedtime story with these requirements:
- Age group: %s (use appropriate vocabulary and complexity)
- Theme: %s
- Length: %s
- Tone: Gentle, calming, perfect for bedtime
%s

Format your response EXACTLY as:
TITLE: [Story title here]
STORY:
[Full story text here with markdown formatting for pages using ## Page N headers]

Rules:
- Make it age-appropriate and educational
- Include gentle life lessons
- Use calming, peaceful imagery
- End with a soothing conclusion perfect for sleep
- Divide into pages with ## Page 1, ## Page 2, etc.
- No scary or overly exciting content`, 
		ageDescription, req.Theme, lengthDescription,
		getCharacterPrompt(req.CharacterNames))
	
	// Call Ollama using resource CLI
	cmd := exec.Command("resource-ollama", "generate", "--model", "llama3.2:3b", "--prompt", prompt)
	output, err := cmd.Output()
	if err != nil {
		// Fallback to direct ollama command if resource-ollama fails
		cmd = exec.Command("ollama", "run", "llama3.2:3b", prompt)
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to generate story: %v", err)
		}
	}
	
	// Parse the response
	response := string(output)
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
			created_at, times_read, last_read, is_favorite
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
		var charactersJSON string
		err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.AgeGroup, &s.Theme,
			&s.StoryLength, &s.ReadingTime, &s.PageCount, &charactersJSON,
			&s.CreatedAt, &s.TimesRead, &s.LastRead, &s.IsFavorite)
		if err != nil {
			continue
		}
		json.Unmarshal([]byte(charactersJSON), &s.CharacterNames)
		stories = append(stories, s)
	}
	
	json.NewEncoder(w).Encode(stories)
}

func getStoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	var s Story
	var charactersJSON string
	err := db.QueryRow(`
		SELECT id, title, content, age_group, theme, story_length,
			reading_time_minutes, page_count, character_names::text,
			created_at, times_read, last_read, is_favorite
		FROM stories WHERE id = $1`, id).Scan(
		&s.ID, &s.Title, &s.Content, &s.AgeGroup, &s.Theme,
		&s.StoryLength, &s.ReadingTime, &s.PageCount, &charactersJSON,
		&s.CreatedAt, &s.TimesRead, &s.LastRead, &s.IsFavorite)
	
	if err == sql.ErrNoRows {
		http.Error(w, "Story not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.Unmarshal([]byte(charactersJSON), &s.CharacterNames)
	json.NewEncoder(w).Encode(s)
}

func toggleFavoriteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	
	_, err := db.Exec(`
		UPDATE stories 
		SET is_favorite = NOT is_favorite, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, id)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
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
		"story_id": storyID,
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
	
	w.WriteHeader(http.StatusNoContent)
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
			"name": name,
			"description": description,
			"emoji": emoji,
			"color": color,
		})
	}
	
	json.NewEncoder(w).Encode(themes)
}