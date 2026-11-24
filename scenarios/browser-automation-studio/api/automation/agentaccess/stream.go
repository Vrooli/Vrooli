package agentaccess

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

type StreamRole string

const (
	StreamObserve StreamRole = "observe"
	StreamControl StreamRole = "control"
)

type StreamReceiptClaims struct {
	SessionID     string
	LeaseID       string
	OwnerID       string
	CallerID      string
	PermissionRef string
	Role          StreamRole
	ExpiresAt     time.Time
}

// StreamReceiptVerifier delegates opaque role receipts to the existing
// authority. Broker-local cookies are issued only after this check succeeds.
type StreamReceiptVerifier interface {
	VerifyBrowserStreamReceipt(context.Context, string, string, StreamRole) (StreamReceiptClaims, error)
}

type SessionStreamCookieIssuer interface {
	IssueSessionStreamCookieRole(string, string, string, string, time.Time) (string, string, error)
	RevokeSessionStreamCookie(string)
}

type StreamCookie struct {
	Name      string     `json:"-"`
	Value     string     `json:"-"`
	Path      string     `json:"-"`
	Role      StreamRole `json:"role"`
	SessionID string     `json:"session_id"`
	ExpiresAt time.Time  `json:"expires_at"`
}

func (b *Broker) SetStreamAuthority(verifier StreamReceiptVerifier, cookies SessionStreamCookieIssuer) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.streamVerifier = verifier
	b.streamCookies = cookies
}

func (b *Broker) IssueStreamCookie(ctx context.Context, sessionID, receipt string, role StreamRole) (StreamCookie, error) {
	if role != StreamObserve && role != StreamControl {
		return StreamCookie{}, errors.New("stream role must be observe or control")
	}
	if strings.TrimSpace(receipt) == "" {
		return StreamCookie{}, errors.New("opaque stream receipt is required")
	}
	e, err := b.get(sessionID)
	if err != nil {
		return StreamCookie{}, err
	}
	if err := b.lockEntry(e); err != nil {
		return StreamCookie{}, err
	}
	defer e.opMu.Unlock()
	if e.session == nil {
		return StreamCookie{}, errors.New("selected-tab bridge has no BAS Hub frame stream")
	}
	b.mu.RLock()
	verifier, cookies := b.streamVerifier, b.streamCookies
	b.mu.RUnlock()
	if verifier == nil || cookies == nil {
		return StreamCookie{}, errors.New("stream receipt authority unavailable")
	}
	claims, err := verifier.VerifyBrowserStreamReceipt(ctx, receipt, sessionID, role)
	if err != nil {
		return StreamCookie{}, fmt.Errorf("verify browser stream receipt: %w", err)
	}
	executionID, leaseID := e.ownerID, e.leaseID
	if e.session != nil {
		executionID, leaseID = e.session.OwnerIdentity()
	}
	if claims.SessionID != sessionID || claims.LeaseID != leaseID || claims.OwnerID != executionID || claims.CallerID == "" || claims.CallerID != e.callerID || claims.PermissionRef != e.permissionRef || claims.Role != role || !claims.ExpiresAt.After(time.Now()) || claims.ExpiresAt.After(time.Now().Add(5*time.Minute)) {
		return StreamCookie{}, errors.New("stream receipt claims do not bind to the active session lease, owner, profile, role, and short expiry")
	}
	if role == StreamObserve && !e.profile.View {
		return StreamCookie{}, sessionpermission.ErrDenied
	}
	if role == StreamControl && (!e.profile.Control || e.profile.Effect == sessionpermission.EffectObserve || e.handedOff) {
		return StreamCookie{}, sessionpermission.ErrDenied
	}
	if e.profile.Approval != sessionpermission.ApprovalNone {
		return StreamCookie{}, errors.New("stream receipt cannot replace a per-action approval requirement")
	}
	name, value, err := cookies.IssueSessionStreamCookieRole(sessionID, executionID, leaseID, string(role), claims.ExpiresAt)
	if err != nil {
		return StreamCookie{}, err
	}
	return StreamCookie{Name: name, Value: value, Path: "/ws/sessions/" + sessionID, Role: role, SessionID: sessionID, ExpiresAt: claims.ExpiresAt}, nil
}

func (b *Broker) RevokeStreamCookie(sessionID string) {
	b.mu.RLock()
	cookies := b.streamCookies
	b.mu.RUnlock()
	if cookies != nil {
		cookies.RevokeSessionStreamCookie(sessionID)
	}
}
