package runsignal

import (
	"encoding/json"
	"slices"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/tokenaccounting"
)

// callUnits splits one tool call into the commands it runs. A call that runs
// no command yields a single non-command unit so the call is still a fact.
// Codex exec JavaScript also returns its literal exec_command calls, which
// nested command results are matched against.
func callUnits(call *domain.ToolCallEventData, extraction runner.CommandExtraction) ([]commandUnit, []jsCommand) {
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
		return []commandUnit{{Kind: unitNone, Reason: reason, Last: true}}, nil
	}
	return checkedUnits(units, lexed), nil
}

func checkedUnits(units []commandUnit, lexed bool) []commandUnit {
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
// Each unit's Last/AllAnd is relative to its own command.
func jsUnits(src string) ([]commandUnit, []jsCommand) {
	commands, unresolved, other := jsExecCommands(src)
	var units []commandUnit
	for i, command := range commands {
		var parsed []commandUnit
		var lexed bool
		if len(command.Args) > 0 {
			parsed, lexed = argvUnits(command.Args)
		} else {
			parsed, lexed = shellUnits(command.Script)
		}
		units = append(units, scriptUnits(checkedUnits(parsed, lexed), i)...)
	}
	if len(commands) == 0 || unresolved > 0 || other {
		units = append(units, commandUnit{Kind: unitCode, Reason: "javascript beyond literal exec_command calls", Last: true, Command: codeCommand})
	}
	return units, commands
}

// codeCommand is the command index of a code unit: no command ran it.
const codeCommand = -1

func scriptUnits(units []commandUnit, command int) []commandUnit {
	for i := range units {
		units[i].FromScript, units[i].Command = true, command
	}
	return units
}

// oneResultCovers reports whether a call's own result is every unit's
// result: all units belong to one command (or to the code). One exit status
// cannot be attributed across independent commands.
func oneResultCovers(units []commandUnit) bool {
	for _, unit := range units {
		if unit.Command != units[0].Command {
			return false
		}
	}
	return true
}

// outputHolders names, per command, the one unit that carries the command's
// duration and output: one wall-clock span or one printed output cannot be
// split between segments that ran in sequence, and copying it to each would
// count it once per segment. The holder is the command's last unit that is
// neither a pipe reader nor a print (`status x | jq`, `wait x && echo done`
// hold time and output on status and wait, whose cost it is), else its last
// unit, so a window's time and output sum to what its commands did.
func outputHolders(units []commandUnit) map[int]int {
	holders, active := map[int]int{}, map[int]bool{}
	for i, unit := range units {
		passive := unit.Piped || len(unit.Tokens) > 0 && printCommands[unit.Tokens[0]]
		if !passive || !active[unit.Command] {
			holders[unit.Command] = i
			active[unit.Command] = active[unit.Command] || !passive
		}
	}
	return holders
}

var printCommands = map[string]bool{"echo": true, "printf": true}

// visibleTokens is the part of a command's printed output its script could
// receive: the bytes as estimated tokens (tokenaccounting.EstimateBytes),
// capped by the command's literal max_output_tokens (exec_command truncates
// beyond it).
func (p *callPlan) visibleTokens(command int, outputBytes int64) int64 {
	tokens := tokenaccounting.EstimateBytes(outputBytes)
	if command >= 0 && command < len(p.commands) && p.commands[command].MaxOutputTokens > 0 {
		tokens = min(tokens, p.commands[command].MaxOutputTokens)
	}
	return tokens
}

// unitOutcome gives a segment its own outcome. Only the segment that ends a
// chain inherits the call's result; earlier segments are known only when the
// whole && chain succeeded. Exit 1 of a search or comparison is no-match; a
// harness refusal means no segment ran.
func unitOutcome(data *domain.ToolResultEventData, unit commandUnit, tokens []string) string {
	call := ToolResultOutcome(data)
	switch {
	case call == OutcomeUnknown, call == OutcomeRefused:
		return call
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

// NestedCommandTool names the result of one command a Codex exec call ran
// (a rollout CommandExecution item; codecs.codexNestedCommandTool). It has no
// tool call: it reports on a command of the exec call it is joined to.
const NestedCommandTool = "command_execution"

// nestedLookback bounds how many earlier exec calls a late-completing command
// is matched against.
const nestedLookback = 64

// nestedResult is one command's own result inside an exec call.
type nestedResult struct {
	event       *domain.RunEvent
	data        *domain.ToolResultEventData
	argv        []string
	outputBytes int64
	command     int // the unit command it reports on, -1 until joined
}

// callPlan is one tool call ready to derive: its units, the result the agent
// saw, and the nested results that report what its commands did.
type callPlan struct {
	call     *domain.ToolCallEventData
	units    []commandUnit
	commands []jsCommand
	claimed  []bool
	result   *domain.RunEvent
	basis    string
	open     bool
	nested   []*nestedResult
}

// planCalls pairs every call with its result by provider correlation ID, else
// by stream ordinal, and joins nested command results to the exec call that
// ran them: the latest one whose literal command matches, else the latest
// still waiting for its result (commands complete before the exec returns),
// else the latest (a command that outlived its exec call). A nested result
// with no exec call before it becomes its own plan.
func planCalls(events []*domain.RunEvent, commandResolvers ...CommandResolver) map[*domain.RunEvent]*callPlan {
	callIDs := map[string]bool{}
	results := map[string]*domain.RunEvent{}
	var ordinal []*domain.RunEvent
	for _, event := range events {
		switch data := eventData(event).(type) {
		case *domain.ToolCallEventData:
			if data.ToolCallID != "" {
				callIDs[data.ToolCallID] = true
			}
		case *domain.ToolResultEventData:
			if data.ToolCallID == "" {
				ordinal = append(ordinal, event)
			} else if _, seen := results[data.ToolCallID]; !seen {
				// A duplicated result keeps the first copy.
				results[data.ToolCallID] = event
			}
		}
	}
	plans := map[*domain.RunEvent]*callPlan{}
	byCallID := map[string]*callPlan{}
	var scripts []*callPlan
	for _, event := range events {
		switch data := eventData(event).(type) {
		case *domain.ToolCallEventData:
			// A tool call is one invocation however many copies of its event
			// were recorded (an importer once appended the events twice).
			if data.ToolCallID != "" && byCallID[data.ToolCallID] != nil {
				continue
			}
			plan := &callPlan{call: data, basis: "unpaired", open: true}
			plan.units, plan.commands = callUnits(data, commandInput(data, commandResolvers...))
			plan.claimed = make([]bool, len(plan.commands))
			if data.ToolCallID != "" {
				byCallID[data.ToolCallID] = plan
				if plan.result = results[data.ToolCallID]; plan.result != nil {
					plan.basis = "call-id"
				}
			} else if len(ordinal) > 0 {
				plan.result, ordinal, plan.basis = ordinal[0], ordinal[1:], "ordinal"
			}
			if _, script := jsExecInput(data); script {
				scripts = append(scripts, plan)
			}
			plans[event] = plan
		case *domain.ToolResultEventData:
			if plan := byCallID[data.ToolCallID]; plan != nil && plan.result == event {
				plan.open = false
			}
			if data.ToolName != NestedCommandTool || data.ToolCallID == "" || callIDs[data.ToolCallID] {
				continue
			}
			nested := decodeNested(event, data)
			if parent := joinNested(scripts, nested); parent != nil {
				parent.nested = append(parent.nested, nested)
				continue
			}
			units, lexed := argvUnits(nested.argv)
			nested.command = 0
			plans[event] = &callPlan{call: &domain.ToolCallEventData{ToolName: NestedCommandTool, ToolCallID: data.ToolCallID}, units: checkedUnits(units, lexed), result: event, basis: "call-id", nested: []*nestedResult{nested}}
		}
	}
	return plans
}

func eventData(event *domain.RunEvent) domain.EventPayload {
	if event == nil {
		return nil
	}
	return event.Data
}

func decodeNested(event *domain.RunEvent, data *domain.ToolResultEventData) *nestedResult {
	var record struct {
		Command     []string `json:"command"`
		OutputBytes int64    `json:"output_bytes"`
	}
	_ = json.Unmarshal([]byte(data.RawOutputJSON), &record)
	return &nestedResult{event: event, data: data, argv: record.Command, outputBytes: record.OutputBytes, command: -1}
}

func joinNested(scripts []*callPlan, nested *nestedResult) *callPlan {
	var open, latest *callPlan
	for i := len(scripts) - 1; i >= 0 && i >= len(scripts)-nestedLookback; i-- {
		plan := scripts[i]
		for k, command := range plan.commands {
			if !plan.claimed[k] && command.runs(nested.argv) {
				plan.claimed[k], nested.command = true, k
				return plan
			}
		}
		if open == nil && plan.open {
			open = plan
		}
		if latest == nil {
			latest = plan
		}
	}
	if open != nil {
		return open
	}
	return latest
}

// runs reports whether an executed argv is this literal command: the harness
// runs shell text as [shell, -lc, text] and argv as given.
func (c jsCommand) runs(argv []string) bool {
	if len(c.Args) > 0 {
		return slices.Equal(c.Args, argv)
	}
	return len(argv) > 0 && strings.TrimSpace(argv[len(argv)-1]) == strings.TrimSpace(c.Script)
}

// joinedUnits adds a unit set for every nested command that matched no
// literal exec_command (one the script built at run time) and indexes the
// nested results by the command they report on.
func (p *callPlan) joinedUnits() ([]commandUnit, map[int]*nestedResult) {
	units := p.units
	evidence := map[int]*nestedResult{}
	next := len(p.commands)
	for _, nested := range p.nested {
		if nested.command < 0 {
			parsed, lexed := argvUnits(nested.argv)
			units = append(slices.Clip(units), scriptUnits(checkedUnits(parsed, lexed), next)...)
			nested.command = next
			next++
		}
		if _, seen := evidence[nested.command]; !seen {
			evidence[nested.command] = nested
		}
	}
	return units, evidence
}
