package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// WSHubSink bridges contract event envelopes to the websocket hub while
// preserving per-execution ordering and drop policy.
type WSHubSink struct {
	hub    wsHub.HubInterface
	limits contracts.EventBufferLimits
	log    *logrus.Logger

	mu     sync.Mutex
	queues map[uuid.UUID]*executionQueue
	closed map[uuid.UUID]struct{}
}

// NewWSHubSink constructs a WSHubSink. Limits are carried for interface
// completeness; backpressure is handled upstream.
func NewWSHubSink(hub wsHub.HubInterface, log *logrus.Logger, limits contracts.EventBufferLimits) *WSHubSink {
	if limits.Validate() != nil {
		limits = contracts.DefaultEventBufferLimits
	}
	return &WSHubSink{
		hub:    hub,
		limits: limits,
		log:    log,
		queues: make(map[uuid.UUID]*executionQueue),
		closed: make(map[uuid.UUID]struct{}),
	}
}

// Publish pushes the contract envelope into the websocket hub with ordering and
// backpressure handled by the per-execution queue.
func (s *WSHubSink) Publish(_ context.Context, event contracts.EventEnvelope) error {
	if s == nil || s.hub == nil {
		return nil
	}
	if s.isClosed(event.ExecutionID) {
		return nil
	}

	queue := s.ensureQueue(event.ExecutionID)
	if queue == nil {
		return nil
	}
	queue.enqueue(event)
	return nil
}

// Limits returns configured limits (best-effort parity with EventSink).
func (s *WSHubSink) Limits() contracts.EventBufferLimits {
	return s.limits
}

func (s *WSHubSink) ensureQueue(executionID uuid.UUID) *executionQueue {
	if s.isClosed(executionID) {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if queue, ok := s.queues[executionID]; ok {
		return queue
	}

	queue := newExecutionQueue(executionID, s.hub, s.limits, s.log)
	s.queues[executionID] = queue
	go queue.run()
	return queue
}

// CloseExecution drains and closes the per-execution queue to avoid goroutine
// leaks after a run finishes.
func (s *WSHubSink) CloseExecution(executionID uuid.UUID) {
	if s == nil {
		return
	}
	s.mu.Lock()
	queue, ok := s.queues[executionID]
	if ok {
		delete(s.queues, executionID)
	}
	s.closed[executionID] = struct{}{}
	s.mu.Unlock()
	if ok && queue != nil {
		queue.close()
	}
	if s.hub != nil {
		s.hub.CloseExecution(executionID)
	}
}

type executionQueue struct {
	executionID uuid.UUID
	hub         wsHub.HubInterface
	limits      contracts.EventBufferLimits
	log         *logrus.Logger

	mu          sync.Mutex
	cond        *sync.Cond
	events      []contracts.EventEnvelope
	perAttempt  map[string]int
	inflight    int
	dropCounter contracts.DropCounters
	closed      bool
}

func newExecutionQueue(executionID uuid.UUID, hub wsHub.HubInterface, limits contracts.EventBufferLimits, log *logrus.Logger) *executionQueue {
	q := &executionQueue{
		executionID: executionID,
		hub:         hub,
		limits:      limits,
		log:         log,
		perAttempt:  make(map[string]int),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *executionQueue) enqueue(event contracts.EventEnvelope) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}

	// Drop oldest droppable events when execution or attempt buffers are full.
	currentSize := len(q.events) + q.inflight
	if currentSize >= q.limits.PerExecution && isDroppable(event.Kind) {
		if droppedSeq, ok := q.dropOldest(nil, nil); ok && droppedSeq > 0 && q.dropCounter.OldestDropped == 0 {
			q.dropCounter.OldestDropped = droppedSeq
		}
	}
	if event.StepIndex != nil && event.Attempt != nil && isDroppable(event.Kind) {
		key := attemptKey(*event.StepIndex, *event.Attempt)
		if q.perAttempt[key] >= q.limits.PerAttempt {
			if droppedSeq, ok := q.dropOldest(event.StepIndex, event.Attempt); ok && droppedSeq > 0 && q.dropCounter.OldestDropped == 0 {
				q.dropCounter.OldestDropped = droppedSeq
			}
		}
	}

	// Attach drop counters to the event if any were accumulated.
	if q.dropCounter.Dropped > 0 {
		event.Drops = q.dropCounter
	}

	if event.StepIndex != nil && event.Attempt != nil {
		q.perAttempt[attemptKey(*event.StepIndex, *event.Attempt)]++
	}

	q.events = append(q.events, event)
	q.cond.Signal()
}

func (q *executionQueue) run() {
	for {
		q.mu.Lock()
		for len(q.events) == 0 && !q.closed {
			q.cond.Wait()
		}
		if q.closed && len(q.events) == 0 {
			q.mu.Unlock()
			return
		}
		event := q.events[0]
		q.events = q.events[1:]
		q.inflight++
		q.mu.Unlock()

		q.emit(event)

		q.mu.Lock()
		q.inflight--
		if event.StepIndex != nil && event.Attempt != nil {
			key := attemptKey(*event.StepIndex, *event.Attempt)
			if q.perAttempt[key] > 0 {
				q.perAttempt[key]--
			}
		}
		q.mu.Unlock()
	}
}

func (q *executionQueue) emit(event contracts.EventEnvelope) {
	if protoMap, ok := eventToProtoMap(event); ok {
		if event.Payload != nil {
			protoMap["legacy_payload"] = event.Payload
		}
		// Add unified timeline_entry for step events to enable shared Record/Execute UX
		if isStepEvent(event.Kind) {
			timelineEntry := eventToTimelineEntry(event, q.executionID)
			if timelineEntry != nil {
				jsonData, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(timelineEntry)
				if err == nil {
					var timelineEntryMap map[string]any
					if json.Unmarshal(jsonData, &timelineEntryMap) == nil {
						protoMap["timeline_entry"] = timelineEntryMap
					}
				}
			}
		}
		q.hub.BroadcastEnvelope(protoMap)
		return
	}
	q.hub.BroadcastEnvelope(event)
}

func (q *executionQueue) dropOldest(stepIndex, attempt *int) (uint64, bool) {
	for i, ev := range q.events {
		if !isDroppable(ev.Kind) {
			continue
		}
		if stepIndex != nil && attempt != nil {
			if ev.StepIndex == nil || ev.Attempt == nil || *ev.StepIndex != *stepIndex || *ev.Attempt != *attempt {
				continue
			}
		}
		seq := ev.Sequence
		q.events = append(q.events[:i], q.events[i+1:]...)
		if ev.StepIndex != nil && ev.Attempt != nil {
			key := attemptKey(*ev.StepIndex, *ev.Attempt)
			if q.perAttempt[key] > 0 {
				q.perAttempt[key]--
			}
		}
		q.dropCounter.Dropped++
		return seq, true
	}
	return 0, false
}

func attemptKey(stepIndex, attempt int) string {
	return fmt.Sprintf("%d:%d", stepIndex, attempt)
}

func (q *executionQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.events = nil
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (s *WSHubSink) isClosed(executionID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.closed[executionID]
	return ok
}

// isStepEvent returns true if the event kind is a step event (started/completed/failed).
func isStepEvent(kind contracts.EventKind) bool {
	return kind == contracts.EventKindStepStarted ||
		kind == contracts.EventKindStepCompleted ||
		kind == contracts.EventKindStepFailed
}

// eventToTimelineEntry converts an event envelope to a unified TimelineEntry.
// Returns nil if the event doesn't have a step outcome or conversion fails.
func eventToTimelineEntry(ev contracts.EventEnvelope, executionID uuid.UUID) *bastimeline.TimelineEntry {
	var outcome *contracts.StepOutcome

	// Extract outcome from payload
	if asMap, ok := ev.Payload.(map[string]any); ok {
		if out, ok := asMap["outcome"].(contracts.StepOutcome); ok {
			outcome = &out
		}
	} else if out, ok := ev.Payload.(contracts.StepOutcome); ok {
		outcome = &out
	}

	if outcome == nil {
		return nil
	}

	return StepOutcomeToTimelineEntry(*outcome, executionID)
}

func eventToProtoMap(ev contracts.EventEnvelope) (map[string]any, bool) {
	pb, err := convertEventToProto(ev)
	if err != nil || pb == nil {
		return nil, false
	}
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pb)
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false
	}
	if ev.Drops.Dropped > 0 || ev.Drops.OldestDropped > 0 {
		out["drops"] = map[string]any{
			"dropped":        ev.Drops.Dropped,
			"oldest_dropped": ev.Drops.OldestDropped,
		}
	}
	return out, true
}

func convertEventToProto(ev contracts.EventEnvelope) (*bastimeline.TimelineStreamMessage, error) {
	switch ev.Kind {
	case contracts.EventKindExecutionStarted,
		contracts.EventKindExecutionProgress,
		contracts.EventKindExecutionCompleted,
		contracts.EventKindExecutionFailed,
		contracts.EventKindExecutionCancelled:
		status := mapExecutionStatus(ev.Kind, ev.Payload)
		progress := int32(extractInt(ev.Payload, "progress"))
		var errMsg *string
		if msg := extractString(ev.Payload, "error"); msg != "" {
			errMsg = &msg
		}
		return &bastimeline.TimelineStreamMessage{
			Type: bastimeline.TimelineMessageType_TIMELINE_MESSAGE_TYPE_STATUS,
			Payload: &bastimeline.TimelineStreamMessage_Status{
				Status: &bastimeline.TimelineStatusUpdate{
					Id:         ev.ExecutionID.String(),
					Status:     status,
					Progress:   progress,
					EntryCount: int32(ev.Sequence),
					Error:      errMsg,
				},
			},
		}, nil

	case contracts.EventKindStepStarted,
		contracts.EventKindStepCompleted,
		contracts.EventKindStepFailed:
		timelineEntry := eventToTimelineEntry(ev, ev.ExecutionID)
		if timelineEntry == nil {
			return nil, fmt.Errorf("unable to build timeline entry for event kind %s", ev.Kind)
		}
		return &bastimeline.TimelineStreamMessage{
			Type: bastimeline.TimelineMessageType_TIMELINE_MESSAGE_TYPE_ENTRY,
			Payload: &bastimeline.TimelineStreamMessage_Entry{
				Entry: timelineEntry,
			},
		}, nil

	case contracts.EventKindStepTelemetry:
		if telemetryPayload, ok := ev.Payload.(contracts.StepTelemetry); ok {
			if telemetryPayload.Kind == contracts.TelemetryKindHeartbeat {
				return &bastimeline.TimelineStreamMessage{
					Type: bastimeline.TimelineMessageType_TIMELINE_MESSAGE_TYPE_HEARTBEAT,
					Payload: &bastimeline.TimelineStreamMessage_Heartbeat{
						Heartbeat: &bastimeline.TimelineHeartbeat{
							Timestamp: timestamppb.New(telemetryPayload.Timestamp),
							SessionId: ev.ExecutionID.String(),
						},
					},
				}, nil
			}
			// Non-heartbeat telemetry is embedded in TimelineEvent.telemetry
		}
		return nil, nil

	default:
		return nil, nil
	}
}

// buildTimelineEntryFromEnvelope creates a TimelineEntry from a contracts.EventEnvelope.
// This is the unified format for timeline data used in both streaming and batch contexts.
func buildTimelineEntryFromEnvelope(ev contracts.EventEnvelope) *bastimeline.TimelineEntry {
	var outcome *contracts.StepOutcome
	var payloadMap map[string]any
	if asMap, ok := ev.Payload.(map[string]any); ok {
		payloadMap = asMap
		if out, ok := asMap["outcome"].(contracts.StepOutcome); ok {
			outcome = &out
		}
	} else if out, ok := ev.Payload.(contracts.StepOutcome); ok {
		outcome = &out
	}

	// Build entry ID from execution and step
	entryID := fmt.Sprintf("%s-step-%d", ev.ExecutionID.String(), ptrOrZero(ev.StepIndex))

	stepIndex := int32(ptrOrZero(ev.StepIndex))
	entry := &bastimeline.TimelineEntry{
		Id:          entryID,
		SequenceNum: int32(ev.Sequence),
		StepIndex:   &stepIndex,
		Timestamp:   timestampFromTime(outcomeStart(outcome)),
	}

	// Set node_id if available
	nodeID := extractNodeID(outcome, payloadMap)
	if nodeID != "" {
		entry.NodeId = &nodeID
	}

	// Set duration
	durationMs := int32(extractDuration(outcome))
	if durationMs > 0 {
		entry.DurationMs = &durationMs
	}

	// Build ActionDefinition with action type
	stepType := extractStepType(outcome, payloadMap)
	if stepType != "" {
		entry.Action = &basactions.ActionDefinition{
			Type: typeconv.StringToActionType(stepType),
		}
	}

	// Build telemetry
	if outcome != nil {
		entry.Telemetry = buildTelemetryFromOutcome(outcome)
	}

	// Build EventContext with execution origin
	success := ev.Kind != contracts.EventKindStepFailed
	entry.Context = &basbase.EventContext{
		Origin:  &basbase.EventContext_ExecutionId{ExecutionId: ev.ExecutionID.String()},
		Success: &success,
	}

	// Add error to context if present
	if outcome != nil && outcome.Failure != nil && outcome.Failure.Message != "" {
		entry.Context.Error = &outcome.Failure.Message
		if outcome.Failure.Code != "" {
			entry.Context.ErrorCode = &outcome.Failure.Code
		}
	}

	// Add assertion to context if present
	if outcome != nil && outcome.Assertion != nil {
		entry.Context.Assertion = convertAssertionOutcome(outcome.Assertion)
	}

	// Build aggregates for batch data (status, progress, final_url, etc.)
	status := mapStepStatus(ev.Kind)
	progress := int32(extractInt(payloadMap, "progress"))
	entry.Aggregates = &bastimeline.TimelineEntryAggregates{
		Status:   status,
		Progress: &progress,
	}
	if outcome != nil && outcome.FinalURL != "" {
		entry.Aggregates.FinalUrl = &outcome.FinalURL
	}
	if outcome != nil && outcome.DOMSnapshot != nil && outcome.DOMSnapshot.Preview != "" {
		entry.Aggregates.DomSnapshotPreview = &outcome.DOMSnapshot.Preview
	}
	if outcome != nil && outcome.FocusedElement != nil {
		entry.Aggregates.FocusedElement = convertElementFocus(outcome.FocusedElement)
	}

	return entry
}

// buildTelemetryFromOutcome creates ActionTelemetry from a StepOutcome.
func buildTelemetryFromOutcome(outcome *contracts.StepOutcome) *basdomain.ActionTelemetry {
	if outcome == nil {
		return nil
	}

	tel := &basdomain.ActionTelemetry{
		Url: outcome.FinalURL,
	}

	if outcome.ElementBoundingBox != nil {
		tel.ElementBoundingBox = convertBoundingBox(outcome.ElementBoundingBox)
	}

	if outcome.ClickPosition != nil {
		tel.ClickPosition = convertPoint(outcome.ClickPosition)
	}

	if len(outcome.CursorTrail) > 0 {
		points := make([]*basbase.Point, 0, len(outcome.CursorTrail))
		for i := range outcome.CursorTrail {
			points = append(points, convertPoint(outcome.CursorTrail[i].Point))
		}
		tel.CursorTrail = points
	}

	if len(outcome.HighlightRegions) > 0 {
		tel.HighlightRegions = convertHighlightRegions(outcome.HighlightRegions)
	}

	if len(outcome.MaskRegions) > 0 {
		tel.MaskRegions = convertMaskRegions(outcome.MaskRegions)
	}

	if outcome.ZoomFactor != 0 {
		tel.ZoomFactor = &outcome.ZoomFactor
	}

	return tel
}


func mapExecutionStatus(kind contracts.EventKind, payload any) basbase.ExecutionStatus {
	if status := extractString(payload, "status"); status != "" {
		switch strings.ToLower(status) {
		case "pending":
			return basbase.ExecutionStatus_EXECUTION_STATUS_PENDING
		case "running":
			return basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING
		case "completed", "success", "succeeded":
			return basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED
		case "failed", "error":
			return basbase.ExecutionStatus_EXECUTION_STATUS_FAILED
		case "cancelled", "canceled":
			return basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED
		}
	}
	switch kind {
	case contracts.EventKindExecutionStarted:
		return basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case contracts.EventKindExecutionCompleted:
		return basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case contracts.EventKindExecutionFailed:
		return basbase.ExecutionStatus_EXECUTION_STATUS_FAILED
	case contracts.EventKindExecutionCancelled:
		return basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return basbase.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

func mapStepStatus(kind contracts.EventKind) basbase.StepStatus {
	switch kind {
	case contracts.EventKindStepStarted:
		return basbase.StepStatus_STEP_STATUS_RUNNING
	case contracts.EventKindStepCompleted:
		return basbase.StepStatus_STEP_STATUS_COMPLETED
	case contracts.EventKindStepFailed:
		return basbase.StepStatus_STEP_STATUS_FAILED
	default:
		return basbase.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

func extractStepType(outcome *contracts.StepOutcome, payload map[string]any) string {
	if outcome != nil && outcome.StepType != "" {
		return outcome.StepType
	}
	if payload != nil {
		if v, ok := payload["step_type"].(string); ok {
			return v
		}
		if v, ok := payload["stepType"].(string); ok {
			return v
		}
	}
	return ""
}

func extractNodeID(outcome *contracts.StepOutcome, payload map[string]any) string {
	if outcome != nil && outcome.NodeID != "" {
		return outcome.NodeID
	}
	if payload != nil {
		if v, ok := payload["node_id"].(string); ok {
			return v
		}
		if v, ok := payload["nodeId"].(string); ok {
			return v
		}
	}
	return ""
}

func extractInt(payload any, keys ...string) int {
	asMap, ok := payload.(map[string]any)
	if !ok || asMap == nil {
		return 0
	}
	for _, key := range keys {
		if raw, ok := asMap[key]; ok {
			switch v := raw.(type) {
			case int:
				return v
			case int32:
				return int(v)
			case int64:
				return int(v)
			case float64:
				return int(v)
			}
		}
	}
	return 0
}

func extractString(payload any, keys ...string) string {
	asMap, ok := payload.(map[string]any)
	if !ok || asMap == nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := asMap[key]; ok {
			if v, ok := raw.(string); ok {
				return v
			}
		}
	}
	return ""
}

func ptrOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func outcomeStart(outcome *contracts.StepOutcome) time.Time {
	if outcome == nil {
		return time.Time{}
	}
	return outcome.StartedAt
}

func extractDuration(outcome *contracts.StepOutcome) int {
	if outcome == nil {
		return 0
	}
	if outcome.DurationMs > 0 {
		return outcome.DurationMs
	}
	if !outcome.StartedAt.IsZero() && outcome.CompletedAt != nil {
		return int(outcome.CompletedAt.Sub(outcome.StartedAt).Milliseconds())
	}
	return 0
}

func timestampFromTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func timestampFromPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func structpbMetrics(values map[string]any) map[string]*structpb.Value {
	if len(values) == 0 {
		return nil
	}
	result, err := structpb.NewStruct(values)
	if err != nil {
		return nil
	}
	return result.Fields
}

func jsonMetrics(values map[string]any) map[string]*commonv1.JsonValue {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]*commonv1.JsonValue, len(values))
	for k, v := range values {
		if jsonVal := toJsonValue(v); jsonVal != nil {
			result[k] = jsonVal
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func toJsonValue(v any) *commonv1.JsonValue {
	switch val := v.(type) {
	case nil:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case uint:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := make(map[string]*commonv1.JsonValue, len(val))
		for key, value := range val {
			if nested := toJsonValue(value); nested != nil {
				obj[key] = nested
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{
			ObjectValue: &commonv1.JsonObject{Fields: obj},
		}}
	case []any:
		items := make([]*commonv1.JsonValue, 0, len(val))
		for _, item := range val {
			if nested := toJsonValue(item); nested != nil {
				items = append(items, nested)
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{
			ListValue: &commonv1.JsonList{Values: items},
		}}
	default:
		return nil
	}
}

func convertAssertionOutcome(assertion *contracts.AssertionOutcome) *basbase.AssertionResult {
	if assertion == nil {
		return nil
	}
	result := &basbase.AssertionResult{
		Mode:          typeconv.StringToAssertionMode(assertion.Mode),
		Selector:      assertion.Selector,
		Success:       assertion.Success,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
	}
	if assertion.Message != "" {
		result.Message = &assertion.Message
	}
	if assertion.Expected != nil {
		result.Expected = toJsonValue(assertion.Expected)
	}
	if assertion.Actual != nil {
		result.Actual = toJsonValue(assertion.Actual)
	}
	return result
}

func convertBoundingBox(b *contracts.BoundingBox) *basbase.BoundingBox {
	if b == nil {
		return nil
	}
	return &basbase.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

func convertPoint(pt *contracts.Point) *basbase.Point {
	if pt == nil {
		return nil
	}
	return &basbase.Point{
		X: pt.X,
		Y: pt.Y,
	}
}

func convertElementFocus(f *contracts.ElementFocus) *bastimeline.ElementFocus {
	if f == nil {
		return nil
	}
	return &bastimeline.ElementFocus{
		Selector:    f.Selector,
		BoundingBox: convertBoundingBox(f.BoundingBox),
	}
}

func convertHighlightRegions(regions []*contracts.HighlightRegion) []*basdomain.HighlightRegion {
	if len(regions) == 0 {
		return nil
	}
	out := make([]*basdomain.HighlightRegion, 0, len(regions))
	for _, r := range regions {
		if r == nil {
			continue
		}
		region := &basdomain.HighlightRegion{
			Selector:       r.Selector,
			BoundingBox:    convertBoundingBox(r.BoundingBox),
			Padding:        r.Padding,
			HighlightColor: r.HighlightColor,
			CustomRgba:     r.CustomRgba,
		}
		out = append(out, region)
	}
	return out
}

func convertMaskRegions(regions []*contracts.MaskRegion) []*basdomain.MaskRegion {
	if len(regions) == 0 {
		return nil
	}
	out := make([]*basdomain.MaskRegion, 0, len(regions))
	for _, r := range regions {
		if r == nil {
			continue
		}
		out = append(out, &basdomain.MaskRegion{
			Selector:    r.Selector,
			BoundingBox: convertBoundingBox(r.BoundingBox),
			Opacity:     r.Opacity,
		})
	}
	return out
}

func convertStructValue(v any) *structpb.Value {
	if v == nil {
		return nil
	}
	val, err := structpb.NewValue(v)
	if err != nil {
		return nil
	}
	return val
}
