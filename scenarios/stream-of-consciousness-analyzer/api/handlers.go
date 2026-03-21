// DOC: docs/reference/api-endpoints.md
package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// --- Scheme Handlers ---

func handleListSchemes(svc SchemeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schemes, err := svc.List()
		if err != nil {
			classifyAndWriteError(w, err, "schemes")
			return
		}
		writeJSON(w, http.StatusOK, schemes)
	}
}

func handleCreateScheme(svc SchemeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input CreateSchemeInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
		scheme, err := svc.Create(&input)
		if err != nil {
			classifyAndWriteError(w, err, "scheme")
			return
		}
		writeJSON(w, http.StatusCreated, scheme)
	}
}

func handleGetScheme(svc SchemeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		scheme, err := svc.GetByID(id)
		if err != nil {
			classifyAndWriteError(w, err, "scheme")
			return
		}
		writeJSON(w, http.StatusOK, scheme)
	}
}

func handleUpdateScheme(svc SchemeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		var input UpdateSchemeInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
		if input.Name == "" {
			writeValidationError(w, "name is required and cannot be empty")
			return
		}
		scheme, err := svc.Update(id, &input)
		if err != nil {
			classifyAndWriteError(w, err, "scheme")
			return
		}
		writeJSON(w, http.StatusOK, scheme)
	}
}

func handleDeleteScheme(svc SchemeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		err := svc.Delete(id)
		if err != nil {
			classifyAndWriteError(w, err, "scheme")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Information Handlers ---

func handleListInformation(svc InformationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schemeID := mux.Vars(r)["schemeId"]
		items, err := svc.ListByScheme(schemeID)
		if err != nil {
			classifyAndWriteError(w, err, "information")
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func handleCreateInformation(svc InformationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schemeID := mux.Vars(r)["schemeId"]
		var input CreateInformationInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
		info, err := svc.Create(schemeID, &input)
		if err != nil {
			classifyAndWriteError(w, err, "information item")
			return
		}
		writeJSON(w, http.StatusCreated, info)
	}
}

func handleUpdateInformation(svc InformationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["infoId"]
		var input UpdateInformationInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
		info, err := svc.Update(id, &input)
		if err != nil {
			classifyAndWriteError(w, err, "information item")
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func handleDeleteInformation(svc InformationStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["infoId"]
		err := svc.Delete(id)
		if err != nil {
			classifyAndWriteError(w, err, "information item")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Thought Handlers ---

func handleListThoughts(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schemeID := r.URL.Query().Get("scheme_id")
		thoughts, err := svc.List(schemeID)
		if err != nil {
			classifyAndWriteError(w, err, "thoughts")
			return
		}
		writeJSON(w, http.StatusOK, thoughts)
	}
}

func handleCreateThought(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input CreateThoughtInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
		thought, err := svc.Create(&input)
		if err != nil {
			classifyAndWriteError(w, err, "thought")
			return
		}
		writeJSON(w, http.StatusCreated, thought)
	}
}

func handleGetThought(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		thought, err := svc.GetByID(id)
		if err != nil {
			classifyAndWriteError(w, err, "thought")
			return
		}
		writeJSON(w, http.StatusOK, thought)
	}
}

func handleUpdateThought(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		var input UpdateThoughtInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
		thought, err := svc.Update(id, &input)
		if err != nil {
			classifyAndWriteError(w, err, "thought")
			return
		}
		writeJSON(w, http.StatusOK, thought)
	}
}

func handleDeleteThought(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		err := svc.Delete(id)
		if err != nil {
			classifyAndWriteError(w, err, "thought")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Edge Handlers ---

// handleCreateEdge validates edge invariants before persisting:
//   - target_id must be non-empty (referential integrity)
//   - target_id != source_id (no self-loops — they break graph traversal)
//   - DB unique constraint on (source_id, target_id) prevents duplicates
func handleCreateEdge(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sourceID := mux.Vars(r)["id"]
		var input CreateEdgeInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
		if input.TargetID == "" {
			writeValidationError(w, "target_id is required")
			return
		}
		if input.TargetID == sourceID {
			writeValidationError(w, "cannot create an edge from a thought to itself")
			return
		}
		edge, err := svc.CreateEdge(sourceID, &input)
		if err != nil {
			classifyAndWriteError(w, err, "edge")
			return
		}
		writeJSON(w, http.StatusCreated, edge)
	}
}

func handleListEdges(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		thoughtID := mux.Vars(r)["id"]
		edges, err := svc.ListEdges(thoughtID)
		if err != nil {
			classifyAndWriteError(w, err, "edges")
			return
		}
		writeJSON(w, http.StatusOK, edges)
	}
}

func handleDeleteEdge(svc ThoughtStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		edgeID := mux.Vars(r)["edgeId"]
		err := svc.DeleteEdge(edgeID)
		if err != nil {
			classifyAndWriteError(w, err, "edge")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// --- Export Handlers ---

func handleExportScheme(svc ExportStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schemeID := mux.Vars(r)["id"]
		data, err := svc.ExportScheme(schemeID)
		if err != nil {
			classifyAndWriteError(w, err, "scheme")
			return
		}
		writeJSON(w, http.StatusOK, data)
	}
}

// --- Suggestion Handlers ---

func handleGetProviders(svc SuggestionProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, svc.GetProviders())
	}
}

func handleGenerateSuggestions(svc SuggestionProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schemeID := mux.Vars(r)["id"]
		suggestions, provider, err := svc.GenerateSuggestions(schemeID)
		if err != nil {
			writeAPIError(w, http.StatusServiceUnavailable, ErrCategoryDependency,
				"LLM provider is unavailable — suggestions will resume when a provider comes online", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"suggestions": suggestions,
			"provider":    provider.Name,
		})
	}
}
