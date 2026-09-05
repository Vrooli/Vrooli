//nolint:gofumpt // golangci-lint's bundled formatter disagrees with the pinned formatter.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	r := apiServer.routes()
	rootMux := http.NewServeMux()
	rootMux.Handle("/", r)
	devrouting.RegisterWithFileRoots(rootMux, db, campaignRoots)

	// CORS headers
	corsHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})

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
		handlers.CORS(corsHeaders, corsMethods, corsOrigins)(apihttp.TestModeMiddleware(rootMux)).ServeHTTP(w, request)
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
