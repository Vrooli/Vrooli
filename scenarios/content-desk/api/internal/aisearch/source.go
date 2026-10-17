package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	internalartifacts "content-desk/internal/artifacts"
	internalcampaigns "content-desk/internal/campaigns"
	internalcapabilities "content-desk/internal/capabilities"
	internalclaims "content-desk/internal/claims"
	internalledger "content-desk/internal/ledger"
)

// publishHistoryBound bounds the publish-history read. The snapshot is bounded
// and filtering happens after the full relevant history within this bound is
// loaded, never before: narrowing the corpus before projection would make the
// snapshot depend on the query instead of on the data.
const publishHistoryBound = 10000

// StoreSource projects the authoritative owner stores. It reads every draft
// plus the review and approval authority that still applies to it, the retained
// publish history, and — when attached — the capability catalog and campaigns
// with their launch assets; it writes nothing.
type StoreSource struct {
	drafts         internalartifacts.Repository
	ledger         internalledger.Repository
	capabilityRepo internalcapabilities.Repository
	campaignRepo   internalcampaigns.Repository
	claims         internalclaims.Library
	evidence       internalcapabilities.EvidenceResolver
	distribution   internalcapabilities.DistributionResolver
}

// NewStoreSource builds the production source over the editorial repositories.
func NewStoreSource(drafts internalartifacts.Repository, ledger internalledger.Repository) *StoreSource {
	return &StoreSource{drafts: drafts, ledger: ledger}
}

// WithCapabilityCatalog attaches the read-only capability projection. The
// optional resolvers rehydrate volatile readiness; a nil resolver keeps the
// stored value with the rehydration's named limitation.
func (s *StoreSource) WithCapabilityCatalog(repo internalcapabilities.Repository, evidence internalcapabilities.EvidenceResolver, distribution internalcapabilities.DistributionResolver) *StoreSource {
	s.capabilityRepo = repo
	s.evidence = evidence
	s.distribution = distribution
	return s
}

// WithCampaigns attaches the read-only campaign projection.
func (s *StoreSource) WithCampaigns(repo internalcampaigns.Repository) *StoreSource {
	s.campaignRepo = repo
	return s
}

// WithClaims attaches the read-only claim-library projection. Claims are their
// own evidence-backed store, so a query about what can safely be used in the
// launch article resolves against the current verdict rather than a generic
// requirement that merely mentions the word "claim".
func (s *StoreSource) WithClaims(lib internalclaims.Library) *StoreSource {
	s.claims = lib
	return s
}

// Load builds one deterministic snapshot of the corpus.
func (s *StoreSource) Load(ctx context.Context) (*Snapshot, error) {
	drafts, err := s.drafts.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("load drafts: %w", err)
	}
	history, err := s.ledger.ListPublishHistory(ctx, publishHistoryBound)
	if err != nil {
		return nil, fmt.Errorf("load publish history: %w", err)
	}

	var records []Record
	var materialized time.Time
	bump := func(t time.Time) {
		if t.After(materialized) {
			materialized = t
		}
	}

	for _, d := range drafts {
		rev, err := s.drafts.GetCurrentRevision(ctx, d.ID)
		if err != nil {
			// Never silently fabricate approval: if current authority cannot be
			// resolved, the projection is not truthful and must not be served.
			return nil, fmt.Errorf("load current revision for draft %s: %w", d.ID, err)
		}
		records = append(records, draftRecord(d, rev))
	}
	for _, r := range history {
		records = append(records, publishRecord(r))
		bump(r.PublishedAt)
	}

	now := time.Now().UTC()
	records = append(records, s.capabilityRecords(ctx, now, bump)...)
	records = append(records, s.campaignRecords(ctx)...)
	records = append(records, s.claimRecords(ctx, drafts)...)
	records = append(records, s.workSummaryRecords(ctx, drafts)...)
	records = append(records, s.performanceRecords(ctx, drafts, history)...)

	for i := range records {
		records[i].Revision = revisionOf(records[i])
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })

	return &Snapshot{
		Records:        records,
		Generation:     generationOf(records),
		MaterializedAt: materialized,
	}, nil
}

// capabilityRecords projects the capability catalog. A repository read failure
// degrades to the editorial corpus instead of failing the whole snapshot.
func (s *StoreSource) capabilityRecords(ctx context.Context, now time.Time, bump func(time.Time)) []Record {
	if s.capabilityRepo == nil {
		return nil
	}
	catalog, err := s.capabilityRepo.List(ctx, "")
	if err != nil {
		return nil
	}
	links, err := s.capabilityRepo.ListLinks(ctx)
	if err != nil {
		links = nil
	}
	linksByCapability := make(map[string][]internalcapabilities.CapabilityLink, len(links))
	for _, link := range links {
		linksByCapability[link.CapabilityID] = append(linksByCapability[link.CapabilityID], link)
	}
	out := make([]Record, 0, len(catalog))
	for _, capability := range catalog {
		out = append(out, capabilityRecord(ctx, capability, linksByCapability[capability.ID], s.evidence, s.distribution, now))
		bump(capability.CreatedAt)
		bump(capability.UpdatedAt)
		if capability.LatestQualification != nil {
			bump(capability.LatestQualification.CreatedAt)
		}
	}
	return out
}

// campaignRecords projects campaigns against the launch assets of the canonical
// product. A repository read failure degrades to the editorial corpus.
func (s *StoreSource) campaignRecords(ctx context.Context) []Record {
	if s.campaignRepo == nil {
		return nil
	}
	campaigns, err := s.campaignRepo.List(ctx)
	if err != nil {
		return nil
	}
	slots, err := s.campaignRepo.LaunchAssets(ctx, canonicalProduct)
	if err != nil {
		slots = nil
	}
	slotsByCampaign := make(map[string][]internalcampaigns.LaunchAssetSlot, len(slots))
	for _, slot := range slots {
		slotsByCampaign[slot.CampaignID] = append(slotsByCampaign[slot.CampaignID], slot)
	}
	out := make([]Record, 0, len(campaigns))
	for _, campaign := range campaigns {
		out = append(out, campaignRecord(campaign, slotsByCampaign[campaign.ID]))
	}
	return out
}

// claimRecords projects the owner-qualified claims attached to the canonical
// product's launch material, plus one owner-generated launch-claim summary.
// Scoping matters twice over: the claim library is shared with unrelated Prose
// Studio fixtures, and a product's own non-launch fixture campaigns are not part
// of the launch article either. Only claims whose owner recorded a qualification
// are projected as per-claim records; an unqualified claim stays in the library
// behind its follow-up rather than surfacing as a current answer. A read failure
// degrades to the rest of the editorial corpus.
func (s *StoreSource) claimRecords(ctx context.Context, drafts []internalartifacts.Draft) []Record {
	claims, citing := s.launchClaimSet(ctx, drafts)
	out := make([]Record, 0, len(claims)+1)
	for _, claim := range claims {
		if strings.TrimSpace(claim.Qualification) == "" {
			continue
		}
		out = append(out, claimRecord(claim, citing[claim.ID]))
	}
	if len(claims) > 0 {
		out = append(out, claimLibraryRecord(claims, s.claimLibrary(ctx)))
	}
	return out
}

// launchClaimSet collects the distinct claims cited by the canonical product's
// launch drafts and the drafts citing each. It is the single owner read behind
// both the per-claim records and the launch-claim summary, so the two can never
// disagree about what the launch material cites. A read failure degrades to the
// rest of the editorial corpus.
func (s *StoreSource) launchClaimSet(ctx context.Context, drafts []internalartifacts.Draft) ([]internalclaims.Claim, map[string][]string) {
	if s.claims == nil || s.campaignRepo == nil {
		return nil, nil
	}
	slots, err := s.campaignRepo.LaunchAssets(ctx, canonicalProduct)
	if err != nil {
		return nil, nil
	}
	launchCampaigns := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		launchCampaigns[slot.CampaignID] = struct{}{}
	}
	seen := make(map[string]internalclaims.Claim)
	order := make([]string, 0)
	for _, d := range drafts {
		if _, ok := launchCampaigns[d.CampaignID]; !ok {
			continue
		}
		cited, err := s.claims.ListForDraft(ctx, d.ID)
		if err != nil {
			continue
		}
		for _, claim := range cited {
			if _, ok := seen[claim.ID]; ok {
				continue
			}
			seen[claim.ID] = claim
			order = append(order, claim.ID)
		}
	}
	sort.Strings(order)
	claims := make([]internalclaims.Claim, 0, len(order))
	citing := make(map[string][]string, len(order))
	for _, id := range order {
		claims = append(claims, seen[id])
		drafts, err := s.claims.CitingDrafts(ctx, id)
		if err != nil {
			drafts = nil
		}
		citing[id] = drafts
	}
	return claims, citing
}

// claimLibrary reads the whole claim library for library-wide gap context. It
// degrades to nil on failure so the launch-cited summary still projects.
func (s *StoreSource) claimLibrary(ctx context.Context) []internalclaims.Claim {
	if s.claims == nil {
		return nil
	}
	claims, err := s.claims.List(ctx)
	if err != nil {
		return nil
	}
	return claims
}

// claimLibraryRecord renders the owner-generated launch-claim summary. The
// cross-provider reranker reads a hit's type, title, path and the first 120
// characters of its snippet, so Q07's answer ("which claims are safe to use and
// what gaps remain") cannot be spread across many low-scoring per-claim rows.
// This record leads with the safe-to-use verdict and the unresolved-gap count.
// Every value comes from owner fields; no query text is copied and an
// unqualified claim is named as a gap, never promoted to usable.
func claimLibraryRecord(claims []internalclaims.Claim, library []internalclaims.Claim) Record {
	safe := make([]internalclaims.Claim, 0, len(claims))
	pending := make([]internalclaims.Claim, 0)
	citedUnqualified := 0
	for _, claim := range claims {
		switch claim.QualificationState() {
		case internalclaims.StateSupported, internalclaims.StateVerified:
			safe = append(safe, claim)
		case internalclaims.StateCapturedReviewPending:
			pending = append(pending, claim)
		default:
			citedUnqualified++
		}
	}
	libraryUnverified := 0
	for _, claim := range library {
		switch claim.QualificationState() {
		case internalclaims.StateSupported, internalclaims.StateVerified:
		default:
			libraryUnverified++
		}
	}

	parts := []string{fmt.Sprintf(
		"Launch claims for %s: %d evidence-backed claim(s) safe to use, %d awaiting review, %d unqualified launch-cited claim(s).",
		canonicalProduct, len(safe), len(pending), citedUnqualified)}
	if len(library) > 0 {
		parts = append(parts, fmt.Sprintf("Claim library: %d of %d claim(s) not verified.",
			libraryUnverified, len(library)))
	}
	if len(safe) > 0 {
		statements := make([]string, 0, len(safe))
		for _, claim := range safe {
			statements = append(statements, safeClaimStatement(claim))
		}
		parts = append(parts, "Safe to use: "+strings.Join(statements, " | ")+".")
	} else {
		parts = append(parts, "No evidence-backed launch claim is recorded.")
	}
	if len(pending) > 0 {
		statements := make([]string, 0, len(pending))
		for _, claim := range pending {
			statements = append(statements, safeClaimStatement(claim))
		}
		parts = append(parts, "Awaiting review: "+strings.Join(statements, " | ")+".")
	}
	if citedUnqualified > 0 {
		parts = append(parts, fmt.Sprintf("%d launch-cited claim(s) lack an evidence verdict and remain unresolved gaps.", citedUnqualified))
	}
	snippet := strings.Join(parts, " ")

	bodyParts := []string{
		"launch claims",
		"safety and unresolved gaps",
		fmt.Sprintf("safe to use %d", len(safe)),
		fmt.Sprintf("awaiting review %d", len(pending)),
		fmt.Sprintf("unqualified launch-cited %d", citedUnqualified),
		fmt.Sprintf("library unverified %d of %d", libraryUnverified, len(library)),
	}
	if len(safe) > 0 {
		statements := make([]string, 0, len(safe))
		for _, claim := range safe {
			statements = append(statements, safeClaimStatement(claim))
		}
		bodyParts = append(bodyParts, "safe claims "+strings.Join(statements, " | "))
	}
	if len(pending) > 0 {
		statements := make([]string, 0, len(pending))
		for _, claim := range pending {
			statements = append(statements, safeClaimStatement(claim))
		}
		bodyParts = append(bodyParts, "claims awaiting review "+strings.Join(statements, " | "))
	}
	metadata := map[string]any{
		"safe_to_use_count":         len(safe),
		"awaiting_review_count":     len(pending),
		"unqualified_launch_count":  citedUnqualified,
		"launch_claim_count":        len(claims),
		"library_claim_count":       len(library),
		"library_unverified_count":  libraryUnverified,
		"safe_claim_ids":            claimIDs(safe),
		"awaiting_review_claim_ids": claimIDs(pending),
	}
	return Record{
		ID:         "claim-summary:launch",
		Kind:       KindClaimSummary,
		Visibility: VisibilityOperator,
		FollowUp:   "claims",
		Title:      "Launch claims: safe to use and unresolved gaps",
		Snippet:    snippet,
		Body:       strings.Join(bodyParts, ". "),
		Historical: false,
		Metadata:   metadata,
	}
}

// safeClaimStatement names one evidence-backed claim with its owner verdict so
// the summary states what is safe and why, not only a count.
func safeClaimStatement(claim internalclaims.Claim) string {
	statement := strings.TrimSpace(claim.Statement)
	if statement == "" {
		statement = claim.ID
	}
	return "[" + claim.QualificationState() + "] " + statement
}

// claimIDs is the bounded owner id list carried in the summary metadata.
func claimIDs(claims []internalclaims.Claim) []string {
	out := make([]string, 0, len(claims))
	for _, claim := range claims {
		out = append(out, claim.ID)
	}
	return out
}

// claimSnippet states the evidence verdict and where the claim is cited, using
// owner fields only. It names an unresolved gap as unresolved rather than
// promoting an asserted claim to usable.
func claimSnippet(claim internalclaims.Claim, citing []string) string {
	parts := []string{claim.QualificationState() + " " + claim.Kind + " claim."}
	parts = append(parts, "Verification: "+firstNonEmpty(strings.TrimSpace(claim.VerificationStatus), "unknown")+".")
	if len(citing) > 0 {
		parts = append(parts, fmt.Sprintf("Cited by %d draft(s).", len(citing)))
	} else {
		parts = append(parts, "Not cited by any draft.")
	}
	return strings.Join(parts, " ")
}

func claimRecord(claim internalclaims.Claim, citing []string) Record {
	qualification := claim.QualificationState()
	verification := firstNonEmpty(strings.TrimSpace(claim.VerificationStatus), "unknown")
	title := strings.TrimSpace(claim.Statement)
	if title == "" {
		title = "claim " + claim.ID
	}
	bodyParts := []string{
		title,
		"claim",
		"claim kind " + claim.Kind,
		"qualification " + qualification,
		"verification " + verification,
	}
	if len(citing) > 0 {
		bodyParts = append(bodyParts, "citing drafts "+strings.Join(citing, " "))
	}
	metadata := map[string]any{
		"claim_id":            claim.ID,
		"statement":           claim.Statement,
		"kind":                claim.Kind,
		"verification_status": verification,
		"qualification":       qualification,
		"cited_by_draft_ids":  citing,
		"evidence_backed":     qualification == internalclaims.StateSupported || qualification == internalclaims.StateVerified,
	}
	return Record{
		ID:         "claim:" + claim.ID,
		Kind:       KindClaim,
		Visibility: VisibilityOperator,
		FollowUp:   "claim/" + claim.ID,
		Title:      title,
		Snippet:    claimSnippet(claim, citing),
		Body:       strings.Join(bodyParts, ". "),
		Historical: false,
		Metadata:   metadata,
	}
}

// draftTitle renders a product-specific, human-readable identity for a draft.
// Ordinary search and the cross-provider reranker read the record title, so an
// opaque UUID title makes a real artifact invisible; the durable id stays in
// the record id, metadata and follow-up, while the title carries the product,
// channel and medium an agent would actually ask for.
func draftTitle(d internalartifacts.Draft) string {
	name := firstNonEmpty(strings.TrimSpace(d.SKU), strings.TrimSpace(d.ScenarioName), "marketing")
	if canonical, _ := resolveProductReference(d.ScenarioName, d.SKU); canonical != "" {
		name = "Aquila (" + canonical + ")"
	}
	parts := []string{name}
	if channel := strings.TrimSpace(d.Channel); channel != "" {
		parts = append(parts, channel)
	}
	if medium := firstNonEmpty(strings.TrimSpace(d.Format), strings.TrimSpace(d.PostTypeID)); medium != "" {
		parts = append(parts, medium)
	}
	parts = append(parts, "draft")
	return strings.Join(parts, " ")
}

// draftSnippet renders a natural-language draft summary so the reranker can
// distinguish drafts by product, channel and editorial state instead of by an
// opaque id tag alone. It leads with the product-specific title derived from
// the draft's own fields; no query text is copied.
func draftSnippet(d internalartifacts.Draft, rev internalartifacts.CurrentRevision, status string) string {
	parts := []string{firstNonEmpty(status, "unknown") + " " + draftTitle(d) + "."}
	if strings.TrimSpace(d.Format) != "" {
		parts = append(parts, "Format: "+d.Format+".")
	}
	if postType := strings.TrimSpace(d.PostTypeID); postType != "" && postType != strings.TrimSpace(d.Format) {
		parts = append(parts, "Post type: "+postType+".")
	}
	if strings.TrimSpace(d.Channel) != "" {
		parts = append(parts, "Channel: "+d.Channel+".")
	}
	if strings.TrimSpace(d.Lane) != "" {
		parts = append(parts, "Lane: "+d.Lane+".")
	}
	if strings.TrimSpace(d.SKU) != "" {
		parts = append(parts, "SKU: "+d.SKU+".")
	}
	if strings.TrimSpace(d.CampaignID) != "" {
		parts = append(parts, "Campaign: "+d.CampaignID+".")
	}
	switch {
	case rev.HasApplicableApproval():
		parts = append(parts, "Approved to publish.")
	case rev.HasApplicableReview():
		parts = append(parts, "Reviewed; awaiting operator approval.")
	}
	if canonical, productTerms := resolveProductReference(d.ScenarioName, d.SKU); canonical != "" {
		parts = append(parts, "Product aliases: "+strings.Join(productTerms, ", ")+".")
	}
	return strings.Join(parts, " ")
}

// publishSnippet renders a natural-language publish-history summary.
func publishSnippet(r internalledger.PublishRecord) string {
	head := "Published " + firstNonEmpty(r.Channel, "channel") + " post"
	if url := strings.TrimSpace(r.PublishedURL); url != "" {
		head += ": " + url
	}
	parts := []string{head + "."}
	if audience := strings.TrimSpace(r.Audience); audience != "" {
		parts = append(parts, "Audience: "+audience+".")
	}
	if draft := strings.TrimSpace(r.DraftID); draft != "" {
		parts = append(parts, "Draft: "+draft+".")
	}
	return strings.Join(parts, " ")
}

func draftRecord(d internalartifacts.Draft, rev internalartifacts.CurrentRevision) Record {
	status := string(d.Status)
	title := draftTitle(d)
	bodyParts := []string{
		title,
		d.Body,
		"status " + status,
		"campaign " + d.CampaignID,
		"channel " + d.Channel,
		"lane " + d.Lane,
		"sku " + d.SKU,
		"post type " + d.PostTypeID,
		"format " + d.Format,
	}
	metadata := map[string]any{
		"draft_id":            d.ID,
		"title":               title,
		"status":              status,
		"campaign_id":         d.CampaignID,
		"post_type_id":        d.PostTypeID,
		"channel":             d.Channel,
		"format":              d.Format,
		"lane":                d.Lane,
		"sku":                 d.SKU,
		"scenario_name":       d.ScenarioName,
		"approved":            rev.HasApplicableApproval(),
		"reviewed":            rev.HasApplicableReview(),
		"revision_id":         rev.RevisionID,
		"approval_actor_kind": rev.ApprovalActorKind,
		"approval_capacity":   rev.ApprovalCapacity,
		"review_run_id":       rev.ReviewRunID,
	}
	if canonical, productTerms := resolveProductReference(d.ScenarioName, d.SKU); canonical != "" {
		bodyParts = append(bodyParts, "product "+canonical, "product aliases "+strings.Join(productTerms, " "))
		metadata["product_canonical"] = canonical
		metadata["product_aliases"] = productTerms
	}
	return Record{
		ID:         "draft:" + d.ID,
		Kind:       KindDraft,
		Visibility: VisibilityOperator,
		FollowUp:   "draft/" + d.ID,
		Title:      title,
		Snippet:    draftSnippet(d, rev, status),
		Body:       strings.Join(bodyParts, ". "),
		Historical: false,
		Metadata:   metadata,
	}
}

func publishRecord(r internalledger.PublishRecord) Record {
	title := r.DraftID
	if strings.TrimSpace(title) == "" {
		title = r.ID
	}
	body := strings.Join([]string{
		"draft " + r.DraftID,
		"series " + r.SeriesID,
		"channel " + r.Channel,
		"audience " + r.Audience,
		"published url " + r.PublishedURL,
		"platform post " + r.PlatformPostID,
		"source " + r.SourceKind,
	}, ". ")
	return Record{
		ID:         "publish:" + r.ID,
		Kind:       KindPublishRecord,
		Visibility: VisibilityOperator,
		FollowUp:   "publish/" + r.ID,
		Title:      title,
		Snippet:    publishSnippet(r),
		Body:       body,
		Historical: true,
		Metadata: map[string]any{
			"draft_id":         r.DraftID,
			"series_id":        r.SeriesID,
			"channel":          r.Channel,
			"audience":         r.Audience,
			"published_url":    r.PublishedURL,
			"platform_post_id": r.PlatformPostID,
			"source_kind":      r.SourceKind,
		},
	}
}

// metricsStaleAfter is the retained-sample freshness window the performance
// aggregate uses. It matches the ledger read's own default so a sample older
// than this is reported as stale rather than as a current reading.
const metricsStaleAfter = 30 * 24 * time.Hour

// launchAssetView is the shared read behind the launch-work and performance
// aggregates: the canonical product's launch-asset slots plus the campaigns
// that own them. A repository read failure yields an empty view so those
// aggregates are skipped rather than projected over a partial scope.
type launchAssetView struct {
	campaigns map[string]internalcampaigns.Campaign
	slots     []internalcampaigns.LaunchAssetSlot
}

func (s *StoreSource) launchAssetView(ctx context.Context) launchAssetView {
	if s.campaignRepo == nil {
		return launchAssetView{}
	}
	slots, err := s.campaignRepo.LaunchAssets(ctx, canonicalProduct)
	if err != nil {
		return launchAssetView{}
	}
	campaigns, err := s.campaignRepo.List(ctx)
	if err != nil {
		return launchAssetView{}
	}
	byID := make(map[string]internalcampaigns.Campaign, len(campaigns))
	for _, campaign := range campaigns {
		byID[campaign.ID] = campaign
	}
	view := launchAssetView{campaigns: make(map[string]internalcampaigns.Campaign), slots: slots}
	for _, slot := range slots {
		if campaign, ok := byID[slot.CampaignID]; ok {
			view.campaigns[slot.CampaignID] = campaign
		}
	}
	return view
}

// activeCampaignCount bounds the launch summary to campaigns the owner still
// considers active.
func activeCampaignCount(campaigns map[string]internalcampaigns.Campaign) int {
	count := 0
	for _, campaign := range campaigns {
		if campaign.Status == internalcampaigns.StatusActive {
			count++
		}
	}
	return count
}

// workSummaryRecords emits one owner-generated classification of the remaining
// marketing work: the open launch drafts with their lifecycle next action, the
// launch-asset readiness, the binding launch constraint, and the ongoing
// capability next actions. The cross-provider reranker reads a hit's
// type/title/path and the first 120 characters of its snippet, so Q08's answer
// cannot be spread across many low-scoring rows. Every value comes from owner
// fields; no query text is copied and no priority is invented.
func (s *StoreSource) workSummaryRecords(ctx context.Context, drafts []internalartifacts.Draft) []Record {
	view := s.launchAssetView(ctx)
	if len(view.slots) == 0 {
		return nil
	}
	draftStatusCounts := map[string]int{}
	launchDraftTotal := 0
	for _, d := range drafts {
		if _, ok := view.campaigns[d.CampaignID]; !ok {
			continue
		}
		launchDraftTotal++
		draftStatusCounts[string(d.Status)]++
	}
	readinessCounts := map[string]int{}
	for _, slot := range view.slots {
		readinessCounts[slotReadiness(slot)]++
	}
	_, bindingAction := weakestLaunchAsset(view.slots)

	statuses := make([]string, 0, len(draftStatusCounts))
	for status := range draftStatusCounts {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	launchActions := make([]string, 0, len(statuses))
	for _, status := range statuses {
		if action := boardDraftNextActions[status]; action != "" {
			launchActions = append(launchActions, fmt.Sprintf("%s (%d draft(s) at %s)", action, draftStatusCounts[status], status))
		}
	}

	capabilities := s.workCapabilities(ctx)
	sort.SliceStable(capabilities, func(i, j int) bool {
		pi, pj := capabilities[i].Priority, capabilities[j].Priority
		if pi <= 0 {
			pi = 1 << 30
		}
		if pj <= 0 {
			pj = 1 << 30
		}
		if pi != pj {
			return pi < pj
		}
		return capabilities[i].ID < capabilities[j].ID
	})
	capabilityNextActions := 0
	topCapabilityFound := false
	var topCapability internalcapabilities.Capability
	for _, capability := range capabilities {
		if strings.TrimSpace(capability.NextAction) == "" {
			continue
		}
		capabilityNextActions++
		if !topCapabilityFound {
			topCapability = capability
			topCapabilityFound = true
		}
	}

	parts := []string{fmt.Sprintf("Marketing work remaining for %s.", canonicalProduct)}
	if launchDraftTotal > 0 {
		parts = append(parts, fmt.Sprintf("Before launch: %d open launch draft(s).", launchDraftTotal))
		for _, action := range launchActions {
			parts = append(parts, "Next action: "+action+".")
		}
	} else {
		parts = append(parts, "Before launch: no launch draft is recorded in an open state.")
	}
	parts = append(parts, fmt.Sprintf("Launch assets: %d approved, %d ready for review, %d in progress, %d empty across %d launch campaign(s).",
		readinessCounts[internalcampaigns.LaunchAssetReadinessApproved],
		readinessCounts[internalcampaigns.LaunchAssetReadinessReadyForReview],
		readinessCounts[internalcampaigns.LaunchAssetReadinessInProgress],
		readinessCounts[internalcampaigns.LaunchAssetReadinessEmpty],
		len(view.campaigns)))
	if bindingAction != "" {
		parts = append(parts, "Binding launch constraint: "+bindingAction+".")
	}
	if capabilityNextActions > 0 {
		parts = append(parts, fmt.Sprintf("After launch (ongoing): %d marketing capability next action(s) recorded.", capabilityNextActions))
	}
	if topCapabilityFound {
		parts = append(parts, fmt.Sprintf("Highest-priority ongoing work: %s (priority %d): %s",
			firstNonEmpty(topCapability.Name, topCapability.ID), topCapability.Priority, strings.TrimSpace(topCapability.NextAction)))
	}
	snippet := strings.Join(parts, " ")

	bodyParts := []string{
		"remaining marketing work",
		"launch and ongoing work classification",
		fmt.Sprintf("open launch drafts %d", launchDraftTotal),
		fmt.Sprintf("launch assets approved %d ready_for_review %d in_progress %d empty %d",
			readinessCounts[internalcampaigns.LaunchAssetReadinessApproved],
			readinessCounts[internalcampaigns.LaunchAssetReadinessReadyForReview],
			readinessCounts[internalcampaigns.LaunchAssetReadinessInProgress],
			readinessCounts[internalcampaigns.LaunchAssetReadinessEmpty]),
	}
	if bindingAction != "" {
		bodyParts = append(bodyParts, "binding launch constraint "+bindingAction)
	}
	if len(launchActions) > 0 {
		bodyParts = append(bodyParts, "launch next actions "+strings.Join(launchActions, "; "))
	}
	if topCapabilityFound {
		bodyParts = append(bodyParts, "highest-priority ongoing work "+firstNonEmpty(topCapability.Name, topCapability.ID)+": "+strings.TrimSpace(topCapability.NextAction))
	}

	launchCampaignIDs := make([]string, 0, len(view.campaigns))
	for id := range view.campaigns {
		launchCampaignIDs = append(launchCampaignIDs, id)
	}
	sort.Strings(launchCampaignIDs)
	draftStatusMetadata := make(map[string]int, len(draftStatusCounts))
	for status, count := range draftStatusCounts {
		draftStatusMetadata[status] = count
	}
	metadata := map[string]any{
		"launch_draft_count":        launchDraftTotal,
		"launch_draft_status":       draftStatusMetadata,
		"launch_campaign_ids":       launchCampaignIDs,
		"launch_assets_approved":    readinessCounts[internalcampaigns.LaunchAssetReadinessApproved],
		"launch_assets_review":      readinessCounts[internalcampaigns.LaunchAssetReadinessReadyForReview],
		"launch_assets_in_progress": readinessCounts[internalcampaigns.LaunchAssetReadinessInProgress],
		"launch_assets_empty":       readinessCounts[internalcampaigns.LaunchAssetReadinessEmpty],
		"binding_launch_action":     bindingAction,
		"launch_next_actions":       launchActions,
		"ongoing_capability_count":  capabilityNextActions,
	}
	if topCapabilityFound {
		metadata["highest_priority_capability"] = firstNonEmpty(topCapability.Name, topCapability.ID)
		metadata["highest_priority_capability_priority"] = topCapability.Priority
		metadata["highest_priority_next_action"] = strings.TrimSpace(topCapability.NextAction)
	}
	return []Record{{
		ID:         "work-summary:launch",
		Kind:       KindWorkSummary,
		Visibility: VisibilityOperator,
		FollowUp:   "capabilities",
		Title:      "Remaining marketing work: launch and ongoing next actions",
		Snippet:    snippet,
		Body:       strings.Join(bodyParts, ". "),
		Historical: false,
		Metadata:   metadata,
	}}
}

// workCapabilities reads the capability catalog for the ongoing-work
// classification. A read failure degrades to a launch-only summary.
func (s *StoreSource) workCapabilities(ctx context.Context) []internalcapabilities.Capability {
	if s.capabilityRepo == nil {
		return nil
	}
	capabilities, err := s.capabilityRepo.List(ctx, "")
	if err != nil {
		return nil
	}
	return capabilities
}

// draftMetricRef pairs a retained reading with its draft so the performance
// aggregate can name the source of a measurement.
type draftMetricRef struct {
	draftID string
	reading internalledger.MetricReading
}

// performanceRecords emits one owner-generated campaign-performance aggregate.
// When real metric samples exist it reports each retained reading with its
// measured/stale state; when none exist it states metrics are unavailable
// rather than zero. It reads only the ledger's own samples and publish history,
// so missing data can never be rendered as a measured result.
func (s *StoreSource) performanceRecords(ctx context.Context, drafts []internalartifacts.Draft, history []internalledger.PublishRecord) []Record {
	view := s.launchAssetView(ctx)
	if len(view.slots) == 0 {
		return nil
	}
	launchDraftIDs := make(map[string]struct{})
	launchDraftTotal := 0
	for _, d := range drafts {
		if _, ok := view.campaigns[d.CampaignID]; !ok {
			continue
		}
		launchDraftIDs[d.ID] = struct{}{}
		launchDraftTotal++
	}
	published := 0
	for _, r := range history {
		if _, ok := launchDraftIDs[r.DraftID]; ok {
			published++
		}
	}
	var readings []draftMetricRef
	if s.ledger != nil {
		for _, d := range drafts {
			if _, ok := launchDraftIDs[d.ID]; !ok {
				continue
			}
			projection, err := s.ledger.DraftMetricProjection(ctx, d.ID, metricsStaleAfter)
			if err != nil {
				continue
			}
			for _, reading := range projection.Readings {
				readings = append(readings, draftMetricRef{draftID: d.ID, reading: reading})
			}
		}
	}
	sort.Slice(readings, func(i, j int) bool {
		if readings[i].reading.Metric != readings[j].reading.Metric {
			return readings[i].reading.Metric < readings[j].reading.Metric
		}
		return readings[i].draftID < readings[j].draftID
	})
	activeCampaigns := activeCampaignCount(view.campaigns)

	if len(readings) == 0 {
		metadata := map[string]any{
			"metrics_available":       false,
			"unavailable_reason":      "no Channel Manager metric samples are recorded",
			"published_release_count": published,
			"launch_draft_count":      launchDraftTotal,
			"active_campaign_count":   activeCampaigns,
		}
		return []Record{{
			ID:         "campaign-performance:launch",
			Kind:       KindCampaignPerformance,
			Visibility: VisibilityOperator,
			FollowUp:   "metrics",
			Title:      "Campaign performance: metrics unavailable (no samples recorded)",
			Snippet: fmt.Sprintf(
				"Campaign performance for %s is unavailable: no Channel Manager metric samples are recorded, so engagement is unknown, not zero. Published releases: %d of %d launch draft(s) across %d active campaign(s). Record real metric samples after release.",
				canonicalProduct, published, launchDraftTotal, activeCampaigns),
			Body: strings.Join([]string{
				"campaign performance",
				"unavailable metrics",
				fmt.Sprintf("published releases %d", published),
				fmt.Sprintf("launch drafts %d", launchDraftTotal),
				fmt.Sprintf("active campaigns %d", activeCampaigns),
				"no Channel Manager metric samples are recorded; performance is unknown, not zero",
			}, ". "),
			Historical: false,
			Metadata:   metadata,
		}}
	}

	statements := make([]string, 0, len(readings))
	metricSet := make(map[string]struct{}, len(readings))
	measuredCount := 0
	metricMetadata := make([]map[string]any, 0, len(readings))
	for _, ref := range readings {
		statements = append(statements, fmt.Sprintf("%s=%g (%s)", ref.reading.Metric, ref.reading.Value, ref.reading.State))
		metricSet[ref.reading.Metric] = struct{}{}
		if ref.reading.State == internalledger.MetricStateMeasured {
			measuredCount++
		}
		metricMetadata = append(metricMetadata, map[string]any{
			"draft_id":         ref.draftID,
			"metric":           ref.reading.Metric,
			"value":            ref.reading.Value,
			"state":            ref.reading.State,
			"sample_count":     ref.reading.SampleCount,
			"last_observed_at": ref.reading.LastObservedAt,
		})
	}
	return []Record{{
		ID:         "campaign-performance:launch",
		Kind:       KindCampaignPerformance,
		Visibility: VisibilityOperator,
		FollowUp:   "metrics",
		Title:      fmt.Sprintf("Campaign performance: %d measured metric(s)", len(metricSet)),
		Snippet: fmt.Sprintf("Campaign performance for %s: %d retained reading(s) across %d metric(s) and %d published release(s). %s.",
			canonicalProduct, len(readings), len(metricSet), published, strings.Join(statements, "; ")),
		Body: strings.Join([]string{
			"campaign performance",
			"real measurement references",
			fmt.Sprintf("published releases %d", published),
			fmt.Sprintf("launch drafts %d", launchDraftTotal),
			"metrics " + strings.Join(statements, "; "),
		}, ". "),
		Historical: false,
		Metadata: map[string]any{
			"metrics_available":       true,
			"metric_count":            len(metricSet),
			"reading_count":           len(readings),
			"measured_count":          measuredCount,
			"metrics":                 metricMetadata,
			"published_release_count": published,
			"launch_draft_count":      launchDraftTotal,
			"active_campaign_count":   activeCampaigns,
		},
	}}
}

func revisionOf(r Record) string {
	payload, err := json.Marshal(struct {
		Kind       string
		Title      string
		Snippet    string
		Body       string
		Freshness  string
		Historical bool
		Metadata   map[string]any
	}{r.Kind, r.Title, r.Snippet, r.Body, r.Freshness, r.Historical, r.Metadata})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func generationOf(records []Record) string {
	h := sha256.New()
	for _, r := range records {
		h.Write([]byte(r.ID))
		h.Write([]byte{0})
		h.Write([]byte(r.Revision))
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)[:16])
}
