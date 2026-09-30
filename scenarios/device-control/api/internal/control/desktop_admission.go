package control

import (
	"context"
	"crypto/ed25519"
	"net"
	"sort"
	"sync"
	"time"

	"device-control/internal/desktophelper"
	"device-control/internal/sessions"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/targetmodel"
)

// LocalDesktopAdmission is the local user's admission owner. It is deliberately
// not mounted on the public SessionService: that service's actor field is not
// authenticated identity. A caller must present an accepted Unix connection.
// Remote admission needs Bridge's authenticated actor and scoped grant path.
type LocalDesktopAdmission struct {
	admissions   sessions.DesktopAdmissions
	webIdentity  owneridentity.Validator
	webSubject   string
	mu           sync.Mutex
	service      *Service
	key          ed25519.PrivateKey
	principal    localprincipal.Principal
	deviceID     string
	surface      targetmodel.SurfaceRef
	registration func(context.Context) (desktophelper.Registration, error)
	grants       map[string]Session
}

// The binding and registration reader come from protected owner provisioning,
// never request fields. The existing device identity is the exclusion key for
// both desktop access and ordinary device/flow sessions.
func NewLocalDesktopAdmission(service *Service, key ed25519.PrivateKey, deviceID string, surface targetmodel.SurfaceRef, registration func(context.Context) (desktophelper.Registration, error)) (*LocalDesktopAdmission, error) {
	principal, err := localprincipal.Current()
	if err != nil || service == nil || len(key) != ed25519.PrivateKeySize || deviceID == "" || surface.Validate() != nil || surface.OwnerScenario != "device-control" || registration == nil {
		return nil, sessions.ErrDesktopAdmission
	}
	admissions, err := sessions.NewSQLiteDesktopAdmissions(context.Background(), service.db)
	if err != nil {
		return nil, err
	}
	return &LocalDesktopAdmission{service: service, admissions: admissions, key: append(ed25519.PrivateKey(nil), key...), principal: principal, deviceID: deviceID, surface: surface, registration: registration, grants: make(map[string]Session)}, nil
}

// Acquire returns server-only helper authority. Tokens must not be projected to
// Portal/renderers. No actor, principal, helper generation, or target is accepted
// from the caller. Observation-only access cannot mint input or takeover rights.
func (a *LocalDesktopAdmission) Acquire(ctx context.Context, peer *net.UnixConn, ttl time.Duration, control bool) (sessions.DesktopGrant, string, error) {
	deny := func() (sessions.DesktopGrant, string, error) {
		return sessions.DesktopGrant{}, "", sessions.ErrDesktopAdmission
	}
	principal, err := localprincipal.Peer(peer)
	if err != nil || principal != a.principal || ctx.Err() != nil || ttl <= 0 || ttl > 10*time.Minute {
		return deny()
	}
	actor := principal.String()
	if identity, ok := ctx.Value(desktopWebIdentityKey{}).(owneridentity.Identity); ok {
		if !desktopIdentityAllows(identity, control, time.Now()) || identity.Subject != a.webSubject {
			return deny()
		}
		actor = identity.Subject
		if remaining := time.Until(identity.ExpiresAt); remaining < ttl {
			ttl = remaining
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, lease := range a.grants {
		if a.service.ValidateObservationLease(ctx, lease.DeviceID, lease.LeaseToken) != nil {
			delete(a.grants, id)
		}
	}
	if len(a.grants) >= 1024 {
		return deny()
	}
	registration, err := a.registration(ctx)
	if err != nil || registration.Surface != a.surface || registration.SessionID == "" || registration.HelperID == "" || registration.Epoch == ^uint64(0) {
		return deny()
	}
	var lease Session
	if control {
		lease, err = a.service.AcquireContext(ctx, a.deviceID, actor, ttl)
	} else {
		lease, err = a.service.AcquireObservationContext(ctx, a.deviceID, actor, ttl)
	}
	if err != nil {
		return sessions.DesktopGrant{}, "", err
	}
	grant := sessions.DesktopGrant{
		ID: uuid.NewString(), Principal: a.principal,
		Lease:      sessions.DesktopLease{Ref: targetmodel.SessionRef{Surface: a.surface, SessionID: lease.ID, DesktopSessionID: registration.SessionID}, Actor: actor, HelperID: registration.HelperID, Epoch: registration.Epoch + 1, ExpiresAt: lease.ExpiresAt, Control: control},
		Operations: []string{"open", "observe", "stop"}, IssuedAt: lease.CreatedAt, ExpiresAt: lease.ExpiresAt,
	}
	if control {
		grant.Operations = append(grant.Operations, "act")
	}
	if identity, ok := ctx.Value(desktopWebIdentityKey{}).(owneridentity.Identity); ok && identity.ExpiresAt.Before(grant.ExpiresAt) {
		grant.ExpiresAt = identity.ExpiresAt
		grant.Lease.ExpiresAt = identity.ExpiresAt
	}
	token, err := sessions.SignDesktopGrant(a.key, grant, time.Now())
	if err != nil {
		// A failed issuance must not leave an invisible held owner lease. Cleanup
		// uses an independent bounded context even if the caller disconnected.
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, releaseErr := a.service.ReleaseContext(cleanup, lease.ID)
		if releaseErr != nil {
			return sessions.DesktopGrant{}, "", releaseErr
		}
		return deny()
	}
	a.grants[grant.ID] = lease
	return grant, token, nil
}

// Active consults the existing owner lease on every check. Release, kill,
// expiry, or owner restart revoke helper authority, including for idle clients.
// The map intentionally does not reconstruct grants from historical sessions.
func (a *LocalDesktopAdmission) Active(ctx context.Context, id string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	lease, ok := a.grants[id]
	if !ok {
		return false, nil
	}
	if err := a.service.ValidateObservationLease(ctx, lease.DeviceID, lease.LeaseToken); err != nil {
		delete(a.grants, id)
		return false, nil
	}
	return true, nil
}

// GrantStatus publishes only non-secret IDs. It never writes lease tokens,
// signing material, or the actor's credential into helper status files.
func (a *LocalDesktopAdmission) GrantStatus(ctx context.Context) (desktophelper.GrantStatus, error) {
	if err := a.admissions.ExpireOpenReservations(ctx, time.Now()); err != nil {
		return desktophelper.GrantStatus{}, err
	}
	if ctx.Err() != nil {
		return desktophelper.GrantStatus{}, ctx.Err()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	now := time.Now().UTC()
	state := desktophelper.GrantStatus{Active: []string{}, ObservedAt: now, ExpiresAt: now.Add(desktophelper.GrantStatusLifetime)}
	for id, lease := range a.grants {
		if a.service.ValidateObservationLease(ctx, lease.DeviceID, lease.LeaseToken) != nil || !now.Before(lease.ExpiresAt) {
			delete(a.grants, id)
			continue
		}
		state.Active = append(state.Active, id)
		if lease.ExpiresAt.Before(state.ExpiresAt) {
			state.ExpiresAt = lease.ExpiresAt
		}
	}
	sort.Strings(state.Active)
	return state, nil
}
