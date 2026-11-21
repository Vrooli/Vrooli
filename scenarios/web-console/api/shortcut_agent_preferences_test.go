package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
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

func TestAgentOrderModePersistsAndSurvivesALaunchRecord(t *testing.T) {
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
			if initial.OrderMode != AgentOrderCustom || len(initial.LastLaunchedAt) != 0 {
				t.Fatalf("defaults: %+v", initial)
			}
			write := AgentPreferences{ProfileID: initial.ProfileID, ProfileInstance: initial.ProfileInstance, OrderedAgentIDs: []string{"codex"}}
			write.OrderMode = AgentOrderRecent
			saved, err := store.UpdateAgentPreferences(ctx, write, initial.Revision)
			if err != nil || saved.OrderMode != AgentOrderRecent {
				t.Fatalf("order mode: %+v %v", saved, err)
			}
			// A launch record names no mode, and must not reset the operator's.
			write.OrderMode, write.LaunchedAgentID = "", "codex"
			before := time.Now().UTC().Truncate(time.Second)
			saved, err = store.UpdateAgentPreferences(ctx, write, saved.Revision)
			if err != nil {
				t.Fatal(err)
			}
			first, err := time.Parse(time.RFC3339, saved.LastLaunchedAt["codex"])
			if err != nil || first.Before(before) {
				t.Fatalf("launch time %q: %v", saved.LastLaunchedAt["codex"], err)
			}
			// Later writes accumulate launches instead of replacing the record.
			write.LaunchedAgentID = "claude"
			if saved, err = store.UpdateAgentPreferences(ctx, write, saved.Revision); err != nil {
				t.Fatal(err)
			}
			got, err := store.AgentPreferences(ctx)
			if err != nil || got.OrderMode != AgentOrderRecent || got.LaunchedAgentID != "" {
				t.Fatalf("reread: %+v %v", got, err)
			}
			if got.LastLaunchedAt["codex"] == "" || got.LastLaunchedAt["claude"] == "" || got.Revision != saved.Revision {
				t.Fatalf("launches lost: %+v", got)
			}
		})
	}
}

func TestAgentPreferencesRejectAnUnknownOrderModeOrLaunchedID(t *testing.T) {
	s := NewShortcutProfileStore()
	ctx := context.Background()
	initial, _ := s.AgentPreferences(ctx)
	base := AgentPreferences{ProfileID: initial.ProfileID, ProfileInstance: initial.ProfileInstance}
	for _, bad := range []AgentPreferences{{OrderMode: "alphabetical"}, {LaunchedAgentID: " codex"}} {
		bad.ProfileID, bad.ProfileInstance = base.ProfileID, base.ProfileInstance
		if _, err := s.UpdateAgentPreferences(ctx, bad, 0); !errors.Is(err, ErrAgentPreferencesInvalid) {
			t.Errorf("%+v: %v", bad, err)
		}
	}
}

func TestDeletingAProfileClearsItsOrderModeAndLaunches(t *testing.T) {
	ctx := context.Background()
	store := NewSQLShortcutStore(setupTestDB(t))
	store.Upsert(ctx, "higher", "workspace", "Higher", []ShortcutEntry{{Label: "Command", Command: "echo hi"}})
	current, err := store.AgentPreferences(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.UpdateAgentPreferences(ctx, AgentPreferences{ProfileID: "higher", ProfileInstance: current.ProfileInstance, OrderMode: AgentOrderRecent, LaunchedAgentID: "codex"}, current.Revision); err != nil {
		t.Fatal(err)
	}
	if !store.Delete(ctx, "higher") {
		t.Fatal("delete failed")
	}
	store.Upsert(ctx, "higher", "workspace", "New", []ShortcutEntry{{Label: "Command", Command: "echo hi"}})
	got, err := store.AgentPreferences(ctx)
	if err != nil || got.OrderMode != AgentOrderCustom || len(got.LastLaunchedAt) != 0 {
		t.Fatalf("deleted profile retained preferences: %+v %v", got, err)
	}
}

func TestExistingPreferenceRowsGainOrderModeAndLaunches(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	for _, statement := range []string{
		`DROP TABLE shortcut_agent_preferences`,
		`CREATE TABLE shortcut_agent_preferences (profile_id TEXT PRIMARY KEY, ordered_agent_ids TEXT NOT NULL DEFAULT '[]', hidden_agent_ids TEXT NOT NULL DEFAULT '[]', revision INTEGER NOT NULL DEFAULT 0)`,
		`INSERT INTO shortcut_agent_preferences (profile_id, ordered_agent_ids, hidden_agent_ids, revision) VALUES ('default', '["codex"]', '[]', 3)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := applyColumnMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	got, err := NewSQLShortcutStore(db).AgentPreferences(ctx)
	if err != nil || got.OrderMode != AgentOrderCustom || got.LastLaunchedAt == nil || got.Revision != 3 {
		t.Fatalf("migrated row: %+v %v", got, err)
	}
}
