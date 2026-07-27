package metricshttp

import (
	"net/http"
	"strings"

	metrics "landing-page-business-suite-api/internal/metrics"
)

// seam: FeedbackNotifier delivers feedback notifications after persistence.
// Production is wired by the root composition package; tests use
// mocks.FakeFeedbackNotifier.
type FeedbackNotifier interface {
	Notify(*metrics.FeedbackRequest)
}

func (d Dependencies) CreateFeedback(svc metrics.FeedbackServicer, notifier FeedbackNotifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input metrics.CreateFeedbackInput
		if !d.DecodeJSON(w, r, &input) {
			return
		}
		if strings.TrimSpace(input.Email) == "" {
			d.WriteErrorType(w, http.StatusBadRequest, "Email address is required.", "validation")
			return
		}
		if strings.TrimSpace(input.Subject) == "" {
			d.WriteErrorType(w, http.StatusBadRequest, "Subject is required.", "validation")
			return
		}
		if strings.TrimSpace(input.Message) == "" {
			d.WriteErrorType(w, http.StatusBadRequest, "Message is required.", "validation")
			return
		}
		if input.Type != "refund" && input.Type != "bug" && input.Type != "feature" && input.Type != "general" {
			input.Type = "general"
		}
		feedback, err := svc.Create(r.Context(), &input)
		if err != nil {
			d.LogError("feedback_create_failed", map[string]interface{}{"error": err.Error()})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to submit feedback. Please try again.", "server_error")
			return
		}
		d.LogError("feedback_created", map[string]interface{}{"id": feedback.ID, "type": feedback.Type, "email": feedback.Email})
		notifier.Notify(feedback)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := d.WriteJSON(w, map[string]interface{}{"success": true, "id": feedback.ID}); err != nil {
			d.LogError("feedback_response_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func (d Dependencies) ListFeedback(svc metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requests, err := svc.List(r.Context(), r.URL.Query().Get("status"))
		if err != nil {
			d.LogError("feedback_list_failed", map[string]interface{}{"error": err.Error()})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to retrieve feedback list. Please try again.", "server_error")
			return
		}
		if requests == nil {
			requests = []metrics.FeedbackRequest{}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := d.WriteJSON(w, requests); err != nil {
			d.LogError("feedback_list_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func (d Dependencies) GetFeedback(svc metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := d.PathInt(w, r, "id")
		if !ok {
			return
		}
		feedback, err := svc.GetByID(r.Context(), id)
		if err != nil {
			d.LogError("feedback_get_failed", map[string]interface{}{"id": id, "error": err.Error()})
			d.WriteErrorType(w, http.StatusNotFound, "Feedback not found.", "not_found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := d.WriteJSON(w, feedback); err != nil {
			d.LogError("feedback_get_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func (d Dependencies) UpdateFeedbackStatus(svc metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := d.PathInt(w, r, "id")
		if !ok {
			return
		}
		var input struct {
			Status string `json:"status"`
		}
		if !d.DecodeJSON(w, r, &input) {
			return
		}
		if !validFeedbackStatus(input.Status) {
			d.WriteErrorType(w, http.StatusBadRequest, "Invalid status. Use: pending, in_progress, resolved, or rejected.", "validation")
			return
		}
		feedback, err := svc.UpdateStatus(r.Context(), id, input.Status)
		if err != nil {
			d.LogError("feedback_update_status_failed", map[string]interface{}{"id": id, "error": err.Error()})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to update feedback status. Please try again.", "server_error")
			return
		}
		d.LogError("feedback_status_updated", map[string]interface{}{"id": id, "status": input.Status})
		w.Header().Set("Content-Type", "application/json")
		if err := d.WriteJSON(w, feedback); err != nil {
			d.LogError("feedback_status_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func (d Dependencies) DeleteFeedback(svc metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := d.PathInt(w, r, "id")
		if !ok {
			return
		}
		if err := svc.Delete(r.Context(), id); err != nil {
			d.LogError("feedback_delete_failed", map[string]interface{}{"id": id, "error": err.Error()})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to delete feedback. Please try again.", "server_error")
			return
		}
		d.LogError("feedback_deleted", map[string]interface{}{"id": id})
		w.Header().Set("Content-Type", "application/json")
		if err := d.WriteJSON(w, map[string]interface{}{"success": true, "id": id}); err != nil {
			d.LogError("feedback_delete_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func (d Dependencies) DeleteFeedbackBulk(svc metrics.FeedbackServicer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			IDs []int `json:"ids"`
		}
		if !d.DecodeJSON(w, r, &input) {
			return
		}
		if len(input.IDs) == 0 {
			d.WriteErrorType(w, http.StatusBadRequest, "No feedback IDs provided.", "validation")
			return
		}
		deleted, err := svc.DeleteBulk(r.Context(), input.IDs)
		if err != nil {
			d.LogError("feedback_bulk_delete_failed", map[string]interface{}{"ids": input.IDs, "error": err.Error()})
			d.WriteErrorType(w, http.StatusInternalServerError, "Failed to delete feedback items. Please try again.", "server_error")
			return
		}
		d.LogError("feedback_bulk_deleted", map[string]interface{}{"ids": input.IDs, "deleted": deleted})
		w.Header().Set("Content-Type", "application/json")
		if err := d.WriteJSON(w, map[string]interface{}{"success": true, "deleted": deleted}); err != nil {
			d.LogError("feedback_bulk_delete_encode_failed", map[string]interface{}{"error": err.Error()})
		}
	}
}

func validFeedbackStatus(status string) bool {
	return status == "pending" || status == "in_progress" || status == "resolved" || status == "rejected"
}
