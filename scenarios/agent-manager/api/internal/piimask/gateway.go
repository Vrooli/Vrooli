package piimask

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// detectRole is the pii-detector resource role routed for a DETECT_PII request.
// The routing path carries a caller-supplied role string straight to the
// provider adapter (ai-gateway's routing has no gateway-side role catalog), so
// this is the resource role, not the inference catalog's detect.pii.default.
// pii.small is the always-present default tier; the resource's own policy
// degrades within it.
const detectRole = "pii.small"

// DefaultGatewayTimeout bounds one DETECT_PII call. PII detection is a local
// CPU pass (regex/checksum + a small NER model), so this is far shorter than a
// generation call, but http.DefaultClient must not be used here — it has no
// timeout at all and a stalled gateway would hang an entire import.
const DefaultGatewayTimeout = 30 * time.Second

// RoutingClient is the generated ai-gateway routing client surface this masker
// needs. Keeping it to one method makes the detector testable with a fake and
// never requires a live gateway or detector resource.
type RoutingClient interface {
	ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error)
}

// GatewayDetector detects PII by calling ai-gateway's DETECT_PII route, which
// the gateway dispatches to the shared pii-detector resource. It uses the same
// governed Connect binding as every other scenario that talks to the gateway
// (routingconnect.NewRoutingServiceClient) — never raw HTTP to the detector.
type GatewayDetector struct {
	Client     RoutingClient
	HTTPClient connect.HTTPClient
	ResolveURL func(context.Context) (string, error)
}

// NewGatewayDetector resolves ai-gateway at call time so an absent gateway is a
// named, fail-closed error when a redact rule fires rather than a startup
// failure for every import.
func NewGatewayDetector() *GatewayDetector {
	return &GatewayDetector{
		HTTPClient: &http.Client{Timeout: DefaultGatewayTimeout},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
	}
}

// Detect sends one text through the gateway DETECT_PII route and decodes the
// pii-detector span result. Any transport, routing, or decode failure is
// returned as an error so the Masker fails closed; a successful but empty
// result (no PII) returns an empty span set with a nil error.
func (d *GatewayDetector) Detect(ctx context.Context, text string) (DetectResult, error) {
	if d == nil {
		return DetectResult{}, errors.New("pii-detector gateway is unavailable")
	}
	client, err := d.resolveClient(ctx)
	if err != nil {
		return DetectResult{}, err
	}
	response, err := client.ExecuteRoute(ctx, connect.NewRequest(&routingv1.ExecuteRouteRequest{
		Request: &sharedv1.GatewayRequest{
			Kind: sharedv1.RequestKind_REQUEST_KIND_DETECT_PII,
			Role: detectRole,
			// PII scanning must never leave the host: the detector is a local
			// resource and local-only refuses any route that would send the
			// content to a remote provider.
			Profile:      sharedv1.Profile_PROFILE_LOCAL_ONLY,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
			Operation:    "conversation_redact",
			Scenario:     "agent-manager",
			TimeoutMs:    int32(DefaultGatewayTimeout.Milliseconds()),
			RequestId:    uuid.NewString(),
		},
		InputText: text,
	}))
	if err != nil {
		return DetectResult{}, fmt.Errorf("call ai-gateway detect-pii: %w", err)
	}
	msg := response.Msg
	if msg == nil {
		return DetectResult{}, errors.New("ai-gateway returned no detect-pii response")
	}
	if !msg.GetValid() {
		reasons := append([]string(nil), msg.GetPolicyReasons()...)
		for _, issue := range msg.GetIssues() {
			reasons = append(reasons, issue.GetMessage())
		}
		if len(reasons) == 0 {
			reasons = []string{"no reason reported"}
		}
		return DetectResult{}, fmt.Errorf("ai-gateway refused detect-pii: %s", strings.Join(reasons, "; "))
	}
	output := msg.GetOutputText()
	if strings.TrimSpace(output) == "" {
		// A valid response with no output text on this route means nothing served
		// the detector (no provider matched role/capability/locality). That is a
		// detection failure, not "clean text" — treat it as an error so masking
		// fails closed rather than passing PII through as unscanned.
		detail := strings.Join(msg.GetPolicyReasons(), "; ")
		if detail == "" {
			detail = "no provider served the detector"
		}
		return DetectResult{}, fmt.Errorf("ai-gateway detect-pii returned no spans output: %s", detail)
	}
	return DecodeDetectResult(output)
}

func (d *GatewayDetector) resolveClient(ctx context.Context) (RoutingClient, error) {
	if d.Client != nil {
		return d.Client, nil
	}
	if d.ResolveURL == nil {
		return nil, errors.New("ai-gateway URL resolver is unavailable")
	}
	baseURL, err := d.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve ai-gateway URL: %w", err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("ai-gateway URL is empty")
	}
	httpClient := d.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return routingconnect.NewRoutingServiceClient(httpClient, baseURL), nil
}

var _ Detector = (*GatewayDetector)(nil)
