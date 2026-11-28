package trials

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const execHelperMode = "MOM_TRIALS_EXEC_HELPER_MODE"

func TestExecRunnerReportsMissingCommand(t *testing.T) {
	name := filepath.Join(t.TempDir(), "missing-command")
	_, err := execRunner(context.Background(), name)
	if err == nil || !strings.Contains(err.Error(), "not found on PATH") {
		t.Fatalf("execRunner() error = %v, want missing-command error", err)
	}
}

func TestExecRunnerCapturesOutputAndReportsProcessFailures(t *testing.T) {
	for _, test := range []struct {
		name    string
		mode    string
		wantOut string
		wantErr bool
	}{
		{name: "success", mode: "success", wantOut: "helper output"},
		{name: "nonzero exit", mode: "fail", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(execHelperMode, test.mode)
			got, err := execRunner(context.Background(), os.Args[0], "-test.run=^TestExecRunnerHelperProcess$")
			if (err != nil) != test.wantErr {
				t.Fatalf("execRunner() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantOut != "" && !strings.Contains(string(got), test.wantOut) {
				t.Fatalf("execRunner() output = %q, want it to contain %q", got, test.wantOut)
			}
		})
	}
}

func TestExecRunnerStopsWhenCallerContextExpires(t *testing.T) {
	t.Setenv(execHelperMode, "wait")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := execRunner(ctx, os.Args[0], "-test.run=^TestExecRunnerHelperProcess$")
	if err == nil {
		t.Fatal("execRunner() error = nil, want context cancellation")
	}
	if time.Since(started) > 5*time.Second {
		t.Fatalf("execRunner() took %s after caller deadline; want prompt cancellation", time.Since(started))
	}
}

// TestExecRunnerHelperProcess is re-executed by execRunner; it stays inert in
// the normal package test process and uses portable Go behavior in the child.
func TestExecRunnerHelperProcess(t *testing.T) {
	switch os.Getenv(execHelperMode) {
	case "success":
		fmt.Fprint(os.Stdout, "helper output")
	case "fail":
		os.Exit(7)
	case "wait":
		fmt.Fprintln(os.Stdout, "waiting for cancellation")
		select {}
	}
}
