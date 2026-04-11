package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPortRegistryReadsTypedJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scripts", "resources", "port_registry.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("{\n  \"resource_ports\": {\"postgres\": 5433},\n  \"reserved_ranges\": {\"db\": \"5432-5499\"}\n}\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	registry, err := LoadPortRegistry(root)
	if err != nil {
		t.Fatalf("LoadPortRegistry: %v", err)
	}
	if got := registry.ResourcePorts["postgres"]; got != 5433 {
		t.Fatalf("postgres port = %d, want 5433", got)
	}
	if got := registry.ReservedRanges["db"]; got != "5432-5499" {
		t.Fatalf("reserved range = %q, want 5432-5499", got)
	}
}

func TestLoadResourceEnvironmentUsesTypedDefaultsAndSecrets(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	writeJSON(t, filepath.Join(root, "scripts", "resources", "port_registry.json"), `{
  "resource_ports": {
    "browserless": 4110,
    "postgres": 5433
  },
  "reserved_ranges": {}
}`)
	writeJSON(t, filepath.Join(root, ".vrooli", "schemas", "resource-definitions.json"), `{
  "definitions": {
    "resourceSchemas": {
      "browserless": {
        "properties": {
          "port": { "default": 4110 },
          "logging": {
            "properties": {
              "level": { "default": "info" }
            }
          }
        }
      },
      "postgres": {
        "properties": {}
      }
    }
  }
}`)
	writeJSON(t, filepath.Join(root, ".vrooli", "secrets.json"), `{
  "POSTGRES_PASSWORD": "secret",
  "POSTGRES_USER": "vrooli",
  "BROWSERLESS_TOKEN": "abc123"
}`)

	postgresEnv, err := LoadResourceEnvironment(root, home, "postgres")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(postgres): %v", err)
	}
	if got := postgresEnv["POSTGRES_PORT"]; got != "5433" {
		t.Fatalf("POSTGRES_PORT = %q, want 5433", got)
	}
	if got := postgresEnv["POSTGRES_HOST"]; got != "localhost" {
		t.Fatalf("POSTGRES_HOST = %q, want localhost", got)
	}
	if got := postgresEnv["POSTGRES_PASSWORD"]; got != "secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want secret", got)
	}

	browserlessEnv, err := LoadResourceEnvironment(root, home, "browserless")
	if err != nil {
		t.Fatalf("LoadResourceEnvironment(browserless): %v", err)
	}
	if got := browserlessEnv["BROWSERLESS_PORT"]; got != "4110" {
		t.Fatalf("BROWSERLESS_PORT = %q, want 4110", got)
	}
	if got := browserlessEnv["BROWSERLESS_BASE_URL"]; got != "http://localhost:4110" {
		t.Fatalf("BROWSERLESS_BASE_URL = %q", got)
	}
	if got := browserlessEnv["BROWSERLESS_LOGGING_LEVEL"]; got != "info" {
		t.Fatalf("BROWSERLESS_LOGGING_LEVEL = %q, want info", got)
	}
	if got := browserlessEnv["BROWSERLESS_TOKEN"]; got != "abc123" {
		t.Fatalf("BROWSERLESS_TOKEN = %q, want abc123", got)
	}
}

func writeJSON(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents+"\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
