package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"

	conv "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	convconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
)

// approvalsFacade is the server-side home of ALL notification-hub access for the
// Approvals inbox. The browser calls Web Console's own REST routes
// (GET /api/v1/approvals, POST /api/v1/approvals/answer); this facade translates
// them into Connect calls to notification-hub's ConversationsService.
//
// Why a Go Connect client and not a UI-side Connect client or a REST
// passthrough:
//   - notification-hub's ConversationsService is Connect-RPC ONLY, so a generic
//     REST path-prefix proxy cannot reach it.
//   - Centralizing the hub call here is what lets the facade inject the
//     operator's Authorization bearer AND the Cloudflare Access assertion onto
//     the OUTBOUND call, so the hub attributes an answer to the verified human
//     (answered_by) — Phase 7. A UI that spoke Connect straight to the hub would
//     bypass this one auth/header-injection point.
//
// Decoupling: this imports notification-hub's generated proto (a shared
// artifact) and NOTHING from agent-manager. agent-manager is one of the
// scenarios that RAISE asks, but the inbox only ever reads/answers them through
// this hub surface, keyed by the ask's domain-neutral `source`. See
// docs/internal/SEAMS.md and api/approvals_facade_test.go.
type approvalsFacade struct {
	resolve notificationsURLResolver
	client  *http.Client
}

func newApprovalsFacade(resolve notificationsURLResolver) *approvalsFacade {
	return &approvalsFacade{resolve: resolve, client: &http.Client{Timeout: 30 * time.Second}}
}

// hubClient builds a ConversationsService Connect client for the resolved hub,
// whose outbound requests carry the operator's identity copied from the inbound
// browser request. The copy happens via a per-request RoundTripper so every RPC
// — not just the next one — carries the headers.
func (f *approvalsFacade) hubClient(r *http.Request) (convconnect.ConversationsServiceClient, bool) {
	base, err := f.resolve(r.Context())
	if err != nil || base == "" {
		return nil, false
	}
	httpClient := &http.Client{
		Timeout: f.client.Timeout,
		Transport: &identityForwardingTransport{
			base:          http.DefaultTransport,
			authorization: r.Header.Get("Authorization"),
			deviceID:      r.Header.Get("X-Vrooli-Device-Id"),
			// Load-bearing (Phase 7): the hub reads this to attribute answers to
			// the actual signed-in human rather than the configured operator.
			accessAssertion: r.Header.Get("Cf-Access-Jwt-Assertion"),
			requestID:       r.Header.Get("X-Request-ID"),
		},
	}
	return convconnect.NewConversationsServiceClient(httpClient, base), true
}

// identityForwardingTransport copies the operator identity headers onto every
// outbound hub request. An empty header is omitted so the hub's bearer/cookie
// fallback still applies when Cloudflare Access is not in front.
type identityForwardingTransport struct {
	base            http.RoundTripper
	authorization   string
	deviceID        string
	accessAssertion string
	requestID       string
}

func (t *identityForwardingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.authorization != "" {
		req.Header.Set("Authorization", t.authorization)
	}
	if t.deviceID != "" {
		req.Header.Set("X-Vrooli-Device-Id", t.deviceID)
	}
	if t.accessAssertion != "" {
		req.Header.Set("Cf-Access-Jwt-Assertion", t.accessAssertion)
	}
	if t.requestID != "" {
		req.Header.Set("X-Request-ID", t.requestID)
	}
	return t.base.RoundTrip(req)
}

// approvalView is the JSON shape the UI's REST client reads. It is the hub Ask,
// flattened to the fields the inbox renders. The UI derives source/kind from
// these; this facade does not parse the event type.
type approvalView struct {
	ID                   string               `json:"id"`
	Question             string               `json:"question"`
	Options              []approvalOptionView `json:"options"`
	Recommended          string               `json:"recommended"`
	RecommendationReason string               `json:"recommendation_reason"`
	DefaultAnswer        string               `json:"default_answer"`
	Reversible           bool                 `json:"reversible"`
	Urgency              string               `json:"urgency"`
	ContextURL           string               `json:"context_url"`
	BigPicture           string               `json:"big_picture"`
	TechnicalDetails     string               `json:"technical_details"`
	Deadline             string               `json:"deadline"`
	State                string               `json:"state"`
	Answer               string               `json:"answer"`
	AnswerLabel          string               `json:"answer_label"`
	Note                 string               `json:"note"`
	AnsweredBy           string               `json:"answered_by"`
	AnsweredAt           string               `json:"answered_at"`
	Late                 bool                 `json:"late"`
	DefaultAppliedAt     string               `json:"default_applied_at"`
	ResolvedAt           string               `json:"resolved_at"`
	CreatedAt            string               `json:"created_at"`
	UpdatedAt            string               `json:"updated_at"`
	Source               string               `json:"source"`
	SourceEventType      string               `json:"source_event_type"`
	RequesterRef         string               `json:"requester_ref"`
}

type approvalOptionView struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

func askToView(ask *conv.Ask) approvalView {
	options := make([]approvalOptionView, 0, len(ask.GetOptions()))
	for _, o := range ask.GetOptions() {
		options = append(options, approvalOptionView{Key: o.GetKey(), Label: o.GetLabel()})
	}
	return approvalView{
		ID: ask.GetId(), Question: ask.GetQuestion(), Options: options,
		Recommended: ask.GetRecommended(), RecommendationReason: ask.GetRecommendationReason(),
		DefaultAnswer: ask.GetDefaultAnswer(), Reversible: ask.GetReversible(), Urgency: ask.GetUrgency(),
		ContextURL: ask.GetContextUrl(), Deadline: ask.GetDeadline(), State: ask.GetState(),
		BigPicture: ask.GetBigPicture(), TechnicalDetails: ask.GetTechnicalDetails(),
		Answer: ask.GetAnswer(), AnswerLabel: ask.GetAnswerLabel(), Note: ask.GetNote(),
		AnsweredBy: ask.GetAnsweredBy(), AnsweredAt: ask.GetAnsweredAt(), Late: ask.GetLate(),
		DefaultAppliedAt: ask.GetDefaultAppliedAt(), ResolvedAt: ask.GetResolvedAt(),
		CreatedAt: ask.GetCreatedAt(), UpdatedAt: ask.GetUpdatedAt(),
		Source: ask.GetSource(), SourceEventType: ask.GetSourceEventType(), RequesterRef: ask.GetRequesterRef(),
	}
}

// handleList serves GET /api/v1/approvals?open_only=&limit=. It lists the
// operator's asks from the hub (open-only by default) and returns them as JSON.
func (f *approvalsFacade) handleList(w http.ResponseWriter, r *http.Request) {
	client, ok := f.hubClient(r)
	if !ok {
		notificationsUnavailable(w)
		return
	}
	openOnly := r.URL.Query().Get("open_only") != "false" // default: open only
	limit := int32(100)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := parsePositiveInt(raw); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	resp, err := client.ListAsks(r.Context(), connect.NewRequest(&conv.ListAsksRequest{OpenOnly: openOnly, Limit: limit}))
	if err != nil {
		writeApprovalsError(w, err)
		return
	}
	out := make([]approvalView, 0, len(resp.Msg.GetAsks()))
	for _, ask := range resp.Msg.GetAsks() {
		out = append(out, askToView(ask))
	}
	writeJSON(w, http.StatusOK, map[string]any{"approvals": out})
}

// handleGet serves GET /api/v1/approvals/{id}. It reads one decision by id —
// the deep-link landing path, where the decision may be resolved or outside
// the current list filter. It uses the hub's read-only GetAsk RPC.
func (f *approvalsFacade) handleGet(w http.ResponseWriter, r *http.Request) {
	askID := mux.Vars(r)["id"]
	if askID == "" {
		http.Error(w, `{"error":"decision id is required"}`, http.StatusBadRequest)
		return
	}
	client, ok := f.hubClient(r)
	if !ok {
		notificationsUnavailable(w)
		return
	}
	resp, err := client.GetAsk(r.Context(), connect.NewRequest(&conv.GetAskRequest{AskId: askID}))
	if err != nil {
		writeApprovalsError(w, err)
		return
	}
	if resp.Msg.GetAsk() == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "decision not found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approval": askToView(resp.Msg.GetAsk())})
}

// handleAnswer serves POST /api/v1/approvals/answer. It routes approve, reject,
// and every other option through the hub's ONE Answer RPC; the chosen option
// key is the `answer`, and the optional operator note rides along.
func (f *approvalsFacade) handleAnswer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AskID  string `json:"ask_id"`
		Answer string `json:"answer"`
		Note   string `json:"note"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid answer body"}`, http.StatusBadRequest)
		return
	}
	if body.AskID == "" || body.Answer == "" {
		http.Error(w, `{"error":"ask_id and answer are required"}`, http.StatusBadRequest)
		return
	}
	client, ok := f.hubClient(r)
	if !ok {
		notificationsUnavailable(w)
		return
	}
	resp, err := client.Answer(r.Context(), connect.NewRequest(&conv.AnswerRequest{AskId: body.AskID, Answer: body.Answer, Note: body.Note}))
	if err != nil {
		writeApprovalsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ask_id":      resp.Msg.GetAskId(),
		"answer":      resp.Msg.GetAnswer(),
		"answered_at": resp.Msg.GetAnsweredAt(),
		"late":        resp.Msg.GetLate(),
	})
}

// writeApprovalsError maps a Connect error to an HTTP status. An unreachable hub
// (CodeUnavailable) becomes a clean JSON 503 so the inbox renders its
// "unavailable" state instead of crashing; the rest map to sensible statuses.
func writeApprovalsError(w http.ResponseWriter, err error) {
	switch connect.CodeOf(err) {
	case connect.CodeUnavailable:
		notificationsUnavailable(w)
	case connect.CodeUnauthenticated, connect.CodePermissionDenied:
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "not authorized for approvals"})
	case connect.CodeNotFound:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "decision not found"})
	case connect.CodeFailedPrecondition:
		writeJSON(w, http.StatusConflict, map[string]any{"error": "decision already resolved"})
	case connect.CodeInvalidArgument:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
	default:
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "approvals request failed"})
	}
}

func parsePositiveInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errInvalidInt
		}
		n = n*10 + int(c-'0')
		if n > 1_000_000 {
			break
		}
	}
	return n, nil
}

var errInvalidInt = &approvalsParseError{"not an integer"}

type approvalsParseError struct{ msg string }

func (e *approvalsParseError) Error() string { return e.msg }

// Compile-time guard: the facade never imports an agent-manager client. If a
// future edit adds one, this file's import block changes and
// TestApprovalsInboxDoesNotImportAgentManager fails.
var _ = context.Background
