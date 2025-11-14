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

	response := RequirementsResponse{
		EntityType: entityType,
		EntityName: entityName,
		UpdatedAt:  time.Now(),
		Groups:     groups,
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

	linkedTargets, unmatched := linkTargetsAndRequirements(targets, groups)

	response := OperationalTargetsResponse{
		EntityType:            entityType,
		EntityName:            entityName,
		Targets:               linkedTargets,
		UnmatchedRequirements: unmatched,
	}

	respondJSON(w, http.StatusOK, response)
}
