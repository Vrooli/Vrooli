package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"

	"gopkg.in/yaml.v3"
)

type IssueStorage interface {
	IssueDir(folder, issueID string) string
	LoadIssueFromDir(issueDir string) (*issuespkg.Issue, error)
	WriteIssueMetadata(issueDir string, issue *issuespkg.Issue) error
	SaveIssue(issue *issuespkg.Issue, folder string) (string, error)
	LoadIssuesFromFolder(folder string) ([]issuespkg.Issue, error)
	FindIssueDirectory(issueID string) (string, string, error)
	LoadIssueWithStatus(issueID string) (*issuespkg.Issue, string, string, error)
	MoveIssue(issueID, toFolder string) (*issuespkg.Issue, string, string, error)
}

type FileIssueStore struct {
	issuesDir string
}

func NewFileIssueStore(issuesDir string) IssueStorage {
	return &FileIssueStore{issuesDir: issuesDir}
}

func (fs *FileIssueStore) IssueDir(folder, issueID string) string {
	return filepath.Join(fs.issuesDir, folder, issueID)
}

func (fs *FileIssueStore) metadataPath(issueDir string) string {
	return filepath.Join(issueDir, issuespkg.MetadataFilename)
}

func (fs *FileIssueStore) LoadIssueFromDir(issueDir string) (*issuespkg.Issue, error) {
	metadata := fs.metadataPath(issueDir)
	data, err := ioutil.ReadFile(metadata)
	if err != nil {
		return nil, err
	}

	var issue issuespkg.Issue
	if err := yaml.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %v", err)
	}

	if issue.ID == "" {
		issue.ID = filepath.Base(issueDir)
	}

	enrichAttachmentsFromArtifacts(issueDir, &issue)

	return &issue, nil
}

func enrichAttachmentsFromArtifacts(issueDir string, issue *issuespkg.Issue) {
	artifactsPath := filepath.Join(issueDir, issuespkg.ArtifactsDirName)
	info, err := os.Stat(artifactsPath)
	if err != nil || !info.IsDir() {
		return
	}

	if issue.Attachments == nil {
		issue.Attachments = []issuespkg.Attachment{}
	}

	attachmentIndex := make(map[string]*issuespkg.Attachment)
	for idx := range issue.Attachments {
		normalized := issuespkg.NormalizeAttachmentPath(issue.Attachments[idx].Path)
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
		normalized := issuespkg.NormalizeAttachmentPath(filepath.ToSlash(rel))
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

		newAttachment := issuespkg.Attachment{
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

func (fs *FileIssueStore) WriteIssueMetadata(issueDir string, issue *issuespkg.Issue) error {
	now := time.Now().UTC().Format(time.RFC3339)
	if issue.Metadata.CreatedAt == "" {
		issue.Metadata.CreatedAt = now
	}
	issue.Metadata.UpdatedAt = now

	data, err := yaml.Marshal(issue)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %v", err)
	}

	return os.WriteFile(fs.metadataPath(issueDir), data, 0o644)
}

func (fs *FileIssueStore) SaveIssue(issue *issuespkg.Issue, folder string) (string, error) {
	issue.Status = folder
	issueDir := fs.IssueDir(folder, issue.ID)
	if err := os.MkdirAll(issueDir, 0o755); err != nil {
		return "", err
	}

	if err := fs.WriteIssueMetadata(issueDir, issue); err != nil {
		return "", err
	}

	return issueDir, nil
}

func (fs *FileIssueStore) LoadIssuesFromFolder(folder string) ([]issuespkg.Issue, error) {
	folderPath := filepath.Join(fs.issuesDir, folder)
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []issuespkg.Issue{}, nil
		}
		return nil, err
	}

	var issues []issuespkg.Issue
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		issueDir := filepath.Join(folderPath, entry.Name())
		issue, err := fs.LoadIssueFromDir(issueDir)
		if err != nil {
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

func (fs *FileIssueStore) FindIssueDirectory(issueID string) (string, string, error) {
	for _, folder := range issuespkg.ValidStatuses() {
		directDir := fs.IssueDir(folder, issueID)
		if _, err := os.Stat(fs.metadataPath(directDir)); err == nil {
			return directDir, folder, nil
		}
	}

	for _, folder := range issuespkg.ValidStatuses() {
		folderPath := filepath.Join(fs.issuesDir, folder)
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
			issue, err := fs.LoadIssueFromDir(issueDir)
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

func (fs *FileIssueStore) LoadIssueWithStatus(issueID string) (*issuespkg.Issue, string, string, error) {
	issueDir, statusFolder, err := fs.FindIssueDirectory(issueID)
	if err != nil {
		return nil, "", "", err
	}

	issue, err := fs.LoadIssueFromDir(issueDir)
	if err != nil {
		return nil, "", "", err
	}

	issue.Status = statusFolder
	return issue, issueDir, statusFolder, nil
}

func (fs *FileIssueStore) MoveIssue(issueID, toFolder string) (*issuespkg.Issue, string, string, error) {
	issueDir, currentFolder, err := fs.FindIssueDirectory(issueID)
	if err != nil {
		return nil, "", "", err
	}

	issue, err := fs.LoadIssueFromDir(issueDir)
	if err != nil {
		return nil, "", "", err
	}

	if currentFolder == toFolder {
		return issue, currentFolder, toFolder, nil
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

	targetDir := fs.IssueDir(toFolder, issue.ID)
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return nil, "", "", err
	}

	if _, statErr := os.Stat(targetDir); statErr == nil {
		if err := os.RemoveAll(targetDir); err != nil {
			return nil, "", "", fmt.Errorf("failed to clear existing target %s: %w", targetDir, err)
		}
	}

	if err := os.Rename(issueDir, targetDir); err != nil {
		return nil, "", "", err
	}

	if err := fs.WriteIssueMetadata(targetDir, issue); err != nil {
		return nil, "", "", err
	}

	return issue, currentFolder, toFolder, nil
}
