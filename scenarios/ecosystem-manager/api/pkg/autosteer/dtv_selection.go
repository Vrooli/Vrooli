package autosteer

import (
	"github.com/ecosystem-manager/api/pkg/dtv"
	"github.com/vrooli/maturity-go/dimensions"
)

// FitnessSnapshot is a per-task, point-in-time view of DTV skill fitness keyed
// by skill id. It is captured once at controller init (and TTL-refreshed) so the
// SELECT hot path never calls DTV synchronously. A skill absent from the map (or
// a nil/empty snapshot) resolves to UNKNOWN — the fail-open default: allow-all,
// uniform prior, i.e. exact P1 behavior.
type FitnessSnapshot struct {
	fits map[string]dtv.Fitness
}

// NewFitnessSnapshot wraps a fitness map (nil is valid and means "no DTV data").
func NewFitnessSnapshot(fits map[string]dtv.Fitness) FitnessSnapshot {
	return FitnessSnapshot{fits: fits}
}

// Get returns the fitness for a skill, or a zero-value (UNKNOWN) fitness when
// absent — never nil-panics, so callers fail open.
func (s FitnessSnapshot) Get(skillID string) dtv.Fitness {
	if s.fits != nil {
		if f, ok := s.fits[skillID]; ok {
			return f
		}
	}
	return dtv.Fitness{SkillID: skillID}
}

// Trust returns a skill's DTV trust (pass-rate, 0–1). It is the ordering key the
// selector uses to pick the "least-bad" skill when the Layer-1 gate degrades and
// every candidate is RED (EM-P2 proceed-cap-flag policy). Absent/UNKNOWN ⇒ 0.
func (s FitnessSnapshot) Trust(skillID string) float64 {
	return s.Get(skillID).PassRate
}

var _ TrustRanker = FitnessSnapshot{}

// DTVEligibilityFilter is the Layer-1 hard gate (EM-P2-002). It denies a skill
// iff DTV currently judges it RED (intrinsic non-convergence / crash — Flavor-1
// thrashing). UNKNOWN / GREEN / YELLOW are allowed; UNKNOWN fails open (never
// gate on missing data — that would break first-run scenarios).
type DTVEligibilityFilter struct {
	snapshot FitnessSnapshot
}

var _ EligibilityFilter = DTVEligibilityFilter{}

// NewDTVEligibilityFilter builds the gate over a fitness snapshot.
func NewDTVEligibilityFilter(s FitnessSnapshot) DTVEligibilityFilter {
	return DTVEligibilityFilter{snapshot: s}
}

// Allow implements EligibilityFilter: deny iff RED.
func (f DTVEligibilityFilter) Allow(skillID string, _ dimensions.Dimension) bool {
	return f.snapshot.Get(skillID).Verdict != dtv.VerdictRed
}

// DTVPriorConfig parameterizes the DTV trust/cost prior mapping. Defaults
// (defaultDTVPriorConfig) preserve "seed priors from DTV, wash out with live
// evidence" while never over-trusting thin or stale data.
type DTVPriorConfig struct {
	// Weight scales the whole prior (profile dtv.prior_weight). <=0 disables it
	// (prior collapses to 0 = uniform P1).
	Weight float64
	// Base is the neutral expected-efficacy-per-ktok seed a perfectly-trusted,
	// fully-convergent skill is assumed to start at.
	Base float64
	// MinRuns guards against thin evidence: below this run count the prior is 0
	// (we do not trust a verdict backed by one or two runs).
	MinRuns int64
	// ConvK is the convergence-confidence constant: confidence = min(1, ConvK/u)
	// for u>1 unique diffs (u==1 ⇒ 1.0).
	ConvK float64
	// StaleFactor multiplies convergence confidence when DTV's validation is
	// stale (reduced confidence, not a failure). 0..1.
	StaleFactor float64
	// TrustFloor is a pass-rate floor (0–1): a skill whose pass_rate is below it
	// gets no prior (0). 0 ⇒ no floor. Carried verbatim from the profile (not
	// defaulted).
	TrustFloor float64
}

// defaultDTVPriorConfig is the conservative default applied when a profile omits
// a knob.
func defaultDTVPriorConfig() DTVPriorConfig {
	return DTVPriorConfig{
		Weight:      1.0,
		Base:        1.0,
		MinRuns:     2,
		ConvK:       1.0,
		StaleFactor: 0.5,
	}
}

// withDefaults fills any zero/invalid knob from the package defaults so a
// partially-specified profile block still yields a sane prior.
func (c DTVPriorConfig) withDefaults() DTVPriorConfig {
	d := defaultDTVPriorConfig()
	if c.Weight != 0 { // 0 is a legitimate "disable"; negative is treated as disable too
		d.Weight = c.Weight
	}
	if c.Base > 0 {
		d.Base = c.Base
	}
	if c.MinRuns > 0 {
		d.MinRuns = c.MinRuns
	}
	if c.ConvK > 0 {
		d.ConvK = c.ConvK
	}
	if c.StaleFactor > 0 {
		d.StaleFactor = c.StaleFactor
	}
	d.TrustFloor = c.TrustFloor // floor is opt-in; never defaulted
	return d
}

// DTVPriorProvider maps a skill's DTV fitness onto the cold-start prior
// (EM-P2-001):
//
//	prior = weight · base · trust · convergenceConfidence
//
// with trust = pass_rate. RED ⇒ trust→0 (it is gated anyway); UNKNOWN or
// thin evidence ⇒ prior 0 (P1 uniform). The bandit blend washes the prior out
// as live evidence accrues, so this only steers the cold start.
type DTVPriorProvider struct {
	snapshot FitnessSnapshot
	cfg      DTVPriorConfig
}

var _ PriorProvider = DTVPriorProvider{}

// NewDTVPriorProvider builds the prior provider over a snapshot and config.
func NewDTVPriorProvider(s FitnessSnapshot, cfg DTVPriorConfig) DTVPriorProvider {
	return DTVPriorProvider{snapshot: s, cfg: cfg.withDefaults()}
}

// Prior implements PriorProvider.
func (p DTVPriorProvider) Prior(skillID string, _ dimensions.Dimension) float64 {
	if p.cfg.Weight <= 0 {
		return 0
	}
	f := p.snapshot.Get(skillID)
	if f.Verdict == dtv.VerdictUnknown || f.TotalRuns < p.cfg.MinRuns {
		return 0 // fail open / thin evidence ⇒ uniform P1 prior
	}
	trust := f.PassRate
	if trust <= 0 || trust < p.cfg.TrustFloor {
		return 0
	}
	return p.cfg.Weight * p.cfg.Base * trust * convergenceConfidence(f, p.cfg)
}

// convergenceConfidence damps the prior for skills that keep producing different
// diffs (non-convergent) and for stale validations.
func convergenceConfidence(f dtv.Fitness, cfg DTVPriorConfig) float64 {
	conf := 1.0
	if f.UniqueDiffHashes > 1 {
		conf = cfg.ConvK / float64(f.UniqueDiffHashes)
		if conf > 1 {
			conf = 1
		}
	}
	if f.AnyStale {
		conf *= cfg.StaleFactor
	}
	return conf
}
