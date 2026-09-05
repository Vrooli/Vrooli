// Package channelmanager contains Content Desk's sole outbound Channel Manager
// boundary. It deliberately exposes release references, never credentials,
// sessions, or platform-specific account state.
package channelmanager

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	channelmanagerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager"
	channelmanagerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/channel-manager/v1/channelmanager/channelmanager_v1connect"
)

const (
	scenarioID     = "channel-manager"
	requestTimeout = 10 * time.Second
)

// Submission is the minimal editorial request Channel Manager needs to create
// a durable release action. The body and platform credentials remain in their
// respective owning scenarios.
type Submission struct {
	IdentityID, Lane, DraftID, IdempotencyKey string
	AssetIDs                                  []string
	DisclosureVisible                         bool
}

// Receipt is a projected Channel Manager release record. A scheduled receipt
// is not a publication claim; Content Desk records publication only after its
// outcome is delivered to the ledger inbox.
type Receipt struct{ ID, ActionID, Status string }

type Submitter interface {
	SubmitRelease(context.Context, Submission) (Receipt, error)
}
type EligibilityChecker interface {
	CheckEligibility(context.Context, string, string) (string, error)
}
type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

// Client resolves the dependency for each call so a restart or test-shadow
// port change cannot leave Content Desk pointing at a stale process.
type Client struct {
	resolver URLResolver
	http     *http.Client
}

func NewClient() *Client {
	return &Client{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: &http.Client{Timeout: requestTimeout}}
}

func (c *Client) SubmitRelease(ctx context.Context, submission Submission) (Receipt, error) {
	if submission.IdentityID == "" || submission.Lane == "" || submission.DraftID == "" || submission.IdempotencyKey == "" {
		return Receipt{}, fmt.Errorf("channel manager release requires identity, lane, draft, and idempotency key")
	}
	if c == nil || c.resolver == nil || c.http == nil {
		return Receipt{}, fmt.Errorf("channel manager integration is not configured")
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, scenarioID)
		if err == nil {
			client := channelmanagerconnect.NewChannelManagerServiceClient(c.http, strings.TrimRight(baseURL, "/"))
			response, callErr := client.SubmitRelease(callCtx, connect.NewRequest(&channelmanagerv1.SubmitReleaseRequest{IdentityId: submission.IdentityID, Lane: submission.Lane, DraftId: submission.DraftID, IdempotencyKey: submission.IdempotencyKey, AssetIds: submission.AssetIDs, DisclosureVisible: submission.DisclosureVisible}))
			if callErr == nil && response != nil && response.Msg != nil && response.Msg.Receipt != nil {
				cancel()
				return Receipt{ID: response.Msg.Receipt.Id, ActionID: response.Msg.Receipt.ActionId, Status: response.Msg.Receipt.Status}, nil
			}
			if callErr == nil {
				callErr = fmt.Errorf("channel manager returned no release receipt")
			}
			err = callErr
		}
		cancel()
		lastErr = err
		if !retryable(err) {
			break
		}
	}
	return Receipt{}, fmt.Errorf("channel manager release unavailable: %w", lastErr)
}

// CheckEligibility exposes the one permitted account-state question. Unknown
// and dependency failure are deliberately not normalized to eligible.
func (c *Client) CheckEligibility(ctx context.Context, identityID, lane string) (string, error) {
	if identityID == "" || lane == "" {
		return "", fmt.Errorf("channel manager eligibility requires identity and lane")
	}
	if c == nil || c.resolver == nil || c.http == nil {
		return "", fmt.Errorf("channel manager integration is not configured")
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, scenarioID)
		if err == nil {
			client := channelmanagerconnect.NewChannelManagerServiceClient(c.http, strings.TrimRight(baseURL, "/"))
			response, callErr := client.GetEligibility(callCtx, connect.NewRequest(&channelmanagerv1.GetEligibilityRequest{IdentityId: identityID, Lane: lane}))
			if callErr == nil && response != nil && response.Msg != nil && response.Msg.Eligibility != "" {
				cancel()
				return response.Msg.Eligibility, nil
			}
			if callErr == nil {
				callErr = fmt.Errorf("channel manager returned no eligibility")
			}
			err = callErr
		}
		cancel()
		lastErr = err
		if !retryable(err) {
			break
		}
	}
	return "unknown", fmt.Errorf("channel manager eligibility unavailable: %w", lastErr)
}

func retryable(err error) bool {
	return connect.CodeOf(err) == connect.CodeUnavailable || connect.CodeOf(err) == connect.CodeDeadlineExceeded
}
