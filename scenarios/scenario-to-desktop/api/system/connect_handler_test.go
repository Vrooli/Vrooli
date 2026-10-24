package system

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type systemTestBuildStore map[string]*BuildStatus

func (s systemTestBuildStore) Snapshot() map[string]*BuildStatus { return s }

func TestGetSystemStatusUsesTypedServiceAndBuildStatistics(t *testing.T) {
	t.Parallel()

	service := NewConnectService(NewHandler(nil, nil, ""))
	response, err := service.GetSystemStatus(context.Background(), connect.NewRequest(&domainv1.GetSystemStatusRequest{}))
	if err != nil {
		t.Fatalf("GetSystemStatus() error = %v", err)
	}
	if response.Msg.GetService().GetName() != "scenario-to-desktop" || response.Msg.GetService().GetStatus() != "running" {
		t.Fatalf("unexpected service info: %#v", response.Msg.GetService())
	}
	if response.Msg.GetStatistics().GetTotalBuilds() != 0 || response.Msg.GetStatistics().GetFailedBuilds() != 0 {
		t.Fatalf("unexpected empty build statistics: %#v", response.Msg.GetStatistics())
	}
}

func TestSystemConnectServiceTemplateAndWineContracts(t *testing.T) {
	t.Parallel()

	templateDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(templateDir, "advanced"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "advanced", "universal-app.json"), []byte(`{"name":"Universal App","type":"universal"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	service := NewConnectService(NewHandler(NewWineService(slog.Default()), nil, templateDir))

	templates, err := service.ListTemplates(context.Background(), connect.NewRequest(&domainv1.ListTemplatesRequest{}))
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if templates.Msg.GetCount() != 4 || len(templates.Msg.GetTemplates()) != 4 {
		t.Fatalf("unexpected templates response: %#v", templates.Msg)
	}

	template, err := service.GetTemplate(context.Background(), connect.NewRequest(&domainv1.GetTemplateRequest{Type: "universal"}))
	if err != nil || template.Msg.GetConfig().GetFields()["type"].GetStringValue() != "universal" {
		t.Fatalf("GetTemplate() = %#v, %v", template.Msg, err)
	}
	_, err = service.GetTemplate(context.Background(), connect.NewRequest(&domainv1.GetTemplateRequest{Type: "invalid"}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("invalid template code = %s, want invalid_argument", connect.CodeOf(err))
	}

	check, err := service.CheckWine(context.Background(), connect.NewRequest(&domainv1.CheckWineRequest{}))
	if err != nil || check.Msg.GetPlatform() == "" {
		t.Fatalf("CheckWine() = %#v, %v", check.Msg, err)
	}
	install, err := service.InstallWine(context.Background(), connect.NewRequest(&domainv1.InstallWineRequest{Method: "skip"}))
	if err != nil || install.Msg.GetInstallId() == "" || install.Msg.GetStatus() != "pending" {
		t.Fatalf("InstallWine() = %#v, %v", install.Msg, err)
	}
	if install.Msg.GetMethod() != "skip" {
		t.Fatalf("InstallWine() method = %q", install.Msg.GetMethod())
	}
	status, err := service.GetWineInstallStatus(context.Background(), connect.NewRequest(&domainv1.GetWineInstallStatusRequest{InstallId: install.Msg.GetInstallId()}))
	if err != nil || status.Msg.GetInstallId() != install.Msg.GetInstallId() {
		t.Fatalf("GetWineInstallStatus() = %#v, %v", status.Msg, err)
	}
	_, err = service.GetWineInstallStatus(context.Background(), connect.NewRequest(&domainv1.GetWineInstallStatusRequest{InstallId: "missing"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing status code = %s, want not_found", connect.CodeOf(err))
	}
}

func TestSystemConnectServiceReportsBuildStatesAndTemplateFailures(t *testing.T) {
	templateDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(templateDir, "advanced"), 0o755); err != nil {
		t.Fatal(err)
	}
	service := NewConnectService(NewHandler(nil, systemTestBuildStore{
		"building": {Status: "building"}, "ready": {Status: "ready"}, "failed": {Status: "failed"}, "other": {Status: "queued"},
	}, templateDir))
	status, err := service.GetSystemStatus(context.Background(), connect.NewRequest(&domainv1.GetSystemStatusRequest{}))
	if err != nil || status.Msg.GetStatistics().GetTotalBuilds() != 4 || status.Msg.GetStatistics().GetActiveBuilds() != 1 || status.Msg.GetStatistics().GetCompletedBuilds() != 1 || status.Msg.GetStatistics().GetFailedBuilds() != 1 {
		t.Fatalf("GetSystemStatus() = %#v, %v", status, err)
	}
	if _, err := service.GetTemplate(context.Background(), connect.NewRequest(&domainv1.GetTemplateRequest{Type: "advanced"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("missing template code = %v", connect.CodeOf(err))
	}
	if err := os.WriteFile(filepath.Join(templateDir, "advanced", "advanced-app.json"), []byte("invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetTemplate(context.Background(), connect.NewRequest(&domainv1.GetTemplateRequest{Type: "advanced"})); connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("invalid template code = %v", connect.CodeOf(err))
	}
	if _, err := service.CheckWine(context.Background(), connect.NewRequest(&domainv1.CheckWineRequest{})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfigured Wine code = %v", connect.CodeOf(err))
	}
	if _, err := service.InstallWine(context.Background(), connect.NewRequest(&domainv1.InstallWineRequest{Method: "invalid"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unconfigured Wine takes precedence code = %v", connect.CodeOf(err))
	}
	if filename, ok := templateFilename("basic"); !ok || filename != "universal-app.json" {
		t.Fatalf("basic template alias = %q, %v", filename, ok)
	}
	if _, ok := templateInstallMethod("invalid"); ok {
		t.Fatal("invalid Wine method unexpectedly accepted")
	}
}
