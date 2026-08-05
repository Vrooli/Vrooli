package server

// mountDomainHandlerTests exposes domain handlers to the legacy HTTP-focused
// tests without putting those compatibility routes back into production. The
// shipped server is Connect-only; these tests exercise the domain behavior
// directly until their assertions are expressed against generated clients.
func mountDomainHandlerTests(s *Server) {
	s.Router.HandleFunc("/api/v1/dependencies/analyze/{scenario}", s.DependenciesHandler.AnalyzeDependencies).Methods("GET")
	s.Router.HandleFunc("/api/v1/fitness/score", s.FitnessHandler.ScoreFitness).Methods("POST")
	s.Router.HandleFunc("/api/v1/profiles", s.ProfilesHandler.List).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles", s.ProfilesHandler.Create).Methods("POST")
	s.Router.HandleFunc("/api/v1/profiles/{id}", s.ProfilesHandler.Get).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}", s.ProfilesHandler.Update).Methods("PUT")
	s.Router.HandleFunc("/api/v1/profiles/{id}", s.ProfilesHandler.Delete).Methods("DELETE")
	s.Router.HandleFunc("/api/v1/profiles/{id}/versions", s.ProfilesHandler.GetVersions).Methods("GET")
	s.Router.HandleFunc("/api/v1/deploy/{profile_id}", s.DeploymentsHandler.Deploy).Methods("POST")
	s.Router.HandleFunc("/api/v1/swaps/analyze/{from}/{to}", s.SwapsHandler.Analyze).Methods("GET")
	s.Router.HandleFunc("/api/v1/swaps/cascade/{from}/{to}", s.SwapsHandler.Cascade).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}/secrets", s.SecretsHandler.IdentifySecrets).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}/secrets/template", s.SecretsHandler.GenerateSecretTemplate).Methods("GET")
	s.Router.HandleFunc("/api/v1/profiles/{id}/validate", s.ProfilesHandler.Validate).Methods("GET")
}
