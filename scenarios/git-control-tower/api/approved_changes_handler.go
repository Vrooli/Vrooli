package main

import (
	"context"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleApprovedChanges(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	preview, err := s.sandbox.GetCommitPreview(ctx, repoDir)
	if err != nil {
		resp.OK(ApprovedChangesResponse{
			Available: false,
			Warning:   err.Error(),
		})
		return
	}

	resp.OK(normalizeApprovedChanges(preview))
}

func (s *Server) handleApprovedChangesPreview(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repoDir := s.git.ResolveRepoRoot(ctx)
	if strings.TrimSpace(repoDir) == "" {
		resp.BadRequest("repository root could not be resolved")
		return
	}

	var req ApprovedChangesPreviewRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	preview, err := s.sandbox.GetCommitPreviewForPaths(ctx, repoDir, req.Paths)
	if err != nil {
		resp.OK(ApprovedChangesResponse{
			Available: false,
			Warning:   err.Error(),
		})
		return
	}

	resp.OK(normalizeApprovedChanges(preview))
}

func normalizeApprovedChanges(preview *workspaceSandboxCommitPreview) ApprovedChangesResponse {
	if preview == nil {
		return ApprovedChangesResponse{Available: false}
	}

	files := make([]ApprovedChangeFile, 0, len(preview.Files))
	for _, file := range preview.Files {
		files = append(files, ApprovedChangeFile{
			RelativePath: file.RelativePath,
			Status:       file.Status,
			SandboxID:    file.SandboxID,
			SandboxOwner: file.SandboxOwner,
			ChangeType:   file.ChangeType,
		})
	}

	return ApprovedChangesResponse{
		Available:        true,
		CommittableFiles: preview.CommittableFiles,
		SuggestedMessage: preview.SuggestedMessage,
		Files:            files,
	}
}
