package main

// merge.go combines the deterministic regex/checksum spans with the NER model
// spans into a single, non-overlapping, sorted result set.
//
// Policy (deterministic, so the HTTP contract is stable and testable):
//   1. All spans are gathered and sorted by (start asc, length desc, source).
//   2. Overlapping spans are resolved keeping the HIGHER-CONFIDENCE span; on a
//      confidence tie the regex span wins (structured IDs are precise), then
//      the longer span wins. A span fully contained in a kept span is dropped.
//   3. The regex pass and the model pass are peers: neither is authoritative,
//      but regex breaks ties because its validators (Luhn/IBAN/SSN) are exact.

import "sort"

// MergeSpans returns the de-overlapped union of regex and model spans.
func MergeSpans(regex, model []Span) []Span {
	all := make([]Span, 0, len(regex)+len(model))
	all = append(all, regex...)
	all = append(all, model...)
	return dedupeOverlaps(all)
}

func sourceRank(s string) int {
	// Lower is preferred when breaking a confidence tie.
	if s == "regex" {
		return 0
	}
	return 1
}

func dedupeOverlaps(spans []Span) []Span {
	if len(spans) == 0 {
		return nil
	}
	sort.SliceStable(spans, func(i, j int) bool {
		a, b := spans[i], spans[j]
		if a.Start != b.Start {
			return a.Start < b.Start
		}
		la, lb := a.End-a.Start, b.End-b.Start
		if la != lb {
			return la > lb // longer first
		}
		if a.Confidence != b.Confidence {
			return a.Confidence > b.Confidence
		}
		return sourceRank(a.Source) < sourceRank(b.Source)
	})

	var kept []Span
	for _, cand := range spans {
		conflict := -1
		for i, k := range kept {
			if overlaps(cand, k) {
				conflict = i
				break
			}
		}
		if conflict == -1 {
			kept = append(kept, cand)
			continue
		}
		if preferred(cand, kept[conflict]) {
			kept[conflict] = cand
		}
	}

	sort.SliceStable(kept, func(i, j int) bool {
		if kept[i].Start != kept[j].Start {
			return kept[i].Start < kept[j].Start
		}
		return kept[i].End < kept[j].End
	})
	return kept
}

func overlaps(a, b Span) bool {
	return a.Start < b.End && b.Start < a.End
}

// preferred reports whether span a should replace span b when they conflict.
func preferred(a, b Span) bool {
	if a.Confidence != b.Confidence {
		return a.Confidence > b.Confidence
	}
	if sourceRank(a.Source) != sourceRank(b.Source) {
		return sourceRank(a.Source) < sourceRank(b.Source)
	}
	return (a.End - a.Start) > (b.End - b.Start)
}
