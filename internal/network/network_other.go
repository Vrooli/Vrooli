//go:build !linux

package network

func ListenersForPort(port int) ([]PortListener, error) {
	return nil, nil
}

func listenerInspectionStatus() ListenerInspection {
	return ListenerInspection{
		Available: false,
		Reason:    "listener inspection is only implemented on linux",
	}
}

func inspectPortListeners(port int) (PortInspection, error) {
	return PortInspection{Inspection: listenerInspectionStatus()}, nil
}
