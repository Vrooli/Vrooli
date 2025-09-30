package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Region struct {
	ID               int     `json:"id"`
	Name             string  `json:"name"`
	State            string  `json:"state"`
	Country          string  `json:"country"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	ElevationMeters  *int    `json:"elevation_meters,omitempty"`
	TypicalPeakWeek  *int    `json:"typical_peak_week,omitempty"`
}

type FoliageData struct {
	RegionID         int     `json:"region_id"`
	ObservationDate  string  `json:"observation_date"`
	FoliagePercent   int     `json:"foliage_percentage"`
	ColorIntensity   int     `json:"color_intensity"`
	PeakStatus       string  `json:"peak_status"`
	PredictedPeak    string  `json:"predicted_peak,omitempty"`
	ConfidenceScore  float64 `json:"confidence_score,omitempty"`
}

type UserReport struct {
	ID            int    `json:"id"`
	RegionID      int    `json:"region_id"`
	ReportDate    string `json:"report_date"`
	FoliageStatus string `json:"foliage_status"`
	Description   string `json:"description"`
	PhotoURL      string `json:"photo_url,omitempty"`
}

func initDB() error {
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5433")
	dbUser := getEnv("POSTGRES_USER", "vrooli")
	dbPass := getEnv("POSTGRES_PASSWORD", "vrooli")
	dbName := getEnv("POSTGRES_DB", "vrooli")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection established")
	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check database connection
	dbStatus := "healthy"
	if db != nil {
		if err := db.Ping(); err != nil {
			dbStatus = "unhealthy"
		}
	} else {
		dbStatus = "not_connected"
	}

	response := Response{
		Status:  "healthy",
		Message: "Fall Foliage Explorer API is running",
		Data: map[string]interface{}{
			"database": dbStatus,
			"version":  "1.0.0",
			"uptime":   time.Now().Unix(),
		},
	}
	json.NewEncoder(w).Encode(response)
}

func regionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if db == nil {
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Database not connected",
		})
		return
	}

	rows, err := db.Query(`
		SELECT id, name, state, country, latitude, longitude, elevation_meters, typical_peak_week
		FROM regions
		ORDER BY name
	`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Failed to query regions: %v", err),
		})
		return
	}
	defer rows.Close()

	var regions []Region
	for rows.Next() {
		var region Region
		err := rows.Scan(
			&region.ID,
			&region.Name,
			&region.State,
			&region.Country,
			&region.Latitude,
			&region.Longitude,
			&region.ElevationMeters,
			&region.TypicalPeakWeek,
		)
		if err != nil {
			log.Printf("Error scanning region: %v", err)
			continue
		}
		regions = append(regions, region)
	}

	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   regions,
	})
}

func foliageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	regionID := r.URL.Query().Get("region_id")
	if regionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "region_id parameter is required",
		})
		return
	}

	rid, err := strconv.Atoi(regionID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Invalid region_id",
		})
		return
	}

	if db == nil {
		// Return mock data if database not connected
		foliageData := FoliageData{
			RegionID:        rid,
			ObservationDate: time.Now().Format("2006-01-02"),
			FoliagePercent:  75,
			ColorIntensity:  7,
			PeakStatus:      "near_peak",
			PredictedPeak:   "2025-10-15",
			ConfidenceScore: 0.85,
		}
		json.NewEncoder(w).Encode(Response{
			Status: "success",
			Data:   foliageData,
		})
		return
	}

	// Get latest observation
	var foliage FoliageData
	err = db.QueryRow(`
		SELECT
			region_id,
			observation_date,
			COALESCE(foliage_percentage, 0),
			COALESCE(color_intensity, 0),
			COALESCE(peak_status, 'unknown')
		FROM foliage_observations
		WHERE region_id = $1
		ORDER BY observation_date DESC
		LIMIT 1
	`, rid).Scan(
		&foliage.RegionID,
		&foliage.ObservationDate,
		&foliage.FoliagePercent,
		&foliage.ColorIntensity,
		&foliage.PeakStatus,
	)

	if err == sql.ErrNoRows {
		// No observations, return default
		foliage = FoliageData{
			RegionID:        rid,
			ObservationDate: time.Now().Format("2006-01-02"),
			FoliagePercent:  0,
			ColorIntensity:  0,
			PeakStatus:      "not_started",
		}
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Failed to query foliage data: %v", err),
		})
		return
	}

	// Get latest prediction if available
	var predictedPeak sql.NullString
	var confidence sql.NullFloat64
	err = db.QueryRow(`
		SELECT predicted_peak_date, confidence_score
		FROM foliage_predictions
		WHERE region_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, rid).Scan(&predictedPeak, &confidence)

	if err == nil {
		if predictedPeak.Valid {
			foliage.PredictedPeak = predictedPeak.String
		}
		if confidence.Valid {
			foliage.ConfidenceScore = confidence.Float64
		}
	}

	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   foliage,
	})
}

func predictHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Method not allowed",
		})
		return
	}

	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Invalid request body",
		})
		return
	}

	regionID, ok := request["region_id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "region_id is required",
		})
		return
	}

	// TODO: Implement actual prediction logic using Ollama
	// For now, return mock prediction
	json.NewEncoder(w).Encode(Response{
		Status:  "success",
		Message: "Prediction triggered",
		Data: map[string]interface{}{
			"region_id":      regionID,
			"predicted_peak": "2025-10-15",
			"confidence":     0.85,
		},
	})
}

func weatherHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	regionID := r.URL.Query().Get("region_id")
	date := r.URL.Query().Get("date")

	if regionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "region_id parameter is required",
		})
		return
	}

	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	rid, err := strconv.Atoi(regionID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Invalid region_id",
		})
		return
	}

	if db == nil {
		// Return mock data
		json.NewEncoder(w).Encode(Response{
			Status: "success",
			Data: map[string]interface{}{
				"region_id":         rid,
				"date":              date,
				"temperature_high":  18.5,
				"temperature_low":   8.2,
				"precipitation_mm":  2.5,
				"humidity_percent":  65,
			},
		})
		return
	}

	var weather struct {
		TempHigh      sql.NullFloat64
		TempLow       sql.NullFloat64
		Precipitation sql.NullFloat64
		Humidity      sql.NullInt32
	}

	err = db.QueryRow(`
		SELECT temperature_high_c, temperature_low_c, precipitation_mm, humidity_percent
		FROM weather_data
		WHERE region_id = $1 AND date = $2
	`, rid, date).Scan(&weather.TempHigh, &weather.TempLow, &weather.Precipitation, &weather.Humidity)

	if err == sql.ErrNoRows {
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: "No weather data available for this date",
			Data:    nil,
		})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Failed to query weather data: %v", err),
		})
		return
	}

	result := map[string]interface{}{
		"region_id": rid,
		"date":      date,
	}

	if weather.TempHigh.Valid {
		result["temperature_high"] = weather.TempHigh.Float64
	}
	if weather.TempLow.Valid {
		result["temperature_low"] = weather.TempLow.Float64
	}
	if weather.Precipitation.Valid {
		result["precipitation_mm"] = weather.Precipitation.Float64
	}
	if weather.Humidity.Valid {
		result["humidity_percent"] = weather.Humidity.Int32
	}

	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   result,
	})
}

func reportsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getReports(w, r)
	case http.MethodPost:
		submitReport(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Method not allowed",
		})
	}
}

func getReports(w http.ResponseWriter, r *http.Request) {
	regionID := r.URL.Query().Get("region_id")
	if regionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "region_id parameter is required",
		})
		return
	}

	rid, err := strconv.Atoi(regionID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Invalid region_id",
		})
		return
	}

	if db == nil {
		// Return mock data
		json.NewEncoder(w).Encode(Response{
			Status: "success",
			Data:   []UserReport{},
		})
		return
	}

	rows, err := db.Query(`
		SELECT id, region_id, report_date, foliage_status, description, photo_url
		FROM user_reports
		WHERE region_id = $1
		ORDER BY report_date DESC
		LIMIT 10
	`, rid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Failed to query reports: %v", err),
		})
		return
	}
	defer rows.Close()

	var reports []UserReport
	for rows.Next() {
		var report UserReport
		var photoURL sql.NullString
		err := rows.Scan(
			&report.ID,
			&report.RegionID,
			&report.ReportDate,
			&report.FoliageStatus,
			&report.Description,
			&photoURL,
		)
		if err != nil {
			log.Printf("Error scanning report: %v", err)
			continue
		}
		if photoURL.Valid {
			report.PhotoURL = photoURL.String
		}
		reports = append(reports, report)
	}

	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   reports,
	})
}

func submitReport(w http.ResponseWriter, r *http.Request) {
	var report UserReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Invalid request body",
		})
		return
	}

	if report.RegionID == 0 || report.FoliageStatus == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "region_id and foliage_status are required",
		})
		return
	}

	if db == nil {
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: "Report submitted (mock)",
			Data:    report,
		})
		return
	}

	var id int
	err := db.QueryRow(`
		INSERT INTO user_reports (region_id, report_date, foliage_status, description, photo_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, report.RegionID, time.Now().Format("2006-01-02"), report.FoliageStatus, report.Description, report.PhotoURL).Scan(&id)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Failed to insert report: %v", err),
		})
		return
	}

	report.ID = id
	json.NewEncoder(w).Encode(Response{
		Status:  "success",
		Message: "Report submitted successfully",
		Data:    report,
	})
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start fall-foliage-explorer

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	// Initialize database connection
	if err := initDB(); err != nil {
		log.Printf("Warning: Database initialization failed: %v", err)
		log.Println("Running in mock data mode")
	}

	port := getEnv("API_PORT", getEnv("PORT", "8080"))

	// Register routes
	http.HandleFunc("/health", enableCORS(healthHandler))
	http.HandleFunc("/api/regions", enableCORS(regionsHandler))
	http.HandleFunc("/api/foliage", enableCORS(foliageHandler))
	http.HandleFunc("/api/predict", enableCORS(predictHandler))
	http.HandleFunc("/api/weather", enableCORS(weatherHandler))
	http.HandleFunc("/api/reports", enableCORS(reportsHandler))

	log.Printf("Fall Foliage Explorer API starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}