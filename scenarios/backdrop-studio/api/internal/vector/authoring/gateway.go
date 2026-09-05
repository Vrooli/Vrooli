package authoring

import (
	"context"
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

// GatewayClient is the only place this scenario talks to ai-gateway directly.
//
// Decision D-006 routes *image* inference through image-tools rather than
// through the gateway, and that decision stands. Its reason is specific: image
// generation has a local lane, and calling the gateway directly would duplicate
// image-tools' host probing and tier selection, producing a second and
// divergent answer to "what can this machine do right now".
//
// None of that applies here. Authoring a generator is text generation, and
// image-tools serves no text — its AI catalog is generation and enhancement of
// pixels. There is no local lane to duplicate and no host-capability question
// to answer twice. Routing text through an image scenario to reach a text model
// would be the indirection with no reason behind it.
//
// What this still owes D-006 is its consequence: the caller records which model
// answered, so an authored generator can be disclosed and re-authored.
type GatewayClient struct {
	HTTPClient *http.Client
	Resolve    func(context.Context) (string, error)
	// Timeout bounds one authoring call. Long, because a generator is a few
	// hundred lines of template and a frontier model writes it slowly.
	Timeout time.Duration
}

// NewGatewayClient resolves ai-gateway at call time, so an absent scenario is a
// named missing capability when someone authors rather than a startup failure
// for everyone who does not.
func NewGatewayClient() *GatewayClient {
	return &GatewayClient{
		HTTPClient: &http.Client{Timeout: 6 * time.Minute},
		Resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
		Timeout: 5 * time.Minute,
	}
}

// Author sends one authoring prompt through ai-gateway's role routing.
func (c *GatewayClient) Author(ctx context.Context, prompt string) (string, string, error) {
	if c.Resolve == nil {
		return "", "", fmt.Errorf("ai-gateway URL resolver is not configured")
	}
	base, err := c.Resolve(ctx)
	if err != nil {
		return "", "", fmt.Errorf("resolve ai-gateway: %w", err)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", "", fmt.Errorf("ai-gateway discovery returned an empty URL")
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	client := routingconnect.NewRoutingServiceClient(httpClient, base)
	response, err := client.ExecuteRoute(ctx, connect.NewRequest(&routingv1.ExecuteRouteRequest{
		Request: &sharedv1.GatewayRequest{
			Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
			Role: Role,
			// A generator is written from a brief this operator wrote and
			// returns source code. Nothing in it is private to a customer, and
			// the frontier candidates for this role are remote — so declaring
			// local-only would refuse every route that can actually serve it.
			Profile:      sharedv1.Profile_PROFILE_REMOTE_ONLY,
			PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
			Operation:    "author_generator",
			Scenario:     "backdrop-studio",
			TimeoutMs:    int32(timeout.Milliseconds()),
			RequestId:    uuid.NewString(),
		},
		InputText: prompt,
	}))
	if err != nil {
		return "", "", err
	}
	msg := response.Msg
	if msg == nil {
		return "", "", fmt.Errorf("ai-gateway returned no response")
	}
	if !msg.GetValid() {
		reasons := append([]string(nil), msg.GetPolicyReasons()...)
		for _, issue := range msg.GetIssues() {
			reasons = append(reasons, issue.GetMessage())
		}
		if len(reasons) == 0 {
			reasons = []string{"no reason reported"}
		}
		return "", "", fmt.Errorf("ai-gateway refused role %q: %s", Role, strings.Join(reasons, "; "))
	}
	text := msg.GetOutputText()
	if strings.TrimSpace(text) == "" {
		// An empty answer with valid=true is what an unroutable role looks like
		// on this edge: the call succeeded and nothing served it. The policy
		// reasons are the actionable part — "no provider satisfied role,
		// capability, locality, and privacy constraints" tells an operator to
		// look at provider policy, where "empty answer" tells them nothing.
		detail := strings.Join(msg.GetPolicyReasons(), "; ")
		if detail == "" {
			detail = "no reason reported"
		}
		provider := msg.GetEvidence().GetSelectedProvider()
		if provider == "" {
			provider = "no provider was selected"
		}
		return "", "", fmt.Errorf("ai-gateway role %q returned an empty answer (%s): %s", Role, provider, detail)
	}
	// The model the gateway actually resolved, never one this scenario asked
	// for. It is the disclosure record for every asset an authored generator
	// later draws, and this is the only moment it exists.
	model := msg.GetEvidence().GetSelectedModel()
	if model == "" {
		model = msg.GetEvidence().GetSelectedProvider()
	}
	return text, model, nil
}
