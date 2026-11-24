package launch

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"

	"research-desk/internal/home"
)

// The charter already tells a manager to wind down at the allowance end. These
// tests pin the missing control: the desk writes the same wind-down once near
// the edge, then pauses a goal that ignored it after the grace.
func TestWindDownWritesOnceAtTheAllowanceEdgeThenPausesAfterGrace(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	clock := r.service.Clock.(*schedule.Fake)
	now := clock.Now()
	r.reader.states[handRun] = home.StateLive
	r.reader.sessions[handRun] = "sess-1"
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.State = home.StateLive
		c.Started = home.Time{Value: now.Add(-9 * time.Hour), Evidence: home.Evidence{Known: true}}
		c.AllowanceEnd = home.Time{Value: now.Add(31 * time.Minute), Evidence: home.Evidence{Known: true}}
	}

	// More than the lead before the edge: nothing is written.
	if acts := r.service.WindDown(ctx); len(acts) != 0 {
		t.Fatalf("acted before the wind-down lead: %q", acts)
	}
	if len(r.codex.typed()) != 0 {
		t.Fatalf("typed before the wind-down lead: %q", r.codex.typed())
	}

	// At the lead edge: one wind-down entry, no pause.
	clock.Advance(time.Minute)
	if acts := r.service.WindDown(ctx); len(acts) != 1 || !strings.Contains(acts[0], "wind-down written") {
		t.Fatalf("acts = %q, want one wind-down", acts)
	}
	operatorPath := filepath.Join(r.research, "cohorts", handRun, home.OperatorFile)
	operatorText, err := os.ReadFile(operatorPath)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ := home.ParseEntries(string(operatorText))
	if len(entries) != 1 || entries[0].Kind != home.EntryWindDown || entries[0].Body != home.WindDownText {
		t.Fatalf("operator entries = %+v", entries)
	}
	if len(r.codex.typed()) != 0 {
		t.Fatalf("paused before the allowance: %q", r.codex.typed())
	}

	// A second pass does not duplicate the entry.
	if acts := r.service.WindDown(ctx); len(acts) != 0 {
		t.Fatalf("a second pass wrote again: %q", acts)
	}
	operatorText, err = os.ReadFile(operatorPath)
	if err != nil {
		t.Fatal(err)
	}
	entries, _ = home.ParseEntries(string(operatorText))
	if len(entries) != 1 {
		t.Fatalf("duplicate wind-down entries = %+v", entries)
	}

	// Before the grace, the manager still owns the closeout.
	clock.Advance(14 * time.Minute)
	if acts := r.service.WindDown(ctx); len(acts) != 0 || len(r.codex.typed()) != 0 {
		t.Fatalf("paused inside the grace: %q %q", acts, r.codex.typed())
	}

	// At the grace end (allowance end + 15 minutes), the desk pauses the goal
	// and records that act.
	clock.Advance(31 * time.Minute)
	if acts := r.service.WindDown(ctx); len(acts) != 1 || !strings.Contains(acts[0], "auto-pause") {
		t.Fatalf("acts = %q, want one auto-pause", acts)
	}
	if got := r.codex.typed(); !slices.Equal(got, []string{"type:/goal pause", "key:Enter"}) {
		t.Fatalf("typed = %q, want /goal pause", got)
	}
	controls := r.launchFile(t, handRun).Controls
	if len(controls) != 1 || controls[0].Action != "auto-pause" {
		t.Fatalf("controls = %+v", controls)
	}

	// The recorded pause is not retried every minute.
	if acts := r.service.WindDown(ctx); len(acts) != 0 {
		t.Fatalf("retried the pause: %q", acts)
	}
}

func TestWindDownLeavesAPausedOrDistantCommissionAlone(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	clock := r.service.Clock.(*schedule.Fake)
	now := clock.Now()
	r.reader.sessions[handRun] = "sess-1"

	r.reader.states[handRun] = home.StatePaused
	r.reader.tweaks[handRun] = func(c *home.Commission) {
		c.AllowanceEnd = home.Time{Value: now.Add(-time.Hour), Evidence: home.Evidence{Known: true}}
	}
	if acts := r.service.WindDown(ctx); len(acts) != 0 {
		t.Fatalf("acted on a paused commission: %q", acts)
	}

	r.reader.states[handRun] = home.StateLive
	if acts := r.service.WindDown(ctx); len(acts) != 1 || len(r.codex.typed()) != 0 {
		t.Fatalf("a live commission past its allowance should get one wind-down: %q %q", acts, r.codex.typed())
	}
}
