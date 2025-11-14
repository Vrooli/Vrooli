package main

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func handleGetRequirements(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	groups, err := loadRequirementsForEntity(entityType, entityName)
	if err != nil {
		respondInternalError(w, "Failed to load requirements", err)
		return
	}

	// Enrich requirements with test file references and PRD validation
	enrichedGroups := enrichRequirementsWithTestsAndValidation(entityType, entityName, groups)

	response := RequirementsResponse{
		EntityType: entityType,
		EntityName: entityName,
		UpdatedAt:  time.Now(),
		Groups:     enrichedGroups,
	}

	respondJSON(w, http.StatusOK, response)
}

func handleGetOperationalTargets(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entityType := vars["type"]
	entityName := vars["name"]

	if !isValidEntityType(entityType) {
		respondInvalidEntityType(w)
		return
	}

	targets, err := extractOperationalTargets(entityType, entityName)
	if err != nil {
		respondInternalError(w, "Failed to parse operational targets", err)
		return
	}

	groups, err := loadRequirementsForEntity(entityType, entityName)
	if err != nil {
		respondInternalError(w, "Failed to load requirements", err)
		return
	}

	// Enrich requirements with test files and PRD validation first
	enrichedGroups := enrichRequirementsWithTestsAndValidation(entityType, entityName, groups)

	// Link targets and requirements bidirectionally
	linkedTargets, enrichedUnmatched := linkTargetsAndRequirements(targets, enrichedGroups)

	response := OperationalTargetsResponse{
		EntityType:            entityType,
		EntityName:            entityName,
		Targets:               linkedTargets,
		UnmatchedRequirements: enrichedUnmatched,
	}

	respondJSON(w, http.StatusOK, response)
}
