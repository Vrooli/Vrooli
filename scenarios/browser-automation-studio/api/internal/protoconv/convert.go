package protoconv

import (
	"encoding/json"
	"fmt"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExecutionToProto converts a database.Execution into the generated proto message.
func ExecutionToProto(execution *database.Execution) (*browser_automation_studio_v1.Execution, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is nil")
	}

	pb := &browser_automation_studio_v1.Execution{
		ExecutionId:     execution.ID.String(),
		WorkflowId:      execution.WorkflowID.String(),
		WorkflowVersion: int32(execution.WorkflowVersion),
		Status:          typeconv.StringToExecutionStatus(execution.Status),
		TriggerType:     typeconv.StringToTriggerType(execution.TriggerType),
		Progress:        int32(execution.Progress),
		StartedAt:       timestamppb.New(execution.StartedAt),
		CreatedAt:       timestamppb.New(execution.CreatedAt),
		UpdatedAt:       timestamppb.New(execution.UpdatedAt),
	}

	// Set current_step if present (optional string)
	if execution.CurrentStep != "" {
		pb.CurrentStep = &execution.CurrentStep
	}

	// Set trigger_metadata if present (typed message)
	if execution.TriggerMetadata != nil {
		pb.TriggerMetadata = mapToTriggerMetadata(execution.TriggerMetadata)
	}

	// Set parameters if present (typed message)
	if execution.Parameters != nil {
		pb.Parameters = mapToExecutionParameters(execution.Parameters)
	}

	if execution.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*execution.CompletedAt)
	}
	if execution.LastHeartbeat != nil {
		pb.LastHeartbeatAt = timestamppb.New(*execution.LastHeartbeat)
	}
	if execution.Error.Valid {
		errMsg := execution.Error.String
		pb.Error = &errMsg
	}

	// Set result if present (typed message)
	if execution.Result != nil {
		pb.Result = mapToExecutionResult(execution.Result)
	}

	return pb, nil
}

// ExecuteAdhocResponseProto converts the execution into the adhoc response proto.
func ExecuteAdhocResponseProto(execution *database.Execution, message string) (*browser_automation_studio_v1.ExecuteAdhocResponse, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is nil")
	}

	pb := &browser_automation_studio_v1.ExecuteAdhocResponse{
		ExecutionId: execution.ID.String(),
		Status:      typeconv.StringToExecutionStatus(execution.Status),
		Message:     message,
	}

	if execution.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*execution.CompletedAt)
	}
	if execution.Error.Valid {
		errMsg := execution.Error.String
		pb.Error = &errMsg
	}

	return pb, nil
}

// ExecutionExportPreviewToProto converts the workflow.ExecutionExportPreview to the proto message.
func ExecutionExportPreviewToProto(preview *workflow.ExecutionExportPreview) (*browser_automation_studio_v1.ExecutionExportPreview, error) {
	if preview == nil {
		return nil, fmt.Errorf("preview is nil")
	}

	return &browser_automation_studio_v1.ExecutionExportPreview{
		ExecutionId:         preview.ExecutionID.String(),
		SpecId:              preview.SpecID,
		Status:              typeconv.StringToExportStatus(preview.Status),
		Message:             preview.Message,
		CapturedFrameCount:  int32(preview.CapturedFrameCount),
		AvailableAssetCount: int32(preview.AvailableAssetCount),
		TotalDurationMs:     int32(preview.TotalDurationMs),
		Package:             toJsonObjectFromAny(preview.Package),
	}, nil
}

// ScreenshotsToProto converts database screenshots to the proto response.
// Uses ExecutionScreenshot which wraps TimelineScreenshot with execution context.
func ScreenshotsToProto(screenshots []*database.Screenshot, executionID string) (*browser_automation_studio_v1.GetScreenshotsResponse, error) {
	if len(screenshots) == 0 {
		return &browser_automation_studio_v1.GetScreenshotsResponse{
			ExecutionId: executionID,
			Total:       0,
		}, nil
	}

	result := make([]*browser_automation_studio_v1.ExecutionScreenshot, 0, len(screenshots))
	for idx, shot := range screenshots {
		if shot == nil {
			return nil, fmt.Errorf("screenshots[%d] is nil", idx)
		}

		var stepIndex int32
		var nodeID string
		if shot.Metadata != nil {
			if v, ok := shot.Metadata["step_index"]; ok {
				stepIndex = toInt32Val(v)
			}
			if v, ok := shot.Metadata["node_id"].(string); ok {
				nodeID = v
			}
		}

		thumbURL := shot.ThumbnailURL
		if thumbURL == "" && shot.Metadata != nil {
			if v, ok := shot.Metadata["thumbnail_url"].(string); ok {
				thumbURL = v
			}
		}

		// Build TimelineScreenshot (the canonical type)
		timelineShot := &browser_automation_studio_v1.TimelineScreenshot{
			ArtifactId:   shot.ID.String(),
			Url:          shot.StorageURL,
			ThumbnailUrl: thumbURL,
			Width:        int32(shot.Width),
			Height:       int32(shot.Height),
			ContentType:  "image/png", // Default, could be from metadata if stored
		}
		if shot.SizeBytes > 0 {
			sizeBytes := shot.SizeBytes
			timelineShot.SizeBytes = &sizeBytes
		}

		// Wrap in ExecutionScreenshot with execution context
		execShot := &browser_automation_studio_v1.ExecutionScreenshot{
			Screenshot: timelineShot,
			StepIndex:  stepIndex,
			NodeId:     nodeID,
			Timestamp:  timestamppb.New(shot.Timestamp),
		}
		if shot.StepName != "" {
			execShot.StepLabel = &shot.StepName
		}

		result = append(result, execShot)
	}

	return &browser_automation_studio_v1.GetScreenshotsResponse{
		ExecutionId: executionID,
		Screenshots: result,
		Total:       int32(len(result)),
	}, nil
}

// TimelineToProto converts the replay-focused ExecutionTimeline into the proto message.
func TimelineToProto(timeline *export.ExecutionTimeline) (*browser_automation_studio_v1.ExecutionTimeline, error) {
	if timeline == nil {
		return nil, fmt.Errorf("timeline is nil")
	}

	pb := &browser_automation_studio_v1.ExecutionTimeline{
		ExecutionId: timeline.ExecutionID.String(),
		WorkflowId:  timeline.WorkflowID.String(),
		Status:      typeconv.StringToExecutionStatus(timeline.Status),
		Progress:    int32(timeline.Progress),
		StartedAt:   timestamppb.New(timeline.StartedAt),
	}

	if timeline.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*timeline.CompletedAt)
	}

	for idx, frame := range timeline.Frames {
		pbEntry, err := timelineFrameToEntry(frame)
		if err != nil {
			return nil, fmt.Errorf("entry %d: %w", idx, err)
		}
		pb.Entries = append(pb.Entries, pbEntry)
	}

	for _, log := range timeline.Logs {
		pb.Logs = append(pb.Logs, timelineLogToProto(log))
	}

	return pb, nil
}

// timelineFrameToEntry converts an export.TimelineFrame to a browser_automation_studio_v1.TimelineEntry.
// This is the unified format for timeline data.
func timelineFrameToEntry(frame export.TimelineFrame) (*browser_automation_studio_v1.TimelineEntry, error) {
	// Build base entry
	stepIndex := int32(frame.StepIndex)
	// Generate entry ID from node_id and step_index since export.TimelineFrame doesn't have an ID field
	entryID := fmt.Sprintf("%s-step-%d", frame.NodeID, frame.StepIndex)
	entry := &browser_automation_studio_v1.TimelineEntry{
		Id:          entryID,
		SequenceNum: int32(frame.StepIndex), // Use step index as sequence for now
		StepIndex:   &stepIndex,
	}

	// Set node_id if available
	if frame.NodeID != "" {
		entry.NodeId = &frame.NodeID
	}

	// Set timestamp from StartedAt
	if frame.StartedAt != nil {
		entry.Timestamp = timestamppb.New(*frame.StartedAt)
	}

	// Set duration
	durationMs := int32(frame.DurationMs)
	if durationMs > 0 {
		entry.DurationMs = &durationMs
	}
	totalDurationMs := int32(frame.TotalDurationMs)
	if totalDurationMs > 0 {
		entry.TotalDurationMs = &totalDurationMs
	}

	// Build ActionDefinition with action type
	if frame.StepType != "" {
		entry.Action = &browser_automation_studio_v1.ActionDefinition{
			Type: typeconv.StringToActionType(frame.StepType),
		}
	}

	// Build Telemetry with visual data (screenshots, cursor trails, regions, etc.)
	entry.Telemetry = buildTelemetryFromFrame(frame)

	// Build EventContext
	entry.Context = &browser_automation_studio_v1.EventContext{
		Success: &frame.Success,
	}
	// Note: execution_id is not available in export.TimelineFrame - it's set at the parent timeline level

	// Add error to context if present
	if frame.Error != "" {
		entry.Context.Error = &frame.Error
	}

	// Build RetryStatus from individual retry fields
	if frame.RetryConfigured != 0 || frame.RetryAttempt > 0 || frame.RetryMaxAttempts > 0 {
		entry.Context.RetryStatus = &browser_automation_studio_v1.RetryStatus{
			CurrentAttempt: int32(frame.RetryAttempt),
			MaxAttempts:    int32(frame.RetryMaxAttempts),
			DelayMs:        int32(frame.RetryDelayMs),
			BackoffFactor:  frame.RetryBackoffFactor,
			Configured:     frame.RetryConfigured != 0,
			History:        convertRetryHistory(frame.RetryHistory),
		}
	}

	// Add assertion to context if present
	if frame.Assertion != nil {
		assertion, err := convertAssertion(frame.Assertion)
		if err != nil {
			return nil, err
		}
		entry.Context.Assertion = assertion
	}

	// Build aggregates for batch data
	progress := int32(frame.Progress)
	entry.Aggregates = &browser_automation_studio_v1.TimelineEntryAggregates{
		Status:            typeconv.StringToStepStatus(frame.Status),
		ConsoleLogCount:   int32(frame.ConsoleLogCount),
		NetworkEventCount: int32(frame.NetworkEventCount),
		Progress:          &progress,
	}

	if frame.FinalURL != "" {
		entry.Aggregates.FinalUrl = &frame.FinalURL
	}

	if frame.DomSnapshotPreview != "" {
		entry.Aggregates.DomSnapshotPreview = &frame.DomSnapshotPreview
	}

	if frame.ExtractedDataPreview != nil {
		entry.Aggregates.ExtractedDataPreview = toJsonValue(frame.ExtractedDataPreview)
	}

	if frame.FocusedElement != nil {
		entry.Aggregates.FocusedElement = convertElementFocus(frame.FocusedElement)
	}

	// Add artifacts to aggregates
	if frame.Artifacts != nil {
		for _, artifact := range frame.Artifacts {
			pbArtifact, err := convertArtifact(artifact)
			if err != nil {
				return nil, fmt.Errorf("artifact %s: %w", artifact.ID, err)
			}
			entry.Aggregates.Artifacts = append(entry.Aggregates.Artifacts, pbArtifact)
		}
	}

	// Add DOM snapshot artifact to aggregates
	if frame.DomSnapshot != nil {
		dom, err := convertArtifact(*frame.DomSnapshot)
		if err != nil {
			return nil, fmt.Errorf("dom_snapshot: %w", err)
		}
		entry.Aggregates.DomSnapshot = dom
	}

	return entry, nil
}

// timelineFrameToProto is deprecated. Use timelineFrameToEntry instead.
// This wrapper exists for backwards compatibility.
func timelineFrameToProto(frame export.TimelineFrame) (*browser_automation_studio_v1.TimelineFrame, error) {
	entry, err := timelineFrameToEntry(frame)
	if err != nil {
		return nil, err
	}
	return &browser_automation_studio_v1.TimelineFrame{Entry: entry}, nil
}

// buildTelemetryFromFrame creates ActionTelemetry from a TimelineFrame.
func buildTelemetryFromFrame(frame export.TimelineFrame) *browser_automation_studio_v1.ActionTelemetry {
	tel := &browser_automation_studio_v1.ActionTelemetry{
		Url: frame.FinalURL,
	}

	if frame.ElementBoundingBox != nil {
		tel.ElementBoundingBox = convertBoundingBox(frame.ElementBoundingBox)
	}

	if frame.ClickPosition != nil {
		tel.ClickPosition = convertPoint(frame.ClickPosition)
	}

	if frame.CursorTrail != nil {
		for _, pt := range frame.CursorTrail {
			point := pt
			tel.CursorTrail = append(tel.CursorTrail, convertPoint(&point))
		}
	}

	if frame.HighlightRegions != nil {
		for _, region := range frame.HighlightRegions {
			tel.HighlightRegions = append(tel.HighlightRegions, convertHighlightRegion(region))
		}
	}

	if frame.MaskRegions != nil {
		for _, region := range frame.MaskRegions {
			tel.MaskRegions = append(tel.MaskRegions, convertMaskRegion(region))
		}
	}

	if frame.Screenshot != nil {
		tel.Screenshot = convertScreenshot(frame.Screenshot)
	}

	return tel
}

func timelineLogToProto(log export.TimelineLog) *browser_automation_studio_v1.TimelineLog {
	pb := &browser_automation_studio_v1.TimelineLog{
		Id:        log.ID,
		Level:     typeconv.StringToLogLevel(log.Level),
		Message:   log.Message,
		Timestamp: timestamppb.New(log.Timestamp),
	}
	if log.StepName != "" {
		stepName := log.StepName
		pb.StepName = &stepName
	}
	return pb
}

func convertRetryHistory(entries []typeconv.RetryHistoryEntry) []*browser_automation_studio_v1.RetryAttempt {
	if len(entries) == 0 {
		return nil
	}
	result := make([]*browser_automation_studio_v1.RetryAttempt, 0, len(entries))
	for _, entry := range entries {
		attempt := &browser_automation_studio_v1.RetryAttempt{
			Attempt:    int32(entry.Attempt),
			Success:    entry.Success,
			DurationMs: int32(entry.DurationMs),
		}
		if entry.Error != "" {
			attempt.Error = &entry.Error
		}
		result = append(result, attempt)
	}
	return result
}

func convertHighlightRegion(region autocontracts.HighlightRegion) *browser_automation_studio_v1.HighlightRegion {
	pb := &browser_automation_studio_v1.HighlightRegion{
		Selector:       region.Selector,
		Padding:        region.Padding,
		HighlightColor: region.HighlightColor,
		CustomRgba:     region.CustomRgba,
	}
	if region.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(region.BoundingBox)
	}
	return pb
}

func convertMaskRegion(region autocontracts.MaskRegion) *browser_automation_studio_v1.MaskRegion {
	pb := &browser_automation_studio_v1.MaskRegion{
		Selector: region.Selector,
		Opacity:  region.Opacity,
	}
	if region.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(region.BoundingBox)
	}
	return pb
}

func convertElementFocus(focus *autocontracts.ElementFocus) *browser_automation_studio_v1.ElementFocus {
	pb := &browser_automation_studio_v1.ElementFocus{
		Selector: focus.Selector,
	}
	if focus.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(focus.BoundingBox)
	}
	return pb
}

func convertBoundingBox(bbox *autocontracts.BoundingBox) *browser_automation_studio_v1.BoundingBox {
	if bbox == nil {
		return nil
	}
	return &browser_automation_studio_v1.BoundingBox{
		X:      bbox.X,
		Y:      bbox.Y,
		Width:  bbox.Width,
		Height: bbox.Height,
	}
}

func convertPoint(pt *autocontracts.Point) *browser_automation_studio_v1.Point {
	if pt == nil {
		return nil
	}
	return &browser_automation_studio_v1.Point{
		X: pt.X,
		Y: pt.Y,
	}
}

func convertScreenshot(screenshot *typeconv.TimelineScreenshot) *browser_automation_studio_v1.TimelineScreenshot {
	if screenshot == nil {
		return nil
	}
	pb := &browser_automation_studio_v1.TimelineScreenshot{
		ArtifactId:   screenshot.ArtifactID,
		Url:          screenshot.URL,
		ThumbnailUrl: screenshot.ThumbnailURL,
		Width:        int32(screenshot.Width),
		Height:       int32(screenshot.Height),
		ContentType:  screenshot.ContentType,
	}
	if screenshot.SizeBytes != nil {
		pb.SizeBytes = screenshot.SizeBytes
	}
	return pb
}

func convertArtifact(artifact typeconv.TimelineArtifact) (*browser_automation_studio_v1.TimelineArtifact, error) {
	pb := &browser_automation_studio_v1.TimelineArtifact{
		Id:          artifact.ID,
		Type:        typeconv.StringToArtifactType(artifact.Type),
		StorageUrl:  artifact.StorageURL,
		ContentType: artifact.ContentType,
	}
	if artifact.Label != "" {
		label := artifact.Label
		pb.Label = &label
	}
	if artifact.ThumbnailURL != "" {
		thumb := artifact.ThumbnailURL
		pb.ThumbnailUrl = &thumb
	}
	if artifact.SizeBytes != nil {
		pb.SizeBytes = artifact.SizeBytes
	}
	if artifact.StepIndex != nil {
		stepIndex := int32(*artifact.StepIndex)
		pb.StepIndex = &stepIndex
	}
	if payload := toJsonValueMap(artifact.Payload); len(payload) > 0 {
		pb.Payload = payload
	}
	return pb, nil
}

func convertAssertion(assertion *autocontracts.AssertionOutcome) (*browser_automation_studio_v1.AssertionResult, error) {
	if assertion == nil {
		return nil, nil
	}
	pb := &browser_automation_studio_v1.AssertionResult{
		Mode:          typeconv.StringToAssertionMode(assertion.Mode),
		Selector:      assertion.Selector,
		Success:       assertion.Success,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
	}
	if assertion.Message != "" {
		pb.Message = &assertion.Message
	}

	if assertion.Expected != nil {
		pb.Expected = toJsonValue(assertion.Expected)
	}

	if assertion.Actual != nil {
		pb.Actual = toJsonValue(assertion.Actual)
	}

	return pb, nil
}

func toJsonValueMap(source map[string]any) map[string]*commonv1.JsonValue {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]*commonv1.JsonValue, len(source))
	for key, value := range source {
		if jsonVal := toJsonValue(value); jsonVal != nil {
			result[key] = jsonVal
		}
	}
	return result
}

// mapToTriggerMetadata converts a map to the typed TriggerMetadata proto.
func mapToTriggerMetadata(source map[string]any) *browser_automation_studio_v1.TriggerMetadata {
	if len(source) == 0 {
		return nil
	}
	meta := &browser_automation_studio_v1.TriggerMetadata{}
	if v, ok := source["user_id"].(string); ok && v != "" {
		meta.UserId = &v
	}
	if v, ok := source["client_id"].(string); ok && v != "" {
		meta.ClientId = &v
	}
	if v, ok := source["schedule_id"].(string); ok && v != "" {
		meta.ScheduleId = &v
	}
	if v, ok := source["webhook_id"].(string); ok && v != "" {
		meta.WebhookId = &v
	}
	if v, ok := source["external_request_id"].(string); ok && v != "" {
		meta.ExternalRequestId = &v
	}
	if v, ok := source["source_ip"].(string); ok && v != "" {
		meta.SourceIp = &v
	}
	if v, ok := source["user_agent"].(string); ok && v != "" {
		meta.UserAgent = &v
	}
	return meta
}

// mapToExecutionParameters converts a map to the typed ExecutionParameters proto.
func mapToExecutionParameters(source map[string]any) *browser_automation_studio_v1.ExecutionParameters {
	if len(source) == 0 {
		return nil
	}
	params := &browser_automation_studio_v1.ExecutionParameters{}
	if v, ok := source["start_url"].(string); ok && v != "" {
		params.StartUrl = &v
	}
	if v, ok := source["variables"].(map[string]any); ok {
		params.Variables = make(map[string]string)
		for k, val := range v {
			if s, ok := val.(string); ok {
				params.Variables[k] = s
			}
		}
	}
	if v, ok := source["viewport_width"]; ok {
		if i := toInt32Val(v); i != 0 {
			params.ViewportWidth = &i
		}
	}
	if v, ok := source["viewport_height"]; ok {
		if i := toInt32Val(v); i != 0 {
			params.ViewportHeight = &i
		}
	}
	if v, ok := source["timeout_ms"]; ok {
		if i := toInt32Val(v); i != 0 {
			params.TimeoutMs = &i
		}
	}
	if v, ok := source["headless"].(bool); ok {
		params.Headless = &v
	}
	if v, ok := source["user_agent"].(string); ok && v != "" {
		params.UserAgent = &v
	}
	if v, ok := source["locale"].(string); ok && v != "" {
		params.Locale = &v
	}
	return params
}

// mapToExecutionResult converts a map to the typed ExecutionResult proto.
func mapToExecutionResult(source map[string]any) *browser_automation_studio_v1.ExecutionResult {
	if len(source) == 0 {
		return nil
	}
	result := &browser_automation_studio_v1.ExecutionResult{}
	if v, ok := source["success"].(bool); ok {
		result.Success = v
	}
	if v, ok := source["steps_executed"]; ok {
		result.StepsExecuted = toInt32Val(v)
	}
	if v, ok := source["steps_failed"]; ok {
		result.StepsFailed = toInt32Val(v)
	}
	if v, ok := source["final_url"].(string); ok && v != "" {
		result.FinalUrl = &v
	}
	if v, ok := source["error"].(string); ok && v != "" {
		result.Error = &v
	}
	if v, ok := source["error_code"].(string); ok && v != "" {
		result.ErrorCode = &v
	}
	if v, ok := source["extracted_data"].(map[string]any); ok {
		result.ExtractedData = toJsonValueMap(v)
	}
	if v, ok := source["screenshot_artifacts"].(map[string]any); ok {
		result.ScreenshotArtifacts = make(map[int32]string)
		for k, val := range v {
			if s, ok := val.(string); ok {
				// Parse key as int
				var idx int
				if _, err := fmt.Sscanf(k, "%d", &idx); err == nil {
					result.ScreenshotArtifacts[int32(idx)] = s
				}
			}
		}
	}
	return result
}

// toInt32Val converts various numeric types to int32.
func toInt32Val(v any) int32 {
	switch t := v.(type) {
	case int:
		return int32(t)
	case int32:
		return t
	case int64:
		return int32(t)
	case float64:
		return int32(t)
	case float32:
		return int32(t)
	}
	return 0
}

func toJsonValue(value any) *commonv1.JsonValue {
	switch v := value.(type) {
	case nil:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case *structpb.Value:
		return toJsonValue(v.AsInterface())
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: v}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)}}
	case int8:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)}}
	case int16:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: v}}
	case uint:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)}}
	case uint32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)}}
	case uint64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(v)}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(v)}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: v}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: v}}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: i}}
		}
		if f, err := v.Float64(); err == nil {
			return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: f}}
		}
		return nil
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: v}}
	case map[string]any:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{ObjectValue: toJsonObject(v)}}
	case []any:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{ListValue: toJsonList(v)}}
	default:
		// Attempt to JSON round-trip unknown types into a generic shape.
		raw, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var tmp any
		if err := json.Unmarshal(raw, &tmp); err != nil {
			return nil
		}
		return toJsonValue(tmp)
	}
}

func toJsonObjectFromAny(value any) *commonv1.JsonObject {
	switch v := value.(type) {
	case map[string]any:
		return toJsonObject(v)
	default:
		if jsonVal := toJsonValue(v); jsonVal != nil {
			return jsonVal.GetObjectValue()
		}
		return nil
	}
}

func toJsonObject(source map[string]any) *commonv1.JsonObject {
	if len(source) == 0 {
		return nil
	}
	result := &commonv1.JsonObject{
		Fields: make(map[string]*commonv1.JsonValue, len(source)),
	}
	for key, value := range source {
		if jsonVal := toJsonValue(value); jsonVal != nil {
			result.Fields[key] = jsonVal
		}
	}
	return result
}

func toJsonList(items []any) *commonv1.JsonList {
	if len(items) == 0 {
		return nil
	}
	result := &commonv1.JsonList{
		Values: make([]*commonv1.JsonValue, 0, len(items)),
	}
	for _, item := range items {
		if jsonVal := toJsonValue(item); jsonVal != nil {
			result.Values = append(result.Values, jsonVal)
		}
	}
	return result
}
