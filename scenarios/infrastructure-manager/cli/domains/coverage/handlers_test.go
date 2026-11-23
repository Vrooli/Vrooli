package coverage

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
	coverageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage/coverage_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/shared"
)

func TestFormatProjection(t *testing.T) {
	tests := []struct {
		name string
		p    *coveragev1.ProjectionCoverage
		want []string
		no   []string
	}{
		{
			name: "complete projection keeps ratio and denominator evidence",
			p: &coveragev1.ProjectionCoverage{
				Projection:   coveragev1.Projection_PROJECTION_AVAILABILITY,
				Ratio:        &coveragev1.Ratio{Value: .75},
				NowCount:     3,
				TotalCells:   4,
				InReachCount: 1,
				MissingCount: 0,
				Confidence: &coveragev1.Confidence{
					Level:     coveragev1.ConfidenceLevel_CONFIDENCE_LEVEL_PARTIAL,
					Rationale: "one source is provisional",
				},
			},
			want: []string{"availability: 75% NOW (3/4)", "in_reach=1", "missing=0", "denominator confidence=partial (one source is provisional)"},
		},
		{
			name: "ratio confidence remains available when duplicate projection field is absent",
			p: &coveragev1.ProjectionCoverage{
				Projection: coveragev1.Projection_PROJECTION_RECOVERY,
				Ratio: &coveragev1.Ratio{
					Value: .5,
					Confidence: &coveragev1.Confidence{
						Level:     coveragev1.ConfidenceLevel_CONFIDENCE_LEVEL_AUTHORITATIVE,
						Rationale: "complete source registry",
					},
				},
				NowCount: 1, TotalCells: 2,
			},
			want: []string{"recovery: 50% NOW (1/2)", "denominator confidence=authoritative (complete source registry)"},
		},
		{
			name: "missing ratio reports its source reason",
			p:    &coveragev1.ProjectionCoverage{Projection: coveragev1.Projection_PROJECTION_RECOVERY, UnavailableReason: "reader missing"},
			want: []string{"recovery: unavailable: reader missing"},
			no:   []string{"%", "NOW"},
		},
		{
			name: "missing both confidence fields gates the ratio",
			p:    &coveragev1.ProjectionCoverage{Projection: coveragev1.Projection_PROJECTION_RECOVERY, Ratio: &coveragev1.Ratio{Value: .5}, NowCount: 1, TotalCells: 2},
			want: []string{"recovery: unavailable (denominator confidence unavailable)", "NOW=1/2"},
			no:   []string{"50%"},
		},
		{name: "absent projection is explicit", p: nil, want: []string{"projection unavailable"}, no: []string{"%", "NOW"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatProjection(tt.p)
			for _, s := range tt.want {
				if !strings.Contains(got, s) {
					t.Errorf("formatProjection() = %q, want substring %q", got, s)
				}
			}
			for _, s := range tt.no {
				if strings.Contains(got, s) {
					t.Errorf("formatProjection() = %q, must not contain %q", got, s)
				}
			}
		})
	}
}

func TestReportsRenderAbsentProjectionAsUnavailable(t *testing.T) {
	status := (&handlers{}).statusReport(nil, &coveragev1.GetCoverageResponse{Projections: []*coveragev1.ProjectionCoverage{nil}})
	if len(status.Results) != 1 || status.Results[0] != "projection unavailable" {
		t.Fatalf("status results = %#v", status.Results)
	}

	show := (&handlers{}).showReport(nil, &coveragev1.GetProjectionResponse{Projection: coveragev1.Projection_PROJECTION_AVAILABILITY})
	if len(show.Results) != 1 || show.Results[0] != "projection unavailable" {
		t.Fatalf("show results = %#v", show.Results)
	}
}

type statusClient struct {
	coverageconnect.CoverageServiceClient
	request  *coveragev1.GetCoverageRequest
	response *connect.Response[coveragev1.GetCoverageResponse]
	err      error
	calls    int
}

func (c *statusClient) GetCoverage(_ context.Context, req *connect.Request[coveragev1.GetCoverageRequest]) (*connect.Response[coveragev1.GetCoverageResponse], error) {
	c.calls++
	c.request = req.Msg
	return c.response, c.err
}

func statusContext(projection string) cliapp.OperationContext {
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "projection"}}},
		Flags:  map[string]string{"projection": projection},
	})
}

func TestStatusCall(t *testing.T) {
	t.Run("maps projection flag to exact request", func(t *testing.T) {
		client := &statusClient{response: connect.NewResponse(&coveragev1.GetCoverageResponse{})}
		_, err := (&handlers{client: client}).statusCall(statusContext("availability"))
		if err != nil {
			t.Fatal(err)
		}
		if client.calls != 1 || len(client.request.GetProjections()) != 1 || client.request.GetProjections()[0] != coveragev1.Projection_PROJECTION_AVAILABILITY {
			t.Fatalf("request = %#v, calls = %d", client.request, client.calls)
		}
	})
	t.Run("invalid projection makes no RPC", func(t *testing.T) {
		client := &statusClient{}
		_, err := (&handlers{client: client}).statusCall(statusContext("unknown"))
		if err == nil || client.calls != 0 {
			t.Fatalf("err = %v, calls = %d", err, client.calls)
		}
	})
	t.Run("connect error retains command context", func(t *testing.T) {
		client := &statusClient{err: errors.New("offline")}
		_, err := (&handlers{client: client}).statusCall(statusContext(""))
		if err == nil || !strings.Contains(err.Error(), "coverage status") {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("nil response returns clear error", func(t *testing.T) {
		client := &statusClient{}
		_, err := (&handlers{client: client}).statusCall(statusContext(""))
		if err == nil || !strings.Contains(err.Error(), "no coverage response") {
			t.Fatalf("error = %v", err)
		}
	})
}

type coverageOperationsClient struct {
	coverageconnect.CoverageServiceClient

	listCellsRequest  *coveragev1.ListCellsRequest
	listCellsResponse *connect.Response[coveragev1.ListCellsResponse]
	listCellsErr      error
	listCellsCalls    int

	openLoopRequest  *coveragev1.ListCellsRequest
	openLoopResponse *connect.Response[coveragev1.ListCellsResponse]
	openLoopErr      error
	openLoopCalls    int

	showRequest  *coveragev1.GetProjectionRequest
	showResponse *connect.Response[coveragev1.GetProjectionResponse]
	showErr      error
	showCalls    int

	validateResponse *connect.Response[coveragev1.ValidateSetpointResponse]
	validateErr      error
	validateCalls    int

	driftRequest  *coveragev1.GetDriftRequest
	driftResponse *connect.Response[coveragev1.GetDriftResponse]
	driftErr      error
	driftCalls    int
}

func (c *coverageOperationsClient) ListCells(_ context.Context, req *connect.Request[coveragev1.ListCellsRequest]) (*connect.Response[coveragev1.ListCellsResponse], error) {
	c.listCellsCalls++
	c.listCellsRequest = req.Msg
	return c.listCellsResponse, c.listCellsErr
}

func (c *coverageOperationsClient) ListOpenLoopCells(_ context.Context, req *connect.Request[coveragev1.ListCellsRequest]) (*connect.Response[coveragev1.ListCellsResponse], error) {
	c.openLoopCalls++
	c.openLoopRequest = req.Msg
	return c.openLoopResponse, c.openLoopErr
}

func (c *coverageOperationsClient) GetProjection(_ context.Context, req *connect.Request[coveragev1.GetProjectionRequest]) (*connect.Response[coveragev1.GetProjectionResponse], error) {
	c.showCalls++
	c.showRequest = req.Msg
	return c.showResponse, c.showErr
}

func (c *coverageOperationsClient) ValidateSetpoint(_ context.Context, _ *connect.Request[coveragev1.ValidateSetpointRequest]) (*connect.Response[coveragev1.ValidateSetpointResponse], error) {
	c.validateCalls++
	return c.validateResponse, c.validateErr
}

func (c *coverageOperationsClient) GetDrift(_ context.Context, req *connect.Request[coveragev1.GetDriftRequest]) (*connect.Response[coveragev1.GetDriftResponse], error) {
	c.driftCalls++
	c.driftRequest = req.Msg
	return c.driftResponse, c.driftErr
}

func coverageContext(flags map[string]string, positionals map[string]string) cliapp.OperationContext {
	flagNames := make([]cliapp.Flag, 0, len(flags))
	for name := range flags {
		flagNames = append(flagNames, cliapp.Flag{Name: name})
	}
	positionalNames := make([]cliapp.Positional, 0, len(positionals))
	for name := range positionals {
		positionalNames = append(positionalNames, cliapp.Positional{Name: name})
	}
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema:      cliapp.ArgSchema{Flags: flagNames, Positionals: positionalNames},
		Flags:       flags,
		Positionals: positionals,
	})
}

func TestCellsCallMapsFiltersAndRejectsInvalidValuesBeforeRPC(t *testing.T) {
	client := &coverageOperationsClient{listCellsResponse: connect.NewResponse(&coveragev1.ListCellsResponse{})}
	ctx := coverageContext(map[string]string{"projection": "recovery", "status": "missing"}, nil)
	if _, err := (&handlers{client: client}).cellsCall(ctx); err != nil {
		t.Fatal(err)
	}
	if client.listCellsCalls != 1 || client.listCellsRequest.GetProjection() != coveragev1.Projection_PROJECTION_RECOVERY || client.listCellsRequest.GetStatus() != coveragev1.CellStatus_CELL_STATUS_MISSING {
		t.Fatalf("ListCells request = %#v, calls = %d", client.listCellsRequest, client.listCellsCalls)
	}

	invalid := &coverageOperationsClient{}
	_, err := (&handlers{client: invalid}).cellsCall(coverageContext(map[string]string{"projection": "not-a-projection", "status": ""}, nil))
	if err == nil || invalid.listCellsCalls != 0 {
		t.Fatalf("invalid projection err = %v, calls = %d", err, invalid.listCellsCalls)
	}
	_, err = (&handlers{client: invalid}).cellsCall(coverageContext(map[string]string{"projection": "", "status": "not-a-status"}, nil))
	if err == nil || invalid.listCellsCalls != 0 {
		t.Fatalf("invalid status err = %v, calls = %d", err, invalid.listCellsCalls)
	}
}

func TestOpenLoopCallPreservesOptionalProjectionAndErrors(t *testing.T) {
	client := &coverageOperationsClient{openLoopResponse: connect.NewResponse(&coveragev1.ListCellsResponse{})}
	ctx := coverageContext(map[string]string{"projection": "validation-cost"}, nil)
	if _, err := (&handlers{client: client}).openLoopCall(ctx); err != nil {
		t.Fatal(err)
	}
	if client.openLoopCalls != 1 || client.openLoopRequest.GetProjection() != coveragev1.Projection_PROJECTION_VALIDATION_COST {
		t.Fatalf("ListOpenLoopCells request = %#v, calls = %d", client.openLoopRequest, client.openLoopCalls)
	}

	noFilter := &coverageOperationsClient{openLoopResponse: connect.NewResponse(&coveragev1.ListCellsResponse{})}
	if _, err := (&handlers{client: noFilter}).openLoopCall(coverageContext(map[string]string{"projection": ""}, nil)); err != nil {
		t.Fatal(err)
	}
	if noFilter.openLoopRequest.GetProjection() != coveragev1.Projection_PROJECTION_UNSPECIFIED {
		t.Fatalf("omitted filter request = %#v", noFilter.openLoopRequest)
	}

	invalid := &coverageOperationsClient{}
	if _, err := (&handlers{client: invalid}).openLoopCall(coverageContext(map[string]string{"projection": "unknown"}, nil)); err == nil || invalid.openLoopCalls != 0 {
		t.Fatalf("invalid projection err = %v, calls = %d", err, invalid.openLoopCalls)
	}
	failed := &coverageOperationsClient{openLoopErr: errors.New("offline")}
	if _, err := (&handlers{client: failed}).openLoopCall(coverageContext(map[string]string{"projection": ""}, nil)); err == nil || !strings.Contains(err.Error(), "coverage open-loop") {
		t.Fatalf("Connect error = %v", err)
	}
	noResponse := &coverageOperationsClient{}
	if _, err := (&handlers{client: noResponse}).openLoopCall(coverageContext(map[string]string{"projection": ""}, nil)); err == nil || !strings.Contains(err.Error(), "no open-loop response") {
		t.Fatalf("nil response error = %v", err)
	}
}

func TestShowCallMapsProjectionAndHandlesFailures(t *testing.T) {
	client := &coverageOperationsClient{showResponse: connect.NewResponse(&coveragev1.GetProjectionResponse{})}
	ctx := coverageContext(nil, map[string]string{"projection": "substrate"})
	if _, err := (&handlers{client: client}).showCall(ctx); err != nil {
		t.Fatal(err)
	}
	if client.showCalls != 1 || client.showRequest.GetProjection() != coveragev1.Projection_PROJECTION_SUBSTRATE {
		t.Fatalf("GetProjection request = %#v, calls = %d", client.showRequest, client.showCalls)
	}

	invalid := &coverageOperationsClient{}
	_, err := (&handlers{client: invalid}).showCall(coverageContext(nil, map[string]string{"projection": "bad"}))
	if err == nil || invalid.showCalls != 0 {
		t.Fatalf("invalid projection err = %v, calls = %d", err, invalid.showCalls)
	}
	failed := &coverageOperationsClient{showErr: errors.New("offline")}
	if _, err := (&handlers{client: failed}).showCall(ctx); err == nil || !strings.Contains(err.Error(), "coverage show") {
		t.Fatalf("Connect error = %v", err)
	}
	noResponse := &coverageOperationsClient{}
	if _, err := (&handlers{client: noResponse}).showCall(ctx); err == nil || !strings.Contains(err.Error(), "no projection response") {
		t.Fatalf("nil response error = %v", err)
	}
}

func TestValidateAndDriftCallsMapResponsesAndErrors(t *testing.T) {
	validate := &coverageOperationsClient{validateResponse: connect.NewResponse(&coveragev1.ValidateSetpointResponse{Ok: true})}
	msg, err := (&handlers{client: validate}).validateCall(nil)
	if err != nil || !msg.GetOk() || validate.validateCalls != 1 {
		t.Fatalf("ValidateSetpoint = %v, err = %v, calls = %d", msg, err, validate.validateCalls)
	}
	validateFailed := &coverageOperationsClient{validateErr: errors.New("offline")}
	if _, err := (&handlers{client: validateFailed}).validateCall(nil); err == nil || !strings.Contains(err.Error(), "coverage validate") {
		t.Fatalf("ValidateSetpoint error = %v", err)
	}
	validateNoResponse := &coverageOperationsClient{}
	if _, err := (&handlers{client: validateNoResponse}).validateCall(nil); err == nil || !strings.Contains(err.Error(), "no validation response") {
		t.Fatalf("ValidateSetpoint nil response = %v", err)
	}

	drift := &coverageOperationsClient{driftResponse: connect.NewResponse(&coveragev1.GetDriftResponse{})}
	ctx := coverageContext(map[string]string{"projection": "headroom"}, nil)
	if _, err := (&handlers{client: drift}).driftCall(ctx); err != nil {
		t.Fatal(err)
	}
	if drift.driftCalls != 1 || drift.driftRequest.GetProjection() != coveragev1.Projection_PROJECTION_HEADROOM {
		t.Fatalf("GetDrift request = %#v, calls = %d", drift.driftRequest, drift.driftCalls)
	}
	noFilter := &coverageOperationsClient{driftResponse: connect.NewResponse(&coveragev1.GetDriftResponse{})}
	if _, err := (&handlers{client: noFilter}).driftCall(coverageContext(map[string]string{"projection": ""}, nil)); err != nil {
		t.Fatal(err)
	}
	if noFilter.driftRequest.GetProjection() != coveragev1.Projection_PROJECTION_UNSPECIFIED {
		t.Fatalf("omitted drift filter = %#v", noFilter.driftRequest)
	}
	invalid := &coverageOperationsClient{}
	if _, err := (&handlers{client: invalid}).driftCall(coverageContext(map[string]string{"projection": "unknown"}, nil)); err == nil || invalid.driftCalls != 0 {
		t.Fatalf("invalid drift projection err = %v, calls = %d", err, invalid.driftCalls)
	}
	failed := &coverageOperationsClient{driftErr: errors.New("offline")}
	if _, err := (&handlers{client: failed}).driftCall(ctx); err == nil || !strings.Contains(err.Error(), "coverage drift") {
		t.Fatalf("GetDrift error = %v", err)
	}
	noResponse := &coverageOperationsClient{}
	if _, err := (&handlers{client: noResponse}).driftCall(ctx); err == nil || !strings.Contains(err.Error(), "no drift response") {
		t.Fatalf("GetDrift nil response = %v", err)
	}
}

func TestCoverageReportsPreserveProjectionAndCellEvidence(t *testing.T) {
	projection := &coveragev1.ProjectionCoverage{
		Projection: coveragev1.Projection_PROJECTION_CAPACITY,
		Ratio:      &coveragev1.Ratio{Value: .5},
		NowCount:   1,
		TotalCells: 2,
		Confidence: &coveragev1.Confidence{Level: coveragev1.ConfidenceLevel_CONFIDENCE_LEVEL_SKETCH, Rationale: "one denominator is inferred"},
	}
	status := (&handlers{}).statusReport(nil, &coveragev1.GetCoverageResponse{
		Projections:       []*coveragev1.ProjectionCoverage{projection},
		IntegrityFindings: []*coveragev1.IntegrityFinding{{Severity: "warning", Code: "missing_bar", Message: "capacity bar is missing"}},
	})
	if len(status.Results) != 2 || !strings.Contains(status.Results[0], "capacity: 50%") || !strings.Contains(status.Results[0], "confidence=sketch (one denominator is inferred)") || status.Results[1] != "[warning] missing_bar — capacity bar is missing" {
		t.Fatalf("status results = %#v", status.Results)
	}

	cell := &sharedv1.Cell{Id: "capacity/cpu", Status: sharedv1.CellStatus_CELL_STATUS_IN_REACH, Question: "CPU budget known?", Owner: "infra"}
	cells := (&handlers{}).cellsReport(nil, &coveragev1.ListCellsResponse{Cells: []*sharedv1.Cell{cell}})
	if len(cells.Results) != 1 || cells.Results[0] != "capacity/cpu [in_reach] CPU budget known? — infra" {
		t.Fatalf("cell results = %#v", cells.Results)
	}
	show := (&handlers{}).showReport(nil, &coveragev1.GetProjectionResponse{Projection: coveragev1.Projection_PROJECTION_CAPACITY, Coverage: projection, Cells: []*sharedv1.Cell{cell}})
	if len(show.Results) != 2 || !strings.Contains(show.Results[0], "capacity: 50%") || !strings.Contains(show.Results[1], "capacity/cpu") {
		t.Fatalf("show results = %#v", show.Results)
	}

	invalid := (&handlers{}).validateReport(nil, &coveragev1.ValidateSetpointResponse{Ok: false, Findings: []*coveragev1.IntegrityFinding{{Severity: "error", Code: "bar_missing", Message: "no capacity target"}}})
	if len(invalid.Summary) != 1 || invalid.Summary[0] != "Setpoint: INVALID (1 finding(s))." || invalid.Results[0] != "[error] bar_missing — no capacity target" {
		t.Fatalf("validation report = %#v", invalid)
	}
	drift := (&handlers{}).driftReport(nil, &coveragev1.GetDriftResponse{Findings: []*coveragev1.DriftFinding{{Code: "bar_drift", CellRef: "capacity/cpu", Message: "measured threshold differs"}}})
	if len(drift.Results) != 1 || drift.Results[0] != "[bar_drift] capacity/cpu — measured threshold differs" {
		t.Fatalf("drift report = %#v", drift)
	}
}
