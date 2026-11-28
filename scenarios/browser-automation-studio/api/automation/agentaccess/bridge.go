package agentaccess

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

const bridgeSubprotocol = "vrooli-browser-bridge-v1"
const bridgeEvidenceWarning = "Agent Manager run evidence could not be recorded"

// SelectedTabBridge is the owner seam for the user's explicitly selected tab.
// The returned session wraps only the bridge socket; it must not expose CDP.
type SelectedTabBridge interface {
	Attach(context.Context, PairRequest, *websocket.Conn) (AttachedTab, error)
}

// TabConsentIssuer adapts the existing human-consent owner. Implementations
// must bind the receipt to every field below and must not persist a second
// broker-local consent record.
type TabConsentIssuer interface {
	IssueBrowserTabConsent(context.Context, TabConsentRequest) (string, error)
}

// TabConsentRevoker removes an issued receipt from its existing consent owner
// when the selected-tab lease ends or the user revokes the extension share.
type TabConsentRevoker interface {
	RevokeBrowserTabConsent(context.Context, string, string, string) error
}

type TabConsentRequest struct {
	CallerID, GrantReceipt, SessionID, LeaseID, OwnerID, PermissionRef string
	TabID, URL, PairingNonce                                           string
	ExpiresAt                                                          time.Time
}

// AttachedTab is a narrow typed adapter over the selected-tab socket. It has
// no API for arbitrary script evaluation, cookies, history, or CDP.
type AttachedTab interface {
	CurrentURL(context.Context) (string, error)
	Observe(context.Context) (TabObservation, error)
	Act(context.Context, TabAction) error
	ExtensionInventory(context.Context) (AttachedExtensionInventory, error)
	Handoff(context.Context) error
	Close() error
	Done() <-chan struct{}
	Revoked() bool
}

// AttachedExtensionInventory is one fresh response from the selected-tab
// browser owner, bound by the enclosing broker receipt to the tab session,
// permission profile, lease and caller.
type AttachedExtensionInventory struct {
	ObservedAt time.Time
	Extensions []sessionpermission.ExtensionInventoryEntry
}

// WebSocketSelectedTabBridge implements the versioned typed protocol used by
// the consent-first extension. It never exposes arbitrary script or CDP calls.
type WebSocketSelectedTabBridge struct{}

func (WebSocketSelectedTabBridge) Attach(_ context.Context, req PairRequest, conn *websocket.Conn) (AttachedTab, error) {
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, payload, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	var lease struct {
		Type    string `json:"type"`
		Receipt string `json:"lease_receipt"`
	}
	if err := json.Unmarshal(payload, &lease); err != nil || lease.Type != "lease" || lease.Receipt == "" || subtleToken(lease.Receipt, req.LeaseReceipt) == false {
		return nil, errors.New("selected-tab lease handshake did not match the issued receipt")
	}
	_ = conn.SetReadDeadline(time.Time{})
	if err := conn.WriteJSON(map[string]string{"type": "lease_accepted"}); err != nil {
		return nil, err
	}
	tab := &webSocketTab{conn: conn, pending: make(map[string]chan bridgeMessage), done: make(chan struct{})}
	go tab.readLoop()
	return tab, nil
}

func subtleToken(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

type bridgeMessage struct {
	Type        string         `json:"type"`
	RequestID   string         `json:"request_id"`
	Error       string         `json:"error,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	SelectedTab TabObservation `json:"selected_tab,omitempty"`
	URL         string         `json:"url,omitempty"`
	ObservedAt  time.Time      `json:"observed_at,omitempty"`
	Extensions  []struct {
		ID                 string                                        `json:"id"`
		Name               string                                        `json:"name"`
		Version            string                                        `json:"version"`
		Enabled            bool                                          `json:"enabled"`
		Permissions        []string                                      `json:"permissions"`
		HostPermissions    []string                                      `json:"hostPermissions"`
		PermissionEvidence sessionpermission.ExtensionPermissionEvidence `json:"permission_evidence"`
	} `json:"extensions,omitempty"`
}

type webSocketTab struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
	mu      sync.Mutex
	pending map[string]chan bridgeMessage
	done    chan struct{}
	once    sync.Once
	revoked atomic.Bool
}

func (t *webSocketTab) readLoop() {
	defer t.once.Do(func() { close(t.done); _ = t.conn.Close() })
	for {
		_, raw, err := t.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg bridgeMessage
		if json.Unmarshal(raw, &msg) != nil {
			continue
		}
		if msg.Type == "revoke" || msg.Type == "scope_violation" {
			t.revoked.Store(true)
			return
		}
		if msg.RequestID != "" {
			t.mu.Lock()
			ch := t.pending[msg.RequestID]
			t.mu.Unlock()
			if ch != nil {
				select {
				case ch <- msg:
				default:
				}
			}
		}
	}
}

func (t *webSocketTab) request(ctx context.Context, command map[string]any, expected string) (bridgeMessage, error) {
	if t.revoked.Load() {
		return bridgeMessage{}, errors.New("selected-tab lease is revoked")
	}
	id := uuid.NewString()
	command["request_id"] = id
	ch := make(chan bridgeMessage, 1)
	t.mu.Lock()
	t.pending[id] = ch
	t.mu.Unlock()
	defer func() { t.mu.Lock(); delete(t.pending, id); t.mu.Unlock() }()
	t.writeMu.Lock()
	if t.revoked.Load() || ctx.Err() != nil {
		t.writeMu.Unlock()
		if err := ctx.Err(); err != nil {
			return bridgeMessage{}, err
		}
		return bridgeMessage{}, errors.New("selected-tab lease is revoked")
	}
	err := t.conn.WriteJSON(command)
	t.writeMu.Unlock()
	if err != nil {
		return bridgeMessage{}, err
	}
	select {
	case msg := <-ch:
		if msg.Type != expected {
			if msg.Reason != "" {
				return msg, errors.New("selected-tab operation denied: " + msg.Reason)
			}
			return msg, errors.New("selected-tab operation failed: " + msg.Error)
		}
		return msg, nil
	case <-t.done:
		return bridgeMessage{}, errors.New("selected-tab bridge disconnected")
	case <-ctx.Done():
		return bridgeMessage{}, ctx.Err()
	}
}

func (t *webSocketTab) Observe(ctx context.Context) (TabObservation, error) {
	msg, err := t.request(ctx, map[string]any{"type": "observe"}, "observed")
	return msg.SelectedTab, err
}
func (t *webSocketTab) CurrentURL(ctx context.Context) (string, error) {
	msg, err := t.request(ctx, map[string]any{"type": "scope_check"}, "scope_checked")
	return msg.URL, err
}
func (t *webSocketTab) Act(ctx context.Context, action TabAction) error {
	_, err := t.request(ctx, map[string]any{"type": "act", "action": action}, "action_complete")
	return err
}
func (t *webSocketTab) Handoff(ctx context.Context) error {
	_, err := t.request(ctx, map[string]any{"type": "handoff"}, "handoff_complete")
	return err
}
func (t *webSocketTab) ExtensionInventory(ctx context.Context) (AttachedExtensionInventory, error) {
	msg, err := t.request(ctx, map[string]any{"type": "extension_inventory"}, "extension_inventory")
	if err != nil {
		return AttachedExtensionInventory{}, err
	}
	if msg.ObservedAt.IsZero() {
		return AttachedExtensionInventory{}, errors.New("selected-tab extension inventory omitted its observation time")
	}
	entries := make([]sessionpermission.ExtensionInventoryEntry, 0, len(msg.Extensions))
	for _, item := range msg.Extensions {
		entries = append(entries, sessionpermission.ExtensionInventoryEntry{ID: item.ID, Name: item.Name, Version: item.Version, Enabled: item.Enabled, Permissions: item.Permissions, HostPermissions: item.HostPermissions, PermissionEvidence: item.PermissionEvidence})
	}
	return AttachedExtensionInventory{ObservedAt: msg.ObservedAt, Extensions: entries}, nil
}
func (t *webSocketTab) Close() error {
	t.writeMu.Lock()
	t.revoked.Store(true)
	_ = t.conn.WriteJSON(map[string]string{"type": "revoke"})
	t.writeMu.Unlock()
	return t.conn.Close()
}
func (t *webSocketTab) Done() <-chan struct{} { return t.done }
func (t *webSocketTab) Revoked() bool         { return t.revoked.Load() }

type TabObservation struct {
	URL   string `json:"url"`
	Title string `json:"title"`
	Text  string `json:"text,omitempty"`
}
type TabAction struct {
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"`
}

type PairRequest struct {
	SessionID, LeaseID, OwnerID, CallerID                          string
	PermissionRef, PairingNonce, TabID, URL, Title, ConsentReceipt string
	Profile                                                        sessionpermission.Profile
	LeaseReceipt                                                   string
}

type PairTabRequest struct {
	PairingCode string `json:"pairing_code"`
	Tab         struct {
		ID    int64  `json:"id"`
		URL   string `json:"url"`
		Title string `json:"title,omitempty"`
	} `json:"tab"`
}

type PairGrant struct {
	SessionID         string        `json:"session_id"`
	WebSocketURL      string        `json:"websocket_url"`
	LeaseReceipt      string        `json:"lease_receipt"`
	PermissionProfile BridgeProfile `json:"permission_profile"`
	ExpiresAt         time.Time     `json:"expires_at"`
	EvidenceWarning   string        `json:"evidence_warning,omitempty"`
}

// BridgeProfile exposes only client-side selected-tab checks. Sensitive
// credential IDs, grant material, budgets, and internal profile IDs stay in BAS.
type BridgeProfile struct {
	Effect          string   `json:"effect"`
	URLAllow        []string `json:"url_allow,omitempty"`
	URLDeny         []string `json:"url_deny,omitempty"`
	ExtensionIDs    []string `json:"extension_ids,omitempty"`
	View            bool     `json:"view"`
	Control         bool     `json:"control"`
	ExtensionAccess bool     `json:"extension_access"`
}

func randomBridgeToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func bridgeDigest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (b *Broker) SetSelectedTabBridge(bridge SelectedTabBridge) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bridge = bridge
}

func (b *Broker) SetTabConsentIssuer(issuer TabConsentIssuer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tabConsentIssuer = issuer
}

// PairSelectedTab consumes a one-time code, binds the actual selected tab to
// the trusted run and current caller, and issues a short-lived socket receipt.
func (b *Broker) PairSelectedTab(ctx context.Context, req PairTabRequest, socketURL string) (PairGrant, error) {
	if req.PairingCode == "" || req.Tab.ID <= 0 || req.Tab.URL == "" {
		return PairGrant{}, errors.New("pairing code and selected tab URL are required")
	}
	if len(req.PairingCode) > 128 || len(req.Tab.URL) > 8192 || len(req.Tab.Title) > 2048 {
		return PairGrant{}, errors.New("selected-tab pairing fields exceed their size limits")
	}
	parsed, err := url.ParseRequestURI(req.Tab.URL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil {
		return PairGrant{}, errors.New("selected tab URL must be an HTTP or HTTPS URL without credentials")
	}
	callerID, _ := b.caller.BrowserCallerID(ctx)
	codeHash := bridgeDigest(req.PairingCode)
	b.mu.Lock()
	var e *entry
	for _, candidate := range b.entries {
		if candidate.pairingCodeHash == codeHash && candidate.pairingExpires.After(time.Now()) && !candidate.pairingConsumed && !candidate.revoking.Load() {
			e = candidate
			break
		}
	}
	if e == nil {
		b.mu.Unlock()
		return PairGrant{}, errors.New("pairing code is invalid, expired, or already used")
	}
	// The one-time code is the extension's authentication material. If an
	// authenticated BAS client context is also present, require it to match.
	if callerID != "" && callerID != e.callerID {
		b.mu.Unlock()
		return PairGrant{}, errors.New("pairing code belongs to a different trusted caller")
	}
	callerID = e.callerID
	e.pairingConsumed = true
	bridge := b.bridge
	consentIssuer := b.tabConsentIssuer
	b.mu.Unlock()
	if err := b.lockEntry(e); err != nil {
		return PairGrant{}, err
	}
	defer e.opMu.Unlock()
	ctx = WithBrowserAgentCredentials(ctx, e.identityToken, e.grantReceipt)
	e.identityToken = ""
	if bridge == nil {
		return PairGrant{}, errors.New("selected-tab bridge owner unavailable")
	}
	if consentIssuer == nil {
		return PairGrant{}, errors.New("selected-tab consent owner unavailable; pairing fails closed")
	}
	if _, ok := consentIssuer.(TabConsentRevoker); !ok {
		return PairGrant{}, errors.New("selected-tab consent revoker unavailable; pairing fails closed")
	}
	if strings.TrimSpace(e.runID) == "" {
		return PairGrant{}, errors.New("trusted Agent Manager run id unavailable for consent revocation")
	}
	if e.profile.ProfileMode != "user_attached" || e.backend != string(sessionpermission.BackendUserAttached) {
		return PairGrant{}, sessionpermission.ErrDenied
	}
	if !e.profile.AllowsURL(req.Tab.URL) {
		return PairGrant{}, sessionpermission.ErrDenied
	}
	expiry := time.Now().Add(5 * time.Minute)
	if e.profile.MaxDuration > 0 {
		deadline := e.openedAt.Add(e.profile.MaxDuration)
		if deadline.Before(expiry) {
			expiry = deadline
		}
	}
	if !expiry.After(time.Now()) {
		return PairGrant{}, sessionpermission.ErrDenied
	}
	consentReceipt, err := consentIssuer.IssueBrowserTabConsent(ctx, TabConsentRequest{CallerID: callerID, GrantReceipt: e.grantReceipt, SessionID: e.sessionID, LeaseID: e.leaseID, OwnerID: e.ownerID, PermissionRef: e.permissionRef, TabID: fmt.Sprint(req.Tab.ID), URL: req.Tab.URL, PairingNonce: e.pairingNonce, ExpiresAt: expiry})
	if err != nil || consentReceipt == "" {
		if err == nil {
			err = errors.New("consent owner returned an empty receipt")
		}
		return PairGrant{}, fmt.Errorf("issue selected-tab consent: %w", err)
	}
	// Consent issuance is an external owner call. Its response must not revive
	// a grant whose bounded tab/session lifetime elapsed while that call ran.
	if e.revoking.Load() {
		return PairGrant{}, errors.New("selected-tab consent was revoked before pairing completed")
	}
	if !time.Now().Before(expiry) {
		return PairGrant{}, errors.New("selected-tab consent expired before pairing completed")
	}
	openReq := sessionpermission.OpenRequest{OwnerID: e.ownerID, CallerID: e.callerID, Backend: sessionpermission.BackendUserAttached, ProfileID: e.permissionRef, InitialURL: req.Tab.URL, GrantReceipt: e.grantReceipt, TabID: fmt.Sprint(req.Tab.ID), ConsentReceipt: consentReceipt}
	if err := openReq.Validate(e.profile); err != nil {
		return PairGrant{}, err
	}
	if err := b.openAuthorizer.AuthorizeOpen(ctx, openReq); err != nil {
		return PairGrant{}, fmt.Errorf("authorize selected-tab attachment: %w", err)
	}
	action := sessionpermission.ActionRequest{SessionID: e.sessionID, LeaseID: e.leaseID, OwnerID: e.ownerID, CallerID: callerID, Kind: "attach", CurrentURL: req.Tab.URL, TabID: fmt.Sprint(req.Tab.ID), ConsentReceipt: consentReceipt}
	if err := b.authorize(ctx, e, action); err != nil {
		return PairGrant{}, err
	}
	// Open admission and live policy verification can also cross the consent
	// deadline. Recheck immediately before minting socket authority.
	if e.revoking.Load() {
		return PairGrant{}, errors.New("selected-tab consent was revoked before pairing completed")
	}
	if !time.Now().Before(expiry) {
		return PairGrant{}, errors.New("selected-tab consent expired before pairing completed")
	}
	ticket, err := randomBridgeToken()
	if err != nil {
		return PairGrant{}, err
	}
	b.mu.Lock()
	if b.entries[e.sessionID] != e || e.closed || e.revoking.Load() {
		b.mu.Unlock()
		return PairGrant{}, ErrSessionNotOwned
	}
	e.tabID, e.tabURL, e.tabTitle, e.consentReceipt = fmt.Sprint(req.Tab.ID), req.Tab.URL, req.Tab.Title, consentReceipt
	e.socketTicketHash, e.socketTicketExpiry = bridgeDigest(ticket), expiry
	e.leaseReceipt, e.leaseCaller = randomLeaseReceipt(), callerID
	b.mu.Unlock()
	b.scheduleSocketExpiry(e, expiry)
	wsURL, err := makeWebSocketURL(socketURL, ticket)
	if err != nil {
		return PairGrant{}, err
	}
	evidenceWarning := ""
	if err := b.recordSessionEvidence(ctx, e, "opened"); err != nil {
		evidenceWarning = bridgeEvidenceWarning
	}
	if err := b.recordActionEvidence(ctx, e, "attach", "allowed"); err != nil {
		evidenceWarning = bridgeEvidenceWarning
	}
	extensionIDs := make([]string, 0, len(e.profile.ExtensionPermissions))
	for id := range e.profile.ExtensionPermissions {
		extensionIDs = append(extensionIDs, id)
	}
	sort.Strings(extensionIDs)
	bridgeProfile := BridgeProfile{Effect: string(e.profile.Effect), URLAllow: append([]string(nil), e.profile.URLAllow...), URLDeny: append([]string(nil), e.profile.URLDeny...), ExtensionIDs: extensionIDs, View: e.profile.View, Control: e.profile.Control, ExtensionAccess: e.profile.ExtensionAccess}
	return PairGrant{SessionID: e.sessionID, WebSocketURL: wsURL, LeaseReceipt: e.leaseReceipt, PermissionProfile: bridgeProfile, ExpiresAt: expiry, EvidenceWarning: evidenceWarning}, nil
}

func randomLeaseReceipt() string {
	value, err := randomBridgeToken()
	if err != nil {
		return uuid.NewString()
	}
	return value
}

func makeWebSocketURL(raw, ticket string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("invalid bridge service origin")
	}
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else {
		u.Scheme = "wss"
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/browser/bridge/socket"
	q := u.Query()
	q.Set("ticket", ticket)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (b *Broker) ConnectSelectedTab(w http.ResponseWriter, r *http.Request, extensionOrigin string) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		http.Error(w, "ticket required", http.StatusUnauthorized)
		return
	}
	hash := bridgeDigest(ticket)
	b.mu.Lock()
	var e *entry
	for _, candidate := range b.entries {
		if candidate.socketTicketHash == hash && candidate.socketTicketExpiry.After(time.Now()) && !candidate.socketConnected && !candidate.closed && !candidate.revoking.Load() {
			e = candidate
			break
		}
	}
	if e == nil {
		b.mu.Unlock()
		http.Error(w, "ticket invalid, expired, or used", http.StatusUnauthorized)
		return
	}
	e.socketConnected = true
	bridge := b.bridge
	request := PairRequest{SessionID: e.sessionID, LeaseID: e.leaseID, OwnerID: e.ownerID, CallerID: e.leaseCaller, PermissionRef: e.permissionRef, PairingNonce: e.pairingNonce, TabID: e.tabID, URL: e.tabURL, Title: e.tabTitle, ConsentReceipt: e.consentReceipt, Profile: e.profile, LeaseReceipt: e.leaseReceipt}
	b.mu.Unlock()
	if bridge == nil {
		http.Error(w, "bridge owner unavailable", http.StatusServiceUnavailable)
		return
	}
	upgrader := websocket.Upgrader{Subprotocols: []string{bridgeSubprotocol}, CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		parsed, err := url.Parse(origin)
		if err != nil {
			return false
		}
		if extensionOrigin != "" && origin == extensionOrigin {
			return true
		}
		return parsed.Host == r.Host && (parsed.Scheme == "https" || parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1")
	}}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		b.clearSocketTicket(e)
		return
	}
	if conn.Subprotocol() != bridgeSubprotocol {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseProtocolError, "protocol required"))
		_ = conn.Close()
		b.clearSocketTicket(e)
		return
	}
	attached, err := bridge.Attach(r.Context(), request, conn)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, mustJSON(map[string]string{"type": "lease_denied"}))
		_ = conn.Close()
		b.clearSocketTicket(e)
		return
	}
	e.opMu.Lock()
	b.mu.Lock()
	attachAllowed := b.entries[e.sessionID] == e && !e.closed && !e.revoking.Load() && time.Now().Before(e.socketTicketExpiry)
	if attachAllowed {
		e.attachedSocket, e.attachedTab = conn, attached
		go func() {
			<-attached.Done()
			b.mu.RLock()
			consentWasAlreadyRevoked := e.closed
			b.mu.RUnlock()
			if attached.Revoked() && !consentWasAlreadyRevoked {
				e.revoking.Store(true)
			}
			e.opMu.Lock()
			b.mu.Lock()
			if b.entries[e.sessionID] == e && e.attachedTab == attached {
				e.attachedSocket, e.attachedTab = nil, nil
				e.socketConnected = false
				if attached.Revoked() {
					e.closed = true
					delete(b.entries, e.sessionID)
				}
			}
			b.mu.Unlock()
			e.opMu.Unlock()
			if attached.Revoked() && !consentWasAlreadyRevoked {
				b.revokeTabConsent(e)
			}
		}()
	} else if b.entries[e.sessionID] == e {
		e.socketConnected = false
		e.socketTicketHash = ""
	}
	b.mu.Unlock()
	e.opMu.Unlock()
	if !attachAllowed {
		_ = attached.Close()
		_ = conn.Close()
	}
}

func (b *Broker) scheduleSocketExpiry(e *entry, expiry time.Time) {
	delay := time.Until(expiry)
	if delay < 0 {
		delay = 0
	}
	time.AfterFunc(delay, func() {
		b.mu.Lock()
		var conn *websocket.Conn
		var tab AttachedTab
		if b.entries[e.sessionID] == e && !time.Now().Before(expiry) {
			e.revoking.Store(true)
			e.closed = true
			e.closedAt = time.Now().UTC()
			conn, tab = e.attachedSocket, e.attachedTab
			delete(b.entries, e.sessionID)
		}
		b.mu.Unlock()
		if tab != nil {
			_ = tab.Close()
		} else if conn != nil {
			_ = conn.Close()
		}
	})
}

func (b *Broker) clearSocketTicket(e *entry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.entries[e.sessionID] == e {
		e.socketConnected = false
		e.socketTicketHash = ""
	}
}
func mustJSON(value any) []byte { data, _ := json.Marshal(value); return data }

func bridgeAction(instruction contracts.CompiledInstruction) (TabAction, string, error) {
	if instruction.Action == nil {
		return TabAction{}, "", errors.New("instruction.action is required")
	}
	if nav := instruction.Action.GetNavigate(); nav != nil {
		if nav.GetUrl() == "" {
			return TabAction{}, "", errors.New("navigation URL is required")
		}
		return TabAction{Type: "navigate", URL: nav.GetUrl()}, nav.GetUrl(), nil
	}
	if click := instruction.Action.GetClick(); click != nil {
		if click.GetSelector() == "" {
			return TabAction{}, "", errors.New("click selector is required")
		}
		return TabAction{Type: "click", Selector: click.GetSelector()}, "", nil
	}
	if input := instruction.Action.GetInput(); input != nil {
		if input.GetSelector() == "" || input.GetIsSensitive() {
			return TabAction{}, "", errors.New("selected-tab bridge rejects sensitive or unscoped input")
		}
		return TabAction{Type: "type", Selector: input.GetSelector(), Value: input.GetValue()}, "", nil
	}
	return TabAction{}, "", errors.New("selected-tab bridge supports only typed navigate, click, and non-sensitive input")
}
