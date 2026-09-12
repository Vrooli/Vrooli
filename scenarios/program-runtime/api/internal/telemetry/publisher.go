package telemetry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventType is the stable platform-bus type for program-runtime evidence.
// The payload is the typed program-runtime ProgramEvent message, while the
// envelope remains owned by vrooli-events for routing and correlation.
const EventType = "vrooli.program_runtime.program_event.v1"

// Publisher is deliberately narrower than the event-bus client. The runtime
// emits facts; it does not aggregate, rank, or otherwise analyze them.
type Publisher interface {
	Publish(context.Context, *telemetryv1.ProgramEvent) error
}

// HTTPPublisher publishes typed events using the canonical vrooli-events HTTP
// envelope contract. A nil/empty base URL is a disabled publisher, which keeps
// the runtime healthy when the optional event service is unavailable.
type HTTPPublisher struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewPublisher uses an explicitly supplied Events endpoint when lifecycle
// provides one and otherwise resolves the local service through api-core's
// standard discovery seam. Discovery is performed per publication only when
// needed, so a missing optional service never delays runtime startup.
func NewPublisher(baseURL string) Publisher {
	return &discoveryPublisher{baseURL: strings.TrimSpace(baseURL), client: &http.Client{Timeout: 2 * time.Second}}
}

type discoveryPublisher struct {
	baseURL string
	client  *http.Client
}

func (p *discoveryPublisher) Publish(ctx context.Context, event *telemetryv1.ProgramEvent) error {
	base := p.baseURL
	if base == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
		if err != nil {
			return err
		}
		base = resolved
	}
	return (HTTPPublisher{BaseURL: base, HTTPClient: p.client}).Publish(ctx, event)
}

func (p HTTPPublisher) Publish(ctx context.Context, event *telemetryv1.ProgramEvent) error {
	if event == nil || strings.TrimSpace(p.BaseURL) == "" {
		return nil
	}
	data, err := anypb.New(event)
	if err != nil {
		return fmt.Errorf("pack program event: %w", err)
	}
	occurred := timestamppb.Now()
	if event.GetOccurredAt() != "" {
		if parsed, parseErr := timeParse(event.GetOccurredAt()); parseErr == nil {
			occurred = timestamppb.New(parsed)
		}
	}
	envelope := &domain.EventEnvelope{
		EventId:    event.GetEventId(),
		EventType:  EventType,
		OccurredAt: occurred,
		Source:     &domain.EventSource{Scenario: "program-runtime", ActorKind: "program"},
		Target:     &domain.EventTarget{Scenario: "vrooli-events", Operation: "program-runtime.telemetry", Protocol: "events"},
		Data:       data,
	}
	body, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal program event envelope: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.BaseURL, "/")+"/api/v1/events", strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create event request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	h := p.HTTPClient
	if h == nil {
		h = http.DefaultClient
	}
	resp, err := h.Do(req)
	if err != nil {
		return fmt.Errorf("publish program event: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusConflict {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("publish program event: %s", resp.Status)
	}
	return nil
}

// timeParse is kept as a small seam so event timestamps remain the source
// event's timestamp when it is valid, while malformed test/fallback timestamps
// still receive a valid envelope timestamp.
func timeParse(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

var _ proto.Message = (*telemetryv1.ProgramEvent)(nil)
