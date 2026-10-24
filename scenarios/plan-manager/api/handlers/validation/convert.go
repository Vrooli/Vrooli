package validation

import (
	planmodel "plan-manager/internal/planmodel"
	"plan-manager/internal/planproto"
	internalvalidation "plan-manager/internal/validation"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"
)

// This file is the only translation point between the proto wire types
// (vrooli.plan_manager.v1.shared) and the validation/plans domain vocabulary.
// The domain layer never imports proto (api-steer §7).

func resultToProto(r internalvalidation.Result) *sharedv1.ValidationResult {
	out := &sharedv1.ValidationResult{
		Id:              r.ID,
		PlanId:          r.PlanID,
		PhaseId:         r.PhaseID,
		Verdict:         verdictToProto(r.Verdict),
		Staleness:       stalenessToProto(r.Staleness),
		CommandsRun:     r.CommandsRun,
		Detail:          r.Detail,
		RanAt:           r.RanAt,
		ExecutionId:     r.ExecutionID,
		OperationId:     r.OperationID,
		ScopeGeneration: int32(r.ScopeGeneration),
		FullInventory:   r.FullInventory,
		RequiredMembers: append([]string(nil), r.RequiredMembers...),
		SelectedMembers: append([]string(nil), r.SelectedMembers...),
	}
	for _, finding := range r.CommandFindings {
		out.CommandFindings = append(out.CommandFindings, &sharedv1.CommandValidationFinding{
			CommandText:     finding.CommandText,
			Verdict:         finding.Verdict,
			ValidationLevel: finding.Level,
			Message:         finding.Message,
			Location:        finding.Location,
			IssueCodes:      append([]string(nil), finding.IssueCodes...),
			Suggestions:     append([]string(nil), finding.Suggestions...),
			Guidance:        append([]string(nil), finding.Guidance...),
		})
	}
	return out
}

func operationToProto(op internalvalidation.ValidationOperation) *validationv1.ValidationOperation {
	out := &validationv1.ValidationOperation{
		SchemaVersion: int32(op.SchemaVersion),
		Id:            op.ID, PlanId: op.PlanID, PhaseId: op.PhaseID,
		IdempotencyKey: op.IdempotencyKey, Status: operationStatusToProto(op.Status),
		Attempt: int32(op.Attempt), ResultRef: op.ResultRef,
		QueuedAt: op.QueuedAt, StartedAt: op.StartedAt, TerminalAt: op.TerminalAt,
		QueueBudgetSeconds:         int32(op.QueueBudgetSeconds),
		ExecutionBudgetSeconds:     int32(op.ExecutionBudgetSeconds),
		TransportWaitBudgetSeconds: int32(op.TransportWaitBudgetSeconds),
		RecommendedWaitSeconds:     int32(op.RecommendedWaitSeconds),
		ScopeFingerprint:           op.ScopeFingerprint,
		QueueReason:                op.QueueReason,
		ProducerWaitArgv:           append([]string(nil), op.ProducerWaitArgv...),
		SyncArgv:                   append([]string(nil), op.SyncArgv...),
		LastSyncedAt:               op.LastSyncedAt,
		ExecutionId:                op.ExecutionID,
		ScopeGeneration:            int32(op.ScopeGeneration),
		RequiredMembers:            append([]string(nil), op.RequiredMembers...),
		SelectedMembers:            append([]string(nil), op.SelectedMembers...),
		FullInventory:              op.FullInventory,
	}
	for _, run := range op.TestRuns {
		out.TestRuns = append(out.TestRuns, &validationv1.TestRunEvidence{Scenario: run.Scenario, RunId: run.RunID, Status: run.Status, Fingerprint: run.Fingerprint, TerminalAt: run.TerminalAt, Detail: run.Detail})
	}
	if op.Terminal() && op.Result != nil {
		out.Result = resultToProto(*op.Result)
	}
	if op.Error != nil {
		out.Error = &validationv1.ValidationOperationError{Code: op.Error.Code, Detail: op.Error.Detail}
	}
	for _, child := range op.Children {
		item := &validationv1.ValidationChildOperation{
			Id: child.ID, Command: child.Command, Oracle: child.Oracle,
			Status: childStatusToProto(child.Status), Attempt: int32(child.Attempt),
			ExternalId: child.ExternalID, Verdict: verdictToProto(child.Verdict), Detail: child.Detail,
			QueuedAt: child.QueuedAt, StartedAt: child.StartedAt, TerminalAt: child.TerminalAt,
		}
		if child.Error != nil {
			item.Error = &validationv1.ValidationOperationError{Code: child.Error.Code, Detail: child.Error.Detail}
		}
		out.Children = append(out.Children, item)
	}
	return out
}

func operationStatusToProto(status internalvalidation.OperationStatus) validationv1.ValidationOperationStatus {
	switch status {
	case internalvalidation.OperationQueued:
		return validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_QUEUED
	case internalvalidation.OperationRunning:
		return validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_RUNNING
	case internalvalidation.OperationTerminal:
		return validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_TERMINAL
	default:
		return validationv1.ValidationOperationStatus_VALIDATION_OPERATION_STATUS_UNSPECIFIED
	}
}

func childStatusToProto(status internalvalidation.ChildStatus) validationv1.ValidationChildStatus {
	switch status {
	case internalvalidation.ChildQueued:
		return validationv1.ValidationChildStatus_VALIDATION_CHILD_STATUS_QUEUED
	case internalvalidation.ChildRunning:
		return validationv1.ValidationChildStatus_VALIDATION_CHILD_STATUS_RUNNING
	case internalvalidation.ChildTerminal:
		return validationv1.ValidationChildStatus_VALIDATION_CHILD_STATUS_TERMINAL
	default:
		return validationv1.ValidationChildStatus_VALIDATION_CHILD_STATUS_UNSPECIFIED
	}
}

func referencesToProto(refs []planmodel.Reference) []*sharedv1.Reference {
	return planproto.ReferencesToProto(refs)
}

func verdictToProto(v internalvalidation.Verdict) sharedv1.ValidationVerdict {
	switch v {
	case internalvalidation.VerdictPass:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS
	case internalvalidation.VerdictFail:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_FAIL
	case internalvalidation.VerdictUnknown:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNKNOWN
	default:
		return sharedv1.ValidationVerdict_VALIDATION_VERDICT_UNSPECIFIED
	}
}

func refKindToProto(k planmodel.ReferenceKind) sharedv1.ReferenceKind {
	return planproto.RefKindToProto(k)
}

func refResolutionToProto(r planmodel.ReferenceResolution) sharedv1.ReferenceResolution {
	return planproto.RefResolutionToProto(r)
}

func stalenessToProto(s planmodel.StalenessTier) sharedv1.StalenessTier {
	return planproto.StalenessToProto(s)
}
