package state

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

func newStateConnectService(t *testing.T) *ConnectService {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "state"))
	if err != nil {
		t.Fatalf("NewStore(): %v", err)
	}
	return NewConnectService(NewHandler(NewService(store, slog.Default())))
}

func stateString(value string) *string { return &value }

func TestStateConnectServicePersistsAndManagesScenarioState(t *testing.T) {
	service := newStateConnectService(t)
	ctx := context.Background()
	payload, err := structpb.NewStruct(map[string]any{
		"form_state":    map[string]any{"selected_template": "universal", "framework": "electron", "app_display_name": "Hello Desktop", "preflight_secrets": map[string]any{"TOKEN": "must-not-persist"}},
		"log_tails":     []any{map[string]any{"service_id": "api", "content": "line one\nline two", "lines": 2}},
		"stage_results": map[string]any{"bundle": map[string]any{"result": "ok"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	saved, err := service.SaveScenarioState(ctx, connect.NewRequest(&domainv1.SaveScenarioStateRequest{ScenarioName: "hello-desktop", Payload: payload}))
	if err != nil || saved.Msg.GetPayload().GetFields()["success"].GetBoolValue() != true {
		t.Fatalf("SaveScenarioState() = %#v, %v", saved, err)
	}
	loaded, err := service.LoadScenarioState(ctx, connect.NewRequest(&domainv1.LoadScenarioStateRequest{ScenarioName: "hello-desktop", IncludeLogs: false}))
	if err != nil || !loaded.Msg.GetFound() {
		t.Fatalf("LoadScenarioState() = %#v, %v", loaded, err)
	}
	state := loaded.Msg.GetPayload().AsMap()["state"].(map[string]any)
	form := state["form_state"].(map[string]any)
	if form["app_display_name"] != "Hello Desktop" || form["preflight_secrets"].(map[string]any)["TOKEN"] != "" {
		t.Fatalf("persisted form state = %#v", form)
	}
	if _, ok := state["compressed_logs"]; ok {
		t.Fatalf("logs returned despite include_logs=false: %#v", state)
	}

	logs, err := service.GetScenarioStateLog(ctx, connect.NewRequest(&domainv1.ScenarioStateLogRequest{ScenarioName: "hello-desktop", ServiceId: "api"}))
	if err != nil || !logs.Msg.GetFound() || logs.Msg.GetPayload().GetFields()["content"].GetStringValue() != "line one\nline two" {
		t.Fatalf("GetScenarioStateLog() = %#v, %v", logs, err)
	}
	if err := service.service.MarkStageValid(ctx, "hello-desktop", StageGenerate, InputFingerprint{TemplateType: "universal"}, nil); err != nil {
		t.Fatalf("MarkStageValid(): %v", err)
	}
	checkConfig, err := structpb.NewStruct(map[string]any{"template_type": "advanced"})
	if err != nil {
		t.Fatal(err)
	}
	checked, err := service.CheckScenarioState(ctx, connect.NewRequest(&domainv1.CheckScenarioStateRequest{ScenarioName: "hello-desktop", CurrentConfig: checkConfig}))
	if err != nil || !checked.Msg.GetPayload().GetFields()["changed"].GetBoolValue() {
		t.Fatalf("CheckScenarioState() = %#v, %v", checked, err)
	}
	invalidated, err := service.InvalidateScenarioState(ctx, connect.NewRequest(&domainv1.InvalidateScenarioStateRequest{ScenarioName: "hello-desktop", FromStage: StageBundle, Reason: stateString("inputs changed")}))
	if err != nil || invalidated.Msg.GetPayload() == nil {
		t.Fatalf("InvalidateScenarioState() = %#v, %v", invalidated, err)
	}
	deleted, err := service.DeleteScenarioState(ctx, connect.NewRequest(&domainv1.StateRequest{ScenarioName: "hello-desktop"}))
	if err != nil || !deleted.Msg.GetPayload().GetFields()["success"].GetBoolValue() {
		t.Fatalf("DeleteScenarioState() = %#v, %v", deleted, err)
	}
	missing, err := service.LoadScenarioState(ctx, connect.NewRequest(&domainv1.LoadScenarioStateRequest{ScenarioName: "hello-desktop"}))
	if err != nil || missing.Msg.GetFound() {
		t.Fatalf("state after delete = %#v, %v", missing, err)
	}
}

func TestStateConnectServiceRejectsUnconfiguredAndMalformedPayloads(t *testing.T) {
	_, err := NewConnectService(nil).LoadScenarioState(context.Background(), connect.NewRequest(&domainv1.LoadScenarioStateRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfigured code = %v", connect.CodeOf(err))
	}
	service := newStateConnectService(t)
	_, err = service.SaveScenarioState(context.Background(), connect.NewRequest(&domainv1.SaveScenarioStateRequest{ScenarioName: "hello"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing payload code = %v", connect.CodeOf(err))
	}
	_, err = service.CheckScenarioState(context.Background(), connect.NewRequest(&domainv1.CheckScenarioStateRequest{ScenarioName: "hello"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("missing current config code = %v", connect.CodeOf(err))
	}
}
