package dtv

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"
	reportconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report/report_v1connect"
)

// scenarioName is the discovery key for development-toolchain-validator.
const scenarioName = "development-toolchain-validator"

// defaultTimeout bounds a single fitness fetch so a slow/hung DTV never stalls
// the controller's init (the SELECT hot path never calls DTV synchronously).
const defaultTimeout = 5 * time.Second

// Client is a Connect-RPC client for DTV's ReportService, resolved per call via
// api-core discovery (mirrors pkg/agentmanager's resolution pattern).
type Client struct {
	httpClient *http.Client
	// resolve returns DTV's base URL; injectable so the contract test can point
	// at an in-process handler without a discovery registry.
	resolve func(ctx context.Context) (string, error)
}

// NewClient builds a DTV client. A non-positive timeout uses defaultTimeout.
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

var _ SkillFitnessProvider = (*Client)(nil)

// Fitness fetches DTV's per-skill fitness aggregate and maps it to the EM-local
// projection. It FAILS OPEN: any resolution/transport/RPC error returns an
// UNKNOWN Fitness alongside the error, so the controller degrades to P1 (uniform
// prior, allow-all) rather than blocking.
func (c *Client) Fitness(ctx context.Context, skillID string) (Fitness, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return Fitness{SkillID: skillID}, fmt.Errorf("resolve %s url: %w", scenarioName, err)
	}
	rc := reportconnect.NewReportServiceClient(c.httpClient, baseURL)
	resp, err := rc.GetSkillFitness(ctx, connect.NewRequest(&reportv1.GetSkillFitnessRequest{SkillId: skillID}))
	if err != nil {
		return Fitness{SkillID: skillID}, fmt.Errorf("dtv GetSkillFitness(%q): %w", skillID, err)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Fitness == nil {
		return Fitness{SkillID: skillID}, nil
	}
	return fitnessFromProto(resp.Msg.Fitness), nil
}
