package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

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
