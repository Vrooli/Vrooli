package devices

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"device-sync-hub/internal/auth"
	"device-sync-hub/internal/clock"
)

// defaultPairingTTL is how long an issued pairing code stays redeemable. Short
// by design: a single-use code that lives for minutes resists online guessing
// far better than a long-lived one. Overridable via Config for tests.
const defaultPairingTTL = 15 * time.Minute

// defaultDeviceName is the fallback name when a joining device supplies none.
const defaultDeviceName = "New device"

// Service is the application-layer surface the devices handlers depend on. It
// owns validation, pairing policy (TTL, single-use, trust transitions), device
// token issuance, and the revocation hand-off to the authenticator. The
// handler is thin around it: decode → call service → translate errors.
type Service interface {
	List(ctx context.Context, ownerID string) ([]Device, error)
	Get(ctx context.Context, ownerID, id string) (Device, error)

	// IssuePairingCode mints a short-TTL single-use code bound to ownerID. The
	// returned PairingCode carries the raw Code (the only time it exists).
	IssuePairingCode(ctx context.Context, ownerID, deviceName string) (PairingCode, error)

	// RedeemPairingCode consumes a valid code and registers a TRUSTED device,
	// returning the device plus its one-time raw token. ErrInvalidPairingCode
	// when the code is unknown/expired/used.
	RedeemPairingCode(ctx context.Context, rawCode string, p Profile) (IssuedToken, error)

	// RequestPairing registers a PENDING device for the hub's owner and returns
	// an inert token that activates when the owner approves. Used by a device
	// that cannot scan/enter a code. Fails if the hub has no owner yet (the
	// first device must use the code path).
	RequestPairing(ctx context.Context, p Profile) (IssuedToken, error)

	// Approve promotes a PENDING device to TRUSTED. ErrDeviceConflict if the
	// device is not pending.
	Approve(ctx context.Context, ownerID, id string) (Device, error)

	// Rename updates a device's human-facing name (validated non-empty).
	Rename(ctx context.Context, ownerID, id, name string) (Device, error)

	// Revoke severs a device immediately: flips it to REVOKED (local token
	// rejected at once) and best-effort revokes its authenticator session. The
	// local revocation is the guarantee; an unreachable authenticator is logged,
	// not fatal, because the hub's own access is already cut.
	Revoke(ctx context.Context, ownerID, id string) (Device, error)
}

// PairingNotifier is the optional realtime hook the service fires when a device
// joins via the fallback request path, so already-trusted devices get an
// approve/reject banner in near-real-time. Implemented at wiring by an adapter
// over the realtime hub; nil disables the push (the device still appears in the
// owner's list on next refresh).
type PairingNotifier interface {
	PairingRequested(ownerID string, device Device)
}

// Config configures NewService. Repo is required; the rest default.
type Config struct {
	Repo         Repository
	Clock        clock.Clock
	Secrets      Secrets
	Auth         auth.Validator
	Logger       *log.Logger
	PairingTTL   time.Duration
	PairNotifier PairingNotifier
}

type service struct {
	repo       Repository
	clock      clock.Clock
	secrets    Secrets
	auth       auth.Validator
	logger     *log.Logger
	pairingTTL time.Duration
	pairNotif  PairingNotifier
}

// NewService constructs the production Service, filling defaults for any
// optional dependency left nil.
func NewService(cfg Config) Service {
	s := &service{
		repo:       cfg.Repo,
		clock:      cfg.Clock,
		secrets:    cfg.Secrets,
		auth:       cfg.Auth,
		logger:     cfg.Logger,
		pairingTTL: cfg.PairingTTL,
		pairNotif:  cfg.PairNotifier,
	}
	if s.clock == nil {
		s.clock = clock.System{}
	}
	if s.secrets == nil {
		s.secrets = CryptoSecrets{}
	}
	if s.logger == nil {
		s.logger = log.Default()
	}
	if s.pairingTTL <= 0 {
		s.pairingTTL = defaultPairingTTL
	}
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) List(ctx context.Context, ownerID string) ([]Device, error) {
	return s.repo.ListDevices(ctx, ownerID)
}

func (s *service) Get(ctx context.Context, ownerID, id string) (Device, error) {
	return s.repo.GetDevice(ctx, ownerID, id)
}

func (s *service) IssuePairingCode(ctx context.Context, ownerID, deviceName string) (PairingCode, error) {
	if strings.TrimSpace(ownerID) == "" {
		return PairingCode{}, ErrInvalidDevice{Field: "owner", Reason: "required"}
	}
	raw, err := s.secrets.PairingCode()
	if err != nil {
		return PairingCode{}, fmt.Errorf("mint pairing code: %w", err)
	}
	now := s.clock.Now().UTC()
	pc := PairingCode{
		Code:       raw,
		CodeHash:   hashPairingCode(raw),
		OwnerID:    ownerID,
		DeviceName: strings.TrimSpace(deviceName),
		ExpiresAt:  now.Add(s.pairingTTL),
		CreatedAt:  now,
	}
	if err := s.repo.CreatePairingCode(ctx, pc); err != nil {
		return PairingCode{}, fmt.Errorf("persist pairing code: %w", err)
	}
	return pc, nil
}

func (s *service) RedeemPairingCode(ctx context.Context, rawCode string, p Profile) (IssuedToken, error) {
	if strings.TrimSpace(rawCode) == "" {
		return IssuedToken{}, ErrInvalidPairingCode{Reason: "pairing code is required"}
	}
	claimed, err := s.repo.ClaimPairingCode(ctx, hashPairingCode(rawCode), s.clock.Now().UTC())
	if err != nil {
		// ClaimPairingCode already returns ErrInvalidPairingCode for the
		// no-match case; propagate other (storage) errors verbatim.
		return IssuedToken{}, err
	}
	name := s.resolveName(p, claimed.DeviceName)
	return s.registerDevice(ctx, claimed.OwnerID, name, p, TrustTrusted)
}

func (s *service) RequestPairing(ctx context.Context, p Profile) (IssuedToken, error) {
	ownerID, err := s.repo.ResolveOwner(ctx)
	if err != nil {
		return IssuedToken{}, err
	}
	issued, err := s.registerDevice(ctx, ownerID, s.resolveName(p, ""), p, TrustPending)
	if err != nil {
		return IssuedToken{}, err
	}
	// Push the approve/reject banner to the owner's online devices. Best-effort:
	// a missing notifier or a dropped event never fails the join — the pending
	// device still surfaces in the device list on the next refresh.
	if s.pairNotif != nil {
		s.pairNotif.PairingRequested(ownerID, issued.Device)
	}
	return issued, nil
}

// registerDevice mints a token, persists the device in the given trust state,
// and returns the one-time raw token alongside the stored device.
func (s *service) registerDevice(ctx context.Context, ownerID, name string, p Profile, state TrustState) (IssuedToken, error) {
	token, err := s.secrets.DeviceToken()
	if err != nil {
		return IssuedToken{}, fmt.Errorf("mint device token: %w", err)
	}
	d := Device{
		OwnerID:      ownerID,
		Name:         name,
		Kind:         strings.TrimSpace(p.Kind),
		Platform:     strings.TrimSpace(p.Platform),
		Capabilities: cleanCapabilities(p.Capabilities),
		TrustState:   state,
	}
	stored, err := s.repo.CreateDevice(ctx, d, hashSecret(token))
	if err != nil {
		return IssuedToken{}, fmt.Errorf("persist device: %w", err)
	}
	return IssuedToken{Device: stored, Token: token}, nil
}

func (s *service) Approve(ctx context.Context, ownerID, id string) (Device, error) {
	d, err := s.repo.GetDevice(ctx, ownerID, id)
	if err != nil {
		return Device{}, err
	}
	switch d.TrustState {
	case TrustTrusted:
		return d, nil // idempotent: approving an already-trusted device is a no-op.
	case TrustRevoked:
		return Device{}, ErrDeviceConflict{Reason: "cannot approve a revoked device"}
	}
	return s.repo.SetTrust(ctx, ownerID, id, TrustTrusted)
}

func (s *service) Rename(ctx context.Context, ownerID, id, name string) (Device, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Device{}, ErrInvalidDevice{Field: "name", Reason: "required"}
	}
	return s.repo.Rename(ctx, ownerID, id, name)
}

func (s *service) Revoke(ctx context.Context, ownerID, id string) (Device, error) {
	d, err := s.repo.GetDevice(ctx, ownerID, id)
	if err != nil {
		return Device{}, err
	}
	revoked, err := s.repo.SetTrust(ctx, ownerID, id, TrustRevoked)
	if err != nil {
		return Device{}, err
	}
	// Best-effort authenticator session revocation. Local trust is already
	// REVOKED (token hash check now fails), so the hub's access is severed
	// regardless of the authenticator's reachability — fail-closed by design.
	if s.auth != nil && d.SessionID != "" {
		if err := s.auth.RevokeSession(ctx, d.SessionID); err != nil {
			s.logger.Printf("devices.Revoke: authenticator session %q not revoked (local revoke stands): %v", d.SessionID, err)
		}
	}
	return revoked, nil
}

// resolveName picks the device name: explicit profile name wins, then the name
// pre-assigned at code-issue time, then the generic default.
func (s *service) resolveName(p Profile, issuedName string) string {
	if n := strings.TrimSpace(p.Name); n != "" {
		return n
	}
	if n := strings.TrimSpace(issuedName); n != "" {
		return n
	}
	return defaultDeviceName
}

// cleanCapabilities trims, drops blanks, and preserves order without dups.
func cleanCapabilities(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, c := range in {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, dup := seen[c]; dup {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}
