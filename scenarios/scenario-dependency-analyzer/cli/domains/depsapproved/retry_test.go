package depsapproved

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
)

const (
	searchProcedure  = "/vrooli.scenario_dependency_analyzer.v1.dependency_governance.DependencyGovernanceService/SearchApprovedDependencies"
	installProcedure = "/vrooli.scenario_dependency_analyzer.v1.dependency_governance.DependencyGovernanceService/InstallDependency"
)

func unavailable() error {
	return connect.NewError(connect.CodeUnavailable, errors.New("unexpected EOF"))
}

func runRetry(procedure string, outcomes ...error) (calls int, slept []time.Duration, notes []string, err error) {
	err = retryTransient(context.Background(), procedure, func() error {
		out := outcomes[min(calls, len(outcomes)-1)]
		calls++
		return out
	}, func(d time.Duration) { slept = append(slept, d) }, func(s string) { notes = append(notes, s) })
	return calls, slept, notes, err
}

func TestReadOnlyCallSurvivesAnAPIThatRestartsMidRequest(t *testing.T) {
	calls, slept, notes, err := runRetry(searchProcedure, unavailable(), unavailable(), nil)
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v; a read that succeeds on the third try must succeed", calls, err)
	}
	if len(slept) != 2 || len(notes) != 2 {
		t.Fatalf("slept=%v notes=%v; each retry must wait and say so", slept, notes)
	}
}

func TestReadOnlyCallGivesUpAfterABoundedNumberOfAttempts(t *testing.T) {
	calls, _, _, err := runRetry(searchProcedure, unavailable())
	if calls != readRetryAttempts || connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("calls=%d err=%v; want %d attempts then the unavailable error", calls, err, readRetryAttempts)
	}
}

func TestMutatingCallIsNeverRepeated(t *testing.T) {
	calls, _, _, err := runRetry(installProcedure, unavailable(), nil)
	if calls != 1 || err == nil {
		t.Fatalf("calls=%d err=%v; an install that may have been applied must not be replayed", calls, err)
	}
}

func TestOnlyUnavailableIsRetried(t *testing.T) {
	for _, code := range []connect.Code{connect.CodeDeadlineExceeded, connect.CodeInvalidArgument, connect.CodeInternal} {
		calls, _, _, _ := runRetry(searchProcedure, connect.NewError(code, errors.New("x")), nil)
		if calls != 1 {
			t.Fatalf("%s was retried %d times; only unavailable is transient", code, calls)
		}
	}
}

func TestReadOnlyProceduresAreExactlyTheReads(t *testing.T) {
	base := "/pkg.DependencyGovernanceService/"
	for name, want := range map[string]bool{
		"ListApprovedDependencies": true, "SearchApprovedDependencies": true, "ExplainApprovedDependency": true,
		"ValidateFleetApprovedDependencies": true, "GetApprovedDependencyUsage": true, "PreviewVulnerabilityRemediation": true,
		"InstallDependency": false, "RemoveDependency": false, "UpsertApprovedDependency": false,
		"ApproveObservedDependency": false, "DenyVulnerableDependency": false, "BatchUpsertApprovedDependencies": false,
		"ProposeApprovedDependencyRecords": false,
	} {
		if got := readOnlyProcedure(base + name); got != want {
			t.Errorf("readOnlyProcedure(%s) = %v, want %v", name, got, want)
		}
	}
}
