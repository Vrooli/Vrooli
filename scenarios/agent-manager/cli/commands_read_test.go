package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	clitest "agent-manager/cli/internal/testutil"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func richResponse(t *testing.T, message proto.Message) string {
	t.Helper()
	encoded, err := protojson.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func newRichReadApp(t *testing.T) *App {
	t.Helper()
	now := timestamppb.New(time.Date(2026, time.July, 23, 12, 0, 0, 0, time.UTC))
	server := clitest.NewRecordingServerForRequests(t, func(request clitest.Request) string {
		switch request.Path {
		case "/api/v1/profiles":
			return richResponse(t, &apipb.ListProfilesResponse{Profiles: []*domainpb.AgentProfile{{Id: "profile-1", Name: "A deliberately long profile name", RoleRef: "code.default", SandboxConfig: &domainpb.SandboxConfig{ManualReview: true}}}})
		case "/api/v1/profiles/profile-1":
			if request.Method == "PUT" {
				return richResponse(t, &apipb.UpdateProfileResponse{Profile: &domainpb.AgentProfile{Id: "profile-1", Name: "updated", RoleRef: "code.default"}})
			}
			return richResponse(t, &apipb.GetProfileResponse{Profile: &domainpb.AgentProfile{Id: "profile-1", Name: "drill", Description: "reliable profile", ProfileKey: "drill", RoleRef: "code.default", MaxTurns: 4, AllowedTools: []string{"shell"}, DeniedTools: []string{"network"}, SandboxConfig: &domainpb.SandboxConfig{ManualReview: true}, CreatedAt: now, UpdatedAt: now}})
		case "/api/v1/tasks":
			return richResponse(t, &apipb.ListTasksResponse{Tasks: []*domainpb.Task{{Id: "task-1", Title: "A deliberately long task title that truncates", Status: domainpb.TaskStatus_TASK_STATUS_RUNNING, CreatedAt: now}}})
		case "/api/v1/tasks/task-1":
			if request.Method == "PUT" {
				return richResponse(t, &apipb.UpdateTaskResponse{Task: &domainpb.Task{Id: "task-1", Title: "updated"}})
			}
			return richResponse(t, &apipb.GetTaskResponse{Task: &domainpb.Task{Id: "task-1", Title: "triage", Description: "inspect evidence", Status: domainpb.TaskStatus_TASK_STATUS_RUNNING, ScopePath: "api", ProjectRoot: "/workspace", CreatedBy: "operator", ContextAttachments: []*domainpb.ContextAttachment{{Type: "file", Path: "api/main.go"}, {Type: "link", Url: "https://example.test"}, {Type: "note", Content: "context"}}, CreatedAt: now, UpdatedAt: now}})
		case "/api/v1/runs":
			return richResponse(t, &apipb.ListRunsResponse{Runs: []*domainpb.Run{{Id: "run-1", Status: domainpb.RunStatus_RUN_STATUS_RUNNING, Phase: domainpb.RunPhase_RUN_PHASE_EXECUTING, ExecutionMode: domainpb.ExecutionMode_EXECUTION_MODE_INTERACTIVE, ProgressPercent: 42, UpdatedAt: now}}})
		case "/api/v1/runs/run-1":
			profileID, sandboxID := "profile-1", "sandbox-1"
			exitCode := int32(1)
			return richResponse(t, &apipb.GetRunResponse{Run: &domainpb.Run{Id: "run-1", Status: domainpb.RunStatus_RUN_STATUS_RUNNING, Phase: domainpb.RunPhase_RUN_PHASE_EXECUTING, ExecutionMode: domainpb.ExecutionMode_EXECUTION_MODE_INTERACTIVE, TaskId: "task-1", AgentProfileId: &profileID, Tag: "drill", SandboxId: &sandboxID, ProgressPercent: 42, StartedAt: now, EndedAt: now, Summary: &domainpb.RunSummary{Description: "completed", TurnsUsed: 2, TokensUsed: 100, ContextTokens: 50, CostEstimate: 0.25}, Result: &domainpb.RunResult{Structured: &domainpb.StructuredResult{Status: domainpb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_SUCCESS, Method: "deterministic", Value: []byte(`{"ok":true}`)}}, ErrorMsg: "example error", ExitCode: &exitCode, ChangedFiles: 2, CreatedAt: now, UpdatedAt: now}})
		case "/api/v1/workflows":
			return richResponse(t, &apipb.ListWorkflowRevisionsResponse{Revisions: []*domainpb.WorkflowRevision{{Owner: "agent-manager", Key: "reliability", SemanticVersion: "v1", Digest: "digest", SourcePath: "workflows/reliability.json", Active: true}}})
		case "/api/v1/workflows/revision", "/api/v1/workflows/explain":
			return richResponse(t, &apipb.GetWorkflowRevisionResponse{Revision: &domainpb.WorkflowRevision{Owner: "agent-manager", Key: "reliability", SemanticVersion: "v1", Digest: "digest", SourcePath: "workflows/reliability.json", Active: true}})
		case "/api/v1/workflow-executions":
			return richResponse(t, &apipb.ListWorkflowExecutionsResponse{Executions: []*domainpb.WorkflowExecution{{Id: "execution-1", WorkflowKey: "reliability", DefinitionDigest: "digest", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING, CurrentNodeId: "run", Version: 2, Depth: 1}}})
		case "/api/v1/workflow-executions/execution-1", "/api/v1/workflow-executions/execution-1/result":
			return richResponse(t, &apipb.WorkflowExecutionResponse{Execution: &domainpb.WorkflowExecution{Id: "execution-1", WorkflowKey: "reliability", DefinitionDigest: "digest", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING, CurrentNodeId: "run", Version: 2}})
		case "/api/v1/health/models":
			return `{"models":[{"runner":"a-very-long-runner-name","model":"a-very-long-model-name","status":"ok","last_checked":"2026-07-23T12:00:00Z","reason":"ready"}]}`
		case "/api/v1/health/runners":
			return `{"runners":[{"runner":"a-very-long-runner-name","status":"ok","last_checked":"2026-07-23T12:00:00Z","reason":"ready"}]}`
		case "/api/v1/health/audit":
			return `{"rows":[{"timestamp":"2026-07-23T12:00:00Z","runnerType":"codex","modelId":"gpt-test","status":"ok","reason":"ready","triggeredBy":"probe"}]}`
		default:
			return `{}`
		}
	})
	api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: server.URL()} }, nil)
	return &App{services: NewServices(api)}
}

func TestReadCommandsRenderRichProfileTaskAndRunResponses(t *testing.T) {
	app := newRichReadApp(t)
	for _, args := range [][]string{{"list"}, {"get", "profile-1"}, {"update", "profile-1", "--name=updated", "--profile-key=updated", "--description=updated", "--role-ref=code.review", "--max-turns=6", "--timeout=30s", "--sandbox-mode=tracking", "--sandbox-retention-mode=keep_active"}} {
		if err := app.cmdProfile(args); err != nil {
			t.Fatalf("profile %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"list"}, {"get", "task-1"}, {"update", "task-1", "--title=updated", "--description=updated", "--scope-path=ui", "--project-root=/repo"}} {
		if err := app.cmdTask(args); err != nil {
			t.Fatalf("task %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"list"}, {"get", "run-1"}} {
		if err := app.cmdRun(args); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"list", "--owner=agent-manager"}, {"get", "--owner=agent-manager", "--key=reliability"}, {"execution-list", "--owner=agent-manager"}, {"execution-get", "execution-1"}, {"execution-result", "execution-1", "--explicitly-authorized"}} {
		if err := app.cmdWorkflow(args); err != nil {
			t.Fatalf("workflow %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"models"}, {"runners"}} {
		if err := app.cmdHealth(args); err != nil {
			t.Fatalf("health %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"audit", "--scope=model", "--runner=codex", "--model=gpt-test", "--status=ok", "--since=2026-07-23T00:00:00Z", "--until=2026-07-24T00:00:00Z", "--limit=1"}, {"audit", "--scope=runner"}} {
		if err := app.cmdHealth(args); err != nil {
			t.Fatalf("health %v: %v", args, err)
		}
	}
}

func TestRenderFallbackHandlesHonestyThresholdSortingAndTruncation(t *testing.T) {
	body := []byte(`{"history":{"has_history":true,"history_days":2.5,"min_sample_meaningful":2},"event_count":4,"runner_attempts":1,"runner_exhausted":1,"model_attempts":2,"model_exhausted":0,"runner_by_reason":{"timeout":2},"model_by_reason":{"capacity":2},"model_by_preset":{"fast":2},"model_by_pair":[{"from":"a very long model name that must truncate","to":"target","reason":"capacity exhaustion","count":2}]}`)
	if err := renderFallback(body); err != nil {
		t.Fatal(err)
	}
	if err := renderFallback([]byte(`not-json`)); err == nil {
		t.Fatal("invalid JSON should fail")
	}
	if got := trim("abcdef", 3); got != "abc" {
		t.Fatalf("trim = %q", got)
	}
	if got := trim("abcdef", 5); got != "ab..." {
		t.Fatalf("trim = %q", got)
	}
	printSortedCounts("stable", map[string]int{"z": 1, "a": 1})
}

func TestPrettyPrintJSONPreservesInvalidPayloads(t *testing.T) {
	if got := string(prettyPrintJSON([]byte(`{"b":1,"a":2}`))); got == `{"b":1,"a":2}` {
		t.Fatalf("expected formatted JSON, got %q", got)
	}
	if got := string(prettyPrintJSON([]byte(`invalid`))); got != "invalid" {
		t.Fatalf("invalid payload = %q", got)
	}
}

func TestCommandHelpSurfacesRemainAvailable(t *testing.T) {
	app := &App{}
	for _, help := range []func() error{app.workflowHelp, app.settingsHelp, app.healthHelp, app.opsHelp, app.permissionPolicyHelp, app.policyHelp, app.runnerHelp, app.profileHelp, app.taskHelp, app.runHelp, app.maintenanceHelp, app.declarationsHelp, app.eventsHelp} {
		if err := help(); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.settingsInvestigationHelp(); err != nil {
		t.Fatal(err)
	}
}

func TestPolicyValidationAndReloadSurfaceUnsuccessfulResponses(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, command := range []func([]string) error{app.permissionPolicyValidate, app.permissionPolicyReload, app.policyValidate, app.policyReload} {
		if err := command(nil); err == nil {
			t.Fatal("unsuccessful policy response should be surfaced as an error")
		}
	}
}

func TestPolicyReadSurfacesRenderNeutralResponses(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{{"status"}, {"catalog"}, {"explain", "profile", "profile-1"}, {"explain", "run", "run-1"}} {
		if err := app.cmdPolicy(args); err != nil {
			t.Fatalf("role policy %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"status"}, {"catalog"}, {"plan"}} {
		if err := app.cmdPermissionPolicy(args); err != nil {
			t.Fatalf("permission policy %v: %v", args, err)
		}
	}
	if err := app.cmdPermissionPolicy([]string{"doctor"}); err == nil {
		t.Fatal("doctor must report an unready policy")
	}
}

func TestOperatorMutationSurfacesRenderNeutralResponses(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	file := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(file, []byte(`{"defaultDepth":"standard"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"investigation", "get"}, {"investigation", "update", "--file", file}, {"investigation", "reset", "--force"}} {
		if err := app.cmdSettings(args); err != nil {
			t.Fatalf("settings %v: %v", args, err)
		}
	}
	if err := app.cmdMaintenance([]string{"purge", "--pattern=drill", "--targets=profiles,tasks,runs", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"plan", "--scenario=agent-manager"}, {"reconcile-scenario", "--scenario=agent-manager", "--dry-run"}} {
		if err := app.cmdDeclarations(args); err != nil {
			t.Fatalf("declarations %v: %v", args, err)
		}
	}
}

func TestRunControlsRenderNeutralResponses(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{{"get-by-tag", "drill"}, {"stop", "run-1"}, {"stop-by-tag", "drill"}, {"stop-all", "--force"}, {"quiesce", "--scenario=agent-manager", "--timeout=1s"}, {"approve", "run-1", "--actor=operator"}, {"reject", "run-1", "--reason=drill"}, {"await-result", "run-1"}, {"recover", "run-1"}, {"apply-investigation", "run-1"}, {"sandbox-sync", "run-1", "--status=stopped"}} {
		if err := app.cmdRun(args); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
	}
}

func TestProfileAndTaskCreateEnsureRenderNeutralResponses(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{{"create", "--name=drill", "--role-ref=code.default"}, {"ensure", "--key=drill", "--role-ref=code.default"}, {"reconcile-scenario", "--scenario=agent-manager", "--dry-run"}} {
		if err := app.cmdProfile(args); err != nil {
			t.Fatalf("profile %v: %v", args, err)
		}
	}
	if err := app.cmdTask([]string{"create", "--title=triage"}); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowValidationAndSimulationSurfaceInvalidResults(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	file := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(file, []byte(`{"kind":"input"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.workflowValidate([]string{"--file", file}); err == nil {
		t.Fatal("invalid workflow validation should fail")
	}
	if err := app.workflowSimulate([]string{"--owner=agent-manager", "--key=drill", "--input-file", file}); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowOperationalControlsRenderNeutralResponses(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{{"plan", "--scenario=agent-manager"}, {"reconcile-scenario", "--scenario=agent-manager", "--dry-run"}, {"execution-advance", "execution-1"}, {"execution-wait", "execution-1", "--timeout-seconds=1"}} {
		if err := app.cmdWorkflow(args); err != nil {
			t.Fatalf("workflow %v: %v", args, err)
		}
	}
}

func TestPrintWorkflowExecutionAcceptsMissingExecution(t *testing.T) {
	printWorkflowExecution(nil)
}

func TestReadOnlyOperationalCommandsUseServiceContracts(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{
		{"summary"}, {"health", "--json"}, {"fallback"}, {"checkpoint"},
	} {
		if err := app.cmdOps(args); err != nil {
			t.Fatalf("ops %v: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"models"}, {"runners", "--json"}, {"audit", "--scope=runner", "--runner=codex", "--limit=3"},
	} {
		if err := app.cmdHealth(args); err != nil {
			t.Fatalf("health %v: %v", args, err)
		}
	}
	if err := app.cmdEvents([]string{"list", "--run=run-1", "--type=tool.call", "--limit=2"}); err != nil {
		t.Fatal(err)
	}
	for _, call := range []func([]string) error{app.cmdOps, app.cmdHealth, app.cmdEvents} {
		if err := call([]string{"unknown"}); err == nil {
			t.Fatal("unknown subcommand should fail")
		}
	}
}

func TestTaskCommandsValidateAndUseServiceContracts(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{
		{"list", "--status=queued", "--quiet"},
		{"get", "task-1", "--json"},
		{"create", "--title=triage failure", "--description=inspect evidence", "--scope-path=api", "--project-root=/workspace"},
		{"delete", "task-1", "--force"},
		{"cancel", "task-1", "--json"},
	} {
		if err := app.cmdTask(args); err != nil {
			t.Fatalf("task %v: %v", args, err)
		}
	}
	if err := app.cmdTask([]string{"create"}); err == nil {
		t.Fatal("create without title should fail")
	}
	for _, args := range [][]string{{"get"}, {"update"}, {"delete"}, {"cancel"}, {"unknown"}} {
		if err := app.cmdTask(args); err == nil {
			t.Fatalf("task %v should fail validation", args)
		}
	}
}

func TestProfileCommandsValidateSandboxPolicyAndUseServiceContracts(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{
		{"list", "--quiet"},
		{"get", "profile-1", "--json"},
		{"create", "--name=drill", "--role-ref=code.default", "--sandbox-mode=protected", "--sandbox-retention-mode=delete_on_terminal", "--sandbox-retention-ttl=30m"},
		{"delete", "profile-1", "--force"},
		{"ensure", "--key=drill", "--role-ref=code.default", "--update"},
		{"reconcile-scenario", "--scenario=agent-manager", "--dry-run", "--json"},
	} {
		if err := app.cmdProfile(args); err != nil {
			t.Fatalf("profile %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"create", "--name=missing-role"}, {"ensure", "--key=missing-role"}, {"reconcile-scenario"}, {"update"}, {"delete"}, {"unknown"}} {
		if err := app.cmdProfile(args); err == nil {
			t.Fatalf("profile %v should fail validation", args)
		}
	}
}

func TestConfigurationAndRunnerCommandsValidateAndUseServiceContracts(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	settingsFile := filepath.Join(t.TempDir(), "investigation.json")
	if err := os.WriteFile(settingsFile, []byte(`{"defaultDepth":"standard"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"list", "--json"}, {"probe", "codex", "--json"}, {"tools", "--json"},
	} {
		if err := app.cmdRunner(args); err != nil {
			t.Fatalf("runner %v: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"status", "--json"}, {"catalog", "--json"}, {"explain", "profile", "profile-1", "--json"}, {"explain", "run", "run-1", "--json"},
	} {
		if err := app.cmdPolicy(args); err != nil {
			t.Fatalf("policy %v: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"status", "--json"}, {"catalog", "--json"}, {"plan", "--json"}, {"doctor", "--json"}, {"reconcile", "--json", "--i-was-explicitly-authorized"},
	} {
		if err := app.cmdPermissionPolicy(args); err != nil {
			t.Fatalf("permission policy %v: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"investigation", "get", "--json"}, {"investigation", "update", "--file", settingsFile, "--json"}, {"investigation", "reset", "--force", "--json"},
	} {
		if err := app.cmdSettings(args); err != nil {
			t.Fatalf("settings %v: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"purge", "--pattern=^drill-", "--targets=profiles,tasks,runs", "--dry-run", "--json"},
	} {
		if err := app.cmdMaintenance(args); err != nil {
			t.Fatalf("maintenance %v: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"reconcile-scenario", "--scenario=agent-manager", "--dry-run", "--json"}, {"plan", "--scenario=agent-manager", "--json"},
	} {
		if err := app.cmdDeclarations(args); err != nil {
			t.Fatalf("declarations %v: %v", args, err)
		}
	}

	for _, call := range []func([]string) error{app.cmdRunner, app.cmdPolicy, app.cmdPermissionPolicy, app.cmdSettings, app.cmdMaintenance, app.cmdDeclarations} {
		if err := call([]string{"unknown"}); err == nil {
			t.Fatal("unknown subcommand should fail")
		}
	}
	for _, args := range [][]string{{"probe"}, {"probe", "bad", "--bad"}} {
		if err := app.cmdRunner(args); err == nil {
			t.Fatalf("runner %v should fail validation", args)
		}
	}
	for _, args := range [][]string{{"explain", "wrong", "id"}, {"explain", "profile"}} {
		if err := app.cmdPolicy(args); err == nil {
			t.Fatalf("policy %v should fail validation", args)
		}
	}
	if err := app.cmdPermissionPolicy([]string{"reconcile"}); err == nil {
		t.Fatal("reconcile without explicit authorization should fail")
	}
	for _, args := range [][]string{{"investigation", "update"}, {"investigation", "update", "--file", "missing.json"}} {
		if err := app.cmdSettings(args); err == nil {
			t.Fatalf("settings %v should fail validation", args)
		}
	}
	for _, args := range [][]string{{"purge"}, {"purge", "--pattern=x"}, {"purge", "--pattern=x", "--targets=invalid"}} {
		if err := app.cmdMaintenance(args); err == nil {
			t.Fatalf("maintenance %v should fail validation", args)
		}
	}
	if err := app.cmdDeclarations([]string{"plan"}); err == nil {
		t.Fatal("declarations plan without scenario should fail")
	}
}

func TestWorkflowCommandsValidateFilesAndUseServiceContracts(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	input := filepath.Join(t.TempDir(), "workflow.json")
	if err := os.WriteFile(input, []byte(`{"kind":"input"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"validate", "--file", input, "--json"},
		{"plan", "--scenario=agent-manager", "--json"},
		{"reconcile-scenario", "--scenario=agent-manager", "--dry-run", "--json"},
		{"reload", "--scenario=agent-manager", "--json"},
		{"list", "--owner=agent-manager", "--json"},
		{"get", "--owner=agent-manager", "--key=drill", "--json"},
		{"explain", "--owner=agent-manager", "--digest=digest", "--json"},
		{"start", "--owner=agent-manager", "--key=drill", "--input-file", input, "--idempotency-key=start-1", "--json"},
		{"execution-list", "--owner=agent-manager", "--status=running", "--json"},
		{"execution-runs", "execution-1", "--json"},
		{"execution-get", "execution-1", "--json"},
		{"execution-result", "execution-1", "--explicitly-authorized", "--json"},
		{"execution-advance", "execution-1", "--json"},
		{"execution-wait", "execution-1", "--timeout-seconds=1", "--json"},
		{"trace", "execution-1", "--after-sequence=2", "--json"},
		{"signal", "execution-1", "--signal=approval", "--payload-file", input, "--idempotency-key=signal-1", "--json"},
		{"cancel", "execution-1", "--idempotency-key=cancel-1", "--json"},
		{"retry", "execution-1", "--idempotency-key=retry-1", "--json"},
		{"resume", "execution-1", "--idempotency-key=resume-1", "--json"},
		{"simulate", "--owner=agent-manager", "--key=drill", "--input-file", input, "--json"},
	} {
		if err := app.cmdWorkflow(args); err != nil {
			t.Fatalf("workflow %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"validate"}, {"list"}, {"get", "--owner=agent-manager"}, {"start", "--owner=agent-manager"}, {"execution-result", "execution-1"}, {"signal", "execution-1"}, {"cancel", "execution-1"}, {"unknown"}} {
		if err := app.cmdWorkflow(args); err == nil {
			t.Fatalf("workflow %v should fail validation", args)
		}
	}
}

func TestRunCommandsValidateAndUseServiceContracts(t *testing.T) {
	services, _ := newContractServices(t)
	app := &App{services: services}
	for _, args := range [][]string{
		{"list", "--status=running", "--json"},
		{"get", "run-1", "--json"},
		{"get-by-tag", "drill", "--json"},
		{"create", "--task-id=task-1", "--profile-id=profile-1", "--run-mode=in_place", "--execution-mode=interactive", "--classify=pass,fail", "--json"},
		{"delete", "run-1", "--force"},
		{"stop", "run-1", "--json"},
		{"stop-by-tag", "drill", "--json"},
		{"stop-all", "--force", "--json"},
		{"quiesce", "--scenario=agent-manager", "--timeout=1s", "--json"},
		{"approve", "run-1", "--actor=operator", "--json"},
		{"reject", "run-1", "--reason=drill", "--json"},
		{"diff", "run-1"},
		{"events", "run-1", "--after-sequence=2", "--json"},
		{"continue", "run-1", "--message=continue", "--json"},
		{"park", "run-1", "--producer=test-genie", "--key=case-1", "--identity-token=token", "--json"},
		{"wake", "run-1", "--result=ready", "--json"},
		{"await-result", "run-1", "--json"},
		{"recover", "run-1", "--json"},
		{"investigate", "--run-ids=run-1,run-2", "--depth=quick", "--json"},
		{"apply-investigation", "run-1", "--json"},
		{"sandbox-sync", "run-1", "--status=stopped", "--json"},
	} {
		if err := app.cmdRun(args); err != nil {
			t.Fatalf("run %v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"get"}, {"create", "--task-id=task-1"}, {"quiesce"}, {"continue", "run-1"}, {"park", "run-1"}, {"investigate"}, {"sandbox-sync", "run-1"}, {"unknown"}} {
		if err := app.cmdRun(args); err == nil {
			t.Fatalf("run %v should fail validation", args)
		}
	}
}
