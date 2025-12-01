package main

import (
	"database/sql"

	"github.com/gorilla/mux"
)

// APIServer centralizes the HTTP surface for the scenario so routes reflect the
// core domains: health, vault coverage, security scanning, resources, and deployment.
type APIServer struct {
	db                 *sql.DB
	validator          *SecretValidator
	orientationBuilder *OrientationBuilder
}

func newAPIServer(db *sql.DB, logger *Logger) *APIServer {
	server := &APIServer{db: db}
	if db != nil {
		server.validator = NewSecretValidator(db)
		server.orientationBuilder = NewOrientationBuilder(db, logger)
	}
	return server
}

// routes wires HTTP endpoints by capability to make the surface area obvious.
func (s *APIServer) routes() *mux.Router {
	r := mux.NewRouter()

	// Health
	r.HandleFunc("/health", s.healthHandler).Methods("GET")

	api := r.PathPrefix("/api/v1").Subrouter()

	// Health (API-prefixed for UI/iframe clients)
	api.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Vault coverage and provisioning
	api.HandleFunc("/vault/secrets/status", s.vaultSecretsStatusHandler).Methods("GET")
	api.HandleFunc("/vault/secrets/provision", s.vaultProvisionHandler).Methods("POST")

	// Security intelligence
	api.HandleFunc("/security/scan", s.securityScanHandler).Methods("GET")
	api.HandleFunc("/security/compliance", s.complianceHandler).Methods("GET")
	api.HandleFunc("/security/vulnerabilities", s.vulnerabilitiesHandler).Methods("GET")
	api.HandleFunc("/vulnerabilities", s.vulnerabilitiesHandler).Methods("GET") // Legacy route for backward compatibility
	api.HandleFunc("/vulnerabilities/fix", s.fixVulnerabilitiesHandler).Methods("POST")
	api.HandleFunc("/vulnerabilities/fix/progress", s.fixProgressHandler).Methods("POST")
	api.HandleFunc("/vulnerabilities/{id}/status", s.vulnerabilityStatusHandler).Methods("POST")
	api.HandleFunc("/files/content", s.fileContentHandler).Methods("GET")
	api.HandleFunc("/orientation/summary", s.orientationSummaryHandler).Methods("GET")

	// Resource intelligence
	api.HandleFunc("/resources/{resource}", s.resourceDetailHandler).Methods("GET")
	api.HandleFunc("/resources/{resource}/secrets/{secret}", s.resourceSecretUpdateHandler).Methods("PATCH")
	api.HandleFunc("/resources/{resource}/secrets/{secret}/strategy", s.secretStrategyHandler).Methods("POST")

	// Legacy secrets endpoints (kept for clients already depending on them)
	api.HandleFunc("/secrets/scan", s.vaultSecretsStatusHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/validate", s.validateHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/provision", s.provisionHandler).Methods("POST")

	// Deployment manifest and bundle consumers
	api.HandleFunc("/deployment/secrets", s.deploymentSecretsHandler).Methods("POST")
	api.HandleFunc("/deployment/secrets/{scenario}", s.deploymentSecretsGetHandler).Methods("GET")

	// Scenario intelligence (fast list for UI selection)
	api.HandleFunc("/scenarios", s.scenarioListHandler).Methods("GET")

	return r
}
