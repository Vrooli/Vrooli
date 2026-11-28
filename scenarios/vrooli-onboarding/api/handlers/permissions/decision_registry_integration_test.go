package permissions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	conversationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
	conversationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations/conversations_v1connect"
	permissionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions"
	permissionsv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/permissions/permissionsv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/decisionpermissions"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
)

type decisionHubFixture struct {
	mu               sync.Mutex
	listCalls        int
	readCalls        int
	currentRevision  string
	flipOnListCall   int
	paginate         bool
	wrongOwnerForBob bool
	seenBearer       []string
}

func newDecisionHub(t *testing.T, fixture *decisionHubFixture) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(conversationsconnect.ConversationsServiceListPendingDecisionReferencesProcedure, connect.NewUnaryHandler(
		conversationsconnect.ConversationsServiceListPendingDecisionReferencesProcedure,
		func(_ context.Context, req *connect.Request[conversationsv1.ListPendingDecisionReferencesRequest]) (*connect.Response[conversationsv1.ListPendingDecisionReferencesResponse], error) {
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			token := req.Header().Get("Authorization")
			fixture.seenBearer = append(fixture.seenBearer, token)
			if token != "Bearer alice" && token != "Bearer bob" {
				return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("owner proof required"))
			}
			fixture.listCalls++
			revision := fixture.currentRevision
			owner := strings.TrimPrefix(token, "Bearer ")
			if fixture.wrongOwnerForBob && owner == "bob" {
				owner = "alice"
			}
			decisions := []*conversationsv1.PendingDecisionReference{{
				DecisionRef: "ask-1", OwnerId: owner, Source: "treasury",
				SourceEventType: "treasury.refund.decision_requested.v1", Revision: revision, State: "pending",
				CreatedAt: "2026-11-20T10:00:00Z", UpdatedAt: "2026-11-20T10:01:00Z", Urgency: "high",
			}}
			cursor := ""
			if req.Msg.GetCursor() == "page-2" {
				decisions = nil
			}
			if fixture.paginate && req.Msg.GetCursor() == "" && !fixture.wrongOwnerForBob && owner == "alice" {
				cursor = "page-2"
			} else if req.Msg.GetCursor() == "page-2" {
				decisions = append(decisions, &conversationsv1.PendingDecisionReference{
					DecisionRef: "ask-2", OwnerId: owner, Source: "billing",
					SourceEventType: "billing.cancel.decision_requested.v1", Revision: revision, State: "pending",
					CreatedAt: "2026-11-20T10:02:00Z", UpdatedAt: "2026-11-20T10:03:00Z", Deadline: "2026-11-22T10:00:00Z",
				})
				cursor = ""
			}
			response := connect.NewResponse(&conversationsv1.ListPendingDecisionReferencesResponse{Decisions: decisions, NextCursor: cursor})
			if fixture.flipOnListCall == fixture.listCalls {
				fixture.currentRevision = "4"
			}
			return response, nil
		},
	))
	mux.Handle(conversationsconnect.ConversationsServiceReadPendingDecisionReferenceProcedure, connect.NewUnaryHandler(
		conversationsconnect.ConversationsServiceReadPendingDecisionReferenceProcedure,
		func(_ context.Context, req *connect.Request[conversationsv1.ReadPendingDecisionReferenceRequest]) (*connect.Response[conversationsv1.ReadPendingDecisionReferenceResponse], error) {
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			fixture.readCalls++
			token := req.Header().Get("Authorization")
			fixture.seenBearer = append(fixture.seenBearer, token)
			owner := strings.TrimPrefix(token, "Bearer ")
			if owner != "alice" {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
			}
			if req.Msg.GetExpectedRevision() != fixture.currentRevision {
				return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("stale"))
			}
			if req.Msg.GetDecisionRef() != "ask-1" {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("not found"))
			}
			return connect.NewResponse(&conversationsv1.ReadPendingDecisionReferenceResponse{Decision: &conversationsv1.PendingDecisionReference{
				DecisionRef: "ask-1", OwnerId: owner, Source: "treasury", SourceEventType: "treasury.refund.decision_requested.v1",
				Revision: fixture.currentRevision, State: "pending", CreatedAt: "2026-11-20T10:00:00Z", UpdatedAt: "2026-11-20T10:01:00Z", Urgency: "high",
			}}), nil
		},
	))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func decisionHandler(t *testing.T, fixture *decisionHubFixture) (permissionsv1connect.PermissionRegistryServiceHandler, *decisionHubFixture) {
	t.Helper()
	server := newDecisionHub(t, fixture)
	owner := &decisionpermissions.Client{BaseURL: func(context.Context) (string, error) { return server.URL, nil }, HTTP: server.Client()}
	service := operatorstateapi.New(nil, nil, decisionpermissions.New(owner))
	return NewConnectHandler(service), fixture
}

func decisionContext(subject string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true})
}

func ownerRequest[T any](msg *T, token string) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", token)
	return req
}

func TestDecisionPermissionTransportUsesOwnerProjectionForListReadAndReplay(t *testing.T) {
	fixture := &decisionHubFixture{currentRevision: "3", paginate: true}
	handler, _ := decisionHandler(t, fixture)
	ctx := decisionContext("alice")
	listed, err := handler.ListPermissions(ctx, ownerRequest(&permissionsv1.ListPermissionsRequest{Owner: "notification-hub"}, "Bearer alice"))
	if err != nil {
		t.Fatalf("list owner permissions: %v", err)
	}
	entries := listed.Msg.GetPermissions()
	if len(entries) != 2 || entries[0].GetCanWrite() || entries[1].GetCanWrite() {
		t.Fatalf("pending decision descriptors = %+v", entries)
	}
	first := entries[0]
	readRequest := func() *connect.Request[permissionsv1.ReadPermissionRequest] {
		return ownerRequest(&permissionsv1.ReadPermissionRequest{PermissionId: first.GetId(), Owner: "notification-hub"}, "Bearer alice")
	}
	one, err := handler.ReadPermission(ctx, readRequest())
	if err != nil {
		t.Fatalf("read pending reference: %v", err)
	}
	two, err := handler.ReadPermission(ctx, readRequest())
	if err != nil {
		t.Fatalf("replay pending reference read: %v", err)
	}
	if one.Msg.GetRevision() != "3" || string(one.Msg.GetValueJson()) != string(two.Msg.GetValueJson()) || one.Msg.GetValueDigest() != two.Msg.GetValueDigest() {
		t.Fatalf("replay changed source-bound value: first=%+v second=%+v", one.Msg, two.Msg)
	}
	var value map[string]any
	if err := json.Unmarshal(one.Msg.GetValueJson(), &value); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"question", "answer", "answerer", "notes", "context", "requester_ref", "correlation_id", "attachments"} {
		if _, exists := value[field]; exists {
			t.Fatalf("private decision field %q was projected: %s", field, one.Msg.GetValueJson())
		}
	}
	if fixture.readCalls != 2 || len(fixture.seenBearer) == 0 {
		t.Fatalf("owner read calls=%d forwarded proof=%v", fixture.readCalls, fixture.seenBearer)
	}
	for _, token := range fixture.seenBearer {
		if token != "Bearer alice" {
			t.Fatalf("owner proof changed across API composition: %q", token)
		}
	}
	if _, err := handler.WritePermission(ctx, ownerRequest(&permissionsv1.WritePermissionRequest{PermissionId: first.GetId(), Owner: "notification-hub", ExpectedRevision: "3", IdempotencyKey: "resolve", ValueJson: []byte(`{"answer":"approve"}`)}, "Bearer alice")); err == nil {
		t.Fatal("decision write unexpectedly succeeded")
	}
	if fixture.readCalls != 2 {
		t.Fatalf("unsupported write reached owner read API: reads=%d", fixture.readCalls)
	}
}

func TestDecisionPermissionTransportRefusesStaleAndCrossPrincipalOwnerReads(t *testing.T) {
	t.Run("stale revision during read", func(t *testing.T) {
		fixture := &decisionHubFixture{currentRevision: "3", flipOnListCall: 2}
		handler, _ := decisionHandler(t, fixture)
		ctx := decisionContext("alice")
		listed, err := handler.ListPermissions(ctx, ownerRequest(&permissionsv1.ListPermissionsRequest{Owner: "notification-hub"}, "Bearer alice"))
		if err != nil || len(listed.Msg.GetPermissions()) == 0 {
			t.Fatalf("list=%v err=%v", listed, err)
		}
		_, err = handler.ReadPermission(ctx, ownerRequest(&permissionsv1.ReadPermissionRequest{PermissionId: listed.Msg.GetPermissions()[0].GetId(), Owner: "notification-hub"}, "Bearer alice"))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("stale permission read error=%v code=%s", err, connect.CodeOf(err))
		}
	})
	t.Run("owner returns another principal's reference", func(t *testing.T) {
		fixture := &decisionHubFixture{currentRevision: "3", wrongOwnerForBob: true}
		handler, _ := decisionHandler(t, fixture)
		_, err := handler.ListPermissions(decisionContext("bob"), ownerRequest(&permissionsv1.ListPermissionsRequest{Owner: "notification-hub"}, "Bearer bob"))
		if connect.CodeOf(err) != connect.CodeDataLoss {
			t.Fatalf("cross-principal projection error=%v code=%s", err, connect.CodeOf(err))
		}
		if fixture.readCalls != 0 {
			t.Fatalf("cross-principal refusal made %d owner reads", fixture.readCalls)
		}
	})
}
