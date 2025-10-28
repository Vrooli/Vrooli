package services

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	issuespkg "app-issue-tracker-api/internal/issues"
	"app-issue-tracker-api/internal/logging"
	"app-issue-tracker-api/internal/server/metadata"
	storagepkg "app-issue-tracker-api/internal/storage"
)

type ProcessorInspector interface {
	IsRunning(issueID string) bool
}

type StatusChange struct {
	From string
	To   string
}

type IssueService struct {
	store     storagepkg.IssueStorage
	artifacts *ArtifactManager
	processor ProcessorInspector
	now       func() time.Time
}

func NewIssueService(store storagepkg.IssueStorage, artifacts *ArtifactManager, processor ProcessorInspector, now func() time.Time) *IssueService {
	if artifacts == nil {
		artifacts = NewArtifactManager()
	}
	if now == nil {
		now = time.Now
	}
	return &IssueService{
		store:     store,
		artifacts: artifacts,
		processor: processor,
		now:       now,
	}
}

func (s *IssueService) CreateIssue(req *issuespkg.CreateIssueRequest) (*issuespkg.Issue, string, error) {
	issue, targetStatus, err := issuespkg.PrepareIssueForCreate(req, s.now().UTC())
	if err != nil {
		return nil, "", err
	}

	issueDir := s.store.IssueDir(targetStatus, issue.ID)
	if err := os.MkdirAll(issueDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("failed to prepare issue directory: %w", err)
	}

	artifactPayloads := MergeCreateArtifacts(req)
	if err := s.artifacts.StoreIssueArtifacts(issue, issueDir, artifactPayloads, true); err != nil {
		return nil, "", fmt.Errorf("failed to store issue artifacts: %w", err)
	}

	if _, err := s.store.SaveIssue(issue, targetStatus); err != nil {
		return nil, "", fmt.Errorf("failed to persist issue: %w", err)
	}

	storagePath := filepath.Join(targetStatus, issue.ID)
	return issue, storagePath, nil
}

func (s *IssueService) UpdateIssue(issueID string, req *issuespkg.UpdateIssueRequest) (*issuespkg.Issue, *StatusChange, error) {
	issue, issueDir, currentFolder, err := s.store.LoadIssueWithStatus(issueID)
	if err != nil {
		return nil, nil, err
	}

	targetStatus, err := issuespkg.ApplyUpdateRequest(issue, req, currentFolder)
	if err != nil {
		return nil, nil, err
	}

	updatedDir := issueDir
	updatedFolder := currentFolder
	var statusChange *StatusChange
	var rollback func() error

	if targetStatus != currentFolder {
		dir, folder, change, revert, transitionErr := s.transitionIssueStatus(issueID, issue, issueDir, currentFolder, targetStatus)
		if transitionErr != nil {
			return nil, nil, transitionErr
		}
		updatedDir = dir
		updatedFolder = folder
		statusChange = change
		rollback = revert
	}

	if err := s.artifacts.StoreIssueArtifacts(issue, updatedDir, req.Artifacts, false); err != nil {
		if rollback != nil {
			if rbErr := rollback(); rbErr != nil {
				logging.LogWarn("Failed to rollback issue move after artifact error", "issue_id", issueID, "error", rbErr)
			}
		}
		return nil, nil, fmt.Errorf("failed to store additional artifacts: %w", err)
	}

	issue.Status = updatedFolder

	if err := s.store.WriteIssueMetadata(updatedDir, issue); err != nil {
		if rollback != nil {
			if rbErr := rollback(); rbErr != nil {
				logging.LogWarn("Failed to rollback issue move after metadata error", "issue_id", issueID, "error", rbErr)
			}
		}
		return nil, nil, fmt.Errorf("failed to persist updated issue: %w", err)
	}

	return issue, statusChange, nil
}

func (s *IssueService) transitionIssueStatus(issueID string, issue *issuespkg.Issue, issueDir, currentFolder, targetStatus string) (string, string, *StatusChange, func() error, error) {
	if s.processor != nil && targetStatus != "active" && s.processor.IsRunning(issueID) {
		return "", "", nil, nil, ErrIssueRunning
	}

	now := s.now().UTC().Format(time.RFC3339)
	isBackwardsTransition := (currentFolder == "completed" || currentFolder == "failed" || currentFolder == "active") && targetStatus == "open"

	if isBackwardsTransition {
		logging.LogInfo(
			"Issue status moved backwards, clearing investigation data",
			"issue_id", issueID,
			"from", currentFolder,
			"to", targetStatus,
		)
		ResetInvestigationForReopen(issue)
	}

	if targetStatus == "active" && strings.TrimSpace(issue.Investigation.StartedAt) == "" {
		issue.Investigation.StartedAt = now
		logging.LogDebug("Auto-populated investigation start time during status update",
			"issue_id", issueID, "started_at", now)
	}
	if targetStatus == "completed" && strings.TrimSpace(issue.Metadata.ResolvedAt) == "" {
		issue.Metadata.ResolvedAt = now
		logging.LogDebug("Auto-populated resolved_at during status update",
			"issue_id", issueID, "resolved_at", now)
	}

	if issue.Metadata.Extra != nil {
		delete(issue.Metadata.Extra, metadata.AgentLastErrorKey)
	}

	targetDir := s.store.IssueDir(targetStatus, issue.ID)
	if _, statErr := os.Stat(targetDir); statErr == nil {
		return "", "", nil, nil, ErrIssueTargetExists
	} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", "", nil, nil, fmt.Errorf("failed to inspect target directory: %w", statErr)
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to prepare target directory: %w", err)
	}

	if err := os.Rename(issueDir, targetDir); err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to move issue: %w", err)
	}

	rolledBack := false
	rollback := func() error {
		if rolledBack {
			return nil
		}
		if err := os.Rename(targetDir, issueDir); err != nil {
			return fmt.Errorf("failed to rollback issue move: %w", err)
		}
		rolledBack = true
		return nil
	}

	return targetDir, targetStatus, &StatusChange{From: currentFolder, To: targetStatus}, rollback, nil
}

func (s *IssueService) DeleteIssue(issueID string) error {
	if s.processor != nil && s.processor.IsRunning(issueID) {
		return ErrIssueRunning
	}

	issueDir, _, err := s.store.FindIssueDirectory(issueID)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(issueDir); err != nil {
		return fmt.Errorf("failed to delete issue: %w", err)
	}
	return nil
}

func (s *IssueService) LoadIssueWithStatus(issueID string) (*issuespkg.Issue, string, string, error) {
	return s.store.LoadIssueWithStatus(issueID)
}

func (s *IssueService) LoadIssueFromDir(issueDir string) (*issuespkg.Issue, error) {
	return s.store.LoadIssueFromDir(issueDir)
}

func (s *IssueService) FindIssueDirectory(issueID string) (string, string, error) {
	return s.store.FindIssueDirectory(issueID)
}

func (s *IssueService) ListIssues(statusFilter, priorityFilter, typeFilter, appIDFilter string, limit int) ([]issuespkg.Issue, error) {
	priorityFilter = strings.TrimSpace(priorityFilter)
	typeFilter = strings.TrimSpace(typeFilter)
	appIDFilter = strings.TrimSpace(appIDFilter)
	var folders []string
	if trimmed := strings.TrimSpace(statusFilter); trimmed != "" {
		if normalized, ok := issuespkg.NormalizeStatus(trimmed); ok {
			folders = []string{normalized}
		} else {
			return nil, fmt.Errorf("%w: %s", ErrInvalidStatusFilter, trimmed)
		}
	} else {
		folders = issuespkg.ValidStatuses()
	}

	filtered := make([]issuespkg.Issue, 0)
	for _, folder := range folders {
		folderIssues, err := s.store.LoadIssuesFromFolder(folder)
		if err != nil {
			logging.LogWarn("Failed to load issues from folder", "folder", folder, "error", err)
			continue
		}

		for _, issue := range folderIssues {
			if priorityFilter != "" && !strings.EqualFold(issue.Priority, priorityFilter) {
				continue
			}
			if typeFilter != "" && !strings.EqualFold(issue.Type, typeFilter) {
				continue
			}
			if appIDFilter != "" && !strings.EqualFold(issue.AppID, appIDFilter) {
				continue
			}
			filtered = append(filtered, issue)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Metadata.CreatedAt > filtered[j].Metadata.CreatedAt
	})

	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return filtered, nil
}

func (s *IssueService) SearchIssues(query string, limit int) ([]issuespkg.Issue, error) {
	queryLower := strings.ToLower(query)
	var results []issuespkg.Issue

	for _, folder := range issuespkg.ValidStatuses() {
		issues, err := s.store.LoadIssuesFromFolder(folder)
		if err != nil {
			continue
		}

		for _, issue := range issues {
			searchText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
				issue.Title,
				issue.Description,
				issue.ErrorContext.ErrorMessage,
				strings.Join(issue.Metadata.Tags, " "),
			))

			if strings.Contains(searchText, queryLower) {
				results = append(results, issue)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		iTitleMatch := strings.Contains(strings.ToLower(results[i].Title), queryLower)
		jTitleMatch := strings.Contains(strings.ToLower(results[j].Title), queryLower)
		if iTitleMatch != jTitleMatch {
			return iTitleMatch
		}
		return results[i].Metadata.CreatedAt > results[j].Metadata.CreatedAt
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func (s *IssueService) StoreArtifacts(issue *issuespkg.Issue, issueDir string, payloads []issuespkg.ArtifactPayload, replaceExisting bool) error {
	return s.artifacts.StoreIssueArtifacts(issue, issueDir, payloads, replaceExisting)
}
