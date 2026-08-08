package actions

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type fakeContext struct {
	method       string
	path         string
	query        url.Values
	payload      any
	response     any
	getResponse  any
	postResponse any
}

func (f *fakeContext) Get(path string, result interface{}) error {
	f.method = "GET"
	f.path = path
	return f.writeResultWith(f.responseOr(f.getResponse), result)
}

func (f *fakeContext) GetWithQuery(path string, query url.Values, result interface{}) error {
	f.method = "GET"
	f.path = path
	f.query = query
	return f.writeResultWith(f.response, result)
}

func (f *fakeContext) Post(path string, payload interface{}, result interface{}) error {
	f.method = "POST"
	f.path = path
	f.payload = payload
	return f.writeResultWith(f.responseOr(f.postResponse), result)
}

func (f *fakeContext) Put(path string, payload interface{}, result interface{}) error {
	f.method = "PUT"
	f.path = path
	f.payload = payload
	return f.writeResult(result)
}

func (f *fakeContext) Delete(path string) error {
	f.method = "DELETE"
	f.path = path
	return nil
}

func (f *fakeContext) DeleteWithQuery(path string, _ url.Values, result interface{}) error {
	f.method = "DELETE"
	f.path = path
	return f.writeResult(result)
}

func (f *fakeContext) writeResult(result interface{}) error {
	return f.writeResultWith(f.response, result)
}

func (f *fakeContext) responseOr(specific any) any {
	if specific != nil {
		return specific
	}
	return f.response
}

func (f *fakeContext) writeResultWith(response any, result interface{}) error {
	if result == nil || response == nil {
		return nil
	}
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, result)
}

func TestCommandsRegistersActionCommand(t *testing.T) {
	group := Commands(nil)
	if group.Title != "Actions" {
		t.Fatalf("unexpected group title: %q", group.Title)
	}
	if len(group.Commands) != 1 {
		t.Fatalf("expected one command, got %d", len(group.Commands))
	}
	cmd := group.Commands[0]
	if cmd.Name != "action" {
		t.Fatalf("unexpected command name: %q", cmd.Name)
	}
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "actions" {
		t.Fatalf("unexpected aliases: %+v", cmd.Aliases)
	}
}

func TestCmdListBuildsFilters(t *testing.T) {
	ctx := &fakeContext{response: []Action{}}
	err := cmdList(ctx, []string{"--pack=core", "--status=active", "--owner=prompt-manager", "--tag=ops"})
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if ctx.method != "GET" || ctx.path != "/actions" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	for key, want := range map[string]string{
		"pack":   "core",
		"status": "active",
		"owner":  "prompt-manager",
		"tag":    "ops",
	} {
		if got := ctx.query.Get(key); got != want {
			t.Fatalf("query %s = %q, want %q", key, got, want)
		}
	}
}

func TestCmdShowGetsActionByID(t *testing.T) {
	ctx := &fakeContext{response: Action{ID: "team.swarm.work.list", Name: "List Work", Status: "active"}}
	if err := cmdShow(ctx, []string{"team.swarm.work.list"}); err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	if ctx.method != "GET" || ctx.path != "/actions/team.swarm.work.list" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
}

func TestCmdCreatePostsActionFileWithPack(t *testing.T) {
	file := writeActionFixture(t, Action{
		ID:      "team.swarm.work.list",
		Name:    "List Work",
		Status:  "draft",
		Owner:   ActionOwner{Type: "scenario", ID: "prompt-manager"},
		Command: ActionCommand{Argv: []string{"swarm-manager", "backlog", "list", "--json"}},
	})
	ctx := &fakeContext{response: MutationResponse{
		Action: &Action{ID: "team.swarm.work.list", Name: "List Work", Status: "draft"},
		Validation: ValidationResponse{
			ActionID: "team.swarm.work.list",
			Valid:    true,
		},
	}}

	// --apply writes; without it the command previews. The preview call (POST
	// /actions/preview) reuses the same fake response, then the apply call
	// (POST /actions) posts the file's contract.
	if err := cmdCreate(ctx, []string{"--file", file, "--pack=core", "--apply"}); err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	action, ok := ctx.payload.(Action)
	if !ok {
		t.Fatalf("payload type = %T, want Action", ctx.payload)
	}
	if action.ID != "team.swarm.work.list" || action.Pack != "core" {
		t.Fatalf("unexpected action payload: %+v", action)
	}
}

func TestCmdCreatePreviewsByDefault(t *testing.T) {
	ctx := &fakeContext{response: ActionPreview{
		Rendered:   &Action{ID: "scenario.status.show", Name: "Show Status", Status: "active"},
		Validation: ValidationResponse{ActionID: "scenario.status.show", Valid: true, Runnable: true},
	}}
	if err := cmdCreate(ctx, []string{"--name", "Show Status", "--command", "vrooli scenario status {{scenario}}"}); err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	// Default (no --apply) must hit the preview endpoint and write nothing.
	if ctx.method != "POST" || ctx.path != "/actions/preview" {
		t.Fatalf("expected preview request, got %s %s", ctx.method, ctx.path)
	}
	draft, ok := ctx.payload.(DraftActionInput)
	if !ok {
		t.Fatalf("payload type = %T, want DraftActionInput", ctx.payload)
	}
	wantArgv := []string{"vrooli", "scenario", "status", "{{scenario}}"}
	if len(draft.Argv) != len(wantArgv) {
		t.Fatalf("argv = %#v, want %#v", draft.Argv, wantArgv)
	}
	for i := range wantArgv {
		if draft.Argv[i] != wantArgv[i] {
			t.Fatalf("argv[%d] = %q, want %q", i, draft.Argv[i], wantArgv[i])
		}
	}
}

func TestCmdCreateRejectsBothCommandAndFile(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdCreate(ctx, []string{"--command", "vrooli scenario status {{s}}", "--file", "x.json"}); err == nil {
		t.Fatal("expected error when both --command and --file are provided")
	}
	if ctx.method != "" {
		t.Fatalf("expected no API call, got %s %s", ctx.method, ctx.path)
	}
}

func TestCmdCreateRejectsInvalidActionFileBeforeAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "action.json")
	if err := os.WriteFile(path, []byte(`{"id":`), 0o600); err != nil {
		t.Fatalf("write invalid fixture: %v", err)
	}
	ctx := &fakeContext{}
	if err := cmdCreate(ctx, []string{"--file", path}); err == nil {
		t.Fatal("expected invalid action file to return an error")
	}
	if ctx.method != "" {
		t.Fatalf("expected no API call, got %s %s", ctx.method, ctx.path)
	}
}

func TestCmdUpdatePutsActionFileByID(t *testing.T) {
	file := writeActionFixture(t, Action{
		ID:      "team.swarm.work.list",
		Name:    "List Work",
		Status:  "active",
		Owner:   ActionOwner{Type: "scenario", ID: "prompt-manager"},
		Command: ActionCommand{Argv: []string{"swarm-manager", "backlog", "list", "--json"}},
	})
	ctx := &fakeContext{response: MutationResponse{
		Action: &Action{ID: "team.swarm.work.list", Name: "List Work", Status: "active"},
		Validation: ValidationResponse{
			ActionID: "team.swarm.work.list",
			Valid:    true,
			Runnable: true,
		},
	}}

	if err := cmdUpdate(ctx, []string{"team.swarm.work.list", "--file", file}); err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}
	if ctx.method != "PUT" || ctx.path != "/actions/team.swarm.work.list" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	action, ok := ctx.payload.(Action)
	if !ok {
		t.Fatalf("payload type = %T, want Action", ctx.payload)
	}
	if action.ID != "team.swarm.work.list" {
		t.Fatalf("unexpected action payload: %+v", action)
	}
}

func TestCmdDeleteArchivesByDefault(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdDelete(ctx, []string{"team.swarm.work.list", "--yes"}); err != nil {
		t.Fatalf("cmdDelete: %v", err)
	}
	if ctx.method != "DELETE" || ctx.path != "/actions/team.swarm.work.list" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
}

func TestCmdDeleteSupportsHardDelete(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdDelete(ctx, []string{"team.swarm.work.list", "--yes", "--hard"}); err != nil {
		t.Fatalf("cmdDelete: %v", err)
	}
	if ctx.method != "DELETE" || ctx.path != "/actions/team.swarm.work.list?hard=true" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
}

func TestCmdValidatePostsToValidateEndpoint(t *testing.T) {
	ctx := &fakeContext{response: ValidationResponse{
		ActionID: "team.swarm.work.list",
		Valid:    true,
		Runnable: true,
		Checks: []ValidationCheck{{
			Code:    "command-target",
			Status:  "passed",
			Message: "ok",
		}},
	}}
	if err := cmdValidate(ctx, []string{"team.swarm.work.list"}); err != nil {
		t.Fatalf("cmdValidate: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions/team.swarm.work.list/validate" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	if ctx.payload == nil {
		t.Fatal("expected validate request payload")
	}
}

func TestCmdValidateReturnsErrorForInvalidAction(t *testing.T) {
	ctx := &fakeContext{response: ValidationResponse{
		ActionID: "bad.action",
		Valid:    false,
		Checks: []ValidationCheck{{
			Code:    "command-target",
			Status:  "failed",
			Message: "unknown command",
		}},
	}}
	if err := cmdValidate(ctx, []string{"bad.action"}); err == nil {
		t.Fatal("expected invalid validation response to return an error")
	}
}

func TestCmdRunPostsInputToRunEndpoint(t *testing.T) {
	exitCode := 0
	ctx := &fakeContext{postResponse: RunResponse{
		ActionID:   "team.swarm.work.list",
		Status:     "completed",
		ExitCode:   &exitCode,
		DurationMs: 12,
		Stdout:     "ok",
	}}

	err := cmdRun(ctx, []string{"team.swarm.work.list", "--input", `{"team":"meta-optimization","limit":3}`})
	if err != nil {
		t.Fatalf("cmdRun: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions/team.swarm.work.list/run" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	req, ok := ctx.payload.(RunRequest)
	if !ok {
		t.Fatalf("payload type = %T, want RunRequest", ctx.payload)
	}
	if req.DryRun {
		t.Fatal("dryRun = true, want false")
	}
	if req.Input["team"] != "meta-optimization" {
		t.Fatalf("team input = %v", req.Input["team"])
	}
	if req.Input["limit"] != float64(3) {
		t.Fatalf("limit input = %#v, want JSON number 3", req.Input["limit"])
	}
}

func TestCmdRunAcceptsDeclaredNamedInputFlags(t *testing.T) {
	exitCode := 0
	ctx := &fakeContext{
		getResponse: Action{ID: "bas.screenshot", Inputs: map[string]ActionInput{
			"url": {Type: "string", Required: true},
			"out": {Type: "path", Required: true},
		}},
		postResponse: RunResponse{ActionID: "bas.screenshot", Status: "completed", ExitCode: &exitCode},
	}

	err := cmdRun(ctx, []string{"bas.screenshot", "--url", "scenario=react-component-library,path=/assets/Button", "--out", "rcl-button"})
	if err != nil {
		t.Fatalf("cmdRun: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions/bas.screenshot/run" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	req, ok := ctx.payload.(RunRequest)
	if !ok {
		t.Fatalf("payload type = %T, want RunRequest", ctx.payload)
	}
	if req.Input["url"] != "scenario=react-component-library,path=/assets/Button" || req.Input["out"] != "rcl-button" {
		t.Fatalf("named inputs = %#v", req.Input)
	}
}

func TestCmdRunRejectsDuplicateNamedAndJSONInput(t *testing.T) {
	ctx := &fakeContext{getResponse: Action{ID: "bas.screenshot", Inputs: map[string]ActionInput{"url": {Type: "string"}}}}
	err := cmdRun(ctx, []string{"bas.screenshot", "--input", `{"url":"one"}`, "--url", "two"})
	if err == nil {
		t.Fatal("expected duplicate input error")
	}
	if ctx.method != "GET" {
		t.Fatalf("expected no POST, got %s %s", ctx.method, ctx.path)
	}
}

func TestCmdRunSupportsInputFileAndDryRun(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(inputPath, []byte(`{"scenario":"prompt-manager"}`), 0o600); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	ctx := &fakeContext{response: RunResponse{
		ActionID: "scenario.status.show",
		Status:   "dry-run",
		Argv:     []string{"vrooli", "scenario", "status", "prompt-manager"},
	}}

	if err := cmdRun(ctx, []string{"scenario.status.show", "--input-file", inputPath, "--dry-run"}); err != nil {
		t.Fatalf("cmdRun: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions/scenario.status.show/run" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	req, ok := ctx.payload.(RunRequest)
	if !ok {
		t.Fatalf("payload type = %T, want RunRequest", ctx.payload)
	}
	if !req.DryRun {
		t.Fatal("dryRun = false, want true")
	}
	if req.Input["scenario"] != "prompt-manager" {
		t.Fatalf("scenario input = %v", req.Input["scenario"])
	}
}

func TestCmdRunRejectsBothInputForms(t *testing.T) {
	ctx := &fakeContext{}
	err := cmdRun(ctx, []string{"team.swarm.work.list", "--input", `{}`, "--input-file", "payload.json"})
	if err == nil {
		t.Fatal("expected mutually exclusive input flags to return an error")
	}
	if ctx.method != "" {
		t.Fatalf("expected no API call, got %s %s", ctx.method, ctx.path)
	}
}

func TestCmdRunRejectsInvalidInputBeforeAPI(t *testing.T) {
	ctx := &fakeContext{}
	err := cmdRun(ctx, []string{"team.swarm.work.list", "--input", `[]`})
	if err == nil {
		t.Fatal("expected non-object input to return an error")
	}
	if ctx.method != "" {
		t.Fatalf("expected no API call, got %s %s", ctx.method, ctx.path)
	}
}

func TestCmdRunReturnsErrorForFailedStatus(t *testing.T) {
	ctx := &fakeContext{response: RunResponse{
		ActionID: "team.swarm.work.list",
		Status:   "failed",
		Error:    "exit status 2",
	}}
	err := cmdRun(ctx, []string{"team.swarm.work.list"})
	if err == nil {
		t.Fatal("expected failed run status to return an error")
	}
	if ctx.method != "POST" || ctx.path != "/actions/team.swarm.work.list/run" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
}

func writeActionFixture(t *testing.T, action Action) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "action.json")
	raw, err := json.Marshal(action)
	if err != nil {
		t.Fatalf("marshal action: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write action fixture: %v", err)
	}
	return path
}
