package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const repoIDHeader = "X-Repo-Id"

// RepoService coordinates repository registry and resolution logic.
type RepoService struct {
	store RepoStore
	git   GitRunner
}

// ResolvedRepo describes a repository resolved for a request.
type ResolvedRepo struct {
	ID     int64
	Path   string
	Source string
}

// NewRepoService creates a new repository service.
func NewRepoService(store RepoStore, git GitRunner) *RepoService {
	return &RepoService{store: store, git: git}
}

func (s *RepoService) List(ctx context.Context) ([]RepoRecord, int64, error) {
	if s.store == nil {
		return nil, 0, fmt.Errorf("repo store not configured")
	}
	repos, err := s.store.List(ctx)
	if err != nil {
		return nil, 0, err
	}
	active, err := s.store.GetActive(ctx)
	if err != nil {
		return nil, 0, err
	}
	var activeID int64
	if active != nil {
		activeID = active.ID
	}
	return repos, activeID, nil
}

func (s *RepoService) GetActive(ctx context.Context) (*RepoRecord, error) {
	if s.store == nil {
		return nil, fmt.Errorf("repo store not configured")
	}
	return s.store.GetActive(ctx)
}

func (s *RepoService) SetActive(ctx context.Context, id int64) (*RepoRecord, error) {
	if s.store == nil {
		return nil, fmt.Errorf("repo store not configured")
	}
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, newRepoError(RepoErrorNotFound, "repository not found", err)
		}
		return nil, err
	}
	if repo == nil {
		return nil, newRepoError(RepoErrorNotFound, "repository not found", nil)
	}
	if err := s.validateRepo(ctx, repo.Path); err != nil {
		return nil, newRepoError(RepoErrorInvalid, "repository is not accessible", err)
	}
	if err := s.store.SetActive(ctx, repo.ID); err != nil {
		return nil, err
	}
	_ = s.store.TouchLastOpened(ctx, repo.ID)
	return repo, nil
}

func (s *RepoService) Remove(ctx context.Context, id int64) error {
	if s.store == nil {
		return fmt.Errorf("repo store not configured")
	}
	active, _ := s.store.GetActive(ctx)
	if err := s.store.Delete(ctx, id); err != nil {
		if err == sql.ErrNoRows {
			return newRepoError(RepoErrorNotFound, "repository not found", err)
		}
		return err
	}
	if active != nil && active.ID == id {
		_ = s.store.ClearActive(ctx)
	}
	return nil
}

func (s *RepoService) Open(ctx context.Context, path string) (*RepoRecord, error) {
	if s.store == nil {
		return nil, fmt.Errorf("repo store not configured")
	}
	root, err := s.resolveRepoRoot(ctx, path)
	if err != nil {
		return nil, err
	}
	name := repoNameFromPath(root)
	remoteURL := s.safeRemoteURL(ctx, root)

	repo, err := s.store.Upsert(ctx, RepoRecord{
		Path:      root,
		Name:      name,
		RemoteURL: remoteURL,
	})
	if err != nil {
		return nil, err
	}
	if err := s.store.SetActive(ctx, repo.ID); err != nil {
		return nil, err
	}
	_ = s.store.TouchLastOpened(ctx, repo.ID)
	return &repo, nil
}

func (s *RepoService) Clone(ctx context.Context, url string, destination string) (*RepoRecord, error) {
	if s.store == nil {
		return nil, fmt.Errorf("repo store not configured")
	}
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, newRepoError(RepoErrorInvalid, "clone url is required", nil)
	}
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return nil, newRepoError(RepoErrorInvalid, "destination path is required", nil)
	}

	if err := s.git.Clone(ctx, destination, url, nil); err != nil {
		return nil, newRepoError(RepoErrorInvalid, "clone failed", err)
	}

	return s.Open(ctx, destination)
}

func (s *RepoService) Resolve(ctx context.Context, r *http.Request) (ResolvedRepo, error) {
	repoID, hasRepoID, err := repoIDFromRequest(r)
	if err != nil {
		return ResolvedRepo{}, newRepoError(RepoErrorInvalid, "invalid repo id", err)
	}

	if hasRepoID {
		repo, err := s.resolveByID(ctx, repoID)
		if err != nil {
			return ResolvedRepo{}, err
		}
		return ResolvedRepo{ID: repo.ID, Path: repo.Path, Source: "explicit"}, nil
	}

	if resolved, tried, resolveErr := s.resolveFromActive(ctx); tried {
		return resolved, resolveErr
	}

	return s.resolveFromRoot(ctx)
}

// resolveFromActive attempts to resolve from the active repo in the store.
// Returns (repo, nil, true) on success, (empty, err, true) on validation failure,
// or (empty, nil, false) if no active repo was found.
func (s *RepoService) resolveFromActive(ctx context.Context) (ResolvedRepo, bool, error) {
	if s.store == nil {
		return ResolvedRepo{}, false, nil
	}
	active, err := s.store.GetActive(ctx)
	if err != nil || active == nil {
		return ResolvedRepo{}, false, nil
	}
	if err := s.validateRepo(ctx, active.Path); err != nil {
		return ResolvedRepo{}, true, newRepoError(RepoErrorInvalid, "active repository is not accessible", err)
	}
	_ = s.store.TouchLastOpened(ctx, active.ID)
	return ResolvedRepo{ID: active.ID, Path: active.Path, Source: "active"}, true, nil
}

func (s *RepoService) resolveFromRoot(ctx context.Context) (ResolvedRepo, error) {
	root := strings.TrimSpace(s.git.ResolveRepoRoot(ctx))
	if root == "" {
		return ResolvedRepo{}, newRepoError(RepoErrorMissing, "repository root could not be resolved", nil)
	}

	if s.store == nil {
		return ResolvedRepo{Path: root, Source: "fallback"}, nil
	}

	repo, err := s.ensureRepo(ctx, root)
	if err != nil {
		return ResolvedRepo{}, err
	}
	_ = s.store.SetActive(ctx, repo.ID)
	return ResolvedRepo{ID: repo.ID, Path: repo.Path, Source: "fallback"}, nil
}

func (s *RepoService) resolveByID(ctx context.Context, id int64) (*RepoRecord, error) {
	if s.store == nil {
		return nil, fmt.Errorf("repo store not configured")
	}
	repo, err := s.store.GetByID(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, newRepoError(RepoErrorNotFound, "repository not found", err)
		}
		return nil, err
	}
	if repo == nil {
		return nil, newRepoError(RepoErrorNotFound, "repository not found", nil)
	}
	if err := s.validateRepo(ctx, repo.Path); err != nil {
		return nil, newRepoError(RepoErrorInvalid, "repository is not accessible", err)
	}
	_ = s.store.TouchLastOpened(ctx, repo.ID)
	return repo, nil
}

func (s *RepoService) ensureRepo(ctx context.Context, root string) (*RepoRecord, error) {
	if s.store == nil {
		return nil, fmt.Errorf("repo store not configured")
	}
	name := repoNameFromPath(root)
	remoteURL := s.safeRemoteURL(ctx, root)
	record, err := s.store.Upsert(ctx, RepoRecord{
		Path:      root,
		Name:      name,
		RemoteURL: remoteURL,
	})
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (s *RepoService) validateRepo(ctx context.Context, path string) error {
	out, err := s.git.RevParse(ctx, path, "--is-inside-work-tree")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("not inside a git work tree")
	}
	return nil
}

func (s *RepoService) resolveRepoRoot(ctx context.Context, raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "", newRepoError(RepoErrorInvalid, "repository path is required", nil)
	}
	path = expandHome(path)
	out, err := s.git.RevParse(ctx, path, "--show-toplevel")
	if err != nil {
		return "", newRepoError(RepoErrorInvalid, "path is not a git repository", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", newRepoError(RepoErrorInvalid, "repository root could not be resolved", nil)
	}
	return root, nil
}

func (s *RepoService) safeRemoteURL(ctx context.Context, root string) string {
	remoteURL, err := s.git.GetRemoteURL(ctx, root, "origin")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(remoteURL)
}

func repoIDFromRequest(r *http.Request) (int64, bool, error) {
	if r == nil {
		return 0, false, nil
	}
	repoIDRaw := strings.TrimSpace(r.Header.Get(repoIDHeader))
	if repoIDRaw == "" {
		repoIDRaw = strings.TrimSpace(r.URL.Query().Get("repo_id"))
	}
	if repoIDRaw == "" {
		return 0, false, nil
	}
	var id int64
	if _, err := fmt.Sscanf(repoIDRaw, "%d", &id); err != nil || id <= 0 {
		if err == nil {
			err = fmt.Errorf("repo id must be positive")
		}
		return 0, true, err
	}
	return id, true, nil
}

func repoNameFromPath(path string) string {
	base := filepath.Base(strings.TrimSpace(path))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "repository"
	}
	return base
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
