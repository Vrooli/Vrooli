package main

import (
	"context"
	"encoding/json"
	"errors"
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

// RepoOperation creates a HandlerContext for repository operations.
// Returns nil and writes an error response if the repository cannot be resolved.
//
// DECISION BOUNDARY: This is where we determine if a request has a valid repository context.
// All repo-dependent handlers should use this to ensure consistent error handling.
func RepoOperation(w http.ResponseWriter, r *http.Request, git GitRunner, repos *RepoService, timeout time.Duration) *HandlerContext {
	resp := NewResponse(w)

	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	// Note: caller must defer cancel() if HandlerContext is returned non-nil
	// We store cancel in closure for cleanup on error paths

	var repoDir string
	var repoID int64
	if repos != nil {
		resolved, err := repos.Resolve(ctx, r)
		if err != nil {
			cancel()
			repoErr := RepoError{}
			if errors.As(err, &repoErr) {
				switch repoErr.Kind {
				case RepoErrorNotFound:
					resp.NotFound(repoErr.Error())
				case RepoErrorInvalid, RepoErrorMissing:
					resp.BadRequest(repoErr.Error())
				default:
					resp.BadRequest(repoErr.Error())
				}
			} else {
				resp.BadRequest(err.Error())
			}
			return nil
		}
		repoDir = resolved.Path
		repoID = resolved.ID
	} else {
		repoDir = git.ResolveRepoRoot(ctx)
		if strings.TrimSpace(repoDir) == "" {
			cancel()
			resp.BadRequest("repository root could not be resolved")
			return nil
		}
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
