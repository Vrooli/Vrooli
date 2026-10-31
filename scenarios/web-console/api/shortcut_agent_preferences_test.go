package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestAgentPreferencesPreserveConcurrentShortcutEdits(t *testing.T) {
	for _, kind := range []string{"memory", "sqlite"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			var store ShortcutStore
			if kind == "memory" {
				store = NewShortcutProfileStore()
			} else {
				store = NewSQLShortcutStore(setupTestDB(t))
			}
			before, err := store.AgentPreferences(ctx)
			if err != nil {
				t.Fatal(err)
			}
			// An older editor can change the profile after preferences were read.
			edited := []ShortcutEntry{{Label: "Custom", Command: "echo preserved"}}
			store.Upsert(ctx, before.ProfileID, "service", "Edited", edited)
			saved, err := store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: before.ProfileID, ProfileInstance: before.ProfileInstance, OrderedAgentIDs: []string{"codex", "claude"}, HiddenAgentIDs: []string{"claude"}}, before.Revision)
			if err != nil {
				t.Fatal(err)
			}
			if saved.Revision != before.Revision+1 {
				t.Fatalf("revision = %d", saved.Revision)
			}
			if got := store.Effective(ctx).Shortcuts; len(got) != 1 || got[0].Command != "echo preserved" {
				t.Fatalf("shortcut edit lost: %+v", got)
			}
			store.Upsert(ctx, before.ProfileID, "service", "Edited again", edited)
			got, err := store.AgentPreferences(ctx)
			if err != nil || !reflect.DeepEqual(got.HiddenAgentIDs, []string{"claude"}) || !reflect.DeepEqual(got.OrderedAgentIDs, []string{"codex", "claude"}) {
				t.Fatalf("preferences lost by legacy edit: %+v %v", got, err)
			}
			_, err = store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: before.ProfileID, ProfileInstance: before.ProfileInstance}, before.Revision)
			if !errors.Is(err, ErrAgentPreferencesConflict) {
				t.Fatalf("stale revision: %v", err)
			}
		})
	}
}

func TestAgentPreferencesProfileSwitchAndDeletion(t *testing.T) {
	for _, kind := range []string{"memory", "sqlite"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			var store ShortcutStore
			if kind == "memory" {
				store = NewShortcutProfileStore()
			} else {
				store = NewSQLShortcutStore(setupTestDB(t))
			}
			initial, err := store.AgentPreferences(ctx)
			if err != nil {
				t.Fatal(err)
			}
			store.Upsert(ctx, "higher", "workspace", "Higher", []ShortcutEntry{{Label: "Command", Command: "echo hi"}})
			if _, err = store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: initial.ProfileID, ProfileInstance: initial.ProfileInstance}, initial.Revision); !errors.Is(err, ErrAgentPreferencesConflict) {
				t.Fatalf("profile switch must conflict: %v", err)
			}
			current, err := store.AgentPreferences(ctx)
			if err != nil || current.ProfileID != "higher" {
				t.Fatalf("effective profile: %+v %v", current, err)
			}
			if _, err = store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: "higher", ProfileInstance: current.ProfileInstance, HiddenAgentIDs: []string{"codex"}}, current.Revision); err != nil {
				t.Fatal(err)
			}
			if !store.Delete(ctx, "higher") {
				t.Fatal("delete failed")
			}
			store.Upsert(ctx, "higher", "workspace", "New", []ShortcutEntry{{Label: "Command", Command: "echo hi"}})
			// Even if creation timestamps collide, the deletion tombstone
			// prevents an old version from mutating the recreated profile.
			switch s := store.(type) {
			case *ShortcutProfileStore:
				s.profiles["higher"].CreatedAt = current.ProfileInstance
			case *SQLShortcutStore:
				if _, err := s.db.ExecContext(ctx, `UPDATE shortcut_profiles SET created_at = ? WHERE id = 'higher'`, current.ProfileInstance); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: "higher", ProfileInstance: current.ProfileInstance}, 0); !errors.Is(err, ErrAgentPreferencesConflict) {
				t.Fatalf("recreated profile must conflict: %v", err)
			}
			current, err = store.AgentPreferences(ctx)
			if err != nil || current.Revision != 2 || len(current.HiddenAgentIDs) != 0 {
				t.Fatalf("deleted profile retained preferences: %+v %v", current, err)
			}
		})
	}
}

func TestAgentPreferencesValidation(t *testing.T) {
	s := NewShortcutProfileStore()
	ctx := context.Background()
	initial, _ := s.AgentPreferences(ctx)
	for _, ids := range [][]string{{""}, {" codex"}, {"codex", "codex"}} {
		_, err := s.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: initial.ProfileID, ProfileInstance: initial.ProfileInstance, HiddenAgentIDs: ids}, 0)
		if !errors.Is(err, ErrAgentPreferencesInvalid) {
			t.Errorf("ids %q: %v", ids, err)
		}
	}
}

func TestAgentPreferencesFallbackAndPersistence(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	store := NewSQLShortcutStore(db)
	if !store.Delete(ctx, "default") {
		t.Fatal("delete default")
	}
	initial, err := store.AgentPreferences(ctx)
	if err != nil || initial.ProfileID != "" || initial.Revision != 0 {
		t.Fatalf("fallback: %+v %v", initial, err)
	}
	_, err = store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: "", HiddenAgentIDs: []string{"codex"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if store.Delete(ctx, "") {
		t.Fatal("the fallback is not a deletable profile")
	}
	reopened := NewSQLShortcutStore(db)
	got, err := reopened.AgentPreferences(ctx)
	if err != nil || got.ProfileID != "" || got.Revision != 1 || !reflect.DeepEqual(got.HiddenAgentIDs, []string{"codex"}) {
		t.Fatalf("persisted fallback: %+v %v", got, err)
	}
	reopened.Upsert(ctx, "default", "service", "Default", []ShortcutEntry{{Label: "custom", Command: "true"}})
	if _, err := reopened.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: ""}, got.Revision); !errors.Is(err, ErrAgentPreferencesConflict) {
		t.Fatalf("new profile must conflict: %v", err)
	}
}

func TestConcurrentAgentPreferenceWritesHaveOneWinner(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	db.SetMaxOpenConns(4)
	store := NewSQLShortcutStore(db)
	initial, err := store.AgentPreferences(ctx)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, id := range []string{"claude", "codex"} {
		go func(id string) {
			<-start
			_, err := store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: initial.ProfileID, ProfileInstance: initial.ProfileInstance, HiddenAgentIDs: []string{id}}, initial.Revision)
			results <- err
		}(id)
	}
	close(start)
	wins, conflicts := 0, 0
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			wins++
		case errors.Is(err, ErrAgentPreferencesConflict):
			conflicts++
		default:
			t.Fatalf("unexpected write error: %v", err)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("wins=%d conflicts=%d", wins, conflicts)
	}
	got, err := store.AgentPreferences(ctx)
	if err != nil || got.Revision != initial.Revision+1 || len(got.HiddenAgentIDs) != 1 {
		t.Fatalf("final preferences: %+v %v", got, err)
	}
}
