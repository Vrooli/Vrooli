package protoconv

import (
	"fmt"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	"github.com/vrooli/browser-automation-studio/services/export"
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ExecutionToProto converts a database.Execution into the generated proto message.
func ExecutionToProto(execution *database.Execution) (*basv1.Execution, error) {
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

	pb := &basv1.Execution{
		Id:              execution.ID.String(),
		WorkflowId:      execution.WorkflowID.String(),
		WorkflowVersion: int32(execution.WorkflowVersion),
		Status:          execution.Status,
		TriggerType:     execution.TriggerType,
		Progress:        int32(execution.Progress),
		CurrentStep:     execution.CurrentStep,
		StartedAt:       timestamppb.New(execution.StartedAt),
		CreatedAt:       timestamppb.New(execution.CreatedAt),
		UpdatedAt:       timestamppb.New(execution.UpdatedAt),
	}

	if len(triggerMetadata) > 0 {
		pb.TriggerMetadata = triggerMetadata
	}
	if len(parameters) > 0 {
		pb.Parameters = parameters
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

	return pb, nil
}

// TimelineToProto converts the replay-focused ExecutionTimeline into the proto message.
func TimelineToProto(timeline *export.ExecutionTimeline) (*basv1.ExecutionTimeline, error) {
	if timeline == nil {
		return nil, fmt.Errorf("timeline is nil")
	}

	pb := &basv1.ExecutionTimeline{
		ExecutionId: timeline.ExecutionID.String(),
		WorkflowId:  timeline.WorkflowID.String(),
		Status:      timeline.Status,
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

func timelineFrameToProto(frame export.TimelineFrame) (*basv1.TimelineFrame, error) {
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

	pbFrame := &basv1.TimelineFrame{
		StepIndex:            int32(frame.StepIndex),
		NodeId:               frame.NodeID,
		StepType:             frame.StepType,
		Status:               frame.Status,
		Success:              frame.Success,
		DurationMs:           int32(frame.DurationMs),
		TotalDurationMs:      int32(frame.TotalDurationMs),
		Progress:             int32(frame.Progress),
		StartedAt:            startedAt,
		CompletedAt:          completedAt,
		FinalUrl:             frame.FinalURL,
		Error:                frame.Error,
		ConsoleLogCount:      int32(frame.ConsoleLogCount),
		NetworkEventCount:    int32(frame.NetworkEventCount),
		ExtractedDataPreview: extractedPreview,
		ZoomFactor:           frame.ZoomFactor,
		RetryAttempt:         int32(frame.RetryAttempt),
		RetryMaxAttempts:     int32(frame.RetryMaxAttempts),
		RetryConfigured:      int32(frame.RetryConfigured),
		RetryDelayMs:         int32(frame.RetryDelayMs),
		RetryBackoffFactor:   frame.RetryBackoffFactor,
		RetryHistory:         convertRetryHistory(frame.RetryHistory),
		DomSnapshotPreview:   frame.DomSnapshotPreview,
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

func timelineLogToProto(log export.TimelineLog) *basv1.TimelineLog {
	return &basv1.TimelineLog{
		Id:        log.ID,
		Level:     log.Level,
		Message:   log.Message,
		StepName:  log.StepName,
		Timestamp: timestamppb.New(log.Timestamp),
	}
}

func convertRetryHistory(entries []typeconv.RetryHistoryEntry) []*basv1.RetryHistoryEntry {
	if len(entries) == 0 {
		return nil
	}
	result := make([]*basv1.RetryHistoryEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, &basv1.RetryHistoryEntry{
			Attempt:        int32(entry.Attempt),
			Success:        entry.Success,
			DurationMs:     int32(entry.DurationMs),
			CallDurationMs: int32(entry.CallDurationMs),
			Error:          entry.Error,
		})
	}
	return result
}

func convertHighlightRegion(region autocontracts.HighlightRegion) *basv1.HighlightRegion {
	pb := &basv1.HighlightRegion{
		Selector: region.Selector,
		Padding:  int32(region.Padding),
		Color:    region.Color,
	}
	if region.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(region.BoundingBox)
	}
	return pb
}

func convertMaskRegion(region autocontracts.MaskRegion) *basv1.MaskRegion {
	pb := &basv1.MaskRegion{
		Selector: region.Selector,
		Opacity:  region.Opacity,
	}
	if region.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(region.BoundingBox)
	}
	return pb
}

func convertElementFocus(focus *autocontracts.ElementFocus) *basv1.ElementFocus {
	pb := &basv1.ElementFocus{
		Selector: focus.Selector,
	}
	if focus.BoundingBox != nil {
		pb.BoundingBox = convertBoundingBox(focus.BoundingBox)
	}
	return pb
}

func convertBoundingBox(bbox *autocontracts.BoundingBox) *basv1.BoundingBox {
	if bbox == nil {
		return nil
	}
	return &basv1.BoundingBox{
		X:      bbox.X,
		Y:      bbox.Y,
		Width:  bbox.Width,
		Height: bbox.Height,
	}
}

func convertPoint(pt *autocontracts.Point) *basv1.Point {
	if pt == nil {
		return nil
	}
	return &basv1.Point{
		X: pt.X,
		Y: pt.Y,
	}
}

func convertScreenshot(screenshot *typeconv.TimelineScreenshot) *basv1.TimelineScreenshot {
	if screenshot == nil {
		return nil
	}
	pb := &basv1.TimelineScreenshot{
		ArtifactId:   screenshot.ArtifactID,
		Url:          screenshot.URL,
		ThumbnailUrl: screenshot.ThumbnailURL,
		Width:        int32(screenshot.Width),
		Height:       int32(screenshot.Height),
		ContentType:  screenshot.ContentType,
	}
	if screenshot.SizeBytes != nil {
		pb.SizeBytes = *screenshot.SizeBytes
	}
	return pb
}

func convertArtifact(artifact typeconv.TimelineArtifact) (*basv1.TimelineArtifact, error) {
	payload, err := toStructMap(artifact.Payload, "artifacts.payload")
	if err != nil {
		return nil, err
	}

	pb := &basv1.TimelineArtifact{
		Id:           artifact.ID,
		Type:         artifact.Type,
		Label:        artifact.Label,
		StorageUrl:   artifact.StorageURL,
		ThumbnailUrl: artifact.ThumbnailURL,
		ContentType:  artifact.ContentType,
	}
	if artifact.SizeBytes != nil {
		pb.SizeBytes = *artifact.SizeBytes
	}
	if artifact.StepIndex != nil {
		stepIndex := int32(*artifact.StepIndex)
		pb.StepIndex = &stepIndex
	}
	if len(payload) > 0 {
		pb.Payload = payload
	}
	return pb, nil
}

func convertAssertion(assertion *autocontracts.AssertionOutcome) (*basv1.AssertionOutcome, error) {
	if assertion == nil {
		return nil, nil
	}
	pb := &basv1.AssertionOutcome{
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
