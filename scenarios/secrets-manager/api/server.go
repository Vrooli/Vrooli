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

	// Create manifest builder for deployment handlers
	manifestBuilder := NewManifestBuilder(ManifestBuilderConfig{
		DB:     db,
		Logger: logger,
	})

	return &APIServer{
		db: db,
		handlers: handlerSet{
			health:      NewHealthHandlers(db),
			vault:       NewVaultHandlers(db, logger, validator),
			security:    NewSecurityHandlers(db, logger),
			resources:   NewResourceHandlers(db),
			deployment:  NewDeploymentHandlers(manifestBuilder),
			scenarios:   NewScenarioHandlers(),
			orientation: NewOrientationHandlers(orientationBuilder),
		},
	}
}

// routes wires HTTP endpoints by capability to make the surface area obvious.
func (s *APIServer) routes() *mux.Router {
	r := mux.NewRouter()

	// Health
	s.handlers.health.RegisterRoutes(r)

	api := r.PathPrefix("/api/v1").Subrouter()

	// Health (API-prefixed for UI/iframe clients)
	s.handlers.health.RegisterRoutes(api)

	// Orientation insights
	orientation := api.PathPrefix("/orientation").Subrouter()
	s.handlers.orientation.RegisterRoutes(orientation)

	// Vault coverage and provisioning
	vault := api.PathPrefix("/vault").Subrouter()
	s.handlers.vault.RegisterRoutes(vault)
	s.handlers.vault.RegisterLegacyRoutes(api)

	// Security intelligence
	security := api.PathPrefix("/security").Subrouter()
	s.handlers.security.RegisterRoutes(security)
	s.handlers.security.RegisterLegacyRoutes(api)

	// Resource intelligence
	resources := api.PathPrefix("/resources").Subrouter()
	s.handlers.resources.RegisterRoutes(resources)

	// Deployment manifest and bundle consumers
	deployment := api.PathPrefix("/deployment").Subrouter()
	s.handlers.deployment.RegisterRoutes(deployment)

	// Scenario intelligence (fast list for UI selection)
	scenarios := api.PathPrefix("/scenarios").Subrouter()
	s.handlers.scenarios.RegisterRoutes(scenarios)

	return r
}
