package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
)

// handleWaitlistCreate handles public email submission to the waitlist
func handleWaitlistCreate(svc WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email  string `json:"email"`
			Source string `json:"source,omitempty"`
		}

		if !decodeJSONBody(w, r, &req) {
			return
		}

		// Validate email
		email, ok := ValidateEmailForHandler(w, req.Email)
		if !ok {
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
			writeJSONError(w, http.StatusInternalServerError, "Failed to add email to waitlist", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccess(w, "Email added to waitlist")
	}
}

// handleWaitlistList returns all waitlist emails (admin only)
func handleWaitlistList(svc WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emails, err := svc.List(r.Context())
		if err != nil {
			logStructuredError("waitlist_list_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to list waitlist emails", ApiErrorTypeServerError)
			return
		}

		if emails == nil {
			emails = []WaitlistEmail{}
		}

		writeJSONSuccessData(w, emails)
	}
}

// handleWaitlistDelete removes an email from the waitlist (admin only)
func handleWaitlistDelete(svc WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := getPathParamInt64(w, r, "id")
		if !ok {
			return
		}

		if err := svc.Delete(r.Context(), id); err != nil {
			logStructuredError("waitlist_delete_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete email", ApiErrorTypeServerError)
			return
		}

		writeJSONSuccessSimple(w)
	}
}

// handleWaitlistExport exports all waitlist emails as CSV (admin only)
func handleWaitlistExport(svc WaitlistServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		emails, err := svc.List(r.Context())
		if err != nil {
			logStructuredError("waitlist_export_failed", map[string]interface{}{
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to export waitlist", ApiErrorTypeServerError)
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
