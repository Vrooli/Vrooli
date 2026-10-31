package runreport

import (
	"sort"

	"agent-manager/internal/runsignal"
)

// ClassifierVersion names the deterministic rules used for a report. It is
// deliberately returned to callers: a projection is evidence, not an opaque
// interpretation whose rules can silently change underneath an investigation.
const ClassifierVersion = "passive-evidence.v2"

// Cohort is a bounded multi-run projection. Reports remain the source of truth
// for drill-down; this type only contains ranked, reproducible signals.
type Cohort struct {
	ClassifierVersion  string                   `json:"classifierVersion"`
	RunIDs             []string                 `json:"runIds"`
	Availability       Availability             `json:"availability"`
	Signals            []CohortSignal           `json:"signals"`
	TimeAccounting     runsignal.TimeAccounting `json:"timeAccounting"`
	InvocationValidity EvidenceValidity         `json:"invocationValidity"`
}

type CohortSignal struct {
	Kind                 string   `json:"kind"`
	Count                int      `json:"count"`
	Impact               int      `json:"impact"`
	Confidence           string   `json:"confidence"`
	RepresentativeRunIDs []string `json:"representativeRunIds"`
}

type GoalCohort struct {
	ClassifierVersion       string       `json:"classifierVersion"`
	Availability            Availability `json:"availability"`
	Population              int          `json:"population"`
	GoalRuns                int          `json:"goalRuns"`
	MetRuns                 int          `json:"metRuns"`
	AverageTimeToFirstMetMS float64      `json:"averageTimeToFirstMetMs"`
	AverageTokensToFirstMet float64      `json:"averageTokensToFirstMet"`
	InterventionsBeforeMet  int          `json:"interventionsBeforeMet"`
	Regressions             int          `json:"regressions"`
	Restatements            int          `json:"restatements"`
}

func BuildGoalCohort(reports []*RunReport) GoalCohort {
	out := GoalCohort{ClassifierVersion: ClassifierVersion, Availability: Availability{State: AvailabilityAvailable}, Population: len(reports)}
	var timeTotal, tokenTotal float64
	for _, report := range reports {
		if report == nil || report.Goal.Availability.State == AvailabilityUnavailable {
			out.Availability = Availability{State: AvailabilityDegraded, Reason: "one or more selected runs carry no goal"}
			continue
		}
		out.GoalRuns++
		if report.Goal.FirstMetEventIndex >= 0 {
			out.MetRuns++
			timeTotal += float64(report.Goal.TimeToFirstMet.Milliseconds())
			tokenTotal += float64(report.Goal.TokenCostBeforeFirstMet)
			out.InterventionsBeforeMet += report.Goal.InterventionsBeforeMet
		}
		if report.Goal.Regression {
			out.Regressions++
		}
		if report.Goal.Restatement {
			out.Restatements++
		}
	}
	if out.MetRuns > 0 {
		out.AverageTimeToFirstMetMS = timeTotal / float64(out.MetRuns)
		out.AverageTokensToFirstMet = tokenTotal / float64(out.MetRuns)
	} else {
		out.Availability = Availability{State: AvailabilityUnavailable, Reason: "no goal-carrying run reached a first-met marker"}
	}
	return out
}

// BuildCohort folds a caller-selected, comparable set of reports. It never
// reads raw transcript data and caps both signals and representatives.
func BuildCohort(reports []*RunReport) Cohort {
	out := Cohort{ClassifierVersion: ClassifierVersion, Availability: Availability{State: AvailabilityAvailable}}
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
			out.Availability = Availability{State: AvailabilityDegraded, Reason: "one or more reports were unavailable"}
			continue
		}
		id := report.RunID.String()
		out.TimeAccounting.ModelGeneratingMS += report.TimeAccounting.ModelGeneratingMS
		out.TimeAccounting.ToolExecutingMS += report.TimeAccounting.ToolExecutingMS
		out.TimeAccounting.IdleWaitingMS += report.TimeAccounting.IdleWaitingMS
		out.TimeAccounting.AwaitingHumanMS += report.TimeAccounting.AwaitingHumanMS
		out.TimeAccounting.UnattributableMS += report.TimeAccounting.UnattributableMS
		out.TimeAccounting.ModelTokens += report.TimeAccounting.ModelTokens
		out.TimeAccounting.ToolTokens += report.TimeAccounting.ToolTokens
		out.TimeAccounting.IdleTokens += report.TimeAccounting.IdleTokens
		out.TimeAccounting.HumanTokens += report.TimeAccounting.HumanTokens
		out.TimeAccounting.UnattributableTokens += report.TimeAccounting.UnattributableTokens
		out.RunIDs = append(out.RunIDs, id)
		out.InvocationValidity.TotalCalls += report.InvocationValidity.TotalCalls
		out.InvocationValidity.ClassifiedBase += report.InvocationValidity.ClassifiedBase
		out.InvocationValidity.UnclassifiedCount += report.InvocationValidity.UnclassifiedCount
		if report.EventsAvailability.State != AvailabilityAvailable {
			out.Availability = Availability{State: AvailabilityDegraded, Reason: "event evidence is unavailable for part of the cohort"}
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
		if report.ReceiptsAvailability.State == AvailabilityDegraded || report.ReceiptsAvailability.State == AvailabilityUnavailable {
			add("receipt_availability", 1, id)
		}
	}
	if out.InvocationValidity.TotalCalls == 0 {
		out.InvocationValidity.Availability = Availability{State: AvailabilityUnavailable, Reason: "no invocation facts are available for the cohort"}
	} else {
		out.InvocationValidity.UnclassifiedShare = float64(out.InvocationValidity.UnclassifiedCount) / float64(out.InvocationValidity.TotalCalls)
		out.InvocationValidity.Availability = Availability{State: AvailabilityAvailable}
		if float64(out.InvocationValidity.ClassifiedBase)/float64(out.InvocationValidity.TotalCalls) < 0.90 {
			out.InvocationValidity.Availability = Availability{State: AvailabilityUnreliable, Reason: "classified invocation share is below the minimum 90%"}
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
