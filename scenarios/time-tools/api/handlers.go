package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "not configured"
	
	if hasDatabase() {
		if err := db.Ping(); err != nil {
			status = "degraded"
			dbStatus = "disconnected"
		} else {
			dbStatus = "connected"
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": status,
		"version": apiVersion,
		"service": serviceName,
		"database": dbStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Timezone conversion handler
func timezoneConvertHandler(w http.ResponseWriter, r *http.Request) {
	var req TimezoneConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Parse the input time
	inputTime, err := time.Parse(time.RFC3339, req.Time)
	if err != nil {
		// Try other common formats
		inputTime, err = time.Parse("2006-01-02 15:04:05", req.Time)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid time format. Use RFC3339 or YYYY-MM-DD HH:MM:SS")
			return
		}
	}
	
	// Load timezone locations
	fromLoc, err := time.LoadLocation(req.FromTimezone)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid source timezone: %s", req.FromTimezone))
		return
	}
	
	toLoc, err := time.LoadLocation(req.ToTimezone)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("Invalid target timezone: %s", req.ToTimezone))
		return
	}
	
	// Convert time
	timeInFromZone := inputTime.In(fromLoc)
	timeInToZone := timeInFromZone.In(toLoc)
	
	// Calculate offset
	_, fromOffset := timeInFromZone.Zone()
	_, toOffset := timeInToZone.Zone()
	offsetMinutes := (toOffset - fromOffset) / 60
	
	response := TimezoneConversionResponse{
		OriginalTime:  timeInFromZone.Format(time.RFC3339),
		ConvertedTime: timeInToZone.Format(time.RFC3339),
		FromTimezone:  req.FromTimezone,
		ToTimezone:    req.ToTimezone,
		OffsetMinutes: offsetMinutes,
		IsDST:         isDST(timeInToZone),
	}
	
	respondJSON(w, http.StatusOK, response)
}

// Duration calculation handler
func durationCalculateHandler(w http.ResponseWriter, r *http.Request) {
	var req DurationCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid start time format")
		return
	}
	
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid end time format")
		return
	}
	
	// Calculate basic duration
	duration := endTime.Sub(startTime)
	totalMinutes := int(duration.Minutes())
	totalHours := duration.Hours()
	totalDays := totalHours / 24
	calendarDays := int(math.Ceil(totalDays))
	
	// Calculate business days if excluding weekends
	businessDays := 0
	if req.ExcludeWeekends {
		current := startTime
		for current.Before(endTime) {
			if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
				businessDays++
			}
			current = current.AddDate(0, 0, 1)
		}
	}
	
	response := DurationCalculationResponse{
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		TotalMinutes: totalMinutes,
		TotalHours:   totalHours,
		TotalDays:    totalDays,
		CalendarDays: calendarDays,
		BusinessDays: businessDays,
	}
	
	respondJSON(w, http.StatusOK, response)
}

// Schedule optimization handler
func scheduleOptimalHandler(w http.ResponseWriter, r *http.Request) {
	var req ScheduleOptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Parse date range
	earliestDate, err := time.Parse("2006-01-02", req.EarliestDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid earliest date format")
		return
	}
	
	latestDate, err := time.Parse("2006-01-02", req.LatestDate)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid latest date format")
		return
	}
	
	// Generate optimal slots based on business logic
	optimalSlots := generateOptimalSlots(earliestDate, latestDate, req)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"optimal_slots": optimalSlots,
		"request": req,
	})
}

// Conflict detection handler
func conflictDetectHandler(w http.ResponseWriter, r *http.Request) {
	var req ConflictDetectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Parse times
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid start time format")
		return
	}
	
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid end time format")
		return
	}
	
	conflicts := []ConflictInfo{}
	
	// If database is available, check for real conflicts
	if hasDatabase() {
		query := `
			SELECT 
				se.id, se.title, 'time_overlap' as conflict_type,
				CASE 
					WHEN se.priority = 'urgent' THEN 'critical'
					WHEN se.priority = 'high' THEN 'high'
					ELSE 'medium'
				END as severity,
				EXTRACT(EPOCH FROM (
					LEAST($2::timestamptz, se.end_time) - 
					GREATEST($1::timestamptz, se.start_time)
				))/60 as overlap_minutes,
				se.start_time, se.end_time
			FROM scheduled_events se
			WHERE se.organizer_id = $3
			  AND se.deleted_at IS NULL
			  AND se.status != 'cancelled'
			  AND (se.start_time, se.end_time) OVERLAPS ($1::timestamptz, $2::timestamptz)`
		
		rows, err := db.Query(query, startTime, endTime, req.OrganizerID)
		if err == nil {
			defer rows.Close()
			
			for rows.Next() {
				var conflict ConflictInfo
				err := rows.Scan(
					&conflict.EventID,
					&conflict.EventTitle,
					&conflict.ConflictType,
					&conflict.Severity,
					&conflict.OverlapMinutes,
					&conflict.StartTime,
					&conflict.EndTime,
				)
				if err == nil {
					conflicts = append(conflicts, conflict)
				}
			}
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"conflicts":       conflicts,
		"has_conflicts":   len(conflicts) > 0,
		"conflict_count":  len(conflicts),
	})
}

// Create event handler
func createEventHandler(w http.ResponseWriter, r *http.Request) {
	if !hasDatabase() {
		respondError(w, http.StatusServiceUnavailable, "Database not available")
		return
	}
	
	var event ScheduledEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Generate UUID
	event.ID = uuid.New()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()
	
	// Insert into database
	query := `
		INSERT INTO scheduled_events 
		(id, title, description, start_time, end_time, timezone, all_day, 
		 event_type, status, priority, organizer_id, participants, location, 
		 location_type, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id`
	
	participantsJSON, _ := json.Marshal(event.Participants)
	tagsArray := pq.Array(event.Tags)
	
	err := db.QueryRow(query,
		event.ID, event.Title, event.Description, event.StartTime, event.EndTime,
		event.Timezone, event.AllDay, event.EventType, event.Status, event.Priority,
		event.OrganizerID, participantsJSON, event.Location, event.LocationType,
		tagsArray, event.CreatedAt, event.UpdatedAt,
	).Scan(&event.ID)
	
	if err != nil {
		logger.Printf("Error creating event: %v", err)
		respondError(w, http.StatusInternalServerError, "Error creating event")
		return
	}
	
	respondJSON(w, http.StatusCreated, event)
}

// List events handler
func listEventsHandler(w http.ResponseWriter, r *http.Request) {
	if !hasDatabase() {
		// Return empty list if no database
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"events": []ScheduledEvent{},
			"count":  0,
		})
		return
	}
	
	// Parse query parameters
	organizerID := r.URL.Query().Get("organizer_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	status := r.URL.Query().Get("status")
	
	// Build query
	query := `
		SELECT id, title, description, start_time, end_time, timezone, all_day,
		       event_type, status, priority, organizer_id, participants, location,
		       location_type, tags, created_at, updated_at
		FROM scheduled_events
		WHERE deleted_at IS NULL`
	
	args := []interface{}{}
	argCount := 0
	
	if organizerID != "" {
		argCount++
		query += fmt.Sprintf(" AND organizer_id = $%d", argCount)
		args = append(args, organizerID)
	}
	
	if startDate != "" {
		argCount++
		query += fmt.Sprintf(" AND start_time >= $%d", argCount)
		args = append(args, startDate)
	}
	
	if endDate != "" {
		argCount++
		query += fmt.Sprintf(" AND end_time <= $%d", argCount)
		args = append(args, endDate)
	}
	
	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}
	
	query += " ORDER BY start_time DESC LIMIT 100"
	
	rows, err := db.Query(query, args...)
	if err != nil {
		logger.Printf("Error listing events: %v", err)
		respondError(w, http.StatusInternalServerError, "Error listing events")
		return
	}
	defer rows.Close()
	
	var events []ScheduledEvent
	for rows.Next() {
		var event ScheduledEvent
		var participantsJSON []byte
		var tagsArray pq.StringArray
		
		err := rows.Scan(
			&event.ID, &event.Title, &event.Description, &event.StartTime, &event.EndTime,
			&event.Timezone, &event.AllDay, &event.EventType, &event.Status, &event.Priority,
			&event.OrganizerID, &participantsJSON, &event.Location, &event.LocationType,
			&tagsArray, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			logger.Printf("Error scanning event: %v", err)
			continue
		}
		
		json.Unmarshal(participantsJSON, &event.Participants)
		event.Tags = []string(tagsArray)
		events = append(events, event)
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

// Format time handler
func formatTimeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Time     string `json:"time"`
		Format   string `json:"format"`
		Timezone string `json:"timezone,omitempty"`
		Locale   string `json:"locale,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Parse the input time
	inputTime, err := time.Parse(time.RFC3339, req.Time)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid time format")
		return
	}
	
	// Apply timezone if specified
	if req.Timezone != "" {
		loc, err := time.LoadLocation(req.Timezone)
		if err == nil {
			inputTime = inputTime.In(loc)
		}
	}
	
	// Format based on requested format
	var formatted string
	switch strings.ToLower(req.Format) {
	case "iso8601", "rfc3339":
		formatted = inputTime.Format(time.RFC3339)
	case "unix":
		formatted = strconv.FormatInt(inputTime.Unix(), 10)
	case "date":
		formatted = inputTime.Format("2006-01-02")
	case "time":
		formatted = inputTime.Format("15:04:05")
	case "datetime":
		formatted = inputTime.Format("2006-01-02 15:04:05")
	case "human":
		formatted = inputTime.Format("Monday, January 2, 2006 at 3:04 PM")
	case "relative":
		formatted = getRelativeTime(inputTime)
	default:
		// Use custom format string
		formatted = inputTime.Format(req.Format)
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"original":  req.Time,
		"formatted": formatted,
		"format":    req.Format,
		"timezone":  req.Timezone,
	})
}

// Helper function for relative time
func getRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	
	if diff < 0 {
		diff = -diff
		if diff < time.Minute {
			return "in a few seconds"
		} else if diff < time.Hour {
			minutes := int(diff.Minutes())
			if minutes == 1 {
				return "in 1 minute"
			}
			return fmt.Sprintf("in %d minutes", minutes)
		} else if diff < 24*time.Hour {
			hours := int(diff.Hours())
			if hours == 1 {
				return "in 1 hour"
			}
			return fmt.Sprintf("in %d hours", hours)
		} else {
			days := int(diff.Hours() / 24)
			if days == 1 {
				return "tomorrow"
			}
			return fmt.Sprintf("in %d days", days)
		}
	}
	
	if diff < time.Minute {
		return "just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// Check if DST is active
func isDST(t time.Time) bool {
	// Simple DST check - compare offset in January vs July
	jan := time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	jul := time.Date(t.Year(), 7, 1, 0, 0, 0, 0, t.Location())
	_, janOffset := jan.Zone()
	_, julOffset := jul.Zone()
	_, currentOffset := t.Zone()
	
	if janOffset != julOffset {
		// Location observes DST
		return currentOffset != janOffset
	}
	return false
}

// Generate optimal time slots for scheduling
func generateOptimalSlots(startDate, endDate time.Time, req ScheduleOptimizationRequest) []OptimalSlot {
	var slots []OptimalSlot
	durationHours := time.Duration(req.DurationMinutes) * time.Minute
	
	// Business hours default: 9 AM to 5 PM
	businessStart := 9
	businessEnd := 17
	
	currentDate := startDate
	for currentDate.Before(endDate.AddDate(0, 0, 1)) {
		// Skip weekends if business hours only
		if req.BusinessHoursOnly && (currentDate.Weekday() == time.Saturday || currentDate.Weekday() == time.Sunday) {
			currentDate = currentDate.AddDate(0, 0, 1)
			continue
		}
		
		// Generate time slots for this day
		startHour := businessStart
		endHour := businessEnd
		
		if !req.BusinessHoursOnly {
			startHour = 8  // Earlier start for non-business hours
			endHour = 20   // Later end for non-business hours
		}
		
		for hour := startHour; hour < endHour; hour += 2 { // 2-hour intervals
			slotStart := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 
				hour, 0, 0, 0, currentDate.Location())
			slotEnd := slotStart.Add(durationHours)
			
			// Make sure slot doesn't exceed business hours
			maxEnd := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 
				endHour, 0, 0, 0, currentDate.Location())
			if slotEnd.After(maxEnd) {
				continue
			}
			
			// Calculate score based on time preferences
			score := calculateSlotScore(slotStart, req)
			
			// Check for conflicts if database is available
			conflictCount := 0
			participantsFree := req.Participants
			
			if hasDatabase() {
				conflictCount = checkSlotConflicts(slotStart, slotEnd, req.Participants)
				participantsFreeCount := len(req.Participants) - conflictCount
				if participantsFreeCount > 0 {
					participantsFree = req.Participants[:participantsFreeCount]
				} else {
					participantsFree = []string{}
				}
			}
			
			slots = append(slots, OptimalSlot{
				StartTime:        slotStart.Format(time.RFC3339),
				EndTime:          slotEnd.Format(time.RFC3339),
				Score:            score,
				ConflictCount:    conflictCount,
				ParticipantsFree: participantsFree,
			})
		}
		
		currentDate = currentDate.AddDate(0, 0, 1)
	}
	
	// Sort by score (highest first) and limit to top 10
	for i := 0; i < len(slots)-1; i++ {
		for j := i + 1; j < len(slots); j++ {
			if slots[j].Score > slots[i].Score {
				slots[i], slots[j] = slots[j], slots[i]
			}
		}
	}
	
	if len(slots) > 10 {
		slots = slots[:10]
	}
	
	return slots
}

// Calculate score for a time slot based on preferences
func calculateSlotScore(slotStart time.Time, req ScheduleOptimizationRequest) float64 {
	score := 0.5 // Base score
	hour := slotStart.Hour()
	
	// Prefer mid-morning and early afternoon
	if hour >= 10 && hour <= 11 {
		score += 0.3 // Morning preference
	} else if hour >= 14 && hour <= 15 {
		score += 0.2 // Afternoon preference
	} else if hour >= 9 && hour <= 16 {
		score += 0.1 // General business hours
	}
	
	// Weekday preference
	weekday := slotStart.Weekday()
	if weekday >= time.Tuesday && weekday <= time.Thursday {
		score += 0.1 // Mid-week preference
	}
	
	// Add some randomness to make it more realistic
	if hour%2 == 0 {
		score += 0.05
	}
	
	return math.Min(score, 1.0)
}

// Check for scheduling conflicts
func checkSlotConflicts(startTime, endTime time.Time, participants []string) int {
	if !hasDatabase() {
		return 0
	}
	
	// Simple conflict check - count overlapping events for participants
	query := `
		SELECT COUNT(*)
		FROM scheduled_events
		WHERE deleted_at IS NULL
		  AND status != 'cancelled'
		  AND (start_time, end_time) OVERLAPS ($1, $2)
		  AND organizer_id = ANY($3)
	`
	
	var conflictCount int
	err := db.QueryRow(query, startTime, endTime, pq.Array(participants)).Scan(&conflictCount)
	if err != nil {
		logger.Printf("Error checking conflicts: %v", err)
		return 0
	}
	
	return conflictCount
}