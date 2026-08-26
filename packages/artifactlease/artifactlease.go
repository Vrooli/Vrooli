// Package artifactlease exposes the control plane's artifact ownership records
// to scenario consumers.
//
// storage-manager's reaper is the reason this door exists: it decides whether
// an installed CLI is an orphan, and that decision now depends on how long the
// owner has been missing rather than on a liveness check that cannot apply to a
// short-lived binary. It lives in its own module, so it cannot import
// internal/artifactlease directly -- the same constraint that produced
// packages/artifactledger and packages/hostreq.
//
// The types are aliases. There is no boundary to defend here: a scenario asking
// whether an artifact may be reclaimed means exactly what the control plane
// means by it, and two declarations would be two things to keep in step.
package artifactlease

import (
	"time"

	lease "github.com/vrooli/vrooli/internal/artifactlease"
)

// Suffix is appended to an artifact path to locate its lease.
const Suffix = lease.Suffix

// DefaultGrace is how long an owner must have been continuously missing before
// its artifact may be reclaimed.
const DefaultGrace = lease.DefaultGrace

// MinObservations is how many independent sightings of an absence are required
// before it may be acted on.
const MinObservations = lease.MinObservations

type (
	// Lease is the ownership record stored beside an installed artifact.
	Lease = lease.Lease
	// Owner identifies who installed an artifact.
	Owner = lease.Owner
	// Eligibility describes whether an artifact may be reclaimed.
	Eligibility = lease.Eligibility
)

// Path returns the lease path for an artifact.
func Path(artifact string) string { return lease.Path(artifact) }

// Load reads an artifact's lease. The second result reports whether one exists.
func Load(artifact string) (Lease, bool, error) { return lease.Load(artifact) }

// Save writes a lease atomically.
func Save(l Lease) error { return lease.Save(l) }

// NoteOwnerMissing records that the owner was observed absent. It never removes
// anything, which is the entire point of separating observation from
// reclamation.
func NoteOwnerMissing(artifact string, now time.Time) (Lease, error) {
	return lease.NoteOwnerMissing(artifact, now)
}

// NoteOwnerPresent clears a recorded absence.
func NoteOwnerPresent(artifact string) error { return lease.NoteOwnerPresent(artifact) }

// EvaluateReclaim reports whether an artifact may be reclaimed now, and names
// the reason when it may not.
func EvaluateReclaim(l Lease, found bool, now time.Time, grace time.Duration) Eligibility {
	return lease.EvaluateReclaim(l, found, now, grace)
}

// Remove deletes an artifact's lease.
func Remove(artifact string) error { return lease.Remove(artifact) }
