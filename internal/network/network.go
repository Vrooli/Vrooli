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

func InspectPortListeners(port int) (PortInspection, error) {
	return inspectPortListeners(port)
}
