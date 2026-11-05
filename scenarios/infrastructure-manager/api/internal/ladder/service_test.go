package ladder

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/spacedoc"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
	internalcoverage "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/coverage"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/focus"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/sources"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".vrooli", "capability-vocabulary.json")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no repository root was found above the test directory")
		}
		dir = parent
	}
}

type stubGraph struct {
	graph sources.DeviceGraph
	err   error
	delay time.Duration
}

func (s stubGraph) ReadGraph(ctx context.Context) (sources.DeviceGraph, error) {
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return sources.DeviceGraph{}, ctx.Err()
		}
	}
	return s.graph, s.err
}

type stubGrid struct {
	grid portability.Grid
	err  error
}

func (s stubGrid) ReadGrid(context.Context) (portability.Grid, error) { return s.grid, s.err }

type stubChecks struct {
	checks []sources.CheckPlatforms
	err    error
}

func (s stubChecks) ReadPlatforms(context.Context) ([]sources.CheckPlatforms, error) {
	return s.checks, s.err
}

func measuredDevice(id, class string, rungs map[Rung]Observation) sources.GraphDevice {
	states := make(map[string]sources.RungState, len(rungs))
	for rung, observation := range rungs {
		states[string(rung)] = sources.RungState{
			Rung:   string(rung),
			State:  string(observation),
			Reason: "fixture",
		}
	}
	return sources.GraphDevice{ID: id, Class: class, Rungs: states}
}

// fullyMeasured grades every rung of the ladder, so a cell built from it is
// limited by the join rather than by a blind lower rung.
func fullyMeasured(id, class string) sources.GraphDevice {
	rungs := make(map[Rung]Observation, len(Rungs))
	for _, rung := range Rungs {
		rungs[rung] = ObservationMeasured
	}
	return measuredDevice(id, class, rungs)
}

// healthyThermalSensor is fully graded AND publishes the readings the SB11 bar
// is authored in, so it exercises the grading path rather than stopping at the
// unit check.
func healthyThermalSensor(id string, celsius, critical float64) sources.GraphDevice {
	device := fullyMeasured(id, "thermal-sensor")
	device.Readings = map[string]float64{
		"temperature_celsius":       celsius,
		"setpoint_critical_celsius": critical,
	}
	return device
}

func newService(t *testing.T, graph DeviceGraphSource, grid PortabilitySource, checks CheckPlatformSource) *Service {
	t.Helper()
	root := repoRoot(t)
	return &Service{
		Coverage:    internalcoverage.NewService(root, nil),
		DeviceGraph: graph,
		Portability: grid,
		Checks:      checks,
		HostOS:      "linux",
		Now:         func() time.Time { return time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC) },
	}
}

func liveGrid(t *testing.T) portability.Grid {
	t.Helper()
	reader, err := portability.NewReader(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	grid, err := reader.Grid(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return grid
}

func cellFor(t *testing.T, snapshot Snapshot, cellRef, class, hostOS string) Cell {
	t.Helper()
	for _, cell := range snapshot.Cells {
		if cell.CellRef == cellRef && cell.Key.DeviceClass == class && cell.Key.HostOS == hostOS {
			return cell
		}
	}
	t.Fatalf("no ladder cell for %s/%s/%s", cellRef, class, hostOS)
	return Cell{}
}

// TestEveryLadderCellResolvesToABarOrIsUngradedWithAReason is the honesty
// rule for the grading side: silently ungraded reads identically to in band.
func TestEveryLadderCellResolvesToABarOrIsUngradedWithAReason(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{healthyThermalSensor("thermal:0", 41, 95)}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())
	if len(snapshot.Cells) == 0 {
		t.Fatal("the ladder produced no cells")
	}
	for _, cell := range snapshot.Cells {
		if cell.Graded {
			if cell.BarID == "" {
				t.Errorf("%s is graded but names no bar", cell.Key)
			}
			continue
		}
		if strings.TrimSpace(cell.UngradedReason) == "" {
			t.Errorf("%s is ungraded and carries no reason", cell.Key)
		}
	}
}

func TestNonLocalHostCellsNameTheUnsampledHostReason(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{healthyThermalSensor("thermal:0", 41, 95)}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())
	for _, cell := range snapshot.Cells {
		if cell.Key.HostOS == service.hostOS() {
			continue
		}
		if cell.ReasonCode != "host_not_sampled" {
			t.Fatalf("non-local cell %s has reason code %q, want host_not_sampled", cell.Key, cell.ReasonCode)
		}
	}
}

// TestUnreachableSourceProducesASensorChannelFindingAndNoPlantFinding is the
// rule the whole trust axis exists for. Reporting instrument fault as plant
// fault routes real engineering effort at an imaginary problem.
func TestUnreachableSourceProducesASensorChannelFindingAndNoPlantFinding(t *testing.T) {
	service := newService(t,
		stubGraph{err: errors.New("connection refused")},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	integrity := 0
	for _, finding := range snapshot.Findings {
		switch finding.Stage {
		case focus.StageIntegrity:
			integrity++
		case focus.StageSubstrate:
			t.Errorf("an unreachable source produced a plant-side substrate finding: %s — %s", finding.Title, finding.Message)
		}
	}
	if integrity == 0 {
		t.Fatal("an unreachable source produced no sensor-channel finding; the outage would be invisible")
	}
	if snapshot.Findings[0].Stage != focus.StageIntegrity {
		t.Errorf("the top-ranked finding is stage %v, want sensor-channel integrity", snapshot.Findings[0].Stage)
	}
	if snapshot.Findings[0].RankExplanation == "" {
		t.Error("the ranking states no cascade stage explanation")
	}
}

// TestUnreachableSourceKeepsTheAuthoredCellStatus is the coverage-model rule:
// an owner outage must never read as a coverage collapse.
func TestUnreachableSourceKeepsTheAuthoredCellStatus(t *testing.T) {
	service := newService(t,
		stubGraph{err: errors.New("connection refused")},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	for _, cell := range snapshot.Cells {
		if cell.Status == spacedoc.StatusMissing {
			t.Errorf("%s was downgraded to MISSING while its source was unreachable: %s", cell.Key, cell.StatusSource)
		}
		if cell.Key.HostOS != "linux" {
			continue
		}
		if isCapabilityJoinClass(cell.Key.DeviceClass) {
			// These rows are backed by the check-platform source, not the
			// device graph that this test deliberately makes unreachable.
			continue
		}
		if cell.Trust != internalcondition.TrustUnavailable {
			t.Errorf("%s carries trust %s while its source was unreachable, want UNAVAILABLE", cell.Key, cell.Trust)
		}
		if cell.UnavailableReason == "" {
			t.Errorf("%s is unavailable with no reason", cell.Key)
		}
	}

	// The authored device-layer cells are IN-REACH, and they must still say so.
	thermal := cellFor(t, snapshot, "substrate/SB11", "thermal-sensor", "linux")
	if thermal.Status != spacedoc.StatusInReach {
		t.Errorf("substrate/SB11 reports status %q during the outage, want the authored in-reach", thermal.Status)
	}
	if thermal.StatusSource != "space document" {
		t.Errorf("substrate/SB11 status source is %q, want the space document", thermal.StatusSource)
	}
}

func isCapabilityJoinClass(class string) bool {
	for _, join := range CapabilityJoins {
		if join.DeviceClass == class {
			return true
		}
	}
	return false
}

// TestUnreadableDeviceIsUntrustedNotMissing separates the two blind states the
// trust vocabulary keeps apart: the source was read, and a device within it
// could not be.
func TestUnreadableDeviceIsUntrustedNotMissing(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{
			measuredDevice("thermal:0", "thermal-sensor", map[Rung]Observation{
				RungIdentity:  ObservationMeasured,
				RungTelemetry: ObservationUnmeasurable,
			}),
		}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	cell := cellFor(t, snapshot, "substrate/SB11", "thermal-sensor", "linux")
	if cell.Trust != internalcondition.TrustUntrusted {
		t.Errorf("an unreadable device produced trust %s, want UNTRUSTED", cell.Trust)
	}
	if cell.Status == spacedoc.StatusMissing {
		t.Error("an unreadable device downgraded its cell to MISSING")
	}
	if cell.Observation != ObservationUnmeasurable {
		t.Errorf("the cell reports %s, want unmeasurable", cell.Observation)
	}
	if cell.BlindDevices != 1 || cell.DeviceCount != 1 {
		t.Errorf("the cell reports %d blind of %d devices, want 1 of 1", cell.BlindDevices, cell.DeviceCount)
	}
	if cell.Band != internalcondition.BandNotEvaluated {
		t.Errorf("an untrusted reading produced band %s, want NOT_EVALUATED", cell.Band)
	}
}

// TestLiveJoinRefinesTheAuthoredStatusToNow is the closing half of the SB9-
// SB13 join: when the sensor is actually joined and grading, the cell stops
// being IN-REACH.
func TestLiveJoinRefinesTheAuthoredStatusToNow(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{
			CollectedAt: time.Date(2026, 8, 20, 11, 59, 0, 0, time.UTC),
			Devices:     []sources.GraphDevice{healthyThermalSensor("thermal:0", 41, 95)},
		}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	cell := cellFor(t, snapshot, "substrate/SB11", "thermal-sensor", "linux")
	if cell.Status != spacedoc.StatusNow {
		t.Errorf("a joined and grading sensor left its cell at %q, want NOW", cell.Status)
	}
	if cell.Trust != internalcondition.TrustValid {
		t.Errorf("a fully measured cell carries trust %s, want VALID", cell.Trust)
	}
	if !strings.Contains(cell.StatusSource, "live device-graph join") {
		t.Errorf("status source is %q; it does not name the live join", cell.StatusSource)
	}
	if !cell.Graded {
		t.Errorf("a trusted reading against a gradeable bar was not graded: %s", cell.UngradedReason)
	}
	if cell.Band != internalcondition.BandInBand {
		t.Errorf("a fully measured cell graded %s, want IN_BAND", cell.Band)
	}
	if cell.FaultUnit != "sensors at or above their critical trip point" || !cell.FaultCounted || cell.FaultCount != 0 {
		t.Errorf("the cell graded %v %q (counted=%v); it must grade the bar's own unit", cell.FaultCount, cell.FaultUnit, cell.FaultCounted)
	}
	if !cell.SeverityKnown || cell.Severity != 0 {
		t.Errorf("an in-band cell reports severity %d (known=%v), want an ordered 0", cell.Severity, cell.SeverityKnown)
	}
	if !cell.ObservedAt.Equal(time.Date(2026, 8, 20, 11, 59, 0, 0, time.UTC)) {
		t.Errorf("the cell reports observed_at %s; it should carry the graph's collection time, not the read time", cell.ObservedAt)
	}
}

// TestSourceAtItsDeadlineIsUnavailableNotAHang pins the per-source deadline.
func TestSourceAtItsDeadlineIsUnavailableNotAHang(t *testing.T) {
	service := newService(t,
		stubGraph{delay: 250 * time.Millisecond},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	service.Timeout = 20 * time.Millisecond

	started := time.Now()
	snapshot := service.Snapshot(context.Background())
	elapsed := time.Since(started)
	if elapsed > 200*time.Millisecond {
		t.Fatalf("the read took %s; the per-source deadline did not bound it", elapsed)
	}

	var graphSource SourceState
	for _, source := range snapshot.Sources {
		if source.ID == sources.DeviceGraphSourceID {
			graphSource = source
		}
	}
	if graphSource.Available {
		t.Fatal("a source that exceeded its deadline was reported available")
	}
	if !strings.Contains(graphSource.Reason, "deadline exceeded") {
		t.Errorf("the deadline failure reads %q; it does not name the deadline", graphSource.Reason)
	}
	for _, finding := range snapshot.Findings {
		if finding.Stage == focus.StageSubstrate {
			t.Errorf("a source at its deadline produced a plant-side finding: %s", finding.Title)
		}
	}
}

func TestSourceReturningAnErrorNamesItVerbatim(t *testing.T) {
	service := newService(t,
		stubGraph{err: fmt.Errorf("read %s: %w", "device graph", errors.New("unexpected EOF"))},
		stubGrid{grid: liveGrid(t)},
		stubChecks{err: errors.New("vrooli-autoheal is not running")})
	snapshot := service.Snapshot(context.Background())

	reasons := map[string]string{}
	for _, source := range snapshot.Sources {
		reasons[source.ID] = source.Reason
	}
	if !strings.Contains(reasons[sources.DeviceGraphSourceID], "unexpected EOF") {
		t.Errorf("the device-graph failure reads %q; the underlying error is lost", reasons[sources.DeviceGraphSourceID])
	}
	if !strings.Contains(reasons[sources.CheckPlatformsSourceID], "vrooli-autoheal is not running") {
		t.Errorf("the check-platform failure reads %q; the underlying error is lost", reasons[sources.CheckPlatformsSourceID])
	}
}

// TestCascadeRankingOrder walks the whole ladder and asserts the ranking is
// monotonically non-decreasing in cascade stage, and that every ranked finding
// states the stage it applied.
func TestCascadeRankingOrder(t *testing.T) {
	service := newService(t,
		stubGraph{err: errors.New("connection refused")},
		stubGrid{err: errors.New("the capability grid is unreadable")},
		stubChecks{err: errors.New("vrooli-autoheal is not running")})
	snapshot := service.Snapshot(context.Background())
	if len(snapshot.Findings) < 2 {
		t.Fatalf("the ladder produced %d findings; the ordering cannot be observed", len(snapshot.Findings))
	}
	previous := snapshot.Findings[0].Stage
	for index, finding := range snapshot.Findings {
		if finding.Stage < previous {
			t.Fatalf("finding %d is stage %v after stage %v; the cascade order is violated", index, finding.Stage, previous)
		}
		previous = finding.Stage
		if finding.Rank != index+1 {
			t.Errorf("finding %d reports rank %d", index, finding.Rank)
		}
		if finding.RankExplanation == "" {
			t.Errorf("finding %q states no cascade stage", finding.ID)
		}
	}
	if snapshot.Findings[0].Stage != focus.StageIntegrity {
		t.Errorf("with three unreachable sources the top finding is stage %v, want sensor-channel integrity", snapshot.Findings[0].Stage)
	}
}

// TestUnconfiguredSourcesStillProduceAFullGrid guards the shape a caller sees
// when nothing is wired: the ladder is still the complete cell set, every cell
// keeps its authored status, and the blindness is reported as findings.
func TestUnconfiguredSourcesStillProduceAFullGrid(t *testing.T) {
	service := newService(t, nil, nil, nil)
	snapshot := service.Snapshot(context.Background())

	expected := 0
	for _, join := range SubstrateJoins {
		expected += len(join.Classes) * len(HostOSes)
	}
	expected += len(CapabilityJoins) * len(HostOSes)
	if len(snapshot.Cells) != expected {
		t.Fatalf("the ladder produced %d cells, want %d", len(snapshot.Cells), expected)
	}
	for _, cell := range snapshot.Cells {
		if cell.Status == spacedoc.StatusMissing {
			t.Errorf("%s reports MISSING with no source configured", cell.Key)
		}
	}
	if len(snapshot.Sources) != 3 {
		t.Fatalf("the ladder reported %d sources, want 3", len(snapshot.Sources))
	}
	for _, source := range snapshot.Sources {
		if source.Available {
			t.Errorf("source %s reported available with nothing wired", source.ID)
		}
		if source.Reason == "" {
			t.Errorf("source %s is unavailable with no reason", source.ID)
		}
	}
}

func TestCapabilityRowsExposePortableResolutionWithoutSyntheticDevices(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-host-pressure"}}})
	snapshot := service.Snapshot(context.Background())

	for _, join := range CapabilityJoins {
		for _, hostOS := range HostOSes {
			cell := cellFor(t, snapshot, join.CellRef, join.DeviceClass, hostOS)
			if cell.Capability != join.Capability {
				t.Errorf("%s/%s joined capability %q, want %q", join.CellRef, hostOS, cell.Capability, join.Capability)
			}
			if cell.DeviceCount != 0 {
				t.Errorf("%s/%s fabricated %d synthetic devices", join.CellRef, hostOS, cell.DeviceCount)
			}
			if hostOS == service.hostOS() {
				if cell.Status != spacedoc.StatusNow || cell.Trust != internalcondition.TrustValid {
					t.Errorf("%s/%s current-platform row has status=%s trust=%s, want NOW/VALID", join.CellRef, hostOS, cell.Status, cell.Trust)
				}
				continue
			}
			if cell.Observation != ObservationUnread {
				t.Errorf("%s/%s is %s, want unread because the host was not sampled", join.CellRef, hostOS, cell.Observation)
			}
		}
	}
	forkMac := cellFor(t, snapshot, "substrate/SB16", "host-pressure-fork-rate", "macos")
	forkWindows := cellFor(t, snapshot, "substrate/SB16", "host-pressure-fork-rate", "windows")
	if forkMac.Observation != ObservationUnread || forkWindows.Observation != ObservationUnread {
		t.Fatalf("fork-rate portability rows must remain unread off Linux: macos=%s windows=%s", forkMac.Observation, forkWindows.Observation)
	}
}

// TestSensorAtItsCriticalTripIsATrustedSubstrateFinding is what keeps the
// HOST_SUBSTRATE cascade stage reachable: a fully readable population whose
// authored quantity is out of band is a real plant fault, and it is the ONLY
// thing that may raise plant-side work here.
func TestSensorAtItsCriticalTripIsATrustedSubstrateFinding(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{
			healthyThermalSensor("thermal:0", 96, 95),
			healthyThermalSensor("thermal:1", 40, 95),
		}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	cell := cellFor(t, snapshot, "substrate/SB11", "thermal-sensor", "linux")
	if cell.Trust != internalcondition.TrustValid {
		t.Fatalf("a fully readable population produced trust %s, want VALID", cell.Trust)
	}
	if !cell.Graded {
		t.Fatalf("a trusted, fully readable population was not graded: %s", cell.UngradedReason)
	}
	if cell.FaultCount != 1 {
		t.Errorf("the cell counted %v sensors at their critical trip, want 1", cell.FaultCount)
	}
	if cell.Band != internalcondition.BandOutOfBand {
		t.Fatalf("a sensor at its critical trip graded %s, want OUT_OF_BAND", cell.Band)
	}
	if !cell.SeverityKnown || cell.Severity != 2 {
		t.Errorf("an out-of-band cell reports severity %d (known=%v), want an ordered 2", cell.Severity, cell.SeverityKnown)
	}

	found := false
	for _, finding := range snapshot.Findings {
		if finding.Stage == focus.StageSubstrate && finding.CellRef == "substrate/SB11" {
			found = true
			if !finding.TrustValid {
				t.Error("the substrate finding is marked trust-invalid")
			}
			if finding.RankExplanation == "" {
				t.Error("the substrate finding states no cascade stage")
			}
		}
	}
	if !found {
		t.Fatal("an out-of-band trusted substrate cell produced no host-substrate finding; the cascade stage is unreachable")
	}
}

// TestPartiallyReadablePopulationIsNotGraded pins the denominator rule: a
// fault count over a population that could not be fully read understates the
// very fault it exists to detect, so it is refused rather than reported.
func TestPartiallyReadablePopulationIsNotGraded(t *testing.T) {
	blind := measuredDevice("thermal:1", "thermal-sensor", map[Rung]Observation{
		RungIdentity:  ObservationMeasured,
		RungTelemetry: ObservationUnavailable,
	})
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{
			healthyThermalSensor("thermal:0", 40, 95), blind,
		}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	cell := cellFor(t, snapshot, "substrate/SB11", "thermal-sensor", "linux")
	if cell.Graded {
		t.Fatalf("a partially readable population was graded %v", cell.FaultCount)
	}
	if cell.Band != internalcondition.BandNotEvaluated {
		t.Errorf("a partially readable population graded %s, want NOT_EVALUATED", cell.Band)
	}
	if cell.SeverityKnown {
		t.Error("an ungraded cell reported a severity; defaulting it would read as OK")
	}
	if !strings.Contains(cell.UngradedReason, "understate") {
		t.Errorf("the refusal reads %q; it does not say why a partial count is refused", cell.UngradedReason)
	}
	for _, finding := range snapshot.Findings {
		if finding.Stage == focus.StageSubstrate && finding.CellRef == "substrate/SB11" {
			t.Error("an ungraded cell produced a plant-side substrate finding")
		}
	}
}

// TestUnitMismatchIsRefusedRatherThanSubstituted is the SB10 case. The
// operator authored the bar in "pre-fail attributes below threshold" and the
// shipped sensor publishes no such quantity. Substituting a nearby SMART
// counter would fire on a quantity nobody authorised.
func TestUnitMismatchIsRefusedRatherThanSubstituted(t *testing.T) {
	disk := fullyMeasured("block:nvme0n1", "block-device")
	disk.Readings = map[string]float64{"smart_reallocated_sectors": 4, "smart_media_errors": 2}
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{disk}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	cell := cellFor(t, snapshot, "substrate/SB10", "block-device", "linux")
	if cell.Graded {
		t.Fatal("a bar was graded against a quantity the sensor does not publish")
	}
	if cell.Trust != internalcondition.TrustUnitMismatch {
		t.Errorf("trust = %s, want UNIT_MISMATCH", cell.Trust)
	}
	if cell.SeverityKnown {
		t.Error("a unit-mismatched cell reported a severity")
	}
	if !strings.Contains(cell.UngradedReason, "pre-fail") {
		t.Errorf("the refusal reads %q; it does not name the authored unit", cell.UngradedReason)
	}
	for _, finding := range snapshot.Findings {
		if finding.Stage == focus.StageSubstrate && finding.CellRef == "substrate/SB10" {
			t.Error("a unit mismatch produced a plant-side substrate finding; it is instrument work")
		}
	}
}

// TestBlockedRungInheritsItsCauseRatherThanItsOwnGrade pins that a
// dependency-suppressed rung is classified by WHY the ladder beneath it went
// blind, not by the fact that it was suppressed.
func TestBlockedRungInheritsItsCauseRatherThanItsOwnGrade(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		identity  Observation
		wantTrust internalcondition.TrustVerdict
	}{
		{"host refused the identity read", ObservationUnmeasurable, internalcondition.TrustUntrusted},
		{"no identity mechanism on this host", ObservationUnavailable, internalcondition.TrustValid},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			service := newService(t,
				stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{
					measuredDevice("thermal:0", "thermal-sensor", map[Rung]Observation{
						RungIdentity:  testCase.identity,
						RungTelemetry: ObservationMeasured,
					}),
				}}},
				stubGrid{grid: liveGrid(t)},
				stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
			snapshot := service.Snapshot(context.Background())

			cell := cellFor(t, snapshot, "substrate/SB11", "thermal-sensor", "linux")
			if cell.Observation != ObservationBlocked {
				t.Fatalf("the telemetry rung reports %s above a blind identity rung, want blocked", cell.Observation)
			}
			if cell.BlockedBy != RungIdentity {
				t.Fatalf("the cell reports blocked by %q, want identity", cell.BlockedBy)
			}
			if cell.Trust != testCase.wantTrust {
				t.Errorf("a rung blocked by %s carries trust %s, want %s", testCase.identity, cell.Trust, testCase.wantTrust)
			}
		})
	}
}

// TestBlindnessAgeDistinguishesUndatedFromZeroDay is the board's differentiator.
// gap_open_days is 0 both for a gap opened today and for a gap nobody ever
// dated, so an undated gap must be reported as undated rather than as a
// zero-day gap — it is the one cell nobody can put a clock on.
func TestBlindnessAgeDistinguishesUndatedFromZeroDay(t *testing.T) {
	service := newService(t, nil, nil, nil)
	snapshot := service.Snapshot(context.Background())
	for _, cell := range snapshot.Cells {
		if cell.GapOpenedOn == "" && cell.GapDated {
			t.Errorf("%s reports a dated gap with no date", cell.Key)
		}
		if cell.GapOpenedOn != "" && !cell.GapDated {
			t.Errorf("%s carries gap date %q but reports undated", cell.Key, cell.GapOpenedOn)
		}
		if !cell.GapDated && cell.GapOpenDays != 0 {
			t.Errorf("%s reports %d days for an undated gap", cell.Key, cell.GapOpenDays)
		}
	}
}

// TestDenominatorConfidenceTravelsWithTheCells pins that no ratio over these
// cells can be reported without the confidence in their denominator.
func TestDenominatorConfidenceTravelsWithTheCells(t *testing.T) {
	snapshot := newService(t, nil, nil, nil).Snapshot(context.Background())
	if !snapshot.Confidence.Available {
		t.Fatalf("the substrate space confidence was not read: %s", snapshot.Confidence.Reason)
	}
	if snapshot.Confidence.Level != "PARTIAL" {
		t.Errorf("confidence level = %q; the substrate space declares PARTIAL", snapshot.Confidence.Level)
	}
	if snapshot.Confidence.Rationale == "" {
		t.Error("the confidence carries no rationale; a level without one claims nothing")
	}
}

// TestUnreadSpaceHasNoConfidenceRatherThanADefaultedOne — claiming
// AUTHORITATIVE about a document nobody opened is the worst available answer.
func TestUnreadSpaceHasNoConfidenceRatherThanADefaultedOne(t *testing.T) {
	service := &Service{HostOS: "linux", Now: func() time.Time { return time.Now() }}
	snapshot := service.Snapshot(context.Background())
	if snapshot.Confidence.Available {
		t.Fatal("an unread space reported a confidence level")
	}
	if snapshot.Confidence.Level != "" {
		t.Errorf("an unread space reported level %q", snapshot.Confidence.Level)
	}
	if snapshot.Confidence.Reason == "" {
		t.Error("an unread space gives no reason")
	}
}

// TestDeviceInventoryCarriesBothGrades pins that the owner's verbatim grade and
// the ladder's dependency verdict are both reported. They answer different
// questions and the UI renders them differently.
func TestDeviceInventoryCarriesBothGrades(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{
			measuredDevice("thermal:0", "thermal-sensor", map[Rung]Observation{
				RungIdentity:  ObservationUnavailable,
				RungTelemetry: ObservationMeasured,
			}),
		}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{})
	snapshot := service.Snapshot(context.Background())

	if len(snapshot.Devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(snapshot.Devices))
	}
	device := snapshot.Devices[0]
	if len(device.Rungs) != len(Rungs) {
		t.Fatalf("device reports %d rungs, want the full ladder of %d", len(device.Rungs), len(Rungs))
	}
	byRung := map[Rung]DeviceRung{}
	for _, rung := range device.Rungs {
		byRung[rung.Rung] = rung
	}
	telemetry := byRung[RungTelemetry]
	if telemetry.Observation != ObservationMeasured {
		t.Errorf("the owner graded telemetry %s; the verbatim grade was rewritten to %s", ObservationMeasured, telemetry.Observation)
	}
	if telemetry.LadderObservation != ObservationBlocked {
		t.Errorf("telemetry's ladder verdict is %s above a blind identity rung, want blocked", telemetry.LadderObservation)
	}
	if telemetry.BlockedBy != RungIdentity {
		t.Errorf("telemetry reports blocked by %q, want identity", telemetry.BlockedBy)
	}
}

// TestDeviceInventoryIsEmptyOnlyWhenTheSourceWasRead guards the claim an empty
// inventory would make: that the host has no hardware.
func TestDeviceInventoryIsEmptyOnlyWhenTheSourceWasRead(t *testing.T) {
	snapshot := newService(t, stubGraph{err: errors.New("connection refused")}, nil, nil).Snapshot(context.Background())
	if len(snapshot.Devices) != 0 {
		t.Fatal("an unread device graph produced devices")
	}
	for _, source := range snapshot.Sources {
		if source.ID != sources.DeviceGraphSourceID {
			continue
		}
		if source.Trust != internalcondition.TrustUnavailable {
			t.Errorf("an unread source carries trust %s, want UNAVAILABLE", source.Trust)
		}
		if source.Reason == "" {
			t.Error("an unread source carries no reason")
		}
	}
}
