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
