package convert

import (
	scriptspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ScriptMetaToProto(meta services.ScriptMeta) *scriptspb.InvestigationScript {
	return &scriptspb.InvestigationScript{
		Id:            meta.ID,
		Name:          meta.Name,
		Description:   meta.Description,
		Category:      meta.Category,
		Author:        meta.Author,
		CreatedAt:     timestamppb.New(meta.CreatedAt),
		UpdatedAt:     timestamppb.New(meta.UpdatedAt),
		Enabled:       meta.Enabled,
		ExecutionMode: meta.ExecutionMode,
		RequiredTools: meta.RequiredTools,
		SkipReason:    meta.SkipReason,
	}
}

func ScriptMetasToProto(metas []services.ScriptMeta) []*scriptspb.InvestigationScript {
	result := make([]*scriptspb.InvestigationScript, len(metas))
	for i, m := range metas {
		result[i] = ScriptMetaToProto(m)
	}
	return result
}

func ScriptExecutionToProto(exec services.ScriptExecution) *scriptspb.ScriptExecution {
	pb := &scriptspb.ScriptExecution{
		ScriptId:      exec.ScriptID,
		ExecutionId:   exec.ExecutionID,
		Status:        scriptExecutionStatusToProto(exec.Status),
		StartedAt:     timestamppb.New(exec.StartedAt),
		CompletedAt:   timestamppb.New(exec.CompletedAt),
		Output:        exec.Stdout,
		Stdout:        exec.Stdout,
		Stderr:        exec.Stderr,
		TimedOut:      exec.TimedOut,
		ExecutionMode: exec.ExecutionMode,
		SkipReason:    exec.SkipReason,
	}
	exitCode := int32(exec.ExitCode)
	pb.ExitCode = &exitCode
	dur := exec.DurationSeconds
	pb.DurationSeconds = &dur
	if exec.Status == "failed" {
		pb.Error = exec.Stderr
	}
	return pb
}
