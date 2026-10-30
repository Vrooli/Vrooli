package runsignal

import (
	"strings"

	"agent-manager/internal/domain"
)

// comparableFingerprintFact is a command fact two calls can be compared on.
func comparableFingerprintFact(fact InvocationFact) bool {
	return fact.Fingerprint != "" && fact.Ownership != OwnershipUnparseable && fact.Ownership != OwnershipCompoundUnresolved
}

// sameCommand is the same command identity: program and subcommand words.
func sameCommand(a, b InvocationFact) bool {
	return a.CommandPath != "" && a.CommandPath == b.CommandPath
}

// sameFamily is the same program and subcommand family: one command path is
// the other, or a sub-command of it.
func sameFamily(a, b InvocationFact) bool {
	if a.CommandPath == "" || b.CommandPath == "" {
		return false
	}
	return a.CommandPath == b.CommandPath || strings.HasPrefix(a.CommandPath, b.CommandPath+" ") || strings.HasPrefix(b.CommandPath, a.CommandPath+" ")
}

// factTokens is the words of the command a fact was made from, program first.
func factTokens(fact InvocationFact, byID map[string]*domain.RunEvent) []string {
	call := toolCallData(byID[fact.CallEventID])
	if call == nil {
		return nil
	}
	units := callUnits(call, commandInput(call))
	if fact.SegmentIndex < len(units) && units[fact.SegmentIndex].Kind == unitCommand {
		return units[fact.SegmentIndex].Tokens
	}
	return nil
}

func isFlag(token string) bool { return strings.HasPrefix(token, "-") && token != "-" }

// isHelpProbe is a call that asks a command for its usage.
func isHelpProbe(fact InvocationFact, byID map[string]*domain.RunEvent) bool {
	if strings.Contains(strings.ToLower(fact.FailureSignature), "usage") {
		return true
	}
	for _, token := range factTokens(fact, byID) {
		if token == "--help" || token == "-h" || token == "help" {
			return true
		}
	}
	return false
}

// isPollingCommand is a read-only status or listing command, which is repeated
// by design.
func isPollingCommand(path string) bool {
	words := strings.Fields(path)
	if len(words) == 0 {
		return false
	}
	switch last := words[len(words)-1]; last {
	case "status", "list", "ps", "ls":
		return true
	}
	return strings.HasSuffix(path, "run identity") || strings.HasSuffix(path, "effort signals")
}

// eventPositions indexes events by ID, to find what lies between two of them.
func eventPositions(events []*domain.RunEvent) map[string]int {
	out := make(map[string]int, len(events))
	for i, event := range events {
		if event != nil {
			out[event.ID.String()] = i
		}
	}
	return out
}

// eventsBetween is the events strictly between the events with the given IDs.
func eventsBetween(events []*domain.RunEvent, pos map[string]int, a, b string) []*domain.RunEvent {
	from, okA := pos[a]
	to, okB := pos[b]
	if !okA || !okB || to <= from+1 {
		return nil
	}
	return events[from+1 : to]
}

func compactedBetween(events []*domain.RunEvent) bool {
	for _, event := range events {
		if event == nil {
			continue
		}
		if _, ok := event.Data.(*domain.CompactionEventData); ok || event.EventType == domain.EventTypeCompaction {
			return true
		}
	}
	return false
}

func fileChangedIn(events []*domain.RunEvent) bool {
	for _, event := range events {
		if len(changedPaths(event)) > 0 {
			return true
		}
	}
	return false
}

// failed is a call that went wrong: it failed and was not interrupted or killed
// (exit 128 and up). Programs whose exit 1 means "no match" are already
// normalised to success when the fact is built (noMatchPrograms).
// failed is an error result. A non-zero exit of a command the manifest declares
// a verify command (lint, acceptance checks) is its verdict, not a failure.
func failed(fact InvocationFact) bool {
	return fact.Outcome == OutcomeFailure && fact.SemanticsKind != "verify" && (fact.ExitCode == nil || *fact.ExitCode < 128)
}

// sameProgram is the same program, or for a project command the same scenario:
// the unit an agent keeps failing at while it guesses arguments.
func sameProgram(a, b InvocationFact) bool { return a.Executable != "" && a.Executable == b.Executable }

// finishedBefore says the first call had its result before the second began.
// The same command issued twice before either returns is one request emitted
// twice, not a deliberate repeat. Facts without events in the window count as
// finished.
func finishedBefore(a, b InvocationFact, pos map[string]int) bool {
	result, okR := pos[a.ResultEventID]
	call, okC := pos[b.CallEventID]
	return !okR || !okC || result < call
}

// resultReportedFailure is a call whose result says it did not succeed, even
// when no exit code makes the fact's outcome known.
func resultReportedFailure(fact InvocationFact, byID map[string]*domain.RunEvent) bool {
	if fact.Outcome == OutcomeFailure {
		return true
	}
	result, ok := byID[fact.ResultEventID]
	if !ok {
		return false
	}
	data, ok := result.Data.(*domain.ToolResultEventData)
	return ok && !data.Success
}
