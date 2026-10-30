package runsignal

import (
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

// callUnits splits one tool call into the commands it runs. A call that runs
// no command yields a single non-command unit so the call is still a fact.
func callUnits(call *domain.ToolCallEventData, extraction runner.CommandExtraction) []commandUnit {
	if src, ok := jsExecInput(call); ok {
		return jsUnits(src)
	}
	var units []commandUnit
	var lexed bool
	switch {
	case extraction.LiteralArgs:
		units, lexed = argvUnits(extraction.Args)
	case extraction.Command != "":
		units, lexed = shellUnits(extraction.Command)
	default:
		reason := extraction.Reason
		if reason == "" {
			reason = "no declared command field"
		}
		return []commandUnit{{Kind: unitNone, Reason: reason, Last: true}}
	}
	if !lexed {
		return []commandUnit{{Kind: unitUnparseable, Reason: "unterminated quote or substitution", Last: true}}
	}
	if len(units) == 0 {
		return []commandUnit{{Kind: unitNone, Reason: "no command word", Last: true}}
	}
	return units
}

// jsUnits reads Codex exec JavaScript: one set of command units per literal
// exec_command call, plus a code unit when the script does anything else.
func jsUnits(src string) []commandUnit {
	commands, unresolved, other := jsExecCommands(src)
	var units []commandUnit
	for _, command := range commands {
		var parsed []commandUnit
		var lexed bool
		if len(command.Args) > 0 {
			parsed, lexed = argvUnits(command.Args)
		} else {
			parsed, lexed = shellUnits(command.Script)
		}
		if !lexed {
			units = append(units, commandUnit{Kind: unitUnparseable, Reason: "unterminated quote or substitution", Last: true})
			continue
		}
		for i := range parsed {
			parsed[i].FromScript = true
		}
		units = append(units, parsed...)
	}
	code := len(commands) == 0 || unresolved > 0 || other
	if code {
		units = append(units, commandUnit{Kind: unitCode, Reason: "javascript beyond literal exec_command calls", Last: true})
	}
	// One exit status cannot be attributed across independent commands.
	if len(commands) > 1 || code && len(commands) > 0 {
		for i := range units {
			units[i].Last, units[i].AllAnd = false, false
		}
	}
	return units
}

// unitOutcome gives a segment its own outcome. Only the segment that ends a
// chain inherits the call's result; earlier segments are known only when the
// whole && chain succeeded. Exit 1 of a search or comparison is no-match.
func unitOutcome(data *domain.ToolResultEventData, unit commandUnit, tokens []string) string {
	call := ToolResultOutcome(data)
	switch {
	case call == OutcomeUnknown:
		return OutcomeUnknown
	case unit.Last:
		if call == OutcomeFailure && data.ExitCode != nil && *data.ExitCode == 1 && len(tokens) > 0 && noMatchPrograms[tokens[0]] {
			return OutcomeNoMatch
		}
		return call
	case call == OutcomeSuccess && unit.AllAnd:
		return OutcomeSuccess
	}
	return OutcomeUnknown
}

// dynamicProgram reports a command name that only the shell could resolve.
func dynamicProgram(program string) bool {
	return strings.HasPrefix(program, "$") || strings.HasPrefix(program, "`") || strings.Contains(program, "$(")
}
