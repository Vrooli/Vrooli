package routing

import (
	"context"

	"github.com/vrooli/ai-provider-go/policy"
)

// JSONRightsContextSource reads the operator-owned context file on every call,
// so an update between preview and execution invalidates the planned decision.
func JSONRightsContextSource(path string) RightsContextSource {
	return func(context.Context) (policy.RightsContext, error) {
		return policy.LoadRightsContextFile(path)
	}
}
