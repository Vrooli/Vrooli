package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// handleGetPublicSections retrieves enabled sections for public display (no auth required)
func handleGetPublicSections(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		variantIDStr := vars["variant_id"]

		variantID, err := strconv.ParseInt(variantIDStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error": "Invalid variant ID"}`, http.StatusBadRequest)
			return
		}

		sections, err := contentService.GetPublicSections(variantID)
		if err != nil {
			logStructuredError("Failed to get public sections", map[string]interface{}{
				"variant_id": variantID,
				"error":      err.Error(),
			})
			http.Error(w, `{"error": "Failed to retrieve sections"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sections": sections,
		})
	}
}

// handleGetSections retrieves all sections for a variant (OT-P0-012: CUSTOM-SPLIT)
func handleGetSections(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		variantIDStr := vars["variant_id"]

		variantID, err := strconv.ParseInt(variantIDStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error": "Invalid variant ID"}`, http.StatusBadRequest)
			return
		}

		sections, err := contentService.GetSections(variantID)
		if err != nil {
			logStructuredError("Failed to get sections", map[string]interface{}{
				"variant_id": variantID,
				"error":      err.Error(),
			})
			http.Error(w, `{"error": "Failed to retrieve sections"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sections": sections,
		})
	}
}

// handleGetSection retrieves a single section (OT-P0-012: CUSTOM-SPLIT)
func handleGetSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sectionIDStr := vars["id"]

		sectionID, err := strconv.ParseInt(sectionIDStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error": "Invalid section ID"}`, http.StatusBadRequest)
			return
		}

		section, err := contentService.GetSection(sectionID)
		if err != nil {
			if err.Error() == "section not found" {
				http.Error(w, `{"error": "Section not found"}`, http.StatusNotFound)
				return
			}
			logStructuredError("Failed to get section", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
			http.Error(w, `{"error": "Failed to retrieve section"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(section)
	}
}

// handleUpdateSection updates a section's content (OT-P0-013: CUSTOM-LIVE)
func handleUpdateSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sectionIDStr := vars["id"]

		sectionID, err := strconv.ParseInt(sectionIDStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error": "Invalid section ID"}`, http.StatusBadRequest)
			return
		}

		var payload struct {
			Content map[string]interface{} `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		if payload.Content == nil {
			http.Error(w, `{"error": "Content field is required"}`, http.StatusBadRequest)
			return
		}

		if err := contentService.UpdateSection(sectionID, payload.Content); err != nil {
			if err.Error() == "section not found" {
				http.Error(w, `{"error": "Section not found"}`, http.StatusNotFound)
				return
			}
			logStructuredError("Failed to update section", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
			http.Error(w, `{"error": "Failed to update section"}`, http.StatusInternalServerError)
			return
		}

		logStructured("Section updated", map[string]interface{}{
			"section_id": sectionID,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Section updated successfully",
		})
	}
}

// handleCreateSection creates a new section
func handleCreateSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var section ContentSection

		if err := json.NewDecoder(r.Body).Decode(&section); err != nil {
			http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
			return
		}

		if section.VariantID == 0 {
			http.Error(w, `{"error": "variant_id is required"}`, http.StatusBadRequest)
			return
		}
		if section.SectionType == "" {
			http.Error(w, `{"error": "section_type is required"}`, http.StatusBadRequest)
			return
		}
		if section.Content == nil {
			http.Error(w, `{"error": "content is required"}`, http.StatusBadRequest)
			return
		}

		created, err := contentService.CreateSection(section)
		if err != nil {
			logStructuredError("Failed to create section", map[string]interface{}{
				"variant_id":   section.VariantID,
				"section_type": section.SectionType,
				"error":        err.Error(),
			})
			http.Error(w, `{"error": "Failed to create section"}`, http.StatusInternalServerError)
			return
		}

		logStructured("Section created", map[string]interface{}{
			"section_id":   created.ID,
			"variant_id":   created.VariantID,
			"section_type": created.SectionType,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}
}

// handleDeleteSection deletes a section
func handleDeleteSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sectionIDStr := vars["id"]

		sectionID, err := strconv.ParseInt(sectionIDStr, 10, 64)
		if err != nil {
			http.Error(w, `{"error": "Invalid section ID"}`, http.StatusBadRequest)
			return
		}

		if err := contentService.DeleteSection(sectionID); err != nil {
			if err.Error() == "section not found" {
				http.Error(w, `{"error": "Section not found"}`, http.StatusNotFound)
				return
			}
			logStructuredError("Failed to delete section", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
			http.Error(w, `{"error": "Failed to delete section"}`, http.StatusInternalServerError)
			return
		}

		logStructured("Section deleted", map[string]interface{}{
			"section_id": sectionID,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Section deleted successfully",
		})
	}
}
