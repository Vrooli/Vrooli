package main

import (
	"encoding/json"
	"net/http"
)

// getProtectedScenariosHandler returns the list of protected scenarios
func getProtectedScenariosHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Fetching protected scenarios")

	scenarios := protectedScenariosStore.GetProtectedScenarios()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]any{
		"success":            true,
		"protected_scenarios": scenarios,
		"count":              len(scenarios),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode protected scenarios response", err)
	}
}

// updateProtectedScenariosHandler updates the list of protected scenarios
func updateProtectedScenariosHandler(w http.ResponseWriter, r *http.Request) {
	logger.Info("Updating protected scenarios")

	var req struct {
		Scenarios []string `json:"scenarios"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		HTTPError(w, "Invalid request body", http.StatusBadRequest, err)
		return
	}

	// Validate that scenarios is not nil (empty array is fine)
	if req.Scenarios == nil {
		HTTPError(w, "scenarios field is required", http.StatusBadRequest, nil)
		return
	}

	// Update the protected scenarios
	if err := protectedScenariosStore.SetProtectedScenarios(req.Scenarios); err != nil {
		HTTPError(w, "Failed to update protected scenarios", http.StatusInternalServerError, err)
		return
	}

	logger.Info("Successfully updated protected scenarios")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]any{
		"success": true,
		"message": "Protected scenarios updated successfully",
		"count":   len(req.Scenarios),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode update response", err)
	}
}
