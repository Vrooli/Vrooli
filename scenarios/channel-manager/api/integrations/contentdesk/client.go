// Package contentdesk is Channel Manager's outbound delivery boundary for
// credential-free publication outcomes and metric samples.
package contentdesk

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts"
	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/artifacts/artifacts_v1connect"
	ledgerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger/ledger_v1connect"
)

const (
	scenarioID = "content-desk"
	timeout    = 10 * time.Second
)

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}
type ReleaseOutcome struct {
	ReceiptID, DraftID, Status, PlatformPostID, PublishedURL string
	PublishedAt                                              time.Time
}
type MetricSample struct {
	ID, ReleaseID, DraftID, Metric string
	Value                          float64
	ObservedAt                     time.Time
}
type Deliverer interface {
	DeliverRelease(context.Context, ReleaseOutcome) error
	DeliverMetric(context.Context, MetricSample) error
}
type Client struct {
	resolver URLResolver
	http     *http.Client
}

func NewClient() *Client {
	return &Client{resolver: discovery.NewResolver(discovery.ResolverConfig{}), http: &http.Client{Timeout: timeout}}
}

func (c *Client) DeliverRelease(ctx context.Context, outcome ReleaseOutcome) error {
	if outcome.ReceiptID == "" || outcome.DraftID == "" || outcome.PlatformPostID == "" || outcome.PublishedURL == "" || outcome.PublishedAt.IsZero() {
		return fmt.Errorf("release delivery requires receipt, draft, post id, URL, and timestamp")
	}
	return c.withBaseURL(ctx, func(ctx context.Context, base string) error {
		client := artifactsconnect.NewArtifactsServiceClient(c.http, base)
		_, err := client.RecordReleaseOutcome(ctx, connect.NewRequest(&artifactsv1.RecordReleaseOutcomeRequest{ReceiptId: outcome.ReceiptID, DraftId: outcome.DraftID, Status: outcome.Status, PlatformPostId: outcome.PlatformPostID, PublishedUrl: outcome.PublishedURL, PublishedAt: outcome.PublishedAt.UTC().Format(time.RFC3339Nano)}))
		return err
	})
}

func (c *Client) DeliverMetric(ctx context.Context, sample MetricSample) error {
	if sample.ID == "" || sample.ReleaseID == "" || sample.DraftID == "" || sample.Metric == "" || sample.ObservedAt.IsZero() {
		return fmt.Errorf("metric delivery requires sample, release, draft, metric, and timestamp")
	}
	return c.withBaseURL(ctx, func(ctx context.Context, base string) error {
		client := ledgerconnect.NewLedgerServiceClient(c.http, base)
		response, err := client.IngestMetricSample(ctx, connect.NewRequest(&ledgerv1.IngestMetricSampleRequest{SampleId: sample.ID, ReleaseId: sample.ReleaseID, DraftId: sample.DraftID, Metric: sample.Metric, Value: sample.Value, ObservedAt: sample.ObservedAt.UTC().Format(time.RFC3339Nano)}))
		if err == nil && (response == nil || response.Msg == nil || !response.Msg.Accepted) {
			return fmt.Errorf("content desk did not acknowledge metric sample")
		}
		return err
	})
}

func (c *Client) withBaseURL(ctx context.Context, call func(context.Context, string) error) error {
	if c == nil || c.resolver == nil || c.http == nil {
		return fmt.Errorf("content desk integration is not configured")
	}
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		base, err := c.resolver.ResolveScenarioURLDefault(callCtx, scenarioID)
		if err == nil {
			err = call(callCtx, strings.TrimRight(base, "/"))
		}
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if connect.CodeOf(err) != connect.CodeUnavailable && connect.CodeOf(err) != connect.CodeDeadlineExceeded {
			break
		}
	}
	return fmt.Errorf("content desk delivery unavailable: %w", lastErr)
}
