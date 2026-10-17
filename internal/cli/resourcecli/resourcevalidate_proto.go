package resourcecli

import (
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/resources"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ResourceValidationResponse maps resources.ResourceValidationReport onto the
// vrooli.cli.v1 wire contract. The call site emits
// WriteFieldsWithSuccess(report.Passed, {"report": report}), so success mirrors
// report.Passed. A field rename in the proto breaks this mapping at compile
// time — that is the drift guard.
func ResourceValidationResponse(report resources.ResourceValidationReport) *cliv1.ResourceValidationResponse {
	out := &cliv1.ResourceValidationResponse{
		Success: report.Passed,
		Report: &cliv1.ResourceValidationReport{
			Count:  int32(report.Count),
			Passed: report.Passed,
		},
	}
	for _, item := range report.Items {
		mapped := &cliv1.ResourceValidationItem{
			Name:         item.Name,
			ManifestPath: item.ManifestPath,
			Driver:       item.Driver,
		}
		for _, issue := range item.Issues {
			mapped.Issues = append(mapped.Issues, &cliv1.ResourceValidationIssue{
				Severity: issue.Severity,
				Message:  issue.Message,
			})
		}
		out.Report.Items = append(out.Report.Items, mapped)
	}
	for _, issue := range report.Issues {
		out.Report.Issues = append(out.Report.Issues, &cliv1.ResourceValidationIssue{
			Severity: issue.Severity,
			Message:  issue.Message,
		})
	}
	return out
}

// WriteValidationReportJSON emits the resource-validate wire contract as JSON.
func WriteValidationReportJSON(w io.Writer, report resources.ResourceValidationReport) error {
	return cliout.WriteProtoJSON(w, ResourceValidationResponse(report))
}
