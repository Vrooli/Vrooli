package runreport

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"agent-manager/internal/domain"
)

// InvocationFactVersion is pinned into every derivation so a rebuild can be
// compared honestly when classifier behavior evolves.
const InvocationFactVersion = "invocation-fact.v1"

// InvocationFact is a bounded, redacted interpretation of one durable tool
// call/result pair. It intentionally excludes raw input and output bodies.
type InvocationFact struct {
	Version            string `json:"version"`
	CallEventID        string `json:"callEventId"`
	ResultEventID      string `json:"resultEventId,omitempty"`
	ToolCallID         string `json:"toolCallId,omitempty"`
	ToolName           string `json:"toolName"`
	Executable         string `json:"executable,omitempty"`
	CommandPath        string `json:"commandPath,omitempty"`
	Ownership          string `json:"ownership"`
	CatalogSnapshot    string `json:"catalogSnapshot,omitempty"`
	Outcome            string `json:"outcome"`
	RetryOfCallEventID string `json:"retryOfCallEventId,omitempty"`
	HelpRecovery       bool   `json:"helpRecovery"`
	Fingerprint        string `json:"fingerprint"`
	Availability       string `json:"availability"`
}

// DeriveInvocationFacts pairs durable tool calls/results by call identity and
// never executes the supplied shell text. Unsupported, absent, and compound
// inputs become explicit unknown facts rather than guessed ownership.
func DeriveInvocationFacts(events []*domain.RunEvent) []InvocationFact {
	results := map[string]*domain.RunEvent{}
	for _, event := range events {
		if event == nil {
			continue
		}
		if result, ok := event.Data.(*domain.ToolResultEventData); ok && result.ToolCallID != "" {
			results[result.ToolCallID] = event
		}
	}
	facts := make([]InvocationFact, 0)
	previousByFingerprint := map[string]string{}
	lastFailureByExecutable := map[string]string{}
	helpAfterFailure := map[string]bool{}
	for _, event := range events {
		if event == nil {
			continue
		}
		call, ok := event.Data.(*domain.ToolCallEventData)
		if !ok {
			continue
		}
		fact := InvocationFact{Version: InvocationFactVersion, CallEventID: event.ID.String(), ToolCallID: call.ToolCallID, ToolName: call.ToolName, Ownership: "unknown", Outcome: "unknown", Availability: "available"}
		command := commandInput(call)
		if command != "" {
			resolution := resolveCatalog(command)
			fact.Executable = resolution.Owner
			fact.CommandPath, fact.CatalogSnapshot, fact.Ownership = resolution.Command, resolution.Snapshot, resolution.State
		} else if isShellTool(call.ToolName) {
			fact.Availability = "unknown"
		}
		if result := results[call.ToolCallID]; result != nil {
			fact.ResultEventID = result.ID.String()
			if data, ok := result.Data.(*domain.ToolResultEventData); ok {
				if data.Success {
					fact.Outcome = "success"
				} else {
					fact.Outcome = "failure"
				}
			}
		}
		fact.Fingerprint = fingerprint(fact.ToolName, fact.CommandPath, fact.Ownership)
		if prior := previousByFingerprint[fact.Fingerprint]; prior != "" {
			fact.RetryOfCallEventID = prior
			if helpAfterFailure[fact.Executable] {
				fact.HelpRecovery = true
			}
		}
		if fact.Outcome == "failure" && fact.Executable != "" {
			lastFailureByExecutable[fact.Executable] = fact.CallEventID
		}
		if isHelpCommand(command) && fact.Executable != "" && lastFailureByExecutable[fact.Executable] != "" {
			helpAfterFailure[fact.Executable] = true
		}
		previousByFingerprint[fact.Fingerprint] = fact.CallEventID
		facts = append(facts, fact)
	}
	return facts
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
