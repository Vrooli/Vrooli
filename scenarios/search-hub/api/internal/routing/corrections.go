package routing

import (
	"strings"
	"unicode"
	"unicode/utf8"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

var correctionStopwords = correctionStopwordSet()

func correctionStopwordSet() map[string]struct{} {
	words := strings.Fields(`about after all also am an and any are as at be because been before being between both but by can could did do does doing down during each few for from further had has have having he her here hers herself him himself his how i if in into is it its itself just me more most my myself no nor not now of off on once only or other our ours ourselves out over own please same search she should so some such than that the their theirs them themselves then there these they this those through to too under until up very was we were what when where which while who whom why will with would you your yours yourself yourselves`)
	stopwords := make(map[string]struct{}, len(words))
	for _, word := range words {
		stopwords[word] = struct{}{}
	}
	return stopwords
}

type correctionToken struct {
	text       string
	normalized string
}

type correctionEvidence struct {
	provider   string
	reference  *routingv1.SearchHitReference
	value      string
	normalized string
}

// buildCorrectionCandidates suggests only a unique one-edit token that is
// present in stable identity text from an actual returned hit. Suggestions
// are possible corrections, not relevance claims, and never rewrite query.
func buildCorrectionCandidates(query string, response *routingv1.QueryResponse) []*routingv1.CorrectionCandidate {
	queryTokens := correctionTokens(query)
	if response == nil || len(queryTokens) == 0 {
		return nil
	}

	var evidence []correctionEvidence
	seen := make(map[string]struct{})
	addHit := func(hit *routingv1.SearchHit, providerID string, ref *routingv1.SearchHitReference) {
		if hit == nil || providerID == "" {
			return
		}
		for _, field := range []string{hit.GetId(), hit.GetTitle(), hit.GetPath()} {
			for _, token := range correctionTokens(field) {
				if utf8.RuneCountInString(token.normalized) < 5 || isCorrectionStopword(token.normalized) {
					continue
				}
				key := strings.Join([]string{providerID, hit.GetId(), hit.GetType(), hit.GetPath(), token.normalized}, "\x00")
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				evidence = append(evidence, correctionEvidence{provider: providerID, reference: ref, value: token.text, normalized: token.normalized})
			}
		}
	}

	// Ranked hits are authoritative when present. Otherwise retain a concrete
	// provider-group/hit location rather than manufacturing a ranked position.
	if len(response.GetRanked()) > 0 {
		for index, hit := range response.GetRanked() {
			if hit == nil {
				continue
			}
			rankedIndex := int32(index)
			addHit(hit, hit.GetProviderId(), &routingv1.SearchHitReference{
				ProviderId: hit.GetProviderId(), RankedIndex: &rankedIndex,
				HitId: hit.GetId(), Type: hit.GetType(), Path: hit.GetPath(),
			})
		}
	} else {
		for groupIndex, group := range response.GetGroups() {
			if group == nil {
				continue
			}
			for hitIndex, hit := range group.GetHits() {
				if hit == nil {
					continue
				}
				providerID := hit.GetProviderId()
				if providerID == "" {
					providerID = group.GetProviderId()
				}
				if providerID == "" {
					continue
				}
				groupPosition, hitPosition := int32(groupIndex), int32(hitIndex)
				addHit(hit, providerID, &routingv1.SearchHitReference{
					ProviderId: providerID, GroupIndex: &groupPosition, HitIndex: &hitPosition,
					HitId: hit.GetId(), Type: hit.GetType(), Path: hit.GetPath(),
				})
			}
		}
	}
	if len(evidence) == 0 {
		return nil
	}

	var result []*routingv1.CorrectionCandidate
	for _, submitted := range queryTokens {
		if utf8.RuneCountInString(submitted.normalized) < 5 || isCorrectionStopword(submitted.normalized) {
			continue
		}
		values := make(map[string]struct{})
		for _, candidate := range evidence {
			if oneEditApart(submitted.normalized, candidate.normalized) {
				values[candidate.normalized] = struct{}{}
			}
		}
		if len(values) != 1 {
			continue
		}
		var onlyValue string
		for value := range values {
			onlyValue = value
		}
		seenProvider := make(map[string]struct{})
		for _, candidate := range evidence {
			if candidate.normalized != onlyValue {
				continue
			}
			if _, exists := seenProvider[candidate.provider]; exists {
				continue
			}
			seenProvider[candidate.provider] = struct{}{}
			result = append(result, &routingv1.CorrectionCandidate{
				SubmittedValue: submitted.text,
				CandidateValue: candidate.value,
				ProviderId:     candidate.provider,
				Strength:       routingv1.CorrectionCandidate_STRENGTH_BAND_POSSIBLE,
				Evidence:       candidate.reference,
			})
		}
	}
	return result
}

func correctionTokens(text string) []correctionToken {
	var tokens []correctionToken
	var token strings.Builder
	flush := func() {
		if token.Len() == 0 {
			return
		}
		text := token.String()
		tokens = append(tokens, correctionToken{text: text, normalized: strings.ToLower(text)})
		token.Reset()
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			token.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return tokens
}

func isCorrectionStopword(token string) bool {
	_, ok := correctionStopwords[token]
	return ok
}
