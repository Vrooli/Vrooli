package bindings

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

// Failure is the closed (status, class) pair a bridge error carries to the
// kernel, where it becomes a typed exception the program branches on.
//
// Before this existed the bridge wrote `{"error": "<message>"}` and every
// contract program carried its own 25-line substring table to turn the message
// back into a class. Sixty-one copies, eleven drifted. The table now lives here,
// next to the code that produces the messages, and travels with the body; the
// kernel's own copy (`program_helper.classify_message`) covers only text that
// never passed through this bridge.
type Failure struct {
	// Status is the envelope status the class implies: unavailable, refused, or failed.
	Status string `json:"status"`
	// Class is the closed error class: scenario_unreachable, no_grant,
	// not_run_eligible, inference_spend_exceeded, delegated_run_spend_exceeded,
	// ambiguous_response, invalid_input, no_governed_binding, remote_error,
	// deadline_exceeded, or binding_error.
	Class string `json:"class"`
	// HTTPStatus is the target scenario's answer when it answered at all; zero otherwise.
	HTTPStatus int `json:"http_status,omitempty"`
}

var remoteStatusPattern = regexp.MustCompile(`remote status (\d{3})`)

type failureRule struct {
	needles []string
	status  string
	class   string
}

// failureRules is ordered. Unreachable first because a refused or timed-out
// call to an unreachable scenario is still unreachable; deadline last because
// several needles above it are substrings of ordinary deadline messages.
var failureRules = []failureRule{
	{[]string{"is unreachable", "bridge unavailable", "bridge is unavailable", "scenario_not_running", "no running runtime ports", "connection refused", "dial tcp", "no such host"}, "unavailable", "scenario_unreachable"},
	{[]string{"requires an explicit grant", "requires explicit confirmation"}, "refused", "no_grant"},
	{[]string{"not run eligible", "run_eligible"}, "refused", "not_run_eligible"},
	{[]string{"inference_spend_exceeded", "inference spend"}, "refused", "inference_spend_exceeded"},
	{[]string{"delegated_run_spend_exceeded", "delegation_spend_exceeded", "delegated run spend"}, "refused", "delegated_run_spend_exceeded"},
	{[]string{"no determinable primary response field", "rows must be one of", "rows must name one of"}, "failed", "ambiguous_response"},
	{[]string{"accepts named proto fields", "invalid arguments for", "no proto field matches", "decode binding arguments", "client-side only", "has an empty proto path", "unknown field"}, "failed", "invalid_input"},
	{[]string{"is not governed", "does not resolve to a governed binding", "binding does not exist"}, "failed", "no_governed_binding"},
	{[]string{"deadline", "timed out", "budget exhausted"}, "failed", "deadline_exceeded"},
}

// ClassifyFailure maps a bridge error to its closed class. It is total: an
// error that matches no rule is the generic `binding_error`, never a bare string.
func ClassifyFailure(err error) Failure {
	if err == nil {
		return Failure{}
	}
	if errors.Is(err, errNoBinding) {
		return Failure{Status: "failed", Class: "no_governed_binding"}
	}
	return classifyFailureMessage(err.Error())
}

func classifyFailureMessage(message string) Failure {
	lower := strings.ToLower(message)
	if match := remoteStatusPattern.FindStringSubmatch(lower); match != nil {
		code, _ := strconv.Atoi(match[1])
		return remoteFailure(code)
	}
	for _, rule := range failureRules {
		for _, needle := range rule.needles {
			if strings.Contains(lower, needle) {
				return Failure{Status: rule.status, Class: rule.class}
			}
		}
	}
	return Failure{Status: "failed", Class: "binding_error"}
}

// remoteFailure classifies the target scenario's own answer. A gateway status
// means the scenario is not serving and the reading is unknown; any other
// status means the scenario ran and its answer is the outcome.
func remoteFailure(code int) Failure {
	switch code {
	case 502, 503, 504:
		return Failure{Status: "unavailable", Class: "scenario_unreachable", HTTPStatus: code}
	default:
		return Failure{Status: "failed", Class: "remote_error", HTTPStatus: code}
	}
}
