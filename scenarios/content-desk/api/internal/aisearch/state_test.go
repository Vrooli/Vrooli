package aisearch

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	internalartifacts "content-desk/internal/artifacts"
	internalcampaigns "content-desk/internal/campaigns"
	internalcapabilities "content-desk/internal/capabilities"
)

type capabilityRepoStub struct {
	internalcapabilities.Repository
	caps     []internalcapabilities.Capability
	links    []internalcapabilities.CapabilityLink
	listErr  error
	linksErr error
}

func (s capabilityRepoStub) List(context.Context, string) ([]internalcapabilities.Capability, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.caps, nil
}

func (s capabilityRepoStub) ListLinks(context.Context) ([]internalcapabilities.CapabilityLink, error) {
	if s.linksErr != nil {
		return nil, s.linksErr
	}
	return s.links, nil
}

type campaignRepoStub struct {
	internalcampaigns.Repository
	campaigns []internalcampaigns.Campaign
	slots     []internalcampaigns.LaunchAssetSlot
	listErr   error
	launchErr error
}

func (s campaignRepoStub) List(context.Context) ([]internalcampaigns.Campaign, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.campaigns, nil
}

func (s campaignRepoStub) LaunchAssets(context.Context, string) ([]internalcampaigns.LaunchAssetSlot, error) {
	if s.launchErr != nil {
		return nil, s.launchErr
	}
	return s.slots, nil
}

func newStateSource(caps internalcapabilities.Repository, campaigns internalcampaigns.Repository) *StoreSource {
	source := NewStoreSource(sourceDraftsStub{}, sourceLedgerStub{})
	if caps != nil {
		source = source.WithCapabilityCatalog(caps, nil, nil)
	}
	if campaigns != nil {
		source = source.WithCampaigns(campaigns)
	}
	return source
}

func indexRecords(snapshot *Snapshot) map[string]Record {
	byID := make(map[string]Record, len(snapshot.Records))
	for _, record := range snapshot.Records {
		byID[record.ID] = record
	}
	return byID
}

func hitIDs(hits []SearchHit) []string {
	ids := make([]string, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.ID)
	}
	return ids
}

func TestCapabilityRecordsProjectReadinessDimensions(t *testing.T) {
	updated := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	repo := capabilityRepoStub{caps: []internalcapabilities.Capability{{
		ID:                       "cap-written",
		Name:                     "Written marketing",
		Medium:                   internalcapabilities.MediumText,
		Aliases:                  []string{"written-marketing", "long-form"},
		Channels:                 []string{"blog"},
		Owner:                    "prose-studio",
		ProducingOperation:       "prose-studio",
		DefinitionStatus:         internalcapabilities.DefinitionDocumented,
		ImplementationStatus:     internalcapabilities.ImplementationImplemented,
		OperationalReadiness:     internalcapabilities.ReadinessUnverified,
		OutputQuality:            internalcapabilities.QualityUnassessed,
		DistributionConnectivity: internalcapabilities.ConnectivityNotApplicable,
		NextAction:               "Exercise the review/approval path.",
		CreatedAt:                updated,
		UpdatedAt:                updated,
		LatestQualification: &internalcapabilities.CapabilityQualification{
			CapabilityID:   "cap-written",
			Environment:    "prose-studio-local",
			ObservedAt:     "2026-09-12T12:00:00Z",
			FreshnessBasis: internalcapabilities.FreshnessCandidateIdentity,
			MaxAgeSeconds:  internalcapabilities.UnknownMaxAgeSeconds,
		},
	}}}

	snapshot, err := newStateSource(repo, nil).Load(context.Background())
	require.NoError(t, err)
	record, ok := indexRecords(snapshot)["capability:cap-written"]
	require.True(t, ok)
	require.Equal(t, KindCapability, record.Kind)
	require.Equal(t, "capability/cap-written", record.FollowUp)
	require.Equal(t, "Written marketing", record.Title)
	require.Equal(t, "prose-studio", record.Metadata["owner"])
	require.Equal(t, []string{"written-marketing", "long-form"}, record.Metadata["aliases"])
	require.Equal(t, internalcapabilities.ReadinessQualified, record.Metadata["operational_readiness"])
	require.Equal(t, internalcapabilities.QualityUnassessed, record.Metadata["output_quality"])
	require.Equal(t, internalcapabilities.ConnectivityNotApplicable, record.Metadata["distribution_connectivity"])
	require.Equal(t, "Exercise the review/approval path.", record.Metadata["next_action"])
	require.Equal(t, true, record.Metadata["supported"])
	require.Contains(t, record.Snippet, "Written marketing is a text marketing capability.")
	require.Contains(t, record.Snippet, "Owner: prose-studio.")
	require.Contains(t, record.Snippet, "Next action: Exercise the review/approval path.")
	require.NotContains(t, record.Snippet, "Latest qualified artifact")
	qualityLimitations, ok := record.Metadata["output_quality_limitations"].([]string)
	require.True(t, ok)
	require.Contains(t, strings.Join(qualityLimitations, " "), "stored inventory value")
	readinessLimitations, ok := record.Metadata["operational_readiness_limitations"].([]string)
	require.True(t, ok)
	require.Empty(t, readinessLimitations)
	qualification, ok := record.Metadata["latest_qualification"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "prose-studio-local", qualification["environment"])
	require.Equal(t, updated, snapshot.MaterializedAt)
}

func TestCapabilityRecordSurfacesLatestQualifiedArtifact(t *testing.T) {
	updated := time.Date(2026, time.September, 13, 1, 10, 0, 0, time.UTC)
	repo := capabilityRepoStub{caps: []internalcapabilities.Capability{{
		ID:                       "cap-shots",
		Name:                     "Product screenshots",
		Medium:                   internalcapabilities.MediumImage,
		Aliases:                  []string{"product-screenshots", "screenshots"},
		Owner:                    "browser-automation-studio",
		ProducingOperation:       "browser-automation-studio capture-surface",
		DefinitionStatus:         internalcapabilities.DefinitionDocumented,
		ImplementationStatus:     internalcapabilities.ImplementationImplemented,
		OperationalReadiness:     internalcapabilities.ReadinessQualified,
		OutputQuality:            internalcapabilities.QualityUnassessed,
		DistributionConnectivity: internalcapabilities.ConnectivityNotApplicable,
		NextAction:               "Keep captures attributable to the launch bundle manifest.",
		CreatedAt:                updated,
		UpdatedAt:                updated,
		LatestQualification: &internalcapabilities.CapabilityQualification{
			CapabilityID:     "cap-shots",
			LatestArtifactID: "aquila-web-console-desktop-2026-09-12.jpg",
			LatestRunID:      "ec335946-eeb0-460e-a175-b1f865a6dde0",
			Environment:      "browser-automation-studio live",
			ObservedAt:       "2026-09-12T12:55:00Z",
		},
	}}}

	snapshot, err := newStateSource(repo, nil).Load(context.Background())
	require.NoError(t, err)
	record, ok := indexRecords(snapshot)["capability:cap-shots"]
	require.True(t, ok)
	require.Contains(t, record.Snippet, "Latest qualified artifact aquila-web-console-desktop-2026-09-12.jpg (run ec335946-eeb0-460e-a175-b1f865a6dde0).")
	require.Contains(t, record.Body, "latest qualified artifact aquila-web-console-desktop-2026-09-12.jpg")
	require.Contains(t, record.Body, "latest qualification run ec335946-eeb0-460e-a175-b1f865a6dde0")
}

func TestCampaignRecordsCarryLaunchAssetReadinessAndNextAction(t *testing.T) {
	repo := campaignRepoStub{
		campaigns: []internalcampaigns.Campaign{{
			ID:            "5919090e",
			Name:          "Aquila launch 2026-09-17",
			Status:        internalcampaigns.StatusActive,
			ScenarioNames: []string{"web-console"},
		}},
		slots: []internalcampaigns.LaunchAssetSlot{{
			CampaignID:          "5919090e",
			CampaignName:        "Aquila launch 2026-09-17",
			Channel:             "blog",
			Format:              "long-form-essay",
			Capacity:            1,
			ReadyForReviewCount: 2,
			InProgressCount:     1,
			Readiness:           internalcampaigns.LaunchAssetReadinessReadyForReview,
		}},
	}

	snapshot, err := newStateSource(nil, repo).Load(context.Background())
	require.NoError(t, err)
	record := indexRecords(snapshot)["campaign:5919090e"]
	require.Equal(t, KindCampaign, record.Kind)
	require.Equal(t, "campaign/5919090e", record.FollowUp)
	require.Equal(t, internalcampaigns.StatusActive, record.Metadata["status"])
	require.Equal(t, []string{"web-console"}, record.Metadata["scenarios"])
	require.Equal(t, "Operator approval required", record.Metadata["next_action"])
	require.Contains(t, record.Snippet, "Campaign Aquila launch 2026-09-17 for web-console.")
	require.Contains(t, record.Snippet, "Next action: Operator approval required")
	slots, ok := record.Metadata["launch_assets"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, slots, 1)
	require.Equal(t, internalcampaigns.LaunchAssetReadinessReadyForReview, slots[0]["readiness"])
	require.Equal(t, "Operator approval required", slots[0]["next_action"])
}

func TestProductAliasResolutionIsCanonicalAndRetrievable(t *testing.T) {
	canonical, ok := resolveProductIdentity("Aquila")
	require.True(t, ok)
	require.Equal(t, canonicalProduct, canonical)
	canonical, ok = resolveProductIdentity("web-console")
	require.True(t, ok)
	require.Equal(t, canonicalProduct, canonical)

	// An unrelated product must never resolve through the Aquila alias map, so a
	// second product's records cannot be returned as canonical web-console state.
	_, ok = resolveProductIdentity("offer-desk")
	require.False(t, ok, "an unrelated product must not resolve to the canonical product")

	source := NewStoreSource(sourceDraftsStub{
		drafts: []internalartifacts.Draft{
			{
				ID: "draft-aquila", CampaignID: "camp-aquila", Body: "launch bundle",
				ScenarioName: "aquila", SKU: "web-console",
			},
			{
				ID: "draft-offer", CampaignID: "camp-offer", Body: "offer detail",
				ScenarioName: "offer-desk",
			},
		},
		revisions: map[string]internalartifacts.CurrentRevision{
			"draft-aquila": {DraftID: "draft-aquila"},
			"draft-offer":  {DraftID: "draft-offer"},
		},
	}, sourceLedgerStub{})
	service := NewService(source)

	aliasHits, err := service.Search(context.Background(), "aquila", 10)
	require.NoError(t, err)
	canonicalHits, err := service.Search(context.Background(), "web-console", 10)
	require.NoError(t, err)
	require.NotEmpty(t, aliasHits.Hits)
	require.Equal(t, hitIDs(aliasHits.Hits), hitIDs(canonicalHits.Hits))
	require.Equal(t, "draft:draft-aquila", aliasHits.Hits[0].ID)

	offerSnapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	offerRecord := indexRecords(offerSnapshot)["draft:draft-offer"]
	require.NotContains(t, offerRecord.Metadata, "product_canonical", "an unrelated product must not be aliased to web-console")
}

func TestOwnerlessCapabilityIsExplicitlyUnsupported(t *testing.T) {
	repo := capabilityRepoStub{caps: []internalcapabilities.Capability{{
		ID:                       "cap-short-form",
		Name:                     "Short-form promotional video",
		Medium:                   internalcapabilities.MediumVideo,
		Owner:                    "",
		DefinitionStatus:         internalcapabilities.DefinitionDocumented,
		ImplementationStatus:     internalcapabilities.ImplementationNotStarted,
		OperationalReadiness:     internalcapabilities.ReadinessUnavailable,
		OutputQuality:            internalcapabilities.QualityUnassessed,
		DistributionConnectivity: internalcapabilities.ConnectivityNotApplicable,
		NextAction:               "Name a producing owner.",
	}}}

	snapshot, err := newStateSource(repo, nil).Load(context.Background())
	require.NoError(t, err)
	record := indexRecords(snapshot)["capability:cap-short-form"]
	require.Equal(t, internalcapabilities.ReadinessUnavailable, record.Metadata["operational_readiness"])
	require.Equal(t, false, record.Metadata["supported"])
	limitations, ok := record.Metadata["readiness_limitations"].([]string)
	require.True(t, ok)
	require.Contains(t, limitations, unsupportedNoOwner)
	readinessLimitations, ok := record.Metadata["operational_readiness_limitations"].([]string)
	require.True(t, ok)
	require.Contains(t, strings.Join(readinessLimitations, " "), "no producing owner")
	require.NotContains(t, strings.Join(readinessLimitations, " "), "qualified-in-environment")
	qualification, ok := record.Metadata["latest_qualification"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, internalcapabilities.Unknown, qualification["observed_at"])
}

func TestStateProjectionIsStableAcrossUnchangedReads(t *testing.T) {
	updated := time.Date(2026, time.September, 12, 12, 0, 0, 0, time.UTC)
	caps := capabilityRepoStub{caps: []internalcapabilities.Capability{{
		ID:                       "cap-campaign-planning",
		Name:                     "Campaign and funnel planning",
		Medium:                   internalcapabilities.MediumGovernance,
		Owner:                    "content-desk",
		DefinitionStatus:         internalcapabilities.DefinitionDocumented,
		ImplementationStatus:     internalcapabilities.ImplementationImplemented,
		OperationalReadiness:     internalcapabilities.ReadinessQualified,
		OutputQuality:            internalcapabilities.QualityUnassessed,
		DistributionConnectivity: internalcapabilities.ConnectivityNotApplicable,
		NextAction:               "Keep campaign active.",
		CreatedAt:                updated,
		UpdatedAt:                updated,
	}}}
	campaigns := campaignRepoStub{
		campaigns: []internalcampaigns.Campaign{{ID: "camp-1", Name: "Aquila launch 2026-09-17", Status: internalcampaigns.StatusActive, ScenarioNames: []string{"web-console"}}},
		slots:     []internalcampaigns.LaunchAssetSlot{{CampaignID: "camp-1", Channel: "blog", Format: "essay", Capacity: 1, InProgressCount: 1, Readiness: internalcampaigns.LaunchAssetReadinessInProgress}},
	}
	source := newStateSource(caps, campaigns)

	first, err := source.Load(context.Background())
	require.NoError(t, err)
	second, err := source.Load(context.Background())
	require.NoError(t, err)
	// capability, campaign, work summary and campaign performance.
	require.Len(t, first.Records, 4)
	require.Equal(t, first.Generation, second.Generation)
	require.Equal(t, first.MaterializedAt, second.MaterializedAt)
	firstByID := indexRecords(first)
	for id, record := range indexRecords(second) {
		require.Equal(t, firstByID[id].Revision, record.Revision, "unchanged record %s must keep its revision", id)
	}
}

func TestStateReadFailureDegradesToEditorialCorpus(t *testing.T) {
	source := newStateSource(
		capabilityRepoStub{listErr: errors.New("capability store down")},
		campaignRepoStub{listErr: errors.New("campaign store down")},
	)

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	require.Empty(t, snapshot.Records)
	require.True(t, snapshot.MaterializedAt.IsZero())
}
