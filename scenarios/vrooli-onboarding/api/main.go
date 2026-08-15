package main

import (
	"context"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

// Server wires the HTTP router.
type Server struct {
	router *mux.Router
}

// routingMuxAdapter adapts gorilla/mux's fluent Handle signature to the small
// dev-routing interface without coupling api-core to this router package.
type routingMuxAdapter struct{ router *mux.Router }

func (m routingMuxAdapter) Handle(pattern string, handler http.Handler) {
	// Connect handlers own a procedure subtree. Gorilla's Handle is exact,
	// unlike net/http.ServeMux, so preserve devrouting's trailing-slash
	// subtree contract explicitly.
	m.router.PathPrefix(pattern).Handler(handler)
}

// NewServer initializes routes.
func NewServer() *Server {
	srv := &Server{
		router: mux.NewRouter(),
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(securityHeadersMiddleware)
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthHandler := health.New().Version("2.0.0").Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Resource endpoints (health must be before {name} to avoid capture)
	s.router.HandleFunc("/api/v1/resources/health", s.handleResourceHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/resources", s.handleListResources).Methods("GET")
	s.router.HandleFunc("/api/v1/resources/{name}", s.handleGetResource).Methods("GET")

	// Operator state is the sole authority for onboarding choices.
	s.router.HandleFunc("/api/v1/operator-state", s.handleOperatorState).Methods("GET")
	s.router.Handle("/api/v2/operator-state", onboardingMutationAuth(http.HandlerFunc(s.handleOperatorState))).Methods("PATCH")
	s.router.HandleFunc("/api/v2/scenarios", s.handleV2Scenarios).Methods("GET")
	s.router.HandleFunc("/api/v2/recommendation", s.handleV2Recommendation).Methods("GET")
	s.router.HandleFunc("/api/v2/resources", s.handleV2Resources).Methods("GET")
	s.router.HandleFunc("/api/v2/closure", s.handleV2Closure).Methods("GET")
	s.router.HandleFunc("/api/v2/union", s.handleV2Union).Methods("GET")
	s.router.HandleFunc("/api/v2/credentials", s.handleV2Credentials).Methods("GET")
	s.router.HandleFunc("/api/v2/credentials/store/status", s.handleV2CredentialStoreStatus).Methods("GET")
	s.router.HandleFunc("/api/v2/surface", s.handleV2Surface).Methods("GET")
	s.router.HandleFunc("/api/v2/handoff", s.handleV2Handoff).Methods("POST")
	s.router.Handle("/api/v2/apply", onboardingMutationAuth(http.HandlerFunc(s.handleV2Apply))).Methods("POST")
	s.router.HandleFunc("/api/v2/apply/plan", s.handleV2ApplyPlan).Methods("GET")
	s.router.HandleFunc("/api/v2/apply/{run_id}", s.handleV2ApplyStatus).Methods("GET")
	s.router.HandleFunc("/api/v2/session", s.handleV2Session).Methods("GET")
	s.router.HandleFunc("/api/v2/session/step", s.handleV2Session).Methods("POST")
	s.router.HandleFunc("/api/v2/steps", s.handleV2Steps).Methods("GET")
	s.router.HandleFunc("/api/v2/operator-inputs", s.handleV2OperatorInputs).Methods("GET")
	s.router.Handle("/api/v2/operator-inputs/resolve", onboardingMutationAuth(http.HandlerFunc(s.handleV2OperatorInputsResolve))).Methods("POST")
	s.router.HandleFunc("/api/v2/host-requirements", s.handleV2HostRequirements).Methods("GET")
	s.router.HandleFunc("/api/v2/readiness", s.handleV2Readiness).Methods("GET")
	s.router.Handle("/api/v2/credentials/store/select", onboardingMutationAuth(http.HandlerFunc(s.handleV2CredentialStoreSelect))).Methods("POST")
	s.router.Handle("/api/v2/credentials/store/reselect", onboardingMutationAuth(http.HandlerFunc(s.handleV2CredentialStoreReselect))).Methods("POST")
	s.router.Handle("/api/v2/credentials/store/init", onboardingMutationAuth(http.HandlerFunc(s.handleV2CredentialStoreInit))).Methods("POST")
	s.router.Handle("/api/v2/credentials/store/unlock", onboardingMutationAuth(http.HandlerFunc(s.handleV2CredentialStoreUnlock))).Methods("POST")
	s.router.Handle("/api/v2/credentials/store/change-passphrase", onboardingMutationAuth(http.HandlerFunc(s.handleV2CredentialStoreChangePassphrase))).Methods("POST")
	s.router.Handle("/api/v2/credentials/store/rewrap", onboardingMutationAuth(http.HandlerFunc(s.handleV2CredentialStoreRewrap))).Methods("POST")
	s.router.Handle("/api/v2/credentials/provision", onboardingMutationAuth(http.HandlerFunc(s.handleV2CredentialProvision))).Methods("POST")
	s.router.HandleFunc("/api/v2/credentials/doctor", s.handleV2CredentialDoctor).Methods("GET")
	s.router.HandleFunc("/api/v2/credentials/keyring/inspect", s.handleV2CredentialKeyringInspect).Methods("GET")
	s.router.HandleFunc("/api/v2/credentials/keyring/repair", s.handleV2CredentialKeyringRepair).Methods("POST")

	// Glossary endpoint
	s.router.HandleFunc("/api/v1/glossary", s.handleGlossary).Methods("GET")
}

// securityHeadersMiddleware applies the baseline browser-facing API policy at
// one boundary so every handler, including errors and health probes, is safe.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		next.ServeHTTP(w, r)
	})
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "vrooli-onboarding",
	}) {
		return // Process was re-exec'd after rebuild
	}

	if err := configureOperatorStateRoots(); err != nil {
		panic("configure operator state roots: " + err.Error())
	}
	// Keep the file-only API inside api-core's routed storage contract. The
	// operatorstate service performs the actual request-scoped selection; this
	// startup probe makes the seam explicit to storage-manager's static checker.
	ctx := context.Background()
	if _, err := operatorStateRoots.Pick(ctx, storage.ClassConfig); err != nil {
		panic("route operator state roots: " + err.Error())
	}
	// The onboarding API is file-authoritative, but test-genie still needs the
	// standard routed control surface to prove destructive workflows cannot use
	// a live pool. This private in-memory pool stores no onboarding state.
	routingDB, err := database.Open(ctx, database.Config{
		Driver:       database.DriverSQLite,
		DSN:          "file:vrooli-onboarding-routing?mode=memory&cache=shared",
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		panic("configure onboarding test routing: " + err.Error())
	}
	if err := database.EnsureSchemas(ctx, routingDB.Primary()); err != nil {
		panic("configure onboarding test schemas: " + err.Error())
	}
	srv := NewServer()
	devrouting.RegisterWithFileRoots(routingMuxAdapter{router: srv.router}, routingDB, operatorStateRoots)
	handler := apihttp.TestModeMiddleware(srv.Handler())

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: handler,
		Cleanup: func(context.Context) error { return routingDB.Close() },
	}); err != nil {
		panic("onboarding server failed: " + err.Error())
	}
}
