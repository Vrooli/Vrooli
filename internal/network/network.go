package network

type PortListener struct {
	PID     int    `json:"pid"`
	Command string `json:"command,omitempty"`
	Zombie  bool   `json:"zombie"`
}

type ListenerInspection struct {
	Available bool   `json:"available"`
	Tool      string `json:"tool,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type PortInspection struct {
	Listeners  []PortListener     `json:"listeners,omitempty"`
	Inspection ListenerInspection `json:"inspection"`
}

func ListenerInspectionStatus() ListenerInspection {
	return listenerInspectionStatus()
}

// InspectPortListeners answers a single-port inspection from a one-shot
// snapshot. Capture failure is folded into the result (Inspection.Available is
// false), so there is no error to return.
func InspectPortListeners(port int) PortInspection {
	return inspectPortListeners(port)
}
