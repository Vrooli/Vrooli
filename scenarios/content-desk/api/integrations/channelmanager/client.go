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

// DistributionState is a credential-free projection of whether Channel Manager
// currently has a live, attributable distribution surface. It never carries an
// account handle, credential reference or platform-specific detail.
type DistributionState struct {
	Connected  bool
	Source     string
	ObservedAt string
}

const (
	overviewSource      = "channel-manager:overview"
	identityStatusAlive = "active"
)

// ConnectedSurfaces reports whether Channel Manager currently has any identity
// in the active state, which is the observed condition for a wired distribution
// surface. A reachable owner with no active identity is a real disconnected
// reading. An unreachable owner returns an error so the caller keeps the stored
// value instead of fabricating a verdict.
func (c *Client) ConnectedSurfaces(ctx context.Context) (DistributionState, error) {
	if c == nil || c.resolver == nil || c.http == nil {
		return DistributionState{}, fmt.Errorf("channel manager integration is not configured")
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		baseURL, err := c.resolver.ResolveScenarioURLDefault(callCtx, scenarioID)
		if err == nil {
			client := channelmanagerconnect.NewChannelManagerServiceClient(c.http, strings.TrimRight(baseURL, "/"))
			response, callErr := client.GetOverview(callCtx, connect.NewRequest(&channelmanagerv1.GetOverviewRequest{}))
			if callErr == nil && response != nil && response.Msg != nil {
				cancel()
				return DistributionState{Connected: hasActiveIdentity(response.Msg.Identities), Source: overviewSource, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}, nil
			}
			if callErr == nil {
				callErr = fmt.Errorf("channel manager returned no overview")
			}
			err = callErr
		}
		cancel()
		lastErr = err
		if !retryable(err) {
			break
		}
	}
	return DistributionState{}, fmt.Errorf("channel manager overview unavailable: %w", lastErr)
}

func hasActiveIdentity(identities []*channelmanagerv1.Identity) bool {
	for _, identity := range identities {
		if identity.GetStatus() == identityStatusAlive {
			return true
		}
	}
	return false
}

func retryable(err error) bool {
	return connect.CodeOf(err) == connect.CodeUnavailable || connect.CodeOf(err) == connect.CodeDeadlineExceeded
}
