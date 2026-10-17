package main

import (
	"context"
	"encoding/json"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	conv "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	convconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// fakeHub is a stand-in notification-hub ConversationsService. It records the
// identity headers the facade forwarded and returns canned asks.
type fakeHub struct {
	convconnect.UnimplementedConversationsServiceHandler
	gotAuth   string
	gotAccess string
	gotAnswer *conv.AnswerRequest
	openOnly  bool
}

func (h *fakeHub) ListAsks(_ context.Context, req *connect.Request[conv.ListAsksRequest]) (*connect.Response[conv.ListAsksResponse], error) {
	h.gotAuth = req.Header().Get("Authorization")
	h.gotAccess = req.Header().Get("Cf-Access-Jwt-Assertion")
	h.openOnly = req.Msg.GetOpenOnly()
	return connect.NewResponse(&conv.ListAsksResponse{Asks: []*conv.Ask{{
		Id: "ask-1", Question: "Promote release?", Source: "deployment-manager",
		SourceEventType: "deployment-manager.release.decision_requested.v1",
		Reversible:      false, State: "pending", Recommended: "promote",
		Options: []*conv.AskOption{{Key: "promote", Label: "Promote"}, {Key: "reject", Label: "Reject"}},
	}}}), nil
}

func (h *fakeHub) GetAsk(_ context.Context, req *connect.Request[conv.GetAskRequest]) (*connect.Response[conv.GetAskResponse], error) {
	return connect.NewResponse(&conv.GetAskResponse{Ask: &conv.Ask{
		Id: req.Msg.GetAskId(), Question: "Grant scope?", Source: "agent-manager",
		SourceEventType: "agent-manager.authorization.decision_requested.v1", State: "answered", Answer: "deny",
	}}), nil
}

func (h *fakeHub) Answer(_ context.Context, req *connect.Request[conv.AnswerRequest]) (*connect.Response[conv.AnswerResponse], error) {
	h.gotAuth = req.Header().Get("Authorization")
	h.gotAccess = req.Header().Get("Cf-Access-Jwt-Assertion")
	h.gotAnswer = req.Msg
	return connect.NewResponse(&conv.AnswerResponse{AskId: req.Msg.GetAskId(), Answer: req.Msg.GetAnswer(), AnsweredAt: "2026-10-16T00:00:00Z"}), nil
}

func newFakeHubServer(t *testing.T, hub *fakeHub) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path, handler := convconnect.NewConversationsServiceHandler(hub)
	mux.Handle(path, handler)
	return httptest.NewServer(mux)
}

func TestApprovalsFacadeListTranslatesConnect(t *testing.T) {
	hub := &fakeHub{}
	server := newFakeHubServer(t, hub)
	defer server.Close()
	facade := newApprovalsFacade(func(context.Context) (string, error) { return server.URL, nil })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil)
	facade.handleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if !hub.openOnly {
		t.Fatal("list should default to open_only=true")
	}
	var body struct {
		Approvals []approvalView `json:"approvals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Approvals) != 1 || body.Approvals[0].ID != "ask-1" || body.Approvals[0].Source != "deployment-manager" {
		t.Fatalf("unexpected approvals: %+v", body.Approvals)
	}
}

func TestApprovalsFacadeAnswerForwardsIdentityAndAnswer(t *testing.T) {
	hub := &fakeHub{}
	server := newFakeHubServer(t, hub)
	defer server.Close()
	facade := newApprovalsFacade(func(context.Context) (string, error) { return server.URL, nil })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/answer", strings.NewReader(`{"ask_id":"ask-1","answer":"promote","note":"ship it"}`))
	req.Header.Set("Authorization", "Bearer owner-token")
	req.Header.Set("Cf-Access-Jwt-Assertion", "eyJhbGciOiJSUzI1NiJ9.payload.sig")
	facade.handleAnswer(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if hub.gotAnswer.GetAskId() != "ask-1" || hub.gotAnswer.GetAnswer() != "promote" || hub.gotAnswer.GetNote() != "ship it" {
		t.Fatalf("answer not forwarded: %+v", hub.gotAnswer)
	}
	// Load-bearing: the OUTBOUND Connect call to the hub carried the operator
	// bearer AND the Cloudflare Access assertion, so the hub attributes the
	// answer to the verified human (Phase 7). This is the whole reason the
	// facade speaks Connect server-side instead of the UI talking to the hub.
	if hub.gotAuth != "Bearer owner-token" {
		t.Fatalf("Authorization not forwarded to hub: %q", hub.gotAuth)
	}
	if hub.gotAccess != "eyJhbGciOiJSUzI1NiJ9.payload.sig" {
		t.Fatalf("Cf-Access-Jwt-Assertion not forwarded to hub: %q", hub.gotAccess)
	}
}

func TestApprovalsFacadeGetByID(t *testing.T) {
	hub := &fakeHub{}
	server := newFakeHubServer(t, hub)
	defer server.Close()
	facade := newApprovalsFacade(func(context.Context) (string, error) { return server.URL, nil })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals/ask-9", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "ask-9"})
	facade.handleGet(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	var body struct {
		Approval approvalView `json:"approval"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Approval.ID != "ask-9" || body.Approval.Source != "agent-manager" {
		t.Fatalf("unexpected approval: %+v", body.Approval)
	}
}

func TestApprovalsFacadeAnswerRejectsEmptyBody(t *testing.T) {
	facade := newApprovalsFacade(func(context.Context) (string, error) { return "http://127.0.0.1:1", nil })
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/approvals/answer", strings.NewReader(`{}`))
	facade.handleAnswer(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty answer, got %d", rec.Code)
	}
}

func TestApprovalsFacadeHubUnresolvedReturns503(t *testing.T) {
	facade := newApprovalsFacade(func(context.Context) (string, error) { return "", errInvalidInt })
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/approvals", nil)
	facade.handleList(rec, req)
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "unavailable") {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
}

// TestApprovalsInboxDoesNotImportAgentManager is the Go-side decoupling
// assertion. The approvals facade reaches human-approval asks ONLY through
// notification-hub's ConversationsService; it must never import an
// agent-manager client, even though agent-manager raises some of those asks.
// A regression that wired the inbox to agent-manager would import an
// agent-manager Go package somewhere in this module and fail here. (The UI-side
// counterpart is api/__tests__/approvals.decoupling.test.ts.)
func TestApprovalsInboxDoesNotImportAgentManager(t *testing.T) {
	forbidden := regexp.MustCompile(`agent-manager`)
	files, err := walkModuleGoFiles(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		fset := token.NewFileSet()
		parsed, parseErr := parser.ParseFile(fset, file, data, parser.ImportsOnly)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", file, parseErr)
		}
		for _, imp := range parsed.Imports {
			path, uqErr := strconv.Unquote(imp.Path.Value)
			if uqErr != nil {
				continue
			}
			if forbidden.MatchString(path) {
				t.Errorf("%s imports an agent-manager package %q — the approvals inbox must facade notification-hub only", file, path)
			}
		}
	}
}
