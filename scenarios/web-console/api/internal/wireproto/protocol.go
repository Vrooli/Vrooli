// Package wireproto owns the terminal WebSocket wire contract.
//
// Keep this package free of transport, PTY, and session dependencies. Both
// local and remote terminal handlers use the same message shape so a client
// cannot accidentally get a subtly different protocol by changing backends.
package wireproto

const ProtocolVersion = "1"

const (
	MsgTypeStdin                 = "stdin"
	MsgTypeStdout                = "stdout"
	MsgTypeResize                = "resize"
	MsgTypeResizeInfo            = "resize_info"
	MsgTypeSizeInfo              = "size_info"
	MsgTypeTakeLease             = "take_lease"
	MsgTypeExit                  = "exit"
	MsgTypeError                 = "error"
	MsgTypePing                  = "ping"
	MsgTypePong                  = "pong"
	MsgTypeSyncWarning           = "sync_warning"
	MsgTypeHistoryEnd            = "history_end"
	MsgTypeConversationAck       = "conversation_event_ack"
	MsgTypeSessionReady          = "session_ready"
	MsgTypeStdinAck              = "stdin_ack"
	MsgTypeControl               = "control"
	MsgTypeHello                 = "hello"
	MsgTypeResync                = "resync"
	MsgTypeSnapshotNotice        = "snapshot_notice"
	MsgTypeEchoState             = "echo_state"
	MsgTypeMouseMode             = "mouse_mode"
	StdinIntentTyping            = "typing"
	StdinIntentBulkText          = "bulk_text"
	StdinIntentNamedKey          = "named_key"
	StdinAckReasonTmuxFailed     = "tmux_write_failed"
	StdinAckReasonPTYClosed      = "pty_closed"
	StdinAckReasonOffsetGap      = "offset_gap"
	StdinAckReasonUnreconcilable = "unreconcilable"
)

// TerminalMessage is the JSON message exchanged by terminal WebSockets.
type TerminalMessage struct {
	Type            string `json:"type"`
	Data            string `json:"data,omitempty"`
	Cols            int    `json:"cols,omitempty"`
	Rows            int    `json:"rows,omitempty"`
	Code            int    `json:"code,omitempty"`
	CoalescedFrames int    `json:"coalesced_frames,omitempty"`
	EventID         string `json:"eventId,omitempty"`
	Source          string `json:"source,omitempty"`
	Stage           string `json:"stage,omitempty"`
	Backend         string `json:"backend,omitempty"`
	Offset          int64  `json:"offset,omitempty"`
	AcceptedThrough int64  `json:"accepted_through,omitempty"`
	HaveThrough     int64  `json:"have_through,omitempty"`
	ProtocolVersion string `json:"protocol_version,omitempty"`
	Ok              bool   `json:"ok,omitempty"`
	Gen             int64  `json:"gen,omitempty"`
	Intent          string `json:"intent,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Leader          string `json:"leader,omitempty"`
	LeaderDevice    string `json:"leaderDevice,omitempty"`
	HoldsLease      bool   `json:"holdsLease"`
	ViewerCount     int    `json:"viewerCount,omitempty"`
	EchoKnown       bool   `json:"echo_known,omitempty"`
	EchoEnabled     bool   `json:"echo_enabled,omitempty"`
	InAltBuffer     bool   `json:"in_alt_buffer,omitempty"`
	CursorAtLineEnd bool   `json:"cursor_at_line_end,omitempty"`
	MouseMode       bool   `json:"mouse_mode,omitempty"`
	MouseModeKnown  bool   `json:"mouse_mode_known,omitempty"`
}
