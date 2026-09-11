// Package apierr decodes the scenario-to-cloud typed error envelope on the
// CLI side (REST body or Connect error detail) and maps stable codes to the
// CLI exit-code contract (0 ok, 1 failed, 2 refused/conflict, 3
// pending/needs-input, 124 observer timeout).
package apierr

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"
)

// Exit codes shared with api/apierrors.
const (
	ExitOK              = 0
	ExitFailed          = 1
	ExitRefused         = 2
	ExitPending         = 3
	ExitObserverTimeout = 124
)

// Exit is a command outcome that carries its own exit code (a non-terminal
// operation standing, an observer timeout, a needs-input handoff). It is not
// an API refusal.
type Exit struct {
	Code    int
	Message string
}

// Error implements error.
func (e *Exit) Error() string { return e.Message }

// Pending returns an exit-3 outcome.
func Pending(format string, args ...any) *Exit {
	return &Exit{Code: ExitPending, Message: fmt.Sprintf(format, args...)}
}

// Failed returns an exit-1 outcome.
func Failed(format string, args ...any) *Exit {
	return &Exit{Code: ExitFailed, Message: fmt.Sprintf(format, args...)}
}

// Refused returns an exit-2 outcome produced locally (an invalid selector,
// a refused combination of flags).
func Refused(format string, args ...any) *Exit {
	return &Exit{Code: ExitRefused, Message: fmt.Sprintf(format, args...)}
}

// refusedCodes map to exit 2. This list mirrors api/apierrors.ExitCodeFor;
// the CLI module cannot import the API module, so the contract is repeated
// here and covered by a test against a recorded envelope.
var refusedCodes = map[string]struct{}{
	"unauthenticated":               {},
	"forbidden_scope":               {},
	"forbidden_target":              {},
	"forbidden_revoked":             {},
	"forbidden_origin":              {},
	"forbidden_host":                {},
	"deployment_selector_ambiguous": {},
	"deployment_identity_conflict":  {},
	"request_key_conflict":          {},
	"plan_stale":                    {},
	"plan_digest_mismatch":          {},
	"operation_conflict":            {},
	"fence_stale":                   {},
	"unsupported_capability":        {},
	"unsupported_schema_version":    {},
	"publication_refused":           {},
	"receipt_invalid":               {},
	"preview_required":              {},
	"recovery_point_conflict":       {},
	"recovery_point_corrupt":        {},
	"recovery_point_protected":      {},
	"recovery_point_required":       {},
	"recovery_key_unavailable":      {},
	"restore_target_not_clean":      {},
	"backup_provider_unavailable":   {},
	"rollback_incompatible":         {},
	"reach_scope_missing":           {},
	"reach_protocol_unsupported":    {},
	"enrollment_revoked":            {},
}

// pendingCodes map to exit 3: the operator must supply something before the
// request can proceed; nothing was mutated.
var pendingCodes = map[string]struct{}{
	"needs_input":            {},
	"pending_operator_input": {},
}

// NextAction mirrors the wire next_action object.
type NextAction struct {
	Owner     string `json:"owner,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Reference string `json:"reference,omitempty"`
	Label     string `json:"label,omitempty"`
}

// Typed is one decoded API error.
type Typed struct {
	Status     int
	Code       string
	Message    string
	Retryable  bool
	NextAction *NextAction
	Details    map[string]any
}

// Error implements error.
func (t *Typed) Error() string {
	if t.Message == "" {
		return t.Code
	}
	return t.Code + ": " + t.Message
}

// ExitCode maps the stable code to the CLI contract.
func (t *Typed) ExitCode() int {
	if t == nil {
		return ExitFailed
	}
	if _, ok := pendingCodes[t.Code]; ok {
		return ExitPending
	}
	if _, ok := refusedCodes[t.Code]; ok {
		return ExitRefused
	}
	if t.Status == 401 || t.Status == 403 {
		return ExitRefused
	}
	return ExitFailed
}

// Decode extracts the typed envelope from a cli-core API error or a Connect
// error carrying the errors.v1.Error detail. It returns false when neither
// applies.
func Decode(err error) (*Typed, bool) {
	if err == nil {
		return nil, false
	}
	var typed *Typed
	if errors.As(err, &typed) && typed != nil {
		return typed, true
	}
	var cerr *connect.Error
	if errors.As(err, &cerr) && cerr != nil {
		return decodeConnect(cerr), true
	}
	var apiErr *cliutil.APIError
	if !errors.As(err, &apiErr) || apiErr == nil {
		return nil, false
	}
	var envelope struct {
		Error struct {
			Code       string         `json:"code"`
			Message    string         `json:"message"`
			Retryable  bool           `json:"retryable"`
			NextAction *NextAction    `json:"next_action"`
			Details    map[string]any `json:"details"`
		} `json:"error"`
	}
	if len(apiErr.RawResponse) == 0 || json.Unmarshal(apiErr.RawResponse, &envelope) != nil || envelope.Error.Code == "" {
		if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
			return &Typed{Status: apiErr.StatusCode, Code: codeForStatus(apiErr.StatusCode), Message: apiErr.Message}, true
		}
		return nil, false
	}
	return &Typed{
		Status: apiErr.StatusCode, Code: envelope.Error.Code, Message: envelope.Error.Message,
		Retryable: envelope.Error.Retryable, NextAction: envelope.Error.NextAction, Details: envelope.Error.Details,
	}, true
}

// decodeConnect reads the errors.v1.Error detail the API attaches to every
// Connect failure; without it the Connect code alone is mapped.
func decodeConnect(cerr *connect.Error) *Typed {
	typed := &Typed{Status: statusForConnect(cerr.Code()), Code: codeForConnect(cerr.Code()), Message: cerr.Message()}
	for _, detail := range cerr.Details() {
		msg, err := detail.Value()
		if err != nil {
			continue
		}
		wire, ok := msg.(*errorsv1.Error)
		if !ok {
			continue
		}
		typed.Code = wire.GetCode()
		typed.Message = wire.GetMessage()
		typed.Retryable = wire.GetRetryable()
		if na := wire.GetNextAction(); na != nil {
			typed.NextAction = &NextAction{Owner: na.GetOwner(), Kind: na.GetKind(), Reference: na.GetReference(), Label: na.GetLabel()}
		}
		if wire.GetDetails() != nil {
			typed.Details = wire.GetDetails().AsMap()
		}
		break
	}
	return typed
}

func statusForConnect(code connect.Code) int {
	switch code {
	case connect.CodeUnauthenticated:
		return 401
	case connect.CodePermissionDenied:
		return 403
	case connect.CodeNotFound:
		return 404
	case connect.CodeAborted, connect.CodeAlreadyExists:
		return 409
	case connect.CodeInvalidArgument:
		return 400
	default:
		return 500
	}
}

func codeForConnect(code connect.Code) string {
	switch code {
	case connect.CodeUnauthenticated:
		return "unauthenticated"
	case connect.CodePermissionDenied:
		return "forbidden_scope"
	case connect.CodeUnavailable:
		return "unavailable"
	default:
		return "internal"
	}
}

func codeForStatus(status int) string {
	switch status {
	case 401:
		return "unauthenticated"
	case 403:
		return "forbidden_scope"
	default:
		return "internal"
	}
}

// ExitCode returns the exit code for any CLI error.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var exit *Exit
	if errors.As(err, &exit) {
		return exit.Code
	}
	if typed, ok := Decode(err); ok {
		return typed.ExitCode()
	}
	return ExitFailed
}

// Format renders an operator-facing message with the next action. For an
// unauthenticated refusal it names the documented sign-in paths; for an
// ambiguous selector it lists the candidate identities.
func Format(err error) string {
	var exit *Exit
	if errors.As(err, &exit) {
		return exit.Message
	}
	typed, ok := Decode(err)
	if !ok {
		return fmt.Sprintf("Error: %v", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Error: %s (%s)", typed.Message, typed.Code)
	if candidates := Candidates(typed); len(candidates) > 0 {
		b.WriteString("\nCandidates:")
		for _, c := range candidates {
			fmt.Fprintf(&b, "\n  --deployment %s  scenario=%s environment=%s target=%s", c.ID, c.ScenarioID, c.Environment, c.TargetKey)
		}
	}
	if typed.NextAction != nil {
		b.WriteString("\nNext: ")
		switch {
		case typed.NextAction.Label != "" && typed.NextAction.Reference != "":
			fmt.Fprintf(&b, "%s (%s)", typed.NextAction.Label, typed.NextAction.Reference)
		case typed.NextAction.Reference != "":
			b.WriteString(typed.NextAction.Reference)
		default:
			b.WriteString(typed.NextAction.Label)
		}
	}
	if typed.Code == "unauthenticated" {
		b.WriteString("\nSign in: provide the runtime-owned local session token (personal_local), or set SCENARIO_TO_CLOUD_API_TOKEN / VROOLI_API_TOKEN to a bearer token from the configured provider (vrooli-bridge auth for a paired node).")
	}
	return b.String()
}

// Candidate is one deployment named by an ambiguous selector.
type Candidate struct {
	ID          string
	ScenarioID  string
	Environment string
	TargetKey   string
}

// Candidates extracts the ambiguity candidates from a typed error's details
// (sorted by id, the same order the API emits).
func Candidates(typed *Typed) []Candidate {
	if typed == nil || typed.Details == nil {
		return nil
	}
	raw, ok := typed.Details["candidates"].([]any)
	if !ok {
		return nil
	}
	out := make([]Candidate, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Candidate{
			ID: str(m["id"]), ScenarioID: str(m["scenario_id"]), Environment: str(m["environment"]), TargetKey: str(m["target_key"]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
