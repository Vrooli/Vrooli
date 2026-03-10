package main

import (
	"net/http"
	"time"
)

func (s *Server) handleApprovedChanges(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	if !s.capabilities.IsAvailable(hctx.Ctx, "workspace-sandbox") {
		hctx.Resp.OK(ApprovedChangesResponse{
			Available: false,
			Warning:   "Workspace Sandbox is not running",
		})
		return
	}

	preview, err := s.sandbox.GetCommitPreview(hctx.Ctx, hctx.RepoDir)
	if err != nil {
		hctx.Resp.OK(ApprovedChangesResponse{
			Available: false,
			Warning:   err.Error(),
		})
		return
	}

	hctx.Resp.OK(normalizeApprovedChanges(preview))
}

func (s *Server) handleApprovedChangesPreview(w http.ResponseWriter, r *http.Request) {
	hctx := RepoOperation(w, r, s.git, s.repos, 5*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req ApprovedChangesPreviewRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	if !s.capabilities.IsAvailable(hctx.Ctx, "workspace-sandbox") {
		hctx.Resp.OK(ApprovedChangesResponse{
			Available: false,
			Warning:   "Workspace Sandbox is not running",
		})
		return
	}

	preview, err := s.sandbox.GetCommitPreviewForPaths(hctx.Ctx, hctx.RepoDir, req.Paths)
	if err != nil {
		hctx.Resp.OK(ApprovedChangesResponse{
			Available: false,
			Warning:   err.Error(),
		})
		return
	}

	hctx.Resp.OK(normalizeApprovedChanges(preview))
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
