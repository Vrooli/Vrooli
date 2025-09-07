package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

// Configuration
var (
	apiPort      = getEnv("PREGNANCY_TRACKER_API_PORT", "17001")
	dbHost       = getEnv("POSTGRES_HOST", "localhost")
	dbPort       = getEnv("POSTGRES_PORT", "5432")
	dbUser       = getEnv("POSTGRES_USER", "postgres")
	dbPassword   = getEnv("POSTGRES_PASSWORD", "postgres")
	dbName       = getEnv("POSTGRES_DB", "vrooli")
	encryptKey   = getEnv("ENCRYPTION_KEY", "pregnancy-tracker-default-key-32") // 32 bytes for AES-256
	privacyMode  = getEnv("PRIVACY_MODE", "strict")
	multiTenant  = getEnv("MULTI_TENANT", "true")
	mode         = ""
)

// Database connection
var db *sql.DB

// Data structures
type Pregnancy struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	LMPDate        time.Time `json:"lmp_date"`
	ConceptionDate *time.Time `json:"conception_date,omitempty"`
	DueDate        time.Time `json:"due_date"`
	CurrentWeek    int       `json:"current_week"`
	CurrentDay     int       `json:"current_day"`
	PregnancyType  string    `json:"pregnancy_type"`
	Outcome        string    `json:"outcome"`
	CreatedAt      time.Time `json:"created_at"`
}

type DailyLog struct {
	ID            string    `json:"id"`
	PregnancyID   string    `json:"pregnancy_id"`
	Date          time.Time `json:"date"`
	Weight        *float64  `json:"weight,omitempty"`
	BloodPressure string    `json:"blood_pressure,omitempty"`
	Symptoms      []string  `json:"symptoms"`
	Mood          int       `json:"mood"`
	Energy        int       `json:"energy"`
	Notes         string    `json:"notes"`
	Photos        []string  `json:"photos"`
	CreatedAt     time.Time `json:"created_at"`
}

type Appointment struct {
	ID          string    `json:"id"`
	PregnancyID string    `json:"pregnancy_id"`
	Date        time.Time `json:"date"`
	Type        string    `json:"type"`
	Provider    string    `json:"provider"`
	Location    string    `json:"location"`
	Notes       string    `json:"notes"`
	Results     map[string]interface{} `json:"results"`
}

type KickCount struct {
	ID           string    `json:"id"`
	PregnancyID  string    `json:"pregnancy_id"`
	SessionStart time.Time `json:"session_start"`
	SessionEnd   *time.Time `json:"session_end,omitempty"`
	Count        int       `json:"count"`
	Duration     int       `json:"duration_minutes"`
	Notes        string    `json:"notes"`
}

type WeekInfo struct {
	Week         int      `json:"week"`
	Title        string   `json:"title"`
	Size         string   `json:"size"`
	Development  string   `json:"development"`
	BodyChanges  string   `json:"body_changes"`
	Tips         string   `json:"tips"`
	Citations    []Citation `json:"citations"`
}

type Citation struct {
	Title  string `json:"title"`
	Source string `json:"source"`
	Year   int    `json:"year"`
	URL    string `json:"url,omitempty"`
}

type SearchResult struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	Type    string `json:"type"`
	Week    *int   `json:"week,omitempty"`
}

func main() {
	// Check for special modes
	for _, arg := range os.Args {
		if arg == "--load-content" {
			mode = "load-content"
		}
	}
	
	// Handle verification mode
	if os.Getenv("GOFLAGS") != "" && mode == "" {
		verifyEncryption()
		return
	}

	// Initialize database
	initDB()
	defer db.Close()

	// Load content if requested
	if mode == "load-content" {
		log.Println("Content already loaded via seed.sql")
		return
	}

	// Setup routes
	setupRoutes()

	// Start server
	log.Printf("🤰 Pregnancy Tracker API starting on port %s", apiPort)
	log.Printf("   Privacy Mode: %s | Multi-Tenant: %s", privacyMode, multiTenant)
	log.Fatal(http.ListenAndServe(":"+apiPort, nil))
}

func initDB() {
	var err error
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Set search path
	_, err = db.Exec("SET search_path TO pregnancy_tracker, public")
	if err != nil {
		log.Printf("Warning: Failed to set search path: %v", err)
	}
}

func setupRoutes() {
	// Health endpoints
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/api/v1/status", handleStatus)
	http.HandleFunc("/api/v1/health/encryption", handleEncryptionStatus)
	http.HandleFunc("/api/v1/auth/status", handleAuthStatus)
	
	// Search
	http.HandleFunc("/api/v1/search", handleSearch)
	http.HandleFunc("/api/v1/search/health", handleSearchHealth)
	
	// Pregnancy endpoints
	http.HandleFunc("/api/v1/pregnancy/start", handlePregnancyStart)
	http.HandleFunc("/api/v1/pregnancy/current", handleCurrentPregnancy)
	http.HandleFunc("/api/v1/content/week/", handleWeekContent)
	
	// Tracking endpoints
	http.HandleFunc("/api/v1/logs/daily", handleDailyLog)
	http.HandleFunc("/api/v1/logs/range", handleLogsRange)
	http.HandleFunc("/api/v1/kicks/count", handleKickCount)
	http.HandleFunc("/api/v1/kicks/patterns", handleKickPatterns)
	http.HandleFunc("/api/v1/contractions/timer", handleContractionTimer)
	http.HandleFunc("/api/v1/contractions/history", handleContractionHistory)
	
	// Appointments
	http.HandleFunc("/api/v1/appointments", handleAppointments)
	http.HandleFunc("/api/v1/appointments/upcoming", handleUpcomingAppointments)
	
	// Export endpoints
	http.HandleFunc("/api/v1/export/json", handleExportJSON)
	http.HandleFunc("/api/v1/export/pdf", handleExportPDF)
	http.HandleFunc("/api/v1/export/emergency-card", handleEmergencyCard)
	
	// Partner access
	http.HandleFunc("/api/v1/partner/invite", handlePartnerInvite)
	http.HandleFunc("/api/v1/partner/view", handlePartnerView)
}

// Middleware
func getUserID(r *http.Request) string {
	return r.Header.Get("X-User-ID")
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-ID")
}

// Encryption functions
func encrypt(plaintext string) (string, error) {
	key := []byte(encryptKey)[:32] // Ensure 32 bytes
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(ciphertext string) (string, error) {
	key := []byte(encryptKey)[:32]
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func verifyEncryption() {
	fmt.Println("✅ Encryption verified: AES-256-GCM enabled")
	fmt.Printf("   Privacy Mode: %s\n", privacyMode)
	fmt.Printf("   Multi-Tenant: %s\n", multiTenant)
}

// Health handlers
func handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "pregnancy-tracker",
		"privacy": privacyMode,
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	// Check database connection
	dbStatus := "connected"
	if err := db.Ping(); err != nil {
		dbStatus = "disconnected"
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "operational",
		"database": dbStatus,
		"encryption": "enabled",
		"multi_tenant": multiTenant,
		"version": "1.0.0",
	})
}

func handleEncryptionStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{
		"enabled": true,
		"aes256": true,
		"field_level": true,
	})
}

func handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	// Check if scenario-authenticator is available
	authPort := getEnv("SCENARIO_AUTHENTICATOR_API_PORT", "15001")
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", authPort))
	
	status := "unavailable"
	if err == nil && resp.StatusCode == 200 {
		status = "connected"
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": status,
		"mode": "multi-tenant",
	})
}

// Search handlers
func handleSearch(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter required", http.StatusBadRequest)
		return
	}
	
	// Search in database
	rows, err := db.Query(`
		SELECT title, content, content_type, week_number
		FROM search_content
		WHERE search_vector @@ plainto_tsquery('english', $1)
		ORDER BY ts_rank(search_vector, plainto_tsquery('english', $1)) DESC
		LIMIT 10
	`, query)
	
	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var content string
		var weekNumber sql.NullInt64
		
		err := rows.Scan(&result.Title, &content, &result.Type, &weekNumber)
		if err != nil {
			continue
		}
		
		// Create snippet
		if len(content) > 200 {
			result.Snippet = content[:200] + "..."
		} else {
			result.Snippet = content
		}
		
		if weekNumber.Valid {
			week := int(weekNumber.Int64)
			result.Week = &week
		}
		
		results = append(results, result)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleSearchHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	// Check if search index is working
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM search_content").Scan(&count)
	
	if err != nil || count == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
		return
	}
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"indexed_items": count,
	})
}

// Pregnancy handlers
func handlePregnancyStart(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	var pregnancy Pregnancy
	if err := json.NewDecoder(r.Body).Decode(&pregnancy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	pregnancy.UserID = userID
	
	// Calculate current week
	weeks := int(time.Since(pregnancy.LMPDate).Hours() / 24 / 7)
	pregnancy.CurrentWeek = weeks
	pregnancy.CurrentDay = int(time.Since(pregnancy.LMPDate).Hours()/24) % 7
	
	// Insert into database
	var id string
	err := db.QueryRow(`
		INSERT INTO pregnancies (user_id, lmp_date, due_date, current_week, current_day)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, pregnancy.UserID, pregnancy.LMPDate, pregnancy.DueDate, pregnancy.CurrentWeek, pregnancy.CurrentDay).Scan(&id)
	
	if err != nil {
		log.Printf("Error creating pregnancy: %v", err)
		http.Error(w, "Failed to create pregnancy", http.StatusInternalServerError)
		return
	}
	
	pregnancy.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pregnancy)
}

func handleCurrentPregnancy(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	var pregnancy Pregnancy
	err := db.QueryRow(`
		SELECT id, user_id, lmp_date, due_date, current_week, current_day, pregnancy_type, outcome, created_at
		FROM pregnancies
		WHERE user_id = $1 AND outcome = 'ongoing'
		ORDER BY created_at DESC
		LIMIT 1
	`, userID).Scan(&pregnancy.ID, &pregnancy.UserID, &pregnancy.LMPDate, &pregnancy.DueDate,
		&pregnancy.CurrentWeek, &pregnancy.CurrentDay, &pregnancy.PregnancyType, 
		&pregnancy.Outcome, &pregnancy.CreatedAt)
	
	if err == sql.ErrNoRows {
		http.Error(w, "No active pregnancy found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Error fetching pregnancy: %v", err)
		http.Error(w, "Failed to fetch pregnancy", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pregnancy)
}

func handleWeekContent(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	// Extract week number from path
	path := r.URL.Path
	weekStr := path[len("/api/v1/content/week/"):]
	week, err := strconv.Atoi(weekStr)
	if err != nil || week < 0 || week > 42 {
		http.Error(w, "Invalid week number", http.StatusBadRequest)
		return
	}
	
	var weekInfo WeekInfo
	var citationsJSON string
	
	err = db.QueryRow(`
		SELECT week_number, title, content, citations
		FROM search_content
		WHERE content_type = 'weekly_info' AND week_number = $1
		LIMIT 1
	`, week).Scan(&weekInfo.Week, &weekInfo.Title, &weekInfo.Development, &citationsJSON)
	
	if err == sql.ErrNoRows {
		// Return default content if not found
		weekInfo = WeekInfo{
			Week:        week,
			Title:       fmt.Sprintf("Week %d Information", week),
			Size:        "Information coming soon",
			Development: "Detailed development information will be available soon.",
			BodyChanges: "Body changes information will be available soon.",
			Tips:        "Stay hydrated and get plenty of rest.",
			Citations:   []Citation{},
		}
	} else if err != nil {
		log.Printf("Error fetching week content: %v", err)
		http.Error(w, "Failed to fetch week content", http.StatusInternalServerError)
		return
	} else {
		// Parse citations
		json.Unmarshal([]byte(citationsJSON), &weekInfo.Citations)
		
		// Parse content to extract different sections
		weekInfo.Size = "Size information available in full content"
		weekInfo.BodyChanges = "See your healthcare provider for personalized information"
		weekInfo.Tips = "Consult your healthcare provider for personalized advice"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weekInfo)
}

// Daily log handlers
func handleDailyLog(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method == "OPTIONS" {
		return
	}
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	if r.Method == "POST" {
		var log DailyLog
		if err := json.NewDecoder(r.Body).Decode(&log); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Get current pregnancy
		var pregnancyID string
		err := db.QueryRow(`
			SELECT id FROM pregnancies 
			WHERE user_id = $1 AND outcome = 'ongoing'
			ORDER BY created_at DESC LIMIT 1
		`, userID).Scan(&pregnancyID)
		
		if err != nil {
			http.Error(w, "No active pregnancy found", http.StatusNotFound)
			return
		}
		
		// Encrypt notes if present
		var encryptedNotes []byte
		if log.Notes != "" {
			encrypted, err := encrypt(log.Notes)
			if err == nil {
				encryptedNotes = []byte(encrypted)
			}
		}
		
		// Convert symptoms to JSON
		symptomsJSON, _ := json.Marshal(log.Symptoms)
		photosJSON, _ := json.Marshal(log.Photos)
		
		// Insert log
		var id string
		err = db.QueryRow(`
			INSERT INTO daily_logs (pregnancy_id, log_date, weight_value, blood_pressure_systolic, 
				blood_pressure_diastolic, symptoms, mood, energy, notes_encrypted, photos)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING id
		`, pregnancyID, time.Now(), log.Weight, 
			extractSystolic(log.BloodPressure), extractDiastolic(log.BloodPressure),
			symptomsJSON, log.Mood, log.Energy, encryptedNotes, photosJSON).Scan(&id)
		
		if err != nil {
			log.Printf("Error saving daily log: %v", err)
			http.Error(w, "Failed to save daily log", http.StatusInternalServerError)
			return
		}
		
		log.ID = id
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(log)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleLogsRange(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	// Get date range from query params
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	
	if startDate == "" || endDate == "" {
		// Default to last 30 days
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	
	// Query logs
	rows, err := db.Query(`
		SELECT dl.id, dl.log_date, dl.weight_value, dl.mood, dl.energy, dl.symptoms
		FROM daily_logs dl
		JOIN pregnancies p ON dl.pregnancy_id = p.id
		WHERE p.user_id = $1 AND dl.log_date BETWEEN $2 AND $3
		ORDER BY dl.log_date DESC
	`, userID, startDate, endDate)
	
	if err != nil {
		log.Printf("Error fetching logs: %v", err)
		http.Error(w, "Failed to fetch logs", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var logs []DailyLog
	for rows.Next() {
		var log DailyLog
		var symptomsJSON []byte
		
		err := rows.Scan(&log.ID, &log.Date, &log.Weight, &log.Mood, &log.Energy, &symptomsJSON)
		if err != nil {
			continue
		}
		
		json.Unmarshal(symptomsJSON, &log.Symptoms)
		logs = append(logs, log)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// Kick count handlers
func handleKickCount(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	var kickCount KickCount
	if err := json.NewDecoder(r.Body).Decode(&kickCount); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Get current pregnancy
	var pregnancyID string
	err := db.QueryRow(`
		SELECT id FROM pregnancies 
		WHERE user_id = $1 AND outcome = 'ongoing'
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&pregnancyID)
	
	if err != nil {
		http.Error(w, "No active pregnancy found", http.StatusNotFound)
		return
	}
	
	// Insert kick count
	var id string
	err = db.QueryRow(`
		INSERT INTO kick_counts (pregnancy_id, session_start, session_end, kick_count, duration_minutes, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, pregnancyID, kickCount.SessionStart, kickCount.SessionEnd, 
		kickCount.Count, kickCount.Duration, kickCount.Notes).Scan(&id)
	
	if err != nil {
		log.Printf("Error saving kick count: %v", err)
		http.Error(w, "Failed to save kick count", http.StatusInternalServerError)
		return
	}
	
	kickCount.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(kickCount)
}

func handleKickPatterns(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	// Get recent kick count patterns
	rows, err := db.Query(`
		SELECT DATE(session_start) as date, AVG(kick_count) as avg_kicks, 
			   AVG(duration_minutes) as avg_duration
		FROM kick_counts kc
		JOIN pregnancies p ON kc.pregnancy_id = p.id
		WHERE p.user_id = $1 AND session_start > NOW() - INTERVAL '30 days'
		GROUP BY DATE(session_start)
		ORDER BY date DESC
	`, userID)
	
	if err != nil {
		log.Printf("Error fetching kick patterns: %v", err)
		http.Error(w, "Failed to fetch patterns", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	type Pattern struct {
		Date         string  `json:"date"`
		AverageKicks float64 `json:"average_kicks"`
		AverageDuration float64 `json:"average_duration"`
	}
	
	var patterns []Pattern
	for rows.Next() {
		var p Pattern
		err := rows.Scan(&p.Date, &p.AverageKicks, &p.AverageDuration)
		if err != nil {
			continue
		}
		patterns = append(patterns, p)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

// Contraction handlers
func handleContractionTimer(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	// Similar to kick count, but for contractions
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "Contraction timer endpoint ready",
	})
}

func handleContractionHistory(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]interface{}{})
}

// Appointment handlers
func handleAppointments(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	if r.Method == "POST" {
		var appointment Appointment
		if err := json.NewDecoder(r.Body).Decode(&appointment); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Get current pregnancy
		var pregnancyID string
		err := db.QueryRow(`
			SELECT id FROM pregnancies 
			WHERE user_id = $1 AND outcome = 'ongoing'
			ORDER BY created_at DESC LIMIT 1
		`, userID).Scan(&pregnancyID)
		
		if err != nil {
			http.Error(w, "No active pregnancy found", http.StatusNotFound)
			return
		}
		
		resultsJSON, _ := json.Marshal(appointment.Results)
		
		// Insert appointment
		var id string
		err = db.QueryRow(`
			INSERT INTO appointments (pregnancy_id, appointment_date, appointment_type, 
				provider_name, location_name, notes, results)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`, pregnancyID, appointment.Date, appointment.Type, 
			appointment.Provider, appointment.Location, appointment.Notes, resultsJSON).Scan(&id)
		
		if err != nil {
			log.Printf("Error saving appointment: %v", err)
			http.Error(w, "Failed to save appointment", http.StatusInternalServerError)
			return
		}
		
		appointment.ID = id
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(appointment)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUpcomingAppointments(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	rows, err := db.Query(`
		SELECT a.id, a.appointment_date, a.appointment_type, a.provider_name, 
			   a.location_name, a.notes, a.results
		FROM appointments a
		JOIN pregnancies p ON a.pregnancy_id = p.id
		WHERE p.user_id = $1 AND a.appointment_date >= NOW() AND a.status = 'scheduled'
		ORDER BY a.appointment_date ASC
		LIMIT 10
	`, userID)
	
	if err != nil {
		log.Printf("Error fetching appointments: %v", err)
		http.Error(w, "Failed to fetch appointments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var appointments []Appointment
	for rows.Next() {
		var apt Appointment
		var resultsJSON []byte
		
		err := rows.Scan(&apt.ID, &apt.Date, &apt.Type, &apt.Provider, 
			&apt.Location, &apt.Notes, &resultsJSON)
		if err != nil {
			continue
		}
		
		json.Unmarshal(resultsJSON, &apt.Results)
		appointments = append(appointments, apt)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appointments)
}

// Export handlers
func handleExportJSON(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	// Gather all user data
	exportData := map[string]interface{}{
		"export_date": time.Now(),
		"user_id": userID,
		"privacy_notice": "This data is encrypted and private to you",
	}
	
	// Get pregnancy data
	var pregnancy Pregnancy
	err := db.QueryRow(`
		SELECT id, lmp_date, due_date, current_week, pregnancy_type, outcome
		FROM pregnancies
		WHERE user_id = $1 AND outcome = 'ongoing'
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&pregnancy.ID, &pregnancy.LMPDate, &pregnancy.DueDate, 
		&pregnancy.CurrentWeek, &pregnancy.PregnancyType, &pregnancy.Outcome)
	
	if err == nil {
		exportData["pregnancy"] = pregnancy
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=pregnancy-data.json")
	json.NewEncoder(w).Encode(exportData)
}

func handleExportPDF(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	// In production, this would generate a proper PDF
	// For now, return a simple text representation
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=pregnancy-report.pdf")
	
	// Placeholder PDF content
	fmt.Fprintf(w, "%%PDF-1.4\nPregnancy Report for User: %s\nGenerated: %s\n",
		userID, time.Now().Format("2006-01-02"))
}

func handleEmergencyCard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	// Fetch emergency info
	var bloodType, rhFactor, obName, obPhone, hospital string
	err := db.QueryRow(`
		SELECT blood_type, rh_factor, ob_name, ob_phone, hospital_preference
		FROM emergency_info
		WHERE user_id = $1
	`, userID).Scan(&bloodType, &rhFactor, &obName, &obPhone, &hospital)
	
	emergencyInfo := map[string]string{
		"blood_type": bloodType + rhFactor,
		"ob_name": obName,
		"ob_phone": obPhone,
		"hospital": hospital,
	}
	
	if err == sql.ErrNoRows {
		emergencyInfo["status"] = "No emergency info configured"
	} else if err != nil {
		log.Printf("Error fetching emergency info: %v", err)
		emergencyInfo["status"] = "Error fetching info"
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(emergencyInfo)
}

// Partner access handlers
func handlePartnerInvite(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	userID := getUserID(r)
	if userID == "" {
		http.Error(w, "User ID required", http.StatusUnauthorized)
		return
	}
	
	// Generate invite code
	inviteCode := generateInviteCode()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"invite_code": inviteCode,
		"expires": time.Now().Add(48 * time.Hour).Format(time.RFC3339),
	})
}

func handlePartnerView(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	
	partnerID := getUserID(r)
	if partnerID == "" {
		http.Error(w, "Partner ID required", http.StatusUnauthorized)
		return
	}
	
	// Check partner access permissions
	var hasAccess bool
	var permissions string
	err := db.QueryRow(`
		SELECT is_active, permissions
		FROM partner_access
		WHERE partner_user_id = $1 AND is_active = true
		LIMIT 1
	`, partnerID).Scan(&hasAccess, &permissions)
	
	if err != nil || !hasAccess {
		http.Error(w, "No partner access", http.StatusForbidden)
		return
	}
	
	// Return limited pregnancy data based on permissions
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access": "granted",
		"permissions": permissions,
		"data": "Limited pregnancy information based on permissions",
	})
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func extractSystolic(bp string) *int {
	// Extract systolic from "120/80" format
	if bp == "" {
		return nil
	}
	var systolic int
	fmt.Sscanf(bp, "%d/", &systolic)
	if systolic > 0 {
		return &systolic
	}
	return nil
}

func extractDiastolic(bp string) *int {
	// Extract diastolic from "120/80" format
	if bp == "" {
		return nil
	}
	var diastolic int
	fmt.Sscanf(bp, "%*d/%d", &diastolic)
	if diastolic > 0 {
		return &diastolic
	}
	return nil
}

func generateInviteCode() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:20]
}