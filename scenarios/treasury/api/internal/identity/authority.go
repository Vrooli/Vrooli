package identity

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
)

// Authorize uses the operation grant carried by verified Agent Manager claims.
// The canonical evaluator preserves wildcard coverage and keeps each policy's
// operations paired with its own targets in a multi-policy consent.
func Authorize(claims Claims, operation, target string, now time.Time) error {
	scope := "treasury:write"
	if operation == "treasury.read" {
		scope = "treasury:read"
	}
	if claims.Subject == "" || claims.RunID == "" || claims.ExpiresAt <= now.Unix() || !scopecatalog.Resolve(claims.Scopes, scope) {
		return fmt.Errorf("%w: current run identity and %s capability are required", ErrUnverifiable, scope)
	}
	var grant requestidentity.OperationGrant
	if json.Unmarshal([]byte(claims.Meta["operation_grant"]), &grant) != nil || !strings.HasPrefix(target, "treasury:") || !grant.Allows(operation, target, now) {
		return fmt.Errorf("%w: operation %s is outside the approved target grant", ErrUnverifiable, operation)
	}
	return nil
}
