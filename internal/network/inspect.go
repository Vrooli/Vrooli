package network

// Single-port inspection, answered from a one-shot TCPListenerSnapshot. This
// is the rich path used by diagnose-port and lifecycle readiness checks; bulk
// consumers (registry listing/cleanup, supervisor ticks, port allocation)
// should capture one snapshot themselves and query it per port instead.

func listenerInspectionStatus() ListenerInspection {
	snapshot := captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true})
	return ListenerInspection{
		Available: snapshot.Known,
		Tool:      snapshot.Tool,
		Reason:    snapshot.Reason,
	}
}

func inspectPortListeners(port int) PortInspection {
	return PortInspectionFromSnapshot(captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true}), port)
}

// PortInspectionFromSnapshot renders the legacy per-port inspection shape
// from a snapshot, letting bulk callers reuse one capture across many ports.
func PortInspectionFromSnapshot(snapshot TCPListenerSnapshot, port int) PortInspection {
	state := snapshot.Listening(port)
	if !state.Known {
		return PortInspection{Inspection: ListenerInspection{
			Available: false,
			Tool:      snapshot.Tool,
			Reason:    snapshot.Reason,
		}}
	}
	inspection := ListenerInspection{Available: true, Tool: snapshot.Tool}
	if !state.Listening {
		return PortInspection{Inspection: inspection}
	}
	if len(state.Listeners) == 0 {
		// The port is bound but the capture tool could not attribute it to a
		// process (other user's socket, or no enrichment tool installed).
		return PortInspection{
			Listeners:  []PortListener{{Command: "listener detected (process attribution unavailable)"}},
			Inspection: inspection,
		}
	}
	listeners := make([]PortListener, 0, len(state.Listeners))
	for _, listener := range state.Listeners {
		listeners = append(listeners, PortListener{
			PID:     listener.PID,
			Command: listener.Label,
			Zombie:  pidIsZombie(listener.PID),
		})
	}
	return PortInspection{Listeners: listeners, Inspection: inspection}
}
