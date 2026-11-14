package readiness

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PromotionClient struct {
	ResolveURL func(context.Context) (string, error)
	HTTPClient *http.Client
}

func NewPromotionClient() *PromotionClient {
	return &PromotionClient{
		ResolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURL(ctx, "offer-desk", "API_PORT")
		},
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// PublishReadinessFact records the deployment-manager verdict as an Offer
// Desk fact. Offer Desk evaluates the trigger; it does not calculate the
// readiness verdict.
func (c *PromotionClient) PublishReadinessFact(ctx context.Context, nodeID, scenario, commit string, approved bool, observedAt time.Time) (*offerspb.Fact, error) {
	if c == nil || c.ResolveURL == nil {
		return nil, fmt.Errorf("offer-desk promotion client is not configured")
	}
	if strings.TrimSpace(nodeID) == "" || strings.TrimSpace(scenario) == "" || strings.TrimSpace(commit) == "" {
		return nil, fmt.Errorf("node_id, scenario, and commit are required")
	}
	baseURL, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve offer-desk: %w", err)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	client := offersconnect.NewGatesServiceClient(httpClient, baseURL)
	factName := fmt.Sprintf("readiness:%s:%s", scenario, commit)
	value := 0.0
	if approved {
		value = 1
	}
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	response, err := client.AddFact(ctx, connect.NewRequest(&offerspb.AddFactRequest{Fact: &offerspb.Fact{
		Name: factName, Value: value, Dimension: "readiness", StaleAfterDays: 7, ObservedAt: timestamppb.New(observedAt.UTC()),
	}}))
	if err != nil {
		return nil, fmt.Errorf("record readiness fact: %w", err)
	}
	return response.Msg.GetFact(), nil
}

func (c *PromotionClient) DeclareReadinessTrigger(ctx context.Context, nodeID, scenario, commit string) (*offerspb.Trigger, error) {
	if c == nil || c.ResolveURL == nil {
		return nil, fmt.Errorf("offer-desk promotion client is not configured")
	}
	baseURL, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, err
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	client := offersconnect.NewGatesServiceClient(httpClient, baseURL)
	response, err := client.DeclareTrigger(ctx, connect.NewRequest(&offerspb.DeclareTriggerRequest{Trigger: &offerspb.Trigger{
		NodeId: nodeID, FactName: fmt.Sprintf("readiness:%s:%s", scenario, commit), Operator: ">=", Threshold: 1,
	}}))
	if err != nil {
		return nil, fmt.Errorf("declare readiness promotion trigger: %w", err)
	}
	return response.Msg.GetTrigger(), nil
}

func (c *PromotionClient) Evaluate(ctx context.Context) (*offerspb.EvaluateResponse, error) {
	if c == nil || c.ResolveURL == nil {
		return nil, fmt.Errorf("offer-desk promotion client is not configured")
	}
	baseURL, err := c.ResolveURL(ctx)
	if err != nil {
		return nil, err
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := offersconnect.NewGatesServiceClient(httpClient, baseURL).Evaluate(ctx, connect.NewRequest(&offerspb.EvaluateRequest{}))
	if err != nil {
		return nil, fmt.Errorf("evaluate readiness promotion trigger: %w", err)
	}
	return response.Msg, nil
}
