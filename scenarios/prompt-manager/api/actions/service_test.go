package actions

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/store"
)

func TestServiceValidateRejectsUnsafeCommands(t *testing.T) {
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:    "prompt-manager",
		Message:   "ok",
	}})

	tests := []struct {
		name string
		argv []string
	}{
		{name: "empty", argv: nil},
		{name: "shell runner", argv: []string{"sh", "-c", "echo hi"}},
		{name: "raw git", argv: []string{"git", "status"}},
		{name: "raw docker", argv: []string{"docker", "ps"}},
		{name: "raw psql", argv: []string{"psql", "-c", "select 1"}},
		{name: "raw curl", argv: []string{"curl", "http://localhost"}},
		{name: "raw grep", argv: []string{"grep", "x", "file"}},
		{name: "raw rg", argv: []string{"rg", "x"}},
		{name: "pipeline", argv: []string{"prompt-manager", "skill", "list", "|", "cat"}},
		{name: "and separator", argv: []string{"prompt-manager", "skill", "list", "&&", "cat"}},
		{name: "semicolon", argv: []string{"prompt-manager", "skill", "list;cat"}},
		{name: "redirect", argv: []string{"prompt-manager", "skill", "list", ">", "out"}},
		{name: "command substitution", argv: []string{"prompt-manager", "skill", "$(whoami)"}},
		{name: "multiline", argv: []string{"prompt-manager", "skill\nlist"}},
		{name: "path executable", argv: []string{"./prompt-manager", "skill", "list"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.Validate(context.Background(), validAction(func(action *store.Action) {
				action.Command.Argv = tt.argv
			}))
			if result.Valid {
				t.Fatalf("expected invalid result for argv %#v", tt.argv)
			}
			if !hasFailedCheck(result, "schema") {
				t.Fatalf("expected schema failure, got %#v", result.Checks)
			}
		})
	}
}

func TestServiceValidateCommandOwnership(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		argv     []string
		valid    bool
		runnable bool
	}{
		{
			name:     "cataloged prompt-manager command",
			status:   store.StatusActive,
			argv:     []string{"prompt-manager", "skill", "read", "{{identifier}}"},
			valid:    true,
			runnable: true,
		},
		{
			name:   "trusted target unknown subcommand",
			status: store.StatusActive,
			argv:   []string{"prompt-manager", "definitely-not-real"},
			valid:  false,
		},
		{
			name:   "cataloged vrooli scenario command with missing permission",
			status: store.StatusActive,
			argv:   []string{"vrooli", "scenario", "status", "{{identifier}}"},
			valid:  false,
		},
		{
			name:     "cataloged vrooli scenario command",
			status:   store.StatusActive,
			argv:     []string{"vrooli", "scenario", "status", "{{identifier}}"},
			valid:    true,
			runnable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := validAction(func(action *store.Action) {
				action.Status = tt.status
				action.Command.Argv = tt.argv
				action.Permissions = store.ActionPermissions{}
				if strings.Join(tt.argv, " ") == "vrooli scenario status {{identifier}}" && tt.valid {
					action.Permissions.FilesystemRead = true
					action.Permissions.ProcessStart = true
				}
				if strings.HasPrefix(strings.Join(tt.argv, " "), "prompt-manager ") {
					action.Permissions.APIRead = true
				}
			})
			result := NewService(nil, NewManifestCommandResolver("")).Validate(context.Background(), action)
			if result.Valid != tt.valid {
				t.Fatalf("valid = %v, want %v; checks=%#v", result.Valid, tt.valid, result.Checks)
			}
			if result.Runnable != tt.runnable {
				t.Fatalf("runnable = %v, want %v; checks=%#v", result.Runnable, tt.runnable, result.Checks)
			}
		})
	}
}

func TestManifestResolverClassifiesPromptManagerSubcommands(t *testing.T) {
	resolver := NewManifestCommandResolver("")

	read, err := resolver.ResolveCommand(context.Background(), []string{"prompt-manager", "team", "decision-list", "meta-optimization", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if read.Certainty != CertaintyCommand || read.Effect != EffectRead || strings.Join(read.CommandPath, " ") != "team decision-list" {
		t.Fatalf("unexpected decision-list resolution: %#v", read)
	}

	write, err := resolver.ResolveCommand(context.Background(), []string{"prompt-manager", "team", "decision-accept", "meta-optimization", "decision-1"})
	if err != nil {
		t.Fatal(err)
	}
	if write.Certainty != CertaintyCommand || write.Effect != EffectWrite || strings.Join(write.CommandPath, " ") != "team decision-accept" {
		t.Fatalf("unexpected decision-accept resolution: %#v", write)
	}

	destructive, err := resolver.ResolveCommand(context.Background(), []string{"prompt-manager", "action", "delete", "team.decisions.list"})
	if err != nil {
		t.Fatal(err)
	}
	if destructive.Certainty != CertaintyCommand || destructive.Effect != EffectDestructive || strings.Join(destructive.CommandPath, " ") != "action delete" {
		t.Fatalf("unexpected action delete resolution: %#v", destructive)
	}

	unknown, err := resolver.ResolveCommand(context.Background(), []string{"prompt-manager", "team", "definitely-not-real"})
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Certainty != CertaintyNone {
		t.Fatalf("unknown team subcommand should not be cataloged: %#v", unknown)
	}
}

func TestServiceValidatePermissionAlignmentIncludesAPIAndProcess(t *testing.T) {
	service := NewService(nil, NewManifestCommandResolver(""))

	missingAPI := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Command.Argv = []string{"prompt-manager", "skill", "read", "{{identifier}}"}
		action.Permissions = store.ActionPermissions{}
	}))
	if missingAPI.Valid {
		t.Fatalf("expected missing api:read permission to fail")
	}
	if !hasFailedCheck(missingAPI, "permissions") {
		t.Fatalf("expected permissions failure, got %#v", missingAPI.Checks)
	}

	missingProcess := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Command.Argv = []string{"vrooli", "scenario", "status", "{{identifier}}"}
		action.Permissions = store.ActionPermissions{FilesystemRead: true}
	}))
	if missingProcess.Valid {
		t.Fatalf("expected missing process:start permission to fail")
	}
	if !hasFailedCheck(missingProcess, "permissions") {
		t.Fatalf("expected permissions failure, got %#v", missingProcess.Checks)
	}
}

func TestServiceValidateManifestOwnedDraftCommand(t *testing.T) {
	repo := t.TempDir()
	writeJSON(t, filepath.Join(repo, "scenarios", "custom-scenario", ".vrooli", "service.json"), map[string]any{
		"cli": map[string]any{
			"enabled": true,
			"command": "custom-scenario",
			"invoke":  map[string]any{"command": "custom-scenario"},
		},
	})
	storeDir := filepath.Join(repo, "scenarios", "prompt-manager", "store")
	resolver := NewManifestCommandResolver(storeDir)
	service := NewService(nil, resolver)

	draft := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.ID = "scenario.custom.inspect"
		action.Status = store.StatusDraft
		action.Command.Argv = []string{"custom-scenario", "inspect", "{{identifier}}"}
	}))
	if !draft.Valid {
		t.Fatalf("draft owner-only command should be valid with warning; checks=%#v", draft.Checks)
	}
	if draft.Command == nil || draft.Command.Certainty != CertaintyOwnerOnly {
		t.Fatalf("expected owner-only certainty, got %#v", draft.Command)
	}
	if draft.Runnable {
		t.Fatalf("owner-only draft command must not be runnable")
	}

	active := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.ID = "scenario.custom.inspect"
		action.Status = store.StatusActive
		action.Command.Argv = []string{"custom-scenario", "inspect", "{{identifier}}"}
	}))
	// Phase 0 of the cli-manifest plan downgrades active+owner-only from
	// CheckFailed to CheckWarning + Unvalidated. The action stays valid +
	// runnable; the Unvalidated flag is the safety-net surface for callers.
	if !active.Valid {
		t.Fatalf("active owner-only command should be valid (unvalidated) after Phase 0; checks=%#v", active.Checks)
	}
	if !active.Unvalidated {
		t.Fatalf("active owner-only command should be flagged Unvalidated; result=%#v", active)
	}
	if !active.Runnable {
		t.Fatalf("active owner-only command should be runnable; result=%#v", active)
	}
	if !hasCheckWithStatus(active, "command_ownership", CheckWarning) {
		t.Fatalf("expected command_ownership warning, got %#v", active.Checks)
	}
}

func TestServiceValidateRejectsRecursiveValidationHook(t *testing.T) {
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:    "prompt-manager",
		Message:   "ok",
	}})
	result := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Validation = &store.ActionValidation{Argv: []string{"prompt-manager", "action", "validate", action.ID}}
	}))
	if result.Valid {
		t.Fatalf("expected recursive validation hook to fail")
	}
	if !hasFailedCheck(result, "validation_hook") {
		t.Fatalf("expected validation_hook failure, got %#v", result.Checks)
	}
}

func TestServiceValidateRejectsSelfRecursiveRunCommand(t *testing.T) {
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty:   CertaintyCommand,
		Owner:       CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:      "prompt-manager",
		Permissions: []string{"api:write", "process:start"},
		RunSurfaces: []string{"action"},
		Message:     "ok",
	}})
	result := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Command.Argv = []string{"prompt-manager", "action", "run", action.ID}
		action.Inputs = nil
		action.Permissions = store.ActionPermissions{APIWrite: true, ProcessStart: true}
	}))
	if result.Valid {
		t.Fatalf("expected self-recursive run command to fail")
	}
	if !hasFailedCheck(result, "recursive_run") {
		t.Fatalf("expected recursive_run failure, got %#v", result.Checks)
	}
}

func TestServiceValidateRejectsUnsafeValidationHook(t *testing.T) {
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:    "prompt-manager",
		Message:   "ok",
	}})
	result := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Validation = &store.ActionValidation{Argv: []string{"sh", "-c", "prompt-manager skill list"}}
	}))
	if result.Valid {
		t.Fatalf("expected unsafe validation hook to fail")
	}
	if !hasFailedCheck(result, "schema") {
		t.Fatalf("expected schema failure, got %#v", result.Checks)
	}
}

func TestServiceValidateRejectsUnknownValidationHookForActiveAction(t *testing.T) {
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyNone,
		Target:    "custom-scenario",
		Message:   "unknown command",
	}})
	result := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Validation = &store.ActionValidation{Argv: []string{"custom-scenario", "validate"}}
	}))
	if result.Valid {
		t.Fatalf("expected unknown validation hook to fail")
	}
	if !hasFailedCheck(result, "command_ownership") || !hasFailedCheck(result, "validation_hook") {
		t.Fatalf("expected command and validation hook failures, got %#v", result.Checks)
	}
}

func TestServiceValidateRejectsDestructiveValidationHook(t *testing.T) {
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "project", ID: "vrooli"},
		Target:    "vrooli",
		Effect:    EffectDestructive,
		Message:   "cataloged destructive command",
	}})
	result := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Validation = &store.ActionValidation{Argv: []string{"vrooli", "clean"}}
	}))
	if result.Valid {
		t.Fatalf("expected destructive validation hook to fail")
	}
	if !hasFailedCheck(result, "validation_hook") {
		t.Fatalf("expected validation_hook failure, got %#v", result.Checks)
	}
}

func TestServiceValidateInputDefaults(t *testing.T) {
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:    "prompt-manager",
		Message:   "ok",
	}})
	result := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Inputs["identifier"] = store.ActionInput{
			Type:    "string",
			Default: "invalid",
			Enum:    []string{"allowed"},
		}
	}))
	if result.Valid {
		t.Fatalf("expected enum-invalid default to fail")
	}
	if !hasFailedCheck(result, "input_defaults") {
		t.Fatalf("expected input_defaults failure, got %#v", result.Checks)
	}
}

func TestServiceValidateIntegerDefaultHonorsBounds(t *testing.T) {
	minimum := 2.0
	service := NewService(nil, stubResolver{resolution: CommandResolution{
		Certainty: CertaintyCommand,
		Owner:     CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:    "prompt-manager",
		Message:   "ok",
	}})
	result := service.Validate(context.Background(), validAction(func(action *store.Action) {
		action.Inputs["count"] = store.ActionInput{
			Type:    "integer",
			Default: 1,
			Min:     &minimum,
		}
	}))
	if result.Valid {
		t.Fatalf("expected min-invalid integer default to fail")
	}
	if !hasFailedCheck(result, "input_defaults") {
		t.Fatalf("expected input_defaults failure, got %#v", result.Checks)
	}
}

func TestCoreScenarioStatusSeedValidatesAndDryRuns(t *testing.T) {
	storeDir, err := filepath.Abs("../../store")
	if err != nil {
		t.Fatal(err)
	}
	actionStore := store.NewFileActionStore(storeDir)
	action, err := actionStore.Get(context.Background(), "scenario.status.show")
	if err != nil {
		t.Fatal(err)
	}

	service := NewService(actionStore, NewManifestCommandResolver(storeDir))
	validation := service.Validate(context.Background(), action)
	if !validation.Valid || !validation.Runnable {
		t.Fatalf("seed action should validate and be runnable; checks=%#v command=%#v", validation.Checks, validation.Command)
	}
	if validation.Command == nil || strings.Join(validation.Command.CommandPath, " ") != "scenario status" {
		t.Fatalf("unexpected command resolution: %#v", validation.Command)
	}

	// Use the real seed contract but a fake store for dry-run so the test does
	// not append run audit history to the checked-in fixture directory.
	dryRunStore := newFakeActionStore(action)
	dryRunService := NewService(dryRunStore, NewManifestCommandResolver(storeDir))
	result, err := dryRunService.Run(context.Background(), "scenario.status.show", RunRequest{
		Input:  map[string]any{"scenario": "prompt-manager"},
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusDryRun {
		t.Fatalf("status = %s, want dry-run; error=%s", result.Status, result.Error)
	}
	if got, want := strings.Join(result.Argv, " "), "vrooli scenario status prompt-manager"; got != want {
		t.Fatalf("argv = %q, want %q", got, want)
	}
	if len(dryRunStore.runHistory) != 1 || dryRunStore.runHistory[0].Status != string(RunStatusDryRun) {
		t.Fatalf("expected dry-run audit entry, got %#v", dryRunStore.runHistory)
	}
}

func TestServiceRunAppliesDefaultsRendersArgvAndAudits(t *testing.T) {
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Command.Argv = []string{"prompt-manager", "skill", "read", "{{identifier}}", "--count", "{{count}}"}
		action.Inputs["count"] = store.ActionInput{Type: "integer", Default: 2}
	}))
	runner := &stubRunner{result: CommandRunResult{ExitCode: 0, Stdout: "ok"}}
	service := NewService(actionStore, runnableResolver())
	service.runner = runner

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"identifier": "implementation-plan-authoring"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusCompleted {
		t.Fatalf("status = %s, error = %s", result.Status, result.Error)
	}
	wantArgv := "prompt-manager skill read implementation-plan-authoring --count 2"
	if strings.Join(runner.argv, " ") != wantArgv {
		t.Fatalf("argv = %q, want %q", strings.Join(runner.argv, " "), wantArgv)
	}
	if len(actionStore.runHistory) != 1 || actionStore.runHistory[0].Status != string(RunStatusCompleted) {
		t.Fatalf("expected completed audit entry, got %#v", actionStore.runHistory)
	}
}

func TestServiceRunRendersSnakeCasePlaceholder(t *testing.T) {
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Command.Argv = []string{"test-genie", "provider-contract", "check", "{{phase_or_provider}}", "{{scenario}}"}
		action.Inputs = map[string]store.ActionInput{
			"phase_or_provider": {Type: "string", Required: true},
			"scenario":          {Type: "scenario", Required: true},
		}
	}))
	runner := &stubRunner{result: CommandRunResult{ExitCode: 0, Stdout: "ok"}}
	service := NewService(actionStore, runnableResolver())
	service.runner = runner

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input:  map[string]any{"phase_or_provider": "cli-health", "scenario": "test-genie"},
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusDryRun {
		t.Fatalf("status = %s, error = %s", result.Status, result.Error)
	}
	wantArgv := "test-genie provider-contract check cli-health test-genie"
	if strings.Join(result.Argv, " ") != wantArgv {
		t.Fatalf("argv = %q, want %q", strings.Join(result.Argv, " "), wantArgv)
	}
}

func TestServiceRunRejectsInvalidInputBeforeExecution(t *testing.T) {
	actionStore := newFakeActionStore(validAction(nil))
	runner := &stubRunner{result: CommandRunResult{ExitCode: 0}}
	service := NewService(actionStore, runnableResolver())
	service.runner = runner

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"identifier": "bad\nvalue"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusRejected {
		t.Fatalf("status = %s, want rejected", result.Status)
	}
	if runner.called {
		t.Fatalf("runner should not be called for rejected input")
	}
	if len(actionStore.runHistory) != 1 || actionStore.runHistory[0].ValidationValid != true {
		t.Fatalf("expected rejected audit with validation outcome, got %#v", actionStore.runHistory)
	}
}

func TestServiceRunEnforcesFileInputPermissionBeforeExecution(t *testing.T) {
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Command.Argv = []string{"prompt-manager", "skill", "read", "{{source}}"}
		action.Inputs = map[string]store.ActionInput{"source": {Type: "path", Required: true}}
		action.Permissions = store.ActionPermissions{APIRead: true}
	}))
	runner := &stubRunner{result: CommandRunResult{ExitCode: 0}}
	service := NewService(actionStore, runnableResolver())
	service.runner = runner

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"source": "docs/README.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusRejected || !strings.Contains(result.Error, "filesystem permission") {
		t.Fatalf("unexpected file input permission result: %#v", result)
	}
	if runner.called {
		t.Fatalf("runner should not be called when file input permission is missing")
	}
}

func TestServiceRunEnforcesEligibilityAndRunSurface(t *testing.T) {
	notEligible := false
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Execution = &store.ActionExecution{RunEligible: &notEligible}
	}))
	service := NewService(actionStore, runnableResolver())
	service.runner = &stubRunner{result: CommandRunResult{ExitCode: 0}}

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"identifier": "implementation-plan-authoring"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusRejected || !strings.Contains(result.Error, "disabled") {
		t.Fatalf("unexpected run eligibility result: %#v", result)
	}

	actionStore = newFakeActionStore(validAction(nil))
	service = NewService(actionStore, stubResolver{resolution: CommandResolution{
		Certainty:   CertaintyCommand,
		Owner:       CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:      "prompt-manager",
		Permissions: []string{"api:read"},
		RunSurfaces: []string{"cli"},
		Message:     "ok",
	}})
	service.runner = &stubRunner{result: CommandRunResult{ExitCode: 0}}
	result, err = service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"identifier": "implementation-plan-authoring"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusRejected || !strings.Contains(result.Error, "not eligible") {
		t.Fatalf("unexpected run surface result: %#v", result)
	}
}

func TestServiceRunThrottlesBeforeStartingProcess(t *testing.T) {
	actionStore := newFakeActionStore(validAction(nil))
	service := NewService(actionStore, runnableResolver())
	service.runner = &stubRunner{result: CommandRunResult{ExitCode: 0}}
	service.runSlots = make(chan struct{}, 1)
	service.runSlots <- struct{}{}

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"identifier": "implementation-plan-authoring"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusThrottled {
		t.Fatalf("status = %s, want throttled", result.Status)
	}
}

func TestServiceRunTimeoutCancellation(t *testing.T) {
	timeoutSeconds := 1
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Execution = &store.ActionExecution{TimeoutSeconds: &timeoutSeconds}
	}))
	service := NewService(actionStore, runnableResolver())
	service.runner = blockingRunner{}

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"identifier": "implementation-plan-authoring"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusTimedOut {
		t.Fatalf("status = %s, want timed-out; error=%s", result.Status, result.Error)
	}
}

func TestServiceRunParsesJSONOutputAndCapsAudit(t *testing.T) {
	actionStore := newFakeActionStore(validAction(func(action *store.Action) {
		action.Execution = &store.ActionExecution{OutputMode: "json"}
	}))
	service := NewService(actionStore, runnableResolver())
	service.auditLimit = 8
	service.runner = &stubRunner{result: CommandRunResult{
		ExitCode:        0,
		Stdout:          `{"value":"` + strings.Repeat("x", 20) + `"}`,
		StdoutTruncated: true,
	}}

	result, err := service.Run(context.Background(), "team.decisions.list", RunRequest{
		Input: map[string]any{"identifier": "implementation-plan-authoring"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != RunStatusCompleted || result.Output["value"] == "" {
		t.Fatalf("unexpected JSON output result: %#v", result)
	}
	if len(actionStore.runHistory) != 1 || len(actionStore.runHistory[0].Stdout) > 8 || !actionStore.runHistory[0].StdoutTruncated {
		t.Fatalf("expected capped audit stdout, got %#v", actionStore.runHistory)
	}
}

type stubResolver struct {
	resolution CommandResolution
}

func (s stubResolver) ResolveCommand(ctx context.Context, argv []string) (CommandResolution, error) {
	return s.resolution, nil
}

func runnableResolver() stubResolver {
	return stubResolver{resolution: CommandResolution{
		Certainty:   CertaintyCommand,
		Owner:       CommandOwner{Type: "scenario", ID: "prompt-manager"},
		Target:      "prompt-manager",
		Permissions: []string{"api:read"},
		RunSurfaces: []string{"action"},
		Message:     "ok",
	}}
}

type stubRunner struct {
	result CommandRunResult
	err    error
	argv   []string
	called bool
}

func (r *stubRunner) Run(ctx context.Context, argv []string, workDir string, outputLimit int) (CommandRunResult, error) {
	r.called = true
	r.argv = append([]string{}, argv...)
	return r.result, r.err
}

type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, argv []string, workDir string, outputLimit int) (CommandRunResult, error) {
	<-ctx.Done()
	return CommandRunResult{ExitCode: -1}, ctx.Err()
}

func validAction(mutate func(*store.Action)) *store.Action {
	action := &store.Action{
		BaseEntity: store.BaseEntity{Kind: store.KindAction, SchemaVersion: store.CurrentSchemaVersion},
		ID:         "team.decisions.list",
		Name:       "List Team Decisions",
		Status:     store.StatusActive,
		Owner:      store.ActionOwner{Type: "scenario", ID: "prompt-manager"},
		Command:    store.ActionCommand{Argv: []string{"prompt-manager", "skill", "read", "{{identifier}}"}},
		Inputs: map[string]store.ActionInput{
			"identifier": {Type: "string", Required: true},
		},
		Permissions: store.ActionPermissions{APIRead: true},
	}
	if mutate != nil {
		mutate(action)
	}
	return action
}

func hasFailedCheck(result ValidationResponse, code string) bool {
	return hasCheckWithStatus(result, code, CheckFailed)
}

func hasCheckWithStatus(result ValidationResponse, code string, status CheckStatus) bool {
	for _, check := range result.Checks {
		if check.Code == code && check.Status == status {
			return true
		}
	}
	return false
}

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
