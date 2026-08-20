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
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{fullyMeasured("pci:0000:01:00.0", "thermal-sensor")}}},
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
			Devices:     []sources.GraphDevice{fullyMeasured("thermal:0", "thermal-sensor")},
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

// TestAbsentMechanismIsATrustedSubstrateFinding is the other half of
// TestUnreadableDeviceIsUntrustedNotMissing, and it is what keeps the
// host-substrate cascade stage reachable at all.
//
// A host with no SMART reader installed is not an unreadable sensor — it is a
// believable measurement of a real, fixable host condition. Classifying it
// UNTRUSTED alongside "the host refused the read" would exclude it from every
// aggregate and make a commissioning gap permanently invisible.
func TestAbsentMechanismIsATrustedSubstrateFinding(t *testing.T) {
	service := newService(t,
		stubGraph{graph: sources.DeviceGraph{Devices: []sources.GraphDevice{
			measuredDevice("thermal:0", "thermal-sensor", map[Rung]Observation{
				RungIdentity:  ObservationMeasured,
				RungTelemetry: ObservationUnavailable,
			}),
		}}},
		stubGrid{grid: liveGrid(t)},
		stubChecks{checks: []sources.CheckPlatforms{{CheckID: "system-gpu", Platforms: []string{"linux"}}}})
	snapshot := service.Snapshot(context.Background())

	cell := cellFor(t, snapshot, "substrate/SB11", "thermal-sensor", "linux")
	if cell.Trust != internalcondition.TrustValid {
		t.Fatalf("an absent mechanism produced trust %s, want VALID", cell.Trust)
	}
	if !cell.Graded {
		t.Fatalf("a trusted reading was not graded: %s", cell.UngradedReason)
	}
	if cell.Band != internalcondition.BandOutOfBand {
		t.Fatalf("a cell with a blind device graded %s, want OUT_OF_BAND", cell.Band)
	}
	if cell.Status == spacedoc.StatusNow {
		t.Error("a cell with a blind device was promoted to NOW")
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
