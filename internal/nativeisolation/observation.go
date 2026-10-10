package nativeisolation

import (
	"context"
	"github.com/vrooli/vrooli/internal/nativeisolation/probeprotocol"
	"path/filepath"
	"time"
)

// FirstObservationContract is immutable owner-supplied nonsecret canary data.
// CanaryRoot must be dedicated to disposable marker material, never identity,
// custody, policy or ledger files. This contract grants no marker preparation.
type FirstObservationContract struct {
	CanaryRoot      string
	CanaryID        string
	WorkspaceSHA256 string
	DeniedSHA256    string
}

func (c FirstObservationContract) deniedSource() string {
	return filepath.Join(c.CanaryRoot, c.CanaryID, "denied.marker")
}
func (s ProbeSpec) observation(now time.Time) (probeprotocol.Observation, error) {
	if s.FirstObservation == nil || s.Purpose != FirstQualification || len(s.Manifest.NetworkAllow) != 0 {
		return probeprotocol.Observation{}, ErrRefused
	}
	c := s.FirstObservation
	if !cleanPath(c.CanaryRoot) || c.CanaryRoot == "/" || filepath.Base(c.CanaryRoot) != "vrooli-isolation-canary" || overlap(c.CanaryRoot, s.Manifest.RootFS) || overlap(c.CanaryRoot, s.Manifest.WorkspaceSource) || overlap(c.CanaryRoot, s.repository()) {
		return probeprotocol.Observation{}, ErrRefused
	}
	for _, p := range s.Manifest.Isolation.ProtectedPaths {
		if overlap(c.CanaryRoot, p) {
			return probeprotocol.Observation{}, ErrRefused
		}
	}
	o := probeprotocol.Observation{Repository: s.repository(), CanaryID: c.CanaryID, WorkspaceSHA256: c.WorkspaceSHA256, DeniedSHA256: c.DeniedSHA256, Deadline: s.Deadline}
	if o.Validate(now) != nil {
		return probeprotocol.Observation{}, ErrRefused
	}
	return o, nil
}

// CheckObservationInstallation is an optional readonly startup check, never a
// marker creator or permission fixer. Unavailable/denied evidence refuses.
func (s ProbeSpec) CheckObservationInstallation(ctx context.Context) error {
	if ctx.Err() != nil || s.validate(ctx, time.Now().UTC()) != nil {
		return ErrRefused
	}
	return checkObservationMarkers(s)
}
