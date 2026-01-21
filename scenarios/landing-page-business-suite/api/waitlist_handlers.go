package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// handleWaitlistCreate handles public email submission to the waitlist
func handleWaitlistCreate(svc *WaitlistService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email  string `json:"email"`
			Source string `json:"source,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate email
		email := strings.TrimSpace(req.Email)
		if email == "" {
			http.Error(w, "Email is required", http.StatusBadRequest)
			return
		}
		if _, err := mail.ParseAddress(email); err != nil {
			http.Error(w, "Invalid email format", http.StatusBadRequest)
			return
		}

		source := strings.TrimSpace(req.Source)
		if source == "" {
			source = "coming_soon"
		}

		_, err := svc.Create(r.Context(), email, source)
		if err != nil {
			logStructuredError("waitlist_create_failed", map[string]interface{}{
				"email": email,
				"error": err.Error(),
			})
			http.Error(w, "Failed to add email to waitlist", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Email added to waitlist",
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handleWaitlistList returns all waitlist emails (admin only)
func handleWaitlistList(svc *WaitlistService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emails, err := svc.List(r.Context())
		if err != nil {
			logStructuredError("waitlist_list_failed", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "Failed to list waitlist emails", http.StatusInternalServerError)
			return
		}

		if emails == nil {
			emails = []WaitlistEmail{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(emails); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handleWaitlistDelete removes an email from the waitlist (admin only)
func handleWaitlistDelete(svc *WaitlistService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := svc.Delete(r.Context(), id); err != nil {
			logStructuredError("waitlist_delete_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			http.Error(w, "Failed to delete email", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// handleWaitlistExport exports all waitlist emails as CSV (admin only)
func handleWaitlistExport(svc *WaitlistService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emails, err := svc.List(r.Context())
		if err != nil {
			logStructuredError("waitlist_export_failed", map[string]interface{}{
				"error": err.Error(),
			})
			http.Error(w, "Failed to export waitlist", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=waitlist.csv")

		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Write header
		if err := writer.Write([]string{"ID", "Email", "Source", "Created At"}); err != nil {
			logStructuredError("waitlist_export_write_header_failed", map[string]interface{}{
				"error": err.Error(),
			})
			return
		}

		// Write data rows
		for _, email := range emails {
			if err := writer.Write([]string{
				fmt.Sprintf("%d", email.ID),
				email.Email,
				email.Source,
				email.CreatedAt.Format("2006-01-02 15:04:05"),
			}); err != nil {
				logStructuredError("waitlist_export_write_row_failed", map[string]interface{}{
					"id":    email.ID,
					"error": err.Error(),
				})
				return
			}
		}
	}
}
