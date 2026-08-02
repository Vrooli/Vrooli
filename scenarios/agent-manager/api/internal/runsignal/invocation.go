// Package runsignal derives bounded, deterministic evidence from durable run events.
package runsignal

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"agent-manager/internal/availability"

	"agent-manager/internal/domain"
)

// InvocationFactVersion is pinned into every derivation so a rebuild can be
// compared honestly when classifier behavior evolves.
// v4 backfills bounded failure signatures into durable facts produced before
// that field was projected, so historical failed invocations remain actionable
// after a replay without retaining raw tool output.
const InvocationFactVersion = "invocation-fact.v5"

// FailureSignatureMaxLength bounds retained failure vocabulary. Signatures are
// controlled class labels, never copied error text.
const FailureSignatureMaxLength = 64

// InvocationFact is a bounded, redacted interpretation of one durable tool
// call/result pair. It intentionally excludes raw input and output bodies.
type InvocationFact struct {
	Version            string             `json:"version"`
	CallEventID        string             `json:"callEventId"`
	ResultEventID      string             `json:"resultEventId,omitempty"`
	ToolCallID         string             `json:"toolCallId,omitempty"`
	ToolName           string             `json:"toolName"`
	Capability         string             `json:"capability"`
	IntentClass        string             `json:"intentClass"`
	Executable         string             `json:"executable,omitempty"`
	CommandPath        string             `json:"commandPath,omitempty"`
	Ownership          string             `json:"ownership"`
	OwnershipReason    string             `json:"ownershipReason,omitempty"`
	SegmentIndex       int                `json:"segmentIndex"`
	SegmentCount       int                `json:"segmentCount"`
	CatalogSnapshot    string             `json:"catalogSnapshot,omitempty"`
	Outcome            string             `json:"outcome"`
	PairingBasis       string             `json:"pairingBasis"`
	FailureSignature   string             `json:"failureSignature,omitempty"`
	SignatureTruncated bool               `json:"signatureTruncated"`
	RetryOfCallEventID string             `json:"retryOfCallEventId,omitempty"`
	HelpRecovery       bool               `json:"helpRecovery"`
	Fingerprint        string             `json:"fingerprint"`
	Availability       availability.State `json:"availability"`
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

func DeriveInvocationFacts(events []*domain.RunEvent) []InvocationFact {
	return DeriveInvocationFactsWithResolver(events, nil)
}

func DeriveInvocationFactsWithResolver(events []*domain.RunEvent, resolver CapabilityResolver) []InvocationFact {
	results := map[string]*domain.RunEvent{}
	ordinalResults := make([]*domain.RunEvent, 0)
	for _, event := range events {
		if event == nil {
			continue
		}
		if result, ok := event.Data.(*domain.ToolResultEventData); ok {
			if result.ToolCallID != "" {
				results[result.ToolCallID] = event
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
	for _, event := range events {
		if event == nil {
			continue
		}
		call, ok := event.Data.(*domain.ToolCallEventData)
		if !ok {
			continue
		}
		capability := CapabilityForTool(call.ToolName)
		if resolver != nil {
			capability = resolver(call.ToolName)
			if capability == "" {
				capability = "other"
			}
		}
		factBase := InvocationFact{Version: InvocationFactVersion, CallEventID: event.ID.String(), ToolCallID: call.ToolCallID, ToolName: call.ToolName, Capability: capability, IntentClass: intentClass(call), Ownership: OwnershipNotACommand, OwnershipReason: "tool input has no command or cmd field", Outcome: "unknown", PairingBasis: "unpaired", Availability: availability.Available}
		command := commandInput(call)
		segments, compound, segmentError := SegmentShell(command)
		if command == "" {
			segments = []string{""}
		}
		if segmentError != "" {
			segments = []string{command}
		}
		if len(segments) == 0 {
			segments = []string{""}
		}
		var result *domain.RunEvent
		if call.ToolCallID != "" {
			result = results[call.ToolCallID]
			if result != nil {
				factBase.PairingBasis = "tool_call_id"
			}
		} else if ordinalResultIndex < len(ordinalResults) {
			result = ordinalResults[ordinalResultIndex]
			ordinalResultIndex++
			factBase.PairingBasis = "ordinal"
		}
		for segmentIndex, segment := range segments {
			fact := factBase
			fact.SegmentIndex, fact.SegmentCount = segmentIndex, len(segments)
			if segmentError != "" {
				fact.Ownership, fact.OwnershipReason = OwnershipUnparseable, segmentError
			} else if command != "" {
				resolution := ResolveCatalog(segment)
				fact.Executable, fact.CommandPath, fact.CatalogSnapshot = resolution.Owner, resolution.Command, resolution.Snapshot
				fact.Ownership, fact.OwnershipReason = string(resolution.State), resolution.Reason
				if fact.Ownership == string(availability.Unknown) {
					fact.Ownership = OwnershipCompoundUnresolved
					if fact.OwnershipReason == "" {
						fact.OwnershipReason = "segment is not resolved by the manifest"
					}
				}
			}
			if compound && fact.Ownership != OwnershipResolved && fact.Ownership != OwnershipExternal {
				fact.Ownership = OwnershipCompoundUnresolved
			}
			if result != nil {
				fact.ResultEventID = result.ID.String()
				if data, ok := result.Data.(*domain.ToolResultEventData); ok {
					fact.Outcome = toolResultOutcome(data)
					if fact.Outcome == "failure" {
						fact.FailureSignature, fact.SignatureTruncated = failureSignature(data)
					}
				}
			}
			fact.Fingerprint = fingerprint(fact.ToolName, fact.CommandPath, fact.Ownership, normalizedArgumentShape(call.Input), strconv.Itoa(segmentIndex))
			if prior := previousByFingerprint[fact.Fingerprint]; prior != "" {
				fact.RetryOfCallEventID = prior
				if helpAfterFailure[fact.Executable] {
					fact.HelpRecovery = true
				}
			}
			if fact.Outcome == "failure" && fact.Executable != "" {
				lastFailureByExecutable[fact.Executable] = fact.CallEventID
			}
			if isHelpCommand(segment) && fact.Executable != "" && lastFailureByExecutable[fact.Executable] != "" {
				helpAfterFailure[fact.Executable] = true
			}
			previousByFingerprint[fact.Fingerprint] = fact.CallEventID
			facts = append(facts, fact)
		}
	}
	return facts
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

func intentClass(call *domain.ToolCallEventData) string {
	if call == nil {
		return "other"
	}
	return CapabilityForTool(call.ToolName) + ":" + normalizedArgumentShape(call.Input)
}

// normalizedArgumentShape returns a deterministic, redacted description of a
// tool input. It keeps keys, flag names, value kinds, file extensions, and
// coarse lengths, never an argument value. That makes fingerprints useful for
// grouping while preserving the no-raw-input contract of invocation facts.
func normalizedArgumentShape(input map[string]any) string {
	if len(input) == 0 {
		return "empty"
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+":"+argumentValueShape(input[key]))
	}
	return strings.Join(parts, ",")
}

func argumentValueShape(value any) string {
	switch value := value.(type) {
	case string:
		return stringArgumentShape(value)
	case bool:
		return "bool"
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "number"
	case nil:
		return "null"
	case []any:
		return "list:" + strconv.Itoa(len(value))
	case map[string]any:
		return "object:" + strconv.Itoa(len(value))
	default:
		return "other"
	}
}

func stringArgumentShape(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "text:0"
	}
	if tokens, ok := safeTokens(unwrapShellCommand(value)); ok && len(tokens) > 0 {
		parts := make([]string, 0, len(tokens))
		for _, token := range tokens {
			parts = append(parts, tokenShape(token))
		}
		return "command:" + strings.Join(parts, "+")
	}
	return "shell:unparseable"
}

func tokenShape(token string) string {
	if strings.HasPrefix(token, "-") {
		return "flag:" + strings.SplitN(token, "=", 2)[0]
	}
	if extension := strings.ToLower(filepath.Ext(token)); extension != "" && len(extension) <= 12 {
		return "path:" + extension
	}
	if allDigits(token) {
		return "number"
	}
	return "text"
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func toolResultOutcome(data *domain.ToolResultEventData) string {
	if data == nil {
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
	if len(signature) <= FailureSignatureMaxLength {
		return signature, false
	}
	return signature[:FailureSignatureMaxLength], true
}

func commandInput(call *domain.ToolCallEventData) string {
	if call == nil {
		return ""
	}
	for _, key := range []string{"command", "cmd"} {
		if value, ok := call.Input[key].(string); ok {
			return strings.TrimSpace(redact(value))
		}
	}
	return ""
}

func isShellTool(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "shell") || strings.Contains(name, "bash") || strings.Contains(name, "command")
}

func isHelpCommand(command string) bool {
	tokens, ok := safeTokens(command)
	if !ok {
		return false
	}
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
