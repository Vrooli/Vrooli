package card_test

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"treasury/internal/rail/card"

	"github.com/stretchr/testify/require"
)

type fixtureIssuer struct{ scope card.Scope }

func (fixtureIssuer) Name() string { return "fixture-card" }
func (i fixtureIssuer) Issue(context.Context, card.IssueCommand) (card.Issued, error) {
	return card.Issued{ExternalID: "provider-card-1", Credential: "sensitive", Scope: i.scope}, nil
}

func (i fixtureIssuer) Inspect(context.Context, card.InspectQuery) (card.Issued, error) {
	return card.Issued{ExternalID: "provider-card-1", Scope: i.scope}, nil
}

// [REQ:TRS-P1-003] The scoped-card contract contains only mandate vocabulary;
// a provider adapter satisfies it without changing the interface.
func TestIssuerContractIsProviderNeutralAndRejectsScopeWidening(t *testing.T) {
	scope := card.Scope{MandateReference: "mandate-1", AmountMinor: 500, Currency: "USD", Counterparty: "merchant-1", ExpiresAt: time.Now().UTC().Add(time.Hour)}
	registry, err := card.NewRegistry(fixtureIssuer{scope: scope})
	require.NoError(t, err)
	issuer, err := registry.Get("FIXTURE-CARD")
	require.NoError(t, err)
	issued, err := issuer.Issue(context.Background(), card.IssueCommand{InstrumentID: "instrument-1", IdempotencyKey: "issue-1", Credential: "provider-secret", Scope: scope})
	require.NoError(t, err)
	require.True(t, card.EqualScope(scope, issued.Scope))

	contractPackage := reflect.TypeOf((*card.Issuer)(nil)).Elem().PkgPath()
	require.NotContains(t, strings.ToLower(contractPackage), "lithic")
	require.NotContains(t, strings.ToLower(contractPackage), "stripe")
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	require.Contains(t, file, "/rail/card/")

	widened := scope
	widened.AmountMinor++
	registry, err = card.NewRegistry(fixtureIssuer{scope: widened})
	require.NoError(t, err)
	issuer, err = registry.Get("fixture-card")
	require.NoError(t, err)
	_, err = issuer.Issue(context.Background(), card.IssueCommand{InstrumentID: "instrument-1", IdempotencyKey: "issue-1", Credential: "provider-secret", Scope: scope})
	require.ErrorIs(t, err, card.ErrInvalid)
}
