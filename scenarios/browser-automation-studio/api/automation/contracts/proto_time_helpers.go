// Package contracts provides time helper functions for working with proto timestamps.
//
// These helpers facilitate the gradual migration from native Go types (with time.Time)
// to proto-native types (with *timestamppb.Timestamp). New code should prefer using
// proto types directly with these helpers, rather than the native Go types.
//
// Migration path:
//  1. Use GetProtoTime/SetProtoTime for reading/writing timestamp fields
//  2. Gradually update call sites to use proto types directly
//  3. Eventually deprecate native Go types in contracts.go
package contracts

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// =============================================================================
// TIMESTAMP CONVERSION HELPERS
// =============================================================================
// These functions provide a clean API for converting between time.Time and
// *timestamppb.Timestamp, handling nil cases appropriately.

// TimestampToTime safely converts a proto timestamp to time.Time.
// Returns zero time for nil input.
func TimestampToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// TimestampToTimePtr safely converts a proto timestamp to *time.Time.
// Returns nil for nil input.
func TimestampToTimePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

// TimeToTimestamp safely converts a time.Time to proto timestamp.
// Returns nil for zero time.
func TimeToTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// TimePtrToTimestamp safely converts a *time.Time to proto timestamp.
// Returns nil for nil input.
func TimePtrToTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

// NowTimestamp returns the current time as a proto timestamp.
// This is a convenience wrapper for timestamppb.Now().
func NowTimestamp() *timestamppb.Timestamp {
	return timestamppb.Now()
}

// =============================================================================
// PROTO TYPE ACCESSORS
// =============================================================================
// These functions provide safe access to timestamp fields on proto types,
// returning native Go time.Time values for code that hasn't migrated yet.

// GetStepOutcomeStartedAt returns the StartedAt field of a ProtoStepOutcome as time.Time.
func GetStepOutcomeStartedAt(outcome *ProtoStepOutcome) time.Time {
	if outcome == nil {
		return time.Time{}
	}
	return TimestampToTime(outcome.StartedAt)
}

// GetStepOutcomeCompletedAt returns the CompletedAt field of a ProtoStepOutcome as *time.Time.
func GetStepOutcomeCompletedAt(outcome *ProtoStepOutcome) *time.Time {
	if outcome == nil {
		return nil
	}
	return TimestampToTimePtr(outcome.CompletedAt)
}

// GetStepFailureOccurredAt returns the OccurredAt field of a ProtoStepFailure as *time.Time.
func GetStepFailureOccurredAt(failure *ProtoStepFailure) *time.Time {
	if failure == nil {
		return nil
	}
	return TimestampToTimePtr(failure.OccurredAt)
}

// GetScreenshotCaptureTime returns the CaptureTime field of a ProtoDriverScreenshot as time.Time.
func GetScreenshotCaptureTime(screenshot *ProtoDriverScreenshot) time.Time {
	if screenshot == nil {
		return time.Time{}
	}
	return TimestampToTime(screenshot.CaptureTime)
}

// GetDOMSnapshotCollectedAt returns the CollectedAt field of a ProtoDOMSnapshot as time.Time.
func GetDOMSnapshotCollectedAt(snapshot *ProtoDOMSnapshot) time.Time {
	if snapshot == nil {
		return time.Time{}
	}
	return TimestampToTime(snapshot.CollectedAt)
}

// =============================================================================
// PROTO TYPE SETTERS
// =============================================================================
// These functions set timestamp fields on proto types from native Go time.Time values.

// SetStepOutcomeStartedAt sets the StartedAt field of a ProtoStepOutcome from time.Time.
func SetStepOutcomeStartedAt(outcome *ProtoStepOutcome, t time.Time) {
	if outcome == nil {
		return
	}
	outcome.StartedAt = TimeToTimestamp(t)
}

// SetStepOutcomeCompletedAt sets the CompletedAt field of a ProtoStepOutcome from *time.Time.
func SetStepOutcomeCompletedAt(outcome *ProtoStepOutcome, t *time.Time) {
	if outcome == nil {
		return
	}
	outcome.CompletedAt = TimePtrToTimestamp(t)
}

// SetStepFailureOccurredAt sets the OccurredAt field of a ProtoStepFailure from *time.Time.
func SetStepFailureOccurredAt(failure *ProtoStepFailure, t *time.Time) {
	if failure == nil {
		return
	}
	failure.OccurredAt = TimePtrToTimestamp(t)
}

// SetScreenshotCaptureTime sets the CaptureTime field of a ProtoDriverScreenshot from time.Time.
func SetScreenshotCaptureTime(screenshot *ProtoDriverScreenshot, t time.Time) {
	if screenshot == nil {
		return
	}
	screenshot.CaptureTime = TimeToTimestamp(t)
}

// SetDOMSnapshotCollectedAt sets the CollectedAt field of a ProtoDOMSnapshot from time.Time.
func SetDOMSnapshotCollectedAt(snapshot *ProtoDOMSnapshot, t time.Time) {
	if snapshot == nil {
		return
	}
	snapshot.CollectedAt = TimeToTimestamp(t)
}
