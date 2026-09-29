package library

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	"github.com/stretchr/testify/require"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
)

type declaredAdmissionServer struct {
	libraryconnect.UnimplementedLibraryServiceHandler
	requests     chan *libraryv1.RunDeclaredProgramRequest
	observations chan *libraryv1.GetDeclaredExecutionRequest
	closures     chan *libraryv1.RunDeclaredProgramRequest
	err          error
}

func (s declaredAdmissionServer) CloseDeclaredAdmission(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	s.closures <- req.Msg
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Program: &programsv1.Program{Id: "original", Status: programsv1.ProgramStatus_PROGRAM_STATUS_RUNNING}}), nil
}

func TestLibraryCloseAdmissionDoesNotDispatchOrClaimEffectsStopped(t *testing.T) {
	s := declaredAdmissionServer{requests: make(chan *libraryv1.RunDeclaredProgramRequest, 1), closures: make(chan *libraryv1.RunDeclaredProgramRequest, 1)}
	_, handler := libraryconnect.NewLibraryServiceHandler(s)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	h := &handlers{client: libraryconnect.NewLibraryServiceClient(server.Client(), server.URL)}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true}}, Flags: []cliapp.Flag{{Name: "idempotency-key"}, {Name: "expected-digest"}, {Name: "admission-deadline"}, {Name: "input"}, {Name: "caller-run-id"}, {Name: "provenance"}, {Name: "caller-agent-profile"}, {Name: "caller-skill-id"}, {Name: "caller-harness"}, {Name: "grant"}}}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"fixture.once", "--idempotency-key", "original-key", "--expected-digest", "original-digest", "--admission-deadline", "2026-09-28T00:00:00Z", "--input", "candidate=frozen", "--caller-run-id", "original-owner", "--grant", "binding:workspace-sandbox/change/promote"}, nil, nil, nil)
	require.NoError(t, err)
	r, err := h.closeAdmission(ctx)
	require.NoError(t, err)
	req := <-s.closures
	require.Equal(t, "original-key", req.IdempotencyKey)
	require.Equal(t, "original-digest", req.ExpectedDigest)
	require.Equal(t, "2026-09-28T00:00:00Z", req.AdmissionDeadline)
	require.Equal(t, "frozen", req.Inputs.AsMap()["candidate"])
	require.Equal(t, "original-owner", req.Caller.RunId)
	require.Equal(t, []string{"binding:workspace-sandbox/change/promote"}, req.Grants)
	require.Empty(t, s.requests)
	require.Contains(t, h.closeAdmissionReport(ctx, r).Result[1], "still draining")
}

func (s declaredAdmissionServer) GetDeclaredExecution(_ context.Context, req *connect.Request[libraryv1.GetDeclaredExecutionRequest]) (*connect.Response[libraryv1.GetDeclaredExecutionResponse], error) {
	s.observations <- req.Msg
	return connect.NewResponse(&libraryv1.GetDeclaredExecutionResponse{AdmissionContractVersion: 1}), nil
}

func TestLibraryExecutionOnlyObservesOriginalKey(t *testing.T) {
	s := declaredAdmissionServer{requests: make(chan *libraryv1.RunDeclaredProgramRequest, 1), observations: make(chan *libraryv1.GetDeclaredExecutionRequest, 1)}
	_, handler := libraryconnect.NewLibraryServiceHandler(s)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	h := &handlers{client: libraryconnect.NewLibraryServiceClient(server.Client(), server.URL)}
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true}}, Flags: []cliapp.Flag{{Name: "idempotency-key", Required: true}}}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"fixture.once", "--idempotency-key", "original-key"}, nil, nil, nil)
	require.NoError(t, err)
	response, err := h.execution(ctx)
	require.NoError(t, err)
	require.False(t, response.Found)
	req := <-s.observations
	require.Equal(t, "fixture.once", req.Name)
	require.Equal(t, "original-key", req.IdempotencyKey)
	require.Empty(t, s.requests, "observation dispatched a program")
	require.Contains(t, h.executionReport(ctx, response).Summary[0], "does not authorize a retry")
}

func (s declaredAdmissionServer) RunDeclaredProgram(_ context.Context, req *connect.Request[libraryv1.RunDeclaredProgramRequest]) (*connect.Response[libraryv1.RunDeclaredProgramResponse], error) {
	s.requests <- req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&libraryv1.RunDeclaredProgramResponse{Program: &programsv1.Program{Id: "retained", Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED}, Terminal: true}), nil
}

func TestLibraryRunForwardsPinnedAdmissionAndExplainsLostResponse(t *testing.T) {
	for _, lostResponse := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted", true: "lost response"}[lostResponse], func(t *testing.T) {
			s := declaredAdmissionServer{requests: make(chan *libraryv1.RunDeclaredProgramRequest, 1)}
			if lostResponse {
				s.err = connect.NewError(connect.CodeUnavailable, errors.New("connection lost"))
			}
			_, handler := libraryconnect.NewLibraryServiceHandler(s)
			server := httptest.NewServer(handler)
			defer server.Close()
			h := &handlers{client: libraryconnect.NewLibraryServiceClient(server.Client(), server.URL)}
			schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "name", Required: true}}, Flags: []cliapp.Flag{{Name: "expected-digest"}, {Name: "idempotency-key"}, {Name: "admission-deadline"}, {Name: "input"}, {Name: "provenance"}, {Name: "caller-run-id"}, {Name: "caller-agent-profile"}, {Name: "caller-skill-id"}, {Name: "caller-harness"}, {Name: "grant"}}}
			ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"fixture.once", "--expected-digest", "exact-digest", "--idempotency-key", "owner-attempt", "--admission-deadline", "2026-09-28T00:00:00Z", "--input", "candidate=frozen", "--caller-run-id", "owner"}, nil, nil, nil)
			require.NoError(t, err)
			response, err := h.run(ctx)
			if lostResponse {
				require.ErrorContains(t, err, "repeat only this exact request")
				require.ErrorContains(t, err, "do not renew the deadline")
			} else {
				require.NoError(t, err)
				require.Equal(t, "retained", response.Program.Id)
			}
			req := <-s.requests
			require.Equal(t, "exact-digest", req.ExpectedDigest)
			require.Equal(t, "owner-attempt", req.IdempotencyKey)
			require.Equal(t, "2026-09-28T00:00:00Z", req.AdmissionDeadline)
			require.Equal(t, "frozen", req.Inputs.AsMap()["candidate"])
			require.Equal(t, "owner", req.Caller.RunId)
			require.True(t, req.Async)
		})
	}
}

func TestRepeatedInputsPreserveChannelAndSelectors(t *testing.T) {
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "input"}}}
	ctx, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--input", "channel=test", "--input", "limit=3", "--input", `request={"a":[1,2]}`}, nil, nil, nil)
	require.NoError(t, err)
	inputs, err := declaredInputs(ctx)
	require.NoError(t, err)
	require.Equal(t, "test", inputs["channel"])
	require.Equal(t, float64(3), inputs["limit"])
	require.Equal(t, map[string]any{"a": []any{float64(1), float64(2)}}, inputs["request"])
	duplicate, err := cliapp.NewTestRunContextFromArgs(schema, []string{"--input", "channel=test", "--input", "channel=operator"}, nil, nil, nil)
	require.NoError(t, err)
	_, err = declaredInputs(duplicate)
	require.ErrorContains(t, err, "duplicate input")
	_, err = parseInputPairs("channel=test,channel=operator")
	require.ErrorContains(t, err, "duplicate input")
}

type interruptedWait struct {
	programsconnect.UnimplementedProgramServiceHandler
}

func (interruptedWait) WaitForProgram(context.Context, *connect.Request[programsv1.WaitForProgramRequest]) (*connect.Response[programsv1.WaitForProgramResponse], error) {
	return nil, connect.NewError(connect.CodeUnavailable, errors.New("unexpected EOF"))
}

func TestInterruptedLibraryWaitRetainsIdentityAndSafeRecovery(t *testing.T) {
	_, handler := programsconnect.NewProgramServiceHandler(interruptedWait{})
	server := httptest.NewServer(handler)
	defer server.Close()
	var progress bytes.Buffer
	h := &handlers{programs: programsconnect.NewProgramServiceClient(server.Client(), server.URL), progress: &progress}
	_, err := h.awaitAccepted(&libraryv1.RunDeclaredProgramResponse{Program: &programsv1.Program{Id: "prog_durable", Status: programsv1.ProgramStatus_PROGRAM_STATUS_ACCEPTED}})
	require.ErrorContains(t, err, "unexpected EOF")
	require.ErrorContains(t, err, "program-runtime programs get prog_durable --json")
	require.ErrorContains(t, err, "do not resubmit automatically")
	require.Contains(t, progress.String(), "program accepted: prog_durable")
}

func TestSplitInputPairsPreservesStructuredJSON(t *testing.T) {
	parts := splitInputPairs(`request={"owner":"fixture","input":{"a":1,"b":["x","y"]}},wait_seconds=2`)
	require.Equal(t, []string{`request={"owner":"fixture","input":{"a":1,"b":["x","y"]}}`, "wait_seconds=2"}, parts)
	inputs, err := parseInputPairs(strings.Join(parts, ","))
	require.NoError(t, err)
	require.Equal(t, float64(2), inputs["wait_seconds"])
}

func TestLibraryRunStatusAndOutcomeGate(t *testing.T) {
	require.Equal(t, "ok", libraryRunStatus("{'status': 'ok', 'phase': 'report'}"))
	require.Equal(t, "ok", libraryRunStatus(`{"phase":"report","status":"ok"}`))
	require.Equal(t, "partial", libraryRunStatus("{'status': 'partial'}"))
	require.Empty(t, libraryRunStatus("no envelope"))

	h := &handlers{}
	require.NoError(t, h.runOutcome(nil, &libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: "{'status': 'partial'}",
	}}))
	require.EqualError(t, h.runOutcome(nil, &libraryv1.RunDeclaredProgramResponse{Terminal: true, Program: &programsv1.Program{
		Status: programsv1.ProgramStatus_PROGRAM_STATUS_SUCCEEDED, Stdout: "{'status': 'failed'}",
	}}), `library run envelope status "failed"`)
}
