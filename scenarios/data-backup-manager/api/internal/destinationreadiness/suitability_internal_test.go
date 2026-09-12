package destinationreadiness

import (
	"strings"
	"testing"
)

// A destination served by a driver that faults in kernel context must be
// refused outright. The on-disk format is irrelevant to that decision: the
// blast radius of a write fault is a property of the driver.
//
// Regression anchor: on 2026-08-19 a backup write to an NTFS volume mounted
// with ntfs3 reached `BUG at fs/iomap/buffered-io.c:1061` and panicked the
// host. Readiness had rated that destination a pass.
func TestCheckFilesystemRefusesKernelFaultingDrivers(t *testing.T) {
	got := checkFilesystem("ntfs", "ntfs3", false)

	if got.Severity != SeverityFail {
		t.Fatalf("severity = %q, want %q for an ntfs3 mount", got.Severity, SeverityFail)
	}
	if !strings.Contains(got.Message, "ntfs3") {
		t.Fatalf("message must name the offending driver, got %q", got.Message)
	}
	if strings.TrimSpace(got.NextAction) == "" {
		t.Fatal("a refusal must carry an actionable next step")
	}
}

// The driver veto outranks format suitability. A format that would otherwise
// rate a pass must still be refused when the driver is unsafe, so the veto
// cannot be bypassed by reformatting metadata alone.
func TestCheckFilesystemDriverVetoOutranksFormat(t *testing.T) {
	if got := checkFilesystem("ext4", "ntfs3", false); got.Severity != SeverityFail {
		t.Fatalf("severity = %q, want the driver veto to win over a passing format", got.Severity)
	}
}

// The same volume served by a userspace driver is contained to the backup
// process. That is a downgrade to warning, not an endorsement — restore
// fidelity still depends on the driver.
func TestCheckFilesystemAllowsUserspaceNTFSWithAWarning(t *testing.T) {
	got := checkFilesystem("ntfs", "fuseblk", false)

	if got.Severity != SeverityWarning {
		t.Fatalf("severity = %q, want %q for userspace-served NTFS", got.Severity, SeverityWarning)
	}
}

// An unmounted NTFS volume offers no driver evidence. It must not be rated a
// pass, because the driver it would receive on mount is unknown and the Linux
// automount default is the refused in-kernel one.
func TestCheckFilesystemDoesNotPassNTFSWithoutDriverEvidence(t *testing.T) {
	got := checkFilesystem("ntfs", "", false)

	if got.Severity == SeverityPass {
		t.Fatal("NTFS with no driver evidence must not rate a pass")
	}
}

// The driver check must not disturb the ordinary format verdicts.
func TestCheckFilesystemKeepsFormatVerdicts(t *testing.T) {
	for _, tc := range []struct {
		name          string
		fs, driver    string
		crossPlatform bool
		want          CheckSeverity
	}{
		{"ext4 linux-only", "ext4", "ext4", false, SeverityPass},
		{"ext4 cross-platform", "ext4", "ext4", true, SeverityWarning},
		{"exfat", "exfat", "exfat", false, SeverityPass},
		{"fat32 file-size limit", "vfat", "vfat", false, SeverityWarning},
		{"unknown format", "", "", false, SeverityUnknown},
		{"unvalidated format", "reiserfs", "reiserfs", false, SeverityWarning},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkFilesystem(tc.fs, tc.driver, tc.crossPlatform); got.Severity != tc.want {
				t.Fatalf("severity = %q, want %q", got.Severity, tc.want)
			}
		})
	}
}
