package templates

import (
	"context"
	"fmt"
	"path/filepath"
	"prompt-manager/store"
	"sort"
	"strings"
)

// Store loads agent file templates from the prompt-manager store directory.
type Store struct {
	storeDir string
}

// NewStore creates a new templates store backed by the file system.
func NewStore(storeDir string) *Store {
	return &Store{storeDir: storeDir}
}

func (s *Store) agentFilesDir() string {
	return filepath.Join(s.storeDir, "templates", "agent-files")
}

// ListAgentFileTemplates returns all agent file templates with content.
func (s *Store) ListAgentFileTemplates(ctx context.Context) ([]AgentFileTemplate, error) {
	_ = ctx
	root := s.agentFilesDir()
	dirs, err := store.ListDirectories(root)
	if err != nil {
		return nil, fmt.Errorf("listing agent file templates: %w", err)
	}

	templates := make([]AgentFileTemplate, 0, len(dirs))
	for _, dir := range dirs {
		metaPath := filepath.Join(root, dir, "template.json")
		meta, err := store.LoadJSON[AgentFileTemplateMeta](metaPath)
		if err != nil {
			continue
		}

		if meta.ID == "" {
			meta.ID = dir
		}
		if meta.FileName == "" {
			continue
		}

		entry := meta.Entry
		if entry == "" {
			entry = "TEMPLATE.md"
		}

		content, err := store.ReadContent(filepath.Join(root, dir, entry))
		if err != nil {
			continue
		}

		templates = append(templates, AgentFileTemplate{
			ID:          meta.ID,
			Name:        meta.Name,
			Description: meta.Description,
			FileName:    meta.FileName,
			Content:     content,
		})
	}

	sort.SliceStable(templates, func(i, j int) bool {
		left := strings.ToLower(templates[i].FileName)
		right := strings.ToLower(templates[j].FileName)
		if left == right {
			return strings.ToLower(templates[i].Name) < strings.ToLower(templates[j].Name)
		}
		return left < right
	})

	return templates, nil
}
