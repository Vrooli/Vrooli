package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// BatchQueueRequest represents a batch operation on queue items
type BatchQueueRequest struct {
	Action string   `json:"action"` // "approve", "reject", "delete"
	IDs    []string `json:"ids"`
}

// BatchQueueResponse represents the result of a batch operation
type BatchQueueResponse struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
	Total     int      `json:"total"`
}

// batchQueueHandler handles batch operations on improvement queue items
func batchQueueHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method not allowed"})
		return
	}

	var req BatchQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	if len(req.IDs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "No IDs provided"})
		return
	}

	resp := BatchQueueResponse{
		Succeeded: []string{},
		Failed:    []string{},
		Total:     len(req.IDs),
	}

	switch req.Action {
	case "approve":
		resp = batchApproveQueueItems(req.IDs)
	case "reject":
		resp = batchRejectQueueItems(req.IDs)
	case "delete":
		resp = batchDeleteQueueItems(req.IDs)
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid action. Must be 'approve', 'reject', or 'delete'"})
		return
	}

	// Publish batch operation event
	publishEvent(EventQueueItemUpdated, map[string]interface{}{
		"action":     req.Action,
		"succeeded":  len(resp.Succeeded),
		"failed":     len(resp.Failed),
		"total":      resp.Total,
	})

	json.NewEncoder(w).Encode(resp)
}

// batchApproveQueueItems approves multiple queue items
func batchApproveQueueItems(ids []string) BatchQueueResponse {
	resp := BatchQueueResponse{
		Succeeded: []string{},
		Failed:    []string{},
		Total:     len(ids),
	}

	query := `UPDATE improvement_queue SET status = 'approved', reviewed_at = NOW() WHERE id = $1`

	for _, id := range ids {
		result, err := db.Exec(query, id)
		if err != nil {
			log.Printf("Error approving queue item %s: %v", id, err)
			resp.Failed = append(resp.Failed, id)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("Queue item %s not found", id)
			resp.Failed = append(resp.Failed, id)
			continue
		}

		resp.Succeeded = append(resp.Succeeded, id)
	}

	return resp
}

// batchRejectQueueItems rejects multiple queue items
func batchRejectQueueItems(ids []string) BatchQueueResponse {
	resp := BatchQueueResponse{
		Succeeded: []string{},
		Failed:    []string{},
		Total:     len(ids),
	}

	query := `UPDATE improvement_queue SET status = 'denied', reviewed_at = NOW() WHERE id = $1`

	for _, id := range ids {
		result, err := db.Exec(query, id)
		if err != nil {
			log.Printf("Error rejecting queue item %s: %v", id, err)
			resp.Failed = append(resp.Failed, id)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("Queue item %s not found", id)
			resp.Failed = append(resp.Failed, id)
			continue
		}

		resp.Succeeded = append(resp.Succeeded, id)
	}

	return resp
}

// batchDeleteQueueItems deletes multiple queue items
func batchDeleteQueueItems(ids []string) BatchQueueResponse {
	resp := BatchQueueResponse{
		Succeeded: []string{},
		Failed:    []string{},
		Total:     len(ids),
	}

	query := `DELETE FROM improvement_queue WHERE id = $1`

	for _, id := range ids {
		result, err := db.Exec(query, id)
		if err != nil {
			log.Printf("Error deleting queue item %s: %v", id, err)
			resp.Failed = append(resp.Failed, id)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("Queue item %s not found", id)
			resp.Failed = append(resp.Failed, id)
			continue
		}

		resp.Succeeded = append(resp.Succeeded, id)
	}

	return resp
}
