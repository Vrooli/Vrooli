package eventbus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DomainEvent struct {
	Source    string
	EventType string
	Payload   map[string]any
	Occurred  time.Time
}

func (c Client) PublishDomainEvent(ctx context.Context, event DomainEvent) error {
	if !c.Enabled() {
		return nil
	}
	payload, err := structpb.NewStruct(event.Payload)
	if err != nil {
		return fmt.Errorf("event payload: %w", err)
	}
	data, err := anypb.New(payload)
	if err != nil {
		return fmt.Errorf("pack event payload: %w", err)
	}
	when := event.Occurred
	if when.IsZero() {
		when = time.Now().UTC()
	}
	seed, _ := json.Marshal(event)
	sum := sha256.Sum256(seed)
	envelope := &domain.EventEnvelope{
		EventId: hex.EncodeToString(sum[:]), EventType: event.EventType, OccurredAt: timestamppb.New(when),
		Source: &domain.EventSource{Scenario: event.Source, ActorKind: "system"}, Data: data,
	}
	body, err := protojson.Marshal(envelope)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL(), "/")+"/api/v1/events", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusConflict {
		return fmt.Errorf("vrooli-events domain event publish: %s", response.Status)
	}
	return nil
}
