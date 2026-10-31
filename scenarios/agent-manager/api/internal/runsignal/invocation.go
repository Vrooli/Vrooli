// Package runsignal derives bounded, deterministic evidence from durable run events.
package runsignal

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/availability"

	"agent-manager/internal/domain"
	"agent-manager/internal/tokenaccounting"
)

// InvocationFactVersion is pinned into every derivation so a rebuild can be
// compared honestly when classifier behavior evolves.
// v32 gives a command's output to the segment that holds its duration: its
// printed bytes and its result tokens (what its nested result says it could
// show, capped by its exec_command max_output_tokens; the script's own unit
// takes the rest of the result). Other segments of the command hold none and
// their tokens are unknown, so a window counts each output once.
// v31 gives a command's duration to one of its segments (not to each), and
// reads each segment's words from its own pipeline of a nested Codex command,
// so segments stop sharing another segment's command text and poll key.
// v30 derives compaction events from the Claude and Codex transcripts, so
// residency segments and context tax stop at each compaction, and reads verify
// semantics from the control-plane manifest and durable-run exceptions.
// v29 estimates a result's tokens from the text blocks of structured results
// (Codex exec output was counted as empty), so output weight and context tax
// see Codex commands.
// v28 keeps the target of a read command in its poll key (only volatile
// values are dropped), attributes a compound call's notice to the segment that
// printed it, and counts HTTP bypass only on API ports and non-health routes.
// v27 stores the tool friction features of each command fact and keeps the
// next word of a manifest group as its command path (`test-genie runs wait`).
// v26 joins Codex rollout CommandExecution results to the exec call that ran
// them (per-command outcome, exit code, duration and output size), reports
// harness refusals as refused, splits the call's result tokens across its
// commands by output size, counts residency in provider requests until the
// next compaction, and takes capability from the one tool-kind table.
// v21 reads command identity from a tolerant shell lexer (heredocs, quotes,
// wrappers) and Codex exec JavaScript, derives IntentClass from CommandPath
// instead of argument shape, gives each segment its own outcome, and reports
// exit 1 of rg/grep/diff/test as no-match.
// v20 derives every token quantity from the one usage account
// (AccountUsage). v17 apportions per-call token quantities across the
// segments of a compound command, so a SUM over facts reconciles to the run.
const InvocationFactVersion = "invocation-fact.v32"

// Outcome vocabulary of an InvocationFact.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
	OutcomeUnknown = "unknown"
	// OutcomeNoMatch is exit 1 of a search or comparison (rg, grep, diff,
	// test): the command ran and found nothing, which is not a failure.
	OutcomeNoMatch = "no-match"
	// OutcomeRefused is a call the harness refused before running it (a
	// Claude Code `<tool_use_error>Blocked` or PreToolUse hook error).
	OutcomeRefused = "refused"
)

// PairingCommandExecution is the pairing basis of a fact whose outcome comes
// from the command's own nested result rather than the call's result.
const PairingCommandExecution = "command-execution"

// FailureSignatureMaxLength bounds retained failure vocabulary. Signatures are
// controlled class labels, never copied error text.
const FailureSignatureMaxLength = 64

// InvocationFact is a bounded, redacted interpretation of one durable tool
// call/result pair. It intentionally excludes raw input and output bodies.
// ResultEventID is the result of the fact's command: a nested command result
// when one reports on it (PairingCommandExecution), else the call's result.
// OutputEventID is the call's own result, the output the agent saw.
type InvocationFact struct {
	Version                     string                `json:"version"`
	CallEventID                 string                `json:"callEventId"`
	ResultEventID               string                `json:"resultEventId,omitempty"`
	OutputEventID               string                `json:"outputEventId,omitempty"`
	ToolCallID                  string                `json:"toolCallId,omitempty"`
	ToolName                    string                `json:"toolName"`
	Capability                  string                `json:"capability"`
	IntentClass                 string                `json:"intentClass"`
	Executable                  string                `json:"executable,omitempty"`
	Wrapper                     string                `json:"wrapper,omitempty"`
	CommandPath                 string                `json:"commandPath,omitempty"`
	ProgramID                   string                `json:"programId,omitempty"`
	SkillID                     string                `json:"skillId,omitempty"`
	Ownership                   string                `json:"ownership"`
	OwnershipReason             string                `json:"ownershipReason,omitempty"`
	SegmentIndex                int                   `json:"segmentIndex"`
	SegmentCount                int                   `json:"segmentCount"`
	CatalogSnapshot             string                `json:"catalogSnapshot,omitempty"`
	Outcome                     string                `json:"outcome"`
	ExitCode                    *int                  `json:"exitCode,omitempty"`
	DurationMS                  *int64                `json:"durationMs,omitempty"`
	PairingBasis                string                `json:"pairingBasis"`
	FailureSignature            string                `json:"failureSignature,omitempty"`
	SignatureTruncated          bool                  `json:"signatureTruncated"`
	RetryOfCallEventID          string                `json:"retryOfCallEventId,omitempty"`
	HelpRecovery                bool                  `json:"helpRecovery"`
	Fingerprint                 string                `json:"fingerprint"`
	Availability                availability.State    `json:"availability"`
	SemanticsKind               string                `json:"semanticsKind,omitempty"`
	SemanticsVerdict            string                `json:"semanticsVerdict,omitempty"`
	ArgTokens                   int64                 `json:"argTokens"`
	ResultTokens                int64                 `json:"resultTokens"`
	OutputBytes                 int64                 `json:"outputBytes"`
	OutputShare                 OutputShare           `json:"-"`
	OwnTokens                   int64                 `json:"-"`
	TokenBasis                  tokenaccounting.Basis `json:"tokenBasis"`
	ResidencyRequests           int64                 `json:"residencyRequests"`
	ResidencySegment            int64                 `json:"residencySegment"`
	IncurredInputTokens         int64                 `json:"incurredInputTokens"`
	IncurredOutputTokens        int64                 `json:"incurredOutputTokens"`
	IncurredCacheReadTokens     int64                 `json:"incurredCacheReadTokens"`
	IncurredCacheCreationTokens int64                 `json:"incurredCacheCreationTokens"`
}

// OutputShare is a fact's part in its call's result, which the read model
// turns into result tokens so that each output is counted once. One segment
// of each command holds the command's output (the holder that also carries
// its duration): ShareOwn when a nested result reports what the command
// printed (OutputBytes, and OwnTokens: that size capped by the command's
// max_output_tokens), ShareRest when it takes part of what no command claims
// (the script's own output, or a call's one result). Any other segment of the
// command holds none (ShareNone): its output is in its holder's, inseparably.
type OutputShare int

const (
	ShareRest OutputShare = iota
	ShareOwn
	ShareNone
)

// CommandResolver delegates command interpretation to the selected runner
// codec. Literal argv stays separate from shell text so safe tokenization can
// never turn an argument's data into syntax.
type CommandResolver func(string, map[string]any) runner.CommandExtraction

// DeriveInvocationFacts derives one fact per command segment of every tool
// call (planCalls pairs calls, results and nested command results). The
// pairing basis is retained on every fact rather than silently treating a
// best-effort historical pairing as provider-proven correlation.
func DeriveInvocationFacts(events []*domain.RunEvent, commandResolvers ...CommandResolver) []InvocationFact {
	plans := planCalls(events, commandResolvers...)
	facts := make([]InvocationFact, 0)
	history := retryHistory{previousByFingerprint: map[string]string{}, lastFailureByExecutable: map[string]string{}, helpAfterFailure: map[string]bool{}}
	for _, event := range events {
		if plan := plans[event]; plan != nil {
			facts = append(facts, plan.facts(event, &history)...)
		}
	}
	return facts
}

// retryHistory links a fact to an earlier fact with the same fingerprint and
// marks help lookups that follow a failure of the same executable.
type retryHistory struct {
	previousByFingerprint, lastFailureByExecutable map[string]string
	helpAfterFailure                               map[string]bool
}

func (p *callPlan) facts(event *domain.RunEvent, history *retryHistory) []InvocationFact {
	call := p.call
	base := InvocationFact{Version: InvocationFactVersion, CallEventID: event.ID.String(), ToolCallID: call.ToolCallID, ToolName: call.ToolName, Capability: CapabilityForTool(call.ToolName), Ownership: OwnershipNotACommand, OwnershipReason: "no declared command field", Outcome: OutcomeUnknown, PairingBasis: p.basis, Availability: availability.Available, TokenBasis: tokenaccounting.BasisUnknown}
	var resultData *domain.ToolResultEventData
	if p.result != nil {
		base.ResultEventID, base.OutputEventID = p.result.ID.String(), p.result.ID.String()
		resultData, _ = p.result.Data.(*domain.ToolResultEventData)
	}
	units, evidence := p.joinedUnits()
	covered := oneResultCovers(units)
	holders := outputHolders(units)
	_, scripted := holders[codeCommand]
	facts := make([]InvocationFact, 0, len(units))
	for unitIndex, unit := range units {
		fact := base
		fact.SegmentIndex, fact.SegmentCount = unitIndex, len(units)
		fact.OwnershipReason, fact.Wrapper = unit.Reason, unit.Wrapper
		var tokens []string
		switch unit.Kind {
		case unitCode:
			fact.Capability, fact.IntentClass = "code", "code:"+strings.ToLower(call.ToolName)
		case unitCommand:
			tokens = unit.Tokens
			if dynamicProgram(tokens[0]) {
				fact.Ownership, fact.OwnershipReason = OwnershipUnparseable, "command name needs shell evaluation"
				break
			}
			if unit.FromScript && fact.Capability == KindOther {
				fact.Capability = KindShell
			}
			resolution := resolveCatalogTokens(tokens)
			fact.Executable, fact.CatalogSnapshot = resolution.Owner, resolution.Snapshot
			fact.CommandPath = commandPathFor(resolution, tokens)
			fact.ProgramID, fact.SkillID = commandIdentifiers(tokens)
			fact.Capability = CapabilityForCommand(fact.CommandPath, fact.Capability)
			// `| head -30` trims another command's output: it reads no file.
			if fact.Capability == KindFileRead && !readsFileOperand(tokens) {
				fact.Capability = base.Capability
			}
			fact.SemanticsKind, fact.SemanticsVerdict = resolution.SemanticsKind, resolution.SemanticsVerdict
			fact.Ownership, fact.OwnershipReason = string(resolution.State), resolution.Reason
			if fact.Ownership == string(availability.Unknown) {
				fact.Ownership = OwnershipCompoundUnresolved
				if fact.OwnershipReason == "" {
					fact.OwnershipReason = "segment is not resolved by the manifest"
				}
			}
		case unitUnparseable:
			fact.Ownership = OwnershipUnparseable
		}
		if unit.Kind != unitCode {
			fact.IntentClass = intentKey(fact.CommandPath, call.ToolName)
		}
		var outcome *domain.ToolResultEventData
		nested := evidence[unit.Command]
		if nested != nil {
			outcome = nested.data
			fact.ResultEventID, fact.PairingBasis = nested.event.ID.String(), PairingCommandExecution
		} else if covered {
			outcome = resultData
		}
		holds := holders[unit.Command] == unitIndex
		switch {
		case !holds:
			fact.OutputShare = ShareNone
		case nested != nil:
			fact.OutputShare, fact.OutputBytes, fact.OwnTokens = ShareOwn, nested.outputBytes, p.visibleTokens(unit.Command, nested.outputBytes)
		case scripted && unit.Command != codeCommand:
			// The script's own unit takes what no command claims; this
			// command's output is somewhere in it, of unknown size.
			fact.OutputShare = ShareNone
		}
		if outcome != nil {
			fact.ExitCode = outcome.ExitCode
			if holds {
				fact.DurationMS = outcome.DurationMS
			}
			fact.Outcome = unitOutcome(outcome, unit, tokens)
			if fact.Outcome == OutcomeFailure {
				fact.FailureSignature, fact.SignatureTruncated = failureSignature(outcome)
			}
		}
		// A missing command is a real absence of a command signature. Do not
		// hash the empty string: otherwise every non-command tool collapses
		// into one retry/thrash bucket and inflates repeated-work rates.
		if unit.Kind == unitCommand && fact.CommandPath != "" && fact.Ownership != OwnershipUnparseable {
			// The command words are hashed into the fingerprint rather than
			// retained as raw evidence.
			fact.Fingerprint = fingerprint(fact.ToolName, fact.CommandPath, fact.Ownership, strings.Join(tokens, " "))
		}
		history.observe(&fact, tokens)
		facts = append(facts, fact)
	}
	return facts
}

func (h *retryHistory) observe(fact *InvocationFact, tokens []string) {
	if fact.Fingerprint != "" {
		if prior := h.previousByFingerprint[fact.Fingerprint]; prior != "" && prior != fact.CallEventID {
			fact.RetryOfCallEventID = prior
			fact.HelpRecovery = h.helpAfterFailure[fact.Executable]
		}
	}
	if fact.Outcome == OutcomeFailure && fact.Executable != "" {
		h.lastFailureByExecutable[fact.Executable] = fact.CallEventID
	}
	if isHelpCommand(tokens) && fact.Executable != "" && h.lastFailureByExecutable[fact.Executable] != "" {
		h.helpAfterFailure[fact.Executable] = true
	}
	if fact.Fingerprint != "" {
		h.previousByFingerprint[fact.Fingerprint] = fact.CallEventID
	}
}

// commandIdentifiers extracts only identifiers from explicitly supported
// command paths. Arbitrary argument values remain absent from the
// privacy-preserving read model.
func commandIdentifiers(tokens []string) (programID, skillID string) {
	if len(tokens) < 3 {
		return "", ""
	}
	for i := 0; i+2 < len(tokens); i++ {
		switch {
		case tokens[i] == "prompt-manager" && tokens[i+1] == "skill" && tokens[i+2] == "read":
			if i+3 < len(tokens) && !strings.HasPrefix(tokens[i+3], "-") {
				skillID = tokens[i+3]
			}
		case tokens[i] == "program-runtime" && tokens[i+1] == "programs" && (tokens[i+2] == "get" || tokens[i+2] == "wait"):
			if i+3 < len(tokens) && !strings.HasPrefix(tokens[i+3], "-") {
				programID = tokens[i+3]
			}
		}
	}
	return programID, skillID
}

// Tool kinds: the one vocabulary for what a harness tool does. Invocation
// facts carry it as Capability; transcript turn facts count calls by it.
const (
	KindShell           = "shell"
	KindFileRead        = "file-read"
	KindFileEdit        = "file-edit"
	KindSearch          = "search"
	KindWait            = "wait"
	KindPlan            = "plan"
	KindDelegate        = "delegate"
	KindWebSearch       = "web-search"
	KindWebFetch        = "web-fetch"
	KindImageGeneration = "image-generation"
	KindMCP             = "mcp"
	KindOther           = "other"
)

// toolKinds maps every runner's native tool names (lower-cased) to a kind.
var toolKinds = func() map[string]string {
	kinds := map[string]string{}
	for kind, names := range map[string][]string{
		KindShell:           {"bash", "shell", "local_shell", "exec", "exec_command", "command_execution", "write_stdin", "bashoutput", "killshell", "killbash"},
		KindFileRead:        {"read", "view", "cat", "view_image", "notebookread"},
		KindFileEdit:        {"edit", "write", "multiedit", "apply_patch", "notebookedit", "file_change"},
		KindSearch:          {"grep", "glob", "search", "toolsearch"},
		KindWait:            {"wait", "sleep", "schedulewakeup", "monitor"},
		KindPlan:            {"update_plan", "todowrite", "exitplanmode", "enterplanmode"},
		KindDelegate:        {"task", "agent", "spawn_agent", "send_message", "sendmessage", "wait_agent", "followup_task", "list_agents", "interrupt_agent", "create_thread"},
		KindWebSearch:       {"websearch", "web_search", "web"},
		KindWebFetch:        {"webfetch", "web_fetch"},
		KindImageGeneration: {"image_gen__imagegen", "imagegen", "image_generation"},
	} {
		for _, name := range names {
			kinds[name] = kind
		}
	}
	return kinds
}()

// CapabilityForTool is the tool's kind. A name no runner declares is a shell
// when it says so (`shell_command`) and other otherwise, never a guessed owner.
func CapabilityForTool(tool string) string {
	name := strings.ToLower(strings.TrimSpace(tool))
	if kind, ok := toolKinds[name]; ok {
		return kind
	}
	switch {
	case strings.HasPrefix(name, "mcp__"):
		return KindMCP
	case isShellTool(name):
		return KindShell
	}
	return KindOther
}

// CallScenario is the Vrooli scenario whose CLI a tool call runs first (a
// manifest-catalogued program other than the control plane), else "".
func CallScenario(tool string, input map[string]any) string {
	call := &domain.ToolCallEventData{ToolName: tool, Input: input}
	units, _ := callUnits(call, genericCommandInput(call))
	for _, unit := range units {
		if unit.Kind != unitCommand || dynamicProgram(unit.Tokens[0]) || unit.Tokens[0] == "vrooli" {
			continue
		}
		if resolution := resolveCatalogTokens(unit.Tokens); resolution.State != availability.External {
			return resolution.Owner
		}
	}
	return ""
}

// intentKey is the comparable identity of what a call tried to do: its
// command path, or the tool name for calls that run no command.
func intentKey(commandPath, toolName string) string {
	if commandPath != "" {
		return commandPath
	}
	return "tool:" + strings.ToLower(strings.TrimSpace(toolName))
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func ToolResultOutcome(data *domain.ToolResultEventData) string {
	if data == nil {
		return "unknown"
	}
	if harnessRefused(data) {
		return OutcomeRefused
	}
	if data.ExitCode != nil {
		if !domain.SupportedToolExitCode(data.ExitCode) {
			return "unknown"
		}
		if *data.ExitCode != 0 {
			return "failure"
		}
		return "success"
	}
	if data.OutcomeKnown != nil && !*data.OutcomeKnown {
		return "unknown"
	}
	// Historical payloads can have an inconsistent Success boolean, but error
	// strings retain decisive evidence (including Codex non-zero exit codes).
	if !data.Success || strings.TrimSpace(data.Error) != "" {
		return "failure"
	}
	return "success"
}

// harnessRefused reports a call the harness blocked before it ran: a Claude
// Code `<tool_use_error>Blocked: …` or a PreToolUse hook error. Other
// `<tool_use_error>`s (unread file, invalid input) are the call failing.
func harnessRefused(data *domain.ToolResultEventData) bool {
	if data.Success {
		return false
	}
	text := strings.TrimSpace(data.Error)
	if text == "" {
		text = strings.TrimSpace(data.Output)
	}
	return strings.HasPrefix(text, "<tool_use_error>Blocked") || strings.HasPrefix(text, "PreToolUse:") && strings.Contains(text, "hook error")
}

func failureSignature(data *domain.ToolResultEventData) (string, bool) {
	if data == nil {
		return "tool_failure", false
	}
	if data.ExitCode != nil && *data.ExitCode != 0 {
		return boundedFailureSignature("exit_code_" + strconv.Itoa(*data.ExitCode))
	}
	message := strings.ToLower(strings.TrimSpace(data.Error))
	signature := "tool_failure"
	switch {
	case strings.Contains(message, "permission denied"), strings.Contains(message, "operation not permitted"):
		signature = "permission_denied"
	case strings.Contains(message, "no such file"), strings.Contains(message, "not found"), strings.Contains(message, "does not exist"):
		signature = "missing_file"
	case strings.Contains(message, "exited with code"):
		// Keep only the numeric exit class, never the command or its arguments.
		for _, field := range strings.Fields(message) {
			field = strings.Trim(field, ".,:;()[]{}")
			if allDigits(field) {
				signature = "exit_code_" + field
				break
			}
		}
	}
	return boundedFailureSignature(signature)
}

func boundedFailureSignature(signature string) (string, bool) {
	if len(signature) <= FailureSignatureMaxLength {
		return signature, false
	}
	return signature[:FailureSignatureMaxLength], true
}

// CapabilityForCommand refines the harness capability with evidence from the
// resolved command path. A shell wrapper therefore cannot mask a search,
// wait, file-read, or file-edit operation.
// readsFileOperand reports whether a cat/head/tail/sed segment names a file to
// read. Without one it reads standard input, a filter on a pipeline. Flags,
// their numeric values and sed's script are not files.
func readsFileOperand(tokens []string) bool {
	if len(tokens) < 2 {
		return false
	}
	name := tokens[0]
	operands := 0
	for _, token := range tokens[1:] {
		if strings.HasPrefix(token, "-") && token != "-" {
			continue
		}
		if _, err := strconv.Atoi(token); err == nil {
			continue // head -n 20, tail -c 100
		}
		operands++
	}
	if name == "sed" {
		return operands >= 2 // the first operand is the script
	}
	return operands >= 1
}

// hasWord reports whether a space-separated command path contains word as a
// whole token, so `await` never counts as `wait`.
func hasWord(path, word string) bool {
	for _, field := range strings.Fields(path) {
		if field == word {
			return true
		}
	}
	return false
}

func CapabilityForCommand(command, fallback string) string {
	name := strings.ToLower(strings.TrimSpace(command))
	switch {
	case name == "rg", name == "find", strings.Contains(name, "search"), strings.Contains(name, "grep"), strings.Contains(name, " rg"), strings.Contains(name, " find"):
		return KindSearch
	case hasWord(name, "wait"):
		return KindWait
	case strings.Contains(name, "apply_patch"), strings.Contains(name, " edit"), strings.Contains(name, " write"):
		return KindFileEdit
	case name == "cat", name == "head", name == "tail", name == "sed", strings.HasPrefix(name, "cat "), strings.HasPrefix(name, "head "), strings.HasPrefix(name, "tail "), strings.HasPrefix(name, "sed "):
		return KindFileRead
	default:
		return fallback
	}
}

func commandInput(call *domain.ToolCallEventData, commandResolvers ...CommandResolver) runner.CommandExtraction {
	if call == nil {
		return runner.CommandExtraction{Reason: "tool call is unavailable"}
	}
	if len(commandResolvers) > 0 && commandResolvers[0] != nil {
		extraction := commandResolvers[0](call.ToolName, call.Input)
		if extraction.Command != "" || len(extraction.Args) > 0 || extraction.Reason != "" {
			return extraction
		}
	}
	return genericCommandInput(call)
}

func genericCommandInput(call *domain.ToolCallEventData) runner.CommandExtraction {
	for _, key := range []string{"command", "cmd", "argv", "args"} {
		value, ok := call.Input[key]
		if !ok {
			continue
		}
		switch value := value.(type) {
		case []any:
			args := make([]string, 0, len(value))
			for _, item := range value {
				arg, ok := item.(string)
				if !ok || strings.TrimSpace(arg) == "" {
					args = nil
					break
				}
				args = append(args, arg)
			}
			if len(args) > 0 {
				return runner.CommandExtraction{Command: strings.Join(args, " "), Args: args, LiteralArgs: true}
			}
		case []string:
			if len(value) > 0 {
				return runner.CommandExtraction{Command: strings.Join(value, " "), Args: value, LiteralArgs: true}
			}
		case string:
			if strings.TrimSpace(value) != "" {
				return runner.CommandExtraction{Command: strings.TrimSpace(redact(value))}
			}
		}
	}
	return runner.CommandExtraction{Reason: "no declared command field"}
}

func isShellTool(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "shell") || strings.Contains(name, "bash") || strings.Contains(name, "command")
}

func isHelpCommand(tokens []string) bool {
	for _, token := range tokens {
		if token == "help" || token == "--help" || token == "-h" {
			return true
		}
	}
	return false
}

func fingerprint(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:16])
}

func redact(value string) string {
	for _, marker := range sensitiveQueryMarkers() {
		if at := strings.Index(strings.ToLower(value), marker); at >= 0 {
			end := strings.IndexAny(value[at:], " \t\n")
			if end < 0 {
				return value[:at] + marker + "[REDACTED]"
			}
			return value[:at] + marker + "[REDACTED]" + value[at+end:]
		}
	}
	return value
}

func sensitiveQueryMarkers() []string {
	return []string{"token" + "=", "password" + "=", "secret" + "=", "api_key" + "="}
}
