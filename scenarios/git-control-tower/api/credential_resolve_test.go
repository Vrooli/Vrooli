package main

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResolveCredentialForRemote_StoredCredentialTakesPrecedence(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner().WithRemoteURL("git@github.com:user/repo.git")

	store := newTestCredStore(t)
	stored := StoredCredential{
		ID:         "cred-origin",
		Remote:     "origin",
		Type:       CredentialTypeSSH,
		SSHKeyPath: "/home/test/.ssh/custom_key",
	}
	if err := store.SaveCredential(stored); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	cred := resolveCredentialForRemote(context.Background(), fake, store, fake.RepoRoot, "origin")
	if cred == nil {
		t.Fatal("expected credential, got nil")
	}
	if cred.SSHKeyPath != "/home/test/.ssh/custom_key" {
		t.Fatalf("expected stored key path, got %s", cred.SSHKeyPath)
	}
}

func TestResolveCredentialForRemote_NilStoreAndHTTPSRemote(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner().WithRemoteURL("https://github.com/user/repo.git")

	cred := resolveCredentialForRemote(context.Background(), fake, nil, fake.RepoRoot, "origin")
	if cred != nil {
		t.Fatalf("expected nil credential for HTTPS remote without store, got %+v", cred)
	}
}

func TestResolveCredentialForRemote_SSHFallbackUsesDiscoveredKey(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner().WithRemoteURL("git@github.com:user/repo.git")

	// No stored credential — resolveCredentialForRemote should fall back to
	// SSH key auto-discovery. On this system, ~/.ssh/ contains keys, so the
	// fallback should return a synthetic credential with one of them.
	cred := resolveCredentialForRemote(context.Background(), fake, nil, fake.RepoRoot, "origin")

	// This test depends on the host having SSH keys in ~/.ssh/.
	// In CI or environments without keys, the fallback correctly returns nil.
	if cred == nil {
		t.Skip("no SSH keys discovered in ~/.ssh — expected in CI")
	}
	if cred.Type != CredentialTypeSSH {
		t.Fatalf("expected SSH credential type, got %s", cred.Type)
	}
	if cred.SSHKeyPath == "" {
		t.Fatal("expected non-empty SSHKeyPath in fallback credential")
	}
}

func TestResolveCredentialForRemote_NoFallbackForHTTPS(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner().WithRemoteURL("https://github.com/user/repo.git")

	// Even with SSH keys on disk, HTTPS remotes should NOT trigger SSH fallback
	cred := resolveCredentialForRemote(context.Background(), fake, nil, fake.RepoRoot, "origin")
	if cred != nil {
		t.Fatalf("expected nil credential for HTTPS remote, got %+v", cred)
	}
}

func TestResolveCredentialForRemote_DefaultsToOrigin(t *testing.T) {
	t.Parallel()

	fake := NewFakeGitRunner().WithRemoteURL("git@github.com:user/repo.git")

	store := newTestCredStore(t)
	stored := StoredCredential{
		ID:         "cred-origin",
		Remote:     "origin",
		Type:       CredentialTypeSSH,
		SSHKeyPath: "/home/test/.ssh/my_key",
	}
	if err := store.SaveCredential(stored); err != nil {
		t.Fatalf("save credential: %v", err)
	}

	// Empty remote should default to "origin"
	cred := resolveCredentialForRemote(context.Background(), fake, store, fake.RepoRoot, "")
	if cred == nil {
		t.Fatal("expected credential for default remote, got nil")
	}
	if cred.SSHKeyPath != "/home/test/.ssh/my_key" {
		t.Fatalf("expected stored key path, got %s", cred.SSHKeyPath)
	}
}

// newTestCredStore creates a CredentialsStore backed by a temp directory.
func newTestCredStore(t *testing.T) *CredentialsStore {
	t.Helper()
	store, err := NewCredentialsStore(filepath.Join(t.TempDir(), "creds.enc"))
	if err != nil {
		t.Fatalf("failed to create test cred store: %v", err)
	}
	return store
}
