package capabilities

import (
	"context"
	"strings"
	"time"
)

// OutputState is the current, owner-held state of one produced artifact. A
// resolver reads it at response time; the capability domain never persists it.
type OutputState struct {
	// ArtifactID is the owner record the state was read from.
	ArtifactID string
	// Resolvable is false when the linked id is not an artifact this owner
	// knows (for example a claim or campaign id), so the link is skipped
	// instead of being reported as an error.
	Resolvable bool
	// Accepted is true when the owner's current state is an accepted output
	// (reviewed, approved or published), independent of any stored verdict.
	Accepted bool
	// ObservedAt is the owner's observation time, empty when unknown.
	ObservedAt string
}

// EvidenceResolver reads current produced-artifact state from its owning
// domain. Implementations must be read-only: they observe owner state and never
// mutate it, and they never write a copy back into the capability catalog.
type EvidenceResolver interface {
	CurrentOutputState(ctx context.Context, artifactID string) (OutputState, error)
}

// OutputQualityProjection is the read-time projection of one capability's
// output_quality dimension. Source names the owner artifact that produced the
// verdict, or Unknown when the stored value was kept.
type OutputQualityProjection struct {
	Quality     string
	Source      string
	ObservedAt  string
	Limitations []string
}

// evidenceTargets returns the distinct, non-empty evidence-link targets in
// link order so the same artifact is resolved at most once.
func evidenceTargets(links []CapabilityLink) []string {
	seen := make(map[string]struct{}, len(links))
	targets := make([]string, 0, len(links))
	for _, link := range links {
		if link.Relation != RelationEvidence {
			continue
		}
		target := strings.TrimSpace(link.TargetID)
		if target == "" {
			continue
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		targets = append(targets, target)
	}
	return targets
}

// RehydrateOutputQuality derives output_quality from the current state of the
// capability's evidence-linked owner artifacts without copying that state.
//
// It returns the stored value unchanged, with a named limitation, when there is
// nothing live to read (no resolver or no evidence links) or when every read
// fails. When a linked artifact is accepted by its owner it reports
// accepted-by-review bound to that artifact. When linked artifacts exist but
// none is accepted it reports unassessed rather than inventing a verdict. A
// linked id the owner does not know is skipped, not treated as an error.
func RehydrateOutputQuality(ctx context.Context, stored string, links []CapabilityLink, resolver EvidenceResolver) OutputQualityProjection {
	targets := evidenceTargets(links)
	if resolver == nil {
		return OutputQualityProjection{
			Quality:     stored,
			Source:      Unknown,
			Limitations: []string{"evidence resolver unavailable; output_quality is the stored inventory value"},
		}
	}
	if len(targets) == 0 {
		return OutputQualityProjection{
			Quality:     stored,
			Source:      Unknown,
			Limitations: []string{"no linked evidence; output_quality is the stored inventory value"},
		}
	}

	var (
		firstResolved *OutputState
		unreadable    []string
	)
	for _, target := range targets {
		state, err := resolver.CurrentOutputState(ctx, target)
		if err != nil {
			unreadable = append(unreadable, target)
			continue
		}
		if !state.Resolvable {
			continue
		}
		state.ArtifactID = target
		if firstResolved == nil {
			copied := state
			firstResolved = &copied
		}
		if state.Accepted {
			return OutputQualityProjection{Quality: QualityAccepted, Source: target, ObservedAt: state.ObservedAt}
		}
	}

	if firstResolved != nil {
		projection := OutputQualityProjection{Quality: QualityUnassessed, Source: firstResolved.ArtifactID, ObservedAt: firstResolved.ObservedAt}
		if len(unreadable) > 0 {
			projection.Limitations = append(projection.Limitations, "some linked evidence could not be read: "+strings.Join(unreadable, ","))
		}
		return projection
	}

	return OutputQualityProjection{
		Quality:     stored,
		Source:      Unknown,
		Limitations: []string{"no linked evidence resolved; unreadable: " + strings.Join(unreadable, ",")},
	}
}

// OperationalReadinessProjection is the read-time projection of one
// capability's operational_readiness dimension. Source names the owner record
// that established the verdict, or Unknown when the stored value was kept.
type OperationalReadinessProjection struct {
	Readiness   string
	Source      string
	ObservedAt  string
	Limitations []string
}

// RehydrateOperationalReadiness derives operational_readiness from source
// records instead of returning the frozen inventory value.
//
// It is structural for a capability with no producing owner: no owner means the
// producing operation does not exist, so readiness is unavailable and never the
// stored value. When an owner exists, the capability's own qualification is the
// owner record: an observed, fresh qualification establishes
// qualified-in-environment bound to the qualification artifact/run; an observed
// but stale qualification degrades to unverified. When no qualification has
// been observed the stored inventory value is kept with a named limitation, so
// a real Phase 1/2 observation is not silently overwritten.
func RehydrateOperationalReadiness(stored string, capability Capability, now time.Time) OperationalReadinessProjection {
	if strings.TrimSpace(capability.Owner) == "" {
		return OperationalReadinessProjection{
			Readiness:   ReadinessUnavailable,
			Source:      capability.ID,
			Limitations: []string{"no producing owner; operational_readiness is unavailable"},
		}
	}
	qualification := capability.LatestQualification
	if qualification == nil || qualification.IsUnknown() {
		return OperationalReadinessProjection{
			Readiness:   stored,
			Source:      Unknown,
			Limitations: []string{"no observed qualification; operational_readiness is the stored inventory value"},
		}
	}
	source := qualificationSource(*qualification)
	if qualificationStale(*qualification, now) {
		return OperationalReadinessProjection{
			Readiness:   ReadinessUnverified,
			Source:      source,
			ObservedAt:  qualification.ObservedAt,
			Limitations: []string{"qualification is stale; operational_readiness is unverified"},
		}
	}
	return OperationalReadinessProjection{
		Readiness:  ReadinessQualified,
		Source:     source,
		ObservedAt: qualification.ObservedAt,
	}
}

// channelManagerOwner is the owner scenario whose live account surface is the
// distribution surface for the capabilities it produces.
const channelManagerOwner = "channel-manager"

// DistributionState is the current, owner-held connectivity of a distribution
// surface. A resolver reads it at response time; the capability domain never
// persists it. It carries no handle, credential or platform-account detail.
type DistributionState struct {
	// Connected is true when the owner reports at least one live, attributable
	// distribution surface.
	Connected bool
	// Source names the owner record/endpoint the state was read from, empty
	// when the owner did not report one.
	Source string
	// ObservedAt is the owner's observation time, empty when unknown.
	ObservedAt string
}

// DistributionResolver reads current distribution-surface connectivity from its
// owning domain. Implementations must be read-only: they observe owner state
// and never mutate it, and they never write a copy back into the catalog.
type DistributionResolver interface {
	ConnectedSurfaces(ctx context.Context) (DistributionState, error)
}

// DistributionConnectivityProjection is the read-time projection of one
// capability's distribution_connectivity dimension. Source names the owner
// record/endpoint that established the verdict, or Unknown when the stored
// value was kept.
type DistributionConnectivityProjection struct {
	Connectivity string
	Source       string
	ObservedAt   string
	Limitations  []string
}

// RehydrateDistributionConnectivity derives distribution_connectivity from the
// current state of the capability's distribution owner instead of returning the
// frozen inventory value.
//
// The producing owner is the discriminator. Only a channel-manager-owned
// capability has a surface this resolver can observe; a reachable owner yields
// a real connected or disconnected verdict bound to the owner read, so a live
// "disconnected" is distinguishable from a stored one. An unreachable or
// unconfigured reader keeps the stored value with a named limitation rather
// than fabricating a verdict.
//
// A distribution-medium capability with no producing owner is disconnected,
// bound to the capability id, because the wiring operation does not exist. A
// distribution capability owned by another scenario has no reader here, so its
// stored value is kept with a named limitation. Any remaining capability has no
// distribution surface to observe and is structurally not-applicable.
func RehydrateDistributionConnectivity(ctx context.Context, stored string, capability Capability, resolver DistributionResolver) DistributionConnectivityProjection {
	if capability.Owner == channelManagerOwner {
		if resolver == nil {
			return DistributionConnectivityProjection{
				Connectivity: stored,
				Source:       Unknown,
				Limitations:  []string{"distribution resolver unavailable; distribution_connectivity is the stored inventory value"},
			}
		}
		state, err := resolver.ConnectedSurfaces(ctx)
		if err != nil {
			return DistributionConnectivityProjection{
				Connectivity: stored,
				Source:       Unknown,
				Limitations:  []string{"distribution owner read failed; distribution_connectivity is the stored inventory value: " + boundedReason(err)},
			}
		}
		source := strings.TrimSpace(state.Source)
		if source == "" {
			source = channelManagerOwner
		}
		connectivity := ConnectivityDisconnected
		if state.Connected {
			connectivity = ConnectivityConnected
		}
		return DistributionConnectivityProjection{
			Connectivity: connectivity,
			Source:       source,
			ObservedAt:   state.ObservedAt,
		}
	}
	if capability.Medium == MediumDistribution {
		if strings.TrimSpace(capability.Owner) == "" {
			return DistributionConnectivityProjection{
				Connectivity: ConnectivityDisconnected,
				Source:       capability.ID,
				Limitations:  []string{"no producing owner; distribution_connectivity is disconnected"},
			}
		}
		return DistributionConnectivityProjection{
			Connectivity: stored,
			Source:       Unknown,
			Limitations:  []string{"distribution owner " + capability.Owner + " has no connectivity reader; distribution_connectivity is the stored inventory value"},
		}
	}
	return DistributionConnectivityProjection{Connectivity: ConnectivityNotApplicable, Source: capability.ID}
}

// boundedReason renders an owner-read error as one bounded line so a response
// limitation names the failure without embedding a raw multi-line payload.
func boundedReason(err error) string {
	reason := strings.Join(strings.Fields(err.Error()), " ")
	const max = 200
	if len(reason) > max {
		reason = reason[:max] + "..."
	}
	return reason
}

// qualificationSource names the owner record behind an observed qualification,
// preferring the produced artifact and falling back to the run.
func qualificationSource(qualification CapabilityQualification) string {
	if artifact := strings.TrimSpace(qualification.LatestArtifactID); artifact != "" && artifact != Unknown {
		return artifact
	}
	if run := strings.TrimSpace(qualification.LatestRunID); run != "" && run != Unknown {
		return run
	}
	return Unknown
}

// qualificationStale reports whether an observed qualification has aged past
// its declared max age. A candidate_identity basis is bound to a source
// revision rather than time, and an unknown/unbounded age is not treated as
// fresh or as stale; both keep their recorded verdict.
func qualificationStale(qualification CapabilityQualification, now time.Time) bool {
	if qualification.FreshnessBasis != FreshnessMaxAge {
		return false
	}
	if qualification.MaxAgeSeconds <= 0 {
		return false
	}
	observed, err := time.Parse(time.RFC3339Nano, qualification.ObservedAt)
	if err != nil {
		return true
	}
	return now.Sub(observed) > time.Duration(qualification.MaxAgeSeconds)*time.Second
}
