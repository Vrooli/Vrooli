package identity

import (
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
)

// Authorize uses the operation grant carried by verified Agent Manager claims.
// The canonical evaluator preserves wildcard coverage and keeps each policy's
// operations paired with its own targets in a multi-policy consent.
func Authorize(claims Claims, operation, target string, now time.Time) error {
	scope := "treasury:write"
	if operation == "treasury.read" {
		scope = "treasury:read"
	}
	if claims.Subject == "" || claims.RunID == "" || claims.ExpiresAt <= now.Unix() {
		return fmt.Errorf("%w: current run identity and %s capability are required", ErrUnverifiable, scope)
	}
	grant := requestidentity.OperationGrantFromMeta(claims.Meta, now)
	evaluation := authn.EvaluateOperationGrant(claims.Scopes, grant, scope, operation, target, now)
	if !evaluation.Capability {
		return fmt.Errorf("%w: current run identity and %s capability are required", ErrUnverifiable, scope)
	}
	if !strings.HasPrefix(target, "treasury:") || !evaluation.Allowed() {
		return fmt.Errorf("%w: operation %s is outside the approved target grant", ErrUnverifiable, operation)
	}
	return nil
}
