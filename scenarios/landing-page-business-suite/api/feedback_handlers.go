package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// handleFeedbackCreate handles POST /api/v1/feedback (public endpoint)
func handleFeedbackCreate(svc *FeedbackService, brandingSvc *BrandingService, emailSvc *EmailService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input CreateFeedbackInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request format.", ApiErrorTypeValidation)
			return
		}

		// Validate required fields
		if strings.TrimSpace(input.Email) == "" {
			writeJSONError(w, http.StatusBadRequest, "Email address is required.", ApiErrorTypeValidation)
			return
		}
		if strings.TrimSpace(input.Subject) == "" {
			writeJSONError(w, http.StatusBadRequest, "Subject is required.", ApiErrorTypeValidation)
			return
		}
		if strings.TrimSpace(input.Message) == "" {
			writeJSONError(w, http.StatusBadRequest, "Message is required.", ApiErrorTypeValidation)
			return
		}

		// Validate feedback type
		validTypes := map[string]bool{
			"refund":  true,
			"bug":     true,
			"feature": true,
			"general": true,
		}
		if !validTypes[input.Type] {
			input.Type = "general"
		}

		feedback, err := svc.Create(&input)
		if err != nil {
			logStructuredError("feedback_create_failed", map[string]interface{}{"error": err.Error()})
			writeJSONError(w, http.StatusInternalServerError, "Failed to submit feedback. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("feedback_created", map[string]interface{}{
			"id":    feedback.ID,
			"type":  feedback.Type,
			"email": feedback.Email,
		})

		// Send email notification if support email and SMTP are configured
		go func() {
			branding, err := brandingSvc.Get()
			if err != nil {
				logStructuredError("feedback_email_branding_fetch_failed", map[string]interface{}{"error": err.Error()})
				return
			}
			if err := emailSvc.SendFeedbackNotification(branding, feedback); err != nil {
				logStructuredError("feedback_email_send_failed", map[string]interface{}{"error": err.Error()})
			}
		}()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      feedback.ID,
		}); err != nil {
			logStructuredError("feedback_response_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleFeedbackList handles GET /api/v1/admin/feedback (admin only)
func handleFeedbackList(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")

		requests, err := svc.List(status)
		if err != nil {
			logStructuredError("feedback_list_failed", map[string]interface{}{"error": err.Error()})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve feedback list. Please try again.", ApiErrorTypeServerError)
			return
		}

		if requests == nil {
			requests = []FeedbackRequest{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(requests); err != nil {
			logStructuredError("feedback_list_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleFeedbackGet handles GET /api/v1/admin/feedback/{id} (admin only)
func handleFeedbackGet(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid feedback ID.", ApiErrorTypeValidation)
			return
		}

		feedback, err := svc.GetByID(id)
		if err != nil {
			logStructuredError("feedback_get_failed", map[string]interface{}{"id": id, "error": err.Error()})
			writeJSONError(w, http.StatusNotFound, "Feedback not found.", ApiErrorTypeNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(feedback); err != nil {
			logStructuredError("feedback_get_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleFeedbackUpdateStatus handles PATCH /api/v1/admin/feedback/{id}/status (admin only)
func handleFeedbackUpdateStatus(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid feedback ID.", ApiErrorTypeValidation)
			return
		}

		var input struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request format.", ApiErrorTypeValidation)
			return
		}

		// Validate status
		validStatuses := map[string]bool{
			"pending":     true,
			"in_progress": true,
			"resolved":    true,
			"rejected":    true,
		}
		if !validStatuses[input.Status] {
			writeJSONError(w, http.StatusBadRequest, "Invalid status. Use: pending, in_progress, resolved, or rejected.", ApiErrorTypeValidation)
			return
		}

		feedback, err := svc.UpdateStatus(id, input.Status)
		if err != nil {
			logStructuredError("feedback_update_status_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to update feedback status. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("feedback_status_updated", map[string]interface{}{
			"id":     id,
			"status": input.Status,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(feedback); err != nil {
			logStructuredError("feedback_status_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleFeedbackDelete handles DELETE /api/v1/admin/feedback/{id} (admin only)
func handleFeedbackDelete(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid feedback ID.", ApiErrorTypeValidation)
			return
		}

		if err := svc.Delete(id); err != nil {
			logStructuredError("feedback_delete_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete feedback. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("feedback_deleted", map[string]interface{}{
			"id": id,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      id,
		}); err != nil {
			logStructuredError("feedback_delete_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

// handleFeedbackDeleteBulk handles POST /api/v1/admin/feedback/bulk-delete (admin only)
func handleFeedbackDeleteBulk(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			IDs []int `json:"ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request format.", ApiErrorTypeValidation)
			return
		}

		if len(input.IDs) == 0 {
			writeJSONError(w, http.StatusBadRequest, "No feedback IDs provided.", ApiErrorTypeValidation)
			return
		}

		deleted, err := svc.DeleteBulk(input.IDs)
		if err != nil {
			logStructuredError("feedback_bulk_delete_failed", map[string]interface{}{
				"ids":   input.IDs,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete feedback items. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("feedback_bulk_deleted", map[string]interface{}{
			"ids":     input.IDs,
			"deleted": deleted,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"deleted": deleted,
		}); err != nil {
			logStructuredError("feedback_bulk_delete_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}
