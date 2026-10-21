package media

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
)

func TestMediaReportDistinguishesUninspectedDimensionsAndShowsRecoveryVersion(t *testing.T) {
	m := &projectsv1.SourceMedia{Id: "source", ProjectId: "project", OriginalFilename: "capture.mp4", MediaType: "video/mp4", Readiness: projectsv1.MediaReadiness_MEDIA_READINESS_PENDING,
		InspectionJob: &projectsv1.InspectionJobStatus{Id: "inspect", State: "failed", Version: 17, CurrentAttemptId: "attempt"}, InspectionIssueCode: "decode_failed", InspectionIssueMessage: "Incomplete video"}
	got := format(m)
	for _, want := range []string{"dimensions unknown", "readiness=pending", "state=failed version=17 attempt=attempt", "issue=decode_failed: Incomplete video"} {
		if !strings.Contains(got, want) {
			t.Errorf("report %q missing %q", got, want)
		}
	}
	w, h := uint32(430), uint32(880)
	m.Width, m.Height = &w, &h
	if got := format(m); !strings.Contains(got, "430x880") || strings.Contains(got, "dimensions unknown") {
		t.Fatalf("known dimensions rendered incorrectly: %s", got)
	}
}

type inspectionClient struct {
	projectsconnect.ProjectsServiceClient
	retry      *projectsv1.RetrySourceMediaInspectionRequest
	cancel     *projectsv1.CancelSourceMediaInspectionRequest
	incomplete bool
}

func (c *inspectionClient) RetrySourceMediaInspection(_ context.Context, r *connect.Request[projectsv1.RetrySourceMediaInspectionRequest]) (*connect.Response[projectsv1.RetrySourceMediaInspectionResponse], error) {
	c.retry = r.Msg
	response := &projectsv1.RetrySourceMediaInspectionResponse{}
	if !c.incomplete {
		response.SourceMedia, response.Job = &projectsv1.SourceMedia{Id: r.Msg.MediaId, ProjectId: r.Msg.ProjectId}, &projectsv1.InspectionJobStatus{Id: "job", State: "succeeded", Version: 23}
	}
	return connect.NewResponse(response), nil
}

func (c *inspectionClient) CancelSourceMediaInspection(_ context.Context, r *connect.Request[projectsv1.CancelSourceMediaInspectionRequest]) (*connect.Response[projectsv1.CancelSourceMediaInspectionResponse], error) {
	c.cancel = r.Msg
	response := &projectsv1.CancelSourceMediaInspectionResponse{}
	if !c.incomplete {
		response.SourceMedia, response.Job = &projectsv1.SourceMedia{Id: r.Msg.MediaId, ProjectId: r.Msg.ProjectId}, &projectsv1.InspectionJobStatus{Id: "job", State: "succeeded", Version: 23}
	}
	return connect.NewResponse(response), nil
}

func inspectionContext(version string) cliapp.OperationContext {
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "expected-version"}}, Positionals: []cliapp.Positional{{Name: "project-id"}, {Name: "id"}}}, Flags: map[string]string{"expected-version": version}, Positionals: map[string]string{"project-id": "project", "id": "source"}})
}

func TestInspectionControlsRejectInvalidVersionBeforeRPC(t *testing.T) {
	for _, version := range []string{"", "0", "-1", "1.5", "9223372036854775808"} {
		t.Run(version, func(t *testing.T) {
			client := &inspectionClient{}
			h := &handlers{client: client}
			if _, err := h.retryInspectionCall(inspectionContext(version)); err == nil {
				t.Fatal("retry accepted invalid version")
			}
			if _, err := h.cancelInspectionCall(inspectionContext(version)); err == nil {
				t.Fatal("cancel accepted invalid version")
			}
			if client.retry != nil || client.cancel != nil {
				t.Fatal("invalid request reached RPC")
			}
		})
	}
}

func TestInspectionControlsForwardVersionAndReportActualDurableState(t *testing.T) {
	client := &inspectionClient{}
	h := &handlers{client: client}
	ctx := inspectionContext("9007199254740993")
	r, err := h.retryInspectionCall(ctx)
	if err != nil {
		t.Fatal(err)
	}
	c, err := h.cancelInspectionCall(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if client.retry.ProjectId != "project" || client.retry.MediaId != "source" || client.retry.ExpectedJobVersion != 9007199254740993 {
		t.Fatalf("incorrect retry: %v", client.retry)
	}
	if client.cancel.ProjectId != "project" || client.cancel.MediaId != "source" || client.cancel.ExpectedJobVersion != 9007199254740993 {
		t.Fatalf("incorrect cancel: %v", client.cancel)
	}
	for _, report := range []cliapp.MutationReport{h.retryInspectionReport(ctx, r), h.cancelInspectionReport(ctx, c), h.inspectReport(ctx, &projectsv1.EnsureSourceMediaInspectionResponse{SourceMedia: r.SourceMedia, Job: r.Job})} {
		if len(report.Result) != 1 || !strings.Contains(report.Result[0], "is succeeded at version 23") {
			t.Fatalf("report invents requested state instead of durable response: %+v", report)
		}
	}
}

func TestInspectionControlsRejectIncompleteStatus(t *testing.T) {
	h := &handlers{client: &inspectionClient{incomplete: true}}
	if _, err := h.retryInspectionCall(inspectionContext("1")); err == nil {
		t.Fatal("missing status accepted")
	}
	if _, err := h.cancelInspectionCall(inspectionContext("1")); err == nil {
		t.Fatal("missing status accepted")
	}
}
