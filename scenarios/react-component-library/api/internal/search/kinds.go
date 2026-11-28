package search

import (
	"strings"
	"unicode"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

// The text forms are the stored vocabulary. They are stable because they are
// part of the observable index (facets and SQL filters speak them); the proto
// enum is the transport vocabulary. The two are translated at the boundary.
const (
	kindWorkspace = "workspace"
	kindPage      = "page"
	kindDisplay   = "display"
	kindState     = "state"
	kindPageType  = "page_type"
	kindAsset     = "asset"
	kindCommand   = "command"
	kindEvidence  = "evidence"

	availabilityAvailable   = "available"
	availabilityDraft       = "draft"
	availabilityPlanned     = "planned"
	availabilityUnavailable = "unavailable"
)

// KindText converts the transport enum to its stored form.
func KindText(kind searchv1.SearchKind) string {
	switch kind {
	case searchv1.SearchKind_SEARCH_KIND_WORKSPACE:
		return kindWorkspace
	case searchv1.SearchKind_SEARCH_KIND_PAGE:
		return kindPage
	case searchv1.SearchKind_SEARCH_KIND_DISPLAY:
		return kindDisplay
	case searchv1.SearchKind_SEARCH_KIND_STATE:
		return kindState
	case searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE:
		return kindPageType
	case searchv1.SearchKind_SEARCH_KIND_ASSET:
		return kindAsset
	case searchv1.SearchKind_SEARCH_KIND_COMMAND:
		return kindCommand
	case searchv1.SearchKind_SEARCH_KIND_EVIDENCE:
		return kindEvidence
	default:
		return ""
	}
}

// ParseKind is the inverse of KindText. Unknown stored values become
// UNSPECIFIED rather than silently pretending to be assets.
func ParseKind(text string) searchv1.SearchKind {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case kindWorkspace:
		return searchv1.SearchKind_SEARCH_KIND_WORKSPACE
	case kindPage:
		return searchv1.SearchKind_SEARCH_KIND_PAGE
	case kindDisplay:
		return searchv1.SearchKind_SEARCH_KIND_DISPLAY
	case kindState:
		return searchv1.SearchKind_SEARCH_KIND_STATE
	case kindPageType:
		return searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE
	case kindAsset:
		return searchv1.SearchKind_SEARCH_KIND_ASSET
	case kindCommand:
		return searchv1.SearchKind_SEARCH_KIND_COMMAND
	case kindEvidence:
		return searchv1.SearchKind_SEARCH_KIND_EVIDENCE
	default:
		return searchv1.SearchKind_SEARCH_KIND_UNSPECIFIED
	}
}

// KindLabel is the human facet label for a stored kind.
func KindLabel(text string) string {
	switch text {
	case kindWorkspace:
		return "Workspace"
	case kindPage:
		return "Page"
	case kindDisplay:
		return "Display"
	case kindState:
		return "State"
	case kindPageType:
		return "Page type"
	case kindAsset:
		return "Asset"
	case kindCommand:
		return "Command"
	case kindEvidence:
		return "Evidence"
	default:
		return text
	}
}

// AvailabilityText converts the transport enum to its stored form.
func AvailabilityText(value searchv1.SearchAvailability) string {
	switch value {
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE:
		return availabilityAvailable
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_DRAFT:
		return availabilityDraft
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_PLANNED:
		return availabilityPlanned
	case searchv1.SearchAvailability_SEARCH_AVAILABILITY_UNAVAILABLE:
		return availabilityUnavailable
	default:
		return availabilityAvailable
	}
}

// ParseAvailability is the inverse of AvailabilityText. Unknown values are
// reported as unavailable: an unrecognised state must never look usable.
func ParseAvailability(text string) searchv1.SearchAvailability {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case availabilityAvailable:
		return searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE
	case availabilityDraft:
		return searchv1.SearchAvailability_SEARCH_AVAILABILITY_DRAFT
	case availabilityPlanned:
		return searchv1.SearchAvailability_SEARCH_AVAILABILITY_PLANNED
	case availabilityUnavailable:
		return searchv1.SearchAvailability_SEARCH_AVAILABILITY_UNAVAILABLE
	default:
		return searchv1.SearchAvailability_SEARCH_AVAILABILITY_UNAVAILABLE
	}
}

// AvailabilityLabel is the human facet label for a stored availability.
func AvailabilityLabel(text string) string {
	switch text {
	case availabilityAvailable:
		return "Available"
	case availabilityDraft:
		return "Draft"
	case availabilityPlanned:
		return "Planned"
	case availabilityUnavailable:
		return "Unavailable"
	default:
		return text
	}
}

// kindPrior is the outcome-first prior from the workbench program (§3.4): a
// design object outranks the component that realizes it, and a component
// outranks a command. Display is not named in the program's table; it sits
// between page and state because it is a child of a page and a parent of a
// state.
func kindPrior(text string) float64 {
	switch text {
	case kindWorkspace:
		return 2.50
	case kindPage:
		return 1.35
	case kindDisplay:
		return 1.32
	case kindState:
		return 1.30
	case kindPageType:
		return 1.20
	case kindAsset:
		return 1.00
	case kindCommand:
		return 0.90
	case kindEvidence:
		return 0.85
	default:
		return 1.00
	}
}

// prioritizeExactPageType removes unrelated page-type neighbors when the query
// names a known page type. The concrete workspace realization still ranks by
// the normal outcome-first prior, ahead of the reusable archetype.
func prioritizeExactPageType(candidates []candidate, plan queryPlan) []candidate {
	var exact bool
	for _, cand := range candidates {
		if KindText(cand.doc.Kind) == kindPageType && identifierTitle(cand.doc.Title) == plan.normalized {
			exact = true
			break
		}
	}
	if !exact {
		return candidates
	}
	out := candidates[:0]
	for _, cand := range candidates {
		if KindText(cand.doc.Kind) == kindPageType {
			if identifierTitle(cand.doc.Title) != plan.normalized {
				continue
			}
		}
		out = append(out, cand)
	}
	return out
}

func identifierTitle(value string) string {
	runes := []rune(value)
	var normalized strings.Builder
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1])) {
			normalized.WriteRune(' ')
		}
		normalized.WriteRune(unicode.ToLower(r))
	}
	return strings.Join(tokenize(normalized.String()), " ")
}

// availabilityFactor rewards results a reader can open now without hiding
// planned work; the penalty is a nudge, never an exclusion.
func availabilityFactor(text string) float64 {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case availabilityAvailable:
		return 1.08
	case availabilityDraft:
		return 1.00
	case availabilityPlanned:
		return 0.90
	case availabilityUnavailable:
		return 0.75
	default:
		return 0.90
	}
}

// browseKindRank is the curated empty-query order. Evidence is deliberately
// absent: it is a review result, not a browse destination. Callers that ask for
// evidence explicitly still get it through the kind filter.
func browseKindRank(text string) int {
	switch text {
	case kindWorkspace:
		return 0
	case kindPage:
		return 1
	case kindDisplay:
		return 2
	case kindState:
		return 3
	case kindPageType:
		return 4
	case kindAsset:
		return 5
	case kindCommand:
		return 6
	default:
		return 99
	}
}

func availabilityRank(text string) int {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case availabilityAvailable:
		return 0
	case availabilityDraft:
		return 1
	case availabilityPlanned:
		return 2
	case availabilityUnavailable:
		return 3
	default:
		return 4
	}
}
