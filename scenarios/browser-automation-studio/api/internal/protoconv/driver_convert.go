package protoconv

import (
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// =============================================================================
// STEP OUTCOME CONVERSIONS
// =============================================================================

// StepOutcomeToProto converts a native Go StepOutcome to the proto type.
// This is used when sending step outcomes over the wire or storing in proto format.
func StepOutcomeToProto(outcome *autocontracts.StepOutcome) *basexecution.StepOutcome {
	if outcome == nil {
		return nil
	}

	pb := &basexecution.StepOutcome{
		SchemaVersion:  outcome.SchemaVersion,
		PayloadVersion: outcome.PayloadVersion,
		StepIndex:      int32(outcome.StepIndex),
		Attempt:        int32(outcome.Attempt),
		NodeId:         outcome.NodeID,
		StepType:       outcome.StepType,
		Success:        outcome.Success,
		StartedAt:      timestamppb.New(outcome.StartedAt),
	}

	// Optional string fields
	if outcome.ExecutionID != uuid.Nil {
		execID := outcome.ExecutionID.String()
		pb.ExecutionId = &execID
	}
	if outcome.CorrelationID != "" {
		pb.CorrelationId = &outcome.CorrelationID
	}
	if outcome.Instruction != "" {
		pb.Instruction = &outcome.Instruction
	}
	if outcome.FinalURL != "" {
		pb.FinalUrl = &outcome.FinalURL
	}

	// Optional time fields
	if outcome.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*outcome.CompletedAt)
	}
	if outcome.DurationMs > 0 {
		durationMs := int32(outcome.DurationMs)
		pb.DurationMs = &durationMs
	}

	// Screenshot
	if outcome.Screenshot != nil {
		pb.Screenshot = ScreenshotToProtoDriver(outcome.Screenshot)
	}

	// DOM Snapshot
	if outcome.DOMSnapshot != nil {
		pb.DomSnapshot = DOMSnapshotToProto(outcome.DOMSnapshot)
	}

	// Console logs
	for _, log := range outcome.ConsoleLogs {
		pb.ConsoleLogs = append(pb.ConsoleLogs, ConsoleLogEntryToProto(&log))
	}

	// Network events
	for _, event := range outcome.Network {
		pb.NetworkEvents = append(pb.NetworkEvents, NetworkEventToProto(&event))
	}

	// Extracted data
	if len(outcome.ExtractedData) > 0 {
		pb.ExtractedData = toJsonValueMap(outcome.ExtractedData)
	}

	// Assertion outcome
	if outcome.Assertion != nil {
		pb.Assertion = AssertionOutcomeToProto(outcome.Assertion)
	}

	// Condition outcome
	if outcome.Condition != nil {
		pb.Condition = ConditionOutcomeToProto(outcome.Condition)
	}

	// Probe result
	if len(outcome.ProbeResult) > 0 {
		pb.ProbeResult = toJsonValueMap(outcome.ProbeResult)
	}

	// Element interaction
	if outcome.ElementBoundingBox != nil {
		pb.ElementBoundingBox = outcome.ElementBoundingBox
	}
	if outcome.ClickPosition != nil {
		pb.ClickPosition = outcome.ClickPosition
	}
	if outcome.FocusedElement != nil {
		pb.FocusedElement = outcome.FocusedElement
	}

	// Visual regions - convert slice of values to slice of pointers
	for i := range outcome.HighlightRegions {
		pb.HighlightRegions = append(pb.HighlightRegions, &outcome.HighlightRegions[i])
	}
	for i := range outcome.MaskRegions {
		pb.MaskRegions = append(pb.MaskRegions, &outcome.MaskRegions[i])
	}

	// Zoom factor
	if outcome.ZoomFactor > 0 {
		pb.ZoomFactor = &outcome.ZoomFactor
	}

	// Cursor trail
	for _, pos := range outcome.CursorTrail {
		pb.CursorTrail = append(pb.CursorTrail, CursorPositionToProto(&pos))
	}

	// Notes
	if len(outcome.Notes) > 0 {
		pb.Notes = outcome.Notes
	}

	// Failure
	if outcome.Failure != nil {
		pb.Failure = StepFailureToProto(outcome.Failure)
	}

	return pb
}

// ProtoToStepOutcome converts a proto StepOutcome to the native Go type.
// This is used when receiving step outcomes from the wire or loading from proto storage.
func ProtoToStepOutcome(pb *basexecution.StepOutcome) *autocontracts.StepOutcome {
	if pb == nil {
		return nil
	}

	outcome := &autocontracts.StepOutcome{
		SchemaVersion:  pb.SchemaVersion,
		PayloadVersion: pb.PayloadVersion,
		StepIndex:      int(pb.StepIndex),
		Attempt:        int(pb.Attempt),
		NodeID:         pb.NodeId,
		StepType:       pb.StepType,
		Success:        pb.Success,
		StartedAt:      pb.StartedAt.AsTime(),
	}

	// Optional string fields
	if pb.ExecutionId != nil {
		if id, err := uuid.Parse(*pb.ExecutionId); err == nil {
			outcome.ExecutionID = id
		}
	}
	if pb.CorrelationId != nil {
		outcome.CorrelationID = *pb.CorrelationId
	}
	if pb.Instruction != nil {
		outcome.Instruction = *pb.Instruction
	}
	if pb.FinalUrl != nil {
		outcome.FinalURL = *pb.FinalUrl
	}

	// Optional time fields
	if pb.CompletedAt != nil {
		t := pb.CompletedAt.AsTime()
		outcome.CompletedAt = &t
	}
	if pb.DurationMs != nil {
		outcome.DurationMs = int(*pb.DurationMs)
	}

	// Screenshot
	if pb.Screenshot != nil {
		outcome.Screenshot = ProtoDriverToScreenshot(pb.Screenshot)
	}

	// DOM Snapshot
	if pb.DomSnapshot != nil {
		outcome.DOMSnapshot = ProtoToDOMSnapshot(pb.DomSnapshot)
	}

	// Console logs
	for _, log := range pb.ConsoleLogs {
		outcome.ConsoleLogs = append(outcome.ConsoleLogs, *ProtoToConsoleLogEntry(log))
	}

	// Network events
	for _, event := range pb.NetworkEvents {
		outcome.Network = append(outcome.Network, *ProtoToNetworkEvent(event))
	}

	// Extracted data
	if len(pb.ExtractedData) > 0 {
		outcome.ExtractedData = fromJsonValueMap(pb.ExtractedData)
	}

	// Assertion outcome
	if pb.Assertion != nil {
		outcome.Assertion = ProtoToAssertionOutcome(pb.Assertion)
	}

	// Condition outcome
	if pb.Condition != nil {
		outcome.Condition = ProtoToConditionOutcome(pb.Condition)
	}

	// Probe result
	if len(pb.ProbeResult) > 0 {
		outcome.ProbeResult = fromJsonValueMap(pb.ProbeResult)
	}

	// Element interaction (direct proto types via aliases)
	outcome.ElementBoundingBox = pb.ElementBoundingBox
	outcome.ClickPosition = pb.ClickPosition
	outcome.FocusedElement = pb.FocusedElement

	// Visual regions - convert slice of pointers to slice of values
	for _, region := range pb.HighlightRegions {
		if region != nil {
			outcome.HighlightRegions = append(outcome.HighlightRegions, *region)
		}
	}
	for _, region := range pb.MaskRegions {
		if region != nil {
			outcome.MaskRegions = append(outcome.MaskRegions, *region)
		}
	}

	// Zoom factor
	if pb.ZoomFactor != nil {
		outcome.ZoomFactor = *pb.ZoomFactor
	}

	// Cursor trail
	for _, pos := range pb.CursorTrail {
		outcome.CursorTrail = append(outcome.CursorTrail, *ProtoToCursorPosition(pos))
	}

	// Notes
	if len(pb.Notes) > 0 {
		outcome.Notes = pb.Notes
	}

	// Failure
	if pb.Failure != nil {
		outcome.Failure = ProtoToStepFailure(pb.Failure)
	}

	return outcome
}

// =============================================================================
// STEP FAILURE CONVERSIONS
// =============================================================================

// StepFailureToProto converts a native Go StepFailure to the proto type.
func StepFailureToProto(failure *autocontracts.StepFailure) *basexecution.StepFailure {
	if failure == nil {
		return nil
	}

	pb := &basexecution.StepFailure{
		Kind:      failureKindToProto(failure.Kind),
		Fatal:     failure.Fatal,
		Retryable: failure.Retryable,
		Source:    failureSourceToProto(failure.Source),
	}

	if failure.Code != "" {
		pb.Code = &failure.Code
	}
	if failure.Message != "" {
		pb.Message = &failure.Message
	}
	if failure.OccurredAt != nil {
		pb.OccurredAt = timestamppb.New(*failure.OccurredAt)
	}
	if len(failure.Details) > 0 {
		pb.Details = toJsonValueMap(failure.Details)
	}

	return pb
}

// ProtoToStepFailure converts a proto StepFailure to the native Go type.
func ProtoToStepFailure(pb *basexecution.StepFailure) *autocontracts.StepFailure {
	if pb == nil {
		return nil
	}

	failure := &autocontracts.StepFailure{
		Kind:      protoToFailureKind(pb.Kind),
		Fatal:     pb.Fatal,
		Retryable: pb.Retryable,
		Source:    protoToFailureSource(pb.Source),
	}

	if pb.Code != nil {
		failure.Code = *pb.Code
	}
	if pb.Message != nil {
		failure.Message = *pb.Message
	}
	if pb.OccurredAt != nil {
		t := pb.OccurredAt.AsTime()
		failure.OccurredAt = &t
	}
	if len(pb.Details) > 0 {
		failure.Details = fromJsonValueMap(pb.Details)
	}

	return failure
}

// =============================================================================
// FAILURE ENUM CONVERSIONS
// =============================================================================

func failureKindToProto(kind autocontracts.FailureKind) basexecution.FailureKind {
	switch kind {
	case autocontracts.FailureKindEngine:
		return basexecution.FailureKind_FAILURE_KIND_ENGINE
	case autocontracts.FailureKindInfra:
		return basexecution.FailureKind_FAILURE_KIND_INFRA
	case autocontracts.FailureKindOrchestration:
		return basexecution.FailureKind_FAILURE_KIND_ORCHESTRATION
	case autocontracts.FailureKindUser:
		return basexecution.FailureKind_FAILURE_KIND_USER
	case autocontracts.FailureKindTimeout:
		return basexecution.FailureKind_FAILURE_KIND_TIMEOUT
	case autocontracts.FailureKindCancelled:
		return basexecution.FailureKind_FAILURE_KIND_CANCELLED
	default:
		return basexecution.FailureKind_FAILURE_KIND_UNSPECIFIED
	}
}

func protoToFailureKind(kind basexecution.FailureKind) autocontracts.FailureKind {
	switch kind {
	case basexecution.FailureKind_FAILURE_KIND_ENGINE:
		return autocontracts.FailureKindEngine
	case basexecution.FailureKind_FAILURE_KIND_INFRA:
		return autocontracts.FailureKindInfra
	case basexecution.FailureKind_FAILURE_KIND_ORCHESTRATION:
		return autocontracts.FailureKindOrchestration
	case basexecution.FailureKind_FAILURE_KIND_USER:
		return autocontracts.FailureKindUser
	case basexecution.FailureKind_FAILURE_KIND_TIMEOUT:
		return autocontracts.FailureKindTimeout
	case basexecution.FailureKind_FAILURE_KIND_CANCELLED:
		return autocontracts.FailureKindCancelled
	default:
		return autocontracts.FailureKindNone
	}
}

func failureSourceToProto(source autocontracts.FailureSource) basexecution.FailureSource {
	switch source {
	case autocontracts.FailureSourceEngine:
		return basexecution.FailureSource_FAILURE_SOURCE_ENGINE
	case autocontracts.FailureSourceExecutor:
		return basexecution.FailureSource_FAILURE_SOURCE_EXECUTOR
	case autocontracts.FailureSourceRecorder:
		return basexecution.FailureSource_FAILURE_SOURCE_RECORDER
	default:
		return basexecution.FailureSource_FAILURE_SOURCE_UNSPECIFIED
	}
}

func protoToFailureSource(source basexecution.FailureSource) autocontracts.FailureSource {
	switch source {
	case basexecution.FailureSource_FAILURE_SOURCE_ENGINE:
		return autocontracts.FailureSourceEngine
	case basexecution.FailureSource_FAILURE_SOURCE_EXECUTOR:
		return autocontracts.FailureSourceExecutor
	case basexecution.FailureSource_FAILURE_SOURCE_RECORDER:
		return autocontracts.FailureSourceRecorder
	default:
		return ""
	}
}

// =============================================================================
// TELEMETRY TYPE CONVERSIONS
// =============================================================================

// ScreenshotToProtoDriver converts a native Screenshot to DriverScreenshot proto.
func ScreenshotToProtoDriver(shot *autocontracts.Screenshot) *basexecution.DriverScreenshot {
	if shot == nil {
		return nil
	}

	pb := &basexecution.DriverScreenshot{
		Data:      shot.Data,
		MediaType: shot.MediaType,
		Width:     int32(shot.Width),
		Height:    int32(shot.Height),
		FromCache: shot.FromCache,
		Truncated: shot.Truncated,
	}

	if !shot.CaptureTime.IsZero() {
		pb.CaptureTime = timestamppb.New(shot.CaptureTime)
	}
	if shot.Hash != "" {
		pb.Hash = &shot.Hash
	}
	if shot.Source != "" {
		pb.Source = &shot.Source
	}

	return pb
}

// ProtoDriverToScreenshot converts a DriverScreenshot proto to native Screenshot.
func ProtoDriverToScreenshot(pb *basexecution.DriverScreenshot) *autocontracts.Screenshot {
	if pb == nil {
		return nil
	}

	shot := &autocontracts.Screenshot{
		Data:      pb.Data,
		MediaType: pb.MediaType,
		Width:     int(pb.Width),
		Height:    int(pb.Height),
		FromCache: pb.FromCache,
		Truncated: pb.Truncated,
	}

	if pb.CaptureTime != nil {
		shot.CaptureTime = pb.CaptureTime.AsTime()
	}
	if pb.Hash != nil {
		shot.Hash = *pb.Hash
	}
	if pb.Source != nil {
		shot.Source = *pb.Source
	}

	return shot
}

// DOMSnapshotToProto converts a native DOMSnapshot to proto.
func DOMSnapshotToProto(dom *autocontracts.DOMSnapshot) *basexecution.DOMSnapshot {
	if dom == nil {
		return nil
	}

	pb := &basexecution.DOMSnapshot{
		Truncated: dom.Truncated,
	}

	if dom.HTML != "" {
		pb.Html = &dom.HTML
	}
	if dom.Preview != "" {
		pb.Preview = &dom.Preview
	}
	if dom.Hash != "" {
		pb.Hash = &dom.Hash
	}
	if !dom.CollectedAt.IsZero() {
		pb.CollectedAt = timestamppb.New(dom.CollectedAt)
	}

	return pb
}

// ProtoToDOMSnapshot converts a proto DOMSnapshot to native.
func ProtoToDOMSnapshot(pb *basexecution.DOMSnapshot) *autocontracts.DOMSnapshot {
	if pb == nil {
		return nil
	}

	dom := &autocontracts.DOMSnapshot{
		Truncated: pb.Truncated,
	}

	if pb.Html != nil {
		dom.HTML = *pb.Html
	}
	if pb.Preview != nil {
		dom.Preview = *pb.Preview
	}
	if pb.Hash != nil {
		dom.Hash = *pb.Hash
	}
	if pb.CollectedAt != nil {
		dom.CollectedAt = pb.CollectedAt.AsTime()
	}

	return dom
}

// ConsoleLogEntryToProto converts a native ConsoleLogEntry to proto.
func ConsoleLogEntryToProto(log *autocontracts.ConsoleLogEntry) *basexecution.DriverConsoleLogEntry {
	if log == nil {
		return nil
	}

	pb := &basexecution.DriverConsoleLogEntry{
		Type:      log.Type,
		Text:      log.Text,
		Timestamp: timestamppb.New(log.Timestamp),
	}

	if log.Stack != "" {
		pb.Stack = &log.Stack
	}
	if log.Location != "" {
		pb.Location = &log.Location
	}

	return pb
}

// ProtoToConsoleLogEntry converts a proto DriverConsoleLogEntry to native.
func ProtoToConsoleLogEntry(pb *basexecution.DriverConsoleLogEntry) *autocontracts.ConsoleLogEntry {
	if pb == nil {
		return nil
	}

	log := &autocontracts.ConsoleLogEntry{
		Type:      pb.Type,
		Text:      pb.Text,
		Timestamp: pb.Timestamp.AsTime(),
	}

	if pb.Stack != nil {
		log.Stack = *pb.Stack
	}
	if pb.Location != nil {
		log.Location = *pb.Location
	}

	return log
}

// NetworkEventToProto converts a native NetworkEvent to proto.
func NetworkEventToProto(event *autocontracts.NetworkEvent) *basexecution.DriverNetworkEvent {
	if event == nil {
		return nil
	}

	pb := &basexecution.DriverNetworkEvent{
		Type:      event.Type,
		Url:       event.URL,
		Timestamp: timestamppb.New(event.Timestamp),
		Truncated: event.Truncated,
	}

	if event.Method != "" {
		pb.Method = &event.Method
	}
	if event.ResourceType != "" {
		pb.ResourceType = &event.ResourceType
	}
	if event.Status != 0 {
		status := int32(event.Status)
		pb.Status = &status
	}
	if event.OK {
		pb.Ok = &event.OK
	}
	if event.Failure != "" {
		pb.Failure = &event.Failure
	}
	if len(event.RequestHeaders) > 0 {
		pb.RequestHeaders = event.RequestHeaders
	}
	if len(event.ResponseHeaders) > 0 {
		pb.ResponseHeaders = event.ResponseHeaders
	}
	if event.RequestBodyPreview != "" {
		pb.RequestBodyPreview = &event.RequestBodyPreview
	}
	if event.ResponseBodyPreview != "" {
		pb.ResponseBodyPreview = &event.ResponseBodyPreview
	}

	return pb
}

// ProtoToNetworkEvent converts a proto DriverNetworkEvent to native.
func ProtoToNetworkEvent(pb *basexecution.DriverNetworkEvent) *autocontracts.NetworkEvent {
	if pb == nil {
		return nil
	}

	event := &autocontracts.NetworkEvent{
		Type:      pb.Type,
		URL:       pb.Url,
		Timestamp: pb.Timestamp.AsTime(),
		Truncated: pb.Truncated,
	}

	if pb.Method != nil {
		event.Method = *pb.Method
	}
	if pb.ResourceType != nil {
		event.ResourceType = *pb.ResourceType
	}
	if pb.Status != nil {
		event.Status = int(*pb.Status)
	}
	if pb.Ok != nil {
		event.OK = *pb.Ok
	}
	if pb.Failure != nil {
		event.Failure = *pb.Failure
	}
	if len(pb.RequestHeaders) > 0 {
		event.RequestHeaders = pb.RequestHeaders
	}
	if len(pb.ResponseHeaders) > 0 {
		event.ResponseHeaders = pb.ResponseHeaders
	}
	if pb.RequestBodyPreview != nil {
		event.RequestBodyPreview = *pb.RequestBodyPreview
	}
	if pb.ResponseBodyPreview != nil {
		event.ResponseBodyPreview = *pb.ResponseBodyPreview
	}

	return event
}

// CursorPositionToProto converts a native CursorPosition to proto.
func CursorPositionToProto(pos *autocontracts.CursorPosition) *basexecution.CursorPosition {
	if pos == nil {
		return nil
	}

	pb := &basexecution.CursorPosition{
		Point: &pos.Point,
	}

	if !pos.RecordedAt.IsZero() {
		pb.RecordedAt = timestamppb.New(pos.RecordedAt)
	}
	if pos.ElapsedMs != 0 {
		pb.ElapsedMs = &pos.ElapsedMs
	}

	return pb
}

// ProtoToCursorPosition converts a proto CursorPosition to native.
func ProtoToCursorPosition(pb *basexecution.CursorPosition) *autocontracts.CursorPosition {
	if pb == nil {
		return nil
	}

	pos := &autocontracts.CursorPosition{}

	if pb.Point != nil {
		pos.Point = *pb.Point
	}
	if pb.RecordedAt != nil {
		pos.RecordedAt = pb.RecordedAt.AsTime()
	}
	if pb.ElapsedMs != nil {
		pos.ElapsedMs = *pb.ElapsedMs
	}

	return pos
}

// AssertionOutcomeToProto converts a native AssertionOutcome to proto.
func AssertionOutcomeToProto(assertion *autocontracts.AssertionOutcome) *basexecution.AssertionOutcome {
	if assertion == nil {
		return nil
	}

	pb := &basexecution.AssertionOutcome{
		Success:       assertion.Success,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
	}

	if assertion.Mode != "" {
		pb.Mode = &assertion.Mode
	}
	if assertion.Selector != "" {
		pb.Selector = &assertion.Selector
	}
	if assertion.Expected != nil {
		pb.Expected = toJsonValue(assertion.Expected)
	}
	if assertion.Actual != nil {
		pb.Actual = toJsonValue(assertion.Actual)
	}
	if assertion.Message != "" {
		pb.Message = &assertion.Message
	}

	return pb
}

// ProtoToAssertionOutcome converts a proto AssertionOutcome to native.
func ProtoToAssertionOutcome(pb *basexecution.AssertionOutcome) *autocontracts.AssertionOutcome {
	if pb == nil {
		return nil
	}

	assertion := &autocontracts.AssertionOutcome{
		Success:       pb.Success,
		Negated:       pb.Negated,
		CaseSensitive: pb.CaseSensitive,
	}

	if pb.Mode != nil {
		assertion.Mode = *pb.Mode
	}
	if pb.Selector != nil {
		assertion.Selector = *pb.Selector
	}
	if pb.Expected != nil {
		assertion.Expected = fromJsonValue(pb.Expected)
	}
	if pb.Actual != nil {
		assertion.Actual = fromJsonValue(pb.Actual)
	}
	if pb.Message != nil {
		assertion.Message = *pb.Message
	}

	return assertion
}

// ConditionOutcomeToProto converts a native ConditionOutcome to proto.
func ConditionOutcomeToProto(condition *autocontracts.ConditionOutcome) *basexecution.ConditionOutcome {
	if condition == nil {
		return nil
	}

	pb := &basexecution.ConditionOutcome{
		Outcome: condition.Outcome,
		Negated: condition.Negated,
	}

	if condition.Type != "" {
		pb.Type = &condition.Type
	}
	if condition.Operator != "" {
		pb.Operator = &condition.Operator
	}
	if condition.Variable != "" {
		pb.Variable = &condition.Variable
	}
	if condition.Selector != "" {
		pb.Selector = &condition.Selector
	}
	if condition.Expression != "" {
		pb.Expression = &condition.Expression
	}
	if condition.Actual != nil {
		pb.Actual = toJsonValue(condition.Actual)
	}
	if condition.Expected != nil {
		pb.Expected = toJsonValue(condition.Expected)
	}

	return pb
}

// ProtoToConditionOutcome converts a proto ConditionOutcome to native.
func ProtoToConditionOutcome(pb *basexecution.ConditionOutcome) *autocontracts.ConditionOutcome {
	if pb == nil {
		return nil
	}

	condition := &autocontracts.ConditionOutcome{
		Outcome: pb.Outcome,
		Negated: pb.Negated,
	}

	if pb.Type != nil {
		condition.Type = *pb.Type
	}
	if pb.Operator != nil {
		condition.Operator = *pb.Operator
	}
	if pb.Variable != nil {
		condition.Variable = *pb.Variable
	}
	if pb.Selector != nil {
		condition.Selector = *pb.Selector
	}
	if pb.Expression != nil {
		condition.Expression = *pb.Expression
	}
	if pb.Actual != nil {
		condition.Actual = fromJsonValue(pb.Actual)
	}
	if pb.Expected != nil {
		condition.Expected = fromJsonValue(pb.Expected)
	}

	return condition
}

// =============================================================================
// PLAN CONVERSIONS
// =============================================================================

// ExecutionPlanToProto converts a native ExecutionPlan to proto.
func ExecutionPlanToProto(plan *autocontracts.ExecutionPlan) *basexecution.ExecutionPlan {
	if plan == nil {
		return nil
	}

	pb := &basexecution.ExecutionPlan{
		SchemaVersion:  plan.SchemaVersion,
		PayloadVersion: plan.PayloadVersion,
		ExecutionId:    plan.ExecutionID.String(),
		WorkflowId:     plan.WorkflowID.String(),
		CreatedAt:      timestamppb.New(plan.CreatedAt),
	}

	// Instructions
	for _, instr := range plan.Instructions {
		pb.Instructions = append(pb.Instructions, CompiledInstructionToProto(&instr))
	}

	// Graph
	if plan.Graph != nil {
		pb.Graph = PlanGraphToProto(plan.Graph)
	}

	// Metadata
	if len(plan.Metadata) > 0 {
		pb.Metadata = toJsonValueMap(plan.Metadata)
	}

	return pb
}

// ProtoToExecutionPlan converts a proto ExecutionPlan to native.
func ProtoToExecutionPlan(pb *basexecution.ExecutionPlan) *autocontracts.ExecutionPlan {
	if pb == nil {
		return nil
	}

	plan := &autocontracts.ExecutionPlan{
		SchemaVersion:  pb.SchemaVersion,
		PayloadVersion: pb.PayloadVersion,
		CreatedAt:      pb.CreatedAt.AsTime(),
	}

	if id, err := uuid.Parse(pb.ExecutionId); err == nil {
		plan.ExecutionID = id
	}
	if id, err := uuid.Parse(pb.WorkflowId); err == nil {
		plan.WorkflowID = id
	}

	// Instructions
	for _, instr := range pb.Instructions {
		plan.Instructions = append(plan.Instructions, *ProtoToCompiledInstruction(instr))
	}

	// Graph
	if pb.Graph != nil {
		plan.Graph = ProtoToPlanGraph(pb.Graph)
	}

	// Metadata
	if len(pb.Metadata) > 0 {
		plan.Metadata = fromJsonValueMap(pb.Metadata)
	}

	return plan
}

// CompiledInstructionToProto converts a native CompiledInstruction to proto.
//
// MIGRATION: This function handles both old (Type/Params) and new (Action) fields.
// When Action is set, it takes precedence. Legacy Type/Params are still populated
// for backward compatibility with consumers that haven't migrated yet.
func CompiledInstructionToProto(instr *autocontracts.CompiledInstruction) *basexecution.CompiledInstruction {
	if instr == nil {
		return nil
	}

	pb := &basexecution.CompiledInstruction{
		Index:  int32(instr.Index),
		NodeId: instr.NodeID,
	}

	// NEW: Populate typed action field if present
	if instr.Action != nil {
		pb.Action = instr.Action
	}

	// DEPRECATED: Still populate legacy fields for backward compatibility
	if instr.Type != "" {
		pb.Type = instr.Type
	}
	if len(instr.Params) > 0 {
		pb.Params = toJsonValueMap(instr.Params)
	}

	if instr.PreloadHTML != "" {
		pb.PreloadHtml = &instr.PreloadHTML
	}
	if len(instr.Context) > 0 {
		pb.Context = toJsonValueMap(instr.Context)
	}
	if len(instr.Metadata) > 0 {
		pb.Metadata = instr.Metadata
	}

	return pb
}

// ProtoToCompiledInstruction converts a proto CompiledInstruction to native.
//
// MIGRATION: This function handles both old (Type/Params) and new (Action) fields.
// When Action is present, it takes precedence and is preserved directly.
func ProtoToCompiledInstruction(pb *basexecution.CompiledInstruction) *autocontracts.CompiledInstruction {
	if pb == nil {
		return nil
	}

	instr := &autocontracts.CompiledInstruction{
		Index:  int(pb.Index),
		NodeID: pb.NodeId,
	}

	// NEW: Preserve typed action field if present
	if pb.Action != nil {
		instr.Action = pb.Action
	}

	// DEPRECATED: Still read legacy fields for backward compatibility
	if pb.Type != "" {
		instr.Type = pb.Type
	}
	if len(pb.Params) > 0 {
		instr.Params = fromJsonValueMap(pb.Params)
	}

	if pb.PreloadHtml != nil {
		instr.PreloadHTML = *pb.PreloadHtml
	}
	if len(pb.Context) > 0 {
		instr.Context = fromJsonValueMap(pb.Context)
	}
	if len(pb.Metadata) > 0 {
		instr.Metadata = pb.Metadata
	}

	return instr
}

// PlanGraphToProto converts a native PlanGraph to proto.
func PlanGraphToProto(graph *autocontracts.PlanGraph) *basexecution.PlanGraph {
	if graph == nil {
		return nil
	}

	pb := &basexecution.PlanGraph{}
	for _, step := range graph.Steps {
		pb.Steps = append(pb.Steps, PlanStepToProto(&step))
	}
	return pb
}

// ProtoToPlanGraph converts a proto PlanGraph to native.
func ProtoToPlanGraph(pb *basexecution.PlanGraph) *autocontracts.PlanGraph {
	if pb == nil {
		return nil
	}

	graph := &autocontracts.PlanGraph{}
	for _, step := range pb.Steps {
		graph.Steps = append(graph.Steps, *ProtoToPlanStep(step))
	}
	return graph
}

// PlanStepToProto converts a native PlanStep to proto.
//
// MIGRATION: Same transition as CompiledInstruction - handles both old and new fields.
func PlanStepToProto(step *autocontracts.PlanStep) *basexecution.PlanStep {
	if step == nil {
		return nil
	}

	pb := &basexecution.PlanStep{
		Index:  int32(step.Index),
		NodeId: step.NodeID,
	}

	// NEW: Populate typed action field if present
	if step.Action != nil {
		pb.Action = step.Action
	}

	// DEPRECATED: Still populate legacy fields for backward compatibility
	if step.Type != "" {
		pb.Type = step.Type
	}
	if len(step.Params) > 0 {
		pb.Params = toJsonValueMap(step.Params)
	}

	for _, edge := range step.Outgoing {
		pb.Outgoing = append(pb.Outgoing, PlanEdgeToProto(&edge))
	}
	if step.Loop != nil {
		pb.Loop = PlanGraphToProto(step.Loop)
	}
	if len(step.Metadata) > 0 {
		pb.Metadata = step.Metadata
	}
	if len(step.Context) > 0 {
		pb.Context = toJsonValueMap(step.Context)
	}
	if step.Preload != "" {
		pb.PreloadHtml = &step.Preload
	}
	if len(step.SourcePos) > 0 {
		pb.SourcePosition = toJsonValueMap(step.SourcePos)
	}

	return pb
}

// ProtoToPlanStep converts a proto PlanStep to native.
//
// MIGRATION: Same transition as CompiledInstruction - handles both old and new fields.
func ProtoToPlanStep(pb *basexecution.PlanStep) *autocontracts.PlanStep {
	if pb == nil {
		return nil
	}

	step := &autocontracts.PlanStep{
		Index:  int(pb.Index),
		NodeID: pb.NodeId,
	}

	// NEW: Preserve typed action field if present
	if pb.Action != nil {
		step.Action = pb.Action
	}

	// DEPRECATED: Still read legacy fields for backward compatibility
	if pb.Type != "" {
		step.Type = pb.Type
	}
	if len(pb.Params) > 0 {
		step.Params = fromJsonValueMap(pb.Params)
	}

	for _, edge := range pb.Outgoing {
		step.Outgoing = append(step.Outgoing, *ProtoToPlanEdge(edge))
	}
	if pb.Loop != nil {
		step.Loop = ProtoToPlanGraph(pb.Loop)
	}
	if len(pb.Metadata) > 0 {
		step.Metadata = pb.Metadata
	}
	if len(pb.Context) > 0 {
		step.Context = fromJsonValueMap(pb.Context)
	}
	if pb.PreloadHtml != nil {
		step.Preload = *pb.PreloadHtml
	}
	if len(pb.SourcePosition) > 0 {
		step.SourcePos = fromJsonValueMap(pb.SourcePosition)
	}

	return step
}

// PlanEdgeToProto converts a native PlanEdge to proto.
func PlanEdgeToProto(edge *autocontracts.PlanEdge) *basexecution.PlanEdge {
	if edge == nil {
		return nil
	}

	pb := &basexecution.PlanEdge{
		Id:     edge.ID,
		Target: edge.Target,
	}

	if edge.Condition != "" {
		pb.Condition = &edge.Condition
	}
	if edge.SourcePort != "" {
		pb.SourcePort = &edge.SourcePort
	}
	if edge.TargetPort != "" {
		pb.TargetPort = &edge.TargetPort
	}

	return pb
}

// ProtoToPlanEdge converts a proto PlanEdge to native.
func ProtoToPlanEdge(pb *basexecution.PlanEdge) *autocontracts.PlanEdge {
	if pb == nil {
		return nil
	}

	edge := &autocontracts.PlanEdge{
		ID:     pb.Id,
		Target: pb.Target,
	}

	if pb.Condition != nil {
		edge.Condition = *pb.Condition
	}
	if pb.SourcePort != nil {
		edge.SourcePort = *pb.SourcePort
	}
	if pb.TargetPort != nil {
		edge.TargetPort = *pb.TargetPort
	}

	return edge
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// fromJsonValueMap converts a map of proto JsonValues to native Go map[string]any.
func fromJsonValueMap(source map[string]*commonv1.JsonValue) map[string]any {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = fromJsonValue(value)
	}
	return result
}

// fromJsonValue converts a proto JsonValue to native Go any.
func fromJsonValue(value *commonv1.JsonValue) any {
	if value == nil {
		return nil
	}

	switch v := value.Kind.(type) {
	case *commonv1.JsonValue_NullValue:
		return nil
	case *commonv1.JsonValue_BoolValue:
		return v.BoolValue
	case *commonv1.JsonValue_IntValue:
		return v.IntValue
	case *commonv1.JsonValue_DoubleValue:
		return v.DoubleValue
	case *commonv1.JsonValue_StringValue:
		return v.StringValue
	case *commonv1.JsonValue_BytesValue:
		return v.BytesValue
	case *commonv1.JsonValue_ObjectValue:
		if v.ObjectValue == nil {
			return nil
		}
		result := make(map[string]any, len(v.ObjectValue.Fields))
		for key, val := range v.ObjectValue.Fields {
			result[key] = fromJsonValue(val)
		}
		return result
	case *commonv1.JsonValue_ListValue:
		if v.ListValue == nil {
			return nil
		}
		result := make([]any, 0, len(v.ListValue.Values))
		for _, val := range v.ListValue.Values {
			result = append(result, fromJsonValue(val))
		}
		return result
	default:
		return nil
	}
}

// Unused variables to satisfy imports
var (
	_ = time.Time{}
	_ = bastimeline.TimelineEntry{}
	_ = basdomain.ActionTelemetry{}
	_ = basbase.BoundingBox{}
)
