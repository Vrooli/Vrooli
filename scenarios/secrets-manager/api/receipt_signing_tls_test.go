package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReceiptSigningServerTLSConfigDevelopmentDoesNotEnableMTLS(t *testing.T) {
	t.Setenv("VROOLI_SCENARIO_DIR", "")
	config, err := receiptSigningServerTLSConfig()
	if err != nil {
		t.Fatalf("receiptSigningServerTLSConfig() error = %v", err)
	}
	if config != nil {
		t.Fatal("development lifecycle unexpectedly enabled receipt-signing mTLS")
	}
}

func TestReceiptSigningServerTLSConfigRejectsIncompleteProductionDeclaration(t *testing.T) {
	scenarioDir := t.TempDir()
	manifestDir := filepath.Join(scenarioDir, ".vrooli")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"trust_signing":{"provider":"vault-transit","operator_credential_file":"/run/operator-token"}}`
	if err := os.WriteFile(filepath.Join(manifestDir, "service.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_SCENARIO_DIR", scenarioDir)
	config, err := receiptSigningServerTLSConfig()
	if err == nil {
		t.Fatalf("receiptSigningServerTLSConfig() = %#v, nil error; want incomplete production declaration error", config)
	}
}
