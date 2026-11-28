package agentaccess

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/principalrole"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

const (
	BrowserPaneIssuerAudience = "scenario-authenticator:browser-automation-studio"
	BrowserPaneIssuerScope    = "browser-automation-studio:browser-pane.issue"
	BrowserPaneIssuerSuffix   = "-browser-pane-issuer"
	BrowserPaneReceiptTTL     = 2 * time.Minute
)

// BrowserPaneIssuerRole admits only Agent Manager's dedicated browser-pane
// delegation identity. It is intentionally separate from the closed-session
// evidence verifier identity and scope.
var BrowserPaneIssuerRole = principalrole.NewRole(BrowserPaneIssuerAudience, BrowserPaneIssuerScope,
	"browser-automation-studio", BrowserPaneIssuerSuffix, "browser pane receipt issuer")

func NewBrowserPaneIssuerTokenVerifier() authn.TokenVerifier {
	return BrowserPaneIssuerRole.NewVerifier()
}

type paneReceiptIssueRequest struct {
	RunID        string `json:"run_id"`
	GrantReceipt string `json:"grant_receipt"`
	ProfileID    string `json:"profile_id"`
	SessionID    string `json:"session_id"`
	Role         string `json:"role"`
	NotAfter     time.Time `json:"not_after,omitempty"`
}

type paneSessionListRequest struct {
	RunID        string `json:"run_id"`
	GrantReceipt string `json:"grant_receipt"`
	ProfileID    string `json:"profile_id"`
}

// RegisterPaneStreamReceiptRoutes adds the narrow AM delegation endpoint.
// Its verifier is nil-safe and the route stays unavailable until the dedicated
// Agent Manager principal is provisioned by the operator.
func RegisterPaneStreamReceiptRoutes(r chi.Router, authority *PaneStreamReceiptAuthority, verifier authn.TokenVerifier) {
	r.Route("/browser/internal/pane-stream-receipts", func(route chi.Router) {
		route.Use(paneIssuerMiddleware(verifier))
		route.Post("/issue", func(w http.ResponseWriter, req *http.Request) {
			if authority == nil {
				writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("browser pane receipt authority unavailable"))
				return
			}
			var body paneReceiptIssueRequest
			decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, 8<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(w, err)
				return
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				writeError(w, errors.New("request body must contain one JSON value"))
				return
			}
			runID, err := uuid.Parse(strings.TrimSpace(body.RunID))
			if err != nil {
				writeErrorStatus(w, http.StatusBadRequest, errors.New("run_id is invalid"))
				return
			}
			receipt, expiresAt, err := authority.IssueBrowserPaneReceiptUntil(req.Context(), runID, body.GrantReceipt, body.ProfileID, body.SessionID, body.Role, body.NotAfter)
			if err != nil {
				writeError(w, err)
				return
			}
			writeResponse(w, map[string]any{"session_id": body.SessionID, "role": body.Role, "receipt": receipt, "expires_at": expiresAt}, nil)
		})
		route.Post("/list", func(w http.ResponseWriter, req *http.Request) {
			if authority == nil {
				writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("browser pane receipt authority unavailable"))
				return
			}
			var body paneSessionListRequest
			decoder := json.NewDecoder(http.MaxBytesReader(w, req.Body, 8<<10))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&body); err != nil {
				writeError(w, err)
				return
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				writeError(w, errors.New("request body must contain one JSON value"))
				return
			}
			runID, err := uuid.Parse(strings.TrimSpace(body.RunID))
			if err != nil || strings.TrimSpace(body.GrantReceipt) == "" || strings.TrimSpace(body.ProfileID) == "" {
				writeErrorStatus(w, http.StatusBadRequest, errors.New("browser pane run selector invalid"))
				return
			}
			writeResponse(w, map[string]any{"sessions": authority.ListBrowserPaneSessions(runID, body.GrantReceipt, body.ProfileID)}, nil)
		})
	})
}

func paneIssuerMiddleware(verifier authn.TokenVerifier) func(http.Handler) http.Handler {
	if verifier == nil {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeErrorStatus(w, http.StatusServiceUnavailable, errors.New("browser pane issuer identity is not configured"))
			})
		}
	}
	provider := authn.TokenProvider{ProviderSource: identity.SourceScenarioAuthenticator, Verifier: verifier,
		Extract: func(req *http.Request) string {
			value, found := strings.CutPrefix(strings.TrimSpace(req.Header.Get("Authorization")), "Bearer ")
			if !found {
				return ""
			}
			return strings.TrimSpace(value)
		}}
	return func(next http.Handler) http.Handler {
		return authn.Middleware(authn.Config{Providers: []authn.Provider{provider}})(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			principal, ok := identity.PrincipalFromContext(req.Context())
			if !ok || !principal.IsVerified() || principal.Kind != identity.ActorService {
				writeErrorStatus(w, http.StatusUnauthorized, errors.New("verified Agent Manager service identity required"))
				return
			}
			source, err := BrowserPaneIssuerRole.Source(principal)
			if err != nil || source != "agent-manager" {
				writeErrorStatus(w, http.StatusForbidden, errors.New("Agent Manager browser pane issuer role required"))
				return
			}
			if _, err := authn.RequireCapability(req.Context(), BrowserPaneIssuerScope); err != nil {
				writeErrorStatus(w, http.StatusForbidden, err)
				return
			}
			next.ServeHTTP(w, req)
		}))
	}
}

type browserPaneReceiptRecord struct {
	claims StreamReceiptClaims
	runID  uuid.UUID
}

// PaneStreamReceiptAuthority stores only hashes of short-lived, single-use
// stream receipts. BAS resolves the selected session against its own Broker
// before minting; AM's service identity proves that AM already checked its
// human/run/thread/grant/profile association.
type PaneStreamReceiptAuthority struct {
	broker  *Broker
	mu      sync.Mutex
	records map[string]browserPaneReceiptRecord
}

func NewPaneStreamReceiptAuthority(broker *Broker) *PaneStreamReceiptAuthority {
	return &PaneStreamReceiptAuthority{broker: broker, records: make(map[string]browserPaneReceiptRecord)}
}

func (a *PaneStreamReceiptAuthority) IssueBrowserPaneReceipt(ctx context.Context, runID uuid.UUID, grantReceipt, profileID, sessionID, role string) (string, time.Time, error) {
	return a.IssueBrowserPaneReceiptUntil(ctx, runID, grantReceipt, profileID, sessionID, role, time.Time{})
}

func (a *PaneStreamReceiptAuthority) IssueBrowserPaneReceiptUntil(_ context.Context, runID uuid.UUID, grantReceipt, profileID, sessionID, role string, notAfter time.Time) (string, time.Time, error) {
	if a == nil || a.broker == nil {
		return "", time.Time{}, errors.New("browser pane receipt authority unavailable")
	}
	if runID == uuid.Nil || strings.TrimSpace(sessionID) == "" || (role != string(StreamObserve) && role != string(StreamControl)) || strings.TrimSpace(grantReceipt) == "" || strings.TrimSpace(profileID) == "" {
		return "", time.Time{}, errors.New("browser pane receipt binding is incomplete")
	}
	e, err := a.broker.get(sessionID)
	if err != nil {
		return "", time.Time{}, err
	}
	if err := a.broker.lockEntry(e); err != nil {
		return "", time.Time{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil || e.closed || e.callerID != "run:"+runID.String() || e.profile.ProfileID != profileID || subtle.ConstantTimeCompare([]byte(e.grantReceipt), []byte(grantReceipt)) != 1 {
		return "", time.Time{}, errors.New("selected BAS session does not match the AM run grant and profile")
	}
	if e.profile.Approval != sessionpermission.ApprovalNone {
		return "", time.Time{}, errors.New("stream receipt cannot replace per-action approval")
	}
	if role == string(StreamObserve) && !e.profile.View {
		return "", time.Time{}, sessionpermission.ErrDenied
	}
	if role == string(StreamControl) && (!e.profile.Control || e.profile.Effect == sessionpermission.EffectObserve || e.handedOff) {
		return "", time.Time{}, sessionpermission.ErrDenied
	}
	ownerID, leaseID := e.ownerID, e.leaseID
	if e.session != nil {
		ownerID, leaseID = e.session.OwnerIdentity()
	}
	if ownerID == "" || leaseID == "" || e.callerID == "" || e.permissionRef == "" {
		return "", time.Time{}, errors.New("active BAS session lacks broker-owned receipt bindings")
	}
	receipt, err := randomBridgeToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expires := time.Now().Add(BrowserPaneReceiptTTL)
	if !notAfter.IsZero() && notAfter.Before(expires) {
		expires = notAfter
	}
	if !expires.After(time.Now()) {
		return "", time.Time{}, errors.New("browser pane control lease has expired")
	}
	digest := sha256.Sum256([]byte(receipt))
	key := hex.EncodeToString(digest[:])
	a.mu.Lock()
	for existing, record := range a.records {
		if !record.claims.ExpiresAt.After(time.Now()) {
			delete(a.records, existing)
		}
	}
	a.records[key] = browserPaneReceiptRecord{runID: runID, claims: StreamReceiptClaims{SessionID: sessionID, LeaseID: leaseID, OwnerID: ownerID, CallerID: e.callerID, PermissionRef: e.permissionRef, Role: StreamRole(role), ExpiresAt: expires}}
	a.mu.Unlock()
	return receipt, expires, nil
}

// ListBrowserPaneSessions returns only active, view-enabled manager sessions
// admitted for this exact run grant and profile. It is a server-side selector
// discovery method; clients receive no lease, owner, caller, URL, or receipt.
func (a *PaneStreamReceiptAuthority) ListBrowserPaneSessions(runID uuid.UUID, grantReceipt, profileID string) []map[string]any {
	if a == nil || a.broker == nil || a.broker.manager == nil || runID == uuid.Nil || strings.TrimSpace(grantReceipt) == "" || strings.TrimSpace(profileID) == "" {
		return []map[string]any{}
	}
	a.broker.mu.RLock()
	entries := make([]*entry, 0, len(a.broker.entries))
	for _, e := range a.broker.entries {
		entries = append(entries, e)
	}
	a.broker.mu.RUnlock()
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		if err := a.broker.lockEntry(e); err != nil {
			continue
		}
		active := browserPaneEntryMatches(e, runID, grantReceipt, profileID)
		if active {
			if owned, ok := a.broker.manager.Get(e.sessionID); !ok || owned != e.session {
				active = false
			}
		}
		if active {
			out = append(out, map[string]any{"session_id": e.sessionID, "mode": e.backend, "can_control": e.profile.Control && e.profile.Effect != sessionpermission.EffectObserve && !e.handedOff})
		}
		e.opMu.Unlock()
	}
	return out
}

func browserPaneEntryMatches(e *entry, runID uuid.UUID, grantReceipt, profileID string) bool {
	return e != nil && runID != uuid.Nil && strings.TrimSpace(grantReceipt) != "" && strings.TrimSpace(profileID) != "" && !e.closed && !e.revoking.Load() && e.callerID == "run:"+runID.String() &&
		subtle.ConstantTimeCompare([]byte(e.grantReceipt), []byte(grantReceipt)) == 1 && e.permissionRef == profileID &&
		e.profile.ProfileID == profileID && e.profile.View && e.profile.Approval == sessionpermission.ApprovalNone && e.session != nil
}

func (a *PaneStreamReceiptAuthority) VerifyBrowserStreamReceipt(_ context.Context, receipt, sessionID string, role StreamRole) (StreamReceiptClaims, error) {
	if a == nil || len(receipt) < 32 {
		return StreamReceiptClaims{}, errors.New("browser stream receipt invalid")
	}
	digest := sha256.Sum256([]byte(receipt))
	key := hex.EncodeToString(digest[:])
	a.mu.Lock()
	record, ok := a.records[key]
	if ok {
		delete(a.records, key)
	}
	a.mu.Unlock()
	if !ok || !record.claims.ExpiresAt.After(time.Now()) || record.claims.SessionID != sessionID || record.claims.Role != role || record.claims.CallerID != "run:"+record.runID.String() {
		return StreamReceiptClaims{}, errors.New("browser stream receipt is missing, expired, consumed, or mismatched")
	}
	return record.claims, nil
}

// BrowserPaneIssuerPrincipalIdentity is documented for the corresponding
// Agent Manager service-principal declaration and credential packet.
const BrowserPaneIssuerPrincipalIdentity = "scenario-authenticator/service-principal/agent-manager-browser-pane-issuer/browser-automation-studio"
