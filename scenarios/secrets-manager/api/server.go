package main

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	credentialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/secrets-manager/v1/credentials/credentials_v1connect"
)

// APIServer centralizes the HTTP surface for the scenario so routes reflect the
// core domains: health, credential coverage, security scanning, resources, and deployment.
type APIServer struct {
	db       *database.RoutedDB
	handlers handlerSet
}

type handlerSet struct {
	health           *HealthHandlers
	credentials      *CredentialHandlers
	security         *SecurityHandlers
	resources        *ResourceHandlers
	deployment       *DeploymentHandlers
	scenarios        *ScenarioHandlers
	orientation      *OrientationHandlers
	campaigns        *CampaignHandlers
	overrides        *ScenarioOverrideHandlers
	adminOverrides   *AdminOverrideHandlers
	receiptSigning   *ReceiptSigningHandlers
	allowlist        *AllowlistHandlers
	watchlist        *WatchlistHandlers
	passwordManager  *passwordManagerHandlers
	credentialBroker *credentialBrokerConnectHandler
	authProxy        *authProxyHandlers
}

func newAPIServer(db *database.RoutedDB, logger *Logger) *APIServer {
	var validator *SecretValidator
	var orientationBuilder *OrientationBuilder
	var campaignStore CampaignStore
	if db != nil {
		validator = NewSecretValidator(db)
		orientationBuilder = NewOrientationBuilder(db, logger)
		campaignStore = NewDBCampaignStore(db)
	}

	// Create manifest builder for deployment handlers
	manifestBuilder := NewManifestBuilder(ManifestBuilderConfig{
		DB:     db,
		Clock:  systemManifestClock{},
		Logger: logger,
	})
	passwordManager := newPasswordManagerHandlers(db)

	return &APIServer{
		db: db,
		handlers: handlerSet{
			health:           NewHealthHandlers(db),
			credentials:      NewCredentialHandlers(db, logger, validator),
			security:         NewSecurityHandlers(db, logger),
			resources:        NewResourceHandlers(db),
			deployment:       NewDeploymentHandlers(manifestBuilder),
			scenarios:        NewScenarioHandlers(),
			orientation:      NewOrientationHandlers(orientationBuilder),
			campaigns:        NewCampaignHandlers(manifestBuilder, campaignStore),
			overrides:        NewScenarioOverrideHandlers(db, logger),
			adminOverrides:   NewAdminOverrideHandlers(db, logger),
			receiptSigning:   NewReceiptSigningHandlers(),
			allowlist:        NewAllowlistHandlers(db, logger),
			watchlist:        NewWatchlistHandlers(db, logger),
			passwordManager:  passwordManager,
			credentialBroker: newCredentialBrokerConnectHandler(passwordManager),
			authProxy:        newAuthProxyHandlers(),
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

	// Credential coverage and provisioning. Receipt signing is an authority-backed
	// operational endpoint, separate from ordinary credential provisioning.
	credentials := api.PathPrefix("/credentials").Subrouter()
	s.handlers.credentials.RegisterRoutes(credentials)
	s.handlers.receiptSigning.RegisterRoutes(credentials)

	// Security intelligence
	security := api.PathPrefix("/security").Subrouter()
	s.handlers.security.RegisterRoutes(security)
	s.handlers.allowlist.RegisterRoutes(security)
	s.handlers.watchlist.RegisterRoutes(security)
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

	// Deployment readiness campaigns
	campaigns := api.PathPrefix("/campaigns").Subrouter()
	s.handlers.campaigns.RegisterRoutes(campaigns)

	// Scenario secret strategy overrides
	overrides := api.PathPrefix("/scenarios").Subrouter()
	s.handlers.overrides.RegisterRoutes(overrides)

	// Admin override management
	admin := api.PathPrefix("/admin").Subrouter()
	s.handlers.adminOverrides.RegisterRoutes(admin)

	// Password-manager operations are a separate product boundary. Ordinary
	// list/detail calls return metadata only; secret-bearing reveal is guarded
	// by the manager's action-bound assurance contract.
	s.handlers.passwordManager.RegisterRoutes(api)
	s.handlers.authProxy.RegisterRoutes(api)
	connectPath, connectHandler := credentialsconnect.NewCredentialBrokerServiceHandler(s.handlers.credentialBroker)
	r.PathPrefix(connectPath).Handler(connectHandler)

	return r
}
