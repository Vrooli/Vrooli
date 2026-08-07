// Package durability projects a bounded durability signal from evidence that
// is independent of the agent's own completion claims.
package durability

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	LaneVerified = "verified"
	LaneObserved = "observed"
	LaneUnlinked = "unlinked"
)

// Verdicts. Only VerdictDurable is a positive statement, and it is reserved
// for work we could actually observe: attributed to a run, with every evidence
// lane read successfully, and nothing adverse found. Everything we could not
// see is VerdictUnknown — absence of evidence is never evidence of durability.
const (
	VerdictDurable       = "durable"
	VerdictSignalPresent = "signal_present"
	VerdictUnknown       = "unknown"
)

// Reasons attached to VerdictUnknown so a consumer can tell why the work could
// not be graded, rather than guessing from an empty verdict.
const (
	ReasonUnattributedWork = "unattributed_work"
	ReasonEvidenceDegraded = "evidence_degraded"
)

type Boundary struct {
	Epoch  time.Time `json:"epoch"`
	Reason string    `json:"reason"`
}

// BoundaryStore owns the durable analysis epoch. Boundary returns the stored
// value, seeding it from seed only if none has been established yet.
//
// seam: durability.BoundaryStore
type BoundaryStore interface {
	Boundary(ctx context.Context, seed Boundary) (Boundary, error)
}

const defaultBoundaryReason = "durability evidence capture began with the verified attribution write seams"

// SeedBoundary is the value used the first time a deployment establishes its
// epoch. It is deliberately "now" rather than a compiled-in date: the honest
// epoch is when this deployment gained attribution capture, and after the first
// read the stored value is authoritative forever.
func SeedBoundary(now time.Time) Boundary {
	return Boundary{Epoch: now.UTC().Truncate(24 * time.Hour), Reason: defaultBoundaryReason}
}

// ResolveBoundary reads the stored epoch. A store that cannot be reached is
// reported, never defaulted around: silently grading against an invented epoch
// would put work in or out of scope without anyone deciding it.
func ResolveBoundary(ctx context.Context, store BoundaryStore, now time.Time) (Boundary, error) {
	if store == nil {
		return Boundary{}, errors.New("durability boundary store is not configured")
	}
	return store.Boundary(ctx, SeedBoundary(now))
}

type Work struct {
	ID        string    `json:"id"`
	Subject   []string  `json:"subject"`
	StartedAt time.Time `json:"startedAt"`
	Lane      string    `json:"lane"`
}

type Evidence struct {
	Kind      string    `json:"kind"`
	Reference string    `json:"reference"`
	At        time.Time `json:"at"`
	Lane      string    `json:"lane"`

	// Degraded marks an item that reports a failed or unavailable evidence
	// read rather than an observation about the work. A degraded item is never
	// counted as a finding — a broken lookup must not look like signal, and an
	// unread lane must not look like a clean one.
	Degraded bool `json:"degraded,omitempty"`
}

type Coverage struct {
	Verified int `json:"verified"`
	Observed int `json:"observed"`
	Unlinked int `json:"unlinked"`
}

type Verdict struct {
	WorkID        string     `json:"workId"`
	Subject       []string   `json:"subject"`
	Status        string     `json:"status,omitempty"`
	Verdict       string     `json:"verdict,omitempty"`
	SampleSize    int        `json:"sampleSize"`
	Boundary      Boundary   `json:"boundary"`
	Coverage      Coverage   `json:"coverage"`
	Evidence      []Evidence `json:"evidence,omitempty"`
	NoVerdict     bool       `json:"noVerdict,omitempty"`
	BoundaryState string     `json:"boundaryState,omitempty"`

	// Lane is the work's own attribution lane, distinct from the per-evidence
	// lanes counted in Coverage. Unattributed work cannot be graded at all.
	Lane string `json:"lane,omitempty"`
	// UnknownReason explains a VerdictUnknown result.
	UnknownReason string `json:"unknownReason,omitempty"`
	// Degradations lists evidence reads that failed, so a consumer can see
	// what was not checked instead of inferring a clean result.
	Degradations []Evidence `json:"degradations,omitempty"`
}

// Project deliberately accepts only Work and externally observed Evidence.
// It has no field for completion, outcome, goal state, or any other agent
// self-report. A signal-present verdict is intentionally descriptive; this
// phase collects evidence and does not tune a confidence threshold.
func Project(boundary Boundary, work Work, evidence []Evidence) Verdict {
	out := Verdict{WorkID: work.ID, Subject: work.Subject, Boundary: boundary, Lane: laneOrUnlinked(work.Lane)}
	for _, item := range evidence {
		if item.Degraded {
			out.Degradations = append(out.Degradations, item)
			continue
		}
		out.Evidence = append(out.Evidence, item)
	}
	sortEvidence(out.Evidence)
	sortEvidence(out.Degradations)
	for _, item := range out.Evidence {
		switch laneOrUnlinked(item.Lane) {
		case LaneVerified:
			out.Coverage.Verified++
		case LaneObserved:
			out.Coverage.Observed++
		default:
			out.Coverage.Unlinked++
		}
	}
	out.SampleSize = len(out.Evidence)
	if !boundary.Epoch.IsZero() && work.StartedAt.Before(boundary.Epoch) {
		out.NoVerdict = true
		out.BoundaryState = "before_analysis_epoch"
		return out
	}
	out.Status = "post_analysis_epoch"

	// Adverse evidence stands on its own: a finding is a finding even if some
	// other lane failed to read, and it is reportable for unattributed work
	// because the finding itself carries its reference.
	if len(out.Evidence) > 0 {
		out.Verdict = VerdictSignalPresent
		return out
	}
	// From here the evidence set is empty, which is only meaningful if we could
	// actually see the work and finish reading every lane.
	switch {
	case out.Lane == LaneUnlinked:
		out.Verdict = VerdictUnknown
		out.UnknownReason = ReasonUnattributedWork
	case len(out.Degradations) > 0:
		out.Verdict = VerdictUnknown
		out.UnknownReason = ReasonEvidenceDegraded
	default:
		out.Verdict = VerdictDurable
	}
	return out
}

func sortEvidence(items []Evidence) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].At.Equal(items[j].At) {
			return items[i].At.Before(items[j].At)
		}
		return items[i].Reference < items[j].Reference
	})
}

func laneOrUnlinked(lane string) string {
	switch normalized := strings.ToLower(strings.TrimSpace(lane)); normalized {
	case LaneVerified, LaneObserved:
		return normalized
	default:
		return LaneUnlinked
	}
}
