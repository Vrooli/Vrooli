package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func handleListBacklog(w http.ResponseWriter, r *http.Request) {
	entries, err := fetchAllBacklogEntries()
	if err != nil {
		respondInternalError(w, "Failed to list backlog entries", err)
		return
	}

	respondJSON(w, http.StatusOK, BacklogListResponse{Entries: entries, Total: len(entries)})
}

func handleCreateBacklogEntries(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	var req BacklogCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	entries := req.Entries
	if req.RawInput != "" {
		parsed := parseBacklogInput(req.RawInput, req.EntityType)
		entries = append(entries, parsed...)
	}

	if len(entries) == 0 {
		respondBadRequest(w, "No backlog items provided")
		return
	}

	if len(entries) > 50 {
		respondBadRequest(w, "Too many backlog items in a single request (max 50)")
		return
	}

	created, err := insertBacklogEntries(entries)
	if err != nil {
		respondInternalError(w, "Failed to create backlog entries", err)
		return
	}

	if len(created) == 0 {
		respondBadRequest(w, "No valid backlog entries could be created")
		return
	}

	respondJSON(w, http.StatusCreated, BacklogCreateResponse{Entries: created})
}

func handleDeleteBacklogEntry(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	entryID := mux.Vars(r)["id"]
	if entryID == "" {
		respondBadRequest(w, "Backlog entry id required")
		return
	}

	result, err := db.Exec(`DELETE FROM backlog_entries WHERE id = $1`, entryID)
	if err != nil {
		respondInternalError(w, "Failed to delete backlog entry", err)
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		respondNotFound(w, "Backlog entry")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleConvertSingleBacklogEntry(w http.ResponseWriter, r *http.Request) {
	entryID := mux.Vars(r)["id"]
	if entryID == "" {
		respondBadRequest(w, "Backlog entry id required")
		return
	}

	results, err := convertBacklogEntries([]string{entryID})
	if err != nil {
		respondInternalError(w, "Failed to convert backlog entry", err)
		return
	}

	if len(results) == 0 {
		respondNotFound(w, "Backlog entry")
		return
	}

	status := http.StatusOK
	if results[0].Error != "" {
		status = http.StatusConflict
	}

	respondJSON(w, status, results[0])
}

func handleConvertBacklogEntries(w http.ResponseWriter, r *http.Request) {
	var req BacklogConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	if len(req.EntryIDs) == 0 {
		respondBadRequest(w, "entry_ids is required")
		return
	}

	results, err := convertBacklogEntries(req.EntryIDs)
	if err != nil {
		respondInternalError(w, "Failed to convert backlog entries", err)
		return
	}

	respondJSON(w, http.StatusOK, BacklogConvertResponse{Results: results})
}
