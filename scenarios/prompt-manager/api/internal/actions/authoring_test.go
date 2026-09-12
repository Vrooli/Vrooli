package actions

import (
	"context"
	"testing"

	"prompt-manager/internal/store"
)

type fakeSemanticSearcher struct {
	hits  []SemanticActionHit
	query string
	err   error
}

func (f *fakeSemanticSearcher) SearchSimilarActions(ctx context.Context, query string, limit int) ([]SemanticActionHit, error) {
	f.query = query
	return f.hits, f.err
}

func commandResolver(owner CommandOwner, effect CommandEffect, permissions []string) stubResolver {
	return stubResolver{resolution: CommandResolution{
		Certainty:   CertaintyCommand,
		Owner:       owner,
		Target:      "x",
		Effect:      effect,
		Permissions: permissions,
		RunSurfaces: []string{"action"},
		Message:     "ok",
	}}
}

func TestInferActionFromCommand(t *testing.T) {
	tests := []struct {
		name        string
		argv        []string
		id          string
		resolver    ControlledCommandResolver
		wantOwnerID string
		wantID      string
		wantInputs  []string
		wantUnval   bool
		wantFSWrite bool
	}{
		{
			name:        "scenario command with placeholders and manifest permissions",
			argv:        []string{"browser-automation-studio", "capture", "{{url}}", "--out", "{{out}}"},
			resolver:    commandResolver(CommandOwner{Type: "scenario", ID: "browser-automation-studio"}, EffectWrite, []string{"filesystem:write", "network:localhost"}),
			wantOwnerID: "browser-automation-studio",
			wantID:      "browser-automation-studio.capture",
			wantInputs:  []string{"out", "url"},
			wantFSWrite: true,
		},
		{
			name:        "explicit id overrides derivation",
			argv:        []string{"vrooli", "scenario", "status", "{{scenario}}"},
			id:          "scenario.status.custom",
			resolver:    commandResolver(CommandOwner{Type: "project", ID: "vrooli"}, EffectRead, []string{"filesystem:read", "process:start"}),
			wantOwnerID: "vrooli",
			wantID:      "scenario.status.custom",
			wantInputs:  []string{"scenario"},
		},
		{
			name:        "owner-only command yields unvalidated note and empty permissions",
			argv:        []string{"some-scenario", "do-thing", "{{arg}}"},
			resolver:    stubResolver{resolution: CommandResolution{Certainty: CertaintyOwnerOnly, Owner: CommandOwner{Type: "scenario", ID: "some-scenario"}, Message: "owner-only"}},
			wantOwnerID: "some-scenario",
			wantID:      "some-scenario.do-thing",
			wantInputs:  []string{"arg"},
			wantUnval:   true,
		},
		{
			name:        "snake_case placeholders are inferred",
			argv:        []string{"test-genie", "provider-contract", "check", "{{phase_or_provider}}", "{{scenario}}"},
			resolver:    commandResolver(CommandOwner{Type: "scenario", ID: "test-genie"}, EffectRead, []string{"process:start"}),
			wantOwnerID: "test-genie",
			wantID:      "test-genie.provider-contract",
			wantInputs:  []string{"phase_or_provider", "scenario"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(newFakeActionStore(), tt.resolver)
			action, notes := service.InferActionFromCommand(context.Background(), tt.argv, "Test Action", "does a thing", tt.id)

			if action.Owner.ID != tt.wantOwnerID {
				t.Fatalf("owner id = %q, want %q", action.Owner.ID, tt.wantOwnerID)
			}
			if action.ID != tt.wantID {
				t.Fatalf("id = %q, want %q", action.ID, tt.wantID)
			}
			if action.Status != store.StatusActive {
				t.Fatalf("status = %q, want active", action.Status)
			}
			for _, name := range tt.wantInputs {
				if _, ok := action.Inputs[name]; !ok {
					t.Fatalf("expected inferred input %q; inputs=%#v", name, action.Inputs)
				}
			}
			if action.Permissions.FilesystemWrite != tt.wantFSWrite {
				t.Fatalf("filesystemWrite = %v, want %v", action.Permissions.FilesystemWrite, tt.wantFSWrite)
			}
			if tt.wantUnval {
				if action.Permissions != (store.ActionPermissions{}) {
					t.Fatalf("owner-only should yield empty permissions, got %#v", action.Permissions)
				}
				if !hasNoteForField(notes, "owner") {
					t.Fatalf("expected an owner inference note for the unvalidated case")
				}
			}
			if len(notes) == 0 {
				t.Fatalf("expected inference notes")
			}
		})
	}
}

func TestInferActionFromCommandValidates(t *testing.T) {
	// A fully inferred scenario command should validate cleanly: permissions
	// inferred from the resolver must satisfy the alignment check.
	resolver := commandResolver(CommandOwner{Type: "scenario", ID: "browser-automation-studio"}, EffectWrite, []string{"filesystem:write", "network:localhost"})
	service := NewService(newFakeActionStore(), resolver)
	action, _ := service.InferActionFromCommand(context.Background(), []string{"browser-automation-studio", "capture", "{{url}}", "--out", "{{out}}"}, "Capture", "capture a page", "")
	validation := service.Validate(context.Background(), action)
	if !validation.Valid {
		t.Fatalf("inferred action should validate; checks=%#v", validation.Checks)
	}
}

func TestFindSimilarActionsStructural(t *testing.T) {
	bas1 := validAction(func(a *store.Action) {
		a.ID = "bas.screenshot"
		a.Command.Argv = []string{"browser-automation-studio", "capture", "{{url}}", "--mode", "screenshot"}
	})
	bas2 := validAction(func(a *store.Action) {
		a.ID = "bas.audit"
		a.Command.Argv = []string{"browser-automation-studio", "capture", "{{url}}", "--mode", "audit"}
	})
	unrelated := validAction(func(a *store.Action) {
		a.ID = "scenario.status.show"
		a.Command.Argv = []string{"vrooli", "scenario", "status", "{{scenario}}"}
	})
	service := NewService(newFakeActionStore(bas1, bas2, unrelated), stubResolver{})

	candidate := &store.Action{
		ID:      "bas.pdf",
		Command: store.ActionCommand{Argv: []string{"browser-automation-studio", "capture", "{{url}}", "--mode", "pdf"}},
	}
	matches := service.FindSimilarActions(context.Background(), candidate)
	got := map[string]string{}
	for _, m := range matches {
		got[m.ID] = m.Reason
	}
	if got["bas.screenshot"] != "same-command" || got["bas.audit"] != "same-command" {
		t.Fatalf("expected both bas.* capture variants matched structurally, got %#v", got)
	}
	if _, ok := got["scenario.status.show"]; ok {
		t.Fatalf("unrelated action should not match; got %#v", got)
	}
}

func TestFindSimilarActionsSemanticSeam(t *testing.T) {
	searcher := &fakeSemanticSearcher{hits: []SemanticActionHit{
		{ID: "team.swarm.work.list", Name: "List Team Work", Score: 0.82},
		{ID: "self", Name: "self", Score: 0.99}, // must be excluded
	}}
	service := NewService(newFakeActionStore(), stubResolver{})
	service.SetSemanticSearcher(searcher)

	candidate := &store.Action{ID: "self", Name: "List work", Description: "show open work", Command: store.ActionCommand{Argv: []string{"swarm-manager", "backlog", "list", "--json"}}}
	matches := service.FindSimilarActions(context.Background(), candidate)
	if len(matches) != 1 || matches[0].ID != "team.swarm.work.list" || matches[0].Reason != "semantic" {
		t.Fatalf("expected one semantic match excluding self, got %#v", matches)
	}
	if searcher.query == "" {
		t.Fatalf("expected the searcher to be queried with name+description")
	}
}

func TestPreviewCreateWritesNothingAndFlagsHardFailure(t *testing.T) {
	fakeStore := newFakeActionStore()
	// Resolver returns CertaintyNone -> command_ownership hard failure.
	service := NewService(fakeStore, stubResolver{resolution: CommandResolution{Certainty: CertaintyNone, Message: "not controlled"}})

	preview, err := service.PreviewCreate(context.Background(), DraftActionInput{
		Name: "Bad",
		Argv: []string{"not-a-real-binary", "do", "{{x}}"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Validation.Valid {
		t.Fatalf("expected invalid preview for uncontrolled command")
	}
	// An uncontrolled command cannot resolve an owner, so it is a hard failure
	// (schema rejects the empty owner before command_ownership is reached).
	if !hasCheckWithStatus(preview.Validation, "schema", CheckFailed) && !hasCheckWithStatus(preview.Validation, "command_ownership", CheckFailed) {
		t.Fatalf("expected a hard validation failure; checks=%#v", preview.Validation.Checks)
	}
	// Nothing persisted.
	if list, _ := fakeStore.List(context.Background()); len(list) != 0 {
		t.Fatalf("preview must not write; store has %d actions", len(list))
	}
}

func TestPreviewCreateOwnerOnlyNotHardFailure(t *testing.T) {
	service := NewService(newFakeActionStore(), stubResolver{resolution: CommandResolution{
		Certainty: CertaintyOwnerOnly,
		Owner:     CommandOwner{Type: "scenario", ID: "some-scenario"},
		Message:   "owner-only",
	}})
	preview, err := service.PreviewCreate(context.Background(), DraftActionInput{
		Name: "Thing",
		Argv: []string{"some-scenario", "do-thing", "{{arg}}"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Validation.Valid {
		t.Fatalf("owner-only should be valid (unvalidated), got checks=%#v", preview.Validation.Checks)
	}
	if !preview.Validation.Unvalidated {
		t.Fatalf("expected unvalidated flag for owner-only command")
	}
}

func TestPreviewCreateAppliesInputOverrides(t *testing.T) {
	service := NewService(newFakeActionStore(), commandResolver(CommandOwner{Type: "scenario", ID: "browser-automation-studio"}, EffectWrite, []string{"filesystem:write"}))
	optional := false
	preview, err := service.PreviewCreate(context.Background(), DraftActionInput{
		Name: "Capture",
		Argv: []string{"browser-automation-studio", "capture", "{{url}}"},
		Inputs: []InputOverride{
			{Name: "url", Type: "string", Required: &optional},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := preview.Rendered.Inputs["url"]
	if got.Required {
		t.Fatalf("override should have made url optional, got %#v", got)
	}
}

func TestCreateDefaultsToActiveLocalPack(t *testing.T) {
	fakeStore := newFakeActionStore()
	service := NewService(fakeStore, runnableResolver())
	action := validAction(func(a *store.Action) { a.ID = "tmp.created" })
	created, validation, err := service.Create(context.Background(), "", action)
	if err != nil {
		t.Fatal(err)
	}
	if !validation.Valid {
		t.Fatalf("expected valid create; checks=%#v", validation.Checks)
	}
	// "drafts" is inactive; the default must land in the active "local" pack so
	// the action is discoverable.
	if created.Pack != "local" {
		t.Fatalf("default pack = %q, want local", created.Pack)
	}
}

func hasNoteForField(notes []InferenceNote, field string) bool {
	for _, note := range notes {
		if note.Field == field {
			return true
		}
	}
	return false
}
