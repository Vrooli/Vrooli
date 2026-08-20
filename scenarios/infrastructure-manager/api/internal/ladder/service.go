package ladder

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/vrooli/internal/deployability"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
	internalcoverage "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/coverage"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/focus"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/sources"
)

// SourceState reports one source's availability exactly as the fan-out saw it.
// A source that could not be read is reported, never omitted: a source that
// disappears when it breaks is the failure the trust axis exists to catch.
type SourceState struct {
	ID        string
	Available bool
	Reason    string
	CheckedAt time.Time
}

// Snapshot is one complete ladder readout.
type Snapshot struct {
	Cells      []Cell
	Sources    []SourceState
	Findings   []focus.RankedFinding
	HostOS     string
	ComputedAt time.Time
	// CoverageAvailable is false when the substrate space document itself
	// could not be read, in which case there are no authored statuses to keep
	// and the readout says so rather than inventing a grid.
	CoverageAvailable bool
	CoverageReason    string
}

// DeviceGraphSource is the ladder's view of the device graph reader.
type DeviceGraphSource interface {
	ReadGraph(ctx context.Context) (sources.DeviceGraph, error)
}

// PortabilitySource is the ladder's view of the capability grid reader.
type PortabilitySource interface {
	ReadGrid(ctx context.Context) (portability.Grid, error)
}

// CheckPlatformSource is the ladder's view of the autoheal check registry's
// platform declarations.
type CheckPlatformSource interface {
	ReadPlatforms(ctx context.Context) ([]sources.CheckPlatforms, error)
}

// Service computes the ladder readout. It reads and grades and does nothing
// else: there is no actuation seam here and there must never be one.
type Service struct {
	Coverage    *internalcoverage.Service
	DeviceGraph DeviceGraphSource
	Portability PortabilitySource
	Checks      CheckPlatformSource
	// HostOS is the host this instrument is running on. Its cells are the ones
	// a live device-graph join can refine; the other host OSes are reasoned
	// about from declarations only.
	HostOS  string
	Timeout time.Duration
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s *Service) timeout() time.Duration {
	if s.Timeout <= 0 {
		return sources.DefaultTimeout
	}
	return s.Timeout
}

func (s *Service) hostOS() string {
	if strings.TrimSpace(s.HostOS) == "" {
		return "linux"
	}
	return s.HostOS
}

// Snapshot reads every source under its own deadline, joins the results onto
// the authored substrate cells, and ranks the findings in cascade order.
func (s *Service) Snapshot(ctx context.Context) Snapshot {
	now := s.now()
	snapshot := Snapshot{HostOS: s.hostOS(), ComputedAt: now}

	graphResult := sources.ReadTyped(ctx, sources.DeviceGraphSourceID, s.readGraph, s.timeout())
	gridResult := sources.ReadTyped(ctx, sources.PortabilitySourceID, s.readGrid, s.timeout())
	checkResult := sources.ReadTyped(ctx, sources.CheckPlatformsSourceID, s.readPlatforms, s.timeout())
	for _, result := range []struct {
		id        string
		available bool
		reason    string
		checkedAt time.Time
	}{
		{graphResult.ID, graphResult.Available, graphResult.Reason, graphResult.CheckedAt},
		{gridResult.ID, gridResult.Available, gridResult.Reason, gridResult.CheckedAt},
		{checkResult.ID, checkResult.Available, checkResult.Reason, checkResult.CheckedAt},
	} {
		snapshot.Sources = append(snapshot.Sources, SourceState{ID: result.id, Available: result.available, Reason: result.reason, CheckedAt: result.checkedAt})
	}

	definition, bars, coverageErr := s.substrate(ctx)
	if coverageErr != nil {
		snapshot.CoverageReason = coverageErr.Error()
	} else {
		snapshot.CoverageAvailable = true
	}

	snapshot.Cells = s.buildCells(definition, bars, graphResult, gridResult, checkResult, now)
	snapshot.Findings = rank(snapshot)
	return snapshot
}

func (s *Service) readGraph(ctx context.Context) (sources.DeviceGraph, error) {
	if s.DeviceGraph == nil {
		return sources.DeviceGraph{}, fmt.Errorf("the device-graph source is not configured")
	}
	return s.DeviceGraph.ReadGraph(ctx)
}

func (s *Service) readGrid(ctx context.Context) (portability.Grid, error) {
	if s.Portability == nil {
		return portability.Grid{}, fmt.Errorf("the capability-grid source is not configured")
	}
	return s.Portability.ReadGrid(ctx)
}

func (s *Service) readPlatforms(ctx context.Context) ([]sources.CheckPlatforms, error) {
	if s.Checks == nil {
		return nil, fmt.Errorf("the check-platform source is not configured")
	}
	return s.Checks.ReadPlatforms(ctx)
}

func (s *Service) substrate(ctx context.Context) (*spacedoc.SpaceDefinition, []internalcoverage.Bar, error) {
	if s.Coverage == nil {
		return nil, nil, fmt.Errorf("the coverage source is not configured")
	}
	snapshot, err := s.Coverage.Snapshot(ctx, []spacedoc.Projection{spacedoc.ProjectionSubstrate})
	if err != nil {
		return nil, nil, err
	}
	item, ok := snapshot.Projections[spacedoc.ProjectionSubstrate]
	if !ok {
		return nil, nil, fmt.Errorf("the substrate projection is unavailable")
	}
	return item.Definition, item.Bars, nil
}

// buildCells produces the full (device class, rung, host OS) grid.
func (s *Service) buildCells(
	definition *spacedoc.SpaceDefinition,
	bars []internalcoverage.Bar,
	graph sources.TypedResult[sources.DeviceGraph],
	grid sources.TypedResult[portability.Grid],
	checks sources.TypedResult[[]sources.CheckPlatforms],
	now time.Time,
) []Cell {
	byBar := make(map[string]internalcoverage.Bar, len(bars))
	for _, bar := range bars {
		byBar[bar.CellRef] = bar
	}
	devices := groupDevices(graph)
	cells := make([]Cell, 0, len(SubstrateJoins)*len(HostOSes)*2)

	for _, join := range SubstrateJoins {
		status, question, authored := authoredStatus(definition, join.CellRef)
		if question == "" {
			question = join.Question
		}
		for _, class := range join.Classes {
			for _, hostOS := range HostOSes {
				cell := Cell{
					Key:        CellKey{DeviceClass: class, Rung: join.Rung, HostOS: hostOS},
					CellRef:    join.CellRef,
					Question:   question,
					Capability: join.Capability,
					Status:     status,
					// Silence keeps the authored status. This assignment is
					// the rule: nothing below ever downgrades a cell to
					// MISSING because a source did not answer.
					StatusSource: "space document",
					Observation:  ObservationUnread,
					Reason:       "no live join produced a grade for this cell",
					Trust:        internalcondition.TrustUntrusted,
					ObservedAt:   now,
				}
				if !authored {
					cell.Status = spacedoc.CellStatus("")
					cell.StatusSource = "unauthored: the substrate space document declares no cell " + join.CellRef
				}
				s.applyDeviceJoin(&cell, join, class, hostOS, devices, graph)
				applyCapabilityJoin(&cell, grid, checks, hostOS)
				applyBar(&cell, byBar)
				cells = append(cells, cell)
			}
		}
	}
	SortCells(cells)
	return cells
}

// applyDeviceJoin refines a cell from the live device graph. It only refines
// the host OS the instrument is actually running on: the device graph is a
// reading of THIS host, and using it to grade another platform's coverage
// would be asserting a fact about a machine nobody observed.
func (s *Service) applyDeviceJoin(cell *Cell, join SubstrateJoin, class, hostOS string, devices map[string][]sources.GraphDevice, graph sources.TypedResult[sources.DeviceGraph]) {
	if hostOS != s.hostOS() {
		return
	}
	if !graph.Available {
		// An unreachable source is UNAVAILABLE, never MISSING. The cell keeps
		// its authored status and the outage is reported as the instrument's
		// finding, not the plant's.
		cell.Trust = internalcondition.TrustUnavailable
		cell.UnavailableReason = graph.Reason
		cell.Observation = ObservationUnread
		cell.Reason = "the device-graph source could not be read: " + graph.Reason
		return
	}

	matching := devices[class]
	cell.DeviceCount = len(matching)
	if len(matching) == 0 {
		// A live read that found no device of this class is a real fact about
		// the host, not a gap in the instrument.
		cell.Observation = ObservationNotApplicable
		cell.Reason = fmt.Sprintf("the device graph enumerated no %s on this host", class)
		cell.Trust = internalcondition.TrustValid
		cell.StatusSource = "live device-graph join"
		cell.ObservedAt = graph.Value.CollectedAt
		return
	}

	worst := RungReading{Rung: join.Rung, Observation: ObservationMeasured}
	worstCause := ObservationMeasured
	blind := 0
	for _, device := range matching {
		ordered := ApplyDependency(rungReadings(device))
		reading := readingFor(ordered, join.Rung)
		if !reading.Observation.Supports() {
			blind++
		}
		if observationRank(reading.Observation) > observationRank(worst.Observation) {
			worst, worstCause = reading, causeOf(ordered, reading)
		}
	}
	cell.BlindDevices = blind
	cell.Observation = worst.Observation
	cell.Reason = worst.Reason
	cell.Mechanism = worst.Mechanism
	cell.Remediation = worst.Remediation
	cell.BlockedBy = worst.BlockedBy
	cell.StatusSource = "live device-graph join"
	cell.ObservedAt = graph.Value.CollectedAt
	cell.Trust = trustFor(worstCause)
	if cell.Reason == "" {
		if blind == 0 {
			cell.Reason = fmt.Sprintf("all %d %s graded %s at the %s rung", cell.DeviceCount, class, worst.Observation, join.Rung)
		} else {
			cell.Reason = fmt.Sprintf("%d of %d %s could not be graded at the %s rung", blind, cell.DeviceCount, class, join.Rung)
		}
	}
	if blind == 0 && cell.Trust == internalcondition.TrustValid {
		cell.Status = spacedoc.StatusNow
		cell.StatusSource = "live device-graph join: the sensor is joined and grading"
	}
}

// causeOf resolves what actually blinded a rung. A blocked rung says nothing
// about itself — its character is the character of the rung beneath it that
// went blind — so trust must be decided from the cause rather than from the
// blocked grade, or every dependency-suppressed rung would be classified
// identically regardless of why.
func causeOf(ordered []RungReading, reading RungReading) Observation {
	if reading.Observation != ObservationBlocked {
		return reading.Observation
	}
	for _, candidate := range ordered {
		if candidate.Rung == reading.BlockedBy {
			return candidate.Observation
		}
	}
	return ObservationUnread
}

// trustFor decides whether a rung grade is evidence or instrument fault.
//
// The split is the one the trust model turns on, and the two blind states are
// NOT the same fact:
//
//   - `unmeasurable` means the host refused a read that should have worked.
//     The reading exists and cannot be believed, so it is UNTRUSTED and never
//     produces plant-side work.
//   - `unavailable` means the mechanism is not present on this host at all —
//     no SMART reader installed, no EDAC controller registered. That is a
//     believable measurement of a real host condition, so it is VALID and it
//     grades against the bar like any other reading. Marking it untrusted
//     would make a fixable commissioning gap permanently invisible, because
//     an untrusted reading is excluded from every aggregate.
//   - `unread` means no source graded the rung at all, which claims nothing.
func trustFor(cause Observation) internalcondition.TrustVerdict {
	switch cause {
	case ObservationMeasured, ObservationNotApplicable, ObservationUnavailable:
		return internalcondition.TrustValid
	default:
		return internalcondition.TrustUntrusted
	}
}

// applyCapabilityJoin joins the capability grid and the check platform
// declarations onto the cell's host OS dimension.
func applyCapabilityJoin(cell *Cell, grid sources.TypedResult[portability.Grid], checks sources.TypedResult[[]sources.CheckPlatforms], hostOS string) {
	if !grid.Available {
		cell.CapabilityReason = "the capability grid could not be read: " + grid.Reason
		if cell.Trust == internalcondition.TrustUntrusted && cell.UnavailableReason == "" {
			cell.UnavailableReason = grid.Reason
		}
		return
	}
	entry, ok := grid.Value.Capability(cell.Capability)
	if !ok {
		cell.CapabilityReason = fmt.Sprintf("the capability vocabulary names no capability %q, so this cell's host OS dimension is ungraded", cell.Capability)
		return
	}
	platform, ok := entry.Platform(deployability.HostOS(hostOS))
	if !ok {
		cell.CapabilityReason = fmt.Sprintf("the capability grid has no %s row for %s", hostOS, entry.Capability)
		return
	}
	cell.CapabilityStatus = string(platform.Status)
	cell.CapabilityReason = platform.Reason
	if !checks.Available {
		return
	}
	// A check that does not apply on this host OS is a declared absence, read
	// live from the owner. It is the one path allowed to say a cell is not
	// served here, and only because a declaration was actually read.
	if !anyCheckApplies(checks.Value, hostOS) {
		cell.CapabilityReason = fmt.Sprintf("no registered check declares %s, so this cell has no sensor there", hostOS)
	}
}

func anyCheckApplies(checks []sources.CheckPlatforms, hostOS string) bool {
	for _, check := range checks {
		if check.AppliesTo(hostOS) {
			return true
		}
	}
	return false
}

// applyBar resolves the cell's setpoint bar. Every ladder cell either grades
// against a bar or is reported UNGRADED with a reason — never silently
// ungraded, which reads identically to in-band.
func applyBar(cell *Cell, byBar map[string]internalcoverage.Bar) {
	bar, ok := byBar[cell.CellRef]
	if !ok {
		cell.Graded = false
		cell.UngradedReason = "no setpoint bar authors a target for " + cell.CellRef
		cell.Band = internalcondition.BandNeedsBaseline
		return
	}
	cell.BarID = bar.ID
	cell.Provisional = bar.Provisional
	if !bar.Gradeable {
		cell.Graded = false
		cell.UngradedReason = bar.NotGradeableReason
		if cell.UngradedReason == "" {
			cell.UngradedReason = "bar " + bar.ID + " authors no threshold"
		}
		cell.Band = internalcondition.BandNotGradeable
		return
	}
	if cell.Trust != internalcondition.TrustValid {
		cell.Graded = false
		cell.UngradedReason = fmt.Sprintf("the reading carries trust %s, so bar %s is not evaluated", cell.Trust, bar.ID)
		cell.Band = internalcondition.BandNotEvaluated
		return
	}
	cell.Graded = true
	// The graded quantity is the count of contributing devices that could not
	// be graded at this rung. Zero blind devices is the only in-band reading:
	// a rung the instrument cannot read is a gap whatever the hardware is
	// doing.
	cell.Band = internalcondition.EvaluateBand(float64(cell.BlindDevices), cell.Trust, internalcondition.Band{
		Min: bar.Min, Max: bar.Max, SustainSatisfied: true, Unit: bar.Unit, Provisional: bar.Provisional,
	})
}

func rungReadings(device sources.GraphDevice) map[Rung]RungReading {
	out := make(map[Rung]RungReading, len(device.Rungs))
	for token, state := range device.Rungs {
		rung, err := ParseRung(token)
		if err != nil {
			continue
		}
		out[rung] = RungReading{
			Rung:        rung,
			Observation: parseObservation(state.State),
			Reason:      state.Reason,
			Mechanism:   state.Mechanism,
			Remediation: state.Remediation,
		}
	}
	return out
}

// parseObservation maps the owner's state token onto the ladder vocabulary. An
// unrecognised token becomes unmeasurable with the token named, never
// measured: a state this instrument does not understand is not a reading it
// may count as healthy.
func parseObservation(token string) Observation {
	switch Observation(token) {
	case ObservationMeasured:
		return ObservationMeasured
	case ObservationUnmeasurable:
		return ObservationUnmeasurable
	case ObservationUnavailable:
		return ObservationUnavailable
	case ObservationNotApplicable:
		return ObservationNotApplicable
	default:
		return ObservationUnmeasurable
	}
}

func readingFor(ordered []RungReading, rung Rung) RungReading {
	for _, reading := range ordered {
		if reading.Rung == rung {
			return reading
		}
	}
	return RungReading{Rung: rung, Observation: ObservationUnread, Reason: "no source graded this rung"}
}

// observationRank orders observations by how much they obstruct a claim, so
// a class takes its worst member. Measured is the only clean outcome; blocked
// ranks highest because it means the ladder beneath the reading is broken.
func observationRank(observation Observation) int {
	switch observation {
	case ObservationMeasured:
		return 0
	case ObservationNotApplicable:
		return 1
	case ObservationUnread:
		return 2
	case ObservationUnavailable:
		return 3
	case ObservationUnmeasurable:
		return 4
	case ObservationBlocked:
		return 5
	default:
		return 6
	}
}

func groupDevices(graph sources.TypedResult[sources.DeviceGraph]) map[string][]sources.GraphDevice {
	out := make(map[string][]sources.GraphDevice)
	if !graph.Available {
		return out
	}
	for _, device := range graph.Value.Devices {
		out[device.Class] = append(out[device.Class], device)
	}
	for class := range out {
		sort.Slice(out[class], func(i, j int) bool { return out[class][i].ID < out[class][j].ID })
	}
	return out
}
