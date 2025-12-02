package bundleruntime

import (
	"fmt"
	"net"

	"scenario-to-desktop-runtime/manifest"
)

// PortRange defines a range of port numbers for allocation.
type PortRange struct {
	Min int
	Max int
}

// DefaultPortRange is the fallback range when not specified in manifest.
var DefaultPortRange = PortRange{Min: 47000, Max: 48000}

// allocatePorts assigns port numbers to each service based on manifest requirements.
// Ports are allocated sequentially within the specified or default range,
// skipping reserved ports and ports already in use.
func (s *Supervisor) allocatePorts() error {
	defaultRange := DefaultPortRange
	if s.opts.Manifest.Ports != nil && s.opts.Manifest.Ports.DefaultRange != nil {
		defaultRange = PortRange{
			Min: s.opts.Manifest.Ports.DefaultRange.Min,
			Max: s.opts.Manifest.Ports.DefaultRange.Max,
		}
	}

	reserved := buildReservedSet(s.opts.Manifest)
	nextPort := defaultRange.Min

	for _, svc := range s.opts.Manifest.Services {
		if s.portMap[svc.ID] == nil {
			s.portMap[svc.ID] = make(map[string]int)
		}
		if svc.Ports == nil || len(svc.Ports.Requested) == 0 {
			continue
		}

		for _, req := range svc.Ports.Requested {
			rng := defaultRange
			if req.Range.Min != 0 && req.Range.Max != 0 {
				rng = PortRange{Min: req.Range.Min, Max: req.Range.Max}
			}
			port, err := pickPort(rng, reserved, &nextPort)
			if err != nil {
				return fmt.Errorf("allocate port for %s:%s: %w", svc.ID, req.Name, err)
			}
			s.portMap[svc.ID][req.Name] = port
		}
	}

	return nil
}

// buildReservedSet creates a set of reserved ports from the manifest.
func buildReservedSet(m *manifest.Manifest) map[int]bool {
	reserved := map[int]bool{}
	if m.Ports != nil {
		for _, p := range m.Ports.Reserved {
			reserved[p] = true
		}
	}
	return reserved
}

// pickPort finds an available port within the given range.
// It skips reserved ports and verifies the port is actually available
// by attempting to bind to it.
func pickPort(rng PortRange, reserved map[int]bool, next *int) (int, error) {
	if rng.Min == 0 || rng.Max == 0 || rng.Max < rng.Min {
		return 0, fmt.Errorf("invalid range %d-%d", rng.Min, rng.Max)
	}

	for p := rng.Min; p <= rng.Max; p++ {
		if reserved[p] {
			continue
		}
		if p < *next {
			continue
		}
		if !isPortAvailable(p) {
			continue
		}
		*next = p + 1
		return p, nil
	}
	return 0, fmt.Errorf("no free port in %d-%d", rng.Min, rng.Max)
}

// isPortAvailable checks if a port is available by attempting to bind to it.
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// resolvePort looks up an allocated port for a service.
func (s *Supervisor) resolvePort(serviceID, portName string) (int, error) {
	portGroup, ok := s.portMap[serviceID]
	if !ok {
		return 0, fmt.Errorf("no ports allocated for %s", serviceID)
	}
	port, ok := portGroup[portName]
	if !ok {
		return 0, fmt.Errorf("port %s missing for %s", portName, serviceID)
	}
	return port, nil
}

// PortMap returns a copy of the current port allocation map.
func (s *Supervisor) PortMap() map[string]map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]map[string]int)
	for svc, ports := range s.portMap {
		out[svc] = make(map[string]int)
		for name, port := range ports {
			out[svc][name] = port
		}
	}
	return out
}
