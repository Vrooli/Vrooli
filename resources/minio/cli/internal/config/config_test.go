package config

import "testing"

func TestDefaultsAreTypedAndStable(t *testing.T) {
	got := Defaults()
	if got.APIPort != 9000 || got.ConsolePort != 9001 || got.Region != "us-east-1" {
		t.Fatalf("unexpected MinIO defaults: %#v", got)
	}
	if got.RootUser == "" || got.RootPassword == "" {
		t.Fatal("credential defaults must be explicit")
	}
}

func TestDefaultMessages(t *testing.T) {
	got := DefaultMessages()
	if got.Healthy == "" || got.HealthCheckFailed == "" {
		t.Fatalf("unexpected messages: %#v", got)
	}
}

func TestLoadPreservesExistingOperatorValues(t *testing.T) {
	values := map[string]string{
		"MINIO_HOST":            "10.0.0.8",
		"RESOURCE_DATA_DIR":     "/operator/data",
		"RESOURCE_CONFIG_DIR":   "/operator/config",
		"MINIO_ROOT_USER":       "operator",
		"MINIO_ROOT_PASSWORD":   "preserve-me",
		"RESOURCE_PORT_API":     "19000",
		"RESOURCE_PORT_CONSOLE": "19001",
	}
	cfg := Load(func(key string) string { return values[key] })
	if cfg.APIHost != "10.0.0.8" || cfg.DataDir != "/operator/data" || cfg.ConfigDir != "/operator/config" || cfg.RootUser != "operator" || cfg.RootPassword != "preserve-me" || cfg.APIPort != 19000 || cfg.ConsolePort != 19001 {
		t.Fatalf("operator config was not preserved: %#v", cfg)
	}
}
