package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// handleFeedbackCreate handles POST /api/v1/feedback (public endpoint)
func handleFeedbackCreate(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input CreateFeedbackInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}

		// Validate required fields
		if strings.TrimSpace(input.Email) == "" {
			http.Error(w, `{"error":"email is required"}`, http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(input.Subject) == "" {
			http.Error(w, `{"error":"subject is required"}`, http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(input.Message) == "" {
			http.Error(w, `{"error":"message is required"}`, http.StatusBadRequest)
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
			http.Error(w, `{"error":"failed to submit feedback"}`, http.StatusInternalServerError)
			return
		}

		logStructured("feedback_created", map[string]interface{}{
			"id":    feedback.ID,
			"type":  feedback.Type,
			"email": feedback.Email,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"id":      feedback.ID,
		})
	}
}

// handleFeedbackList handles GET /api/v1/admin/feedback (admin only)
func handleFeedbackList(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := r.URL.Query().Get("status")

		requests, err := svc.List(status)
		if err != nil {
			logStructuredError("feedback_list_failed", map[string]interface{}{"error": err.Error()})
			http.Error(w, `{"error":"failed to list feedback"}`, http.StatusInternalServerError)
			return
		}

		if requests == nil {
			requests = []FeedbackRequest{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(requests)
	}
}

// handleFeedbackGet handles GET /api/v1/admin/feedback/{id} (admin only)
func handleFeedbackGet(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, `{"error":"invalid feedback id"}`, http.StatusBadRequest)
			return
		}

		feedback, err := svc.GetByID(id)
		if err != nil {
			http.Error(w, `{"error":"feedback not found"}`, http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(feedback)
	}
}

// handleFeedbackUpdateStatus handles PATCH /api/v1/admin/feedback/{id}/status (admin only)
func handleFeedbackUpdateStatus(svc *FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			http.Error(w, `{"error":"invalid feedback id"}`, http.StatusBadRequest)
			return
		}

		var input struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
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
			http.Error(w, `{"error":"invalid status"}`, http.StatusBadRequest)
			return
		}

		feedback, err := svc.UpdateStatus(id, input.Status)
		if err != nil {
			logStructuredError("feedback_update_status_failed", map[string]interface{}{
				"id":    id,
				"error": err.Error(),
			})
			http.Error(w, `{"error":"failed to update feedback status"}`, http.StatusInternalServerError)
			return
		}

		logStructured("feedback_status_updated", map[string]interface{}{
			"id":     id,
			"status": input.Status,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(feedback)
	}
}
