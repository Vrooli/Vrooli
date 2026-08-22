package ladder

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/spacedoc"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/sources"
)

// HostOSes is the host OS axis of the ladder grid, matching the capability
// grid's axis so the two join without translation.
var HostOSes = []string{"linux", "macos", "windows"}

// CellKey is one ladder cell: a device class, a rung, and a host OS. All three
// are needed. The same class can be fully identified on one host and invisible
// on another, and collapsing the host OS out of the key is how a Linux-only
// sensor comes to look like platform-wide coverage.
type CellKey struct {
	DeviceClass string
	Rung        Rung
	HostOS      string
}

func (k CellKey) String() string {
	return k.DeviceClass + "/" + string(k.Rung) + "/" + k.HostOS
}

// SubstrateJoin binds one authored substrate cell to the (device class, rung)
// pairs that answer it. This table is the SB9-SB13 join the substrate space
// document describes: each authored cell already names a shipped sensor, and
// what was missing was this binding, not an instrument.
//
// Do not fold entries together. SB12 covers two classes because a memory error
// is graded per controller and per DIMM, and a controller that registers no
// EDAC device is a different fact from a DIMM with no counters.
type SubstrateJoin struct {
	CellRef string
	Rung    Rung
	Classes []string
	// Capability names the vocabulary capability that serves this cell. It is
	// the join key onto the portability grid's host OS axis: the grid answers
	// "does an implementation of this capability resolve on that platform?",
	// which is precisely the question the ladder's host OS dimension asks.
	Capability string
	// Question is carried so a ladder cell can explain itself without the
	// caller re-reading the space document.
	Question string
	// FaultUnit names the quantity this cell's setpoint bar grades, spelled
	// exactly as the operator authored it. It is checked against the bar's own
	// unit before anything is graded.
	//
	// An empty FaultUnit means this join computes no fault quantity for the
	// cell. That is not a defect: two of the five device-layer bars author no
	// threshold at all, and inventing a quantity for them would grade a number
	// the operator never asked for.
	FaultUnit string
	// Fault computes the graded quantity over the contributing devices. It
	// returns ok=false with a reason when the shipped sensor publishes no
	// reading in the bar's unit — which is a unit mismatch, not a zero.
	Fault func(devices []sources.GraphDevice) (value float64, ok bool, reason string)
}

// SubstrateJoins is the authored device-layer cell set, in cell order.
var SubstrateJoins = []SubstrateJoin{
	{
		CellRef: "substrate/SB9", Rung: RungIdentity,
		Classes:    []string{"block-device", "graphics-device", "network-interface", "pci-device", "usb-device"},
		Capability: "system-monitor-device-graph",
		Question:   "Is every attached device identified by vendor, model and driver rather than a bare numeric id?",
	},
	{
		CellRef: "substrate/SB10", Rung: RungAnticipation,
		Classes:    []string{"block-device"},
		Capability: "system-monitor-storage-health",
		Question:   "Do storage devices report predictive-failure indicators before they fail?",
		FaultUnit:  "pre-fail attributes below threshold",
		Fault:      countPreFailAttributes,
	},
	{
		CellRef: "substrate/SB11", Rung: RungTelemetry,
		Classes:    []string{"thermal-sensor"},
		Capability: "system-monitor-thermal",
		Question:   "Are host temperatures readable and below their trip points?",
		FaultUnit:  "sensors at or above their critical trip point",
		Fault:      countSensorsAtCriticalTrip,
	},
	{
		CellRef: "substrate/SB12", Rung: RungTelemetry,
		Classes:    []string{"memory-controller", "memory-module"},
		Capability: "system-monitor-memory-errors",
		Question:   "Are correctable and uncorrectable memory errors counted?",
		FaultUnit:  "uncorrectable ECC errors",
		Fault:      sumUncorrectableErrors,
	},
	{
		CellRef: "substrate/SB13", Rung: RungTelemetry,
		Classes:    []string{"network-interface"},
		Capability: "system-monitor-network",
		Question:   "Do hardware network interfaces carry traffic without error or drop growth?",
	},
}

// countSensorsAtCriticalTrip counts thermal sensors whose current temperature
// has reached their critical trip point. A sensor that publishes no trip point
// is not counted as safe — it is excluded from the population and named, so
// the bar is never satisfied by sensors nobody could grade.
func countSensorsAtCriticalTrip(devices []sources.GraphDevice) (float64, bool, string) {
	faults, graded := 0.0, 0
	ungradeable := make([]string, 0)
	for _, device := range devices {
		current, hasCurrent := device.Readings["temperature_celsius"]
		critical, hasCritical := device.Readings["setpoint_critical_celsius"]
		if !hasCurrent || !hasCritical {
			ungradeable = append(ungradeable, device.ID)
			continue
		}
		graded++
		if current >= critical {
			faults++
		}
	}
	if graded == 0 {
		return 0, false, fmt.Sprintf("no thermal sensor published both a temperature and a critical trip point (%d sensor(s) seen)", len(devices))
	}
	if len(ungradeable) > 0 {
		return 0, false, fmt.Sprintf("%d of %d sensors publish no critical trip point (%v); a count over the remainder would understate the population", len(ungradeable), len(devices), ungradeable)
	}
	return faults, true, ""
}

// sumUncorrectableErrors totals the uncorrectable ECC counter across memory
// controllers and DIMMs.
func sumUncorrectableErrors(devices []sources.GraphDevice) (float64, bool, string) {
	total, graded := 0.0, 0
	silent := make([]string, 0)
	for _, device := range devices {
		value, ok := device.Readings["uncorrectable_errors_total"]
		if !ok {
			silent = append(silent, device.ID)
			continue
		}
		graded++
		total += value
	}
	if graded == 0 {
		return 0, false, fmt.Sprintf("no memory device published an uncorrectable error counter (%d device(s) seen); an absent counter is not zero errors", len(devices))
	}
	if len(silent) > 0 {
		return 0, false, fmt.Sprintf("%d of %d memory devices publish no uncorrectable error counter (%v); a total over the remainder would understate it", len(silent), len(devices), silent)
	}
	return total, true, ""
}

// countPreFailAttributes is deliberately a refusal, and the refusal is the
// finding.
//
// The operator authored this bar in the unit "pre-fail attributes below
// threshold" — the ATA SMART notion of an attribute whose normalised value has
// fallen under its manufacturer threshold. The shipped device-graph sensor
// publishes reallocated sectors, pending sectors, media errors, wear percent
// and a pass/fail health flag; it publishes no pre-fail attribute count.
//
// Substituting any of those would grade a different quantity under the
// operator's label, which is precisely the unit merge the coverage model
// forbids: the bar would then fire, or fail to fire, for reasons the operator
// never authorised. The cell is reported UNIT_MISMATCH until either the sensor
// publishes the authored quantity or the operator re-authors the bar.
func countPreFailAttributes(devices []sources.GraphDevice) (float64, bool, string) {
	return 0, false, fmt.Sprintf(
		"the device graph publishes no pre-fail attribute count for the %d storage device(s) seen; it publishes reallocated, pending and uncorrectable sector counts, media errors, wear percent and a pass/fail health flag. Grading any of those against a bar authored in pre-fail attributes would fire on a different quantity than the operator authorised",
		len(devices))
}

// Cell is one graded ladder cell.
type Cell struct {
	Key      CellKey
	CellRef  string
	Question string

	// Status is the coverage status. It starts as the status authored in the
	// space document and is refined ONLY by a live join. A cell absent from a
	// live join keeps its authored status: fabricating MISSING from silence
	// would turn an owner outage into a coverage collapse.
	Status spacedoc.CellStatus
	// StatusSource explains where Status came from — the space document, or a
	// live join that refined it.
	StatusSource string

	// Observation is the rung grade after the dependency ordering is applied.
	Observation Observation
	Reason      string
	ReasonCode  string
	Mechanism   string
	Remediation string
	BlockedBy   Rung

	// Trust is the trust verdict for this reading. An unreachable source is
	// UNAVAILABLE and an unreadable device is UNTRUSTED; neither ever becomes
	// MISSING, because MISSING is a statement about coverage and both of these
	// are statements about the instrument.
	Trust             internalcondition.TrustVerdict
	UnavailableReason string

	// DeviceCount is how many devices of this class contributed. Zero with a
	// live join means the host has none of that class; zero with no live join
	// means nothing was read, which Observation distinguishes.
	DeviceCount int
	// BlindDevices counts the contributing devices whose rung could not be
	// graded, so a partially readable class is visible as partial rather than
	// collapsing to its worst member with no denominator.
	BlindDevices int

	// BarID names the setpoint bar this cell grades against. Graded is false
	// when no bar resolves or the bar authors no threshold, and
	// UngradedReason then says which of those it was.
	BarID          string
	Graded         bool
	UngradedReason string
	Band           internalcondition.BandVerdict
	Provisional    bool

	// FaultUnit and FaultCount are the quantity actually graded, in the bar's
	// own unit. FaultCounted is false when no quantity could be computed.
	FaultUnit    string
	FaultCount   float64
	FaultCounted bool

	// Severity projects the graded verdict onto the substrate projection's
	// ordered severity — 0 OK, 1 WARNING, 2 CRITICAL — so a device-layer cell
	// is comparable with the projection's other cells. SeverityKnown is false
	// whenever the cell was not graded: an ungraded cell has no severity, and
	// defaulting it to 0 would read as OK.
	Severity      int
	SeverityKnown bool

	// Capability names the vocabulary capability serving this cell, and
	// CapabilityStatus is its resolution status on this cell's host OS,
	// joined from the portability grid.
	Capability       string
	CapabilityStatus string
	CapabilityReason string

	// GapOpenedOn and GapOpenDays are the blindness age carried from the space
	// document. GapOpenDays is 0 both for a gap opened today and for a gap
	// nobody ever dated, so GapDated is the field that tells them apart — an
	// undated gap is the one nobody can put a clock on and must not render as
	// a zero-day gap.
	GapOpenedOn string
	GapOpenDays int
	GapDated    bool

	ObservedAt time.Time
}

// Device is one graded hardware device as the ladder sees it.
type Device struct {
	ID       string
	Class    string
	ParentID string
	Vendor   string
	Model    string
	Driver   string
	SysPath  string
	// Attributes carry the owner's per-class facts, including kernel node
	// names, exactly as it recorded them.
	Attributes map[string]string
	Readings   map[string]float64
	// Rungs is the dependency-ordered ladder. Each entry carries BOTH the
	// owner's verbatim grade and this instrument's dependency verdict, because
	// they answer different questions: "what did the sensor find?" and "may a
	// claim rest on it?".
	Rungs []DeviceRung
}

// DeviceRung is one rung of one device.
type DeviceRung struct {
	Rung Rung
	// Observation is the owner's grade, verbatim. measured, unmeasurable,
	// unavailable and not_applicable are four different facts and none is ever
	// collapsed into another here.
	Observation Observation
	// LadderObservation is the grade after the ladder's dependency ordering.
	// It differs from Observation only by becoming BLOCKED when a rung beneath
	// this one is blind.
	LadderObservation Observation
	Reason            string
	Mechanism         string
	Remediation       string
	BlockedBy         Rung
}

// Confidence is the substrate space's denominator confidence. It travels with
// every ratio computed over that denominator; a ratio without it is not a
// valid instrument response.
type Confidence struct {
	Level     string
	Rationale string
	Available bool
	Reason    string
}

// SortCells orders cells deterministically: authored cell first, then class,
// then host OS. Rung is implied by the cell.
func SortCells(cells []Cell) {
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].CellRef != cells[j].CellRef {
			return cells[i].CellRef < cells[j].CellRef
		}
		if cells[i].Key.DeviceClass != cells[j].Key.DeviceClass {
			return cells[i].Key.DeviceClass < cells[j].Key.DeviceClass
		}
		return hostOSRank(cells[i].Key.HostOS) < hostOSRank(cells[j].Key.HostOS)
	})
}

func hostOSRank(hostOS string) int {
	for index, candidate := range HostOSes {
		if candidate == hostOS {
			return index
		}
	}
	return len(HostOSes)
}

// authoredStatus reports the status the space document declares for a cell.
// The device-layer cells are IN-REACH by authorship: their sensor ships and
// emits, and only the join was missing.
func authoredStatus(definition *spacedoc.SpaceDefinition, cellRef string) (spacedoc.Cell, bool) {
	if definition == nil {
		return spacedoc.Cell{}, false
	}
	id := cellRef
	if index := strings.Index(cellRef, "/"); index >= 0 {
		id = cellRef[index+1:]
	}
	for _, cell := range definition.Cells {
		if cell.ID == id {
			return cell, true
		}
	}
	return spacedoc.Cell{}, false
}

// gapAge converts an authored gap date into an age in days. An unparseable or
// absent date yields dated=false rather than a zero age, so "opened today" and
// "nobody dated this" stay distinguishable.
func gapAge(openedOn string, now time.Time) (int, bool) {
	openedOn = strings.TrimSpace(openedOn)
	if openedOn == "" {
		return 0, false
	}
	parsed, err := time.Parse("2006-01-02", openedOn)
	if err != nil {
		return 0, false
	}
	days := int(now.Sub(parsed).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return days, true
}
