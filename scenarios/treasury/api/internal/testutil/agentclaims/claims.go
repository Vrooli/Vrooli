// Package agentclaims builds explicit synthetic grants for owner tests.
// No production package imports this fixture.
package agentclaims

import (
	"encoding/json"
	requestidentity "github.com/vrooli/api-core/identity"
	"time"
	"treasury/internal/identity"
)

func For(subject string) identity.Claims {
	expires := time.Now().Add(24 * time.Hour)
	grant := requestidentity.OperationGrant{ID: "test-grant", Approver: "test-operator", Operations: []string{"treasury.read", "treasury.spend"}, Targets: []string{"treasury:*"}, ExpiresAt: expires}
	raw, _ := json.Marshal(grant)
	return identity.Claims{Subject: subject, RunID: "test-run", Scopes: []string{"treasury:read", "treasury:write"}, ExpiresAt: expires.Unix(), Meta: map[string]string{"operation_grant": string(raw)}}
}
