package agentharness

import (
	"errors"
	"github.com/vrooli/cli-core/agentcatalog"
)

// deliveryDeclaredEffort selects only evidence for the resolved model. Source
// preferences cannot quietly override an ambiguous effort declaration. Other
// resource roles retain their existing response bytes and caller contract.
func deliveryDeclaredEffort(role string, r agentcatalog.CodingRole) (string, error) {
	if role != "code.delivery" {
		return "", nil
	}
	effort := ""
	found := false
	for _, m := range r.Models {
		if m.Model != r.Model {
			continue
		}
		if m.Effort == nil {
			return "", errors.New("delivery model has no declared effort")
		}
		value := *m.Effort
		switch value {
		case "low", "medium", "high", "xhigh", "max":
		default:
			return "", errors.New("delivery model effort is invalid")
		}
		if found && effort != value {
			return "", errors.New("delivery model effort is ambiguous")
		}
		effort, found = value, true
	}
	if !found {
		return "", errors.New("resolved delivery model lacks effort evidence")
	}
	return effort, nil
}
