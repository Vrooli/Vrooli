package aisearch

import (
	"context"
	"errors"
	"testing"
	"time"

	internalartifacts "content-desk/internal/artifacts"
	internalcampaigns "content-desk/internal/campaigns"
	internalcapabilities "content-desk/internal/capabilities"
	internalclaims "content-desk/internal/claims"
	internalledger "content-desk/internal/ledger"

	"github.com/stretchr/testify/require"
)

type sourceClaimsStub struct {
	internalclaims.Library
	byDraft map[string][]internalclaims.Claim
	cited   map[string][]string
	all     []internalclaims.Claim
}

func (s sourceClaimsStub) List(context.Context) ([]internalclaims.Claim, error) {
	return s.all, nil
}

func (s sourceClaimsStub) ListForDraft(_ context.Context, draftID string) ([]internalclaims.Claim, error) {
	return s.byDraft[draftID], nil
}

func (s sourceClaimsStub) CitingDrafts(_ context.Context, id string) ([]string, error) {
	return s.cited[id], nil
}

type sourceDraftsStub struct {
	internalartifacts.Repository
	drafts      []internalartifacts.Draft
	revisions   map[string]internalartifacts.CurrentRevision
	revisionErr error
}

func (s sourceDraftsStub) List(context.Context) ([]internalartifacts.Draft, error) {
	return s.drafts, nil
}

func (s sourceDraftsStub) GetCurrentRevision(_ context.Context, id string) (internalartifacts.CurrentRevision, error) {
	if s.revisionErr != nil {
		return internalartifacts.CurrentRevision{}, s.revisionErr
	}
	return s.revisions[id], nil
}

type sourceLedgerStub struct {
	internalledger.Repository
	records   []internalledger.PublishRecord
	metrics   map[string]internalledger.DraftMetricProjection
	metricErr error
}

func (s sourceLedgerStub) ListPublishHistory(context.Context, int) ([]internalledger.PublishRecord, error) {
	return s.records, nil
}

func (s sourceLedgerStub) DraftMetricProjection(_ context.Context, draftID string, _ time.Duration) (internalledger.DraftMetricProjection, error) {
	if s.metricErr != nil {
		return internalledger.DraftMetricProjection{}, s.metricErr
	}
	if projection, ok := s.metrics[draftID]; ok {
		return projection, nil
	}
	return internalledger.DraftMetricProjection{DraftID: draftID}, nil
}

func TestLoadProjectsDraftsWithAuthorityAndPublishHistory(t *testing.T) {
	older := time.Date(2026, time.January, 1, 8, 0, 0, 0, time.UTC)
	newest := time.Date(2026, time.March, 3, 9, 15, 0, 0, time.UTC)
	drafts := sourceDraftsStub{
		drafts: []internalartifacts.Draft{{
			ID: "draft-1", CampaignID: "camp-1", PostTypeID: "post-1", Body: "body text",
			Channel: "linkedin", Format: "long", Lane: "lane-a", SKU: "sku-1", ScenarioName: "aquila",
			Status: internalartifacts.DraftApproved,
		}},
		revisions: map[string]internalartifacts.CurrentRevision{
			"draft-1": {
				DraftID: "draft-1", RevisionID: "rev-1", ApprovalActorKind: "operator",
				ApprovalCapacity: "operator", ReviewRunID: "run-1",
			},
		},
	}
	ledger := sourceLedgerStub{records: []internalledger.PublishRecord{
		{ID: "pub-old", DraftID: "draft-1", Channel: "linkedin", Audience: "devs", PublishedURL: "https://example.test/a", PublishedAt: older},
		{ID: "pub-new", PublishedAt: newest},
	}}
	source := NewStoreSource(drafts, ledger)

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, 3, snapshot.IndexedCount())
	require.NotEmpty(t, snapshot.Generation)
	require.Equal(t, newest, snapshot.MaterializedAt, "materialized time is the newest observed mutation, not the read time")

	byID := make(map[string]Record)
	for _, r := range snapshot.Records {
		require.Equal(t, VisibilityOperator, r.Visibility)
		require.NotEmpty(t, r.Revision)
		require.NotEmpty(t, r.FollowUp)
		byID[r.ID] = r
	}

	draft := byID["draft:draft-1"]
	require.Equal(t, KindDraft, draft.Kind)
	require.False(t, draft.Historical)
	// The title must carry the product identity an agent asks for, not an
	// opaque UUID; the durable id stays in the record id and follow-up.
	require.Equal(t, "Aquila (web-console) linkedin long draft", draft.Title)
	require.NotEqual(t, "draft-1", draft.Title)
	require.Equal(t, "draft-1", draft.Metadata["draft_id"])
	require.Equal(t, "draft/draft-1", draft.FollowUp)
	require.Equal(t, true, draft.Metadata["approved"])
	require.Equal(t, true, draft.Metadata["reviewed"])
	require.Equal(t, "rev-1", draft.Metadata["revision_id"])
	require.Equal(t, "operator", draft.Metadata["approval_actor_kind"])
	// The cross-provider reranker reads the title and snippet, so they must
	// carry the semantic state in natural language rather than a machine tag
	// line or a bare id.
	require.Contains(t, draft.Snippet, "approved Aquila (web-console) linkedin long draft.")
	require.Contains(t, draft.Snippet, "Format: long.")
	require.Contains(t, draft.Snippet, "Channel: linkedin.")
	require.Contains(t, draft.Snippet, "Approved to publish.")

	published := byID["publish:pub-old"]
	require.Equal(t, KindPublishRecord, published.Kind)
	require.True(t, published.Historical)
	require.Equal(t, "draft-1", published.Title)
	require.Equal(t, "https://example.test/a", published.Metadata["published_url"])
	require.Contains(t, published.Snippet, "Published linkedin post: https://example.test/a.")

	// A publish record with no draft id falls back to its own id as title.
	require.Equal(t, "pub-new", byID["publish:pub-new"].Title)

	again, err := source.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, snapshot.Generation, again.Generation, "unchanged corpus must keep its generation")
	require.Equal(t, snapshot.MaterializedAt, again.MaterializedAt)
}

func TestLoadFailsWhenCurrentRevisionCannotBeResolved(t *testing.T) {
	source := NewStoreSource(
		sourceDraftsStub{
			drafts:      []internalartifacts.Draft{{ID: "draft-1", Status: internalartifacts.DraftApproved}},
			revisionErr: errors.New("authority unavailable"),
		},
		sourceLedgerStub{},
	)

	_, err := source.Load(context.Background())
	require.Error(t, err)
	require.ErrorContains(t, err, "authority unavailable")
}

func TestEmptySourceIsStableAndTruthful(t *testing.T) {
	source := NewStoreSource(sourceDraftsStub{}, sourceLedgerStub{})

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	require.Empty(t, snapshot.Records)
	require.Zero(t, snapshot.IndexedCount())
	require.True(t, snapshot.MaterializedAt.IsZero(), "no source data means no claimed materialization")

	again, err := source.Load(context.Background())
	require.NoError(t, err)
	require.Equal(t, snapshot.Generation, again.Generation)
}

func TestRevisionChangesWhenContentChanges(t *testing.T) {
	draft := func(body string) sourceDraftsStub {
		return sourceDraftsStub{
			drafts:    []internalartifacts.Draft{{ID: "draft-1", Body: body}},
			revisions: map[string]internalartifacts.CurrentRevision{"draft-1": {DraftID: "draft-1"}},
		}
	}
	first, err := NewStoreSource(draft("first body"), sourceLedgerStub{}).Load(context.Background())
	require.NoError(t, err)
	second, err := NewStoreSource(draft("second body"), sourceLedgerStub{}).Load(context.Background())
	require.NoError(t, err)

	require.NotEqual(t, first.Records[0].Revision, second.Records[0].Revision)
	require.NotEqual(t, first.Generation, second.Generation, "content movement must move the generation")
}

func TestLoadProjectsLaunchCampaignQualifiedClaims(t *testing.T) {
	drafts := sourceDraftsStub{
		drafts: []internalartifacts.Draft{
			{ID: "draft-aquila", CampaignID: "camp-aquila"},
			{ID: "draft-fixture", CampaignID: "camp-fixture"},
		},
		revisions: map[string]internalartifacts.CurrentRevision{
			"draft-aquila":  {DraftID: "draft-aquila"},
			"draft-fixture": {DraftID: "draft-fixture"},
		},
	}
	source := NewStoreSource(drafts, sourceLedgerStub{}).
		WithCampaigns(campaignRepoStub{
			campaigns: []internalcampaigns.Campaign{
				{ID: "camp-aquila", ScenarioNames: []string{"web-console"}},
				{ID: "camp-fixture"},
			},
			slots: []internalcampaigns.LaunchAssetSlot{{CampaignID: "camp-aquila"}},
		}).
		WithClaims(sourceClaimsStub{
			byDraft: map[string][]internalclaims.Claim{
				"draft-aquila": {
					{ID: "claim-supported", Statement: "Aquila is the first ranked marketed deliverable", Kind: internalclaims.KindCapability, VerificationStatus: internalclaims.StateAsserted, Qualification: internalclaims.StateSupported},
					{ID: "claim-asserted", Statement: "This claim still lacks evidence", Kind: internalclaims.KindCapability, VerificationStatus: internalclaims.StateAsserted},
				},
				"draft-fixture": {
					{ID: "claim-fixture", Statement: "An unrelated fixture claim", Kind: internalclaims.KindCapability, VerificationStatus: internalclaims.StateAsserted, Qualification: internalclaims.StateSupported},
				},
			},
			cited: map[string][]string{"claim-supported": {"draft-aquila"}},
			all: []internalclaims.Claim{
				{ID: "claim-supported", Statement: "Aquila is the first ranked marketed deliverable", Kind: internalclaims.KindCapability, VerificationStatus: internalclaims.StateAsserted, Qualification: internalclaims.StateSupported},
				{ID: "claim-asserted", Statement: "This claim still lacks evidence", Kind: internalclaims.KindCapability, VerificationStatus: internalclaims.StateAsserted},
				{ID: "claim-fixture", Statement: "An unrelated fixture claim", Kind: internalclaims.KindCapability, VerificationStatus: internalclaims.StateAsserted, Qualification: internalclaims.StateSupported},
			},
		})

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)

	byID := make(map[string]Record)
	for _, r := range snapshot.Records {
		byID[r.ID] = r
	}
	_, fixturePresent := byID["claim:claim-fixture"]
	require.False(t, fixturePresent, "claims outside the canonical product's launch campaigns must not pollute the corpus")
	_, assertedPresent := byID["claim:claim-asserted"]
	require.False(t, assertedPresent, "an unqualified claim stays in the library instead of surfacing as a current answer")

	supported := byID["claim:claim-supported"]
	require.Equal(t, KindClaim, supported.Kind)
	require.Equal(t, "Aquila is the first ranked marketed deliverable", supported.Title)
	require.Equal(t, "claim/claim-supported", supported.FollowUp)
	require.Contains(t, supported.Snippet, "supported capability claim.")
	require.Contains(t, supported.Snippet, "Verification: asserted.")
	require.Contains(t, supported.Snippet, "Cited by 1 draft(s).")
	require.Equal(t, true, supported.Metadata["evidence_backed"])
	require.Equal(t, []string{"draft-aquila"}, supported.Metadata["cited_by_draft_ids"])

	summary := byID["claim-summary:launch"]
	require.Equal(t, KindClaimSummary, summary.Kind)
	require.Equal(t, "claims", summary.FollowUp)
	require.Contains(t, summary.Snippet, "1 evidence-backed claim(s) safe to use")
	require.Contains(t, summary.Snippet, "0 awaiting review")
	require.Contains(t, summary.Snippet, "1 unqualified launch-cited claim(s)")
	require.Contains(t, summary.Snippet, "Aquila is the first ranked marketed deliverable")
	require.Contains(t, summary.Snippet, "1 of 3 claim(s) not verified")
	require.Equal(t, 1, summary.Metadata["safe_to_use_count"])
	require.Equal(t, 1, summary.Metadata["unqualified_launch_count"])
	require.Equal(t, 1, summary.Metadata["library_unverified_count"])
}

func TestLaunchClaimSummaryNamesAwaitingReviewAndGaps(t *testing.T) {
	drafts := sourceDraftsStub{
		drafts:    []internalartifacts.Draft{{ID: "draft-aquila", CampaignID: "camp-aquila"}},
		revisions: map[string]internalartifacts.CurrentRevision{"draft-aquila": {DraftID: "draft-aquila"}},
	}
	supported := internalclaims.Claim{ID: "c-supported", Statement: "A supported claim", Kind: internalclaims.KindCapability, Qualification: internalclaims.StateSupported}
	pending := internalclaims.Claim{ID: "c-pending", Statement: "A pending claim", Kind: internalclaims.KindCapability, Qualification: internalclaims.StateCapturedReviewPending}
	asserted := internalclaims.Claim{ID: "c-asserted", Statement: "An unresolved claim", Kind: internalclaims.KindCapability}
	source := NewStoreSource(drafts, sourceLedgerStub{}).
		WithCampaigns(campaignRepoStub{
			campaigns: []internalcampaigns.Campaign{{ID: "camp-aquila", ScenarioNames: []string{"web-console"}}},
			slots:     []internalcampaigns.LaunchAssetSlot{{CampaignID: "camp-aquila"}},
		}).
		WithClaims(sourceClaimsStub{
			byDraft: map[string][]internalclaims.Claim{"draft-aquila": {supported, pending, asserted}},
			all:     []internalclaims.Claim{supported, pending, asserted},
		})

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	byID := make(map[string]Record)
	for _, r := range snapshot.Records {
		byID[r.ID] = r
	}
	summary := byID["claim-summary:launch"]
	require.Contains(t, summary.Snippet, "1 evidence-backed claim(s) safe to use")
	require.Contains(t, summary.Snippet, "1 awaiting review")
	require.Contains(t, summary.Snippet, "1 launch-cited claim(s) lack an evidence verdict")
	require.Contains(t, summary.Snippet, "[supported] A supported claim")
	require.Contains(t, summary.Snippet, "[captured-review-pending] A pending claim")
	require.NotContains(t, summary.Snippet, "An unresolved claim", "an unqualified claim is a gap, not a safe-to-use statement")
	require.Equal(t, []string{"c-supported"}, summary.Metadata["safe_claim_ids"])
	require.Equal(t, []string{"c-pending"}, summary.Metadata["awaiting_review_claim_ids"])
}

func TestWorkSummaryClassifiesLaunchAndOngoingWork(t *testing.T) {
	drafts := sourceDraftsStub{
		drafts: []internalartifacts.Draft{
			{ID: "d-drafted", CampaignID: "camp-aquila", Status: internalartifacts.DraftDrafted},
			{ID: "d-approved", CampaignID: "camp-aquila", Status: internalartifacts.DraftApproved},
		},
		revisions: map[string]internalartifacts.CurrentRevision{
			"d-drafted":  {DraftID: "d-drafted"},
			"d-approved": {DraftID: "d-approved"},
		},
	}
	source := NewStoreSource(drafts, sourceLedgerStub{}).
		WithCampaigns(campaignRepoStub{
			campaigns: []internalcampaigns.Campaign{{
				ID: "camp-aquila", Name: "Aquila launch 2026-09-17",
				Status: internalcampaigns.StatusActive, ScenarioNames: []string{"web-console"},
			}},
			slots: []internalcampaigns.LaunchAssetSlot{
				{CampaignID: "camp-aquila", Channel: "blog", Format: "long-form-essay", Readiness: internalcampaigns.LaunchAssetReadinessReadyForReview},
				{CampaignID: "camp-aquila", Channel: "youtube", Format: "short-form-video", Readiness: internalcampaigns.LaunchAssetReadinessEmpty},
			},
		}).
		WithCapabilityCatalog(capabilityRepoStub{caps: []internalcapabilities.Capability{{
			ID: "cap-truth", Name: "Product truth and positioning", Owner: "offer-desk", Priority: 1,
			NextAction: "Verify the asserted claims before launch copy is approved.",
		}}}, nil, nil)

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	summary := indexRecords(snapshot)["work-summary:launch"]
	require.Equal(t, KindWorkSummary, summary.Kind)
	require.Equal(t, "capabilities", summary.FollowUp)
	require.Contains(t, summary.Snippet, "Marketing work remaining for web-console.")
	require.Contains(t, summary.Snippet, "Before launch: 2 open launch draft(s).")
	require.Contains(t, summary.Snippet, "Start checking and review")
	require.Contains(t, summary.Snippet, "Submit for release")
	require.Contains(t, summary.Snippet, "Binding launch constraint: Begin drafting.")
	require.Contains(t, summary.Snippet, "Highest-priority ongoing work: Product truth and positioning")
	require.Equal(t, 2, summary.Metadata["launch_draft_count"])
	require.Equal(t, 1, summary.Metadata["ongoing_capability_count"])
	require.Equal(t, "Begin drafting", summary.Metadata["binding_launch_action"])
	require.Equal(t, []string{"camp-aquila"}, summary.Metadata["launch_campaign_ids"])
}

func TestCampaignPerformanceStatesUnavailableMetrics(t *testing.T) {
	drafts := sourceDraftsStub{
		drafts:    []internalartifacts.Draft{{ID: "d-drafted", CampaignID: "camp-aquila", Status: internalartifacts.DraftDrafted}},
		revisions: map[string]internalartifacts.CurrentRevision{"d-drafted": {DraftID: "d-drafted"}},
	}
	source := NewStoreSource(drafts, sourceLedgerStub{}).
		WithCampaigns(campaignRepoStub{
			campaigns: []internalcampaigns.Campaign{{
				ID: "camp-aquila", Name: "Aquila launch 2026-09-17",
				Status: internalcampaigns.StatusActive, ScenarioNames: []string{"web-console"},
			}},
			slots: []internalcampaigns.LaunchAssetSlot{{CampaignID: "camp-aquila"}},
		})

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	performance := indexRecords(snapshot)["campaign-performance:launch"]
	require.Equal(t, KindCampaignPerformance, performance.Kind)
	require.Equal(t, "metrics", performance.FollowUp)
	require.Contains(t, performance.Snippet, "Campaign performance for web-console is unavailable")
	require.Contains(t, performance.Snippet, "unknown, not zero")
	require.Equal(t, false, performance.Metadata["metrics_available"])
	require.Equal(t, 0, performance.Metadata["published_release_count"])
}

func TestCampaignPerformanceReportsMeasuredSamples(t *testing.T) {
	observed := time.Date(2026, time.September, 13, 0, 0, 0, 0, time.UTC)
	drafts := sourceDraftsStub{
		drafts:    []internalartifacts.Draft{{ID: "d-drafted", CampaignID: "camp-aquila", Status: internalartifacts.DraftDrafted}},
		revisions: map[string]internalartifacts.CurrentRevision{"d-drafted": {DraftID: "d-drafted"}},
	}
	ledger := sourceLedgerStub{
		metrics: map[string]internalledger.DraftMetricProjection{
			"d-drafted": {
				DraftID: "d-drafted", HasMeasurements: true,
				Readings: []internalledger.MetricReading{{
					Metric: "impressions", Value: 42, State: internalledger.MetricStateMeasured,
					SampleCount: 1, LastObservedAt: observed,
				}},
			},
		},
	}
	source := NewStoreSource(drafts, ledger).
		WithCampaigns(campaignRepoStub{
			campaigns: []internalcampaigns.Campaign{{
				ID: "camp-aquila", Name: "Aquila launch 2026-09-17",
				Status: internalcampaigns.StatusActive, ScenarioNames: []string{"web-console"},
			}},
			slots: []internalcampaigns.LaunchAssetSlot{{CampaignID: "camp-aquila"}},
		})

	snapshot, err := source.Load(context.Background())
	require.NoError(t, err)
	performance := indexRecords(snapshot)["campaign-performance:launch"]
	require.Equal(t, true, performance.Metadata["metrics_available"])
	require.Contains(t, performance.Snippet, "impressions=42 (measured)")
	require.Equal(t, 1, performance.Metadata["measured_count"])
	require.Equal(t, 1, performance.Metadata["metric_count"])
}
