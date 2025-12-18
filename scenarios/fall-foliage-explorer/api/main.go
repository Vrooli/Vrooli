package main

import (
	"github.com/vrooli/api-core/preflight"
	"bytes"
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
	ID              int     `json:"id"`
	Name            string  `json:"name"`
	State           string  `json:"state"`
	Country         string  `json:"country"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	ElevationMeters *int    `json:"elevation_meters,omitempty"`
	TypicalPeakWeek *int    `json:"typical_peak_week,omitempty"`
	DataSource      string  `json:"data_source,omitempty"`
}

type ResponseMeta struct {
	Source            string `json:"source"`
	SourceDescription string `json:"source_description,omitempty"`
	RetrievedAt       string `json:"retrieved_at"`
	RowCount          int    `json:"row_count"`
	UsingFallback     bool   `json:"using_fallback"`
}

type RegionsPayload struct {
	Regions []Region     `json:"regions"`
	Meta    ResponseMeta `json:"meta"`
}

type FoliageData struct {
	RegionID          int     `json:"region_id"`
	ObservationDate   string  `json:"observation_date"`
	FoliagePercent    int     `json:"foliage_percentage"`
	ColorIntensity    int     `json:"color_intensity"`
	PeakStatus        string  `json:"peak_status"`
	PredictedPeak     string  `json:"predicted_peak,omitempty"`
	ConfidenceScore   float64 `json:"confidence_score,omitempty"`
	DataSource        string  `json:"data_source,omitempty"`
	SourceDescription string  `json:"source_description,omitempty"`
}

type UserReport struct {
	ID            int    `json:"id"`
	RegionID      int    `json:"region_id"`
	ReportDate    string `json:"report_date"`
	FoliageStatus string `json:"foliage_status"`
	Description   string `json:"description"`
	PhotoURL      string `json:"photo_url,omitempty"`
}

type TripPlan struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Regions   []int  `json:"regions"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
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
		regions := sampleRegionsDataset()
		payload := RegionsPayload{
			Regions: regions,
			Meta:    buildRegionsMeta(len(regions), true, "sample_dataset"),
		}
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: "Database not connected. Serving packaged sample dataset.",
			Data:    payload,
		})
		return
	}

	rows, err := db.Query(`
		SELECT id, name, state, country, latitude, longitude, elevation_meters, typical_peak_week
		FROM regions
		ORDER BY name
	`)
	if err != nil {
		log.Printf("Failed to query regions: %v", err)
		regions := sampleRegionsDataset()
		payload := RegionsPayload{
			Regions: regions,
			Meta:    buildRegionsMeta(len(regions), true, "sample_dataset"),
		}
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: fmt.Sprintf("Returning sample dataset because database query failed: %v", err),
			Data:    payload,
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
		region.DataSource = "postgres.regions"
		regions = append(regions, region)
	}

	payload := RegionsPayload{
		Regions: regions,
		Meta:    buildRegionsMeta(len(regions), false, "postgres.regions"),
	}

	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   payload,
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
			RegionID:          rid,
			ObservationDate:   time.Now().Format("2006-01-02"),
			FoliagePercent:    75,
			ColorIntensity:    7,
			PeakStatus:        "near_peak",
			PredictedPeak:     "2025-10-15",
			ConfidenceScore:   0.85,
			DataSource:        "sample_dataset",
			SourceDescription: "Static sample foliage reading used when the live database is unavailable.",
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
			RegionID:          rid,
			ObservationDate:   time.Now().Format("2006-01-02"),
			FoliagePercent:    0,
			ColorIntensity:    0,
			PeakStatus:        "not_started",
			DataSource:        "postgres.foliage_observations",
			SourceDescription: "No observations recorded yet; providing synthesized baseline values.",
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

	if foliage.DataSource == "" {
		foliage.DataSource = "postgres.foliage_observations"
		foliage.SourceDescription = "Live foliage observation retrieved from the database."
	}

	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   foliage,
	})
}

func sampleRegionsDataset() []Region {
	return []Region{
		{ID: 1, Name: "White Mountains", State: "New Hampshire", Country: "USA", Latitude: 44.2700, Longitude: -71.3034, ElevationMeters: intPtr(1917), TypicalPeakWeek: intPtr(40), DataSource: "sample_dataset"},
		{ID: 2, Name: "Green Mountains", State: "Vermont", Country: "USA", Latitude: 43.9207, Longitude: -72.8986, ElevationMeters: intPtr(1340), TypicalPeakWeek: intPtr(40), DataSource: "sample_dataset"},
		{ID: 3, Name: "Adirondacks", State: "New York", Country: "USA", Latitude: 44.1127, Longitude: -74.0524, ElevationMeters: intPtr(1629), TypicalPeakWeek: intPtr(40), DataSource: "sample_dataset"},
		{ID: 4, Name: "Great Smoky Mountains", State: "Tennessee", Country: "USA", Latitude: 35.6532, Longitude: -83.5070, ElevationMeters: intPtr(2025), TypicalPeakWeek: intPtr(42), DataSource: "sample_dataset"},
		{ID: 5, Name: "Blue Ridge Parkway", State: "Virginia", Country: "USA", Latitude: 37.5615, Longitude: -79.3553, ElevationMeters: intPtr(1200), TypicalPeakWeek: intPtr(42), DataSource: "sample_dataset"},
		{ID: 6, Name: "Berkshires", State: "Massachusetts", Country: "USA", Latitude: 42.3604, Longitude: -73.2290, ElevationMeters: intPtr(1064), TypicalPeakWeek: intPtr(41), DataSource: "sample_dataset"},
		{ID: 7, Name: "Pocono Mountains", State: "Pennsylvania", Country: "USA", Latitude: 41.1247, Longitude: -75.3821, ElevationMeters: intPtr(748), TypicalPeakWeek: intPtr(41), DataSource: "sample_dataset"},
		{ID: 8, Name: "Shenandoah Valley", State: "Virginia", Country: "USA", Latitude: 38.4833, Longitude: -78.8500, ElevationMeters: intPtr(1234), TypicalPeakWeek: intPtr(42), DataSource: "sample_dataset"},
		{ID: 9, Name: "Finger Lakes", State: "New York", Country: "USA", Latitude: 42.6500, Longitude: -76.8833, ElevationMeters: intPtr(382), TypicalPeakWeek: intPtr(41), DataSource: "sample_dataset"},
		{ID: 10, Name: "Upper Peninsula", State: "Michigan", Country: "USA", Latitude: 46.5000, Longitude: -87.5000, ElevationMeters: intPtr(603), TypicalPeakWeek: intPtr(39), DataSource: "sample_dataset"},
	}
}

func buildRegionsMeta(count int, fallback bool, source string) ResponseMeta {
	description := "Live data from the Fall Foliage Explorer regions table (PostgreSQL)."
	if fallback {
		description = "Packaged sample dataset used when the live database is unavailable."
	}

	return ResponseMeta{
		Source:            source,
		SourceDescription: description,
		RetrievedAt:       time.Now().UTC().Format(time.RFC3339),
		RowCount:          count,
		UsingFallback:     fallback,
	}
}

func intPtr(value int) *int {
	return &value
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

	regionIDFloat, ok := request["region_id"].(float64)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "region_id is required and must be a number",
		})
		return
	}
	regionID := int(regionIDFloat)

	// Get region data from database
	var region Region
	err := db.QueryRow(`
		SELECT id, name, state, latitude, longitude, elevation_meters, typical_peak_week
		FROM regions WHERE id = $1
	`, regionID).Scan(&region.ID, &region.Name, &region.State, &region.Latitude,
		&region.Longitude, &region.ElevationMeters, &region.TypicalPeakWeek)

	if err == sql.ErrNoRows {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Region not found",
		})
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Database error: %v", err),
		})
		return
	}

	// Call Ollama for prediction
	prediction, confidence, err := generateFoliagePrediction(region)
	if err != nil {
		// Fallback to typical peak week if Ollama fails
		log.Printf("Ollama prediction failed, using fallback: %v", err)
		typicalWeek := 41
		if region.TypicalPeakWeek != nil {
			typicalWeek = *region.TypicalPeakWeek
		}
		fallbackDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, (typicalWeek-1)*7)

		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: "Prediction generated (fallback mode)",
			Data: map[string]interface{}{
				"region_id":      regionID,
				"region_name":    region.Name,
				"predicted_peak": fallbackDate.Format("2006-01-02"),
				"confidence":     0.65,
				"method":         "typical_week_fallback",
			},
		})
		return
	}

	// Store prediction in database
	_, err = db.Exec(`
		INSERT INTO foliage_predictions (region_id, predicted_peak_date, confidence_score, created_at)
		VALUES ($1, $2, $3, $4)
	`, regionID, prediction, confidence, time.Now())

	if err != nil {
		log.Printf("Failed to store prediction: %v", err)
	}

	json.NewEncoder(w).Encode(Response{
		Status:  "success",
		Message: "Prediction generated using AI",
		Data: map[string]interface{}{
			"region_id":      regionID,
			"region_name":    region.Name,
			"predicted_peak": prediction,
			"confidence":     confidence,
			"method":         "ollama_ai",
		},
	})
}

func generateFoliagePrediction(region Region) (string, float64, error) {
	// Build prompt for Ollama
	prompt := fmt.Sprintf(`You are a foliage prediction expert. Based on the following region data, predict when fall foliage will reach peak colors in 2025.

Region: %s, %s
Latitude: %.4f
Longitude: %.4f
Elevation: %dm
Typical Peak Week: %d (week of year)

Current date: October 2, 2025

Consider:
1. Higher latitudes peak earlier
2. Higher elevations peak earlier
3. Northern regions (lat > 45) typically peak late September to early October
4. Mid-Atlantic regions (lat 37-45) typically peak mid to late October
5. Southern Appalachians (lat < 37) typically peak late October to early November

Respond with ONLY a JSON object in this exact format:
{"predicted_date": "YYYY-MM-DD", "confidence": 0.XX}

No other text, just the JSON.`,
		region.Name,
		region.State,
		region.Latitude,
		region.Longitude,
		getIntValue(region.ElevationMeters),
		getIntValue(region.TypicalPeakWeek))

	// Call Ollama API
	ollamaURL := getEnv("OLLAMA_URL", "http://localhost:11434")
	reqBody := map[string]interface{}{
		"model":  "llama3.2:latest",
		"prompt": prompt,
		"stream": false,
		"format": "json",
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(ollamaURL+"/api/generate", "application/json",
		bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", 0, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	// Parse the JSON response from Ollama
	var predictionData struct {
		PredictedDate string  `json:"predicted_date"`
		Confidence    float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(ollamaResp.Response), &predictionData); err != nil {
		return "", 0, fmt.Errorf("failed to parse prediction: %w", err)
	}

	// Validate the date
	_, err = time.Parse("2006-01-02", predictionData.PredictedDate)
	if err != nil {
		return "", 0, fmt.Errorf("invalid date format: %w", err)
	}

	// Ensure confidence is between 0 and 1
	if predictionData.Confidence < 0 {
		predictionData.Confidence = 0
	} else if predictionData.Confidence > 1 {
		predictionData.Confidence = 1
	}

	return predictionData.PredictedDate, predictionData.Confidence, nil
}

func getIntValue(val *int) int {
	if val == nil {
		return 0
	}
	return *val
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
				"region_id":        rid,
				"date":             date,
				"temperature_high": 18.5,
				"temperature_low":  8.2,
				"precipitation_mm": 2.5,
				"humidity_percent": 65,
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

func tripsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		getTrips(w, r)
	case http.MethodPost:
		saveTrip(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Method not allowed",
		})
	}
}

func getTrips(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		json.NewEncoder(w).Encode(Response{
			Status: "success",
			Data:   []TripPlan{},
		})
		return
	}

	rows, err := db.Query(`
		SELECT id, name, start_date, end_date, regions, notes, created_at
		FROM trip_plans
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Failed to query trips: %v", err),
		})
		return
	}
	defer rows.Close()

	var trips []TripPlan
	for rows.Next() {
		var trip TripPlan
		var regionsJSON []byte
		var notes sql.NullString
		err := rows.Scan(
			&trip.ID,
			&trip.Name,
			&trip.StartDate,
			&trip.EndDate,
			&regionsJSON,
			&notes,
			&trip.CreatedAt,
		)
		if err != nil {
			log.Printf("Error scanning trip: %v", err)
			continue
		}

		// Parse regions JSON
		if err := json.Unmarshal(regionsJSON, &trip.Regions); err != nil {
			log.Printf("Error parsing regions JSON: %v", err)
			continue
		}

		if notes.Valid {
			trip.Notes = notes.String
		}

		trips = append(trips, trip)
	}

	json.NewEncoder(w).Encode(Response{
		Status: "success",
		Data:   trips,
	})
}

func saveTrip(w http.ResponseWriter, r *http.Request) {
	var trip TripPlan
	if err := json.NewDecoder(r.Body).Decode(&trip); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Invalid request body",
		})
		return
	}

	if trip.Name == "" || trip.StartDate == "" || trip.EndDate == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "name, start_date, and end_date are required",
		})
		return
	}

	if db == nil {
		json.NewEncoder(w).Encode(Response{
			Status:  "success",
			Message: "Trip saved (mock)",
			Data:    trip,
		})
		return
	}

	// Convert regions to JSON
	regionsJSON, err := json.Marshal(trip.Regions)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  "Failed to encode regions",
		})
		return
	}

	var id int
	err = db.QueryRow(`
		INSERT INTO trip_plans (name, start_date, end_date, regions, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		RETURNING id
	`, trip.Name, trip.StartDate, trip.EndDate, regionsJSON, trip.Notes, time.Now()).Scan(&id)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Status: "error",
			Error:  fmt.Sprintf("Failed to save trip: %v", err),
		})
		return
	}

	trip.ID = id
	trip.CreatedAt = time.Now().Format(time.RFC3339)

	json.NewEncoder(w).Encode(Response{
		Status:  "success",
		Message: "Trip saved successfully",
		Data:    trip,
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "fall-foliage-explorer",
	}) {
		return // Process was re-exec'd after rebuild
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
	http.HandleFunc("/api/trips", enableCORS(tripsHandler))

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
