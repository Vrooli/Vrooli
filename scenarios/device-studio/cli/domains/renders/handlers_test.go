package renders

import (
	"strings"
	"testing"

	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestSubmitReportUsesActualReusedJobStatus(t *testing.T) { // [REQ:DS-INTEGRATION-OPERATIONS]
	r := (&handlers{}).submitReport(nil, &shotsv1.SubmitRenderResponse{Job: &shotsv1.RenderJob{
		Id: "render-1", ProjectId: "project-1", Version: 4,
		Status: shotsv1.RenderJobStatus_RENDER_JOB_STATUS_SUCCEEDED,
	}})
	if len(r.Result) != 1 || !strings.Contains(r.Result[0], "SUCCEEDED") {
		t.Fatalf("submit report did not reflect returned job state: %#v", r.Result)
	}
}
