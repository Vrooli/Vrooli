package storage

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/utils"

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
	ops       *FileOps
}

type FileOps struct {
	ReadFile  func(string) ([]byte, error)
	WriteFile func(string, []byte, os.FileMode) error
	ReadDir   func(string) ([]os.DirEntry, error)
	Stat      func(string) (os.FileInfo, error)
	WalkDir   func(string, iofs.WalkDirFunc) error
	MkdirAll  func(string, os.FileMode) error
	RemoveAll func(string) error
	Rename    func(string, string) error
}

func defaultFileOps() *FileOps {
	return &FileOps{
		ReadFile:  os.ReadFile,
		WriteFile: os.WriteFile,
		ReadDir:   os.ReadDir,
		Stat:      os.Stat,
		WalkDir: func(root string, fn iofs.WalkDirFunc) error {
			return filepath.WalkDir(root, fn)
		},
		MkdirAll:  os.MkdirAll,
		RemoveAll: os.RemoveAll,
		Rename:    os.Rename,
	}
}

func NewFileIssueStore(issuesDir string) IssueStorage {
	return NewFileIssueStoreWithOps(issuesDir, nil)
}

func NewFileIssueStoreWithOps(issuesDir string, ops *FileOps) IssueStorage {
	if ops == nil {
		ops = defaultFileOps()
	}
	return &FileIssueStore{issuesDir: issuesDir, ops: ops}
}

var ErrIssueNotFound = errors.New("issue not found")

func (fs *FileIssueStore) IssueDir(folder, issueID string) string {
	return filepath.Join(fs.issuesDir, folder, issueID)
}

func (fs *FileIssueStore) metadataPath(issueDir string) string {
	return filepath.Join(issueDir, issuespkg.MetadataFilename)
}

func (fs *FileIssueStore) LoadIssueFromDir(issueDir string) (*issuespkg.Issue, error) {
	metadata := fs.metadataPath(issueDir)
	data, err := fs.ops.ReadFile(metadata)
	if err != nil {
		return nil, err
	}

	var issue issuespkg.Issue
	if err := yaml.Unmarshal(data, &issue); err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	if issue.ID == "" {
		issue.ID = filepath.Base(issueDir)
	}

	if err := fs.enrichAttachmentsFromArtifacts(issueDir, &issue); err != nil {
		return nil, fmt.Errorf("enrich attachments: %w", err)
	}

	return &issue, nil
}

func (fs *FileIssueStore) enrichAttachmentsFromArtifacts(issueDir string, issue *issuespkg.Issue) error {
	artifactsPath := filepath.Join(issueDir, issuespkg.ArtifactsDirName)
	info, err := fs.ops.Stat(artifactsPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat artifacts directory: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	index, err := fs.indexExistingAttachments(issueDir, issue)
	if err != nil {
		return err
	}

	if walkErr := fs.ops.WalkDir(artifactsPath, func(path string, d iofs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		return fs.upsertAttachmentFromFile(issueDir, path, d, issue, index)
	}); walkErr != nil {
		return fmt.Errorf("walk artifacts: %w", walkErr)
	}

	return nil
}

func (fs *FileIssueStore) indexExistingAttachments(issueDir string, issue *issuespkg.Issue) (map[string]*issuespkg.Attachment, error) {
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
		stat, statErr := fs.ops.Stat(fsPath)
		if statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat attachment %s: %w", fsPath, statErr)
		}

		if issue.Attachments[idx].Size == 0 {
			issue.Attachments[idx].Size = stat.Size()
		}
		if strings.TrimSpace(issue.Attachments[idx].Type) == "" {
			issue.Attachments[idx].Type = detectContentType(fsPath)
		}
	}
	return attachmentIndex, nil
}

func (fs *FileIssueStore) upsertAttachmentFromFile(issueDir, fullPath string, entry iofs.DirEntry, issue *issuespkg.Issue, index map[string]*issuespkg.Attachment) error {
	rel, relErr := filepath.Rel(issueDir, fullPath)
	if relErr != nil {
		return fmt.Errorf("relative path: %w", relErr)
	}

	normalized := issuespkg.NormalizeAttachmentPath(filepath.ToSlash(rel))
	if normalized == "" {
		return nil
	}

	info, infoErr := entry.Info()
	if infoErr != nil {
		return fmt.Errorf("stat artifact %s: %w", fullPath, infoErr)
	}

	if existing, ok := index[normalized]; ok {
		if existing.Size == 0 {
			existing.Size = info.Size()
		}
		if strings.TrimSpace(existing.Type) == "" {
			existing.Type = detectContentType(fullPath)
		}
		return nil
	}

	attachment := issuespkg.Attachment{
		Name:     info.Name(),
		Type:     detectContentType(fullPath),
		Path:     normalized,
		Size:     info.Size(),
		Category: inferAttachmentCategory(info.Name()),
	}

	issue.Attachments = append(issue.Attachments, attachment)
	index[normalized] = &issue.Attachments[len(issue.Attachments)-1]
	return nil
}

func detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		return contentType
	}
	switch ext {
	case ".md":
		return "text/markdown"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func inferAttachmentCategory(filename string) string {
	lowerName := strings.ToLower(filename)
	switch {
	case strings.Contains(lowerName, "prompt"):
		return "investigation_prompt"
	case strings.Contains(lowerName, "report"):
		return "investigation_report"
	default:
		return "artifact"
	}
}

func (fs *FileIssueStore) WriteIssueMetadata(issueDir string, issue *issuespkg.Issue) error {
	now := utils.NowRFC3339()
	if issue.Metadata.CreatedAt == "" {
		issue.Metadata.CreatedAt = now
	}
	issue.Metadata.UpdatedAt = now

	data, err := yaml.Marshal(issue)
	if err != nil {
		return fmt.Errorf("error marshaling YAML: %w", err)
	}

	return fs.ops.WriteFile(fs.metadataPath(issueDir), data, 0o644)
}

func (fs *FileIssueStore) SaveIssue(issue *issuespkg.Issue, folder string) (string, error) {
	issue.Status = folder
	issueDir := fs.IssueDir(folder, issue.ID)
	if err := fs.ops.MkdirAll(issueDir, 0o755); err != nil {
		return "", err
	}

	if err := fs.WriteIssueMetadata(issueDir, issue); err != nil {
		return "", err
	}

	return issueDir, nil
}

func (fs *FileIssueStore) LoadIssuesFromFolder(folder string) ([]issuespkg.Issue, error) {
	folderPath := filepath.Join(fs.issuesDir, folder)
	entries, err := fs.ops.ReadDir(folderPath)
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
		if _, err := fs.ops.Stat(fs.metadataPath(directDir)); err == nil {
			return directDir, folder, nil
		}
	}

	for _, folder := range issuespkg.ValidStatuses() {
		folderPath := filepath.Join(fs.issuesDir, folder)
		entries, err := fs.ops.ReadDir(folderPath)
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

	return "", "", fmt.Errorf("%w: %s", ErrIssueNotFound, issueID)
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

	now := utils.NowRFC3339()
	issue.Status = toFolder
	issue.Metadata.UpdatedAt = now

	switch toFolder {
	case "active":
		if issue.Investigation.StartedAt == "" {
			issue.Investigation.StartedAt = now
			logging.LogDebug("Auto-populated investigation start time during move to active",
				"issue_id", issue.ID, "started_at", now)
		}
	case "completed":
		if issue.Metadata.ResolvedAt == "" {
			issue.Metadata.ResolvedAt = now
			logging.LogDebug("Auto-populated resolved_at during move to completed",
				"issue_id", issue.ID, "resolved_at", now)
		}
	}

	targetDir := fs.IssueDir(toFolder, issue.ID)
	if err := fs.ops.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return nil, "", "", err
	}

	if _, statErr := fs.ops.Stat(targetDir); statErr == nil {
		if err := fs.ops.RemoveAll(targetDir); err != nil {
			return nil, "", "", fmt.Errorf("failed to clear existing target %s: %w", targetDir, err)
		}
	}

	if err := fs.ops.Rename(issueDir, targetDir); err != nil {
		return nil, "", "", err
	}

	if err := fs.WriteIssueMetadata(targetDir, issue); err != nil {
		return nil, "", "", err
	}

	return issue, currentFolder, toFolder, nil
}
