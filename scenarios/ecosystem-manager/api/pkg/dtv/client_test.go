package dtv

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"
	reportconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report/report_v1connect"
)

// stubReportHandler implements the generated ReportServiceHandler with canned
// GetSkillFitness output, standing in for a real DTV instance over the wire.
type stubReportHandler struct {
	fitness *reportv1.SkillFitness
	err     error
}

func (s *stubReportHandler) GetGoldenSummary(context.Context, *connect.Request[reportv1.GetGoldenSummaryRequest]) (*connect.Response[reportv1.GetGoldenSummaryResponse], error) {
	return connect.NewResponse(&reportv1.GetGoldenSummaryResponse{}), nil
}

func (s *stubReportHandler) GetTupleHistory(context.Context, *connect.Request[reportv1.GetTupleHistoryRequest]) (*connect.Response[reportv1.GetTupleHistoryResponse], error) {
	return connect.NewResponse(&reportv1.GetTupleHistoryResponse{}), nil
}

func (s *stubReportHandler) GetCoverage(context.Context, *connect.Request[reportv1.GetCoverageRequest]) (*connect.Response[reportv1.GetCoverageResponse], error) {
	return connect.NewResponse(&reportv1.GetCoverageResponse{}), nil
}

func (s *stubReportHandler) GetSkillFitness(_ context.Context, _ *connect.Request[reportv1.GetSkillFitnessRequest]) (*connect.Response[reportv1.GetSkillFitnessResponse], error) {
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&reportv1.GetSkillFitnessResponse{Fitness: s.fitness}), nil
}

// newStubClient mounts the stub DTV handler on an httptest server and returns a
// dtv.Client whose resolver points at it (bypassing the discovery registry).
func newStubClient(t *testing.T, h *stubReportHandler) *Client {
	t.Helper()
	path, handler := reportconnect.NewReportServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Client{
		httpClient: srv.Client(),
		resolve:    func(context.Context) (string, error) { return srv.URL, nil },
	}
}

func TestClientFitness_MapsProto(t *testing.T) {
	c := newStubClient(t, &stubReportHandler{fitness: &reportv1.SkillFitness{
		SkillId:          "ux",
		Verdict:          reportv1.SkillFitnessVerdict_SKILL_FITNESS_VERDICT_YELLOW,
		PassRate:         0.75,
		TotalRuns:        8,
		AvgTokens:        4200,
		UniqueDiffHashes: 2,
		AnyStale:         true,
	}})
	got, err := c.Fitness(context.Background(), "ux")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Verdict != VerdictYellow {
		t.Errorf("verdict = %v, want yellow", got.Verdict)
	}
	if got.PassRate != 0.75 || got.TotalRuns != 8 || got.AvgTokens != 4200 {
		t.Errorf("scalar fields mismatched: %+v", got)
	}
	if got.UniqueDiffHashes != 2 || !got.AnyStale {
		t.Errorf("convergence/stale fields mismatched: %+v", got)
	}
	if !got.Known() {
		t.Errorf("expected Known() true for a resolved verdict")
	}
}

func TestClientFitness_FailsOpenOnRPCError(t *testing.T) { // [REQ:EM-P2-002]
	c := newStubClient(t, &stubReportHandler{err: connect.NewError(connect.CodeInternal, errors.New("boom"))})
	got, err := c.Fitness(context.Background(), "ux")
	if err == nil {
		t.Fatal("expected an error to be surfaced")
	}
	if got.Verdict != VerdictUnknown {
		t.Errorf("on error, verdict must be UNKNOWN (fail open); got %v", got.Verdict)
	}
	if got.SkillID != "ux" {
		t.Errorf("skill id should be preserved on error; got %q", got.SkillID)
	}
}

func TestClientFitness_FailsOpenOnResolveError(t *testing.T) {
	c := &Client{
		httpClient: http.DefaultClient,
		resolve:    func(context.Context) (string, error) { return "", errors.New("no registry") },
	}
	got, err := c.Fitness(context.Background(), "ux")
	if err == nil {
		t.Fatal("expected resolve error")
	}
	if got.Known() {
		t.Errorf("resolve failure must yield UNKNOWN fitness; got %v", got.Verdict)
	}
}

func TestFakeProvider(t *testing.T) {
	f := &FakeProvider{Fits: map[string]Fitness{
		"test": {Verdict: VerdictGreen, PassRate: 1.0, TotalRuns: 5},
	}}

	got, err := f.Fitness(context.Background(), "test")
	if err != nil || got.Verdict != VerdictGreen || got.SkillID != "test" {
		t.Fatalf("known skill: got %+v err %v", got, err)
	}

	missing, err := f.Fitness(context.Background(), "absent")
	if err != nil || missing.Known() {
		t.Errorf("absent skill must be UNKNOWN with no error; got %+v err %v", missing, err)
	}

	failing := &FakeProvider{Err: errors.New("dtv down")}
	deg, err := failing.Fitness(context.Background(), "test")
	if err == nil || deg.Known() {
		t.Errorf("Err must surface alongside UNKNOWN fitness; got %+v err %v", deg, err)
	}
}
