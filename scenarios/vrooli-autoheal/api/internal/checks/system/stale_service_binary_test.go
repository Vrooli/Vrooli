package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// stubResolver answers per pattern so a test can describe a mixed fleet.
type stubResolver struct {
	byUnit map[string]struct {
		path    string
		deleted bool
		found   bool
	}
	calls int
}

func (s *stubResolver) Resolve(_ context.Context, unit string) (string, bool, bool) {
	s.calls++
	v, ok := s.byUnit[unit]
	if !ok {
		return "", false, false
	}
	return v.path, v.deleted, v.found
}

func resolver(entries map[string][3]any) *stubResolver {
	m := map[string]struct {
		path    string
		deleted bool
		found   bool
	}{}
	for k, v := range entries {
		m[k] = struct {
			path    string
			deleted bool
			found   bool
		}{v[0].(string), v[1].(bool), v[2].(bool)}
	}
	return &stubResolver{byUnit: m}
}

var twoUnits = []SupervisedUnit{
	{Unit: "a.service"},
	{Unit: "b.service"},
}

func runStaleCheck(t *testing.T, r ProcessExeResolver) checks.Result {
	t.Helper()
	if checkOS != "linux" {
		repocontracttest.SkipPlatform(t, "stale binary check reads /proc")
	}
	return NewStaleServiceBinaryCheck(
		WithSupervisedUnits(twoUnits),
		WithProcessExeResolver(r),
	).Run(context.Background())
}

// The condition this exists for: a supervisor still executing a deleted binary
// after a rebuild replaced it.
func TestDetectsServiceRunningAReplacedBinary(t *testing.T) {
	r := runStaleCheck(t, resolver(map[string][3]any{
		"a.service": {"/home/u/.vrooli/bin/vrooli", true, true},
		"b.service": {"/home/u/.vrooli/bin/loop", false, true},
	}))

	if r.Status != checks.StatusWarning {
		t.Fatalf("status = %v, want warning", r.Status)
	}
	if !strings.Contains(r.Message, "a.service") {
		t.Errorf("message should name the stale unit, got %q", r.Message)
	}
	if strings.Contains(r.Message, "b.service") {
		t.Errorf("a fresh unit must not be reported stale, got %q", r.Message)
	}
}

func TestFreshServicesAreHealthy(t *testing.T) {
	r := runStaleCheck(t, resolver(map[string][3]any{
		"a.service": {"/bin/a", false, true},
		"b.service": {"/bin/b", false, true},
	}))
	if r.Status != checks.StatusOK {
		t.Fatalf("status = %v, want ok; details %v", r.Status, r.Details)
	}
}

// A unit that is not running is a liveness problem owned by other checks.
// Reporting it here would double-report one condition.
func TestStoppedServiceIsNotReportedStale(t *testing.T) {
	r := runStaleCheck(t, resolver(map[string][3]any{}))

	if r.Status != checks.StatusOK {
		t.Fatalf("status = %v, want ok for absent units", r.Status)
	}
	if r.Details["unitsChecked"] != 0 {
		t.Errorf("unitsChecked = %v, want 0", r.Details["unitsChecked"])
	}
}

// One unit going stale repeatedly must dedupe into one incident, so the
// fingerprint dimension has to be the unit set rather than a timestamp.
func TestFindingKeyIsTheStaleUnitSet(t *testing.T) {
	r := runStaleCheck(t, resolver(map[string][3]any{
		"a.service": {"/bin/a", true, true},
		"b.service": {"/bin/b", true, true},
	}))
	if got := r.Details["findingKey"]; got != "a.service,b.service" {
		t.Fatalf("findingKey = %v", got)
	}
}

// The recovery action must only offer itself when there is something to fix.
func TestRestartOfferedOnlyWhenStale(t *testing.T) {
	c := NewStaleServiceBinaryCheck(WithSupervisedUnits(twoUnits))

	clean := &checks.Result{Details: map[string]interface{}{"staleUnits": []string{}}}
	if c.RecoveryActions(clean)[0].Available {
		t.Error("restart must not be offered when nothing is stale")
	}

	dirty := &checks.Result{Details: map[string]interface{}{"staleUnits": []string{"a.service"}}}
	if !c.RecoveryActions(dirty)[0].Available {
		t.Error("restart should be available when a unit is stale")
	}
}

// The action ID must stay "restart" so it inherits the host-pressure gate that
// defers restarts while the machine is saturated.
func TestRestartActionIDIsGated(t *testing.T) {
	c := NewStaleServiceBinaryCheck()
	if id := c.RecoveryActions(nil)[0].ID; id != "restart" {
		t.Fatalf("action ID = %q; it must be \"restart\" to inherit the pressure gate", id)
	}
}

// Execution re-resolves rather than trusting stale evidence, so a condition
// that cleared between detection and action does not cause a pointless restart.
func TestExecuteReResolvesAndSkipsRecoveredUnits(t *testing.T) {
	if checkOS != "linux" {
		repocontracttest.SkipPlatform(t, "stale binary check reads /proc")
	}
	var restarted []string
	c := NewStaleServiceBinaryCheck(
		WithSupervisedUnits(twoUnits),
		WithProcessExeResolver(resolver(map[string][3]any{
			"a.service": {"/bin/a", false, true}, // recovered since detection
			"b.service": {"/bin/b", true, true},
		})),
		WithUnitRestarter(func(_ context.Context, unit string) (string, error) {
			restarted = append(restarted, unit)
			return "ok", nil
		}),
	)

	res := c.ExecuteAction(context.Background(), "restart")

	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if len(restarted) != 1 || restarted[0] != "b.service" {
		t.Fatalf("restarted = %v, want only the still-stale unit", restarted)
	}
}

func TestExecuteReportsRestartFailure(t *testing.T) {
	if checkOS != "linux" {
		repocontracttest.SkipPlatform(t, "stale binary check reads /proc")
	}
	c := NewStaleServiceBinaryCheck(
		WithSupervisedUnits(twoUnits[:1]),
		WithProcessExeResolver(resolver(map[string][3]any{"a.service": {"/bin/a", true, true}})),
		WithUnitRestarter(func(context.Context, string) (string, error) {
			return "", errors.New("unit not loaded")
		}),
	)

	res := c.ExecuteAction(context.Background(), "restart")

	if res.Success {
		t.Fatal("a failed restart must not report success")
	}
	if !strings.Contains(res.Error, "a.service") {
		t.Errorf("error should name the unit, got %q", res.Error)
	}
}

// Nothing to do is a success, not a failure — otherwise a race between the
// check and the action ratchets the failure backoff for no reason.
func TestExecuteWithNothingStaleSucceeds(t *testing.T) {
	if checkOS != "linux" {
		repocontracttest.SkipPlatform(t, "stale binary check reads /proc")
	}
	c := NewStaleServiceBinaryCheck(
		WithSupervisedUnits(twoUnits),
		WithProcessExeResolver(resolver(map[string][3]any{"a.service": {"/bin/a", false, true}})),
		WithUnitRestarter(func(context.Context, string) (string, error) {
			t.Fatal("must not restart a healthy unit")
			return "", nil
		}),
	)

	if res := c.ExecuteAction(context.Background(), "restart"); !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
}

// A Result that has been through JSON — which is what the auto-heal path
// actually feeds in — carries []interface{}, not []string. Asserting only the
// native shape silently disables recovery for every persisted result.
func TestRestartOfferedAfterAJSONRoundTrip(t *testing.T) {
	c := NewStaleServiceBinaryCheck(WithSupervisedUnits(twoUnits))

	roundTripped := &checks.Result{Details: map[string]interface{}{
		"staleUnits": []interface{}{"a.service", "b.service"},
	}}
	if !c.RecoveryActions(roundTripped)[0].Available {
		t.Fatal("restart must be offered for a result that came back through JSON")
	}

	empty := &checks.Result{Details: map[string]interface{}{"staleUnits": []interface{}{}}}
	if c.RecoveryActions(empty)[0].Available {
		t.Error("an empty JSON list must not offer a restart")
	}

	if c.RecoveryActions(&checks.Result{Details: nil})[0].Available {
		t.Error("a result with no details must not offer a restart")
	}
}

// Resolution must be by unit, never by command-line pattern. A `pgrep -f`
// search matches any process whose command line contains the string — including
// the shell of whoever is grepping for it — and a collision would report a
// stale service as healthy. This pins the seam's contract: the resolver is
// asked for a unit name and nothing else.
func TestResolverIsAskedForUnitNamesNotPatterns(t *testing.T) {
	if checkOS != "linux" {
		repocontracttest.SkipPlatform(t, "stale binary check reads /proc")
	}
	seen := map[string]bool{}
	spy := &recordingResolver{seen: seen}

	NewStaleServiceBinaryCheck(
		WithSupervisedUnits(twoUnits),
		WithProcessExeResolver(spy),
	).Run(context.Background())

	for _, want := range []string{"a.service", "b.service"} {
		if !seen[want] {
			t.Errorf("resolver was never asked about %q; asked for %v", want, seen)
		}
	}
}

type recordingResolver struct{ seen map[string]bool }

func (r *recordingResolver) Resolve(_ context.Context, unit string) (string, bool, bool) {
	r.seen[unit] = true
	return "", false, false
}
