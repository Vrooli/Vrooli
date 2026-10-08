package dbdetect

import (
	"context"
	"fmt"
)

// Resolver runs collectors once per Resolve call, evaluates each profile, and
// produces a DetectionReport.
type Resolver struct {
	collectors []Collector
	byName     map[string]Collector
	profiles   []Profile
}

// NewResolver constructs a Resolver. It returns an error if any profile
// references a collector name that is not present, or if a profile is
// malformed (empty DB name, source with no collector).
func NewResolver(collectors []Collector, profiles []Profile) (*Resolver, error) {
	byName := make(map[string]Collector, len(collectors))
	for _, c := range collectors {
		if c == nil {
			return nil, fmt.Errorf("nil collector in registry")
		}
		name := c.Name()
		if name == "" {
			return nil, fmt.Errorf("collector with empty name")
		}
		if _, dup := byName[name]; dup {
			return nil, fmt.Errorf("duplicate collector name %q", name)
		}
		byName[name] = c
	}
	for _, p := range profiles {
		if p.DB == "" {
			return nil, fmt.Errorf("profile with empty DB name")
		}
		for _, s := range p.Sources {
			if s.Collector == "" {
				return nil, fmt.Errorf("profile %q has source with empty collector", p.DB)
			}
			if _, ok := byName[s.Collector]; !ok {
				return nil, fmt.Errorf("profile %q references unknown collector %q", p.DB, s.Collector)
			}
			if s.Match == nil {
				return nil, fmt.Errorf("profile %q source %s has nil matcher", p.DB, s.Collector)
			}
		}
	}
	// Register the union of source tokens so the source collector knows what
	// to scan for.
	SetSourceTokens(SourceTokens(profiles))
	return &Resolver{
		collectors: collectors,
		byName:     byName,
		profiles:   profiles,
	}, nil
}

// DefaultCollectors returns the canonical collector set: manifest, godeps,
// source.
func DefaultCollectors() []Collector {
	return []Collector{
		ManifestCollector{},
		GodepsCollector{},
		SourceCollector{},
	}
}

// Resolve runs every collector once, then evaluates each profile against the
// collected observations and returns a DetectionReport. Collector errors are
// captured per-DB as conflicts so a flaky single collector cannot poison the
// whole resolution.
func (r *Resolver) Resolve(ctx context.Context, in ScenarioInputs) DetectionReport {
	if in.Filesystem == nil {
		in.Filesystem = OSFilesystem{}
	}
	observations := make(map[string][]Observation, len(r.collectors))
	collectorErrs := make(map[string]error, len(r.collectors))
	for _, c := range r.collectors {
		obs, err := c.Collect(ctx, in)
		if err != nil {
			collectorErrs[c.Name()] = err
			continue
		}
		observations[c.Name()] = obs
	}

	results := make(map[string]DBResult, len(r.profiles))
	order := make([]string, 0, len(r.profiles))
	for _, p := range r.profiles {
		order = append(order, p.DB)
		results[p.DB] = resolveProfile(p, observations, collectorErrs)
	}
	return DetectionReport{Results: results, Order: order}
}

// resolveProfile applies one profile's sources to the observation set.
//
// Decision rules (from the implementation plan §7 Phase 3 Step B table):
//   - No source matched anywhere → Required:false, Decision:nil
//   - Single source matched → Required:true, that hit is Decision
//   - Multiple sources matched → highest priority wins as Decision, rest go
//     to Corroborating
//   - Ties on highest priority → first registered source wins (deterministic
//     by profile order); the rest are Corroborating
//   - Lower-priority source listed in profile but found nothing while a
//     higher-priority source did → Conflict{Kind:"missing-corroboration"};
//     still Required:true
//   - Collector errored → Conflict{Kind:"collector-error"}; DB requirement
//     is determined from whatever other sources did report
func resolveProfile(p Profile, observations map[string][]Observation, collectorErrs map[string]error) DBResult {
	result := DBResult{DB: p.DB}
	type hit struct {
		src  ProfileSource
		evs  []Evidence
		hit  bool
		errd bool
	}
	hits := make([]hit, len(p.Sources))
	anyMatched := false
	for i, s := range p.Sources {
		h := hit{src: s}
		if _, errd := collectorErrs[s.Collector]; errd {
			h.errd = true
			result.Conflicts = append(result.Conflicts, Conflict{
				Kind:   "collector-error",
				Detail: fmt.Sprintf("%s: %v", s.Collector, collectorErrs[s.Collector]),
			})
			hits[i] = h
			continue
		}
		for _, obs := range observations[s.Collector] {
			if s.Match(obs) {
				h.evs = append(h.evs, Evidence{
					Source:    s.Label,
					Priority:  s.Priority,
					Detail:    obs.Value,
					Locations: obs.Locations,
				})
				h.hit = true
				anyMatched = true
			}
		}
		hits[i] = h
	}
	if !anyMatched {
		return result
	}
	result.Required = true

	// Pick decision = highest priority, first registered on ties.
	var decision *Evidence
	for i := range hits {
		if !hits[i].hit {
			continue
		}
		for j := range hits[i].evs {
			ev := hits[i].evs[j]
			if decision == nil || ev.Priority > decision.Priority {
				cp := ev
				decision = &cp
			}
		}
	}
	result.Decision = decision

	// Everything else is corroborating, in profile-source order.
	for i := range hits {
		if !hits[i].hit {
			continue
		}
		for j := range hits[i].evs {
			ev := hits[i].evs[j]
			if decision != nil && ev.Source == decision.Source && ev.Detail == decision.Detail {
				continue
			}
			result.Corroborating = append(result.Corroborating, ev)
		}
	}

	// Missing-corroboration conflicts: every profile source that yielded
	// nothing while at least one other did, and is not a collector-error.
	for i := range hits {
		if hits[i].hit || hits[i].errd {
			continue
		}
		result.Conflicts = append(result.Conflicts, Conflict{
			Kind:   "missing-corroboration",
			Detail: hits[i].src.Label,
		})
	}
	return result
}
