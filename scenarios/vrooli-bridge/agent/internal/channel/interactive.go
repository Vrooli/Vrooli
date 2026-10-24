package channel

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	desktopconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	presenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence/presence_v1connect"
	"vrooli-bridge/agent/internal/nodecred"
)

var (
	ErrInteractiveGrant   = errors.New("invalid interactive grant")
	ErrInteractiveExpired = errors.New("interactive grant expired")
)

const maxInteractiveSignalBytes = 256 * 1024

const (
	desktopCompanionPortEnv     = "DEVICE_CONTROL_COMPANION_PORT"
	defaultDesktopCompanionPort = 16465
)

// InteractiveSignalAdapter is the node-local companion boundary. The agent
// authenticates and bounds the Bridge grant, then delegates SDP/ICE to Device
// Control; it never parses media or owns desktop semantics.
type InteractiveSignalAdapter interface {
	HandleInteractiveSignal(context.Context, *channelv1.InteractiveSignal) (*channelv1.InteractiveSignalResponse, error)
}

type InteractiveRevokeAdapter interface {
	HandleInteractiveRevoke(context.Context, *channelv1.InteractiveRevoke) error
}

type InteractiveSignalReporter interface {
	ReportInteractiveSignalResponse(context.Context, *channelv1.InteractiveSignalResponse) error
}

type presenceInteractiveReporter struct {
	rpc    presenceconnect.PresenceServiceClient
	cred   *nodecred.Credential
	nodeID string
	now    func() time.Time
}

func (r presenceInteractiveReporter) ReportInteractiveSignalResponse(ctx context.Context, response *channelv1.InteractiveSignalResponse) error {
	req := connect.NewRequest(&presencev1.ReportInteractiveSignalResponseRequest{Response: response})
	if r.cred != nil {
		now := time.Now
		if r.now != nil {
			now = r.now
		}
		for k, v := range r.cred.Headers(r.nodeID, now().UTC()) {
			req.Header().Set(k, v)
		}
	}
	_, err := r.rpc.ReportInteractiveSignalResponse(ctx, req)
	return err
}

// localDeviceControlInteractiveAdapter is the node-local hop. The Bridge
// agent resolves the local Device Control API through the same scenario-port
// ladder used by other node-local scenario requests; no node URL or bearer
// credential crosses the owner-facing Bridge API.
type localDeviceControlInteractiveAdapter struct {
	httpClient  *http.Client
	resolvePort func(context.Context, string) (int, error)
}

func resolveDesktopCompanionPort(context.Context, string) (int, error) {
	value := strings.TrimSpace(os.Getenv(desktopCompanionPortEnv))
	if value == "" {
		return defaultDesktopCompanionPort, nil
	}
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid %s %q", desktopCompanionPortEnv, value)
	}
	return port, nil
}

func (a localDeviceControlInteractiveAdapter) HandleInteractiveSignal(ctx context.Context, signal *channelv1.InteractiveSignal) (*channelv1.InteractiveSignalResponse, error) {
	if signal == nil || strings.TrimSpace(signal.GetSessionId()) == "" || strings.TrimSpace(signal.GetLeaseId()) == "" {
		return nil, errors.New("interactive signal is missing the Device Control session binding")
	}
	if a.resolvePort == nil {
		return nil, errors.New("local Device Control port resolver is unavailable")
	}
	port, err := a.resolvePort(ctx, "device-control-companion")
	if err != nil {
		return nil, err
	}
	client := desktopconnect.NewDesktopSessionServiceClient(a.httpClient, fmt.Sprintf("http://127.0.0.1:%d", port))
	response, err := client.Signal(ctx, connect.NewRequest(&desktopv1.DesktopSignalRequest{
		Session: &commonv1.SessionRef{SessionId: signal.GetSessionId()}, LeaseId: signal.GetLeaseId(), LeaseEpoch: signal.GetLeaseEpoch(), Kind: desktopv1.DesktopSignalKind(signal.GetKind()), Generation: signal.GetGeneration(), Payload: append([]byte(nil), signal.GetPayload()...), IceServers: iceServers(signal.GetRoutes()),
	}))
	if err != nil {
		return nil, err
	}
	return &channelv1.InteractiveSignalResponse{Kind: uint32(response.Msg.GetKind()), Generation: response.Msg.GetGeneration(), Payload: append([]byte(nil), response.Msg.GetPayload()...), Accepted: response.Msg.GetAccepted(), ReasonCode: response.Msg.GetReasonCode()}, nil
}

func (a localDeviceControlInteractiveAdapter) HandleInteractiveRevoke(ctx context.Context, revoke *channelv1.InteractiveRevoke) error {
	if revoke == nil || strings.TrimSpace(revoke.GetSessionId()) == "" || strings.TrimSpace(revoke.GetLeaseId()) == "" || revoke.GetLeaseEpoch() == 0 {
		return errors.New("interactive revoke is missing the Device Control session binding")
	}
	if a.resolvePort == nil {
		return errors.New("local Device Control port resolver is unavailable")
	}
	port, err := a.resolvePort(ctx, "device-control-companion")
	if err != nil {
		return err
	}
	client := desktopconnect.NewDesktopSessionServiceClient(a.httpClient, fmt.Sprintf("http://127.0.0.1:%d", port))
	_, err = client.CloseSession(ctx, connect.NewRequest(&desktopv1.CloseSessionRequest{Session: &commonv1.SessionRef{SessionId: revoke.GetSessionId()}, Reason: revoke.GetReason()}))
	return err
}

func iceServers(routes []*interactivev1.RouteCandidate) []*desktopv1.IceServer {
	servers := make([]*desktopv1.IceServer, 0, len(routes))
	for _, route := range routes {
		if route == nil || strings.TrimSpace(route.GetUrl()) == "" {
			continue
		}
		if route.GetKind() != interactivev1.RouteKind_ROUTE_KIND_TURN {
			continue
		}
		servers = append(servers, &desktopv1.IceServer{Url: route.GetUrl(), Username: route.GetUsername(), Credential: route.GetCredential()})
		if len(servers) == 8 {
			break
		}
	}
	return servers
}

// VerifyInteractiveSignal is the node-agent admission fence. It validates the
// Bridge-issued grant before forwarding signaling to the local companion; it
// does not inspect or implement desktop pixels/input.
func VerifyInteractiveSignal(grant *interactivev1.ChannelGrant, signal *interactivev1.SignalRequest, now time.Time) error {
	if grant == nil || signal == nil || signal.GetGrant() == nil || strings.TrimSpace(grant.GetChannelId()) == "" || grant.GetProtocol() != interactivev1.ChannelProtocol_CHANNEL_PROTOCOL_WEBRTC_VP8 || signal.GetGrant().GetChannelId() != grant.GetChannelId() || signal.GetGrant().GetLeaseEpoch() != grant.GetLeaseEpoch() {
		return ErrInteractiveGrant
	}
	if grant.GetExpiresAt() == nil || !now.Before(grant.GetExpiresAt().AsTime()) {
		return ErrInteractiveExpired
	}
	return nil
}

func (c *Client) handleInteractiveSignal(signal *channelv1.InteractiveSignal) {
	if !validInteractiveSignal(signal, c.cfg.NodeID) {
		return
	}
	response := &channelv1.InteractiveSignalResponse{ChannelId: signal.GetChannelId(), NodeId: c.cfg.NodeID, LeaseEpoch: signal.GetLeaseEpoch(), Kind: signal.GetKind(), Generation: signal.GetGeneration(), RequestId: signal.GetRequestId()}
	if c.interactiveAdapter == nil {
		response.ReasonCode = "companion_unavailable"
	} else {
		out, err := c.interactiveAdapter.HandleInteractiveSignal(c.baseCtxOrBackground(), signal)
		if err != nil {
			response.ReasonCode = boundedDesktopError(err)
		} else if out != nil {
			response = out
			response.ChannelId = signal.GetChannelId()
			response.NodeId = c.cfg.NodeID
			response.LeaseEpoch = signal.GetLeaseEpoch()
			response.RequestId = signal.GetRequestId()
		}
	}
	if c.interactiveReporter != nil {
		if err := c.interactiveReporter.ReportInteractiveSignalResponse(c.baseCtxOrBackground(), response); err != nil {
			c.logger.Printf("channel: interactive signal response report failed: %v", err)
		}
	}
}

func validInteractiveSignal(signal *channelv1.InteractiveSignal, nodeID string) bool {
	if signal == nil || strings.TrimSpace(signal.GetChannelId()) == "" || strings.TrimSpace(signal.GetNodeId()) == "" || signal.GetNodeId() != nodeID ||
		strings.TrimSpace(signal.GetRequestId()) == "" || strings.TrimSpace(signal.GetGeneration()) == "" || strings.TrimSpace(signal.GetSessionId()) == "" || strings.TrimSpace(signal.GetLeaseId()) == "" ||
		signal.GetLeaseEpoch() == 0 || len(signal.GetPayload()) == 0 || len(signal.GetPayload()) > maxInteractiveSignalBytes {
		return false
	}
	switch signal.GetKind() {
	case uint32(desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_OFFER), uint32(desktopv1.DesktopSignalKind_DESKTOP_SIGNAL_KIND_ICE):
		return true
	default:
		return false
	}
}

func validInteractiveRevoke(revoke *channelv1.InteractiveRevoke, nodeID string) bool {
	return revoke != nil && strings.TrimSpace(revoke.GetChannelId()) != "" && strings.TrimSpace(revoke.GetNodeId()) == nodeID && strings.TrimSpace(revoke.GetSessionId()) != "" && strings.TrimSpace(revoke.GetLeaseId()) != "" && revoke.GetLeaseEpoch() > 0
}
