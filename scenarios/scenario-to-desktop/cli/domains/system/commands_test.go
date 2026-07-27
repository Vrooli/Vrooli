package system

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fakeRecordsRPC struct {
	list    *domainv1.DesktopRecordsResponse
	move    *domainv1.MoveDesktopRecordRequest
	deleted *domainv1.DeleteDesktopScenarioRequest
}
type fakeSystemRPC struct {
	templates *domainv1.ListTemplatesResponse
	template  *domainv1.TemplateConfigResponse
	wine      *domainv1.WineCheckResponse
	install   *domainv1.InstallWineRequest
	status    *domainv1.GetWineInstallStatusRequest
}
type fakeOperationsRPC struct {
	response *domainv1.DesktopScenarioStatusResponse
}

func (f *fakeOperationsRPC) ListDesktopScenarioStatus(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopScenarioStatusResponse], error) {
	return connect.NewResponse(f.response), nil
}

func (f *fakeSystemRPC) ListTemplates(context.Context, *connect.Request[domainv1.ListTemplatesRequest]) (*connect.Response[domainv1.ListTemplatesResponse], error) {
	return connect.NewResponse(f.templates), nil
}

func (f *fakeSystemRPC) GetTemplate(context.Context, *connect.Request[domainv1.GetTemplateRequest]) (*connect.Response[domainv1.TemplateConfigResponse], error) {
	return connect.NewResponse(f.template), nil
}

func (f *fakeSystemRPC) CheckWine(context.Context, *connect.Request[domainv1.CheckWineRequest]) (*connect.Response[domainv1.WineCheckResponse], error) {
	return connect.NewResponse(f.wine), nil
}

func (f *fakeSystemRPC) InstallWine(_ context.Context, r *connect.Request[domainv1.InstallWineRequest]) (*connect.Response[domainv1.WineInstallResponse], error) {
	f.install = r.Msg
	return connect.NewResponse(&domainv1.WineInstallResponse{InstallId: "inst-001", Method: r.Msg.GetMethod(), Status: "started"}), nil
}

func (f *fakeSystemRPC) GetWineInstallStatus(_ context.Context, r *connect.Request[domainv1.GetWineInstallStatusRequest]) (*connect.Response[domainv1.WineInstallStatusResponse], error) {
	f.status = r.Msg
	return connect.NewResponse(&domainv1.WineInstallStatusResponse{InstallId: r.Msg.GetInstallId(), Status: "complete"}), nil
}

func (f *fakeRecordsRPC) ListDesktopRecords(context.Context, *connect.Request[emptypb.Empty]) (*connect.Response[domainv1.DesktopRecordsResponse], error) {
	return connect.NewResponse(f.list), nil
}

func (f *fakeRecordsRPC) MoveDesktopRecord(_ context.Context, r *connect.Request[domainv1.MoveDesktopRecordRequest]) (*connect.Response[domainv1.MoveDesktopRecordResponse], error) {
	f.move = r.Msg
	return connect.NewResponse(&domainv1.MoveDesktopRecordResponse{RecordId: r.Msg.GetRecordId(), Status: "moved"}), nil
}

func (f *fakeRecordsRPC) DeleteDesktopScenario(_ context.Context, r *connect.Request[domainv1.DeleteDesktopScenarioRequest]) (*connect.Response[domainv1.DeleteDesktopScenarioResponse], error) {
	f.deleted = r.Msg
	return connect.NewResponse(&domainv1.DeleteDesktopScenarioResponse{ScenarioName: r.Msg.GetScenarioName(), Status: "success"}), nil
}

func newTestDependencies(t *testing.T, handler http.Handler) support.Dependencies {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "scenario-to-desktop-test", Version: "test", Description: "test", DefaultAPIBase: server.URL, AllowAnonymous: true, CommandGroups: func(*cliapp.ScenarioApp) []cliapp.CommandGroup { return nil }, SubcommandGroups: func(*cliapp.ScenarioApp) []cliapp.SubcommandGroup { return nil }})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return support.Dependencies{Core: func() *cliapp.ScenarioApp { return app }}
}

func assertPrimitiveModes(t *testing.T, handler cliapp.PrimitiveHandler, schema cliapp.ArgSchema, args []string, core *cliapp.ScenarioApp) {
	t.Helper()
	modes := cliapptest.RunPrimitiveHandlerModes(t, handler, schema, args, core)
	if modes.HumanErr != nil || modes.JSONErr != nil {
		t.Fatalf("primitive errors: human=%v json=%v", modes.HumanErr, modes.JSONErr)
	}
	if !strings.Contains(modes.Human, "Result:") && !strings.Contains(modes.Human, "Summary:") && !strings.Contains(modes.Human, "Status:") {
		t.Fatalf("missing human primitive report: %q", modes.Human)
	}
	var body any
	if err := json.Unmarshal([]byte(modes.JSON), &body); err != nil {
		t.Fatalf("invalid JSON output: %v (%q)", err, modes.JSON)
	}
}

func TestSystemPrimitivesUseTypedContracts(t *testing.T) {
	records := &fakeRecordsRPC{list: &domainv1.DesktopRecordsResponse{Records: []*domainv1.DesktopRecordWithBuild{{Record: &domainv1.DesktopRecord{Id: "record-1", ScenarioName: "demo"}}}}}
	systemRPC := &fakeSystemRPC{templates: &domainv1.ListTemplatesResponse{Templates: []*domainv1.TemplateInfo{{Type: "basic", Name: "Basic"}}}, template: &domainv1.TemplateConfigResponse{}, wine: &domainv1.WineCheckResponse{Installed: false, InstallMethods: []*domainv1.WineInstallMethod{{Id: "flatpak"}}}}
	operations := &fakeOperationsRPC{response: &domainv1.DesktopScenarioStatusResponse{Scenarios: []*domainv1.DesktopScenarioStatus{{Name: "demo", Built: true}}, Stats: &domainv1.DesktopScenarioStats{Total: 1, Built: 1}}}
	cmds := &Commands{records: records, system: systemRPC, operations: operations}
	assertPrimitiveModes(t, cmds.templatesListPrimitive(), cliapp.ArgSchema{}, nil, nil)
	assertPrimitiveModes(t, cmds.templateGetPrimitive(), cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "type", Required: true}}}, []string{"basic"}, nil)
	assertPrimitiveModes(t, cmds.recordsListPrimitive(), cliapp.ArgSchema{}, nil, nil)
	assertPrimitiveModes(t, cmds.recordsMovePrimitive(), cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "target", Default: "destination"}, {Name: "path"}}}, []string{"record-1", "--target", "custom", "--path", "/opt/apps"}, nil)
	if records.move.GetTarget() != "custom" || records.move.GetDestinationPath() != "/opt/apps" {
		t.Fatalf("move request = %#v", records.move)
	}
	assertPrimitiveModes(t, cmds.recordsDeletePrimitive(), cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}}}, []string{"demo"}, nil)
	if records.deleted.GetScenarioName() != "demo" {
		t.Fatalf("delete request = %#v", records.deleted)
	}
	assertPrimitiveModes(t, cmds.desktopStatusPrimitive(), cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "name"}}}, []string{"--name", "demo"}, nil)
	assertPrimitiveModes(t, cmds.wineCheckPrimitive(), cliapp.ArgSchema{}, nil, nil)
	assertPrimitiveModes(t, cmds.wineInstallPrimitive(), cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "method", Required: true}}}, []string{"--method", "flatpak"}, nil)
	if systemRPC.install.GetMethod() != "flatpak" {
		t.Fatalf("install request = %#v", systemRPC.install)
	}
	assertPrimitiveModes(t, cmds.wineStatusPrimitive(), cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "install_id", Required: true}}}, []string{"inst-001"}, nil)
	if systemRPC.status.GetInstallId() != "inst-001" {
		t.Fatalf("status request = %#v", systemRPC.status)
	}
}

func TestDownloadPrimitiveWritesPackagesAndUsesDeclaredPlatforms(t *testing.T) {
	deps := newTestDependencies(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/desktop/download/demo/linux" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("desktop-package"))
	}))
	cmds := &Commands{deps: deps}
	output := filepath.Join(t.TempDir(), "demo.AppImage")
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true}, {Name: "platform", Required: true}}, Flags: []cliapp.Flag{{Name: "output"}}}
	assertPrimitiveModes(t, cmds.downloadPrimitive(), schema, []string{"demo", "linux", "--output", output}, deps.Core())
	body, err := os.ReadFile(output)
	if err != nil || string(body) != "desktop-package" {
		t.Fatalf("download = %q, err=%v", body, err)
	}
}

func TestSystemCommandGroupsCarryOnlyPrimitiveHandlers(t *testing.T) {
	deps := newTestDependencies(t, http.NotFoundHandler())
	for _, group := range CommandGroups(deps) {
		for _, command := range group.Commands {
			if command.PrimitiveEvidence() == "" || command.Run != nil {
				t.Fatalf("command %q has non-primitive production path", command.Name)
			}
		}
	}
	for _, command := range WineRegister(deps).Subcommands {
		if command.PrimitiveEvidence() == "" || command.Run != nil {
			t.Fatalf("wine command %q has non-primitive production path", command.Name)
		}
	}
}
