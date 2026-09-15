// Package effortworkspace exposes a bounded, read-only projection of the
// protected Plan Manager effort workspaces linked from a Prompt Manager team.
package effortworkspace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"prompt-manager/internal/store"
)

const (
	maxWorkspaces = 8
	maxFiles      = 500
	maxFileBytes  = 1 << 20
)

type Store struct {
	root      string
	teamStore store.TeamStore
}

type File struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
}

type Workspace struct {
	EffortRef string `json:"effortRef"`
	Slug      string `json:"slug"`
	Stage     string `json:"stage,omitempty"`
	Files     []File `json:"files"`
}

type Unavailable struct {
	EffortRef string `json:"effortRef"`
	Reason    string `json:"reason"`
}

type ListResult struct {
	TeamID      string        `json:"teamId"`
	Workspaces  []Workspace   `json:"workspaces"`
	Unavailable []Unavailable `json:"unavailable"`
}

type Content struct {
	EffortRef string `json:"effortRef"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

type manifest struct {
	EffortRef string `json:"effort_ref"`
	Slug      string `json:"slug"`
	Stage     string `json:"stage"`
}

func New(root string, teamStore store.TeamStore) *Store {
	return &Store{root: filepath.Clean(root), teamStore: teamStore}
}

func (s *Store) List(ctx context.Context, teamID string) (ListResult, error) {
	team, err := s.teamStore.Get(ctx, teamID)
	if err != nil {
		return ListResult{}, err
	}
	refs := uniqueNonEmpty(team.EffortRefs)
	if len(refs) > maxWorkspaces {
		refs = refs[:maxWorkspaces]
	}
	result := ListResult{TeamID: teamID, Workspaces: []Workspace{}, Unavailable: []Unavailable{}}
	for _, ref := range refs {
		workspace, err := s.resolve(ref)
		if err != nil {
			result.Unavailable = append(result.Unavailable, Unavailable{EffortRef: ref, Reason: err.Error()})
			continue
		}
		workspace.Files, err = listFiles(workspace.root)
		if err != nil {
			result.Unavailable = append(result.Unavailable, Unavailable{EffortRef: ref, Reason: err.Error()})
			continue
		}
		result.Workspaces = append(result.Workspaces, workspace.Workspace)
	}
	sort.Slice(result.Workspaces, func(i, j int) bool { return result.Workspaces[i].EffortRef < result.Workspaces[j].EffortRef })
	return result, nil
}

func (s *Store) Read(ctx context.Context, effortRef, relPath string) (Content, error) {
	if strings.TrimSpace(effortRef) == "" {
		return Content{}, errors.New("effortRef is required")
	}
	workspace, err := s.resolve(effortRef)
	if err != nil {
		return Content{}, err
	}
	clean, err := cleanRelative(relPath)
	if err != nil {
		return Content{}, err
	}
	fullPath := filepath.Join(workspace.root, filepath.FromSlash(clean))
	if err := ensureContained(workspace.root, fullPath); err != nil {
		return Content{}, err
	}
	info, err := os.Lstat(fullPath)
	if err != nil {
		return Content{}, fmt.Errorf("workspace file not found: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Content{}, errors.New("workspace path is not a regular file")
	}
	if info.Size() > maxFileBytes {
		return Content{}, fmt.Errorf("workspace file exceeds %d-byte read limit", maxFileBytes)
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return Content{}, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if err != nil {
		return Content{}, err
	}
	if len(b) > maxFileBytes {
		return Content{}, fmt.Errorf("workspace file exceeds %d-byte read limit", maxFileBytes)
	}
	return Content{EffortRef: effortRef, Path: clean, Content: string(b)}, nil
}

type resolvedWorkspace struct {
	Workspace
	root string
}

func (s *Store) resolve(effortRef string) (resolvedWorkspace, error) {
	if strings.TrimSpace(effortRef) == "" {
		return resolvedWorkspace{}, errors.New("effortRef is required")
	}
	effortsRoot := filepath.Join(s.root, "efforts")
	entries, err := os.ReadDir(effortsRoot)
	if err != nil {
		return resolvedWorkspace{}, fmt.Errorf("list effort workspaces: %w", err)
	}
	legacySlug := legacyWorkspaceSlug(effortRef)
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		workspaceRoot := filepath.Join(effortsRoot, entry.Name())
		if info, err := os.Lstat(workspaceRoot); err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		manifestPath := filepath.Join(workspaceRoot, "effort.json")
		manifestInfo, err := os.Lstat(manifestPath)
		if err != nil || manifestInfo.Mode()&os.ModeSymlink != 0 || !manifestInfo.Mode().IsRegular() || manifestInfo.Size() > maxFileBytes {
			continue
		}
		manifestBytes, err := os.ReadFile(manifestPath)
		if err != nil || len(manifestBytes) > maxFileBytes {
			continue
		}
		var m manifest
		if json.Unmarshal(manifestBytes, &m) != nil {
			continue
		}
		exactMatch := m.EffortRef == effortRef
		legacyMatch := legacySlug != "" && entry.Name() == legacySlug && m.EffortRef == ""
		if !exactMatch && !legacyMatch {
			continue
		}
		slug := m.Slug
		if slug == "" {
			slug = entry.Name()
		}
		return resolvedWorkspace{Workspace: Workspace{EffortRef: effortRef, Slug: slug, Stage: m.Stage}, root: workspaceRoot}, nil
	}
	return resolvedWorkspace{}, fmt.Errorf("effort workspace is unavailable for %s", effortRef)
}

// legacyWorkspaceSlug extracts the bounded folder identity from the older
// workspace reference form. The path portion is deliberately ignored; only a
// single safe slug may select a child of the configured PlanArtifacts root.
func legacyWorkspaceSlug(effortRef string) string {
	if strings.HasPrefix(effortRef, "workspace:") {
		parts := strings.SplitN(strings.TrimPrefix(effortRef, "workspace:"), "#", 2)
		if len(parts) == 2 {
			return safeSlug(parts[1])
		}
	}
	if strings.HasPrefix(effortRef, "legacy-effort:") {
		return safeSlug(strings.TrimPrefix(effortRef, "legacy-effort:"))
	}
	return ""
}

func safeSlug(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." || strings.ContainsAny(value, `/\\`) || strings.HasPrefix(value, ".") {
		return ""
	}
	return value
}

func listFiles(root string) ([]File, error) {
	files := make([]File, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, File{Path: filepath.ToSlash(rel), IsDir: entry.IsDir(), Size: info.Size()})
		if len(files) >= maxFiles {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func cleanRelative(path string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(path))
	if strings.TrimSpace(path) == "" || clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("invalid workspace path: %s", path)
	}
	return filepath.ToSlash(clean), nil
}

func ensureContained(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("workspace path escapes protected root")
	}
	return nil
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
