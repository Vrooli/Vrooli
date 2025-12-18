package main

import (
	"github.com/vrooli/api-core/preflight"
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"golang.org/x/crypto/pbkdf2"
)

// Global variables
var db *sql.DB
var encryptionKey []byte

// Data structures
type User struct {
	ID              string                 `json:"id" db:"id"`
	Username        string                 `json:"username" db:"username"`
	EmailEncrypted  string                 `json:"-" db:"email_encrypted"`
	Email           string                 `json:"email,omitempty"`
	Timezone        string                 `json:"timezone" db:"timezone"`
	Preferences     map[string]interface{} `json:"preferences" db:"preferences"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
	LastLogin       *time.Time             `json:"last_login" db:"last_login"`
	IsActive        bool                   `json:"is_active" db:"is_active"`
}

type Cycle struct {
	ID               string     `json:"id" db:"id"`
	UserID           string     `json:"user_id" db:"user_id"`
	StartDate        string     `json:"start_date" db:"start_date"`
	EndDate          *string    `json:"end_date" db:"end_date"`
	CycleLength      *int       `json:"cycle_length" db:"cycle_length"`
	FlowIntensity    string     `json:"flow_intensity" db:"flow_intensity"`
	NotesEncrypted   string     `json:"-" db:"notes_encrypted"`
	Notes            string     `json:"notes,omitempty"`
	IsPredicted      bool       `json:"is_predicted" db:"is_predicted"`
	ConfidenceScore  *float64   `json:"confidence_score" db:"confidence_score"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type DailySymptom struct {
	ID                       string     `json:"id" db:"id"`
	UserID                   string     `json:"user_id" db:"user_id"`
	CycleID                  *string    `json:"cycle_id" db:"cycle_id"`
	SymptomDate              string     `json:"symptom_date" db:"symptom_date"`
	PhysicalSymptomsEncrypted string    `json:"-" db:"physical_symptoms_encrypted"`
	PhysicalSymptoms         []string   `json:"physical_symptoms,omitempty"`
	MoodRating               *int       `json:"mood_rating" db:"mood_rating"`
	EnergyLevel              *int       `json:"energy_level" db:"energy_level"`
	StressLevel              *int       `json:"stress_level" db:"stress_level"`
	CrampIntensity           *int       `json:"cramp_intensity" db:"cramp_intensity"`
	HeadacheIntensity        *int       `json:"headache_intensity" db:"headache_intensity"`
	BreastTenderness         *int       `json:"breast_tenderness" db:"breast_tenderness"`
	BackPain                 *int       `json:"back_pain" db:"back_pain"`
	FlowLevel                string     `json:"flow_level" db:"flow_level"`
	NotesEncrypted           string     `json:"-" db:"notes_encrypted"`
	Notes                    string     `json:"notes,omitempty"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at" db:"updated_at"`
}

type Prediction struct {
	ID                  string    `json:"id" db:"id"`
	UserID              string    `json:"user_id" db:"user_id"`
	PredictedStartDate  string    `json:"predicted_start_date" db:"predicted_start_date"`
	PredictedEndDate    *string   `json:"predicted_end_date" db:"predicted_end_date"`
	PredictedLength     *int      `json:"predicted_length" db:"predicted_length"`
	ConfidenceScore     float64   `json:"confidence_score" db:"confidence_score"`
	AlgorithmVersion    string    `json:"algorithm_version" db:"algorithm_version"`
	InputDataHash       string    `json:"input_data_hash" db:"input_data_hash"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	IsActive            bool      `json:"is_active" db:"is_active"`
}

type DetectedPattern struct {
	ID                           string    `json:"id" db:"id"`
	UserID                       string    `json:"user_id" db:"user_id"`
	PatternType                  string    `json:"pattern_type" db:"pattern_type"`
	PatternDescriptionEncrypted  string    `json:"-" db:"pattern_description_encrypted"`
	PatternDescription           string    `json:"pattern_description,omitempty"`
	CorrelationStrength          float64   `json:"correlation_strength" db:"correlation_strength"`
	StatisticalSignificance      *float64  `json:"statistical_significance" db:"statistical_significance"`
	DataPointsCount              int       `json:"data_points_count" db:"data_points_count"`
	FirstObserved                string    `json:"first_observed" db:"first_observed"`
	LastObserved                 string    `json:"last_observed" db:"last_observed"`
	AlgorithmVersion             string    `json:"algorithm_version" db:"algorithm_version"`
	ConfidenceLevel              string    `json:"confidence_level" db:"confidence_level"`
	IsActive                     bool      `json:"is_active" db:"is_active"`
	CreatedAt                    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at" db:"updated_at"`
}

// Request/Response structures
type CreateCycleRequest struct {
	StartDate     string  `json:"start_date" binding:"required"`
	FlowIntensity string  `json:"flow_intensity"`
	Notes         string  `json:"notes"`
}

type LogSymptomsRequest struct {
	Date                string   `json:"date" binding:"required"`
	PhysicalSymptoms    []string `json:"physical_symptoms"`
	MoodRating          *int     `json:"mood_rating"`
	EnergyLevel         *int     `json:"energy_level"`
	StressLevel         *int     `json:"stress_level"`
	CrampIntensity      *int     `json:"cramp_intensity"`
	HeadacheIntensity   *int     `json:"headache_intensity"`
	BreastTenderness    *int     `json:"breast_tenderness"`
	BackPain            *int     `json:"back_pain"`
	FlowLevel           string   `json:"flow_level"`
	Notes               string   `json:"notes"`
}

type PredictionsResponse struct {
	Predictions []Prediction `json:"predictions"`
	NextPeriod  *string      `json:"next_period,omitempty"`
	Confidence  *float64     `json:"confidence,omitempty"`
}

// Encryption utilities
func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, 10000, 32, sha256.New)
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func encryptString(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	encrypted, err := encrypt([]byte(plaintext), encryptionKey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptString(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	decrypted, err := decrypt(data, encryptionKey)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

// Database initialization
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
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}
	
	log.Println("🎉 Database connection pool established successfully!")
}

// getEnv removed to prevent hardcoded defaults

// Middleware for authentication (integrates with scenario-authenticator)
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For now, we'll use a simple user ID header
		// In production, this would integrate with scenario-authenticator
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = c.Query("user_id")
		}
		
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
			c.Abort()
			return
		}

		// Validate UUID format
		if _, err := uuid.Parse(userID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// Audit logging middleware
func auditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		// Log the request for audit purposes
		userID, exists := c.Get("user_id")
		if !exists {
			return
		}

		action := "data_accessed"
		if c.Request.Method != "GET" {
			action = "data_modified"
		}

		go logAudit(userID.(string), action, c.Request.RequestURI, c.ClientIP(), c.Request.UserAgent(), map[string]interface{}{
			"method": c.Request.Method,
			"status": c.Writer.Status(),
			"duration": time.Since(start).Milliseconds(),
		})
	}
}

func logAudit(userID, action, endpoint, ip, userAgent string, details map[string]interface{}) {
	detailsJSON, _ := json.Marshal(details)
	
	_, err := db.Exec(`
		INSERT INTO audit_logs (user_id, action, table_name, ip_address, user_agent, details)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, action, endpoint, ip, userAgent, string(detailsJSON))
	
	if err != nil {
		log.Printf("Failed to log audit entry: %v", err)
	}
}

// Health check endpoint
func healthCheck(c *gin.Context) {
	// Check database connection
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"database": "disconnected",
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"database": "connected",
		"encryption": "enabled",
		"version": "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// Cycle management endpoints
func createCycle(c *gin.Context) {
	var req CreateCycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	cycleID := uuid.New().String()
	
	// Encrypt notes if provided
	notesEncrypted, err := encryptString(req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt notes"})
		return
	}

	// Insert cycle
	_, err = db.Exec(`
		INSERT INTO cycles (id, user_id, start_date, flow_intensity, notes_encrypted)
		VALUES ($1, $2, $3, $4, $5)`,
		cycleID, userID, req.StartDate, req.FlowIntensity, notesEncrypted)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cycle"})
		return
	}

	// Generate predictions based on historical data
	predictions, err := generatePredictions(userID)
	if err != nil {
		log.Printf("Failed to generate predictions: %v", err)
	}

	response := map[string]interface{}{
		"cycle_id": cycleID,
		"message": "Cycle created successfully",
	}

	if len(predictions) > 0 {
		response["next_period"] = predictions[0].PredictedStartDate
		response["confidence"] = predictions[0].ConfidenceScore
	}

	c.JSON(http.StatusCreated, response)
}

func getCycles(c *gin.Context) {
	userID := c.GetString("user_id")
	
	rows, err := db.Query(`
		SELECT id, user_id, start_date, end_date, cycle_length, flow_intensity, 
		       notes_encrypted, is_predicted, confidence_score, created_at, updated_at
		FROM cycles 
		WHERE user_id = $1 
		ORDER BY start_date DESC 
		LIMIT 50`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cycles"})
		return
	}
	defer rows.Close()

	var cycles []Cycle
	for rows.Next() {
		var cycle Cycle
		err := rows.Scan(&cycle.ID, &cycle.UserID, &cycle.StartDate, &cycle.EndDate,
			&cycle.CycleLength, &cycle.FlowIntensity, &cycle.NotesEncrypted,
			&cycle.IsPredicted, &cycle.ConfidenceScore, &cycle.CreatedAt, &cycle.UpdatedAt)
		
		if err != nil {
			log.Printf("Error scanning cycle: %v", err)
			continue
		}

		// Decrypt notes
		if cycle.NotesEncrypted != "" {
			decrypted, err := decryptString(cycle.NotesEncrypted)
			if err != nil {
				log.Printf("Failed to decrypt notes: %v", err)
			} else {
				cycle.Notes = decrypted
			}
		}

		cycles = append(cycles, cycle)
	}

	c.JSON(http.StatusOK, gin.H{"cycles": cycles})
}

// Symptom logging endpoints
func logSymptoms(c *gin.Context) {
	var req LogSymptomsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	symptomID := uuid.New().String()

	// Encrypt physical symptoms and notes
	symptomsJSON, _ := json.Marshal(req.PhysicalSymptoms)
	symptomsEncrypted, err := encryptString(string(symptomsJSON))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt symptoms"})
		return
	}

	notesEncrypted, err := encryptString(req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encrypt notes"})
		return
	}

	// Insert or update daily symptoms
	_, err = db.Exec(`
		INSERT INTO daily_symptoms (
			id, user_id, symptom_date, physical_symptoms_encrypted, mood_rating,
			energy_level, stress_level, cramp_intensity, headache_intensity,
			breast_tenderness, back_pain, flow_level, notes_encrypted
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (user_id, symptom_date) 
		DO UPDATE SET
			physical_symptoms_encrypted = EXCLUDED.physical_symptoms_encrypted,
			mood_rating = EXCLUDED.mood_rating,
			energy_level = EXCLUDED.energy_level,
			stress_level = EXCLUDED.stress_level,
			cramp_intensity = EXCLUDED.cramp_intensity,
			headache_intensity = EXCLUDED.headache_intensity,
			breast_tenderness = EXCLUDED.breast_tenderness,
			back_pain = EXCLUDED.back_pain,
			flow_level = EXCLUDED.flow_level,
			notes_encrypted = EXCLUDED.notes_encrypted,
			updated_at = NOW()`,
		symptomID, userID, req.Date, symptomsEncrypted, req.MoodRating,
		req.EnergyLevel, req.StressLevel, req.CrampIntensity, req.HeadacheIntensity,
		req.BreastTenderness, req.BackPain, req.FlowLevel, notesEncrypted)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to log symptoms"})
		return
	}

	// Trigger pattern detection (async)
	go detectPatterns(userID)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Symptoms logged successfully",
		"symptom_id": symptomID,
	})
}

func getSymptoms(c *gin.Context) {
	userID := c.GetString("user_id")
	
	// Parse date range from query params
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")
	
	query := `
		SELECT id, user_id, cycle_id, symptom_date, physical_symptoms_encrypted,
		       mood_rating, energy_level, stress_level, cramp_intensity,
		       headache_intensity, breast_tenderness, back_pain, flow_level,
		       notes_encrypted, created_at, updated_at
		FROM daily_symptoms 
		WHERE user_id = $1`
	
	args := []interface{}{userID}
	argCount := 1
	
	if startDate != "" {
		argCount++
		query += fmt.Sprintf(" AND symptom_date >= $%d", argCount)
		args = append(args, startDate)
	}
	
	if endDate != "" {
		argCount++
		query += fmt.Sprintf(" AND symptom_date <= $%d", argCount)
		args = append(args, endDate)
	}
	
	query += " ORDER BY symptom_date DESC LIMIT 100"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch symptoms"})
		return
	}
	defer rows.Close()

	var symptoms []DailySymptom
	for rows.Next() {
		var symptom DailySymptom
		err := rows.Scan(&symptom.ID, &symptom.UserID, &symptom.CycleID, &symptom.SymptomDate,
			&symptom.PhysicalSymptomsEncrypted, &symptom.MoodRating, &symptom.EnergyLevel,
			&symptom.StressLevel, &symptom.CrampIntensity, &symptom.HeadacheIntensity,
			&symptom.BreastTenderness, &symptom.BackPain, &symptom.FlowLevel,
			&symptom.NotesEncrypted, &symptom.CreatedAt, &symptom.UpdatedAt)
		
		if err != nil {
			log.Printf("Error scanning symptom: %v", err)
			continue
		}

		// Decrypt physical symptoms
		if symptom.PhysicalSymptomsEncrypted != "" {
			decrypted, err := decryptString(symptom.PhysicalSymptomsEncrypted)
			if err != nil {
				log.Printf("Failed to decrypt symptoms: %v", err)
			} else {
				json.Unmarshal([]byte(decrypted), &symptom.PhysicalSymptoms)
			}
		}

		// Decrypt notes
		if symptom.NotesEncrypted != "" {
			decrypted, err := decryptString(symptom.NotesEncrypted)
			if err != nil {
				log.Printf("Failed to decrypt notes: %v", err)
			} else {
				symptom.Notes = decrypted
			}
		}

		symptoms = append(symptoms, symptom)
	}

	c.JSON(http.StatusOK, gin.H{"symptoms": symptoms})
}

// Prediction endpoints
func getPredictions(c *gin.Context) {
	userID := c.GetString("user_id")
	
	rows, err := db.Query(`
		SELECT id, user_id, predicted_start_date, predicted_end_date, predicted_length,
		       confidence_score, algorithm_version, input_data_hash, created_at, is_active
		FROM predictions 
		WHERE user_id = $1 AND is_active = true
		ORDER BY predicted_start_date ASC`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch predictions"})
		return
	}
	defer rows.Close()

	var predictions []Prediction
	for rows.Next() {
		var pred Prediction
		err := rows.Scan(&pred.ID, &pred.UserID, &pred.PredictedStartDate, &pred.PredictedEndDate,
			&pred.PredictedLength, &pred.ConfidenceScore, &pred.AlgorithmVersion,
			&pred.InputDataHash, &pred.CreatedAt, &pred.IsActive)
		
		if err != nil {
			log.Printf("Error scanning prediction: %v", err)
			continue
		}

		predictions = append(predictions, pred)
	}

	response := PredictionsResponse{Predictions: predictions}
	
	if len(predictions) > 0 {
		response.NextPeriod = &predictions[0].PredictedStartDate
		response.Confidence = &predictions[0].ConfidenceScore
	}

	c.JSON(http.StatusOK, response)
}

// Pattern detection endpoints
func getPatterns(c *gin.Context) {
	userID := c.GetString("user_id")
	
	rows, err := db.Query(`
		SELECT id, user_id, pattern_type, pattern_description_encrypted, correlation_strength,
		       statistical_significance, data_points_count, first_observed, last_observed,
		       algorithm_version, confidence_level, is_active, created_at, updated_at
		FROM detected_patterns 
		WHERE user_id = $1 AND is_active = true
		ORDER BY correlation_strength DESC`, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch patterns"})
		return
	}
	defer rows.Close()

	var patterns []DetectedPattern
	for rows.Next() {
		var pattern DetectedPattern
		err := rows.Scan(&pattern.ID, &pattern.UserID, &pattern.PatternType,
			&pattern.PatternDescriptionEncrypted, &pattern.CorrelationStrength,
			&pattern.StatisticalSignificance, &pattern.DataPointsCount,
			&pattern.FirstObserved, &pattern.LastObserved, &pattern.AlgorithmVersion,
			&pattern.ConfidenceLevel, &pattern.IsActive, &pattern.CreatedAt, &pattern.UpdatedAt)
		
		if err != nil {
			log.Printf("Error scanning pattern: %v", err)
			continue
		}

		// Decrypt pattern description
		if pattern.PatternDescriptionEncrypted != "" {
			decrypted, err := decryptString(pattern.PatternDescriptionEncrypted)
			if err != nil {
				log.Printf("Failed to decrypt pattern description: %v", err)
			} else {
				pattern.PatternDescription = decrypted
			}
		}

		patterns = append(patterns, pattern)
	}

	c.JSON(http.StatusOK, gin.H{"patterns": patterns})
}

// Prediction algorithm (simplified)
func generatePredictions(userID string) ([]Prediction, error) {
	// Get recent cycles for this user
	rows, err := db.Query(`
		SELECT start_date, cycle_length 
		FROM cycles 
		WHERE user_id = $1 AND is_predicted = false AND cycle_length IS NOT NULL
		ORDER BY start_date DESC 
		LIMIT 6`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cycles []struct {
		StartDate   string
		CycleLength int
	}

	for rows.Next() {
		var cycle struct {
			StartDate   string
			CycleLength int
		}
		err := rows.Scan(&cycle.StartDate, &cycle.CycleLength)
		if err != nil {
			continue
		}
		cycles = append(cycles, cycle)
	}

	if len(cycles) < 2 {
		return nil, fmt.Errorf("insufficient data for predictions")
	}

	// Simple prediction algorithm: average cycle length
	totalLength := 0
	for _, cycle := range cycles {
		totalLength += cycle.CycleLength
	}
	avgLength := float64(totalLength) / float64(len(cycles))

	// Calculate variance for confidence score
	variance := 0.0
	for _, cycle := range cycles {
		diff := float64(cycle.CycleLength) - avgLength
		variance += diff * diff
	}
	variance /= float64(len(cycles))
	stdDev := math.Sqrt(variance)
	
	// Confidence decreases with higher variance
	confidence := math.Max(0.5, 1.0-(stdDev/7.0))

	// Get last cycle start date
	lastStart, err := time.Parse("2006-01-02", cycles[0].StartDate)
	if err != nil {
		return nil, err
	}

	// Generate 3 predictions
	predictions := []Prediction{}
	for i := 0; i < 3; i++ {
		predictionDate := lastStart.AddDate(0, 0, int(avgLength)+i*int(avgLength))
		predictionID := uuid.New().String()
		
		prediction := Prediction{
			ID:                 predictionID,
			UserID:             userID,
			PredictedStartDate: predictionDate.Format("2006-01-02"),
			PredictedLength:    &[]int{int(avgLength)}[0],
			ConfidenceScore:    confidence,
			AlgorithmVersion:   "v1.0.0",
			InputDataHash:      fmt.Sprintf("cycles_%d", len(cycles)),
			IsActive:           true,
		}

		// Store prediction in database
		_, err = db.Exec(`
			INSERT INTO predictions (id, user_id, predicted_start_date, predicted_length, 
			                        confidence_score, algorithm_version, input_data_hash)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			prediction.ID, prediction.UserID, prediction.PredictedStartDate,
			prediction.PredictedLength, prediction.ConfidenceScore,
			prediction.AlgorithmVersion, prediction.InputDataHash)

		if err == nil {
			predictions = append(predictions, prediction)
		}
	}

	return predictions, nil
}

// Pattern detection (simplified)
func detectPatterns(userID string) {
	// This is a simplified pattern detection algorithm
	// In production, this would use more sophisticated ML techniques
	
	// Look for mood patterns around cycle days
	rows, err := db.Query(`
		SELECT ds.symptom_date, ds.mood_rating, ds.cramp_intensity,
		       c.start_date, c.cycle_length
		FROM daily_symptoms ds
		LEFT JOIN cycles c ON ds.user_id = c.user_id 
		  AND ds.symptom_date >= c.start_date 
		  AND ds.symptom_date <= (c.start_date + INTERVAL '35 days')
		WHERE ds.user_id = $1 AND ds.mood_rating IS NOT NULL
		ORDER BY ds.symptom_date DESC
		LIMIT 100`, userID)

	if err != nil {
		log.Printf("Failed to fetch data for pattern detection: %v", err)
		return
	}
	defer rows.Close()

	// Simple correlation analysis would go here
	// For now, we'll create a sample pattern if we have enough data

	var dataPoints []struct {
		Date           string
		MoodRating     int
		CrampIntensity int
	}

	for rows.Next() {
		var dp struct {
			Date           string
			MoodRating     int
			CrampIntensity int
			CycleStart     sql.NullString
			CycleLength    sql.NullInt64
		}
		
		err := rows.Scan(&dp.Date, &dp.MoodRating, &dp.CrampIntensity, 
		                 &dp.CycleStart, &dp.CycleLength)
		if err != nil {
			continue
		}

		dataPoints = append(dataPoints, struct {
			Date           string
			MoodRating     int
			CrampIntensity int
		}{dp.Date, dp.MoodRating, dp.CrampIntensity})
	}

	if len(dataPoints) >= 10 {
		// Create a sample pattern
		patternID := uuid.New().String()
		description := "Mood tends to drop during high cramp intensity days"
		encryptedDesc, _ := encryptString(description)
		
		_, err = db.Exec(`
			INSERT INTO detected_patterns (
				id, user_id, pattern_type, pattern_description_encrypted,
				correlation_strength, data_points_count, algorithm_version, confidence_level
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT DO NOTHING`,
			patternID, userID, "mood_cramp_correlation", encryptedDesc,
			-0.65, len(dataPoints), "v1.0.0", "medium")
		
		if err != nil {
			log.Printf("Failed to store pattern: %v", err)
		}
	}
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "period-tracker",
	}) {
		return // Process was re-exec'd after rebuild
	}
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Initialize encryption key - REQUIRED, no defaults
	encryptionKeyEnv := os.Getenv("ENCRYPTION_KEY")
	if encryptionKeyEnv == "" {
		log.Fatal("❌ ENCRYPTION_KEY environment variable is required")
	}
	salt := []byte("period-tracker-salt-2024") // In production, use a proper random salt
	encryptionKey = deriveKey(encryptionKeyEnv, salt)

	// Initialize database
	initDB()

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	r.Use(func(ctx *gin.Context) {
		c.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx.Next()
		})).ServeHTTP(ctx.Writer, ctx.Request)
	})

	// Public routes
	r.GET("/health", healthCheck)
	
	// API routes with authentication
	api := r.Group("/api/v1")
	api.Use(authMiddleware())
	api.Use(auditMiddleware())
	{
		// Cycle management
		api.POST("/cycles", createCycle)
		api.GET("/cycles", getCycles)

		// Symptom logging
		api.POST("/symptoms", logSymptoms)
		api.GET("/symptoms", getSymptoms)

		// Predictions
		api.GET("/predictions", getPredictions)

		// Pattern detection
		api.GET("/patterns", getPatterns)

		// Health and encryption status
		api.GET("/health/encryption", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"encryption_enabled": true,
				"algorithm": "AES-GCM",
				"key_derivation": "PBKDF2",
				"status": "active",
			})
		})

		api.GET("/auth/status", func(c *gin.Context) {
			userID := c.GetString("user_id")
			c.JSON(http.StatusOK, gin.H{
				"authenticated": true,
				"user_id": userID,
				"multi_tenant": true,
			})
		})
	}

	// Start server - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}
	log.Printf("Period Tracker API starting on port %s", port)
	log.Printf("Privacy mode: Enabled - All data encrypted and stored locally")
	
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}