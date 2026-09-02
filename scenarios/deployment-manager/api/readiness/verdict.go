package readiness

import (
	"fmt"
	"strings"
	"time"
)

type SignalStatus string

const (
	SignalPassed      SignalStatus = "passed"
	SignalFailed      SignalStatus = "failed"
	SignalUnavailable SignalStatus = "unavailable"
	SignalUnknown     SignalStatus = "unknown"
	SignalStale       SignalStatus = "stale"
)

type Signal struct {
	ItemID     string       `json:"item_id"`
	Status     SignalStatus `json:"status"`
	Source     string       `json:"source"`
	Commit     string       `json:"commit,omitempty"`
	RunID      string       `json:"run_id,omitempty"`
	ObservedAt time.Time    `json:"observed_at"`
	Detail     string       `json:"detail,omitempty"`
}

type Finding struct {
	ItemID   string       `json:"item_id"`
	Severity string       `json:"severity"`
	Status   SignalStatus `json:"status"`
	Message  string       `json:"message"`
	Signal   Signal       `json:"signal"`
}

type Verdict struct {
	Scenario  string    `json:"scenario"`
	Commit    string    `json:"commit"`
	Approved  bool      `json:"approved"`
	CheckedAt time.Time `json:"checked_at"`
	Findings  []Finding `json:"findings"`
}

// Aggregate joins producer-owned signals to the declarative checklist. A
// missing signal is a finding, never an implicit pass. Required failures and
// unknowns with required impact withhold approval; advisory failures remain
// visible without blocking; uncheckable checks stay as agent work.
func Aggregate(scenario, commit string, checklist Checklist, signals []Signal, now time.Time) (Verdict, error) {
	if err := checklist.Validate(); err != nil {
		return Verdict{}, err
	}
	if strings.TrimSpace(scenario) == "" || strings.TrimSpace(commit) == "" {
		return Verdict{}, fmt.Errorf("scenario and commit are required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	byItem := make(map[string]Signal, len(signals))
	knownItems := make(map[string]struct{}, len(checklist.Items))
	for _, item := range checklist.Items {
		knownItems[item.ID] = struct{}{}
	}
	for _, signal := range signals {
		if strings.TrimSpace(signal.ItemID) == "" || strings.TrimSpace(signal.Source) == "" || signal.ObservedAt.IsZero() {
			return Verdict{}, fmt.Errorf("signal requires item_id, source, and observed_at")
		}
		if _, ok := knownItems[signal.ItemID]; !ok {
			return Verdict{}, fmt.Errorf("signal %q does not belong to checklist", signal.ItemID)
		}
		switch signal.Status {
		case SignalPassed, SignalFailed, SignalUnavailable, SignalUnknown, SignalStale:
		default:
			return Verdict{}, fmt.Errorf("signal %q has invalid status %q", signal.ItemID, signal.Status)
		}
		if _, exists := byItem[signal.ItemID]; exists {
			return Verdict{}, fmt.Errorf("duplicate signal for checklist item %q", signal.ItemID)
		}
		if signal.Commit != "" && signal.Commit != commit {
			signal.Status = SignalStale
			if signal.Detail == "" {
				signal.Detail = fmt.Sprintf("signal belongs to commit %s, evaluated commit is %s", signal.Commit, commit)
			}
		}
		byItem[signal.ItemID] = signal
	}
	verdict := Verdict{Scenario: scenario, Commit: commit, CheckedAt: now.UTC()}
	for _, item := range checklist.Items {
		signal, ok := byItem[item.ID]
		if !ok {
			signal = Signal{ItemID: item.ID, Status: SignalUnknown, Source: "readiness-aggregator", ObservedAt: now.UTC(), Detail: "no producer signal was returned"}
		}
		if signal.Status == SignalPassed {
			continue
		}
		severity := "advisory"
		if item.CleanRequirement == Required {
			severity = "error"
		} else if item.CleanRequirement == Uncheckable {
			severity = "unknown"
		}
		verdict.Findings = append(verdict.Findings, Finding{ItemID: item.ID, Severity: severity, Status: signal.Status, Message: findingMessage(item, signal), Signal: signal})
	}
	verdict.Approved = true
	for _, finding := range verdict.Findings {
		if finding.Severity == "error" {
			verdict.Approved = false
		}
	}
	return verdict, nil
}

func findingMessage(item Item, signal Signal) string {
	detail := strings.TrimSpace(signal.Detail)
	if detail == "" {
		detail = string(signal.Status)
	}
	return fmt.Sprintf("%s: %s", item.Title, detail)
}
