package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// RepoListResponse returns the repository registry and active repo id.
type RepoListResponse struct {
	Repos     []RepoRecord `json:"repos"`
	ActiveID  int64        `json:"active_id,omitempty"`
	Timestamp string       `json:"timestamp"`
}

// RepoActiveResponse returns the currently active repository.
type RepoActiveResponse struct {
	Repo      *RepoRecord `json:"repo,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// RepoOpenRequest opens an existing repository.
type RepoOpenRequest struct {
	Path string `json:"path"`
}

// RepoCloneRequest clones a repository to a destination.
type RepoCloneRequest struct {
	URL         string `json:"url"`
	Destination string `json:"destination"`
}

// RepoActiveRequest sets the active repository by id.
type RepoActiveRequest struct {
	ID int64 `json:"id"`
}

// RepoMutationResponse returns a single repo mutation result.
type RepoMutationResponse struct {
	Repo      *RepoRecord `json:"repo,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// RepoRemoveResponse returns result of removal.
type RepoRemoveResponse struct {
	Removed   bool   `json:"removed"`
	Timestamp string `json:"timestamp"`
}

func (s *Server) handleRepoList(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	repos, activeID, err := s.repos.List(ctx)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(RepoListResponse{
		Repos:     repos,
		ActiveID:  activeID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleRepoActive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	active, err := s.repos.GetActive(ctx)
	if err != nil {
		resp.InternalError(err.Error())
		return
	}

	resp.OK(RepoActiveResponse{
		Repo:      active,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleRepoSetActive(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	var req RepoActiveRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if req.ID <= 0 {
		resp.BadRequest("repository id is required")
		return
	}

	repo, err := s.repos.SetActive(ctx, req.ID)
	if err != nil {
		s.respondRepoError(resp, err)
		return
	}

	resp.OK(RepoMutationResponse{
		Repo:      repo,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleRepoOpen(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	resp := NewResponse(w)
	var req RepoOpenRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Path) == "" {
		resp.BadRequest("path is required")
		return
	}

	repo, err := s.repos.Open(ctx, req.Path)
	if err != nil {
		s.respondRepoError(resp, err)
		return
	}

	resp.OK(RepoMutationResponse{
		Repo:      repo,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleRepoClone(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp := NewResponse(w)
	var req RepoCloneRequest
	if !ParseJSONBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.URL) == "" {
		resp.BadRequest("url is required")
		return
	}
	if strings.TrimSpace(req.Destination) == "" {
		resp.BadRequest("destination is required")
		return
	}

	repo, err := s.repos.Clone(ctx, req.URL, req.Destination)
	if err != nil {
		s.respondRepoError(resp, err)
		return
	}

	resp.OK(RepoMutationResponse{
		Repo:      repo,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleRepoRemove(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp := NewResponse(w)
	idRaw := strings.TrimSpace(mux.Vars(r)["id"])
	if idRaw == "" {
		resp.BadRequest("repository id is required")
		return
	}
	var id int64
	if _, err := fmt.Sscanf(idRaw, "%d", &id); err != nil || id <= 0 {
		resp.BadRequest("repository id is invalid")
		return
	}

	if err := s.repos.Remove(ctx, id); err != nil {
		s.respondRepoError(resp, err)
		return
	}

	resp.OK(RepoRemoveResponse{Removed: true, Timestamp: time.Now().UTC().Format(time.RFC3339)})
}

func (s *Server) respondRepoError(resp *HTTPResponse, err error) {
	var repoErr RepoError
	if errors.As(err, &repoErr) {
		switch repoErr.Kind {
		case RepoErrorNotFound:
			resp.NotFound(repoErr.Error())
		case RepoErrorInvalid:
			resp.BadRequest(repoErr.Error())
		case RepoErrorMissing:
			resp.BadRequest(repoErr.Error())
		default:
			resp.BadRequest(repoErr.Error())
		}
		return
	}
	resp.InternalError(err.Error())
}
