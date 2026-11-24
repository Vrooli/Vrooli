package audits

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/scheduletest"
	"github.com/vrooli/cli-core/cliapptest"
	auditsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits"
	auditsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits/audits_v1connect"
)

type pollingAuditService struct {
	statuses []auditsv1.AuditStatus
	requests []*auditsv1.GetAuditRequest
}

func (s *pollingAuditService) RunSnapshotAudit(context.Context, *connect.Request[auditsv1.RunSnapshotAuditRequest]) (*connect.Response[auditsv1.RunSnapshotAuditResponse], error) {
	return connect.NewResponse(&auditsv1.RunSnapshotAuditResponse{}), nil
}

func (s *pollingAuditService) GetAudit(_ context.Context, req *connect.Request[auditsv1.GetAuditRequest]) (*connect.Response[auditsv1.GetAuditResponse], error) {
	s.requests = append(s.requests, req.Msg)
	index := len(s.requests) - 1
	if index >= len(s.statuses) {
		index = len(s.statuses) - 1
	}
	return connect.NewResponse(&auditsv1.GetAuditResponse{Audit: &auditsv1.Audit{Id: req.Msg.Id, Status: s.statuses[index]}}), nil
}

func (s *pollingAuditService) ListAudits(context.Context, *connect.Request[auditsv1.ListAuditsRequest]) (*connect.Response[auditsv1.ListAuditsResponse], error) {
	return connect.NewResponse(&auditsv1.ListAuditsResponse{}), nil
}

func TestPollToTerminalWaitsForCompletedAudit(t *testing.T) {
	service := &pollingAuditService{statuses: []auditsv1.AuditStatus{
		auditsv1.AuditStatus_AUDIT_STATUS_RUNNING,
		auditsv1.AuditStatus_AUDIT_STATUS_COMPLETED,
	}}
	mux := http.NewServeMux()
	path, handler := auditsconnect.NewAuditsServiceHandler(service)
	mux.Handle(path, handler)
	app := cliapptest.NewTestApp(t, mux)
	h := newHandlers(app)
	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	h.clock = scheduletest.New(start)

	got, err := h.pollToTerminal("audit-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != "audit-1" || got.Status != auditsv1.AuditStatus_AUDIT_STATUS_COMPLETED {
		t.Fatalf("pollToTerminal() = %+v", got)
	}
	if len(service.requests) != 2 || service.requests[0].Id != "audit-1" || service.requests[1].Id != "audit-1" {
		t.Fatalf("GetAudit requests = %+v", service.requests)
	}
	if want := start.Add(auditPollInterval); !h.clock.Now().Equal(want) {
		t.Fatalf("clock = %s, want one %s wait to %s", h.clock.Now(), auditPollInterval, want)
	}
	if !isTerminalAudit(auditsv1.AuditStatus_AUDIT_STATUS_FAILED) || isTerminalAudit(auditsv1.AuditStatus_AUDIT_STATUS_RUNNING) {
		t.Fatal("audit failed/running terminal classification changed")
	}
}

var _ auditsconnect.AuditsServiceHandler = (*pollingAuditService)(nil)
