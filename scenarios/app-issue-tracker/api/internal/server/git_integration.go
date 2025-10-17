package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"
)

// GitConfig holds GitHub configuration
type GitConfig struct {
	Token      string `json:"token"`
	Owner      string `json:"owner"`
	Repo       string `json:"repo"`
	BaseBranch string `json:"base_branch"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	ID          string    `json:"id"`
	IssueID     string    `json:"issue_id"`
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Branch      string    `json:"branch"`
	URL         string    `json:"url"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// GitIntegrationService handles Git and GitHub operations
type GitIntegrationService struct {
	config     *GitConfig
	workingDir string
}

// NewGitIntegrationService creates a new Git integration service

func requiredEnvValue(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("%s is not set", key)
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s cannot be empty", key)
	}
	return trimmed, nil
}

func envOrDefault(key, defaultValue string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return defaultValue
}

func NewGitIntegrationService(configPath string) (*GitIntegrationService, error) {
	config := &GitConfig{
		BaseBranch: envOrDefault("GITHUB_BASE_BRANCH", "main"),
	}

	// Try to load config from file if exists
	if configPath != "" {
		if data, err := ioutil.ReadFile(configPath); err == nil {
			json.Unmarshal(data, config)
		}
	}

	missing := make([]string, 0, 3)
	if strings.TrimSpace(config.Owner) == "" {
		if owner, err := requiredEnvValue("GITHUB_OWNER"); err == nil {
			config.Owner = owner
		} else {
			missing = append(missing, "GITHUB_OWNER")
		}
	}
	if strings.TrimSpace(config.Repo) == "" {
		if repo, err := requiredEnvValue("GITHUB_REPO"); err == nil {
			config.Repo = repo
		} else {
			missing = append(missing, "GITHUB_REPO")
		}
	}
	if strings.TrimSpace(config.Token) == "" {
		if token, err := requiredEnvValue("GITHUB_TOKEN"); err == nil {
			config.Token = token
		} else {
			missing = append(missing, "GITHUB_TOKEN")
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing GitHub configuration: %s", strings.Join(missing, ", "))
	}

	// Create working directory for Git operations
	workingDir := filepath.Join(os.TempDir(), "app-issue-tracker-git")
	if err := os.MkdirAll(workingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create working directory: %v", err)
	}

	return &GitIntegrationService{
		config:     config,
		workingDir: workingDir,
	}, nil
}

// CreatePullRequest creates a pull request from issue investigation results
func (g *GitIntegrationService) CreatePullRequest(issue Issue, fixes []string) (*PullRequest, error) {
	if g.config.Token == "" || g.config.Owner == "" || g.config.Repo == "" {
		return nil, fmt.Errorf("GitHub configuration incomplete - set GITHUB_TOKEN, GITHUB_OWNER, GITHUB_REPO")
	}

	// Create branch name from issue
	branchName := fmt.Sprintf("fix/%s-%s", issue.ID, sanitizeBranchName(issue.Title))

	// Clone or update repository
	repoDir := filepath.Join(g.workingDir, g.config.Repo)
	if err := g.ensureRepository(repoDir); err != nil {
		return nil, fmt.Errorf("failed to setup repository: %v", err)
	}

	// Create and checkout new branch
	if err := g.createBranch(repoDir, branchName); err != nil {
		return nil, fmt.Errorf("failed to create branch: %v", err)
	}

	// Apply fixes
	for _, fix := range fixes {
		if err := g.applyFix(repoDir, fix); err != nil {
			LogWarn("Failed to apply generated fix", "error", err, "issue_id", issue.ID)
		}
	}

	// Commit changes
	commitMsg := fmt.Sprintf("Fix: %s\n\nResolves issue %s", issue.Title, issue.ID)
	if err := g.commitChanges(repoDir, commitMsg); err != nil {
		return nil, fmt.Errorf("failed to commit changes: %v", err)
	}

	// Push branch
	if err := g.pushBranch(repoDir, branchName); err != nil {
		return nil, fmt.Errorf("failed to push branch: %v", err)
	}

	// Create pull request via GitHub API
	pr, err := g.createGitHubPR(issue, branchName)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %v", err)
	}

	return pr, nil
}

// ensureRepository clones or updates the repository
func (g *GitIntegrationService) ensureRepository(repoDir string) error {
	if _, err := os.Stat(repoDir); os.IsNotExist(err) {
		// Clone repository
		cloneURL := fmt.Sprintf("https://github.com/%s/%s.git", g.config.Owner, g.config.Repo)
		cmd := exec.Command("git", "clone", cloneURL, repoDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %v - %s", err, output)
		}
	} else {
		// Update existing repository
		cmd := exec.Command("git", "fetch", "origin")
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git fetch failed: %v - %s", err, output)
		}

		// Reset to origin/main
		cmd = exec.Command("git", "reset", "--hard", fmt.Sprintf("origin/%s", g.config.BaseBranch))
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git reset failed: %v - %s", err, output)
		}
	}

	return nil
}

// createBranch creates and checks out a new branch
func (g *GitIntegrationService) createBranch(repoDir, branchName string) error {
	cmd := exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try to checkout existing branch
		cmd = exec.Command("git", "checkout", branchName)
		cmd.Dir = repoDir
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("git checkout failed: %v - %s %s", err, output, output2)
		}
	}
	return nil
}

// applyFix applies a fix to the repository
func (g *GitIntegrationService) applyFix(repoDir, fix string) error {
	// Parse fix format (expected: "file:line:change" or similar)
	// This is a simplified implementation - extend based on your fix format

	// For now, just create a FIXES.md file with the fix content
	fixesFile := filepath.Join(repoDir, "FIXES.md")
	content := ""
	if data, err := ioutil.ReadFile(fixesFile); err == nil {
		content = string(data)
	}

	content += fmt.Sprintf("\n## Fix Applied\n%s\n", fix)

	return ioutil.WriteFile(fixesFile, []byte(content), 0644)
}

// commitChanges commits all changes
func (g *GitIntegrationService) commitChanges(repoDir, message string) error {
	// Add all changes
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %v - %s", err, output)
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %v - %s", err, output)
	}

	return nil
}

// pushBranch pushes the branch to remote
func (g *GitIntegrationService) pushBranch(repoDir, branchName string) error {
	cmd := exec.Command("git", "push", "-u", "origin", branchName)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_ASKPASS=echo"))

	// Add authentication if token is available
	if g.config.Token != "" {
		remoteURL := fmt.Sprintf("https://%s@github.com/%s/%s.git",
			g.config.Token, g.config.Owner, g.config.Repo)

		// Set remote URL with token
		setURLCmd := exec.Command("git", "remote", "set-url", "origin", remoteURL)
		setURLCmd.Dir = repoDir
		if output, err := setURLCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git remote set-url failed: %v - %s", err, output)
		}
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %v - %s", err, output)
	}

	return nil
}

// createGitHubPR creates a pull request via GitHub API
func (g *GitIntegrationService) createGitHubPR(issue Issue, branchName string) (*PullRequest, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", g.config.Owner, g.config.Repo)

	prBody := fmt.Sprintf(`## Issue Resolution

This pull request addresses issue **%s**: %s

### Problem
%s

### Solution
Automated fix generated by App Issue Tracker AI investigation.

### Testing
- [ ] Tests added/updated
- [ ] Manual testing completed
- [ ] No regressions identified

Closes #%s`, issue.ID, issue.Title, issue.Description, issue.ID)

	payload := map[string]interface{}{
		"title": fmt.Sprintf("Fix: %s", issue.Title),
		"body":  prBody,
		"head":  branchName,
		"base":  g.config.BaseBranch,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.config.Token))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, body)
	}

	var ghResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ghResponse); err != nil {
		return nil, err
	}

	pr := &PullRequest{
		ID:          fmt.Sprintf("pr-%s", issue.ID),
		IssueID:     issue.ID,
		Number:      int(ghResponse["number"].(float64)),
		Title:       ghResponse["title"].(string),
		Description: prBody,
		Branch:      branchName,
		URL:         ghResponse["html_url"].(string),
		Status:      "open",
		CreatedAt:   time.Now(),
	}

	return pr, nil
}

// sanitizeBranchName creates a valid Git branch name
func sanitizeBranchName(name string) string {
	// Remove special characters and replace spaces with hyphens
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")

	// Keep only alphanumeric and hyphens
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result = append(result, r)
		}
	}

	// Limit length
	branchName := string(result)
	if len(branchName) > 50 {
		branchName = branchName[:50]
	}

	return branchName
}

// gitPRHandler handles pull request creation requests
func (s *Server) gitPRHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	issueID := vars["id"]

	// Load issue
	issueDir, folder, err := s.findIssueDirectory(issueID)
	if err != nil || issueDir == "" {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}
	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	// Get fixes from investigation results
	fixes := s.extractFixesFromInvestigation(issueID)
	if len(fixes) == 0 {
		response := ApiResponse{
			Success: false,
			Message: "No fixes available for this issue. Run investigation first.",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Initialize Git service
	gitService, err := NewGitIntegrationService("")
	if err != nil {
		response := ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to initialize Git service: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create pull request
	pr, err := gitService.CreatePullRequest(*issue, fixes)
	if err != nil {
		response := ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create pull request: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Update issue with PR information
	if issue.Metadata.Extra == nil {
		issue.Metadata.Extra = make(map[string]string)
	}
	issue.Metadata.Extra["pull_request"] = pr.URL
	issue.Metadata.Extra["pr_number"] = fmt.Sprintf("%d", pr.Number)
	issue.Metadata.UpdatedAt = time.Now().Format(time.RFC3339)

	// Save updated issue
	if _, err := s.saveIssue(issue, folder); err != nil {
		LogWarn("Failed to persist issue metadata after PR creation", "error", err, "issue_id", issueID)
	}

	response := ApiResponse{
		Success: true,
		Message: "Pull request created successfully",
		Data: map[string]interface{}{
			"pull_request": pr,
			"issue_id":     issueID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// extractFixesFromInvestigation extracts fixes from investigation results
func (s *Server) extractFixesFromInvestigation(issueID string) []string {
	// Look for investigation results in the issue's artifacts directory
	artifactsDir := filepath.Join(s.config.IssuesDir, issueID, artifactsDirName)

	var fixes []string

	// Check for fix files
	fixFiles, err := filepath.Glob(filepath.Join(artifactsDir, "fix_*.txt"))
	if err == nil {
		for _, fixFile := range fixFiles {
			if content, err := ioutil.ReadFile(fixFile); err == nil {
				fixes = append(fixes, string(content))
			}
		}
	}

	// Check for resolution.yaml
	resolutionFile := filepath.Join(artifactsDir, "resolution.yaml")
	if data, err := ioutil.ReadFile(resolutionFile); err == nil {
		var resolution map[string]interface{}
		if err := yaml.Unmarshal(data, &resolution); err == nil {
			if fixList, ok := resolution["fixes"].([]interface{}); ok {
				for _, fix := range fixList {
					if fixStr, ok := fix.(string); ok {
						fixes = append(fixes, fixStr)
					}
				}
			}
		}
	}

	return fixes
}
