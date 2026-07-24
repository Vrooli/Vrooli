package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/receiptsigning"
)

func TestGorillaMuxAdapterMountsTrailingSlashServicesAsPrefixes(t *testing.T) {
	router := mux.NewRouter()
	gorillaMuxAdapter{router: router}.Handle("/service/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/service/operation", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("service prefix status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestDiscoverScenarioNames(t *testing.T) {
	// Create a temporary directory structure mimicking scenarios/<name>/store
	tmpDir := t.TempDir()

	// scenarios/
	scenariosDir := filepath.Join(tmpDir, "scenarios")
	if err := os.Mkdir(scenariosDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create scenario directories
	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := os.Mkdir(filepath.Join(scenariosDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create a hidden directory (should be skipped)
	if err := os.Mkdir(filepath.Join(scenariosDir, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file (should be skipped)
	if err := os.WriteFile(filepath.Join(scenariosDir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	names := discoverScenarioNames(scenariosDir)

	expected := map[string]bool{"alpha": true, "beta": true, "gamma": true}
	if len(names) != len(expected) {
		t.Fatalf("expected %d scenario names, got %d: %v", len(expected), len(names), names)
	}
	for _, n := range names {
		if !expected[n] {
			t.Errorf("unexpected scenario name: %s", n)
		}
	}
}

func TestReceiptSignerFromLifecycleDeclarationUsesVaultTransit(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	credential := filepath.Join(root, "identity-token")
	if err := os.WriteFile(credential, []byte("workload-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"trust_signing":{"provider":"vault-transit","resource":"vault","address":"https://vault.example.test","key_name":"prompt-manager-experiment-receipts","credential_file":"` + credential + `"}}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_SCENARIO_DIR", root)
	signer, production, err := receiptSignerFromRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !production {
		t.Fatal("lifecycle Vault Transit declaration did not require production signing")
	}
	if _, ok := signer.(*receiptsigning.VaultTransitSigner); !ok {
		t.Fatalf("signer = %T, want VaultTransitSigner", signer)
	}
}

func TestDiscoverScenarioNames_NonexistentDir(t *testing.T) {
	names := discoverScenarioNames("/nonexistent/path/scenarios")
	if len(names) != 0 {
		t.Fatalf("expected 0 names for nonexistent dir, got %d", len(names))
	}
}

func TestResolveQdrantURL(t *testing.T) {
	// Save original env and restore after test
	origURL := os.Getenv("QDRANT_URL")
	origBase := os.Getenv("QDRANT_BASE_URL")
	origPort := os.Getenv("QDRANT_PORT")
	t.Cleanup(func() {
		os.Setenv("QDRANT_URL", origURL)
		os.Setenv("QDRANT_BASE_URL", origBase)
		os.Setenv("QDRANT_PORT", origPort)
	})

	tests := []struct {
		name     string
		envURL   string
		envBase  string
		envPort  string
		expected string
	}{
		{
			name:     "QDRANT_URL takes priority",
			envURL:   "http://qdrant:6333",
			envBase:  "http://localhost:6333",
			envPort:  "6333",
			expected: "http://qdrant:6333",
		},
		{
			name:     "falls back to QDRANT_BASE_URL",
			envURL:   "",
			envBase:  "http://localhost:6333",
			envPort:  "6333",
			expected: "http://localhost:6333",
		},
		{
			name:     "constructs from QDRANT_PORT",
			envURL:   "",
			envBase:  "",
			envPort:  "6333",
			expected: "http://localhost:6333",
		},
		{
			name:     "custom port from QDRANT_PORT",
			envURL:   "",
			envBase:  "",
			envPort:  "16333",
			expected: "http://localhost:16333",
		},
		{
			name:     "returns empty when nothing set",
			envURL:   "",
			envBase:  "",
			envPort:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("QDRANT_URL", tt.envURL)
			os.Setenv("QDRANT_BASE_URL", tt.envBase)
			os.Setenv("QDRANT_PORT", tt.envPort)

			got := resolveQdrantURL()
			if got != tt.expected {
				t.Errorf("resolveQdrantURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}
