package protoconv

import (
	"encoding/json"
	"fmt"
	"strings"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// String to enum converters

func stringToExecutionStatus(s string) browser_automation_studio_v1.ExecutionStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "EXECUTION_STATUS_PENDING":
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_PENDING
	case "RUNNING", "EXECUTION_STATUS_RUNNING":
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case "COMPLETED", "EXECUTION_STATUS_COMPLETED":
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case "FAILED", "EXECUTION_STATUS_FAILED":
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_FAILED
	case "CANCELLED", "CANCELED", "EXECUTION_STATUS_CANCELLED":
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

func stringToTriggerType(s string) browser_automation_studio_v1.TriggerType {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "MANUAL", "TRIGGER_TYPE_MANUAL":
		return browser_automation_studio_v1.TriggerType_TRIGGER_TYPE_MANUAL
	case "SCHEDULED", "TRIGGER_TYPE_SCHEDULED":
		return browser_automation_studio_v1.TriggerType_TRIGGER_TYPE_SCHEDULED
	case "API", "TRIGGER_TYPE_API":
		return browser_automation_studio_v1.TriggerType_TRIGGER_TYPE_API
	case "WEBHOOK", "TRIGGER_TYPE_WEBHOOK":
		return browser_automation_studio_v1.TriggerType_TRIGGER_TYPE_WEBHOOK
	default:
		return browser_automation_studio_v1.TriggerType_TRIGGER_TYPE_UNSPECIFIED
	}
}

func stringToExportStatus(s string) browser_automation_studio_v1.ExportStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "READY", "EXPORT_STATUS_READY":
		return browser_automation_studio_v1.ExportStatus_EXPORT_STATUS_READY
	case "PENDING", "NOT_READY", "EXPORT_STATUS_PENDING":
		return browser_automation_studio_v1.ExportStatus_EXPORT_STATUS_PENDING
	case "ERROR", "EXPORT_STATUS_ERROR":
		return browser_automation_studio_v1.ExportStatus_EXPORT_STATUS_ERROR
	case "UNAVAILABLE", "EXPORT_STATUS_UNAVAILABLE":
		return browser_automation_studio_v1.ExportStatus_EXPORT_STATUS_UNAVAILABLE
	default:
		return browser_automation_studio_v1.ExportStatus_EXPORT_STATUS_UNSPECIFIED
	}
}

func stringToStepType(s string) browser_automation_studio_v1.StepType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "navigate", "step_type_navigate":
		return browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE
	case "click", "step_type_click":
		return browser_automation_studio_v1.StepType_STEP_TYPE_CLICK
	case "input", "type", "step_type_input":
		return browser_automation_studio_v1.StepType_STEP_TYPE_INPUT
	case "assert", "step_type_assert":
		return browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT
	case "subflow", "step_type_subflow":
		return browser_automation_studio_v1.StepType_STEP_TYPE_SUBFLOW
	case "custom", "step_type_custom":
		return browser_automation_studio_v1.StepType_STEP_TYPE_CUSTOM
	// Map other step types to CUSTOM since they're not in the proto
	case "wait", "extract", "screenshot", "scroll", "select", "hover", "keyboard", "condition", "loop":
		return browser_automation_studio_v1.StepType_STEP_TYPE_CUSTOM
	default:
		return browser_automation_studio_v1.StepType_STEP_TYPE_UNSPECIFIED
	}
}

func stringToStepStatus(s string) browser_automation_studio_v1.StepStatus {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "PENDING", "STEP_STATUS_PENDING":
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_PENDING
	case "RUNNING", "STEP_STATUS_RUNNING":
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_RUNNING
	case "COMPLETED", "STEP_STATUS_COMPLETED":
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED
	case "FAILED", "STEP_STATUS_FAILED":
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_FAILED
	case "CANCELLED", "STEP_STATUS_CANCELLED":
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_CANCELLED
	case "SKIPPED", "STEP_STATUS_SKIPPED":
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_SKIPPED
	case "RETRYING", "STEP_STATUS_RETRYING":
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_RETRYING
	default:
		return browser_automation_studio_v1.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

func stringToLogLevel(s string) browser_automation_studio_v1.LogLevel {
	normalized := strings.ToUpper(strings.TrimSpace(s))
	switch normalized {
	case "DEBUG", "LOG_LEVEL_DEBUG":
		return browser_automation_studio_v1.LogLevel_LOG_LEVEL_DEBUG
	case "INFO", "LOG_LEVEL_INFO":
		return browser_automation_studio_v1.LogLevel_LOG_LEVEL_INFO
	case "WARN", "WARNING", "LOG_LEVEL_WARN":
		return browser_automation_studio_v1.LogLevel_LOG_LEVEL_WARN
	case "ERROR", "LOG_LEVEL_ERROR":
		return browser_automation_studio_v1.LogLevel_LOG_LEVEL_ERROR
	default:
		return browser_automation_studio_v1.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

func stringToArtifactType(s string) browser_automation_studio_v1.ArtifactType {
	normalized := strings.ToLower(strings.TrimSpace(s))
	switch normalized {
	case "screenshot", "artifact_type_screenshot":
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_SCREENSHOT
	case "dom", "dom_snapshot", "artifact_type_dom_snapshot":
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_DOM_SNAPSHOT
	case "timeline_frame", "artifact_type_timeline_frame":
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_TIMELINE_FRAME
	case "console_log", "artifact_type_console_log":
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_CONSOLE_LOG
	case "network_event", "artifact_type_network_event":
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT
	case "trace", "artifact_type_trace":
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_TRACE
	case "custom", "artifact_type_custom", "extracted_data", "video", "har":
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_CUSTOM
	default:
		return browser_automation_studio_v1.ArtifactType_ARTIFACT_TYPE_UNSPECIFIED
	}
}

// ExecutionToProto converts a database.Execution into the generated proto message.
func ExecutionToProto(execution *database.Execution) (*browser_automation_studio_v1.Execution, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is nil")
	}

	triggerMetadata, err := toStructMap(map[string]any(execution.TriggerMetadata), "trigger_metadata")
	if err != nil {
		return nil, err
	}

	parameters, err := toStructMap(map[string]any(execution.Parameters), "parameters")
	if err != nil {
		return nil, err
	}

	result, err := toStructMap(map[string]any(execution.Result), "result")
	if err != nil {
		return nil, err
	}

	pb := &browser_automation_studio_v1.Execution{
		ExecutionId:     execution.ID.String(),
		WorkflowId:      execution.WorkflowID.String(),
		WorkflowVersion: int32(execution.WorkflowVersion),
		Status:          stringToExecutionStatus(execution.Status),
		TriggerType:     stringToTriggerType(execution.TriggerType),
		Progress:        int32(execution.Progress),
		CurrentStep:     execution.CurrentStep,
		StartedAt:       timestamppb.New(execution.StartedAt),
		CreatedAt:       timestamppb.New(execution.CreatedAt),
		UpdatedAt:       timestamppb.New(execution.UpdatedAt),
	}

	if len(triggerMetadata) > 0 {
		pb.TriggerMetadata = triggerMetadata
	}
	if typed := toJsonValueMap(map[string]any(execution.TriggerMetadata)); len(typed) > 0 {
		pb.TriggerMetadataTyped = typed
	}
	if len(parameters) > 0 {
		pb.Parameters = parameters
	}
	if typed := toJsonValueMap(map[string]any(execution.Parameters)); len(typed) > 0 {
		pb.ParametersTyped = typed
	}
	if execution.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*execution.CompletedAt)
	}
	if execution.LastHeartbeat != nil {
		pb.LastHeartbeat = timestamppb.New(*execution.LastHeartbeat)
	}
	if execution.Error.Valid {
		errMsg := execution.Error.String
		pb.Error = &errMsg
	}
	if len(result) > 0 {
		pb.Result = result
	}
	if typed := toJsonValueMap(map[string]any(execution.Result)); len(typed) > 0 {
		pb.ResultTyped = typed
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
		Status:      stringToExecutionStatus(execution.Status),
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

	var pkg *structpb.Struct
	if preview.Package != nil {
		raw, err := json.Marshal(preview.Package)
		if err != nil {
			return nil, fmt.Errorf("package marshal: %w", err)
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("package unmarshal: %w", err)
		}
		pkg, err = structpb.NewStruct(m)
		if err != nil {
			return nil, fmt.Errorf("package: %w", err)
		}
	}

	return &browser_automation_studio_v1.ExecutionExportPreview{
		ExecutionId:         preview.ExecutionID.String(),
		SpecId:              preview.SpecID,
		Status:              stringToExportStatus(preview.Status),
		Message:             preview.Message,
		CapturedFrameCount:  int32(preview.CapturedFrameCount),
		AvailableAssetCount: int32(preview.AvailableAssetCount),
		TotalDurationMs:     int32(preview.TotalDurationMs),
		Package:             pkg,
		PackageTyped:        toJsonObjectFromAny(preview.Package),
	}, nil
}

// ScreenshotsToProto converts database screenshots to the proto response.
func ScreenshotsToProto(screenshots []*database.Screenshot) (*browser_automation_studio_v1.GetScreenshotsResponse, error) {
	if len(screenshots) == 0 {
		return &browser_automation_studio_v1.GetScreenshotsResponse{}, nil
	}

	result := make([]*browser_automation_studio_v1.Screenshot, 0, len(screenshots))
	for idx, shot := range screenshots {
		if shot == nil {
			return nil, fmt.Errorf("screenshots[%d] is nil", idx)
		}

		var stepIndex int32
		if shot.Metadata != nil {
			if v, ok := shot.Metadata["step_index"]; ok {
				switch t := v.(type) {
				case int:
					stepIndex = int32(t)
				case int32:
					stepIndex = t
				case int64:
					stepIndex = int32(t)
				case float64:
					stepIndex = int32(t)
				case float32:
					stepIndex = int32(t)
				}
			}
		}

		thumbURL := shot.ThumbnailURL
		if thumbURL == "" && shot.Metadata != nil {
			if v, ok := shot.Metadata["thumbnail_url"].(string); ok {
				thumbURL = v
			}
		}

		result = append(result, &browser_automation_studio_v1.Screenshot{
			Id:           shot.ID.String(),
			ExecutionId:  shot.ExecutionID.String(),
			StepName:     shot.StepName,
			StepIndex:    stepIndex,
			Timestamp:    timestamppb.New(shot.Timestamp),
			StorageUrl:   shot.StorageURL,
			ThumbnailUrl: thumbURL,
			Width:        int32(shot.Width),
			Height:       int32(shot.Height),
			SizeBytes:    shot.SizeBytes,
		})
	}

	return &browser_automation_studio_v1.GetScreenshotsResponse{Screenshots: result}, nil
}

// TimelineToProto converts the replay-focused ExecutionTimeline into the proto message.
func TimelineToProto(timeline *export.ExecutionTimeline) (*browser_automation_studio_v1.ExecutionTimeline, error) {
	if timeline == nil {
		return nil, fmt.Errorf("timeline is nil")
	}

	pb := &browser_automation_studio_v1.ExecutionTimeline{
		ExecutionId: timeline.ExecutionID.String(),
		WorkflowId:  timeline.WorkflowID.String(),
		Status:      stringToExecutionStatus(timeline.Status),
		Progress:    int32(timeline.Progress),
		StartedAt:   timestamppb.New(timeline.StartedAt),
	}

	if timeline.CompletedAt != nil {
		pb.CompletedAt = timestamppb.New(*timeline.CompletedAt)
	}

	for idx, frame := range timeline.Frames {
		pbFrame, err := timelineFrameToProto(frame)
		if err != nil {
			return nil, fmt.Errorf("frame %d: %w", idx, err)
		}
		pb.Frames = append(pb.Frames, pbFrame)
	}

	for _, log := range timeline.Logs {
		pb.Logs = append(pb.Logs, timelineLogToProto(log))
	}

	return pb, nil
}

func timelineFrameToProto(frame export.TimelineFrame) (*browser_automation_studio_v1.TimelineFrame, error) {
	extractedPreview, err := toProtoValue(frame.ExtractedDataPreview, "frames.extracted_data_preview")
	if err != nil {
		return nil, err
	}

	var startedAt *timestamppb.Timestamp
	if frame.StartedAt != nil {
		startedAt = timestamppb.New(*frame.StartedAt)
	}
	var completedAt *timestamppb.Timestamp
	if frame.CompletedAt != nil {
		completedAt = timestamppb.New(*frame.CompletedAt)
	}

	pbFrame := &browser_automation_studio_v1.TimelineFrame{
		StepIndex:                 int32(frame.StepIndex),
		NodeId:                    frame.NodeID,
		StepType:                  stringToStepType(frame.StepType),
		Status:                    stringToStepStatus(frame.Status),
		Success:                   frame.Success,
		DurationMs:                int32(frame.DurationMs),
		TotalDurationMs:           int32(frame.TotalDurationMs),
		Progress:                  int32(frame.Progress),
		StartedAt:                 startedAt,
		CompletedAt:               completedAt,
		FinalUrl:                  frame.FinalURL,
		ConsoleLogCount:           int32(frame.ConsoleLogCount),
		NetworkEventCount:         int32(frame.NetworkEventCount),
		ExtractedDataPreview:      extractedPreview,
		ExtractedDataPreviewTyped: toJsonValue(frame.ExtractedDataPreview),
		ZoomFactor:                frame.ZoomFactor,
		RetryAttempt:              int32(frame.RetryAttempt),
		RetryMaxAttempts:          int32(frame.RetryMaxAttempts),
		RetryDelayMs:              int32(frame.RetryDelayMs),
		RetryBackoffFactor:        frame.RetryBackoffFactor,
		RetryHistory:              convertRetryHistory(frame.RetryHistory),
		DomSnapshotPreview:        frame.DomSnapshotPreview,
	}
	if frame.RetryConfigured != 0 {
		configured := true
		pbFrame.RetryConfigured = &configured
	}
	if frame.Error != "" {
		errMsg := frame.Error
		pbFrame.Error = &errMsg
	}

	if frame.HighlightRegions != nil {
		for _, region := range frame.HighlightRegions {
			pbFrame.HighlightRegions = append(pbFrame.HighlightRegions, convertHighlightRegion(region))
		}
	}

	if frame.MaskRegions != nil {
		for _, region := range frame.MaskRegions {
			pbFrame.MaskRegions = append(pbFrame.MaskRegions, convertMaskRegion(region))
		}
	}

	if frame.FocusedElement != nil {
		pbFrame.FocusedElement = convertElementFocus(frame.FocusedElement)
	}

	if frame.ElementBoundingBox != nil {
		pbFrame.ElementBoundingBox = convertBoundingBox(frame.ElementBoundingBox)
	}

	if frame.ClickPosition != nil {
		pbFrame.ClickPosition = convertPoint(frame.ClickPosition)
	}

	if frame.CursorTrail != nil {
		for _, pt := range frame.CursorTrail {
			point := pt
			pbFrame.CursorTrail = append(pbFrame.CursorTrail, convertPoint(&point))
		}
	}

	if frame.Screenshot != nil {
		pbFrame.Screenshot = convertScreenshot(frame.Screenshot)
	}

	if frame.Artifacts != nil {
		for _, artifact := range frame.Artifacts {
			pbArtifact, err := convertArtifact(artifact)
			if err != nil {
				return nil, fmt.Errorf("artifact %s: %w", artifact.ID, err)
			}
			pbFrame.Artifacts = append(pbFrame.Artifacts, pbArtifact)
		}
	}

	if frame.Assertion != nil {
		assertion, err := convertAssertion(frame.Assertion)
		if err != nil {
			return nil, err
		}
		pbFrame.Assertion = assertion
	}

	if frame.DomSnapshot != nil {
		dom, err := convertArtifact(*frame.DomSnapshot)
		if err != nil {
			return nil, fmt.Errorf("dom_snapshot: %w", err)
		}
		pbFrame.DomSnapshot = dom
	}

	return pbFrame, nil
}

func timelineLogToProto(log export.TimelineLog) *browser_automation_studio_v1.TimelineLog {
	pb := &browser_automation_studio_v1.TimelineLog{
		Id:        log.ID,
		Level:     stringToLogLevel(log.Level),
		Message:   log.Message,
		Timestamp: timestamppb.New(log.Timestamp),
	}
	if log.StepName != "" {
		stepName := log.StepName
		pb.StepName = &stepName
	}
	return pb
}

func convertRetryHistory(entries []typeconv.RetryHistoryEntry) []*browser_automation_studio_v1.RetryHistoryEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]*browser_automation_studio_v1.RetryHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, &browser_automation_studio_v1.RetryHistoryEntry{
			Attempt:        int32(entry.Attempt),
			Success:        entry.Success,
			DurationMs:     int32(entry.DurationMs),
			CallDurationMs: int32(entry.CallDurationMs),
			Error:          entry.Error,
		})
	}
	return result
}

func convertHighlightRegion(region autocontracts.HighlightRegion) *browser_automation_studio_v1.HighlightRegion {
	pb := &browser_automation_studio_v1.HighlightRegion{
		Selector: region.Selector,
		Padding:  int32(region.Padding),
		Color:    region.Color,
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
	payload, err := toStructMap(artifact.Payload, "artifacts.payload")
	if err != nil {
		return nil, err
	}

	pb := &browser_automation_studio_v1.TimelineArtifact{
		Id:          artifact.ID,
		Type:        stringToArtifactType(artifact.Type),
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
	if len(payload) > 0 {
		pb.Payload = payload
	}
	if typed := toJsonValueMap(artifact.Payload); len(typed) > 0 {
		pb.PayloadTyped = typed
	}
	return pb, nil
}

func convertAssertion(assertion *autocontracts.AssertionOutcome) (*browser_automation_studio_v1.AssertionOutcome, error) {
	if assertion == nil {
		return nil, nil
	}
	pb := &browser_automation_studio_v1.AssertionOutcome{
		Mode:          assertion.Mode,
		Selector:      assertion.Selector,
		Success:       assertion.Success,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
		Message:       assertion.Message,
	}

	if assertion.Expected != nil {
		val, err := toProtoValue(assertion.Expected, "assertion.expected")
		if err != nil {
			return nil, err
		}
		pb.Expected = val
	}

	if assertion.Actual != nil {
		val, err := toProtoValue(assertion.Actual, "assertion.actual")
		if err != nil {
			return nil, err
		}
		pb.Actual = val
	}

	return pb, nil
}

func toStructMap(source map[string]any, field string) (map[string]*structpb.Value, error) {
	if len(source) == 0 {
		return nil, nil
	}
	result := make(map[string]*structpb.Value, len(source))
	for key, value := range source {
		val, err := structpb.NewValue(value)
		if err != nil {
			return nil, fmt.Errorf("%s[%s]: %w", field, key, err)
		}
		result[key] = val
	}
	return result, nil
}

func toProtoValue(value any, field string) (*structpb.Value, error) {
	if value == nil {
		return nil, nil
	}
	val, err := structpb.NewValue(value)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", field, err)
	}
	return val, nil
}

func toJsonValueMap(source map[string]any) map[string]*browser_automation_studio_v1.JsonValue {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]*browser_automation_studio_v1.JsonValue, len(source))
	for key, value := range source {
		if jsonVal := toJsonValue(value); jsonVal != nil {
			result[key] = jsonVal
		}
	}
	return result
}

func toJsonValue(value any) *browser_automation_studio_v1.JsonValue {
	switch v := value.(type) {
	case nil:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_NullValue{NullValue: structpb.NullValue_NULL_VALUE}}
	case *structpb.Value:
		return toJsonValue(v.AsInterface())
	case bool:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_BoolValue{BoolValue: v}}
	case int:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(v)}}
	case int8:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(v)}}
	case int16:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(v)}}
	case int32:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(v)}}
	case int64:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: v}}
	case uint:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(v)}}
	case uint32:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(v)}}
	case uint64:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: int64(v)}}
	case float32:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_DoubleValue{DoubleValue: float64(v)}}
	case float64:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_DoubleValue{DoubleValue: v}}
	case string:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_StringValue{StringValue: v}}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_IntValue{IntValue: i}}
		}
		if f, err := v.Float64(); err == nil {
			return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_DoubleValue{DoubleValue: f}}
		}
		return nil
	case []byte:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_BytesValue{BytesValue: v}}
	case map[string]any:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_ObjectValue{ObjectValue: toJsonObject(v)}}
	case []any:
		return &browser_automation_studio_v1.JsonValue{Kind: &browser_automation_studio_v1.JsonValue_ListValue{ListValue: toJsonList(v)}}
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

func toJsonObjectFromAny(value any) *browser_automation_studio_v1.JsonObject {
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

func toJsonObject(source map[string]any) *browser_automation_studio_v1.JsonObject {
	if len(source) == 0 {
		return nil
	}
	result := &browser_automation_studio_v1.JsonObject{
		Fields: make(map[string]*browser_automation_studio_v1.JsonValue, len(source)),
	}
	for key, value := range source {
		if jsonVal := toJsonValue(value); jsonVal != nil {
			result.Fields[key] = jsonVal
		}
	}
	return result
}

func toJsonList(items []any) *browser_automation_studio_v1.JsonList {
	if len(items) == 0 {
		return nil
	}
	result := &browser_automation_studio_v1.JsonList{
		Values: make([]*browser_automation_studio_v1.JsonValue, 0, len(items)),
	}
	for _, item := range items {
		if jsonVal := toJsonValue(item); jsonVal != nil {
			result.Values = append(result.Values, jsonVal)
		}
	}
	return result
}
