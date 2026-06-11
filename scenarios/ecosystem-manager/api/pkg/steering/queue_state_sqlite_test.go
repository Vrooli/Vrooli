package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/internal/testdb"
)

func mustSaveQueueState(t *testing.T, repo QueueStateRepository, state *QueueState) {
	t.Helper()
	if err := repo.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestSQLiteQueueStateRepository_NilDB(t *testing.T) {
	repo := NewSQLiteQueueStateRepository(nil)
	state := NewQueueState("task-1", 2)

	if _, err := repo.Get("task-1"); err == nil {
		t.Fatal("Get() expected error for nil DB")
	}
	if err := repo.Save(state); err == nil {
		t.Fatal("Save() expected error for nil DB")
	}
	if err := repo.Delete("task-1"); err == nil {
		t.Fatal("Delete() expected error for nil DB")
	}
}

func TestSQLiteQueueStateRepository_RoundTrip(t *testing.T) {
	repo := NewSQLiteQueueStateRepository(testdb.NewSQLite(t, Schema()))

	mustSaveQueueState(t, repo, NewQueueState("task-1", 3))

	if err := repo.SetPosition("task-1", 2); err != nil {
		t.Fatalf("SetPosition() error = %v", err)
	}
	got, err := repo.Get("task-1")
	if err != nil || got == nil {
		t.Fatalf("Get() = (%v, %v)", got, err)
	}
	if got.CurrentIndex != 2 {
		t.Fatalf("CurrentIndex = %d, want 2", got.CurrentIndex)
	}

	if err := repo.ResetPosition("task-1"); err != nil {
		t.Fatalf("ResetPosition() error = %v", err)
	}
	if got, _ := repo.Get("task-1"); got.CurrentIndex != 0 {
		t.Fatalf("after reset CurrentIndex = %d, want 0", got.CurrentIndex)
	}

	if err := repo.Delete("task-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if got, _ := repo.Get("task-1"); got != nil {
		t.Fatal("Delete() did not remove state")
	}

	// SetPosition on an absent task surfaces a not-found error.
	if err := repo.SetPosition("ghost", 1); err == nil {
		t.Fatal("SetPosition() on missing task should error")
	}
}

func TestInMemoryQueueStateRepository_SaveGetCopySemantics(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	state := NewQueueState("task-1", 3)
	state.CurrentIndex = 1
	mustSaveQueueState(t, repo, state)

	got, err := repo.Get("task-1")
	if err != nil || got == nil {
		t.Fatalf("Get() = (%v, %v)", got, err)
	}
	if got.QueueLength != 3 || got.CurrentIndex != 1 {
		t.Fatalf("Get() = %#v", got)
	}

	got.CurrentIndex = 99
	again, _ := repo.Get("task-1")
	if again.CurrentIndex == 99 {
		t.Fatal("Get() should return a copy")
	}
}

func TestInMemoryQueueStateRepository_SaveUpdateDelete(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	first := NewQueueState("task-1", 1)
	mustSaveQueueState(t, repo, first)

	second := NewQueueState("task-1", 4)
	second.CurrentIndex = 2
	mustSaveQueueState(t, repo, second)

	got, _ := repo.Get("task-1")
	if got.QueueLength != 4 || got.CurrentIndex != 2 {
		t.Fatalf("updated state = %#v", got)
	}

	if err := repo.Delete("task-1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if got, _ := repo.Get("task-1"); got != nil {
		t.Fatal("Delete() did not remove state")
	}
}
