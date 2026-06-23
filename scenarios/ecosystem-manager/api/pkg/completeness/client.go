package completeness

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	scoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring"
	scoringconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-completeness-scoring/v1/scoring/scoring_v1connect"
)

// scenarioName is the discovery key for the scoring authority.
const scenarioName = "scenario-completeness-scoring"

// defaultTimeout bounds a single score fetch. completeness-scoring's read path
// is filesystem-only with a sub-second warm budget; this is a generous ceiling.
const defaultTimeout = 10 * time.Second

// Client is a Connect-RPC client for completeness-scoring's ScoreService,
// resolved per call via api-core discovery (mirrors pkg/dtv's pattern).
type Client struct {
	httpClient *http.Client
	// resolve returns the scoring service's base URL; injectable so the contract
	// test can point at an in-process handler without a discovery registry.
	resolve func(ctx context.Context) (string, error)
}

// NewClient builds a completeness-scoring client. A non-positive timeout uses
// defaultTimeout.
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, scenarioName)
		},
	}
}

var _ Provider = (*Client)(nil)

// Score fetches the cached completeness payload for a scenario and maps it to the
// EM projection. Unlike the DTV seam it does NOT fail open: any
// resolution/transport/RPC error is returned so the controller degrades loudly
// (plan D2) — measurement is load-bearing for termination.
func (c *Client) Score(ctx context.Context, scenario string) (Score, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return Score{}, fmt.Errorf("resolve %s url: %w", scenarioName, err)
	}
	rc := scoringconnect.NewScoreServiceClient(c.httpClient, baseURL)
	resp, err := rc.GetScore(ctx, connect.NewRequest(&scoringv1.GetScoreRequest{Scenario: scenario}))
	if err != nil {
		return Score{}, fmt.Errorf("completeness GetScore(%q): %w", scenario, err)
	}
	if resp == nil || resp.Msg == nil {
		return Score{}, fmt.Errorf("completeness GetScore(%q): empty response", scenario)
	}
	return scoreFromProto(resp.Msg), nil
}
