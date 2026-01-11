package main

import (
	"time"

	bundleruntime "scenario-to-desktop-runtime"
	bundlemanifest "scenario-to-desktop-runtime/manifest"
)

type preflightStatusError struct {
	Status int
	Err    error
}

func (e *preflightStatusError) Error() string {
	return e.Err.Error()
}

type preflightIssue struct {
	status string
	detail string
}

type preflightSession struct {
	id         string
	manifest   *bundlemanifest.Manifest
	bundleDir  string
	appData    string
	supervisor *bundleruntime.Supervisor
	baseURL    string
	token      string
	createdAt  time.Time
	expiresAt  time.Time
}

type preflightJob struct {
	id        string
	status    string
	steps     map[string]BundlePreflightStep
	result    *BundlePreflightResponse
	err       string
	startedAt time.Time
	updatedAt time.Time
}
