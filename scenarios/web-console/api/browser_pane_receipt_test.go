package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/servicetoken"
	"web-console/internal/sessionstore"
)

func TestBrowserPaneReceiptResolverChecksWebConsoleOwnerAndUsesDedicatedAMIdentity(t *testing.T) {
	var calls int
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/browser/pane/resolve" || r.Method != http.MethodPost {
			t.Errorf("unexpected AM route: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer web-console-service-token" {
			t.Errorf("forwarded or wrong authorization: %q", r.Header.Get("Authorization"))
		}
		var request paneReceiptIssueRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.OwnerSubject != "human-1" || request.ThreadID != "thread-from-store" || request.SessionID != "bas-session" || request.Role != "observe" {
			t.Errorf("AM request=%+v", request)
		}
		_ = json.NewEncoder(w).Encode(paneReceiptIssueResponse{SessionID: "bas-session", Role: "observe", Receipt: "opaque-server-only-receipt"})
	}))
	defer owner.Close()

	store := browserPaneTestSessionStore{meta: sessionstore.Metadata{ID: "web-session", Owner: "human-1", Status: sessionstore.StatusLive, NativeThreadID: "thread-from-store"}}
	resolver := newWebConsoleBrowserPaneReceiptResolver(store)
	resolver.resolve = func(context.Context) (string, error) { return owner.URL, nil }
	resolver.tokens = webConsoleBrowserPaneRole.NewTokens("web-console", func(context.Context) (string, error) { return "synthetic-credential", nil }, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "web-console-service-token", ExpiresAt: time.Now().Add(time.Hour), Subject: webConsoleBrowserPaneRole.Principal("web-console"), Audience: webConsoleBrowserPaneRole.Audience, Scopes: []string{webConsoleBrowserPaneRole.Scope}}, nil
	}))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Verified: true, Source: identity.SourcePersonalLocal, Scopes: []string{"web-console:write"}, ExpiresAt: time.Now().Add(time.Hour)})
	receipt, err := resolver.Issue(ctx, "web-session", "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c", "bas-session", "observe")
	if err != nil || receipt != "opaque-server-only-receipt" || calls != 1 {
		t.Fatalf("receipt=%q calls=%d err=%v", receipt, calls, err)
	}

	wrongOwner := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-2", Verified: true, Source: identity.SourcePersonalLocal, ExpiresAt: time.Now().Add(time.Hour)})
	if _, err := resolver.Issue(wrongOwner, "web-session", "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c", "bas-session", "observe"); err == nil || calls != 1 {
		t.Fatalf("wrong owner got receipt or reached AM: calls=%d err=%v", calls, err)
	}
	store.meta.ArchivedAt = time.Now()
	resolver.store = store
	if _, err := resolver.Issue(ctx, "web-session", "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c", "bas-session", "observe"); err == nil || calls != 1 {
		t.Fatalf("archived session got receipt or reached AM: calls=%d err=%v", calls, err)
	}
}

func TestBrowserPaneHandleKeepsRunAndBASIDsServerSide(t *testing.T) {
	var listBody, issueBody []byte
	var ownerCalls []string
	terminalRun := false
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ownerCalls = append(ownerCalls, r.URL.Path)
		if r.Header.Get("Authorization") != "Bearer pane-service" {
			t.Errorf("wrong service authorization: %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v1/browser/pane/sessions":
			listBody, _ = io.ReadAll(r.Body)
			if terminalRun {
				w.WriteHeader(http.StatusGone)
				_, _ = w.Write([]byte(`{"error":"browser_run_not_active"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(paneSessionInternalResponse{RunID: "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c", ThreadID: "thread-private", Sessions: []paneSessionInternal{{SessionID: "bas-private", Mode: "managed", CanControl: true}}})
		case "/api/v1/browser/pane/yield", "/api/v1/browser/pane/resolve":
			issueBody, _ = io.ReadAll(r.Body)
			var issue paneReceiptIssueRequest
			_ = json.Unmarshal(issueBody, &issue)
			response := paneReceiptIssueResponse{SessionID: "bas-private", Role: issue.Role, Receipt: "receipt-private"}
			if r.URL.Path == "/api/v1/browser/pane/yield" {
				response.YieldReceipt = "server-only-yield"
				response.ExpiresAt = time.Now().Add(time.Minute)
			}
			_ = json.NewEncoder(w).Encode(response)
		case "/api/v1/browser/pane/return":
			var returned paneReceiptIssueRequest
			if err := json.NewDecoder(r.Body).Decode(&returned); err != nil {
				t.Fatal(err)
			}
			if returned.RunID != "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c" || returned.ThreadID != "thread-private" || returned.SessionID != "bas-private" || returned.YieldReceipt != "server-only-yield" || returned.Role != "observe" {
				t.Errorf("return binding=%+v", returned)
			}
			_ = json.NewEncoder(w).Encode(map[string]bool{"returned": true})
		default:
			t.Errorf("unexpected AM path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer owner.Close()
	store := browserPaneTestSessionStore{meta: sessionstore.Metadata{ID: "wc-private", Owner: "human-1", Status: sessionstore.StatusLive, NativeThreadID: "thread-private"}}
	resolver := newWebConsoleBrowserPaneReceiptResolver(store)
	resolver.resolve = func(context.Context) (string, error) { return owner.URL, nil }
	resolver.tokens = webConsoleBrowserPaneRole.NewTokens("web-console", func(context.Context) (string, error) { return "synthetic", nil }, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "pane-service", ExpiresAt: time.Now().Add(time.Hour), Subject: webConsoleBrowserPaneRole.Principal("web-console"), Audience: webConsoleBrowserPaneRole.Audience, Scopes: []string{webConsoleBrowserPaneRole.Scope}}, nil
	}))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Verified: true, Source: identity.SourcePersonalLocal, ExpiresAt: time.Now().Add(time.Hour)})
	listed, err := resolver.List(ctx, "wc-private")
	if err != nil || len(listed.Sessions) != 1 {
		t.Fatalf("list=%+v err=%v", listed, err)
	}
	terminalRun = true
	finished, err := resolver.List(ctx, "wc-private")
	if err != nil || !finished.RunFinished || len(finished.Sessions) != 0 {
		t.Fatalf("finished run discovery=%+v err=%v", finished, err)
	}
	terminalRun = false
	public, _ := json.Marshal(listed)
	for _, secret := range []string{"d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c", "thread-private", "bas-private"} {
		if strings.Contains(string(public), secret) {
			t.Fatalf("browser response leaked %q: %s", secret, public)
		}
	}
	if strings.Contains(string(listBody), "owner_subject") != true || !strings.Contains(string(listBody), "wc-private") {
		t.Fatalf("expected exact server-owned AM selector, body=%s", listBody)
	}
	resolver.handlesMu.Lock()
	record := resolver.handles[listed.Sessions[0].Handle]
	record.CanControl = false
	resolver.handles[listed.Sessions[0].Handle] = record
	resolver.handlesMu.Unlock()
	if _, err := resolver.IssueHandle(ctx, listed.Sessions[0].Handle, "control"); err == nil {
		t.Fatal("control role issued when the owner listed observe-only")
	}
	record.CanControl = true
	record.ControlIssuePending = true
	resolver.handlesMu.Lock()
	resolver.handles[listed.Sessions[0].Handle] = record
	resolver.handlesMu.Unlock()
	ownerCallsBeforePending := len(ownerCalls)
	if _, err := resolver.IssueHandle(ctx, listed.Sessions[0].Handle, "control"); err == nil || len(ownerCalls) != ownerCallsBeforePending {
		t.Fatalf("in-flight control yield reached AM: calls=%v err=%v", ownerCalls, err)
	}
	record.ControlIssuePending = false
	resolver.handlesMu.Lock()
	resolver.handles[listed.Sessions[0].Handle] = record
	resolver.handlesMu.Unlock()
	control, err := resolver.IssueHandle(ctx, listed.Sessions[0].Handle, "control")
	if err != nil || control.SessionID != "bas-private" || control.Receipt != "receipt-private" || !strings.Contains(string(issueBody), `"role":"control"`) {
		t.Fatalf("owner-attested takeover failed: grant=%+v err=%v request=%s", control, err, issueBody)
	}
	controlCalls := len(ownerCalls)
	if _, err := resolver.IssueHandle(ctx, listed.Sessions[0].Handle, "control"); err == nil || len(ownerCalls) != controlCalls {
		t.Fatalf("overlapping control yield reached AM: calls=%v err=%v", ownerCalls, err)
	}
	issueBody = nil
	grant, err := resolver.IssueHandle(ctx, listed.Sessions[0].Handle, "observe")
	if err != nil || grant.SessionID != "bas-private" || grant.Receipt != "receipt-private" {
		t.Fatalf("grant=%+v err=%v", grant, err)
	}
	if !strings.Contains(string(issueBody), "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c") || !strings.Contains(string(issueBody), "bas-private") || !strings.Contains(string(issueBody), "thread-private") {
		t.Fatalf("server did not resolve handle to exact owner binding: %s", issueBody)
	}
	if strings.Contains(string(issueBody), "receipt-private") || strings.Contains(string(issueBody), "lease") {
		t.Fatalf("browser handle resolution leaked authority: %s", issueBody)
	}
	if err := resolver.ReturnHandle(ctx, listed.Sessions[0].Handle); err != nil {
		t.Fatalf("return exact parked run after observer receipt rotation: %v", err)
	}
	if len(ownerCalls) < 4 || ownerCalls[len(ownerCalls)-2] != "/api/v1/browser/pane/resolve" || ownerCalls[len(ownerCalls)-1] != "/api/v1/browser/pane/return" {
		t.Fatalf("AM was not woken after the observer exchange: %v", ownerCalls)
	}
}

func TestBrowserPaneSessionsRouteRequiresVerifiedHuman(t *testing.T) {
	resolver := newWebConsoleBrowserPaneReceiptResolver(browserPaneTestSessionStore{meta: sessionstore.Metadata{ID: "wc-1", Owner: "human-1", Status: sessionstore.StatusLive, NativeThreadID: "thread-1"}})
	handler := browserPaneSessionsHandler(resolver)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/pane/sessions", strings.NewReader(`{"web_console_session_id":"wc-1"}`))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("unverified HTTP caller received pane candidates: status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestBrowserPaneSessionsRoutePreservesFinishedRunState(t *testing.T) {
	handler := browserPaneSessionsHandler(&fixedBrowserPaneReceiptResolver{
		list: paneSessionListResponse{Sessions: []paneSessionSummary{}, RunFinished: true},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/pane/sessions", strings.NewReader(`{"web_console_session_id":"wc-1"}`))
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	var body paneSessionListResponse
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &body) != nil || !body.RunFinished || len(body.Sessions) != 0 {
		t.Fatalf("terminal owner state was not preserved: status=%d body=%s decoded=%+v", res.Code, res.Body.String(), body)
	}
}

func TestBrowserPaneHandleExpiresAndCapacityFailsClosed(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Verified: true, Source: identity.SourcePersonalLocal, Scopes: []string{"web-console:write"}, ExpiresAt: time.Now().Add(time.Hour)})
	resolver := newWebConsoleBrowserPaneReceiptResolver(nil)
	resolver.handles["expired"] = paneHandleRecord{Owner: "human-1", WebSessionID: "wc-1", Expires: time.Now().Add(-time.Second)}
	if _, err := resolver.IssueHandle(ctx, "expired", "observe"); err == nil {
		t.Fatal("expired handle unexpectedly issued a receipt")
	}
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/browser/pane/sessions" {
			t.Errorf("unexpected route %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(paneSessionInternalResponse{RunID: "d28e41fa-c82a-4e6f-8c3b-0e6c50d9dc6c", ThreadID: "thread-1", Sessions: []paneSessionInternal{{SessionID: "bas-1", Mode: "managed"}}})
	}))
	defer owner.Close()
	store := browserPaneTestSessionStore{meta: sessionstore.Metadata{ID: "wc-1", Owner: "human-1", Status: sessionstore.StatusLive, NativeThreadID: "thread-1"}}
	resolver = newWebConsoleBrowserPaneReceiptResolver(store)
	resolver.resolve = func(context.Context) (string, error) { return owner.URL, nil }
	resolver.tokens = webConsoleBrowserPaneRole.NewTokens("web-console", func(context.Context) (string, error) { return "synthetic", nil }, servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "pane-service", ExpiresAt: time.Now().Add(time.Hour), Subject: webConsoleBrowserPaneRole.Principal("web-console"), Audience: webConsoleBrowserPaneRole.Audience, Scopes: []string{webConsoleBrowserPaneRole.Scope}}, nil
	}))
	for i := 0; i < 2048; i++ {
		resolver.handles[fmt.Sprintf("occupied-%04d", i)] = paneHandleRecord{Owner: "human-1", WebSessionID: "wc-1", Expires: time.Now().Add(time.Hour)}
	}
	if _, err := resolver.List(ctx, "wc-1"); err == nil {
		t.Fatal("handle capacity exhaustion returned a pane list")
	}
}
