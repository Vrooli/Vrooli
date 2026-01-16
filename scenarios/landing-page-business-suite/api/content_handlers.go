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
			writeJSONError(w, http.StatusBadRequest, "Invalid variant ID", ApiErrorTypeValidation)
			return
		}

		sections, err := contentService.GetPublicSections(variantID)
		if err != nil {
			logStructuredError("public_sections_get_failed", map[string]interface{}{
				"variant_id": variantID,
				"error":      err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve sections. Please try again.", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"sections": sections,
		}); err != nil {
			logStructuredError("public_sections_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleGetSections retrieves all sections for a variant (OT-P0-012: CUSTOM-SPLIT)
func handleGetSections(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		variantIDStr := vars["variant_id"]

		variantID, err := strconv.ParseInt(variantIDStr, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid variant ID", ApiErrorTypeValidation)
			return
		}

		sections, err := contentService.GetSections(variantID)
		if err != nil {
			logStructuredError("sections_get_failed", map[string]interface{}{
				"variant_id": variantID,
				"error":      err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve sections. Please try again.", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"sections": sections,
		}); err != nil {
			logStructuredError("sections_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleGetSection retrieves a single section (OT-P0-012: CUSTOM-SPLIT)
func handleGetSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sectionIDStr := vars["id"]

		sectionID, err := strconv.ParseInt(sectionIDStr, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid section ID", ApiErrorTypeValidation)
			return
		}

		section, err := contentService.GetSection(sectionID)
		if err != nil {
			if err.Error() == "section not found" {
				writeJSONError(w, http.StatusNotFound, "Section not found", ApiErrorTypeNotFound)
				return
			}
			logStructuredError("section_get_failed", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve section. Please try again.", ApiErrorTypeServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(section); err != nil {
			logStructuredError("section_encode_failed", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
		}
	}
}

// handleUpdateSection updates a section's content (OT-P0-013: CUSTOM-LIVE)
func handleUpdateSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sectionIDStr := vars["id"]

		sectionID, err := strconv.ParseInt(sectionIDStr, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid section ID", ApiErrorTypeValidation)
			return
		}

		var payload struct {
			Content map[string]interface{} `json:"content"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if payload.Content == nil {
			writeJSONError(w, http.StatusBadRequest, "Content field is required", ApiErrorTypeValidation)
			return
		}

		if err := contentService.UpdateSection(sectionID, payload.Content); err != nil {
			if err.Error() == "section not found" {
				writeJSONError(w, http.StatusNotFound, "Section not found", ApiErrorTypeNotFound)
				return
			}
			logStructuredError("section_update_failed", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to update section. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("section_updated", map[string]interface{}{
			"section_id": sectionID,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Section updated successfully",
		}); err != nil {
			logStructuredError("section_update_response_encode_failed", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
		}
	}
}

// handleCreateSection creates a new section
func handleCreateSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var section ContentSection

		if err := json.NewDecoder(r.Body).Decode(&section); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid request body", ApiErrorTypeValidation)
			return
		}

		if section.VariantID == 0 {
			writeJSONError(w, http.StatusBadRequest, "variant_id is required", ApiErrorTypeValidation)
			return
		}
		if section.SectionType == "" {
			writeJSONError(w, http.StatusBadRequest, "section_type is required", ApiErrorTypeValidation)
			return
		}
		if section.Content == nil {
			writeJSONError(w, http.StatusBadRequest, "content is required", ApiErrorTypeValidation)
			return
		}

		created, err := contentService.CreateSection(section)
		if err != nil {
			logStructuredError("section_create_failed", map[string]interface{}{
				"variant_id":   section.VariantID,
				"section_type": section.SectionType,
				"error":        err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to create section. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("section_created", map[string]interface{}{
			"section_id":   created.ID,
			"variant_id":   created.VariantID,
			"section_type": created.SectionType,
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(created); err != nil {
			logStructuredError("section_create_response_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleDeleteSection deletes a section
func handleDeleteSection(contentService *ContentService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		sectionIDStr := vars["id"]

		sectionID, err := strconv.ParseInt(sectionIDStr, 10, 64)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid section ID", ApiErrorTypeValidation)
			return
		}

		if err := contentService.DeleteSection(sectionID); err != nil {
			if err.Error() == "section not found" {
				writeJSONError(w, http.StatusNotFound, "Section not found", ApiErrorTypeNotFound)
				return
			}
			logStructuredError("section_delete_failed", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
			writeJSONError(w, http.StatusInternalServerError, "Failed to delete section. Please try again.", ApiErrorTypeServerError)
			return
		}

		logStructured("section_deleted", map[string]interface{}{
			"section_id": sectionID,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Section deleted successfully",
		}); err != nil {
			logStructuredError("section_delete_response_encode_failed", map[string]interface{}{
				"section_id": sectionID,
				"error":      err.Error(),
			})
		}
	}
}
