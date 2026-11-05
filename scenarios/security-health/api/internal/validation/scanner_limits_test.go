package validation

import (
	"context"
	"runtime"
	"testing"
)

func TestResolveScannerMaxProcsDefaultsAndClampsInvalidValues(t *testing.T) {
	if got := resolveScannerMaxProcs("", 32); got != 8 {
		t.Fatalf("default = %d, want 8", got)
	}
	if got := resolveScannerMaxProcs("0", 32); got != 8 {
		t.Fatalf("zero = %d, want default 8", got)
	}
	if got := resolveScannerMaxProcs("33", 32); got != 8 {
		t.Fatalf("above host = %d, want default 8", got)
	}
	if got := resolveScannerMaxProcs("5", 32); got != 5 {
		t.Fatalf("valid override = %d, want 5", got)
	}
	if got := defaultScannerMaxProcs(1); got != 1 {
		t.Fatalf("single-core default = %d, want 1", got)
	}
}

func TestScannerEnvironmentCarriesResolvedGOMAXPROCS(t *testing.T) {
	t.Setenv(EnvScannerMaxProcs, "3")
	if got := scannerEnvironment()["GOMAXPROCS"]; got != "3" {
		t.Fatalf("GOMAXPROCS = %q, want 3", got)
	}

	if runtime.NumCPU() < 3 {
		t.Skip("host has fewer than three CPUs")
	}
	stdout, _, _, _, err := execCommander{}.RunProcessWithEnv(context.Background(), ".", scannerEnvironment(), "sh", "-c", "printf %s $GOMAXPROCS")
	if err != nil {
		t.Fatalf("run child: %v", err)
	}
	if string(stdout) != "3" {
		t.Fatalf("child GOMAXPROCS = %q, want 3", stdout)
	}
}
