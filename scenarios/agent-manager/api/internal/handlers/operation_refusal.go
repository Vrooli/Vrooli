package handlers

import (
	"errors"
	"net/http"

	"github.com/vrooli/api-core/authn"
)

// operationRefusalMessage extends a handler's refusal text with the exact
// request the operator can approve, when err carries one. A refused agent then
// sees what to run instead of only that it was refused. The command comes from
// authn's reviewed-policy catalogue; no handler names a policy itself.
func operationRefusalMessage(err error, message string) string {
	var refusal *authn.OperationRefusal
	if !errors.As(err, &refusal) {
		return message
	}
	switch {
	case refusal.Command() != "":
		return message + "; ask the operator with: " + refusal.Command()
	case refusal.OperatorDecisionWhere != "":
		return message + "; " + refusal.Operation + " records an operator decision, so no agent grant covers it; the operator answers it in " + refusal.OperatorDecisionWhere
	}
	return message
}

// writeOperationRefusal answers a refused RequireOperation with 403. For an
// agent it adds `request_authority`, the ready-to-run command, plus the
// operation, target and reviewed policy it names; the CLI prints the command.
func writeOperationRefusal(w http.ResponseWriter, err error, message string) {
	body := map[string]any{"error": operationRefusalMessage(err, message)}
	var refusal *authn.OperationRefusal
	if errors.As(err, &refusal) {
		body["operation"] = refusal.Operation
		body["target"] = refusal.Target
		if len(refusal.Policies) > 0 {
			body["policy"] = refusal.Policies[0]
		}
		if command := refusal.Command(); command != "" {
			body["request_authority"] = command
		}
	}
	writeJSON(w, http.StatusForbidden, body)
}
