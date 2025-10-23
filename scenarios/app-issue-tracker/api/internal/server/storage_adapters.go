package server

import (
	storagepkg "app-issue-tracker-api/internal/storage"
)

type IssueStorage = storagepkg.IssueStorage

func NewFileIssueStore(issuesDir string) IssueStorage {
	return storagepkg.NewFileIssueStore(issuesDir)
}

func (s *Server) issueDir(folder, issueID string) string {
	return s.store.IssueDir(folder, issueID)
}

func (s *Server) loadIssueFromDir(issueDir string) (*Issue, error) {
	return s.store.LoadIssueFromDir(issueDir)
}

func (s *Server) writeIssueMetadata(issueDir string, issue *Issue) error {
	return s.store.WriteIssueMetadata(issueDir, issue)
}

func (s *Server) saveIssue(issue *Issue, folder string) (string, error) {
	return s.store.SaveIssue(issue, folder)
}

func (s *Server) loadIssuesFromFolder(folder string) ([]Issue, error) {
	return s.store.LoadIssuesFromFolder(folder)
}

func (s *Server) findIssueDirectory(issueID string) (string, string, error) {
	return s.store.FindIssueDirectory(issueID)
}

func (s *Server) loadIssueWithStatus(issueID string) (*Issue, string, string, error) {
	return s.store.LoadIssueWithStatus(issueID)
}

func (s *Server) moveIssue(issueID, toFolder string) error {
	issue, fromFolder, newFolder, err := s.store.MoveIssue(issueID, toFolder)
	if err != nil {
		return err
	}

	if fromFolder == newFolder {
		return nil
	}

	s.hub.Publish(NewEvent(EventIssueStatusChanged, IssueStatusChangedData{
		IssueID:   issueID,
		OldStatus: fromFolder,
		NewStatus: newFolder,
	}))

	s.hub.Publish(NewEvent(EventIssueUpdated, IssueEventData{Issue: issue}))

	return nil
}

func (s *Server) getAllIssues(statusFilter, priorityFilter, typeFilter, appIDFilter string, limit int) ([]Issue, error) {
	return s.issues.ListIssues(statusFilter, priorityFilter, typeFilter, appIDFilter, limit)
}
