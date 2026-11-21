package wiring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/supervision"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/eventbus"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

type decisionClientFixture struct {
	enabled bool
	calls   int
	err     error
}

type blockingDecisionClient struct{}

func (blockingDecisionClient) Enabled() bool { return true }
func (blockingDecisionClient) PublishDomainEvent(ctx context.Context, _ eventbus.DomainEvent) error {
	<-ctx.Done()
	return ctx.Err()
}

type deadlineDecisionClient struct{ remaining time.Duration }

func (*deadlineDecisionClient) Enabled() bool { return true }
func (f *deadlineDecisionClient) PublishDomainEvent(ctx context.Context, _ eventbus.DomainEvent) error {
	deadline, ok := ctx.Deadline()
	if ok {
		f.remaining = time.Until(deadline)
	}
	return nil
}

func TestDecisionPublisherUsesConfiguredEventsOwner(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/events" {
			t.Errorf("unexpected event request: %s %s", r.Method, r.URL.Path)
		}
		requests++
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	t.Setenv("VROOLI_EVENTS_API_BASE", server.URL)
	err := newDecisionPublisher().PublishDomainEvent(context.Background(), eventbus.DomainEvent{Source: "agent-manager", EventType: "fixture.event", Payload: map[string]any{"decision_id": "test"}})
	if err != nil || requests != 1 {
		t.Fatalf("configured owner did not receive the event: requests=%d err=%v", requests, err)
	}
}

func TestDecisionDeliveryReconciliationTargetsExactOwnerAndEvent(t *testing.T) {
	var reconciled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.RawQuery == "owner=notification-hub":
			_, _ = w.Write([]byte(`[{"id":17,"name":"notification-hub-decisions","enabled":true,"event_pattern":"**.decision_requested.v1"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/subscriptions/17/deliveries/reconcile":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body) != 1 || body["event_id"] != "event-identity" {
				t.Errorf("unexpected reconciliation: %+v %v", body, err)
			}
			reconciled = true
			w.WriteHeader(http.StatusAccepted)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	publisher := rediscoveringDecisionPublisher{resolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	if err := publisher.ReconcileDecisionDelivery(context.Background(), "event-identity"); err != nil || !reconciled {
		t.Fatalf("reconcile failed: %v", err)
	}
}

func TestDecisionDeliveryReconciliationDoesNotGuessSubscription(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("must not write without exact subscription")
		}
		_, _ = w.Write([]byte(`[{"id":17,"name":"other","enabled":true,"event_pattern":"**.decision_requested.v1"}]`))
	}))
	defer server.Close()
	publisher := rediscoveringDecisionPublisher{resolveURL: func(context.Context) (string, error) { return server.URL, nil }}
	if err := publisher.ReconcileDecisionDelivery(context.Background(), "event-identity"); err == nil {
		t.Fatal("missing exact owner must remain unverified")
	}
}

func (f *decisionClientFixture) Enabled() bool { return f.enabled }
func (f *decisionClientFixture) PublishDomainEvent(_ context.Context, _ eventbus.DomainEvent) error {
	f.calls++
	return f.err
}

func TestDecisionPublisherRediscoversAfterStartupUnavailability(t *testing.T) {
	client := &decisionClientFixture{}
	resolutions := 0
	publisher := rediscoveringDecisionPublisher{resolve: func(context.Context) decisionEventClient {
		resolutions++
		return client
	}}
	event := eventbus.DomainEvent{Source: "agent-manager", EventType: "test"}
	if err := publisher.PublishDomainEvent(context.Background(), event); err == nil || client.calls != 0 {
		t.Fatalf("unavailable bus must fail without reporting publication: err=%v calls=%d", err, client.calls)
	}
	client.enabled = true
	if err := publisher.PublishDomainEvent(context.Background(), event); err != nil || client.calls != 1 || resolutions != 2 {
		t.Fatalf("next attempt must rediscover and publish: err=%v calls=%d resolutions=%d", err, client.calls, resolutions)
	}
	client.err = errors.New("transport failure")
	if err := publisher.PublishDomainEvent(context.Background(), event); !errors.Is(err, client.err) {
		t.Fatalf("transport failure must remain visible: %v", err)
	}
}

func TestDecisionPublisherRespectsCancellation(t *testing.T) {
	publisher := rediscoveringDecisionPublisher{resolve: func(context.Context) decisionEventClient { return blockingDecisionClient{} }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := publisher.PublishDomainEvent(ctx, eventbus.DomainEvent{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("blocked publication must respect owner cancellation: %v", err)
	}
}

func TestDecisionPublisherBoundsUncancelledOwnerRequest(t *testing.T) {
	client := &deadlineDecisionClient{}
	publisher := rediscoveringDecisionPublisher{resolve: func(context.Context) decisionEventClient { return client }}
	if err := publisher.PublishDomainEvent(context.Background(), eventbus.DomainEvent{}); err != nil {
		t.Fatal(err)
	}
	if client.remaining <= 0 || client.remaining > 3*time.Second {
		t.Fatalf("publication deadline must be bounded to three seconds, got %s", client.remaining)
	}
}

func TestDecisionOutboxRecoversThroughSchedulerAfterDatabaseReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "supervision.db")
	openDB := func() *sqlx.DB {
		db, err := sqlx.Connect("sqlite", dbPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(supervision.Schema)); err != nil {
			t.Fatal(err)
		}
		return db
	}
	client := &decisionClientFixture{}
	db := openDB()
	service := supervision.NewEffortService(supervision.NewRepository(db), nil, supervision.EffortDiscoveryConfig{Root: t.TempDir()})
	service.SetDecisionPublisher(rediscoveringDecisionPublisher{resolve: func(context.Context) decisionEventClient { return client }})
	request := &pb.RaiseEffortDecisionRequest{Decision: &pb.EffortDecision{
		GoalHome: "scenarios/agent-manager/docs/internal/goal", DecisionId: "RECOVERY-TEST", Question: "Continue?",
		Options:     []*pb.EffortDecisionOption{{Key: "yes", Label: "Yes"}, {Key: "no", Label: "No"}},
		Recommended: "yes", Urgency: "normal", Deadline: timestamppb.New(time.Now().Add(time.Hour)),
	}}
	created, err := service.RaiseDecision(ctx, request, "fixture-run")
	if err != nil || created.GetPublished() || created.GetDecision().GetPublishError() == "" {
		t.Fatalf("unavailable owner must retain the decision: %+v %v", created, err)
	}
	eventID := created.GetDecision().GetEventId()
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	var received []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event struct {
			EventID string `json:"eventId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Error(err)
		}
		received = append(received, event.EventID)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	t.Setenv("VROOLI_EVENTS_API_BASE", server.URL)
	db = openDB()
	defer db.Close()
	repo := supervision.NewRepository(db)
	restarted := supervision.NewEffortService(repo, nil, supervision.EffortDiscoveryConfig{Root: t.TempDir()})
	restarted.SetDecisionPublisher(newDecisionPublisher())
	scheduler := supervision.NewScheduler(repo, nil, func(err error) { t.Errorf("scheduler recovery: %v", err) })
	scheduler.SetEffortService(restarted)
	if _, err := scheduler.RecoverOnce(ctx); err != nil {
		t.Fatal(err)
	}
	listed, err := restarted.ListDecisions(ctx, &pb.ListEffortDecisionsRequest{GoalHome: request.Decision.GoalHome})
	if err != nil || len(listed.GetDecisions()) != 1 || listed.GetDecisions()[0].GetPublishedAt() == nil {
		t.Fatalf("recovered decision not recorded as published: %+v %v", listed, err)
	}
	if len(received) != 1 || received[0] != eventID {
		t.Fatalf("recovery must publish the original identity once: got %v want %s", received, eventID)
	}
}
