package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/receiptsigning"
	"github.com/vrooli/api-core/storage"
	credentialauthoritysigning "github.com/vrooli/vrooli/packages/credential-authority-go/receiptsigning"
)

// ReceiptSigningHandlers exposes non-sensitive operational truth for the
// trusted experiment receipt signer. It intentionally never returns the
// credential-file contents, a Vault Transit token, or signing key material.
type rotationSigner interface {
	Rotate(context.Context) (receiptsigning.Health, error)
}

type ReceiptSigningHandlers struct {
	configPath     func() (string, error)
	operatorSigner func() (rotationSigner, []string, error)
}

func NewReceiptSigningHandlers() *ReceiptSigningHandlers {
	return &ReceiptSigningHandlers{configPath: receiptSigningConfigPath, operatorSigner: loadOperatorRotationSigner}
}

func receiptSigningConfigPath() (string, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return "", err
	}
	scenarioID, err := storage.ScenarioNamespace("prompt-manager")
	if err != nil {
		return "", err
	}
	return resolver.Path(storage.Options{ScenarioID: scenarioID}, storage.ClassConfig, "receipt-signing.json")
}

func (h *ReceiptSigningHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/receipt-signing/status", h.Status).Methods(http.MethodGet)
	router.HandleFunc("/receipt-signing/rotate", h.Rotate).Methods(http.MethodPost)
}

func (h *ReceiptSigningHandlers) Status(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Configured bool                   `json:"configured"`
		Mode       string                 `json:"mode,omitempty"`
		Health     *receiptsigning.Health `json:"health,omitempty"`
		Error      string                 `json:"error,omitempty"`
	}{}
	path, err := h.configPath()
	if err != nil {
		response.Error = "resolve receipt signing configuration"
		writeReceiptSigningStatus(w, response)
		return
	}
	config, err := receiptsigning.LoadRuntimeConfig(path)
	if os.IsNotExist(err) {
		response.Configured = true
		response.Mode = "development"
		health := receiptsigning.Health{Ready: true, Provider: "development-hmac", KeyID: "development-only", Production: false, RotationOK: false, Description: "development-only signer; receipts are never production eligible"}
		response.Health = &health
		writeReceiptSigningStatus(w, response)
		return
	}
	if err != nil {
		response.Error = "invalid receipt signing configuration"
		writeReceiptSigningStatus(w, response)
		return
	}
	response.Configured, response.Mode = true, config.Mode
	var signer receiptsigning.ReceiptSigner
	if config.Mode == receiptsigning.ModeCredentialAuthorityEd25519 {
		signer, _, err = credentialauthoritysigning.NewSignerFromRuntimeConfig(config)
	} else {
		signer, _, err = receiptsigning.NewSignerFromRuntimeConfig(config)
	}
	if err != nil {
		response.Error = "receipt signer is unavailable"
		writeReceiptSigningStatus(w, response)
		return
	}
	health, err := signer.Health(r.Context())
	response.Health = &health
	if err != nil {
		response.Error = "receipt signer health check failed"
	}
	writeReceiptSigningStatus(w, response)
}

func writeReceiptSigningStatus(w http.ResponseWriter, response any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// Rotate uses a verified mTLS operator certificate. The signer itself writes
// only through the credential authority; a caller-provided decision
// ID/header is never trusted as authorization.
func (h *ReceiptSigningHandlers) Rotate(w http.ResponseWriter, r *http.Request) {
	signer, subjects, err := h.operatorSigner()
	if err != nil {
		http.Error(w, "receipt-signing rotation is unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := authorizeReceiptSigningOperator(r.TLS, subjects); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	health, err := signer.Rotate(r.Context())
	if err != nil {
		http.Error(w, "receipt-signing rotation failed", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Health receiptsigning.Health `json:"health"`
	}{Health: health})
}

func authorizeReceiptSigningOperator(connection *tls.ConnectionState, subjects []string) error {
	if connection == nil || len(connection.VerifiedChains) == 0 || len(connection.PeerCertificates) == 0 {
		return fmt.Errorf("receipt-signing rotation requires a verified operator mTLS certificate")
	}
	name := strings.TrimSpace(connection.PeerCertificates[0].Subject.CommonName)
	for _, subject := range subjects {
		if name == strings.TrimSpace(subject) {
			return nil
		}
	}
	return fmt.Errorf("receipt-signing operator certificate is not authorized")
}

func loadOperatorRotationSigner() (rotationSigner, []string, error) {
	type trustSigning struct {
		Provider           string   `json:"provider"`
		Identity           string   `json:"identity"`
		Field              string   `json:"field"`
		OperatorSubjects   []string `json:"operator_subjects"`
		LegacyVaultTransit *struct {
			Address        string `json:"address"`
			KeyName        string `json:"key_name"`
			CredentialFile string `json:"credential_file"`
		} `json:"legacy_vault_transit"`
	}
	type manifest struct {
		TrustSigning *trustSigning `json:"trust_signing"`
	}
	scenarioDir := strings.TrimSpace(os.Getenv("VROOLI_SCENARIO_DIR"))
	if scenarioDir == "" {
		return nil, nil, fmt.Errorf("lifecycle scenario directory is unavailable")
	}
	contents, err := os.ReadFile(filepath.Join(scenarioDir, ".vrooli", "service.json"))
	if err != nil {
		return nil, nil, err
	}
	var service manifest
	if err := json.Unmarshal(contents, &service); err != nil || service.TrustSigning == nil {
		return nil, nil, fmt.Errorf("missing trust-signing operator lifecycle declaration")
	}
	config := service.TrustSigning
	if config.Provider != receiptsigning.ModeCredentialAuthorityEd25519 || config.Identity == "" || config.Field == "" || len(config.OperatorSubjects) == 0 {
		return nil, nil, fmt.Errorf("trust-signing operator rotation is not configured")
	}
	runtimeConfig := receiptsigning.RuntimeConfig{
		Version: receiptsigning.RuntimeConfigVersion,
		Mode:    receiptsigning.ModeCredentialAuthorityEd25519,
		CredentialAuthority: &receiptsigning.CredentialAuthorityRuntimeConfig{
			Identity: config.Identity,
			Field:    config.Field,
		},
	}
	if legacy := config.LegacyVaultTransit; legacy != nil {
		runtimeConfig.LegacyVaultTransit = &receiptsigning.VaultTransitRuntimeConfig{Address: legacy.Address, KeyName: legacy.KeyName, CredentialFile: legacy.CredentialFile}
	}
	signer, _, err := credentialauthoritysigning.NewSignerFromRuntimeConfig(runtimeConfig)
	if err != nil {
		return nil, nil, err
	}
	rotator, ok := signer.(rotationSigner)
	if !ok {
		return nil, nil, fmt.Errorf("credential authority receipt signer does not support rotation")
	}
	return rotator, config.OperatorSubjects, nil
}
