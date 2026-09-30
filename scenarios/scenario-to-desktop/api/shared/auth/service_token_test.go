package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeploymentManagerServiceTokenPrefersInjectedEnvironment(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN", "env-token")
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN_FILE", filepath.Join(t.TempDir(), "token"))
	if got := DeploymentManagerServiceToken(); got != "env-token" {
		t.Fatalf("token = %q, want env-token", got)
	}
}

func TestDeploymentManagerServiceTokenReadsConfiguredFile(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN", "")
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte("file-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN_FILE", path)
	if got := DeploymentManagerServiceToken(); got != "file-token" {
		t.Fatalf("token = %q, want file-token", got)
	}
}

func TestDeploymentManagerServiceTokenDoesNotInventCredential(t *testing.T) {
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN", "")
	t.Setenv("DEPLOYMENT_MANAGER_SERVICE_TOKEN_FILE", filepath.Join(t.TempDir(), "missing"))
	if got := DeploymentManagerServiceToken(); got != "" {
		t.Fatalf("token = %q, want empty", got)
	}
}
