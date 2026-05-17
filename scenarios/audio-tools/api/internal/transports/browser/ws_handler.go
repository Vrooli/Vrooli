// Package browser previously hosted a thin scaffold around the legacy
// voice.Service.HandleStreamWS path. The streaming-STT decoupling plan
// retired both: the active /api/v1/voice/stream route now lives in
// handlers/stt/stream_ws.go::StreamWSHandler and routes through the
// shared Segmenter + StrategySelector pipeline. This file is kept as a
// placeholder for future browser-specific transport concerns
// (e.g. WebM transcoding, MediaRecorder framing) that don't belong on
// the strategy axis.
package browser

import (
	"context"
	"sync/atomic"
)

// SessionIDFromContext returns the active browser-voice session ID, if
// any. Retained for downstream consumers that previously called this
// helper; the new transport does not inject session IDs on its own.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(sessionIDKey{}).(string)
	return v, ok && v != ""
}

type sessionIDKey struct{}

var activeSessionsGauge atomic.Int64

// ActiveSessions returns the current in-flight session count.
func ActiveSessions() int64 { return activeSessionsGauge.Load() }
