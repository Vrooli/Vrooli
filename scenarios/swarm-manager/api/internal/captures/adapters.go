package captures

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"swarm-manager/internal/backlog"
)

// backlogItemCreatorAdapter bridges backlog.Store to BacklogItemCreator.
type backlogItemCreatorAdapter struct {
	store backlog.Store
}

// NewBacklogItemCreatorAdapter creates a BacklogItemCreator backed by the given backlog.Store.
func NewBacklogItemCreatorAdapter(store backlog.Store) *backlogItemCreatorAdapter {
	return &backlogItemCreatorAdapter{store: store}
}

func (a *backlogItemCreatorAdapter) ItemDir(kind, name string) string {
	return a.store.ItemDir(backlog.BacklogKind(kind), name)
}

func (a *backlogItemCreatorAdapter) SaveItem(kind, name, title, description string, tags []string) error {
	bk := backlog.BacklogKind(kind)
	itemDir := a.store.ItemDir(bk, name)
	if err := os.MkdirAll(filepath.Dir(itemDir), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.Mkdir(itemDir, 0o755); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("already exists")
		}
		return fmt.Errorf("create item dir: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := backlog.BacklogItem{
		Name:        name,
		Title:       title,
		Description: description,
		Status:      backlog.StatusBacklog,
		Priority:    5,
		Tags:        tags,
		Created:     now,
		Updated:     now,
		Kind:        bk,
	}
	if err := a.store.SaveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		return fmt.Errorf("save item: %w", err)
	}
	return nil
}
