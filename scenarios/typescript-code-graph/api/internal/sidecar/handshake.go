package sidecar

// ProtocolVersion is the IPC protocol major version this Go side
// speaks. The sidecar must echo the same number in its handshake_ok
// reply; any other value is a fatal mismatch.
const ProtocolVersion = 1

// handshakeRequest / handshakeResponse mirror plan §8.4 wire shapes.
type handshakeRequest struct {
	Type            string `json:"type"`
	RequestID       string `json:"request_id"`
	ProtocolVersion int    `json:"protocol_version"`
}

type handshakeResponse struct {
	Type            string `json:"type"`
	RequestID       string `json:"request_id"`
	ProtocolVersion int    `json:"protocol_version"`
	SidecarVersion  string `json:"sidecar_version"`
}
