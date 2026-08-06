package telemetry

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	"github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestEmittedEventIsRetrievableFromPlatformBus(t *testing.T) { // [REQ:PRT-P0-006]
	var mu sync.Mutex
	var received *domain.EventEnvelope
	bus := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/events" {
			http.NotFound(w, r)
			return
		}
		var envelope domain.EventEnvelope
		if err := protojson.Unmarshal(readBody(t, r), &envelope); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = &envelope
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer bus.Close()

	event := &telemetryv1.ProgramEvent{EventId: "event-1", Kind: telemetryv1.EventKind_PROGRAM_FAILED, ProgramId: "program-1", FailureLocation: "line 4"}
	if err := (HTTPPublisher{BaseURL: bus.URL}).Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	envelope := received
	mu.Unlock()
	if envelope == nil || envelope.GetEventType() != EventType {
		t.Fatalf("received envelope=%v", envelope)
	}
	decoded := &telemetryv1.ProgramEvent{}
	if err := anypb.UnmarshalTo(envelope.GetData(), decoded, proto.UnmarshalOptions{}); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(decoded, event) {
		t.Fatalf("decoded event=%v, want=%v", decoded, event)
	}
}

func readBody(t *testing.T, r *http.Request) []byte {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
