package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/discovery"

	eligpb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility/eligibility_v1connect"
)

// TestGenieEligibilityClient calls test-genie's EligibilityService Connect-RPC.
// It is a thin wrapper: the GCT isolation handler is the only consumer.
type TestGenieEligibilityClient struct {
	timeout      time.Duration
	resolveURL   func(ctx context.Context) (string, error)
	makeClient   func(baseURL string) eligibility_v1connect.EligibilityServiceClient
}

// NewTestGenieEligibilityClient returns a client that resolves the live
// test-genie API URL through service discovery on every call.
func NewTestGenieEligibilityClient(timeout time.Duration) *TestGenieEligibilityClient {
	httpClient := &http.Client{Timeout: timeout}
	return &TestGenieEligibilityClient{
		timeout: timeout,
		resolveURL: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "test-genie")
		},
		makeClient: func(baseURL string) eligibility_v1connect.EligibilityServiceClient {
			return eligibility_v1connect.NewEligibilityServiceClient(httpClient, baseURL)
		},
	}
}

// Check returns test-genie's routed-eligibility decision for the named
// scenario.
func (c *TestGenieEligibilityClient) Check(ctx context.Context, scenario string) (*eligpb.CheckResponse, error) {
	baseURL, err := c.resolveURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve test-genie url: %w", err)
	}
	client := c.makeClient(baseURL)
	resp, err := client.Check(ctx, connect.NewRequest(&eligpb.CheckRequest{Scenario: scenario}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}
