package runannotations

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// ErrDecisionUnavailable wraps every failure to reach or obtain an answer from
// ai-gateway. Labels then store state unavailable, never a fabricated zero.
var ErrDecisionUnavailable = errors.New("ai-gateway decision capability unavailable")

const decisionTimeout = 150 * time.Second

// DecisionClient is the generated inference surface this package uses. Tests
// implement it directly; GatewayDecisionClient adapts ai-gateway discovery.
type DecisionClient interface {
	DecideBatch(context.Context, *inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error)
}

// GatewayDecisionClient resolves ai-gateway lazily so an absent gateway
// degrades annotation instead of failing API startup.
type GatewayDecisionClient struct {
	ResolveURL func(context.Context) (string, error)
	HTTPClient connect.HTTPClient

	mu       sync.Mutex
	resolved inferenceconnect.InferenceServiceClient
}

// NewGatewayDecisionClient returns the default discovery-backed client.
func NewGatewayDecisionClient() *GatewayDecisionClient {
	return &GatewayDecisionClient{
		HTTPClient: &http.Client{Timeout: decisionTimeout},
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		},
	}
}

// DecideBatch implements DecisionClient.
func (c *GatewayDecisionClient) DecideBatch(ctx context.Context, request *inferencev1.DecideBatchRequest) (*inferencev1.DecideBatchResponse, error) {
	client, err := c.resolve(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.DecideBatch(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecisionUnavailable, err)
	}
	return response.Msg, nil
}

func (c *GatewayDecisionClient) resolve(ctx context.Context) (inferenceconnect.InferenceServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.resolved != nil {
		return c.resolved, nil
	}
	if c.ResolveURL == nil {
		return nil, errors.New("ai-gateway URL resolver is unset")
	}
	base, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve ai-gateway: %v", ErrDecisionUnavailable, err)
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return nil, fmt.Errorf("%w: ai-gateway URL is empty", ErrDecisionUnavailable)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: decisionTimeout}
	}
	c.resolved = inferenceconnect.NewInferenceServiceClient(httpClient, base)
	return c.resolved, nil
}

// taskContract is the source policy every annotation decision carries.
// Conversation content is confidential and local-only: a hosted decision
// upstream is never eligible for these labels.
type taskContract struct {
	Role          string
	Profile       sharedv1.Profile
	PrivacyClass  sharedv1.PrivacyClass
	PrivacySource string
}

func (s *Service) contract(role string) taskContract {
	if strings.TrimSpace(role) == "" {
		role = s.pack.Pack.Runtime.DefaultRole
	}
	if strings.TrimSpace(role) == "" {
		role = DefaultDecisionRole
	}
	return taskContract{
		Role:          role,
		Profile:       sharedv1.Profile_PROFILE_LOCAL_ONLY,
		PrivacyClass:  sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacySource: "agent_manager.run_annotations",
	}
}

// questionFor builds the gateway question a pack label declares.
func questionFor(label Label) (*inferencev1.DecisionQuestion, error) {
	q := label.Question
	if q == nil {
		return nil, fmt.Errorf("%s: decision label has no question", label.ID)
	}
	out := &inferencev1.DecisionQuestion{
		Id:           label.ID,
		Instructions: q.Instructions,
	}
	switch q.Type {
	case QuestionChoice:
		options := make([]*inferencev1.DecisionOption, len(q.Options))
		for i, option := range q.Options {
			options[i] = &inferencev1.DecisionOption{Label: option.Label, Description: option.Description}
		}
		out.Type = inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE
		out.Options = options
	case QuestionScore:
		out.Type = inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE
		out.Levels = append([]string(nil), q.Levels...)
	case QuestionNoul:
		out.Type = inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_NOUL
		out.TrueDescription = q.TrueDescription
		out.FalseDescription = q.FalseDescription
	default:
		return nil, fmt.Errorf("%s: unknown question type %q", label.ID, q.Type)
	}
	return out, nil
}
