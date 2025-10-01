package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// TravelTimeCalculator handles travel time calculations between locations
type TravelTimeCalculator struct {
	config *Config
}

// NewTravelTimeCalculator creates a new travel time calculator
func NewTravelTimeCalculator(config *Config) *TravelTimeCalculator {
	return &TravelTimeCalculator{
		config: config,
	}
}

// TravelMode represents the mode of transportation
type TravelMode string

const (
	TravelModeDriving   TravelMode = "driving"
	TravelModeTransit   TravelMode = "transit"
	TravelModeWalking   TravelMode = "walking"
	TravelModeBicycling TravelMode = "bicycling"
)

// TravelTimeRequest represents a request for travel time calculation
type TravelTimeRequest struct {
	Origin        string     `json:"origin"`
	Destination   string     `json:"destination"`
	Mode          TravelMode `json:"mode"`
	DepartureTime *time.Time `json:"departure_time,omitempty"`
}

// TravelTimeResponse contains the calculated travel time
type TravelTimeResponse struct {
	Duration         int        `json:"duration_seconds"`
	Distance         float64    `json:"distance_meters"`
	Mode             TravelMode `json:"mode"`
	DepartureTime    time.Time  `json:"departure_time"`
	ArrivalTime      time.Time  `json:"arrival_time"`
	TrafficCondition string     `json:"traffic_condition,omitempty"`
}

// CalculateTravelTime calculates travel time between two locations
func (t *TravelTimeCalculator) CalculateTravelTime(req TravelTimeRequest) (*TravelTimeResponse, error) {
	// For now, use a simple heuristic calculation based on distance
	// In production, this would integrate with Google Maps API or similar

	// Simple heuristic: estimate based on mode
	distance := t.estimateDistance(req.Origin, req.Destination)
	duration := t.estimateDuration(distance, req.Mode)

	departureTime := time.Now()
	if req.DepartureTime != nil {
		departureTime = *req.DepartureTime
	}

	return &TravelTimeResponse{
		Duration:         duration,
		Distance:         distance,
		Mode:             req.Mode,
		DepartureTime:    departureTime,
		ArrivalTime:      departureTime.Add(time.Duration(duration) * time.Second),
		TrafficCondition: t.estimateTrafficCondition(departureTime),
	}, nil
}

// estimateDistance provides a rough distance estimate (in meters)
func (t *TravelTimeCalculator) estimateDistance(origin, destination string) float64 {
	// Simple heuristic based on location strings
	// In production, would use geocoding and actual distance calculation

	// Check if locations are in same building/area
	if strings.Contains(strings.ToLower(origin), "room") &&
		strings.Contains(strings.ToLower(destination), "room") {
		return 100 // Same building
	}

	// Check if locations mention same street/area
	originLower := strings.ToLower(origin)
	destLower := strings.ToLower(destination)

	if strings.Contains(originLower, "campus") || strings.Contains(destLower, "campus") {
		return 500 // Same campus
	}

	// Default to 5km for different locations
	return 5000
}

// estimateDuration estimates travel duration based on distance and mode
func (t *TravelTimeCalculator) estimateDuration(distance float64, mode TravelMode) int {
	// Speed estimates in meters per second
	var speed float64

	switch mode {
	case TravelModeWalking:
		speed = 1.4 // ~5 km/h
	case TravelModeBicycling:
		speed = 4.2 // ~15 km/h
	case TravelModeTransit:
		speed = 8.3 // ~30 km/h average with stops
	case TravelModeDriving:
		speed = 11.1 // ~40 km/h in city
	default:
		speed = 11.1 // Default to driving
	}

	// Calculate base duration
	duration := distance / speed

	// Add buffer time for preparation and parking
	switch mode {
	case TravelModeDriving:
		duration += 300 // 5 minutes for parking
	case TravelModeTransit:
		duration += 600 // 10 minutes for waiting
	}

	return int(math.Ceil(duration))
}

// estimateTrafficCondition estimates traffic based on time of day
func (t *TravelTimeCalculator) estimateTrafficCondition(departureTime time.Time) string {
	hour := departureTime.Hour()
	weekday := departureTime.Weekday()

	// Skip weekends
	if weekday == time.Saturday || weekday == time.Sunday {
		return "light"
	}

	// Rush hours
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		return "heavy"
	}

	// Mid-day
	if hour >= 11 && hour <= 14 {
		return "moderate"
	}

	return "light"
}

// CalculateBufferTime calculates buffer time needed between events based on locations
func (t *TravelTimeCalculator) CalculateBufferTime(event1, event2 *Event) (int, error) {
	if event1.Location == "" || event2.Location == "" {
		// If no location specified, use default buffer
		return 900, nil // 15 minutes default
	}

	// If same location, minimal buffer
	if event1.Location == event2.Location {
		return 300, nil // 5 minutes
	}

	// Calculate travel time
	travelReq := TravelTimeRequest{
		Origin:        event1.Location,
		Destination:   event2.Location,
		Mode:          TravelModeDriving, // Default to driving
		DepartureTime: &event1.EndTime,
	}

	resp, err := t.CalculateTravelTime(travelReq)
	if err != nil {
		return 900, err // Default to 15 minutes on error
	}

	// Add 10% buffer to travel time
	bufferTime := int(float64(resp.Duration) * 1.1)

	// Minimum 5 minutes, maximum 2 hours
	if bufferTime < 300 {
		bufferTime = 300
	}
	if bufferTime > 7200 {
		bufferTime = 7200
	}

	return bufferTime, nil
}

// SuggestDepartureTime suggests when to leave for an event based on location
func (t *TravelTimeCalculator) SuggestDepartureTime(event *Event, origin string, mode TravelMode) (*time.Time, error) {
	if event.Location == "" {
		return nil, fmt.Errorf("event has no location specified")
	}

	travelReq := TravelTimeRequest{
		Origin:      origin,
		Destination: event.Location,
		Mode:        mode,
	}

	resp, err := t.CalculateTravelTime(travelReq)
	if err != nil {
		return nil, err
	}

	// Calculate departure time with 5-minute buffer
	departureTime := event.StartTime.Add(-time.Duration(resp.Duration+300) * time.Second)

	return &departureTime, nil
}

// HTTP Handlers

// handleCalculateTravelTime handles travel time calculation requests
func handleCalculateTravelTime(w http.ResponseWriter, r *http.Request) {
	var req TravelTimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorHandler.HandleError(w, r, err)
		return
	}

	// Validate request
	if req.Origin == "" || req.Destination == "" {
		http.Error(w, "Origin and destination are required", http.StatusBadRequest)
		return
	}

	if req.Mode == "" {
		req.Mode = TravelModeDriving // Default to driving
	}

	calculator := NewTravelTimeCalculator(&Config{})
	resp, err := calculator.CalculateTravelTime(req)
	if err != nil {
		errorHandler.HandleError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleSuggestDepartureTime handles departure time suggestion requests
func handleSuggestDepartureTime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	// Get event from database
	event, err := getEventByID(eventID)
	if err != nil {
		errorHandler.HandleError(w, r, err)
		return
	}

	// Parse query parameters
	origin := r.URL.Query().Get("origin")
	if origin == "" {
		http.Error(w, "Origin location is required", http.StatusBadRequest)
		return
	}

	mode := TravelMode(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = TravelModeDriving
	}

	calculator := NewTravelTimeCalculator(&Config{})
	departureTime, err := calculator.SuggestDepartureTime(event, origin, mode)
	if err != nil {
		errorHandler.HandleError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"event_id":       eventID,
		"event_title":    event.Title,
		"event_location": event.Location,
		"event_start":    event.StartTime,
		"departure_time": departureTime,
		"travel_mode":    mode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getEventByID retrieves an event by ID from the database
func getEventByID(eventID string) (*Event, error) {
	var event Event
	var metadata, automationConfig sql.NullString

	query := `
		SELECT id, user_id, title, description, start_time, end_time, 
		       timezone, location, event_type, status, metadata, 
		       automation_config, created_at, updated_at
		FROM events
		WHERE id = $1 AND status = 'active'
	`

	err := db.QueryRow(query, eventID).Scan(
		&event.ID, &event.UserID, &event.Title, &event.Description,
		&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
		&event.EventType, &event.Status, &metadata, &automationConfig,
		&event.CreatedAt, &event.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if metadata.Valid && metadata.String != "" {
		json.Unmarshal([]byte(metadata.String), &event.Metadata)
	}
	if automationConfig.Valid && automationConfig.String != "" {
		json.Unmarshal([]byte(automationConfig.String), &event.AutomationConfig)
	}

	return &event, nil
}
