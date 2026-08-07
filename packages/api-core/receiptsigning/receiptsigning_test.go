package receiptsigning

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type staticCredential string

func (s staticCredential) Credential(context.Context) (string, error) { return string(s), nil }

func TestDevelopmentSignerBindsPurposeAndCanonicalBytes(t *testing.T) {
	t.Parallel()
	signer := NewDevelopmentSigner()
	canonical := []byte("{\"receipt\":\"audit\"}")
	envelope, err := signer.Sign(context.Background(), PurposeExperimentAuditReceipt, canonical)
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if envelope.Version != EnvelopeVersionV1 || envelope.Purpose != PurposeExperimentAuditReceipt || envelope.Digest != Digest(canonical) {
		t.Fatalf("unexpected envelope: %#v", envelope)
	}
	if err := signer.Verify(context.Background(), envelope, canonical); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if err := signer.Verify(context.Background(), envelope, []byte("tampered")); err == nil {
		t.Fatal("Verify() accepted tampered canonical bytes")
	}
	envelope.Purpose = PurposeExperimentHoldoutReceipt
	if err := signer.Verify(context.Background(), envelope, canonical); err == nil {
		t.Fatal("Verify() accepted replay under a different purpose")
	}
}

func TestEnvelopeRejectsMalformedDigest(t *testing.T) {
	t.Parallel()
	err := (SignatureEnvelope{Version: EnvelopeVersionV1, Purpose: PurposeExperimentAuditReceipt, Algorithm: AlgorithmVaultTransit, KeyID: "key", Digest: "sha256:nope", Signature: "sig"}).Validate()
	if err == nil {
		t.Fatal("Validate() accepted malformed digest")
	}
}

func TestVaultTransitSignerUsesPurposeContextAndNeverReadsKey(t *testing.T) {
	t.Parallel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "workload-token" {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
		if strings.Contains(r.URL.Path, "/rotate") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if strings.Contains(r.URL.Path, "/keys/") {
			_, _ = io.WriteString(w, `{"data":{"latest_version":2,"deletion_allowed":false}}`)
			return
		}
		if strings.Contains(r.URL.Path, "/sign/") {
			_, _ = io.WriteString(w, `{"data":{"signature":"vault:v1:signature","key_version":2}}`)
			return
		}
		if strings.Contains(r.URL.Path, "/verify/") {
			_, _ = io.WriteString(w, `{"data":{"valid":true}}`)
			return
		}
		http.Error(w, "unexpected path", http.StatusNotFound)
	}))
	defer server.Close()
	signer, err := NewVaultTransitSigner(VaultTransitConfig{Address: server.URL, KeyName: "prompt-manager-experiment-receipts", Client: server.Client(), Credentials: staticCredential("workload-token"), AllowedPurposes: []Purpose{PurposeExperimentAuditReceipt}})
	if err != nil {
		t.Fatalf("NewVaultTransitSigner() error = %v", err)
	}
	envelope, err := signer.Sign(context.Background(), PurposeExperimentAuditReceipt, []byte("canonical"))
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if envelope.KeyID != "prompt-manager-experiment-receipts:v2" {
		t.Fatalf("key ID = %q", envelope.KeyID)
	}
	if err := signer.Verify(context.Background(), envelope, []byte("canonical")); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	envelope.KeyID = "different-key:v2"
	if err := signer.Verify(context.Background(), envelope, []byte("canonical")); err == nil {
		t.Fatal("Verify() accepted an envelope for a different Transit key")
	}
	health, err := signer.Health(context.Background())
	if err != nil || !health.Ready || !health.Production || !health.RotationOK {
		t.Fatalf("Health() = %#v, %v", health, err)
	}
	if _, err := signer.Sign(context.Background(), PurposeExperimentHoldoutReceipt, []byte("canonical")); err == nil {
		t.Fatal("Sign() accepted unauthorized purpose")
	}
	if _, err := signer.Rotate(context.Background()); err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}
}

func TestVaultTransitSignerRejectsInsecureAddress(t *testing.T) {
	_, err := NewVaultTransitSigner(VaultTransitConfig{Address: "http://vault.local", KeyName: "key", Credentials: staticCredential("token"), AllowedPurposes: []Purpose{PurposeExperimentAuditReceipt}})
	if err == nil || !strings.Contains(fmt.Sprint(err), "HTTPS") {
		t.Fatalf("insecure address error = %v", err)
	}
}

func TestRuntimeConfigBuildsProductionSignerWithoutEmbeddingCredential(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	credentialPath := filepath.Join(dir, "vault-token")
	if err := os.WriteFile(credentialPath, []byte("workload-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	signer, production, err := NewSignerFromRuntimeConfig(RuntimeConfig{Version: RuntimeConfigVersion, Mode: "vault-transit", VaultTransit: &VaultTransitRuntimeConfig{Address: "https://vault.example.test", KeyName: "prompt-manager-experiment-receipts", CredentialFile: credentialPath}})
	if err != nil || !production || signer == nil {
		t.Fatalf("NewSignerFromRuntimeConfig() = %#v, %t, %v", signer, production, err)
	}
	credential, err := (FileCredentialSource{Path: credentialPath}).Credential(context.Background())
	if err != nil || credential != "workload-token" {
		t.Fatalf("Credential() = %q, %v", credential, err)
	}
}

func TestRuntimeConfigAcceptsCredentialAuthorityMetadataWithoutProviderSecrets(t *testing.T) {
	t.Parallel()
	config := RuntimeConfig{
		Version: RuntimeConfigVersion,
		Mode:    ModeCredentialAuthorityEd25519,
		CredentialAuthority: &CredentialAuthorityRuntimeConfig{
			Identity: "vrooli/prompt-manager/experiment-receipts",
			Field:    "key-ring",
		},
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	config.VaultTransit = &VaultTransitRuntimeConfig{Address: "https://vault.example.test", KeyName: "legacy", CredentialFile: "/run/legacy-token"}
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() accepted active Vault Transit configuration alongside credential authority")
	}
}

func TestFileCredentialSourceRejectsBroadPermissions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "credential")
	if err := os.WriteFile(path, []byte("token"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := (FileCredentialSource{Path: path}).Credential(context.Background()); err == nil {
		t.Fatal("Credential() accepted world-readable file")
	}
}
