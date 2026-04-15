package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// NOTE: Handlers that used database-backed ContentService have been removed.
// Content sections are stored in tracked config JSON files and accessed via ConfigStore.

// handleGetPublicSectionsFromConfigStore retrieves enabled sections from ConfigStore (no auth required)
func handleGetPublicSectionsFromConfigStore(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slug := vars["variant_slug"]

		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug is required", ApiErrorTypeValidation)
			return
		}

		variant, err := cs.GetVariant(slug)
		if err != nil {
			logStructuredError("public_sections_get_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusNotFound, "Variant not found", ApiErrorTypeNotFound)
			return
		}

		// Filter to enabled sections only
		enabledSections := make([]VariantSection, 0)
		for _, section := range variant.Sections {
			if section.Enabled {
				enabledSections = append(enabledSections, section)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"sections": enabledSections,
		}); err != nil {
			logStructuredError("public_sections_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}

// handleGetSectionsFromConfigStore retrieves all sections from ConfigStore (admin endpoint)
func handleGetSectionsFromConfigStore(cs *ConfigStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		slug := vars["variant_slug"]

		if slug == "" {
			writeJSONError(w, http.StatusBadRequest, "Variant slug is required", ApiErrorTypeValidation)
			return
		}

		variant, err := cs.GetVariant(slug)
		if err != nil {
			logStructuredError("sections_get_failed", map[string]interface{}{
				"slug":  slug,
				"error": err.Error(),
			})
			writeJSONError(w, http.StatusNotFound, "Variant not found", ApiErrorTypeNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"sections": variant.Sections,
		}); err != nil {
			logStructuredError("sections_encode_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}
