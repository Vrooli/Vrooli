package channel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	connectrpc "connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/session"
)

// DesktopAdapter is the node-local authority boundary for a remote desktop
// session. The Bridge agent never injects native input itself; Device Control
// remains responsible for the active grant, lease epoch, policy and evidence.
type DesktopAdapter interface {
	Execute(context.Context, *sessionv1.Binding, *sessionv1.DesktopCommand) (*sessionv1.DesktopResult, error)
	Stop(context.Context, *sessionv1.Binding) error
}

// unixDesktopAdapter calls Device Control's private owner service over an
// explicitly configured Unix socket. Kernel peer credentials authorize the
// local caller; no bearer or helper token crosses the Bridge channel.
type unixDesktopAdapter struct {
	client desktopv1connect.DesktopOwnerServiceClient
	close  func()
}

func NewUnixDesktopAdapter(socket string) DesktopAdapter {
	socket = strings.TrimSpace(socket)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		if !filepath.IsAbs(socket) {
			return nil, errors.New("desktop owner socket must be an absolute path")
		}
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	return &unixDesktopAdapter{
		client: desktopv1connect.NewDesktopOwnerServiceClient(&http.Client{Transport: transport}, "http://desktop-owner", connectrpc.WithReadMaxBytes(33*1024*1024)),
		close:  transport.CloseIdleConnections,
	}
}

func (a *unixDesktopAdapter) Execute(ctx context.Context, binding *sessionv1.Binding, command *sessionv1.DesktopCommand) (*sessionv1.DesktopResult, error) {
	if a == nil || a.client == nil {
		return nil, errors.New("desktop owner adapter is not configured")
	}
	ref, err := desktopSessionRef(binding)
	if err != nil {
		return nil, err
	}
	if command == nil || strings.TrimSpace(command.GetCommandId()) == "" {
		return nil, errors.New("desktop command id is required")
	}
	result := &sessionv1.DesktopResult{CommandId: command.GetCommandId(), RemoteCommandId: remoteCommandID(binding, command.GetCommandId())}
	switch command.GetOperation() {
	case sessionv1.DesktopCommandOperation_DESKTOP_COMMAND_OPERATION_OBSERVE:
		response, callErr := a.client.Observe(ctx, connectrpc.NewRequest(&desktopv1.OwnerObserveRequest{
			Session:   ref,
			ProcessId: command.GetProcessId(), ApplicationId: command.GetApplicationId(), ApplicationRevision: command.GetApplicationRevision(),
		}))
		if callErr != nil {
			return result, callErr
		}
		if response == nil || response.Msg == nil {
			return result, errors.New("desktop owner returned no observation")
		}
		result.Observation = response.Msg
	case sessionv1.DesktopCommandOperation_DESKTOP_COMMAND_OPERATION_ACT:
		if command.GetAction() == nil {
			return result, errors.New("desktop act command requires an action")
		}
		response, callErr := a.client.Act(ctx, connectrpc.NewRequest(&desktopv1.OwnerActRequest{
			Session: ref, CommandId: command.GetCommandId(), GeometryRevision: command.GetGeometryRevision(), Action: command.GetAction(),
		}))
		if callErr != nil {
			return result, callErr
		}
		if response == nil || response.Msg == nil {
			return result, errors.New("desktop owner returned no act receipt")
		}
		result.Act = response.Msg
	case sessionv1.DesktopCommandOperation_DESKTOP_COMMAND_OPERATION_STOP:
		if _, callErr := a.client.Stop(ctx, connectrpc.NewRequest(&desktopv1.OwnerStopRequest{Session: ref})); callErr != nil {
			return result, callErr
		}
	default:
		return result, errors.New("desktop command operation is unsupported")
	}
	return result, nil
}

func (a *unixDesktopAdapter) Stop(ctx context.Context, binding *sessionv1.Binding) error {
	if a == nil || a.client == nil {
		return errors.New("desktop owner adapter is not configured")
	}
	ref, err := desktopSessionRef(binding)
	if err != nil {
		return err
	}
	_, err = a.client.Stop(ctx, connectrpc.NewRequest(&desktopv1.OwnerStopRequest{Session: ref}))
	return err
}

func desktopSessionRef(binding *sessionv1.Binding) (*commonv1.SessionRef, error) {
	if binding == nil || binding.GetSurface() == nil || strings.TrimSpace(binding.GetLeaseId()) == "" || binding.GetLeaseEpoch() == 0 {
		return nil, errors.New("desktop binding requires surface, lease id, and lease epoch")
	}
	return &commonv1.SessionRef{Surface: binding.GetSurface(), SessionId: binding.GetLeaseId()}, nil
}

func remoteCommandID(binding *sessionv1.Binding, commandID string) string {
	lease := "unknown"
	if binding != nil && binding.GetLeaseId() != "" {
		lease = binding.GetLeaseId()
	}
	return fmt.Sprintf("%s:%s", lease, commandID)
}
