package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type DeleteRecordResponse struct {
	RecordID   string `json:"record_id"`
	Collection string `json:"collection"`
	Deleted    bool   `json:"deleted"`
	TookMS     int64  `json:"took_ms"`
}

func (s *Server) handleDeleteRecord(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	vars := mux.Vars(r)
	recordID := strings.TrimSpace(vars["record_id"])
	if recordID == "" {
		s.respondError(w, http.StatusBadRequest, "record_id is required")
		return
	}

	if s == nil || s.ingestService == nil {
		s.respondError(w, http.StatusInternalServerError, "Ingest service unavailable")
		return
	}

	collection, err := s.ingestService.DeleteRecord(r.Context(), recordID)
	if err != nil {
		s.log("delete failed", map[string]interface{}{"error": err.Error(), "record_id": recordID})
		s.respondError(w, http.StatusInternalServerError, "Failed to delete record")
		return
	}

	resp := DeleteRecordResponse{
		RecordID:   recordID,
		Collection: collection,
		Deleted:    true,
		TookMS:     time.Since(start).Milliseconds(),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

