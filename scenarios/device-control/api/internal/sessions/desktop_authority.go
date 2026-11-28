package sessions

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"strings"
	"time"

	"github.com/vrooli/api-core/localprincipal"
)

const desktopGrantDomain = "vrooli.device-control.desktop-grant.v1."

// DesktopGrant is issued by destination admission after operator authorization.
// Principal binds the grant to the local API-to-helper peer, not to a UID from
// a request body. The lease names the epoch that Open will install.
type DesktopGrant struct {
	Revoked    bool                     `json:"revoked,omitempty"`
	ID         string                   `json:"id"`
	Principal  localprincipal.Principal `json:"principal"`
	Lease      DesktopLease             `json:"lease"`
	Operations []string                 `json:"operations"`
	IssuedAt   time.Time                `json:"issued_at"`
	ExpiresAt  time.Time                `json:"expires_at"`
}

func validDesktopGrant(g DesktopGrant, now time.Time) bool {
	if g.ID == "" || len(g.ID) > 128 || g.Principal == "" || len(g.Principal) > 256 || g.Lease.Ref.Validate() != nil || g.Lease.Actor == "" || g.Lease.HelperID == "" || g.Lease.Ref.DesktopSessionID == "" || g.Lease.Epoch == 0 || len(g.Operations) == 0 || len(g.Operations) > 5 || g.IssuedAt.IsZero() || now.Before(g.IssuedAt) || !now.Before(g.ExpiresAt) || !g.IssuedAt.Before(g.ExpiresAt) || g.ExpiresAt.Sub(g.IssuedAt) > 10*time.Minute || g.Lease.ExpiresAt.After(g.ExpiresAt) || !now.Before(g.Lease.ExpiresAt) {
		return false
	}
	seen := map[string]bool{}
	for _, op := range g.Operations {
		if seen[op] {
			return false
		}
		seen[op] = true
		switch op {
		case "open", "transfer", "stop", "observe":
		case "act":
			if !g.Lease.Control {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// SignDesktopGrant runs in the trusted admission owner. Signing keys must never
// be available to Portal content, helper clients, or the helper verifier.
func SignDesktopGrant(key ed25519.PrivateKey, grant DesktopGrant, now time.Time) (string, error) {
	if len(key) != ed25519.PrivateKeySize || !validDesktopGrant(grant, now) {
		return "", ErrDesktopAdmission
	}
	data, err := json.Marshal(grant)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(data)
	if len(body) > 12*1024 {
		return "", ErrDesktopAdmission
	}
	signature := ed25519.Sign(key, []byte(desktopGrantDomain+body))
	return "DCG1." + body + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// DesktopGrantStatus checks revocation against owner state on every operation.
// Failure to consult that state refuses admission; signature validity alone
// does not prove continued authority.
type DesktopGrantStatus func(context.Context, string) (active bool, err error)

type SignedDesktopAuthority struct {
	public ed25519.PublicKey
	active DesktopGrantStatus
	now    func() time.Time
}

func NewSignedDesktopAuthority(public ed25519.PublicKey, active DesktopGrantStatus) (*SignedDesktopAuthority, error) {
	if len(public) != ed25519.PublicKeySize || active == nil {
		return nil, ErrDesktopAdmission
	}
	return &SignedDesktopAuthority{public: append(ed25519.PublicKey(nil), public...), active: active, now: time.Now}, nil
}

type desktopPeerCredential struct {
	principal   localprincipal.Principal
	token       string
	cleanupOnly bool
}
type desktopPeerContextKey struct{}

// AuthenticatePeer extracts kernel credentials from the accepted connection.
// The private context key prevents an HTTP header or JSON field from becoming
// a principal. The helper listener must use this on its accepted Unix socket;
// unsupported peer credential transports fail closed, with no token fallback.
func (a *SignedDesktopAuthority) AuthenticatePeer(ctx context.Context, conn *net.UnixConn, token string) (context.Context, error) {
	if len(token) > 16*1024 {
		return nil, ErrDesktopAdmission
	}
	peer, err := localprincipal.Peer(conn)
	if err != nil {
		return nil, ErrDesktopAdmission
	}
	credential := desktopPeerCredential{principal: peer, token: token}
	if _, err := a.verify(ctx, credential); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, desktopPeerContextKey{}, credential), nil
}

// Cleanup credentials have a separate verification path and cannot authorize input.
func (a *SignedDesktopAuthority) verify(ctx context.Context, credential desktopPeerCredential) (DesktopGrant, error) {
	if credential.cleanupOnly {
		return DesktopGrant{}, ErrDesktopAdmission
	}
	return a.verifyCredential(ctx, credential, false)
}

func (a *SignedDesktopAuthority) verifyCredential(ctx context.Context, credential desktopPeerCredential, historical bool) (DesktopGrant, error) {
	deny := func() (DesktopGrant, error) { return DesktopGrant{}, ErrDesktopAdmission }
	if ctx.Err() != nil || len(credential.token) > 16*1024 {
		return deny()
	}
	parts := strings.Split(credential.token, ".")
	if len(parts) != 3 || parts[0] != "DCG1" {
		return deny()
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(a.public, []byte(desktopGrantDomain+parts[1]), signature) {
		return deny()
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return deny()
	}
	var grant DesktopGrant
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&grant) != nil || decoder.Decode(new(any)) != io.EOF || grant.Principal != credential.principal {
		return deny()
	}
	verifiedAt := a.now()
	if historical {
		if verifiedAt.Before(grant.IssuedAt) {
			return deny()
		}
		verifiedAt = grant.IssuedAt
	}
	if !validDesktopGrant(grant, verifiedAt) || (!historical && grant.Revoked) {
		return deny()
	}
	if historical {
		return grant, nil
	}
	active, err := a.active(ctx, grant.ID)
	if err != nil || !active {
		return deny()
	}
	return grant, nil
}

func (a *SignedDesktopAuthority) Authorize(ctx context.Context, lease DesktopLease, operation string) error {
	credential, ok := ctx.Value(desktopPeerContextKey{}).(desktopPeerCredential)
	if !ok {
		return ErrDesktopAdmission
	}
	grant, err := a.verify(ctx, credential)
	if err != nil || !sameDesktopLease(grant.Lease, lease) {
		return ErrDesktopAdmission
	}
	for _, op := range grant.Operations {
		if op == operation {
			return nil
		}
	}
	return ErrDesktopAdmission
}

// DesktopGrantMonitor binds an accepted lease to a non-secret grant ID so
// destination maintenance can observe revocation without retaining bearer tokens.
type DesktopGrantMonitor interface {
	BindGrant(context.Context, DesktopLease) (string, error)
	CheckGrant(context.Context, string) error
}

func (a *SignedDesktopAuthority) BindGrant(ctx context.Context, lease DesktopLease) (string, error) {
	credential, ok := ctx.Value(desktopPeerContextKey{}).(desktopPeerCredential)
	if !ok {
		return "", ErrDesktopAdmission
	}
	grant, err := a.verify(ctx, credential)
	if err != nil || !sameDesktopLease(grant.Lease, lease) {
		return "", ErrDesktopAdmission
	}
	return grant.ID, nil
}

func (a *SignedDesktopAuthority) CheckGrant(ctx context.Context, id string) error {
	if id == "" || ctx.Err() != nil {
		return ErrDesktopAdmission
	}
	active, err := a.active(ctx, id)
	if err != nil || !active {
		return ErrDesktopAdmission
	}
	return nil
}

// AuthenticateCleanupPeer accepts a signed historical lease solely for receipt
// readback. It still requires the original kernel peer and never consults or
// restores the revoked input grant. The owner must authenticate its actor first.
func (a *SignedDesktopAuthority) AuthenticateCleanupPeer(ctx context.Context, conn *net.UnixConn, token string) (context.Context, error) {
	peer, err := localprincipal.Peer(conn)
	if err != nil {
		return nil, ErrDesktopAdmission
	}
	credential := desktopPeerCredential{principal: peer, token: token, cleanupOnly: true}
	if _, err := a.verifyCredential(ctx, credential, true); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, desktopPeerContextKey{}, credential), nil
}

func (a *SignedDesktopAuthority) AuthorizeCleanupRead(ctx context.Context, lease DesktopLease) error {
	credential, ok := ctx.Value(desktopPeerContextKey{}).(desktopPeerCredential)
	if !ok || !credential.cleanupOnly {
		return ErrDesktopAdmission
	}
	grant, err := a.verifyCredential(ctx, credential, true)
	if err != nil || !sameDesktopLease(grant.Lease, lease) {
		return ErrDesktopAdmission
	}
	for _, op := range grant.Operations {
		if op == "stop" {
			return nil
		}
	}
	return ErrDesktopAdmission
}

// RevokedCleanupGrant verifies an owner-signed permanent revocation statement.
// An ordinary historical grant, even currently absent from status, is insufficient.
func (a *SignedDesktopAuthority) RevokedCleanupGrant(ctx context.Context, lease DesktopLease) (string, error) {
	if err := a.AuthorizeCleanupRead(ctx, lease); err != nil {
		return "", err
	}
	credential := ctx.Value(desktopPeerContextKey{}).(desktopPeerCredential)
	grant, err := a.verifyCredential(ctx, credential, true)
	if err != nil || !grant.Revoked {
		return "", ErrDesktopAdmission
	}
	return grant.ID, nil
}
