package depsapproved

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckGoSumDriftCleanModulePasses(t *testing.T) {
	list := func(dir string) (string, error) { return "", nil }

	report := checkGoSumDrift([]string{"/repo/scenarios/a/cli", "/repo/scenarios/a/api"}, list)
	if report.Modules != 2 {
		t.Fatalf("expected 2 modules checked, got %d", report.Modules)
	}
	if len(report.Drift) != 0 {
		t.Fatalf("expected no drift, got %+v", report.Drift)
	}
	if err := reconcileCheck([]string{"/repo/scenarios/a/cli"}, list, true); err != nil {
		t.Fatalf("clean fleet should exit zero, got %v", err)
	}
}

func TestCheckGoSumDriftReportsDriftedModule(t *testing.T) {
	const drifted = "/repo/scenarios/brand-manager/cli"
	list := func(dir string) (string, error) {
		if dir != drifted {
			return "", nil
		}
		out := "package buf.build/x: missing go.sum entry for module providing package buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate; to add:\n\tgo get x\n"
		return out, errors.New("exit status 1")
	}

	report := checkGoSumDrift([]string{drifted, "/repo/scenarios/other/cli"}, list)
	if len(report.Drift) != 1 {
		t.Fatalf("expected exactly one drifted module, got %+v", report.Drift)
	}
	if report.Drift[0].Module != drifted {
		t.Fatalf("expected drift on %s, got %s", drifted, report.Drift[0].Module)
	}
	if len(report.Drift[0].Details) != 1 || !strings.Contains(report.Drift[0].Details[0], "missing go.sum entry") {
		t.Fatalf("expected a missing go.sum detail, got %+v", report.Drift[0].Details)
	}

	err := reconcileCheck([]string{drifted}, list, true)
	if err == nil {
		t.Fatalf("drifted fleet must exit non-zero")
	}
	if !strings.Contains(err.Error(), drifted) && !strings.Contains(err.Error(), "1 module") {
		t.Fatalf("error should name the drift count or module, got %v", err)
	}
}

func TestMissingGoSumDetailsDeduplicates(t *testing.T) {
	out := "a: missing go.sum entry for x\nb: compile error\na: missing go.sum entry for x\n"
	details := missingGoSumDetails(out)
	if len(details) != 1 {
		t.Fatalf("expected one deduplicated detail, got %v", details)
	}
}

func TestMissingGoSumDetailsIgnoresOtherFailures(t *testing.T) {
	if details := missingGoSumDetails("# pkg\n./main.go:1: undefined: foo\n"); len(details) != 0 {
		t.Fatalf("compile errors are not drift, got %v", details)
	}
}
