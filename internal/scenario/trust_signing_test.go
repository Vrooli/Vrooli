package scenario

import "testing"

func TestTrustSigningConfigRequiresMTLSFilesForOperatorRotation(t *testing.T) {
	config := TrustSigningConfig{
		Provider:               "vault-transit",
		Resource:               "vault",
		Address:                "https://vault.example.test",
		KeyName:                "prompt-manager-experiment-receipts",
		CredentialFile:         "/run/prompt-manager-token",
		OperatorCredentialFile: "/run/secrets-manager-operator-token",
		OperatorSubjects:       []string{"operator"},
	}
	dependencies := Dependencies{Resources: map[string]Dependency{"vault": {Type: "vault"}}}
	if err := config.Validate(dependencies); err == nil {
		t.Fatal("Validate() accepted operator rotation without mTLS file declarations")
	}
	config.OperatorTLSCertFile = "/run/tls/server.crt"
	config.OperatorTLSKeyFile = "/run/tls/server.key"
	config.OperatorTLSClientCAFile = "/run/tls/client-ca.pem"
	if err := config.Validate(dependencies); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
