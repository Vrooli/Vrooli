package ladder

import (
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/spacedoc"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
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
	},
	{
		CellRef: "substrate/SB11", Rung: RungTelemetry,
		Classes:    []string{"thermal-sensor"},
		Capability: "system-monitor-thermal",
		Question:   "Are host temperatures readable and below their trip points?",
	},
	{
		CellRef: "substrate/SB12", Rung: RungTelemetry,
		Classes:    []string{"memory-controller", "memory-module"},
		Capability: "system-monitor-memory-errors",
		Question:   "Are correctable and uncorrectable memory errors counted?",
	},
	{
		CellRef: "substrate/SB13", Rung: RungTelemetry,
		Classes:    []string{"network-interface"},
		Capability: "system-monitor-network",
		Question:   "Do hardware network interfaces carry traffic without error or drop growth?",
	},
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

	// Capability names the vocabulary capability serving this cell, and
	// CapabilityStatus is its resolution status on this cell's host OS,
	// joined from the portability grid.
	Capability       string
	CapabilityStatus string
	CapabilityReason string

	ObservedAt time.Time
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
func authoredStatus(definition *spacedoc.SpaceDefinition, cellRef string) (spacedoc.CellStatus, string, bool) {
	if definition == nil {
		return "", "", false
	}
	id := cellRef
	if index := strings.Index(cellRef, "/"); index >= 0 {
		id = cellRef[index+1:]
	}
	for _, cell := range definition.Cells {
		if cell.ID == id {
			return cell.Status, cell.Question, true
		}
	}
	return "", "", false
}
