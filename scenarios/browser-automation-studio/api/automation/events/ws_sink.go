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
	wsHub "github.com/vrooli/browser-automation-studio/websocket"
	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

func convertEventToProto(ev contracts.EventEnvelope) (*browser_automation_studio_v1.ExecutionEventEnvelope, error) {
	pb := &browser_automation_studio_v1.ExecutionEventEnvelope{
		SchemaVersion:  ev.SchemaVersion,
		PayloadVersion: ev.PayloadVersion,
		Kind:           mapEventKind(ev.Kind),
		ExecutionId:    ev.ExecutionID.String(),
		WorkflowId:     ev.WorkflowID.String(),
		Timestamp:      timestamppb.New(ev.Timestamp),
	}
	if ev.StepIndex != nil {
		pb.StepIndex = proto.Int32(int32(*ev.StepIndex))
	}
	if ev.Attempt != nil {
		pb.Attempt = proto.Int32(int32(*ev.Attempt))
	}
	if ev.Sequence > 0 {
		pb.Sequence = proto.Int64(int64(ev.Sequence))
	}

	switch ev.Kind {
	case contracts.EventKindExecutionStarted,
		contracts.EventKindExecutionProgress,
		contracts.EventKindExecutionCompleted,
		contracts.EventKindExecutionFailed,
		contracts.EventKindExecutionCancelled:
		status := mapExecutionStatus(ev.Kind, ev.Payload)
		progress := int32(extractInt(ev.Payload, "progress"))
		currentStep := extractString(ev.Payload, "current_step", "currentStep")
		var errMsg *string
		if msg := extractString(ev.Payload, "error"); msg != "" {
			errMsg = &msg
		}
		pb.Payload = &browser_automation_studio_v1.ExecutionEventEnvelope_StatusUpdate{
			StatusUpdate: &browser_automation_studio_v1.StatusUpdateEvent{
				Status:   status,
				Progress: progress,
				CurrentStep: func() *string {
					if currentStep == "" {
						return nil
					}
					return &currentStep
				}(),
				Error: errMsg,
				OccurredAt: func() *timestamppb.Timestamp {
					if !ev.Timestamp.IsZero() {
						return timestamppb.New(ev.Timestamp)
					}
					return nil
				}(),
			},
		}

	case contracts.EventKindStepStarted,
		contracts.EventKindStepCompleted,
		contracts.EventKindStepFailed:
		frame := buildTimelineFrame(ev)
		if frame == nil {
			return nil, fmt.Errorf("unable to build timeline frame for event kind %s", ev.Kind)
		}
		pb.Payload = &browser_automation_studio_v1.ExecutionEventEnvelope_TimelineFrame{
			TimelineFrame: &browser_automation_studio_v1.TimelineFrameEvent{Frame: frame},
		}

	case contracts.EventKindStepTelemetry:
		if telemetryPayload, ok := ev.Payload.(contracts.StepTelemetry); ok {
			if telemetryPayload.Kind == contracts.TelemetryKindHeartbeat {
				progress := int32(0)
				metrics := map[string]any{}
				if telemetryPayload.Heartbeat != nil {
					progress = int32(telemetryPayload.Heartbeat.Progress)
					if telemetryPayload.Heartbeat.Message != "" {
						metrics["message"] = telemetryPayload.Heartbeat.Message
					}
				}
				if telemetryPayload.ElapsedMs > 0 {
					metrics["elapsed_ms"] = telemetryPayload.ElapsedMs
				}
				typedMetrics := jsonMetrics(metrics)
				pb.Payload = &browser_automation_studio_v1.ExecutionEventEnvelope_Heartbeat{
					Heartbeat: &browser_automation_studio_v1.HeartbeatEvent{
						ReceivedAt: timestamppb.New(telemetryPayload.Timestamp),
						Progress:   progress,
						Metrics:    structpbMetrics(metrics),
						MetricsTyped: func() map[string]*browser_automation_studio_v1.JsonValue {
							if len(typedMetrics) == 0 {
								return nil
							}
							return typedMetrics
						}(),
					},
				}
			} else {
				typedMetrics := jsonMetrics(nil)
				pb.Payload = &browser_automation_studio_v1.ExecutionEventEnvelope_Telemetry{
					Telemetry: &browser_automation_studio_v1.TelemetryEvent{
						Metrics: structpbMetrics(nil),
						MetricsTyped: func() map[string]*browser_automation_studio_v1.JsonValue {
							if len(typedMetrics) == 0 {
								return nil
							}
							return typedMetrics
						}(),
						RecordedAt: timestamppb.New(telemetryPayload.Timestamp),
					},
				}
			}
		}

	default:
		// Unsupported kinds fall back to the legacy envelope.
	}

	return pb, nil
}

func buildTimelineFrame(ev contracts.EventEnvelope) *browser_automation_studio_v1.TimelineFrame {
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

	frame := &browser_automation_studio_v1.TimelineFrame{
		StepIndex: int32(ptrOrZero(ev.StepIndex)),
		Status:    mapStepStatus(ev.Kind),
		StepType:  mapStepType(extractStepType(outcome, payloadMap)),
		NodeId:    extractNodeID(outcome, payloadMap),
		Success:   ev.Kind != contracts.EventKindStepFailed,
		Progress:  int32(extractInt(payloadMap, "progress")),
		StartedAt: timestampFromTime(outcomeStart(outcome)),
		CompletedAt: func() *timestamppb.Timestamp {
			if outcome != nil && outcome.CompletedAt != nil {
				return timestampFromPtr(outcome.CompletedAt)
			}
			return nil
		}(),
		DurationMs: int32(extractDuration(outcome)),
		FinalUrl: func() string {
			if outcome != nil {
				return outcome.FinalURL
			}
			return ""
		}(),
	}

	if outcome != nil {
		frame.Assertion = convertAssertionOutcome(outcome.Assertion)
		frame.ElementBoundingBox = convertBoundingBox(outcome.ElementBoundingBox)
		frame.ClickPosition = convertPoint(outcome.ClickPosition)
		frame.FocusedElement = convertElementFocus(outcome.FocusedElement)
		frame.HighlightRegions = convertHighlightRegions(outcome.HighlightRegions)
		frame.MaskRegions = convertMaskRegions(outcome.MaskRegions)
		frame.ZoomFactor = outcome.ZoomFactor
		if len(outcome.CursorTrail) > 0 {
			points := make([]*browser_automation_studio_v1.Point, 0, len(outcome.CursorTrail))
			for _, trail := range outcome.CursorTrail {
				points = append(points, convertPoint(&trail.Point))
			}
			frame.CursorTrail = points
		}
		if outcome.DOMSnapshot != nil && outcome.DOMSnapshot.Preview != "" {
			frame.DomSnapshotPreview = outcome.DOMSnapshot.Preview
		}
	}

	if outcome != nil && outcome.Failure != nil && outcome.Failure.Message != "" {
		errMsg := outcome.Failure.Message
		frame.Error = &errMsg
	}
	return frame
}

func mapEventKind(kind contracts.EventKind) browser_automation_studio_v1.EventKind {
	switch kind {
	case contracts.EventKindExecutionStarted,
		contracts.EventKindExecutionProgress,
		contracts.EventKindExecutionCompleted,
		contracts.EventKindExecutionFailed,
		contracts.EventKindExecutionCancelled:
		return browser_automation_studio_v1.EventKind_EVENT_KIND_STATUS_UPDATE
	case contracts.EventKindStepStarted,
		contracts.EventKindStepCompleted,
		contracts.EventKindStepFailed:
		return browser_automation_studio_v1.EventKind_EVENT_KIND_TIMELINE_FRAME
	case contracts.EventKindStepTelemetry:
		return browser_automation_studio_v1.EventKind_EVENT_KIND_TELEMETRY
	default:
		return browser_automation_studio_v1.EventKind_EVENT_KIND_UNSPECIFIED
	}
}

func mapExecutionStatus(kind contracts.EventKind, payload any) browser_automation_studio_v1.ExecutionStatus {
	if status := extractString(payload, "status"); status != "" {
		switch strings.ToLower(status) {
		case "pending":
			return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_PENDING
		case "running":
			return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_RUNNING
		case "completed", "success", "succeeded":
			return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
		case "failed", "error":
			return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_FAILED
		case "cancelled", "canceled":
			return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_CANCELLED
		}
	}
	switch kind {
	case contracts.EventKindExecutionStarted:
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case contracts.EventKindExecutionCompleted:
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case contracts.EventKindExecutionFailed:
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_FAILED
	case contracts.EventKindExecutionCancelled:
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

func mapStepStatus(kind contracts.EventKind) browser_automation_studio_v1.StepStatus {
	switch kind {
	case contracts.EventKindStepStarted:
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_RUNNING
	case contracts.EventKindStepCompleted:
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED
	case contracts.EventKindStepFailed:
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_FAILED
	default:
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

func mapStepType(stepType string) browser_automation_studio_v1.StepType {
	switch strings.ToLower(strings.TrimSpace(stepType)) {
	case "navigate":
		return browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE
	case "click":
		return browser_automation_studio_v1.StepType_STEP_TYPE_CLICK
	case "input", "type":
		return browser_automation_studio_v1.StepType_STEP_TYPE_INPUT
	case "assert":
		return browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT
	case "subflow":
		return browser_automation_studio_v1.StepType_STEP_TYPE_SUBFLOW
	case "custom", "wait", "extract", "screenshot", "scroll", "select", "hover", "keyboard", "condition", "loop":
		return browser_automation_studio_v1.StepType_STEP_TYPE_CUSTOM
	default:
		return browser_automation_studio_v1.StepType_STEP_TYPE_UNSPECIFIED
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

func jsonMetrics(values map[string]any) map[string]*browser_automation_studio_v1.JsonValue {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]*browser_automation_studio_v1.JsonValue, len(values))
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

func toJsonValue(v any) *browser_automation_studio_v1.JsonValue {
	switch val := v.(type) {
	case nil:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case bool:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: val}}
	case uint:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint32:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint64:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(val)}}
	case float32:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := make(map[string]*browser_automation_studio_v1.JsonValue, len(val))
		for key, value := range val {
			if nested := toJsonValue(value); nested != nil {
				obj[key] = nested
			}
		}
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_ObjectValue{
			ObjectValue: &browser_automation_studio_v1.JsonObject{Fields: obj},
		}}
	case []any:
		items := make([]*browser_automation_studio_v1.JsonValue, 0, len(val))
		for _, item := range val {
			if nested := toJsonValue(item); nested != nil {
				items = append(items, nested)
			}
		}
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_ListValue{
			ListValue: &browser_automation_studio_v1.JsonList{Values: items},
		}}
	default:
		return nil
	}
}

func convertAssertionOutcome(assertion *contracts.AssertionOutcome) *browser_automation_studio_v1.AssertionOutcome {
	if assertion == nil {
		return nil
	}
	return &browser_automation_studio_v1.AssertionOutcome{
		Mode:          assertion.Mode,
		Selector:      assertion.Selector,
		Expected:      convertStructValue(assertion.Expected),
		Actual:        convertStructValue(assertion.Actual),
		Success:       assertion.Success,
		Message:       assertion.Message,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
	}
}

func convertBoundingBox(b *contracts.BoundingBox) *browser_automation_studio_v1.BoundingBox {
	if b == nil {
		return nil
	}
	return &browser_automation_studio_v1.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

func convertPoint(pt *contracts.Point) *browser_automation_studio_v1.Point {
	if pt == nil {
		return nil
	}
	return &browser_automation_studio_v1.Point{
		X: pt.X,
		Y: pt.Y,
	}
}

func convertElementFocus(f *contracts.ElementFocus) *browser_automation_studio_v1.ElementFocus {
	if f == nil {
		return nil
	}
	return &browser_automation_studio_v1.ElementFocus{
		Selector:    f.Selector,
		BoundingBox: convertBoundingBox(f.BoundingBox),
	}
}

func convertHighlightRegions(regions []contracts.HighlightRegion) []*browser_automation_studio_v1.HighlightRegion {
	if len(regions) == 0 {
		return nil
	}
	out := make([]*browser_automation_studio_v1.HighlightRegion, 0, len(regions))
	for _, r := range regions {
		out = append(out, &browser_automation_studio_v1.HighlightRegion{
			Selector:    r.Selector,
			BoundingBox: convertBoundingBox(r.BoundingBox),
			Padding:     int32(r.Padding),
			Color:       r.Color,
		})
	}
	return out
}

func convertMaskRegions(regions []contracts.MaskRegion) []*browser_automation_studio_v1.MaskRegion {
	if len(regions) == 0 {
		return nil
	}
	out := make([]*browser_automation_studio_v1.MaskRegion, 0, len(regions))
	for _, r := range regions {
		out = append(out, &browser_automation_studio_v1.MaskRegion{
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
