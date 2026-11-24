package entrypoint

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

type testRunner struct {
	args     []string
	err      error
	runCalls int
}

func (r *testRunner) Run(args []string) error {
	r.runCalls++
	r.args = append([]string(nil), args...)
	return r.err
}

func TestRunAndExitReportsConstructorFailure(t *testing.T) {
	wantErr := errors.New("configuration invalid")
	var output bytes.Buffer
	var exitCodes []int
	runner := &testRunner{}

	RunAndExit(func() (*testRunner, error) { return runner, wantErr }, []string{"ignored"}, &output, func(code int) { exitCodes = append(exitCodes, code) })

	if output.String() != "Error: configuration invalid\n" {
		t.Fatalf("stderr = %q", output.String())
	}
	if !reflect.DeepEqual(exitCodes, []int{1}) {
		t.Fatalf("exit codes = %v, want [1]", exitCodes)
	}
	if runner.runCalls != 0 {
		t.Fatalf("Run called %d times after constructor failure", runner.runCalls)
	}
}

func TestRunAndExitForwardsArgsAndReportsRunFailure(t *testing.T) {
	wantErr := errors.New("command failed")
	runner := &testRunner{err: wantErr}
	args := []string{"resource", "list", "--json"}
	var output bytes.Buffer
	var exitCodes []int

	RunAndExit(func() (*testRunner, error) { return runner, nil }, args, &output, func(code int) { exitCodes = append(exitCodes, code) })

	if !reflect.DeepEqual(runner.args, args) {
		t.Fatalf("Run args = %v, want %v", runner.args, args)
	}
	if runner.runCalls != 1 {
		t.Fatalf("Run called %d times, want once", runner.runCalls)
	}
	if output.String() != "Error: command failed\n" {
		t.Fatalf("stderr = %q", output.String())
	}
	if !reflect.DeepEqual(exitCodes, []int{1}) {
		t.Fatalf("exit codes = %v, want [1]", exitCodes)
	}
}

func TestRunAndExitLeavesSuccessfulRunAlone(t *testing.T) {
	runner := &testRunner{}
	var output bytes.Buffer
	var exitCodes []int

	RunAndExit(func() (*testRunner, error) { return runner, nil }, []string{"help"}, &output, func(code int) { exitCodes = append(exitCodes, code) })

	if runner.runCalls != 1 || len(exitCodes) != 0 || output.Len() != 0 {
		t.Fatalf("success state: Run calls %d, exits %v, stderr %q", runner.runCalls, exitCodes, output.String())
	}
}
