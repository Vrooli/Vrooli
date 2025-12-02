package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func benchmarkServer() *APIServer {
	if logger == nil {
		logger = NewLogger("test")
	}
	return newAPIServer(nil, logger)
}

// BenchmarkHealthHandler benchmarks the health check endpoint
func BenchmarkHealthHandler(b *testing.B) {
	router := benchmarkServer().routes()
	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkVaultSecretsStatusHandler benchmarks the vault status endpoint
func BenchmarkVaultSecretsStatusHandler(b *testing.B) {
	router := benchmarkServer().routes()
	req, _ := http.NewRequest("GET", "/api/v1/vault/secrets/status", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkValidateHandler benchmarks the validation endpoint
func BenchmarkValidateHandler(b *testing.B) {
	router := benchmarkServer().routes()
	req, _ := http.NewRequest("GET", "/api/v1/secrets/validate", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkSecurityScanHandler benchmarks the security scan endpoint
func BenchmarkSecurityScanHandler(b *testing.B) {
	router := benchmarkServer().routes()
	req, _ := http.NewRequest("GET", "/api/v1/security/scan", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkProvisionHandler benchmarks the provision endpoint
func BenchmarkProvisionHandler(b *testing.B) {
	router := benchmarkServer().routes()
	provReq := ProvisionRequest{
		Secrets: map[string]string{"TEST_KEY": "test-value"},
	}
	body, _ := json.Marshal(provReq)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/secrets/provision", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
	}
}

// BenchmarkIsLikelySecret benchmarks secret detection
func BenchmarkIsLikelySecret(b *testing.B) {
	testVars := []string{
		"API_KEY",
		"DATABASE_URL",
		"SECRET_TOKEN",
		"PORT",
		"DEBUG",
		"PASSWORD",
		"TIMEOUT",
		"MAX_CONNECTIONS",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, varName := range testVars {
			_ = isLikelySecret(varName)
		}
	}
}

// BenchmarkClassifySecretType benchmarks secret type classification
func BenchmarkClassifySecretType(b *testing.B) {
	testVars := []string{
		"API_KEY",
		"DATABASE_PASSWORD",
		"AUTH_TOKEN",
		"API_ENDPOINT",
		"SSL_CERTIFICATE",
		"QUOTA_LIMIT",
		"CONFIG_VALUE",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, varName := range testVars {
			_ = classifySecretType(varName)
		}
	}
}

// BenchmarkIsLikelyRequired benchmarks required secret detection
func BenchmarkIsLikelyRequired(b *testing.B) {
	testVars := []string{
		"API_KEY",
		"PASSWORD",
		"SECRET",
		"DEBUG",
		"TIMEOUT",
		"LOG_LEVEL",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, varName := range testVars {
			_ = isLikelyRequired(varName)
		}
	}
}

// BenchmarkParseVaultCLIOutput benchmarks vault CLI output parsing
func BenchmarkParseVaultCLIOutput(b *testing.B) {
	output := `Resource: postgres
Status: Configured
Secrets Found: 5
- DATABASE_URL (configured)
- DB_PASSWORD (configured)
- DB_USER (configured)
- DB_PORT (configured)
- DB_NAME (configured)

Resource: openai
Status: Missing
Missing Secrets:
- OPENAI_API_KEY (required)
- OPENAI_ORG_ID (optional)
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseVaultCLIOutput(output, "")
	}
}

// BenchmarkParseVaultScanOutput benchmarks vault scan output parsing
func BenchmarkParseVaultScanOutput(b *testing.B) {
	output := `Scanning resources...
Found: postgres
Found: vault
Found: openai
Found: n8n
Found: redis
Found: qdrant
Found: ollama
Found: claude-code
Scan complete: 8 resources
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseVaultScanOutput(output)
	}
}

// BenchmarkParseVaultValidationOutput benchmarks validation output parsing
func BenchmarkParseVaultValidationOutput(b *testing.B) {
	output := `Validation Results:
Total Secrets: 25
Valid: 20
Invalid: 3
Missing: 2

Resource: postgres
- DATABASE_URL: valid
- DB_PASSWORD: valid
- DB_USER: valid

Resource: openai
- OPENAI_API_KEY: missing
- OPENAI_ORG_ID: valid
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseVaultValidationOutput(output)
	}
}

// BenchmarkScanResourceDirectory benchmarks resource directory scanning
func BenchmarkScanResourceDirectory(b *testing.B) {
	// Setup
	tempDir, _ := os.MkdirTemp("", "bench-secrets-manager")
	defer os.RemoveAll(tempDir)

	resourceDir := filepath.Join(tempDir, "test-resource")
	os.MkdirAll(resourceDir, 0755)

	// Create test files
	files := map[string]string{
		".env": `API_KEY=test
DATABASE_URL=postgresql://localhost
SECRET_TOKEN=abc123
PORT=8080
`,
		"config.yaml": `
api_key: test
database:
  password: secret
  host: localhost
`,
		"settings.json": `{
  "apiKey": "test",
  "secretToken": "abc123"
}`,
	}

	for name, content := range files {
		os.WriteFile(filepath.Join(resourceDir, name), []byte(content), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanResourceDirectory("test-resource", resourceDir)
	}
}

// BenchmarkIsTextFile benchmarks text file detection
func BenchmarkIsTextFile(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "bench-text-detection")
	defer os.RemoveAll(tempDir)

	textFile := filepath.Join(tempDir, "text.txt")
	os.WriteFile(textFile, []byte("This is a text file with some content"), 0644)

	binaryFile := filepath.Join(tempDir, "binary.bin")
	os.WriteFile(binaryFile, []byte{0xFF, 0xFE, 0x00, 0x01, 0x02, 0x03}, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = isTextFile(textFile)
		_ = isTextFile(binaryFile)
	}
}

// BenchmarkJSONEncoding benchmarks JSON encoding of responses
func BenchmarkJSONEncoding(b *testing.B) {
	response := VaultSecretsStatus{
		TotalResources:      10,
		ConfiguredResources: 7,
		MissingSecrets: []VaultMissingSecret{
			{
				ResourceName: "openai",
				SecretName:   "OPENAI_API_KEY",
				SecretPath:   "vault/openai/api_key",
				Required:     true,
				Description:  "OpenAI API key for AI features",
			},
		},
		ResourceStatuses: []VaultResourceStatus{
			{
				ResourceName:    "postgres",
				SecretsTotal:    5,
				SecretsFound:    5,
				SecretsMissing:  0,
				SecretsOptional: 0,
				HealthStatus:    "healthy",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(response)
	}
}

// BenchmarkConcurrentHealthChecks benchmarks concurrent health check requests
func BenchmarkConcurrentHealthChecks(b *testing.B) {
	router := benchmarkServer().routes()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkConcurrentValidation benchmarks concurrent validation requests
func BenchmarkConcurrentValidation(b *testing.B) {
	router := benchmarkServer().routes()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/v1/secrets/validate", nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
		}
	})
}

// BenchmarkEstimateFileCount benchmarks file count estimation
func BenchmarkEstimateFileCount(b *testing.B) {
	tempDir, _ := os.MkdirTemp("", "bench-file-count")
	defer os.RemoveAll(tempDir)

	scenariosDir := filepath.Join(tempDir, "scenarios")
	resourcesDir := filepath.Join(tempDir, "resources")
	os.MkdirAll(scenariosDir, 0755)
	os.MkdirAll(resourcesDir, 0755)

	// Create some test structure
	for i := 0; i < 10; i++ {
		dir := filepath.Join(resourcesDir, "resource-"+string(rune('a'+i)))
		os.MkdirAll(dir, 0755)
		for j := 0; j < 5; j++ {
			file := filepath.Join(dir, "file"+string(rune('0'+j))+".txt")
			os.WriteFile(file, []byte("content"), 0644)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = estimateFileCount(scenariosDir, resourcesDir, "")
	}
}

// Performance test with measurements
func TestPerformanceMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()
	server := benchmarkServer()
	router := server.routes()

	t.Run("HealthCheckResponseTime", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)

		// Warmup
		for i := 0; i < 10; i++ {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
		}

		// Measure
		iterations := 1000
		for i := 0; i < iterations; i++ {
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}
		}

		t.Logf("Completed %d health check requests", iterations)
	})

	t.Run("ValidationResponseTime", func(t *testing.T) {
		// Skip this test as it requires database/validator initialization
		t.Skip("Requires database/validator initialization - integration test only")
	})

	t.Run("SecretDetectionPerformance", func(t *testing.T) {
		testVars := []string{
			"API_KEY", "SECRET_TOKEN", "PASSWORD", "DATABASE_URL",
			"PORT", "DEBUG", "TIMEOUT", "MAX_CONNECTIONS",
		}

		iterations := 10000
		for i := 0; i < iterations; i++ {
			for _, varName := range testVars {
				_ = isLikelySecret(varName)
				_ = classifySecretType(varName)
				_ = isLikelyRequired(varName)
			}
		}

		t.Logf("Completed %d secret detection operations", iterations*len(testVars)*3)
	})
}
