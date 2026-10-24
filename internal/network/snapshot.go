package network

// SnapshotListener identifies a process attached to a listening socket, when
// attribution is available. Attribution is best-effort: a port can be
// listening with an empty listener list (e.g. the socket belongs to another
// user and the capture tool cannot see its process).
type SnapshotListener struct {
	PID   int    `json:"pid"`
	Label string `json:"label,omitempty"`
}

// ListenerState is the per-port answer derived from a TCPListenerSnapshot.
// Known=false means the snapshot could not answer at all for this capture;
// callers must treat that as missing evidence — never as "not listening".
type ListenerState struct {
	Known     bool
	Listening bool
	Listeners []SnapshotListener
}

// TCPListenerSnapshot is a one-shot capture of every listening TCP socket on
// the host (all users, IPv4+IPv6). Capture failure is folded into the
// snapshot itself rather than returned as an error: every Listening() call
// then answers Known:false, which keeps the "unknown is not absence"
// invariant in one place.
type TCPListenerSnapshot struct {
	// Known reports whether the capture produced a usable global port set.
	Known bool
	// Reason describes why the capture is unusable when Known is false.
	Reason string
	// Tool names the evidence source(s) for diagnostics.
	Tool string
	// Ports maps each listening TCP port to its (possibly empty) process
	// attribution.
	Ports map[int][]SnapshotListener
}

// Listening answers whether anything is listening on the given port.
func (s TCPListenerSnapshot) Listening(port int) ListenerState {
	if !s.Known || port <= 0 {
		return ListenerState{}
	}
	listeners, listening := s.Ports[port]
	return ListenerState{Known: true, Listening: listening, Listeners: listeners}
}

// CaptureOptions tunes how much evidence one capture collects.
type CaptureOptions struct {
	// AttributeProcesses collects best-effort PID and label attribution for
	// each listening port. Attribution is the expensive half of a capture: it
	// costs one extra subprocess on Linux (ss) and macOS (lsof), and nothing
	// on Windows, where the iphlpapi table already carries owner PIDs.
	//
	// Leave it false when the caller only needs to know whether a port is
	// bound. Reconciliation does exactly that — it reduces every capture to
	// Known/Listening — so paying for attribution there forks a subprocess per
	// lookup and discards the result.
	AttributeProcesses bool
}

// CaptureTCPListenerSnapshot collects the host-global listening-port set once,
// with process attribution. Callers that need answers for many ports must
// capture once and query the snapshot instead of inspecting per port.
func CaptureTCPListenerSnapshot() TCPListenerSnapshot {
	return captureTCPListenerSnapshot(CaptureOptions{AttributeProcesses: true})
}

// CaptureTCPListenerPorts collects the same host-global listening-port set
// without process attribution. The Ports map is populated with empty listener
// lists, so Listening() still answers Known and Listening correctly; only the
// per-port Listeners slice is left empty.
func CaptureTCPListenerPorts() TCPListenerSnapshot {
	return captureTCPListenerSnapshot(CaptureOptions{})
}
