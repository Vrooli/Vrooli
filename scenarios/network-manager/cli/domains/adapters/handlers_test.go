package adapters

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters"
	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters/adapters_v1connect"
)

func TestCapabilitiesShowsSupportAndSafetyMetadataBeforeOfferingAction(t *testing.T) {
	core := newAdapterTestApp(t, &fakeAdapterService{
		listResponse: &adaptersv1.ListCapabilitiesResponse{Capabilities: []*adaptersv1.Capability{{
			Adapter: "manual", Action: "router_dns_enforcement", Supported: false,
			RequiresAdmin: true, RollbackSupported: false,
			Reason: "no router adapter selected",
		}}},
	})
	h := newHandlers(core)
	var out bytes.Buffer

	if err := h.capabilities(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, Stdout: &out})); err != nil {
		t.Fatalf("capabilities: %v", err)
	}

	for _, want := range []string{
		"manual action=router_dns_enforcement supported=false admin=true rollback=false reason=no router adapter selected",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("capability output omitted safety metadata %q: %s", want, out.String())
		}
	}
}

func TestExplainUnsupportedActionForwardsActionAndShowsManualSteps(t *testing.T) {
	service := &fakeAdapterService{explainResponse: &adaptersv1.ExplainUnsupportedActionResponse{
		Capability:  &adaptersv1.Capability{Adapter: "manual", Action: "router_dns_enforcement", Supported: false, Reason: "no router adapter selected"},
		ManualSteps: []string{"Review router DNS settings.", "Use a supported adapter before applying changes."},
	}}
	core := newAdapterTestApp(t, service)
	h := newHandlers(core)
	var out bytes.Buffer
	err := h.explain(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Stdout: &out,
		Schema:      cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action", Required: true}}},
		Positionals: map[string]string{"action": "router_dns_enforcement"},
	}))
	if err != nil {
		t.Fatalf("explain: %v", err)
	}

	if service.gotAction != "router_dns_enforcement" {
		t.Fatalf("requested action = %q", service.gotAction)
	}
	for _, want := range []string{"supported=false", "reason=no router adapter selected", "Review router DNS settings.", "Use a supported adapter before applying changes."} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("explanation omitted %q: %s", want, out.String())
		}
	}
}

func TestPlatformSummaryShowsDetectedIdentityAndNotes(t *testing.T) {
	core := newAdapterTestApp(t, &fakeAdapterService{platformResponse: &adaptersv1.GetPlatformSummaryResponse{Summary: &adaptersv1.PlatformSummary{
		Os: "linux", Arch: "arm64", Profile: "router-pilot", Notes: []string{"router enforcement remains manual"},
	}}})
	h := newHandlers(core)
	var out bytes.Buffer

	if err := h.platform(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Core: core, Stdout: &out})); err != nil {
		t.Fatalf("platform: %v", err)
	}
	for _, want := range []string{"os=linux arch=arm64 profile=router-pilot", "router enforcement remains manual"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("platform output omitted %q: %s", want, out.String())
		}
	}
}

func TestRegisterBindsManifestCapabilityCommandToLiveAdapterHandler(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read CLI manifest: %v", err)
	}
	core := newAdapterTestApp(t, &fakeAdapterService{listResponse: &adaptersv1.ListCapabilitiesResponse{Capabilities: []*adaptersv1.Capability{{
		Adapter: "router-pilot", Action: "router_dns_enforcement", Supported: true,
		RequiresAdmin: true, RollbackSupported: true,
	}}}})
	group, err := Register(core, manifest)
	if err != nil {
		t.Fatalf("register adapters group: %v", err)
	}
	var command *cliapp.Command
	for i := range group.Subcommands {
		if group.Subcommands[i].Name == "capabilities" {
			command = &group.Subcommands[i]
			break
		}
	}
	if command == nil || command.RunCtx == nil {
		t.Fatalf("manifest did not bind the capabilities command: %+v", group.Subcommands)
	}

	var out bytes.Buffer
	err = command.RunCtx(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Schema: command.Args, Stdout: &out,
	}))
	if err != nil {
		t.Fatalf("run registered capabilities command: %v", err)
	}
	for _, want := range []string{"router-pilot action=router_dns_enforcement supported=true admin=true rollback=true"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("registered command output omitted %q: %s", want, out.String())
		}
	}
}

func TestRegisterFailsClosedWithAdapterContextWhenManifestGroupIsMissing(t *testing.T) {
	core := newAdapterTestApp(t, &fakeAdapterService{})
	group, err := Register(core, []byte(`{"name":"network-manager","groups":[]}`))
	if err == nil || !strings.Contains(err.Error(), "adapters: load from manifest") {
		t.Fatalf("expected adapter registration context for missing manifest group, got %v", err)
	}
	if group.Name != "" || len(group.Subcommands) != 0 {
		t.Fatalf("failed registration returned a partial command group: %+v", group)
	}
}

func newAdapterTestApp(t *testing.T, service adaptersconnect.AdapterServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, handler := adaptersconnect.NewAdapterServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return clitest.NewTestApp(t, mux)
}

type fakeAdapterService struct {
	adaptersconnect.UnimplementedAdapterServiceHandler
	listResponse     *adaptersv1.ListCapabilitiesResponse
	explainResponse  *adaptersv1.ExplainUnsupportedActionResponse
	platformResponse *adaptersv1.GetPlatformSummaryResponse
	gotAction        string
}

func (f *fakeAdapterService) ListCapabilities(context.Context, *connect.Request[adaptersv1.ListCapabilitiesRequest]) (*connect.Response[adaptersv1.ListCapabilitiesResponse], error) {
	return connect.NewResponse(f.listResponse), nil
}

func (f *fakeAdapterService) ExplainUnsupportedAction(_ context.Context, req *connect.Request[adaptersv1.ExplainUnsupportedActionRequest]) (*connect.Response[adaptersv1.ExplainUnsupportedActionResponse], error) {
	f.gotAction = req.Msg.GetAction()
	return connect.NewResponse(f.explainResponse), nil
}

func (f *fakeAdapterService) GetPlatformSummary(context.Context, *connect.Request[adaptersv1.GetPlatformSummaryRequest]) (*connect.Response[adaptersv1.GetPlatformSummaryResponse], error) {
	return connect.NewResponse(f.platformResponse), nil
}
