package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandlerContext provides common dependencies and utilities for HTTP handlers.
// This reduces cognitive load by centralizing request handling patterns.
type HandlerContext struct {
	Git     GitRunner
	RepoDir string
	RepoID  int64
	Ctx     context.Context
	Resp    *HTTPResponse
	Cancel  context.CancelFunc
}

// RepoRead creates a HandlerContext for read-only repository operations.
//
// No per-repo lock is acquired because read-only git commands use
// --no-optional-locks (via readArgs in git_runner_core.go), which prevents
// them from contending for .git/index.lock. This allows reads to proceed
// concurrently with each other and with write operations, eliminating the
// serial bottleneck that caused ~5s latency when polling queries queued
// behind the lock.
//
// DECISION BOUNDARY: This is where we determine if a request has a valid repository context.
// All read-only repo handlers should use this for consistent error handling.
func RepoRead(w http.ResponseWriter, r *http.Request, git GitRunner, repos *RepoService, timeout time.Duration) *HandlerContext {
	resp := NewResponse(w)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)

	repoDir, repoID, err := resolveRepo(ctx, git, repos, r)
	if err != nil {
		cancel()
		writeRepoError(resp, err)
		return nil
	}

	return &HandlerContext{
		Git:     git,
		RepoDir: repoDir,
		RepoID:  repoID,
		Ctx:     ctx,
		Resp:    resp,
		Cancel:  cancel,
	}
}

// RepoWrite creates a HandlerContext for operations that modify the git
// index or working tree (stage, unstage, commit, checkout, discard, etc.).
//
// Acquires a per-repo lock to serialize index-modifying commands and prevent
// .git/index.lock contention between concurrent requests. Cleans stale lock
// files before acquiring the application-level lock.
//
// DECISION BOUNDARY: This is where we determine if a request has a valid repository context.
// All index-modifying repo handlers should use this for consistent error handling.
func RepoWrite(w http.ResponseWriter, r *http.Request, git GitRunner, repos *RepoService, repoLock *RepoLock, timeout time.Duration) *HandlerContext {
	resp := NewResponse(w)
	ctx, cancel := context.WithTimeout(r.Context(), timeout)

	repoDir, repoID, err := resolveRepo(ctx, git, repos, r)
	if err != nil {
		cancel()
		writeRepoError(resp, err)
		return nil
	}

	// Remove stale .git/index.lock left by crashed git processes before
	// acquiring our application-level lock. This must happen outside the
	// lock to avoid blocking other requests while we stat/remove the file.
	cleanStaleLock(repoDir)

	// Acquire the per-repo lock to serialize git index operations.
	var unlock func()
	if repoLock != nil {
		var lockErr error
		unlock, lockErr = repoLock.Acquire(ctx, repoDir)
		if lockErr != nil {
			cancel()
			log.Printf("repo lock acquisition timed out for %s: %v", repoDir, lockErr)
			resp.Error(http.StatusServiceUnavailable, "repository is busy, please retry")
			return nil
		}
	}

	return &HandlerContext{
		Git:     git,
		RepoDir: repoDir,
		RepoID:  repoID,
		Ctx:     ctx,
		Resp:    resp,
		Cancel: func() {
			cancel()
			if unlock != nil {
				unlock()
			}
		},
	}
}

// resolveRepo determines the repository path and ID from the request.
func resolveRepo(ctx context.Context, git GitRunner, repos *RepoService, r *http.Request) (string, int64, error) {
	if repos != nil {
		resolved, err := repos.Resolve(ctx, r)
		if err != nil {
			return "", 0, err
		}
		return resolved.Path, resolved.ID, nil
	}
	repoDir := git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		return "", 0, fmt.Errorf("repository root could not be resolved")
	}
	return repoDir, 0, nil
}

// writeRepoError sends the appropriate HTTP error response for repo resolution failures.
func writeRepoError(resp *HTTPResponse, err error) {
	repoErr := RepoError{}
	if errors.As(err, &repoErr) {
		switch repoErr.Kind {
		case RepoErrorNotFound:
			resp.NotFound(repoErr.Error())
		default:
			resp.BadRequest(repoErr.Error())
		}
	} else {
		resp.BadRequest(err.Error())
	}
}

// ParseJSONBody decodes a JSON request body into the target struct.
// Returns false and writes an error response if parsing fails.
//
// DECISION BOUNDARY: This is where we decide if a request body is valid JSON.
func ParseJSONBody(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	resp := NewResponse(w)
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		resp.BadRequest("invalid request body: " + err.Error())
		return false
	}
	return true
}

// StagingRequest represents the common fields for stage/unstage operations.
// This avoids duplication between StageRequest and UnstageRequest handling.
type StagingRequest interface {
	GetPaths() []string
	GetScope() string
}

// Implement StagingRequest for StageRequest
func (r StageRequest) GetPaths() []string { return r.Paths }
func (r StageRequest) GetScope() string   { return r.Scope }

// Implement StagingRequest for UnstageRequest
func (r UnstageRequest) GetPaths() []string { return r.Paths }
func (r UnstageRequest) GetScope() string   { return r.Scope }

// ValidateStagingRequest checks that a staging request has either paths or scope.
// Returns false and writes an error response if validation fails.
//
// DECISION BOUNDARY: This defines what constitutes a valid staging request.
func ValidateStagingRequest(w http.ResponseWriter, req StagingRequest) bool {
	resp := NewResponse(w)
	if len(req.GetPaths()) == 0 && req.GetScope() == "" {
		resp.BadRequest("paths or scope required")
		return false
	}
	return true
}
