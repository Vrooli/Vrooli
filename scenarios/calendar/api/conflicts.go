package main

import (
	"database/sql"
	"fmt"
	"time"
)

// EventConflict represents a scheduling conflict
type EventConflict struct {
	ConflictingEvent Event                `json:"conflicting_event"`
	ConflictType     string               `json:"conflict_type"` // "overlap", "too_close", "resource_conflict"
	Severity         string               `json:"severity"`      // "high", "medium", "low"
	Message          string               `json:"message"`
	SuggestedActions []ConflictResolution `json:"suggested_actions"`
}

// ConflictResolution represents a suggested resolution for a conflict
type ConflictResolution struct {
	Action      string                 `json:"action"` // "reschedule", "shorten", "cancel", "merge"
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Confidence  float64                `json:"confidence"`
}

// ConflictDetector handles conflict detection and resolution
type ConflictDetector struct {
	db                  *sql.DB
	bufferMinutes       int // Minimum buffer time between events
	maxConcurrentEvents int // Maximum concurrent events allowed
	travelCalculator    *TravelTimeCalculator
}

// NewConflictDetector creates a new conflict detector
func NewConflictDetector(db *sql.DB) *ConflictDetector {
	return &ConflictDetector{
		db:                  db,
		bufferMinutes:       15, // Default 15 minute buffer between events
		maxConcurrentEvents: 3,  // Default max 3 concurrent events
		travelCalculator:    NewTravelTimeCalculator(&Config{}),
	}
}

// CheckConflicts checks for conflicts with existing events
func (cd *ConflictDetector) CheckConflicts(userID string, startTime, endTime time.Time, excludeEventID string) ([]EventConflict, error) {
	conflicts := []EventConflict{}

	// Add buffer time for checking "too close" conflicts
	bufferDuration := time.Duration(cd.bufferMinutes) * time.Minute
	checkStartTime := startTime.Add(-bufferDuration)
	checkEndTime := endTime.Add(bufferDuration)

	// Query for potentially conflicting events
	query := `
		SELECT id, user_id, title, description, start_time, end_time, timezone, 
		       location, event_type, status, metadata, automation_config, created_at, updated_at
		FROM events 
		WHERE user_id = $1 
		AND status = 'active'
		AND (
			-- Direct overlap
			(start_time < $3 AND end_time > $2) OR
			-- Event starts during new event
			(start_time >= $2 AND start_time < $3) OR
			-- Event ends during new event
			(end_time > $2 AND end_time <= $3) OR
			-- Events within buffer time
			(end_time > $4 AND start_time < $5)
		)
	`

	args := []interface{}{userID, startTime, endTime, checkStartTime, checkEndTime}
	argCount := 5

	// Exclude specific event ID (for updates)
	if excludeEventID != "" {
		argCount++
		query += fmt.Sprintf(" AND id != $%d", argCount)
		args = append(args, excludeEventID)
	}

	query += " ORDER BY start_time"

	rows, err := cd.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to check conflicts: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event Event
		var metadataJSON, automationJSON []byte

		err := rows.Scan(
			&event.ID, &event.UserID, &event.Title, &event.Description,
			&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
			&event.EventType, &event.Status, &metadataJSON, &automationJSON,
			&event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %v", err)
		}

		conflict := cd.analyzeConflict(event, startTime, endTime)
		if conflict != nil {
			conflicts = append(conflicts, *conflict)
		}
	}

	// Check for too many concurrent events
	concurrentCount := len(conflicts)
	if concurrentCount >= cd.maxConcurrentEvents {
		// Add a special conflict for too many concurrent events
		conflicts = append(conflicts, EventConflict{
			ConflictType: "too_many_concurrent",
			Severity:     "medium",
			Message:      fmt.Sprintf("You already have %d events scheduled during this time. Consider spreading out your schedule.", concurrentCount),
			SuggestedActions: []ConflictResolution{
				{
					Action:      "reschedule",
					Description: "Move this event to a less busy time slot",
					Parameters: map[string]interface{}{
						"suggested_times": cd.findAvailableSlots(userID, startTime, endTime),
					},
					Confidence: 0.9,
				},
			},
		})
	}

	return conflicts, nil
}

// analyzeConflict determines the type and severity of a conflict
func (cd *ConflictDetector) analyzeConflict(existingEvent Event, newStartTime, newEndTime time.Time) *EventConflict {
	bufferDuration := time.Duration(cd.bufferMinutes) * time.Minute

	// Check for direct time overlap
	if existingEvent.StartTime.Before(newEndTime) && existingEvent.EndTime.After(newStartTime) {
		overlap := cd.calculateOverlap(existingEvent.StartTime, existingEvent.EndTime, newStartTime, newEndTime)

		return &EventConflict{
			ConflictingEvent: existingEvent,
			ConflictType:     "overlap",
			Severity:         "high",
			Message:          fmt.Sprintf("This time directly conflicts with '%s' (%.0f%% overlap)", existingEvent.Title, overlap),
			SuggestedActions: cd.generateResolutions(existingEvent, newStartTime, newEndTime, "overlap"),
		}
	}

	// Check if events are too close together
	if existingEvent.EndTime.Add(bufferDuration).After(newStartTime) &&
		existingEvent.EndTime.Before(newStartTime) {
		minutesApart := newStartTime.Sub(existingEvent.EndTime).Minutes()

		// Check if travel time is needed between locations
		travelTimeNeeded := cd.checkTravelTimeNeeded(existingEvent, newStartTime, newEndTime)
		if travelTimeNeeded > 0 {
			if minutesApart < float64(travelTimeNeeded/60) {
				return &EventConflict{
					ConflictingEvent: existingEvent,
					ConflictType:     "insufficient_travel_time",
					Severity:         "medium",
					Message: fmt.Sprintf("Only %.0f minutes between events, but %.0f minutes needed for travel from '%s' to new location",
						minutesApart, float64(travelTimeNeeded/60), existingEvent.Location),
					SuggestedActions: cd.generateResolutions(existingEvent, newStartTime, newEndTime, "travel_time"),
				}
			}
		}

		return &EventConflict{
			ConflictingEvent: existingEvent,
			ConflictType:     "too_close",
			Severity:         "low",
			Message:          fmt.Sprintf("Only %.0f minutes after '%s' ends. Consider adding more buffer time.", minutesApart, existingEvent.Title),
			SuggestedActions: cd.generateResolutions(existingEvent, newStartTime, newEndTime, "too_close"),
		}
	}

	// Check if new event ends too close to existing event
	if newEndTime.Add(bufferDuration).After(existingEvent.StartTime) &&
		newEndTime.Before(existingEvent.StartTime) {
		minutesApart := existingEvent.StartTime.Sub(newEndTime).Minutes()
		return &EventConflict{
			ConflictingEvent: existingEvent,
			ConflictType:     "too_close",
			Severity:         "low",
			Message:          fmt.Sprintf("Only %.0f minutes before '%s' starts. Consider adding more buffer time.", minutesApart, existingEvent.Title),
			SuggestedActions: cd.generateResolutions(existingEvent, newStartTime, newEndTime, "too_close"),
		}
	}

	return nil
}

// calculateOverlap calculates the percentage of overlap between two time periods
func (cd *ConflictDetector) calculateOverlap(start1, end1, start2, end2 time.Time) float64 {
	// Find the overlapping period
	overlapStart := start1
	if start2.After(start1) {
		overlapStart = start2
	}

	overlapEnd := end1
	if end2.Before(end1) {
		overlapEnd = end2
	}

	if overlapEnd.Before(overlapStart) {
		return 0
	}

	overlapDuration := overlapEnd.Sub(overlapStart).Minutes()
	totalDuration := end2.Sub(start2).Minutes()

	if totalDuration == 0 {
		return 0
	}

	return (overlapDuration / totalDuration) * 100
}

// generateResolutions creates suggested resolutions for a conflict
func (cd *ConflictDetector) generateResolutions(existingEvent Event, newStartTime, newEndTime time.Time, conflictType string) []ConflictResolution {
	resolutions := []ConflictResolution{}

	switch conflictType {
	case "overlap":
		// Suggest rescheduling to just after the existing event
		suggestedStart := existingEvent.EndTime.Add(time.Duration(cd.bufferMinutes) * time.Minute)
		duration := newEndTime.Sub(newStartTime)
		suggestedEnd := suggestedStart.Add(duration)

		resolutions = append(resolutions, ConflictResolution{
			Action: "reschedule",
			Description: fmt.Sprintf("Move to %s - %s (right after '%s')",
				suggestedStart.Format("3:04 PM"),
				suggestedEnd.Format("3:04 PM"),
				existingEvent.Title),
			Parameters: map[string]interface{}{
				"new_start_time": suggestedStart.Format(time.RFC3339),
				"new_end_time":   suggestedEnd.Format(time.RFC3339),
			},
			Confidence: 0.8,
		})

		// Suggest shortening the new event
		if newEndTime.Sub(newStartTime) > 30*time.Minute {
			shortenedEnd := newStartTime.Add(30 * time.Minute)
			if !cd.hasOverlap(existingEvent.StartTime, existingEvent.EndTime, newStartTime, shortenedEnd) {
				resolutions = append(resolutions, ConflictResolution{
					Action:      "shorten",
					Description: "Shorten to 30 minutes to avoid conflict",
					Parameters: map[string]interface{}{
						"new_end_time": shortenedEnd.Format(time.RFC3339),
					},
					Confidence: 0.6,
				})
			}
		}

		// Suggest merging if events are similar
		if cd.areEventsSimilar(existingEvent, newStartTime, newEndTime) {
			resolutions = append(resolutions, ConflictResolution{
				Action:      "merge",
				Description: fmt.Sprintf("Combine with existing event '%s'", existingEvent.Title),
				Parameters: map[string]interface{}{
					"merge_with_id": existingEvent.ID,
				},
				Confidence: 0.7,
			})
		}

	case "too_close":
		// Suggest adding more buffer time
		bufferDuration := time.Duration(cd.bufferMinutes) * time.Minute
		adjustedStart := newStartTime

		if existingEvent.EndTime.Before(newStartTime) {
			// Existing event is before, push new event later
			adjustedStart = existingEvent.EndTime.Add(bufferDuration)
		}

		duration := newEndTime.Sub(newStartTime)
		adjustedEnd := adjustedStart.Add(duration)

		resolutions = append(resolutions, ConflictResolution{
			Action:      "adjust_time",
			Description: fmt.Sprintf("Add %d minute buffer", cd.bufferMinutes),
			Parameters: map[string]interface{}{
				"new_start_time": adjustedStart.Format(time.RFC3339),
				"new_end_time":   adjustedEnd.Format(time.RFC3339),
			},
			Confidence: 0.9,
		})

		// Suggest proceeding anyway for low severity conflicts
		resolutions = append(resolutions, ConflictResolution{
			Action:      "proceed_anyway",
			Description: "Schedule anyway with minimal buffer time",
			Parameters:  map[string]interface{}{},
			Confidence:  0.5,
		})
	}

	return resolutions
}

// hasOverlap checks if two time periods overlap
func (cd *ConflictDetector) hasOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

// areEventsSimilar checks if two events might be related or similar
func (cd *ConflictDetector) areEventsSimilar(existingEvent Event, newStartTime, newEndTime time.Time) bool {
	// Check if events are close in time (within 1 hour)
	timeDiff := newStartTime.Sub(existingEvent.StartTime).Abs().Minutes()
	if timeDiff > 60 {
		return false
	}

	// Check if event types match
	if existingEvent.EventType == "meeting" || existingEvent.EventType == "appointment" {
		return true
	}

	// Could add more sophisticated similarity checks here (title similarity, location, etc.)

	return false
}

// findAvailableSlots finds available time slots for scheduling
func (cd *ConflictDetector) findAvailableSlots(userID string, preferredStart, preferredEnd time.Time) []map[string]string {
	slots := []map[string]string{}
	duration := preferredEnd.Sub(preferredStart)

	// Look for slots in the next 7 days
	searchStart := preferredStart
	searchEnd := preferredStart.AddDate(0, 0, 7)

	// Query for all events in the search period
	query := `
		SELECT start_time, end_time 
		FROM events 
		WHERE user_id = $1 
		AND status = 'active'
		AND start_time >= $2 
		AND start_time < $3
		ORDER BY start_time
	`

	rows, err := cd.db.Query(query, userID, searchStart, searchEnd)
	if err != nil {
		return slots
	}
	defer rows.Close()

	var lastEndTime time.Time = searchStart

	for rows.Next() {
		var eventStart, eventEnd time.Time
		if err := rows.Scan(&eventStart, &eventEnd); err != nil {
			continue
		}

		// Check if there's a gap before this event
		gap := eventStart.Sub(lastEndTime)
		if gap >= duration+time.Duration(cd.bufferMinutes*2)*time.Minute {
			// Found a suitable gap
			slotStart := lastEndTime.Add(time.Duration(cd.bufferMinutes) * time.Minute)
			slotEnd := slotStart.Add(duration)

			slots = append(slots, map[string]string{
				"start_time": slotStart.Format(time.RFC3339),
				"end_time":   slotEnd.Format(time.RFC3339),
				"label":      slotStart.Format("Mon Jan 2, 3:04 PM"),
			})

			if len(slots) >= 3 {
				break // Return at most 3 suggestions
			}
		}

		lastEndTime = eventEnd
	}

	// Check if there's space after the last event
	if len(slots) < 3 && lastEndTime.Before(searchEnd) {
		slotStart := lastEndTime.Add(time.Duration(cd.bufferMinutes) * time.Minute)
		slotEnd := slotStart.Add(duration)

		if slotEnd.Before(searchEnd) {
			slots = append(slots, map[string]string{
				"start_time": slotStart.Format(time.RFC3339),
				"end_time":   slotEnd.Format(time.RFC3339),
				"label":      slotStart.Format("Mon Jan 2, 3:04 PM"),
			})
		}
	}

	return slots
}

// checkTravelTimeNeeded calculates travel time needed between events
func (cd *ConflictDetector) checkTravelTimeNeeded(existingEvent Event, newStartTime, newEndTime time.Time) int {
	// If no travel calculator or no locations, use default buffer
	if cd.travelCalculator == nil || existingEvent.Location == "" {
		return 0
	}

	// Create a temporary event for calculation
	newEvent := &Event{
		Location:  "", // This would be passed in from the request
		StartTime: newStartTime,
		EndTime:   newEndTime,
	}

	// Calculate travel time between events
	travelTime, err := cd.travelCalculator.CalculateBufferTime(&existingEvent, newEvent)
	if err != nil {
		// On error, return default buffer time in seconds
		return cd.bufferMinutes * 60
	}

	return travelTime
}

// ResolveConflict applies a conflict resolution
func (cd *ConflictDetector) ResolveConflict(resolution ConflictResolution, eventRequest *CreateEventRequest) error {
	switch resolution.Action {
	case "reschedule":
		if newStart, ok := resolution.Parameters["new_start_time"].(string); ok {
			eventRequest.StartTime = newStart
		}
		if newEnd, ok := resolution.Parameters["new_end_time"].(string); ok {
			eventRequest.EndTime = newEnd
		}

	case "shorten":
		if newEnd, ok := resolution.Parameters["new_end_time"].(string); ok {
			eventRequest.EndTime = newEnd
		}

	case "adjust_time":
		if newStart, ok := resolution.Parameters["new_start_time"].(string); ok {
			eventRequest.StartTime = newStart
		}
		if newEnd, ok := resolution.Parameters["new_end_time"].(string); ok {
			eventRequest.EndTime = newEnd
		}

	case "proceed_anyway":
		// No changes needed
		return nil

	default:
		return fmt.Errorf("unknown resolution action: %s", resolution.Action)
	}

	return nil
}
