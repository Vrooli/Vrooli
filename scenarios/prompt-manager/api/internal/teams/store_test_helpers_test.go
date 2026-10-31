package teams

import (
	"testing"

	"prompt-manager/internal/paths"
	"prompt-manager/internal/store"
)

func newFileStore(t testing.TB, roots paths.Roots) *store.FileStore {
	t.Helper()
	fileStore, err := store.NewFileStore(roots)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	return fileStore
}
