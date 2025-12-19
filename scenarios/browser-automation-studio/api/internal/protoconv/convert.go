package protoconv

import (
	"fmt"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// ExecutionToProto converts a database.ExecutionIndex (DB index-only row) into the generated proto message.
// Detailed execution data is sourced from filesystem artifacts, not the database.
func ExecutionToProto(execution *database.ExecutionIndex) (*basexecution.Execution, error) {
	if execution == nil {
		return nil, fmt.Errorf("execution is nil")
	}

	pb := &basexecution.Execution{
		ExecutionId:     execution.ID.String(),
		WorkflowId:      execution.WorkflowID.String(),
		Status:          StringToExecutionStatus(execution.Status),
		StartedAt:       autocontracts.TimeToTimestamp(execution.StartedAt),
		CreatedAt:       autocontracts.TimeToTimestamp(execution.CreatedAt),
		UpdatedAt:       autocontracts.TimeToTimestamp(execution.UpdatedAt),
	}

	pb.CompletedAt = autocontracts.TimePtrToTimestamp(execution.CompletedAt)
	if execution.ErrorMessage != "" {
		errMsg := execution.ErrorMessage
		pb.Error = &errMsg
	}

	return pb, nil
}

// ExecutionExportPreviewToProto converts the workflow.ExecutionExportPreview to the proto message.
func ExecutionExportPreviewToProto(preview *workflow.ExecutionExportPreview) (*basexecution.ExecutionExportPreview, error) {
	if preview == nil {
		return nil, fmt.Errorf("preview is nil")
	}

	return &basexecution.ExecutionExportPreview{
		ExecutionId:         preview.ExecutionID.String(),
		SpecId:              preview.SpecID,
		Status:              StringToExportStatus(preview.Status),
		Message:             preview.Message,
		CapturedFrameCount:  int32(preview.CapturedFrameCount),
		AvailableAssetCount: int32(preview.AvailableAssetCount),
		TotalDurationMs:     int32(preview.TotalDurationMs),
		Package:             typeconv.ToJsonObjectFromAny(preview.Package),
	}, nil
}

// TimelineToProto converts the replay-focused ExecutionTimeline into the proto message.
func TimelineToProto(timeline *export.ExecutionTimeline) (*bastimeline.ExecutionTimeline, error) {
	if timeline == nil {
		return nil, fmt.Errorf("timeline is nil")
	}

	pb := &bastimeline.ExecutionTimeline{
		ExecutionId: timeline.ExecutionID.String(),
		WorkflowId:  timeline.WorkflowID.String(),
		Status:      StringToExecutionStatus(timeline.Status),
		Progress:    int32(timeline.Progress),
		StartedAt:   autocontracts.TimeToTimestamp(timeline.StartedAt),
	}

	pb.CompletedAt = autocontracts.TimePtrToTimestamp(timeline.CompletedAt)

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

// timelineFrameToEntry converts an export.TimelineFrame to a bastimeline.TimelineEntry.
// This is the unified format for timeline data.
func timelineFrameToEntry(frame export.TimelineFrame) (*bastimeline.TimelineEntry, error) {
	// Build base entry
	stepIndex := int32(frame.StepIndex)
	// Generate entry ID from node_id and step_index since export.TimelineFrame doesn't have an ID field
	entryID := fmt.Sprintf("%s-step-%d", frame.NodeID, frame.StepIndex)
	entry := &bastimeline.TimelineEntry{
		Id:          entryID,
		SequenceNum: int32(frame.StepIndex), // Use step index as sequence for now
		StepIndex:   &stepIndex,
	}

	// Set node_id if available
	if frame.NodeID != "" {
		entry.NodeId = &frame.NodeID
	}

	// Set timestamp from StartedAt
	entry.Timestamp = autocontracts.TimePtrToTimestamp(frame.StartedAt)

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
		entry.Action = &basactions.ActionDefinition{
			Type: StringToActionType(frame.StepType),
		}
	}

	// Build Telemetry with visual data (screenshots, cursor trails, regions, etc.)
	entry.Telemetry = buildTelemetryFromFrame(frame)

	// Build EventContext
	entry.Context = &basbase.EventContext{
		Success: &frame.Success,
	}
	// Note: execution_id is not available in export.TimelineFrame - it's set at the parent timeline level

	// Add error to context if present
	if frame.Error != "" {
		entry.Context.Error = &frame.Error
	}

	// Build RetryStatus from individual retry fields
	if frame.RetryConfigured != 0 || frame.RetryAttempt > 0 || frame.RetryMaxAttempts > 0 {
		entry.Context.RetryStatus = &basbase.RetryStatus{
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
	entry.Aggregates = &bastimeline.TimelineEntryAggregates{
		Status:            StringToStepStatus(frame.Status),
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
		entry.Aggregates.ExtractedDataPreview = typeconv.AnyToJsonValue(frame.ExtractedDataPreview)
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

// buildTelemetryFromFrame creates ActionTelemetry from a TimelineFrame.
func buildTelemetryFromFrame(frame export.TimelineFrame) *basdomain.ActionTelemetry {
	tel := &basdomain.ActionTelemetry{
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
			tel.CursorTrail = append(tel.CursorTrail, convertPoint(pt))
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

func timelineLogToProto(log export.TimelineLog) *bastimeline.TimelineLog {
	pb := &bastimeline.TimelineLog{
		Id:        log.ID,
		Level:     StringToLogLevel(log.Level),
		Message:   log.Message,
		Timestamp: autocontracts.TimeToTimestamp(log.Timestamp),
	}
	if log.StepName != "" {
		stepName := log.StepName
		pb.StepName = &stepName
	}
	return pb
}

func convertRetryHistory(entries []typeconv.RetryHistoryEntry) []*basbase.RetryAttempt {
	if len(entries) == 0 {
		return nil
	}
	result := make([]*basbase.RetryAttempt, 0, len(entries))
	for _, entry := range entries {
		attempt := &basbase.RetryAttempt{
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

func convertHighlightRegion(region *autocontracts.HighlightRegion) *basdomain.HighlightRegion {
	if region == nil {
		return nil
	}
	pb := &basdomain.HighlightRegion{
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

func convertMaskRegion(region *autocontracts.MaskRegion) *basdomain.MaskRegion {
	if region == nil {
		return nil
	}
	pb := &basdomain.MaskRegion{
		Selector: region.Selector,
		Opacity:  region.Opacity,
	}
	if region.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(region.BoundingBox)
	}
	return pb
}

func convertElementFocus(focus *autocontracts.ElementFocus) *bastimeline.ElementFocus {
	pb := &bastimeline.ElementFocus{
		Selector: focus.Selector,
	}
	if focus.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(focus.BoundingBox)
	}
	return pb
}

func convertBoundingBox(bbox *autocontracts.BoundingBox) *basbase.BoundingBox {
	if bbox == nil {
		return nil
	}
	return &basbase.BoundingBox{
		X:      bbox.X,
		Y:      bbox.Y,
		Width:  bbox.Width,
		Height: bbox.Height,
	}
}

func convertPoint(pt *autocontracts.Point) *basbase.Point {
	if pt == nil {
		return nil
	}
	return &basbase.Point{
		X: pt.X,
		Y: pt.Y,
	}
}

func convertScreenshot(screenshot *typeconv.TimelineScreenshot) *basdomain.TimelineScreenshot {
	if screenshot == nil {
		return nil
	}
	pb := &basdomain.TimelineScreenshot{
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

func convertArtifact(artifact typeconv.TimelineArtifact) (*bastimeline.TimelineArtifact, error) {
	pb := &bastimeline.TimelineArtifact{
		Id:          artifact.ID,
		Type:        StringToArtifactType(artifact.Type),
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
	if payload := typeconv.ToJsonValueMap(artifact.Payload); len(payload) > 0 {
		pb.Payload = payload
	}
	return pb, nil
}

func convertAssertion(assertion *autocontracts.AssertionOutcome) (*basbase.AssertionResult, error) {
	if assertion == nil {
		return nil, nil
	}
	pb := &basbase.AssertionResult{
		Mode:          StringToAssertionMode(assertion.Mode),
		Selector:      assertion.Selector,
		Success:       assertion.Success,
		Negated:       assertion.Negated,
		CaseSensitive: assertion.CaseSensitive,
	}
	if assertion.Message != "" {
		pb.Message = &assertion.Message
	}

	if assertion.Expected != nil {
		pb.Expected = typeconv.AnyToJsonValue(assertion.Expected)
	}

	if assertion.Actual != nil {
		pb.Actual = typeconv.AnyToJsonValue(assertion.Actual)
	}

	return pb, nil
}


// mapToTriggerMetadata converts a map to the typed TriggerMetadata proto.
func mapToTriggerMetadata(source map[string]any) *basexecution.TriggerMetadata {
	if len(source) == 0 {
		return nil
	}
	meta := &basexecution.TriggerMetadata{}
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
func mapToExecutionParameters(source map[string]any) *basexecution.ExecutionParameters {
	if len(source) == 0 {
		return nil
	}
	params := &basexecution.ExecutionParameters{}
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
		if i := typeconv.ToInt32Val(v); i != 0 {
			params.ViewportWidth = &i
		}
	}
	if v, ok := source["viewport_height"]; ok {
		if i := typeconv.ToInt32Val(v); i != 0 {
			params.ViewportHeight = &i
		}
	}
	if v, ok := source["timeout_ms"]; ok {
		if i := typeconv.ToInt32Val(v); i != 0 {
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
func mapToExecutionResult(source map[string]any) *basexecution.ExecutionResult {
	if len(source) == 0 {
		return nil
	}
	result := &basexecution.ExecutionResult{}
	if v, ok := source["success"].(bool); ok {
		result.Success = v
	}
	if v, ok := source["steps_executed"]; ok {
		result.StepsExecuted = typeconv.ToInt32Val(v)
	}
	if v, ok := source["steps_failed"]; ok {
		result.StepsFailed = typeconv.ToInt32Val(v)
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
		result.ExtractedData = typeconv.ToJsonValueMap(v)
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

