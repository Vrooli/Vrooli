// Package portspec defines the canonical scenario port bands and the helpers
// that detect the host OS's ephemeral port window. Both the port allocator
// (internal/ports) and the scenario manifest loader (internal/scenario) depend
// on this leaf package; keeping it dependency-free prevents an import cycle.
package portspec

// Canonical port bands for scenario listeners.
//
// Every role sits below 32768 because Linux's default ephemeral range starts
// there (see ephemeral.go and docs/reference/port-allocation.md). Using a
// listener port inside the OS ephemeral range lets the kernel pick the same
// number as an outbound source port while the scenario is down, which causes
// intermittent "port already in use" failures with no visible offender.
const (
	APIRangeStart = 15000
	APIRangeEnd   = 19999
	UIRangeStart  = 20000
	UIRangeEnd    = 24999
	WSRangeStart  = 25000
	WSRangeEnd    = 29999

	// ReservedHeadroomStart/End carve a future-use band that remains below the
	// Linux ephemeral floor (32768). New roles should claim a sub-band here
	// rather than creeping into ephemeral territory.
	ReservedHeadroomStart = 30000
	ReservedHeadroomEnd   = 32767

	// CanonicalMax is the hard upper bound for any scenario listener port.
	// It matches the lowest ephemeral floor across supported operating
	// systems (Linux: 32768; macOS/Windows: 49152).
	CanonicalMax = 32767
)

// CanonicalRole describes which band a port belongs to.
type CanonicalRole string

const (
	RoleAPI      CanonicalRole = "api"
	RoleUI       CanonicalRole = "ui"
	RoleWS       CanonicalRole = "ws"
	RoleHeadroom CanonicalRole = "headroom"
	RoleUnknown  CanonicalRole = "unknown"
)

// CanonicalBand reports which canonical band the given port belongs to.
// Returns RoleUnknown when the port is not inside any canonical band; the
// second return value is true only when the port is inside a role band.
func CanonicalBand(port int) (CanonicalRole, bool) {
	switch {
	case port >= APIRangeStart && port <= APIRangeEnd:
		return RoleAPI, true
	case port >= UIRangeStart && port <= UIRangeEnd:
		return RoleUI, true
	case port >= WSRangeStart && port <= WSRangeEnd:
		return RoleWS, true
	case port >= ReservedHeadroomStart && port <= ReservedHeadroomEnd:
		return RoleHeadroom, true
	default:
		return RoleUnknown, false
	}
}

// IsAboveCanonicalMax reports whether the port is outside the canonical safe
// zone (i.e. inside a range that may overlap OS ephemeral allocations).
func IsAboveCanonicalMax(port int) bool {
	return port > CanonicalMax
}
