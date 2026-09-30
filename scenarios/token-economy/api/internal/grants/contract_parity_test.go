package grants_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	grantsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/grants"
	mandatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/mandate"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// [REQ:TKE-P0-002] Grant stays field-congruent with Treasury's live 17-field
// Mandate descriptor, and every permitted divergence is recorded durably.
func TestGrantContractParityWithTreasuryMandate(t *testing.T) {
	mandate := mandatev1.File_treasury_v1_mandate_mandate_proto.Messages().ByName("Mandate")
	grant := grantsv1.File_token_economy_v1_grants_grants_proto.Messages().ByName("Grant")
	require.NotNil(t, mandate)
	require.NotNil(t, grant)
	require.Equal(t, 17, mandate.Fields().Len(), "Treasury Mandate changed; review every mapping")

	mapping := map[string]string{
		"id": "id", "book_id": "token_type_id", "budget_id": "grant_source_id",
		"authorizer": "authorizer", "cap_minor": "amount_minor",
		"allowed_counterparties": "allowed_catalog_scopes", "denied_counterparties": "denied_catalog_scopes",
		"expires_at": "expires_at", "issued_at": "issued_at", "status": "status",
		"idempotency_key": "idempotency_key", "required_evidence": "required_evidence",
		"recurrence_seconds": "recurrence_seconds", "next_charge_at": "next_issue_at",
		"cancelled_at": "cancelled_at",
	}
	for treasuryName, grantName := range mapping {
		treasuryField := mandate.Fields().ByName(protoreflect.Name(treasuryName))
		grantField := grant.Fields().ByName(protoreflect.Name(grantName))
		require.NotNil(t, treasuryField, "missing Treasury field %s", treasuryName)
		require.NotNil(t, grantField, "missing Grant mapping %s -> %s", treasuryName, grantName)
		require.Equal(t, treasuryField.Kind(), grantField.Kind(), "kind drift for %s -> %s", treasuryName, grantName)
		require.Equal(t, treasuryField.Cardinality(), grantField.Cardinality(), "cardinality drift for %s -> %s", treasuryName, grantName)
	}
	for _, omitted := range []string{"currency", "signature"} {
		require.NotNil(t, mandate.Fields().ByName(protoreflect.Name(omitted)))
		require.Nil(t, grant.Fields().ByName(protoreflect.Name(omitted)))
	}
	require.NotNil(t, grant.Fields().ByName("holder_id"))
	require.NotNil(t, grant.Fields().ByName("rules"))
	require.Equal(t, len(mapping)+2, grant.Fields().Len(), "unrecorded Grant field divergence")

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	decisionPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "docs", "internal", "DECISIONS.md")
	decisions, err := os.ReadFile(decisionPath)
	require.NoError(t, err)
	for _, recorded := range []string{
		"| `book_id` | `token_type_id` |", "| `budget_id` | `grant_source_id` |",
		"| `cap_minor` | `amount_minor` |", "| `currency` | omitted |",
		"| `allowed_counterparties` | `allowed_catalog_scopes` |",
		"| `denied_counterparties` | `denied_catalog_scopes` |",
		"| `signature` | omitted |", "| `next_charge_at` | `next_issue_at` |",
		"| — | `holder_id` |", "| — | `rules` |",
	} {
		require.Contains(t, string(decisions), recorded, "unrecorded parity divergence")
	}
}

// [REQ:TKE-P0-003] Neither the public grant request nor its closed rule shape
// accepts code or a caller-supplied authorization decision.
func TestGrantContractAcceptsNoAuthorizationDecisionOrCode(t *testing.T) {
	file := grantsv1.File_token_economy_v1_grants_grants_proto
	for _, messageName := range []string{"CreateGrantRequest", "GrantRule"} {
		message := file.Messages().ByName(protoreflect.Name(messageName))
		require.NotNil(t, message)
		for i := 0; i < message.Fields().Len(); i++ {
			name := strings.ToLower(string(message.Fields().Get(i).Name()))
			for _, forbidden := range []string{"authorized", "authorization", "decision", "script", "expression", "code"} {
				require.NotContains(t, name, forbidden, "%s exposes forbidden caller claim/code field %s", messageName, name)
			}
		}
	}
}
