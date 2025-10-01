package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

// Resource represents a bookable resource (room, equipment, etc.)
type Resource struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	ResourceType      string                 `json:"resource_type"`
	Description       string                 `json:"description"`
	Location          string                 `json:"location"`
	Capacity          *int                   `json:"capacity,omitempty"`
	Metadata          map[string]interface{} `json:"metadata"`
	AvailabilityRules map[string]interface{} `json:"availability_rules"`
	Status            string                 `json:"status"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// ResourceBooking represents a resource booking for an event
type ResourceBooking struct {
	ID            string    `json:"id"`
	EventID       string    `json:"event_id"`
	ResourceID    string    `json:"resource_id"`
	BookingStatus string    `json:"booking_status"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
}

// ResourceConflict represents a scheduling conflict for a resource
type ResourceConflict struct {
	EventID      string    `json:"event_id"`
	EventTitle   string    `json:"event_title"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	ConflictType string    `json:"conflict_type"`
}

// ResourceManager handles resource booking operations
type ResourceManager struct {
	db *sql.DB
}

// NewResourceManager creates a new resource manager instance
func NewResourceManager(db *sql.DB) *ResourceManager {
	return &ResourceManager{db: db}
}

// CreateResource creates a new bookable resource
func (rm *ResourceManager) CreateResource(w http.ResponseWriter, r *http.Request) {
	var resource Resource
	if err := json.NewDecoder(r.Body).Decode(&resource); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if resource.Name == "" {
		http.Error(w, "Resource name is required", http.StatusBadRequest)
		return
	}

	if resource.ResourceType == "" {
		resource.ResourceType = "room"
	}

	if resource.Status == "" {
		resource.Status = "active"
	}

	// Insert resource into database
	query := `
		INSERT INTO resources (name, resource_type, description, location, capacity, metadata, availability_rules, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	var metadataJSON, rulesJSON []byte
	var err error

	if resource.Metadata != nil {
		metadataJSON, err = json.Marshal(resource.Metadata)
		if err != nil {
			http.Error(w, "Invalid metadata format", http.StatusBadRequest)
			return
		}
	} else {
		metadataJSON = []byte("{}")
	}

	if resource.AvailabilityRules != nil {
		rulesJSON, err = json.Marshal(resource.AvailabilityRules)
		if err != nil {
			http.Error(w, "Invalid availability rules format", http.StatusBadRequest)
			return
		}
	} else {
		rulesJSON = []byte("{}")
	}

	err = rm.db.QueryRow(query,
		resource.Name,
		resource.ResourceType,
		resource.Description,
		resource.Location,
		resource.Capacity,
		metadataJSON,
		rulesJSON,
		resource.Status,
	).Scan(&resource.ID, &resource.CreatedAt, &resource.UpdatedAt)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resource)
}

// ListResources returns all available resources
func (rm *ResourceManager) ListResources(w http.ResponseWriter, r *http.Request) {
	resourceType := r.URL.Query().Get("type")
	status := r.URL.Query().Get("status")

	query := `
		SELECT id, name, resource_type, description, location, capacity, 
		       metadata, availability_rules, status, created_at, updated_at
		FROM resources
		WHERE 1=1
	`

	var args []interface{}
	argCount := 0

	if resourceType != "" {
		argCount++
		query += fmt.Sprintf(" AND resource_type = $%d", argCount)
		args = append(args, resourceType)
	}

	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	} else {
		// Default to active resources only
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, "active")
	}

	query += " ORDER BY name"

	rows, err := rm.db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query resources: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var resources []Resource
	for rows.Next() {
		var r Resource
		var metadataJSON, rulesJSON []byte

		err := rows.Scan(
			&r.ID, &r.Name, &r.ResourceType, &r.Description, &r.Location, &r.Capacity,
			&metadataJSON, &rulesJSON, &r.Status, &r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &r.Metadata)
		}
		if rulesJSON != nil {
			json.Unmarshal(rulesJSON, &r.AvailabilityRules)
		}

		resources = append(resources, r)
	}

	response := map[string]interface{}{
		"resources": resources,
		"count":     len(resources),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CheckResourceAvailability checks if a resource is available for a given time period
func (rm *ResourceManager) CheckResourceAvailability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	resourceID := vars["id"]

	startTimeStr := r.URL.Query().Get("start_time")
	endTimeStr := r.URL.Query().Get("end_time")
	excludeEventID := r.URL.Query().Get("exclude_event_id")

	if startTimeStr == "" || endTimeStr == "" {
		http.Error(w, "start_time and end_time are required", http.StatusBadRequest)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		http.Error(w, "Invalid start_time format", http.StatusBadRequest)
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		http.Error(w, "Invalid end_time format", http.StatusBadRequest)
		return
	}

	// Check availability using the database function
	var isAvailable bool
	query := `SELECT is_resource_available($1, $2, $3, $4)`

	var excludeEventPtr *string
	if excludeEventID != "" {
		excludeEventPtr = &excludeEventID
	}

	err = rm.db.QueryRow(query, resourceID, startTime, endTime, excludeEventPtr).Scan(&isAvailable)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check availability: %v", err), http.StatusInternalServerError)
		return
	}

	// Get conflicts if not available
	var conflicts []ResourceConflict
	if !isAvailable {
		conflictQuery := `SELECT * FROM get_resource_conflicts($1, $2, $3)`
		rows, err := rm.db.Query(conflictQuery, resourceID, startTime, endTime)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var c ResourceConflict
				rows.Scan(&c.EventID, &c.EventTitle, &c.StartTime, &c.EndTime, &c.ConflictType)
				conflicts = append(conflicts, c)
			}
		}
	}

	response := map[string]interface{}{
		"resource_id": resourceID,
		"available":   isAvailable,
		"start_time":  startTime,
		"end_time":    endTime,
		"conflicts":   conflicts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// BookResourceForEvent books a resource for an event
func (rm *ResourceManager) BookResourceForEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	var request struct {
		ResourceID    string `json:"resource_id"`
		BookingStatus string `json:"booking_status"`
		Notes         string `json:"notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if request.ResourceID == "" {
		http.Error(w, "resource_id is required", http.StatusBadRequest)
		return
	}

	if request.BookingStatus == "" {
		request.BookingStatus = "confirmed"
	}

	// First check if the event exists and get its time range
	var startTime, endTime time.Time
	eventQuery := `SELECT start_time, end_time FROM events WHERE id = $1 AND status = 'active'`
	err := rm.db.QueryRow(eventQuery, eventID).Scan(&startTime, &endTime)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Event not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to query event: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Check if resource is available for this time period
	var isAvailable bool
	availQuery := `SELECT is_resource_available($1, $2, $3, $4)`
	err = rm.db.QueryRow(availQuery, request.ResourceID, startTime, endTime, eventID).Scan(&isAvailable)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to check availability: %v", err), http.StatusInternalServerError)
		return
	}

	if !isAvailable {
		// Get conflicts to provide helpful error message
		conflictQuery := `SELECT * FROM get_resource_conflicts($1, $2, $3)`
		rows, _ := rm.db.Query(conflictQuery, request.ResourceID, startTime, endTime)
		defer rows.Close()

		var conflicts []ResourceConflict
		for rows.Next() {
			var c ResourceConflict
			rows.Scan(&c.EventID, &c.EventTitle, &c.StartTime, &c.EndTime, &c.ConflictType)
			conflicts = append(conflicts, c)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":     "Resource is not available for the requested time period",
			"conflicts": conflicts,
		})
		return
	}

	// Book the resource
	var bookingID string
	var createdAt time.Time
	insertQuery := `
		INSERT INTO event_resources (event_id, resource_id, booking_status, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	err = rm.db.QueryRow(insertQuery, eventID, request.ResourceID, request.BookingStatus, request.Notes).Scan(&bookingID, &createdAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // Unique constraint violation
			http.Error(w, "Resource already booked for this event", http.StatusConflict)
		} else {
			http.Error(w, fmt.Sprintf("Failed to book resource: %v", err), http.StatusInternalServerError)
		}
		return
	}

	response := ResourceBooking{
		ID:            bookingID,
		EventID:       eventID,
		ResourceID:    request.ResourceID,
		BookingStatus: request.BookingStatus,
		Notes:         request.Notes,
		CreatedAt:     createdAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetEventResources returns all resources booked for an event
func (rm *ResourceManager) GetEventResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]

	query := `
		SELECT er.id, er.resource_id, r.name, r.resource_type, r.location, 
		       er.booking_status, er.notes, er.created_at
		FROM event_resources er
		JOIN resources r ON er.resource_id = r.id
		WHERE er.event_id = $1
		ORDER BY r.name
	`

	rows, err := rm.db.Query(query, eventID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to query event resources: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var resources []map[string]interface{}
	for rows.Next() {
		var bookingID, resourceID, name, resourceType, location, bookingStatus, notes string
		var createdAt time.Time

		err := rows.Scan(&bookingID, &resourceID, &name, &resourceType, &location, &bookingStatus, &notes, &createdAt)
		if err != nil {
			continue
		}

		resources = append(resources, map[string]interface{}{
			"booking_id":     bookingID,
			"resource_id":    resourceID,
			"resource_name":  name,
			"resource_type":  resourceType,
			"location":       location,
			"booking_status": bookingStatus,
			"notes":          notes,
			"created_at":     createdAt,
		})
	}

	response := map[string]interface{}{
		"event_id":  eventID,
		"resources": resources,
		"count":     len(resources),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CancelResourceBooking cancels a resource booking for an event
func (rm *ResourceManager) CancelResourceBooking(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["event_id"]
	resourceID := vars["resource_id"]

	query := `
		UPDATE event_resources 
		SET booking_status = 'cancelled' 
		WHERE event_id = $1 AND resource_id = $2 
		RETURNING id
	`

	var bookingID string
	err := rm.db.QueryRow(query, eventID, resourceID).Scan(&bookingID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Booking not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to cancel booking: %v", err), http.StatusInternalServerError)
		}
		return
	}

	response := map[string]interface{}{
		"message":     "Resource booking cancelled successfully",
		"booking_id":  bookingID,
		"event_id":    eventID,
		"resource_id": resourceID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
