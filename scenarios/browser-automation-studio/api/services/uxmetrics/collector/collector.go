// Package collector implements the UX metrics data collection layer.
// It provides an EventSink decorator that passively captures interaction
// data from workflow executions without modifying the existing event pipeline.
package collector

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics"
	"github.com/vrooli/browser-automation-studio/services/uxmetrics/contracts"
)

// Collector implements both automation/events.Sink and uxmetrics.Collector.
// It decorates the existing EventSink to passively capture UX data.
type Collector struct {
	delegate autoevents.Sink
	repo     uxmetrics.Repository

	// In-flight cursor paths (flushed on step completion)
	cursorBuffers sync.Map // map[string][]contracts.TimedPoint (key: executionID:stepIndex)
}

// NewCollector creates a collector that wraps an existing EventSink.
func NewCollector(delegate autoevents.Sink, repo uxmetrics.Repository) *Collector {
	return &Collector{
		delegate: delegate,
		repo:     repo,
	}
}

// Publish implements automation/events.Sink.
// This is the key integration point - every event flows through here.
func (c *Collector) Publish(ctx context.Context, event autocontracts.EventEnvelope) error {
	// Extract UX-relevant data from events
	switch event.Kind {
	case autocontracts.EventKindStepCompleted:
		if payload, ok := event.Payload.(*autocontracts.StepOutcome); ok {
			c.onStepCompleted(ctx, event.ExecutionID, payload)
		}
	case autocontracts.EventKindStepFailed:
		if payload, ok := event.Payload.(*autocontracts.StepOutcome); ok {
			c.onStepFailed(ctx, event.ExecutionID, payload)
		}
	case autocontracts.EventKindExecutionCompleted, autocontracts.EventKindExecutionFailed:
		// Flush any remaining cursor buffers for this execution
		_ = c.FlushExecution(ctx, event.ExecutionID)
	}

	// Always delegate to underlying sink
	return c.delegate.Publish(ctx, event)
}

// Limits delegates to the underlying sink.
func (c *Collector) Limits() autocontracts.EventBufferLimits {
	return c.delegate.Limits()
}

// OnStepOutcome implements uxmetrics.Collector for direct calls.
func (c *Collector) OnStepOutcome(ctx context.Context, executionID uuid.UUID, outcome uxmetrics.StepOutcomeData) error {
	// Build cursor path from trail
	if len(outcome.CursorTrail) > 0 {
		trail := make([]autocontracts.CursorPosition, len(outcome.CursorTrail))
		for i, p := range outcome.CursorTrail {
			trail[i] = autocontracts.CursorPosition{
				Point: autocontracts.Point{X: p.X, Y: p.Y},
			}
		}
		path := c.buildCursorPath(outcome.StepIndex, trail, outcome.StartedAt, outcome.CompletedAt)
		if err := c.repo.SaveCursorPath(ctx, executionID, path); err != nil {
			return err
		}
	}

	// Create interaction trace
	trace := &contracts.InteractionTrace{
		ID:          uuid.New(),
		ExecutionID: executionID,
		StepIndex:   outcome.StepIndex,
		ActionType:  mapStepTypeToAction(outcome.StepType),
		ElementID:   outcome.NodeID,
		Position:    outcome.Position,
		Timestamp:   outcome.StartedAt,
		DurationMs:  int64(outcome.DurationMs),
		Success:     outcome.Success,
	}
	return c.repo.SaveInteractionTrace(ctx, trace)
}

// OnCursorUpdate implements uxmetrics.Collector.
func (c *Collector) OnCursorUpdate(ctx context.Context, executionID uuid.UUID, stepIndex int, point contracts.TimedPoint) error {
	key := cursorBufferKey(executionID, stepIndex)

	val, _ := c.cursorBuffers.LoadOrStore(key, &[]contracts.TimedPoint{})
	buffer := val.(*[]contracts.TimedPoint)
	*buffer = append(*buffer, point)

	return nil
}

// FlushExecution implements uxmetrics.Collector.
func (c *Collector) FlushExecution(ctx context.Context, executionID uuid.UUID) error {
	// Clean up any remaining cursor buffers for this execution
	prefix := executionID.String() + ":"
	c.cursorBuffers.Range(func(key, value any) bool {
		if k, ok := key.(string); ok && len(k) > 36 && k[:37] == prefix {
			c.cursorBuffers.Delete(key)
		}
		return true
	})
	return nil
}

func (c *Collector) onStepCompleted(ctx context.Context, executionID uuid.UUID, outcome *autocontracts.StepOutcome) {
	// Convert cursor trail to cursor path
	if len(outcome.CursorTrail) > 0 {
		path := c.buildCursorPath(outcome.StepIndex, outcome.CursorTrail, outcome.StartedAt, outcome.CompletedAt)
		_ = c.repo.SaveCursorPath(ctx, executionID, path)
	}

	// Create interaction trace
	var selector string
	if outcome.FocusedElement != nil {
		selector = outcome.FocusedElement.Selector
	}

	trace := &contracts.InteractionTrace{
		ID:          uuid.New(),
		ExecutionID: executionID,
		StepIndex:   outcome.StepIndex,
		ActionType:  mapStepTypeToAction(outcome.StepType),
		ElementID:   outcome.NodeID,
		Selector:    selector,
		Position:    convertPoint(outcome.ClickPosition),
		Timestamp:   outcome.StartedAt,
		DurationMs:  int64(outcome.DurationMs),
		Success:     outcome.Success,
	}
	_ = c.repo.SaveInteractionTrace(ctx, trace)
}

func (c *Collector) onStepFailed(ctx context.Context, executionID uuid.UUID, outcome *autocontracts.StepOutcome) {
	// Same as completed but with Success=false (already set in outcome)
	c.onStepCompleted(ctx, executionID, outcome)
}

func (c *Collector) buildCursorPath(stepIndex int, trail []autocontracts.CursorPosition, start time.Time, end *time.Time) *contracts.CursorPath {
	points := make([]contracts.TimedPoint, len(trail))
	totalDist := 0.0
	maxSpeed := 0.0
	hesitations := 0

	for i, pos := range trail {
		recordedAt := pos.RecordedAt
		if recordedAt.IsZero() {
			// Fallback: use start time + elapsed
			recordedAt = start.Add(time.Duration(pos.ElapsedMs) * time.Millisecond)
		}

		points[i] = contracts.TimedPoint{
			X:         pos.Point.X,
			Y:         pos.Point.Y,
			Timestamp: recordedAt,
		}

		if i > 0 {
			dx := pos.Point.X - trail[i-1].Point.X
			dy := pos.Point.Y - trail[i-1].Point.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			totalDist += dist

			prevTime := trail[i-1].RecordedAt
			if prevTime.IsZero() {
				prevTime = start.Add(time.Duration(trail[i-1].ElapsedMs) * time.Millisecond)
			}
			dt := recordedAt.Sub(prevTime).Milliseconds()
			if dt > 0 {
				speed := dist / float64(dt)
				if speed > maxSpeed {
					maxSpeed = speed
				}
			}
			if dt > 200 { // Hesitation threshold: >200ms pause
				hesitations++
			}
		}
	}

	// Calculate directness (straight-line efficiency)
	directDist := 0.0
	if len(points) >= 2 {
		first, last := points[0], points[len(points)-1]
		dx := last.X - first.X
		dy := last.Y - first.Y
		directDist = math.Sqrt(dx*dx + dy*dy)
	}

	directness := 1.0
	if totalDist > 0 && directDist > 0 {
		directness = directDist / totalDist
		if directness > 1.0 {
			directness = 1.0 // Clamp to valid range
		}
	}

	durationMs := int64(0)
	if end != nil {
		durationMs = end.Sub(start).Milliseconds()
	} else if len(points) >= 2 {
		durationMs = points[len(points)-1].Timestamp.Sub(points[0].Timestamp).Milliseconds()
	}

	avgSpeed := 0.0
	if durationMs > 0 {
		avgSpeed = totalDist / float64(durationMs)
	}

	return &contracts.CursorPath{
		StepIndex:       stepIndex,
		Points:          points,
		TotalDistancePx: totalDist,
		DirectDistance:  directDist,
		DurationMs:      durationMs,
		Directness:      directness,
		ZigZagScore:     1.0 - directness, // Inverse of directness
		AverageSpeed:    avgSpeed,
		MaxSpeed:        maxSpeed,
		Hesitations:     hesitations,
	}
}

func cursorBufferKey(executionID uuid.UUID, stepIndex int) string {
	return fmt.Sprintf("%s:%d", executionID.String(), stepIndex)
}

func mapStepTypeToAction(stepType string) contracts.ActionType {
	switch stepType {
	case "click":
		return contracts.ActionClick
	case "type", "input", "fill":
		return contracts.ActionType_
	case "scroll":
		return contracts.ActionScroll
	case "navigate", "navigation", "goto":
		return contracts.ActionNavigation
	case "wait":
		return contracts.ActionWait
	case "hover":
		return contracts.ActionHover
	case "drag", "dragdrop":
		return contracts.ActionDrag
	default:
		return contracts.ActionClick
	}
}

func convertPoint(p *autocontracts.Point) *contracts.Point {
	if p == nil {
		return nil
	}
	return &contracts.Point{X: p.X, Y: p.Y}
}

// Compile-time interface checks
var (
	_ autoevents.Sink    = (*Collector)(nil)
	_ uxmetrics.Collector = (*Collector)(nil)
)
