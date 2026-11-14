package autodrain

import "testing"

func TestStore_DefaultsOffThenPersists(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)

	// Absent file → OFF (the D4 default).
	if store.AutoDrainEnabled() {
		t.Fatal("expected auto-drain OFF by default")
	}
	st, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Enabled {
		t.Fatal("default state should be disabled")
	}

	// Enable and read back.
	if err := store.Save(State{Enabled: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !store.AutoDrainEnabled() {
		t.Fatal("expected auto-drain ON after save")
	}

	// A fresh store over the same root sees the persisted value.
	reloaded := NewStore(root)
	if !reloaded.AutoDrainEnabled() {
		t.Fatal("expected persisted ON to survive a new store")
	}

	// Disable again.
	if err := store.Save(State{Enabled: false}); err != nil {
		t.Fatalf("Save off: %v", err)
	}
	if store.AutoDrainEnabled() {
		t.Fatal("expected OFF after disabling")
	}
}
