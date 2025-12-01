package main

import (
	"database/sql"

	"github.com/gorilla/mux"
)

// APIServer centralizes the HTTP surface for the scenario so routes reflect the
// core domains: health, vault coverage, security scanning, resources, and deployment.
type APIServer struct {
	db       *sql.DB
	handlers handlerSet
}

type handlerSet struct {
	health      *HealthHandlers
	vault       *VaultHandlers
	security    *SecurityHandlers
	resources   *ResourceHandlers
	deployment  *DeploymentHandlers
	scenarios   *ScenarioHandlers
	orientation *OrientationHandlers
}

func newAPIServer(db *sql.DB, logger *Logger) *APIServer {
	var validator *SecretValidator
	var orientationBuilder *OrientationBuilder
	if db != nil {
		validator = NewSecretValidator(db)
		orientationBuilder = NewOrientationBuilder(db, logger)
	}

	return &APIServer{
		db: db,
		handlers: handlerSet{
			health:      NewHealthHandlers(db),
			vault:       NewVaultHandlers(db, logger, validator),
			security:    NewSecurityHandlers(db, logger),
			resources:   NewResourceHandlers(db),
			deployment:  NewDeploymentHandlers(),
			scenarios:   NewScenarioHandlers(),
			orientation: NewOrientationHandlers(orientationBuilder),
		},
	}
}

// routes wires HTTP endpoints by capability to make the surface area obvious.
func (s *APIServer) routes() *mux.Router {
	r := mux.NewRouter()

	// Health
	r.HandleFunc("/health", s.healthHandler).Methods("GET")

	api := r.PathPrefix("/api/v1").Subrouter()

	// Health (API-prefixed for UI/iframe clients)
	api.HandleFunc("/health", s.healthHandler).Methods("GET")

	// Orientation insights
	orientation := api.PathPrefix("/orientation").Subrouter()
	orientation.HandleFunc("/summary", s.orientationSummaryHandler).Methods("GET")

	// Vault coverage and provisioning
	vault := api.PathPrefix("/vault").Subrouter()
	vault.HandleFunc("/secrets/status", s.vaultSecretsStatusHandler).Methods("GET")
	vault.HandleFunc("/secrets/provision", s.vaultProvisionHandler).Methods("POST")

	// Security intelligence
	security := api.PathPrefix("/security").Subrouter()
	security.HandleFunc("/scan", s.securityScanHandler).Methods("GET")
	security.HandleFunc("/compliance", s.complianceHandler).Methods("GET")
	security.HandleFunc("/vulnerabilities", s.vulnerabilitiesHandler).Methods("GET")
	security.HandleFunc("/vulnerabilities/fix", s.fixVulnerabilitiesHandler).Methods("POST")
	security.HandleFunc("/vulnerabilities/fix/progress", s.fixProgressHandler).Methods("POST")
	security.HandleFunc("/vulnerabilities/{id}/status", s.vulnerabilityStatusHandler).Methods("POST")
	api.HandleFunc("/vulnerabilities", s.vulnerabilitiesHandler).Methods("GET") // Legacy route for backward compatibility
	api.HandleFunc("/files/content", s.fileContentHandler).Methods("GET")

	// Resource intelligence
	resources := api.PathPrefix("/resources").Subrouter()
	resources.HandleFunc("/{resource}", s.resourceDetailHandler).Methods("GET")
	resources.HandleFunc("/{resource}/secrets/{secret}", s.resourceSecretUpdateHandler).Methods("PATCH")
	resources.HandleFunc("/{resource}/secrets/{secret}/strategy", s.secretStrategyHandler).Methods("POST")

	// Legacy secrets endpoints (kept for clients already depending on them)
	api.HandleFunc("/secrets/scan", s.vaultSecretsStatusHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/validate", s.validateHandler).Methods("GET", "POST")
	api.HandleFunc("/secrets/provision", s.provisionHandler).Methods("POST")

	// Deployment manifest and bundle consumers
	deployment := api.PathPrefix("/deployment").Subrouter()
	deployment.HandleFunc("/secrets", s.deploymentSecretsHandler).Methods("POST")
	deployment.HandleFunc("/secrets/{scenario}", s.deploymentSecretsGetHandler).Methods("GET")

	// Scenario intelligence (fast list for UI selection)
	scenarios := api.PathPrefix("/scenarios").Subrouter()
	scenarios.HandleFunc("", s.scenarioListHandler).Methods("GET")

	return r
}
