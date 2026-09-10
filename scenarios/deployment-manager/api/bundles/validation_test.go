package bundles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBundleSchemaSamplesValidate(t *testing.T) {
	samples := []string{
		"desktop-happy.json",
		"desktop-playwright.json",
	}
	for _, sample := range samples {
		t.Run(sample, func(t *testing.T) {
			path := filepath.Join("..", "..", "docs", "examples", "manifests", sample)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read sample %s: %v", sample, err)
			}
			if err := ValidateManifestBytes(data); err != nil {
				t.Fatalf("sample %s did not validate: %v", sample, err)
			}
		})
	}
}

func TestBundleSchemaRejectsInvalidManifest(t *testing.T) {
	invalid := []byte(`{"schema_version":"v0.1","target":"desktop","services":[]}`)
	if err := ValidateManifestBytes(invalid); err == nil {
		t.Fatalf("expected validation error for incomplete manifest")
	}
}

func TestBundleSchemaAcceptsAllocatorIPCPort(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "manifests", "desktop-happy.json"))
	if err != nil {
		t.Fatalf("failed to read valid bundle sample: %v", err)
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("failed to decode valid bundle sample: %v", err)
	}
	ipc, ok := manifest["ipc"].(map[string]interface{})
	if !ok {
		t.Fatal("valid bundle sample omitted ipc object")
	}
	ipc["port"] = 0
	data, err = json.Marshal(manifest)
	if err != nil {
		t.Fatalf("failed to encode allocator-port manifest: %v", err)
	}

	if err := ValidateManifestBytes(data); err != nil {
		t.Fatalf("allocator input port 0 should be valid: %v", err)
	}
}

func TestBundleValidationCoversSecretAndServiceRules(t *testing.T) {
	validSecret := ManifestSecret{ID: "token", Class: "user_prompt", Target: SecretTarget{Type: "env", Name: "TOKEN"}}
	if err := validateSecret(validSecret); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []ManifestSecret{
		{ID: "bad", Class: "unknown", Target: SecretTarget{Type: "env", Name: "TOKEN"}},
		{ID: "missing-target", Class: "user_prompt"},
		{ID: "bad-target", Class: "user_prompt", Target: SecretTarget{Type: "socket", Name: "x"}},
	} {
		if err := validateSecret(secret); err == nil {
			t.Errorf("expected secret validation failure for %+v", secret)
		}
	}
	base := ServiceEntry{
		ID: "api", Type: "api-binary", Binaries: map[string]ServiceBinary{"linux": {Path: "api"}},
		Health: HealthCheck{Type: "http"}, Readiness: ReadinessCheck{Type: "health_success"},
	}
	if err := validateService(base); err != nil {
		t.Fatal(err)
	}
	for _, svc := range []ServiceEntry{
		{},
		{ID: "bad-type", Type: "mystery", Health: base.Health, Readiness: base.Readiness},
		{ID: "no-binary", Type: "api-binary", Health: base.Health, Readiness: base.Readiness},
		{ID: "bad-binary", Type: "api-binary", Binaries: map[string]ServiceBinary{"linux": {}}, Health: base.Health, Readiness: base.Readiness},
		{ID: "no-health", Type: "api-binary", Binaries: base.Binaries, Readiness: base.Readiness},
		{ID: "no-readiness", Type: "api-binary", Binaries: base.Binaries, Health: base.Health},
	} {
		if err := validateService(svc); err == nil {
			t.Errorf("expected service validation failure for %+v", svc)
		}
	}
	if err := validateService(ServiceEntry{ID: "storage", Type: "embedded-storage"}); err == nil {
		// The service still needs health/readiness even when no binary is required.
		t.Fatal("embedded storage without health/readiness should fail")
	}
}

func TestBundleValidationAcceptsAllDeclaredAuthenticationModeProfiles(t *testing.T) {
	profile := &AuthenticationProfile{
		Version: 1, Mode: "personal_local", HumanSignIn: "disabled", Offline: true,
		ModeProfiles: map[string]AuthenticationModeProfile{
			"local_multi_user": {
				Provider: "scenario-authenticator", Resource: "demo", Audience: "scenario:demo",
				ProviderServiceID: "scenario-authenticator", HumanSignIn: "required", RequiresAuthenticator: true,
			},
			"remote_vrooli": {
				Provider: "scenario-authenticator", Resource: "demo", Audience: "scenario:demo",
				ProviderEndpoint: "https://auth.example.test", HumanSignIn: "required",
			},
			"shared_provider": {
				Provider: "landing-page-business-suite", Resource: "demo", Audience: "scenario:demo",
				ProviderEndpoint: "https://provider.example.test", LeasePath: "runtime/lease.json", HumanSignIn: "required",
			},
		},
	}
	services := []ServiceEntry{{ID: "api"}, {ID: "scenario-authenticator"}}
	if err := validateAuthentication(profile, services); err != nil {
		t.Fatalf("all declared authentication modes should validate: %v", err)
	}
	manifest := Manifest{
		SchemaVersion:  "v0.1",
		Target:         "desktop",
		App:            ManifestApp{Name: "demo", Version: "1.0.0"},
		IPC:            ManifestIPC{Mode: "loopback-http", Host: "127.0.0.1", Port: 48000, AuthTokenPath: "runtime/token"},
		Telemetry:      ManifestTelemetry{File: "runtime/telemetry.jsonl"},
		Authentication: profile,
		Services: []ServiceEntry{
			{ID: "api", Type: "api-binary", Binaries: map[string]ServiceBinary{"linux-x64": {Path: "bin/api"}}, Health: HealthCheck{Type: "http"}, Readiness: ReadinessCheck{Type: "health_success"}},
			{ID: "scenario-authenticator", Type: "api-binary", Binaries: map[string]ServiceBinary{"linux-x64": {Path: "bin/auth"}}, Health: HealthCheck{Type: "http"}, Readiness: ReadinessCheck{Type: "health_success"}},
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal authentication mode manifest: %v", err)
	}
	if err := ValidateManifestBytes(data); err != nil {
		t.Fatalf("serialized authentication mode manifest should validate: %v", err)
	}

	profile.ModeProfiles["local_multi_user"] = AuthenticationModeProfile{
		Provider: "scenario-authenticator", Resource: "demo", HumanSignIn: "required",
		ProviderServiceID: "missing-authenticator", RequiresAuthenticator: true,
	}
	if err := validateAuthentication(profile, services); err == nil {
		t.Fatal("missing alternate authenticator service was accepted")
	}
}
