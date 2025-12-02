package main

// setupRoutes configures all HTTP routes for the server.
func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// Health endpoints at both locations (root for infrastructure checks, /api/v1 for clients)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET")

	// Dependency analysis endpoints
	s.router.HandleFunc("/api/v1/dependencies/analyze/{scenario}", s.handleAnalyzeDependencies).Methods("GET")

	// Fitness scoring endpoints
	s.router.HandleFunc("/api/v1/fitness/score", s.handleScoreFitness).Methods("POST")

	// Profile management endpoints
	s.router.HandleFunc("/api/v1/profiles", s.handleListProfiles).Methods("GET")
	s.router.HandleFunc("/api/v1/profiles", s.handleCreateProfile).Methods("POST")
	s.router.HandleFunc("/api/v1/profiles/{id}", s.handleGetProfile).Methods("GET")
	s.router.HandleFunc("/api/v1/profiles/{id}", s.handleUpdateProfile).Methods("PUT")
	s.router.HandleFunc("/api/v1/profiles/{id}", s.handleDeleteProfile).Methods("DELETE")
	s.router.HandleFunc("/api/v1/profiles/{id}/versions", s.handleGetProfileVersions).Methods("GET")

	// Deployment endpoints
	s.router.HandleFunc("/api/v1/deploy/{profile_id}", s.handleDeploy).Methods("POST")
	s.router.HandleFunc("/api/v1/deployments/{deployment_id}", s.handleDeploymentStatus).Methods("GET")

	// Swap analysis endpoints
	s.router.HandleFunc("/api/v1/swaps/analyze/{from}/{to}", s.handleSwapAnalyze).Methods("GET")
	s.router.HandleFunc("/api/v1/swaps/cascade/{from}/{to}", s.handleSwapCascade).Methods("GET")

	// Validation endpoints
	s.router.HandleFunc("/api/v1/profiles/{id}/validate", s.handleValidateProfile).Methods("GET")
	s.router.HandleFunc("/api/v1/profiles/{id}/cost-estimate", s.handleCostEstimate).Methods("GET")

	// Secret management endpoints
	s.router.HandleFunc("/api/v1/profiles/{id}/secrets", s.handleIdentifySecrets).Methods("GET")
	s.router.HandleFunc("/api/v1/profiles/{id}/secrets/template", s.handleGenerateSecretTemplate).Methods("GET")
	s.router.HandleFunc("/api/v1/profiles/{id}/secrets/validate", s.handleValidateSecrets).Methods("POST")
	s.router.HandleFunc("/api/v1/secrets/validate", s.handleValidateSecret).Methods("POST")
	s.router.HandleFunc("/api/v1/secrets/test", s.handleTestSecret).Methods("GET", "POST")

	// Bundle validation and assembly
	s.router.HandleFunc("/api/v1/bundles/validate", s.handleValidateBundle).Methods("POST")
	s.router.HandleFunc("/api/v1/bundles/merge-secrets", s.handleMergeBundleSecrets).Methods("POST")
	s.router.HandleFunc("/api/v1/bundles/assemble", s.handleAssembleBundle).Methods("POST")

	// Telemetry ingestion and summaries
	s.router.HandleFunc("/api/v1/telemetry", s.handleListTelemetry).Methods("GET")
	s.router.HandleFunc("/api/v1/telemetry/upload", s.handleUploadTelemetry).Methods("POST")
}
