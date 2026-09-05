package agentharness

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestHookBrokerReconcileIsIdempotentAndPreservesUnmanaged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	initial := map[string]any{"other": true, "hooks": map[string]any{"Stop": []any{map[string]any{"matcher": "Edit", "hooks": []any{map[string]any{"type": "command", "command": "user-hook"}}}}}}
	data, _ := json.Marshal(initial)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	broker := NewHookBroker()
	target := HookTarget{Agent: "claude-code", Path: path}
	registration := HookRegistration{Event: "Stop", ID: "web-console-tts", Hook: map[string]any{"type": "command", "command": "vrooli-hook"}}
	first, err := broker.Reconcile(target, registration)
	if err != nil || first.Status != "applied" {
		t.Fatalf("first reconcile: %+v, %v", first, err)
	}
	second, err := broker.Reconcile(target, registration)
	if err != nil || second.Status != "unchanged" {
		t.Fatalf("second reconcile: %+v, %v", second, err)
	}
	entries, err := broker.List([]HookTarget{target})
	if err != nil || len(entries) != 1 || entries[0].ID != registration.ID {
		t.Fatalf("managed entries: %+v, %v", entries, err)
	}
	raw, _ := os.ReadFile(path)
	if !json.Valid(raw) || string(raw) == "" || !containsJSON(raw, "user-hook") || !containsJSON(raw, "other") {
		t.Fatalf("unmanaged document was not preserved: %s", raw)
	}
}

func TestHookBrokerConcurrentReconcilesLoseNoRegistration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	target := HookTarget{Agent: "codex", Path: path}
	broker := NewHookBroker()
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := broker.Reconcile(target, HookRegistration{Event: "PreToolUse", ID: "hook-" + string(rune('a'+i)), Hook: map[string]any{"type": "command", "command": "hook"}})
			if err != nil {
				t.Errorf("reconcile %d: %v", i, err)
			}
		}()
	}
	wg.Wait()
	entries, err := broker.List([]HookTarget{target})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 24 {
		t.Fatalf("concurrent reconciles lost registrations: got %d, want 24", len(entries))
	}
}

func TestHookBrokerMigratesLegacyMarkers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	document := map[string]any{"hooks": map[string]any{"PreToolUse": []any{map[string]any{"hooks": []any{map[string]any{"managedBy": "vrooli-memory-capture", "command": "memory"}, map[string]any{"command": "vrooli-policy-runner hook"}}}}}}
	data, _ := json.Marshal(document)
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	count, err := NewHookBroker().Migrate(HookTarget{Agent: "claude-code", Path: path})
	if err != nil || count != 2 {
		t.Fatalf("migrate: count=%d err=%v", count, err)
	}
	entries, err := NewHookBroker().List([]HookTarget{{Agent: "claude-code", Path: path}})
	if err != nil || len(entries) != 2 {
		t.Fatalf("migrated entries: %+v, %v", entries, err)
	}
}

func TestHookBrokerReconcilesGroupOwnedHooks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	broker := NewHookBroker()
	target := HookTarget{Agent: "claude-code", Path: path}
	registration := HookRegistration{
		Event: "PreToolUse", ID: "vrooli-policy-runner", Matcher: "Bash",
		Group: map[string]any{"patterns": []any{"Bash(rm -rf *)"}},
		Hook:  map[string]any{"type": "command", "command": "vrooli-policy-runner hook"},
	}
	first, err := broker.Reconcile(target, registration)
	if err != nil || first.Status != "applied" {
		t.Fatalf("first group reconcile: %+v, %v", first, err)
	}
	second, err := broker.Reconcile(target, registration)
	if err != nil || second.Status != "unchanged" {
		t.Fatalf("second group reconcile: %+v, %v", second, err)
	}
	entries, err := broker.List([]HookTarget{target})
	if err != nil || len(entries) != 1 || entries[0].ID != registration.ID {
		t.Fatalf("group list: %+v, %v", entries, err)
	}
	removed, err := broker.Remove(target, registration.Event, registration.ID)
	if err != nil || removed.Status != "removed" {
		t.Fatalf("group remove: %+v, %v", removed, err)
	}
	entries, err = broker.List([]HookTarget{target})
	if err != nil || len(entries) != 0 {
		t.Fatalf("group list after remove: %+v, %v", entries, err)
	}
}

func containsJSON(data []byte, value string) bool {
	return bytes.Contains(data, []byte(value))
}
