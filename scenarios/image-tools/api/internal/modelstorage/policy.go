// Package modelstorage owns Image Tools' model-weight storage lifecycle beyond
// a single install: on-demand acquisition for jobs that need weights that are
// not on disk, and adaptive offload of idle weights driven by free-disk
// headroom above the shared platform floor. The Installer remains the only
// component that downloads or deletes weights; this package decides when.
package modelstorage

import (
	"math"
	"sort"
	"time"

	"image-tools/internal/diskpolicy"
)

// Pressure modes, from most to least relaxed.
const (
	// ModeAmple: headroom is at or above Policy.AmpleBytes. Nothing idles out.
	ModeAmple = "ample"
	// ModeAdaptive: headroom is between the reserve and ample. The idle
	// threshold shrinks geometrically from MaxIdle toward MinIdle.
	ModeAdaptive = "adaptive"
	// ModeReserveDeficit: headroom is below the reserve (or below the reserve
	// plus a pending acquisition). The least valuable eligible models are
	// offloaded until the target is restored.
	ModeReserveDeficit = "reserve_deficit"
)

// Selection reasons recorded on candidates and receipts.
const (
	ReasonIdle      = "idle_beyond_threshold"
	ReasonReserve   = "reserve_deficit"
	ReasonRequested = "operator_requested"
)

// Protection reasons. A candidate with any protection is never offloaded.
const (
	ProtectInUse             = "in_use"
	ProtectActiveJob         = "held_by_active_job"
	ProtectKept              = "kept_by_operator"
	ProtectRecentlyInstalled = "installed_within_grace"
	ProtectNotReacquirable   = "not_auto_reacquirable"
	ProtectLocallyBuilt      = "locally_built_export"
	ProtectUseUnknown        = "use_unknown"
)

// Last-use bases. Install time is the floor for last use: a model installed
// (or re-installed) at T was wanted at T even when no job has used it since.
const (
	UseBasisSuccessfulJob = "successful_job"
	UseBasisInstall       = "install_receipt"
	UseBasisUnknown       = "unknown"
)

// Policy turns free-disk headroom into an idle threshold and a reserve target.
// All byte values are headroom above FloorBytes, except FloorBytes itself.
type Policy struct {
	// FloorBytes is the shared host disk floor every Image Tools disk admission
	// requests (internal/diskpolicy). Headroom is free minus floor.
	FloorBytes int64
	// ReserveBytes is the headroom the offloader restores when it falls short.
	ReserveBytes int64
	// AmpleBytes is the headroom at or above which no model idles out.
	AmpleBytes int64
	// MinIdle is the idle threshold at (or below) the reserve.
	MinIdle time.Duration
	// MaxIdle is the idle threshold just below ample headroom.
	MaxIdle time.Duration
	// RecentInstall protects freshly installed weights from any offload.
	RecentInstall time.Duration
	// ThroughputBytesPerSecond estimates re-acquisition time.
	ThroughputBytesPerSecond int64
	// ReacquireOverhead is the fixed per-acquisition cost (connection setup,
	// verification, runtime readiness) added to the transfer time.
	ReacquireOverhead time.Duration
}

// DefaultPolicy is the shipped policy. See docs/internal/DECISIONS.md
// ("Model weights: auto-acquire and adaptive offload") for the rationale.
func DefaultPolicy() Policy {
	return Policy{
		FloorBytes:               diskpolicy.HostMinimumFloorBytes,
		ReserveBytes:             24 << 30,
		AmpleBytes:               96 << 30,
		MinIdle:                  24 * time.Hour,
		MaxIdle:                  30 * 24 * time.Hour,
		RecentInstall:            time.Hour,
		ThroughputBytesPerSecond: 15 << 20,
		ReacquireOverhead:        2 * time.Minute,
	}
}

// Pressure is the policy's reading of one free-space observation.
type Pressure struct {
	FreeBytes     int64  `json:"free_bytes"`
	HeadroomBytes int64  `json:"headroom_bytes"`
	Mode          string `json:"mode"`
	// IdleThreshold is zero when Unlimited (ample headroom keeps everything).
	IdleThreshold time.Duration `json:"idle_threshold"`
	Unlimited     bool          `json:"unlimited"`
}

// Evaluate maps free bytes to a pressure mode and idle threshold. The
// threshold interpolates geometrically, so each step of lost headroom divides
// the tolerated idle time by the same factor: with the defaults, half-way
// between reserve and ample tolerates about 5.5 days of idleness.
func (p Policy) Evaluate(freeBytes int64) Pressure {
	headroom := freeBytes - p.FloorBytes
	out := Pressure{FreeBytes: freeBytes, HeadroomBytes: headroom}
	switch {
	case headroom >= p.AmpleBytes:
		out.Mode, out.Unlimited = ModeAmple, true
	case headroom < p.ReserveBytes:
		out.Mode, out.IdleThreshold = ModeReserveDeficit, p.MinIdle
	default:
		out.Mode = ModeAdaptive
		span := float64(p.AmpleBytes - p.ReserveBytes)
		fraction := 0.0
		if span > 0 {
			fraction = float64(headroom-p.ReserveBytes) / span
		}
		ratio := float64(p.MaxIdle) / float64(p.MinIdle)
		if p.MinIdle <= 0 || ratio <= 1 {
			out.IdleThreshold = p.MinIdle
		} else {
			out.IdleThreshold = time.Duration(float64(p.MinIdle) * math.Pow(ratio, fraction))
		}
	}
	return out
}

// ReacquireSeconds estimates how long getting bytes back would take.
func (p Policy) ReacquireSeconds(bytes int64) float64 {
	throughput := p.ThroughputBytesPerSecond
	if throughput <= 0 {
		throughput = 15 << 20
	}
	return float64(bytes)/float64(throughput) + p.ReacquireOverhead.Seconds()
}

// Priority ranks offload value: bytes freed per second of re-acquisition,
// scaled by idle hours. Transfer time grows with size, so the fixed overhead
// is what makes freeing one large idle model better than many small ones,
// and idleness dominates between models of similar size.
func (p Policy) Priority(bytes int64, idle time.Duration) float64 {
	if bytes <= 0 {
		return 0
	}
	hours := math.Max(idle.Hours(), 0) + 1
	return hours * float64(bytes) / p.ReacquireSeconds(bytes)
}

// Candidate is one installed, weight-backed model as the policy sees it.
type Candidate struct {
	ModelID           string        `json:"model_id"`
	Bytes             int64         `json:"bytes"`
	BytesBasis        string        `json:"bytes_basis"`
	InstalledAt       time.Time     `json:"installed_at,omitempty"`
	LastSuccessfulUse time.Time     `json:"last_successful_use,omitempty"`
	LastUse           time.Time     `json:"last_use,omitempty"`
	LastUseBasis      string        `json:"last_use_basis"`
	Idle              time.Duration `json:"idle"`
	ReacquireSeconds  float64       `json:"reacquire_seconds"`
	Priority          float64       `json:"priority"`
	Protected         []string      `json:"protected,omitempty"`
	Selected          bool          `json:"selected"`
	SelectionReason   string        `json:"selection_reason,omitempty"`
	Checksum          string        `json:"checksum,omitempty"`
}

// Eligible reports whether no protection applies.
func (c Candidate) Eligible() bool { return len(c.Protected) == 0 }

// Selection is the outcome of applying the policy to candidates.
type Selection struct {
	Pressure Pressure `json:"pressure"`
	// NeedBytes is extra headroom a pending acquisition requires.
	NeedBytes int64 `json:"need_bytes"`
	// TargetHeadroomBytes is the reserve plus NeedBytes.
	TargetHeadroomBytes    int64       `json:"target_headroom_bytes"`
	SelectedBytes          int64       `json:"selected_bytes"`
	ProjectedHeadroomBytes int64       `json:"projected_headroom_bytes"`
	ShortfallBytes         int64       `json:"shortfall_bytes"`
	Candidates             []Candidate `json:"candidates"`
}

// Select ranks candidates and marks which to offload. requested, when
// non-empty, restricts selection to exactly those eligible models (an
// operator's targeted offload), bypassing the idle threshold but never a
// protection. Otherwise idle models beyond the threshold are selected first,
// then the highest-priority eligible models until the target is reached.
func (p Policy) Select(candidates []Candidate, pressure Pressure, needBytes int64, requested []string) Selection {
	ranked := append([]Candidate(nil), candidates...)
	for i := range ranked {
		ranked[i].Selected, ranked[i].SelectionReason = false, ""
		ranked[i].ReacquireSeconds = p.ReacquireSeconds(ranked[i].Bytes)
		ranked[i].Priority = p.Priority(ranked[i].Bytes, ranked[i].Idle)
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Eligible() != ranked[j].Eligible() {
			return ranked[i].Eligible()
		}
		if ranked[i].Priority != ranked[j].Priority {
			return ranked[i].Priority > ranked[j].Priority
		}
		if ranked[i].Bytes != ranked[j].Bytes {
			return ranked[i].Bytes > ranked[j].Bytes
		}
		return ranked[i].ModelID < ranked[j].ModelID
	})
	if needBytes < 0 {
		needBytes = 0
	}
	out := Selection{Pressure: pressure, NeedBytes: needBytes, TargetHeadroomBytes: p.ReserveBytes + needBytes, ProjectedHeadroomBytes: pressure.HeadroomBytes}
	if out.ProjectedHeadroomBytes < out.TargetHeadroomBytes && out.Pressure.Mode != ModeReserveDeficit {
		// A pending acquisition can push ample or adaptive headroom below the
		// reserve; the pass then runs at the reserve's (shortest) threshold.
		out.Pressure.Mode, out.Pressure.Unlimited, out.Pressure.IdleThreshold = ModeReserveDeficit, false, p.MinIdle
	}
	pick := func(i int, reason string) {
		ranked[i].Selected, ranked[i].SelectionReason = true, reason
		out.SelectedBytes += ranked[i].Bytes
		out.ProjectedHeadroomBytes += ranked[i].Bytes
	}
	if len(requested) > 0 {
		want := make(map[string]bool, len(requested))
		for _, id := range requested {
			want[id] = true
		}
		for i := range ranked {
			if want[ranked[i].ModelID] && ranked[i].Eligible() {
				pick(i, ReasonRequested)
			}
		}
	} else {
		if !out.Pressure.Unlimited {
			for i := range ranked {
				if ranked[i].Eligible() && ranked[i].Idle > out.Pressure.IdleThreshold {
					pick(i, ReasonIdle)
				}
			}
		}
		for i := range ranked {
			if out.ProjectedHeadroomBytes >= out.TargetHeadroomBytes {
				break
			}
			if ranked[i].Eligible() && !ranked[i].Selected {
				pick(i, ReasonReserve)
			}
		}
	}
	if out.ProjectedHeadroomBytes < out.TargetHeadroomBytes {
		out.ShortfallBytes = out.TargetHeadroomBytes - out.ProjectedHeadroomBytes
	}
	out.Candidates = ranked
	return out
}
