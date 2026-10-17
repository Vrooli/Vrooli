package capabilities_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"content-desk/internal/capabilities"

	"github.com/stretchr/testify/require"
)

type fakeResolver struct {
	states map[string]capabilities.OutputState
	errs   map[string]error
	calls  []string
}

func (f *fakeResolver) CurrentOutputState(_ context.Context, artifactID string) (capabilities.OutputState, error) {
	f.calls = append(f.calls, artifactID)
	if err, ok := f.errs[artifactID]; ok && err != nil {
		return capabilities.OutputState{}, err
	}
	if state, ok := f.states[artifactID]; ok {
		return state, nil
	}
	return capabilities.OutputState{ArtifactID: artifactID, Resolvable: false}, nil
}

func evidenceLink(capabilityID, target string) capabilities.CapabilityLink {
	return capabilities.CapabilityLink{CapabilityID: capabilityID, Relation: capabilities.RelationEvidence, TargetID: target}
}

func TestRehydrateOutputQualityKeepsStoredWhenNothingLive(t *testing.T) {
	projection := capabilities.RehydrateOutputQuality(context.Background(), capabilities.QualityAccepted, nil, nil)
	require.Equal(t, capabilities.QualityAccepted, projection.Quality)
	require.Equal(t, capabilities.Unknown, projection.Source)
	require.Len(t, projection.Limitations, 1)

	noLinks := capabilities.RehydrateOutputQuality(context.Background(), capabilities.QualityAccepted, nil, &fakeResolver{})
	require.Equal(t, capabilities.QualityAccepted, noLinks.Quality)
	require.Equal(t, capabilities.Unknown, noLinks.Source)
	require.Len(t, noLinks.Limitations, 1)
}

func TestRehydrateOutputQualityAcceptsFromOwnerAndBindsSource(t *testing.T) {
	resolver := &fakeResolver{states: map[string]capabilities.OutputState{
		"draft-1": {ArtifactID: "draft-1", Resolvable: true, Accepted: true, ObservedAt: "2026-09-12T00:00:00Z"},
	}}
	links := []capabilities.CapabilityLink{
		{CapabilityID: "cap", Relation: capabilities.RelationProduces, TargetID: "program-x"},
		evidenceLink("cap", "draft-1"),
	}
	projection := capabilities.RehydrateOutputQuality(context.Background(), capabilities.QualityUnassessed, links, resolver)
	require.Equal(t, capabilities.QualityAccepted, projection.Quality)
	require.Equal(t, "draft-1", projection.Source)
	require.Equal(t, "2026-09-12T00:00:00Z", projection.ObservedAt)
	require.Empty(t, projection.Limitations)
	require.Equal(t, []string{"draft-1"}, resolver.calls)
}

func TestRehydrateOutputQualityUnassessedWhenOwnerHasNotAccepted(t *testing.T) {
	resolver := &fakeResolver{states: map[string]capabilities.OutputState{
		"draft-1": {ArtifactID: "draft-1", Resolvable: true, Accepted: false},
		"draft-2": {ArtifactID: "draft-2", Resolvable: false},
	}}
	links := []capabilities.CapabilityLink{evidenceLink("cap", "draft-1"), evidenceLink("cap", "draft-2")}
	projection := capabilities.RehydrateOutputQuality(context.Background(), capabilities.QualityAccepted, links, resolver)
	require.Equal(t, capabilities.QualityUnassessed, projection.Quality)
	require.Equal(t, "draft-1", projection.Source)
	require.Empty(t, projection.Limitations)
	require.Equal(t, []string{"draft-1", "draft-2"}, resolver.calls)
}

func TestRehydrateOutputQualitySkipsUnknownArtifactsAndDedupes(t *testing.T) {
	resolver := &fakeResolver{}
	links := []capabilities.CapabilityLink{
		evidenceLink("cap", "claim-1"),
		evidenceLink("cap", "draft-1"),
		evidenceLink("cap", "draft-1"),
	}
	projection := capabilities.RehydrateOutputQuality(context.Background(), capabilities.QualityAccepted, links, resolver)
	require.Equal(t, capabilities.QualityAccepted, projection.Quality)
	require.Equal(t, capabilities.Unknown, projection.Source)
	require.Len(t, projection.Limitations, 1)
	require.Equal(t, []string{"claim-1", "draft-1"}, resolver.calls)
}

func TestRehydrateOutputQualityKeepsStoredWhenEveryReadFails(t *testing.T) {
	resolver := &fakeResolver{errs: map[string]error{"draft-1": errors.New("owner unavailable")}}
	projection := capabilities.RehydrateOutputQuality(context.Background(), capabilities.QualityUnassessed, []capabilities.CapabilityLink{evidenceLink("cap", "draft-1")}, resolver)
	require.Equal(t, capabilities.QualityUnassessed, projection.Quality)
	require.Equal(t, capabilities.Unknown, projection.Source)
	require.Len(t, projection.Limitations, 1)
}

type fakeDistributionResolver struct {
	state capabilities.DistributionState
	err   error
	calls int
}

func (f *fakeDistributionResolver) ConnectedSurfaces(context.Context) (capabilities.DistributionState, error) {
	f.calls++
	return f.state, f.err
}

func distributionCapability(owner string) capabilities.Capability {
	return capabilities.Capability{ID: "cap", Medium: capabilities.MediumDistribution, Owner: owner, DistributionConnectivity: capabilities.ConnectivityDisconnected}
}

func TestRehydrateDistributionConnectivityNotApplicableForNonDistribution(t *testing.T) {
	resolver := &fakeDistributionResolver{}
	capability := capabilities.Capability{ID: "cap", Medium: capabilities.MediumText, Owner: "prose-studio", DistributionConnectivity: capabilities.ConnectivityConnected}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityConnected, capability, resolver)
	require.Equal(t, capabilities.ConnectivityNotApplicable, projection.Connectivity)
	require.Equal(t, "cap", projection.Source)
	require.Empty(t, projection.Limitations)
	require.Zero(t, resolver.calls)
}

func TestRehydrateDistributionConnectivityReadsChannelManagerRegardlessOfMedium(t *testing.T) {
	resolver := &fakeDistributionResolver{state: capabilities.DistributionState{Connected: true, Source: "channel-manager:overview"}}
	capability := capabilities.Capability{ID: "cap", Medium: capabilities.MediumMeasurement, Owner: "channel-manager", DistributionConnectivity: capabilities.ConnectivityDisconnected}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityDisconnected, capability, resolver)
	require.Equal(t, capabilities.ConnectivityConnected, projection.Connectivity)
	require.Equal(t, "channel-manager:overview", projection.Source)
	require.Equal(t, 1, resolver.calls)
}

func TestRehydrateDistributionConnectivityNotApplicableForOwnerlessNonDistribution(t *testing.T) {
	resolver := &fakeDistributionResolver{}
	capability := capabilities.Capability{ID: "cap", Medium: capabilities.MediumVideo}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityNotApplicable, capability, resolver)
	require.Equal(t, capabilities.ConnectivityNotApplicable, projection.Connectivity)
	require.Equal(t, "cap", projection.Source)
	require.Empty(t, projection.Limitations)
	require.Zero(t, resolver.calls)
}

func TestRehydrateDistributionConnectivityDisconnectedWithoutOwner(t *testing.T) {
	resolver := &fakeDistributionResolver{}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityDisconnected, distributionCapability(""), resolver)
	require.Equal(t, capabilities.ConnectivityDisconnected, projection.Connectivity)
	require.Equal(t, "cap", projection.Source)
	require.Len(t, projection.Limitations, 1)
	require.Zero(t, resolver.calls)
}

func TestRehydrateDistributionConnectivityKeepsStoredForForeignOwner(t *testing.T) {
	resolver := &fakeDistributionResolver{state: capabilities.DistributionState{Connected: true}}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityDisconnected, distributionCapability("landing-page-business-suite"), resolver)
	require.Equal(t, capabilities.ConnectivityDisconnected, projection.Connectivity)
	require.Equal(t, capabilities.Unknown, projection.Source)
	require.Len(t, projection.Limitations, 1)
	require.Zero(t, resolver.calls)
}

func TestRehydrateDistributionConnectivityKeepsStoredWhenResolverUnavailable(t *testing.T) {
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityDisconnected, distributionCapability("channel-manager"), nil)
	require.Equal(t, capabilities.ConnectivityDisconnected, projection.Connectivity)
	require.Equal(t, capabilities.Unknown, projection.Source)
	require.Len(t, projection.Limitations, 1)
}

func TestRehydrateDistributionConnectivityKeepsStoredWhenReadFails(t *testing.T) {
	resolver := &fakeDistributionResolver{err: errors.New("owner unavailable")}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityDisconnected, distributionCapability("channel-manager"), resolver)
	require.Equal(t, capabilities.ConnectivityDisconnected, projection.Connectivity)
	require.Equal(t, capabilities.Unknown, projection.Source)
	require.Len(t, projection.Limitations, 1)
}

func TestRehydrateDistributionConnectivityConnectedFromOwner(t *testing.T) {
	resolver := &fakeDistributionResolver{state: capabilities.DistributionState{Connected: true, Source: "channel-manager:overview", ObservedAt: "2026-09-12T00:00:00Z"}}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityDisconnected, distributionCapability("channel-manager"), resolver)
	require.Equal(t, capabilities.ConnectivityConnected, projection.Connectivity)
	require.Equal(t, "channel-manager:overview", projection.Source)
	require.Equal(t, "2026-09-12T00:00:00Z", projection.ObservedAt)
	require.Empty(t, projection.Limitations)
	require.Equal(t, 1, resolver.calls)
}

func TestRehydrateDistributionConnectivityDisconnectedFromReachableOwner(t *testing.T) {
	resolver := &fakeDistributionResolver{state: capabilities.DistributionState{Connected: false, Source: "channel-manager:overview"}}
	projection := capabilities.RehydrateDistributionConnectivity(context.Background(), capabilities.ConnectivityConnected, distributionCapability("channel-manager"), resolver)
	require.Equal(t, capabilities.ConnectivityDisconnected, projection.Connectivity)
	require.Equal(t, "channel-manager:overview", projection.Source)
	require.Empty(t, projection.Limitations)
	require.Equal(t, 1, resolver.calls)
}

func observedQualification(artifactID string) *capabilities.CapabilityQualification {
	return &capabilities.CapabilityQualification{
		CapabilityID:     "cap",
		LatestArtifactID: artifactID,
		LatestRunID:      "run-1",
		Environment:      "local",
		ObservedAt:       "2026-09-12T00:00:00Z",
		FreshnessBasis:   capabilities.FreshnessMaxAge,
		MaxAgeSeconds:    604800,
	}
}

func TestRehydrateOperationalReadinessUnavailableWithoutProducingOwner(t *testing.T) {
	projection := capabilities.RehydrateOperationalReadiness(capabilities.ReadinessUnverified, capabilities.Capability{ID: "cap"}, time.Now())
	require.Equal(t, capabilities.ReadinessUnavailable, projection.Readiness)
	require.Equal(t, "cap", projection.Source)
	require.Len(t, projection.Limitations, 1)
}

func TestRehydrateOperationalReadinessKeepsStoredWithoutObservation(t *testing.T) {
	capability := capabilities.Capability{ID: "cap", Owner: "prose-studio", OperationalReadiness: capabilities.ReadinessQualified}
	projection := capabilities.RehydrateOperationalReadiness(capabilities.ReadinessQualified, capability, time.Now())
	require.Equal(t, capabilities.ReadinessQualified, projection.Readiness)
	require.Equal(t, capabilities.Unknown, projection.Source)
	require.Len(t, projection.Limitations, 1)
}

func TestRehydrateOperationalReadinessQualifiesFromFreshObservation(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	capability := capabilities.Capability{ID: "cap", Owner: "browser-automation-studio", LatestQualification: observedQualification("capture-1")}
	projection := capabilities.RehydrateOperationalReadiness(capabilities.ReadinessUnverified, capability, now)
	require.Equal(t, capabilities.ReadinessQualified, projection.Readiness)
	require.Equal(t, "capture-1", projection.Source)
	require.Equal(t, "2026-09-12T00:00:00Z", projection.ObservedAt)
	require.Empty(t, projection.Limitations)
}

func TestRehydrateOperationalReadinessDegradesStaleObservation(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	capability := capabilities.Capability{ID: "cap", Owner: "browser-automation-studio", LatestQualification: observedQualification("capture-1")}
	projection := capabilities.RehydrateOperationalReadiness(capabilities.ReadinessQualified, capability, now)
	require.Equal(t, capabilities.ReadinessUnverified, projection.Readiness)
	require.Equal(t, "capture-1", projection.Source)
	require.Len(t, projection.Limitations, 1)
}

func TestRehydrateOperationalReadinessCandidateIdentityNeverTimeStale(t *testing.T) {
	now := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	qualification := observedQualification("capture-1")
	qualification.FreshnessBasis = capabilities.FreshnessCandidateIdentity
	qualification.MaxAgeSeconds = capabilities.UnknownMaxAgeSeconds
	capability := capabilities.Capability{ID: "cap", Owner: "browser-automation-studio", LatestQualification: qualification}
	projection := capabilities.RehydrateOperationalReadiness(capabilities.ReadinessUnverified, capability, now)
	require.Equal(t, capabilities.ReadinessQualified, projection.Readiness)
	require.Equal(t, "capture-1", projection.Source)
	require.Empty(t, projection.Limitations)
}

func TestRehydrateOperationalReadinessDegradesUnparseableObservation(t *testing.T) {
	qualification := observedQualification("")
	qualification.ObservedAt = "not-a-timestamp"
	capability := capabilities.Capability{ID: "cap", Owner: "channel-manager", LatestQualification: qualification}
	projection := capabilities.RehydrateOperationalReadiness(capabilities.ReadinessUnavailable, capability, time.Now())
	require.Equal(t, capabilities.ReadinessUnverified, projection.Readiness)
	require.Equal(t, "run-1", projection.Source)
	require.Len(t, projection.Limitations, 1)
}
