package runreport

import (
	"sort"
)

// ClassifierVersion names the deterministic rules used for a report. It is
// deliberately returned to callers: a projection is evidence, not an opaque
// interpretation whose rules can silently change underneath an investigation.
const ClassifierVersion = "passive-evidence.v1"

// Cohort is a bounded multi-run projection. Reports remain the source of truth
// for drill-down; this type only contains ranked, reproducible signals.
type Cohort struct {
	ClassifierVersion string         `json:"classifierVersion"`
	RunIDs            []string       `json:"runIds"`
	Availability      Availability   `json:"availability"`
	Signals           []CohortSignal `json:"signals"`
}

type CohortSignal struct {
	Kind                 string   `json:"kind"`
	Count                int      `json:"count"`
	Impact               int      `json:"impact"`
	Confidence           string   `json:"confidence"`
	RepresentativeRunIDs []string `json:"representativeRunIds"`
}

type EpisodeCohort struct {
	Availability Availability    `json:"availability"`
	Signals      []EpisodeSignal `json:"signals"`
}
type EpisodeSignal struct {
	Fingerprint          string   `json:"fingerprint"`
	Occurrences          int      `json:"occurrences"`
	DistinctRuns         int      `json:"distinctRuns"`
	SummedCostMS         int64    `json:"summedCostMs"`
	Confidence           string   `json:"confidence"`
	RepresentativeRunIDs []string `json:"representativeRunIds"`
}

func BuildEpisodeCohort(episodesByRun map[string][]FrictionEpisode) EpisodeCohort {
	out := EpisodeCohort{Availability: Availability{State: "available"}}
	type bucket struct {
		occurrences int
		cost        int64
		ids         map[string]bool
	}
	buckets := map[string]*bucket{}
	for runID, episodes := range episodesByRun {
		if len(episodes) == 0 {
			out.Availability = Availability{State: "degraded", Detail: "one or more selected runs have no derived episodes"}
			continue
		}
		for _, episode := range episodes {
			b := buckets[episode.Fingerprint]
			if b == nil {
				b = &bucket{ids: map[string]bool{}}
				buckets[episode.Fingerprint] = b
			}
			b.occurrences++
			b.cost += episode.WallClockMS
			b.ids[runID] = true
		}
	}
	for fingerprint, b := range buckets {
		ids := make([]string, 0, len(b.ids))
		for id := range b.ids {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		confidence := "medium"
		if len(ids) >= 2 {
			confidence = "high"
		}
		if len(ids) > 5 {
			ids = ids[:5]
		}
		out.Signals = append(out.Signals, EpisodeSignal{Fingerprint: fingerprint, Occurrences: b.occurrences, DistinctRuns: len(b.ids), SummedCostMS: b.cost, Confidence: confidence, RepresentativeRunIDs: ids})
	}
	sort.Slice(out.Signals, func(i, j int) bool {
		if out.Signals[i].SummedCostMS != out.Signals[j].SummedCostMS {
			return out.Signals[i].SummedCostMS > out.Signals[j].SummedCostMS
		}
		return out.Signals[i].DistinctRuns > out.Signals[j].DistinctRuns
	})
	return out
}

// BuildCohort folds a caller-selected, comparable set of reports. It never
// reads raw transcript data and caps both signals and representatives.
func BuildCohort(reports []*RunReport) Cohort {
	out := Cohort{ClassifierVersion: ClassifierVersion, Availability: Availability{State: "available"}}
	type bucket struct {
		count, impact int
		ids           []string
	}
	buckets := map[string]*bucket{}
	add := func(kind string, impact int, id string) {
		b := buckets[kind]
		if b == nil {
			b = &bucket{}
			buckets[kind] = b
		}
		b.count++
		b.impact += impact
		if len(b.ids) < 5 {
			b.ids = append(b.ids, id)
		}
	}
	for _, report := range reports {
		if report == nil {
			out.Availability = Availability{State: "degraded", Detail: "one or more reports were unavailable"}
			continue
		}
		id := report.RunID.String()
		out.RunIDs = append(out.RunIDs, id)
		if report.EventsAvailability.State != "available" {
			out.Availability = Availability{State: "degraded", Detail: "event evidence is unavailable for part of the cohort"}
		}
		for _, tool := range report.Tools {
			if tool.Failures > 0 {
				add("command_failure", tool.Failures, id)
			}
		}
		if report.RepeatedToolCalls > 0 {
			add("repeated_work", report.RepeatedToolCalls, id)
		}
		if report.FilesReadMoreThanOnce > 0 {
			add("reread", report.FilesReadMoreThanOnce, id)
		}
		if report.ExternalToolCalls > 0 {
			add("external_fallback", report.ExternalToolCalls, id)
		}
		if report.HelpRecoveries > 0 {
			add("help_recovery", report.HelpRecoveries, id)
		}
		if report.UnknownInvocations > 0 {
			add("unknown_invocation", report.UnknownInvocations, id)
		}
		if report.FallbackCount > 0 {
			add("model_fallback", report.FallbackCount, id)
		}
		if report.ReceiptsAvailability.State == "degraded" || report.ReceiptsAvailability.State == "unavailable" {
			add("receipt_availability", 1, id)
		}
	}
	for kind, b := range buckets {
		confidence := "medium"
		if b.count >= 2 {
			confidence = "high"
		}
		out.Signals = append(out.Signals, CohortSignal{Kind: kind, Count: b.count, Impact: b.impact, Confidence: confidence, RepresentativeRunIDs: b.ids})
	}
	sort.Strings(out.RunIDs)
	sort.Slice(out.Signals, func(i, j int) bool {
		if out.Signals[i].Impact != out.Signals[j].Impact {
			return out.Signals[i].Impact > out.Signals[j].Impact
		}
		if out.Signals[i].Count != out.Signals[j].Count {
			return out.Signals[i].Count > out.Signals[j].Count
		}
		return out.Signals[i].Kind < out.Signals[j].Kind
	})
	if len(out.Signals) > 20 {
		out.Signals = out.Signals[:20]
	}
	return out
}
