package bundleruntime

// PortRange defines a range of port numbers for allocation.
type PortRange struct {
	Min int
	Max int
}

// DefaultPortRange is the fallback range when not specified in manifest.
var DefaultPortRange = PortRange{Min: 47000, Max: 48000}

// resolvePort looks up an allocated port for a service.
// Delegates to the injected PortAllocator.
func (s *Supervisor) resolvePort(serviceID, portName string) (int, error) {
	return s.portAllocator.Resolve(serviceID, portName)
}

// PortMap returns a copy of the current port allocation map.
// Delegates to the injected PortAllocator.
func (s *Supervisor) PortMap() map[string]map[string]int {
	return s.portAllocator.Map()
}
