package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Global variables
var (
	db            *pgxpool.Pool
	redisClient   *redis.Client
	logger        *logrus.Logger
	quizProcessor *QuizProcessor
)

// Quiz structures
type Quiz struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description,omitempty"`
	Questions    []Question `json:"questions"`
	TimeLimit    int        `json:"time_limit,omitempty"`
	PassingScore int        `json:"passing_score"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Question struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	QuestionText string      `json:"question"`
	Options      []string    `json:"options,omitempty"`
	CorrectAnswer interface{} `json:"correct_answer,omitempty"`
	Explanation  string      `json:"explanation,omitempty"`
	Difficulty   string      `json:"difficulty"`
	Points       int         `json:"points"`
	OrderIndex   int         `json:"order_index"`
}

type QuizGenerateRequest struct {
	Content       string   `json:"content" binding:"required"`
	QuestionCount int      `json:"question_count"`
	Difficulty    string   `json:"difficulty"`
	QuestionTypes []string `json:"question_types"`
	Tags          []string `json:"tags"`
}

type QuizSubmitRequest struct {
	Responses []QuestionResponse `json:"responses"`
	TimeTaken int                `json:"time_taken"`
}

type QuestionResponse struct {
	QuestionID string      `json:"question_id"`
	Answer     interface{} `json:"answer"`
}

type QuizResult struct {
	Score        int                       `json:"score"`
	Percentage   float64                   `json:"percentage"`
	Passed       bool                      `json:"passed"`
	CorrectAnswers map[string]interface{}  `json:"correct_answers"`
	Explanations map[string]string         `json:"explanations"`
}

type SearchRequest struct {
	Query      string   `json:"query" binding:"required"`
	Tags       []string `json:"tags"`
	Difficulty string   `json:"difficulty"`
	Limit      int      `json:"limit"`
}

// Initialize logger
func initLogger() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
}

// Initialize database connection
func initDB() error {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Build from individual components - REQUIRED, no defaults
		dbHost := os.Getenv("POSTGRES_HOST")
		dbPort := os.Getenv("POSTGRES_PORT")
		dbUser := os.Getenv("POSTGRES_USER")
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		dbName := os.Getenv("POSTGRES_DB")
		
		if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
			return fmt.Errorf("database configuration missing. Provide DATABASE_URL or all of: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
		}
		
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			dbUser, dbPassword, dbHost, dbPort, dbName)
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return fmt.Errorf("unable to parse database URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	db, err = pgxpool.New(context.Background(), config.ConnString())
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second
	
	logger.Info("🔄 Attempting database connection with exponential backoff...")
	
	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping(context.Background())
		if pingErr == nil {
			logger.Infof("✅ Database connected successfully on attempt %d", attempt + 1)
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
		
		logger.Warnf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		logger.Infof("⏳ Waiting %v before next attempt", actualDelay)
		
		time.Sleep(actualDelay)
	}
	
	if pingErr != nil {
		return fmt.Errorf("database connection failed after %d attempts: %w", maxRetries, pingErr)
	}

	logger.Info("🎉 Database connection pool established successfully!")
	return nil
}

// Initialize Redis connection
func initRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:         redisURL,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           0,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Warnf("Redis connection failed (will use fallback): %v", err)
		redisClient = nil
	} else {
		logger.Info("Redis connected successfully")
	}
}

// API Handlers

func healthCheck(c *gin.Context) {
	status := "healthy"
	dbStatus := "connected"
	redisStatus := "connected"

	// Check database
	if err := db.Ping(context.Background()); err != nil {
		dbStatus = "disconnected"
		status = "degraded"
	}

	// Check Redis
	if redisClient != nil {
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			redisStatus = "disconnected"
		}
	} else {
		redisStatus = "not configured"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   status,
		"database": dbStatus,
		"redis":    redisStatus,
		"version":  "1.0.0",
		"uptime":   time.Since(startTime).String(),
	})
}

func generateQuiz(c *gin.Context) {
	var req QuizGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	
	// Use quiz processor to generate quiz (replaces n8n workflow)
	quiz, err := quizProcessor.GenerateQuizFromContent(ctx, req)
	if err != nil {
		logger.Errorf("Failed to generate quiz: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate quiz"})
		return
	}

	// Quiz is already saved by the processor, just return the result
	c.JSON(http.StatusCreated, gin.H{
		"quiz_id":        quiz.ID,
		"title":          quiz.Title,
		"questions":      quiz.Questions,
		"estimated_time": quiz.TimeLimit,
	})
}

func getQuiz(c *gin.Context) {
	quizID := c.Param("id")
	
	var quiz Quiz
	ctx := context.Background()
	
	err := db.QueryRow(ctx,
		`SELECT id, title, description, time_limit, passing_score, created_at, updated_at
		 FROM quiz_generator.quizzes WHERE id = $1`,
		quizID,
	).Scan(&quiz.ID, &quiz.Title, &quiz.Description, &quiz.TimeLimit, &quiz.PassingScore, &quiz.CreatedAt, &quiz.UpdatedAt)
	
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quiz not found"})
		return
	}

	// TODO: Load questions from database
	quiz.Questions = generateMockQuestions(5)

	c.JSON(http.StatusOK, quiz)
}

func listQuizzes(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}

	ctx := context.Background()
	rows, err := db.Query(ctx,
		`SELECT id, title, difficulty, created_at
		 FROM quiz_generator.quizzes
		 ORDER BY created_at DESC
		 LIMIT $1`,
		limit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch quizzes"})
		return
	}
	defer rows.Close()

	quizzes := []map[string]interface{}{}
	for rows.Next() {
		var id, title, difficulty string
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &difficulty, &createdAt); err != nil {
			continue
		}
		quizzes = append(quizzes, map[string]interface{}{
			"id":             id,
			"title":          title,
			"difficulty":     difficulty,
			"question_count": 10, // Mock value
			"created_at":     createdAt,
		})
	}

	c.JSON(http.StatusOK, quizzes)
}

func submitQuiz(c *gin.Context) {
	quizID := c.Param("id")
	
	var req QuizSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := context.Background()
	
	// Use quiz processor to grade the quiz
	result, err := quizProcessor.GradeQuiz(ctx, quizID, req)
	if err != nil {
		logger.Errorf("Failed to grade quiz: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grade quiz"})
		return
	}

	// Result is already saved by the processor
	c.JSON(http.StatusOK, result)
}

func searchQuestions(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	// Mock search results
	questions := generateMockQuestions(req.Limit)
	relevanceScores := make([]float64, len(questions))
	for i := range relevanceScores {
		relevanceScores[i] = 0.8 + float64(i)*0.02
	}

	c.JSON(http.StatusOK, gin.H{
		"questions":        questions,
		"relevance_scores": relevanceScores,
	})
}

func exportQuiz(c *gin.Context) {
	quizID := c.Param("id")
	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	// Fetch quiz
	var quiz Quiz
	quiz.ID = quizID
	quiz.Title = "Sample Quiz"
	quiz.Questions = generateMockQuestions(5)

	switch format {
	case "json":
		c.JSON(http.StatusOK, quiz)
	case "qti":
		// TODO: Implement QTI format export
		c.String(http.StatusOK, "<?xml version=\"1.0\"?><qti>...</qti>")
	case "moodle":
		// TODO: Implement Moodle format export
		c.String(http.StatusOK, "<?xml version=\"1.0\"?><quiz>...</quiz>")
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported format"})
	}
}

func getStats(c *gin.Context) {
	ctx := context.Background()
	
	var totalQuizzes, totalQuestions, activeSessions int
	
	db.QueryRow(ctx, "SELECT COUNT(*) FROM quiz_generator.quizzes").Scan(&totalQuizzes)
	db.QueryRow(ctx, "SELECT COUNT(*) FROM quiz_generator.questions").Scan(&totalQuestions)
	
	// Mock active sessions
	activeSessions = 5

	c.JSON(http.StatusOK, gin.H{
		"total_quizzes":   totalQuizzes,
		"total_questions": totalQuestions,
		"active_sessions": activeSessions,
	})
}

// Helper functions

func generateMockQuestions(count int) []Question {
	questions := make([]Question, count)
	for i := 0; i < count; i++ {
		questions[i] = Question{
			ID:           fmt.Sprintf("q%d", i+1),
			Type:         "mcq",
			QuestionText: fmt.Sprintf("Sample question %d?", i+1),
			Options:      []string{"A) Option 1", "B) Option 2", "C) Option 3", "D) Option 4"},
			CorrectAnswer: "A",
			Explanation:  "This is the explanation for the correct answer.",
			Difficulty:   "medium",
			Points:       2,
			OrderIndex:   i + 1,
		}
	}
	return questions
}

// CORS middleware
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

var startTime time.Time

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start quiz-generator

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Load environment variables
	godotenv.Load()

	// Initialize components
	initLogger()
	startTime = time.Now()

	// Initialize database
	if err := initDB(); err != nil {
		logger.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize Redis (optional)
	initRedis()
	if redisClient != nil {
		defer redisClient.Close()
	}
	
	// Initialize quiz processor
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}
	quizProcessor = NewQuizProcessor(db, redisClient, ollamaURL, qdrantURL)

	// Setup Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", healthCheck)
	router.GET("/ready", healthCheck)

	// API routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/quiz/generate", generateQuiz)
		v1.GET("/quiz/:id", getQuiz)
		v1.GET("/quizzes", listQuizzes)
		v1.POST("/quiz/:id/submit", submitQuiz)
		v1.GET("/quiz/:id/export", exportQuiz)
		v1.POST("/question-bank/search", searchQuestions)
		v1.GET("/stats", getStats)
	}

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3250"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Infof("Starting Quiz Generator API on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}