package records

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

func recordTestHandler(t *testing.T) (*Handler, *FileStore) {
	t.Helper()
	store, err := NewFileStore(filepath.Join(t.TempDir(), "records.json"))
	if err != nil {
		t.Fatalf("create record store: %v", err)
	}
	return NewHandler(store, nil, slog.Default()), store
}

func TestConnectServiceListsRecordsWithOptionalContractFields(t *testing.T) {
	handler, store := recordTestHandler(t)
	if err := store.Upsert(&DesktopAppRecord{
		ID: "record-1", ScenarioName: "hello-desktop", BuildID: "build-1", OutputPath: "/tmp/output",
		AppDisplayName: "Hello Desktop", TemplateType: "vanilla", Framework: "electron", LocationMode: "proper",
		DestinationPath: "/tmp/destination", Icon: "/tmp/icon.png",
	}); err != nil {
		t.Fatalf("store record: %v", err)
	}
	service := NewConnectService(handler)
	response, err := service.ListDesktopRecords(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		t.Fatalf("ListDesktopRecords(): %v", err)
	}
	if len(response.Msg.GetRecords()) != 1 {
		t.Fatalf("records = %#v", response.Msg.GetRecords())
	}
	record := response.Msg.GetRecords()[0].GetRecord()
	if record.GetId() != "record-1" || record.GetFramework() != "electron" || record.GetAppDisplayName() != "Hello Desktop" {
		t.Fatalf("record proto = %#v", record)
	}
}

func TestConnectServiceMovesRecordAndUpdatesPersistentState(t *testing.T) {
	handler, store := recordTestHandler(t)
	root := t.TempDir()
	source := filepath.Join(root, "platforms", "electron", "app.txt")
	destination := filepath.Join(root, "releases", "app.txt")
	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("desktop artifact"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(&DesktopAppRecord{ID: "record-1", ScenarioName: "hello-desktop", OutputPath: source, DestinationPath: destination}); err != nil {
		t.Fatal(err)
	}

	response, err := NewConnectService(handler).MoveDesktopRecord(context.Background(), connect.NewRequest(&domainv1.MoveDesktopRecordRequest{RecordId: "record-1"}))
	if err != nil || response.Msg.GetStatus() != "moved" || response.Msg.GetTo() != destination {
		t.Fatalf("MoveDesktopRecord() = %#v, %v", response, err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	if data, err := os.ReadFile(destination); err != nil || string(data) != "desktop artifact" {
		t.Fatalf("destination = %q, %v", data, err)
	}
	updated, ok := store.Get("record-1")
	if !ok || updated.OutputPath != destination || updated.LocationMode != "proper" {
		t.Fatalf("updated record = %#v, present=%v", updated, ok)
	}
	if _, err := NewConnectService(handler).MoveDesktopRecord(context.Background(), connect.NewRequest(&domainv1.MoveDesktopRecordRequest{RecordId: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing move error code = %v", connect.CodeOf(err))
	}
}

func TestConnectServiceDeletesOnlyElectronOutputAndCleansRecords(t *testing.T) {
	handler, store := recordTestHandler(t)
	root := t.TempDir()
	output := filepath.Join(root, "platforms", "electron", "hello-desktop")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "artifact"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(&DesktopAppRecord{ID: "record-1", ScenarioName: "hello-desktop", OutputPath: output}); err != nil {
		t.Fatal(err)
	}
	handler.outputPathFunc = func(string) string { return output }
	service := NewConnectService(handler)

	response, err := service.DeleteDesktopScenario(context.Background(), connect.NewRequest(&domainv1.DeleteDesktopScenarioRequest{ScenarioName: "hello-desktop"}))
	if err != nil || response.Msg.GetRemovedRecords() != 1 || response.Msg.GetStatus() != "success" {
		t.Fatalf("DeleteDesktopScenario() = %#v, %v", response, err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output still exists: %v", err)
	}
	if _, ok := store.Get("record-1"); ok {
		t.Fatal("record still exists")
	}
	if _, err := service.DeleteDesktopScenario(context.Background(), connect.NewRequest(&domainv1.DeleteDesktopScenarioRequest{ScenarioName: "../escape"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unsafe name error code = %v", connect.CodeOf(err))
	}
	handler.outputPathFunc = func(string) string { return filepath.Join(root, "outside") }
	if _, err := service.DeleteDesktopScenario(context.Background(), connect.NewRequest(&domainv1.DeleteDesktopScenarioRequest{ScenarioName: "hello-desktop"})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("unsafe path error code = %v", connect.CodeOf(err))
	}
}
