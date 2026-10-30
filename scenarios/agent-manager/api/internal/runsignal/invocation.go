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
// v4 backfills bounded failure signatures into durable facts produced before
// that field was projected, so historical failed invocations remain actionable
// after a replay without retaining raw tool output.
// v21 reads command identity from a tolerant shell lexer (heredocs, quotes,
// wrappers) and Codex exec JavaScript, derives IntentClass from CommandPath
// instead of argument shape, gives each segment its own outcome, and reports
// exit 1 of rg/grep/diff/test as no-match.
// v20 derives every token quantity from the one usage account
// (AccountUsage): Codex totals are last cumulative counter minus the counter
// before the run's first report, so re-tails and replays no longer inflate
// runs, and Claude usage counts each provider message once.
// v19 applies canonical per-invocation authority and replay selection to durable time-accounting token buckets.
// v18 adds bounded identifiers for known program and skill command paths.
// v17 apportions per-call token quantities across the segments of a compound
// command. Facts at v16 and earlier copied the whole call's payload onto every
// segment, so any SUM over them overcounts a run in proportion to how compound
// its commands were. The version bump is what makes a refresh reproject those
// runs instead of leaving the two conventions mixed in one cohort.
const InvocationFactVersion = "invocation-fact.v25"

// Outcome vocabulary of an InvocationFact.
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
	OutcomeUnknown = "unknown"
	// OutcomeNoMatch is exit 1 of a search or comparison (rg, grep, diff,
	// test): the command ran and found nothing, which is not a failure.
	OutcomeNoMatch = "no-match"
)

// FailureSignatureMaxLength bounds retained failure vocabulary. Signatures are
// controlled class labels, never copied error text.
const FailureSignatureMaxLength = 64

// InvocationFact is a bounded, redacted interpretation of one durable tool
// call/result pair. It intentionally excludes raw input and output bodies.
type InvocationFact struct {
	Version                     string                `json:"version"`
	CallEventID                 string                `json:"callEventId"`
	ResultEventID               string                `json:"resultEventId,omitempty"`
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
	TokenBasis                  tokenaccounting.Basis `json:"tokenBasis"`
	ResidencyTurns              int64                 `json:"residencyTurns"`
	ResidencySegment            int64                 `json:"residencySegment"`
	IncurredInputTokens         int64                 `json:"incurredInputTokens"`
	IncurredOutputTokens        int64                 `json:"incurredOutputTokens"`
	IncurredCacheReadTokens     int64                 `json:"incurredCacheReadTokens"`
	IncurredCacheCreationTokens int64                 `json:"incurredCacheCreationTokens"`
}

// DeriveInvocationFacts pairs durable tool calls/results by their provider
// correlation identifier when present. Imported transcripts frequently omit
// that identifier, so unmatched calls/results fall back to their stable stream
// ordinal. The basis is retained on every fact rather than silently treating a
// best-effort historical pairing as provider-proven correlation.
// CapabilityResolver is supplied by the runner codec that owns a harness
// tool surface. A nil resolver retains the conservative generic fallback used
// by standalone callers and historical fixtures.
type CapabilityResolver func(string) string

// CommandResolver delegates command interpretation to the selected runner
// codec. Literal argv stays separate from shell text so safe tokenization can
// never turn an argument's data into syntax.
type CommandResolver func(string, map[string]any) runner.CommandExtraction

func DeriveInvocationFacts(events []*domain.RunEvent) []InvocationFact {
	return DeriveInvocationFactsWithResolver(events, nil)
}

func DeriveInvocationFactsWithResolver(events []*domain.RunEvent, resolver CapabilityResolver, commandResolvers ...CommandResolver) []InvocationFact {
	results := map[string]*domain.RunEvent{}
	ordinalResults := make([]*domain.RunEvent, 0)
	for _, event := range events {
		if event == nil {
			continue
		}
		if result, ok := event.Data.(*domain.ToolResultEventData); ok {
			if result.ToolCallID != "" {
				// A duplicated result keeps the first copy.
				if _, seen := results[result.ToolCallID]; !seen {
					results[result.ToolCallID] = event
				}
			} else {
				ordinalResults = append(ordinalResults, event)
			}
		}
	}
	facts := make([]InvocationFact, 0)
	previousByFingerprint := map[string]string{}
	lastFailureByExecutable := map[string]string{}
	helpAfterFailure := map[string]bool{}
	ordinalResultIndex := 0
	seenCalls := map[string]bool{}
	for _, event := range events {
		if event == nil {
			continue
		}
		call, ok := event.Data.(*domain.ToolCallEventData)
		if !ok {
			continue
		}
		// A tool call is one invocation however many copies of its event were
		// recorded (an importer once appended the events twice): keep the first.
		if call.ToolCallID != "" {
			if seenCalls[call.ToolCallID] {
				continue
			}
			seenCalls[call.ToolCallID] = true
		}
		capability := CapabilityForTool(call.ToolName)
		if resolver != nil {
			capability = resolver(call.ToolName)
			if capability == "" {
				capability = "other"
			}
		}
		factBase := InvocationFact{Version: InvocationFactVersion, CallEventID: event.ID.String(), ToolCallID: call.ToolCallID, ToolName: call.ToolName, Capability: capability, Ownership: OwnershipNotACommand, OwnershipReason: "no declared command field", Outcome: OutcomeUnknown, PairingBasis: "unpaired", Availability: availability.Available, TokenBasis: tokenaccounting.BasisUnknown}
		units := callUnits(call, commandInput(call, commandResolvers...))
		var result *domain.RunEvent
		if call.ToolCallID != "" {
			result = results[call.ToolCallID]
			if result != nil {
				factBase.PairingBasis = "call-id"
			}
		} else if ordinalResultIndex < len(ordinalResults) {
			result = ordinalResults[ordinalResultIndex]
			ordinalResultIndex++
			factBase.PairingBasis = "ordinal"
		}
		var resultData *domain.ToolResultEventData
		if result != nil {
			resultData, _ = result.Data.(*domain.ToolResultEventData)
		}
		for unitIndex, unit := range units {
			fact := factBase
			fact.SegmentIndex, fact.SegmentCount = unitIndex, len(units)
			fact.OwnershipReason = unit.Reason
			fact.Wrapper = unit.Wrapper
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
				if unit.FromScript && fact.Capability == "other" {
					fact.Capability = "shell"
				}
				resolution := resolveCatalogTokens(tokens)
				fact.Executable, fact.CatalogSnapshot = resolution.Owner, resolution.Snapshot
				fact.CommandPath = commandPathFor(resolution, tokens)
				fact.ProgramID, fact.SkillID = commandIdentifiers(tokens)
				fact.Capability = CapabilityForCommand(fact.CommandPath, fact.Capability)
				// `| head -30` trims another command's output: it reads no file.
				if fact.Capability == "file-read" && !readsFileOperand(tokens) {
					fact.Capability = factBase.Capability
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
			if resultData != nil {
				fact.ResultEventID = result.ID.String()
				fact.ExitCode, fact.DurationMS = resultData.ExitCode, resultData.DurationMS
				fact.Outcome = unitOutcome(resultData, unit, tokens)
				if fact.Outcome == OutcomeFailure {
					fact.FailureSignature, fact.SignatureTruncated = failureSignature(resultData)
				}
			} else if result != nil {
				fact.ResultEventID = result.ID.String()
			}
			// A missing command is a real absence of a command signature. Do not
			// hash the empty string: otherwise every non-command tool collapses
			// into one retry/thrash bucket and inflates repeated-work rates.
			if unit.Kind == unitCommand && fact.CommandPath != "" && fact.Ownership != OwnershipUnparseable {
				// The command words are hashed into the fingerprint rather than
				// retained as raw evidence.
				fact.Fingerprint = fingerprint(fact.ToolName, fact.CommandPath, fact.Ownership, strings.Join(tokens, " "))
			}
			if fact.Fingerprint != "" {
				if prior := previousByFingerprint[fact.Fingerprint]; prior != "" && prior != fact.CallEventID {
					fact.RetryOfCallEventID = prior
					if helpAfterFailure[fact.Executable] {
						fact.HelpRecovery = true
					}
				}
			}
			if fact.Outcome == OutcomeFailure && fact.Executable != "" {
				lastFailureByExecutable[fact.Executable] = fact.CallEventID
			}
			if isHelpCommand(tokens) && fact.Executable != "" && lastFailureByExecutable[fact.Executable] != "" {
				helpAfterFailure[fact.Executable] = true
			}
			if fact.Fingerprint != "" {
				previousByFingerprint[fact.Fingerprint] = fact.CallEventID
			}
			facts = append(facts, fact)
		}
	}
	return facts
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

// CapabilityForTool is the closed harness capability vocabulary. Runner
// adapters use these same labels when translating their native tool names;
// the classifier accepts unknown names as other rather than guessing a
// scenario-specific owner.
func CapabilityForTool(tool string) string {
	name := strings.ToLower(strings.TrimSpace(tool))
	switch {
	case name == "read" || strings.Contains(name, "read") || name == "cat" || name == "view":
		return "file-read"
	case name == "write" || name == "edit" || name == "apply_patch" || name == "file_change" || strings.Contains(name, "write"):
		return "file-edit"
	case strings.Contains(name, "grep") || strings.Contains(name, "search") || name == "glob":
		return "search"
	case name == "wait" || strings.Contains(name, "poll"):
		return "wait"
	case strings.Contains(name, "plan"):
		return "plan"
	case name == "task" || strings.Contains(name, "delegate") || strings.Contains(name, "agent"):
		return "delegate"
	case strings.Contains(name, "web") || strings.Contains(name, "network") || strings.Contains(name, "fetch"):
		return "network"
	case isShellTool(name) || name == "exec" || name == "bash":
		return "shell"
	default:
		return "other"
	}
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
		return "search"
	case hasWord(name, "wait"):
		return "wait"
	case strings.Contains(name, "apply_patch"), strings.Contains(name, " edit"), strings.Contains(name, " write"):
		return "file-edit"
	case name == "cat", name == "head", name == "tail", name == "sed", strings.HasPrefix(name, "cat "), strings.HasPrefix(name, "head "), strings.HasPrefix(name, "tail "), strings.HasPrefix(name, "sed "):
		return "file-read"
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
