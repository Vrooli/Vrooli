package facts

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/validation/validation_v1connect"
)

type candidateReceiptResolverFunc func(context.Context, string) (string, error)

func (f candidateReceiptResolverFunc) ResolveScenarioURLDefault(ctx context.Context, scenario string) (string, error) {
	return f(ctx, scenario)
}

type candidateReceiptMatchStub struct {
	validationconnect.UnimplementedValidationServiceHandler
	request  *validationv1.GetCandidateReceiptMatchRequest
	response *validationv1.GetCandidateReceiptMatchResponse
	err      error
}

func (s *candidateReceiptMatchStub) GetCandidateReceiptMatch(_ context.Context, req *connect.Request[validationv1.GetCandidateReceiptMatchRequest]) (*connect.Response[validationv1.GetCandidateReceiptMatchResponse], error) {
	s.request = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.response), nil
}

func TestCandidateReceiptMatchHTTPHandlerPreservesOwnerVerdictAndIdentity(t *testing.T) {
	owner := &candidateReceiptMatchStub{response: &validationv1.GetCandidateReceiptMatchResponse{
		Status:    validationv1.CandidateReceiptMatchStatus_CANDIDATE_RECEIPT_MATCH_STATUS_EXACT_MATCH,
		ReceiptId: "receipt-1", Reason: "exact source identity match",
	}}
	path, handler := validationconnect.NewValidationServiceHandler(owner)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != path+"GetCandidateReceiptMatch" {
			t.Errorf("request path = %q, want %q", r.URL.Path, path+"GetCandidateReceiptMatch")
		}
		handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	requestBody := `{"receiptId":"receipt-1","candidate":{"candidateId":"code-facts:invariant-candidate:demo:runtime_guard:api/server.go","scenario":"demo","relativePath":"api/server.go","sourceLine":42,"sourceFileSha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/invariants/candidate-receipt-match", strings.NewReader(requestBody))
	response := httptest.NewRecorder()
	CandidateReceiptMatchHTTPHandler(candidateReceiptResolverFunc(func(_ context.Context, scenario string) (string, error) {
		if scenario != "test-genie" {
			t.Errorf("resolver scenario = %q, want test-genie", scenario)
		}
		return server.URL, nil
	}), server.Client())(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}
	var body candidateReceiptMatchResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Status != "EXACT_MATCH" || body.ReceiptID != "receipt-1" || body.Candidate.CandidateID != "code-facts:invariant-candidate:demo:runtime_guard:api/server.go" || body.Reason != "exact source identity match" {
		t.Fatalf("owner verdict or identity was lost: %+v", body)
	}
	if owner.request == nil || owner.request.GetCandidate().GetSourceLine() != 42 || owner.request.GetReceiptId() != "receipt-1" {
		t.Fatalf("candidate receipt request was not forwarded exactly: %+v", owner.request)
	}
}

func TestCandidateReceiptMatchHTTPHandlerKeepsUnknownAndUnavailableDistinct(t *testing.T) {
	owner := &candidateReceiptMatchStub{response: &validationv1.GetCandidateReceiptMatchResponse{
		Status: validationv1.CandidateReceiptMatchStatus_CANDIDATE_RECEIPT_MATCH_STATUS_UNKNOWN, ReceiptId: "receipt-legacy", Reason: "legacy receipt has no candidate link",
	}}
	path, handler := validationconnect.NewValidationServiceHandler(owner)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	defer server.Close()
	requestBody := `{"receiptId":"receipt-legacy","candidate":{"candidateId":"code-facts:invariant-candidate:demo:runtime_guard:api/server.go","scenario":"demo","relativePath":"api/server.go","sourceLine":42,"sourceFileSha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}`

	unknown := httptest.NewRecorder()
	CandidateReceiptMatchHTTPHandler(candidateReceiptResolverFunc(func(context.Context, string) (string, error) { return server.URL, nil }), server.Client())(
		unknown, httptest.NewRequest(http.MethodPost, "/api/v1/invariants/candidate-receipt-match", strings.NewReader(requestBody)))
	if unknown.Code != http.StatusOK || !strings.Contains(unknown.Body.String(), `"status":"UNKNOWN"`) {
		t.Fatalf("owner UNKNOWN was not preserved: %d %s", unknown.Code, unknown.Body.String())
	}

	unavailable := httptest.NewRecorder()
	CandidateReceiptMatchHTTPHandler(candidateReceiptResolverFunc(func(context.Context, string) (string, error) { return "", errors.New("not running") }), server.Client())(
		unavailable, httptest.NewRequest(http.MethodPost, "/api/v1/invariants/candidate-receipt-match", strings.NewReader(requestBody)))
	if unavailable.Code != http.StatusServiceUnavailable || strings.Contains(unavailable.Body.String(), `"status":"UNKNOWN"`) {
		t.Fatalf("resolver failure should remain unavailable: %d %s", unavailable.Code, unavailable.Body.String())
	}

	if path == "" {
		t.Fatal("Test Genie generated route was empty")
	}
}

func TestCandidateReceiptMatchHTTPHandlerRejectsIncompleteAndWrongMethod(t *testing.T) {
	handler := CandidateReceiptMatchHTTPHandler(nil, nil)
	for _, tc := range []struct {
		name   string
		method string
		body   string
		want   int
	}{
		{name: "method", method: http.MethodGet, want: http.StatusMethodNotAllowed},
		{name: "incomplete candidate", method: http.MethodPost, body: `{"receiptId":"receipt-1","candidate":{"candidateId":"candidate"}}`, want: http.StatusBadRequest},
		{name: "extra value", method: http.MethodPost, body: `{} {}`, want: http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, "/api/v1/invariants/candidate-receipt-match", strings.NewReader(tc.body))
			handler(response, request)
			if response.Code != tc.want {
				t.Fatalf("status = %d, want %d body=%s", response.Code, tc.want, response.Body.String())
			}
		})
	}
}
