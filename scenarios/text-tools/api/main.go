package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/google/uuid"
)

// Configuration
type Config struct {
	Port         string
	DatabaseURL  string
	MinIOURL     string
	RedisURL     string
	OllamaURL    string
}

// Global database connection
var db *sql.DB
var config Config

// Request/Response structures
type DiffRequest struct {
	Text1   interface{} `json:"text1"`
	Text2   interface{} `json:"text2"`
	Options DiffOptions `json:"options,omitempty"`
}

type DiffOptions struct {
	Type             string `json:"type,omitempty"`
	IgnoreWhitespace bool   `json:"ignore_whitespace,omitempty"`
	IgnoreCase       bool   `json:"ignore_case,omitempty"`
}

type DiffResponse struct {
	Changes        []Change `json:"changes"`
	SimilarityScore float64  `json:"similarity_score"`
	Summary        string   `json:"summary"`
}

type Change struct {
	Type      string `json:"type"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Content   string `json:"content"`
}

type SearchRequest struct {
	Text    interface{}   `json:"text"`
	Pattern string        `json:"pattern"`
	Options SearchOptions `json:"options,omitempty"`
}

type SearchOptions struct {
	Regex         bool `json:"regex,omitempty"`
	CaseSensitive bool `json:"case_sensitive,omitempty"`
	WholeWord     bool `json:"whole_word,omitempty"`
	Fuzzy         bool `json:"fuzzy,omitempty"`
	Semantic      bool `json:"semantic,omitempty"`
}

type SearchResponse struct {
	Matches      []Match `json:"matches"`
	TotalMatches int     `json:"total_matches"`
}

type Match struct {
	Line    int     `json:"line"`
	Column  int     `json:"column"`
	Length  int     `json:"length"`
	Context string  `json:"context"`
	Score   float64 `json:"score,omitempty"`
}

type TransformRequest struct {
	Text            string            `json:"text"`
	Transformations []Transformation  `json:"transformations"`
}

type Transformation struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type TransformResponse struct {
	Result                string   `json:"result"`
	TransformationsApplied []string `json:"transformations_applied"`
	Warnings              []string `json:"warnings,omitempty"`
}

type ExtractRequest struct {
	Source  interface{}     `json:"source"`
	Format  string          `json:"format,omitempty"`
	Options ExtractOptions  `json:"options,omitempty"`
}

type ExtractOptions struct {
	OCR                bool `json:"ocr,omitempty"`
	PreserveFormatting bool `json:"preserve_formatting,omitempty"`
	ExtractMetadata    bool `json:"extract_metadata,omitempty"`
}

type ExtractResponse struct {
	Text     string            `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
}

type AnalyzeRequest struct {
	Text     string          `json:"text"`
	Analyses []string        `json:"analyses"`
	Options  AnalyzeOptions  `json:"options,omitempty"`
}

type AnalyzeOptions struct {
	SummaryLength int      `json:"summary_length,omitempty"`
	EntityTypes   []string `json:"entity_types,omitempty"`
}

type AnalyzeResponse struct {
	Entities  []Entity  `json:"entities,omitempty"`
	Sentiment Sentiment `json:"sentiment,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	Keywords  []Keyword `json:"keywords,omitempty"`
	Language  Language  `json:"language,omitempty"`
}

type Entity struct {
	Type       string  `json:"type"`
	Value      string  `json:"value"`
	Confidence float64 `json:"confidence"`
}

type Sentiment struct {
	Score float64 `json:"score"`
	Label string  `json:"label"`
}

type Keyword struct {
	Word  string  `json:"word"`
	Score float64 `json:"score"`
}

type Language struct {
	Code       string  `json:"code"`
	Confidence float64 `json:"confidence"`
}

// Initialize database connection
func initDB() error {
	var err error
	config.DatabaseURL = os.Getenv("DATABASE_URL")
	if config.DatabaseURL == "" {
		config.DatabaseURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}
	
	db, err = sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		return err
	}
	
	// Set schema search path
	_, err = db.Exec("SET search_path TO text_tools, public")
	if err != nil {
		log.Printf("Warning: Could not set search path: %v", err)
	}
	
	return db.Ping()
}

// Initialize configuration
func initConfig() {
	config.Port = os.Getenv("TEXT_TOOLS_PORT")
	if config.Port == "" {
		config.Port = "14000"
	}
	
	config.MinIOURL = os.Getenv("MINIO_URL")
	if config.MinIOURL == "" {
		config.MinIOURL = "http://localhost:9000"
	}
	
	config.RedisURL = os.Getenv("REDIS_URL")
	if config.RedisURL == "" {
		config.RedisURL = "redis://localhost:6379"
	}
	
	config.OllamaURL = os.Getenv("OLLAMA_URL")
	if config.OllamaURL == "" {
		config.OllamaURL = "http://localhost:11434"
	}
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"database": "unknown",
	}
	
	// Check database connection
	if db != nil {
		err := db.Ping()
		if err == nil {
			health["database"] = "connected"
		} else {
			health["database"] = "disconnected"
			health["status"] = "degraded"
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Diff endpoint
func diffHandler(w http.ResponseWriter, r *http.Request) {
	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Extract text from request
	text1 := extractText(req.Text1)
	text2 := extractText(req.Text2)
	
	// Perform diff based on type
	diffType := req.Options.Type
	if diffType == "" {
		diffType = "line"
	}
	
	var changes []Change
	var similarity float64
	
	switch diffType {
	case "line":
		changes, similarity = performLineDiff(text1, text2, req.Options)
	case "word":
		changes, similarity = performWordDiff(text1, text2, req.Options)
	case "character":
		changes, similarity = performCharDiff(text1, text2, req.Options)
	default:
		changes, similarity = performLineDiff(text1, text2, req.Options)
	}
	
	// Create summary
	summary := fmt.Sprintf("Found %d changes with %.2f%% similarity", len(changes), similarity*100)
	
	// Store operation in database
	go storeOperation("diff", map[string]interface{}{
		"type": diffType,
		"text1_length": len(text1),
		"text2_length": len(text2),
		"changes": len(changes),
		"similarity": similarity,
	})
	
	response := DiffResponse{
		Changes:         changes,
		SimilarityScore: similarity,
		Summary:         summary,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Search endpoint
func searchHandler(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	text := extractText(req.Text)
	
	var matches []Match
	
	if req.Options.Regex {
		matches = performRegexSearch(text, req.Pattern, req.Options)
	} else if req.Options.Fuzzy {
		matches = performFuzzySearch(text, req.Pattern, req.Options)
	} else {
		matches = performPlainSearch(text, req.Pattern, req.Options)
	}
	
	// Store operation
	go storeOperation("search", map[string]interface{}{
		"pattern": req.Pattern,
		"options": req.Options,
		"matches": len(matches),
	})
	
	response := SearchResponse{
		Matches:      matches,
		TotalMatches: len(matches),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Transform endpoint
func transformHandler(w http.ResponseWriter, r *http.Request) {
	var req TransformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	result := req.Text
	applied := []string{}
	warnings := []string{}
	
	for _, transform := range req.Transformations {
		switch transform.Type {
		case "case":
			caseType := transform.Parameters["type"].(string)
			if caseType == "upper" {
				result = strings.ToUpper(result)
			} else if caseType == "lower" {
				result = strings.ToLower(result)
			} else if caseType == "title" {
				result = strings.Title(result)
			}
			applied = append(applied, fmt.Sprintf("case_%s", caseType))
			
		case "encode":
			// Handle encoding transformations
			applied = append(applied, "encode")
			
		case "format":
			// Handle format transformations
			applied = append(applied, "format")
			
		case "sanitize":
			// Remove HTML tags, normalize whitespace, etc.
			result = sanitizeText(result)
			applied = append(applied, "sanitize")
		}
	}
	
	// Store operation
	go storeOperation("transform", map[string]interface{}{
		"transformations": applied,
		"input_length": len(req.Text),
		"output_length": len(result),
	})
	
	response := TransformResponse{
		Result:                result,
		TransformationsApplied: applied,
		Warnings:              warnings,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Extract endpoint
func extractHandler(w http.ResponseWriter, r *http.Request) {
	var req ExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Extract text based on source type
	text := ""
	metadata := make(map[string]interface{})
	warnings := []string{}
	
	// For now, simple extraction
	switch v := req.Source.(type) {
	case string:
		text = v
	case map[string]interface{}:
		if url, ok := v["url"].(string); ok {
			// Extract from URL
			text = fmt.Sprintf("Text extracted from URL: %s", url)
			metadata["source"] = "url"
		} else if docID, ok := v["document_id"].(string); ok {
			// Extract from database
			text = fmt.Sprintf("Text from document: %s", docID)
			metadata["source"] = "database"
		}
	}
	
	// Store operation
	go storeOperation("extract", map[string]interface{}{
		"format": req.Format,
		"text_length": len(text),
	})
	
	response := ExtractResponse{
		Text:     text,
		Metadata: metadata,
		Warnings: warnings,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Analyze endpoint
func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	response := AnalyzeResponse{}
	
	for _, analysis := range req.Analyses {
		switch analysis {
		case "entities":
			response.Entities = extractEntities(req.Text)
		case "sentiment":
			response.Sentiment = analyzeSentiment(req.Text)
		case "summary":
			response.Summary = generateSummary(req.Text, req.Options.SummaryLength)
		case "keywords":
			response.Keywords = extractKeywords(req.Text)
		case "language":
			response.Language = detectLanguage(req.Text)
		}
	}
	
	// Store operation
	go storeOperation("analyze", map[string]interface{}{
		"analyses": req.Analyses,
		"text_length": len(req.Text),
	})
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions
func extractText(input interface{}) string {
	switch v := input.(type) {
	case string:
		return v
	case map[string]interface{}:
		if text, ok := v["text"].(string); ok {
			return text
		}
		if docID, ok := v["document_id"].(string); ok {
			// Fetch from database
			return fetchDocumentText(docID)
		}
	}
	return ""
}

func fetchDocumentText(docID string) string {
	var text string
	err := db.QueryRow("SELECT content FROM text_documents WHERE id = $1", docID).Scan(&text)
	if err != nil {
		return ""
	}
	return text
}

func performLineDiff(text1, text2 string, options DiffOptions) ([]Change, float64) {
	lines1 := strings.Split(text1, "\n")
	lines2 := strings.Split(text2, "\n")
	
	changes := []Change{}
	matchCount := 0
	
	// Simple line-by-line comparison
	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}
	
	for i := 0; i < maxLines; i++ {
		if i >= len(lines1) {
			changes = append(changes, Change{
				Type:      "add",
				LineStart: i + 1,
				LineEnd:   i + 1,
				Content:   lines2[i],
			})
		} else if i >= len(lines2) {
			changes = append(changes, Change{
				Type:      "remove",
				LineStart: i + 1,
				LineEnd:   i + 1,
				Content:   lines1[i],
			})
		} else if lines1[i] != lines2[i] {
			changes = append(changes, Change{
				Type:      "modify",
				LineStart: i + 1,
				LineEnd:   i + 1,
				Content:   fmt.Sprintf("%s -> %s", lines1[i], lines2[i]),
			})
		} else {
			matchCount++
		}
	}
	
	similarity := float64(matchCount) / float64(maxLines)
	return changes, similarity
}

func performWordDiff(text1, text2 string, options DiffOptions) ([]Change, float64) {
	// Simple word-based diff
	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)
	
	changes := []Change{}
	matchCount := 0
	maxWords := len(words1)
	if len(words2) > maxWords {
		maxWords = len(words2)
	}
	
	for i := 0; i < maxWords; i++ {
		if i >= len(words1) || i >= len(words2) || words1[i] != words2[i] {
			changes = append(changes, Change{
				Type:      "modify",
				LineStart: 0,
				LineEnd:   0,
				Content:   "Word difference detected",
			})
		} else {
			matchCount++
		}
	}
	
	similarity := float64(matchCount) / float64(maxWords)
	return changes, similarity
}

func performCharDiff(text1, text2 string, options DiffOptions) ([]Change, float64) {
	// Character-level diff
	changes := []Change{}
	matchCount := 0
	maxLen := len(text1)
	if len(text2) > maxLen {
		maxLen = len(text2)
	}
	
	for i := 0; i < maxLen; i++ {
		if i >= len(text1) || i >= len(text2) || text1[i] != text2[i] {
			changes = append(changes, Change{
				Type:      "modify",
				LineStart: 0,
				LineEnd:   0,
				Content:   "Character difference",
			})
		} else {
			matchCount++
		}
	}
	
	similarity := float64(matchCount) / float64(maxLen)
	return changes, similarity
}

func performPlainSearch(text, pattern string, options SearchOptions) []Match {
	matches := []Match{}
	lines := strings.Split(text, "\n")
	
	searchPattern := pattern
	if !options.CaseSensitive {
		searchPattern = strings.ToLower(pattern)
	}
	
	for lineNum, line := range lines {
		searchLine := line
		if !options.CaseSensitive {
			searchLine = strings.ToLower(line)
		}
		
		if idx := strings.Index(searchLine, searchPattern); idx >= 0 {
			matches = append(matches, Match{
				Line:    lineNum + 1,
				Column:  idx + 1,
				Length:  len(pattern),
				Context: line,
			})
		}
	}
	
	return matches
}

func performRegexSearch(text, pattern string, options SearchOptions) []Match {
	matches := []Match{}
	
	flags := ""
	if !options.CaseSensitive {
		flags = "(?i)"
	}
	
	re, err := regexp.Compile(flags + pattern)
	if err != nil {
		return matches
	}
	
	lines := strings.Split(text, "\n")
	for lineNum, line := range lines {
		if locs := re.FindAllStringIndex(line, -1); locs != nil {
			for _, loc := range locs {
				matches = append(matches, Match{
					Line:    lineNum + 1,
					Column:  loc[0] + 1,
					Length:  loc[1] - loc[0],
					Context: line,
				})
			}
		}
	}
	
	return matches
}

func performFuzzySearch(text, pattern string, options SearchOptions) []Match {
	// Simple fuzzy search implementation
	return performPlainSearch(text, pattern, options)
}

func sanitizeText(text string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text = re.ReplaceAllString(text, "")
	
	// Normalize whitespace
	text = strings.TrimSpace(text)
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	
	return text
}

func extractEntities(text string) []Entity {
	// Simple entity extraction
	entities := []Entity{}
	
	// Extract emails
	emailRe := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	for _, match := range emailRe.FindAllString(text, -1) {
		entities = append(entities, Entity{
			Type:       "email",
			Value:      match,
			Confidence: 0.95,
		})
	}
	
	// Extract URLs
	urlRe := regexp.MustCompile(`https?://[^\s]+`)
	for _, match := range urlRe.FindAllString(text, -1) {
		entities = append(entities, Entity{
			Type:       "url",
			Value:      match,
			Confidence: 0.95,
		})
	}
	
	return entities
}

func analyzeSentiment(text string) Sentiment {
	// Simple sentiment analysis
	positiveWords := []string{"good", "great", "excellent", "happy", "love", "wonderful"}
	negativeWords := []string{"bad", "terrible", "hate", "awful", "horrible", "poor"}
	
	score := 0.0
	textLower := strings.ToLower(text)
	
	for _, word := range positiveWords {
		if strings.Contains(textLower, word) {
			score += 0.1
		}
	}
	
	for _, word := range negativeWords {
		if strings.Contains(textLower, word) {
			score -= 0.1
		}
	}
	
	label := "neutral"
	if score > 0.2 {
		label = "positive"
	} else if score < -0.2 {
		label = "negative"
	}
	
	return Sentiment{
		Score: score,
		Label: label,
	}
}

func generateSummary(text string, length int) string {
	if length == 0 {
		length = 100
	}
	
	if len(text) <= length {
		return text
	}
	
	return text[:length] + "..."
}

func extractKeywords(text string) []Keyword {
	// Simple keyword extraction based on word frequency
	words := strings.Fields(strings.ToLower(text))
	wordCount := make(map[string]int)
	
	for _, word := range words {
		// Skip common words
		if len(word) > 3 {
			wordCount[word]++
		}
	}
	
	keywords := []Keyword{}
	for word, count := range wordCount {
		if count > 1 {
			keywords = append(keywords, Keyword{
				Word:  word,
				Score: float64(count) / float64(len(words)),
			})
		}
	}
	
	return keywords
}

func detectLanguage(text string) Language {
	// Simple language detection
	// In production, would use a proper library or API
	return Language{
		Code:       "en",
		Confidence: 0.85,
	}
}

func storeOperation(opType string, params map[string]interface{}) {
	if db == nil {
		return
	}
	
	id := uuid.New().String()
	paramsJSON, _ := json.Marshal(params)
	
	_, err := db.Exec(`
		INSERT INTO text_operations (id, operation_type, parameters, status, created_at)
		VALUES ($1, $2, $3, 'completed', $4)
	`, id, opType, paramsJSON, time.Now())
	
	if err != nil {
		log.Printf("Failed to store operation: %v", err)
	}
}

func calculateHash(text string) string {
	hash := sha256.Sum256([]byte(text))
	return hex.EncodeToString(hash[:])
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Initialize configuration
	initConfig()
	
	// Initialize database
	if err := initDB(); err != nil {
		log.Printf("Warning: Database connection failed: %v", err)
		log.Println("Running in degraded mode without database")
	} else {
		defer db.Close()
		log.Println("Database connected successfully")
	}
	
	// Create router
	router := mux.NewRouter()
	
	// Register routes
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/api/v1/text/diff", diffHandler).Methods("POST")
	router.HandleFunc("/api/v1/text/search", searchHandler).Methods("POST")
	router.HandleFunc("/api/v1/text/transform", transformHandler).Methods("POST")
	router.HandleFunc("/api/v1/text/extract", extractHandler).Methods("POST")
	router.HandleFunc("/api/v1/text/analyze", analyzeHandler).Methods("POST")
	
	// Apply CORS middleware
	handler := corsMiddleware(router)
	
	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("Text Tools API starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}