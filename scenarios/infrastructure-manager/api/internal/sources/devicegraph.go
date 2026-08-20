package sources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// deviceGraphScenario owns the device graph. The graph itself is shipped and
// emitting on a 60s collector inside that scenario; what this file adds is the
// join, which is the thing the substrate space document says is missing.
const deviceGraphScenario = "system-monitor"

// deviceGraphVerb names the typed read this source consumes. It is stated as a
// constant so an unavailable source can name the verb it wanted rather than
// reporting a bare transport failure — "system-monitor is unreachable" and
// "system-monitor publishes no device-graph verb" are different problems with
// different owners.
const deviceGraphVerb = "system-monitor DeviceGraphService/GetDeviceGraph"

// RungState is one rung's grade for one device or subsystem, carried across
// the transport boundary exactly as the owner graded it. The tokens are the
// owner's (`measured`, `unmeasurable`, `unavailable`, `not_applicable`); this
// package does not reinterpret them, because a transport that re-grades a
// reading is a second opinion pretending to be a measurement.
type RungState struct {
	Rung        string
	State       string
	Reason      string
	Mechanism   string
	Remediation string
	ObservedAt  time.Time
}

// GraphDevice is one graded hardware device.
type GraphDevice struct {
	ID     string
	Class  string
	Vendor string
	Model  string
	Driver string
	Rungs  map[string]RungState
}

// GraphSubsystem is a host-wide graded fact not attached to a single device,
// such as "no EDAC memory controller registers on this host".
type GraphSubsystem struct {
	Name  string
	Rungs map[string]RungState
}

// DeviceGraph is one complete observation of the host's device topology.
type DeviceGraph struct {
	CollectedAt time.Time
	Platform    string
	Devices     []GraphDevice
	Subsystems  []GraphSubsystem
	// VirtualNetworkInterfaces names interfaces that exist in the kernel but
	// are not hardware. They are carried so their exclusion from the hardware
	// grading is visible rather than silent.
	VirtualNetworkInterfaces []string
	Warnings                 []string
}

// ErrDeviceGraphVerbUnpublished is returned when the owning scenario resolves
// but exposes no typed device-graph read. It is deliberately distinct from a
// transport error: a peer that is up and simply does not publish the verb is
// an unclosed join, which is the instrument's own work, whereas a peer that is
// down is an outage, which is not.
var ErrDeviceGraphVerbUnpublished = errors.New("the device graph is not reachable over a typed read verb")

// DeviceGraphTransport is the typed read this source needs from the device
// graph's owner, resolved to a base URL by discovery. It is an interface so
// the generated Connect client is injected rather than assumed: the reader's
// availability semantics, deadline handling and unit mapping are what this
// package owns and tests, and they must not change when the client does.
type DeviceGraphTransport interface {
	DeviceGraph(ctx context.Context, baseURL string) (DeviceGraph, error)
}

// DeviceGraphReader reads the graded device graph from its owning scenario.
// It assigns no trust and grades nothing; the ladder domain owns those rules.
type DeviceGraphReader struct {
	Resolver  *discovery.Resolver
	HTTP      *http.Client
	Transport DeviceGraphTransport
}

// ReadGraph resolves the owner and reads the graph. Both failure modes are
// reported by returning an error, never by returning an empty graph: an empty
// graph is a claim that the host has no hardware.
func (r DeviceGraphReader) ReadGraph(ctx context.Context) (DeviceGraph, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, deviceGraphScenario)
	if err != nil {
		return DeviceGraph{}, fmt.Errorf("resolve %s: %w", deviceGraphScenario, err)
	}
	if r.Transport == nil {
		return DeviceGraph{}, fmt.Errorf("%w: %s is reachable at %s but %s is not wired into this source",
			ErrDeviceGraphVerbUnpublished, deviceGraphScenario, base, deviceGraphVerb)
	}
	graph, err := r.Transport.DeviceGraph(ctx, base)
	if err != nil {
		return DeviceGraph{}, fmt.Errorf("read %s: %w", deviceGraphVerb, err)
	}
	return graph, nil
}

// DeviceGraphSourceID identifies this source in availability reporting.
const DeviceGraphSourceID = "system-monitor/device-graph"

// ReadDeviceGraph runs the device-graph read under the standard per-source
// deadline.
func ReadDeviceGraph(ctx context.Context, reader DeviceGraphReader, timeout time.Duration) TypedResult[DeviceGraph] {
	return ReadTyped(ctx, DeviceGraphSourceID, reader.ReadGraph, timeout)
}
