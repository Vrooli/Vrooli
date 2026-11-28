package devices

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	clitest "github.com/vrooli/cli-core/cliapptest"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory/inventory_v1connect"
)

func TestListShowsStableIdentityAndConfidenceNotes(t *testing.T) {
	service := &fakeInventoryService{listResponse: &inventoryv1.ListDevicesResponse{Devices: []*inventoryv1.Device{{
		Id: "device-12", Hostname: "living-room", IpAddress: "192.0.2.12", MacAddress: "02:00:00:00:00:12",
		Group: "trusted", IdentityConfidence: "medium", Notes: []string{"randomized address", "hostname match only"},
	}}}}
	core := newDeviceTestApp(t, service)
	h := newHandlers(core)
	var out bytes.Buffer
	err := h.list(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Stdout: &out,
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "group"}}},
		Flags:  map[string]string{"group": "trusted"},
	}))
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}
	if service.gotGroup != "trusted" {
		t.Fatalf("group filter = %q, want trusted", service.gotGroup)
	}
	for _, want := range []string{"device-12", "host=living-room", "ip=192.0.2.12", "mac=02:00:00:00:00:12", "group=trusted", "confidence=medium", `notes="randomized address; hostname match only"`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("device output omitted %q: %s", want, out.String())
		}
	}
}

func TestRefreshForwardsDryRunAndRendersFindings(t *testing.T) {
	service := &fakeInventoryService{refreshResponse: &inventoryv1.RefreshInventoryResponse{
		Devices:  []*inventoryv1.Device{{Id: "device-12", MacAddress: "02:00:00:00:00:12"}},
		Findings: []string{"identity is ambiguous"},
	}}
	core := newDeviceTestApp(t, service)
	h := newHandlers(core)
	var out bytes.Buffer
	err := h.refresh(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Stdout: &out,
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "dry-run"}}},
		Flags:  map[string]string{"dry-run": "true"},
	}))
	if err != nil {
		t.Fatalf("refresh inventory: %v", err)
	}
	if !service.gotDryRun {
		t.Fatal("refresh did not forward dry-run")
	}
	for _, want := range []string{"Inventory refresh complete.", "device-12", "mac=02:00:00:00:00:12", "identity is ambiguous"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("refresh output omitted %q: %s", want, out.String())
		}
	}
}

func TestGroupForwardsDeviceAndGroupAndShowsReturnedDevice(t *testing.T) {
	service := &fakeInventoryService{updateResponse: &inventoryv1.UpdateDeviceGroupResponse{Device: &inventoryv1.Device{
		Id: "device-12", Group: "trusted", IdentityConfidence: "high",
	}}}
	core := newDeviceTestApp(t, service)
	h := newHandlers(core)
	var out bytes.Buffer
	err := h.group(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Stdout: &out,
		Schema:      cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "group"}}},
		Positionals: map[string]string{"id": "device-12"}, Flags: map[string]string{"group": "trusted"},
	}))
	if err != nil {
		t.Fatalf("update device group: %v", err)
	}
	if service.gotID != "device-12" || service.gotGroup != "trusted" {
		t.Fatalf("update request id=%q group=%q", service.gotID, service.gotGroup)
	}
	if !strings.Contains(out.String(), "device-12 host= ip= mac= group=trusted confidence=high") {
		t.Fatalf("mutation output omitted updated device: %s", out.String())
	}
}

func TestExplainForwardsIdentityAndRendersEvidenceAndDeviceNotes(t *testing.T) {
	service := &fakeInventoryService{explainResponse: &inventoryv1.ExplainDeviceIdentityResponse{
		Device:   &inventoryv1.Device{Id: "device-12", Notes: []string{"randomized address"}},
		Evidence: []string{"hostname matched once"},
	}}
	core := newDeviceTestApp(t, service)
	h := newHandlers(core)
	var out bytes.Buffer
	err := h.explain(cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Core: core, Stdout: &out,
		Schema:      cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}},
		Positionals: map[string]string{"id": "device-12"},
	}))
	if err != nil {
		t.Fatalf("explain device identity: %v", err)
	}
	if service.gotID != "device-12" {
		t.Fatalf("explain id = %q", service.gotID)
	}
	for _, want := range []string{`notes="randomized address"`, "hostname matched once"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("identity explanation omitted %q: %s", want, out.String())
		}
	}
}

func newDeviceTestApp(t *testing.T, service inventoryconnect.InventoryServiceHandler) *cliapp.ScenarioApp {
	t.Helper()
	path, handler := inventoryconnect.NewInventoryServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return clitest.NewTestApp(t, mux)
}

type fakeInventoryService struct {
	inventoryconnect.UnimplementedInventoryServiceHandler
	listResponse    *inventoryv1.ListDevicesResponse
	refreshResponse *inventoryv1.RefreshInventoryResponse
	updateResponse  *inventoryv1.UpdateDeviceGroupResponse
	explainResponse *inventoryv1.ExplainDeviceIdentityResponse
	gotGroup        string
	gotID           string
	gotDryRun       bool
}

func (f *fakeInventoryService) RefreshInventory(_ context.Context, req *connect.Request[inventoryv1.RefreshInventoryRequest]) (*connect.Response[inventoryv1.RefreshInventoryResponse], error) {
	f.gotDryRun = req.Msg.GetDryRun()
	return connect.NewResponse(f.refreshResponse), nil
}

func (f *fakeInventoryService) ListDevices(_ context.Context, req *connect.Request[inventoryv1.ListDevicesRequest]) (*connect.Response[inventoryv1.ListDevicesResponse], error) {
	f.gotGroup = req.Msg.GetGroup()
	return connect.NewResponse(f.listResponse), nil
}

func (f *fakeInventoryService) UpdateDeviceGroup(_ context.Context, req *connect.Request[inventoryv1.UpdateDeviceGroupRequest]) (*connect.Response[inventoryv1.UpdateDeviceGroupResponse], error) {
	f.gotID = req.Msg.GetId()
	f.gotGroup = req.Msg.GetGroup()
	return connect.NewResponse(f.updateResponse), nil
}

func (f *fakeInventoryService) ExplainDeviceIdentity(_ context.Context, req *connect.Request[inventoryv1.ExplainDeviceIdentityRequest]) (*connect.Response[inventoryv1.ExplainDeviceIdentityResponse], error) {
	f.gotID = req.Msg.GetId()
	return connect.NewResponse(f.explainResponse), nil
}
