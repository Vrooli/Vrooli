package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func canonicalReceiptBody(t *testing.T, attributed bool) []byte {
	t.Helper()
	projection, err := structpb.NewStruct(map[string]any{"plan.id": "plan-1"})
	if err != nil {
		t.Fatal(err)
	}
	data, err := anypb.New(&domain.ReceiptData{Outcome: "success", StatusCode: 201, Projection: projection})
	if err != nil {
		t.Fatal(err)
	}
	eventID := "receipt-provenance-anonymous"
	if attributed {
		eventID = "receipt-provenance-agent"
	}
	env := &domain.EventEnvelope{EventId: eventID, EventType: receiptEventType, OccurredAt: timestamppb.Now(), Source: &domain.EventSource{Scenario: "plan-manager", ActorKind: "system"}, Target: &domain.EventTarget{Scenario: "plan-manager", Operation: "POST /plans/CreatePlan", Protocol: "connect"}, Data: data}
	if attributed {
		env.Correlation = &domain.EventCorrelation{AgentRunId: "run-1"}
		env.Attribution = &domain.EventAttribution{SubjectKind: "agent", SubjectId: "agent-1", Verified: true}
	}
	body, err := protojson.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestReceiptIngestionAllowsAnonymousAndVerifiesSuppliedAgentIdentity(t *testing.T) {
	s, _ := newTestServer(t)
	handler := provenance.Middleware(provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		if token != "valid" {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-1", ProfileKey: "agent-1"}}, nil
	}))(s.routes())
	for _, tc := range []struct {
		name, token string
		attributed  bool
		want        int
	}{{name: "absent", want: http.StatusAccepted}, {name: "invalid", token: "invalid", want: http.StatusUnauthorized}, {name: "verified", token: "valid", attributed: true, want: http.StatusAccepted}} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(canonicalReceiptBody(t, tc.attributed)))
			if tc.token != "" {
				req.Header.Set(cliutil.HeaderAgentIdentityToken, tc.token)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d want=%d: %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

func TestReceiptIngestionRejectsUnverifiedClaimedAgentAttribution(t *testing.T) {
	s, _ := newTestServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", bytes.NewReader(canonicalReceiptBody(t, true)))
	rec := httptest.NewRecorder()
	s.handleIngest(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want=%d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
