package agentaccess

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/driver"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestWebSocketSelectedTabBridgeBindsLeaseAndTypedOperations(t *testing.T) {
	attachedCh := make(chan AttachedTab, 1)
	attachErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{Subprotocols: []string{bridgeSubprotocol}, CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
		if err != nil {
			attachErr <- err
			return
		}
		attached, err := (WebSocketSelectedTabBridge{}).Attach(context.Background(), PairRequest{LeaseReceipt: "receipt-1"}, conn)
		if err != nil {
			attachErr <- err
			return
		}
		attachedCh <- attached
		<-attached.Done()
	}))
	defer server.Close()

	dialer := websocket.Dialer{Subprotocols: []string{bridgeSubprotocol}}
	url := "ws" + server.URL[len("http"):]
	client, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_ = client.WriteJSON(map[string]string{"type": "lease", "lease_receipt": "receipt-1"})
	var accepted map[string]string
	if err := client.ReadJSON(&accepted); err != nil || accepted["type"] != "lease_accepted" {
		t.Fatalf("lease reply = %#v, %v", accepted, err)
	}
	var attached AttachedTab
	select {
	case attached = <-attachedCh:
	case err := <-attachErr:
		t.Fatal(err)
	}
	defer attached.Close()

	go func() {
		for {
			var command map[string]json.RawMessage
			if err := client.ReadJSON(&command); err != nil {
				return
			}
			var kind, requestID string
			_ = json.Unmarshal(command["type"], &kind)
			_ = json.Unmarshal(command["request_id"], &requestID)
			switch kind {
			case "scope_check":
				_ = client.WriteJSON(map[string]string{"type": "scope_checked", "request_id": requestID, "url": "https://example.test/account"})
			case "observe":
				_ = client.WriteJSON(map[string]any{"type": "observed", "request_id": requestID, "selected_tab": map[string]string{"url": "https://example.test/account", "title": "Account", "text": "signed in"}})
			case "act":
				_ = client.WriteJSON(map[string]string{"type": "action_complete", "request_id": requestID})
			case "handoff":
				_ = client.WriteJSON(map[string]string{"type": "handoff_complete", "request_id": requestID, "role": "observe"})
			case "revoke":
				return
			}
		}
	}()
	currentURL, err := attached.CurrentURL(context.Background())
	if err != nil || currentURL != "https://example.test/account" {
		t.Fatalf("CurrentURL() = %q, %v", currentURL, err)
	}
	observation, err := attached.Observe(context.Background())
	if err != nil || observation.Text != "signed in" {
		t.Fatalf("Observe() = %#v, %v", observation, err)
	}
	if err := attached.Act(context.Background(), TabAction{Type: "click", Selector: "#continue"}); err != nil {
		t.Fatal(err)
	}
	if err := attached.Handoff(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestWebSocketSelectedTabBridgeRejectsWrongLeaseReceipt(t *testing.T) {
	attachErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{Subprotocols: []string{bridgeSubprotocol}, CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
		if err != nil {
			attachErr <- err
			return
		}
		defer conn.Close()
		_, err = (WebSocketSelectedTabBridge{}).Attach(context.Background(), PairRequest{LeaseReceipt: "owner-value"}, conn)
		attachErr <- err
	}))
	defer server.Close()
	dialer := websocket.Dialer{Subprotocols: []string{bridgeSubprotocol}}
	conn, _, err := dialer.Dial("ws"+server.URL[len("http"):], nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.WriteJSON(map[string]string{"type": "lease", "lease_receipt": "attacker-value"})
	if err := <-attachErr; err == nil {
		t.Fatal("wrong lease receipt was accepted")
	}
}

func TestWebSocketSelectedTabInventoryReturnsOwnerBoundPermissionEvidence(t *testing.T) {
	attachedCh := make(chan AttachedTab, 1)
	attachErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{Subprotocols: []string{bridgeSubprotocol}, CheckOrigin: func(*http.Request) bool { return true }}).Upgrade(w, r, nil)
		if err != nil {
			attachErr <- err
			return
		}
		attached, err := (WebSocketSelectedTabBridge{}).Attach(context.Background(), PairRequest{LeaseReceipt: "receipt-effective-permissions"}, conn)
		if err != nil {
			attachErr <- err
			return
		}
		attachedCh <- attached
		<-attached.Done()
	}))
	defer server.Close()
	dialer := websocket.Dialer{Subprotocols: []string{bridgeSubprotocol}}
	client, _, err := dialer.Dial("ws"+server.URL[len("http"):], nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	_ = client.WriteJSON(map[string]string{"type": "lease", "lease_receipt": "receipt-effective-permissions"})
	var accepted map[string]string
	if err := client.ReadJSON(&accepted); err != nil || accepted["type"] != "lease_accepted" {
		t.Fatalf("lease reply = %#v, %v", accepted, err)
	}
	var attached AttachedTab
	select {
	case attached = <-attachedCh:
	case err := <-attachErr:
		t.Fatal(err)
	case <-time.After(time.Second):
		t.Fatal("bridge did not accept the exact lease")
	}
	defer attached.Close()
	broker, entry := newSelectedTabInventoryBroker(t, attached, "https://example.test/public/start")
	go func() {
		var request map[string]json.RawMessage
		if client.ReadJSON(&request) != nil {
			return
		}
		var kind, requestID string
		_ = json.Unmarshal(request["type"], &kind)
		_ = json.Unmarshal(request["request_id"], &requestID)
		if kind != "extension_inventory" || requestID == "" {
			return
		}
		_ = client.WriteJSON(map[string]any{"type": "extension_inventory", "request_id": requestID, "observed_at": time.Now().UTC(), "extensions": []any{
			map[string]any{"id": "abcdefghijklmnopabcdefghijklmnop", "name": "Approved", "enabled": true, "permissions": []string{"storage"}, "hostPermissions": []string{"https://example.test/*"}, "permission_evidence": map[string]string{"status": "effective", "source": "chrome.management.getAll", "scope": "grant_set_only", "per_tab_site_access": "not_measured"}},
		}})
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	inventory, err := broker.SelectedTabExtensions(withBridgeContextCaller(ctx, entry.callerID), entry.sessionID)
	if err != nil || len(inventory.Extensions) != 1 || inventory.ObservedAt.IsZero() {
		t.Fatalf("inventory=%#v err=%v", inventory, err)
	}
	item := inventory.Extensions[0]
	if inventory.Binding.SessionID != entry.sessionID || inventory.Binding.ProfileID != entry.profile.ProfileID || inventory.Binding.OwnerID != entry.ownerID || inventory.Binding.LeaseID != entry.leaseID || inventory.Binding.CallerID != entry.callerID {
		t.Fatalf("inventory was not bound to its live selected-tab owner: %#v", inventory.Binding)
	}
	if item.PermissionEvidence.Status != sessionpermission.ExtensionPermissionEvidenceEffective || item.PermissionEvidence.Source != sessionpermission.ExtensionPermissionSourceChromeManagement || item.PermissionEvidence.Scope != sessionpermission.ExtensionPermissionScopeGrantSetOnly || item.PermissionEvidence.PerTabSiteAccess != sessionpermission.ExtensionPerTabSiteAccessNotMeasured || len(item.Permissions) != 1 || item.Permissions[0] != "storage" || item.ArtifactVerified || item.ArtifactSHA256 != "" {
		t.Fatalf("bridge dropped or changed browser-reported active permission evidence: %#v", item)
	}
}

type snapshotAttachedTab struct {
	response     AttachedExtensionInventory
	observedURL  string
	reads        atomic.Int64
	observations atomic.Int64
	revoked      atomic.Bool
	revokeOnRead bool
	started      chan struct{}
	release      chan struct{}
}

func (*snapshotAttachedTab) CurrentURL(context.Context) (string, error) {
	return "https://example.test/public/start", nil
}
func (t *snapshotAttachedTab) Observe(context.Context) (TabObservation, error) {
	t.observations.Add(1)
	url := t.observedURL
	if url == "" {
		url = "https://example.test/public/start"
	}
	return TabObservation{URL: url, Title: "observed", Text: "sensitive page text"}, nil
}
func (*snapshotAttachedTab) Act(context.Context, TabAction) error { return nil }
func (t *snapshotAttachedTab) ExtensionInventory(ctx context.Context) (AttachedExtensionInventory, error) {
	t.reads.Add(1)
	if t.started != nil {
		select {
		case t.started <- struct{}{}:
		default:
		}
	}
	if t.release != nil {
		select {
		case <-t.release:
		case <-ctx.Done():
			return AttachedExtensionInventory{}, ctx.Err()
		}
	}
	if t.revokeOnRead {
		t.revoked.Store(true)
	}
	return t.response, nil
}
func (*snapshotAttachedTab) Handoff(context.Context) error { return nil }
func (t *snapshotAttachedTab) Close() error                { t.revoked.Store(true); return nil }
func (*snapshotAttachedTab) Done() <-chan struct{}         { return nil }
func (t *snapshotAttachedTab) Revoked() bool               { return t.revoked.Load() }

type exactOriginConsent struct {
	origin           string
	calls            atomic.Int64
	revokeAfterFirst bool
	revoked          atomic.Bool
}

func (v *exactOriginConsent) VerifyBrowserTabConsent(_ context.Context, _ string, req sessionpermission.ActionRequest) error {
	call := v.calls.Add(1)
	u, err := url.Parse(req.CurrentURL)
	if err != nil || u.Scheme+"://"+u.Host != v.origin {
		return errors.New("origin is outside selected-tab consent")
	}
	if v.revoked.Load() || (v.revokeAfterFirst && call > 1) {
		return errors.New("selected-tab consent was revoked")
	}
	if v.revokeAfterFirst && call == 1 {
		v.revoked.Store(true)
	}
	return nil
}

func TestSelectedTabObservationRechecksConsentForObservedOrigin(t *testing.T) {
	profile := sessionpermission.Profile{
		ID: "observe-profile", ProfileID: "observe-profile", Effect: sessionpermission.EffectObserve,
		Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached",
		URLAllow: []string{"https://example.test/*", "https://other.test/*"}, View: true, MaxActions: 1,
	}
	consent := &exactOriginConsent{origin: "https://example.test"}
	policy, err := sessionpermission.NewWithVerifiers(profile, nil, consent)
	if err != nil {
		t.Fatal(err)
	}
	caller := bridgeCaller("run:observe-owner")
	tab := &snapshotAttachedTab{observedURL: "https://other.test/private"}
	e := &entry{
		sessionID: "observe-session", permissionRef: profile.ID, profile: profile, policy: policy,
		backend: string(sessionpermission.BackendUserAttached), leaseID: "observe-lease", ownerID: "observe-owner",
		callerID: string(caller), tabID: "9", tabURL: "https://example.test/public/start", consentReceipt: "consent-receipt",
		attachedTab: tab,
	}
	b := &Broker{
		profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: bridgeContextCaller{},
		entries: map[string]*entry{e.sessionID: e},
	}
	ctx := withBridgeContextCaller(context.Background(), e.callerID)
	if _, err := b.Observe(withBridgeContextCaller(context.Background(), "run:wrong-owner"), e.sessionID, ObserveRequest{}); err == nil || tab.observations.Load() != 0 || consent.calls.Load() != 0 {
		t.Fatalf("wrong owner reached selected-tab observation: observations=%d consent-checks=%d err=%v", tab.observations.Load(), consent.calls.Load(), err)
	}
	observation, err := b.Observe(ctx, e.sessionID, ObserveRequest{})
	if err == nil || observation.Text != "" {
		t.Fatalf("observation from an unconsented origin was returned: observation=%#v err=%v", observation, err)
	}
	if got := consent.calls.Load(); got != 2 {
		t.Fatalf("consent checks=%d, want pre-read and post-read checks", got)
	}

	// The pre-read admission is still one browser action: the result-bound
	// consent recheck does not consume a second MaxActions slot.
	if err := policy.Authorize(ctx, sessionpermission.ActionRequest{
		SessionID: e.sessionID, LeaseID: e.leaseID, OwnerID: e.ownerID, CallerID: e.callerID,
		Kind: "observe", CurrentURL: "https://example.test/public/start",
	}); err == nil {
		t.Fatal("the single admitted observation did not consume its action budget")
	}

	sameConsent := &exactOriginConsent{origin: "https://example.test"}
	samePolicy, err := sessionpermission.NewWithVerifiers(profile, nil, sameConsent)
	if err != nil {
		t.Fatal(err)
	}
	sameTab := &snapshotAttachedTab{observedURL: "https://example.test/public/result"}
	sameEntry := &entry{
		sessionID: "same-origin-session", permissionRef: profile.ID, profile: profile, policy: samePolicy,
		backend: string(sessionpermission.BackendUserAttached), leaseID: e.leaseID, ownerID: e.ownerID,
		callerID: e.callerID, tabID: e.tabID, tabURL: e.tabURL, consentReceipt: e.consentReceipt, attachedTab: sameTab,
	}
	sameBroker := &Broker{
		profiles: bridgeProfileResolver{profile: profile, authorizer: samePolicy}, caller: bridgeContextCaller{},
		entries: map[string]*entry{sameEntry.sessionID: sameEntry},
	}
	sameObservation, err := sameBroker.Observe(ctx, sameEntry.sessionID, ObserveRequest{})
	if err != nil || sameObservation.Text != "sensitive page text" || sameConsent.calls.Load() != 2 {
		t.Fatalf("same-origin authorized observation failed: observation=%#v consent-checks=%d err=%v", sameObservation, sameConsent.calls.Load(), err)
	}
	reported := false
	for _, capability := range capabilities(profile, string(sessionpermission.BackendUserAttached)) {
		if capability == "observe.selected-tab-consent-bound-result-v1" {
			reported = true
			break
		}
	}
	if !reported {
		t.Fatal("user-attached capability report omitted the result-bound consent guarantee")
	}

	outOfScopeConsent := &exactOriginConsent{origin: "https://example.test"}
	outOfScopePolicy, err := sessionpermission.NewWithVerifiers(profile, nil, outOfScopeConsent)
	if err != nil {
		t.Fatal(err)
	}
	outOfScopeTab := &snapshotAttachedTab{observedURL: "https://blocked.test/private"}
	outOfScopeEntry := &entry{
		sessionID: "out-of-scope-session", permissionRef: profile.ID, profile: profile, policy: outOfScopePolicy,
		backend: string(sessionpermission.BackendUserAttached), leaseID: e.leaseID, ownerID: e.ownerID,
		callerID: e.callerID, tabID: e.tabID, tabURL: e.tabURL, consentReceipt: e.consentReceipt, attachedTab: outOfScopeTab,
	}
	outOfScopeBroker := &Broker{
		profiles: bridgeProfileResolver{profile: profile, authorizer: outOfScopePolicy}, caller: bridgeContextCaller{},
		entries: map[string]*entry{outOfScopeEntry.sessionID: outOfScopeEntry},
	}
	outOfScopeObservation, err := outOfScopeBroker.Observe(ctx, outOfScopeEntry.sessionID, ObserveRequest{})
	if err == nil || outOfScopeObservation.Text != "" || outOfScopeConsent.calls.Load() != 1 {
		t.Fatalf("out-of-scope observed URL was returned or consented: observation=%#v consent-checks=%d err=%v", outOfScopeObservation, outOfScopeConsent.calls.Load(), err)
	}

	revokedConsent := &exactOriginConsent{origin: "https://example.test", revokeAfterFirst: true}
	revokedPolicy, err := sessionpermission.NewWithVerifiers(profile, nil, revokedConsent)
	if err != nil {
		t.Fatal(err)
	}
	revokedTab := &snapshotAttachedTab{observedURL: "https://example.test/public/result"}
	revokedEntry := &entry{
		sessionID: "revoked-consent-session", permissionRef: profile.ID, profile: profile, policy: revokedPolicy,
		backend: string(sessionpermission.BackendUserAttached), leaseID: e.leaseID, ownerID: e.ownerID,
		callerID: e.callerID, tabID: e.tabID, tabURL: e.tabURL, consentReceipt: e.consentReceipt, attachedTab: revokedTab,
	}
	revokedBroker := &Broker{
		profiles: bridgeProfileResolver{profile: profile, authorizer: revokedPolicy}, caller: bridgeContextCaller{},
		entries: map[string]*entry{revokedEntry.sessionID: revokedEntry},
	}
	revokedObservation, err := revokedBroker.Observe(ctx, revokedEntry.sessionID, ObserveRequest{})
	if err == nil || revokedObservation.Text != "" || revokedConsent.calls.Load() != 2 {
		t.Fatalf("mid-read consent revocation returned page data: observation=%#v consent-checks=%d err=%v", revokedObservation, revokedConsent.calls.Load(), err)
	}
}

func newSelectedTabInventoryBroker(t *testing.T, tab AttachedTab, tabURL string) (*Broker, *entry) {
	t.Helper()
	const extensionID = "abcdefghijklmnopabcdefghijklmnop"
	profile := sessionpermission.Profile{
		ID: "inventory-profile-ref", ProfileID: "inventory-profile", Effect: sessionpermission.EffectAct,
		Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached",
		URLAllow: []string{"https://example.test/public/*"}, View: true, Control: true,
		ExtensionAccess: true, ExtensionPermissions: map[string][]string{extensionID: {"storage", "https://example.test/*"}},
	}
	policy, err := sessionpermission.NewWithVerifiers(profile, nil, allowSelectedTabConsent{})
	if err != nil {
		t.Fatal(err)
	}
	callerID := bridgeCaller("run:inventory-owner")
	e := &entry{
		sessionID: "selected-tab-inventory", permissionRef: profile.ID, profile: profile, policy: policy,
		backend: string(sessionpermission.BackendUserAttached), leaseID: "inventory-lease",
		ownerID: "inventory-execution", callerID: string(callerID), tabID: "9", tabURL: tabURL, consentReceipt: "test-consent",
		socketTicketExpiry: time.Now().Add(time.Minute), attachedTab: tab,
	}
	b := &Broker{
		profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: bridgeContextCaller{},
		entries: map[string]*entry{e.sessionID: e},
	}
	return b, e
}

func TestSelectedTabPermissionSnapshotIsOwnerBoundAndRecheckedAfterRead(t *testing.T) {
	const extensionID = "abcdefghijklmnopabcdefghijklmnop"
	snapshot := AttachedExtensionInventory{
		ObservedAt: time.Now().UTC(),
		Extensions: []sessionpermission.ExtensionInventoryEntry{{
			ID: extensionID, Name: "Approved", Enabled: true, Permissions: []string{"storage"},
			PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{}),
		}},
	}
	tab := &snapshotAttachedTab{response: snapshot}
	b, e := newSelectedTabInventoryBroker(t, tab, "https://example.test/public/start")
	ctx := withBridgeContextCaller(context.Background(), e.callerID)
	result, err := b.SelectedTabExtensions(ctx, e.sessionID)
	if err != nil || result.ObservedAt != snapshot.ObservedAt || result.Binding.SessionID != e.sessionID || result.Binding.ProfileID != e.profile.ProfileID || result.Binding.PermissionRef != e.permissionRef || result.Binding.OwnerID != e.ownerID || result.Binding.LeaseID != e.leaseID || result.Binding.CallerID != e.callerID {
		t.Fatalf("permission snapshot lost fresh owner binding: result=%#v err=%v", result, err)
	}
	if len(result.Extensions) != 1 || result.Extensions[0].PermissionEvidence.Status != sessionpermission.ExtensionPermissionEvidenceEffective || result.Extensions[0].ArtifactVerified || result.Extensions[0].ArtifactSHA256 != "" {
		t.Fatalf("user-installed permissions were misrepresented as artifact-bound: %#v", result.Extensions)
	}
	if _, err := b.SelectedTabExtensions(withBridgeContextCaller(context.Background(), "run:wrong-owner"), e.sessionID); err == nil || tab.reads.Load() != 1 {
		t.Fatalf("wrong owner accessed permission snapshot: reads=%d err=%v", tab.reads.Load(), err)
	}

	outOfScope := &snapshotAttachedTab{response: snapshot}
	outOfScopeBroker, outOfScopeEntry := newSelectedTabInventoryBroker(t, outOfScope, "https://example.test/private/secret")
	if _, err := outOfScopeBroker.SelectedTabExtensions(withBridgeContextCaller(context.Background(), outOfScopeEntry.callerID), outOfScopeEntry.sessionID); err == nil || outOfScope.reads.Load() != 0 {
		t.Fatalf("out-of-scope inventory reached browser owner: reads=%d err=%v", outOfScope.reads.Load(), err)
	}

	staleSnapshot := snapshot
	staleSnapshot.ObservedAt = time.Now().Add(-time.Minute)
	stale := &snapshotAttachedTab{response: staleSnapshot}
	staleBroker, staleEntry := newSelectedTabInventoryBroker(t, stale, "https://example.test/public/start")
	if _, err := staleBroker.SelectedTabExtensions(withBridgeContextCaller(context.Background(), staleEntry.callerID), staleEntry.sessionID); err == nil || stale.reads.Load() != 1 {
		t.Fatalf("stale permission snapshot was returned: reads=%d err=%v", stale.reads.Load(), err)
	}

	revoked := &snapshotAttachedTab{response: snapshot, revokeOnRead: true}
	revokedBroker, revokedEntry := newSelectedTabInventoryBroker(t, revoked, "https://example.test/public/start")
	if _, err := revokedBroker.SelectedTabExtensions(withBridgeContextCaller(context.Background(), revokedEntry.callerID), revokedEntry.sessionID); err == nil || revoked.reads.Load() != 1 {
		t.Fatalf("permission inventory survived a mid-read revoke: reads=%d err=%v", revoked.reads.Load(), err)
	}
}

func TestCompetingRunCannotTakeOverSelectedTabPermissionRead(t *testing.T) {
	const extensionID = "abcdefghijklmnopabcdefghijklmnop"
	tab := &snapshotAttachedTab{
		response: AttachedExtensionInventory{ObservedAt: time.Now().UTC(), Extensions: []sessionpermission.ExtensionInventoryEntry{{ID: extensionID, Name: "Approved", Enabled: true, PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{})}}},
		started:  make(chan struct{}, 1), release: make(chan struct{}),
	}
	b, e := newSelectedTabInventoryBroker(t, tab, "https://example.test/public/start")
	ownerDone := make(chan error, 1)
	go func() {
		_, err := b.SelectedTabExtensions(withBridgeContextCaller(context.Background(), e.callerID), e.sessionID)
		ownerDone <- err
	}()
	select {
	case <-tab.started:
	case <-time.After(time.Second):
		t.Fatal("owner inventory request did not reach the session bridge")
	}
	competitorDone := make(chan error, 1)
	go func() {
		_, err := b.SelectedTabExtensions(withBridgeContextCaller(context.Background(), "run:competitor"), e.sessionID)
		competitorDone <- err
	}()
	close(tab.release)
	if err := <-ownerDone; err != nil {
		t.Fatalf("authorized owner inventory failed: %v", err)
	}
	if err := <-competitorDone; err == nil || tab.reads.Load() != 1 {
		t.Fatalf("competing run took over permission read: reads=%d err=%v", tab.reads.Load(), err)
	}
}

type bridgeProfileResolver struct {
	profile    sessionpermission.Profile
	authorizer sessionpermission.Authorizer
}

func (r bridgeProfileResolver) Resolve(context.Context, string) (sessionpermission.Profile, sessionpermission.Authorizer, error) {
	return r.profile, r.authorizer, nil
}

type allowSelectedTabConsent struct{}

func (allowSelectedTabConsent) VerifyBrowserTabConsent(context.Context, string, sessionpermission.ActionRequest) error {
	return nil
}

type bridgePolicyRecorder struct {
	profile  sessionpermission.Profile
	requests []sessionpermission.ActionRequest
}

func (p *bridgePolicyRecorder) ProfileSnapshot() sessionpermission.Profile { return p.profile }
func (p *bridgePolicyRecorder) Authorize(_ context.Context, req sessionpermission.ActionRequest) error {
	p.requests = append(p.requests, req)
	return nil
}

type bridgeCaller string

func (c bridgeCaller) BrowserCallerID(context.Context) (string, error) { return string(c), nil }
func (c bridgeCaller) BrowserCallerIDValue() string                    { return string(c) }

type bridgeOpenRecorder struct {
	requests []sessionpermission.OpenRequest
}

func (a *bridgeOpenRecorder) AuthorizeOpen(_ context.Context, req sessionpermission.OpenRequest) error {
	a.requests = append(a.requests, req)
	return nil
}

type bridgeConsentRecorder struct {
	request   TabConsentRequest
	revoked   bool
	revokeRun string
	revokeSes string
	revokeID  string
}

func (c *bridgeConsentRecorder) IssueBrowserTabConsent(_ context.Context, req TabConsentRequest) (string, error) {
	c.request = req
	return "owner-consent-receipt", nil
}

func (c *bridgeConsentRecorder) RevokeBrowserTabConsent(_ context.Context, receipt, runID, sessionID string) error {
	c.revoked, c.revokeID, c.revokeRun, c.revokeSes = true, receipt, runID, sessionID
	return nil
}

type bridgeConsentRevoker struct{ bridgeConsentRecorder }

func (c *bridgeConsentRevoker) IssueBrowserTabConsent(context.Context, TabConsentRequest) (string, error) {
	return "", errors.New("not used by this test")
}

type bridgeConsentIssuerFunc func(context.Context, TabConsentRequest) (string, error)

func (f bridgeConsentIssuerFunc) IssueBrowserTabConsent(ctx context.Context, req TabConsentRequest) (string, error) {
	return f(ctx, req)
}

func (bridgeConsentIssuerFunc) RevokeBrowserTabConsent(context.Context, string, string, string) error {
	return nil
}

type bridgeConsentIssueOnly struct{}

func (bridgeConsentIssueOnly) IssueBrowserTabConsent(context.Context, TabConsentRequest) (string, error) {
	return "receipt", nil
}

type bridgeNoopOwner struct{}

func (bridgeNoopOwner) Attach(context.Context, PairRequest, *websocket.Conn) (AttachedTab, error) {
	return nil, nil
}

func TestPairSelectedTabBindsTrustedRunAndOmitsSensitiveProfileFields(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile-internal", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", ProfileID: "user-state", URLAllow: []string{"https://example.test/*"}, CredentialIDs: []string{"vault:secret-id"}, MaxCredits: 12, View: true, Control: true, ExtensionAccess: true}
	policy := &bridgePolicyRecorder{profile: profile}
	caller := bridgeCaller("run:owner")
	consent := &bridgeConsentRecorder{}
	open := &bridgeOpenRecorder{}
	code := "one-time-pair-code"
	e := &entry{permissionRef: "profile-ref", profile: profile, backend: string(sessionpermission.BackendUserAttached), sessionID: "session-1", leaseID: "lease-internal", ownerID: "execution-1", callerID: string(caller), grantReceipt: "am-grant", runID: "run-1", pairingCodeHash: bridgeDigest(code), pairingNonce: "nonce-1", pairingExpires: time.Now().Add(time.Minute), policy: policy}
	b := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, openAuthorizer: open, caller: caller, tabConsentIssuer: consent, bridge: bridgeNoopOwner{}, entries: map[string]*entry{e.sessionID: e}}
	request := PairTabRequest{PairingCode: code}
	request.Tab.ID = 7
	request.Tab.URL = "https://example.test/account"
	request.Tab.Title = "Account"
	wrongCallerBroker := &Broker{profiles: b.profiles, openAuthorizer: open, caller: bridgeCaller("run:other"), tabConsentIssuer: consent, bridge: bridgeNoopOwner{}, entries: b.entries}
	if _, err := wrongCallerBroker.PairSelectedTab(context.Background(), request, "http://localhost:8080/api/v1"); err == nil {
		t.Fatal("pairing code was accepted from another run")
	}
	if e.pairingConsumed {
		t.Fatal("wrong-run attempt consumed the owner's pairing code")
	}
	grant, err := b.PairSelectedTab(context.Background(), request, "http://localhost:8080/api/v1")
	if err != nil {
		t.Fatal(err)
	}
	if grant.SessionID != e.sessionID || grant.LeaseReceipt == "" || grant.PermissionProfile.Control != true {
		t.Fatalf("incomplete pair grant: %#v", grant)
	}
	if grant.EvidenceWarning != bridgeEvidenceWarning {
		t.Fatalf("missing run-evidence warning when recorder is unconfigured: %#v", grant)
	}
	expiryInvalid := !consent.request.ExpiresAt.After(time.Now())
	if profile.MaxDuration > 0 && consent.request.ExpiresAt.After(e.openedAt.Add(profile.MaxDuration)) {
		expiryInvalid = true
	}
	if consent.request.CallerID != caller.BrowserCallerIDValue() || consent.request.GrantReceipt != e.grantReceipt || consent.request.SessionID != e.sessionID || consent.request.LeaseID != e.leaseID || consent.request.OwnerID != e.ownerID || consent.request.PermissionRef != e.permissionRef || consent.request.TabID != "7" || consent.request.PairingNonce != e.pairingNonce || consent.request.URL != request.Tab.URL || expiryInvalid {
		t.Fatalf("consent request not bound to exact caller/grant/session/lease/owner/profile/tab/URL/expiry: %#v", consent.request)
	}
	if len(open.requests) != 1 || open.requests[0].CallerID != e.callerID || open.requests[0].TabID != "7" || open.requests[0].ConsentReceipt != "owner-consent-receipt" {
		t.Fatalf("open admission not bound: %#v", open.requests)
	}
	if len(policy.requests) != 1 || policy.requests[0].Kind != "attach" || policy.requests[0].CallerID != e.callerID {
		t.Fatalf("attach authorization not bound: %#v", policy.requests)
	}
	encoded, _ := json.Marshal(grant)
	if strings.Contains(string(encoded), "vault:secret-id") || strings.Contains(string(encoded), "profile-internal") || strings.Contains(string(encoded), "MaxCredits") {
		t.Fatalf("pair grant exposed internal profile fields: %s", encoded)
	}
	if _, err := b.PairSelectedTab(context.Background(), request, "http://localhost:8080/api/v1"); err == nil {
		t.Fatal("one-time pairing code was reusable")
	}
	if err := b.revokeTabConsent(e); err != nil || !consent.revoked || consent.revokeID != "owner-consent-receipt" || consent.revokeRun != "run-1" || consent.revokeSes != "session-1" {
		t.Fatalf("revocation was not bound to the issued session: err=%v consent=%#v", err, consent)
	}
}

func TestPairSelectedTabRejectsConsentThatExpiresDuringIssuance(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", URLAllow: []string{"https://example.test/*"}, MaxDuration: 100 * time.Millisecond, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	open := &bridgeOpenRecorder{}
	caller := bridgeCaller("run:owner")
	code := "one-time-pair-code"
	openedAt := time.Now().Add(-50 * time.Millisecond)
	e := &entry{permissionRef: "profile-ref", profile: profile, backend: string(sessionpermission.BackendUserAttached), sessionID: "session-1", leaseID: "lease-1", ownerID: "execution-1", callerID: string(caller), grantReceipt: "am-grant", runID: "run-1", pairingCodeHash: bridgeDigest(code), pairingNonce: "nonce-1", pairingExpires: time.Now().Add(time.Minute), openedAt: openedAt, policy: policy}
	issuer := bridgeConsentIssuerFunc(func(_ context.Context, req TabConsentRequest) (string, error) {
		if req.ExpiresAt.After(openedAt.Add(profile.MaxDuration)) {
			t.Fatalf("consent outlived its session: %v", req.ExpiresAt)
		}
		time.Sleep(75 * time.Millisecond)
		return "late-owner-receipt", nil
	})
	b := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, openAuthorizer: open, caller: caller, tabConsentIssuer: issuer, bridge: bridgeNoopOwner{}, entries: map[string]*entry{e.sessionID: e}}
	request := PairTabRequest{PairingCode: code}
	request.Tab.ID = 7
	request.Tab.URL = "https://example.test/account"
	if _, err := b.PairSelectedTab(context.Background(), request, "http://localhost:8080/api/v1"); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("late consent result = %v, want expired denial", err)
	}
	if len(open.requests) != 0 || len(policy.requests) != 0 || e.consentReceipt != "" || e.leaseReceipt != "" || e.socketTicketHash != "" {
		t.Fatalf("late consent crossed into attachment admission: open=%#v actions=%#v entry=%#v", open.requests, policy.requests, e)
	}
}

func TestPairSelectedTabRejectsConsentRevokedDuringIssuance(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", URLAllow: []string{"https://example.test/*"}, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	open := &bridgeOpenRecorder{}
	caller := bridgeCaller("run:owner")
	code := "one-time-pair-code"
	e := &entry{permissionRef: "profile-ref", profile: profile, backend: string(sessionpermission.BackendUserAttached), sessionID: "session-1", leaseID: "lease-1", ownerID: "execution-1", callerID: string(caller), grantReceipt: "am-grant", runID: "run-1", pairingCodeHash: bridgeDigest(code), pairingNonce: "nonce-1", pairingExpires: time.Now().Add(time.Minute), openedAt: time.Now(), policy: policy}
	issuer := bridgeConsentIssuerFunc(func(context.Context, TabConsentRequest) (string, error) {
		e.revoking.Store(true)
		return "revoked-owner-receipt", nil
	})
	b := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, openAuthorizer: open, caller: caller, tabConsentIssuer: issuer, bridge: bridgeNoopOwner{}, entries: map[string]*entry{e.sessionID: e}}
	request := PairTabRequest{PairingCode: code}
	request.Tab.ID = 7
	request.Tab.URL = "https://example.test/account"
	if _, err := b.PairSelectedTab(context.Background(), request, "http://localhost:8080/api/v1"); err == nil || !strings.Contains(err.Error(), "revoked") {
		t.Fatalf("revoked consent result = %v, want revoked denial", err)
	}
	if len(open.requests) != 0 || len(policy.requests) != 0 || e.consentReceipt != "" || e.leaseReceipt != "" || e.socketTicketHash != "" {
		t.Fatalf("revoked consent crossed into attachment admission: open=%#v actions=%#v entry=%#v", open.requests, policy.requests, e)
	}
}

func TestPairSelectedTabFailsClosedWithoutExistingConsentOwner(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", URLAllow: []string{"https://example.test/public/*"}, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	open := &bridgeOpenRecorder{}
	caller := bridgeCaller("run:owner")
	code := "one-time-pair-code"
	e := &entry{permissionRef: "profile-ref", profile: profile, backend: string(sessionpermission.BackendUserAttached), sessionID: "session-1", leaseID: "lease-1", ownerID: "execution-1", callerID: string(caller), grantReceipt: "am-grant", pairingCodeHash: bridgeDigest(code), pairingNonce: "nonce-1", pairingExpires: time.Now().Add(time.Minute), policy: policy}
	b := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, openAuthorizer: open, caller: caller, bridge: bridgeNoopOwner{}, entries: map[string]*entry{e.sessionID: e}}
	request := PairTabRequest{PairingCode: code}
	request.Tab.ID = 7
	request.Tab.URL = "https://example.test/public/start"
	if _, err := b.PairSelectedTab(context.Background(), request, "http://localhost:8080/api/v1"); err == nil || !strings.Contains(err.Error(), "consent owner unavailable") {
		t.Fatalf("pair without existing consent owner = %v, want explicit fail-closed error", err)
	}
	if e.consentReceipt != "" || e.leaseReceipt != "" || e.socketTicketHash != "" || e.attachedTab != nil {
		t.Fatalf("missing consent authority produced attachment credentials: %#v", e)
	}
	if len(open.requests) != 0 || len(policy.requests) != 0 {
		t.Fatalf("authorization ran without an existing consent receipt: open=%#v action=%#v", open.requests, policy.requests)
	}
}

func TestPairSelectedTabFailsClosedWithoutOwnerRevocation(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, ProfileMode: "user_attached", URLAllow: []string{"https://example.test/public/*"}, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	open := &bridgeOpenRecorder{}
	caller := bridgeCaller("run:owner")
	code := "one-time-pair-code"
	e := &entry{permissionRef: "profile-ref", profile: profile, backend: string(sessionpermission.BackendUserAttached), sessionID: "session-1", leaseID: "lease-1", ownerID: "execution-1", callerID: string(caller), grantReceipt: "am-grant", runID: "run-1", pairingCodeHash: bridgeDigest(code), pairingNonce: "nonce-1", pairingExpires: time.Now().Add(time.Minute), policy: policy}
	b := &Broker{profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, openAuthorizer: open, caller: caller, tabConsentIssuer: bridgeConsentIssueOnly{}, bridge: bridgeNoopOwner{}, entries: map[string]*entry{e.sessionID: e}}
	request := PairTabRequest{PairingCode: code}
	request.Tab.ID = 7
	request.Tab.URL = "https://example.test/public/start"
	if _, err := b.PairSelectedTab(context.Background(), request, "http://localhost:8080/api/v1"); err == nil || !strings.Contains(err.Error(), "revoker unavailable") {
		t.Fatalf("pair without consent revoker = %v, want explicit fail-closed error", err)
	}
	if e.consentReceipt != "" || e.leaseReceipt != "" || e.socketTicketHash != "" || len(open.requests) != 0 || len(policy.requests) != 0 {
		t.Fatalf("consent without revocation authority crossed into attachment admission: entry=%#v open=%#v actions=%#v", e, open.requests, policy.requests)
	}
}

func TestSessionAuthorizerRejectsAnotherRunEvenWhenProfileMatches(t *testing.T) {
	profile := sessionpermission.Profile{ID: "profile", Effect: sessionpermission.EffectAct, Approval: sessionpermission.ApprovalNone, URLAllow: []string{"https://example.test/*"}, View: true, Control: true}
	policy := &bridgePolicyRecorder{profile: profile}
	resolver := bridgeProfileResolver{profile: profile, authorizer: policy}
	owner := &sessionAuthorizer{resolver: resolver, caller: bridgeCaller("run:attacker"), permissionRef: "profile-ref", profile: profile, policy: policy, expectedCallerID: "run:owner"}
	request := sessionpermission.ActionRequest{SessionID: "session-1", LeaseID: "lease-1", OwnerID: "owner-1", Kind: "observe", CurrentURL: "https://example.test/account"}
	if err := owner.Authorize(context.Background(), request); err == nil {
		t.Fatal("another active run with the same profile accessed the session")
	}
	if len(policy.requests) != 0 {
		t.Fatalf("permission policy saw unauthorized requests: %#v", policy.requests)
	}

	owner.caller = bridgeCaller("run:owner")
	if err := owner.Authorize(context.Background(), request); err != nil {
		t.Fatalf("owner run was denied: %v", err)
	}
	if len(policy.requests) != 1 || policy.requests[0].CallerID != "run:owner" {
		t.Fatalf("owner request was not bound to expected principal: %#v", policy.requests)
	}
}

func TestExpiredSelectedTabLeaseCannotEnterOperation(t *testing.T) {
	e := &entry{sessionID: "expired-session", backend: string(sessionpermission.BackendUserAttached), socketTicketExpiry: time.Now().Add(-time.Second)}
	b := &Broker{entries: map[string]*entry{e.sessionID: e}}
	if err := b.lockEntry(e); !errors.Is(err, ErrSessionNotOwned) {
		t.Fatalf("expired lease admission err = %v, want ErrSessionNotOwned", err)
	}
	if _, ok := b.entries[e.sessionID]; ok {
		t.Fatal("expired lease remained addressable after operation admission")
	}
}

func TestExtensionInventoryIsFilteredToExplicitIDs(t *testing.T) {
	approvedID := "abcdefghijklmnopabcdefghijklmnop"
	profile := sessionpermission.Profile{ExtensionAccess: true, ExtensionPermissions: map[string][]string{approvedID: {"storage", "https://example.test/*"}}}
	items := []sessionpermission.ExtensionInventoryEntry{{ID: approvedID, Name: "Approved", Enabled: true, PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{})}, {ID: "other-extension", Name: "Other"}}
	filtered, err := filterExtensionInventory(profile, items)
	if err != nil || len(filtered) != 1 || filtered[0].ID != approvedID {
		t.Fatalf("filtered inventory=%#v err=%v", filtered, err)
	}
	items[0].Permissions = []string{"tabs"}
	if _, err := filterExtensionInventory(profile, items); err == nil {
		t.Fatal("inventory disclosed an API permission outside the profile allowlist")
	}
	items[0].Permissions = nil
	items[0].HostPermissions = []string{"https://outside.example/*"}
	if _, err := filterExtensionInventory(profile, items); err == nil {
		t.Fatal("inventory disclosed a host permission outside the profile allowlist")
	}
	profile.ExtensionPermissions = nil
	if _, err := filterExtensionInventory(profile, items); err == nil {
		t.Fatal("inventory without an explicit ID allowlist was disclosed")
	}
}

func TestExtensionPermissionEvidenceIsExplicitAndConsistent(t *testing.T) {
	id := "abcdefghijklmnopabcdefghijklmnop"
	profile := sessionpermission.Profile{ExtensionAccess: true, ExtensionPermissions: map[string][]string{id: {"storage"}}}
	base := sessionpermission.ExtensionInventoryEntry{ID: id, Name: "Verified", Enabled: true, Permissions: []string{"storage"}, PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{}), DeclaredPermissions: []string{"storage"}, ArtifactVerified: true}
	encoded, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["permissions"] != nil || wire["effective_permissions"] == nil || wire["declared_permissions"] == nil || wire["permission_evidence"] == nil {
		t.Fatalf("inventory wire response does not separate effective, declared and provenance fields: %s", encoded)
	}
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{base}); err != nil {
		t.Fatalf("explicit Chrome management active-permission evidence was rejected: %v", err)
	}
	for _, tc := range []struct {
		name string
		edit func(*sessionpermission.ExtensionPermissionEvidence)
	}{
		{name: "missing scope", edit: func(e *sessionpermission.ExtensionPermissionEvidence) { e.Scope = "" }},
		{name: "forged per-tab access", edit: func(e *sessionpermission.ExtensionPermissionEvidence) { e.PerTabSiteAccess = "effective" }},
		{name: "unknown scope", edit: func(e *sessionpermission.ExtensionPermissionEvidence) { e.Scope = "site_access" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base
			candidate.PermissionEvidence = base.PermissionEvidence
			tc.edit(&candidate.PermissionEvidence)
			if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{candidate}); err == nil {
				t.Fatalf("invalid permission scope metadata was accepted: %#v", candidate.PermissionEvidence)
			}
		})
	}
	artifactBound := base
	artifactBound.ArtifactVerified = true
	artifactBound.ArtifactSHA256 = strings.Repeat("a", 64)
	artifactBound.ManifestSHA256 = strings.Repeat("b", 64)
	artifactBound.PermissionEvidence = permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeGetAll, time.Now().UTC())
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{artifactBound}); err != nil {
		t.Fatalf("digest-bound Chrome permission API evidence was rejected: %v", err)
	}
	staleEvidence := artifactBound
	staleEvidence.PermissionEvidence.ObservedAt = time.Now().UTC().Add(-2 * time.Minute)
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{staleEvidence}); err == nil {
		t.Fatal("stale managed runtime permission evidence was accepted")
	}
	artifactBound.ArtifactVerified = false
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{artifactBound}); err == nil {
		t.Fatal("runtime permission probe was accepted without an artifact binding")
	}
	disabled := sessionpermission.ExtensionInventoryEntry{ID: id, Name: "Disabled", Enabled: false, PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceUnavailable, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{})}
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{disabled}); err != nil {
		t.Fatalf("disabled extension's unavailable active state was rejected: %v", err)
	}
	base.Enabled = false
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{base}); err == nil {
		t.Fatal("disabled extension was represented as having effective permissions")
	}
	base.Enabled = true
	base.PermissionEvidence = permissionEvidence(sessionpermission.ExtensionPermissionEvidenceUnavailable, sessionpermission.ExtensionPermissionSourceChromiumCDP, time.Time{})
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{base}); err == nil {
		t.Fatal("unavailable permission evidence was accepted with nonempty permission values")
	}
	base.Permissions = nil
	base.PermissionEvidence.Source = "manifest"
	if _, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{base}); err == nil {
		t.Fatal("manifest metadata was accepted as effective permission evidence")
	}
}

func TestActiveTabGrantSetDoesNotProvePerTabAccessOrExpandURLScope(t *testing.T) {
	id := "abcdefghijklmnopabcdefghijklmnop"
	profile := sessionpermission.Profile{
		Effect: sessionpermission.EffectObserve, Approval: sessionpermission.ApprovalNone,
		View: true, ExtensionAccess: true, URLAllow: []string{"https://example.test/*"},
		ExtensionPermissions: map[string][]string{id: {"activeTab"}},
	}
	entry := sessionpermission.ExtensionInventoryEntry{
		ID: id, Name: "Temporary access extension", Enabled: true, Permissions: []string{"activeTab"},
		PermissionEvidence: permissionEvidence(sessionpermission.ExtensionPermissionEvidenceEffective, sessionpermission.ExtensionPermissionSourceChromeManagement, time.Time{}),
	}
	filtered, err := filterExtensionInventory(profile, []sessionpermission.ExtensionInventoryEntry{entry})
	if err != nil || len(filtered) != 1 || filtered[0].PermissionEvidence.Scope != sessionpermission.ExtensionPermissionScopeGrantSetOnly || filtered[0].PermissionEvidence.PerTabSiteAccess != sessionpermission.ExtensionPerTabSiteAccessNotMeasured || len(filtered[0].HostPermissions) != 0 {
		t.Fatalf("inventory overstated temporary page access: filtered=%#v err=%v", filtered, err)
	}
	policy, err := sessionpermission.New(profile, nil)
	if err != nil {
		t.Fatal(err)
	}
	request := sessionpermission.ActionRequest{Kind: "extension_inventory", ExtensionID: id, ExtensionPermission: "activeTab", CurrentURL: "https://blocked.test/private"}
	if err := policy.Authorize(context.Background(), request); !errors.Is(err, sessionpermission.ErrDenied) {
		t.Fatalf("activeTab grant widened the session URL profile: %v", err)
	}
}

func TestUserAttachedOpenRequiresLivePreservationAndReconnectCapabilities(t *testing.T) {
	required := requiredCapabilities(sessionpermission.Profile{Control: true, ExtensionAccess: true, Handoff: true}, string(sessionpermission.BackendUserAttached))
	set := map[sessionpermission.Capability]bool{}
	for _, item := range required {
		set[item] = true
	}
	for _, item := range []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl, sessionpermission.CapabilityPreserveUserState, sessionpermission.CapabilityReconnect, sessionpermission.CapabilityExtensionInventory, sessionpermission.CapabilityExtensionHostPermissions, sessionpermission.CapabilityHandoff} {
		if !set[item] {
			t.Errorf("user-attached admission did not require %q", item)
		}
	}
}

func TestIdentityProfileRequiresManagedCustomizationQualification(t *testing.T) {
	identity := &sessionpermission.BrowserIdentityProfile{Preset: "balanced", Locale: "en-GB"}
	managedRequired := requiredCapabilities(sessionpermission.Profile{IdentityProfile: identity}, string(sessionpermission.BackendManaged))
	if !containsCapability(managedRequired, sessionpermission.CapabilityIdentityCustomization) {
		t.Fatal("identity profile did not require identity_customization qualification")
	}
	attachedRequired := requiredCapabilities(sessionpermission.Profile{IdentityProfile: identity}, string(sessionpermission.BackendUserAttached))
	if !containsCapability(attachedRequired, sessionpermission.CapabilityIdentityCustomization) {
		t.Fatal("profile must require the capability that preflight will deny for attached mode")
	}
	reports := []sessionpermission.BackendReport{{Backend: sessionpermission.BackendUserAttached, Qualified: true, EvidenceID: "attached", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityIdentityCustomization}}}
	rows := (&Broker{}).filterCapabilities(reports)
	negotiated := sessionpermission.NegotiateCapabilities(rows, attachedRequired)
	if len(negotiated) != 1 || negotiated[0].Accepted || negotiated[0].Reasons[sessionpermission.CapabilityIdentityCustomization] != "fresh-session identity customization is unsupported for attached and remote modes" {
		t.Fatalf("selected-tab profile with fresh identity controls was not denied with an explicit reason: %#v", negotiated)
	}
}

func TestCredentialProfilesRequireQualifiedCredentialUseCapability(t *testing.T) {
	withoutCredential := requiredCapabilities(sessionpermission.Profile{}, string(sessionpermission.BackendManaged))
	if containsCapability(withoutCredential, sessionpermission.CapabilityCredentialUse) {
		t.Fatal("profile without credential IDs unexpectedly requires credential use")
	}
	withCredential := requiredCapabilities(sessionpermission.Profile{CredentialIDs: []string{"vault-ref-1"}}, string(sessionpermission.BackendManaged))
	if !containsCapability(withCredential, sessionpermission.CapabilityCredentialUse) {
		t.Fatal("profile with credential IDs must require a qualified credential-use capability")
	}
	rows := sessionpermission.NegotiateCapabilities([]sessionpermission.BackendReport{{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "observe-control-only", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl}}}, withCredential)
	if len(rows) != 1 || rows[0].Accepted {
		t.Fatalf("credential profile accepted without a qualified credential-use capability: %#v", rows)
	}
}

func containsCapability(values []sessionpermission.Capability, want sessionpermission.Capability) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestApprovalProfilesRequireQualifiedApprovalReceiptCapability(t *testing.T) {
	for _, approval := range []sessionpermission.Approval{sessionpermission.ApprovalSafetyPass, sessionpermission.ApprovalEveryAction} {
		required := requiredCapabilities(sessionpermission.Profile{Approval: approval, Control: true}, string(sessionpermission.BackendManaged))
		if !containsCapability(required, sessionpermission.CapabilityApprovalReceipt) {
			t.Errorf("approval profile %q must require a qualified receipt verifier", approval)
		}
		rows := sessionpermission.NegotiateCapabilities([]sessionpermission.BackendReport{{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "control-only", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl}}}, required)
		if len(rows) != 1 || rows[0].Accepted {
			t.Errorf("approval profile %q was admitted without the approval receipt capability: %#v", approval, rows)
		}
	}
	if containsCapability(requiredCapabilities(sessionpermission.Profile{Approval: sessionpermission.ApprovalNone}, string(sessionpermission.BackendManaged)), sessionpermission.CapabilityApprovalReceipt) {
		t.Fatal("no-approval profile should not require an approval receipt capability")
	}
}

func TestCreditBudgetProfileRequiresQualifiedCreditCapability(t *testing.T) {
	required := requiredCapabilities(sessionpermission.Profile{MaxCredits: 25}, string(sessionpermission.BackendManaged))
	if !containsCapability(required, sessionpermission.CapabilityCreditBudget) {
		t.Fatal("positive credit budget must require an owner-backed credit authority capability")
	}
	rows := sessionpermission.NegotiateCapabilities([]sessionpermission.BackendReport{{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "observe-control-only", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityControl}}}, required)
	if len(rows) != 1 || rows[0].Accepted {
		t.Fatalf("credit-limited profile accepted without qualified credit authority: %#v", rows)
	}
}

func TestHistoryRetentionProfileRequiresQualifiedRetentionCapability(t *testing.T) {
	required := requiredCapabilities(sessionpermission.Profile{HistoryRead: true, HistoryRetention: time.Hour}, string(sessionpermission.BackendManaged))
	if !containsCapability(required, sessionpermission.CapabilityHistoryRetention) {
		t.Fatal("history retention request must require an owner-qualified retention capability")
	}
	rows := sessionpermission.NegotiateCapabilities([]sessionpermission.BackendReport{{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "history-read-only", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve, sessionpermission.CapabilityHistoryRead}}}, required)
	if len(rows) != 1 || rows[0].Accepted {
		t.Fatalf("history-retention profile accepted without retention support: %#v", rows)
	}
}

func TestExtensionInstallDoesNotRequireCallerSuppliedArtifactID(t *testing.T) {
	var broker Broker
	if _, err := broker.InstallExtension(context.Background(), "session-1", "", "artifact-ref", "digest"); err == nil || !strings.Contains(err.Error(), "controller unavailable") {
		t.Fatalf("extension install without caller ID = %v, want the fail-closed missing controller result", err)
	}
}

const selectedTabOwner = "run:00000000-0000-0000-0000-000000000001"

type bridgeContextCallerKey struct{}

type bridgeContextCaller struct{}

func (bridgeContextCaller) BrowserCallerID(ctx context.Context) (string, error) {
	caller, _ := ctx.Value(bridgeContextCallerKey{}).(string)
	if caller == "" {
		return "", errors.New("caller missing")
	}
	return caller, nil
}

func withBridgeContextCaller(ctx context.Context, caller string) context.Context {
	return context.WithValue(ctx, bridgeContextCallerKey{}, caller)
}

type bridgeNoopEvidence struct{}

func (bridgeNoopEvidence) RecordBrowserSession(context.Context, string, sessionpermission.Backend, string, string) error {
	return nil
}
func (bridgeNoopEvidence) RecordBrowserAction(context.Context, string, sessionpermission.Backend, string, string, string) error {
	return nil
}

type selectedTabBrokerHarness struct {
	broker    *Broker
	entry     *entry
	consent   *bridgeConsentRevoker
	server    *httptest.Server
	peer      *selectedTabTestPeer
	websocket string
	connected <-chan struct{}
}

type blockedSelectedTabBridge struct {
	started chan struct{}
	release chan struct{}
	tab     *snapshotAttachedTab
}

func (b *blockedSelectedTabBridge) Attach(context.Context, PairRequest, *websocket.Conn) (AttachedTab, error) {
	close(b.started)
	<-b.release
	return b.tab, nil
}

func TestConnectSelectedTabRejectsRevocationDuringOwnerAttach(t *testing.T) {
	e := &entry{sessionID: "session", backend: string(sessionpermission.BackendUserAttached), socketTicketHash: bridgeDigest("ticket"), socketTicketExpiry: time.Now().Add(time.Minute)}
	bridge := &blockedSelectedTabBridge{started: make(chan struct{}), release: make(chan struct{}), tab: &snapshotAttachedTab{}}
	broker := &Broker{bridge: bridge, entries: map[string]*entry{e.sessionID: e}}
	handlerDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		broker.ConnectSelectedTab(w, r, "")
		close(handlerDone)
	}))
	defer server.Close()
	dialer := websocket.Dialer{Subprotocols: []string{bridgeSubprotocol}}
	wsURL := "ws" + server.URL[len("http"):] + "/browser/bridge/socket?ticket=ticket"
	client, _, err := dialer.Dial(wsURL, http.Header{"Origin": []string{server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	select {
	case <-bridge.started:
	case <-time.After(time.Second):
		t.Fatal("bridge owner attach did not start")
	}
	e.revoking.Store(true)
	close(bridge.release)
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("connect handler did not finish")
	}
	if e.attachedTab != nil || e.attachedSocket != nil || e.socketConnected || e.socketTicketHash != "" || !bridge.tab.revoked.Load() {
		t.Fatalf("revoked owner result was attached or retained: entry=%#v tab=%#v", e, bridge.tab)
	}
}

type selectedTabTestPeer struct {
	conn           *websocket.Conn
	writeMu        sync.Mutex
	actionSeen     chan struct{}
	revoked        chan struct{}
	commit         chan struct{}
	actionOnce     sync.Once
	revokedOnce    sync.Once
	actionCount    atomic.Int64
	sideEffectDone atomic.Bool
}

func newSelectedTabBrokerHarness(t *testing.T) *selectedTabBrokerHarness {
	t.Helper()
	profile := sessionpermission.Profile{
		ID: "selected-tab-profile", ProfileMode: "user_attached", Effect: sessionpermission.EffectAct,
		Approval: sessionpermission.ApprovalNone, View: true, Control: true, Handoff: true,
		URLAllow: []string{"https://example.test/public/*"},
	}
	policy := &bridgePolicyRecorder{profile: profile}
	e := &entry{
		permissionRef: profile.ID, profile: profile, policy: policy, backend: string(sessionpermission.BackendUserAttached),
		sessionID: "selected-tab-session", leaseID: "lease-1", ownerID: "execution-1", callerID: selectedTabOwner, runID: "run-1",
		grantReceipt: "browser:run-1", pairingNonce: "nonce-1", pairingConsumed: true,
		tabID: "7", tabURL: "https://example.test/public/start", tabTitle: "Start", consentReceipt: "consent-1",
		leaseReceipt: "opaque-lease-receipt", leaseCaller: selectedTabOwner, openedAt: time.Now(),
		socketTicketHash: bridgeDigest("one-use-ticket"), socketTicketExpiry: time.Now().Add(time.Minute),
	}
	consent := &bridgeConsentRevoker{}
	broker := &Broker{
		profiles: bridgeProfileResolver{profile: profile, authorizer: policy}, caller: bridgeContextCaller{},
		bridge: WebSocketSelectedTabBridge{}, evidence: bridgeNoopEvidence{}, tabConsentIssuer: consent, entries: map[string]*entry{e.sessionID: e},
	}
	connected := make(chan struct{})
	var connectedOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		broker.ConnectSelectedTab(w, r, "")
		connectedOnce.Do(func() { close(connected) })
	}))
	wsURL := "ws" + server.URL[len("http"):] + "/browser/bridge/socket?ticket=one-use-ticket"
	dialer := websocket.Dialer{Subprotocols: []string{bridgeSubprotocol}}
	dialer.HandshakeTimeout = 2 * time.Second
	dialer.EnableCompression = false
	client, response, err := dialer.Dial(wsURL, http.Header{"Origin": []string{server.URL}})
	if err != nil {
		server.Close()
		t.Fatalf("connect actual Broker selected-tab socket: status=%v err=%v", response, err)
	}
	if err := client.WriteJSON(map[string]string{"type": "lease", "lease_receipt": e.leaseReceipt}); err != nil {
		client.Close()
		server.Close()
		t.Fatal(err)
	}
	var leaseResponse map[string]string
	if err := client.ReadJSON(&leaseResponse); err != nil || leaseResponse["type"] != "lease_accepted" {
		client.Close()
		server.Close()
		t.Fatalf("actual bridge lease response=%#v err=%v", leaseResponse, err)
	}
	peer := &selectedTabTestPeer{conn: client, actionSeen: make(chan struct{}), revoked: make(chan struct{}), commit: make(chan struct{})}
	go peer.run()
	t.Cleanup(func() { _ = client.Close(); server.Close() })
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("Broker did not attach the actual WebSocketSelectedTabBridge")
	}
	return &selectedTabBrokerHarness{broker: broker, entry: e, consent: consent, server: server, peer: peer, websocket: wsURL, connected: connected}
}

func (p *selectedTabTestPeer) run() {
	for {
		var message struct {
			Type      string `json:"type"`
			RequestID string `json:"request_id"`
			URL       string `json:"url"`
		}
		if err := p.conn.ReadJSON(&message); err != nil {
			return
		}
		switch message.Type {
		case "scope_check":
			p.write(map[string]string{"type": "scope_checked", "request_id": message.RequestID, "url": "https://example.test/public/start"})
		case "act":
			p.actionCount.Add(1)
			p.actionOnce.Do(func() { close(p.actionSeen) })
			go func(requestID string) {
				select {
				case <-p.commit:
					p.sideEffectDone.Store(true)
					p.write(map[string]string{"type": "action_complete", "request_id": requestID})
				case <-p.revoked:
					p.write(map[string]string{"type": "denied", "request_id": requestID, "reason": "lease_revoked_before_action_commit"})
				}
			}(message.RequestID)
		case "revoke":
			p.revokedOnce.Do(func() { close(p.revoked) })
			return
		}
	}
}

func (p *selectedTabTestPeer) write(message any) {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	_ = p.conn.WriteJSON(message)
}

func (p *selectedTabTestPeer) scopeEscape() {
	p.revokedOnce.Do(func() { close(p.revoked) })
	p.write(map[string]string{"type": "scope_violation", "observed_url": "https://example.test/private/secret"})
}

func selectedTabClickRequest() ActRequest {
	return ActRequest{Instruction: contracts.CompiledInstruction{Action: &actionsv1.ActionDefinition{
		Type:   actionsv1.ActionType_ACTION_TYPE_CLICK,
		Params: &actionsv1.ActionDefinition_Click{Click: &actionsv1.ClickParams{Selector: "#continue"}},
	}}}
}

func waitForSignal(t *testing.T, signal <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}

func TestBrokerSelectedTabCloseRevokesInFlightActionAndBlocksSecondRun(t *testing.T) {
	h := newSelectedTabBrokerHarness(t)
	attackerCtx := withBridgeContextCaller(context.Background(), "run:attacker")
	if _, err := h.broker.Act(attackerCtx, h.entry.sessionID, selectedTabClickRequest()); err == nil {
		t.Fatal("second run acted on the selected-tab lease")
	}
	if got := h.peer.actionCount.Load(); got != 0 {
		t.Fatalf("wrong-owner action reached browser socket %d times", got)
	}
	// The ticket is one-use while the legitimate run owns the socket.
	dialer := websocket.Dialer{Subprotocols: []string{bridgeSubprotocol}}
	_, response, err := dialer.Dial(h.websocket, http.Header{"Origin": []string{h.server.URL}})
	if err == nil || response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("second socket takeover response=%v err=%v", response, err)
	}
	ownerCtx := withBridgeContextCaller(context.Background(), selectedTabOwner)
	actionDone := make(chan error, 1)
	go func() {
		_, err := h.broker.Act(ownerCtx, h.entry.sessionID, selectedTabClickRequest())
		actionDone <- err
	}()
	waitForSignal(t, h.peer.actionSeen, "in-flight action")
	closeDone := make(chan error, 1)
	go func() { closeDone <- h.broker.Close(ownerCtx, h.entry.sessionID) }()
	waitForSignal(t, h.peer.revoked, "broker revoke frame")
	select {
	case err := <-actionDone:
		if err == nil {
			t.Fatal("in-flight action reported success after revoke")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("broker close did not interrupt the pending socket action")
	}
	select {
	case err := <-closeDone:
		if err != nil {
			t.Fatalf("owner close: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("broker close did not complete after revoking the socket")
	}
	if h.peer.sideEffectDone.Load() {
		t.Fatal("browser-side action committed after the broker revoke frame")
	}
	if !h.consent.revoked || h.consent.revokeID != "consent-1" || h.consent.revokeRun != "run-1" || h.consent.revokeSes != h.entry.sessionID {
		t.Fatalf("owner consent was not revoked with exact session binding: %#v", h.consent)
	}
	if _, err := h.broker.get(h.entry.sessionID); !errors.Is(err, ErrSessionNotOwned) {
		t.Fatalf("closed session remained available to another run: %v", err)
	}
}

func TestBrokerSelectedTabReconnectUsesSameLeaseAndDoesNotCreateAnotherSession(t *testing.T) {
	h := newSelectedTabBrokerHarness(t)
	ownerContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx := withBridgeContextCaller(ownerContext, selectedTabOwner)
	if err := h.peer.conn.Close(); err != nil {
		t.Fatalf("disconnect original bridge socket: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		h.broker.mu.RLock()
		connected := h.entry.socketConnected
		h.broker.mu.RUnlock()
		if !connected {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("broker did not release disconnected socket for reconnect")
		}
		time.Sleep(time.Millisecond)
	}

	dialer := websocket.Dialer{Subprotocols: []string{bridgeSubprotocol}}
	client, response, err := dialer.Dial(h.websocket, http.Header{"Origin": []string{h.server.URL}})
	if err != nil {
		t.Fatalf("same-lease reconnect: status=%v err=%v", response, err)
	}
	defer client.Close()
	if err := client.WriteJSON(map[string]string{"type": "lease", "lease_receipt": h.entry.leaseReceipt}); err != nil {
		t.Fatal(err)
	}
	var leaseResponse map[string]string
	if err := client.ReadJSON(&leaseResponse); err != nil || leaseResponse["type"] != "lease_accepted" {
		t.Fatalf("reconnected lease response=%#v err=%v", leaseResponse, err)
	}
	h.broker.mu.RLock()
	gotSessions := len(h.broker.entries)
	h.broker.mu.RUnlock()
	if gotSessions != 1 {
		t.Fatalf("reconnect created %d broker sessions, want original single session", gotSessions)
	}
	if err := client.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	actDone := make(chan error, 1)
	go func() { _, err := h.broker.Act(ctx, h.entry.sessionID, selectedTabClickRequest()); actDone <- err }()
	type readResult struct {
		message map[string]any
		err     error
	}
	readDriverMessage := func(stage string) map[string]any {
		readDone := make(chan readResult, 1)
		go func() {
			var message map[string]any
			err := client.ReadJSON(&message)
			readDone <- readResult{message: message, err: err}
		}()
		select {
		case result := <-readDone:
			if result.err != nil {
				t.Fatalf("read reconnected %s: %v", stage, result.err)
			}
			return result.message
		case err := <-actDone:
			t.Fatalf("reconnected action ended before %s (err=%v)", stage, err)
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for reconnected %s", stage)
		}
		return nil
	}
	action := readDriverMessage("scope check or action")
	if action["type"] == "scope_check" {
		if err := client.WriteJSON(map[string]string{"type": "scope_checked", "request_id": action["request_id"].(string), "url": "https://example.test/public/start"}); err != nil {
			t.Fatal(err)
		}
		action = readDriverMessage("action")
	}
	if action["type"] != "act" {
		t.Fatalf("reconnected action=%#v err=%v", action, err)
	}
	if err := client.WriteJSON(map[string]string{"type": "action_complete", "request_id": action["request_id"].(string)}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-actDone:
		if err != nil {
			t.Fatalf("owner action after same-lease reconnect: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("owner action did not complete after same-lease reconnect")
	}
}

func TestBrokerSelectedTabExpiryAndScopeLossCancelInFlightAction(t *testing.T) {
	t.Run("expiry", func(t *testing.T) {
		h := newSelectedTabBrokerHarness(t)
		ctx := withBridgeContextCaller(context.Background(), selectedTabOwner)
		actionDone := make(chan error, 1)
		go func() { _, err := h.broker.Act(ctx, h.entry.sessionID, selectedTabClickRequest()); actionDone <- err }()
		waitForSignal(t, h.peer.actionSeen, "expiry test action")
		h.broker.scheduleSocketExpiry(h.entry, time.Now().Add(30*time.Millisecond))
		waitForSignal(t, h.peer.revoked, "expiry revoke")
		select {
		case err := <-actionDone:
			if err == nil {
				t.Fatal("expired action reported success")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("expiry did not interrupt the pending action")
		}
		if h.peer.sideEffectDone.Load() {
			t.Fatal("expired browser action committed")
		}
	})
	t.Run("scope escape", func(t *testing.T) {
		h := newSelectedTabBrokerHarness(t)
		ctx := withBridgeContextCaller(context.Background(), selectedTabOwner)
		actionDone := make(chan error, 1)
		go func() { _, err := h.broker.Act(ctx, h.entry.sessionID, selectedTabClickRequest()); actionDone <- err }()
		waitForSignal(t, h.peer.actionSeen, "scope test action")
		h.peer.scopeEscape()
		select {
		case err := <-actionDone:
			if err == nil {
				t.Fatal("out-of-scope action reported success")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("scope loss did not interrupt the pending action")
		}
		if h.peer.sideEffectDone.Load() {
			t.Fatal("out-of-scope browser action committed")
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			if _, err := h.broker.get(h.entry.sessionID); errors.Is(err, ErrSessionNotOwned) {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("scope-revoked session remained available")
			}
			time.Sleep(time.Millisecond)
		}
	})
}

type extensionControllerFixture struct{}

func (extensionControllerFixture) Install(context.Context, ExtensionInstallRequest) (ExtensionReceipt, error) {
	return ExtensionReceipt{}, nil
}
func (extensionControllerFixture) Load(context.Context, ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	return ExtensionReceipt{}, nil
}
func (extensionControllerFixture) Reload(context.Context, ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	return ExtensionReceipt{}, nil
}
func (extensionControllerFixture) Unload(context.Context, ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	return ExtensionReceipt{}, nil
}
func (extensionControllerFixture) Validate(context.Context, ExtensionLifecycleRequest) (ExtensionValidation, error) {
	return ExtensionValidation{}, nil
}
func (extensionControllerFixture) Inventory(_ context.Context, binding ExtensionSessionBinding) (ExtensionInventoryReceipt, error) {
	return ExtensionInventoryReceipt{Binding: binding, ObservedAt: time.Now().UTC()}, nil
}

func TestExtensionOperationsFailBeforeSessionLookupWithoutAuthority(t *testing.T) {
	var broker Broker
	if _, err := broker.InstallExtension(context.Background(), "session-1", "ext-1", "artifact-ref", "digest"); err == nil || !strings.Contains(err.Error(), "extension controller unavailable") {
		t.Fatalf("install without controller = %v, want explicit preflight failure", err)
	}
	if _, err := broker.ReloadExtension(context.Background(), "session-1", "ext-1"); err == nil || !strings.Contains(err.Error(), "extension controller unavailable") {
		t.Fatalf("reload without controller = %v, want explicit preflight failure", err)
	}
	if _, err := broker.ValidateExtension(context.Background(), "session-1", "ext-1"); err == nil || !strings.Contains(err.Error(), "extension controller unavailable") {
		t.Fatalf("validation without controller = %v, want explicit preflight failure", err)
	}

	broker.extensions = extensionControllerFixture{}
	if _, err := broker.InstallExtension(context.Background(), "session-1", "ext-1", "artifact-ref", "digest"); err == nil || !strings.Contains(err.Error(), "artifact authority unavailable") {
		t.Fatalf("install without artifact verifier = %v, want explicit preflight failure", err)
	}
}

func TestVerifiedExtensionArtifactMustFitProfilePermissionAllowlist(t *testing.T) {
	profile := sessionpermission.Profile{ExtensionAccess: true, ExtensionPermissions: map[string][]string{"ext-1": {"storage", "https://shop.example/*"}}}
	valid := ExtensionArtifact{ArtifactRef: "build-1", ExtensionID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Digest: strings.Repeat("a", 64), ManifestSHA256: strings.Repeat("b", 64), Version: "1.0.0", Permissions: []string{"storage"}, HostPermissions: []string{"https://shop.example/*"}}
	profile.ExtensionPermissions = map[string][]string{valid.ExtensionID: {"storage", "https://shop.example/*"}}
	if err := verifyExtensionArtifactProfile(profile, valid, valid.ExtensionID, "build-1", strings.Repeat("a", 64)); err != nil {
		t.Fatalf("permitted verified artifact rejected: %v", err)
	}
	for name, artifact := range map[string]ExtensionArtifact{
		"wrong artifact ref": {ArtifactRef: "another-build", ExtensionID: valid.ExtensionID, Digest: valid.Digest, ManifestSHA256: valid.ManifestSHA256, Version: valid.Version},
		"wrong id":           {ArtifactRef: "build-1", ExtensionID: strings.Repeat("b", 32), Digest: valid.Digest, ManifestSHA256: valid.ManifestSHA256, Version: valid.Version},
		"wrong digest":       {ArtifactRef: "build-1", ExtensionID: valid.ExtensionID, Digest: strings.Repeat("c", 64), ManifestSHA256: valid.ManifestSHA256, Version: valid.Version},
		"extra host grant":   {ArtifactRef: "build-1", ExtensionID: valid.ExtensionID, Digest: valid.Digest, ManifestSHA256: valid.ManifestSHA256, Version: valid.Version, HostPermissions: []string{"https://outside.example/*"}},
		"extra permission":   {ArtifactRef: "build-1", ExtensionID: valid.ExtensionID, Digest: valid.Digest, ManifestSHA256: valid.ManifestSHA256, Version: valid.Version, Permissions: []string{"tabs"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := verifyExtensionArtifactProfile(profile, artifact, valid.ExtensionID, "build-1", valid.Digest); err == nil {
				t.Fatalf("unapproved artifact accepted: %#v", artifact)
			}
		})
	}
}

func TestByteBudgetProfileRequiresQualifiedByteBudgetCapability(t *testing.T) {
	required := requiredCapabilities(sessionpermission.Profile{MaxBytes: 1024}, string(sessionpermission.BackendManaged))
	if !containsCapability(required, sessionpermission.CapabilityByteBudget) {
		t.Fatal("positive byte budget must require an owner-qualified byte budget capability")
	}
	rows := sessionpermission.NegotiateCapabilities([]sessionpermission.BackendReport{{Backend: sessionpermission.BackendManaged, Qualified: true, EvidenceID: "observe-only", Capabilities: []sessionpermission.Capability{sessionpermission.CapabilityObserve}}}, required)
	if len(rows) != 1 || rows[0].Accepted {
		t.Fatalf("byte-limited profile accepted without qualified byte accounting: %#v", rows)
	}
}

func TestNavigationHistoryFiltersDisallowedURLs(t *testing.T) {
	profile := sessionpermission.Profile{HistoryRead: true, URLAllow: []string{"https://example.test/public/*"}, URLDeny: []string{"https://example.test/public/private/*"}}
	history := driver.NavigationStackResponse{
		SessionID:    "session-1",
		BackStack:    []driver.NavigationStackEntry{{URL: "https://outside.test/path", Title: "Outside"}, {URL: "https://example.test/public/old", Title: "Allowed"}, {URL: "https://example.test/public/private/old", Title: "Denied"}},
		Current:      &driver.NavigationStackEntry{URL: "https://example.test/public/current", Title: "Current"},
		ForwardStack: []driver.NavigationStackEntry{{URL: "https://outside.test/future", Title: "Outside"}, {URL: "https://example.test/public/future", Title: "Allowed"}},
	}
	filtered, err := filterNavigationHistory(profile, history)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.BackStack) != 1 || filtered.BackStack[0].Title != "Allowed" || len(filtered.ForwardStack) != 1 || filtered.ForwardStack[0].Title != "Allowed" {
		t.Fatalf("history filtering leaked or removed the wrong entries: %#v", filtered)
	}
	history.Current.URL = "https://outside.test/current"
	if _, err := filterNavigationHistory(profile, history); !errors.Is(err, sessionpermission.ErrDenied) {
		t.Fatalf("out-of-scope current page history error = %v, want denial", err)
	}
}

func TestNavigationHistoryFailsClosedForSelectedTab(t *testing.T) {
	e := &entry{sessionID: "selected-tab", backend: string(sessionpermission.BackendUserAttached), profile: sessionpermission.Profile{HistoryRead: true}}
	broker := &Broker{entries: map[string]*entry{e.sessionID: e}}
	if _, err := broker.NavigationHistory(context.Background(), e.sessionID); err == nil || !strings.Contains(err.Error(), "unsupported for user-attached") {
		t.Fatalf("selected-tab navigation history error = %v, want explicit unsupported denial", err)
	}
}
