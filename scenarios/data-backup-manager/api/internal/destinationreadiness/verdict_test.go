package destinationreadiness

import (
	"testing"

	"data-backup-manager/internal/sysmounts"
)

func TestEvaluateWriteVerdict(t *testing.T) {
	passed := WriteProbeResult{Ran: true, Passed: true}
	failed := WriteProbeResult{Ran: true, Passed: false, Error: "EIO"}

	cases := []struct {
		name string
		in   WriteVerdictInput
		want WriteVerdict
	}{
		{name: "not mounted", in: WriteVerdictInput{FilesystemState: sysmounts.FilesystemStateClean}, want: WriteBlocked},
		{name: "read only", in: WriteVerdictInput{Mounted: true, ReadOnly: true, FilesystemState: sysmounts.FilesystemStateClean}, want: WriteBlocked},
		{name: "write protected", in: WriteVerdictInput{Mounted: true, DeviceWriteProtected: true, FilesystemState: sysmounts.FilesystemStateClean}, want: WriteBlocked},
		{name: "probe failed", in: WriteVerdictInput{Mounted: true, FilesystemState: sysmounts.FilesystemStateClean, Probe: failed}, want: WriteBlocked},
		{name: "clean", in: WriteVerdictInput{Mounted: true, FilesystemState: sysmounts.FilesystemStateClean, Probe: passed}, want: WriteReady},
		{name: "dirty wins over probe", in: WriteVerdictInput{Mounted: true, FilesystemState: sysmounts.FilesystemStateDirty, Probe: passed}, want: WriteBlocked},
		{name: "unknown strict", in: WriteVerdictInput{Mounted: true, FilesystemState: sysmounts.FilesystemStateUnknown, Probe: passed}, want: WriteUnknown},
		{name: "unknown degraded without probe", in: WriteVerdictInput{Mounted: true, FilesystemState: sysmounts.FilesystemStateUnknown, AllowDegraded: true}, want: WriteUnknown},
		{name: "unknown degraded with probe", in: WriteVerdictInput{Mounted: true, FilesystemState: sysmounts.FilesystemStateUnknown, AllowDegraded: true, Probe: passed}, want: WriteDegraded},
		{name: "offline check verified", in: WriteVerdictInput{Mounted: true, FilesystemState: sysmounts.FilesystemStateUnknown, OfflineCheckVerified: true}, want: WriteReady},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := EvaluateWriteVerdict(tc.in)
			if got != tc.want {
				t.Fatalf("verdict = %q (%s), want %q", got, reason, tc.want)
			}
			if reason == "" {
				t.Fatal("verdict reason should never be empty")
			}
		})
	}
}
