package main

import (
	"time"

	ampb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func agentRunSummaryToAPI(summary *ampb.RunSummary) *AgentRunSummary {
	if summary == nil {
		return nil
	}
	return &AgentRunSummary{
		FilesModified: summary.GetFilesModified(), FilesCreated: summary.GetFilesCreated(),
		FilesDeleted: summary.GetFilesDeleted(), TokensUsed: summary.GetTokensUsed(),
		TurnsUsed: int(summary.GetTurnsUsed()), CostEstimate: summary.GetCostEstimate(),
	}
}

func agentRunActionsToAPI(actions *ampb.RunActions) *AgentRunActions {
	if actions == nil {
		return nil
	}
	return &AgentRunActions{
		CanInvestigate: actions.GetCanInvestigate(), CanApplyInvestigation: actions.GetCanApplyInvestigation(),
		CanDelete: actions.GetCanDelete(), CanStop: actions.GetCanStop(), CanRetry: actions.GetCanRetry(),
		CanContinue: actions.GetCanContinue(), CanApprove: actions.GetCanApprove(), CanReject: actions.GetCanReject(),
		CanReview: actions.GetCanReview(),
	}
}

func agentRunToAPI(run *ampb.Run) AgentRun {
	if run == nil {
		return AgentRun{}
	}
	return AgentRun{
		ID: run.GetId(), TaskID: run.GetTaskId(), SessionID: run.GetSessionId(),
		Status:          normalizeEnum(run.GetStatus().String(), "RUN_STATUS_"),
		Phase:           normalizeEnum(run.GetPhase().String(), "RUN_PHASE_"),
		ProgressPercent: int(run.GetProgressPercent()), ErrorMsg: run.GetErrorMsg(),
		ApprovalState: normalizeEnum(run.GetApprovalState().String(), "APPROVAL_STATE_"),
		PromptPreview: run.GetPromptPreview(), SandboxID: run.GetSandboxId(), CommitHash: run.GetCommitHash(),
		Summary: agentRunSummaryToAPI(run.GetSummary()), Actions: agentRunActionsToAPI(run.GetActions()),
		CreatedAt: timestampString(run.GetCreatedAt()), StartedAt: timestampString(run.GetStartedAt()),
		EndedAt: timestampString(run.GetEndedAt()),
	}
}

func timestampString(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format(time.RFC3339Nano)
}

func agentRunEventToAPI(event *ampb.RunEvent) AgentRunEvent {
	if event == nil {
		return AgentRunEvent{}
	}
	result := AgentRunEvent{
		ID: event.GetId(), RunID: event.GetRunId(), Sequence: event.GetSequence(),
		EventType: normalizeEventType(event.GetEventType().String()), Timestamp: timestampString(event.GetTimestamp()),
	}
	data := map[string]interface{}{}
	switch value := event.GetData().(type) {
	case *ampb.RunEvent_Message:
		data["content"], data["role"] = value.Message.GetContent(), value.Message.GetRole()
	case *ampb.RunEvent_ToolCall:
		data["name"] = value.ToolCall.GetToolName()
		if value.ToolCall.GetInput() != nil {
			data["input"] = value.ToolCall.GetInput().AsMap()
		}
	case *ampb.RunEvent_ToolResult:
		data["result"] = value.ToolResult.GetOutput()
		if name := value.ToolResult.GetToolName(); name != "" {
			data["name"] = name
		}
	case *ampb.RunEvent_Error:
		data["message"] = value.Error.GetMessage()
		if code := value.Error.GetCode(); code != "" {
			data["code"] = code
		}
	case *ampb.RunEvent_Status:
		if value.Status.GetNewStatus() != "" {
			data["newStatus"] = normalizeEnum(value.Status.GetNewStatus(), "RUN_STATUS_")
		}
		if value.Status.GetOldStatus() != "" {
			data["oldStatus"] = normalizeEnum(value.Status.GetOldStatus(), "RUN_STATUS_")
		}
	case *ampb.RunEvent_Log:
		data["message"], data["level"] = value.Log.GetMessage(), value.Log.GetLevel()
	case *ampb.RunEvent_Progress:
		data["percent"] = int(value.Progress.GetPercentComplete())
		if message := value.Progress.GetCurrentAction(); message != "" {
			data["message"] = message
		}
	}
	if len(data) != 0 {
		result.Data = data
	}
	return result
}

func agentFileDiffToAPI(file *ampb.FileDiff) AgentRunDiffFile {
	if file == nil {
		return AgentRunDiffFile{}
	}
	return AgentRunDiffFile{Path: file.GetPath(), ChangeType: file.GetChangeType(), Additions: int(file.GetAdditions()), Deletions: int(file.GetDeletions()), IsBinary: file.GetIsBinary(), Patch: file.GetPatch()}
}

func agentProfileToAPI(profile *ampb.AgentProfile) AgentProfile {
	if profile == nil {
		return AgentProfile{}
	}
	return AgentProfile{ID: profile.GetId(), Key: profile.GetProfileKey(), Name: profile.GetName(), Description: profile.GetDescription(), RoleRef: profile.GetRoleRef()}
}
