package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleSchemaSamplesValidate(t *testing.T) {
	samples := []string{
		"desktop-happy.json",
		"desktop-playwright.json",
	}
	for _, sample := range samples {
		t.Run(sample, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "..", "..", "docs", "deployment", "examples", "manifests", sample)
			if _, err := os.Stat(path); err != nil {
				t.Skipf("bundle schema sample not present in this checkout: %s (%v)", path, err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read sample %s: %v", sample, err)
			}
			if err := validateDesktopBundleManifestBytes(data); err != nil {
				t.Fatalf("sample %s did not validate: %v", sample, err)
			}
		})
	}
}

func TestBundleSchemaRejectsInvalidManifest(t *testing.T) {
	invalid := []byte(`{"schema_version":"v0.1","target":"desktop","services":[]}`)
	if err := validateDesktopBundleManifestBytes(invalid); err == nil {
		t.Fatalf("expected validation error for incomplete manifest")
	}
}

func TestValidateDesktopBundleManifestBytes(t *testing.T) {
	// Helper to create a valid base manifest
	validManifest := func() map[string]interface{} {
		return map[string]interface{}{
			"schema_version": "v0.1",
			"target":         "desktop",
			"app": map[string]interface{}{
				"name":        "test-app",
				"version":     "1.0.0",
				"description": "Test application",
			},
			"ipc": map[string]interface{}{
				"mode":            "loopback-http",
				"host":            "127.0.0.1",
				"port":            9876,
				"auth_token_path": "auth.token",
			},
			"telemetry": map[string]interface{}{
				"file":       "telemetry.jsonl",
				"upload_url": "",
			},
			"services": []interface{}{
				map[string]interface{}{
					"id":          "test-api",
					"type":        "api-binary",
					"description": "Test API service",
					"binaries": map[string]interface{}{
						"linux-x64": map[string]interface{}{
							"path": "bin/test-api",
						},
					},
					"health": map[string]interface{}{
						"type":        "http",
						"path":        "/health",
						"port_name":   "api",
						"interval_ms": 5000,
						"timeout_ms":  3000,
						"retries":     3,
					},
					"readiness": map[string]interface{}{
						"type":       "health_success",
						"port_name":  "api",
						"pattern":    "",
						"timeout_ms": 30000,
					},
				},
			},
		}
	}

	t.Run("InvalidJSON", func(t *testing.T) {
		err := validateDesktopBundleManifestBytes([]byte(`{invalid json`))
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		if !strings.Contains(err.Error(), "invalid JSON") {
			t.Errorf("expected error to mention invalid JSON, got: %v", err)
		}
	})

	t.Run("UnknownFields", func(t *testing.T) {
		manifest := validManifest()
		manifest["unknown_field"] = "unexpected"
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for unknown fields")
		}
		if !strings.Contains(err.Error(), "unexpected fields") {
			t.Errorf("expected error to mention unexpected fields, got: %v", err)
		}
	})

	t.Run("WrongSchemaVersion", func(t *testing.T) {
		manifest := validManifest()
		manifest["schema_version"] = "v0.2"
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for wrong schema version")
		}
		if !strings.Contains(err.Error(), "schema_version must be v0.1") {
			t.Errorf("expected schema version error, got: %v", err)
		}
	})

	t.Run("WrongTarget", func(t *testing.T) {
		manifest := validManifest()
		manifest["target"] = "mobile"
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for wrong target")
		}
		if !strings.Contains(err.Error(), "target must be desktop") {
			t.Errorf("expected target error, got: %v", err)
		}
	})

	t.Run("MissingAppName", func(t *testing.T) {
		manifest := validManifest()
		manifest["app"] = map[string]interface{}{
			"name":        "",
			"version":     "1.0.0",
			"description": "Test",
		}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for missing app name")
		}
		if !strings.Contains(err.Error(), "app.name") {
			t.Errorf("expected app.name error, got: %v", err)
		}
	})

	t.Run("MissingAppVersion", func(t *testing.T) {
		manifest := validManifest()
		manifest["app"] = map[string]interface{}{
			"name":        "test",
			"version":     "",
			"description": "Test",
		}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for missing app version")
		}
		if !strings.Contains(err.Error(), "app.version") {
			t.Errorf("expected app.version error, got: %v", err)
		}
	})

	t.Run("InvalidIPCMode", func(t *testing.T) {
		manifest := validManifest()
		manifest["ipc"] = map[string]interface{}{
			"mode":            "unix-socket",
			"host":            "127.0.0.1",
			"port":            9876,
			"auth_token_path": "auth.token",
		}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for invalid IPC mode")
		}
		if !strings.Contains(err.Error(), "loopback-http") {
			t.Errorf("expected IPC mode error, got: %v", err)
		}
	})

	t.Run("MissingIPCHost", func(t *testing.T) {
		manifest := validManifest()
		manifest["ipc"] = map[string]interface{}{
			"mode":            "loopback-http",
			"host":            "",
			"port":            9876,
			"auth_token_path": "auth.token",
		}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for missing IPC host")
		}
	})

	t.Run("MissingIPCPort", func(t *testing.T) {
		manifest := validManifest()
		manifest["ipc"] = map[string]interface{}{
			"mode":            "loopback-http",
			"host":            "127.0.0.1",
			"port":            0,
			"auth_token_path": "auth.token",
		}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for missing IPC port")
		}
	})

	t.Run("MissingIPCAuthTokenPath", func(t *testing.T) {
		manifest := validManifest()
		manifest["ipc"] = map[string]interface{}{
			"mode":            "loopback-http",
			"host":            "127.0.0.1",
			"port":            9876,
			"auth_token_path": "",
		}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for missing IPC auth_token_path")
		}
	})

	t.Run("MissingTelemetryFile", func(t *testing.T) {
		manifest := validManifest()
		manifest["telemetry"] = map[string]interface{}{
			"file":       "",
			"upload_url": "",
		}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for missing telemetry file")
		}
		if !strings.Contains(err.Error(), "telemetry.file") {
			t.Errorf("expected telemetry.file error, got: %v", err)
		}
	})

	t.Run("EmptyServices", func(t *testing.T) {
		manifest := validManifest()
		manifest["services"] = []interface{}{}
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for empty services")
		}
		if !strings.Contains(err.Error(), "at least one service") {
			t.Errorf("expected services error, got: %v", err)
		}
	})
}

func TestValidateSecret(t *testing.T) {
	tests := []struct {
		name        string
		secret      manifestSecret
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidPerInstallGenerated",
			secret: manifestSecret{
				ID:          "auth-token",
				Class:       "per_install_generated",
				Description: "Auto-generated auth token",
				Format:      "hex:32",
				Target: manifestSecretTarget{
					Type: "env",
					Name: "AUTH_TOKEN",
				},
			},
			expectError: false,
		},
		{
			name: "ValidUserPrompt",
			secret: manifestSecret{
				ID:          "api-key",
				Class:       "user_prompt",
				Description: "User API key",
				Format:      "string",
				Target: manifestSecretTarget{
					Type: "env",
					Name: "API_KEY",
				},
				Prompt: &manifestSecretPrompt{
					Label:       "API Key",
					Description: "Enter your API key",
				},
			},
			expectError: false,
		},
		{
			name: "ValidRemoteFetch",
			secret: manifestSecret{
				ID:          "remote-secret",
				Class:       "remote_fetch",
				Description: "Remotely fetched secret",
				Format:      "string",
				Target: manifestSecretTarget{
					Type: "file",
					Name: "secrets/remote.txt",
				},
			},
			expectError: false,
		},
		{
			name: "ValidInfrastructure",
			secret: manifestSecret{
				ID:          "db-password",
				Class:       "infrastructure",
				Description: "Database password",
				Format:      "string",
				Target: manifestSecretTarget{
					Type: "env",
					Name: "DB_PASSWORD",
				},
			},
			expectError: false,
		},
		{
			name: "ValidEmptyClass",
			secret: manifestSecret{
				ID:          "legacy-secret",
				Class:       "",
				Description: "Legacy secret with no class",
				Format:      "string",
				Target: manifestSecretTarget{
					Type: "env",
					Name: "LEGACY",
				},
			},
			expectError: false,
		},
		{
			name: "InvalidClass",
			secret: manifestSecret{
				ID:          "bad-secret",
				Class:       "unknown_class",
				Description: "Secret with invalid class",
				Format:      "string",
				Target: manifestSecretTarget{
					Type: "env",
					Name: "BAD",
				},
			},
			expectError: true,
			errorMsg:    "unsupported class",
		},
		{
			name: "MissingTargetType",
			secret: manifestSecret{
				ID:          "no-target-type",
				Class:       "per_install_generated",
				Description: "Secret missing target type",
				Format:      "string",
				Target: manifestSecretTarget{
					Type: "",
					Name: "VAR_NAME",
				},
			},
			expectError: true,
			errorMsg:    "target.type",
		},
		{
			name: "MissingTargetName",
			secret: manifestSecret{
				ID:          "no-target-name",
				Class:       "per_install_generated",
				Description: "Secret missing target name",
				Format:      "string",
				Target: manifestSecretTarget{
					Type: "env",
					Name: "",
				},
			},
			expectError: true,
			errorMsg:    "target.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecret(tt.secret)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorMsg)
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateService(t *testing.T) {
	tests := []struct {
		name        string
		service     manifestServiceEntry
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidAPIBinary",
			service: manifestServiceEntry{
				ID:          "api-service",
				Type:        "api-binary",
				Description: "API service",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {Path: "bin/api"},
				},
				Health: manifestHealth{
					Type:     "http",
					Path:     "/health",
					PortName: "api",
					Interval: 5000,
					Timeout:  3000,
					Retries:  3,
				},
				Readiness: manifestReadiness{
					Type:     "health_success",
					PortName: "api",
					Timeout:  30000,
				},
			},
			expectError: false,
		},
		{
			name: "ValidUIBundle",
			service: manifestServiceEntry{
				ID:          "ui-service",
				Type:        "ui-bundle",
				Description: "UI bundle",
				Binaries: map[string]manifestServiceBinary{
					"darwin-arm64": {Path: "ui/dist"},
				},
				Health: manifestHealth{
					Type:     "tcp",
					PortName: "ui",
					Interval: 5000,
					Timeout:  3000,
					Retries:  3,
				},
				Readiness: manifestReadiness{
					Type:     "port_open",
					PortName: "ui",
					Timeout:  30000,
				},
			},
			expectError: false,
		},
		{
			name: "ValidWorker",
			service: manifestServiceEntry{
				ID:          "worker-service",
				Type:        "worker",
				Description: "Background worker",
				Binaries: map[string]manifestServiceBinary{
					"windows-amd64": {Path: "bin/worker.exe"},
				},
				Health: manifestHealth{
					Type:    "command",
					Command: []string{"./health-check.sh"},
				},
				Readiness: manifestReadiness{
					Type:    "log_match",
					Pattern: "Worker started",
					Timeout: 60000,
				},
			},
			expectError: false,
		},
		{
			name: "ValidResource",
			service: manifestServiceEntry{
				ID:          "db-resource",
				Type:        "resource",
				Description: "Database resource",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {Path: "bin/sqlite"},
				},
				Health: manifestHealth{
					Type:    "tcp",
					Timeout: 3000,
				},
				Readiness: manifestReadiness{
					Type:    "port_open",
					Timeout: 10000,
				},
			},
			expectError: false,
		},
		{
			name: "MissingServiceID",
			service: manifestServiceEntry{
				ID:          "",
				Type:        "api-binary",
				Description: "Service without ID",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {Path: "bin/api"},
				},
				Health: manifestHealth{
					Type: "http",
				},
				Readiness: manifestReadiness{
					Type: "health_success",
				},
			},
			expectError: true,
			errorMsg:    "id is required",
		},
		{
			name: "InvalidServiceType",
			service: manifestServiceEntry{
				ID:          "bad-service",
				Type:        "unknown-type",
				Description: "Service with invalid type",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {Path: "bin/bad"},
				},
				Health: manifestHealth{
					Type: "http",
				},
				Readiness: manifestReadiness{
					Type: "health_success",
				},
			},
			expectError: true,
			errorMsg:    "not supported",
		},
		{
			name: "MissingBinaries",
			service: manifestServiceEntry{
				ID:          "no-binaries",
				Type:        "api-binary",
				Description: "Service without binaries",
				Binaries:    map[string]manifestServiceBinary{},
				Health: manifestHealth{
					Type: "http",
				},
				Readiness: manifestReadiness{
					Type: "health_success",
				},
			},
			expectError: true,
			errorMsg:    "at least one platform binary",
		},
		{
			name: "EmptyBinaryPath",
			service: manifestServiceEntry{
				ID:          "empty-path",
				Type:        "api-binary",
				Description: "Service with empty binary path",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {Path: ""},
				},
				Health: manifestHealth{
					Type: "http",
				},
				Readiness: manifestReadiness{
					Type: "health_success",
				},
			},
			expectError: true,
			errorMsg:    "binary path is required",
		},
		{
			name: "MissingHealthType",
			service: manifestServiceEntry{
				ID:          "no-health",
				Type:        "api-binary",
				Description: "Service without health type",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {Path: "bin/api"},
				},
				Health: manifestHealth{
					Type: "",
				},
				Readiness: manifestReadiness{
					Type: "health_success",
				},
			},
			expectError: true,
			errorMsg:    "health.type is required",
		},
		{
			name: "MissingReadinessType",
			service: manifestServiceEntry{
				ID:          "no-readiness",
				Type:        "api-binary",
				Description: "Service without readiness type",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {Path: "bin/api"},
				},
				Health: manifestHealth{
					Type: "http",
				},
				Readiness: manifestReadiness{
					Type: "",
				},
			},
			expectError: true,
			errorMsg:    "readiness.type is required",
		},
		{
			name: "MultiplePlatforms",
			service: manifestServiceEntry{
				ID:          "multi-platform",
				Type:        "api-binary",
				Description: "Service with multiple platforms",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64":   {Path: "bin/api-linux"},
					"darwin-arm64":  {Path: "bin/api-darwin"},
					"windows-amd64": {Path: "bin/api.exe"},
				},
				Health: manifestHealth{
					Type:     "http",
					Path:     "/health",
					PortName: "api",
				},
				Readiness: manifestReadiness{
					Type:     "health_success",
					PortName: "api",
				},
			},
			expectError: false,
		},
		{
			name: "ServiceWithAllOptionalFields",
			service: manifestServiceEntry{
				ID:          "full-service",
				Type:        "api-binary",
				Description: "Service with all optional fields",
				Binaries: map[string]manifestServiceBinary{
					"linux-amd64": {
						Path: "bin/api",
						Args: []string{"--port", "8080"},
						Env:  map[string]string{"LOG_LEVEL": "debug"},
						Cwd:  "/app",
					},
				},
				Env:          map[string]string{"SHARED_VAR": "value"},
				Secrets:      []string{"api-key", "db-password"},
				DataDirs:     []string{"data/cache", "data/uploads"},
				LogDir:       "logs",
				Dependencies: []string{"db-resource"},
				Migrations: []manifestMigration{
					{
						Version: "1.0.0",
						Command: []string{"./migrate.sh", "up"},
						Env:     map[string]string{"MIGRATE_DIR": "migrations"},
						RunOn:   "upgrade",
					},
				},
				Assets: []manifestAsset{
					{
						Path:      "assets/model.bin",
						SHA256:    "abc123",
						SizeBytes: 1024,
					},
				},
				GPU: &manifestGPU{
					Requirement: "optional_with_cpu_fallback",
				},
				Critical: boolPtr(true),
				Health: manifestHealth{
					Type:     "http",
					Path:     "/health",
					PortName: "api",
					Interval: 5000,
					Timeout:  3000,
					Retries:  3,
				},
				Readiness: manifestReadiness{
					Type:     "health_success",
					PortName: "api",
					Pattern:  "",
					Timeout:  30000,
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateService(tt.service)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorMsg)
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateDesktopBundleManifestWithSecrets(t *testing.T) {
	// Helper to create a valid base manifest with secrets
	validManifestWithSecrets := func(secrets []map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"schema_version": "v0.1",
			"target":         "desktop",
			"app": map[string]interface{}{
				"name":        "test-app",
				"version":     "1.0.0",
				"description": "Test application",
			},
			"ipc": map[string]interface{}{
				"mode":            "loopback-http",
				"host":            "127.0.0.1",
				"port":            9876,
				"auth_token_path": "auth.token",
			},
			"telemetry": map[string]interface{}{
				"file":       "telemetry.jsonl",
				"upload_url": "",
			},
			"secrets": secrets,
			"services": []interface{}{
				map[string]interface{}{
					"id":          "test-api",
					"type":        "api-binary",
					"description": "Test API",
					"binaries": map[string]interface{}{
						"linux-x64": map[string]interface{}{
							"path": "bin/test-api",
						},
					},
					"health": map[string]interface{}{
						"type":        "http",
						"path":        "/health",
						"port_name":   "api",
						"interval_ms": 5000,
						"timeout_ms":  3000,
						"retries":     3,
					},
					"readiness": map[string]interface{}{
						"type":       "health_success",
						"port_name":  "api",
						"pattern":    "",
						"timeout_ms": 30000,
					},
				},
			},
		}
	}

	t.Run("ValidSecrets", func(t *testing.T) {
		secrets := []map[string]interface{}{
			{
				"id":          "auth-token",
				"class":       "per_install_generated",
				"description": "Auth token",
				"format":      "hex:32",
				"target": map[string]interface{}{
					"type": "env",
					"name": "AUTH_TOKEN",
				},
			},
		}
		manifest := validManifestWithSecrets(secrets)
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("InvalidSecretClass", func(t *testing.T) {
		secrets := []map[string]interface{}{
			{
				"id":          "bad-secret",
				"class":       "invalid_class",
				"description": "Bad secret",
				"format":      "string",
				"target": map[string]interface{}{
					"type": "env",
					"name": "BAD",
				},
			},
		}
		manifest := validManifestWithSecrets(secrets)
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for invalid secret class")
		}
		if !strings.Contains(err.Error(), "unsupported class") {
			t.Errorf("expected unsupported class error, got: %v", err)
		}
	})

	t.Run("SecretMissingTarget", func(t *testing.T) {
		secrets := []map[string]interface{}{
			{
				"id":          "no-target",
				"class":       "per_install_generated",
				"description": "Secret without target",
				"format":      "string",
				"target": map[string]interface{}{
					"type": "",
					"name": "",
				},
			},
		}
		manifest := validManifestWithSecrets(secrets)
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err == nil {
			t.Fatal("expected error for secret missing target")
		}
	})
}

func TestValidateDesktopBundleManifestWithSwaps(t *testing.T) {
	// Helper to create a manifest with swaps
	validManifestWithSwaps := func(swaps []map[string]interface{}) map[string]interface{} {
		return map[string]interface{}{
			"schema_version": "v0.1",
			"target":         "desktop",
			"app": map[string]interface{}{
				"name":        "test-app",
				"version":     "1.0.0",
				"description": "Test application",
			},
			"ipc": map[string]interface{}{
				"mode":            "loopback-http",
				"host":            "127.0.0.1",
				"port":            9876,
				"auth_token_path": "auth.token",
			},
			"telemetry": map[string]interface{}{
				"file":       "telemetry.jsonl",
				"upload_url": "",
			},
			"swaps": swaps,
			"services": []interface{}{
				map[string]interface{}{
					"id":          "test-api",
					"type":        "api-binary",
					"description": "Test API",
					"binaries": map[string]interface{}{
						"linux-x64": map[string]interface{}{
							"path": "bin/test-api",
						},
					},
					"health": map[string]interface{}{
						"type":        "http",
						"path":        "/health",
						"port_name":   "api",
						"interval_ms": 5000,
						"timeout_ms":  3000,
						"retries":     3,
					},
					"readiness": map[string]interface{}{
						"type":       "health_success",
						"port_name":  "api",
						"pattern":    "",
						"timeout_ms": 30000,
					},
				},
			},
		}
	}

	t.Run("ValidSwaps", func(t *testing.T) {
		swaps := []map[string]interface{}{
			{
				"original":    "postgres",
				"replacement": "sqlite",
				"reason":      "Desktop bundle compatibility",
				"limitations": "No concurrent writes",
			},
		}
		manifest := validManifestWithSwaps(swaps)
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("MultipleSwaps", func(t *testing.T) {
		swaps := []map[string]interface{}{
			{
				"original":    "postgres",
				"replacement": "sqlite",
				"reason":      "Desktop bundle compatibility",
				"limitations": "No concurrent writes",
			},
			{
				"original":    "redis",
				"replacement": "in-memory-cache",
				"reason":      "Offline support",
				"limitations": "No persistence",
			},
		}
		manifest := validManifestWithSwaps(swaps)
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestValidateDesktopBundleManifestWithPorts(t *testing.T) {
	// Helper to create a manifest with ports configuration
	validManifestWithPorts := func(ports map[string]interface{}) map[string]interface{} {
		manifest := map[string]interface{}{
			"schema_version": "v0.1",
			"target":         "desktop",
			"app": map[string]interface{}{
				"name":        "test-app",
				"version":     "1.0.0",
				"description": "Test application",
			},
			"ipc": map[string]interface{}{
				"mode":            "loopback-http",
				"host":            "127.0.0.1",
				"port":            9876,
				"auth_token_path": "auth.token",
			},
			"telemetry": map[string]interface{}{
				"file":       "telemetry.jsonl",
				"upload_url": "",
			},
			"services": []interface{}{
				map[string]interface{}{
					"id":          "test-api",
					"type":        "api-binary",
					"description": "Test API",
					"binaries": map[string]interface{}{
						"linux-x64": map[string]interface{}{
							"path": "bin/test-api",
						},
					},
					"health": map[string]interface{}{
						"type":        "http",
						"path":        "/health",
						"port_name":   "api",
						"interval_ms": 5000,
						"timeout_ms":  3000,
						"retries":     3,
					},
					"readiness": map[string]interface{}{
						"type":       "health_success",
						"port_name":  "api",
						"pattern":    "",
						"timeout_ms": 30000,
					},
				},
			},
		}
		if ports != nil {
			manifest["ports"] = ports
		}
		return manifest
	}

	t.Run("ValidPortsConfiguration", func(t *testing.T) {
		ports := map[string]interface{}{
			"default_range": map[string]interface{}{
				"min": 10000,
				"max": 20000,
			},
			"reserved": []interface{}{8080, 3000, 5432},
		}
		manifest := validManifestWithPorts(ports)
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("NilPortsConfiguration", func(t *testing.T) {
		manifest := validManifestWithPorts(nil)
		data, _ := json.Marshal(manifest)
		err := validateDesktopBundleManifestBytes(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
