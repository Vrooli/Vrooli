package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/intelligence"
)

func isProductionEnvironment() bool {
	env := strings.ToLower(strings.TrimSpace(envx.Get("LPBS_ENVIRONMENT")))
	return env == "production" || env == "prod"
}

func NewAPIKeyService(db administration.APIKeyStore) (*administration.APIKeyService, error) {
	return NewAPIKeyServiceWithHTTPClient(db, &http.Client{Timeout: 15 * time.Second})
}

func NewAPIKeyServiceWithHTTPClient(db administration.APIKeyStore, client administration.APIKeyHTTPDoer) (*administration.APIKeyService, error) {
	return NewAPIKeyServiceWithOptions(db, client, "postgres")
}

func NewAPIKeyServiceWithOptions(db administration.APIKeyStore, client administration.APIKeyHTTPDoer, dialect string) (*administration.APIKeyService, error) {
	return administration.NewAPIKeyServiceWithRuntime(db, client, dialect, resolveSecret, isProductionEnvironment, logStructured, logStructuredError)
}

func newAPIKeyServiceForTest(db administration.APIKeyStore, client administration.APIKeyHTTPDoer, dialect string, key []byte) *administration.APIKeyService {
	return administration.NewAPIKeyServiceForTest(db, client, dialect, key, nil, nil)
}

func GenerateEncryptionKey() string { return administration.GenerateEncryptionKey() }

// init registers the key generation command for setup
func init() {
	if len(os.Args) > 1 && os.Args[1] == "generate-encryption-key" {
		_, _ = os.Stdout.WriteString("New encryption key (add to LPBS_API_KEY_ENCRYPTION_KEY):\n")
		_, _ = os.Stdout.WriteString(GenerateEncryptionKey() + "\n")
		os.Exit(0)
	}
}

// Compile-time interface check for APIKeyServicer
var _ intelligence.APIKeyServicer = (*administration.APIKeyService)(nil)
