package services

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
)

func bandTestSettings() Settings {
	s := defaultSettings
	s.Active = true
	s.DiskThreshold = 80
	s.DiskHighPercent = 90
	s.DiskCriticalPercent = 95
	s.DiskEscalationDebounceTicks = 1
	s.DiskEscalationCooldownSeconds = 1800
	s.DiskFastFillJumpPercent = 5
	return s
}

// TestClassifyBand walks usage across every boundary. The classifier is pure,
// so the whole table can be asserted directly.
func TestClassifyBand(t *testing.T) {
	settings := bandTestSettings()

	tests := []struct {
		percent float64
		want    PressureBand
	}{
		{0, BandNormal},
		{70, BandNormal},
		{79.9, BandNormal},
		{80, BandWarning}, // boundary is inclusive
		{85, BandWarning},
		{89.9, BandWarning},
		{90, BandHigh},
		{92, BandHigh},
		{94.9, BandHigh},
		{95, BandCritical},
		{96, BandCritical},
		{100, BandCritical},
	}

	for _, tc := range tests {
		if got := classifyBand(tc.percent, settings); got != tc.want {
			t.Errorf("classifyBand(%v) = %s, want %s", tc.percent, got, tc.want)
		}
	}
}

// TestClassifyBand_UsesSettingsNotConstants proves the boundaries are read from
// settings. If any boundary were hardcoded, moving all three would not move the
// classification.
func TestClassifyBand_UsesSettingsNotConstants(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskThreshold = 50
	settings.DiskHighPercent = 60
	settings.DiskCriticalPercent = 70

	tests := []struct {
		percent float64
		want    PressureBand
	}{
		{49, BandNormal},
		{50, BandWarning},
		{60, BandHigh},
		{70, BandCritical},
		// Values that would be warning/high/critical under the defaults are all
		// critical here.
		{85, BandCritical},
		{92, BandCritical},
	}

	for _, tc := range tests {
		if got := classifyBand(tc.percent, settings); got != tc.want {
			t.Errorf("with custom bands classifyBand(%v) = %s, want %s", tc.percent, got, tc.want)
		}
	}
}

// trackerRun feeds a sequence of percentages through a tracker and returns the
// bands and the samples that produced a record.
func trackerRun(t *testing.T, settings Settings, percents []float64, tickInterval time.Duration) (bands []PressureBand, emitted []float64, transitions []*BandObservation) {
	t.Helper()

	tracker := &bandTracker{}
	now := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	previous := 0.0
	hasPrevious := false

	for _, pct := range percents {
		usage := usageAt(pct)
		decision := tracker.evaluate(usage, previous, hasPrevious, settings, now)
		bands = append(bands, decision.Band)
		if decision.Emit {
			emitted = append(emitted, pct)
		}
		if decision.Transition != nil {
			transitions = append(transitions, decision.Transition)
		}
		previous = pct
		hasPrevious = true
		now = now.Add(tickInterval)
	}
	return bands, emitted, transitions
}

// TestBandTracker_EmitsOncePerBandEntered walks usage from 70 to 96 percent and
// asserts one record per band, not one per tick.
//
// This is the property that makes escalation usable. During the incident the
// disk sat above its threshold for days; a level-only rule would have produced
// thousands of identical records.
func TestBandTracker_EmitsOncePerBandEntered(t *testing.T) {
	settings := bandTestSettings()

	// Rising slowly enough that the fast-fill bypass never engages.
	percents := []float64{70, 72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 91, 92, 93, 94, 95, 96}
	bands, emitted, transitions := trackerRun(t, settings, percents, 20*time.Second)

	if got, want := bands[len(bands)-1], BandCritical; got != want {
		t.Errorf("final band = %s, want %s", got, want)
	}

	// Exactly three escalations: normal→warning, warning→high, high→critical.
	if len(emitted) != 3 {
		t.Fatalf("emitted %d records for %d ticks, want 3 (one per band entered): %v", len(emitted), len(percents), emitted)
	}
	if emitted[0] != 80 || emitted[1] != 90 || emitted[2] != 95 {
		t.Errorf("records emitted at %v, want at the boundary crossings 80, 90, 95", emitted)
	}

	// Every escalation must carry the observation that caused it.
	for _, tr := range transitions {
		if tr.At.IsZero() {
			t.Error("transition recorded with no timestamp")
		}
		if tr.UsedPercent == 0 {
			t.Error("transition recorded with no usage evidence")
		}
	}
}

// TestBandTracker_HeldAtBandEdgeEmitsOncePerCooldown asserts a disk parked
// inside one band across many ticks does not re-alert on every tick.
func TestBandTracker_HeldAtBandEdgeEmitsOncePerCooldown(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationCooldownSeconds = 1800 // 30 minutes

	// Twenty ticks a minute apart, all inside the warning band.
	percents := make([]float64, 20)
	for i := range percents {
		percents[i] = 85
	}
	_, emitted, _ := trackerRun(t, settings, percents, time.Minute)

	// 20 minutes of ticks is less than one 30-minute cooldown after the first
	// record, so exactly one record is correct.
	if len(emitted) != 1 {
		t.Errorf("a disk held at 85%% for 20 ticks emitted %d records, want 1", len(emitted))
	}
}

// TestBandTracker_CooldownExpiryReEmits asserts pressure that persists past the
// cooldown is re-reported, so a long-running problem is not forgotten.
func TestBandTracker_CooldownExpiryReEmits(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationCooldownSeconds = 600 // 10 minutes

	percents := make([]float64, 20)
	for i := range percents {
		percents[i] = 85
	}
	// Ticks five minutes apart across 100 minutes.
	_, emitted, _ := trackerRun(t, settings, percents, 5*time.Minute)

	if len(emitted) < 2 {
		t.Errorf("sustained pressure across 100 minutes emitted %d records, want repeats once the 10-minute cooldown expires", len(emitted))
	}
}

// TestBandTracker_DeEscalationResetsCooldown walks usage back down and asserts
// the band falls immediately and the next escalation is not suppressed by the
// previous band's cooldown.
func TestBandTracker_DeEscalationResetsCooldown(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationCooldownSeconds = 3600 // deliberately long

	// Up into high, back down to normal, then straight back up into high.
	percents := []float64{92, 92, 70, 70, 92}
	bands, emitted, _ := trackerRun(t, settings, percents, time.Minute)

	if bands[0] != BandHigh {
		t.Errorf("band after first high sample = %s, want high", bands[0])
	}
	if bands[2] != BandNormal {
		t.Errorf("band after dropping to 70%% = %s, want normal (de-escalation is not debounced)", bands[2])
	}
	if bands[4] != BandHigh {
		t.Errorf("band after climbing back = %s, want high", bands[4])
	}

	// Two escalations into high, despite a one-hour cooldown and only five
	// minutes of elapsed time: crossing back down resets the cooldown.
	if len(emitted) != 2 {
		t.Errorf("emitted %d records, want 2 (the cooldown must reset on de-escalation): %v", len(emitted), emitted)
	}
}

// TestBandTracker_DebounceDelaysEscalation asserts a new band must be observed
// for the configured number of consecutive ticks before it takes effect.
func TestBandTracker_DebounceDelaysEscalation(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationDebounceTicks = 2

	// Rise by less than the fast-fill jump so debounce is what governs.
	bands, emitted, _ := trackerRun(t, settings, []float64{78, 81, 83}, 20*time.Second)

	if bands[0] != BandNormal {
		t.Errorf("band at 78%% = %s, want normal", bands[0])
	}
	if bands[1] != BandNormal {
		t.Errorf("band at the first 81%% sample = %s, want normal (one observation is not enough)", bands[1])
	}
	if bands[2] != BandWarning {
		t.Errorf("band at the second in-band sample = %s, want warning", bands[2])
	}
	if len(emitted) != 1 || emitted[0] != 83 {
		t.Errorf("records emitted at %v, want a single record once debounce completed at 83", emitted)
	}
}

// TestBandTracker_SingleNoisySampleDoesNotEscalate asserts one outlier tick is
// absorbed rather than escalating on its own.
func TestBandTracker_SingleNoisySampleDoesNotEscalate(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationDebounceTicks = 2

	// The outlier rises by 3 points, below the 5-point fast-fill bypass, so
	// debounce is genuinely what is under test here.
	bands, emitted, _ := trackerRun(t, settings, []float64{78, 81, 78, 78}, 20*time.Second)

	for i, b := range bands {
		if b != BandNormal {
			t.Errorf("band at tick %d = %s, want normal throughout", i, b)
		}
	}
	if len(emitted) != 0 {
		t.Errorf("a single noisy sample emitted %d records, want 0", len(emitted))
	}
}

// TestBandTracker_FastFillSkipsDebounce asserts a large single-tick jump
// escalates immediately.
//
// This bounds the delay debounce introduces. The incident's own growth was 3-5
// GB per day, but a runaway process can fill 100 GB in minutes, and waiting for
// a second confirming tick is exactly the wrong response to that.
func TestBandTracker_FastFillSkipsDebounce(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationDebounceTicks = 5 // would normally stall for five ticks
	settings.DiskFastFillJumpPercent = 5

	bands, emitted, transitions := trackerRun(t, settings, []float64{70, 96}, 20*time.Second)

	if bands[1] != BandCritical {
		t.Fatalf("band after a 26-point single-tick jump = %s, want critical without waiting for debounce", bands[1])
	}
	if len(emitted) != 1 || emitted[0] != 96 {
		t.Errorf("records emitted at %v, want an immediate record at 96", emitted)
	}
	if len(transitions) == 0 || !transitions[len(transitions)-1].FastFill {
		t.Error("the escalation was not marked as a fast fill, so an operator cannot tell why debounce was bypassed")
	}
}

// TestBandTracker_SlowRiseStillDebounces is the counterpart: a rise smaller
// than the fast-fill jump must not bypass debounce.
func TestBandTracker_SlowRiseStillDebounces(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationDebounceTicks = 2
	settings.DiskFastFillJumpPercent = 10

	// A 3-point rise across the warning boundary: under the 10-point bypass.
	bands, _, _ := trackerRun(t, settings, []float64{78, 81}, 20*time.Second)

	if bands[1] != BandNormal {
		t.Errorf("a 3-point rise with a 10-point fast-fill setting escalated immediately (band = %s); debounce was bypassed", bands[1])
	}
}

// TestBandTracker_NormalBandNeverEmits asserts healthy usage stays silent no
// matter how long it is observed.
func TestBandTracker_NormalBandNeverEmits(t *testing.T) {
	settings := bandTestSettings()
	settings.DiskEscalationCooldownSeconds = 1

	percents := make([]float64, 50)
	for i := range percents {
		percents[i] = 40
	}
	_, emitted, transitions := trackerRun(t, settings, percents, time.Minute)

	if len(emitted) != 0 {
		t.Errorf("healthy usage emitted %d records, want 0", len(emitted))
	}
	if len(transitions) != 0 {
		t.Errorf("healthy usage recorded %d band transitions, want 0", len(transitions))
	}
}

// TestSanitizeDiskBands_RepairsNonAscendingOrder asserts a configuration whose
// bands do not ascend is repaired rather than obeyed. Bands out of order make a
// higher band unreachable, so pressure could climb past critical while only
// ever classifying as warning.
func TestSanitizeDiskBands_RepairsNonAscendingOrder(t *testing.T) {
	tests := []struct {
		name                    string
		warning, high, critical float64
	}{
		{"high below warning", 90, 80, 95},
		{"critical below high", 80, 95, 90},
		{"all inverted", 95, 90, 80},
		{"warning equals high", 90, 90, 95},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			settings := defaultSettings
			settings.DiskThreshold = tc.warning
			settings.DiskHighPercent = tc.high
			settings.DiskCriticalPercent = tc.critical

			changed := sanitizeDiskBands(&settings)
			if !changed {
				t.Fatal("a non-ascending band configuration was accepted unchanged")
			}
			if !diskBandsAscend(settings) {
				t.Errorf("bands still do not ascend after repair: %v < %v < %v",
					settings.DiskThreshold, settings.DiskHighPercent, settings.DiskCriticalPercent)
			}
		})
	}
}

// TestSanitizeDiskBands_FillsUnsetValues asserts a settings file written before
// the bands existed gets working defaults rather than zeros, which would
// classify every sample as critical.
func TestSanitizeDiskBands_FillsUnsetValues(t *testing.T) {
	settings := Settings{DiskThreshold: 80}

	sanitizeDiskBands(&settings)

	if settings.DiskHighPercent != defaultSettings.DiskHighPercent {
		t.Errorf("DiskHighPercent = %v, want the default %v", settings.DiskHighPercent, defaultSettings.DiskHighPercent)
	}
	if settings.DiskCriticalPercent != defaultSettings.DiskCriticalPercent {
		t.Errorf("DiskCriticalPercent = %v, want the default %v", settings.DiskCriticalPercent, defaultSettings.DiskCriticalPercent)
	}
	if settings.DiskEscalationDebounceTicks <= 0 {
		t.Error("DiskEscalationDebounceTicks was left at zero, which would disable debounce")
	}
	if settings.DiskEscalationCooldownSeconds <= 0 {
		t.Error("DiskEscalationCooldownSeconds was left at zero, which would re-alert every tick")
	}
	if settings.DiskFastFillJumpPercent <= 0 {
		t.Error("DiskFastFillJumpPercent was left at zero, which would treat every rise as a fast fill")
	}
	if classifyBand(10, settings) != BandNormal {
		t.Error("healthy usage classified as pressure after sanitising an unset band configuration")
	}
}

// usageAt builds a DiskUsage at a given percentage for band tests.
func usageAt(pct float64) collectors.DiskUsage {
	return collectors.DiskUsage{
		TotalBytes:     1000,
		UsedBytes:      int64(pct * 10),
		AvailableBytes: int64((100 - pct) * 10),
		UsedPercent:    pct,
	}
}

// TestPressureBandJSON asserts bands serialise by name, so the operator
// surface is legible during an incident, and that an unknown name is rejected
// rather than decoding to normal.
func TestPressureBandJSON(t *testing.T) {
	for _, band := range []PressureBand{BandNormal, BandWarning, BandHigh, BandCritical} {
		encoded, err := json.Marshal(band)
		if err != nil {
			t.Fatalf("marshal %s: %v", band, err)
		}
		if string(encoded) != `"`+band.String()+`"` {
			t.Errorf("marshalled %s as %s, want the name", band, encoded)
		}

		var decoded PressureBand
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal %s: %v", encoded, err)
		}
		if decoded != band {
			t.Errorf("round-trip of %s produced %s", band, decoded)
		}
	}

	var decoded PressureBand
	if err := json.Unmarshal([]byte(`"catastrophic"`), &decoded); err == nil {
		t.Error("an unknown band name decoded successfully; it must be rejected")
	}
}
