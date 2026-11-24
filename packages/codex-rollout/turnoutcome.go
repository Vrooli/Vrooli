package codexrollout

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// Turn boundary events in a rollout, as Codex writes them in event_msg
// records.
const (
	EventTurnStarted  = "task_started"
	EventTurnComplete = "task_complete"
	EventTurnAborted  = "turn_aborted"
)

// TurnError is the error a task_complete record carries when its turn ended
// on one.
type TurnError struct {
	Message string
	// Info is codex_error_info normalised to one key: the string itself
	// ("other", "server_overloaded"), or the single key of an object
	// ({"http_connection_failed": {...}} is "http_connection_failed").
	Info string
	// HTTPStatus is the http_status_code inside codex_error_info, or 0.
	HTTPStatus int
}

// TurnOutcome is the newest turn boundary in one thread's rollout.
type TurnOutcome struct {
	// Event is EventTurnStarted (a turn is running), EventTurnComplete or
	// EventTurnAborted.
	Event  string
	TurnID string
	// At is the boundary record's timestamp.
	At time.Time
	// Error is set when the newest turn completed on an error.
	Error *TurnError
	// AbortReason is turn_aborted's reason ("interrupted").
	AbortReason string
}

// EndedOnError reports whether the newest turn ended on an error.
func (o TurnOutcome) EndedOnError() bool { return o.Event == EventTurnComplete && o.Error != nil }

// ErrNoTurnBoundary means no turn boundary was found within the bytes read.
var ErrNoTurnBoundary = errors.New("no turn boundary in the rollout's tail")

// tailChunk is how much is read per step backwards from the end.
const tailChunk = 256 << 10

// ReadLastTurn returns the newest turn boundary of the rollout at path,
// reading backwards from the end at most maxBytes. Records after the
// boundary (token counts, sub-agent completions) are skipped. A rollout with
// no boundary within maxBytes is ErrNoTurnBoundary: the outcome is unknown.
func ReadLastTurn(path string, maxBytes int64) (TurnOutcome, error) {
	file, err := os.Open(path)
	if err != nil {
		return TurnOutcome{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return TurnOutcome{}, err
	}
	return lastTurn(file, info.Size(), maxBytes)
}

func lastTurn(r io.ReaderAt, size, maxBytes int64) (TurnOutcome, error) {
	if maxBytes <= 0 {
		return TurnOutcome{}, fmt.Errorf("a positive byte limit is required")
	}
	floor := max(size-maxBytes, 0)
	pos := size
	// partial holds the bytes after the newest newline not yet consumed: the
	// tail of a line whose start lies further back.
	var partial []byte
	for pos > floor {
		n := min(int64(tailChunk), pos-floor)
		pos -= n
		chunk := make([]byte, n, n+int64(len(partial)))
		if _, err := r.ReadAt(chunk, pos); err != nil && !errors.Is(err, io.EOF) {
			return TurnOutcome{}, err
		}
		buf := append(chunk, partial...)
		// Every line but the first in buf is complete; the first is
		// complete only at the start of the file.
		for {
			i := bytes.LastIndexByte(buf, '\n')
			if i < 0 {
				break
			}
			if outcome, ok, err := boundary(buf[i+1:]); ok || err != nil {
				return outcome, err
			}
			buf = buf[:i]
		}
		if int64(len(buf)) > MaxLineBytes {
			return TurnOutcome{}, fmt.Errorf("rollout line longer than %d bytes", MaxLineBytes)
		}
		partial = buf
	}
	if floor == 0 {
		if outcome, ok, err := boundary(partial); ok || err != nil {
			return outcome, err
		}
	}
	return TurnOutcome{}, ErrNoTurnBoundary
}

// boundaryMarkers are substrings every boundary record holds; a line
// without any is skipped undecoded, so long records cost one scan.
var boundaryMarkers = [][]byte{[]byte(EventTurnStarted), []byte(EventTurnComplete), []byte(EventTurnAborted)}

type boundaryRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type            string `json:"type"`
		TurnID          string `json:"turn_id"`
		Reason          string `json:"reason"`
		TurnAttribution struct {
			TurnID string `json:"turn_id"`
		} `json:"turn_attribution"`
		Error *struct {
			Message        string          `json:"message"`
			CodexErrorInfo json.RawMessage `json:"codex_error_info"`
		} `json:"error"`
	} `json:"payload"`
}

// boundary decodes one line when it is a turn boundary record.
func boundary(line []byte) (TurnOutcome, bool, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return TurnOutcome{}, false, nil
	}
	marked := false
	for _, m := range boundaryMarkers {
		if bytes.Contains(line, m) {
			marked = true
			break
		}
	}
	if !marked {
		return TurnOutcome{}, false, nil
	}
	var rec boundaryRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		// A line naming a boundary inside other text (a tool output that
		// quotes a rollout) is not one.
		return TurnOutcome{}, false, nil
	}
	if rec.Type != "event_msg" {
		return TurnOutcome{}, false, nil
	}
	p := rec.Payload
	out := TurnOutcome{Event: p.Type, TurnID: p.TurnID}
	switch p.Type {
	case EventTurnStarted:
		if out.TurnID == "" {
			out.TurnID = p.TurnAttribution.TurnID
		}
	case EventTurnComplete:
		if p.Error != nil {
			out.Error = &TurnError{Message: p.Error.Message}
			out.Error.Info, out.Error.HTTPStatus = errorInfo(p.Error.CodexErrorInfo)
		}
	case EventTurnAborted:
		out.AbortReason = p.Reason
	default:
		return TurnOutcome{}, false, nil
	}
	at, err := time.Parse(time.RFC3339Nano, rec.Timestamp)
	if err != nil {
		return TurnOutcome{}, false, fmt.Errorf("turn boundary record with an unreadable timestamp %q", rec.Timestamp)
	}
	out.At = at
	return out, true, nil
}

// errorInfo normalises codex_error_info: a string is its own key; an object
// is keyed by its single member, whose http_status_code is the HTTP status.
func errorInfo(raw json.RawMessage) (string, int) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", 0
	}
	var key string
	if json.Unmarshal(raw, &key) == nil {
		return key, 0
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil || len(object) != 1 {
		return string(raw), 0
	}
	for k, v := range object {
		var detail struct {
			HTTPStatusCode int `json:"http_status_code"`
		}
		_ = json.Unmarshal(v, &detail)
		return k, detail.HTTPStatusCode
	}
	return "", 0
}
