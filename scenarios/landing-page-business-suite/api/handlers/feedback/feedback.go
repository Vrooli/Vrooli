// Package feedback owns HTTP transport for the feedback domain.
package feedback

import (
	"net/http"
	"strings"

	metrics "landing-page-business-suite-api/internal/metrics"
)

type Dependencies struct {
	DecodeJSON     func(http.ResponseWriter, *http.Request, interface{}) bool
	PathInt        func(http.ResponseWriter, *http.Request, string) (int, bool)
	WriteErrorType func(http.ResponseWriter, int, string, string)
	WriteJSON      func(http.ResponseWriter, interface{}) error
	LogError       func(string, map[string]interface{})
}

// Notifier delivers feedback notifications after persistence. Production is
// wired by the root composition package; tests use a small fake.
type Notifier interface {
	Notify(*metrics.FeedbackRequest)
}

func Create(dependencies Dependencies, service metrics.FeedbackServicer, notifier Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input metrics.CreateFeedbackInput
		if !dependencies.DecodeJSON(w, r, &input) {
			return
		}
		if strings.TrimSpace(input.Email) == "" {
			dependencies.WriteErrorType(w, http.StatusBadRequest, "Email address is required.", "validation")
			return
		}
		if strings.TrimSpace(input.Subject) == "" {
			dependencies.WriteErrorType(w, http.StatusBadRequest, "Subject is required.", "validation")
			return
		}
		if strings.TrimSpace(input.Message) == "" {
			dependencies.WriteErrorType(w, http.StatusBadRequest, "Message is required.", "validation")
			return
		}
		if !validType(input.Type) {
			input.Type = "general"
		}
		request, err := service.Create(r.Context(), &input)
		if err != nil {
			dependencies.LogError("feedback_create_failed", map[string]interface{}{"error": err.Error()})
			dependencies.WriteErrorType(w, http.StatusInternalServerError, "Failed to submit feedback. Please try again.", "server_error")
			return
		}
		dependencies.LogError("feedback_created", map[string]interface{}{"id": request.ID, "type": request.Type, "email": request.Email})
		notifier.Notify(request)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := dependencies.WriteJSON(w, map[string]interface{}{"success": true, "id": request.ID}); err != nil {
			dependencies.LogError("feedback_response_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func List(dependencies Dependencies, service metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requests, err := service.List(r.Context(), r.URL.Query().Get("status"))
		if err != nil {
			dependencies.LogError("feedback_list_failed", map[string]interface{}{"error": err.Error()})
			dependencies.WriteErrorType(w, http.StatusInternalServerError, "Failed to retrieve feedback list. Please try again.", "server_error")
			return
		}
		if requests == nil {
			requests = []metrics.FeedbackRequest{}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := dependencies.WriteJSON(w, requests); err != nil {
			dependencies.LogError("feedback_list_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func Get(dependencies Dependencies, service metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := dependencies.PathInt(w, r, "id")
		if !ok {
			return
		}
		request, err := service.GetByID(r.Context(), id)
		if err != nil {
			dependencies.LogError("feedback_get_failed", map[string]interface{}{"id": id, "error": err.Error()})
			dependencies.WriteErrorType(w, http.StatusNotFound, "Feedback not found.", "not_found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := dependencies.WriteJSON(w, request); err != nil {
			dependencies.LogError("feedback_get_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func UpdateStatus(dependencies Dependencies, service metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := dependencies.PathInt(w, r, "id")
		if !ok {
			return
		}
		var input struct {
			Status string `json:"status"`
		}
		if !dependencies.DecodeJSON(w, r, &input) {
			return
		}
		if !validStatus(input.Status) {
			dependencies.WriteErrorType(w, http.StatusBadRequest, "Invalid status. Use: pending, in_progress, resolved, or rejected.", "validation")
			return
		}
		request, err := service.UpdateStatus(r.Context(), id, input.Status)
		if err != nil {
			dependencies.LogError("feedback_update_status_failed", map[string]interface{}{"id": id, "error": err.Error()})
			dependencies.WriteErrorType(w, http.StatusInternalServerError, "Failed to update feedback status. Please try again.", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := dependencies.WriteJSON(w, request); err != nil {
			dependencies.LogError("feedback_status_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func Delete(dependencies Dependencies, service metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := dependencies.PathInt(w, r, "id")
		if !ok {
			return
		}
		if err := service.Delete(r.Context(), id); err != nil {
			dependencies.LogError("feedback_delete_failed", map[string]interface{}{"id": id, "error": err.Error()})
			dependencies.WriteErrorType(w, http.StatusInternalServerError, "Failed to delete feedback. Please try again.", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := dependencies.WriteJSON(w, map[string]interface{}{"success": true, "id": id}); err != nil {
			dependencies.LogError("feedback_delete_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func DeleteBulk(dependencies Dependencies, service metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			IDs []int `json:"ids"`
		}
		if !dependencies.DecodeJSON(w, r, &input) {
			return
		}
		if len(input.IDs) == 0 {
			dependencies.WriteErrorType(w, http.StatusBadRequest, "No feedback IDs provided.", "validation")
			return
		}
		deleted, err := service.DeleteBulk(r.Context(), input.IDs)
		if err != nil {
			dependencies.LogError("feedback_bulk_delete_failed", map[string]interface{}{"ids": input.IDs, "error": err.Error()})
			dependencies.WriteErrorType(w, http.StatusInternalServerError, "Failed to delete feedback items. Please try again.", "server_error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := dependencies.WriteJSON(w, map[string]interface{}{"success": true, "deleted": deleted}); err != nil {
			dependencies.LogError("feedback_bulk_delete_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func validType(value string) bool {
	return value == "refund" || value == "bug" || value == "feature" || value == "general"
}

func validStatus(value string) bool {
	return value == "pending" || value == "in_progress" || value == "resolved" || value == "rejected"
}
