package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Travel struct {
	ID           int64     `json:"id"`
	UserID       string    `json:"user_id"`
	Location     string    `json:"location"`
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	Date         string    `json:"date"`
	Type         string    `json:"type"`
	Notes        string    `json:"notes"`
	Country      string    `json:"country"`
	City         string    `json:"city"`
	Continent    string    `json:"continent"`
	DurationDays int       `json:"duration_days"`
	Rating       int       `json:"rating"`
	Photos       []string  `json:"photos"`
	Tags         []string  `json:"tags"`
	CreatedAt    time.Time `json:"created_at"`
}

type Stats struct {
	TotalCountries       int     `json:"total_countries"`
	TotalCities          int     `json:"total_cities"`
	TotalContinents      int     `json:"total_continents"`
	TotalDistanceKm      float64 `json:"total_distance_km"`
	TotalDaysTraveled    int     `json:"total_days_traveled"`
	WorldCoveragePercent float64 `json:"world_coverage_percent"`
}

type Achievement struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	UnlockedAt  time.Time `json:"unlocked_at"`
}

type BucketListItem struct {
	ID             int      `json:"id"`
	Location       string   `json:"location"`
	Country        string   `json:"country"`
	City           string   `json:"city"`
	Priority       int      `json:"priority"`
	Notes          string   `json:"notes"`
	EstimatedDate  string   `json:"estimated_date"`
	BudgetEstimate float64  `json:"budget_estimate"`
	Tags           []string `json:"tags"`
	Completed      bool     `json:"completed"`
}

var db *sql.DB

// Initialize database connection with exponential backoff
func initDB() error {
	// ALL database configuration MUST come from environment - no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		return fmt.Errorf("❌ Missing required database configuration. Please set: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📊 Connecting to: %s:%s/%s as user %s", dbHost, dbPort, dbName, dbUser)

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

		// Add random jitter to prevent thundering herd
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.25)
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v", actualDelay)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		return fmt.Errorf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")
	return nil
}

// Removed getEnv function - no defaults allowed

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "travel-map-filler",
	})
}

func travelsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get query parameters for filtering
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default_user"
	}

	year := r.URL.Query().Get("year")
	travelType := r.URL.Query().Get("type")

	// Build query with filters
	query := `SELECT id, user_id, location, lat, lng, date, type, notes, 
			  country, city, continent, duration_days, rating, photos, tags, created_at
			  FROM travels WHERE user_id = $1`
	args := []interface{}{userID}
	argCount := 1

	if year != "" {
		argCount++
		query += fmt.Sprintf(" AND EXTRACT(year FROM date) = $%d", argCount)
		args = append(args, year)
	}

	if travelType != "" {
		argCount++
		query += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, travelType)
	}

	query += " ORDER BY date DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var travels []Travel
	for rows.Next() {
		var t Travel
		var photosJSON, tagsJSON []byte

		err := rows.Scan(&t.ID, &t.UserID, &t.Location, &t.Lat, &t.Lng, &t.Date,
			&t.Type, &t.Notes, &t.Country, &t.City, &t.Continent,
			&t.DurationDays, &t.Rating, &photosJSON, &tagsJSON, &t.CreatedAt)
		if err != nil {
			log.Printf("Error scanning travel row: %v", err)
			continue
		}

		// Parse JSON fields
		if len(photosJSON) > 0 {
			json.Unmarshal(photosJSON, &t.Photos)
		}
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &t.Tags)
		}

		travels = append(travels, t)
	}

	if travels == nil {
		travels = []Travel{}
	}

	json.NewEncoder(w).Encode(travels)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default_user"
	}

	// Get stats from database
	query := `SELECT total_countries, total_cities, total_continents, 
			  total_distance_km, total_days_traveled, world_coverage_percent
			  FROM travel_stats WHERE user_id = $1`

	var stats Stats
	err := db.QueryRow(query, userID).Scan(&stats.TotalCountries, &stats.TotalCities,
		&stats.TotalContinents, &stats.TotalDistanceKm, &stats.TotalDaysTraveled,
		&stats.WorldCoveragePercent)

	if err == sql.ErrNoRows {
		// No stats found, calculate from travels table
		countQuery := `SELECT COUNT(DISTINCT country) as countries,
					   COUNT(DISTINCT city) as cities,
					   COUNT(DISTINCT continent) as continents,
					   COALESCE(SUM(duration_days), 0) as total_days
					   FROM travels WHERE user_id = $1`

		err = db.QueryRow(countQuery, userID).Scan(&stats.TotalCountries,
			&stats.TotalCities, &stats.TotalContinents, &stats.TotalDaysTraveled)
		if err != nil {
			log.Printf("Error calculating stats: %v", err)
		}

		// Calculate world coverage (simplified: countries visited / 195 total countries)
		stats.WorldCoveragePercent = float64(stats.TotalCountries) / 195.0 * 100.0

		// For now, set distance to 0 - this would require geospatial calculations
		stats.TotalDistanceKm = 0.0
	} else if err != nil {
		log.Printf("Error fetching stats: %v", err)
		http.Error(w, "Failed to fetch statistics", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(stats)
}

func achievementsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default_user"
	}

	// Get achievements from database
	query := `SELECT achievement_type, achievement_name, description, icon, unlocked_at
			  FROM achievements 
			  WHERE user_id = $1 
			  ORDER BY unlocked_at DESC`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error fetching achievements: %v", err)
		http.Error(w, "Failed to fetch achievements", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var achievements []Achievement
	for rows.Next() {
		var a Achievement
		err := rows.Scan(&a.Type, &a.Name, &a.Description, &a.Icon, &a.UnlockedAt)
		if err != nil {
			log.Printf("Error scanning achievement row: %v", err)
			continue
		}
		achievements = append(achievements, a)
	}

	if achievements == nil {
		achievements = []Achievement{}
	}

	json.NewEncoder(w).Encode(achievements)
}

func bucketListHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default_user"
	}

	// Get bucket list from database
	query := `SELECT id, location, country, city, priority, notes, 
			  estimated_date, budget_estimate, tags, completed
			  FROM bucket_list 
			  WHERE user_id = $1 
			  ORDER BY priority DESC, created_at ASC`

	rows, err := db.Query(query, userID)
	if err != nil {
		log.Printf("Error fetching bucket list: %v", err)
		http.Error(w, "Failed to fetch bucket list", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var bucketList []BucketListItem
	for rows.Next() {
		var item BucketListItem
		var estimatedDate sql.NullString
		var tagsJSON []byte

		err := rows.Scan(&item.ID, &item.Location, &item.Country, &item.City,
			&item.Priority, &item.Notes, &estimatedDate, &item.BudgetEstimate,
			&tagsJSON, &item.Completed)
		if err != nil {
			log.Printf("Error scanning bucket list row: %v", err)
			continue
		}

		if estimatedDate.Valid {
			item.EstimatedDate = estimatedDate.String
		}

		// Parse tags JSON
		if len(tagsJSON) > 0 {
			json.Unmarshal(tagsJSON, &item.Tags)
		}

		bucketList = append(bucketList, item)
	}

	if bucketList == nil {
		bucketList = []BucketListItem{}
	}

	json.NewEncoder(w).Encode(bucketList)
}

func addTravelHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var travel Travel
	if err := json.NewDecoder(r.Body).Decode(&travel); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set default user ID if not provided
	if travel.UserID == "" {
		travel.UserID = "default_user"
	}

	// Set creation time
	travel.CreatedAt = time.Now()

	// Generate ID (use timestamp for simplicity - in production use proper ID generation)
	travel.ID = time.Now().UnixNano()

	// Basic location parsing to extract city and country
	locationParts := strings.Split(travel.Location, ",")
	if len(locationParts) >= 2 {
		travel.City = strings.TrimSpace(locationParts[0])
		travel.Country = strings.TrimSpace(locationParts[len(locationParts)-1])
	}

	// Set default coordinates if not provided (should be geocoded in production)
	if travel.Lat == 0 && travel.Lng == 0 {
		// Default to approximate world center coordinates
		travel.Lat = 40.7128
		travel.Lng = -74.0060
	}

	// Convert slices to JSON for database storage
	photosJSON, _ := json.Marshal(travel.Photos)
	tagsJSON, _ := json.Marshal(travel.Tags)

	// Insert into database
	query := `INSERT INTO travels (id, user_id, location, lat, lng, date, type, notes, 
			  country, city, continent, duration_days, rating, photos, tags, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`

	_, err := db.Exec(query, travel.ID, travel.UserID, travel.Location, travel.Lat, travel.Lng,
		travel.Date, travel.Type, travel.Notes, travel.Country, travel.City, travel.Continent,
		travel.DurationDays, travel.Rating, photosJSON, tagsJSON, travel.CreatedAt)

	if err != nil {
		log.Printf("Error inserting travel: %v", err)
		http.Error(w, "Failed to save travel", http.StatusInternalServerError)
		return
	}

	// Check and unlock achievements
	checkAchievements(travel.UserID)

	// Return success response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Travel added successfully",
		"id":      travel.ID,
		"travel":  travel,
	})
}

// Check and unlock achievements for user
func checkAchievements(userID string) {
	// Get travel counts for achievement checking
	var countryCount, cityCount int
	countQuery := `SELECT COUNT(DISTINCT country), COUNT(DISTINCT city) FROM travels WHERE user_id = $1`
	db.QueryRow(countQuery, userID).Scan(&countryCount, &cityCount)

	achievements := []map[string]interface{}{}

	// First trip achievement
	if countryCount >= 1 {
		achievements = append(achievements, map[string]interface{}{
			"type": "first_trip", "name": "First Steps", "description": "Complete your first trip", "icon": "👣",
		})
	}

	// Explorer achievements
	if countryCount >= 5 {
		achievements = append(achievements, map[string]interface{}{
			"type": "five_countries", "name": "Explorer", "description": "Visit 5 different countries", "icon": "🧭",
		})
	}

	if countryCount >= 10 {
		achievements = append(achievements, map[string]interface{}{
			"type": "ten_countries", "name": "Adventurer", "description": "Visit 10 different countries", "icon": "🎒",
		})
	}

	// Insert achievements that don't exist yet
	for _, ach := range achievements {
		insertQuery := `INSERT INTO achievements (user_id, achievement_type, achievement_name, description, icon)
						VALUES ($1, $2, $3, $4, $5)
						ON CONFLICT (user_id, achievement_type) DO NOTHING`

		db.Exec(insertQuery, userID, ach["type"], ach["name"], ach["description"], ach["icon"])
	}
}

func searchTravelsHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse query parameter or request body
	query := r.URL.Query().Get("q")
	limitParam := r.URL.Query().Get("limit")
	limit := 10
	if limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	// If no query in URL, try request body
	if query == "" && r.Method == "POST" {
		var searchReq struct {
			Query  string `json:"query"`
			Limit  int    `json:"limit"`
			UserID string `json:"user_id"`
		}
		json.NewDecoder(r.Body).Decode(&searchReq)
		query = searchReq.Query
		if searchReq.Limit > 0 {
			limit = searchReq.Limit
		}
	}

	if query == "" {
		http.Error(w, "Query parameter required", http.StatusBadRequest)
		return
	}

	results, err := searchTravels(query, limit)
	if err != nil {
		log.Printf("Travel search failed: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"results": results,
		"count":   len(results),
	}
	json.NewEncoder(w).Encode(response)
}

// searchTravels performs a simple full-text match across location fields.
func searchTravels(query string, limit int) ([]Travel, error) {
	if db == nil {
		return []Travel{}, nil
	}
	pattern := "%" + query + "%"
	rows, err := db.Query(`
		SELECT id, user_id, location, lat, lng, date, type, notes,
		       country, city, continent, duration_days, rating, photos, tags, created_at
		FROM travels
		WHERE location ILIKE $1 OR country ILIKE $1 OR city ILIKE $1 OR notes ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2
	`, pattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Travel
	for rows.Next() {
		var t Travel
		var photosJSON, tagsJSON []byte
		if err := rows.Scan(&t.ID, &t.UserID, &t.Location, &t.Lat, &t.Lng, &t.Date, &t.Type, &t.Notes, &t.Country, &t.City, &t.Continent, &t.DurationDays, &t.Rating, &photosJSON, &tagsJSON, &t.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(photosJSON, &t.Photos)
		_ = json.Unmarshal(tagsJSON, &t.Tags)
		results = append(results, t)
	}

	return results, nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start travel-map-filler

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	// Initialize database connection
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// API routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/travels", travelsHandler)
	http.HandleFunc("/api/stats", statsHandler)
	http.HandleFunc("/api/achievements", achievementsHandler)
	http.HandleFunc("/api/bucket-list", bucketListHandler)
	http.HandleFunc("/api/add-travel", addTravelHandler)
	http.HandleFunc("/api/travels/search", searchTravelsHandler)

	fmt.Printf("🗺️ Travel Map API running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
