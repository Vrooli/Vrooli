package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
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
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	store := deps.Store
	if store == nil {
		var err error
		store, err = NewCredentialsStore(deps.StorePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize credentials store: %w", err)
		}
	}

	// Get stored credentials
	storedCreds, err := store.ListCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Get current remote URL to enrich with actual remote data
	remoteURL, _ := deps.Git.GetRemoteURL(ctx, repoDir, "origin")

	// Convert to response format (masks tokens)
	credentials := make([]Credential, 0, len(storedCreds))
	for _, sc := range storedCreds {
		cred := sc.ToCredential()
		// Update URL from git remote if available and matches
		if sc.Remote == "origin" && remoteURL != "" {
			cred.URL = remoteURL
			cred.Type = detectCredentialType(remoteURL)
		}
		credentials = append(credentials, cred)
	}

	// If no stored credential for origin but remote exists, add placeholder
	if remoteURL != "" {
		hasOrigin := false
		for _, c := range credentials {
			if c.Remote == "origin" {
				hasOrigin = true
				break
			}
		}
		if !hasOrigin {
			credentials = append(credentials, Credential{
				ID:           credentialIDFromRemote("origin"),
				Remote:       "origin",
				URL:          remoteURL,
				Type:         detectCredentialType(remoteURL),
				IsConfigured: false,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			})
		}
	}

	return &CredentialsListResponse{
		Credentials: credentials,
		Timestamp:   time.Now().UTC(),
	}, nil
}

// SaveCredential saves or updates a credential.
func SaveCredential(ctx context.Context, deps CredentialsDeps, req CredentialSaveRequest) (*CredentialSaveResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	// Validate request
	remote := strings.TrimSpace(req.Remote)
	if remote == "" {
		return &CredentialSaveResponse{
			Success:   false,
			Error:     "remote is required",
			Timestamp: time.Now().UTC(),
		}, nil
	}

	sshKeyPath := strings.TrimSpace(req.SSHKeyPath)
	username := strings.TrimSpace(req.Username)
	token := strings.TrimSpace(req.Token)

	// Branch validation: SSH requires key path, HTTPS requires username+token
	if sshKeyPath != "" {
		// Validate SSH key file exists
		if _, err := os.Stat(sshKeyPath); err != nil {
			return &CredentialSaveResponse{
				Success:   false,
				Error:     fmt.Sprintf("SSH key file not found: %s", sshKeyPath),
				Timestamp: time.Now().UTC(),
			}, nil
		}
	} else {
		// HTTPS path: require username and token
		if username == "" {
			return &CredentialSaveResponse{
				Success:   false,
				Error:     "username is required",
				Timestamp: time.Now().UTC(),
			}, nil
		}
		if token == "" {
			return &CredentialSaveResponse{
				Success:   false,
				Error:     "token is required",
				Timestamp: time.Now().UTC(),
			}, nil
		}
	}

	store := deps.Store
	if store == nil {
		var err error
		store, err = NewCredentialsStore(deps.StorePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize credentials store: %w", err)
		}
	}

	// Get current remote URL
	remoteURL, err := deps.Git.GetRemoteURL(ctx, repoDir, remote)
	if err != nil {
		return &CredentialSaveResponse{
			Success:   false,
			Error:     fmt.Sprintf("remote '%s' not found: %v", remote, err),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Optionally update remote URL if provided
	if req.URL != "" && req.URL != remoteURL {
		if err := deps.Git.SetRemoteURL(ctx, repoDir, remote, req.URL); err != nil {
			return &CredentialSaveResponse{
				Success:   false,
				Error:     fmt.Sprintf("failed to update remote URL: %v", err),
				Timestamp: time.Now().UTC(),
			}, nil
		}
		remoteURL = req.URL
	}

	// Create stored credential
	credType := detectCredentialType(remoteURL)
	if sshKeyPath != "" {
		credType = CredentialTypeSSH
	}
	storedCred := StoredCredential{
		ID:         credentialIDFromRemote(remote),
		Remote:     remote,
		URL:        remoteURL,
		Type:       credType,
		Username:   username,
		Token:      token,
		SSHKeyPath: sshKeyPath,
	}

	// Save to store
	if err := store.SaveCredential(storedCred); err != nil {
		return &CredentialSaveResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to save credential: %v", err),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	cred := storedCred.ToCredential()
	return &CredentialSaveResponse{
		Success:    true,
		Credential: &cred,
		Timestamp:  time.Now().UTC(),
	}, nil
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

	store := deps.Store
	if store == nil {
		var err error
		store, err = NewCredentialsStore(deps.StorePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize credentials store: %w", err)
		}
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
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	remote := strings.TrimSpace(req.Remote)
	if remote == "" {
		remote = "origin"
	}

	store := deps.Store
	if store == nil {
		var err error
		store, err = NewCredentialsStore(deps.StorePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize credentials store: %w", err)
		}
	}

	// Verify remote exists
	if _, err := deps.Git.GetRemoteURL(ctx, repoDir, remote); err != nil {
		return &CredentialTestResponse{
			Success:   false,
			Reachable: false,
			Error:     fmt.Sprintf("remote '%s' not found: %v", remote, err),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Get stored credential if UseStored is true
	var cred *StoredCredential
	if req.UseStored {
		cred, _ = store.GetCredentialByRemote(remote)
	}

	// Test connection using ls-remote
	lsErr := deps.Git.LsRemote(ctx, repoDir, remote, cred)
	if lsErr != nil {
		// Parse error to determine if it's a reachability or auth issue
		errStr := lsErr.Error()
		isAuthError := strings.Contains(errStr, "Authentication failed") ||
			strings.Contains(errStr, "could not read Username") ||
			strings.Contains(errStr, "Permission denied") ||
			strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "403")

		if isAuthError {
			return &CredentialTestResponse{
				Success:    false,
				Reachable:  true, // We reached the server but auth failed
				Authorized: false,
				Error:      "Authentication failed - check username and token",
				Timestamp:  time.Now().UTC(),
			}, nil
		}

		// Other errors are likely reachability issues
		return &CredentialTestResponse{
			Success:    false,
			Reachable:  false,
			Authorized: false,
			Error:      fmt.Sprintf("Cannot reach remote: %v", lsErr),
			Timestamp:  time.Now().UTC(),
		}, nil
	}

	return &CredentialTestResponse{
		Success:    true,
		Reachable:  true,
		Authorized: true,
		Timestamp:  time.Now().UTC(),
	}, nil
}

// UpdateRemoteURL updates a remote's URL.
func UpdateRemoteURL(ctx context.Context, deps CredentialsDeps, req RemoteURLUpdateRequest) (*RemoteURLUpdateResponse, error) {
	if deps.Git == nil {
		return nil, fmt.Errorf("git runner is required")
	}
	repoDir := strings.TrimSpace(deps.RepoDir)
	if repoDir == "" {
		return nil, fmt.Errorf("repo dir is required")
	}

	remote := strings.TrimSpace(req.Remote)
	if remote == "" {
		return &RemoteURLUpdateResponse{
			Success:   false,
			Error:     "remote is required",
			Timestamp: time.Now().UTC(),
		}, nil
	}

	newURL := strings.TrimSpace(req.URL)
	if newURL == "" {
		return &RemoteURLUpdateResponse{
			Success:   false,
			Error:     "URL is required",
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Validate URL format
	if !isValidGitURL(newURL) {
		return &RemoteURLUpdateResponse{
			Success:   false,
			Error:     "invalid git URL format",
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Get old URL
	oldURL, err := deps.Git.GetRemoteURL(ctx, repoDir, remote)
	if err != nil {
		return &RemoteURLUpdateResponse{
			Success:   false,
			Error:     fmt.Sprintf("remote '%s' not found: %v", remote, err),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	// Update URL
	if err := deps.Git.SetRemoteURL(ctx, repoDir, remote, newURL); err != nil {
		return &RemoteURLUpdateResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to update remote URL: %v", err),
			Timestamp: time.Now().UTC(),
		}, nil
	}

	return &RemoteURLUpdateResponse{
		Success:   true,
		OldURL:    oldURL,
		NewURL:    newURL,
		Timestamp: time.Now().UTC(),
	}, nil
}

// detectCredentialType determines the credential type from a URL.
func detectCredentialType(url string) CredentialType {
	url = strings.TrimSpace(url)
	if strings.HasPrefix(url, "git@") || strings.HasPrefix(url, "ssh://") {
		return CredentialTypeSSH
	}
	return CredentialTypeHTTPS
}

// isValidGitURL checks if a URL is a valid git remote URL.
func isValidGitURL(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return false
	}

	// HTTPS URL pattern
	httpsPattern := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if httpsPattern.MatchString(url) {
		return true
	}

	// SSH URL pattern (git@host:user/repo.git or ssh://git@host/user/repo.git)
	sshPattern := regexp.MustCompile(`^(git@[^\s:]+:[^\s]+|ssh://[^\s]+)$`)
	return sshPattern.MatchString(url)
}

// ConvertSSHToHTTPS converts an SSH URL to HTTPS format.
// Example: git@github.com:user/repo.git -> https://github.com/user/repo.git
func ConvertSSHToHTTPS(sshURL string) string {
	sshURL = strings.TrimSpace(sshURL)

	// Handle git@host:user/repo.git format
	if strings.HasPrefix(sshURL, "git@") {
		// Remove git@ prefix
		rest := strings.TrimPrefix(sshURL, "git@")
		// Replace : with /
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) == 2 {
			return "https://" + parts[0] + "/" + parts[1]
		}
	}

	// Handle ssh://git@host/user/repo.git format
	if strings.HasPrefix(sshURL, "ssh://") {
		rest := strings.TrimPrefix(sshURL, "ssh://")
		rest = strings.TrimPrefix(rest, "git@")
		return "https://" + rest
	}

	return sshURL
}

// ConvertHTTPSToSSH converts an HTTPS URL to SSH format.
// Example: https://github.com/user/repo.git -> git@github.com:user/repo.git
func ConvertHTTPSToSSH(httpsURL string) string {
	httpsURL = strings.TrimSpace(httpsURL)

	// Handle https://host/user/repo.git format
	if strings.HasPrefix(httpsURL, "https://") || strings.HasPrefix(httpsURL, "http://") {
		// Remove protocol
		rest := strings.TrimPrefix(httpsURL, "https://")
		rest = strings.TrimPrefix(rest, "http://")
		// Split host from path
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 2 {
			return "git@" + parts[0] + ":" + parts[1]
		}
	}

	return httpsURL
}
