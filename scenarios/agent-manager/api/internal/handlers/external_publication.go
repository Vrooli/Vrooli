package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/conversationsearch"
	"agent-manager/internal/orchestration"
)

// TombstoneExternalConversation revokes only the derived search projection;
// the owning scenario remains responsible for its canonical evidence. The
// durable tombstone is written before the asynchronous index notification so
// a crash cannot make a later repair resurrect the source.
func TombstoneExternalConversation(svc orchestration.RunService, db *database.DB, indexer *conversationsearch.Indexer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			SourceHarness   string `json:"sourceHarness"`
			SourceSessionID string `json:"sourceSessionId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || strings.TrimSpace(request.SourceHarness) == "" || strings.TrimSpace(request.SourceSessionID) == "" {
			http.Error(w, `{"error":"sourceHarness and sourceSessionId are required"}`, http.StatusBadRequest)
			return
		}
		if svc == nil || db == nil || indexer == nil {
			http.Error(w, `{"error":"external publication is unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		run, err := svc.GetRunByImportProvenance(r.Context(), request.SourceHarness, request.SourceSessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if run == nil {
			http.Error(w, `{"error":"external source was not imported"}`, http.StatusNotFound)
			return
		}
		if _, err := db.ExecContext(r.Context(), `INSERT INTO conversation_search_external_tombstones(source_harness, source_session_id, tombstoned_at) VALUES (?, ?, ?) ON CONFLICT(source_harness, source_session_id) DO UPDATE SET tombstoned_at=excluded.tombstoned_at`, request.SourceHarness, request.SourceSessionID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := indexer.Notify(r.Context(), conversationsearch.ChangeDeleteRun, run.ID.String(), ""); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tombstoned": true, "sourceHarness": request.SourceHarness, "sourceSessionId": request.SourceSessionID, "runId": run.ID.String()})
	}
}
