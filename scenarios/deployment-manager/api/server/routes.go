package server

// setupRoutes configures all HTTP routes for the server.
func (s *Server) setupRoutes() {
	s.Router.Use(LoggingMiddleware)

	// Health endpoints at both locations (root for infrastructure checks, /api/v1 for clients)
	s.Router.HandleFunc("/health", s.HealthHandler.Health).Methods("GET")
	s.Router.HandleFunc("/api/v1/health", s.HealthHandler.Health).Methods("GET")

	// Dependency analysis endpoints
	s.Router.HandleFunc("/api/v1/dependencies/analyze/{scenario}", s.DependenciesHandler.AnalyzeDependencies).Methods("GET")

	// Fitness scoring endpoints
	s.Router.HandleFunc("/api/v1/fitness/score", s.FitnessHandler.ScoreFitness).Methods("POST")

	// Profile management endpoints
	s.Router.HandleFunc("/api/v1/profiles", s.ProfilesHandler.List).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles", s.ProfilesHandler.Create).Methods("POST")
	s.Router.HandleFunc("/api/v1/profiles/{id}", s.ProfilesHandler.Get).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}", s.ProfilesHandler.Update).Methods("PUT")
	s.Router.HandleFunc("/api/v1/profiles/{id}", s.ProfilesHandler.Delete).Methods("DELETE")
	s.Router.HandleFunc("/api/v1/profiles/{id}/versions", s.ProfilesHandler.GetVersions).Methods("GET")

	// Deployment endpoints
	s.Router.HandleFunc("/api/v1/deploy/{profile_id}", s.DeploymentsHandler.Deploy).Methods("POST")
	s.Router.HandleFunc("/api/v1/deployments/{deployment_id}", s.DeploymentsHandler.Status).Methods("GET")

	// Swap analysis endpoints
	s.Router.HandleFunc("/api/v1/swaps/suggest/{scenario}", s.SwapsHandler.List).Methods("GET")
	s.Router.HandleFunc("/api/v1/swaps/list/{scenario}", s.SwapsHandler.List).Methods("GET") // Alias
	s.Router.HandleFunc("/api/v1/swaps/analyze/{from}/{to}", s.SwapsHandler.Analyze).Methods("GET")
	s.Router.HandleFunc("/api/v1/swaps/cascade/{from}/{to}", s.SwapsHandler.Cascade).Methods("GET")
	s.Router.HandleFunc("/api/v1/swaps/apply", s.SwapsHandler.Apply).Methods("POST")
	s.Router.HandleFunc("/api/v1/profiles/{id}/swaps", s.SwapsHandler.ApplyToProfile).Methods("POST")

	// Validation endpoints
	s.Router.HandleFunc("/api/v1/profiles/{id}/validate", s.ProfilesHandler.Validate).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}/cost-estimate", s.ProfilesHandler.CostEstimate).Methods("GET")

	// Secret management endpoints
	s.Router.HandleFunc("/api/v1/profiles/{id}/secrets", s.SecretsHandler.IdentifySecrets).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}/secrets/template", s.SecretsHandler.GenerateSecretTemplate).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}/secrets/validate", s.SecretsHandler.ValidateSecrets).Methods("POST")
	s.Router.HandleFunc("/api/v1/secrets/validate", s.SecretsHandler.ValidateSecret).Methods("POST")
	s.Router.HandleFunc("/api/v1/secrets/test", s.SecretsHandler.TestSecret).Methods("GET", "POST")

	// Bundle validation, assembly, and export
	s.Router.HandleFunc("/api/v1/bundles/validate", s.BundlesHandler.ValidateBundle).Methods("POST")
	s.Router.HandleFunc("/api/v1/bundles/merge-secrets", s.BundlesHandler.MergeBundleSecrets).Methods("POST")
	s.Router.HandleFunc("/api/v1/bundles/assemble", s.BundlesHandler.AssembleBundle).Methods("POST")
	s.Router.HandleFunc("/api/v1/bundles/export", s.BundlesHandler.ExportBundle).Methods("POST")
	s.Router.HandleFunc("/api/v1/bundles/signing-config", s.BundlesHandler.GenerateSigningConfig).Methods("POST")

	// Telemetry ingestion and summaries
	s.Router.HandleFunc("/api/v1/telemetry", s.TelemetryHandler.List).Methods("GET")
	s.Router.HandleFunc("/api/v1/telemetry/upload", s.TelemetryHandler.Upload).Methods("POST")

	// Code signing configuration endpoints
	s.Router.HandleFunc("/api/v1/profiles/{id}/signing", s.SigningHandler.GetSigning).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}/signing", s.SigningHandler.SetSigning).Methods("PUT")
	s.Router.HandleFunc("/api/v1/profiles/{id}/signing/{platform}", s.SigningHandler.SetPlatformSigning).Methods("PATCH")
	s.Router.HandleFunc("/api/v1/profiles/{id}/signing", s.SigningHandler.DeleteSigning).Methods("DELETE")
	s.Router.HandleFunc("/api/v1/profiles/{id}/signing/{platform}", s.SigningHandler.DeletePlatformSigning).Methods("DELETE")
	s.Router.HandleFunc("/api/v1/profiles/{id}/signing/validate", s.SigningHandler.ValidateSigning).Methods("POST")
	s.Router.HandleFunc("/api/v1/signing/prerequisites", s.SigningHandler.CheckPrerequisites).Methods("GET")
	s.Router.HandleFunc("/api/v1/signing/discover/{platform}", s.SigningHandler.DiscoverCertificates).Methods("GET")
}
