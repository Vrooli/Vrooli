// Package performance provides types and utilities for debug performance mode.
//
// Debug performance mode instruments the frame streaming pipeline to collect
// timing data at each stage (capture, compare, send, receive, broadcast).
// This enables identification of bottlenecks and tracking of optimization progress.
//
// # Wire Format Stability
//
// These types define the JSON contract shared with playwright-driver/src/performance/types.ts.
// - Adding new fields with omitempty is safe
// - Removing or renaming fields is a BREAKING CHANGE
// - Changes require coordinated updates to the TypeScript types
package performance

import "time"

// FrameTimings captures timing data for a single frame.
// Collected at each stage of the streaming pipeline.
type FrameTimings struct {
	// FrameID is a unique identifier for correlation: "{sessionId}-{sequenceNum}"
	FrameID string `json:"frame_id"`

	// SessionID is the session this frame belongs to
	SessionID string `json:"session_id"`

	// SequenceNum is the frame sequence number within session (monotonically increasing)
	SequenceNum int `json:"sequence_num"`

	// Timestamp is the ISO 8601 timestamp when capture started
	Timestamp time.Time `json:"timestamp"`

	// Driver-side timings (all in milliseconds)

	// DriverCaptureMs is the time to capture screenshot from Playwright
	DriverCaptureMs float64 `json:"capture_ms"`

	// DriverCompareMs is the time to compare with previous frame buffer
	DriverCompareMs float64 `json:"compare_ms"`

	// DriverWsSendMs is the time to send frame over WebSocket to API
	DriverWsSendMs float64 `json:"ws_send_ms"`

	// DriverTotalMs is the total driver-side processing time
	DriverTotalMs float64 `json:"driver_total_ms"`

	// API-side timings (added by Go API)

	// APIReceiveMs is the time to receive frame from driver WebSocket
	APIReceiveMs float64 `json:"api_receive_ms,omitempty"`

	// APIBroadcastMs is the time to broadcast frame to all subscribed clients
	APIBroadcastMs float64 `json:"api_broadcast_ms,omitempty"`

	// APITotalMs is the total API-side processing time
	APITotalMs float64 `json:"api_total_ms,omitempty"`

	// Frame metadata

	// FrameBytes is the frame size in bytes (JPEG payload)
	FrameBytes int `json:"frame_bytes"`

	// Skipped indicates whether frame was skipped (identical to previous)
	Skipped bool `json:"skipped"`
}

// FrameHeader is the binary header sent with each frame when perf mode is enabled.
// This is a subset of FrameTimings - just the driver-side data.
//
// Wire format:
// [4 bytes: header length (uint32 big-endian)]
// [N bytes: JSON of FrameHeader]
// [remaining: JPEG data]
type FrameHeader struct {
	FrameID    string  `json:"frame_id"`
	CaptureMs  float64 `json:"capture_ms"`
	CompareMs  float64 `json:"compare_ms"`
	WsSendMs   float64 `json:"ws_send_ms"`
	FrameBytes int     `json:"frame_bytes"`
}

// FrameStatsAggregated holds percentile statistics over a window of frames.
type FrameStatsAggregated struct {
	// SessionID is the session these stats belong to
	SessionID string `json:"session_id"`

	// WindowStartTime is when this stats window started
	WindowStartTime time.Time `json:"window_start_time"`

	// WindowDurationMs is the duration of the stats window in milliseconds
	WindowDurationMs int64 `json:"window_duration_ms"`

	// FrameCount is the total frames captured in this window
	FrameCount int `json:"frame_count"`

	// SkippedCount is frames skipped due to unchanged content
	SkippedCount int `json:"skipped_count"`

	// Capture timing percentiles (milliseconds)

	// CaptureP50Ms is the 50th percentile (median) capture time
	CaptureP50Ms float64 `json:"capture_p50_ms"`

	// CaptureP90Ms is the 90th percentile capture time
	CaptureP90Ms float64 `json:"capture_p90_ms"`

	// CaptureP99Ms is the 99th percentile capture time
	CaptureP99Ms float64 `json:"capture_p99_ms"`

	// CaptureMaxMs is the maximum capture time observed
	CaptureMaxMs float64 `json:"capture_max_ms"`

	// End-to-end timing percentiles (driver capture start -> API broadcast complete)

	// E2EP50Ms is the 50th percentile end-to-end time
	E2EP50Ms float64 `json:"e2e_p50_ms"`

	// E2EP90Ms is the 90th percentile end-to-end time
	E2EP90Ms float64 `json:"e2e_p90_ms"`

	// E2EP99Ms is the 99th percentile end-to-end time
	E2EP99Ms float64 `json:"e2e_p99_ms"`

	// E2EMaxMs is the maximum end-to-end time observed
	E2EMaxMs float64 `json:"e2e_max_ms"`

	// Throughput metrics

	// ActualFps is the actual frames per second achieved
	ActualFps float64 `json:"actual_fps"`

	// TargetFps is the target FPS configured for the session
	TargetFps int `json:"target_fps"`

	// AvgFrameBytes is the average frame size in bytes
	AvgFrameBytes int `json:"avg_frame_bytes"`

	// BandwidthBytesPerSec is the bandwidth in bytes per second
	BandwidthBytesPerSec int64 `json:"bandwidth_bytes_per_sec"`

	// Bottleneck identification

	// PrimaryBottleneck is the primary bottleneck identified
	PrimaryBottleneck BottleneckType `json:"primary_bottleneck"`

	// BottleneckDescription is a human-readable description of the bottleneck
	BottleneckDescription string `json:"bottleneck_description"`
}

// BottleneckType represents possible bottleneck types in the streaming pipeline.
type BottleneckType string

const (
	// BottleneckCapture indicates screenshot capture is slow
	BottleneckCapture BottleneckType = "capture"

	// BottleneckEncode indicates JPEG encoding is slow (rare with Playwright)
	BottleneckEncode BottleneckType = "encode"

	// BottleneckNetwork indicates network/WebSocket is slow
	BottleneckNetwork BottleneckType = "network"

	// BottleneckDecode indicates client-side decoding is slow
	BottleneckDecode BottleneckType = "decode"

	// BottleneckDraw indicates canvas rendering is slow
	BottleneckDraw BottleneckType = "draw"

	// BottleneckNone indicates no significant bottleneck detected
	BottleneckNone BottleneckType = "none"
)

// PerfStatsMessage is the WebSocket message type for performance stats broadcast.
type PerfStatsMessage struct {
	Type      string               `json:"type"` // Always "perf_stats"
	SessionID string               `json:"session_id"`
	Stats     FrameStatsAggregated `json:"stats"`
}
