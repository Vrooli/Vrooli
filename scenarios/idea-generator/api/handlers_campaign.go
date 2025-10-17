package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// campaignsHandler handles listing and creating campaigns
func (s *ApiServer) campaignsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Query campaigns from database
		query := `SELECT id, name, description, color, created_at FROM campaigns WHERE is_active = true ORDER BY created_at DESC`
		rows, err := s.db.Query(query)
		if err != nil {
			log.Printf("Failed to query campaigns: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		campaigns := []Campaign{}
		for rows.Next() {
			var campaign Campaign
			if err := rows.Scan(&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Color, &campaign.CreatedAt); err != nil {
				log.Printf("Failed to scan campaign: %v", err)
				continue
			}
			campaigns = append(campaigns, campaign)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(campaigns); err != nil {
			log.Printf("Failed to encode campaigns response: %v", err)
		}

	case "POST":
		var campaign Campaign
		if err := json.NewDecoder(r.Body).Decode(&campaign); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Validate required fields
		if campaign.Name == "" {
			http.Error(w, "Campaign name is required", http.StatusBadRequest)
			return
		}
		if len(campaign.Name) > 100 {
			http.Error(w, "Campaign name must be 100 characters or less", http.StatusBadRequest)
			return
		}
		if len(campaign.Description) > 500 {
			http.Error(w, "Campaign description must be 500 characters or less", http.StatusBadRequest)
			return
		}

		// Insert into database
		query := `INSERT INTO campaigns (name, description, color) VALUES ($1, $2, $3) RETURNING id, created_at`
		err := s.db.QueryRow(query, campaign.Name, campaign.Description, campaign.Color).Scan(&campaign.ID, &campaign.CreatedAt)
		if err != nil {
			log.Printf("Failed to create campaign: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(campaign); err != nil {
			log.Printf("Failed to encode campaign response: %v", err)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// campaignByIDHandler handles getting and deleting a specific campaign
func (s *ApiServer) campaignByIDHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	campaignID := vars["id"]

	// Validate UUID format (basic check)
	if campaignID == "" {
		http.Error(w, "Campaign ID is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		// Query single campaign from database
		query := `SELECT id, name, description, color, created_at FROM campaigns WHERE id = $1 AND is_active = true`
		var campaign Campaign
		err := s.db.QueryRow(query, campaignID).Scan(&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Color, &campaign.CreatedAt)

		if err == sql.ErrNoRows {
			http.Error(w, "Campaign not found", http.StatusNotFound)
			return
		} else if err != nil {
			log.Printf("Failed to query campaign: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(campaign); err != nil {
			log.Printf("Failed to encode campaign response: %v", err)
		}

	case "DELETE":
		// Soft delete campaign by marking as inactive
		query := `UPDATE campaigns SET is_active = false WHERE id = $1`
		result, err := s.db.Exec(query, campaignID)

		if err != nil {
			log.Printf("Failed to delete campaign: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("Failed to get rows affected: %v", err)
			// Continue - if we got here, delete likely succeeded
		}
		if rowsAffected == 0 {
			http.Error(w, "Campaign not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
