package sse

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// Frame is one parsed Server-Sent-Event frame.
//
// The parser supports both `event: <name>\ndata: <payload>\n\n` (named
// event) and bare `data: <payload>\n\n` (default `message` event).
// `data:` lines accumulate (joined by `\n` per the spec); a blank
// line dispatches.
//
// Frame is the inverse of encodeFrame in this package — together they
// pin the wire-format invariant tested by TestEncodeFrame_RoundTrips.
type Frame struct {
	// Event is the event name (e.g., "exit", "end"). Empty when the
	// frame had no `event:` line — by spec that means the default
	// `message` event.
	Event string

	// Data is the raw payload (may be empty). Multi-line payloads
	// preserve `\n` separators.
	Data []byte
}

// ParseStream reads `r` to EOF and returns every dispatched frame.
// Comment lines (`:` prefix) and unknown fields are skipped silently.
//
// Why a strict parser lives next to the encoder: SSE consumers across
// the codebase (the agent-manager runtime, the handler tests, every
// future SSE-shaped seam) need a single source of truth for the wire
// contract. Putting parser and encoder in the same package keeps them
// from drifting and means round-trip tests can reach them both
// without import gymnastics.
func ParseStream(r io.Reader) []Frame {
	var frames []Frame
	var event string
	var data bytes.Buffer

	dispatch := func() {
		if event == "" && data.Len() == 0 {
			return
		}
		// Trim the trailing \n that accumulates between data fields.
		payload := data.Bytes()
		if len(payload) > 0 && payload[len(payload)-1] == '\n' {
			payload = payload[:len(payload)-1]
		}
		out := make([]byte, len(payload))
		copy(out, payload)
		frames = append(frames, Frame{Event: event, Data: out})
		event = ""
		data.Reset()
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			dispatch()
			continue
		}
		if strings.HasPrefix(line, ":") {
			// Comment line per spec; ignore.
			continue
		}
		field := line
		value := ""
		if idx := strings.Index(line, ":"); idx >= 0 {
			field = line[:idx]
			value = strings.TrimPrefix(line[idx+1:], " ")
		}
		switch field {
		case "event":
			event = value
		case "data":
			data.WriteString(value)
			data.WriteByte('\n')
		case "id", "retry":
			// Tracked by clients but not needed for our assertions.
		}
	}
	// Final dispatch in case stream didn't end with a blank line.
	dispatch()
	return frames
}
