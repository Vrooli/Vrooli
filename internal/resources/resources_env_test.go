package resources

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceEnvForResourceIncludesCanonicalStorageVariables(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	env := resourceEnvForResource(root, home, "home-assistant")
	values := map[string]string{}
	for _, entry := range env {
		for i := 0; i < len(entry); i++ {
			if entry[i] == '=' {
				values[entry[:i]] = entry[i+1:]
				break
			}
		}
	}

	if got := values["RESOURCE_CONFIG_DIR"]; got == "" {
		t.Fatal("expected RESOURCE_CONFIG_DIR")
	}
	if got := values["RESOURCE_DATA_DIR"]; got == "" {
		t.Fatal("expected RESOURCE_DATA_DIR")
	}
	wantRoot := filepath.Join(root, "resources", "home-assistant")
	if got := values["RESOURCE_ROOT"]; got != wantRoot {
		t.Fatalf("RESOURCE_ROOT = %q, want %q", got, wantRoot)
	}
}

func TestManagedResourceCLIReceivesProviderRuntimeContext(t *testing.T) {
	root := projectRootForResourcesTest(t)
	home := t.TempDir()
	store := filepath.Join(home, "signed-artifacts")
	t.Setenv("VROOLI_RESOURCE_ARTIFACT_DIR", store)
	writeExecutableOnPath(t, "resource-vault", "#!/usr/bin/env sh\nexit 0\n")
	controller := NewController(root, home)
	command, err := controller.commandForResource("vault", "status")
	if err != nil {
		t.Fatal(err)
	}
	values := managedServiceEnvValues(command.Env)
	if values["VROOLI_VAULT_RUNTIME"] != "managed" {
		t.Fatalf("VROOLI_VAULT_RUNTIME = %q, want managed", values["VROOLI_VAULT_RUNTIME"])
	}
	if values["VAULT_ADDR"] != "http://127.0.0.1:8200" {
		t.Fatalf("VAULT_ADDR = %q, want resolved local endpoint", values["VAULT_ADDR"])
	}
	artifact := values["VROOLI_MANAGED_SERVICE_ARTIFACT"]
	if !strings.HasPrefix(artifact, filepath.Join(store, "vault", "1.17.6")+string(filepath.Separator)) {
		t.Fatalf("managed artifact = %q, want signed artifact store path", artifact)
	}
}
