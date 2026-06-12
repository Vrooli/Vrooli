package hygienecli

import (
	"io"
	"time"

	hygieneapp "github.com/vrooli/vrooli/internal/app/hygiene"
	planapp "github.com/vrooli/vrooli/internal/app/plans"
	shareddriftapp "github.com/vrooli/vrooli/internal/app/shareddrift"
	"github.com/vrooli/vrooli/internal/cli/contractcli"
	"github.com/vrooli/vrooli/internal/cliout"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// HygieneReportMessage maps the internal hygiene report onto the vrooli.cli.v1
// wire contract. A proto field rename breaks this mapping at compile time —
// that is the drift guard. The embedded contract-validation output reuses the
// contractcli mapping; the shared-drift report reuses the shared_drift contract.
func HygieneReportMessage(report hygieneapp.Report) *cliv1.HygieneReport {
	msg := &cliv1.HygieneReport{
		Success:          report.Success,
		Root:             report.Root,
		ConfigFixes:      report.ConfigFixes,
		Contract:         contractcli.ContractValidationOutputMessage(report.Contract),
		SharedDrift:      sharedDriftReportMessage(report.SharedDrift),
		BlockingFailures: int32(report.BlockingFailures),
		Warnings:         int32(report.Warnings),
	}
	for _, check := range report.Checks {
		msg.Checks = append(msg.Checks, &cliv1.HygieneCheck{
			Name:     check.Name,
			Passed:   check.Passed,
			Severity: string(check.Severity),
			Message:  check.Message,
		})
	}
	for _, finding := range report.Findings {
		msg.Findings = append(msg.Findings, hygieneFindingMessage(finding))
	}
	for _, action := range report.Actions {
		msg.Actions = append(msg.Actions, hygieneActionMessage(action))
	}
	for _, candidate := range report.PlanCandidates {
		msg.PlanCandidates = append(msg.PlanCandidates, &cliv1.HygienePlanCandidate{
			Path:   candidate.Path,
			Status: candidate.Status,
			Reason: candidate.Reason,
		})
	}
	for _, fix := range report.FixesApplied {
		msg.FixesApplied = append(msg.FixesApplied, &cliv1.HygienePlanFix{
			Source: fix.Source,
			Plan:   hygienePlanRecordMessage(fix.Plan),
		})
	}
	return msg
}

func hygieneFindingMessage(finding hygieneapp.Finding) *cliv1.HygieneFinding {
	msg := &cliv1.HygieneFinding{
		Severity:   string(finding.Severity),
		Code:       finding.Code,
		Path:       finding.Path,
		Locations:  finding.Locations,
		Message:    finding.Message,
		Why:        finding.Why,
		Fixability: string(finding.Fixability),
	}
	for _, action := range finding.NextActions {
		msg.NextActions = append(msg.NextActions, hygieneActionMessage(action))
	}
	return msg
}

func hygieneActionMessage(action hygieneapp.Action) *cliv1.HygieneAction {
	return &cliv1.HygieneAction{
		Code:       action.Code,
		Message:    action.Message,
		Command:    action.Command,
		Fixability: string(action.Fixability),
	}
}

func hygienePlanRecordMessage(record planapp.PlanRecord) *cliv1.HygienePlanRecord {
	return &cliv1.HygienePlanRecord{
		Id:          record.ID,
		Title:       record.Title,
		Slug:        record.Slug,
		Path:        record.Path,
		CreatedAt:   formatTime(record.CreatedAt),
		UpdatedAt:   formatTime(record.UpdatedAt),
		Archived:    record.Archived,
		ArchivedAt:  formatTime(record.ArchivedAt),
		SourcePath:  record.SourcePath,
		ContentHash: record.ContentHash,
	}
}

// sharedDriftReportMessage maps the optional embedded shared-drift report. A nil
// report (drift check skipped) maps to a nil message (absent in JSON).
func sharedDriftReportMessage(report *shareddriftapp.Report) *cliv1.SharedDriftReport {
	if report == nil {
		return nil
	}
	msg := &cliv1.SharedDriftReport{
		Clean:                report.Clean,
		Root:                 report.Root,
		TouchedPackages:      report.TouchedPackages,
		OnlyTouched:          report.OnlyTouchedUsed,
		BuildChecked:         report.BuildChecked,
		FixApplied:           report.FixApplied,
		ModifiedTrackedFiles: report.ModifiedTrackedOK,
		ElapsedMs:            int32(report.ElapsedMs),
	}
	for _, sc := range report.Scenarios {
		msg.Scenarios = append(msg.Scenarios, &cliv1.SharedDriftScenario{
			Path:       sc.Path,
			ApiDir:     sc.APIDir,
			Status:     string(sc.Status),
			DiffPaths:  sc.DiffPaths,
			BuildError: sc.BuildError,
			Error:      sc.Error,
			Replaces:   sc.Replaces,
		})
	}
	return msg
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

func writeHygieneReportJSON(w io.Writer, report hygieneapp.Report) error {
	return cliout.WriteProtoJSON(w, HygieneReportMessage(report))
}
