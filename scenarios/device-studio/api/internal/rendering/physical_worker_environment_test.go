package rendering

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPhysicalBlenderEnvironmentKeepsChildHomePrivate(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "attempt")
	env := physicalBlenderEnvironment(workspace)
	values := make(map[string]string, len(env))
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			t.Fatalf("environment entry %q is malformed", entry)
		}
		if _, exists := values[key]; exists {
			t.Fatalf("environment key %q is duplicated", key)
		}
		values[key] = value
	}
	if got, want := values["HOME"], filepath.Join(workspace, "home"); got != want {
		t.Fatalf("Blender child HOME = %q, want private attempt directory %q", got, want)
	}
	for key, dir := range map[string]string{
		"TMPDIR":                 "tmp",
		"BLENDER_USER_CONFIG":    "config",
		"BLENDER_USER_SCRIPTS":   "scripts",
		"BLENDER_USER_DATAFILES": "data",
		"PYTHONPYCACHEPREFIX":    "pycache",
	} {
		if got, want := values[key], filepath.Join(workspace, dir); got != want {
			t.Errorf("%s = %q, want private attempt directory %q", key, got, want)
		}
	}
}

func TestPhysicalBlenderDiagnosticsStayBounded(t *testing.T) {
	var stderr physicalBlenderLogBuffer
	stderr.limit = 4
	if n, err := stderr.Write([]byte("fail")); err != nil || n != 4 {
		t.Fatalf("write at bound = (%d, %v), want (4, nil)", n, err)
	}
	if n, err := stderr.Write([]byte("verbose-tail")); err != nil || n != len("verbose-tail") {
		t.Fatalf("verbose Blender log was not drained: (%d, %v)", n, err)
	}
	nativeErr := errors.New("exit status 17")
	got := physicalBlenderFailure(nativeErr, nil, stderr.data.String())
	if !strings.Contains(got.Error(), "tail") || stderr.data.Len() > stderr.limit {
		t.Fatalf("worker error did not retain a bounded log tail: err=%v retained=%q", got, stderr.data.String())
	}
	if !errors.Is(got, nativeErr) {
		t.Fatal("worker error did not retain the native exit error")
	}
}

func TestPhysicalBlenderInvocationUsesPrivatePythonEnvironmentAndDisablesAutoexec(t *testing.T) {
	args := physicalBlenderArgs("/attempt")
	want := []string{"--background", "--factory-startup", "--disable-autoexec", "--python-use-system-env", "--threads", "2", "--python-exit-code", "17", "--python", "/attempt/physical_blender.py", "--", "/attempt/physical-worker-input.json"}
	if !slices.Equal(args, want) {
		t.Fatalf("physical Blender argv = %#v, want fixed secure argv %#v", args, want)
	}
	env := physicalBlenderEnvironment("/attempt")
	joined := strings.Join(env, "\x00")
	if !strings.Contains(joined, "PYTHONDONTWRITEBYTECODE=1") || !strings.Contains(joined, "PYTHONPYCACHEPREFIX=/attempt/pycache") {
		t.Fatalf("physical Blender environment does not isolate Python cache writes: %#v", env)
	}
}
