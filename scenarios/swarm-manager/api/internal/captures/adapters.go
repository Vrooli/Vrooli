package captures

import (
	"fmt"
	"log/slog"
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

func (a *backlogItemCreatorAdapter) SaveItem(draft BacklogItemDraft) error {
	bk := backlog.BacklogKind(draft.Kind)
	itemDir := a.store.ItemDir(bk, draft.Name)
	if err := os.MkdirAll(filepath.Dir(itemDir), 0o750); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.Mkdir(itemDir, 0o750); err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("already exists")
		}
		return fmt.Errorf("create item dir: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := backlog.BacklogItem{
		Name:        draft.Name,
		Title:       draft.Title,
		Description: draft.Description,
		Status:      backlog.StatusBacklog,
		Priority:    draft.Priority,
		Tags:        draft.Tags,
		Created:     now,
		Updated:     now,
		Kind:        bk,
		SpawnedFrom: draft.SpawnedFrom,
	}
	if err := a.store.SaveItem(item); err != nil {
		if rmErr := os.RemoveAll(itemDir); rmErr != nil {
			slog.Debug("captures: rollback item dir failed", "err", rmErr, "dir", itemDir)
		}
		return fmt.Errorf("save item: %w", err)
	}
	return nil
}
