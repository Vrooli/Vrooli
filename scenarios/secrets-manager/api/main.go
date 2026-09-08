//nolint:gofumpt // golangci-lint's bundled formatter disagrees with the pinned formatter.
package main

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"secrets-manager-api/internal/envx"

	"github.com/gorilla/handlers"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/receiptsigning"
	apiserver "github.com/vrooli/api-core/server"

	// Register the driver selected by database.DriverPostgres. Without this
	// import the process builds successfully but exits before serving /health.

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// Package-level logger
var logger *Logger

// Database connection
var (
	db            *database.RoutedDB
	campaignRoots *filerouting.RoutedRoots
)

func initDB(desktopMode bool) *database.RoutedDB {
	config := database.Config{Driver: database.DriverPostgres, TestDriver: database.DriverSQLite}
	if desktopMode {
		var err error
		db, err = openDesktopDatabase(context.Background())
		if err != nil {
			log.Fatal("Desktop database initialization failed:", err)
		}
		db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
			return initializeDesktopSchema(ctx, pool)
		})
		return db
	}
	db, err := database.Open(context.Background(), config)
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	if err := ensurePostgresSchema(context.Background(), db); err != nil {
		_ = db.Close()
		log.Fatal("Database schema initialization failed:", err)
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		return initializeDesktopSchema(ctx, pool)
	})
	return db
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "secrets-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}
	if err := configureCampaignRoots(); err != nil {
		log.Fatal("campaign storage routing initialization failed:", err)
	}

	// Initialize structured logger
	logger = NewLogger("secrets-manager")
	startup, err := loadStartupEnvironment(envx.OS{})
	if err != nil {
		log.Fatal("invalid Secrets Manager environment:", err)
	}

	if startup.skipDB {
		logger.Info("⚠️ Skipping database initialization (SECRETS_MANAGER_SKIP_DB=true)")
	} else {
		db = initDB(startup.desktopMode)
		defer db.Close()
		warmSecurityScanCache()
	}
	logger.Info("🚀 Starting Secrets Manager API (database optional)")

	apiServer := newAPIServer(db, logger)
	// Backup and restore execution remain owned by Data Backup Manager. The
	// password-manager recovery surface receives only its metadata-only evidence
	// projection, including explicit degraded/unavailable states.
	apiServer.handlers.passwordManager.backupEvidence = productionRecoveryEvidence
	r := apiServer.routes()
	rootMux := http.NewServeMux()
	rootMux.Handle("/", r)
	devrouting.RegisterWithFileRoots(rootMux, db, campaignRoots)

	// CORS headers
	corsHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization", "X-Agent-Identity-Token", "X-Machine-Principal-Token", "X-Workspace-ID", "X-Actor-ID", "X-Assurance-Token", "X-Broker-Session", "X-Download-Token", "X-Secrets-Manager-Native-Host-Token", "Idempotency-Key", "If-Match"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"})
	corsOrigins := handlers.AllowedOrigins(configuredCORSOrigins())
	corsCredentials := handlers.AllowCredentials()
	authenticatorVerifier := newAuthenticatorVerifierFromEnv()
	agentIdentityVerifier := newAgentIdentityVerifierFromEnv()

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		log.Fatal("❌ API_PORT environment variable is required")
	}

	logger.Info("🔐 Secrets Manager API starting on port %s", port)
	logger.Info("   📊 Health check: http://localhost:%s/health", port)
	logger.Info("   🔍 Scan endpoint: http://localhost:%s/api/v1/secrets/scan", port)
	logger.Info("   ✅ Validate endpoint: http://localhost:%s/api/v1/secrets/validate", port)

	// Production receipt-key rotation requires the API process itself to verify
	// the operator client certificate. A TLS-terminating proxy cannot provide the
	// verified TLS state required by the rotation handler.
	tlsConfig, err := receiptSigningServerTLSConfig()
	if err != nil {
		log.Fatal("receipt-signing TLS configuration failed:", err)
	}

	securityHeaders := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		handlers.CORS(corsHeaders, corsMethods, corsOrigins, corsCredentials)(apihttp.TestModeMiddleware(ownerAuthBoundary(rootMux, authenticatorVerifier, agentIdentityVerifier, apiServer.handlers.passwordManager.manager))).ServeHTTP(w, request)
	})
	httpServer := &http.Server{
		Addr:      ":" + port,
		Handler:   securityHeaders,
		TLSConfig: tlsConfig,
	}
	if err := apiserver.Run(apiserver.Config{
		Port: port,
		StartServer: func(addr string) error {
			httpServer.Addr = addr
			if tlsConfig != nil {
				return httpServer.ListenAndServeTLS("", "")
			}
			return httpServer.ListenAndServe()
		},
		ShutdownServer: httpServer.Shutdown,
		Logger:         logger.Info,
	}); err != nil {
		log.Fatal("Secrets Manager API server failed:", err)
	}
}

// ownerAuthBoundary protects the legacy operator endpoints that predate the
// password-manager handler. Health and preflight remain public; test-mode
// requests use the isolated test database and are intentionally allowed to
// exercise the handler surface without production credentials.
type machinePrincipalAuthenticator interface {
	AuthenticateMachinePrincipal(context.Context, string, string) (ownerAuthContext, error)
}

func ownerAuthBoundary(next http.Handler, verifier *authenticatorVerifier, agentVerifier *agentIdentityVerifier, machineAuthenticators ...machinePrincipalAuthenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" || r.URL.Path == "/api/v1/enrollment/status" || r.URL.Path == "/api/v1/enrollment/complete" || r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/register" || r.URL.Path == "/api/v1/auth/refresh" {
			next.ServeHTTP(w, r)
			return
		}
		// The dev-routing service is mounted only in development mode and is
		// the control-plane seam used by Test Genie to install an isolated
		// database/file lease. Keep it outside the product owner-auth boundary;
		// production mode does not mount this route at all.
		if strings.HasPrefix(r.URL.Path, "/vrooli.dev_routing.v1.routing.RoutingService/") {
			next.ServeHTTP(w, r)
			return
		}
		if database.IsTestMode(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		provided := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(r.Header.Get("Authorization")), "Bearer "))
		if deploymentRouteAllowed(r) {
			expected, _ := configuredSecretsManagerDeploymentToken()
			if expected != "" && provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1 {
				workspace, _ := requestIdentityFromHeaders(r)
				setOwnerAuthContext(r, ownerAuthContext{WorkspaceID: workspace, PrincipalID: "secrets-manager-deployment", Role: "service"})
				next.ServeHTTP(w, r)
				return
			}
			if provided == "" || expected == "" {
				writeAuthBoundaryError(w, http.StatusServiceUnavailable, `{"error":"deployment_auth_unconfigured","message":"deployment service authentication is required"}`)
				return
			}
			writeAuthBoundaryError(w, http.StatusUnauthorized, `{"error":"deployment_token_invalid","message":"deployment service authentication was rejected"}`)
			return
		}
		expected, _ := configuredSecretsManagerOwnerToken()
		if expected != "" && provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1 {
			next.ServeHTTP(w, r)
			return
		}
		if provided == "" {
			if token := strings.TrimSpace(r.Header.Get("X-Agent-Identity-Token")); token != "" {
				if !agentRouteAllowed(r) || agentVerifier == nil {
					http.Error(w, "agent identity is not authorized for this route", http.StatusForbidden)
					return
				}
				workspace, _ := requestIdentityFromHeaders(r)
				identity, err := agentVerifier.VerifyForWorkspace(r.Context(), token, workspace)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"agent_identity_invalid","message":"agent identity verification was rejected"}`))
					return
				}
				setOwnerAuthContext(r, ownerAuthContext{WorkspaceID: workspace, PrincipalID: identity.PrincipalID, Role: "agent", RunID: identity.RunID})
				next.ServeHTTP(w, r)
				return
			}
			if token := strings.TrimSpace(r.Header.Get("X-Machine-Principal-Token")); token != "" {
				if !agentRouteAllowed(r) || len(machineAuthenticators) == 0 {
					http.Error(w, "machine principal is not authorized for this route", http.StatusForbidden)
					return
				}
				workspace, _ := requestIdentityFromHeaders(r)
				auth, err := machineAuthenticators[0].AuthenticateMachinePrincipal(r.Context(), workspace, token)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"machine_principal_invalid","message":"machine principal verification was rejected"}`))
					return
				}
				setOwnerAuthContext(r, auth)
				next.ServeHTTP(w, r)
				return
			}
		}
		if verifier != nil && provided != "" {
			identity, err := verifier.Verify(r.Context(), provided)
			if err == nil {
				workspace, _ := requestIdentityFromHeaders(r)
				setOwnerAuthContext(r, ownerAuthContext{WorkspaceID: workspace, PrincipalID: identity.Subject, AuthenticatedAt: identity.IssuedAt})
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"authenticator_token_invalid","message":"owner authentication was rejected"}`))
			return
		}
		if expected == "" || provided == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":"owner_auth_unconfigured","message":"owner authentication is required"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"owner_token_invalid","message":"owner authentication was rejected"}`))
	})
}

func writeAuthBoundaryError(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func deploymentRouteAllowed(r *http.Request) bool {
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/deployment/secrets/") {
		return true
	}
	return r.Method == http.MethodPost && (r.URL.Path == "/api/v1/deployment/secrets" || r.URL.Path == "/api/v1/deployment/readiness")
}

func agentRouteAllowed(r *http.Request) bool {
	path := r.URL.Path
	if r.Method == http.MethodPost && path == "/api/v1/access-requests" {
		return true
	}
	if path == "/api/v1/broker/sessions" || strings.HasPrefix(path, "/api/v1/broker/sessions/") {
		return true
	}
	if r.Method == http.MethodPost && strings.HasPrefix(path, "/api/v1/credential-use/runs/") && strings.HasSuffix(path, "/revoke") {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/credential-use/browser/") || strings.HasPrefix(path, "/api/v1/credential-use/ssh/") {
		return true
	}
	return strings.HasPrefix(path, "/vrooli.secrets_manager.v1.credentials.CredentialBrokerService/")
}

// configuredCORSOrigins returns explicit browser origins only. A wildcard is
// unsafe for a credential authority because it permits arbitrary pages to
// issue credential-management requests. The lifecycle port is used for the
// bundled UI and operators may add a trusted reverse-proxy origin explicitly.
func configuredCORSOrigins() []string {
	origins := make([]string, 0, 8)
	uiPort := strings.TrimSpace(os.Getenv("UI_PORT"))
	if uiPort != "" {
		origins = append(origins, "http://localhost:"+uiPort, "http://127.0.0.1:"+uiPort)
	}
	for _, value := range strings.Split(os.Getenv("SECRETS_MANAGER_ALLOWED_ORIGINS"), ",") {
		value = strings.TrimRight(strings.TrimSpace(value), "/")
		if value != "" {
			origins = append(origins, value)
		}
	}
	if len(origins) == 0 {
		// A non-browser API remains usable in development without silently
		// widening the policy to every origin.
		return []string{"http://localhost", "http://127.0.0.1"}
	}
	seen := make(map[string]bool, len(origins))
	result := make([]string, 0, len(origins))
	for _, origin := range origins {
		if !seen[origin] {
			seen[origin] = true
			result = append(result, origin)
		}
	}
	return result
}

// receiptSigningServerTLSConfig loads only lifecycle-declared file locations.
// It never reads a certificate or private key from an environment variable.
func receiptSigningServerTLSConfig() (*tls.Config, error) {
	type trustSigning struct {
		Provider                string `json:"provider"`
		OperatorTLSCertFile     string `json:"operator_tls_cert_file"`
		OperatorTLSKeyFile      string `json:"operator_tls_key_file"`
		OperatorTLSClientCAFile string `json:"operator_tls_client_ca_file"`
	}
	type manifest struct {
		TrustSigning *trustSigning `json:"trust_signing"`
	}
	scenarioDir, err := optionalScenarioDirectory(envx.OS{})
	if err != nil {
		return nil, err
	}
	if scenarioDir == "" {
		return nil, nil // Development lifecycle has no rotation authority.
	}
	contents, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return nil, fmt.Errorf("read receipt-signing lifecycle declaration: %w", err)
	}
	var service manifest
	if err := json.Unmarshal(contents, &service); err != nil {
		return nil, fmt.Errorf("parse receipt-signing lifecycle declaration: %w", err)
	}
	config := service.TrustSigning
	if config == nil || config.Provider == "development" {
		return nil, nil
	}
	if config.Provider != receiptsigning.ModeCredentialAuthorityEd25519 || config.OperatorTLSCertFile == "" || config.OperatorTLSKeyFile == "" || config.OperatorTLSClientCAFile == "" {
		return nil, fmt.Errorf("credential authority operator rotation requires lifecycle-declared server certificate, server key, and client CA files")
	}
	certificate, err := tls.LoadX509KeyPair(config.OperatorTLSCertFile, config.OperatorTLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load receipt-signing server certificate: %w", err)
	}
	caBytes, err := os.ReadFile(config.OperatorTLSClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read receipt-signing client CA: %w", err)
	}
	clientCAs := x509.NewCertPool()
	if !clientCAs.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("parse receipt-signing client CA")
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: clientCAs}, nil
}
