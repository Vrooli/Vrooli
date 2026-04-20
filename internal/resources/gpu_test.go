package resources

import (
	"context"
	"testing"
)

func withStubGPUProbe(t *testing.T, result bool) {
	t.Helper()
	orig := gpuProbe
	gpuProbe = func(ctx context.Context, probe string) bool { return result }
	t.Cleanup(func() { gpuProbe = orig })
}

func TestShouldUseGPU(t *testing.T) {
	cases := []struct {
		name     string
		probe    string
		override string
		stub     bool // value returned by the stubbed probe in "auto" cases
		want     bool
	}{
		{"empty probe never uses GPU", "", "auto", true, false},
		{"override=on forces true", "nvidia", "on", false, true},
		{"override=off forces false", "nvidia", "off", true, false},
		{"override=auto delegates to probe (pass)", "nvidia", "auto", true, true},
		{"override=auto delegates to probe (fail)", "nvidia", "auto", false, false},
		{"unset override defaults to auto (pass)", "nvidia", "", true, true},
		{"unset override defaults to auto (fail)", "nvidia", "", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withStubGPUProbe(t, tc.stub)
			t.Setenv(gpuOverrideEnvVar, tc.override)
			if got := shouldUseGPU(context.Background(), tc.probe); got != tc.want {
				t.Fatalf("shouldUseGPU(%q) with override=%q stub=%v: got %v, want %v",
					tc.probe, tc.override, tc.stub, got, tc.want)
			}
		})
	}
}

func TestRunGPUProbeUnknownProbeReturnsFalse(t *testing.T) {
	if runGPUProbe(context.Background(), "radeon") {
		t.Fatal("unknown probe should return false")
	}
}

func TestNvidiaProbeMissingNvidiaSmiReturnsFalse(t *testing.T) {
	orig := lookPathCommandFn
	lookPathCommandFn = func(string) (string, error) { return "", errFakeNotFound{} }
	t.Cleanup(func() { lookPathCommandFn = orig })
	if nvidiaProbe(context.Background()) {
		t.Fatal("nvidiaProbe should return false when nvidia-smi is not on PATH")
	}
}

type errFakeNotFound struct{}

func (errFakeNotFound) Error() string { return "not found" }
