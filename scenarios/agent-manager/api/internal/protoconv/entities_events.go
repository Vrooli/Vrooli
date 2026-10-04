// Responsibility: retain entities declarations within their original package.
package protoconv

import (
	"agent-manager/internal/domain"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RunEventToProto converts a domain RunEvent to proto RunEvent.
func RunEventToProto(e *domain.RunEvent) *pb.RunEvent {
	if e == nil {
		return nil
	}

	event := &pb.RunEvent{
		Id:        UUIDToString(e.ID),
		RunId:     UUIDToString(e.RunID),
		EventType: RunEventTypeToProto(e.EventType),
		Timestamp: TimestampToProto(e.Timestamp),
		Sequence:  e.Sequence,
	}

	// Convert event data based on type
	switch data := e.Data.(type) {
	case *domain.LogEventData:
		event.Data = &pb.RunEvent_Log{
			Log: &pb.LogEventData{
				Level:   data.Level,
				Message: data.Message,
			},
		}
	case *domain.MessageEventData:
		var pbAttachments []*pb.MessageAttachmentInfo
		for _, att := range data.Attachments {
			pbAttachments = append(pbAttachments, &pb.MessageAttachmentInfo{
				Id:          att.ID,
				FileName:    att.FileName,
				ContentType: att.ContentType,
				Url:         att.URL,
			})
		}
		event.Data = &pb.RunEvent_Message{
			Message: &pb.MessageEventData{
				Role:               data.Role,
				Content:            data.Content,
				Attachments:        pbAttachments,
				MessageId:          data.MessageID,
				ConversationId:     data.ConversationID,
				TurnId:             data.TurnID,
				ProviderOrigin:     data.ProviderOrigin,
				CompletionReason:   data.CompletionReason,
				Terminal:           data.Terminal,
				ParentMessageId:    data.ParentMessageID,
				ProviderEventType:  data.ProviderEventType,
				RawEvidenceRef:     data.RawEvidenceRef,
				EvidenceOnly:       data.EvidenceOnly,
				EvidenceForEventId: data.EvidenceForEventID,
			},
		}
	case *domain.MessageDeletedEventData:
		event.Data = &pb.RunEvent_MessageDeleted{
			MessageDeleted: &pb.MessageDeletedEventData{
				TargetEventId: data.TargetEventID,
			},
		}
	case *domain.ToolCallEventData:
		var input *structpb.Struct
		if len(data.Input) > 0 {
			if parsed, err := structpb.NewStruct(data.Input); err == nil {
				input = parsed
			}
		}
		event.Data = &pb.RunEvent_ToolCall{
			ToolCall: &pb.ToolCallEventData{
				ToolName:   data.ToolName,
				ToolCallId: data.ToolCallID,
				Input:      input,
			},
		}
	case *domain.ToolResultEventData:
		event.Data = &pb.RunEvent_ToolResult{
			ToolResult: &pb.ToolResultEventData{
				ToolName:   data.ToolName,
				ToolCallId: data.ToolCallID,
				Output:     data.Output,
				Error:      data.Error,
				Success:    data.Success,
			},
		}
	case *domain.StatusEventData:
		event.Data = &pb.RunEvent_Status{
			Status: &pb.StatusEventData{
				OldStatus: data.OldStatus,
				NewStatus: data.NewStatus,
				Reason:    data.Reason,
			},
		}
	case *domain.GoalStatusChangedEventData:
		event.Data = &pb.RunEvent_GoalStatusChanged{GoalStatusChanged: &pb.GoalStatusChangedEventData{Objective: data.Objective, Status: data.Status, Iteration: int32(data.Iteration), LastReason: data.LastReason}}
	case *domain.MetricEventData:
		event.Data = &pb.RunEvent_Metric{
			Metric: &pb.MetricEventData{
				Name:  data.Name,
				Value: data.Value,
				Unit:  data.Unit,
				Tags:  data.Tags,
			},
		}
	case *domain.ArtifactEventData:
		event.Data = &pb.RunEvent_Artifact{
			Artifact: &pb.ArtifactEventData{
				Type:     data.Type,
				Path:     data.Path,
				Size:     data.Size,
				MimeType: data.MimeType,
			},
		}
	case *domain.UsageEventData:
		event.Data = &pb.RunEvent_Metric{Metric: &pb.MetricEventData{
			Name:  "usage_tokens",
			Value: float64(data.InputTokens + data.OutputTokens + data.CacheReadTokens + data.CacheCreationTokens),
			Unit:  "tokens",
			Tags:  map[string]string{"model": data.Model, "runnerType": data.RunnerType},
		}}
	case *domain.ChargeEventData:
		value := float64(0)
		if data.AmountMicroUSD != nil {
			value = float64(*data.AmountMicroUSD) / 1_000_000
		}
		event.Data = &pb.RunEvent_Metric{Metric: &pb.MetricEventData{
			Name:  "usage_charge",
			Value: value,
			Unit:  "USD",
			Tags:  map[string]string{"basis": string(data.Basis), "model": data.Model, "runnerType": data.RunnerType},
		}}
	case *domain.ProgressEventData:
		event.Data = &pb.RunEvent_Progress{
			Progress: &pb.ProgressEventData{
				Phase:              RunPhaseToProto(data.Phase),
				PercentComplete:    int32(data.PercentComplete),
				CurrentAction:      data.CurrentAction,
				TurnsCompleted:     int32(data.TurnsCompleted),
				TurnsTotal:         int32(data.TurnsTotal),
				TokensUsed:         int32(data.TokensUsed),
				ElapsedSeconds:     data.ElapsedSeconds,
				EstimatedRemaining: data.EstimatedRemaining,
			},
		}
	case *domain.RateLimitEventData:
		var resetTime *timestamppb.Timestamp
		if data.ResetTime != nil {
			resetTime = TimestampToProto(*data.ResetTime)
		}
		event.Data = &pb.RunEvent_RateLimit{
			RateLimit: &pb.RateLimitEventData{
				LimitType:     data.LimitType,
				ResetTime:     resetTime,
				RetryAfter:    int32(data.RetryAfter),
				CurrentUsed:   int32(data.CurrentUsed),
				Limit:         int32(data.Limit),
				Message:       data.Message,
				Provider:      data.Provider,
				Pool:          data.Pool,
				UsedPercent:   data.UsedPercent,
				WindowMinutes: data.WindowMinutes,
				Provenance:    data.Provenance,
			},
		}
	case *domain.ErrorEventData:
		var details *structpb.Struct
		if len(data.Details) > 0 {
			if parsed, err := structpb.NewStruct(data.Details); err == nil {
				details = parsed
			}
		}
		event.Data = &pb.RunEvent_Error{
			Error: &pb.ErrorEventData{
				Code:       data.Code,
				Message:    data.Message,
				Retryable:  data.Retryable,
				Recovery:   RecoveryActionToProto(data.Recovery),
				StackTrace: data.StackTrace,
				Details:    details,
			},
		}
	}

	return event
}

// RunResultToProto converts the canonical terminal result.
func RunResultToProto(result *domain.RunResult) *pb.RunResult {
	if result == nil {
		return nil
	}
	candidates := make([]*pb.FinalOutputCandidate, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		candidates = append(candidates, &pb.FinalOutputCandidate{
			Id: validUTF8(candidate.ID), EventId: validUTF8(candidate.EventID), Sequence: candidate.Sequence,
			Content: validUTF8(candidate.Content), MessageId: validUTF8(candidate.MessageID),
			ConversationId: validUTF8(candidate.ConversationID), TurnId: validUTF8(candidate.TurnID),
			ProviderOrigin: validUTF8(candidate.ProviderOrigin), CompletionReason: validUTF8(candidate.CompletionReason),
			Terminal: candidate.Terminal, ParentMessageId: validUTF8(candidate.ParentMessageID),
			ProviderEventType: validUTF8(candidate.ProviderEventType), RawEvidenceRef: validUTF8(candidate.RawEvidenceRef),
			EvidenceTier: int32(candidate.EvidenceTier),
		})
	}
	return &pb.RunResult{
		FinalOutput: validUTF8(result.FinalOutput),
		Selection: &pb.FinalOutputSelection{
			Status:              FinalOutputSelectionStatusToProto(result.Selection.Status),
			SelectedCandidateId: validUTF8(result.Selection.SelectedCandidateID),
			Rule:                validUTF8(result.Selection.Rule),
			AlgorithmVersion:    validUTF8(result.Selection.AlgorithmVersion),
			Evidence:            validUTF8Slice(result.Selection.Evidence),
		},
		Candidates:     candidates,
		Success:        result.Success,
		ExitCode:       int32(result.ExitCode),
		TerminalReason: validUTF8(result.TerminalReason),
		Structured:     StructuredResultToProto(result.Structured),
	}
}

func StructuredResultToProto(result *domain.StructuredResult) *pb.StructuredResult {
	if result == nil {
		return nil
	}
	diagnostics := make([]*pb.StructuredDiagnostic, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		diagnostics = append(diagnostics, &pb.StructuredDiagnostic{Code: validUTF8(diagnostic.Code), Path: validUTF8(diagnostic.Path), Message: validUTF8(diagnostic.Message)})
	}
	var extractor *pb.StructuredExtractionProvenance
	if result.Extractor != nil {
		extractor = &pb.StructuredExtractionProvenance{
			RoleRef: validUTF8(result.Extractor.RoleRef), Provider: validUTF8(result.Extractor.Provider), Model: validUTF8(result.Extractor.Model),
			PolicySnapshot: ExecutionPolicySnapshotToProto(result.Extractor.PolicySnapshot),
		}
	}
	return &pb.StructuredResult{
		Status: StructuredResultStatusToProto(result.Status), SpecKind: ResultSpecKindToProto(result.SpecKind),
		SchemaDigest: validUTF8(result.SchemaDigest), Value: append([]byte(nil), result.Value...), Method: validUTF8(result.Method),
		SourceCandidateId: validUTF8(result.SourceCandidateID), Extractor: extractor, Diagnostics: diagnostics,
	}
}

func FinalOutputSelectionStatusToProto(status domain.FinalOutputSelectionStatus) pb.FinalOutputSelectionStatus {
	switch status {
	case domain.FinalOutputSelectionSelected:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED
	case domain.FinalOutputSelectionAmbiguous:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_AMBIGUOUS
	case domain.FinalOutputSelectionUnavailable:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNAVAILABLE
	default:
		return pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNSPECIFIED
	}
}

func RunResultFromProto(result *pb.RunResult) *domain.RunResult {
	if result == nil {
		return nil
	}
	candidates := make([]domain.FinalOutputCandidate, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		if candidate == nil {
			continue
		}
		candidates = append(candidates, domain.FinalOutputCandidate{
			ID: candidate.Id, EventID: candidate.EventId, Sequence: candidate.Sequence,
			Content: candidate.Content, MessageID: candidate.MessageId,
			ConversationID: candidate.ConversationId, TurnID: candidate.TurnId,
			ProviderOrigin: candidate.ProviderOrigin, CompletionReason: candidate.CompletionReason,
			Terminal: candidate.Terminal, ParentMessageID: candidate.ParentMessageId,
			ProviderEventType: candidate.ProviderEventType, RawEvidenceRef: candidate.RawEvidenceRef,
			EvidenceTier: int(candidate.EvidenceTier),
		})
	}
	selection := domain.FinalOutputSelection{}
	if result.Selection != nil {
		selection = domain.FinalOutputSelection{
			Status:              FinalOutputSelectionStatusFromProto(result.Selection.Status),
			SelectedCandidateID: result.Selection.SelectedCandidateId,
			Rule:                result.Selection.Rule,
			AlgorithmVersion:    result.Selection.AlgorithmVersion,
			Evidence:            result.Selection.Evidence,
		}
	}
	return &domain.RunResult{
		FinalOutput: result.FinalOutput, Selection: selection, Candidates: candidates,
		Success: result.Success, ExitCode: int(result.ExitCode), TerminalReason: result.TerminalReason,
		Structured: StructuredResultFromProto(result.Structured),
	}
}

func StructuredResultFromProto(result *pb.StructuredResult) *domain.StructuredResult {
	if result == nil {
		return nil
	}
	diagnostics := make([]domain.StructuredDiagnostic, 0, len(result.Diagnostics))
	for _, diagnostic := range result.Diagnostics {
		if diagnostic != nil {
			diagnostics = append(diagnostics, domain.StructuredDiagnostic{Code: diagnostic.Code, Path: diagnostic.Path, Message: diagnostic.Message})
		}
	}
	var extractor *domain.StructuredExtractionProvenance
	if result.Extractor != nil {
		extractor = &domain.StructuredExtractionProvenance{
			RoleRef: result.Extractor.RoleRef, Provider: result.Extractor.Provider, Model: result.Extractor.Model,
			PolicySnapshot: ExecutionPolicySnapshotFromProto(result.Extractor.PolicySnapshot),
		}
	}
	return &domain.StructuredResult{
		Status: StructuredResultStatusFromProto(result.Status), SpecKind: ResultSpecKindFromProto(result.SpecKind),
		SchemaDigest: result.SchemaDigest, Value: append([]byte(nil), result.Value...), Method: result.Method,
		SourceCandidateID: result.SourceCandidateId, Extractor: extractor, Diagnostics: diagnostics,
	}
}

func StructuredResultStatusToProto(status domain.StructuredResultStatus) pb.StructuredResultStatus {
	switch status {
	case domain.StructuredResultSuccess:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_SUCCESS
	case domain.StructuredResultUnavailable:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_UNAVAILABLE
	case domain.StructuredResultInvalid:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_INVALID
	case domain.StructuredResultAmbiguous:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_AMBIGUOUS
	case domain.StructuredResultAbstained:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_ABSTAINED
	default:
		return pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_UNSPECIFIED
	}
}

func StructuredResultStatusFromProto(status pb.StructuredResultStatus) domain.StructuredResultStatus {
	switch status {
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_SUCCESS:
		return domain.StructuredResultSuccess
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_UNAVAILABLE:
		return domain.StructuredResultUnavailable
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_INVALID:
		return domain.StructuredResultInvalid
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_AMBIGUOUS:
		return domain.StructuredResultAmbiguous
	case pb.StructuredResultStatus_STRUCTURED_RESULT_STATUS_ABSTAINED:
		return domain.StructuredResultAbstained
	default:
		return ""
	}
}

func FinalOutputSelectionStatusFromProto(status pb.FinalOutputSelectionStatus) domain.FinalOutputSelectionStatus {
	switch status {
	case pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED:
		return domain.FinalOutputSelectionSelected
	case pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_AMBIGUOUS:
		return domain.FinalOutputSelectionAmbiguous
	case pb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_UNAVAILABLE:
		return domain.FinalOutputSelectionUnavailable
	default:
		return ""
	}
}

// RunEventsToProto converts a slice of domain RunEvent to proto.
func RunEventsToProto(events []*domain.RunEvent) []*pb.RunEvent {
	result := make([]*pb.RunEvent, len(events))
	for i, e := range events {
		result[i] = RunEventToProto(e)
	}
	return result
}

// =============================================================================
// RUNNER STATUS
// =============================================================================
