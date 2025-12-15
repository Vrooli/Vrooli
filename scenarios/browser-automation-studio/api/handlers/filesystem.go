package handlers

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/browser-automation-studio/constants"
)

type ListDirectoriesRequest struct {
	Path string `json:"path,omitempty"`
}

type DirectoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type ListDirectoriesResponse struct {
	Path    string           `json:"path"`
	Parent  string           `json:"parent,omitempty"`
	Entries []DirectoryEntry `json:"entries"`
}

// ListDirectories handles POST /api/v1/fs/list-directories
func (h *Handler) ListDirectories(w http.ResponseWriter, r *http.Request) {
	var req ListDirectoriesRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest)
		return
	}

	target := strings.TrimSpace(req.Path)
	if target == "" {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			target = home
		} else {
			target = string(filepath.Separator)
		}
	}

	absPath, err := filepath.Abs(target)
	if err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "invalid path",
		}))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), constants.DefaultRequestTimeout)
	defer cancel()

	_ = ctx
	info, statErr := os.Stat(absPath)
	if statErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "path does not exist",
		}))
		return
	}
	if !info.IsDir() {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": "path is not a directory",
		}))
		return
	}

	items, readErr := os.ReadDir(absPath)
	if readErr != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{
			"field": "path",
			"error": readErr.Error(),
		}))
		return
	}

	entries := make([]DirectoryEntry, 0, len(items))
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		name := item.Name()
		if strings.TrimSpace(name) == "" || name == "." || name == ".." {
			continue
		}
		entries = append(entries, DirectoryEntry{
			Name: name,
			Path: filepath.Join(absPath, name),
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	parent := filepath.Dir(absPath)
	if parent == absPath {
		parent = ""
	}

	h.respondSuccess(w, http.StatusOK, ListDirectoriesResponse{
		Path:    absPath,
		Parent:  parent,
		Entries: entries,
	})
}
