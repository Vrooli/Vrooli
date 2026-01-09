package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

func handleListBacklog(w http.ResponseWriter, r *http.Request) {
	// Sync filesystem backlog with database
	if db != nil {
		if err := syncBacklogFilesystemWithDatabase(db); err != nil {
			slog.Warn("Failed to sync filesystem backlog", "error", err)
		}
	}

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

	// Delete from filesystem (git-backed persistence)
	if err := deleteBacklogFile(entryID); err != nil {
		slog.Warn("Failed to delete backlog file (database deleted successfully)", "id", entryID, "error", err)
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

func handleUpdateBacklogEntry(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		respondServiceUnavailable(w, "Database not available")
		return
	}

	entryID := mux.Vars(r)["id"]
	if entryID == "" {
		respondBadRequest(w, "Backlog entry id required")
		return
	}

	var req BacklogUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondInvalidJSON(w, err)
		return
	}

	if req.Notes == nil {
		respondBadRequest(w, "At least one field is required to update a backlog entry")
		return
	}

	entry, err := updateBacklogEntry(entryID, req)
	if err != nil {
		if errors.Is(err, errBacklogEntryNotFound) {
			respondNotFound(w, "Backlog entry")
			return
		}
		respondInternalError(w, "Failed to update backlog entry", err)
		return
	}

	respondJSON(w, http.StatusOK, entry)
}
