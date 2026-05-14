// events.go: ControlEvent emission.
//
// Some consumers care less about the printable grid and more about the
// stream of *control* events the parser sees: alt-buffer enter/exit,
// CSI DA/DECRQM queries (which the server must answer), OSC color/title
// updates, etc. ControlEvents() returns a bounded channel of such events
// so observers can subscribe without re-parsing the byte stream.
//
// Backpressure: the channel is bounded (256 events). When full, the
// oldest event is dropped and DroppedControlEvents() is incremented.
// This matches the read-loop's other "best effort" surfaces — the goal
// is "never block the PTY read," not "deliver every event."

package terminal

import "sync/atomic"

// ControlEventKind enumerates the parsed control events the emulator
// surfaces. New kinds may be added; consumers should ignore unknown
// kinds for forward compatibility.
type ControlEventKind int

const (
	EventUnknown ControlEventKind = iota
	// EventAltBufferEnter fires when the emulator switches into the
	// alternate screen buffer (DEC modes 47/1047/1049 SET).
	EventAltBufferEnter
	// EventAltBufferExit fires when the emulator leaves the alternate
	// screen buffer.
	EventAltBufferExit
	// EventCSIQuery fires for CSI sequences that expect a server reply:
	//   DA1 (CSI c), DA3 (CSI = c), DECRQM 2026 (CSI ? 2026 $ p), etc.
	// Params is the parameter list; Final is the final byte.
	EventCSIQuery
	// EventOSC fires for OSC sequences. Payload is the OSC body without
	// the leading ESC ] or trailing BEL / ST.
	EventOSC
)

// ControlEvent is a single parsed control event.
type ControlEvent struct {
	Kind    ControlEventKind
	Private bool   // for CSI: whether the private (? / > / <) flag was set
	Params  []int  // for CSI
	Final   byte   // for CSI: final byte (e.g. 'c', 'p')
	Payload []byte // for OSC: body bytes
}

const defaultControlEventBuffer = 256

// ControlEvents returns the read end of the bounded event channel. The
// channel is lazily created on first call; multiple calls return the
// same channel. Closing is the emulator's responsibility (currently
// never closed — the emulator outlives the channel's consumers).
func (e *Emulator) ControlEvents() <-chan ControlEvent {
	if e.events == nil {
		e.events = make(chan ControlEvent, defaultControlEventBuffer)
	}
	return e.events
}

// DroppedControlEvents returns the running count of control events
// dropped due to a full buffer.
func (e *Emulator) DroppedControlEvents() uint64 {
	return atomic.LoadUint64(&e.droppedEvents)
}

// emitControlEvent pushes an event onto the channel without blocking.
// If the buffer is full the oldest event is dropped (drop-oldest) and
// the drop counter is incremented.
func (e *Emulator) emitControlEvent(ev ControlEvent) {
	if e.events == nil {
		return
	}
	for {
		select {
		case e.events <- ev:
			return
		default:
			// Drop the oldest event to make room.
			select {
			case <-e.events:
				atomic.AddUint64(&e.droppedEvents, 1)
			default:
				// Buffer emptied between checks; loop and retry the send.
			}
		}
	}
}
