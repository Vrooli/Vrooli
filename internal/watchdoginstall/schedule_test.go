package watchdoginstall

import (
	"runtime"
	"testing"
	"time"
)

func TestForUsesNativeBackendsAndMinimumInterval(t *testing.T) {
	tests := []struct {
		goos    string
		backend string
	}{
		{goos: "linux", backend: "systemd-user-timer"},
		{goos: "darwin", backend: "launchd-user-agent"},
		{goos: "macos", backend: "launchd-user-agent"},
		{goos: "windows", backend: "windows-task-scheduler"},
	}
	for _, tt := range tests {
		schedule := For(tt.goos, time.Second)
		if !schedule.Supported || schedule.Backend != tt.backend {
			t.Fatalf("For(%q)=%+v, want supported %q", tt.goos, schedule, tt.backend)
		}
		if schedule.Interval != time.Minute {
			t.Fatalf("For(%q) interval=%v, want one-minute floor", tt.goos, schedule.Interval)
		}
	}
}

func TestForUsesFiveMinuteDefaultWhenIntervalIsUnset(t *testing.T) {
	if got := For("linux", 0).Interval; got != DefaultInterval {
		t.Fatalf("unset interval=%v, want %v", got, DefaultInterval)
	}
}

func TestForUnsupportedPlatformHasRemediation(t *testing.T) {
	schedule := For("plan9", DefaultInterval)
	if schedule.Supported || schedule.Remediation == "" {
		t.Fatalf("unsupported schedule=%+v, want remediation", schedule)
	}
}

func TestCurrentBackendMatchesBuildTarget(t *testing.T) {
	want := map[string]string{"linux": "systemd-user-timer", "darwin": "launchd-user-agent", "windows": "windows-task-scheduler"}[runtime.GOOS]
	if want != "" && CurrentBackend() != want {
		t.Fatalf("CurrentBackend()=%q, want %q", CurrentBackend(), want)
	}
}
