package services

import (
	"errors"

	storagepkg "app-issue-tracker-api/internal/storage"
)

var (
	ErrIssueRunning        = errors.New("cannot change issue status while an agent is running")
	ErrIssueTargetExists   = errors.New("issue already exists in target status")
	ErrIssueNotFound       = storagepkg.ErrIssueNotFound
	ErrInvalidStatusFilter = errors.New("invalid status filter")
)
