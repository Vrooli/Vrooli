// Package devicegraph builds a parent-linked graph of the host's physical
// devices and grades every device against the five-rung capability ladder.
//
// The ladder rungs are ordered by dependency: a device must be identified
// before it can be measured, measured before the measurement can be retained,
// retained before an operator can act on it, and acted upon before a forward
// looking signal means anything.
//
//	identity     the device is enumerated under a platform-durable address
//	telemetry    a current value is readable right now
//	evidence     that value is retained where it can be audited later
//	control      the host grants the access needed to read or act on the device
//	anticipation a forward-looking signal exists (trend, wear, error accrual)
//
// The single most important invariant in this package: a reading that could
// not be taken is reported as StateUnmeasurable WITH A REASON. It is never
// reported as zero, never as healthy, and never silently omitted.
package devicegraph

import (
	"fmt"
	"sort"
	"time"
)

// Rung names one step of the capability ladder.
type Rung string

// The five ladder rungs, in dependency order.
const (
	RungIdentity     Rung = "identity"
	RungTelemetry    Rung = "telemetry"
	RungEvidence     Rung = "evidence"
	RungControl      Rung = "control"
	RungAnticipation Rung = "anticipation"
)

// Rungs is the ladder in dependency order.
var Rungs = []Rung{RungIdentity, RungTelemetry, RungEvidence, RungControl, RungAnticipation}

// State is the outcome of grading one rung for one device.
type State string

const (
	// StateMeasured means a real value was obtained from the host.
	StateMeasured State = "measured"
	// StateUnmeasurable means the rung applies to this device and a value
	// SHOULD exist, but the host refused or could not produce it. This is
	// distinct from zero, from healthy, and from unavailable. Every
	// unmeasurable state carries a Reason naming what blocked the read.
	StateUnmeasurable State = "unmeasurable"
	// StateUnavailable means the mechanism that would produce the value is
	// not present on this host at all (missing tool, missing interface).
	StateUnavailable State = "unavailable"
	// StateNotApplicable means the rung is meaningless for this device class.
	StateNotApplicable State = "not_applicable"
)

// RungState records one rung's grade together with the evidence for it.
type RungState struct {
	Rung  Rung  `json:"rung"`
	State State `json:"state"`
	// Reason is mandatory for every state other than StateMeasured. It names
	// what blocked the reading in operator-readable terms.
	Reason string `json:"reason,omitempty"`
	// Mechanism names the interface the reading was (or would be) taken from.
	Mechanism string `json:"mechanism,omitempty"`
	// Remediation names the declared host change that would close the gap.
	// It is set on control-rung gaps that a commissioning step can fix.
	Remediation string    `json:"remediation,omitempty"`
	ObservedAt  time.Time `json:"observed_at"`
}

// Class names the kind of hardware a device is.
type Class string

const (
	ClassPCIDevice        Class = "pci-device"
	ClassUSBDevice        Class = "usb-device"
	ClassBlockDevice      Class = "block-device"
	ClassGraphicsDevice   Class = "graphics-device"
	ClassNetworkInterface Class = "network-interface"
	ClassThermalSensor    Class = "thermal-sensor"
	ClassMemoryController Class = "memory-controller"
	ClassMemoryModule     Class = "memory-module"
)

// Device is one node of the graph. Identity is the durable, platform-stable
// address (for example "pci:0000:01:00.0") so the same physical part keeps the
// same ID across reboots and across the collectors that observe it.
type Device struct {
	ID       string `json:"id"`
	Class    Class  `json:"class"`
	ParentID string `json:"parent_id,omitempty"`
	Vendor   string `json:"vendor,omitempty"`
	Model    string `json:"model,omitempty"`
	Driver   string `json:"driver,omitempty"`
	// SysPath is the resolved platform path the device was enumerated from.
	// It is provenance, not identity, and is used to attach sensors to the
	// device they measure.
	SysPath    string             `json:"sys_path,omitempty"`
	Attributes map[string]string  `json:"attributes,omitempty"`
	Readings   map[string]float64 `json:"readings,omitempty"`
	Rungs      map[Rung]RungState `json:"rungs"`
}

// Subsystem records a host-wide fact that is not attached to a single device,
// such as "no EDAC memory controller registers on this host". Grading a
// subsystem uses the same ladder as a device.
type Subsystem struct {
	Name       string             `json:"name"`
	Attributes map[string]string  `json:"attributes,omitempty"`
	Rungs      map[Rung]RungState `json:"rungs"`
}

// Graph is one complete observation of the host's device topology.
type Graph struct {
	CollectedAt time.Time   `json:"collected_at"`
	Platform    string      `json:"platform"`
	Devices     []Device    `json:"devices"`
	Subsystems  []Subsystem `json:"subsystems"`
	// VirtualNetworkInterfaces names interfaces that exist in the kernel but
	// are not hardware (bridges, veth, loopback, container interfaces). They
	// are recorded so their exclusion is visible, and deliberately not graded.
	VirtualNetworkInterfaces []string `json:"virtual_network_interfaces,omitempty"`
	Warnings                 []string `json:"warnings,omitempty"`
}

func (g *Graph) addDevice(d Device) {
	g.Devices = append(g.Devices, d)
}

func (g *Graph) addSubsystem(s Subsystem) {
	g.Subsystems = append(g.Subsystems, s)
}

func (g *Graph) warn(format string, args ...any) {
	g.Warnings = append(g.Warnings, fmt.Sprintf(format, args...))
}

// DeviceByID returns the device with the given identity.
func (g *Graph) DeviceByID(id string) (Device, bool) {
	for _, device := range g.Devices {
		if device.ID == id {
			return device, true
		}
	}
	return Device{}, false
}

// DevicesOfClass returns every device of one class, in enumeration order.
func (g *Graph) DevicesOfClass(class Class) []Device {
	matches := make([]Device, 0, len(g.Devices))
	for _, device := range g.Devices {
		if device.Class == class {
			matches = append(matches, device)
		}
	}
	return matches
}

// SubsystemByName returns the named subsystem grade.
func (g *Graph) SubsystemByName(name string) (Subsystem, bool) {
	for _, subsystem := range g.Subsystems {
		if subsystem.Name == name {
			return subsystem, true
		}
	}
	return Subsystem{}, false
}

// Children returns the devices whose parent is the given identity.
func (g *Graph) Children(id string) []Device {
	matches := make([]Device, 0)
	for _, device := range g.Devices {
		if device.ParentID == id {
			matches = append(matches, device)
		}
	}
	return matches
}

// RungCounts summarizes how many devices sit in each state on each rung. It is
// the graph-level readout the monitor publishes, and it makes an unmeasurable
// population impossible to mistake for a healthy one.
func (g *Graph) RungCounts() map[Rung]map[State]int {
	counts := make(map[Rung]map[State]int, len(Rungs))
	for _, rung := range Rungs {
		counts[rung] = map[State]int{}
	}
	for _, device := range g.Devices {
		for rung, state := range device.Rungs {
			if _, known := counts[rung]; !known {
				counts[rung] = map[State]int{}
			}
			counts[rung][state.State]++
		}
	}
	return counts
}

// Validate enforces the invariants the rest of the system relies on: every
// device carries a grade for every rung, every non-measured grade explains
// itself, and every declared parent exists in the graph.
func (g *Graph) Validate() error {
	ids := make(map[string]struct{}, len(g.Devices))
	for _, device := range g.Devices {
		if device.ID == "" {
			return fmt.Errorf("device of class %q has no identity", device.Class)
		}
		if _, duplicate := ids[device.ID]; duplicate {
			return fmt.Errorf("device identity %q is declared twice", device.ID)
		}
		ids[device.ID] = struct{}{}
	}
	for _, device := range g.Devices {
		if device.ParentID != "" {
			if _, ok := ids[device.ParentID]; !ok {
				return fmt.Errorf("device %q declares parent %q which is not in the graph", device.ID, device.ParentID)
			}
		}
		if err := validateRungs(fmt.Sprintf("device %q", device.ID), device.Rungs); err != nil {
			return err
		}
	}
	for _, subsystem := range g.Subsystems {
		if err := validateRungs(fmt.Sprintf("subsystem %q", subsystem.Name), subsystem.Rungs); err != nil {
			return err
		}
	}
	return nil
}

func validateRungs(subject string, rungs map[Rung]RungState) error {
	if len(rungs) == 0 {
		return fmt.Errorf("%s carries no ladder grades", subject)
	}
	missing := make([]string, 0, len(Rungs))
	for _, rung := range Rungs {
		state, ok := rungs[rung]
		if !ok {
			missing = append(missing, string(rung))
			continue
		}
		if state.Rung != rung {
			return fmt.Errorf("%s stores rung %q under key %q", subject, state.Rung, rung)
		}
		switch state.State {
		case StateMeasured:
		case StateUnmeasurable, StateUnavailable, StateNotApplicable:
			if state.Reason == "" {
				return fmt.Errorf("%s grades rung %q as %q without a reason", subject, rung, state.State)
			}
		default:
			return fmt.Errorf("%s grades rung %q with unknown state %q", subject, rung, state.State)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("%s is missing ladder grades: %v", subject, missing)
	}
	return nil
}
