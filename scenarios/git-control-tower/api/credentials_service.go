package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// CredentialsDeps contains dependencies for credential operations.
type CredentialsDeps struct {
	Git       GitRunner
	RepoDir   string
	Store     *CredentialsStore
	StorePath string // Optional override for store path
}

// ListCredentials retrieves all stored credentials for the repository's remotes.
func ListCredentials(ctx context.Context, deps CredentialsDeps) (*CredentialsListResponse, error) {
	repoDir, err := validateCredentialsDeps(deps)
	if err != nil {
		return nil, err
	}

	store, err := resolveCredentialsStore(deps)
	if err != nil {
		return nil, err
	}

	// Get stored credentials
	storedCreds, err := store.ListCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Get current remote URL to enrich with actual remote data
	remoteURL, _ := deps.Git.GetRemoteURL(ctx, repoDir, "origin")

	credentials := convertStoredCredentials(storedCreds, remoteURL)
	credentials = ensureOriginPlaceholder(credentials, remoteURL)

	return &CredentialsListResponse{
		Credentials: credentials,
		Timestamp:   time.Now().UTC(),
	}, nil
}

// convertStoredCredentials converts stored credentials to response format, enriching origin with remote URL.
func convertStoredCredentials(storedCreds []StoredCredential, remoteURL string) []Credential {
	credentials := make([]Credential, 0, len(storedCreds))
	for _, sc := range storedCreds {
		cred := sc.ToCredential()
		if sc.Remote == "origin" && remoteURL != "" {
			cred.URL = remoteURL
			cred.Type = detectCredentialType(remoteURL)
		}
		credentials = append(credentials, cred)
	}
	return credentials
}

// ensureOriginPlaceholder adds a placeholder credential for origin if none exists.
func ensureOriginPlaceholder(credentials []Credential, remoteURL string) []Credential {
	if remoteURL == "" {
		return credentials
	}
	for _, c := range credentials {
		if c.Remote == "origin" {
			return credentials
		}
	}
	return append(credentials, Credential{
		ID:           credentialIDFromRemote("origin"),
		Remote:       "origin",
		URL:          remoteURL,
		Type:         detectCredentialType(remoteURL),
		IsConfigured: false,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})
}

// SaveCredential saves or updates a credential.
func SaveCredential(ctx context.Context, deps CredentialsDeps, req CredentialSaveRequest) (*CredentialSaveResponse, error) {
	repoDir, err := validateCredentialsDeps(deps)
	if err != nil {
		return nil, err
	}

	remote := strings.TrimSpace(req.Remote)
	if remote == "" {
		return credSaveError("remote is required"), nil
	}

	sshKeyPath := strings.TrimSpace(req.SSHKeyPath)
	username := strings.TrimSpace(req.Username)
	token := strings.TrimSpace(req.Token)

	if errMsg := validateCredentialAuth(sshKeyPath, username, token); errMsg != "" {
		return credSaveError(errMsg), nil
	}

	store, err := resolveCredentialsStore(deps)
	if err != nil {
		return nil, err
	}

	remoteURL, err := resolveRemoteURL(ctx, deps, repoDir, remote, req.URL)
	if err != nil {
		return credSaveError(err.Error()), nil
	}

	credType := detectCredentialType(remoteURL)
	if sshKeyPath != "" {
		credType = CredentialTypeSSH
	}
	storedCred := StoredCredential{
		ID: credentialIDFromRemote(remote), Remote: remote, URL: remoteURL,
		Type: credType, Username: username, Token: token, SSHKeyPath: sshKeyPath,
	}

	if err := store.SaveCredential(storedCred); err != nil {
		return credSaveError(fmt.Sprintf("failed to save credential: %v", err)), nil
	}

	cred := storedCred.ToCredential()
	return &CredentialSaveResponse{
		Success:    true,
		Credential: &cred,
		Timestamp:  time.Now().UTC(),
	}, nil
}

// credSaveError returns a failed CredentialSaveResponse.
func credSaveError(msg string) *CredentialSaveResponse {
	return &CredentialSaveResponse{Success: false, Error: msg, Timestamp: time.Now().UTC()}
}

// validateCredentialAuth validates SSH or HTTPS credential fields.
func validateCredentialAuth(sshKeyPath, username, token string) string {
	if sshKeyPath != "" {
		if _, err := os.Stat(sshKeyPath); err != nil {
			return fmt.Sprintf("SSH key file not found: %s", sshKeyPath)
		}
		return ""
	}
	if username == "" {
		return "username is required"
	}
	if token == "" {
		return "token is required"
	}
	return ""
}

// resolveRemoteURL gets the remote URL and optionally updates it.
func resolveRemoteURL(ctx context.Context, deps CredentialsDeps, repoDir, remote, reqURL string) (string, error) {
	remoteURL, err := deps.Git.GetRemoteURL(ctx, repoDir, remote)
	if err != nil {
		return "", fmt.Errorf("remote '%s' not found: %v", remote, err)
	}
	if reqURL != "" && reqURL != remoteURL {
		if err := deps.Git.SetRemoteURL(ctx, repoDir, remote, reqURL); err != nil {
			return "", fmt.Errorf("failed to update remote URL: %v", err)
		}
		return reqURL, nil
	}
	return remoteURL, nil
}

// DeleteCredential removes a stored credential.
func DeleteCredential(ctx context.Context, deps CredentialsDeps, req CredentialDeleteRequest) (*CredentialDeleteResponse, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return &CredentialDeleteResponse{
			Success:   false,
			Error:     "credential ID is required",
			Timestamp: time.Now().UTC(),
		}, nil
	}

	store, err := resolveCredentialsStore(deps)
	if err != nil {
		return nil, err
	}

	if err := store.DeleteCredential(id); err != nil {
		return &CredentialDeleteResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to delete credential: %v", err),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	return &CredentialDeleteResponse{
		Success:   true,
		Timestamp: time.Now().UTC(),
	}, nil
}

// TestCredential tests authentication to a remote.
func TestCredential(ctx context.Context, deps CredentialsDeps, req CredentialTestRequest) (*CredentialTestResponse, error) {
	repoDir, err := validateCredentialsDeps(deps)
	if err != nil {
		return nil, err
	}

	remote := strings.TrimSpace(req.Remote)
	if remote == "" {
		remote = "origin"
	}

	store, err := resolveCredentialsStore(deps)
	if err != nil {
		return nil, err
	}

	if _, err := deps.Git.GetRemoteURL(ctx, repoDir, remote); err != nil {
		return credTestResult(false, false, false, fmt.Sprintf("remote '%s' not found: %v", remote, err)), nil
	}

	var cred *StoredCredential
	if req.UseStored {
		cred = resolveCredentialForRemote(ctx, deps.Git, store, repoDir, remote)
	}

	lsErr := deps.Git.LsRemote(ctx, repoDir, remote, cred)
	if lsErr == nil {
		return credTestResult(true, true, true, ""), nil
	}

	if isAuthenticationError(lsErr) {
		return credTestResult(false, true, false, "Authentication failed - check username and token"), nil
	}
	return credTestResult(false, false, false, fmt.Sprintf("Cannot reach remote: %v", lsErr)), nil
}

// credTestResult builds a CredentialTestResponse.
func credTestResult(success, reachable, authorized bool, errMsg string) *CredentialTestResponse {
	return &CredentialTestResponse{
		Success:    success,
		Reachable:  reachable,
		Authorized: authorized,
		Error:      errMsg,
		Timestamp:  time.Now().UTC(),
	}
}

// isAuthenticationError checks if an error indicates an authentication failure.
func isAuthenticationError(err error) bool {
	errStr := err.Error()
	authPatterns := []string{"Authentication failed", "could not read Username", "Permission denied", "401", "403"}
	for _, p := range authPatterns {
		if strings.Contains(errStr, p) {
			return true
		}
	}
	return false
}

// validateCredentialsDeps validates common credentials dependencies.
func validateCredentialsDeps(deps CredentialsDeps) (string, error) {
	if deps.Git == nil {
		return "", fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return "", fmt.Errorf("repo dir is required")
	}
	return repoDir, nil
}

// resolveCredentialsStore returns the existing store or creates a new one.
func resolveCredentialsStore(deps CredentialsDeps) (*CredentialsStore, error) {
	if deps.Store != nil {
		return deps.Store, nil
	}
	store, err := NewCredentialsStore(deps.StorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize credentials store: %w", err)
	}
	return store, nil
}
