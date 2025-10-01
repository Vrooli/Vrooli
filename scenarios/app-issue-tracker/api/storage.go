package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// issueDir returns the full path to an issue directory
func (s *Server) issueDir(folder, issueID string) string {
	return filepath.Join(s.config.IssuesDir, folder, issueID)
}

// metadataPath returns the full path to an issue's metadata file
func (s *Server) metadataPath(issueDir string) string {
	return filepath.Join(issueDir, metadataFilename)
}

// loadIssueFromDir loads an issue from a directory
func (s *Server) loadIssueFromDir(issueDir string) (*Issue, error) {
	metadata := s.metadataPath(issueDir)
	data, err := ioutil.ReadFile(metadata)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := yaml.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	if issue.ID == "" {
		issue.ID = filepath.Base(issueDir)
	}

	return &issue, nil
}

// writeIssueMetadata writes issue metadata to disk
func (s *Server) writeIssueMetadata(issueDir string, issue *Issue) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if issue.Metadata.CreatedAt == "" {
		issue.Metadata.CreatedAt = now
	}
	issue.Metadata.UpdatedAt = now

	data, err := yaml.Marshal(issue)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %v", err)
	}

	return os.WriteFile(s.metadataPath(issueDir), data, 0644)
}

// saveIssue saves an issue to a specific folder
func (s *Server) saveIssue(issue *Issue, folder string) (string, error) {
	issue.Status = folder
	issueDir := s.issueDir(folder, issue.ID)
	if err := os.MkdirAll(issueDir, 0755); err != nil {
		return "", err
	}

	if err := s.writeIssueMetadata(issueDir, issue); err != nil {
		return "", err
	}

	return issueDir, nil
}

// loadIssuesFromFolder loads all issues from a folder
func (s *Server) loadIssuesFromFolder(folder string) ([]Issue, error) {
	folderPath := filepath.Join(s.config.IssuesDir, folder)
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Issue{}, nil
		}
		return nil, err
	}

	var issues []Issue
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		issueDir := filepath.Join(folderPath, entry.Name())
		issue, err := s.loadIssueFromDir(issueDir)
		if err != nil {
			log.Printf("Warning: could not load issue from %s: %v", issueDir, err)
			continue
		}
		issue.Status = folder
		issues = append(issues, *issue)
	}

	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Metadata.CreatedAt > issues[j].Metadata.CreatedAt
	})

	return issues, nil
}

// findIssueDirectory finds an issue's directory and current folder
func (s *Server) findIssueDirectory(issueID string) (string, string, error) {
	folders := []string{"open", "active", "waiting", "completed", "failed", "archived"}

	for _, folder := range folders {
		directDir := s.issueDir(folder, issueID)
		if _, err := os.Stat(s.metadataPath(directDir)); err == nil {
			return directDir, folder, nil
		}
	}

	for _, folder := range folders {
		folderPath := filepath.Join(s.config.IssuesDir, folder)
		entries, err := os.ReadDir(folderPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", "", err
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			issueDir := filepath.Join(folderPath, entry.Name())
			issue, err := s.loadIssueFromDir(issueDir)
			if err != nil {
				continue
			}
			if issue.ID == issueID {
				return issueDir, folder, nil
			}
		}
	}

	return "", "", fmt.Errorf("issue not found: %s", issueID)
}

// moveIssue moves an issue from one folder to another
func (s *Server) moveIssue(issueID, toFolder string) error {
	issueDir, currentFolder, err := s.findIssueDirectory(issueID)
	if err != nil {
		return err
	}

	if currentFolder == toFolder {
		return nil
	}

	issue, err := s.loadIssueFromDir(issueDir)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	issue.Status = toFolder
	issue.Metadata.UpdatedAt = now

	switch toFolder {
	case "active":
		if issue.Investigation.StartedAt == "" {
			issue.Investigation.StartedAt = now
		}
	case "completed":
		if issue.Metadata.ResolvedAt == "" {
			issue.Metadata.ResolvedAt = now
		}
	}

	targetDir := s.issueDir(toFolder, issue.ID)
	if err := os.MkdirAll(filepath.Dir(targetDir), 0755); err != nil {
		return err
	}

	if err := os.Rename(issueDir, targetDir); err != nil {
		return err
	}

	return s.writeIssueMetadata(targetDir, issue)
}

// getAllIssues retrieves all issues with optional filters
func (s *Server) getAllIssues(statusFilter, priorityFilter, typeFilter string, limit int) ([]Issue, error) {
	var allIssues []Issue

	folders := []string{"open", "active", "completed", "failed", "archived"}
	if statusFilter != "" {
		folders = []string{statusFilter}
	}

	for _, folder := range folders {
		folderIssues, err := s.loadIssuesFromFolder(folder)
		if err != nil {
			log.Printf("Warning: Could not load issues from %s: %v", folder, err)
			continue
		}
		allIssues = append(allIssues, folderIssues...)
	}

	// Apply filters
	var filteredIssues []Issue
	for _, issue := range allIssues {
		if priorityFilter != "" && issue.Priority != priorityFilter {
			continue
		}
		if typeFilter != "" && issue.Type != typeFilter {
			continue
		}
		filteredIssues = append(filteredIssues, issue)
	}

	// Sort by creation date (newest first)
	sort.Slice(filteredIssues, func(i, j int) bool {
		return filteredIssues[i].Metadata.CreatedAt > filteredIssues[j].Metadata.CreatedAt
	})

	// Apply limit
	if limit > 0 && len(filteredIssues) > limit {
		filteredIssues = filteredIssues[:limit]
	}

	return filteredIssues, nil
}
