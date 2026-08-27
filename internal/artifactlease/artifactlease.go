// Package artifactlease records who owns an installed artifact and how long an
// absence has lasted.
//
// # The guard it replaces
//
// Reclamation used to be gated on "is this binary running?". A CLI is invoked
// and exits, so it is essentially never running when sampled -- the guard is
// structurally incapable of protecting the artifact class it guards. Worse, the
// age check measured the binary's own mtime, so a freshly rebuilt CLI was
// protected while a stable one was not, which is backwards.
//
// # Absence is aged, not the artifact
//
// The central move is splitting "the owner looks missing" from "reclaim it".
// Vrooli runs several agents in one environment by design, so a scenario
// directory can be absent for a moment while another agent rebuilds or
// regenerates it. Treating that instant as authority to delete is what let a
// freshly built CLI be reclaimed on a stale observation.
//
// So an observed absence stamps a timestamp and counts an observation. Nothing
// is deleted. Reclamation additionally requires that the absence has persisted
// for the grace window and has been seen more than once, and a successful
// install clears the mark entirely.
//
// # Why there is no liveness check here
//
// An earlier draft carried the owner's pid and process start time so a
// reclaimer could ask whether the owner was still alive. That turned out to be
// the least portable part of the design and unnecessary: protection here is
// time-based, not process-based. A lease says "this artifact was installed by
// someone recently, leave it alone until it expires", which holds whether or
// not the installing process is still running -- and a CLI install exits within
// seconds, so process liveness would have expired far sooner than the
// protection is meant to last.
package artifactlease

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/api-core/storage"
)

const (
	artifactleaseParameterA = 2
)

// Suffix is appended to an artifact path to locate its lease.
const Suffix = ".lease.json"

// Schema identifies the lease format.
const Schema = "vrooli.artifact-lease/1"

// DefaultGrace is how long an owner must have been continuously missing before
// its artifact may be reclaimed.
//
// Twenty-four hours was chosen deliberately: it absorbs any plausible rebuild,
// test run, or scenario regeneration under concurrent agents, and the cost is
// only disk. The backlog that motivated this work was roughly 2GB against 436GB
// free, so the conservative direction is close to free.
const DefaultGrace = tuning.DailyRetentionWindow

// MinObservations is how many independent sightings of an absence are required
// before it may be acted on. One observation is a sample; two separated by the
// grace window is a trend.
const MinObservations = 2

// Owner identifies who installed an artifact.
//
// Agent is empty when agent-manager was unavailable, which is expected rather
// than exceptional: an ownership record that only exists when a particular
// scenario is up would be missing exactly when the control plane is repairing
// itself. Node is always present.
type Owner struct {
	Node  string `json:"node"`
	User  string `json:"user,omitempty"`
	Agent string `json:"agent,omitempty"`
}

// Lease is the ownership record stored beside an installed artifact.
type Lease struct {
	Schema   string `json:"schema"`
	Artifact string `json:"artifact"`
	// Generation increments on every replace. A reclamation decided against one
	// generation must not be applied to another.
	Generation int64  `json:"generation"`
	Owner      Owner  `json:"owner"`
	AcquiredAt string `json:"acquired_at"`
	ExpiresAt  string `json:"expires_at"`
	// OwnerModule is the path whose absence makes this artifact an orphan.
	OwnerModule string `json:"owner_module,omitempty"`
	// OwnerMissingSince is when the owner was first observed absent, and
	// Observations counts how many times that has been seen since. Both reset
	// the moment the owner is seen again.
	OwnerMissingSince string `json:"owner_missing_since,omitempty"`
	Observations      int    `json:"observations,omitempty"`
}

// Path returns the lease path for an artifact.
func Path(artifact string) string {
	return filepath.Clean(artifact) + Suffix
}

// Load reads an artifact's lease. The second result reports whether one exists;
// an artifact installed before this protocol has none, and that is not an error.
func Load(artifact string) (Lease, bool, error) {
	data, err := os.ReadFile(Path(artifact))
	if err != nil {
		if os.IsNotExist(err) {
			return Lease{}, false, nil
		}
		return Lease{}, false, fmt.Errorf("read lease for %s: %w", filepath.Base(artifact), err)
	}
	var lease Lease
	if err := json.Unmarshal(data, &lease); err != nil {
		// A lease that cannot be parsed is not permission to reclaim. Callers
		// treat the error as "ownership unknown", which refuses.
		return Lease{}, false, fmt.Errorf("decode lease for %s: %w", filepath.Base(artifact), err)
	}
	return lease, true, nil
}

// Save writes a lease atomically.
func Save(lease Lease) error {
	if strings.TrimSpace(lease.Artifact) == "" {
		return errors.New("artifactlease: lease needs an artifact path")
	}
	lease.Schema = Schema
	data, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		return fmt.Errorf("encode lease for %s: %w", filepath.Base(lease.Artifact), err)
	}
	return storage.WriteFileAtomic(Path(lease.Artifact), append(data, '\n'), tuning.PermSecret)
}

// Claim records a fresh installation: new bytes were written to this path.
//
// It bumps the generation and clears any recorded absence, because an artifact
// that was just installed is by definition not an orphan. This is what makes a
// scenario that is deleted and recreated inside the grace window keep its CLI.
//
// Call this only when the artifact actually changed. A freshness check that
// finds the installed binary already current is a Renew, not a Claim -- see
// there for why the distinction matters.
func Claim(artifact string, owner Owner, ownerModule string, ttl time.Duration, now time.Time) (Lease, error) {
	existing, _, err := Load(artifact)
	if err != nil {
		// A corrupt lease must not block an install; the fresh claim replaces
		// it. Refusing here would let a bad file make an artifact permanently
		// uninstallable.
		existing = Lease{}
	}
	lease := Lease{
		Schema:      Schema,
		Artifact:    filepath.Clean(artifact),
		Generation:  existing.Generation + 1,
		Owner:       owner,
		AcquiredAt:  now.UTC().Format(time.RFC3339Nano),
		ExpiresAt:   now.UTC().Add(ttl).Format(time.RFC3339Nano),
		OwnerModule: strings.TrimSpace(ownerModule),
	}
	if err := Save(lease); err != nil {
		return Lease{}, err
	}
	return lease, nil
}

// NoteOwnerMissing records that the owner was observed absent. It never
// removes anything.
//
// The first sighting stamps the time; later sightings only increment the count.
// Keeping the original timestamp is the point: the grace window measures how
// long the absence has lasted, not how recently it was last noticed.
func NoteOwnerMissing(artifact string, now time.Time) (Lease, error) {
	lease, found, err := Load(artifact)
	if err != nil {
		return Lease{}, err
	}
	if !found {
		// An artifact with no lease predates the protocol. Recording the
		// absence gives it one, which starts its grace clock now rather than
		// treating "no lease" as "no owner, reclaim freely".
		lease = Lease{Schema: Schema, Artifact: filepath.Clean(artifact)}
	}
	if strings.TrimSpace(lease.OwnerMissingSince) == "" {
		lease.OwnerMissingSince = now.UTC().Format(time.RFC3339Nano)
	}
	lease.Observations++
	if err := Save(lease); err != nil {
		return Lease{}, err
	}
	return lease, nil
}

// NoteOwnerPresent clears a recorded absence. It is a no-op when none is
// recorded, so callers may call it unconditionally on every sighting.
func NoteOwnerPresent(artifact string) error {
	lease, found, err := Load(artifact)
	if err != nil || !found {
		return err
	}
	if lease.OwnerMissingSince == "" && lease.Observations == 0 {
		return nil
	}
	lease.OwnerMissingSince = ""
	lease.Observations = 0
	return Save(lease)
}

// Eligibility describes whether an artifact may be reclaimed.
type Eligibility struct {
	Reclaimable bool
	// Reason explains a refusal in the operator's terms. It is empty when
	// Reclaimable is true.
	Reason string
	// Generation is the generation the decision was made against.
	Generation int64
}

// EvaluateReclaim reports whether an artifact may be reclaimed now.
//
// Every refusal names its reason, because "not reclaimable" with no explanation
// is how a reaper's behaviour becomes folklore.
func EvaluateReclaim(lease Lease, found bool, now time.Time, grace time.Duration) Eligibility {
	if grace <= 0 {
		grace = DefaultGrace
	}
	if !found {
		return Eligibility{Reason: "no lease recorded; the absence has not been observed even once"}
	}
	if expiry := parseTime(lease.ExpiresAt); !expiry.IsZero() && now.Before(expiry) {
		return Eligibility{
			Generation: lease.Generation,
			Reason:     "lease is held until " + lease.ExpiresAt,
		}
	}
	missingSince := parseTime(lease.OwnerMissingSince)
	if missingSince.IsZero() {
		return Eligibility{Generation: lease.Generation, Reason: "the owner has never been observed missing"}
	}
	if lease.Observations < MinObservations {
		return Eligibility{
			Generation: lease.Generation,
			Reason:     fmt.Sprintf("the absence has been observed %d time(s); %d are required", lease.Observations, MinObservations),
		}
	}
	// A clock that moved backwards yields a negative interval. Treating that as
	// "long elapsed" would reclaim everything the moment a host's time was
	// corrected, so it reads as not yet elapsed instead.
	elapsed := now.Sub(missingSince)
	if elapsed < grace {
		return Eligibility{
			Generation: lease.Generation,
			Reason:     fmt.Sprintf("the owner has been missing for %s; %s are required", elapsed.Round(time.Second), grace),
		}
	}
	return Eligibility{Reclaimable: true, Generation: lease.Generation}
}

// Remove deletes an artifact's lease. It is called when the artifact itself is
// removed, so a later install starts from a clean record.
func Remove(artifact string) error {
	if err := os.Remove(Path(artifact)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove lease for %s: %w", filepath.Base(artifact), err)
	}
	return nil
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

// Renew extends an existing lease without claiming a new installation.
//
// The distinction from Claim is not cosmetic. A freshness check that finds the
// binary already current runs on every invocation, and treating that as an
// install made the generation count checks rather than replacements -- it
// climbed from 3 to 8 in under an hour on an artifact nothing had rebuilt. A
// generation that counts observations cannot be used to detect a replacement,
// which is the one thing it exists for.
//
// Renew also declines to write when nothing meaningful would change. Rewriting
// a lease every few seconds is pointless disk churn, and it drowns real
// artifact activity in the very filesystem traces used to diagnose this system.
func Renew(artifact string, owner Owner, ownerModule string, ttl time.Duration, now time.Time) (Lease, error) {
	lease, found, err := Load(artifact)
	if err != nil || !found {
		// No usable record: a first sighting is a claim.
		return Claim(artifact, owner, ownerModule, ttl, now)
	}

	expiry := parseTime(lease.ExpiresAt)
	stillFresh := !expiry.IsZero() && now.Add(ttl/artifactleaseParameterA).Before(expiry)
	if stillFresh && lease.OwnerMissingSince == "" && lease.Observations == 0 {
		return lease, nil
	}

	lease.Owner = owner
	if strings.TrimSpace(ownerModule) != "" {
		lease.OwnerModule = strings.TrimSpace(ownerModule)
	}
	lease.ExpiresAt = now.UTC().Add(ttl).Format(time.RFC3339Nano)
	lease.OwnerMissingSince = ""
	lease.Observations = 0
	if err := Save(lease); err != nil {
		return Lease{}, err
	}
	return lease, nil
}
