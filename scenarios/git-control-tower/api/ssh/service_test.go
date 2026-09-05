package ssh

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestListKeys_EmptyDir(t *testing.T) {
	// Test with a non-existent directory path (platform returns valid path but dir doesn't exist)
	platform := &FakePlatform{
		SSHDirPath:  "/nonexistent/.ssh",
		HomeDirPath: "/nonexistent",
	}
	deps := SSHDeps{Platform: platform}

	result, err := ListKeys(context.Background(), deps)
	if err != nil {
		t.Fatalf("ListKeys() error = %v, want nil", err)
	}
	if result == nil {
		t.Fatal("ListKeys() returned nil result")
	}
	if len(result.Keys) != 0 {
		t.Errorf("ListKeys() keys = %d, want 0", len(result.Keys))
	}
	if result.SSHDir != "/nonexistent/.ssh" {
		t.Errorf("ListKeys() ssh_dir = %v, want /nonexistent/.ssh", result.SSHDir)
	}
}

func TestGenerateKeyService_InvalidType(t *testing.T) {
	platform := &FakePlatform{
		SSHDirPath:  "/tmp/test-ssh",
		HomeDirPath: "/tmp",
	}
	deps := SSHDeps{Platform: platform}

	result, err := GenerateKeyService(context.Background(), deps, GenerateKeyRequest{
		Type: "invalid",
	})
	if err != nil {
		t.Fatalf("GenerateKeyService() error = %v, want nil", err)
	}
	if result.Success {
		t.Error("GenerateKeyService() success = true, want false for invalid type")
	}
	if result.Error == "" {
		t.Error("GenerateKeyService() error message should not be empty")
	}
}

func TestGetPublicKeyService_EmptyPath(t *testing.T) {
	platform := &FakePlatform{
		SSHDirPath:  "/tmp/test-ssh",
		HomeDirPath: "/tmp",
	}
	deps := SSHDeps{Platform: platform}

	result, err := GetPublicKeyService(context.Background(), deps, GetPublicKeyRequest{
		KeyPath: "",
	})
	if err != nil {
		t.Fatalf("GetPublicKeyService() error = %v, want nil", err)
	}
	if result.Success {
		t.Error("GetPublicKeyService() success = true, want false for empty path")
	}
	if result.Error == "" {
		t.Error("GetPublicKeyService() error message should not be empty")
	}
}

func TestTestGitHubConnectionService_EmptyPath(t *testing.T) {
	platform := &FakePlatform{
		SSHDirPath:  "/tmp/test-ssh",
		HomeDirPath: "/tmp",
	}
	deps := SSHDeps{Platform: platform}

	result, err := TestGitHubConnectionService(context.Background(), deps, TestConnectionRequest{
		KeyPath: "",
	})
	if err != nil {
		t.Fatalf("TestGitHubConnectionService() error = %v, want nil", err)
	}
	if result.Success {
		t.Error("TestGitHubConnectionService() success = true, want false for empty path")
	}
	if result.Status != "missing_key_path" {
		t.Errorf("TestGitHubConnectionService() status = %v, want missing_key_path", result.Status)
	}
}

func TestDeleteKeyService_EmptyPath(t *testing.T) {
	platform := &FakePlatform{
		SSHDirPath:  "/tmp/test-ssh",
		HomeDirPath: "/tmp",
	}
	deps := SSHDeps{Platform: platform}

	result, err := DeleteKeyService(context.Background(), deps, DeleteKeyRequest{
		KeyPath: "",
	})
	if err != nil {
		t.Fatalf("DeleteKeyService() error = %v, want nil", err)
	}
	if result.Success {
		t.Error("DeleteKeyService() success = true, want false for empty path")
	}
	if result.Error == "" {
		t.Error("DeleteKeyService() error message should not be empty")
	}
}

func TestDeleteKeyService_ProtectedFile(t *testing.T) {
	platform := &FakePlatform{
		SSHDirPath:  "/home/user/.ssh",
		HomeDirPath: "/home/user",
	}
	deps := SSHDeps{Platform: platform}

	result, err := DeleteKeyService(context.Background(), deps, DeleteKeyRequest{
		KeyPath: "/home/user/.ssh/authorized_keys",
	})
	if err != nil {
		t.Fatalf("DeleteKeyService() error = %v, want nil", err)
	}
	if result.Success {
		t.Error("DeleteKeyService() success = true, want false for protected file")
	}
	if result.Error == "" {
		t.Error("DeleteKeyService() error message should not be empty")
	}
}

func TestDeleteKeyService_PathTraversal(t *testing.T) {
	platform := &FakePlatform{
		SSHDirPath:  "/home/user/.ssh",
		HomeDirPath: "/home/user",
	}
	deps := SSHDeps{Platform: platform}

	result, err := DeleteKeyService(context.Background(), deps, DeleteKeyRequest{
		KeyPath: "/home/user/.ssh/../.bashrc",
	})
	if err != nil {
		t.Fatalf("DeleteKeyService() error = %v, want nil", err)
	}
	if result.Success {
		t.Error("DeleteKeyService() success = true, want false for path traversal")
	}
}

func TestDeleteKeyService_DeletesPrivateAndPublicKey(t *testing.T) {
	homeDir := t.TempDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.Mkdir(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir ssh dir: %v", err)
	}
	keyPath := filepath.Join(sshDir, "id_ed25519")
	writeFile(t, keyPath, "private")
	writeFile(t, keyPath+".pub", "public")
	deps := SSHDeps{Platform: &FakePlatform{
		SSHDirPath:  sshDir,
		HomeDirPath: homeDir,
	}}

	result, err := DeleteKeyService(context.Background(), deps, DeleteKeyRequest{
		KeyPath: keyPath + ".pub",
	})
	if err != nil {
		t.Fatalf("DeleteKeyService() error = %v, want nil", err)
	}
	if !result.Success {
		t.Fatalf("Success = false, want true: %s", result.Error)
	}
	if !result.PrivateDeleted || !result.PublicDeleted {
		t.Fatalf("deleted private/public = %v/%v, want true/true", result.PrivateDeleted, result.PublicDeleted)
	}
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("private key still exists or stat failed unexpectedly: %v", err)
	}
	if _, err := os.Stat(keyPath + ".pub"); !os.IsNotExist(err) {
		t.Fatalf("public key still exists or stat failed unexpectedly: %v", err)
	}
}

func TestDeleteKeyService_KeyFilesNotFound(t *testing.T) {
	homeDir := t.TempDir()
	sshDir := filepath.Join(homeDir, ".ssh")
	if err := os.Mkdir(sshDir, 0o700); err != nil {
		t.Fatalf("mkdir ssh dir: %v", err)
	}
	deps := SSHDeps{Platform: &FakePlatform{
		SSHDirPath:  sshDir,
		HomeDirPath: homeDir,
	}}

	result, err := DeleteKeyService(context.Background(), deps, DeleteKeyRequest{
		KeyPath: filepath.Join(sshDir, "missing"),
	})
	if err != nil {
		t.Fatalf("DeleteKeyService() error = %v, want nil", err)
	}
	if result.Success {
		t.Fatal("Success = true, want false")
	}
	if result.Error != "Key files not found" {
		t.Fatalf("Error = %q, want Key files not found", result.Error)
	}
}
