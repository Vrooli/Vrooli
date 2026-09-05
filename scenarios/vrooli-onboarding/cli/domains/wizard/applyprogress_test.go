package wizard

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeClock struct{ at time.Time }

func (c *fakeClock) now() time.Time          { return c.at }
func (c *fakeClock) advance(d time.Duration) { c.at = c.at.Add(d) }

func newTestReporter() (*applyProgressReporter, *bytes.Buffer, *fakeClock) {
	buffer := &bytes.Buffer{}
	clock := &fakeClock{at: time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)}
	return newApplyProgressReporter(buffer, clock.now), buffer, clock
}

func item(id, kind, name, outcome string) applyRunItem {
	return applyRunItem{ID: id, Kind: kind, Name: name, Outcome: outcome}
}

// The operator pressed enter and saw nothing for minutes. Silence during a
// long step is the defect, so the reporter must name the item it is working on
// as soon as the runner picks it up.
func TestApplyProgressAnnouncesTheInFlightItem(t *testing.T) {
	reporter, out, _ := newTestReporter()

	reporter.Observe([]applyRunItem{
		item("safeguard:remote_desktop_access", "safeguard", "remote_desktop_access", "applying"),
		item("resource:postgres", "resource", "postgres", ""),
	})

	output := out.String()
	for _, want := range []string{"APPLY  · Applying 2 item(s)", "↳ safeguard remote_desktop_access (1/2)"} {
		if !strings.Contains(output, want) {
			t.Fatalf("missing %q in:\n%s", want, output)
		}
	}
}

// A poll every 500ms must not print every 500ms. The item is announced once and
// then only heartbeats.
func TestApplyProgressAnnouncesEachItemOnce(t *testing.T) {
	reporter, out, _ := newTestReporter()
	snapshot := []applyRunItem{item("resource:postgres", "resource", "postgres", "applying")}

	for range 5 {
		reporter.Observe(snapshot)
	}

	if got := strings.Count(out.String(), "↳ resource postgres"); got != 1 {
		t.Fatalf("in-flight item announced %d times, want 1:\n%s", got, out.String())
	}
}

// The heartbeat is the whole point: it distinguishes a five-minute scenario
// start from a wedged run.
func TestApplyProgressHeartbeatsWhileAnItemIsSlow(t *testing.T) {
	reporter, out, clock := newTestReporter()
	snapshot := []applyRunItem{item("scenario:web-console", "scenario", "web-console", "applying")}

	reporter.Observe(snapshot)
	if strings.Contains(out.String(), "still working") {
		t.Fatalf("heartbeat fired before the first interval elapsed:\n%s", out.String())
	}

	clock.advance(applyFirstHeartbeat)
	reporter.Observe(snapshot)
	if !strings.Contains(out.String(), "still working: scenario web-console (10s elapsed)") {
		t.Fatalf("expected a first heartbeat at %s:\n%s", applyFirstHeartbeat, out.String())
	}

	clock.advance(applyHeartbeatEvery)
	reporter.Observe(snapshot)
	if got := strings.Count(out.String(), "still working"); got != 2 {
		t.Fatalf("heartbeat count = %d, want 2:\n%s", got, out.String())
	}
}

func TestApplyProgressReportsOutcomesAndTiming(t *testing.T) {
	reporter, out, clock := newTestReporter()
	id := "resource:postgres"

	reporter.Observe([]applyRunItem{item(id, "resource", "postgres", "applying")})
	clock.advance(42 * time.Second)
	reporter.Observe([]applyRunItem{item(id, "resource", "postgres", "applied")})

	if !strings.Contains(out.String(), "✓ resource postgres applied (42s)") {
		t.Fatalf("expected a timed completion line:\n%s", out.String())
	}
}

// A failure must not render with the success marker, and its reason must be
// visible where it happened rather than only in the final report.
func TestApplyProgressSurfacesFailureReasonInline(t *testing.T) {
	reporter, out, _ := newTestReporter()
	failed := applyRunItem{ID: "tool:kopia", Kind: "tool", Name: "kopia", Outcome: "failed", Error: "exit status 1"}

	reporter.Observe([]applyRunItem{failed})

	output := out.String()
	if !strings.Contains(output, "! tool kopia failed") {
		t.Fatalf("failure not marked as such:\n%s", output)
	}
	if !strings.Contains(output, "exit status 1") {
		t.Fatalf("failure reason not shown inline:\n%s", output)
	}
	if strings.Contains(output, "✓ tool kopia") {
		t.Fatalf("failure rendered with the success marker:\n%s", output)
	}
}

// A partially applied run must not close with wording that reads as clean.
func TestApplyProgressFinishTalliesEveryOutcome(t *testing.T) {
	reporter, out, _ := newTestReporter()
	items := []applyRunItem{
		item("a", "tool", "git", "applied"),
		item("b", "tool", "jq", "applied"),
		item("c", "tool", "kopia", "failed"),
	}

	reporter.Observe(items)
	reporter.Finish("partially_applied", items)

	final := out.String()
	if !strings.Contains(final, "APPLY  · partially_applied") {
		t.Fatalf("final line does not carry the run status:\n%s", final)
	}
	if !strings.Contains(final, "2 applied") || !strings.Contains(final, "1 failed") {
		t.Fatalf("final tally does not report both outcomes:\n%s", final)
	}
}

// The API restarting mid-apply is expected on this path. Elapsed time measured
// across that gap is the wizard's downtime, not the item's, so it is dropped
// rather than reported as if the item had been running the whole time.
func TestApplyProgressDoesNotBillReconnectGapToTheItem(t *testing.T) {
	reporter, out, clock := newTestReporter()
	id := "scenario:web-console"

	reporter.Observe([]applyRunItem{item(id, "scenario", "web-console", "applying")})
	clock.advance(4 * time.Minute)
	reporter.Reconnected()
	reporter.Observe([]applyRunItem{item(id, "scenario", "web-console", "applied")})

	if strings.Contains(out.String(), "4m0s") {
		t.Fatalf("reconnect downtime billed to the item:\n%s", out.String())
	}
}

// A run with no items must not print an empty progress block above the report.
func TestApplyProgressStaysSilentWithoutItems(t *testing.T) {
	reporter, out, _ := newTestReporter()

	reporter.Observe(nil)
	reporter.Finish("already_satisfied", nil)

	if out.String() != "" {
		t.Fatalf("expected no output for an empty run, got:\n%s", out.String())
	}
}

// A control-plane failure carries the whole probe result -- the remote-desktop
// safeguard returns about 1.5KB of single-line JSON. Printing it raw buries the
// progress log the line was meant to annotate.
func TestApplyProgressKeepsTheInlineFailureLineReadable(t *testing.T) {
	reporter, out, _ := newTestReporter()
	blob := `control plane host safeguard remote_desktop_access failed: exit status 1: {"name":"remote_desktop_access","notes":["` +
		strings.Repeat("very long probe output ", 80) + `"]}`

	reporter.Observe([]applyRunItem{{ID: "safeguard:rd", Kind: "safeguard", Name: "remote_desktop_access", Outcome: "failed", Error: blob}})

	output := out.String()
	if strings.Contains(output, "very long probe output") {
		t.Fatalf("raw probe payload printed inline:\n%s", output)
	}
	if !strings.Contains(output, "control plane host safeguard remote_desktop_access failed") {
		t.Fatalf("the recognisable part of the failure was dropped:\n%s", output)
	}
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if len(line) > applyErrorSummaryLimit+40 {
			t.Fatalf("progress line is %d chars, too long to read:\n%s", len(line), line)
		}
	}
}

// A run is accepted with every item already marked "pending". Treating that as
// a finished outcome printed the entire selection as complete -- with a failure
// marker -- the instant the operator consented, before any work had happened.
func TestApplyProgressTreatsPendingItemsAsNotStarted(t *testing.T) {
	reporter, out, _ := newTestReporter()

	reporter.Observe([]applyRunItem{
		item("tool:git", "tool", "git", "pending"),
		item("resource:postgres", "resource", "postgres", "pending"),
	})

	output := out.String()
	if strings.Contains(output, "✓") || strings.Contains(output, "!") {
		t.Fatalf("a freshly accepted run reported finished items:\n%s", output)
	}
	if !strings.Contains(output, "APPLY  · Applying 2 item(s)") {
		t.Fatalf("expected the opening line:\n%s", output)
	}
}

// The in-flight marker crosses a module boundary: the API writes it, this CLI
// reads it. Nothing in the compiler connects the two, so the literal is pinned
// against the API source. If that constant is renamed, every heartbeat above
// silently stops firing while the tests still pass.
func TestApplyingOutcomeMatchesTheAPIContract(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "..", "api", "v2_apply.go"))
	if err != nil {
		t.Fatalf("read the API apply source: %v", err)
	}
	want := `applyOutcomeApplying = "` + applyOutcomeApplying + `"`
	if !strings.Contains(string(source), want) {
		t.Fatalf("the API no longer declares %s; the wizard's progress markers are reading a value nothing writes", want)
	}
	// The pending marker has no constant on the API side; it is written as a
	// literal when the run is built. Pin the literal so a rename there cannot
	// silently turn every unstarted item into a reported failure here.
	if !strings.Contains(string(source), `Outcome: "`+applyOutcomePending+`"`) {
		t.Fatalf("the API no longer writes %q as the initial item outcome", applyOutcomePending)
	}
}
