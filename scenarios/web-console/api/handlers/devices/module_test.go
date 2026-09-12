package devices

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/devices"
)

type fakeService struct {
	rows       []Device
	listSelf   string
	disconnect struct {
		deviceID, connectionID string
	}
}

func (f *fakeService) List(_ context.Context, self string) ([]Device, error) {
	f.listSelf = self
	for i := range f.rows {
		f.rows[i].IsSelf = self != "" && f.rows[i].ID == self
	}
	return f.rows, nil
}

func (f *fakeService) Disconnect(_ context.Context, deviceID, connectionID string) (int, error) {
	f.disconnect.deviceID, f.disconnect.connectionID = deviceID, connectionID
	return 1, nil
}

func (f *fakeService) GiveControl(_ context.Context, deviceID, sessionID string) (bool, error) {
	return deviceID != "" && sessionID != "", nil
}

func TestDeviceService_ListGroupsConnectionsByDevice(t *testing.T) {
	fake := &fakeService{rows: []Device{{
		ID: "phone-1", Label: "Phone", Class: "phone", ConnectionCount: 2,
		FirstSeenUnix: 1_724_851_200, Reconnecting: true,
		Sessions: []SessionAttachment{{SessionID: "session-1", SessionName: "Shell", HoldsLease: true}},
	}}}
	h := NewConnectHandler(Deps{Service: fake})
	got, err := h.List(context.Background(), connect.NewRequest(&devicesv1.ListRequest{SelfDeviceId: "desktop-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if fake.listSelf != "desktop-1" || len(got.Msg.GetDevices()) != 1 {
		t.Fatalf("list = %#v, self = %q", got.Msg.GetDevices(), fake.listSelf)
	}
	device := got.Msg.GetDevices()[0]
	if device.GetConnectionCount() != 2 || !device.GetReconnecting() || len(device.GetSessions()) != 1 || device.GetSessions()[0].GetSessionName() != "Shell" {
		t.Fatalf("device projection = %#v", device)
	}
	if device.GetFirstSeenAt().GetSeconds() != 1_724_851_200 {
		t.Fatalf("first seen = %v", device.GetFirstSeenAt())
	}
}

func TestDeviceService_ListMarksTheCallersOwnDevice(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{rows: []Device{{ID: "phone-1"}, {ID: "desktop-1"}}}})
	got, err := h.List(context.Background(), connect.NewRequest(&devicesv1.ListRequest{SelfDeviceId: "desktop-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetDevices()[0].GetIsSelf() || !got.Msg.GetDevices()[1].GetIsSelf() {
		t.Fatalf("self markers = %#v", got.Msg.GetDevices())
	}
}

func TestDeviceService_DisconnectRefusesToCloseTheCaller(t *testing.T) {
	fake := &fakeService{}
	h := NewConnectHandler(Deps{Service: fake})
	request := connect.NewRequest(&devicesv1.DisconnectRequest{DeviceId: "desktop-1", ConnectionId: "conn-1"})
	request.Header().Set("X-Vrooli-Device-Id", "desktop-1")
	if _, err := h.Disconnect(context.Background(), request); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("error = %v, code = %v", err, connect.CodeOf(err))
	}
	if fake.disconnect.deviceID != "" {
		t.Fatal("self disconnect reached the service")
	}
}

func TestDeviceService_GiveControlRequiresDevice(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &fakeService{}})
	if _, err := h.GiveControl(context.Background(), connect.NewRequest(&devicesv1.GiveControlRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error = %v, code = %v", err, connect.CodeOf(err))
	}
}
