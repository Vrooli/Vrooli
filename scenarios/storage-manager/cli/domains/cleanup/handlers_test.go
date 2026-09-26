package cleanup

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	cleanupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/storage-manager/v1/cleanup"
)

func TestCreatePlanReportSurfacesProviderWarnings(t *testing.T) {
	msg := &cleanupv1.CreatePlanResponse{Plan: &cleanupv1.Plan{
		Id: "plan-1",
		Providers: []*cleanupv1.ProviderPlan{{
			ProviderId:     "tmp",
			EstimatedBytes: 12,
			ItemCount:      1,
			ApprovalMode:   "operator",
			Warnings:       []string{"measurement budget exhausted: 3 entries were not measured"},
		}},
	}}

	rendered, err := cliapp.RenderOperationalReportString((&handlers{}).createPlanReport(nil, msg))
	if err != nil {
		t.Fatalf("render report: %v", err)
	}
	if !strings.Contains(rendered, "tmp 12 bytes 1 item(s)") {
		t.Fatalf("provider row missing from report: %q", rendered)
	}
	if !strings.Contains(rendered, "warning: measurement budget exhausted: 3 entries were not measured") {
		t.Fatalf("provider warning missing from human report: %q", rendered)
	}
}

func TestCreatePlanReportLeavesWarningFreeProviderRowUnchanged(t *testing.T) {
	msg := &cleanupv1.CreatePlanResponse{Plan: &cleanupv1.Plan{
		Id: "plan-2",
		Providers: []*cleanupv1.ProviderPlan{{
			ProviderId:     "tmp",
			EstimatedBytes: 12,
			ItemCount:      1,
			ApprovalMode:   "operator",
		}},
	}}

	rendered, err := cliapp.RenderOperationalReportString((&handlers{}).createPlanReport(nil, msg))
	if err != nil {
		t.Fatalf("render report: %v", err)
	}
	if !strings.Contains(rendered, "tmp 12 bytes 1 item(s)") {
		t.Fatalf("provider row missing from report: %q", rendered)
	}
	if strings.Contains(rendered, "warning:") {
		t.Fatalf("warning-free provider row changed: %q", rendered)
	}
}
