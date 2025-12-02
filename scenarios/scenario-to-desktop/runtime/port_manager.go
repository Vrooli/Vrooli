package bundleruntime

import (
	"fmt"
	"sync"

	"scenario-to-desktop-runtime/manifest"
)

// PortManager implements PortAllocator for managing dynamic port allocation.
type PortManager struct {
	manifest *manifest.Manifest
	dialer   NetworkDialer

	mu      sync.RWMutex
	portMap map[string]map[string]int // service ID -> port name -> port
}

// NewPortManager creates a new PortManager with the given dependencies.
func NewPortManager(m *manifest.Manifest, dialer NetworkDialer) *PortManager {
	return &PortManager{
		manifest: m,
		dialer:   dialer,
		portMap:  make(map[string]map[string]int),
	}
}

// Allocate assigns port numbers to each service based on manifest requirements.
// Ports are allocated sequentially within the specified or default range,
// skipping reserved ports and ports already in use.
func (p *PortManager) Allocate() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	defaultRange := DefaultPortRange
	if p.manifest.Ports != nil && p.manifest.Ports.DefaultRange != nil {
		defaultRange = PortRange{
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
				rng = PortRange{Min: req.Range.Min, Max: req.Range.Max}
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
func (p *PortManager) Resolve(serviceID, portName string) (int, error) {
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
func (p *PortManager) Map() map[string]map[string]int {
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

// buildReservedSet creates a set of reserved ports from the manifest.
func (p *PortManager) buildReservedSet() map[int]bool {
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
func (p *PortManager) pickPort(rng PortRange, reserved map[int]bool, next *int) (int, error) {
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
func (p *PortManager) isPortAvailable(port int) bool {
	ln, err := p.dialer.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// Ensure PortManager implements PortAllocator.
var _ PortAllocator = (*PortManager)(nil)
