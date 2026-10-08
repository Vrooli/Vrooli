// Package lifecycle contains the shared provider-side maintenance-fence
// contract used by the control plane and scenarios. Wire types are generated
// from packages/proto; this package owns transport policy and validation.
package lifecycle

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"connectrpc.com/connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	commonv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1/commonv1connect"
)

const ProtocolVersion = "maintenance-fence-v1"

type Client struct {
	rpc commonv1connect.LifecycleMaintenanceServiceClient
}

// LoopbackOnly protects lifecycle mutation endpoints that are intentionally
// called by the local control plane. It is a transport boundary, not an
// identity system: deployments that expose a scenario through a proxy must
// put the proxy behind an equivalent authenticated local boundary.
func LoopbackOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.RemoteAddr
		if parsed, err := netip.ParseAddr(strings.TrimSpace(host)); err == nil {
			if parsed.IsLoopback() {
				next.ServeHTTP(w, r)
				return
			}
		}
		if host, _, err := net.SplitHostPort(host); err == nil {
			if parsed, err := netip.ParseAddr(host); err == nil && parsed.IsLoopback() {
				next.ServeHTTP(w, r)
				return
			}
		}
		http.Error(w, "lifecycle endpoint requires a loopback caller", http.StatusForbidden)
	})
}

func NewClient(httpClient *http.Client, baseURL string, opts ...connect.ClientOption) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("lifecycle: http client is required")
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("lifecycle: base URL is required")
	}
	return &Client{rpc: commonv1connect.NewLifecycleMaintenanceServiceClient(httpClient, strings.TrimRight(baseURL, "/"), opts...)}, nil
}

func (c *Client) Prepare(ctx context.Context, req *commonv1.LifecyclePrepareRequest) (*commonv1.LifecyclePrepareResponse, error) {
	if c == nil || c.rpc == nil {
		return nil, fmt.Errorf("lifecycle: client is not configured")
	}
	resp, err := c.rpc.Prepare(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c *Client) Status(ctx context.Context, req *commonv1.LifecycleStatusRequest) (*commonv1.LifecycleStatusResponse, error) {
	if c == nil || c.rpc == nil {
		return nil, fmt.Errorf("lifecycle: client is not configured")
	}
	resp, err := c.rpc.Status(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c *Client) Drain(ctx context.Context, req *commonv1.LifecycleDrainRequest) (*commonv1.LifecycleDrainResponse, error) {
	if c == nil || c.rpc == nil {
		return nil, fmt.Errorf("lifecycle: client is not configured")
	}
	resp, err := c.rpc.Drain(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func (c *Client) Resume(ctx context.Context, req *commonv1.LifecycleResumeRequest) (*commonv1.LifecycleResumeResponse, error) {
	if c == nil || c.rpc == nil {
		return nil, fmt.Errorf("lifecycle: client is not configured")
	}
	resp, err := c.rpc.Resume(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func ValidateStanding(s *commonv1.LifecycleStanding) error {
	if s == nil {
		return fmt.Errorf("lifecycle: standing is required")
	}
	if strings.TrimSpace(s.GetProvider()) == "" || strings.TrimSpace(s.GetProviderInstanceId()) == "" {
		return fmt.Errorf("lifecycle: provider identity is incomplete")
	}
	if strings.TrimSpace(s.GetOperationId()) == "" {
		return fmt.Errorf("lifecycle: operation ID is required")
	}
	if s.GetAdmissionClosed() && strings.TrimSpace(s.GetFenceToken()) == "" {
		return fmt.Errorf("lifecycle: closed standing has no fence token")
	}
	if !s.GetInventoryComplete() {
		return fmt.Errorf("lifecycle: inventory is incomplete")
	}
	return nil
}
