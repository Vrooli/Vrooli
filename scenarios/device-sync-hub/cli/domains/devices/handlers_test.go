package devices

import (
	"context"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/devices/devices_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "device-sync-hub/cli/internal/testutil"
)

// devicesService is a hand-rolled fake DevicesService that records inputs and
// returns canned responses/errors per RPC.
type devicesService struct {
	mu sync.Mutex

	setupResp   *devicesv1.SetupOwnerDeviceResponse
	listResp    *devicesv1.ListDevicesResponse
	getResp     *devicesv1.GetDeviceResponse
	issueResp   *devicesv1.IssuePairingCodeResponse
	redeemResp  *devicesv1.RedeemPairingCodeResponse
	requestResp *devicesv1.RequestPairingResponse
	approveResp *devicesv1.ApprovePairingResponse
	renameResp  *devicesv1.RenameDeviceResponse
	revokeResp  *devicesv1.RevokeDeviceResponse

	listErr   error
	getErr    error
	revokeErr error

	setupInputs   []*devicesv1.DeviceProfile
	redeemInputs  []*devicesv1.RedeemPairingCodeRequest
	approveInputs []string
	revokeInputs  []string
}

func (s *devicesService) SetupOwnerDevice(_ context.Context, req *connect.Request[devicesv1.SetupOwnerDeviceRequest]) (*connect.Response[devicesv1.SetupOwnerDeviceResponse], error) {
	s.mu.Lock()
	s.setupInputs = append(s.setupInputs, req.Msg.Profile)
	s.mu.Unlock()
	if s.setupResp == nil {
		s.setupResp = &devicesv1.SetupOwnerDeviceResponse{}
	}
	return connect.NewResponse(s.setupResp), nil
}

func (s *devicesService) ListDevices(context.Context, *connect.Request[devicesv1.ListDevicesRequest]) (*connect.Response[devicesv1.ListDevicesResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp == nil {
		s.listResp = &devicesv1.ListDevicesResponse{}
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *devicesService) GetDevice(_ context.Context, req *connect.Request[devicesv1.GetDeviceRequest]) (*connect.Response[devicesv1.GetDeviceResponse], error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	if s.getResp == nil {
		s.getResp = &devicesv1.GetDeviceResponse{}
	}
	return connect.NewResponse(s.getResp), nil
}

func (s *devicesService) IssuePairingCode(context.Context, *connect.Request[devicesv1.IssuePairingCodeRequest]) (*connect.Response[devicesv1.IssuePairingCodeResponse], error) {
	if s.issueResp == nil {
		s.issueResp = &devicesv1.IssuePairingCodeResponse{}
	}
	return connect.NewResponse(s.issueResp), nil
}

func (s *devicesService) RedeemPairingCode(_ context.Context, req *connect.Request[devicesv1.RedeemPairingCodeRequest]) (*connect.Response[devicesv1.RedeemPairingCodeResponse], error) {
	s.mu.Lock()
	s.redeemInputs = append(s.redeemInputs, req.Msg)
	s.mu.Unlock()
	if s.redeemResp == nil {
		s.redeemResp = &devicesv1.RedeemPairingCodeResponse{}
	}
	return connect.NewResponse(s.redeemResp), nil
}

func (s *devicesService) RequestPairing(context.Context, *connect.Request[devicesv1.RequestPairingRequest]) (*connect.Response[devicesv1.RequestPairingResponse], error) {
	if s.requestResp == nil {
		s.requestResp = &devicesv1.RequestPairingResponse{}
	}
	return connect.NewResponse(s.requestResp), nil
}

func (s *devicesService) ApprovePairing(_ context.Context, req *connect.Request[devicesv1.ApprovePairingRequest]) (*connect.Response[devicesv1.ApprovePairingResponse], error) {
	s.mu.Lock()
	s.approveInputs = append(s.approveInputs, req.Msg.DeviceId)
	s.mu.Unlock()
	if s.approveResp == nil {
		s.approveResp = &devicesv1.ApprovePairingResponse{}
	}
	return connect.NewResponse(s.approveResp), nil
}

func (s *devicesService) RenameDevice(context.Context, *connect.Request[devicesv1.RenameDeviceRequest]) (*connect.Response[devicesv1.RenameDeviceResponse], error) {
	if s.renameResp == nil {
		s.renameResp = &devicesv1.RenameDeviceResponse{}
	}
	return connect.NewResponse(s.renameResp), nil
}

func (s *devicesService) RevokeDevice(_ context.Context, req *connect.Request[devicesv1.RevokeDeviceRequest]) (*connect.Response[devicesv1.RevokeDeviceResponse], error) {
	s.mu.Lock()
	s.revokeInputs = append(s.revokeInputs, req.Msg.DeviceId)
	s.mu.Unlock()
	if s.revokeErr != nil {
		return nil, s.revokeErr
	}
	if s.revokeResp == nil {
		s.revokeResp = &devicesv1.RevokeDeviceResponse{}
	}
	return connect.NewResponse(s.revokeResp), nil
}

func connectAPI(t *testing.T, svc *devicesService) http.Handler {
	t.Helper()
	path, handler := devicesconnect.NewDevicesServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func device(id, name string, state devicesv1.TrustState, online bool) *devicesv1.Device {
	ts := timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return &devicesv1.Device{
		Id:         id,
		OwnerId:    "owner-1",
		Name:       name,
		Kind:       "laptop",
		TrustState: state,
		Online:     online,
		CreatedAt:  ts,
		UpdatedAt:  ts,
	}
}

func TestDevicesList_RendersResults(t *testing.T) {
	svc := &devicesService{listResp: &devicesv1.ListDevicesResponse{Devices: []*devicesv1.Device{
		device("a", "Phone", devicesv1.TrustState_TRUST_STATE_TRUSTED, true),
		device("b", "Tablet", devicesv1.TrustState_TRUST_STATE_PENDING, false),
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	require.NoError(t, h.list(ctx))
	require.Contains(t, out.String(), "Found 2 device(s)")
	require.Contains(t, out.String(), "Phone")
	require.Contains(t, out.String(), "TRUSTED")
	require.Contains(t, out.String(), "PENDING")
	require.Contains(t, out.String(), "online")
}

func TestDevicesList_JSONIsProtoWireShape(t *testing.T) {
	svc := &devicesService{listResp: &devicesv1.ListDevicesResponse{Devices: []*devicesv1.Device{
		device("a", "Phone", devicesv1.TrustState_TRUST_STATE_TRUSTED, true),
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{JSON: true})

	require.NoError(t, h.list(ctx))
	body := out.String()
	require.NotContains(t, body, "summary", "--json must be proto wire shape")
	require.NotContains(t, body, "retrieval_hints", "--json must be proto wire shape")

	var got devicesv1.ListDevicesResponse
	require.NoError(t, protojson.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Devices, 1)
	require.Equal(t, "a", got.Devices[0].Id)
}

func TestDevicesList_SurfacesConnectErrors(t *testing.T) {
	svc := &devicesService{listErr: connect.NewError(connect.CodeInternal, io.ErrUnexpectedEOF)}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})

	err := h.list(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal")
}

// [REQ:REQ-P0-003] `devices setup` admits this machine to the trust group as the owner's first device.
func TestDevicesSetup_SendsProfileAndPrintsToken(t *testing.T) {
	svc := &devicesService{setupResp: &devicesv1.SetupOwnerDeviceResponse{
		Device:      device("owner-dev", "Workstation", devicesv1.TrustState_TRUST_STATE_TRUSTED, true),
		DeviceToken: "hub-owner-token",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "device-name", Aliases: []string{"name"}}, {Name: "kind"}, {Name: "platform"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{
		"device-name": "Workstation", "kind": "laptop", "platform": "linux",
	}})

	require.NoError(t, h.setup(ctx))
	require.Len(t, svc.setupInputs, 1)
	require.Equal(t, "Workstation", svc.setupInputs[0].DeviceName)
	require.Equal(t, "laptop", svc.setupInputs[0].Kind)
	require.Contains(t, out.String(), "hub-owner-token")
	require.Contains(t, out.String(), "DEVICE_SYNC_HUB_DEVICE_TOKEN")
	require.Contains(t, out.String(), "TRUSTED")
}

func TestDevicesPair_PrintsCode(t *testing.T) {
	expires := timestamppb.New(time.Date(2026, 1, 1, 0, 15, 0, 0, time.UTC))
	svc := &devicesService{issueResp: &devicesv1.IssuePairingCodeResponse{PairingCode: &devicesv1.PairingCode{
		Code: "ABCDE-FGHIJ", OwnerId: "owner-1", ExpiresAt: expires,
	}}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "name"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{"name": "My Phone"}})

	require.NoError(t, h.pair(ctx))
	require.Contains(t, out.String(), "ABCDE-FGHIJ")
	require.Contains(t, out.String(), "single-use pairing code")
}

func TestDevicesRedeem_SendsProfileAndPrintsToken(t *testing.T) {
	svc := &devicesService{redeemResp: &devicesv1.RedeemPairingCodeResponse{
		Device:      device("new", "Tablet", devicesv1.TrustState_TRUST_STATE_TRUSTED, true),
		DeviceToken: "hub-token-xyz",
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Flags: []cliapp.Flag{{Name: "code"}, {Name: "device-name", Aliases: []string{"name"}}, {Name: "kind"}, {Name: "platform"}},
	}, cliapptest.TestRunContextOptions{Flags: map[string]string{
		"code": "ABCDE-FGHIJ", "device-name": "Tablet", "kind": "tablet", "platform": "android",
	}})

	require.NoError(t, h.redeem(ctx))
	require.Len(t, svc.redeemInputs, 1)
	require.Equal(t, "ABCDE-FGHIJ", svc.redeemInputs[0].Code)
	require.Equal(t, "Tablet", svc.redeemInputs[0].Profile.DeviceName)
	require.Equal(t, "android", svc.redeemInputs[0].Profile.Platform)
	require.Contains(t, out.String(), "hub-token-xyz")
	require.Contains(t, out.String(), "DEVICE_SYNC_HUB_DEVICE_TOKEN")
}

func TestDevicesApprove_CallsServer(t *testing.T) {
	svc := &devicesService{approveResp: &devicesv1.ApprovePairingResponse{
		Device: device("pend", "Tablet", devicesv1.TrustState_TRUST_STATE_TRUSTED, false),
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "device-id", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"device-id": "pend"}})

	require.NoError(t, h.approve(ctx))
	require.Equal(t, []string{"pend"}, svc.approveInputs)
	require.Contains(t, out.String(), "now TRUSTED")
}

func TestDevicesRevoke_CallsServer(t *testing.T) {
	svc := &devicesService{revokeResp: &devicesv1.RevokeDeviceResponse{
		Device: device("gone", "Old", devicesv1.TrustState_TRUST_STATE_REVOKED, false),
	}}
	core := clitest.NewTestApp(t, connectAPI(t, svc))
	h := newHandlers(core)
	ctx, out := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "device-id", Required: true}},
	}, cliapptest.TestRunContextOptions{Positionals: map[string]string{"device-id": "gone"}})

	require.NoError(t, h.revoke(ctx))
	require.Equal(t, []string{"gone"}, svc.revokeInputs)
	require.Contains(t, out.String(), "access severed")
	require.Contains(t, out.String(), "REVOKED")
}

func registerForTest(t *testing.T, core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	t.Helper()
	manifest := readDevicesManifest(t)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	return group
}

func TestDevicesApprove_RequiresID(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &devicesService{}))
	group := registerForTest(t, core)
	var approve cliapp.Command
	for _, sc := range group.Subcommands {
		if sc.Name == "approve" {
			approve = sc
		}
	}
	require.NotNil(t, approve.RunCtx)
	_, err := cliapptest.NewTestRunContextFromArgs(approve.Args, []string{}, core, nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required positional")
}

func TestRegister_Wiring(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(t, &devicesService{}))
	group := registerForTest(t, core)

	require.Equal(t, "devices", group.Name)
	require.True(t, group.NeedsAPI)
	names := make([]string, 0, len(group.Subcommands))
	for _, sc := range group.Subcommands {
		names = append(names, sc.Name)
		require.NotNil(t, sc.RunCtx, "subcommand %s should use RunCtx", sc.Name)
	}
	require.ElementsMatch(t, []string{"setup", "list", "get", "pair", "redeem", "request", "approve", "rename", "revoke"}, names)
}
