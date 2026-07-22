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
	"strings"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"

	"github.com/gorilla/handlers"
)

// Package-level logger
var logger *Logger

// Database connection
var db *sql.DB

func initDB() *sql.DB {
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	return db
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "secrets-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Initialize structured logger
	logger = NewLogger("secrets-manager")

	skipDB := strings.EqualFold(os.Getenv("SECRETS_MANAGER_SKIP_DB"), "true")
	if skipDB {
		logger.Info("⚠️ Skipping database initialization (SECRETS_MANAGER_SKIP_DB=true)")
	} else {
		db = initDB()
		defer db.Close()
		warmSecurityScanCache()
	}
	logger.Info("🚀 Starting Secrets Manager API (database optional)")

	apiServer := newAPIServer(db, logger)
	r := apiServer.routes()

	// CORS headers
	corsHeaders := handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	corsMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	corsOrigins := handlers.AllowedOrigins([]string{"*"})

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT
		if port == "" {
			log.Fatal("❌ API_PORT or PORT environment variable is required")
		}
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

	server := &http.Server{
		Addr:      ":" + port,
		Handler:   handlers.CORS(corsHeaders, corsMethods, corsOrigins)(r),
		TLSConfig: tlsConfig,
	}
	if tlsConfig != nil {
		log.Fatal(server.ListenAndServeTLS("", ""))
	}
	log.Fatal(server.ListenAndServe())
}

// receiptSigningServerTLSConfig loads only lifecycle-declared file locations.
// It never reads a certificate or private key from an environment variable.
func receiptSigningServerTLSConfig() (*tls.Config, error) {
	type trustSigning struct {
		Provider                string `json:"provider"`
		OperatorCredentialFile  string `json:"operator_credential_file"`
		OperatorTLSCertFile     string `json:"operator_tls_cert_file"`
		OperatorTLSKeyFile      string `json:"operator_tls_key_file"`
		OperatorTLSClientCAFile string `json:"operator_tls_client_ca_file"`
	}
	type manifest struct {
		TrustSigning *trustSigning `json:"trust_signing"`
	}
	scenarioDir := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_DIR"))
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
	if config.Provider != "vault-transit" || config.OperatorCredentialFile == "" || config.OperatorTLSCertFile == "" || config.OperatorTLSKeyFile == "" || config.OperatorTLSClientCAFile == "" {
		return nil, fmt.Errorf("Vault Transit operator rotation requires lifecycle-declared credential, server certificate, server key, and client CA files")
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
