package reindex

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	depdomain "security-health/internal/dependencies"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/reindex"
)

type stubReindexer struct {
	statusOK bool
	cancelOK bool
}

func (s *stubReindexer) Reindex(_ context.Context, scenario string, dryRun bool) (depdomain.ReindexResult, error) {
	return depdomain.ReindexResult{JobID: "job-1", PlannedUpserts: 3, PlannedDeletes: 1, DryRun: dryRun}, nil
}
func (s *stubReindexer) ReindexStatus(string) (string, int, int, string, bool) {
	return "running", 2, 5, "", s.statusOK
}
func (s *stubReindexer) ReindexCancel(string) (bool, bool) { return true, s.cancelOK }

func TestReindex_MapsPlan(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &stubReindexer{}})
	resp, err := h.Reindex(context.Background(), connect.NewRequest(&reindexv1.ReindexRequest{DryRun: true}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetJobId() != "job-1" || resp.Msg.GetPlannedUpserts() != 3 || !resp.Msg.GetDryRun() {
		t.Errorf("reindex mapping wrong: %+v", resp.Msg)
	}
}

func TestReindexStatus_UnknownJobIsNotFound(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &stubReindexer{statusOK: false}})
	_, err := h.ReindexStatus(context.Background(), connect.NewRequest(&reindexv1.ReindexStatusRequest{JobId: "x"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("want NotFound, got %v", connect.CodeOf(err))
	}
}

func TestReindexCancel_Maps(t *testing.T) {
	h := NewConnectHandler(Deps{Service: &stubReindexer{cancelOK: true}})
	resp, err := h.ReindexCancel(context.Background(), connect.NewRequest(&reindexv1.ReindexCancelRequest{JobId: "job-1"}))
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Msg.GetCancelled() {
		t.Error("expected cancelled=true")
	}
}
