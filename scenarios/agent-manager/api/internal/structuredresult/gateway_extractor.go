package structuredresult

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
)

// GatewayClient is the generated typed-inference client. Keeping this small
// interface makes the migration testable without starting a coding runner or
// depending on a live provider.
type GatewayClient interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

// GatewayExtractor sends constrained extraction to ai-gateway. It never
// resolves a coding-agent role and never starts a runner process.
type GatewayExtractor struct {
	Client     GatewayClient
	HTTPClient connect.HTTPClient
	ResolveURL func(context.Context) (string, error)
}

// DefaultGatewayTimeout bounds one typed-inference call. It sits above the
// largest timeout ai-gateway declares for a role so the server-side bound is
// the one that reports a reason, while this client still refuses to wait
// forever if the gateway itself stops responding. http.DefaultClient must not
// be used here: it has no timeout at all.
const DefaultGatewayTimeout = 3 * time.Minute

func NewGatewayExtractor(httpClient connect.HTTPClient) *GatewayExtractor {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultGatewayTimeout}
	}
	return &GatewayExtractor{
		HTTPClient: httpClient,
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
	}
}

func (e *GatewayExtractor) Extract(ctx context.Context, request ExtractRequest) (ExtractResponse, error) {
	if e == nil {
		return ExtractResponse{Abstained: true}, errors.New("ai-gateway extractor is unavailable")
	}
	client, err := e.client(ctx)
	if err != nil {
		return ExtractResponse{Abstained: true}, err
	}
	role := strings.TrimSpace(request.RoleRef)
	if role == "" {
		role = DefaultExtractRole
	}
	instruction := strings.TrimSpace(request.Instruction)
	if instruction == "" {
		// The gateway keeps schema descriptions as metadata and never treats
		// them as instruction, so a run result carries no intent unless one is
		// stated here.
		instruction = "Extract the structured result described by the schema from the agent run output."
	}
	response, err := client.Run(ctx, connect.NewRequest(&inferencev1.RunRequest{
		Source: request.Source, SchemaJson: string(request.Schema), Role: role, Instruction: instruction,
	}))
	if err != nil {
		return ExtractResponse{Abstained: true}, fmt.Errorf("call ai-gateway typed inference: %w", err)
	}
	if response == nil || response.Msg == nil {
		return ExtractResponse{Abstained: true}, errors.New("ai-gateway returned no typed inference response")
	}
	msg := response.Msg
	result := ExtractResponse{Provider: msg.GetProvider(), Model: msg.GetModel()}
	if failure := msg.GetError(); failure != nil {
		result.Abstained = true
		return result, fmt.Errorf("ai-gateway typed inference %s: %s", failure.GetCode().String(), failure.GetMessage())
	}
	if !msg.GetValidated() || strings.TrimSpace(msg.GetValueJson()) == "" {
		result.Abstained = true
		return result, errors.New("ai-gateway returned an unvalidated typed inference value")
	}
	result.Candidate = []byte(msg.GetValueJson())
	return result, nil
}

func (e *GatewayExtractor) client(ctx context.Context) (GatewayClient, error) {
	if e.Client != nil {
		return e.Client, nil
	}
	if e.ResolveURL == nil {
		return nil, errors.New("ai-gateway URL resolver is unavailable")
	}
	baseURL, err := e.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve ai-gateway URL: %w", err)
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("ai-gateway URL is empty")
	}
	httpClient := e.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return inferenceconnect.NewInferenceServiceClient(httpClient, baseURL), nil
}

var _ Extractor = (*GatewayExtractor)(nil)
