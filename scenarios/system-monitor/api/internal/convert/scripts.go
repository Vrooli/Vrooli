package convert

import (
	scriptspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/scripts"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/investigations"
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
		Platforms:     meta.Platforms,
		Source:        meta.Source,
	}
}

func InvestigationRunToProto(run investigations.Run) *scriptspb.InvestigationRun {
	findings := make([]*scriptspb.InvestigationFinding, 0, len(run.Findings))
	for _, finding := range run.Findings {
		findings = append(findings, &scriptspb.InvestigationFinding{Severity: finding.Severity, Code: finding.Code, Summary: finding.Summary, DetailJson: finding.DetailJSON})
	}
	return &scriptspb.InvestigationRun{Id: run.ID, EntryId: run.EntryID, ExecutionMode: run.ExecutionMode, Status: run.Status, SkipReason: run.SkipReason, ExitCode: int32(run.ExitCode), TimedOut: run.TimedOut, StartedAt: timestamppb.New(run.StartedAt.UTC()), CompletedAt: timestamppb.New(run.CompletedAt.UTC()), DurationSeconds: run.DurationSeconds, HostOs: run.HostOS, HostArch: run.HostArch, ResultJson: run.ResultJSON, StderrTail: run.StderrTail, AnomalyId: run.AnomalyID, Findings: findings}
}

func InvestigationRunsToProto(runs []investigations.Run) []*scriptspb.InvestigationRun {
	result := make([]*scriptspb.InvestigationRun, 0, len(runs))
	for _, run := range runs {
		result = append(result, InvestigationRunToProto(run))
	}
	return result
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
