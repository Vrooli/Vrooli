// Package ports provides dynamic port allocation for bundle services.
package ports

import (
	"fmt"
	"sync"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
)

// Range defines a range of port numbers for allocation.
type Range struct {
	Min int
	Max int
}

// DefaultRange is the fallback range when not specified in manifest.
var DefaultRange = Range{Min: 47000, Max: 48000}

// Allocator abstracts port allocation for testing.
type Allocator interface {
	// Allocate assigns ports to all services based on manifest requirements.
	Allocate() error
	// Resolve looks up an allocated port for a service.
	Resolve(serviceID, portName string) (int, error)
	// Map returns a copy of all allocated ports.
	Map() map[string]map[string]int
}

// Manager implements Allocator for managing dynamic port allocation.
type Manager struct {
	manifest *manifest.Manifest
	dialer   infra.NetworkDialer

	mu      sync.RWMutex
	portMap map[string]map[string]int // service ID -> port name -> port
}

// NewManager creates a new Manager with the given dependencies.
func NewManager(m *manifest.Manifest, dialer infra.NetworkDialer) *Manager {
	return &Manager{
		manifest: m,
		dialer:   dialer,
		portMap:  make(map[string]map[string]int),
	}
}

// Allocate assigns port numbers to each service based on manifest requirements.
// Ports are allocated sequentially within the specified or default range,
// skipping reserved ports and ports already in use.
func (p *Manager) Allocate() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	defaultRange := DefaultRange
	if p.manifest.Ports != nil && p.manifest.Ports.DefaultRange != nil {
		defaultRange = Range{
			Min: p.manifest.Ports.DefaultRange.Min,
			Max: p.manifest.Ports.DefaultRange.Max,
		}
	}

	reserved := p.buildReservedSet()
	nextPort := defaultRange.Min

	for _, svc := range p.manifest.Services {
		if p.portMap[svc.ID] == nil {
			p.portMap[svc.ID] = make(map[string]int)
		}
		if svc.Ports == nil || len(svc.Ports.Requested) == 0 {
			continue
		}

		for _, req := range svc.Ports.Requested {
			rng := defaultRange
			if req.Range.Min != 0 && req.Range.Max != 0 {
				rng = Range{Min: req.Range.Min, Max: req.Range.Max}
			}
			port, err := p.pickPort(rng, reserved, &nextPort)
			if err != nil {
				return fmt.Errorf("allocate port for %s:%s: %w", svc.ID, req.Name, err)
			}
			p.portMap[svc.ID][req.Name] = port
		}
	}

	return nil
}

// Resolve looks up an allocated port for a service.
func (p *Manager) Resolve(serviceID, portName string) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	portGroup, ok := p.portMap[serviceID]
	if !ok {
		return 0, fmt.Errorf("no ports allocated for %s", serviceID)
	}
	port, ok := portGroup[portName]
	if !ok {
		return 0, fmt.Errorf("port %s missing for %s", portName, serviceID)
	}
	return port, nil
}

// Map returns a copy of the current port allocation map.
func (p *Manager) Map() map[string]map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make(map[string]map[string]int)
	for svc, ports := range p.portMap {
		out[svc] = make(map[string]int)
		for name, port := range ports {
			out[svc][name] = port
		}
	}
	return out
}

// SetPorts allows directly setting the port map (for testing).
func (p *Manager) SetPorts(portMap map[string]map[string]int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.portMap = portMap
}

// buildReservedSet creates a set of reserved ports from the manifest.
func (p *Manager) buildReservedSet() map[int]bool {
	reserved := map[int]bool{}
	if p.manifest.Ports != nil {
		for _, port := range p.manifest.Ports.Reserved {
			reserved[port] = true
		}
	}
	return reserved
}

// pickPort finds an available port within the given range.
// It skips reserved ports and verifies the port is actually available
// by attempting to bind to it.
func (p *Manager) pickPort(rng Range, reserved map[int]bool, next *int) (int, error) {
	if rng.Min == 0 || rng.Max == 0 || rng.Max < rng.Min {
		return 0, fmt.Errorf("invalid range %d-%d", rng.Min, rng.Max)
	}

	for port := rng.Min; port <= rng.Max; port++ {
		if reserved[port] {
			continue
		}
		if port < *next {
			continue
		}
		if !p.isPortAvailable(port) {
			continue
		}
		*next = port + 1
		return port, nil
	}
	return 0, fmt.Errorf("no free port in %d-%d", rng.Min, rng.Max)
}

// isPortAvailable checks if a port is available by attempting to bind to it.
func (p *Manager) isPortAvailable(port int) bool {
	ln, err := p.dialer.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Ensure Manager implements Allocator.
var _ Allocator = (*Manager)(nil)
