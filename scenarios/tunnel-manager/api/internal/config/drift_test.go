package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// stateOf finds the classified entry for a hostname in a DriftReport.
func stateOf(t *testing.T, rep DriftReport, host string) IngressEntry {
	t.Helper()
	for _, e := range rep.Entries {
		if e.Hostname == host {
			return e
		}
	}
	t.Fatalf("hostname %q not present in drift report (%d entries)", host, len(rep.Entries))
	return IngressEntry{}
}

func desired(host, svc string, src RouteSource, scenario string) DesiredEntry {
	return DesiredEntry{Hostname: host, Service: svc, Source: src, Scenario: scenario}
}

func liveRule(host, svc string) IngressRule { return IngressRule{Hostname: host, Service: svc} }

func ledgerEntry(host string, owner Owner) LedgerEntry {
	return LedgerEntry{Hostname: host, Owner: owner}
}

// TestReconcile_StateTable walks every row of the §5 ownership state table in
// a single fixture so the classification matrix is covered exhaustively.
func TestReconcile_StateTable(t *testing.T) {
	desiredSet := []DesiredEntry{
		// scenario route, live → MANAGED
		desired("managed.example.invalid", "http://localhost:21100", SourceScenario, "agent-manager"),
		// scenario route, not live → MISSING
		desired("missing.example.invalid", "http://localhost:21200", SourceScenario, "new-scenario"),
		// external route, live → EXTERNAL_OK (desired-external)
		desired("ext-desired.example.invalid", "http://127.0.0.1:9000", SourceExternal, ""),
	}
	liveSet := []IngressRule{
		liveRule("managed.example.invalid", "http://localhost:21100"),
		liveRule("ext-desired.example.invalid", "http://127.0.0.1:9000"),
		// ledger EXTERNAL, live, not desired → EXTERNAL_OK (ledger-external)
		liveRule("ext-ledger.example.invalid", "http://127.0.0.1:9100"),
		// ledger IGNORED, live → IGNORED
		liveRule("ignored.example.invalid", "http://127.0.0.1:9200"),
		// live, not desired, no ledger → UNMANAGED
		liveRule("drift.example.invalid", "http://127.0.0.1:9300"),
		{Service: "http_status:404"}, // catch-all, ignored
	}
	ledgerSet := []LedgerEntry{
		ledgerEntry("ext-ledger.example.invalid", OwnerExternal),
		ledgerEntry("ignored.example.invalid", OwnerIgnored),
		// ledger MANAGED, route gone (not desired) → ORPHANED
		ledgerEntry("orphaned.example.invalid", OwnerManaged),
	}

	rep := reconcile(ModeRemote, desiredSet, liveSet, ledgerSet)
	require.Equal(t, ModeRemote, rep.Mode)

	require.Equal(t, StateManaged, stateOf(t, rep, "managed.example.invalid").State)
	require.Equal(t, SourceScenario, stateOf(t, rep, "managed.example.invalid").Source)
	require.Equal(t, StateMissing, stateOf(t, rep, "missing.example.invalid").State)
	require.Equal(t, StateExternalOK, stateOf(t, rep, "ext-desired.example.invalid").State)
	require.Equal(t, StateExternalOK, stateOf(t, rep, "ext-ledger.example.invalid").State)
	require.Equal(t, StateIgnored, stateOf(t, rep, "ignored.example.invalid").State)
	require.Equal(t, StateUnmanaged, stateOf(t, rep, "drift.example.invalid").State)
	require.Equal(t, StateOrphaned, stateOf(t, rep, "orphaned.example.invalid").State)

	// The catch-all (no hostname) is never classified.
	for _, e := range rep.Entries {
		require.NotEmpty(t, e.Hostname)
	}

	// Counts mirror the entries.
	require.Equal(t, 1, rep.Counts[StateManaged])
	require.Equal(t, 1, rep.Counts[StateMissing])
	require.Equal(t, 2, rep.Counts[StateExternalOK])
	require.Equal(t, 1, rep.Counts[StateIgnored])
	require.Equal(t, 1, rep.Counts[StateUnmanaged])
	require.Equal(t, 1, rep.Counts[StateOrphaned])
}

// TestReconcile_EmptyLedgerMakesLiveExtrasUnmanaged is the safe default: with
// no ledger, every live hostname TM did not author is drift, never silently
// dropped.
func TestReconcile_EmptyLedgerMakesLiveExtrasUnmanaged(t *testing.T) {
	rep := reconcile(ModeRemote,
		[]DesiredEntry{desired("mine.example.invalid", "http://localhost:1", SourceScenario, "mine")},
		[]IngressRule{
			liveRule("mine.example.invalid", "http://localhost:1"),
			liveRule("foreign.example.com", "http://localhost:2"),
			{Service: "http_status:404"},
		},
		nil,
	)
	require.Equal(t, StateManaged, stateOf(t, rep, "mine.example.invalid").State)
	require.Equal(t, StateUnmanaged, stateOf(t, rep, "foreign.example.com").State)
}

// TestReconcile_IgnoredOverridesDesired: an IGNORED ledger record wins even if
// a desired route would otherwise classify the hostname as MANAGED — the
// operator's "never touch this" decision is authoritative.
func TestReconcile_IgnoredOverridesDesired(t *testing.T) {
	rep := reconcile(ModeLocal,
		[]DesiredEntry{desired("h.example.invalid", "http://localhost:1", SourceScenario, "s")},
		[]IngressRule{liveRule("h.example.invalid", "http://localhost:1")},
		[]LedgerEntry{ledgerEntry("h.example.invalid", OwnerIgnored)},
	)
	require.Equal(t, StateIgnored, stateOf(t, rep, "h.example.invalid").State)
}

// TestReconcile_ExternalMissingIsAddCandidate: an external desired route not
// yet live is MISSING (the next additive sync publishes it).
func TestReconcile_ExternalMissingIsAddCandidate(t *testing.T) {
	rep := reconcile(ModeRemote,
		[]DesiredEntry{desired("ext.example.invalid", "http://127.0.0.1:9000", SourceExternal, "")},
		nil,
		nil,
	)
	require.Equal(t, StateMissing, stateOf(t, rep, "ext.example.invalid").State)
}

// TestReconcile_OrphanedManagedNotLive: a ledger-MANAGED hostname whose route
// is gone and is no longer live is still ORPHANED — a cleanup candidate, never
// silently forgotten.
func TestReconcile_OrphanedManagedNotLive(t *testing.T) {
	rep := reconcile(ModeRemote, nil, nil,
		[]LedgerEntry{ledgerEntry("gone.example.invalid", OwnerManaged)},
	)
	require.Equal(t, StateOrphaned, stateOf(t, rep, "gone.example.invalid").State)
}
