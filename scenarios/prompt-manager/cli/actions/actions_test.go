package actions

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type fakeContext struct {
	method   string
	path     string
	query    url.Values
	payload  any
	response any
}

func (f *fakeContext) Get(path string, result interface{}) error {
	f.method = "GET"
	f.path = path
	return f.writeResult(result)
}

func (f *fakeContext) GetWithQuery(path string, query url.Values, result interface{}) error {
	f.method = "GET"
	f.path = path
	f.query = query
	return f.writeResult(result)
}

func (f *fakeContext) Post(path string, payload interface{}, result interface{}) error {
	f.method = "POST"
	f.path = path
	f.payload = payload
	return f.writeResult(result)
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
	if result == nil || f.response == nil {
		return nil
	}
	raw, err := json.Marshal(f.response)
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
	ctx := &fakeContext{response: Action{ID: "team.decisions.list", Name: "List Decisions", Status: "active"}}
	if err := cmdShow(ctx, []string{"team.decisions.list"}); err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	if ctx.method != "GET" || ctx.path != "/actions/team.decisions.list" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
}

func TestCmdCreatePostsActionFileWithPack(t *testing.T) {
	file := writeActionFixture(t, Action{
		ID:      "team.decisions.list",
		Name:    "List Decisions",
		Status:  "draft",
		Owner:   ActionOwner{Type: "scenario", ID: "prompt-manager"},
		Command: ActionCommand{Argv: []string{"prompt-manager", "team", "decision-list", "meta-optimization"}},
	})
	ctx := &fakeContext{response: MutationResponse{
		Action: &Action{ID: "team.decisions.list", Name: "List Decisions", Status: "draft"},
		Validation: ValidationResponse{
			ActionID: "team.decisions.list",
			Valid:    true,
		},
	}}

	if err := cmdCreate(ctx, []string{"--file", file, "--pack=core"}); err != nil {
		t.Fatalf("cmdCreate: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	action, ok := ctx.payload.(Action)
	if !ok {
		t.Fatalf("payload type = %T, want Action", ctx.payload)
	}
	if action.ID != "team.decisions.list" || action.Pack != "core" {
		t.Fatalf("unexpected action payload: %+v", action)
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
		ID:      "team.decisions.list",
		Name:    "List Decisions",
		Status:  "active",
		Owner:   ActionOwner{Type: "scenario", ID: "prompt-manager"},
		Command: ActionCommand{Argv: []string{"prompt-manager", "team", "decision-list", "meta-optimization"}},
	})
	ctx := &fakeContext{response: MutationResponse{
		Action: &Action{ID: "team.decisions.list", Name: "List Decisions", Status: "active"},
		Validation: ValidationResponse{
			ActionID: "team.decisions.list",
			Valid:    true,
			Runnable: true,
		},
	}}

	if err := cmdUpdate(ctx, []string{"team.decisions.list", "--file", file}); err != nil {
		t.Fatalf("cmdUpdate: %v", err)
	}
	if ctx.method != "PUT" || ctx.path != "/actions/team.decisions.list" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
	action, ok := ctx.payload.(Action)
	if !ok {
		t.Fatalf("payload type = %T, want Action", ctx.payload)
	}
	if action.ID != "team.decisions.list" {
		t.Fatalf("unexpected action payload: %+v", action)
	}
}

func TestCmdDeleteArchivesByDefault(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdDelete(ctx, []string{"team.decisions.list", "--yes"}); err != nil {
		t.Fatalf("cmdDelete: %v", err)
	}
	if ctx.method != "DELETE" || ctx.path != "/actions/team.decisions.list" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
}

func TestCmdDeleteSupportsHardDelete(t *testing.T) {
	ctx := &fakeContext{}
	if err := cmdDelete(ctx, []string{"team.decisions.list", "--yes", "--hard"}); err != nil {
		t.Fatalf("cmdDelete: %v", err)
	}
	if ctx.method != "DELETE" || ctx.path != "/actions/team.decisions.list?hard=true" {
		t.Fatalf("unexpected request: %s %s", ctx.method, ctx.path)
	}
}

func TestCmdValidatePostsToValidateEndpoint(t *testing.T) {
	ctx := &fakeContext{response: ValidationResponse{
		ActionID: "team.decisions.list",
		Valid:    true,
		Runnable: true,
		Checks: []ValidationCheck{{
			Code:    "command-target",
			Status:  "passed",
			Message: "ok",
		}},
	}}
	if err := cmdValidate(ctx, []string{"team.decisions.list"}); err != nil {
		t.Fatalf("cmdValidate: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions/team.decisions.list/validate" {
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
	ctx := &fakeContext{response: RunResponse{
		ActionID:   "team.decisions.list",
		Status:     "completed",
		ExitCode:   &exitCode,
		DurationMs: 12,
		Stdout:     "ok",
	}}

	err := cmdRun(ctx, []string{"team.decisions.list", "--input", `{"team":"meta-optimization","limit":3}`})
	if err != nil {
		t.Fatalf("cmdRun: %v", err)
	}
	if ctx.method != "POST" || ctx.path != "/actions/team.decisions.list/run" {
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
	err := cmdRun(ctx, []string{"team.decisions.list", "--input", `{}`, "--input-file", "payload.json"})
	if err == nil {
		t.Fatal("expected mutually exclusive input flags to return an error")
	}
	if ctx.method != "" {
		t.Fatalf("expected no API call, got %s %s", ctx.method, ctx.path)
	}
}

func TestCmdRunRejectsInvalidInputBeforeAPI(t *testing.T) {
	ctx := &fakeContext{}
	err := cmdRun(ctx, []string{"team.decisions.list", "--input", `[]`})
	if err == nil {
		t.Fatal("expected non-object input to return an error")
	}
	if ctx.method != "" {
		t.Fatalf("expected no API call, got %s %s", ctx.method, ctx.path)
	}
}

func TestCmdRunReturnsErrorForFailedStatus(t *testing.T) {
	ctx := &fakeContext{response: RunResponse{
		ActionID: "team.decisions.list",
		Status:   "failed",
		Error:    "exit status 2",
	}}
	err := cmdRun(ctx, []string{"team.decisions.list"})
	if err == nil {
		t.Fatal("expected failed run status to return an error")
	}
	if ctx.method != "POST" || ctx.path != "/actions/team.decisions.list/run" {
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
