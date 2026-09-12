package main

import (
	"context"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (s *Server) listRepositoriesConnect(ctx context.Context, _ *repov1.ListRepositoriesRequest) (*repov1.ListRepositoriesResponse, error) {
	repos, activeID, err := s.repos.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := make([]*repov1.RepoRecord, 0, len(repos))
	for _, repo := range repos {
		result = append(result, repoRecordProto(repo))
	}
	return &repov1.ListRepositoriesResponse{Repos: result, ActiveId: activeID, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) getActiveRepositoryConnect(ctx context.Context, _ *repov1.GetActiveRepositoryRequest) (*repov1.GetActiveRepositoryResponse, error) {
	repo, err := s.repos.GetActive(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	result := &repov1.GetActiveRepositoryResponse{Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}
	if repo != nil {
		result.Repo = repoRecordProto(*repo)
	}
	return result, nil
}

func (s *Server) setActiveRepositoryConnect(ctx context.Context, req *repov1.SetActiveRepositoryRequest) (*repov1.RepoMutationResponse, error) {
	if req.GetRepositoryId() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("repository_id is required"))
	}
	repo, err := s.repos.SetActive(ctx, req.GetRepositoryId())
	if err != nil {
		return nil, connectRepoRegistryError(err)
	}
	return &repov1.RepoMutationResponse{Repo: repoRecordProto(*repo), Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) openRepositoryConnect(ctx context.Context, req *repov1.OpenRepositoryRequest) (*repov1.RepoMutationResponse, error) {
	if strings.TrimSpace(req.GetPath()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("path is required"))
	}
	repo, err := s.repos.Open(ctx, req.GetPath())
	if err != nil {
		return nil, connectRepoRegistryError(err)
	}
	return &repov1.RepoMutationResponse{Repo: repoRecordProto(*repo), Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) cloneRepositoryConnect(ctx context.Context, req *repov1.CloneRepositoryRequest) (*repov1.RepoMutationResponse, error) {
	if strings.TrimSpace(req.GetUrl()) == "" || strings.TrimSpace(req.GetDestination()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("url and destination are required"))
	}
	repo, err := s.repos.Clone(ctx, req.GetUrl(), req.GetDestination())
	if err != nil {
		return nil, connectRepoRegistryError(err)
	}
	return &repov1.RepoMutationResponse{Repo: repoRecordProto(*repo), Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Server) removeRepositoryConnect(ctx context.Context, req *repov1.RemoveRepositoryRequest) (*repov1.RemoveRepositoryResponse, error) {
	if req.GetRepositoryId() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("repository_id is required"))
	}
	if err := s.repos.Remove(ctx, req.GetRepositoryId()); err != nil {
		return nil, connectRepoRegistryError(err)
	}
	return &repov1.RemoveRepositoryResponse{Removed: true, Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func repoRecordProto(repo RepoRecord) *repov1.RepoRecord {
	return &repov1.RepoRecord{Id: repo.ID, Path: repo.Path, Name: repo.Name, RemoteUrl: repo.RemoteURL, AddedAt: repo.AddedAt, LastOpenedAt: repo.LastOpenedAt, Favorite: repo.Favorite}
}

func connectRepoRegistryError(err error) error {
	if err == nil {
		return nil
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "not found") {
		return connect.NewError(connect.CodeNotFound, err)
	}
	if strings.Contains(message, "required") || strings.Contains(message, "invalid") || strings.Contains(message, "accessible") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
