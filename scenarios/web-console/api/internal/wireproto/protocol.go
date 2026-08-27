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
	MsgTypePresence              = "presence"
	MsgTypeDeviceState           = "device_state"
	StdinIntentTyping            = "typing"
	StdinIntentBulkText          = "bulk_text"
	StdinIntentNamedKey          = "named_key"
	StdinAckReasonTmuxFailed     = "tmux_write_failed"
	StdinAckReasonPTYClosed      = "pty_closed"
	StdinAckReasonOffsetGap      = "offset_gap"
	StdinAckReasonUnreconcilable = "unreconcilable"
	StdinAckReasonQueueFull      = "input_queue_full"
)

// TerminalMessage is the JSON message exchanged by terminal WebSockets.
//
// Tag casing is historically mixed: the lease and presence fields are
// camelCase while the stream and echo fields are snake_case. Both spellings
// are load-bearing on the wire, so normalizing them is a breaking change that
// belongs behind a ProtocolVersion bump rather than an in-place rename. New
// fields follow the casing of the group they join.
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
	RenderedThrough int64  `json:"rendered_through,omitempty"`
	OutputCursor    int64  `json:"output_cursor,omitempty"`
	WantResume      bool   `json:"want_resume,omitempty"`
	ProtocolVersion string `json:"protocol_version,omitempty"`
	Ok              bool   `json:"ok,omitempty"`
	Gen             int64  `json:"gen,omitempty"`
	Intent          string `json:"intent,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Leader          string `json:"leader,omitempty"`
	LeaderDevice    string `json:"leaderDevice,omitempty"`
	// DeviceClass is the leader's self-declared device family, used only to
	// choose a follower's decorative frame. It is never an authorization
	// signal and is not a hardware identity claim; an operator can edit it.
	DeviceClass string `json:"deviceClass,omitempty"`
	// KbOpen reports whether the leader's virtual keyboard currently covers
	// part of its viewport. Followers draw this state instead of inferring it
	// from the grid, which changes for many reasons besides a keyboard.
	KbOpen          bool `json:"kbOpen,omitempty"`
	HoldsLease      bool `json:"holdsLease"`
	ViewerCount     int  `json:"viewerCount,omitempty"`
	EchoKnown       bool `json:"echo_known,omitempty"`
	EchoEnabled     bool `json:"echo_enabled,omitempty"`
	InAltBuffer     bool `json:"in_alt_buffer,omitempty"`
	CursorAtLineEnd bool `json:"cursor_at_line_end,omitempty"`
	MouseMode       bool `json:"mouse_mode,omitempty"`
	MouseModeKnown  bool `json:"mouse_mode_known,omitempty"`
}
