package main

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	enrichAttachmentsFromArtifacts(issueDir, &issue)

	return &issue, nil
}

func enrichAttachmentsFromArtifacts(issueDir string, issue *Issue) {
	artifactsPath := filepath.Join(issueDir, artifactsDirName)
	info, err := os.Stat(artifactsPath)
	if err != nil || !info.IsDir() {
		return
	}

	if issue.Attachments == nil {
		issue.Attachments = []Attachment{}
	}

	attachmentIndex := make(map[string]*Attachment)
	for idx := range issue.Attachments {
		normalized := normalizeAttachmentPath(issue.Attachments[idx].Path)
		if normalized == "" {
			continue
		}
		issue.Attachments[idx].Path = normalized
		attachmentIndex[normalized] = &issue.Attachments[idx]

		fsPath := filepath.Join(issueDir, filepath.FromSlash(normalized))
		if stat, statErr := os.Stat(fsPath); statErr == nil {
			if issue.Attachments[idx].Size == 0 {
				issue.Attachments[idx].Size = stat.Size()
			}
			if strings.TrimSpace(issue.Attachments[idx].Type) == "" {
				if detected := mime.TypeByExtension(strings.ToLower(filepath.Ext(fsPath))); detected != "" {
					issue.Attachments[idx].Type = detected
				}
			}
		}
	}

	filepath.WalkDir(artifactsPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		rel, relErr := filepath.Rel(issueDir, path)
		if relErr != nil {
			return nil
		}
		normalized := normalizeAttachmentPath(filepath.ToSlash(rel))
		if normalized == "" {
			return nil
		}

		att, exists := attachmentIndex[normalized]
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}

		if exists {
			if att.Size == 0 {
				att.Size = info.Size()
			}
			if strings.TrimSpace(att.Type) == "" {
				if detected := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); detected != "" {
					att.Type = detected
				}
			}
			return nil
		}

		filename := info.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			switch ext {
			case ".md":
				contentType = "text/markdown"
			case ".txt":
				contentType = "text/plain"
			case ".json":
				contentType = "application/json"
			default:
				contentType = "application/octet-stream"
			}
		}

		category := "artifact"
		lowerName := strings.ToLower(filename)
		switch {
		case strings.Contains(lowerName, "prompt"):
			category = "investigation_prompt"
		case strings.Contains(lowerName, "report"):
			category = "investigation_report"
		}

		newAttachment := Attachment{
			Name:     filename,
			Type:     contentType,
			Path:     normalized,
			Size:     info.Size(),
			Category: category,
		}

		issue.Attachments = append(issue.Attachments, newAttachment)
		attachmentIndex[normalized] = &issue.Attachments[len(issue.Attachments)-1]
		return nil
	})
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

	if _, statErr := os.Stat(targetDir); statErr == nil {
		log.Printf("moveIssue: removing stale %s before moving issue %s", targetDir, issue.ID)
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to clear existing target %s: %w", targetDir, err)
		}
	}

	if err := os.Rename(issueDir, targetDir); err != nil {
		return err
	}

	if err := s.writeIssueMetadata(targetDir, issue); err != nil {
		return err
	}

	// Publish status change event for real-time updates
	s.hub.Publish(NewEvent(EventIssueStatusChanged, IssueStatusChangedData{
		IssueID:   issueID,
		OldStatus: currentFolder,
		NewStatus: toFolder,
	}))

	// Also publish issue.updated event to ensure clients have latest data
	s.hub.Publish(NewEvent(EventIssueUpdated, IssueEventData{Issue: issue}))

	return nil
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
