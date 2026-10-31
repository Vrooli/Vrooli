package outcomeledger

import "sort"

// Correction is an operator message classified as a correction of the agent,
// either by the intervention category or by the operator-signal label.
type Correction struct {
	RunID    string
	Category string
	// Probability of the category; zero when the method gives none.
	Probability float64
	// Signal is set instead of Category for an operator-signal label:
	// SignalReportedFailure or SignalCorrection.
	Signal string
	// Final marks the last operator message of its run.
	Final bool
	// Precision is the classifier's measured hold-out precision for the label at
	// the operating point it was measured at. Zero means unmeasured and the signal
	// is ignored.
	Precision float64
}

// Operator-signal labels the ledger reads (interventions.ReportedFailure and
// interventions.Correction).
const (
	SignalReportedFailure = "reported_failure"
	SignalCorrection      = "correction"
)

const (
	// minCorrectionProbability bounds the older intervention categories, which
	// have no measured operating point: below even odds the classifier is guessing.
	minCorrectionProbability = 0.5
	// MinSignalPrecision is the lowest measured hold-out precision at which an
	// operator-signal label may move an estimate.
	MinSignalPrecision = 0.5
)

// correctionCategories are the classified categories that say work was wrong.
// The others (approval_or_info, scope_change, too_dense, false_inability) say
// nothing about whether the result was accepted.
var correctionCategories = map[string]bool{"quality_miss": true, "wrong_claim": true, "early_stop": true}

// IsCorrection reports a category that counts as a correction.
func IsCorrection(category string) bool { return correctionCategories[category] }

// Estimate is a work unit's outcome estimate and the evidence it rests on.
type Estimate struct {
	Outcome string
	Source  string
}

func isManaged(run RunFacts) bool { return run.Path == PathInteractive || run.Path == PathHeadless }

// Attempts are a unit's top-level runs in time order; a sub-agent is not an attempt.
func attempts(runs []RunFacts) []RunFacts {
	var out []RunFacts
	for _, run := range runs {
		if !run.Child() {
			out = append(out, run)
		}
	}
	if len(out) == 0 {
		out = append(out, runs...)
	}
	sort.SliceStable(out, func(i, j int) bool { return runTime(out[i]).Before(runTime(out[j])) })
	return out
}

// EstimateUnit applies the documented rules in order; the first match wins.
// The same rules read imported and managed runs. Accepted needs evidence beside
// the run's lifecycle. A headless "complete" alone is completed, a separate tier
// that says nothing about acceptance; an imported "complete" only means the
// transcript ended and is no evidence at all.
func EstimateUnit(runs []RunFacts, corrections []Correction) Estimate {
	if len(runs) == 0 {
		return Estimate{OutcomeUnknown, SourceNoSignal}
	}
	for _, run := range runs {
		if run.Live {
			return Estimate{OutcomeUnknown, SourceInProgress}
		}
	}
	list := attempts(runs)
	last := list[len(list)-1]
	switch last.ApprovalState {
	case "approved":
		return Estimate{OutcomeAccepted, SourceOperatorApproval}
	case "rejected":
		return Estimate{OutcomeFailed, SourceOperatorRejection}
	}
	if estimate, ok := operatorEstimate(last, corrections); ok {
		return estimate
	}
	if isManaged(last) && last.Status == "failed" {
		return Estimate{OutcomeFailed, SourceRunStatus}
	}
	if last.Status == "cancelled" || (last.TerminalClass == "interruption" && last.StopReason == "crash") {
		return Estimate{OutcomeAbandoned, SourceRunStatus}
	}
	for _, attempt := range list[:len(list)-1] {
		if attempt.Status == "failed" || attempt.Status == "cancelled" {
			return Estimate{OutcomeCorrected, SourceRetry}
		}
	}
	if last.TerminalClass == "verdict" && last.StopReason == "blocked" {
		return Estimate{OutcomeFailed, SourceAgentVerdict}
	}
	var validation RunFacts
	for _, run := range runs {
		if run.LastValidation != "" && !run.LastValidationAt.Before(validation.LastValidationAt) {
			validation = run
		}
	}
	switch validation.LastValidation {
	case "failure":
		return Estimate{OutcomeFailed, SourceValidation}
	case "success":
		return Estimate{OutcomeAccepted, SourceValidation}
	}
	if last.TerminalClass == "verdict" && last.StopReason == "complete" {
		return Estimate{OutcomeAccepted, SourceAgentVerdict}
	}
	if last.Path == PathHeadless && last.Status == "complete" {
		return Estimate{OutcomeCompleted, SourceHeadlessStatus}
	}
	return Estimate{OutcomeUnknown, SourceNoSignal}
}

// operatorEstimate reads the operator's own reports in a unit. A category below
// even odds is the classifier guessing, and a method that reports no probability
// stays in. A signal arrives at its measured operating point and is read when its
// measured precision is at least MinSignalPrecision. A reported failure outranks
// a correction. When it is the final operator message of the unit's last attempt
// the operator ended on a failure and the unit failed; earlier, the agent kept
// working on it and the unit was corrected.
func operatorEstimate(last RunFacts, corrections []Correction) (Estimate, bool) {
	var failureFinal, failure, correction, category bool
	for _, c := range corrections {
		switch {
		case c.Signal != "" && c.Precision < MinSignalPrecision:
			// measured below even odds: a flagged message is more likely not one
		case c.Signal == SignalReportedFailure && c.Final && c.RunID == last.RunID:
			failureFinal = true
		case c.Signal == SignalReportedFailure:
			failure = true
		case c.Signal == SignalCorrection:
			correction = true
		case c.Signal == "" && IsCorrection(c.Category) && (c.Probability <= 0 || c.Probability >= minCorrectionProbability):
			category = true
		}
	}
	switch {
	case failureFinal:
		return Estimate{OutcomeFailed, SourceOperatorReportedFailure}, true
	case failure:
		return Estimate{OutcomeCorrected, SourceOperatorReportedFailure}, true
	case category:
		return Estimate{OutcomeCorrected, SourceOperatorCorrection}, true
	case correction:
		return Estimate{OutcomeCorrected, SourceOperatorSignalCorrection}, true
	}
	return Estimate{}, false
}
