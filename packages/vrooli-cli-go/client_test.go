package vroolicli

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type stubResponse struct {
	output []byte
	err    error
	delay  time.Duration
}

type stubRunner struct {
	responses []stubResponse
	calls     []stubCall
}

type stubCall struct {
	name        string
	args        []string
	hasDeadline bool
}

var _ Runner = (*stubRunner)(nil)

func (s *stubRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := s.next(ctx, name, args)
	if err != nil {
		// Run mirrors exec.Cmd.Output: no stdout bytes returned on error.
		return nil, err
	}
	return out, nil
}

// RunCombined mirrors exec.Cmd.CombinedOutput: the captured output is returned
// even when the command fails, so combined-output callers can inspect it.
func (s *stubRunner) RunCombined(ctx context.Context, name string, args ...string) ([]byte, error) {
	return s.next(ctx, name, args)
}

func (s *stubRunner) next(ctx context.Context, name string, args []string) ([]byte, error) {
	_, hasDeadline := ctx.Deadline()
	s.calls = append(s.calls, stubCall{
		name:        name,
		args:        append([]string(nil), args...),
		hasDeadline: hasDeadline,
	})

	if len(s.responses) == 0 {
		return nil, errors.New("no stub response")
	}
	resp := s.responses[0]
	s.responses = s.responses[1:]

	if resp.delay > 0 {
		select {
		case <-time.After(resp.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if resp.err != nil {
		return resp.output, resp.err
	}
	return resp.output, nil
}

func TestListResourcesFallsBackFromVerboseToPlain(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{err: errors.New("verbose unsupported")},
			{output: []byte(`{"success":true,"resources":[{"name":"redis","path":"/r/redis","exists":true,"registered":true,"enabled":true,"has_cli":true,"unknown_field":"ignored"}]}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources returned error: %v", err)
	}
	resources := resp.GetResources()
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if got := resources[0].GetName(); got != "redis" {
		t.Fatalf("resource name = %q, want redis", got)
	}
	if !resources[0].GetHasCli() {
		t.Fatalf("expected snake_case has_cli to decode")
	}

	wantCalls := [][]string{
		{"--no-stale-check", "resource", "list", "--json", "--verbose"},
		{"--no-stale-check", "resource", "list", "--json"},
	}
	if len(runner.calls) != len(wantCalls) {
		t.Fatalf("expected %d calls, got %d", len(wantCalls), len(runner.calls))
	}
	for i, want := range wantCalls {
		if runner.calls[i].name != "vrooli" || !reflect.DeepEqual(runner.calls[i].args, want) {
			t.Fatalf("call %d = %s %v, want vrooli %v", i, runner.calls[i].name, runner.calls[i].args, want)
		}
	}
}

func TestLocksDecodesRegistryClaims(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"success":true,"locks":[],"registry_claims":[{"claim_id":"claim-1","instance_id":"inst-1","scenario":"agent-inbox","port_name":"api","port":17701,"claim_status":"bound","recommendation_code":"port-ok"}]}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.Locks(context.Background())
	if err != nil {
		t.Fatalf("Locks returned error: %v", err)
	}
	if !resp.GetSuccess() || len(resp.GetRegistryClaims()) != 1 {
		t.Fatalf("unexpected locks payload: %+v", resp)
	}
	claim := resp.GetRegistryClaims()[0]
	if claim.GetClaimId() != "claim-1" || claim.GetPort() != 17701 || claim.GetClaimStatus() != "bound" {
		t.Fatalf("registry claim not decoded: %+v", claim)
	}

	wantArgs := []string{"--no-stale-check", "locks", "--json"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("call args = %v, want %v", runner.calls, wantArgs)
	}
}

func TestRuntimeSupervisorStatusDecodes(t *testing.T) {
	// effective_renew_interval is an int64 (nanoseconds); protojson emits int64
	// as a JSON string, so the fixture mirrors that.
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"supervisor_id":"sup-1","status":"running","pid":1098674,"supervised_instance_count":41,"effective_renew_interval":"10000000000","last_tick":{"renewed":3}}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.RuntimeSupervisorStatus(context.Background())
	if err != nil {
		t.Fatalf("RuntimeSupervisorStatus returned error: %v", err)
	}
	if resp.GetSupervisorId() != "sup-1" || resp.GetStatus() != "running" || resp.GetSupervisedInstanceCount() != 41 {
		t.Fatalf("unexpected supervisor status: %+v", resp)
	}
	if resp.GetEffectiveRenewInterval() != 10000000000 {
		t.Fatalf("int64 duration mismatch: %d", resp.GetEffectiveRenewInterval())
	}
	if resp.GetLastTick().GetRenewed() != 3 {
		t.Fatalf("nested last_tick not decoded: %+v", resp.GetLastTick())
	}

	wantArgs := []string{"--no-stale-check", "runtime", "supervisor", "status", "--json"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("call args = %v, want %v", runner.calls, wantArgs)
	}
}

func TestScenarioPortDecodesTypedFields(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"success":true,"scenario":"scenario-dependency-analyzer","port_name":"API_PORT","port":16542}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.ScenarioPort(context.Background(), "scenario-dependency-analyzer", "API_PORT")
	if err != nil {
		t.Fatalf("ScenarioPort returned error: %v", err)
	}
	if !resp.GetSuccess() || resp.GetPort() != 16542 {
		t.Fatalf("unexpected port payload: success=%v port=%d", resp.GetSuccess(), resp.GetPort())
	}

	wantArgs := []string{"--no-stale-check", "scenario", "port", "scenario-dependency-analyzer", "API_PORT", "--json"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("call args = %v, want %v", runner.calls, wantArgs)
	}
}

func TestScenarioPortRequiresNameAndPort(t *testing.T) {
	client := New(WithRunner(&stubRunner{}))
	if _, err := client.ScenarioPort(context.Background(), "", "API_PORT"); err == nil {
		t.Fatal("expected error for empty name")
	}
	if _, err := client.ScenarioPort(context.Background(), "analyzer", "  "); err == nil {
		t.Fatal("expected error for empty port name")
	}
}

func TestScenarioPortAtPathPassesPhysicalScenarioDirectory(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"success":true,"scenario":"generated","port_name":"API_PORT","port":16542}`)}}}
	client := New(WithRunner(runner))

	_, err := client.ScenarioPortAtPath(context.Background(), "generated", "API_PORT", "/tmp/workspace/scenarios/generated")
	if err != nil {
		t.Fatalf("ScenarioPortAtPath returned error: %v", err)
	}
	wantArgs := []string{"--no-stale-check", "scenario", "port", "generated", "API_PORT", "--json", "--path", "/tmp/workspace/scenarios/generated"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("call args = %v, want %v", runner.calls, wantArgs)
	}
}

func TestListScenariosDecodesTypedFields(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{
			{output: []byte(`{"success":true,"summary":{"total_scenarios":1,"running":1,"available":0},"scenarios":[{"name":"ecosystem-manager","description":"Control plane","version":"1.0.0","status":"running","path":"/s/em"}]}`)},
		},
	}
	client := New(WithRunner(runner))

	resp, err := client.ListScenarios(context.Background())
	if err != nil {
		t.Fatalf("ListScenarios returned error: %v", err)
	}
	if got := resp.GetSummary().GetTotalScenarios(); got != 1 {
		t.Fatalf("total_scenarios = %d, want 1", got)
	}
	if got := resp.GetScenarios()[0].GetStatus(); got != "running" {
		t.Fatalf("scenario status = %q, want running", got)
	}
}

func TestRunnerErrorIsReturned(t *testing.T) {
	client := New(WithRunner(&stubRunner{
		responses: []stubResponse{{err: errors.New("boom")}},
	}))

	_, err := client.ListScenarios(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "run vrooli --no-stale-check scenario list --json") {
		t.Fatalf("error %q does not include command", err)
	}
}

func TestBadJSONIsReturned(t *testing.T) {
	client := New(WithRunner(&stubRunner{
		responses: []stubResponse{{output: []byte(`not json`)}},
	}))

	_, err := client.ListScenarios(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "scenario list") || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTimeoutAppliedWhenContextHasNoDeadline(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{{delay: 50 * time.Millisecond}},
	}
	client := New(WithRunner(runner), WithTimeout(10*time.Millisecond))

	_, err := client.ListScenarios(context.Background())
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if len(runner.calls) != 1 || !runner.calls[0].hasDeadline {
		t.Fatalf("expected runner call with deadline, got %+v", runner.calls)
	}
}

func TestNoStaleCheckInjectedByDefault(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"success":true,"scenarios":[]}`)}}}
	if _, err := New(WithRunner(runner)).ListScenarios(context.Background()); err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	want := []string{"--no-stale-check", "scenario", "list", "--json"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, want) {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, want)
	}
}

func TestStaleCheckCanBeReenabled(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{{output: []byte(`{"success":true,"scenarios":[]}`)}}}
	if _, err := New(WithRunner(runner), WithStaleCheck(true)).ListScenarios(context.Background()); err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	want := []string{"scenario", "list", "--json"}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, want) {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, want)
	}
}

// TestVerboseTimeoutDoesNotRetry proves the fallback shares one operation
// budget: a first attempt that exhausts the deadline is not retried (which
// would otherwise grant a second full timeout).
func TestVerboseTimeoutDoesNotRetry(t *testing.T) {
	runner := &stubRunner{responses: []stubResponse{
		{delay: 50 * time.Millisecond},
		{output: []byte(`{"success":true,"resources":[]}`)},
	}}
	client := New(WithRunner(runner), WithTimeout(10*time.Millisecond))

	_, err := client.ListResources(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected exactly 1 call (no retry after timeout), got %d", len(runner.calls))
	}
}

func TestCallerDeadlineIsPreserved(t *testing.T) {
	runner := &stubRunner{
		responses: []stubResponse{{output: []byte(`{"success":true,"scenarios":[]}`)}},
	}
	client := New(WithRunner(runner), WithTimeout(time.Hour))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if _, err := client.ListScenarios(ctx); err != nil {
		t.Fatalf("ListScenarios returned error: %v", err)
	}
	if len(runner.calls) != 1 || !runner.calls[0].hasDeadline {
		t.Fatalf("expected caller deadline to be passed to runner")
	}
}
