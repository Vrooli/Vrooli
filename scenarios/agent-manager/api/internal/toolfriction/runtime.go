package toolfriction

import (
	"sort"
	"strings"
	"time"

	"agent-manager/internal/runsignal"
)

const (
	RuntimeStoppedByCheck      = "stopped_by_check"
	RuntimeWaitingOnValidation = "waiting_on_validation"
	RuntimeAgentManagerFailure = "agent_manager_failure"
	RuntimeRework              = "rework"
	RuntimeSandboxEnvironment  = "sandbox_environment"
	RuntimeBuilding            = "building"
	RuntimeGuardCheck          = "guard_check"

	RuntimeResponseStopped    = "stopped"
	RuntimeResponseWorkaround = "workaround"
	RuntimeResponseAsked      = "asked"
	RuntimeResponseSucceeded  = "succeeded"
	RuntimeResponseUnknown    = "unknown"
	RuntimeOutcomeBlocked     = "blocked"
)

// RuntimeSignal is a bounded fact extracted from a run's event stream. It
// contains no raw event payload; Text is limited to assistant-authored text
// used only while classifying the response to a blocked check.
type RuntimeSignal struct {
	At            time.Time
	Kind          string
	Check         string
	CheckID       string
	EvaluatorID   string
	ReceiptID     string
	CommandPath   string
	Owner         string
	Outcome       string
	SemanticsKind string
	DurationMS    int64
	DurationKnown bool
	DurationBasis string
	Retry         bool
	Text          string
}

// RuntimeRun combines durable run identity/lifecycle with a conservative
// time-accounting result and signals derived from its retained events.
type RuntimeRun struct {
	ID, TaskID, Status, TerminalClass, StopReason string
	Imported                                      bool
	StartedAt, EndedAt                            *time.Time
	Time                                          runsignal.TimeAccounting
	HasTime                                       bool
	Events                                        []RuntimeSignal
}

type RuntimeWindow struct {
	// Since and Until select the run cohort upstream. The current cohort uses a
	// half-open anchor range; it does not clip run durations to this interval.
	Since time.Time `json:"since"`
	Until time.Time `json:"until"`
}

type RuntimeCoverage struct {
	State                   string `json:"state"`
	Reason                  string `json:"reason,omitempty"`
	RunsObserved            int    `json:"runsObserved"`
	RunsWithDuration        int    `json:"runsWithDuration"`
	RunsWithEvents          int    `json:"runsWithEvents"`
	ImportedRuns            int    `json:"importedRuns"`
	ImportedRunsIncluded    bool   `json:"importedRunsIncluded"`
	UnknownRuns             int    `json:"unknownRuns"`
	UnclassifiedBlockedRuns int    `json:"unclassifiedBlockedRuns"`
	UnknownDurationMS       int64  `json:"unknownDurationMs"`
}

type RuntimeTotals struct {
	TotalDurationMS   int64 `json:"totalDurationMs"`
	KnownDurationMS   int64 `json:"knownDurationMs"`
	UnknownDurationMS int64 `json:"unknownDurationMs"`
}

type RuntimeLoss struct {
	Category      string `json:"category"`
	DurationMS    int64  `json:"durationMs"`
	Runs          int    `json:"runs"`
	EvidenceCount int    `json:"evidenceCount"`
}

type BlockedCheckResponses struct {
	Stopped    int `json:"stopped"`
	Workaround int `json:"workaround"`
	Asked      int `json:"asked"`
	Succeeded  int `json:"succeeded"`
	Unknown    int `json:"unknown"`
}

type BlockedCheckOutcomes struct {
	Blocked int `json:"blocked"`
	Allowed int `json:"allowed"`
	Unknown int `json:"unknown"`
}

type BlockedCheck struct {
	CheckID                    string                `json:"checkId"`
	Check                      string                `json:"check"`
	Runs                       int                   `json:"runs"`
	Occurrences                int                   `json:"occurrences"`
	BlockedRunIDs              []string              `json:"blockedRunIds"`
	Outcomes                   BlockedCheckOutcomes  `json:"outcomes"`
	Responses                  BlockedCheckResponses `json:"responses"`
	EnforcementDurationMS      int64                 `json:"enforcementDurationMs"`
	EnforcementDurationKnown   int                   `json:"enforcementDurationKnown"`
	EnforcementDurationUnknown int                   `json:"enforcementDurationUnknown"`
	RunIDs                     []string              `json:"runIds"`
}

// RuntimeReport is a duration-conserving view of observed run-time. Category
// duration is assigned only from measured state or operation durations;
// uncovered intervals remain unknown instead of being presented as work.
type RuntimeReport struct {
	Window        RuntimeWindow   `json:"window"`
	Coverage      RuntimeCoverage `json:"coverage"`
	Totals        RuntimeTotals   `json:"totals"`
	Categories    []RuntimeLoss   `json:"categories"`
	BlockedChecks []BlockedCheck  `json:"blockedChecks"`
}

// AnalyzeRuntime summarizes one already-window-filtered set of Agent Manager
// and imported runs. A duration is classified only when a measured operation
// signal supports that category. Other measured run time remains unknown to
// the friction taxonomy, even when its broad model/tool state is known.
func AnalyzeRuntime(runs []RuntimeRun, window RuntimeWindow) RuntimeReport {
	report := RuntimeReport{
		Window:   window,
		Coverage: RuntimeCoverage{State: "available", ImportedRunsIncluded: true},
		Categories: []RuntimeLoss{
			{Category: RuntimeStoppedByCheck},
			{Category: RuntimeWaitingOnValidation},
			{Category: RuntimeAgentManagerFailure},
			{Category: RuntimeRework},
			{Category: RuntimeSandboxEnvironment},
			{Category: RuntimeBuilding},
		},
	}
	categoryIndex := map[string]int{}
	categoryRuns := map[string]map[string]bool{}
	for i, category := range report.Categories {
		categoryIndex[category.Category] = i
	}
	blocked := map[string]*BlockedCheck{}
	for _, run := range runs {
		report.Coverage.RunsObserved++
		if run.Imported {
			report.Coverage.ImportedRuns++
		}
		collectBlockedChecks(blocked, run, &report.Coverage)
		total, hasDuration := runtimeDuration(run)
		if !hasDuration {
			report.Coverage.UnknownRuns++
			continue
		}
		report.Coverage.RunsWithDuration++
		report.Totals.TotalDurationMS += total
		if len(run.Events) > 0 {
			report.Coverage.RunsWithEvents++
		} else {
			report.Coverage.UnknownRuns++
		}

		// Category evidence pools are intentionally narrower than total elapsed
		// time. Model-generating and residual tool time can be measured without
		// revealing whether the agent was building, investigating, or doing
		// another kind of work.
		remainingToolMS := total
		remainingUnattributedMS := total
		remainingTotalMS := total
		if run.HasTime {
			remainingToolMS = min(total, max(int64(0), run.Time.ToolExecutingMS))
			remainingUnattributedMS = min(total, max(int64(0), run.Time.UnattributableMS))
		} else {
			if len(run.Events) == 0 {
				report.Totals.UnknownDurationMS += total
				report.Coverage.UnknownDurationMS += total
				continue
			}
		}
		classified := int64(0)
		for _, signal := range sortedSignals(run.Events) {
			category := runtimeCategory(signal)
			if category == "" {
				continue
			}
			duration := int64(0)
			if signal.DurationKnown && signal.DurationMS > 0 {
				if run.HasTime {
					switch signal.DurationBasis {
					case "unattributed":
						duration = min(signal.DurationMS, remainingUnattributedMS)
						remainingUnattributedMS -= duration
					default:
						duration = min(signal.DurationMS, remainingToolMS)
						remainingToolMS -= duration
					}
				} else {
					duration = min(signal.DurationMS, remainingTotalMS)
				}
				duration = min(duration, remainingTotalMS)
				remainingTotalMS -= duration
				classified += duration
			}
			addRuntimeEvidence(&report.Categories[categoryIndex[category]], categoryRuns, category, run.ID, duration)
		}
		remaining := max(int64(0), total-classified)
		report.Totals.KnownDurationMS += classified
		report.Totals.UnknownDurationMS += remaining
		report.Coverage.UnknownDurationMS += remaining
	}
	if report.Coverage.RunsObserved == 0 {
		report.Coverage.State = "unavailable"
		report.Coverage.Reason = "no runs were retained in the selected window"
	} else if report.Coverage.UnknownRuns > 0 || report.Totals.UnknownDurationMS > 0 {
		report.Coverage.State = "partial"
		report.Coverage.Reason = "some runs or time intervals lack enough retained evidence for classification"
	}
	for _, entry := range blocked {
		sort.Strings(entry.RunIDs)
		sort.Strings(entry.BlockedRunIDs)
		report.BlockedChecks = append(report.BlockedChecks, *entry)
	}
	sort.Slice(report.BlockedChecks, func(i, j int) bool { return report.BlockedChecks[i].Check < report.BlockedChecks[j].Check })
	return report
}

func runtimeDuration(run RuntimeRun) (int64, bool) {
	if run.StartedAt == nil || run.EndedAt == nil || !run.EndedAt.After(*run.StartedAt) {
		return 0, false
	}
	return run.EndedAt.Sub(*run.StartedAt).Milliseconds(), true
}

func sortedSignals(events []RuntimeSignal) []RuntimeSignal {
	ordered := append([]RuntimeSignal(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].At.Before(ordered[j].At) })
	return ordered
}

func runtimeCategory(signal RuntimeSignal) string {
	if signal.Kind == RuntimeAgentManagerFailure || signal.Kind == RuntimeSandboxEnvironment || signal.Kind == RuntimeRework {
		return signal.Kind
	}
	if signal.Kind != "invocation" {
		if signal.Kind == RuntimeGuardCheck && signal.Outcome == RuntimeOutcomeBlocked {
			return RuntimeStoppedByCheck
		}
		return ""
	}
	if signal.Retry {
		return RuntimeRework
	}
	if signal.SemanticsKind == "verify" {
		if strings.Contains(signal.CommandPath, "wait") {
			return RuntimeWaitingOnValidation
		}
		if signal.Outcome == runsignal.OutcomeFailure {
			return RuntimeStoppedByCheck
		}
		return ""
	}
	if signal.Outcome != runsignal.OutcomeFailure {
		return ""
	}
	switch {
	case signal.Owner == "agent-manager" || strings.HasPrefix(signal.CommandPath, "agent-manager "):
		return RuntimeAgentManagerFailure
	case signal.Owner == "workspace-sandbox" || strings.HasPrefix(signal.CommandPath, "workspace-sandbox "):
		return RuntimeSandboxEnvironment
	}
	return ""
}

func addRuntimeEvidence(category *RuntimeLoss, runs map[string]map[string]bool, categoryName, runID string, duration int64) {
	category.DurationMS += duration
	category.EvidenceCount++
	if runs[categoryName] == nil {
		runs[categoryName] = map[string]bool{}
	}
	if !runs[categoryName][runID] {
		runs[categoryName][runID] = true
		category.Runs++
	}
}

func collectBlockedChecks(groups map[string]*BlockedCheck, run RuntimeRun, coverage *RuntimeCoverage) {
	type checkEvidence struct {
		label        string
		failedAt     time.Time
		succeededAt  time.Time
		guardDenials []RuntimeSignal
	}
	checks := map[string]*checkEvidence{}
	for _, signal := range sortedSignals(run.Events) {
		if signal.Kind == RuntimeGuardCheck {
			checkID := strings.TrimSpace(signal.CheckID)
			groupKey := checkID
			if groupKey == "" {
				groupKey = "guard:unknown"
			}
			check := checks[groupKey]
			if check == nil {
				label := strings.TrimSpace(signal.Check)
				if checkID != "" {
					label = checkID
				}
				if label == "" {
					label = "guard check (identity unavailable)"
				}
				check = &checkEvidence{label: label}
				checks[groupKey] = check
			}
			check.guardDenials = append(check.guardDenials, signal)
			continue
		}
		if signal.Kind != "invocation" || signal.SemanticsKind != "verify" {
			continue
		}
		checkID := signal.CheckID
		groupKey := checkID
		if groupKey == "" {
			// Keep command-level observations together for reporting, but never
			// promote the display label to an evaluator identity.
			groupKey = "label:" + signal.Check
		}
		check := checks[groupKey]
		if check == nil {
			check = &checkEvidence{label: signal.Check}
			checks[groupKey] = check
		}
		switch signal.Outcome {
		case runsignal.OutcomeFailure:
			check.failedAt = signal.At
		case runsignal.OutcomeSuccess:
			check.succeededAt = signal.At
		}
	}
	for groupKey, check := range checks {
		if check.failedAt.IsZero() && len(check.guardDenials) == 0 {
			continue
		}
		checkID := ""
		if !strings.HasPrefix(groupKey, "label:") && groupKey != "guard:unknown" {
			checkID = groupKey
		}
		entryKey := groupKey
		entry := groups[entryKey]
		if entry == nil {
			entry = &BlockedCheck{CheckID: checkID, Check: check.label, BlockedRunIDs: make([]string, 0)}
			groups[entryKey] = entry
		}
		if len(check.guardDenials) > 0 {
			if checkID == "" {
				coverage.UnclassifiedBlockedRuns++
			}
			if !containsString(entry.RunIDs, run.ID) {
				entry.Runs++
				entry.RunIDs = append(entry.RunIDs, run.ID)
			}
			for _, denial := range check.guardDenials {
				entry.Occurrences++
				switch denial.Outcome {
				case RuntimeOutcomeBlocked:
					entry.Outcomes.Blocked++
					if !containsString(entry.BlockedRunIDs, run.ID) {
						entry.BlockedRunIDs = append(entry.BlockedRunIDs, run.ID)
					}
				case "allowed":
					entry.Outcomes.Allowed++
				default:
					entry.Outcomes.Unknown++
				}
				if denial.DurationKnown {
					entry.EnforcementDurationKnown++
					entry.EnforcementDurationMS += max(int64(0), denial.DurationMS)
				} else {
					entry.EnforcementDurationUnknown++
				}
				response := guardResponseAfter(run, denial.At)
				incrementResponse(&entry.Responses, response)
			}
			continue
		}
		entry.Runs++
		entry.Occurrences++
		entry.RunIDs = append(entry.RunIDs, run.ID)
		response := RuntimeResponseUnknown
		if checkID == "" {
			coverage.UnclassifiedBlockedRuns++
		}
		if check.succeededAt.After(check.failedAt) {
			response = RuntimeResponseSucceeded
		} else if run.TerminalClass == "verdict" && run.StopReason == "blocked" {
			response = RuntimeResponseStopped
		} else if responseTextAfter(run.Events, check.failedAt, "workaround", "bypass", "skip this check") {
			response = RuntimeResponseWorkaround
		} else if responseAskedAfter(run.Events, check.failedAt) {
			response = RuntimeResponseAsked
		}
		incrementResponse(&entry.Responses, response)
	}
}

func guardResponseAfter(run RuntimeRun, at time.Time) string {
	if run.TerminalClass == "verdict" && run.StopReason == "blocked" {
		return RuntimeResponseStopped
	}
	if responseTextAfter(run.Events, at, "workaround", "bypass", "skip this check") {
		return RuntimeResponseWorkaround
	}
	if responseAskedAfter(run.Events, at) {
		return RuntimeResponseAsked
	}
	return RuntimeResponseUnknown
}

func incrementResponse(responses *BlockedCheckResponses, response string) {
	switch response {
	case RuntimeResponseStopped:
		responses.Stopped++
	case RuntimeResponseWorkaround:
		responses.Workaround++
	case RuntimeResponseAsked:
		responses.Asked++
	case RuntimeResponseSucceeded:
		responses.Succeeded++
	default:
		responses.Unknown++
	}
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func responseTextAfter(events []RuntimeSignal, after time.Time, phrases ...string) bool {
	for _, signal := range events {
		if signal.Kind != "message" || !signal.At.After(after) {
			continue
		}
		text := strings.ToLower(signal.Text)
		for _, phrase := range phrases {
			if strings.Contains(text, phrase) {
				return true
			}
		}
	}
	return false
}

func responseAskedAfter(events []RuntimeSignal, after time.Time) bool {
	for _, signal := range events {
		if signal.Kind != "message" || !signal.At.After(after) {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(signal.Text))
		if strings.Contains(text, "can you ") || strings.Contains(text, "could you ") || strings.Contains(text, "please choose") || strings.Contains(text, "which option") || strings.Contains(text, "?") {
			return true
		}
	}
	return false
}
