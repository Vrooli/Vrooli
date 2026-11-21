package aisearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	internalcampaigns "content-desk/internal/campaigns"
	internalcapabilities "content-desk/internal/capabilities"
)

// canonicalProduct is the product identity Content Desk projects launch
// readiness for. Offer Desk remains the commercial-identity owner; this alias
// map is a small read-only convenience so a query for the alias and a query for
// the canonical id retrieve the same records.
const canonicalProduct = "web-console"

// productAliasIndex maps every observed spelling of a product identity to its
// canonical id.
var productAliasIndex = map[string]string{
	"aquila":      canonicalProduct,
	"web-console": canonicalProduct,
	"webconsole":  canonicalProduct,
	"web console": canonicalProduct,
}

// productAliasTerms is the stable alias set indexed with every record that
// references the canonical product.
var productAliasTerms = []string{"aquila", canonicalProduct}

// unsupportedNoOwner is the explicit limitation for a capability with no
// producing owner. Absence of an owner is stated, never rendered as an empty or
// zero verdict.
const unsupportedNoOwner = "unsupported: no producing owner"

// resolveProductIdentity maps a product reference to its canonical id.
func resolveProductIdentity(reference string) (string, bool) {
	canonical, ok := productAliasIndex[strings.ToLower(strings.TrimSpace(reference))]
	return canonical, ok
}

// resolveProductReference returns the canonical product id and its full alias
// set when any reference names a known product. It is the single resolution
// used by both record indexing and query projection.
func resolveProductReference(references ...string) (string, []string) {
	for _, reference := range references {
		if canonical, ok := resolveProductIdentity(reference); ok {
			return canonical, append([]string(nil), productAliasTerms...)
		}
	}
	return "", nil
}

// normalizeProductQueryAliases rewrites every query token that names a product
// to its canonical id so the alias and canonical queries resolve identically.
func normalizeProductQueryAliases(query string) string {
	fields := strings.Fields(query)
	for i, field := range fields {
		if canonical, ok := resolveProductIdentity(field); ok {
			fields[i] = canonical
		}
	}
	return strings.Join(fields, " ")
}

// boardDraftNextActions mirrors the board program's NEXT_ACTION lifecycle table
// (scenarios/content-desk/.vrooli/program-runtime/board-read.py) so the search
// projection derives the same next action a board read would.
var boardDraftNextActions = map[string]string{
	"requested": "Begin drafting",
	"drafting":  "Complete drafting",
	"drafted":   "Start checking and review",
	"checking":  "Complete the review run",
	"blocked":   "Resolve the blocking gate",
	"reviewed":  "Operator approval required",
	"approved":  "Submit for release",
}

// launchAssetNextAction maps a launch-asset readiness tier to the board's
// draft-status lifecycle action; it never introduces a divergent table.
func launchAssetNextAction(readiness string) string {
	switch readiness {
	case internalcampaigns.LaunchAssetReadinessApproved:
		return boardDraftNextActions["approved"]
	case internalcampaigns.LaunchAssetReadinessReadyForReview:
		return boardDraftNextActions["reviewed"]
	case internalcampaigns.LaunchAssetReadinessInProgress:
		return boardDraftNextActions["drafting"]
	case internalcampaigns.LaunchAssetReadinessEmpty:
		return boardDraftNextActions["requested"]
	}
	return ""
}

func slotReadiness(slot internalcampaigns.LaunchAssetSlot) string {
	if strings.TrimSpace(slot.Readiness) != "" {
		return slot.Readiness
	}
	return slot.ReadinessTier()
}

// weakestLaunchAsset returns the least-advanced readiness tier among the slots
// and its derived action, so a campaign's next action names its binding
// constraint.
func weakestLaunchAsset(slots []internalcampaigns.LaunchAssetSlot) (string, string) {
	rank := map[string]int{
		internalcampaigns.LaunchAssetReadinessEmpty:          0,
		internalcampaigns.LaunchAssetReadinessInProgress:     1,
		internalcampaigns.LaunchAssetReadinessReadyForReview: 2,
		internalcampaigns.LaunchAssetReadinessApproved:       3,
	}
	weakest := ""
	weakestRank := 1 << 30
	for _, slot := range slots {
		tier := slotReadiness(slot)
		value, ok := rank[tier]
		if !ok {
			continue
		}
		if value < weakestRank {
			weakestRank = value
			weakest = tier
		}
	}
	return weakest, launchAssetNextAction(weakest)
}

// firstNonEmpty returns the first value that carries non-whitespace content.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// capabilitySnippet renders a natural-language summary of a capability record.
// The cross-provider reranker reads a hit's type/title/path/snippet, not its
// body, so a terse machine tag line leaves the owner record semantically
// invisible next to a generic tool description. Every term here comes from the
// capability's own fields — no query text is copied — so the summary states the
// same readiness, next action and prerequisites a board read would surface.
func capabilitySnippet(capability internalcapabilities.Capability, readiness internalcapabilities.OperationalReadinessProjection) string {
	parts := []string{fmt.Sprintf("%s is a %s marketing capability.", firstNonEmpty(capability.Name, capability.ID), firstNonEmpty(capability.Medium, "general"))}
	// The latest qualification names the actual artifact that proves the
	// capability. Surface it early from the owner record: the cross-provider
	// reranker reads only type/title/path and the first 120 characters of the
	// snippet, so an artifact identity hidden in metadata leaves a real
	// producer output (for example a captured product screenshot) invisible
	// next to a generic requirement. The value is copied from the owner
	// qualification, never inferred.
	if artifact := latestQualifiedArtifact(capability); artifact != "" {
		parts = append(parts, "Latest qualified artifact "+artifact+".")
	}
	if owner := strings.TrimSpace(capability.Owner); owner != "" {
		parts = append(parts, "Owner: "+owner+".")
	} else {
		parts = append(parts, "No producing owner is named, so the capability is unsupported.")
	}
	if len(capability.Aliases) > 0 {
		parts = append(parts, "Also known as "+strings.Join(capability.Aliases, ", ")+".")
	}
	if len(capability.Channels) > 0 {
		parts = append(parts, "Channels: "+strings.Join(capability.Channels, ", ")+".")
	}
	parts = append(parts, "Operational readiness: "+firstNonEmpty(readiness.Readiness, internalcapabilities.ReadinessUnavailable)+".")
	if next := strings.TrimSpace(capability.NextAction); next != "" {
		parts = append(parts, "Next action: "+next)
	}
	if len(capability.Prerequisites) > 0 {
		parts = append(parts, "Prerequisites: "+strings.Join(capability.Prerequisites, "; ")+".")
	}
	if capability.Priority > 0 {
		priority := fmt.Sprintf("Priority %d", capability.Priority)
		if reason := strings.TrimSpace(capability.PriorityReason); reason != "" {
			priority += " (" + reason + ")"
		}
		parts = append(parts, priority+".")
	}
	if len(readiness.Limitations) > 0 {
		parts = append(parts, "Readiness limitations: "+strings.Join(readiness.Limitations, "; ")+".")
	}
	return strings.Join(parts, " ")
}

// latestQualifiedArtifact renders the identity of the most recent qualified
// artifact for a capability, including its producing run when the owner
// recorded one. It returns an empty string when no qualification or artifact
// exists, so callers never invent a reference.
func latestQualifiedArtifact(capability internalcapabilities.Capability) string {
	qualification := capability.LatestQualification
	if qualification == nil {
		return ""
	}
	artifact := strings.TrimSpace(qualification.LatestArtifactID)
	if artifact == "" {
		return ""
	}
	if run := strings.TrimSpace(qualification.LatestRunID); run != "" {
		return artifact + " (run " + run + ")"
	}
	return artifact
}

// campaignSnippet renders a natural-language campaign summary carrying the
// canonical product, status, launch-asset readiness and the binding next action.
func campaignSnippet(campaign internalcampaigns.Campaign, canonical string, productTerms, slotBodies []string, nextAction string) string {
	head := "Campaign " + firstNonEmpty(campaign.Name, campaign.ID)
	if canonical != "" {
		head += " for " + canonical
	}
	parts := []string{head + "."}
	parts = append(parts, "Status: "+firstNonEmpty(campaign.Status, "unknown")+".")
	if len(campaign.ScenarioNames) > 0 {
		parts = append(parts, "Products: "+strings.Join(campaign.ScenarioNames, ", ")+".")
	}
	if len(productTerms) > 0 {
		parts = append(parts, "Product aliases: "+strings.Join(productTerms, ", ")+".")
	}
	if len(slotBodies) > 0 {
		parts = append(parts, "Launch assets: "+strings.Join(slotBodies, "; ")+".")
	}
	if strings.TrimSpace(nextAction) != "" {
		parts = append(parts, "Next action: "+nextAction)
	}
	return strings.Join(parts, " ")
}

func capabilityRecord(ctx context.Context, capability internalcapabilities.Capability, links []internalcapabilities.CapabilityLink, evidence internalcapabilities.EvidenceResolver, distribution internalcapabilities.DistributionResolver, now time.Time) Record {
	qualification := capability.LatestQualification
	if qualification == nil {
		unknown := internalcapabilities.UnknownQualification(capability.ID)
		qualification = &unknown
	}
	quality := internalcapabilities.RehydrateOutputQuality(ctx, capability.OutputQuality, links, evidence)
	readiness := internalcapabilities.RehydrateOperationalReadiness(capability.OperationalReadiness, capability, now)
	connectivity := internalcapabilities.RehydrateDistributionConnectivity(ctx, capability.DistributionConnectivity, capability, distribution)
	limitations := make([]string, 0, len(quality.Limitations)+len(readiness.Limitations)+len(connectivity.Limitations)+1)
	supported := strings.TrimSpace(capability.Owner) != ""
	if !supported {
		limitations = append(limitations, unsupportedNoOwner)
	}
	limitations = append(limitations, quality.Limitations...)
	limitations = append(limitations, readiness.Limitations...)
	limitations = append(limitations, connectivity.Limitations...)

	body := strings.Join([]string{
		capability.Name,
		"capability",
		"medium " + capability.Medium,
		"aliases " + strings.Join(capability.Aliases, " "),
		"channels " + strings.Join(capability.Channels, " "),
		"owner " + capability.Owner,
		"producing operation " + capability.ProducingOperation,
		"definition status " + capability.DefinitionStatus,
		"implementation status " + capability.ImplementationStatus,
		"operational readiness " + readiness.Readiness,
		"output quality " + quality.Quality,
		"distribution connectivity " + connectivity.Connectivity,
		"next action " + capability.NextAction,
		"limitations " + strings.Join(limitations, "; "),
		"latest qualified artifact " + qualification.LatestArtifactID,
		"latest qualification run " + qualification.LatestRunID,
		"qualification environment " + qualification.Environment,
		"qualification observed at " + qualification.ObservedAt,
	}, ". ")
	snippet := capabilitySnippet(capability, readiness)
	return Record{
		ID:         "capability:" + capability.ID,
		Kind:       KindCapability,
		Visibility: VisibilityOperator,
		FollowUp:   "capability/" + capability.ID,
		Title:      capability.Name,
		Snippet:    snippet,
		Body:       body,
		Historical: false,
		Metadata: map[string]any{
			"capability_id":                         capability.ID,
			"name":                                  capability.Name,
			"medium":                                capability.Medium,
			"aliases":                               capability.Aliases,
			"channels":                              capability.Channels,
			"owner":                                 capability.Owner,
			"supported":                             supported,
			"producing_operation":                   capability.ProducingOperation,
			"prerequisites":                         capability.Prerequisites,
			"priority":                              capability.Priority,
			"priority_reason":                       capability.PriorityReason,
			"priority_scope":                        capability.PriorityScope,
			"definition_status":                     capability.DefinitionStatus,
			"implementation_status":                 capability.ImplementationStatus,
			"operational_readiness":                 readiness.Readiness,
			"operational_readiness_source":          readiness.Source,
			"operational_readiness_limitations":     readiness.Limitations,
			"output_quality":                        quality.Quality,
			"output_quality_source":                 quality.Source,
			"output_quality_limitations":            quality.Limitations,
			"distribution_connectivity":             connectivity.Connectivity,
			"distribution_connectivity_source":      connectivity.Source,
			"distribution_connectivity_limitations": connectivity.Limitations,
			"readiness_limitations":                 limitations,
			"next_action":                           capability.NextAction,
			"source_refs":                           capability.SourceRefs,
			"latest_qualification": map[string]any{
				"latest_artifact_id": qualification.LatestArtifactID,
				"latest_run_id":      qualification.LatestRunID,
				"environment":        qualification.Environment,
				"validated_at":       qualification.ValidatedAt,
				"observed_at":        qualification.ObservedAt,
				"freshness_basis":    qualification.FreshnessBasis,
				"candidate_identity": qualification.CandidateIdentity,
				"max_age_seconds":    qualification.MaxAgeSeconds,
				"limitation":         qualification.Limitation,
				"next_action":        qualification.NextAction,
			},
		},
	}
}

func campaignRecord(campaign internalcampaigns.Campaign, slots []internalcampaigns.LaunchAssetSlot) Record {
	canonical, productTerms := resolveProductReference(campaign.ScenarioNames...)
	slotMetadata := make([]map[string]any, 0, len(slots))
	slotBodies := make([]string, 0, len(slots))
	for _, slot := range slots {
		tier := slotReadiness(slot)
		slotMetadata = append(slotMetadata, map[string]any{
			"channel":                slot.Channel,
			"format":                 slot.Format,
			"capacity":               slot.Capacity,
			"reserved":               slot.Reserved,
			"approved_count":         slot.ApprovedCount,
			"ready_for_review_count": slot.ReadyForReviewCount,
			"in_progress_count":      slot.InProgressCount,
			"readiness":              tier,
			"next_action":            launchAssetNextAction(tier),
		})
		slotBodies = append(slotBodies, fmt.Sprintf("%s:%s readiness %s", slot.Channel, slot.Format, tier))
	}
	_, nextAction := weakestLaunchAsset(slots)

	bodyParts := []string{
		campaign.Name,
		"campaign",
		"status " + campaign.Status,
		"scenarios " + strings.Join(campaign.ScenarioNames, " "),
	}
	if canonical != "" {
		bodyParts = append(bodyParts, "product "+canonical, "product aliases "+strings.Join(productTerms, " "))
	}
	if len(slotBodies) > 0 {
		bodyParts = append(bodyParts, "launch assets "+strings.Join(slotBodies, "; "))
	}
	if nextAction != "" {
		bodyParts = append(bodyParts, "next action "+nextAction)
	}
	snippet := campaignSnippet(campaign, canonical, productTerms, slotBodies, nextAction)

	metadata := map[string]any{
		"campaign_id":   campaign.ID,
		"name":          campaign.Name,
		"status":        campaign.Status,
		"scenarios":     campaign.ScenarioNames,
		"launch_assets": slotMetadata,
		"next_action":   nextAction,
	}
	if canonical != "" {
		metadata["product_canonical"] = canonical
		metadata["product_aliases"] = productTerms
	}
	return Record{
		ID:         "campaign:" + campaign.ID,
		Kind:       KindCampaign,
		Visibility: VisibilityOperator,
		FollowUp:   "campaign/" + campaign.ID,
		Title:      campaign.Name,
		Snippet:    snippet,
		Body:       strings.Join(bodyParts, ". "),
		Historical: false,
		Metadata:   metadata,
	}
}
