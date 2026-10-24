package focus

import (
	"context"
	"testing"
	"time"

	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
)

type stubConditionReader struct{ snapshot internalcondition.Snapshot }

func (s stubConditionReader) ReadAll(context.Context) internalcondition.Snapshot { return s.snapshot }

func floatPtr(value float64) *float64 { return &value }

func availableSource() internalcondition.SourceAvailability {
	return internalcondition.SourceAvailability{Source: "vrooli-autoheal", Available: true, CheckedAt: time.Now()}
}

// The ranked surface must be able to see an out-of-band substrate reading.
// Before the condition join existed, a critical GPU severity produced no
// finding at all while a documentation gap ranked first.
func TestConditionSourceRanksOutOfBandSubstrateAboveCoverageGaps(t *testing.T) {
	source := ConditionSource{Condition: stubConditionReader{snapshot: internalcondition.Snapshot{
		Sources: []internalcondition.SourceAvailability{availableSource()},
		Readings: []internalcondition.Observation{{
			ID: "system-gpu", CellRef: "substrate/SB4", Value: 2, Unit: "severity",
			Trust: internalcondition.TrustValid, BandVerdict: internalcondition.BandOutOfBand,
			Band: internalcondition.Band{Max: floatPtr(0)},
		}},
	}}}
	findings, gaps, err := source.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected one finding, got %d", len(findings))
	}
	if findings[0].Stage != StageSubstrate {
		t.Fatalf("stage = %v, want StageSubstrate", findings[0].Stage)
	}
	if findings[0].SensorRef != "system-gpu" {
		t.Fatalf("sensor ref = %q, want the contributing check", findings[0].SensorRef)
	}

	// A coverage gap is measurement-improvement, the outermost tier. Ranking
	// the two together must put the substrate excursion first.
	ranked := Rank(append(findings, Finding{ID: "coverage-gap", Stage: StageMeasurement, Severity: 1}))
	if ranked[0].ID != findings[0].ID {
		t.Fatalf("rank 1 = %q, want the substrate excursion", ranked[0].ID)
	}
	if !sourceAvailable(gaps, "out-of-band") {
		t.Fatal("out-of-band source should report available")
	}
}

// An untrusted reading is a fault in the measuring channel, so it ranks at the
// integrity tier and routes to the instrument's owner, not to plant work.
func TestConditionSourceRanksUntrustedAtIntegrity(t *testing.T) {
	source := ConditionSource{Condition: stubConditionReader{snapshot: internalcondition.Snapshot{
		Sources: []internalcondition.SourceAvailability{availableSource()},
		Readings: []internalcondition.Observation{{
			ID: "system-mce-recent", CellRef: "substrate/SB3", Unit: "severity",
			Trust: internalcondition.TrustUntrusted, BandVerdict: internalcondition.BandNotEvaluated,
			UnavailableReason: "ras-mc-ctl rejected --since",
		}},
	}}}
	findings, _, err := source.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(findings) != 1 || findings[0].Stage != StageIntegrity {
		t.Fatalf("findings = %#v, want one integrity-stage finding", findings)
	}
	// A bare verdict is not actionable; the offending signal must be named.
	if findings[0].Message == "" || findings[0].Message == string(internalcondition.TrustUntrusted) {
		t.Fatalf("message = %q, want the underlying reason", findings[0].Message)
	}
}

// A shelved check is a deliberate stop, not a broken sensor, so it must not be
// reported as an integrity fault.
func TestConditionSourceDoesNotTreatShelvedAsUntrusted(t *testing.T) {
	source := ConditionSource{Condition: stubConditionReader{snapshot: internalcondition.Snapshot{
		Sources: []internalcondition.SourceAvailability{availableSource()},
		Readings: []internalcondition.Observation{{
			ID: "scenario-paused", CellRef: "availability/A1", Trust: internalcondition.TrustShelved,
		}},
	}}}
	findings, _, err := source.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings for a shelved check, got %#v", findings)
	}
}

// Zero out-of-band readings from an unreadable source is a clean sheet earned
// by not looking. The source must report itself unavailable instead.
func TestConditionSourceReportsUnavailableWhenAPeerCannotBeRead(t *testing.T) {
	source := ConditionSource{Condition: stubConditionReader{snapshot: internalcondition.Snapshot{
		Sources: []internalcondition.SourceAvailability{{Source: "vrooli-autoheal", Available: false, Reason: "deadline exceeded"}},
	}}}
	_, gaps, err := source.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	for _, id := range []string{"out-of-band", "untrusted"} {
		if sourceAvailable(gaps, id) {
			t.Fatalf("source %q reported available despite an unreadable peer", id)
		}
	}
}

// The merged surface is what a member reads with one call; both joins must
// contribute their own availability rather than one speaking for the other.
func TestMergedSourceReportsEveryGapSource(t *testing.T) {
	merged := MergedSource{Sources: []Source{
		staticSource{gaps: []GapSource{{ID: "open-loop", Available: true, FindingCount: 6}}},
		ConditionSource{Condition: stubConditionReader{snapshot: internalcondition.Snapshot{
			Sources: []internalcondition.SourceAvailability{availableSource()},
		}}},
	}}
	_, gaps, err := merged.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	for _, id := range []string{"open-loop", "out-of-band", "untrusted"} {
		if findGap(gaps, id) == nil {
			t.Fatalf("merged gap sources are missing %q: %#v", id, gaps)
		}
	}
}

type staticSource struct {
	findings []Finding
	gaps     []GapSource
}

func (s staticSource) Read(context.Context) ([]Finding, []GapSource, error) {
	return s.findings, s.gaps, nil
}

func findGap(gaps []GapSource, id string) *GapSource {
	for i := range gaps {
		if gaps[i].ID == id {
			return &gaps[i]
		}
	}
	return nil
}

func sourceAvailable(gaps []GapSource, id string) bool {
	gap := findGap(gaps, id)
	return gap != nil && gap.Available
}

// An instrument that floods its own ranked surface with one row per sensor is
// reproducing the alarm flood it exists to report. Above the threshold a group
// collapses to one counted finding; below it, per-sensor findings survive so
// efficacy can still be re-read against a named sensor.
func TestConditionSourceCollapsesAFloodButKeepsSmallGroupsPerSensor(t *testing.T) {
	readings := make([]internalcondition.Observation, 0, 12)
	for i := 0; i < 12; i++ {
		readings = append(readings, internalcondition.Observation{
			ID: "check-" + string(rune('a'+i)), CellRef: "availability/A1", Value: 90, Unit: "percent",
			Trust: internalcondition.TrustValid, BandVerdict: internalcondition.BandOutOfBand,
			Band: internalcondition.Band{Min: floatPtr(99.5)},
		})
	}
	// Two substrate cells, one reading each — well under the threshold.
	readings = append(readings,
		internalcondition.Observation{
			ID: "system-gpu", CellRef: "substrate/SB4", Value: 2, Unit: "severity",
			Trust: internalcondition.TrustValid, BandVerdict: internalcondition.BandOutOfBand,
			Band: internalcondition.Band{Max: floatPtr(0)},
		},
		internalcondition.Observation{
			ID: "system-mce-recent", CellRef: "substrate/SB3", Value: 1, Unit: "severity",
			Trust: internalcondition.TrustValid, BandVerdict: internalcondition.BandOutOfBand,
			Band: internalcondition.Band{Max: floatPtr(0)},
		},
	)
	source := ConditionSource{Condition: stubConditionReader{snapshot: internalcondition.Snapshot{
		Sources: []internalcondition.SourceAvailability{availableSource()}, Readings: readings,
	}}}
	findings, gaps, err := source.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	byCell := map[string]int{}
	for _, finding := range findings {
		byCell[finding.CellRef]++
	}
	if byCell["availability/A1"] != 1 {
		t.Fatalf("flooded cell produced %d findings, want 1 aggregate", byCell["availability/A1"])
	}
	if byCell["substrate/SB4"] != 1 || byCell["substrate/SB3"] != 1 {
		t.Fatalf("substrate cells = %#v, want one finding each", byCell)
	}
	// Collapsing must not hide the true volume.
	gap := findGap(gaps, "out-of-band")
	if gap == nil || gap.FindingCount != 14 {
		t.Fatalf("out-of-band count = %#v, want the full 14 readings", gap)
	}
	// The substrate excursion must still outrank the availability aggregate.
	ranked := Rank(findings)
	if projectionOf(ranked[0].CellRef) != "substrate" {
		t.Fatalf("rank 1 = %q, want a substrate finding", ranked[0].CellRef)
	}
}
