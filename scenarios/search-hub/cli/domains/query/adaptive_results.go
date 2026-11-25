package query

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

var (
	workspaceSourcePath = regexp.MustCompile(`^scenarios/[a-z0-9][a-z0-9-]*(?:/[A-Za-z0-9_.@+-]+)+$`)
	lineAnchor          = regexp.MustCompile(`^L([1-9][0-9]*)(?:-L?([1-9][0-9]*))?$`)
	cliCommandToken     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:-]*$`)
	actionWord          = regexp.MustCompile(`[a-z0-9]+`)
)

var actionStopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {}, "be": {}, "by": {},
	"do": {}, "find": {}, "for": {}, "from": {}, "give": {}, "how": {}, "i": {},
	"in": {}, "into": {}, "is": {}, "it": {}, "me": {}, "my": {}, "of": {},
	"on": {}, "or": {}, "the": {}, "then": {}, "this": {}, "to": {}, "what": {},
	"when": {}, "where": {}, "with": {}, "next": {}, "safe": {},
}

func searchedCorpusCount(msg *routingv1.QueryResponse) int {
	if len(msg.GetCorporaSearched()) > 0 {
		seen := make(map[string]struct{}, len(msg.GetCorporaSearched()))
		for _, corpus := range msg.GetCorporaSearched() {
			if corpus = strings.TrimSpace(corpus); corpus != "" {
				seen[corpus] = struct{}{}
			}
		}
		return len(seen)
	}
	return len(msg.GetGroups())
}

func responseHitCount(msg *routingv1.QueryResponse) (int, string) {
	reported := 0
	if msg != nil {
		for _, group := range msg.GetGroups() {
			if group != nil && group.GetCount() > 0 {
				reported += int(group.GetCount())
			}
		}
	}
	if reported > 0 {
		return reported, "provider-reported hit(s)"
	}
	return len(allResponseHits(msg)), "returned hit(s)"
}

func responseDegradationSummary(msg *routingv1.QueryResponse) string {
	if msg == nil || (!msg.GetDegraded() && !msg.GetPartial() && len(msg.GetDegradations()) == 0 && strings.TrimSpace(msg.GetRoutingDegradeReason()) == "" && !hasDegradedGroup(msg.GetGroups())) {
		return ""
	}
	var details []string
	for _, degradation := range msg.GetDegradations() {
		if degradation == nil || len(details) >= 3 {
			continue
		}
		parts := make([]string, 0, 3)
		if component := strings.TrimSpace(degradation.GetComponent()); component != "" {
			parts = append(parts, component)
		}
		if provider := strings.TrimSpace(degradation.GetProviderId()); provider != "" {
			parts = append(parts, truncate(provider, 48))
		}
		if reason := strings.TrimSpace(degradation.GetReason()); reason != "" {
			parts = append(parts, truncate(reason, 48))
		}
		if detail := strings.TrimSpace(degradation.GetDetail()); detail != "" {
			parts = append(parts, truncate(detail, 72))
		}
		if len(parts) == 0 {
			continue
		}
		detail := strings.Join(parts, ": ")
		if degradation.GetFallbackActive() {
			detail += " (fallback active)"
		}
		details = append(details, detail)
	}
	for _, group := range msg.GetGroups() {
		if group == nil || !group.GetDegraded() || len(details) >= 3 {
			continue
		}
		provider := strings.TrimSpace(group.GetProviderId())
		if provider == "" {
			provider = "provider"
		}
		groupDetails := group.GetDegradations()
		if len(groupDetails) > 0 {
			for _, degradation := range groupDetails {
				if degradation == nil || len(details) >= 3 {
					continue
				}
				parts := []string{truncate(provider, 48)}
				if leg := strings.TrimSpace(degradation.GetLeg()); leg != "" {
					parts = append(parts, truncate(leg, 32))
				}
				if reason := strings.TrimSpace(degradation.GetReason()); reason != "" {
					parts = append(parts, truncate(reason, 48))
				}
				if detail := strings.TrimSpace(degradation.GetDetail()); detail != "" {
					parts = append(parts, truncate(detail, 72))
				}
				details = append(details, strings.Join(parts, ": "))
			}
			continue
		}
		note := strings.TrimSpace(group.GetNote())
		if note == "" {
			note = "degraded"
		}
		details = append(details, truncate(provider+": "+note, 140))
	}
	if reason := strings.TrimSpace(msg.GetRoutingDegradeReason()); reason != "" && len(details) < 3 {
		details = append(details, "routing: "+truncate(reason, 72))
	}
	if msg.GetPartial() {
		pending := msg.GetPendingProviders()
		if pending > 0 {
			details = append(details, fmt.Sprintf("partial; %d provider(s) pending", pending))
		} else {
			details = append(details, "partial provider results")
		}
	}
	if msg.GetDegraded() && len(details) == 0 {
		selector := strings.TrimSpace(msg.GetSelectorLeg())
		if selector == "" {
			selector = "unreported"
		}
		details = append(details, fmt.Sprintf("retrieval degraded; provider-specific cause not reported (selector %s; ordering by %s)", selector, orderedBy(msg)))
	}
	if len(details) == 0 {
		return ""
	}
	return " (" + strings.Join(details, "; ") + ")"
}

func hasDegradedGroup(groups []*routingv1.ProviderResultGroup) bool {
	for _, group := range groups {
		if group != nil && (group.GetDegraded() || len(group.GetDegradations()) > 0) {
			return true
		}
	}
	return false
}

func flattenGroupHits(groups []*routingv1.ProviderResultGroup) []*routingv1.SearchHit {
	var hits []*routingv1.SearchHit
	for _, group := range groups {
		if group != nil {
			hits = append(hits, group.GetHits()...)
		}
	}
	return hits
}

func deduplicateHits(hits []*routingv1.SearchHit) []*routingv1.SearchHit {
	seen := make(map[string]struct{}, len(hits))
	out := make([]*routingv1.SearchHit, 0, len(hits))
	for _, hit := range hits {
		key := hitDedupeKey(hit)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, hit)
	}
	return out
}

func hitDedupeKey(hit *routingv1.SearchHit) string {
	if hit == nil {
		return "<nil>"
	}
	if source := strings.TrimSpace(hit.GetPath()); source != "" {
		return "path:" + path.Clean(strings.SplitN(source, "#", 2)[0])
	}
	if id := strings.TrimSpace(hit.GetId()); id != "" {
		return strings.Join([]string{"id", hit.GetProviderId(), hit.GetType(), id}, ":")
	}
	return strings.Join([]string{"fallback", hit.GetProviderId(), hit.GetTitle(), hit.GetSnippet()}, ":")
}

func allResponseHits(msg *routingv1.QueryResponse) []*routingv1.SearchHit {
	if msg == nil {
		return nil
	}
	if msg.GetReranked() && len(msg.GetRanked()) > 0 {
		return deduplicateHits(msg.GetRanked())
	}
	return deduplicateHits(flattenGroupHits(msg.GetGroups()))
}

func displayTitle(hit *routingv1.SearchHit) string {
	if hit == nil {
		return ""
	}
	if title := strings.TrimSpace(hit.GetTitle()); title != "" {
		return title
	}
	return hit.GetId()
}

func directEntityMatch(ctx interface{ Positional(string) string }, msg *routingv1.QueryResponse) *routingv1.SearchHit {
	if ctx == nil || msg == nil {
		return nil
	}
	query := strings.Trim(strings.TrimSpace(ctx.Positional("text")), "\"'` ")
	if query == "" {
		return nil
	}
	for _, hit := range allResponseHits(msg) {
		for _, value := range []string{hit.GetId(), hit.GetTitle(), hit.GetPath()} {
			if strings.EqualFold(query, strings.Trim(strings.TrimSpace(value), "\"'` ")) {
				return hit
			}
		}
	}
	return nil
}

// directEntityViewHits keeps the exact match first while applying the same
// global human hit budget to the rest of the response. The exact row is a
// priority ordering choice, not a reason to hide every other returned hit.
func directEntityViewHits(ctx interface {
	Positional(string) string
	BoolFlag(string) bool
}, msg *routingv1.QueryResponse) []*routingv1.SearchHit {
	if msg == nil {
		return nil
	}
	exact := directEntityMatch(ctx, msg)
	if exact == nil {
		return nil
	}
	limit := defaultVisibleHits
	if ctx != nil && ctx.BoolFlag("show-all-results") {
		limit = 0
	}
	hits := make([]*routingv1.SearchHit, 0, len(allResponseHits(msg)))
	seen := make(map[string]struct{})
	add := func(hit *routingv1.SearchHit) bool {
		key := hitDedupeKey(hit)
		if _, ok := seen[key]; ok {
			return true
		}
		if limit > 0 && len(hits) >= limit {
			return false
		}
		seen[key] = struct{}{}
		hits = append(hits, hit)
		return true
	}
	add(exact)
	for _, hit := range allResponseHits(msg) {
		if !add(hit) {
			break
		}
	}
	return hits
}

func renderDirectEntityView(ctx interface {
	Positional(string) string
	BoolFlag(string) bool
}, msg *routingv1.QueryResponse, exact *routingv1.SearchHit, maxHits int) []string {
	hits := directEntityViewHits(ctx, msg)
	if exact == nil || len(hits) == 0 {
		return renderDirectMatch(exact, maxHits, msg.GetGroups())
	}
	out := []string{formatHit(1, hits[0])}
	if detail := hitDegradationDetail(hits[0], msg.GetGroups()); detail != "" {
		out[0] += " — provider coverage degraded: " + detail
	}
	for i, hit := range hits[1:] {
		line := formatHit(i+2, hit)
		if detail := hitDegradationDetail(hit, msg.GetGroups()); detail != "" {
			line += " — provider coverage degraded: " + detail
		}
		out = append(out, line)
	}
	allCount := len(allResponseHits(msg))
	if allCount > len(hits) {
		out = append(out, fmt.Sprintf("… %d additional unique returned hit(s) omitted; use --show-all-results to display them", allCount-len(hits)))
	}
	return out
}

func renderDirectMatch(hit *routingv1.SearchHit, maxHits int, groupSets ...[]*routingv1.ProviderResultGroup) []string {
	if hit == nil {
		return []string{"(no exact entity match)"}
	}
	if maxHits == 0 {
		return renderRankedLimit([]*routingv1.SearchHit{hit}, maxHits, groupSets...)
	}
	line := formatHit(1, hit)
	if len(groupSets) > 0 {
		if detail := hitDegradationDetail(hit, groupSets[0]); detail != "" {
			line += " — provider coverage degraded: " + detail
		}
	}
	return []string{line}
}

func renderTaskParts(msg *routingv1.QueryResponse, maxHits int) []string {
	out := []string{}
	seen := make(map[string]string)
	shownHits := 0

	if len(msg.GetQueryParts()) > 0 {
		out = append(out, "Request parts:")
	}
	parts := msg.GetQueryParts()
	for _, part := range parts {
		if part == nil {
			continue
		}
		label := strings.TrimSpace(part.GetLabel())
		if label == "" {
			label = part.GetPartId()
		}
		status := "status unavailable"
		switch part.GetStatus() {
		case routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED:
			status = "candidate hits returned (relevance not judged)"
		case routingv1.QueryPartEvidence_STATUS_SEARCHED_EMPTY:
			status = "searched; no candidates returned"
		case routingv1.QueryPartEvidence_STATUS_PROVIDER_DEGRADED:
			status = "provider degraded; coverage incomplete"
		case routingv1.QueryPartEvidence_STATUS_NOT_ROUTED:
			status = "not routed"
		default:
			if part.GetStatus() != routingv1.QueryPartEvidence_STATUS_UNSPECIFIED {
				status = fmt.Sprintf("unrecognized status (%d)", int32(part.GetStatus()))
			}
		}
		switch part.GetSource() {
		case routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT:
			status += " (derived segment)"
		case routingv1.QueryPartEvidence_SOURCE_UNSPECIFIED, routingv1.QueryPartEvidence_SOURCE_USER_QUERY:
		default:
			status += fmt.Sprintf("; unrecognized source (%d)", int32(part.GetSource()))
		}
		if part.GetDegraded() && part.GetStatus() != routingv1.QueryPartEvidence_STATUS_PROVIDER_DEGRADED {
			status += "; degraded part coverage"
		}
		if part.GetPartial() {
			status += "; partial part coverage"
		}
		providers := shortProviderList(part.GetSearchedProviderIds())
		if providers != "" {
			status += "; searched " + providers
		}
		if degradation := partProviderDegradation(part, msg); degradation != "" {
			status += "; " + degradation
		}
		if reason := partReasonText(part.GetReason()); reason != "" {
			status += "; " + truncate(reason, 100)
		}

		type referencedHit struct {
			hit *routingv1.SearchHit
			ref *routingv1.SearchHitReference
		}
		var validHits []referencedHit
		for _, ref := range part.GetEvidence() {
			if hit := resolveHitReference(msg, ref); hit != nil {
				validHits = append(validHits, referencedHit{hit: hit, ref: ref})
			}
		}
		if part.GetStatus() == routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED && len(validHits) > 0 {
			weakCount := 0
			for _, evidence := range validHits {
				if evidence.hit.GetConfidence().GetWeak() {
					weakCount++
				}
			}
			if weakCount == len(validHits) {
				status += "; all referenced candidates are weak"
			} else if weakCount > 0 {
				status += "; some hits are weak"
			}
		} else if part.GetStatus() == routingv1.QueryPartEvidence_STATUS_CANDIDATES_RETURNED && len(validHits) == 0 {
			status += "; supporting hit references unavailable"
		}
		out = append(out, fmt.Sprintf("• %s — %s", truncate(label, 100), status))

		omittedEvidence := 0
		omittedKeys := make(map[string]struct{})
		for _, evidence := range validHits {
			hit := evidence.hit
			key := hitDedupeKey(hit)
			if previous, ok := seen[key]; ok {
				line := "  same candidate shown above: " + truncate(previous, 120)
				if unknown := unrecognizedRetrievalSources(evidence.ref); unknown != "" {
					line += " — " + unknown
				}
				out = append(out, line)
				continue
			}
			if maxHits > 0 && shownHits >= maxHits {
				if _, counted := omittedKeys[key]; !counted {
					omittedKeys[key] = struct{}{}
					omittedEvidence++
				}
				continue
			}
			seen[key] = displayTitle(hit) + " (" + hit.GetProviderId() + ")"
			shownHits++
			line := "  " + truncate(displayTitle(hit), 100) + " [" + confidenceBand(hit) + " " + truncate(hit.GetProviderId(), 48) + "/" + truncate(provenance(hit), 92) + "]"
			if part.GetSource() == routingv1.QueryPartEvidence_SOURCE_DERIVED_SEGMENT && hasRetrievalSource(evidence.ref, routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY) {
				line += " (also found for full query)"
			}
			if unknown := unrecognizedRetrievalSources(evidence.ref); unknown != "" {
				line += " (" + unknown + ")"
			}
			out = append(out, line)
		}
		if omittedEvidence > 0 {
			out = append(out, fmt.Sprintf("  %d unique evidence hit(s) omitted by shared display budget", omittedEvidence))
		}
	}
	if len(msg.GetCorrectionCandidates()) > 0 {
		out = append(out, "Possible entity corrections (not verified):")
		candidates := msg.GetCorrectionCandidates()
		alternatives := correctionAlternativeCounts(msg, candidates)
		shownCandidates := 0
		omittedCandidates := 0
		for _, candidate := range candidates {
			if candidate == nil {
				continue
			}
			hit := resolveHitReference(msg, candidate.GetEvidence())
			// A suggestion without a returned, identity-checked hit is not evidence
			// and must not be presented as a correction possibility.
			if hit == nil || strings.TrimSpace(candidate.GetSubmittedValue()) == "" || strings.TrimSpace(candidate.GetCandidateValue()) == "" {
				omittedCandidates++
				continue
			}
			if maxHits > 0 && shownCandidates >= maxHits {
				omittedCandidates++
				continue
			}
			key := hitDedupeKey(hit)
			previous, evidenceShown := seen[key]
			if !evidenceShown && maxHits > 0 && shownHits >= maxHits {
				omittedCandidates++
				continue
			}
			shownCandidates++
			strength := "strength unreported"
			switch candidate.GetStrength() {
			case routingv1.CorrectionCandidate_STRENGTH_BAND_STRONG:
				strength = "strong candidate, not verified"
			case routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE:
				strength = "possible candidate, not verified"
			}
			line := fmt.Sprintf("  %q → %q (%s; %s)", truncate(candidate.GetSubmittedValue(), 70), truncate(candidate.GetCandidateValue(), 70), strength, truncate(candidate.GetProviderId(), 48))
			if alternatives[strings.ToLower(candidate.GetSubmittedValue())] > 1 {
				line += fmt.Sprintf(" [ambiguous: %d alternatives]", alternatives[strings.ToLower(candidate.GetSubmittedValue())])
			}
			if hit.GetConfidence().GetWeak() {
				line += " — weak supporting evidence"
			}
			if evidenceShown {
				line += " — evidence shown above: " + truncate(previous, 120)
			} else {
				seen[key] = displayTitle(hit) + " (" + hit.GetProviderId() + ")"
				shownHits++
				line += " — " + truncate(displayTitle(hit), 70) + " (" + truncate(provenance(hit), 92) + ")"
			}
			out = append(out, line)
		}
		if omittedCandidates > 0 {
			out = append(out, fmt.Sprintf("  … %d correction candidate(s) omitted (unusable evidence or over display budget)", omittedCandidates))
		}
	}
	return out
}

func hitDegradationDetail(hit *routingv1.SearchHit, groups []*routingv1.ProviderResultGroup) string {
	if hit == nil {
		return ""
	}
	provider := strings.TrimSpace(hit.GetProviderId())
	if provider == "" {
		provider = strings.TrimSpace(hit.GetProviderGroup())
	}
	for _, group := range groups {
		if group == nil || group.GetProviderId() != provider || (!group.GetDegraded() && len(group.GetDegradations()) == 0) {
			continue
		}
		return truncate(providerGroupDegradation(group), 120)
	}
	return ""
}

func partProviderDegradation(part *routingv1.QueryPartEvidence, msg *routingv1.QueryResponse) string {
	if part == nil || msg == nil {
		return ""
	}
	groups := msg.GetGroups()
	searched := make(map[string]struct{}, len(part.GetSearchedProviderIds()))
	for _, provider := range part.GetSearchedProviderIds() {
		if provider = strings.TrimSpace(provider); provider != "" {
			searched[provider] = struct{}{}
		}
	}
	for _, ref := range part.GetEvidence() {
		if hit := resolveHitReference(msg, ref); hit != nil {
			for _, provider := range []string{hit.GetProviderId(), ref.GetProviderId()} {
				if provider = strings.TrimSpace(provider); provider != "" {
					searched[provider] = struct{}{}
				}
			}
		}
	}
	var details []string
	for _, group := range groups {
		if group == nil || (!group.GetDegraded() && len(group.GetDegradations()) == 0) {
			continue
		}
		provider := strings.TrimSpace(group.GetProviderId())
		if _, ok := searched[provider]; !ok {
			continue
		}
		details = append(details, truncate(provider, 40)+": "+truncate(providerGroupDegradation(group), 72))
		if len(details) == 2 {
			break
		}
	}
	if len(details) == 0 {
		return ""
	}
	return "degraded provider coverage " + strings.Join(details, "; ")
}

func partReasonText(raw string) string {
	reason := strings.TrimSpace(raw)
	switch reason {
	case "ambiguous_provider_evidence":
		return "multiple owners have similar evidence; owner is ambiguous"
	case "no_positive_provider_evidence":
		return "no provider returned positive evidence for this part"
	case "ambiguous_lexical_boundary":
		return "routing stopped at a tied owner boundary; owner is ambiguous"
	default:
		return reason
	}
}

func hasRetrievalSource(ref *routingv1.SearchHitReference, source routingv1.SearchHitReference_RetrievalSource) bool {
	for _, got := range ref.GetRetrievalSources() {
		if got == source {
			return true
		}
	}
	return false
}

func unrecognizedRetrievalSources(ref *routingv1.SearchHitReference) string {
	if ref == nil {
		return ""
	}
	seen := make(map[int32]struct{})
	var values []int32
	for _, source := range ref.GetRetrievalSources() {
		switch source {
		case routingv1.SearchHitReference_RETRIEVAL_SOURCE_UNSPECIFIED,
			routingv1.SearchHitReference_RETRIEVAL_SOURCE_USER_QUERY,
			routingv1.SearchHitReference_RETRIEVAL_SOURCE_DERIVED_SEGMENT:
			continue
		}
		value := int32(source)
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	if len(values) == 0 {
		return ""
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	labels := make([]string, len(values))
	for i, value := range values {
		labels[i] = fmt.Sprint(value)
	}
	return "unrecognized retrieval source (" + strings.Join(labels, ", ") + ")"
}

func renderCorrections(msg *routingv1.QueryResponse, maxHits int) []string {
	return renderTaskParts(msg, maxHits)
}

func correctionAlternativeCounts(msg *routingv1.QueryResponse, candidates []*routingv1.CorrectionCandidate) map[string]int {
	values := make(map[string]map[string]struct{})
	for _, candidate := range candidates {
		if candidate == nil || resolveHitReference(msg, candidate.GetEvidence()) == nil {
			continue
		}
		submitted := strings.ToLower(strings.TrimSpace(candidate.GetSubmittedValue()))
		value := strings.ToLower(strings.TrimSpace(candidate.GetCandidateValue()))
		if submitted == "" || value == "" {
			continue
		}
		if values[submitted] == nil {
			values[submitted] = make(map[string]struct{})
		}
		values[submitted][value] = struct{}{}
	}
	counts := make(map[string]int, len(values))
	for submitted, alternatives := range values {
		counts[submitted] = len(alternatives)
	}
	return counts
}

func resolveHitReference(msg *routingv1.QueryResponse, ref *routingv1.SearchHitReference) *routingv1.SearchHit {
	if msg == nil || ref == nil {
		return nil
	}
	var hit *routingv1.SearchHit
	if ref.RankedIndex != nil {
		index := int(ref.GetRankedIndex())
		if index >= 0 && index < len(msg.GetRanked()) {
			hit = msg.GetRanked()[index]
		}
	} else if ref.GroupIndex != nil && ref.HitIndex != nil {
		groupIndex, hitIndex := int(ref.GetGroupIndex()), int(ref.GetHitIndex())
		if groupIndex >= 0 && groupIndex < len(msg.GetGroups()) {
			group := msg.GetGroups()[groupIndex]
			if group != nil && hitIndex >= 0 && hitIndex < len(group.GetHits()) {
				hit = group.GetHits()[hitIndex]
			}
		}
	}
	if hit == nil || (ref.GetHitId() != "" && hit.GetId() != ref.GetHitId()) {
		return nil
	}
	if ref.GetProviderId() != "" && hit.GetProviderId() != ref.GetProviderId() {
		return nil
	}
	if ref.GetType() != "" && hit.GetType() != ref.GetType() {
		return nil
	}
	if ref.GetPath() != "" && hit.GetPath() != ref.GetPath() {
		return nil
	}
	return hit
}

func shortProviderList(providers []string) string {
	if len(providers) == 0 {
		return ""
	}
	shown := providers
	if len(shown) > 3 {
		shown = shown[:3]
	}
	clean := make([]string, 0, len(shown))
	for _, provider := range shown {
		clean = append(clean, truncate(provider, 40))
	}
	result := strings.Join(clean, ", ")
	if len(providers) > len(shown) {
		result += fmt.Sprintf(" (+%d more)", len(providers)-len(shown))
	}
	return result
}

func nextActionHints(ctx interface {
	Positional(string) string
	BoolFlag(string) bool
}, msg *routingv1.QueryResponse) []string {
	if hit := bestCommandCandidate(ctx, msg); hit != nil {
		if command := commandHelpSuggestion(hit); command != "" {
			return []string{"Inspect this command's supported options before using it: `" + command + "`"}
		}
	}
	for _, hit := range visibleResponseHits(ctx, msg) {
		if actionCandidateRelevant(ctx, msg, hit) {
			if command := sourceReadCommand(hit.GetPath()); command != "" {
				return []string{"Read candidate evidence before relying on it: `" + command + "`"}
			}
			for _, location := range hit.GetLocations() {
				if command := sourceReadCommand(location); command != "" {
					return []string{"Read candidate evidence before relying on it: `" + command + "`"}
				}
			}
		}
	}
	if hit := bestOperationCandidate(ctx, msg); hit != nil {
		if operation := operationHintForHit(hit); len(operation) > 0 {
			for _, line := range operation {
				if strings.Contains(line, "program-runtime library run ") {
					return []string{"Inspect the candidate program contract and required skill: `" + strings.TrimSpace(line) + "`"}
				}
			}
		}
	}
	return nil
}

// visibleResponseHits mirrors the active human view so handoff hints never
// point at a candidate the user cannot see in the same response.
func visibleResponseHits(ctx interface {
	Positional(string) string
	BoolFlag(string) bool
}, msg *routingv1.QueryResponse) []*routingv1.SearchHit {
	if msg == nil {
		return nil
	}
	limit := defaultVisibleHits
	if ctx != nil && ctx.BoolFlag("show-all-results") {
		limit = 0
	}
	if len(msg.GetQueryParts()) > 0 || len(msg.GetCorrectionCandidates()) > 0 {
		var visible []*routingv1.SearchHit
		seen := make(map[string]struct{}, defaultVisibleHits)
		add := func(hit *routingv1.SearchHit) bool {
			if hit == nil {
				return false
			}
			key := hitDedupeKey(hit)
			if _, ok := seen[key]; ok {
				return true
			}
			if limit > 0 && len(visible) >= limit {
				return false
			}
			seen[key] = struct{}{}
			visible = append(visible, hit)
			return true
		}
		for _, part := range msg.GetQueryParts() {
			if part == nil {
				continue
			}
			for _, ref := range part.GetEvidence() {
				if hit := resolveHitReference(msg, ref); hit != nil {
					if !add(hit) {
						return visible
					}
				}
			}
		}
		shownCandidates := 0
		for _, candidate := range msg.GetCorrectionCandidates() {
			if candidate == nil || strings.TrimSpace(candidate.GetSubmittedValue()) == "" || strings.TrimSpace(candidate.GetCandidateValue()) == "" {
				continue
			}
			if limit > 0 && shownCandidates >= limit {
				break // The human correction view caps suggestion rows at the hit budget.
			}
			hit := resolveHitReference(msg, candidate.GetEvidence())
			if hit == nil {
				continue
			}
			_, alreadyVisible := seen[hitDedupeKey(hit)]
			if !alreadyVisible && limit > 0 && len(visible) >= limit {
				continue // Task-part hits own the shared budget before optional correction evidence.
			}
			shownCandidates++
			if !add(hit) {
				return visible
			}
		}
		return visible
	}
	if exact := directEntityMatch(ctx, msg); exact != nil {
		return directEntityViewHits(ctx, msg)
	}
	hits := allResponseHits(msg)
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func actionCandidateRelevant(ctx interface{ Positional(string) string }, msg *routingv1.QueryResponse, hit *routingv1.SearchHit) bool {
	if ctx == nil || msg == nil || hit == nil {
		return false
	}
	queryTerms := significantWords(strings.TrimSpace(ctx.Positional("text")))
	if len(queryTerms) == 0 {
		return false
	}
	minimum := len(queryTerms)
	if minimum < 2 {
		minimum = 2
	}
	if minimum > 4 {
		minimum = 4
	}
	return actionCandidateScore(ctx, msg, hit) >= minimum
}

func actionCandidateScore(ctx interface{ Positional(string) string }, msg *routingv1.QueryResponse, hit *routingv1.SearchHit) int {
	if ctx == nil || msg == nil || hit == nil {
		return 0
	}
	query := strings.TrimSpace(ctx.Positional("text"))
	for _, value := range []string{hit.GetId(), hit.GetTitle(), hit.GetPath()} {
		if strings.EqualFold(query, strings.TrimSpace(value)) {
			return 100
		}
	}
	queryTerms := significantWords(query)
	// Provider IDs are ownership/provenance, not task text (e.g. cli-health
	// must not make an unrelated status command look relevant to a health task).
	// Detected intent is useful to explain retrieval, but it is inferred routing
	// metadata; it must not ground an executable follow-up recommendation.
	candidate := hit.GetTitle()
	for _, correction := range msg.GetCorrectionCandidates() {
		if resolveHitReference(msg, correction.GetEvidence()) == hit {
			candidate += " " + correction.GetSubmittedValue() + " " + correction.GetCandidateValue()
		}
	}
	for _, part := range msg.GetQueryParts() {
		if part != nil && queryPartReferencesHit(msg, part, hit) {
			candidate += " " + part.GetLabel()
		}
	}
	return tokenOverlap(queryTerms, significantWords(candidate))
}

func tokenOverlap(query, candidate map[string]struct{}) int {
	matched := 0
	for queryWord := range query {
		for candidateWord := range candidate {
			if queryWord == candidateWord || (len(queryWord) >= 5 && len(candidateWord) >= 5 && (strings.HasPrefix(queryWord, candidateWord) || strings.HasPrefix(candidateWord, queryWord))) {
				matched++
				break
			}
		}
	}
	return matched
}

func bestCommandCandidate(ctx interface {
	Positional(string) string
	BoolFlag(string) bool
}, msg *routingv1.QueryResponse) *routingv1.SearchHit {
	var selected *routingv1.SearchHit
	bestScore := 0
	for _, hit := range visibleResponseHits(ctx, msg) {
		if commandHelpSuggestion(hit) == "" {
			continue
		}
		score := actionCandidateScore(ctx, msg, hit)
		if actionCandidateRelevant(ctx, msg, hit) && score > bestScore {
			selected, bestScore = hit, score
		}
	}
	return selected
}

func bestOperationCandidate(ctx interface {
	Positional(string) string
	BoolFlag(string) bool
}, msg *routingv1.QueryResponse) *routingv1.SearchHit {
	var selected *routingv1.SearchHit
	bestScore := 0
	for _, hit := range visibleResponseHits(ctx, msg) {
		if len(operationHintForHit(hit)) == 0 || !actionCandidateRelevant(ctx, msg, hit) {
			continue
		}
		score := actionCandidateScore(ctx, msg, hit)
		if score > bestScore {
			selected, bestScore = hit, score
		}
	}
	return selected
}

func queryPartReferencesHit(msg *routingv1.QueryResponse, part *routingv1.QueryPartEvidence, expected *routingv1.SearchHit) bool {
	for _, ref := range part.GetEvidence() {
		if resolveHitReference(msg, ref) == expected {
			return true
		}
	}
	return false
}

func significantWords(value string) map[string]struct{} {
	words := make(map[string]struct{})
	for _, word := range actionWord.FindAllString(strings.ToLower(value), -1) {
		if len(word) < 4 {
			continue
		}
		if _, stop := actionStopWords[word]; stop {
			continue
		}
		words[word] = struct{}{}
	}
	if strings.Contains(strings.ToLower(value), "ios") {
		words["ios"] = struct{}{}
	}
	return words
}

func commandHelpSuggestion(hit *routingv1.SearchHit) string {
	if hit == nil || hit.GetProviderId() != "cli-health.commands" || hit.GetConfidence().GetWeak() {
		return ""
	}
	command := strings.TrimSpace(hit.GetPath())
	if command == "" {
		command = strings.TrimSpace(hit.GetTitle())
	}
	tokens := strings.Fields(command)
	if len(tokens) == 0 {
		return ""
	}
	for _, token := range tokens {
		if !cliCommandToken.MatchString(token) {
			return ""
		}
	}
	return strings.Join(tokens, " ") + " --help"
}

func sourceReadCommand(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, "\r\n\x00") {
		return ""
	}
	parts := strings.SplitN(raw, "#", 2)
	file := parts[0]
	if !workspaceSourcePath.MatchString(file) || path.Clean(file) != file {
		return ""
	}
	ext := strings.ToLower(path.Ext(file))
	switch ext {
	case ".md", ".mdx", ".go", ".tsx", ".ts", ".jsx", ".js", ".json", ".proto", ".yaml", ".yml", ".toml", ".sql", ".sh":
	default:
		return ""
	}
	start, end := 1, 140
	if len(parts) == 2 {
		if anchor := lineAnchor.FindStringSubmatch(parts[1]); len(anchor) == 3 {
			var err error
			if _, err = fmt.Sscanf(anchor[1], "%d", &start); err != nil {
				return ""
			}
			if anchor[2] != "" {
				if _, err = fmt.Sscanf(anchor[2], "%d", &end); err != nil {
					return ""
				}
			}
			if end < start || end-start > 140 {
				end = start + 140
			}
		}
	}
	return fmt.Sprintf("sed -n '%d,%dp' '%s'", start, end, file)
}
