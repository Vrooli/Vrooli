package eventbus

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPublishUsesDeterministicReceiptEnvelope(t *testing.T) {
	var body string
	var identity string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body = mustRead(t, r)
		identity = r.Header.Get(cliutil.HeaderAgentIdentityToken)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer s.Close()
	r := Receipt{Source: "agent-manager", Target: "plan-manager", Operation: "plans.create", Outcome: "success", StatusCode: 201, Correlation: Correlation{RunID: "run-1"}, SubjectID: "agent-1", IdentityToken: "verified-token"}
	if err := (Client{BaseURL: s.URL}).Publish(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, r.IdempotencyKey()) || !strings.Contains(body, ReceiptEventType) || !strings.Contains(body, "ReceiptData") || !strings.Contains(body, `"subjectId":"agent-1"`) {
		t.Fatalf("unexpected envelope: %s", body)
	}
	if identity != "verified-token" {
		t.Fatalf("identity header = %q", identity)
	}
}

func TestPublishWithoutAgentIdentityOmitsAttribution(t *testing.T) {
	var envelope domain.EventEnvelope
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := mustRead(t, r)
		if err := protojson.Unmarshal([]byte(body), &envelope); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer s.Close()
	if err := (Client{BaseURL: s.URL}).Publish(context.Background(), Receipt{Source: "plan-manager", Target: "plan-manager", Operation: "plans.create", Outcome: "success", StatusCode: http.StatusCreated}); err != nil {
		t.Fatal(err)
	}
	if envelope.Attribution != nil {
		t.Fatalf("anonymous receipt attribution = %+v, want nil", envelope.Attribution)
	}
}

func TestDisabledClientIsNonBlocking(t *testing.T) {
	if err := (Client{}).Publish(context.Background(), Receipt{}); err != nil {
		t.Fatal(err)
	}
}

func TestClientEnabled(t *testing.T) {
	if (Client{}).Enabled() {
		t.Fatal("zero client must be disabled")
	}
	if !(Client{BaseURL: "http://events"}).Enabled() {
		t.Fatal("configured client must be enabled")
	}
}

func mustRead(t *testing.T, r *http.Request) string {
	t.Helper()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
