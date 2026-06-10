package findings

import (
	"context"
	"time"

	"web-search/internal/clock"
)

// GC defaults. All are conservative — the GC only ever SUPERSEDES (soft-retires)
// findings that are simultaneously old, never-surfaced, and decayed below the
// confidence floor; everything else is reported, never mutated.
const (
	// DefaultGCDecayMinAge is how old a never-surfaced active finding must be
	// before it is a supersede candidate (two confidence half-lives — genuinely
	// stale, not merely unused for a while).
	DefaultGCDecayMinAge = 2 * DecayHalfLife
	// DefaultGCConfidenceFloor is the effective-confidence ceiling a decay
	// candidate must be BELOW to be retired. A still-trusted (slowly-decaying,
	// high-confidence) claim is kept even if never surfaced.
	DefaultGCConfidenceFloor = 0.25
	// DefaultGCColdArchiveTTL is how long a superseded finding lingers before the
	// GC reports it as a cold-archive candidate (report only — the GC never
	// hard-deletes; operator-invoked `findings prune` is the only delete path).
	DefaultGCColdArchiveTTL = 90 * 24 * time.Hour
	// DefaultGCDisputeExpiryTTL is how long a dispute may stay unresolved before
	// the GC flags it as stale (for a human — the GC never auto-picks a side).
	DefaultGCDisputeExpiryTTL = 90 * 24 * time.Hour
	// DefaultGCMaxSupersede caps how many decay candidates one run retires, so a
	// first run over a large backlog cannot retire the world in one sweep.
	DefaultGCMaxSupersede = 200
)

// GCConfig tunes the periodic store-consistency GC.
type GCConfig struct {
	DecayMinAge      time.Duration
	ConfidenceFloor  float64
	ColdArchiveTTL   time.Duration
	DisputeExpiryTTL time.Duration
	MaxSupersede     int
}

func (c GCConfig) withDefaults() GCConfig {
	if c.DecayMinAge <= 0 {
		c.DecayMinAge = DefaultGCDecayMinAge
	}
	if c.ConfidenceFloor <= 0 {
		c.ConfidenceFloor = DefaultGCConfidenceFloor
	}
	if c.ColdArchiveTTL <= 0 {
		c.ColdArchiveTTL = DefaultGCColdArchiveTTL
	}
	if c.DisputeExpiryTTL <= 0 {
		c.DisputeExpiryTTL = DefaultGCDisputeExpiryTTL
	}
	if c.MaxSupersede <= 0 {
		c.MaxSupersede = DefaultGCMaxSupersede
	}
	return c
}

// GCReport is one GC run's outcome. SupersededDecayed lists the ids retired (or,
// in a dry run, that WOULD be retired). The other three are report-only signals
// the GC never mutates: cold-archive candidates (superseded past TTL — never
// hard-deleted), stale disputes (past TTL — flagged for a human, never
// auto-resolved), and orphans (brief_id referencing a missing brief).
type GCReport struct {
	DryRun                bool
	SupersededDecayed     []string
	ColdArchiveCandidates []string
	StaleDisputes         []string
	Orphans               []string
}

// GCService runs the periodic full-store consistency pass (OT-P2-003). It is
// separate from the per-query reconcile: it sweeps the whole store on a cadence,
// soft-retiring stale unused knowledge and surfacing inconsistencies, while
// honoring the supersede-only (never hard-delete) and never-auto-resolve-disputes
// invariants.
type GCService struct {
	svc   Service
	clock clock.Clock
	cfg   GCConfig
}

// NewGCService constructs the GC over the findings application service.
func NewGCService(svc Service, clk clock.Clock, cfg GCConfig) *GCService {
	return &GCService{svc: svc, clock: clk, cfg: cfg.withDefaults()}
}

// Run executes one consistency pass. With dryRun set it computes and reports the
// candidates without mutating anything. The mutating path supersedes only the
// confidence-gated decay candidates; cold-archive / stale-dispute / orphan
// findings are always report-only.
func (g *GCService) Run(ctx context.Context, dryRun bool) (GCReport, error) {
	now := g.clock.Now().UTC()
	report := GCReport{DryRun: dryRun}

	// 1. Supersede decay candidates (the only mutating action), confidence-gated.
	candidates, err := g.svc.ListDecayCandidates(ctx, g.cfg.DecayMinAge, g.cfg.MaxSupersede)
	if err != nil {
		return report, err
	}
	for _, f := range candidates {
		// Never-surfaced ⇒ usage factor is already at its floor; gate on the
		// age-decayed confidence so a still-trusted claim is kept.
		if EffectiveConfidence(f, now) >= g.cfg.ConfidenceFloor {
			continue
		}
		if !dryRun {
			if _, err := g.svc.Supersede(ctx, f.ID, "", "GC: never-surfaced and decayed below the confidence floor"); err != nil {
				return report, err
			}
		}
		report.SupersededDecayed = append(report.SupersededDecayed, f.ID)
	}

	// 2. Cold-archive candidates: superseded findings past the TTL (report only).
	superseded, err := g.svc.List(ctx, ListFilter{Status: StatusSuperseded, IncludeArchived: true, Limit: 1000})
	if err != nil {
		return report, err
	}
	coldCutoff := now.Add(-g.cfg.ColdArchiveTTL)
	for _, f := range superseded {
		if f.UpdatedAt.Before(coldCutoff) {
			report.ColdArchiveCandidates = append(report.ColdArchiveCandidates, f.ID)
		}
	}

	// 3. Stale disputes: disputed past the TTL (flagged for a human, never resolved).
	disputed, err := g.svc.List(ctx, ListFilter{Status: StatusDisputed, Limit: 1000})
	if err != nil {
		return report, err
	}
	disputeCutoff := now.Add(-g.cfg.DisputeExpiryTTL)
	for _, f := range disputed {
		if f.UpdatedAt.Before(disputeCutoff) {
			report.StaleDisputes = append(report.StaleDisputes, f.ID)
		}
	}

	// 4. Orphans: findings whose brief_id references a missing brief (report only).
	orphans, err := g.svc.ListOrphanedFindings(ctx)
	if err != nil {
		return report, err
	}
	for _, f := range orphans {
		report.Orphans = append(report.Orphans, f.ID)
	}

	return report, nil
}
