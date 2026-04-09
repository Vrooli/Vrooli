package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// handleListCredentials handles GET /api/v1/credentials
func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	result, err := ListCredentials(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
		Store:   s.credStore,
	})
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleSaveCredential handles POST /api/v1/credentials
func (s *Server) handleSaveCredential(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req CredentialSaveRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := SaveCredential(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
		Store:   s.credStore,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}

// handleDeleteCredential handles DELETE /api/v1/credentials/{id}
func (s *Server) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)

	vars := mux.Vars(r)
	id := vars["id"]
	if strings.TrimSpace(id) == "" {
		resp.BadRequest("credential ID is required")
		return
	}

	result, err := DeleteCredential(ctx, CredentialsDeps{}, CredentialDeleteRequest{ID: id})
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		resp.UnprocessableEntity(result)
		return
	}
	resp.OK(result)
}

// handleTestCredential handles POST /api/v1/credentials/test
func (s *Server) handleTestCredential(w http.ResponseWriter, r *http.Request) {
	hctx := RepoRead(w, r, s.git, s.repos, 30*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req CredentialTestRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := TestCredential(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
		Store:   s.credStore,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	hctx.Resp.OK(result)
}

// handleUpdateRemoteURL handles POST /api/v1/repo/remote/url
func (s *Server) handleUpdateRemoteURL(w http.ResponseWriter, r *http.Request) {
	hctx := RepoWrite(w, r, s.git, s.repos, s.repoLock, 10*time.Second)
	if hctx == nil {
		return
	}
	defer hctx.Cancel()

	var req RemoteURLUpdateRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}

	result, err := UpdateRemoteURL(hctx.Ctx, CredentialsDeps{
		Git:     hctx.Git,
		RepoDir: hctx.RepoDir,
	}, req)
	if err != nil {
		hctx.Resp.InternalError(err.Error())
		return
	}

	if !result.Success {
		hctx.Resp.UnprocessableEntity(result)
		return
	}
	hctx.Resp.OK(result)
}
