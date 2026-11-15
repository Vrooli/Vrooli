package ledger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

// A number the operator typed is sent exactly or refused before anything is
// sent: "1,000" must never become 1, and "abc" must never become 0.
func TestNumbersAreSentExactlyOrRefused(t *testing.T) {
	withOwnerSeams(t, func(context.Context) error { return nil }, func(context.Context) (string, error) { return "owner-session", nil })
	server := &recordingServer{received: map[string]string{}}
	core := cliapptest.NewTestApp(t, server)
	manifest := cliapptest.ReadManifest(t)
	ledgerGroup, err := Register(core, manifest)
	require.NoError(t, err)
	ingestGroup, err := RegisterIngest(core, manifest)
	require.NoError(t, err)
	groups := []cliapp.SubcommandGroup{ledgerGroup, ingestGroup}

	transfer := []string{"--from-account-id", "a", "--to-account-id", "b", "--currency", "USD", "--external-id", "x", "--local-owner"}
	goal := []string{"--book-id", "b", "--name", "n", "--metric", "cash", "--comparator", "gte", "--buffer-multiple", "1.5", "--local-owner"}
	event := []string{"--external-id", "x", "--adapter-id", "manual", "--account-id", "a", "--book-id", "b", "--currency", "USD", "--local-owner"}
	cases := []struct {
		name, group, command, procedure, refusal string
		argv                                     []string
	}{
		{"separator in an amount", "ledger", "journal-transfer", "/vrooli.money_ledger.v1.ledger.JournalService/Transfer", "--amount-minor must be a whole number", append([]string{"--amount-minor", "1,000"}, transfer...)},
		{"word for an amount", "ingest", "event-ingest", "/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent", "--amount-minor must be a whole number", append([]string{"--amount-minor", "abc", "--basis", "derived"}, event...)},
		{"count beyond range", "ledger", "goal-declare", "/vrooli.money_ledger.v1.ledger.PositionService/DeclareGoal", "--sustain-periods must be a whole number", append([]string{"--sustain-periods", "9999999999"}, goal...)},
		{"ratio that is not a number", "ledger", "goal-declare", "/vrooli.money_ledger.v1.ledger.PositionService/DeclareGoal", "--threshold-ratio must be a finite decimal", append([]string{"--sustain-periods", "3", "--threshold-ratio", "NaN"}, goal...)},
		{"unknown basis", "ingest", "event-ingest", "/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent", "basis", append([]string{"--amount-minor", "100", "--basis", "estimate"}, event...)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server.received = map[string]string{}
			_, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, tc.group, tc.command), core, tc.argv...)
			require.ErrorContains(t, err, tc.refusal)
			_, reached := server.credential(tc.procedure)
			require.False(t, reached, "a refused flag must stop the command before it calls Money Ledger")
		})
	}

	t.Run("exact values and the basis alias are sent", func(t *testing.T) {
		server.received = map[string]string{}
		_, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "ingest", "event-ingest"), core, append([]string{"--amount-minor", "-1250", "--basis", "operator_asserted"}, event...)...)
		require.ErrorContains(t, err, "fixture stops here")
		_, reached := server.credential("/vrooli.money_ledger.v1.ingest.IngestService/IngestEvent")
		require.True(t, reached)
	})
}

var basisSchema = cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "basis"}}}

func TestBasisNamesHowAFigureIsKnown(t *testing.T) {
	for value, want := range map[string]string{"": "BASIS_AUTHORITATIVE", "authoritative": "BASIS_AUTHORITATIVE", "derived": "BASIS_DERIVED", "operator-asserted": "BASIS_OPERATOR_ASSERTED", "Operator_Asserted": "BASIS_OPERATOR_ASSERTED"} {
		f := readFlags(cliapptest.NewTestRunContext(cliapptest.TestRunContextOptions{Schema: basisSchema, Flags: map[string]string{"basis": value}}))
		require.Equal(t, want, f.Basis("basis").String(), value)
		require.NoError(t, f.Err(), value)
	}
	f := readFlags(cliapptest.NewTestRunContext(cliapptest.TestRunContextOptions{Schema: basisSchema, Flags: map[string]string{"basis": "estimate"}}))
	require.Equal(t, "BASIS_UNSPECIFIED", f.Basis("basis").String())
	require.ErrorContains(t, f.Err(), "authoritative, derived or operator-asserted")
}
