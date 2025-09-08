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
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
)

type Article struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	URL            string    `json:"url"`
	Source         string    `json:"source"`
	PublishedAt    time.Time `json:"published_at"`
	Summary        string    `json:"summary"`
	BiasScore      float64   `json:"bias_score"`
	BiasAnalysis   string    `json:"bias_analysis"`
	PerspectiveCount int     `json:"perspective_count"`
	FetchedAt      time.Time `json:"fetched_at"`
}

type Feed struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Category    string    `json:"category"`
	BiasRating  string    `json:"bias_rating"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var (
	db *sql.DB
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
	clients = make(map[*websocket.Conn]bool)
	broadcast = make(chan interface{})
	clientsMutex sync.RWMutex
)

func main() {
	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Database configuration - support both POSTGRES_URL and individual components
	postgresURL := os.Getenv("POSTGRES_URL")
	if postgresURL == "" {
		// Try to build from individual components
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
	defer db.Close()
	
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
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")

	// Initialize Redis client (optional)
	var redisClient *redis.Client
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err == nil {
			redisClient = redis.NewClient(opt)
		}
	}

	// Initialize feed processor - OLLAMA_URL is required
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		log.Fatal("❌ OLLAMA_URL environment variable is required")
	}
	processor = NewFeedProcessor(db, ollamaURL, redisClient)

	// Start background feed processing
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		
		// Initial processing
		time.Sleep(10 * time.Second) // Wait for system to stabilize
		processor.ProcessAllFeeds()
		
		for range ticker.C {
			log.Println("Processing all feeds...")
			processor.ProcessAllFeeds()
		}
	}()

	router := mux.NewRouter()
	
	router.HandleFunc("/health", healthHandler).Methods("GET")
	router.HandleFunc("/articles", getArticlesHandler).Methods("GET")
	router.HandleFunc("/articles/{id}", getArticleHandler).Methods("GET")
	router.HandleFunc("/feeds", getFeedsHandler).Methods("GET")
	router.HandleFunc("/feeds", addFeedHandler).Methods("POST")
	router.HandleFunc("/feeds/{id}", updateFeedHandler).Methods("PUT")
	router.HandleFunc("/feeds/{id}", deleteFeedHandler).Methods("DELETE")
	router.HandleFunc("/refresh", refreshFeedsHandler).Methods("POST")
	router.HandleFunc("/analyze/{id}", analyzeBiasHandler).Methods("POST")
	router.HandleFunc("/perspectives/{topic}", getPerspectivesHandler).Methods("GET")
	router.HandleFunc("/perspectives/aggregate", aggregatePerspectivesHandler).Methods("POST")
	router.HandleFunc("/ws", handleWebSocket)

	// Start WebSocket broadcast handler
	go handleBroadcast()

	handler := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	}).Handler(router)

	log.Printf("News Aggregator API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}


func healthHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "news-aggregator-bias-analysis",
	})
}

func getArticlesHandler(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	source := r.URL.Query().Get("source")
	limit := r.URL.Query().Get("limit")
	
	if limit == "" {
		limit = "50"
	}

	query := `
		SELECT id, title, url, source, published_at, summary, 
		       bias_score, bias_analysis, perspective_count, fetched_at
		FROM articles
		WHERE 1=1
	`
	
	args := []interface{}{}
	argCount := 0

	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND category = $%d", argCount)
		args = append(args, category)
	}
	
	if source != "" {
		argCount++
		query += fmt.Sprintf(" AND source = $%d", argCount)
		args = append(args, source)
	}
	
	argCount++
	query += fmt.Sprintf(" ORDER BY published_at DESC LIMIT $%d", argCount)
	args = append(args, limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	articles := []Article{}
	for rows.Next() {
		var a Article
		err := rows.Scan(&a.ID, &a.Title, &a.URL, &a.Source, &a.PublishedAt,
			&a.Summary, &a.BiasScore, &a.BiasAnalysis, &a.PerspectiveCount, &a.FetchedAt)
		if err != nil {
			continue
		}
		articles = append(articles, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}

func getArticleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var a Article
	err := db.QueryRow(`
		SELECT id, title, url, source, published_at, summary,
		       bias_score, bias_analysis, perspective_count, fetched_at
		FROM articles WHERE id = $1
	`, id).Scan(&a.ID, &a.Title, &a.URL, &a.Source, &a.PublishedAt,
		&a.Summary, &a.BiasScore, &a.BiasAnalysis, &a.PerspectiveCount, &a.FetchedAt)
	
	if err != nil {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(a)
}

func getFeedsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, url, category, bias_rating, active, created_at, updated_at
		FROM feeds ORDER BY name
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	feeds := []Feed{}
	for rows.Next() {
		var f Feed
		err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Category, 
			&f.BiasRating, &f.Active, &f.CreatedAt, &f.UpdatedAt)
		if err != nil {
			continue
		}
		feeds = append(feeds, f)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feeds)
}

func addFeedHandler(w http.ResponseWriter, r *http.Request) {
	var feed Feed
	if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	feed.Active = true

	err := db.QueryRow(`
		INSERT INTO feeds (name, url, category, bias_rating, active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, feed.Name, feed.URL, feed.Category, feed.BiasRating, feed.Active).Scan(&feed.ID, &feed.CreatedAt, &feed.UpdatedAt)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

func updateFeedHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid feed ID", http.StatusBadRequest)
		return
	}

	var feed Feed
	if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`
		UPDATE feeds 
		SET name = $2, url = $3, category = $4, bias_rating = $5, active = $6
		WHERE id = $1
	`, id, feed.Name, feed.URL, feed.Category, feed.BiasRating, feed.Active)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	feed.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(feed)
}

func deleteFeedHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	_, err := db.Exec("DELETE FROM feeds WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func refreshFeedsHandler(w http.ResponseWriter, r *http.Request) {
	// Process all feeds asynchronously
	go func() {
		// Broadcast refresh status to WebSocket clients
		broadcastUpdate("refresh_started", map[string]string{
			"status": "in_progress",
			"message": "Fetching latest news from all sources",
		})
		
		if processor != nil {
			err := processor.ProcessAllFeeds()
			if err != nil {
				log.Printf("Error processing feeds: %v", err)
			}
		}
		
		broadcastUpdate("refresh_complete", map[string]string{
			"status": "complete",
			"message": "News feed refresh completed",
		})
	}()

	json.NewEncoder(w).Encode(map[string]string{
		"status": "refresh_triggered",
		"message": "Feed refresh initiated",
	})
}

func analyzeBiasHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Get article from database
	var article Article
	query := `SELECT id, title, url, source, summary FROM articles WHERE id = $1`
	err := db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.URL, &article.Source, &article.Summary)
	if err != nil {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	// Analyze bias using processor
	if processor != nil {
		processor.analyzeArticleBias(&article)
		
		// Update article in database
		updateQuery := `UPDATE articles SET bias_score = $1, bias_analysis = $2 WHERE id = $3`
		db.Exec(updateQuery, article.BiasScore, article.BiasAnalysis, article.ID)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "analysis_completed",
		"article_id": id,
		"bias_score": article.BiasScore,
		"bias_analysis": article.BiasAnalysis,
	})
}

func getPerspectivesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic := vars["topic"]

	rows, err := db.Query(`
		SELECT id, title, source, bias_score, bias_analysis
		FROM articles
		WHERE title ILIKE $1 OR summary ILIKE $1
		ORDER BY ABS(bias_score) DESC
		LIMIT 10
	`, "%"+topic+"%")
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	perspectives := []map[string]interface{}{}
	for rows.Next() {
		var id, title, source, analysis string
		var score float64
		err := rows.Scan(&id, &title, &source, &score, &analysis)
		if err != nil {
			continue
		}
		perspectives = append(perspectives, map[string]interface{}{
			"id": id,
			"title": title,
			"source": source,
			"bias_score": score,
			"bias_analysis": analysis,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perspectives)
}

func aggregatePerspectivesHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Topic string `json:"topic"`
		TimeRange string `json:"time_range"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if request.Topic == "" {
		http.Error(w, "Topic is required", http.StatusBadRequest)
		return
	}

	// Default time range to 24 hours
	timeRange := "24 hours"
	if request.TimeRange != "" {
		timeRange = request.TimeRange
	}

	// Get articles from different bias perspectives
	rows, err := db.Query(`
		WITH bias_groups AS (
			SELECT 
				CASE 
					WHEN bias_score < -33 THEN 'left'
					WHEN bias_score > 33 THEN 'right'
					ELSE 'center'
				END as bias_group,
				id, title, source, summary, bias_score, bias_analysis, published_at
			FROM articles
			WHERE (title ILIKE $1 OR summary ILIKE $1)
				AND published_at > NOW() - INTERVAL $2
		),
		ranked_articles AS (
			SELECT *,
				ROW_NUMBER() OVER (PARTITION BY bias_group ORDER BY published_at DESC) as rn
			FROM bias_groups
		)
		SELECT id, title, source, summary, bias_score, bias_analysis, bias_group, published_at
		FROM ranked_articles
		WHERE rn <= 3
		ORDER BY bias_score
	`, "%"+request.Topic+"%", timeRange)
	
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	leftArticles := []map[string]interface{}{}
	centerArticles := []map[string]interface{}{}
	rightArticles := []map[string]interface{}{}
	
	for rows.Next() {
		var id, title, source, summary, analysis, biasGroup string
		var score float64
		var publishedAt time.Time
		
		err := rows.Scan(&id, &title, &source, &summary, &score, &analysis, &biasGroup, &publishedAt)
		if err != nil {
			continue
		}
		
		article := map[string]interface{}{
			"id": id,
			"title": title,
			"source": source,
			"summary": summary,
			"bias_score": score,
			"bias_analysis": analysis,
			"published_at": publishedAt,
		}
		
		switch biasGroup {
		case "left":
			leftArticles = append(leftArticles, article)
		case "right":
			rightArticles = append(rightArticles, article)
		default:
			centerArticles = append(centerArticles, article)
		}
	}

	// Trigger perspective aggregation workflow in n8n
	n8nURL := os.Getenv("N8N_BASE_URL")
	if n8nURL != "" {
		payload, _ := json.Marshal(map[string]interface{}{
			"topic": request.Topic,
			"left_articles": leftArticles,
			"center_articles": centerArticles,
			"right_articles": rightArticles,
		})
		
		client := &http.Client{Timeout: 30 * time.Second}
		go func() {
			client.Post(n8nURL+"/webhook/perspective-aggregator", "application/json", 
				bytes.NewBuffer(payload))
		}()
	}

	response := map[string]interface{}{
		"topic": request.Topic,
		"time_range": timeRange,
		"perspectives": map[string]interface{}{
			"left": map[string]interface{}{
				"count": len(leftArticles),
				"articles": leftArticles,
			},
			"center": map[string]interface{}{
				"count": len(centerArticles),
				"articles": centerArticles,
			},
			"right": map[string]interface{}{
				"count": len(rightArticles),
				"articles": rightArticles,
			},
		},
		"total_articles": len(leftArticles) + len(centerArticles) + len(rightArticles),
		"bias_distribution": map[string]float64{
			"left_percentage": float64(len(leftArticles)) / float64(len(leftArticles) + len(centerArticles) + len(rightArticles)) * 100,
			"center_percentage": float64(len(centerArticles)) / float64(len(leftArticles) + len(centerArticles) + len(rightArticles)) * 100,
			"right_percentage": float64(len(rightArticles)) / float64(len(leftArticles) + len(centerArticles) + len(rightArticles)) * 100,
		},
		"aggregated_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	// Send initial connection message
	conn.WriteJSON(map[string]string{
		"type": "connected",
		"message": "Connected to news aggregator real-time updates",
	})

	// Keep connection alive and handle messages
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			clientsMutex.Lock()
			delete(clients, conn)
			clientsMutex.Unlock()
			break
		}

		// Handle client messages (e.g., subscribe to topics)
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "subscribe":
				if topic, ok := msg["topic"].(string); ok {
					// Send immediate update for subscribed topic
					go sendTopicUpdate(conn, topic)
				}
			case "ping":
				conn.WriteJSON(map[string]string{"type": "pong"})
			}
		}
	}
}

func handleBroadcast() {
	for {
		msg := <-broadcast
		clientsMutex.RLock()
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
			}
		}
		clientsMutex.RUnlock()
	}
}

func broadcastUpdate(updateType string, data interface{}) {
	update := map[string]interface{}{
		"type": updateType,
		"data": data,
		"timestamp": time.Now().Unix(),
	}
	
	select {
	case broadcast <- update:
	default:
		// Channel full, skip this update
	}
}

func sendTopicUpdate(conn *websocket.Conn, topic string) {
	// Query latest articles for topic
	rows, err := db.Query(`
		SELECT id, title, source, bias_score, published_at
		FROM articles
		WHERE title ILIKE $1 OR summary ILIKE $1
		ORDER BY published_at DESC
		LIMIT 5
	`, "%"+topic+"%")
	
	if err != nil {
		return
	}
	defer rows.Close()

	articles := []map[string]interface{}{}
	for rows.Next() {
		var id, title, source string
		var score float64
		var publishedAt time.Time
		
		if err := rows.Scan(&id, &title, &source, &score, &publishedAt); err == nil {
			articles = append(articles, map[string]interface{}{
				"id": id,
				"title": title,
				"source": source,
				"bias_score": score,
				"published_at": publishedAt,
			})
		}
	}

	conn.WriteJSON(map[string]interface{}{
		"type": "topic_update",
		"topic": topic,
		"articles": articles,
		"timestamp": time.Now().Unix(),
	})
}
