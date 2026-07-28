package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"prompt-manager/internal/paths"
)

func setupActionStore(t *testing.T) (*FileActionStore, string) {
	t.Helper()
	storeDir := t.TempDir()
	for _, pack := range []string{"core", "local", "drafts"} {
		if err := os.MkdirAll(filepath.Join(storeDir, "actions", "packs", pack), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	order := &PackOrder{
		ActivePacks:   []string{"local", "core"},
		InactivePacks: []string{"drafts"},
	}
	if err := SaveJSON(filepath.Join(storeDir, "actions", "_pack-order.json"), order); err != nil {
		t.Fatal(err)
	}
	return NewFileActionStore(storeDir), storeDir
}

func testAction(id string) *Action {
	return &Action{
		ID:          id,
		Name:        "List Decisions",
		Description: "List pending decisions for a team.",
		Status:      StatusDraft,
		Owner: ActionOwner{
			Type: "scenario",
			ID:   "prompt-manager",
		},
		Command: ActionCommand{
			Argv: []string{"prompt-manager", "team", "decisions", "list", "{{team}}"},
		},
		Inputs: map[string]ActionInput{
			"team": {Type: "team", Required: true},
		},
		Outputs: map[string]ActionOutput{
			"decisions": {Type: "json"},
		},
		Examples: []ActionExample{{
			Description: "List the director swarm decisions.",
			Input:       map[string]any{"team": "director-swarm"},
		}},
	}
}

func TestActionStore_CreateGetUpdateArchiveDelete(t *testing.T) {
	store, _ := setupActionStore(t)
	ctx := context.Background()

	if err := store.Create(ctx, "local", testAction("team.decisions.list")); err != nil {
		t.Fatalf("create action: %v", err)
	}

	got, err := store.Get(ctx, "team.decisions.list")
	if err != nil {
		t.Fatalf("get action: %v", err)
	}
	if got.Kind != KindAction {
		t.Errorf("Kind = %q, want %q", got.Kind, KindAction)
	}
	if got.SchemaVersion != CurrentSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
	}
	if got.Pack != "local" {
		t.Errorf("Pack = %q, want local", got.Pack)
	}
	if got.Revision != 1 || got.CreatedAt == "" || got.UpdatedAt == "" {
		t.Errorf("expected initial timestamps/revision, got revision=%d created=%q updated=%q", got.Revision, got.CreatedAt, got.UpdatedAt)
	}

	if err := store.Update(ctx, "team.decisions.list", &Action{
		Name:   "List Team Decisions",
		Status: StatusActive,
		Tags:   []string{"teams", "decisions"},
	}); err != nil {
		t.Fatalf("update action: %v", err)
	}
	got, err = store.Get(ctx, "team.decisions.list")
	if err != nil {
		t.Fatalf("get updated action: %v", err)
	}
	if got.Name != "List Team Decisions" {
		t.Errorf("Name = %q, want updated name", got.Name)
	}
	if got.Status != StatusActive {
		t.Errorf("Status = %q, want active", got.Status)
	}
	if got.Revision != 2 {
		t.Errorf("Revision = %d, want 2", got.Revision)
	}

	if err := store.Archive(ctx, "team.decisions.list"); err != nil {
		t.Fatalf("archive action: %v", err)
	}
	got, err = store.Get(ctx, "team.decisions.list")
	if err != nil {
		t.Fatalf("get archived action: %v", err)
	}
	if got.Status != StatusArchived {
		t.Errorf("Status = %q, want archived", got.Status)
	}

	if err := store.Delete(ctx, "team.decisions.list"); err != nil {
		t.Fatalf("delete action: %v", err)
	}
	if _, err := store.Get(ctx, "team.decisions.list"); err == nil {
		t.Fatal("expected deleted action to be missing")
	}
}

func TestActionStore_AcceptsSnakeCasePlaceholders(t *testing.T) {
	store, _ := setupActionStore(t)
	ctx := context.Background()
	action := testAction("test-genie.provider-contract")
	action.Command.Argv = []string{"test-genie", "provider-contract", "check", "{{phase_or_provider}}", "{{scenario}}"}
	action.Inputs = map[string]ActionInput{
		"phase_or_provider": {Type: "string", Required: true},
		"scenario":          {Type: "scenario", Required: true},
	}
	action.Outputs = nil

	if err := store.Create(ctx, "local", action); err != nil {
		t.Fatalf("Create rejected snake_case placeholder: %v", err)
	}
}

func TestActionStore_DottedIDValidation(t *testing.T) {
	valid := []string{
		"scenario.ui.screenshot",
		"team-decisions.list",
		"skill.health-audit",
	}
	for _, id := range valid {
		if !IsValidActionID(id) {
			t.Errorf("IsValidActionID(%q) = false, want true", id)
		}
	}

	invalid := []string{
		"",
		"Scenario.ui",
		"scenario..ui",
		"scenario_ui",
		"scenario/ui",
		"scenario.",
		"-scenario",
		strings.Repeat("a", maxActionIDLength+1),
	}
	for _, id := range invalid {
		if IsValidActionID(id) {
			t.Errorf("IsValidActionID(%q) = true, want false", id)
		}
	}
}

func TestActionStore_PackPrecedence(t *testing.T) {
	store, dir := setupActionStore(t)
	ctx := context.Background()

	core := testAction("scenario.logs.tail")
	core.Name = "Core Logs"
	local := testAction("scenario.logs.tail")
	local.Name = "Local Logs"
	writeActionFixture(t, dir, "core", core)
	writeActionFixture(t, dir, "local", local)

	got, err := store.Get(ctx, "scenario.logs.tail")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Local Logs" || got.Pack != "local" {
		t.Errorf("got %q from %q, want Local Logs from local", got.Name, got.Pack)
	}

	actions, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 {
		t.Fatalf("List returned %d actions, want 1", len(actions))
	}
}

func writeActionFixture(t *testing.T, storeDir, pack string, action *Action) {
	t.Helper()
	action.Kind = KindAction
	action.SchemaVersion = CurrentSchemaVersion
	action.Timestamps = NewTimestamps()
	if action.Status == "" {
		action.Status = StatusDraft
	}
	path := filepath.Join(storeDir, "actions", "packs", pack, action.ID, "action.json")
	if err := SaveJSON(path, action); err != nil {
		t.Fatal(err)
	}
}

func TestActionStore_MalformedAndInvalidFilesAreSkipped(t *testing.T) {
	store, dir := setupActionStore(t)
	ctx := context.Background()

	if err := store.Create(ctx, "core", testAction("scenario.test.run")); err != nil {
		t.Fatal(err)
	}
	badDir := filepath.Join(dir, "actions", "packs", "local", "scenario.test.run")
	if err := os.MkdirAll(badDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "action.json"), []byte(`{"kind":"action"`), 0o644); err != nil {
		t.Fatal(err)
	}
	invalid := testAction("scenario.bad")
	invalid.Command.Argv = []string{"bash", "-lc", "echo nope"}
	invalid.Kind = KindAction
	invalid.SchemaVersion = CurrentSchemaVersion
	invalid.Timestamps = NewTimestamps()
	if err := SaveJSON(filepath.Join(dir, "actions", "packs", "local", "scenario.bad", "action.json"), invalid); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(ctx, "scenario.test.run")
	if err != nil {
		t.Fatal(err)
	}
	if got.Pack != "core" {
		t.Errorf("got pack %q, want core fallback after malformed local file", got.Pack)
	}

	actions, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].ID != "scenario.test.run" {
		t.Fatalf("List = %#v, want only valid core action", actions)
	}
}

func TestActionStore_RejectsInvalidCreate(t *testing.T) {
	store, _ := setupActionStore(t)
	ctx := context.Background()

	cases := []struct {
		name   string
		action *Action
	}{
		{name: "bad id", action: testAction("scenario/ui")},
		{name: "empty argv", action: func() *Action {
			a := testAction("scenario.empty")
			a.Command.Argv = nil
			return a
		}()},
		{name: "raw external executable", action: func() *Action {
			a := testAction("scenario.raw")
			a.Command.Argv = []string{"git", "status"}
			return a
		}()},
		{name: "dynamic executable", action: func() *Action {
			a := testAction("scenario.dynamic")
			a.Command.Argv = []string{"{{tool}}", "team", "list"}
			a.Inputs["tool"] = ActionInput{Type: "string", Required: true}
			return a
		}()},
		{name: "shell syntax", action: func() *Action {
			a := testAction("scenario.shell")
			a.Command.Argv = []string{"prompt-manager", "skill", "list", "&&", "echo"}
			return a
		}()},
		{name: "undeclared placeholder", action: func() *Action {
			a := testAction("scenario.placeholder")
			a.Command.Argv = []string{"prompt-manager", "team", "show", "{{missing}}"}
			return a
		}()},
		{name: "invalid execution output mode", action: func() *Action {
			a := testAction("scenario.outputmode")
			a.Execution = &ActionExecution{OutputMode: "yaml"}
			return a
		}()},
		{name: "invalid validation mode", action: func() *Action {
			a := testAction("scenario.validationmode")
			a.Validation = &ActionValidation{Mode: "custom"}
			return a
		}()},
		{name: "unsafe validation argv", action: func() *Action {
			a := testAction("scenario.validationargv")
			a.Validation = &ActionValidation{Argv: []string{"bash", "-c", "prompt-manager skill list"}}
			return a
		}()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := store.Create(ctx, "local", tc.action); err == nil {
				t.Fatal("Create succeeded, want validation error")
			}
		})
	}
}

func TestActionStore_DuplicateCreateRejected(t *testing.T) {
	store, _ := setupActionStore(t)
	ctx := context.Background()

	if err := store.Create(ctx, "local", testAction("skill.health.audit")); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "core", testAction("skill.health.audit")); err == nil {
		t.Fatal("duplicate Action ID across active packs should fail")
	}
}

func TestActionStore_RunHistoryIsBounded(t *testing.T) {
	store, storeDir := setupActionStore(t)
	ctx := context.Background()
	action := testAction("team.decisions.list")
	if err := store.Create(ctx, "local", action); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < maxActionRunHistoryEntries+5; i++ {
		if err := store.AppendRunHistory(ctx, action.ID, ActionRunHistoryEntry{
			ActionID:        action.ID,
			StartedAt:       time.Now().UTC(),
			FinishedAt:      time.Now().UTC(),
			Status:          "completed",
			Stdout:          strings.Repeat("x", 12),
			ValidationValid: true,
		}); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(storeDir, "actions", "packs", "local", action.ID, "runs.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != maxActionRunHistoryEntries {
		t.Fatalf("history entries = %d, want %d", len(lines), maxActionRunHistoryEntries)
	}
}

func TestNewFileStoreInitializesActionDirectories(t *testing.T) {
	roots := paths.RootsForTest(t)
	fs, err := NewFileStore(roots)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	if fs.Actions() == nil {
		t.Fatal("Actions store should be wired")
	}
	for _, path := range []string{
		filepath.Join(roots.Config, "actions", "packs", "core"),
		filepath.Join(roots.Config, "actions", "packs", "local"),
		filepath.Join(roots.Config, "actions", "packs", "drafts"),
		filepath.Join(roots.Config, "actions", "_pack-order.json"),
	} {
		if !FileExists(path) {
			t.Fatalf("expected initialized action path: %s", path)
		}
	}
}
