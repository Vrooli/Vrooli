package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/principalrole"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"web-console/internal/sessionstore"
)

var webConsoleBrowserPaneRole = principalrole.NewRole("scenario-authenticator:agent-manager", "agent-manager:browser-pane.delegate",
	"agent-manager", "-browser-pane-delegator", "browser pane delegator")

type paneReceiptIssueRequest struct {
	RunID               string `json:"run_id"`
	WebConsoleSessionID string `json:"web_console_session_id"`
	ThreadID            string `json:"thread_id"`
	SessionID           string `json:"session_id"`
	Role                string `json:"role"`
	OwnerSubject        string `json:"owner_subject"`
	YieldReceipt        string `json:"yield_receipt,omitempty"`
}

type paneReceiptIssueResponse struct {
	SessionID    string    `json:"session_id"`
	Role         string    `json:"role"`
	Receipt      string    `json:"receipt"`
	YieldReceipt string    `json:"yield_receipt,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

type paneSessionSummary struct {
	Handle     string `json:"pane_handle"`
	Mode       string `json:"mode"`
	CanControl bool   `json:"can_control"`
}
type paneSessionListResponse struct {
	Sessions   []paneSessionSummary `json:"sessions"`
	RunFinished bool                `json:"run_finished,omitempty"`
}
type paneSessionInternal struct {
	SessionID  string `json:"session_id"`
	Mode       string `json:"mode"`
	CanControl bool   `json:"can_control"`
}
type paneSessionInternalResponse struct {
	RunID    string                `json:"run_id"`
	ThreadID string                `json:"thread_id"`
	Sessions []paneSessionInternal `json:"sessions"`
}
type paneHandleRecord struct {
	WebSessionID, Owner, RunID, ThreadID, SessionID, Mode string
	CanControl                                            bool
	YieldReceipt                                          string
	ControlIssuePending                                   bool
	Expires                                               time.Time
}
type paneReceiptGrant struct{ SessionID, Receipt string }

type browserPaneReceiptResolver interface {
	Issue(context.Context, string, string, string, string) (string, error)
	List(context.Context, string) (paneSessionListResponse, error)
	IssueHandle(context.Context, string, string) (paneReceiptGrant, error)
	ReturnHandle(context.Context, string) error
}

func browserPaneSessionsHandler(resolver browserPaneReceiptResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			WebConsoleSessionID string `json:"web_console_session_id"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
		dec.DisallowUnknownFields()
		if dec.Decode(&body) != nil || dec.Decode(new(any)) != io.EOF {
			http.Error(w, "invalid browser pane request", http.StatusBadRequest)
			return
		}
		body.WebConsoleSessionID = strings.TrimSpace(body.WebConsoleSessionID)
		if body.WebConsoleSessionID == "" {
			http.Error(w, "invalid browser pane session", http.StatusBadRequest)
			return
		}
		result, err := resolver.List(r.Context(), body.WebConsoleSessionID)
		if err != nil {
			http.Error(w, "authorized browser session unavailable", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}
}

func browserPaneReturnHandler(resolver browserPaneReceiptResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Handle string `json:"pane_handle"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<10))
		dec.DisallowUnknownFields()
		if dec.Decode(&body) != nil || dec.Decode(new(any)) != io.EOF || !validBrowserPaneHandle(body.Handle) {
			http.Error(w, "invalid browser pane handle", http.StatusBadRequest)
			return
		}
		if err := resolver.ReturnHandle(r.Context(), body.Handle); err != nil {
			http.Error(w, "browser control return refused", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"returned":true}`)
	}
}

type webConsoleBrowserPaneReceiptResolver struct {
	store     sessionstore.Store
	resolve   func(context.Context) (string, error)
	client    *http.Client
	tokens    *servicetoken.Tokens
	handlesMu sync.Mutex
	handles   map[string]paneHandleRecord
}

func newWebConsoleBrowserPaneReceiptResolver(store sessionstore.Store) *webConsoleBrowserPaneReceiptResolver {
	credential := func(ctx context.Context) (string, error) {
		authority, err := credentialauthority.Default()
		if err != nil {
			return "", err
		}
		id, err := credentialauthority.ParseIdentity(webConsoleBrowserPaneRole.CredentialIdentity("web-console"))
		if err != nil {
			return "", err
		}
		return authority.Require(id, "credential")
	}
	return &webConsoleBrowserPaneReceiptResolver{store: store, handles: make(map[string]paneHandleRecord), resolve: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	}, client: &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConns: 8, IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 5 * time.Second}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		tokens: webConsoleBrowserPaneRole.NewTokens("web-console", credential)}
}

// Issue proves ownership through Web Console's live session store, derives the
// thread from that owner record, then calls AM with Web Console's dedicated
// service identity. No inbound bearer, thread assertion or owner header is
// forwarded. BAS receipt material is returned only to the Web Console server.
func (r *webConsoleBrowserPaneReceiptResolver) Issue(ctx context.Context, webSessionID, runIDRaw, basSessionID, role string) (string, error) {
	result, err := r.issue(ctx, webSessionID, runIDRaw, basSessionID, role)
	return result.Receipt, err
}

func (r *webConsoleBrowserPaneReceiptResolver) issue(ctx context.Context, webSessionID, runIDRaw, basSessionID, role string) (paneReceiptIssueResponse, error) {
	if r == nil || r.store == nil || r.resolve == nil || r.client == nil || r.tokens == nil {
		return paneReceiptIssueResponse{}, errors.New("browser pane owner unavailable")
	}
	principal, err := authn.RequireIndependentHuman(ctx)
	if err != nil || !principal.IsHuman() || principal.ExpiresAt.IsZero() || !time.Now().Before(principal.ExpiresAt) {
		return paneReceiptIssueResponse{}, errors.New("browser pane human session is not verified")
	}
	meta, err := r.store.Get(ctx, webSessionID)
	if err != nil || meta.ID != webSessionID || strings.TrimSpace(meta.Owner) == "" || meta.Owner != principal.OwnerSubject() || meta.Status != sessionstore.StatusLive || !meta.ArchivedAt.IsZero() {
		return paneReceiptIssueResponse{}, errors.New("browser pane Web Console session is not live and human-owned")
	}
	threadID := strings.TrimSpace(meta.NativeThreadID)
	if threadID == "" {
		threadID = strings.TrimSpace(meta.AgentSessionID)
	}
	if threadID == "" {
		return paneReceiptIssueResponse{}, errors.New("browser pane session has no owner-recorded thread")
	}
	if _, err := uuid.Parse(strings.TrimSpace(runIDRaw)); err != nil || basSessionID == "" || (role != "observe" && role != "control") {
		return paneReceiptIssueResponse{}, errors.New("browser pane selector is invalid")
	}
	base, err := r.resolve(ctx)
	if err != nil {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane owner unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane owner unavailable")
	}
	token, err := r.tokens.Token(ctx)
	if err != nil {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane delegation identity unavailable")
	}
	payload, err := json.Marshal(paneReceiptIssueRequest{RunID: runIDRaw, WebConsoleSessionID: webSessionID, ThreadID: threadID,
		SessionID: basSessionID, Role: role, OwnerSubject: principal.OwnerSubject()})
	if err != nil {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane request invalid")
	}
	endpoint := *parsed
	path := "/api/v1/browser/pane/resolve"
	if role == "control" {
		path = "/api/v1/browser/pane/yield"
	}
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane request unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := r.client.Do(request)
	if err != nil {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane request failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return paneReceiptIssueResponse{}, fmt.Errorf("Agent Manager refused browser pane (HTTP %d)", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (8<<10)+1))
	if err != nil || len(body) > 8<<10 {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane response exceeded limit")
	}
	var result paneReceiptIssueResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane response invalid")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane response has trailing data")
	}
	if result.SessionID != basSessionID || result.Role != role || strings.TrimSpace(result.Receipt) == "" || (role == "control" && (strings.TrimSpace(result.YieldReceipt) == "" || result.ExpiresAt.IsZero())) {
		return paneReceiptIssueResponse{}, errors.New("Agent Manager pane response binding mismatch")
	}
	return result, nil
}

func (r *webConsoleBrowserPaneReceiptResolver) List(ctx context.Context, webSessionID string) (paneSessionListResponse, error) {
	if r == nil || r.store == nil || r.resolve == nil || r.client == nil || r.tokens == nil {
		return paneSessionListResponse{}, errors.New("browser pane owner unavailable")
	}
	principal, err := authn.RequireIndependentHuman(ctx)
	if err != nil || !principal.IsHuman() || principal.ExpiresAt.IsZero() || !time.Now().Before(principal.ExpiresAt) {
		return paneSessionListResponse{}, errors.New("browser pane human session is not verified")
	}
	meta, err := r.store.Get(ctx, webSessionID)
	if err != nil || meta.ID != webSessionID || strings.TrimSpace(meta.Owner) == "" || meta.Owner != principal.OwnerSubject() || meta.Status != sessionstore.StatusLive || !meta.ArchivedAt.IsZero() {
		return paneSessionListResponse{}, errors.New("browser pane Web Console session is not live and human-owned")
	}
	thread := strings.TrimSpace(meta.NativeThreadID)
	if thread == "" {
		thread = strings.TrimSpace(meta.AgentSessionID)
	}
	if thread == "" {
		return paneSessionListResponse{}, errors.New("browser pane session has no owner-recorded thread")
	}
	base, err := r.resolve(ctx)
	if err != nil {
		return paneSessionListResponse{}, errors.New("Agent Manager pane owner unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return paneSessionListResponse{}, errors.New("Agent Manager pane owner unavailable")
	}
	token, err := r.tokens.Token(ctx)
	if err != nil {
		return paneSessionListResponse{}, errors.New("Agent Manager pane delegation identity unavailable")
	}
	payload, err := json.Marshal(map[string]string{"web_console_session_id": webSessionID, "thread_id": thread, "owner_subject": principal.OwnerSubject()})
	if err != nil {
		return paneSessionListResponse{}, errors.New("Agent Manager pane request invalid")
	}
	endpoint := *parsed
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/v1/browser/pane/sessions"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return paneSessionListResponse{}, errors.New("Agent Manager pane request unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := r.client.Do(request)
	if err != nil {
		return paneSessionListResponse{}, errors.New("Agent Manager pane request failed")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusGone {
		// AM's exact owner/thread lookup confirms there is no longer a running
		// eligible run. Do not collapse this into generic authorization failure:
		// Web Console must remove the cached-handle reconnect affordance.
		return paneSessionListResponse{Sessions: []paneSessionSummary{}, RunFinished: true}, nil
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return paneSessionListResponse{}, fmt.Errorf("Agent Manager refused browser pane (HTTP %d)", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (16<<10)+1))
	if err != nil || len(body) > 16<<10 {
		return paneSessionListResponse{}, errors.New("Agent Manager pane response exceeded limit")
	}
	var result paneSessionInternalResponse
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&result) != nil {
		return paneSessionListResponse{}, errors.New("Agent Manager pane response invalid")
	}
	var trailing any
	if dec.Decode(&trailing) != io.EOF {
		return paneSessionListResponse{}, errors.New("Agent Manager pane response has trailing data")
	}
	if result.ThreadID != thread || strings.TrimSpace(result.RunID) == "" || len(result.Sessions) > 64 {
		return paneSessionListResponse{}, errors.New("Agent Manager pane session binding mismatch")
	}
	public := paneSessionListResponse{Sessions: make([]paneSessionSummary, 0, len(result.Sessions))}
	for _, session := range result.Sessions {
		if strings.TrimSpace(session.SessionID) == "" || strings.TrimSpace(session.Mode) == "" {
			return paneSessionListResponse{}, errors.New("Agent Manager pane session invalid")
		}
		raw := make([]byte, 24)
		if _, err := rand.Read(raw); err != nil {
			return paneSessionListResponse{}, errors.New("browser pane handle unavailable")
		}
		handle := hex.EncodeToString(raw)
		r.handlesMu.Lock()
		for key, cached := range r.handles {
			if cached.Expires.Before(time.Now()) {
				delete(r.handles, key)
			}
		}
		if len(r.handles) >= 2048 {
			r.handlesMu.Unlock()
			return paneSessionListResponse{}, errors.New("browser pane handle capacity reached")
		}
		r.handles[handle] = paneHandleRecord{WebSessionID: webSessionID, Owner: principal.OwnerSubject(), RunID: result.RunID, ThreadID: thread, SessionID: session.SessionID, Mode: session.Mode, CanControl: session.CanControl, Expires: time.Now().Add(2 * time.Minute)}
		r.handlesMu.Unlock()
		public.Sessions = append(public.Sessions, paneSessionSummary{Handle: handle, Mode: session.Mode, CanControl: session.CanControl})
	}
	return public, nil
}

// IssueHandle resolves a short-lived Web Console alias server-side. Neither
// browser/run IDs nor the one-use BAS receipt cross the browser boundary.
func (r *webConsoleBrowserPaneReceiptResolver) IssueHandle(ctx context.Context, handle, role string) (paneReceiptGrant, error) {
	if role != "observe" && role != "control" {
		return paneReceiptGrant{}, errors.New("browser pane role is invalid")
	}
	principal, err := authn.RequireIndependentHuman(ctx)
	if err != nil || !principal.IsHuman() || principal.ExpiresAt.IsZero() || !time.Now().Before(principal.ExpiresAt) {
		return paneReceiptGrant{}, errors.New("browser pane human session is not verified")
	}
	r.handlesMu.Lock()
	record, ok := r.handles[handle]
	if ok && (record.Expires.Before(time.Now()) || record.Owner != principal.OwnerSubject()) {
		delete(r.handles, handle)
		ok = false
	}
	r.handlesMu.Unlock()
	if !ok {
		return paneReceiptGrant{}, errors.New("browser pane handle is missing or expired")
	}
	if role == "control" && !record.CanControl {
		return paneReceiptGrant{}, errors.New("browser pane controller permission is unavailable")
	}
	if role == "control" {
		// A control yield is a one-use handoff. Reserve this alias before the
		// owner call so concurrent clicks cannot issue overlapping yield receipts.
		r.handlesMu.Lock()
		current, exists := r.handles[handle]
		if !exists || current.RunID != record.RunID || current.SessionID != record.SessionID || current.Owner != record.Owner || !current.Expires.After(time.Now()) {
			r.handlesMu.Unlock()
			return paneReceiptGrant{}, errors.New("browser pane handle changed before control yield")
		}
		if current.ControlIssuePending || strings.TrimSpace(current.YieldReceipt) != "" {
			r.handlesMu.Unlock()
			return paneReceiptGrant{}, errors.New("browser pane control yield is already active")
		}
		current.ControlIssuePending = true
		r.handles[handle] = current
		r.handlesMu.Unlock()
		defer r.finishControlIssue(handle, record)
	}
	webSessionID := record.WebSessionID
	meta, err := r.store.Get(ctx, webSessionID)
	if err != nil || meta.ID != webSessionID || meta.Owner != principal.OwnerSubject() || meta.Status != sessionstore.StatusLive || !meta.ArchivedAt.IsZero() {
		return paneReceiptGrant{}, errors.New("browser pane Web Console session is not live and human-owned")
	}
	thread := strings.TrimSpace(meta.NativeThreadID)
	if thread == "" {
		thread = strings.TrimSpace(meta.AgentSessionID)
	}
	if thread != record.ThreadID {
		return paneReceiptGrant{}, errors.New("browser pane thread association changed")
	}
	issued, err := r.issue(ctx, webSessionID, record.RunID, record.SessionID, role)
	if err != nil {
		return paneReceiptGrant{}, err
	}
	if role == "control" {
		if strings.TrimSpace(issued.YieldReceipt) == "" || issued.ExpiresAt.IsZero() {
			return paneReceiptGrant{}, errors.New("Agent Manager did not return an exclusive control yield")
		}
		r.handlesMu.Lock()
		current, exists := r.handles[handle]
		if !exists || current.RunID != record.RunID || current.SessionID != record.SessionID || current.Owner != record.Owner || !current.Expires.After(time.Now()) {
			r.handlesMu.Unlock()
			return paneReceiptGrant{}, errors.New("browser pane handle changed during control yield")
		}
		current.YieldReceipt = issued.YieldReceipt
		current.ControlIssuePending = false
		if issued.ExpiresAt.Before(current.Expires) {
			current.Expires = issued.ExpiresAt
		}
		r.handles[handle] = current
		r.handlesMu.Unlock()
	}
	return paneReceiptGrant{SessionID: record.SessionID, Receipt: issued.Receipt}, nil
}

func (r *webConsoleBrowserPaneReceiptResolver) finishControlIssue(handle string, record paneHandleRecord) {
	r.handlesMu.Lock()
	defer r.handlesMu.Unlock()
	if current, exists := r.handles[handle]; exists && current.RunID == record.RunID && current.SessionID == record.SessionID && current.Owner == record.Owner && current.ControlIssuePending {
		current.ControlIssuePending = false
		r.handles[handle] = current
	}
}

// ReturnHandle resumes the exact parked run only after the stream proxy has
// attached the observer role, which rotates BAS's one-use controller cookie.
func (r *webConsoleBrowserPaneReceiptResolver) ReturnHandle(ctx context.Context, handle string) error {
	if r == nil || r.store == nil || r.resolve == nil || r.client == nil || r.tokens == nil {
		return errors.New("browser pane owner unavailable")
	}
	principal, err := authn.RequireIndependentHuman(ctx)
	if err != nil || !principal.IsHuman() || principal.ExpiresAt.IsZero() || !time.Now().Before(principal.ExpiresAt) {
		return errors.New("browser pane human session is not verified")
	}
	r.handlesMu.Lock()
	record, ok := r.handles[handle]
	if ok && (!record.Expires.After(time.Now()) || record.Owner != principal.OwnerSubject() || strings.TrimSpace(record.YieldReceipt) == "") {
		if !record.Expires.After(time.Now()) {
			delete(r.handles, handle)
		}
		ok = false
	}
	r.handlesMu.Unlock()
	if !ok {
		return errors.New("browser pane control return is unavailable")
	}
	meta, err := r.store.Get(ctx, record.WebSessionID)
	if err != nil || meta.ID != record.WebSessionID || meta.Owner != principal.OwnerSubject() || meta.Status != sessionstore.StatusLive || !meta.ArchivedAt.IsZero() {
		return errors.New("browser pane Web Console session is not live and human-owned")
	}
	thread := strings.TrimSpace(meta.NativeThreadID)
	if thread == "" {
		thread = strings.TrimSpace(meta.AgentSessionID)
	}
	if thread != record.ThreadID {
		return errors.New("browser pane thread association changed")
	}
	base, err := r.resolve(ctx)
	if err != nil {
		return errors.New("Agent Manager pane owner unavailable")
	}
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return errors.New("Agent Manager pane owner unavailable")
	}
	token, err := r.tokens.Token(ctx)
	if err != nil {
		return errors.New("Agent Manager pane delegation identity unavailable")
	}
	payload, err := json.Marshal(paneReceiptIssueRequest{RunID: record.RunID, WebConsoleSessionID: record.WebSessionID, ThreadID: thread, SessionID: record.SessionID, Role: "observe", OwnerSubject: principal.OwnerSubject(), YieldReceipt: record.YieldReceipt})
	if err != nil {
		return errors.New("Agent Manager pane request invalid")
	}
	endpoint := *parsed
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/v1/browser/pane/return"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return errors.New("Agent Manager pane request unavailable")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := r.client.Do(request)
	if err != nil {
		return errors.New("Agent Manager browser control return failed")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Agent Manager refused browser control return (HTTP %d)", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (2<<10)+1))
	if err != nil || len(body) > 2<<10 {
		return errors.New("Agent Manager browser control return response exceeded limit")
	}
	var returned struct {
		Returned bool `json:"returned"`
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&returned) != nil || !returned.Returned || decoder.Decode(new(any)) != io.EOF {
		return errors.New("Agent Manager browser control return response was not confirmed")
	}
	r.handlesMu.Lock()
	if current, exists := r.handles[handle]; exists && current.YieldReceipt == record.YieldReceipt {
		current.YieldReceipt = ""
		r.handles[handle] = current
	}
	r.handlesMu.Unlock()
	return nil
}
